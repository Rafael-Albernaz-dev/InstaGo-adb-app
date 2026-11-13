package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/codex"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	domain "github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/agent"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
)

// TestRealCodexCLIImageGenE2E runs a real end-to-end call through Go -> Codex CLI -> image_gen -> Library -> Document
// only when INSTAGO_REAL_CODEX_E2E=1 is explicitly set, keeping regular unit test suites offline and quota-free.
func TestRealCodexCLIImageGenE2E(t *testing.T) {
	if os.Getenv("INSTAGO_REAL_CODEX_E2E") != "1" {
		t.Skip("set INSTAGO_REAL_CODEX_E2E=1 to execute real Codex CLI image generation E2E test")
	}

	ctx := context.Background()
	tmp := t.TempDir()
	store, err := sqlite.Open(ctx, filepath.Join(tmp, "e2e.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.SaveLibraryImport(ctx, library.CompanyProfile{
		ID:              library.DefaultCompanyID,
		Brand:           "Rural Conecta",
		InstagramHandle: "@ruralconectamg",
		Region:          "Minas Gerais",
		ClientAudience:  "Moradores de zona rural",
		PhoneDisplay:    "(38) 99740-2599",
		SiteURL:         "https://ruralconectamg.com.br/",
	}, nil, nil, nil, library.ImportSummary{
		RunID:     "run-e2e",
		CompanyID: library.DefaultCompanyID,
		SourceDir: tmp,
		Status:    "completed",
		CreatedAt: "2026-11-13T00:00:00Z",
	}); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	doc, err := store.SaveDocument(ctx, document.Document{
		ID:              "e2e-carousel",
		SchemaVersion:   1,
		Kind:            "piece",
		PieceType:       "carousel",
		Title:           "Teste Real Codex",
		CompanyID:       library.DefaultCompanyID,
		Format:          document.FormatFeed,
		WorkspaceWidth:  600,
		WorkspaceHeight: 750,
		ExportWidth:     1080,
		ExportHeight:    1350,
		ExportScale:     1.8,
		Revision:        1,
		Status:          "draft",
		Pages: []document.Page{
			{
				ID:         "p1",
				Index:      0,
				Title:      "Capa",
				Background: "#F6F7EE",
				Layers: []document.Layer{
					{ID: "txt1", Kind: "text", Name: "Chamada", Text: "Internet Rural de Verdade", ZIndex: 2},
				},
			},
			{
				ID:         "p2",
				Index:      1,
				Title:      "Página 2 intacta",
				Background: "#FFFFFF",
				Layers: []document.Layer{
					{ID: "txt2", Kind: "text", Name: "Preservado", Text: "Mantido intacto", ZIndex: 1},
				},
			},
		},
	}, "seed e2e")
	if err != nil {
		t.Fatalf("save doc: %v", err)
	}

	runner := codex.NewRunner("")
	health := runner.CheckHealth(ctx)
	if !health.Available || !health.Authenticated {
		t.Fatalf("codex health not ready: %+v", health)
	}
	t.Logf("Codex CLI ready: version=%q auth=%q", health.Version, health.AuthStatusText)

	svc := NewService(store, runner, filepath.Join(tmp, "agent-jobs"), filepath.Join(tmp, "library"), "", nil)
	task, err := svc.StartTask(ctx, domain.StartTaskRequest{
		DocumentID: doc.ID,
		PageIndex:  0,
		Mode:       domain.ModeBackgroundLayer,
		Prompt:     "Gere uma fotografia fotorrealista simples de uma varanda de sítio em Minas Gerais ao pôr do sol com céu limpo, sem textos ou logos.",
	})
	if err != nil {
		t.Fatalf("start task: %v", err)
	}

	deadline := time.Now().Add(4 * time.Minute)
	var final domain.TaskDetail
	for time.Now().Before(deadline) {
		det, getErr := svc.GetTask(ctx, task.ID)
		if getErr == nil && det.Task.Status != domain.StatusRunning {
			final = det
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if final.Task.ID == "" {
		t.Fatalf("timed out waiting for real Codex CLI task %s", task.ID)
	}
	for _, l := range final.Logs {
		t.Logf("[log:%s/%s] %s", l.Stream, l.EventType, l.Text)
	}
	if final.Task.Status != domain.StatusCompleted {
		t.Fatalf("expected completed real Codex CLI task, got status=%s code=%s err=%s", final.Task.Status, final.Task.ErrorCode, final.Task.ErrorMessage)
	}
	if final.Task.OutputAssetURL == "" || final.Task.ResultRevision <= doc.Revision {
		t.Fatalf("missing output asset or revision bump: %+v", final.Task)
	}

	reloaded, err := store.GetDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("reload doc: %v", err)
	}
	if reloaded.Pages[1].Layers[0].Text != "Mantido intacto" {
		t.Fatalf("expected page 2 preserved, got %q", reloaded.Pages[1].Layers[0].Text)
	}
	t.Logf("E2E SUCCESS: rev=%d asset=%s path=%s", final.Task.ResultRevision, final.Task.OutputAssetURL, final.Task.OutputImagePath)
}
