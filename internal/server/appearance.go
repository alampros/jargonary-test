package server

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"jargonary/internal/themes"
)

func (s *Server) appearanceAsset(w http.ResponseWriter, r *http.Request, name string) bool {
	if strings.HasPrefix(name, "themes/") {
		theme := strings.TrimSuffix(strings.TrimPrefix(name, "themes/"), ".css")
		if !themes.Valid(theme) || name != "themes/"+theme+".css" {
			http.NotFound(w, r)
			return true
		}
		data, _ := themes.Files.ReadFile(theme + ".css")
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(data)
		return true
	}
	if !strings.HasPrefix(name, "custom/") {
		return false
	}
	file := strings.TrimPrefix(name, "custom/")
	extension := ".css"
	files := s.Config.CustomCSSData
	contentType := "text/css; charset=utf-8"
	if strings.HasSuffix(file, ".js") {
		extension, files, contentType = ".js", s.Config.CustomJSData, "text/javascript; charset=utf-8"
	}
	index, err := strconv.Atoi(strings.TrimSuffix(file, extension))
	if err != nil || index < 0 || index >= len(files) || file != fmt.Sprintf("%d%s", index, extension) {
		http.NotFound(w, r)
		return true
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(files[index])
	return true
}

var appearanceTags = template.Must(template.New("appearance").Parse(`<link rel="icon" type="image/svg+xml" href="{{.FaviconURL}}">
<link rel="stylesheet" href="{{.ThemeURL}}">
{{range .CSS}}<link rel="stylesheet" href="{{.}}">
{{end}}{{range .JS}}<script defer src="{{.}}"></script>
{{end}}`))

func (s *Server) appearanceHead(base string) string {
	theme := s.Config.Theme
	if theme == "" {
		theme = "default"
	}
	data := struct {
		ThemeURL, FaviconURL string
		CSS, JS              []string
	}{ThemeURL: base + "/ui/themes/" + theme + ".css", FaviconURL: base + "/favicon.svg"}
	for i := range s.Config.CustomCSSData {
		data.CSS = append(data.CSS, fmt.Sprintf("%s/ui/custom/%d.css", base, i))
	}
	for i := range s.Config.CustomJSData {
		data.JS = append(data.JS, fmt.Sprintf("%s/ui/custom/%d.js", base, i))
	}
	var out bytes.Buffer
	_ = appearanceTags.Execute(&out, data)
	return out.String()
}
