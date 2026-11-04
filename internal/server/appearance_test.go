package server

import (
	"net/http"
	"strings"
	"testing"

	"jargonary/internal/themes"
)

func TestAppearanceOnPublicAdminAndDevelopmentPages(t *testing.T) {
	s := withPageAssets(t)
	s.Config.SiteName = `Example & Partners <Company>`
	s.Config.Theme = "github-dark"
	s.Config.CustomCSSData = [][]byte{[]byte(":root { --jg-accent: purple; }"), []byte("body { margin: 0; }")}
	s.Config.CustomJSData = [][]byte{[]byte("document.documentElement.dataset.custom = 'yes';")}
	for _, mode := range []string{"public", "admin", "dev"} {
		t.Run(mode, func(t *testing.T) {
			handler := s.Public()
			base := "/base"
			if mode == "admin" {
				handler, base = s.Admin(), ""
			}
			if mode == "dev" {
				s.DevUI = s.Assets
				defer func() { s.DevUI = nil }()
			}
			w := request(handler, "/", "", "")
			body := w.Body.String()
			if w.Code != 200 || !strings.Contains(body, `data-theme="github-dark"`) || !strings.Contains(body, `data-ui="`) {
				t.Fatalf("themed HTML missing: %d %s", w.Code, body)
			}
			if strings.Contains(body, "<Company>") || bootstrap(t, body).SiteName != s.Config.SiteName {
				t.Fatal("brand escaping or bootstrap incorrect")
			}
			theme := strings.Index(body, base+"/ui/themes/github-dark.css")
			first := strings.Index(body, base+"/ui/custom/0.css")
			second := strings.Index(body, base+"/ui/custom/1.css")
			js := strings.Index(body, `<script defer src="`+base+`/ui/custom/0.js"`)
			if theme < 0 || first < theme || second < first || js < second {
				t.Fatal("appearance order or base path incorrect")
			}
			if mode != "dev" && strings.Contains(w.Header().Get("Content-Security-Policy"), "unsafe-inline") {
				t.Fatal("custom scripts weakened production policy")
			}
		})
	}
}

func TestAppearanceAssetsExposeOnlyConfiguredFiles(t *testing.T) {
	s := withPageAssets(t)
	s.Config.CustomCSSData = [][]byte{[]byte("/* first */"), []byte("/* second */")}
	s.Config.CustomJSData = [][]byte{[]byte("/* hooks */")}
	pub := s.Public()
	for _, tc := range []struct{ path, content, mime string }{{"/ui/custom/0.css", "/* first */", "text/css"}, {"/ui/custom/1.css", "/* second */", "text/css"}, {"/ui/custom/0.js", "/* hooks */", "text/javascript"}} {
		w := request(pub, tc.path, "", "")
		if w.Code != 200 || w.Body.String() != tc.content || !strings.HasPrefix(w.Header().Get("Content-Type"), tc.mime) || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("custom asset response incorrect: %s", tc.path)
		}
	}
	for _, theme := range themes.Names {
		for _, handler := range []http.Handler{s.Public(), s.Admin()} {
			w := request(handler, "/ui/themes/"+theme+".css", "", "")
			if w.Code != 200 || !strings.Contains(w.Body.String(), "color-scheme") {
				t.Fatalf("bundled theme unavailable: %s", theme)
			}
		}
	}
	for _, path := range []string{"/ui/custom/2.css", "/ui/custom/-1.js", "/ui/custom/00.css", "/ui/custom/hooks.js", "/ui/custom/etc/passwd", "/ui/themes/missing.css"} {
		if w := request(pub, path, "", ""); w.Code != 404 {
			t.Fatalf("unconfigured asset exposed: %s", path)
		}
	}
}
