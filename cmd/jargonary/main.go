package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"jargonary/glossary"
	"jargonary/internal/config"
	"jargonary/internal/mcp"
	"jargonary/internal/scraper"
	"jargonary/internal/server"
	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "", "YAML configuration file")
	devUI := flag.Bool("dev-ui", false, "load frontend modules from local Vite (run from repository root)")
	validate := flag.String("validate", "", "validate a dictionary file and exit")
	format := flag.String("format", "json", "dictionary format for -validate: json, yaml, csv")
	schema := flag.Bool("schema", false, "print the structural dictionary JSON Schema and exit")
	flag.Parse()
	if *schema {
		registry := huma.NewMapRegistry("#/$defs/", huma.DefaultSchemaNamer)
		root := registry.Schema(reflect.TypeFor[vocabulary.Dictionary](), true, "")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "$ref": root.Ref, "$defs": registry.Map()})
	}
	if *validate != "" {
		b, err := os.ReadFile(*validate)
		if err != nil {
			return errors.New("cannot read validation input")
		}
		d, err := vocabulary.Parse(b, *format)
		if err != nil {
			return err
		}
		fmt.Printf("valid dictionary: %d entries\n", len(d.Entries))
		return nil
	}
	c, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	st, err := store.Open(c.Database)
	if err != nil {
		return errors.New("cannot initialize SQLite database; verify path, permissions, and schema compatibility")
	}
	defer st.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := initializeSources(ctx, c, st); err != nil {
		return err
	}
	svc := &server.Server{Config: c, Store: st}
	if *devUI {
		svc.DevUI = os.DirFS("web")
		slog.Info("development UI enabled", "vite", "http://127.0.0.1:5173")
	}
	type listener struct {
		name string
		http *http.Server
		net  net.Listener
	}
	var listeners []listener
	add := func(name, addr string, handler http.Handler) error {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("cannot bind %s listener; verify address availability", name)
		}
		listeners = append(listeners, listener{name, &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}, ln})
		return nil
	}
	defer func() {
		for _, l := range listeners {
			_ = l.net.Close()
		}
	}()
	if err := add("public", c.Server.Listen, svc.Public()); err != nil {
		return err
	}
	if c.Admin.Enabled {
		if err := add("admin", c.Admin.Listen, svc.Admin()); err != nil {
			return err
		}
		slog.Info("admin diagnostics enabled; restrict network access")
	}
	if c.MCP.Enabled {
		if err := add("mcp", c.MCP.Listen, mcp.New(st)); err != nil {
			return err
		}
		slog.Info("mcp server enabled; restrict network access")
	}
	errs := make(chan error, len(listeners))
	for _, l := range listeners {
		go func() {
			slog.Info("listener started", "interface", l.name, "address", l.net.Addr().String())
			errs <- l.http.Serve(l.net)
		}()
	}
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		if err := scraper.New(c, st).Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("scraper stopped")
			errs <- errors.New("scraper failed")
		}
	}()
	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-errs:
		stop()
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, l := range listeners {
		if err := l.http.Shutdown(shutdownCtx); err != nil {
			_ = l.http.Close()
		}
	}
	<-workerDone
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return errors.New("application worker or listener failed")
	}
	slog.Info("shutdown complete")
	return nil
}

func initializeSources(ctx context.Context, c config.Config, st *store.Store) error {
	ids := []string{}
	for _, source := range c.Sources {
		ids = append(ids, source.ID)
	}
	if c.IncludeProjectGlossary() {
		ids = append(ids, "project")
	}
	if err := st.Keep(ctx, ids); err != nil {
		return err
	}
	if !c.IncludeProjectGlossary() {
		return nil
	}
	d, err := vocabulary.Parse(glossary.JSON, "json")
	if err != nil {
		return fmt.Errorf("embedded glossary: %w", err)
	}
	_, err = st.Replace(ctx, "project", d, "", "")
	return err
}
