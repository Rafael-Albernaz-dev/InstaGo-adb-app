package library_test

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	domainlib "github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/usecase/library"
)

func createTestJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 61, G: 106, B: 47, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}
}

func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func buildFixtureCreativeBase(t *testing.T) string {
	t.Helper()
	base := t.TempDir()

	writeFile := func(rel, content string) {
		p := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("01-contexto/DADOS-DA-MARCA.json", `{
		"brand": "Rural Conecta",
		"instagramHandle": "ruralconectamg",
		"instagramUrl": "https://www.instagram.com/ruralconectamg/",
		"site": "https://ruralconectamg.com.br/",
		"camerasPage": "https://ruralconectamg.com.br/cameras/",
		"phoneDisplay": "(38) 99740-2599",
		"phoneE164": "+5538997402599",
		"whatsappLink": "https://wa.me/5538997402599",
		"region": "Paracatu-MG e região",
		"approvedVisualVersion": "V5",
		"approvalDate": "2026-11-11",
		"clientAudience": "Pessoas e famílias do campo",
		"profileName": "Rural Conecta | Internet Rural Paracatu-MG",
		"bioFile": "04-textos/bio/bio-instagram.txt",
		"primaryPalette": "02-identidade/paleta.json"
	}`)
	writeFile("02-identidade/paleta.json", `{"verde_principal":"#3D6A2F"}`)
	writeFile("04-textos/bio/bio-instagram.txt", "Internet rural em Paracatu-MG")
	writeFile("02-identidade/fontes/sora.woff2", "woff2-font-bytes")
	writeFile("02-identidade/GUIA-DE-USO.md", "# Guia de uso")
	writeFile("06-mockup-aprovado/v5/index.html", "<html><body>V5</body></html>")
	writeFile("08-prompts-e-modelos/MODELO-DE-PROMPT-FOTOGRAFICO.md", "# Prompt")
	writeFile("AGENTS.md", "# Brand rules")
	writeFile(".agents/skills/rural-conecta-instagram/SKILL.md", "---\nname: rural-conecta-instagram\n---\n")

	writeFile("01-contexto/OFERTAS-APROVADAS-2026-11-11.md", strings.Join([]string{
		"| Fibra | Depois da promoção / mês |",
		"|---|---:|",
		"| 100 megas | R$ 189,00 |",
		"",
		"| Combo | Velocidades fibra / rádio | Serviços descritos na base | Depois / mês |",
		"|---|---|---|---:|",
		"| Completo | 100 / 25 megas | 1 câmera + alarme + RuralTV + HBO | R$ 199,90 |",
		"",
		"| Serviço | Mensalidade |",
		"|---|---:|",
		"| Somente câmeras | R$ 39,90 |",
	}, "\n"))

	writeFile("04-textos/legendas/02-combo.txt", "BLACK FRIDAY Rural Conecta!")
	writeFile("04-textos/textos-e-composicoes-por-peca.json", `{
		"posts": [
			{"id":"combo","title":"Combo Completo na Black Friday","caption":"Legenda combo","pages":[{"title":"Capa"},{"title":"100 megas na Black Friday"}]}
		],
		"highlights": [
			{"id":"black","name":"Black Friday","stories":[{"title":"Black Friday da Rural","cta":"Fale com a Rural"}]}
		]
	}`)

	createTestJPEG(t, filepath.Join(base, "03-banners-prontos/timeline/02-combo/02-combo-01.jpg"), 400, 500)
	createTestJPEG(t, filepath.Join(base, "03-banners-prontos/timeline/02-combo/02-combo-02.jpg"), 480, 600)
	createTestJPEG(t, filepath.Join(base, "03-banners-prontos/stories-destaques/01-black-friday/01-black-01.jpg"), 400, 710)
	createTestJPEG(t, filepath.Join(base, "03-banners-prontos/capas-destaques/01-black.jpg"), 300, 300)
	createTestJPEG(t, filepath.Join(base, "03-banners-prontos/capas-timeline-copias/02-combo-01.jpg"), 400, 500)

	createTestPNG(t, filepath.Join(base, "02-identidade/logos/rural-conecta/logo.png"), 200, 80)
	createTestPNG(t, filepath.Join(base, "05-biblioteca/imagens/timeline/p-combo-casa.png"), 420, 300)
	createTestJPEG(t, filepath.Join(base, "09-historico-e-referencias/assets-fora-da-v5/old.jpg"), 300, 200)

	writeFile("05-biblioteca/catalogo-assets.json", `[
		{"originalFile":"p-combo-casa.png","file":"05-biblioteca/imagens/timeline/p-combo-casa.png","category":"cena_timeline","usedAsSceneIn":["02-combo-02"]},
		{"originalFile":"old.jpg","file":"09-historico-e-referencias/assets-fora-da-v5/old.jpg","category":"historico_fora_da_composicao_final","usedAsSceneIn":[]}
	]`)

	writeFile("07-agente-publicacao/execucao-2026-11-11/publicacao-v5-com-links-reais.json", `{
		"account":"ruralconectamg",
		"date":"2026-11-11",
		"executionDate":"2026-11-11",
		"posts":[
			{"id":"combo","positionInApprovedGrid":2,"title":"Combo Completo na Black Friday","pin":true,"status":"publicado_confirmado","instagramUrl":"https://www.instagram.com/ruralconectamg/p/DfnWpF9n9Ls/","captionFile":"04-textos/legendas/02-combo.txt","images":[{"page":1,"file":"03-banners-prontos/timeline/02-combo/02-combo-01.jpg"}]}
		],
		"highlights":[
			{"id":"black","name":"Black Friday","positionInApprovedHighlights":1,"status":"criado_confirmado_usuario_finalizou_capas","stories":[{"page":1,"title":"Black Friday da Rural","file":"03-banners-prontos/stories-destaques/01-black-friday/01-black-01.jpg"}]}
		]
	}`)

	return base
}

func TestImportIntegrityProfileAndBackup(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	baseDir := buildFixtureCreativeBase(t)

	store, err := sqlite.Open(ctx, filepath.Join(dataDir, "instago.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	svc := library.NewService(
		store,
		filepath.Join(dataDir, "library"),
		filepath.Join(dataDir, "backups"),
		filepath.Join(dataDir, "exports"),
		baseDir,
		"",
		"0.1.0-test",
	)

	var openedDirs []string
	svc.SetOpenDirFunc(func(dir string) error {
		openedDirs = append(openedDirs, dir)
		return nil
	})

	overview, err := svc.GetOverview(ctx, domainlib.LibraryFilter{})
	if err != nil {
		t.Fatalf("GetOverview failed: %v", err)
	}
	if overview.Summary == nil {
		t.Fatal("expected import summary")
	}
	if overview.Summary.UniqueArtsCount != 4 {
		t.Fatalf("expected 4 unique arts in fixture, got %d", overview.Summary.UniqueArtsCount)
	}
	if overview.Summary.AuxiliaryCoversCount != 1 {
		t.Fatalf("expected 1 auxiliary cover in fixture, got %d", overview.Summary.AuxiliaryCoversCount)
	}
	if overview.Summary.EditableModelsCount != 1 {
		t.Fatalf("expected 1 editable model (02-combo-02), got %d", overview.Summary.EditableModelsCount)
	}
	if !overview.Summary.SkillLoaded {
		t.Fatal("expected Rural Conecta skill to be detected")
	}
	if len(overview.Offers) != 3 {
		t.Fatalf("expected 3 parsed offers, got %d", len(overview.Offers))
	}
	for _, o := range overview.Offers {
		if !o.ReadOnly {
			t.Fatalf("offer %s must be read-only in R1", o.ID)
		}
	}
	if len(overview.History) != 2 {
		t.Fatalf("expected 2 history items (1 post + 1 highlight), got %d", len(overview.History))
	}
	for _, h := range overview.History {
		if !h.ReadOnly {
			t.Fatalf("history item %s must be read-only in R1", h.ID)
		}
	}

	// Verify thumbnail generated for >360px unique art and editable flag on 02-combo-02.
	foundCombo02 := false
	for _, a := range overview.Assets {
		if a.FileName == "02-combo-02.jpg" && a.Classification == "arte_unica" {
			foundCombo02 = true
			if !a.IsEditableModel || a.LinkedDocumentID == "" {
				t.Fatalf("expected 02-combo-02 to be linked to seed editable document")
			}
			if a.ThumbRelPath == "" {
				t.Fatalf("expected thumbnail for 480px wide 02-combo-02.jpg")
			}
		}
	}
	if !foundCombo02 {
		t.Fatal("missing 02-combo-02.jpg in imported assets")
	}

	// Verify integrity passes before tampering.
	rep, err := svc.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rep.MissingCount != 0 || rep.CorruptedCount != 0 || rep.OKCount != rep.TotalAssets {
		t.Fatalf("expected clean integrity report, got %+v", rep)
	}

	// Profile update persists across re-import.
	prof := overview.Profile
	prof.Region = "Paracatu e Unaí-MG"
	if _, err := svc.UpdateProfile(ctx, prof); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportBase(ctx, baseDir); err != nil {
		t.Fatal(err)
	}
	afterReimport, err := svc.GetOverview(ctx, domainlib.LibraryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if afterReimport.Profile.Region != "Paracatu e Unaí-MG" {
		t.Fatalf("expected custom profile region preserved on re-import, got %q", afterReimport.Profile.Region)
	}

	// OpenFolder validates scope.
	if _, err := svc.OpenFolder(ctx, "asset_dir", "art-feed-02-combo-02"); err != nil {
		t.Fatalf("OpenFolder failed: %v", err)
	}
	if len(openedDirs) != 1 || !strings.HasPrefix(openedDirs[0], filepath.Join(dataDir, "library")) {
		t.Fatalf("unexpected opened folder: %v", openedDirs)
	}
	if _, err := svc.OpenFolder(ctx, "arbitrary_shell", ""); err == nil {
		t.Fatal("expected unsupported scope error")
	}

	// Create portable .zip backup and verify contents + manifest hashes.
	backup, err := svc.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}
	zr, err := zip.OpenReader(backup.FilePath)
	if err != nil {
		t.Fatalf("open backup zip: %v", err)
	}
	defer func() { _ = zr.Close() }()

	zipEntries := make(map[string]*zip.File)
	for _, f := range zr.File {
		zipEntries[f.Name] = f
	}
	if _, ok := zipEntries["database/instago.db"]; !ok {
		t.Fatal("missing database/instago.db in backup zip")
	}
	manFile, ok := zipEntries["manifest.json"]
	if !ok {
		t.Fatal("missing manifest.json in backup zip")
	}
	mr, err := manFile.Open()
	if err != nil {
		t.Fatal(err)
	}
	manBytes, _ := io.ReadAll(mr)
	_ = mr.Close()

	var manifest struct {
		FilesSHA256 map[string]string `json:"filesSha256"`
	}
	if err := json.Unmarshal(manBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	for name, expectedHash := range manifest.FilesSHA256 {
		zf, exists := zipEntries[name]
		if !exists {
			t.Fatalf("manifest entry %s missing in zip", name)
		}
		rc, err := zf.Open()
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.New()
		_, _ = io.Copy(h, rc)
		_ = rc.Close()
		gotHash := hex.EncodeToString(h.Sum(nil))
		if gotHash != expectedHash {
			t.Fatalf("hash mismatch in backup for %s: got %s want %s", name, gotHash, expectedHash)
		}
	}

	// Tamper one managed file and verify integrity catches it.
	tampered := filepath.Join(dataDir, "library", "artes-unicas", "capas-destaques", "01-black.jpg")
	if err := os.WriteFile(tampered, []byte("corrupted"), 0o644); err != nil {
		t.Fatal(err)
	}
	repAfter, err := svc.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if repAfter.CorruptedCount != 1 {
		t.Fatalf("expected 1 corrupted file detected, got %+v", repAfter)
	}
}
