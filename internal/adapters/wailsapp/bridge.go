package wailsapp

import (
	"context"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/usecase/editor"
)

// EditorBridge exposes editor use cases as bound Wails 3 service methods.
type EditorBridge struct {
	svc *editor.Service
}

// NewEditorBridge wraps editor.Service for Wails 3 IPC binding.
func NewEditorBridge(svc *editor.Service) *EditorBridge {
	return &EditorBridge{svc: svc}
}

func (b *EditorBridge) GetWorkspaceInfo() editor.WorkspaceInfo {
	return b.svc.GetWorkspaceInfo()
}

func (b *EditorBridge) EnsureSeedDocument() (document.Document, error) {
	return b.svc.EnsureSeedDocument(context.Background())
}

func (b *EditorBridge) EnsureStarterDocuments() ([]document.Document, error) {
	return b.svc.EnsureStarterDocuments(context.Background())
}

func (b *EditorBridge) ListDocuments() ([]document.Document, error) {
	return b.svc.ListDocuments(context.Background())
}

func (b *EditorBridge) GetDocument(id string) (document.Document, error) {
	return b.svc.GetDocument(context.Background(), id)
}

func (b *EditorBridge) CreatePieceFromTemplate(templateKind, offerID, title string) (document.Document, error) {
	return b.svc.CreatePieceFromTemplate(context.Background(), templateKind, offerID, title)
}

func (b *EditorBridge) CreatePieceFromLibraryAsset(assetID, format, title string) (document.Document, error) {
	return b.svc.CreatePieceFromLibraryAsset(context.Background(), assetID, format, title)
}

func (b *EditorBridge) ImportEditorImage(fileName, dataBase64 string) (library.LibraryAsset, error) {
	return b.svc.ImportEditorImage(context.Background(), fileName, dataBase64)
}

func (b *EditorBridge) SaveDraft(doc document.Document) (document.Document, error) {
	return b.svc.SaveDraft(context.Background(), doc)
}

func (b *EditorBridge) SaveDocument(doc document.Document, note string) (document.Document, error) {
	return b.svc.SaveDocument(context.Background(), doc, note)
}

func (b *EditorBridge) ResetSeedDocument() (document.Document, error) {
	return b.svc.ResetSeedDocument(context.Background())
}

func (b *EditorBridge) ListRevisions(documentID string) ([]sqlite.RevisionSummary, error) {
	return b.svc.ListRevisions(context.Background(), documentID)
}

func (b *EditorBridge) GetRevision(documentID string, revision int) (sqlite.RevisionRecord, error) {
	return b.svc.GetRevision(context.Background(), documentID, revision)
}

func (b *EditorBridge) RestoreRevision(documentID string, revision int) (document.Document, error) {
	return b.svc.RestoreRevision(context.Background(), documentID, revision)
}

func (b *EditorBridge) StartDraftFromApproved(documentID string) (document.Document, error) {
	return b.svc.StartDraftFromApproved(context.Background(), documentID)
}

func (b *EditorBridge) CheckOfferConsistency(doc document.Document) (document.OfferCheckResult, error) {
	return b.svc.CheckOfferConsistency(context.Background(), doc)
}

func (b *EditorBridge) ApproveDocument(doc document.Document, confirmConflicts bool) (editor.ApproveResult, error) {
	return b.svc.ApproveDocument(context.Background(), doc, confirmConflicts)
}

func (b *EditorBridge) ListExports(documentID string) ([]sqlite.ExportRecord, error) {
	return b.svc.ListExports(context.Background(), documentID)
}

func (b *EditorBridge) ExportAsset(req editor.ExportRequest) (sqlite.ExportRecord, error) {
	return b.svc.ExportAsset(context.Background(), req)
}

func (b *EditorBridge) ExportBundle(req editor.ExportBundleRequest) (sqlite.ExportBundleRecord, error) {
	return b.svc.ExportBundle(context.Background(), req)
}

func (b *EditorBridge) ListExportBundles(documentID string) ([]sqlite.ExportBundleRecord, error) {
	return b.svc.ListExportBundles(context.Background(), documentID)
}

func (b *EditorBridge) OpenExportPath(path string) error {
	return b.svc.OpenExportPath(path)
}
