package document_test

import (
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/seed"
)

func TestSeedV5ComboDocumentIsValid(t *testing.T) {
	doc := seed.NewV5ComboPage2Document()
	if err := doc.Validate(); err != nil {
		t.Fatalf("expected representative V5 seed to validate, got: %v", err)
	}
	if doc.ExportWidth != 1080 || doc.ExportHeight != 1350 {
		t.Fatalf("unexpected export size: %dx%d", doc.ExportWidth, doc.ExportHeight)
	}
	if len(doc.Pages) != 1 || len(doc.Pages[0].Layers) < 10 {
		t.Fatalf("expected 1 page with structured layers, got %d pages", len(doc.Pages))
	}
}

func TestDocumentValidateRejectsInvalidDimensionsAndDuplicates(t *testing.T) {
	doc := seed.NewV5ComboPage2Document()
	doc.ExportHeight = 1920 // mismatch for feed
	if err := doc.Validate(); err == nil {
		t.Fatal("expected dimension mismatch error")
	}

	doc = seed.NewV5ComboPage2Document()
	doc.Pages[0].Layers = append(doc.Pages[0].Layers, doc.Pages[0].Layers[0])
	if err := doc.Validate(); err == nil {
		t.Fatal("expected duplicate layer id error")
	}
}

func TestPresetDimensions(t *testing.T) {
	wsW, wsH, expW, expH, scale, err := document.PresetDimensions(document.FormatStory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wsW != 600 || expW != 1080 || expH != 1920 || scale != 1.8 || wsH < 1066 {
		t.Fatalf("unexpected story preset: ws=%vx%v exp=%dx%d scale=%v", wsW, wsH, expW, expH, scale)
	}
}
