package editor_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/seed"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/usecase/editor"
)

func encodeSolidPNG(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 0xF6, G: 0xF7, B: 0xEE, A: 0xFF})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestServiceEnsureSeedSaveAndVerifiedExport(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	store, err := sqlite.Open(ctx, filepath.Join(tmp, "instago.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = store.Close() }()

	exportsDir := filepath.Join(tmp, "exports")
	svc := editor.NewService(store, tmp, func() bool { return true }, exportsDir, "0.0.0-test")

	doc, err := svc.EnsureSeedDocument(ctx)
	if err != nil {
		t.Fatalf("ensure seed: %v", err)
	}
	if doc.ID != seed.RepresentativeV5ComboID || doc.Revision != 1 {
		t.Fatalf("unexpected seed doc: id=%s rev=%d", doc.ID, doc.Revision)
	}

	// Reject wrong dimensions before writing export file.
	if _, err := svc.ExportAsset(ctx, editor.ExportRequest{
		DocumentID: doc.ID,
		Format:     "png",
		DataBase64: encodeSolidPNG(t, 600, 750),
	}); err == nil {
		t.Fatal("expected dimension mismatch error when exporting 600x750 instead of 1080x1350")
	}

	// Accept valid 1080x1350 PNG export and persist verified record.
	rec, err := svc.ExportAsset(ctx, editor.ExportRequest{
		DocumentID: doc.ID,
		Format:     "png",
		DataBase64: encodeSolidPNG(t, 1080, 1350),
		FileName:   "combo-02-r0-check",
	})
	if err != nil {
		t.Fatalf("export valid png: %v", err)
	}
	if rec.Width != 1080 || rec.Height != 1350 || rec.SHA256 == "" {
		t.Fatalf("unexpected export record: %+v", rec)
	}
	if _, err := os.Stat(rec.FilePath); err != nil {
		t.Fatalf("exported file missing at %s: %v", rec.FilePath, err)
	}

	history, err := svc.ListExports(ctx, doc.ID)
	if err != nil {
		t.Fatalf("list exports: %v", err)
	}
	if len(history) != 1 || history[0].SHA256 != rec.SHA256 {
		t.Fatalf("expected 1 persisted export record, got %+v", history)
	}
}
