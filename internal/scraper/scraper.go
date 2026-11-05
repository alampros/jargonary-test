// Package scraper synchronizes complete dictionary snapshots without deleting last-good data on failure.
package scraper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	"jargonary/internal/config"
	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
)

type Scraper struct {
	Config config.Config
	Store  *store.Store
	Client *http.Client
}

func New(c config.Config, s *store.Store) *Scraper {
	return &Scraper{Config: c, Store: s, Client: &http.Client{Timeout: time.Duration(c.Timeout), CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (s *Scraper) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Bounded concurrency; each source has at most one in-flight scrape.
	sem := make(chan struct{}, 4)
	done := make(chan error, len(s.Config.Sources))
	for _, source := range s.Config.Sources {
		go func(source config.Source) {
			var workerErr error
			defer func() { done <- workerErr }()
			var st store.Status
			states, _ := s.Store.Statuses(ctx)
			for _, old := range states {
				if old.Source == source.ID {
					st = old
					break
				}
			}
			st.Source = source.ID
			st.Running = false
			for {
				select {
				case <-ctx.Done():
					return
				case sem <- struct{}{}:
				}
				st = s.Once(ctx, source, st)
				<-sem
				if ctx.Err() != nil {
					return
				}
				delay := time.Duration(source.ScrapeInterval)
				if st.Failures > 0 {
					power := min(st.Failures, 5)
					multiplier := time.Duration(1 << power)
					if delay > time.Hour/multiplier {
						delay = time.Hour
					} else {
						delay *= multiplier
					}
				}
				delay = time.Duration(float64(delay) * (0.9 + rand.Float64()*0.2))
				st.NextAttempt = time.Now().UTC().Add(delay)
				if err := s.Store.SaveStatus(ctx, st); err != nil {
					if ctx.Err() == nil {
						workerErr = errors.New("cannot persist scrape scheduling state")
					}
					return
				}
				t := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					t.Stop()
					return
				case <-t.C:
				}
			}
		}(source)
	}
	var firstError error
	for range s.Config.Sources {
		if err := <-done; err != nil && firstError == nil {
			firstError = err
			cancel()
		}
	}
	if firstError != nil {
		return firstError
	}
	return ctx.Err()
}

func (s *Scraper) Once(ctx context.Context, source config.Source, st store.Status) store.Status {
	start := time.Now().UTC()
	st.Source = source.ID
	st.Running = true
	st.LastAttempt = start
	st.NextAttempt = time.Time{}
	if err := s.Store.SaveStatus(ctx, st); err != nil {
		return st
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.Config.Timeout))
	defer cancel()
	snapshots, err := s.Store.Snapshots(ctx)
	var previous *store.Snapshot
	if err == nil {
		for _, sn := range snapshots {
			if sn.Source == source.ID {
				copy := sn
				previous = &copy
				break
			}
		}
	}
	var body []byte
	var etag, modified string
	var unchanged bool
	if err == nil {
		body, etag, modified, unchanged, err = s.fetch(ctx, source, previous)
	}
	changed := false
	if err == nil && !unchanged {
		var d vocabulary.Dictionary
		d, err = vocabulary.Parse(body, source.Format)
		if err == nil {
			changed, err = s.Store.Replace(ctx, source.ID, d, etag, modified, fetchKey(source))
			if err == nil {
				st.Entries = len(d.Entries)
			}
		}
	}
	st.Running = false
	outcome := store.Outcome{At: start, DurationMS: time.Since(start).Milliseconds()}
	if err != nil {
		st.Failures++
		outcome.Result = "failed"
		// fetch/parse errors are deliberately generated here, not forwarded from net/http or source payloads.
		outcome.Detail = err.Error()
	} else {
		st.Failures = 0
		st.LastSuccess = time.Now().UTC()
		outcome.Result = "unchanged"
		if previous != nil && st.LastChange.IsZero() {
			st.LastChange = previous.AcceptedAt
			st.Entries = len(previous.Dictionary.Entries)
		}
		if changed {
			st.LastChange = st.LastSuccess
			outcome.Result = "updated"
		}
	}
	st.Recent = append(st.Recent, outcome)
	if len(st.Recent) > 20 {
		st.Recent = st.Recent[len(st.Recent)-20:]
	}
	// Use a short independent context so shutdown persists final diagnostics when the request was canceled.
	saveCtx, cancelSave := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancelSave()
	_ = s.Store.SaveStatus(saveCtx, st)
	return st
}

func (s *Scraper) fetch(ctx context.Context, source config.Source, previous *store.Snapshot) ([]byte, string, string, bool, error) {
	var r io.ReadCloser
	var etag, modified string
	if source.File != "" {
		// Inspect before opening to reject devices and other special files;
		// the nonblocking Unix open and descriptor check also cover replacement
		// with a FIFO between this check and open.
		info, err := os.Stat(source.File)
		if err != nil {
			return nil, "", "", false, errors.New("cannot inspect source file; retaining last successful snapshot")
		}
		if !info.Mode().IsRegular() {
			return nil, "", "", false, errors.New("source must be a regular file")
		}
		f, err := openSourceFile(source.File)
		if err != nil {
			return nil, "", "", false, errors.New("cannot open source file; retaining last successful snapshot")
		}
		info, err = f.Stat()
		if err != nil || !info.Mode().IsRegular() {
			f.Close()
			return nil, "", "", false, errors.New("source must be a regular file")
		}
		r = f
	} else {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
		if err != nil {
			return nil, "", "", false, errors.New("cannot create source request")
		}
		req.Header.Set("Accept", "application/json, application/yaml, text/yaml")
		if source.Token != "" {
			req.Header.Set("Authorization", "Bearer "+source.Token)
		}
		if previous != nil && previous.FetchKey == fetchKey(source) {
			if previous.ETag != "" {
				req.Header.Set("If-None-Match", previous.ETag)
			}
			if previous.LastModified != "" {
				req.Header.Set("If-Modified-Since", previous.LastModified)
			}
		}
		resp, err := s.Client.Do(req)
		if err != nil {
			return nil, "", "", false, errors.New("source request failed (network, TLS, or timeout); retaining last successful snapshot")
		}
		if resp.StatusCode == http.StatusNotModified {
			resp.Body.Close()
			if previous == nil || previous.FetchKey != fetchKey(source) || (previous.ETag == "" && previous.LastModified == "") {
				return nil, "", "", false, errors.New("received 304 without a cached snapshot")
			}
			return nil, "", "", true, nil
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, "", "", false, fmt.Errorf("source returned HTTP %d; retaining last successful snapshot", resp.StatusCode)
		}
		r = resp.Body
		etag = resp.Header.Get("ETag")
		modified = resp.Header.Get("Last-Modified")
	}
	defer r.Close()
	b, err := io.ReadAll(io.LimitReader(r, s.Config.MaxBytes+1))
	if err != nil {
		return nil, "", "", false, errors.New("cannot read source body")
	}
	if int64(len(b)) > s.Config.MaxBytes {
		return nil, "", "", false, errors.New("source exceeds max_bytes")
	}
	return b, etag, modified, false, nil
}

// Validators are scoped to the target, format, and credential principal. Reusing
// a source ID for a different resource must not validate the old representation.
func fetchKey(s config.Source) string {
	h := sha256.Sum256([]byte(s.URL + "\x00" + s.File + "\x00" + s.Format + "\x00" + s.Token))
	return hex.EncodeToString(h[:])
}
