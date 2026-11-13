package editor_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/usecase/editor"
)

func makePNGBase64(t *testing.T, w, h int, c color.RGBA) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestServiceR2StartersLibraryPieceOfferApproveAndBundleExport(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "instago.db")
	exportsDir := filepath.Join(tmp, "exports")
	libDir := filepath.Join(tmp, "library")

	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = store.Close() }()

	svc := editor.NewService(store, tmp, func() bool { return true }, exportsDir, "0.0.0-test")
	svc.SetLibraryDir(libDir)

	// 1. Starter documents (Feed seed + Story + Carousel)
	docs, err := svc.EnsureStarterDocuments(ctx)
	if err != nil {
		t.Fatalf("ensure starters: %v", err)
	}
	if len(docs) < 3 {
		t.Fatalf("expected at least 3 starter docs, got %d", len(docs))
	}

	// 2. Seed an offer in SQLite so offer checking works
	profile := library.CompanyProfile{
		ID:              library.DefaultCompanyID,
		Brand:           "Rural Conecta",
		InstagramHandle: "@ruralconectamg",
		UpdatedAt:       "2026-11-13T10:00:00Z",
	}
	offer := library.DatedOffer{
		ID:              "offer-fibra-100-megas",
		CompanyID:       library.DefaultCompanyID,
		Category:        "internet_fibra",
		Title:           "Fibra 100 megas",
		PromoPriceCents: 9999,
		PromoPeriod:     "Nos 3 primeiros meses",
		AfterPriceCents: 11990,
		InstallFeeCents: 24990,
		CheckedAt:       "2026-11-11",
		SourceRef:       "OFERTAS.md",
		ReadOnly:        true,
	}
	if err := store.SaveLibraryImport(ctx, profile, nil, []library.DatedOffer{offer}, nil, library.ImportSummary{
		RunID: "test-run", CompanyID: library.DefaultCompanyID, Status: "ok", CreatedAt: "2026-11-13T10:00:00Z",
	}); err != nil {
		t.Fatalf("seed offer: %v", err)
	}

	// 3. Create Carousel piece linked to offer
	carousel, err := svc.CreatePieceFromTemplate(ctx, "carousel", offer.ID, "Carrossel Teste Fibra")
	if err != nil {
		t.Fatalf("create carousel: %v", err)
	}
	if len(carousel.Pages) != 2 || carousel.PieceType != "carousel" {
		t.Fatalf("expected 2-page carousel, got %d pages (%s)", len(carousel.Pages), carousel.PieceType)
	}

	// Inject conflicting price into page 2 and ensure ApproveDocument blocks without confirmConflicts
	for i := range carousel.Pages[1].Layers {
		if carousel.Pages[1].Layers[i].Kind == "text" && strings.Contains(carousel.Pages[1].Layers[i].Text, "99,99") {
			carousel.Pages[1].Layers[i].Text = "Por R$ 79,90/mês nos 3 primeiros meses.\nDepois: R$ 119,90/mês · Instalação: R$ 249,90."
		}
	}
	resBlocked, err := svc.ApproveDocument(ctx, carousel, false)
	if err != nil {
		t.Fatalf("approve blocked call error: %v", err)
	}
	if resBlocked.Approved || resBlocked.OfferCheck.Status != "conflict" {
		t.Fatalf("expected conflict block on approve, got approved=%v status=%s", resBlocked.Approved, resBlocked.OfferCheck.Status)
	}

	// Confirm explicitly -> approves rev 2 cleanly
	resApproved, err := svc.ApproveDocument(ctx, carousel, true)
	if err != nil {
		t.Fatalf("approve confirm error: %v", err)
	}
	if !resApproved.Approved || resApproved.Document.Status != "approved" || resApproved.Document.ApprovedRevision != 2 {
		t.Fatalf("unexpected approved doc: %+v", resApproved)
	}

	// Unlock approved -> creates rev 3 draft while keeping rev 2 approved
	unlocked, err := svc.StartDraftFromApproved(ctx, carousel.ID)
	if err != nil || unlocked.Status != "draft" || unlocked.Revision != 3 {
		t.Fatalf("unlock failed: rev=%d status=%s err=%v", unlocked.Revision, unlocked.Status, err)
	}

	// 4. Import custom image & open from library
	imgB64 := makePNGBase64(t, 1080, 1350, color.RGBA{R: 61, G: 106, B: 47, A: 255})
	imported, err := svc.ImportEditorImage(ctx, "foto-fazenda.png", imgB64)
	if err != nil {
		t.Fatalf("import editor image: %v", err)
	}
	if !strings.HasPrefix(imported.AssetURL, "/library-assets/custom/") {
		t.Fatalf("unexpected assetUrl: %s", imported.AssetURL)
	}
	fromLib, err := svc.CreatePieceFromLibraryAsset(ctx, imported.ID, string(document.FormatFeed), "Arte da Biblioteca")
	if err != nil {
		t.Fatalf("create piece from library asset: %v", err)
	}
	if len(fromLib.Pages[0].Layers) != 1 || fromLib.Pages[0].Layers[0].AssetURL != imported.AssetURL {
		t.Fatalf("unexpected layer in library piece: %+v", fromLib.Pages[0].Layers)
	}

	// 5. Export 2-page carousel bundle (unlocked draft -> isDraftWarning must be true!)
	pagePNG := makePNGBase64(t, 1080, 1350, color.RGBA{R: 152, G: 180, B: 50, A: 255})
	bundle, err := svc.ExportBundle(ctx, editor.ExportBundleRequest{
		DocumentID: unlocked.ID,
		Format:     "png",
		Pages: []editor.ExportBundlePage{
			{PageIndex: 0, Title: "Capa", DataBase64: pagePNG},
			{PageIndex: 1, Title: "Página 2", DataBase64: pagePNG},
		},
	})
	if err != nil {
		t.Fatalf("export bundle: %v", err)
	}
	if bundle.PageCount != 2 || !bundle.IsDraftWarning {
		t.Fatalf("unexpected bundle metadata: %+v", bundle)
	}
	captionBytes, err := os.ReadFile(bundle.CaptionPath)
	if err != nil {
		t.Fatalf("read legenda.txt: %v", err)
	}
	if !strings.Contains(string(captionBytes), "RASCUNHO") || !strings.Contains(string(captionBytes), unlocked.NativeInstruction) {
		t.Fatalf("legenda.txt missing expected draft warning / instruction: %s", string(captionBytes))
	}
	manifestBytes, err := os.ReadFile(bundle.ManifestPath)
	if err != nil {
		t.Fatalf("read manifesto.json: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("parse manifesto.json: %v", err)
	}
	if pagesSlice, ok := manifest["pages"].([]any); !ok || len(pagesSlice) != 2 {
		t.Fatalf("unexpected manifesto pages: %+v", manifest["pages"])
	}
}
