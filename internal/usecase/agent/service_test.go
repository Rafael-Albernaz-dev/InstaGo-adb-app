package agent

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/codex"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	domain "github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/agent"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
)

type fakeRunner struct {
	health  domain.CLIHealth
	execFn  func(ctx context.Context, req codex.RunRequest, onLog codex.LogCallback) codex.RunOutcome
	mu      sync.Mutex
	calls   int
	lastReq codex.RunRequest
}

func (f *fakeRunner) CheckHealth(context.Context) domain.CLIHealth {
	return f.health
}

func (f *fakeRunner) RunExec(ctx context.Context, req codex.RunRequest, onLog codex.LogCallback) codex.RunOutcome {
	f.mu.Lock()
	f.calls++
	f.lastReq = req
	f.mu.Unlock()
	return f.execFn(ctx, req, onLog)
}

func writePNGFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 1080, 1350))
	img.Set(10, 10, color.RGBA{R: 61, G: 106, B: 47, A: 255})
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}

func setupAgentServiceTest(t *testing.T, runner CLIRunner) (*Service, *sqlite.Store, document.Document) {
	t.Helper()
	ctx := context.Background()
	tmp := t.TempDir()
	store, err := sqlite.Open(ctx, filepath.Join(tmp, "instago.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	profile := library.CompanyProfile{
		ID:              library.DefaultCompanyID,
		Brand:           "Rural Conecta",
		InstagramHandle: "@ruralconectamg",
		PhoneDisplay:    "(38) 99740-2599",
		SiteURL:         "https://ruralconectamg.com.br/",
	}
	if err := store.SaveLibraryImport(ctx, profile, nil, []library.DatedOffer{
		{
			ID:              "offer-combo",
			CompanyID:       library.DefaultCompanyID,
			Category:        "combo",
			Title:           "Combo 600 Mega",
			PromoPriceCents: 9999,
			PromoPeriod:     "3 primeiros meses",
			AfterPriceCents: 13999,
			InstallFeeCents: 15000,
			CheckedAt:       "2026-11-11",
			SourceRef:       "test",
			ReadOnly:        true,
		},
	}, nil, library.ImportSummary{
		RunID:     "run-test",
		CompanyID: library.DefaultCompanyID,
		SourceDir: tmp,
		Status:    "completed",
		CreatedAt: "2026-11-13T00:00:00Z",
	}); err != nil {
		t.Fatalf("seed offer: %v", err)
	}

	doc := document.Document{
		ID:              "doc-carousel",
		SchemaVersion:   1,
		Kind:            "piece",
		PieceType:       "carousel",
		Title:           "Carrossel Teste",
		CompanyID:       library.DefaultCompanyID,
		OfferID:         "offer-combo",
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
				ID:         "page-1",
				Index:      0,
				Title:      "Capa",
				Background: "#F6F7EE",
				Layers: []document.Layer{
					{ID: "bg-1", Kind: "image", Name: "Foto Fundo", AssetURL: "/creative-assets/photo/p-combo-casa.png", ObjectFit: "cover", ZIndex: 1},
					{ID: "title-1", Kind: "text", Name: "Título", Text: "Combo 600 Mega R$ 99,99", ZIndex: 2},
				},
			},
			{
				ID:         "page-2",
				Index:      1,
				Title:      "Interna preservada",
				Background: "#FFFFFF",
				Layers: []document.Layer{
					{ID: "p2-text", Kind: "text", Name: "Texto pág 2", Text: "Página 2 intacta", ZIndex: 1},
				},
			},
		},
	}
	saved, err := store.SaveDocument(ctx, doc, "seed carousel")
	if err != nil {
		t.Fatalf("save doc: %v", err)
	}

	svc := NewService(store, runner, filepath.Join(tmp, "agent-jobs"), filepath.Join(tmp, "library"), "", nil)
	return svc, store, saved
}

