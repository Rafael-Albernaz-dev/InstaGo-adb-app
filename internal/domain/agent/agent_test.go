package agent

import (
	"strings"
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
)

func sampleDocAndOffer() (document.Document, *library.DatedOffer) {
	doc := document.Document{
		ID:              "doc-1",
		Title:           "Combo Rural",
		Format:          document.FormatFeed,
		WorkspaceWidth:  600,
		WorkspaceHeight: 750,
		ExportWidth:     1080,
		ExportHeight:    1350,
		OfferID:         "offer-combo",
		Pages: []document.Page{
			{
				ID:    "p1",
				Index: 0,
				Title: "Página 1",
				Layers: []document.Layer{
					{ID: "bg", Kind: "image", Name: "Foto"},
					{ID: "txt", Kind: "text", Name: "Título", Text: "Combo 600 Mega R$ 99,99"},
				},
				Annotations: []document.AgentAnnotation{
					{ID: "a1", Kind: "rect", X: 20, Y: 40, Width: 200, Height: 180, Note: "Trocar foto por família na varanda"},
				},
			},
		},
	}
	offer := &library.DatedOffer{
		ID:              "offer-combo",
		Title:           "Combo 600 Mega + Fixo",
		PromoPriceCents: 9999,
		PromoPeriod:     "nos 3 primeiros meses",
		AfterPriceCents: 13999,
		InstallFeeCents: 15000,
	}
	return doc, offer
}

func TestResolveDeliveryMode(t *testing.T) {
	doc, _ := sampleDocAndOffer()
	if got := ResolveDeliveryMode("Gerar foto de família na varanda", ModeAuto, doc.Pages[0]); got != ModeBackgroundLayer {
		t.Fatalf("expected background_layer for scene request with editable texts, got %s", got)
	}
	if got := ResolveDeliveryMode("Refazer a arte inteira com outro layout", ModeAuto, doc.Pages[0]); got != ModeFullPiece {
		t.Fatalf("expected full_piece when explicitly requesting full banner edit, got %s", got)
	}

	singleImagePage := document.Page{
		ID:     "p-img",
		Layers: []document.Layer{{ID: "only", Kind: "image"}},
	}
	if got := ResolveDeliveryMode("Melhorar a iluminação", ModeAuto, singleImagePage); got != ModeFullPiece {
		t.Fatalf("expected full_piece on single flattened image page, got %s", got)
	}
}

func TestAnalyzePreflightRejectsEmptyAndFlagsPriceConflict(t *testing.T) {
	doc, offer := sampleDocAndOffer()
	docNoMarks := doc
	docNoMarks.Pages = []document.Page{{ID: "p1", Layers: doc.Pages[0].Layers}}

	emptyRes := AnalyzePreflight("   ", ModeAuto, docNoMarks, 0, offer, document.OfferCheckResult{Status: "ok"})
	if emptyRes.Valid {
		t.Fatal("expected empty prompt without annotations to be invalid")
	}

	conflictRes := AnalyzePreflight(
		"Colocar preço R$ 79,90 na arte inteira",
		ModeFullPiece,
		doc,
		0,
		offer,
		document.OfferCheckResult{Status: "ok"},
	)
	if !conflictRes.Valid || !conflictRes.NeedsConfirmation {
		t.Fatalf("expected NeedsConfirmation=true for divergent prompt price, got %+v", conflictRes)
	}

	okRes := AnalyzePreflight(
		"Trocar foto de fundo por família rural feliz na varanda",
		ModeAuto,
		doc,
		0,
		offer,
		document.OfferCheckResult{Status: "ok"},
	)
	if !okRes.Valid || okRes.NeedsConfirmation || okRes.ResolvedMode != ModeBackgroundLayer {
		t.Fatalf("unexpected preflight result for clear scene request: %+v", okRes)
	}
}

func TestBuildGenerationPromptIncludesContextAndAnnotations(t *testing.T) {
	doc, offer := sampleDocAndOffer()
	prompt := BuildGenerationPrompt(PromptBuildInput{
		Document:         doc,
		Page:             doc.Pages[0],
		Offer:            offer,
		Mode:             ModeBackgroundLayer,
		Prompt:           "Gerar cena realista na varanda",
		Annotations:      doc.Pages[0].Annotations,
		SnapshotAttached: true,
		TargetOutputPath: "/tmp/out.png",
	})
	for _, needle := range []string{
		"APENAS FOTO/CENA DE FUNDO EDITÁVEL",
		"NÃO escreva textos",
		"Trocar foto por família na varanda",
		"/tmp/out.png",
		"R$ 99,99",
	} {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("expected prompt to contain %q", needle)
		}
	}
}
