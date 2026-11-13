package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/seed"
)

func TestStoreDraftApproveBranchAndRestore(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "versions.db")

	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	doc := seed.NewCarouselStarterTemplate()
	doc.ID = "piece-carousel-01"
	doc.Kind = "piece"

	// 1. Initial SaveDraft creates revision 1 cleanly
	draft1, err := store.SaveDraft(ctx, doc)
	if err != nil {
		t.Fatalf("initial save draft: %v", err)
	}
	if draft1.Revision != 1 || draft1.Status != "draft" {
		t.Fatalf("unexpected draft1 state: rev=%d status=%s", draft1.Revision, draft1.Status)
	}

	// 2. Subsequent SaveDraft updates working payload without incrementing revision
	draft1.Title = "Carrossel Editado"
	draft2, err := store.SaveDraft(ctx, draft1)
	if err != nil {
		t.Fatalf("second save draft: %v", err)
	}
	if draft2.Revision != 1 {
		t.Fatalf("expected autosave draft to keep revision 1, got %d", draft2.Revision)
	}

	revs, err := store.ListRevisions(ctx, doc.ID)
	if err != nil || len(revs) != 1 {
		t.Fatalf("expected 1 revision after autosave, got %d (%v)", len(revs), err)
	}

	// 3. ApproveDocument snapshots revision 2 as approved
	approved, err := store.ApproveDocument(ctx, draft2, "aprovado para teste")
	if err != nil {
		t.Fatalf("approve document: %v", err)
	}
	if approved.Revision != 2 || approved.ApprovedRevision != 2 || approved.Status != "approved" {
		t.Fatalf("unexpected approved state: rev=%d approvedRev=%d status=%s", approved.Revision, approved.ApprovedRevision, approved.Status)
	}

	// 4. Direct edits to approved document must be rejected
	approved.Title = "Tentativa direta"
	if _, err := store.SaveDraft(ctx, approved); !errors.Is(err, sqlite.ErrApprovedDocumentImmutable) {
		t.Fatalf("expected ErrApprovedDocumentImmutable on SaveDraft, got %v", err)
	}
	if _, err := store.SaveDocument(ctx, approved, "direto"); !errors.Is(err, sqlite.ErrApprovedDocumentImmutable) {
		t.Fatalf("expected ErrApprovedDocumentImmutable on SaveDocument, got %v", err)
	}

	// 5. Unlock via StartDraftFromApproved creates rev 3 in draft while keeping rev 2 approved
	unlocked, err := store.StartDraftFromApproved(ctx, doc.ID)
	if err != nil {
		t.Fatalf("start draft from approved: %v", err)
	}
	if unlocked.Status != "draft" || unlocked.Revision != 3 || unlocked.ApprovedRevision != 2 {
		t.Fatalf("unexpected unlocked state: status=%s rev=%d approvedRev=%d", unlocked.Status, unlocked.Revision, unlocked.ApprovedRevision)
	}

	rev2, err := store.GetRevision(ctx, doc.ID, 2)
	if err != nil || rev2.Status != "approved" || rev2.Document.Title != "Carrossel Editado" {
		t.Fatalf("approved snapshot rev 2 corrupted: %+v (%v)", rev2.RevisionSummary, err)
	}

	// 6. Save explicit new version (rev 4) then restore rev 2 into draft rev 5
	unlocked.Title = "Nova ideia"
	saved4, err := store.SaveDocument(ctx, unlocked, "nova versao")
	if err != nil || saved4.Revision != 4 {
		t.Fatalf("expected rev 4, got %d (%v)", saved4.Revision, err)
	}
	restored, err := store.RestoreRevisionAsDraft(ctx, doc.ID, 2)
	if err != nil {
		t.Fatalf("restore rev 2: %v", err)
	}
	if restored.Revision != 5 || restored.Status != "draft" || restored.Title != "Carrossel Editado" {
		t.Fatalf("unexpected restored state: rev=%d status=%s title=%s", restored.Revision, restored.Status, restored.Title)
	}
}