func waitForTaskTerminal(t *testing.T, svc *Service, taskID string) domain.Task {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		det, err := svc.GetTask(context.Background(), taskID)
		if err == nil && det.Task.Status != domain.StatusRunning {
			return det.Task
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for task %s", taskID)
	return domain.Task{}
}

func TestAgentServiceBackgroundEditPreservesTextAndOtherPages(t *testing.T) {
	runner := &fakeRunner{
		health: domain.CLIHealth{Available: true, Authenticated: true, Version: "0.167.2"},
		execFn: func(_ context.Context, req codex.RunRequest, onLog codex.LogCallback) codex.RunOutcome {
			onLog("agent", "agent_message", "Gerando cena rural realista...")
			writePNGFile(t, req.TargetOutputPath)
			return codex.RunOutcome{
				ThreadID:        "thr-bg-1",
				OutputImagePath: req.TargetOutputPath,
				LastMessage:     "Nova cena gerada.",
			}
		},
	}

	svc, store, doc := setupAgentServiceTest(t, runner)
	task, err := svc.StartTask(context.Background(), domain.StartTaskRequest{
		DocumentID: doc.ID,
		PageIndex:  0,
		Mode:       domain.ModeAuto,
		Prompt:     "Trocar a foto de fundo por família na varanda ao entardecer",
	})
	if err != nil {
		t.Fatalf("start task: %v", err)
	}

	done := waitForTaskTerminal(t, svc, task.ID)
	if done.Status != domain.StatusCompleted || done.ResultRevision <= doc.Revision || done.OutputAssetURL == "" {
		t.Fatalf("unexpected finished task: %+v", done)
	}

	updatedDoc, err := store.GetDocument(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("reload updated doc: %v", err)
	}
	if updatedDoc.Pages[0].Layers[0].AssetURL != done.OutputAssetURL {
		t.Fatalf("expected page 1 background updated to %s, got %s", done.OutputAssetURL, updatedDoc.Pages[0].Layers[0].AssetURL)
	}
	if updatedDoc.Pages[0].Layers[1].Text != "Combo 600 Mega R$ 99,99" {
		t.Fatalf("expected editable text layer preserved, got %q", updatedDoc.Pages[0].Layers[1].Text)
	}
	if updatedDoc.Pages[1].Layers[0].Text != "Página 2 intacta" {
		t.Fatalf("expected page 2 to remain untouched, got %q", updatedDoc.Pages[1].Layers[0].Text)
	}
}

func TestAgentServiceConflictRequiresConfirmationAndRecoversArtifactWithoutRepeat(t *testing.T) {
	runner := &fakeRunner{
		health: domain.CLIHealth{Available: true, Authenticated: true, Version: "0.167.2"},
		execFn: func(ctx context.Context, req codex.RunRequest, _ codex.LogCallback) codex.RunOutcome {
			writePNGFile(t, req.TargetOutputPath)
			return codex.RunOutcome{
				ThreadID:            "thr-rec",
				OutputImagePath:     req.TargetOutputPath,
				ArtifactRecoverable: true,
				Canceled:            true,
				ErrorCode:           "canceled",
				ErrorMessage:        "Interrompido antes de concluir",
			}
		},
	}

	svc, _, doc := setupAgentServiceTest(t, runner)
	pending, err := svc.StartTask(context.Background(), domain.StartTaskRequest{
		DocumentID: doc.ID,
		PageIndex:  0,
		Mode:       domain.ModeFullPiece,
		Prompt:     "Mudar o banner inteiro para R$ 49,90 por mês",
	})
	if err != nil {
		t.Fatalf("start conflict task: %v", err)
	}
	if pending.Status != domain.StatusNeedsConfirmation {
		t.Fatalf("expected needs_confirmation before running CLI, got %s", pending.Status)
	}
	if runner.calls != 0 {
		t.Fatalf("expected 0 CLI calls before confirmation, got %d", runner.calls)
	}

	confirmed, err := svc.ConfirmTask(context.Background(), pending.ID)
	if err != nil {
		t.Fatalf("confirm task: %v", err)
	}
	canceledTask := waitForTaskTerminal(t, svc, confirmed.ID)
	if canceledTask.Status != domain.StatusCanceled || !canceledTask.ArtifactRecoverable {
		t.Fatalf("expected canceled with recoverable artifact, got %+v", canceledTask)
	}

	// Resuming without follow-up must recover existing artifact WITHOUT calling CLI a second time!
	recovered, err := svc.ResumeTask(context.Background(), canceledTask.ID, "")
	if err != nil {
		t.Fatalf("resume with existing artifact: %v", err)
	}
	if recovered.Status != domain.StatusCompleted {
		t.Fatalf("expected recovered completed status, got %s", recovered.Status)
	}
	if runner.calls != 1 {
		t.Fatalf("expected ResumeTask not to re-invoke Codex when artifact already exists, calls=%d", runner.calls)
	}
}
