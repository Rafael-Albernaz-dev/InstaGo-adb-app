package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/seed"
)

func TestStoreSaveReopenAndRevisionIncrement(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "instago.db")

	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	doc := seed.NewV5ComboPage2Document()
	saved1, err := store.SaveDocument(ctx, doc, "initial seed")
	if err != nil {
		t.Fatalf("save initial: %v", err)
	}
	if saved1.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", saved1.Revision)
	}

	saved1.Pages[0].Layers[0].X = 99
	saved2, err := store.SaveDocument(ctx, saved1, "moved logo card")
	if err != nil {
		t.Fatalf("save update: %v", err)
	}
	if saved2.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", saved2.Revision)
	}

	_ = store.Close()

	reopened, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer func() { _ = reopened.Close() }()

	loaded, err := reopened.GetDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if loaded.Revision != 2 || loaded.Pages[0].Layers[0].X != 99 {
		t.Fatalf("state mismatch after reopen: rev=%d x=%v", loaded.Revision, loaded.Pages[0].Layers[0].X)
	}
}
