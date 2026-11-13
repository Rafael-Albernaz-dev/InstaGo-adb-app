package assets_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/assets"
)

func TestResolverServesMappedAssetAndRejectsUnmapped(t *testing.T) {
	base := t.TempDir()
	fontPath := filepath.Join(base, "02-identidade", "fontes", "sora.woff2")
	if err := os.MkdirAll(filepath.Dir(fontPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fontPath, []byte("woff2-test-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := assets.NewResolver(base)
	handler := res.Middleware(http.NotFoundHandler())

	okReq := httptest.NewRequest(http.MethodGet, "/creative-assets/fonts/sora.woff2", nil)
	okRec := httptest.NewRecorder()
	handler.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK || okRec.Body.String() != "woff2-test-bytes" {
		t.Fatalf("expected 200 with font bytes, got %d %q", okRec.Code, okRec.Body.String())
	}

	badReq := httptest.NewRequest(http.MethodGet, "/creative-assets/../../etc/passwd", nil)
	badRec := httptest.NewRecorder()
	handler.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmapped path, got %d", badRec.Code)
	}

	libDir := t.TempDir()
	imgPath := filepath.Join(libDir, "artes-unicas", "timeline", "01-marca-01.jpg")
	if err := os.MkdirAll(filepath.Dir(imgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imgPath, []byte("jpg-test"), 0o644); err != nil {
		t.Fatal(err)
	}
	res.SetLibraryDir(libDir)

	libOk := httptest.NewRequest(http.MethodGet, "/library-assets/artes-unicas/timeline/01-marca-01.jpg", nil)
	libOkRec := httptest.NewRecorder()
	handler.ServeHTTP(libOkRec, libOk)
	if libOkRec.Code != http.StatusOK || libOkRec.Body.String() != "jpg-test" {
		t.Fatalf("expected 200 from library-assets, got %d", libOkRec.Code)
	}

	libBad := httptest.NewRequest(http.MethodGet, "/library-assets/../../etc/passwd", nil)
	libBadRec := httptest.NewRecorder()
	handler.ServeHTTP(libBadRec, libBad)
	if libBadRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for traversal in library-assets, got %d", libBadRec.Code)
	}
}
