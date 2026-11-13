package document_test

import (
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/seed"
)

func TestCheckOfferConsistencyOnRepresentativeV5Combo(t *testing.T) {
	doc := seed.NewV5ComboPage2Document()
	offer := &library.DatedOffer{
		ID:              "offer-combo-combo-completo",
		Title:           "Combo Completo",
		PromoPriceCents: 9999,
		PromoPeriod:     "Nos 3 primeiros meses",
		AfterPriceCents: 19990,
		InstallFeeCents: 24990,
		CheckedAt:       "2026-11-11",
		ValidUntil:      "2026-11-30",
	}

	res := document.CheckOfferConsistency(doc, offer)
	if res.Status != "ok" {
		t.Fatalf("expected status 'ok' for V5 combo baseline, got %q (%v)", res.Status, res.Messages)
	}
}

func TestCheckOfferConsistencyDetectsPriceConflictAndWarning(t *testing.T) {
	doc := seed.NewV5ComboPage2Document()
	offer := &library.DatedOffer{
		ID:              "offer-combo-combo-completo",
		Title:           "Combo Completo",
		PromoPriceCents: 9999,
		PromoPeriod:     "Nos 3 primeiros meses",
		AfterPriceCents: 19990,
		InstallFeeCents: 24990,
		CheckedAt:       "2026-11-11",
	}

	// Mutate promo price from 99,99 to 79,90 -> conflict
	for i := range doc.Pages[0].Layers {
		if doc.Pages[0].Layers[i].ID == "promo-price-value" {
			doc.Pages[0].Layers[i].Text = "79,90"
		}
	}
	res := document.CheckOfferConsistency(doc, offer)
	if res.Status != "conflict" {
		t.Fatalf("expected status 'conflict' after price change, got %q (%v)", res.Status, res.Messages)
	}

	// Restore promo price but remove footer conditions -> warning (missing after price)
	doc = seed.NewV5ComboPage2Document()
	for i := range doc.Pages[0].Layers {
		if doc.Pages[0].Layers[i].ID == "footer-conditions" {
			doc.Pages[0].Layers[i].Visible = false
		}
	}
	resWarn := document.CheckOfferConsistency(doc, offer)
	if resWarn.Status != "warning" {
		t.Fatalf("expected status 'warning' when after-price condition is hidden, got %q (%v)", resWarn.Status, resWarn.Messages)
	}
}
