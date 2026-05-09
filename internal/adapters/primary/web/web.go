package web

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

func Handler() http.Handler {
	files, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return http.NotFoundHandler()
	}
	return spaHandler{files: files}
}

type spaHandler struct {
	files fs.FS
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" || path == "/yandex/callback" {
		serveFile(w, r, h.files, "index.html")
		return
	}

	name := path[1:]
	if _, err := fs.Stat(h.files, name); err != nil {
		http.NotFound(w, r)
		return
	}
	serveFile(w, r, h.files, name)
}

func serveFile(w http.ResponseWriter, r *http.Request, files fs.FS, name string) {
	switch name {
	case "index.html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case "styles.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case "app.js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	}
	http.ServeFileFS(w, r, files, name)
}

func AnimeCatalogHandler() http.Handler {
	client := &http.Client{Timeout: 10 * time.Second}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, err := animeCatalogURL(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
		if err != nil {
			http.Error(w, "build upstream request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "GolangProjectTemplate/1.0")

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "anime upstream unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})
}

func animeCatalogURL(query url.Values) (string, error) {
	mode := query.Get("mode")
	if mode == "" {
		mode = "top"
	}

	values := url.Values{}
	values.Set("limit", "8")

	switch mode {
	case "top":
		values.Set("filter", "bypopularity")
		return "https://api.jikan.moe/v4/top/anime?" + values.Encode(), nil
	case "airing":
		values.Set("filter", "airing")
		return "https://api.jikan.moe/v4/top/anime?" + values.Encode(), nil
	case "upcoming":
		values.Set("filter", "upcoming")
		return "https://api.jikan.moe/v4/top/anime?" + values.Encode(), nil
	case "search":
		q := query.Get("q")
		if q == "" {
			return "", errors.New("query is required")
		}
		values.Set("sfw", "true")
		values.Set("q", q)
		return "https://api.jikan.moe/v4/anime?" + values.Encode(), nil
	default:
		return "", errors.New("unsupported anime catalog mode")
	}
}
