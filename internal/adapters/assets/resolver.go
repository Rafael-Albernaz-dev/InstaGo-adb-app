package assets

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// CreativeAssetsMap binds stable URL keys under /creative-assets/ to relative
// paths inside the configured creative base directory (read-only).
var CreativeAssetsMap = map[string]string{
	"fonts/sora.woff2":          "02-identidade/fontes/sora.woff2",
	"fonts/inter.woff2":         "02-identidade/fontes/inter.woff2",
	"logo/rural-conecta.png":    "02-identidade/logos/rural-conecta/logo.png",
	"photo/p-combo-casa.png":    "05-biblioteca/imagens/timeline/p-combo-casa.png",
	"reference/02-combo-02.jpg": "03-banners-prontos/timeline/02-combo/02-combo-02.jpg",
}

// Resolver maps /creative-assets/* HTTP requests safely to files under baseDir.
type Resolver struct {
	baseDir string
}

// NewResolver creates a read-only asset resolver rooted at baseDir.
func NewResolver(baseDir string) *Resolver {
	return &Resolver{baseDir: filepath.Clean(baseDir)}
}

// BaseDir returns the configured creative base directory.
func (r *Resolver) BaseDir() string {
	return r.baseDir
}

// BaseExists checks whether the configured creative base directory is present.
func (r *Resolver) BaseExists() bool {
	if r.baseDir == "" {
		return false
	}
	info, err := os.Stat(r.baseDir)
	return err == nil && info.IsDir()
}

// ResolvePath returns the validated filesystem path for a mapped asset key.
func (r *Resolver) ResolvePath(key string) (string, bool) {
	rel, ok := CreativeAssetsMap[strings.TrimPrefix(key, "/")]
	if !ok || !r.BaseExists() {
		return "", false
	}
	full := filepath.Clean(filepath.Join(r.baseDir, rel))
	if !strings.HasPrefix(full, r.baseDir+string(filepath.Separator)) {
		return "", false
	}
	if _, err := os.Stat(full); err != nil {
		return "", false
	}
	return full, true
}

// ServeHTTP serves mapped creative assets under /creative-assets/ and delegates
// all other paths to next if non-nil.
func (r *Resolver) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		const prefix = "/creative-assets/"
		if !strings.HasPrefix(req.URL.Path, prefix) {
			if next != nil {
				next.ServeHTTP(w, req)
				return
			}
			http.NotFound(w, req)
			return
		}
		key := strings.TrimPrefix(req.URL.Path, prefix)
		full, ok := r.ResolvePath(key)
		if !ok {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, req, full)
	})
}
