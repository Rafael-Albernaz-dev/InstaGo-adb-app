package wailsapp

import (
	"context"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
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

func (b *EditorBridge) GetDocument(id string) (document.Document, error) {
	return b.svc.GetDocument(context.Background(), id)
}

func (b *EditorBridge) SaveDocument(doc document.Document, note string) (document.Document, error) {
	return b.svc.SaveDocument(context.Background(), doc, note)
}

func (b *EditorBridge) ResetSeedDocument() (document.Document, error) {
	return b.svc.ResetSeedDocument(context.Background())
}

func (b *EditorBridge) ListExports(documentID string) ([]sqlite.ExportRecord, error) {
	return b.svc.ListExports(context.Background(), documentID)
}

func (b *EditorBridge) ExportAsset(req editor.ExportRequest) (sqlite.ExportRecord, error) {
	return b.svc.ExportAsset(context.Background(), req)
}
