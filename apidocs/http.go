package apidocs

import (
	"bytes"
	"embed"
	"net/http"
	"time"

	swgui "github.com/swaggest/swgui/v5emb"
)

const (
	SpecPath = "/docs/openapi.yaml"
	UIPath   = "/docs/"
)

//go:embed openapi.yaml
var files embed.FS

func UIHandler() http.Handler {
	return swgui.New("Marketplace API", SpecPath, UIPath)
}

func RedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, UIPath, http.StatusTemporaryRedirect)
	})
}

func SpecHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spec, err := files.ReadFile("openapi.yaml")
		if err != nil {
			http.Error(w, "openapi spec is unavailable", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		http.ServeContent(w, r, "openapi.yaml", time.Time{}, bytes.NewReader(spec))
	})
}
