package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/codex"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	domain "github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/agent"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
)

// CLIRunner abstracts Codex CLI invocation for unit tests and desktop runtime.
type CLIRunner interface {
	CheckHealth(ctx context.Context) domain.CLIHealth
	RunExec(ctx context.Context, req codex.RunRequest, onLog codex.LogCallback) codex.RunOutcome
}

// EventEmitter broadcasts asynchronous agent progress events to the desktop UI.
type EventEmitter func(eventName string, payload any)

// Service orchestrates contextual AI generation/editing via Codex CLI while keeping
// strict persistence, single active task control, and safe document versioning.
type Service struct {
	store      *sqlite.Store
	runner     CLIRunner
	jobsDir    string
	libraryDir string
	skillsDir  string
	emit       EventEmitter

	activeMu     sync.Mutex
	activeTaskID string
	activeCancel context.CancelFunc
	shuttingDown bool
}

// NewService constructs the R3 agent orchestration service and reconciles interrupted runs.
func NewService(
	store *sqlite.Store,
	runner CLIRunner,
	jobsDir string,
	libraryDir string,
	skillsDir string,
	emit EventEmitter,
) *Service {
	if runner == nil {
		runner = codex.NewRunner("")
	}
	svc := &Service{
		store:      store,
		runner:     runner,
		jobsDir:    filepath.Clean(jobsDir),
		libraryDir: filepath.Clean(libraryDir),
		skillsDir:  filepath.Clean(skillsDir),
		emit:       emit,
	}
	return svc
}

// ReconcileInterrupted recovers any tasks left active when a previous process exited unexpectedly.
func (s *Service) ReconcileInterrupted(ctx context.Context) (int64, error) {
	return s.store.RecoverInterruptedAgentTasks(ctx)
}

// Shutdown cancels any active Codex CLI execution gracefully before app termination.
func (s *Service) Shutdown() {
	s.activeMu.Lock()
	s.shuttingDown = true
	cancel := s.activeCancel
	s.activeMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// CheckHealth probes Codex CLI binary presence and ChatGPT login status.
func (s *Service) CheckHealth(ctx context.Context) domain.CLIHealth {
	return s.runner.CheckHealth(ctx)
}

// ListTasks returns recent agent tasks for a document (or all documents when empty).
func (s *Service) ListTasks(ctx context.Context, documentID string) ([]domain.Task, error) {
	return s.store.ListAgentTasks(ctx, documentID, 30)
}

// GetTask returns a task together with its persisted terminal streaming logs.
func (s *Service) GetTask(ctx context.Context, taskID string) (domain.TaskDetail, error) {
	task, err := s.store.GetAgentTask(ctx, taskID)
	if err != nil {
		return domain.TaskDetail{}, err
	}
	logs, err := s.store.ListAgentLogs(ctx, taskID)
	if err != nil {
		return domain.TaskDetail{}, err
	}
	return domain.TaskDetail{Task: task, Logs: logs}, nil
}

// StartTask validates the request, checks CLI health and commercial rules, persists the task,
// and starts Codex execution (or returns needs_confirmation without starting if flagged).
func (s *Service) StartTask(ctx context.Context, req domain.StartTaskRequest) (domain.Task, error) {
	s.activeMu.Lock()
	if s.activeTaskID != "" {
		s.activeMu.Unlock()
		return domain.Task{}, errors.New("já existe uma geração do agente em andamento; aguarde terminar ou clique em Cancelar")
	}
	s.activeMu.Unlock()

	health := s.runner.CheckHealth(ctx)
	if !health.Available || !health.Authenticated {
		msg := health.ErrorMessage
		if msg == "" {
			msg = "Codex CLI indisponível ou sem login ChatGPT ativo."
		}
		return domain.Task{}, errors.New(msg)
	}

	doc, err := s.store.GetDocument(ctx, req.DocumentID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("documento não encontrado: %w", err)
	}
	if req.PageIndex < 0 || req.PageIndex >= len(doc.Pages) {
		return domain.Task{}, fmt.Errorf("página %d fora do intervalo", req.PageIndex)
	}

	var offer *library.DatedOffer
	if strings.TrimSpace(doc.OfferID) != "" {
		offer, err = s.store.GetDatedOffer(ctx, doc.OfferID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, err
		}
	}
	offerCheck := document.CheckOfferConsistency(doc, offer)

	preflight := domain.AnalyzePreflight(req.Prompt, req.Mode, doc, req.PageIndex, offer, offerCheck)
	if !preflight.Valid {
		return domain.Task{}, errors.New(preflight.ErrorMessage)
	}

	taskID := "agent-" + uuid.NewString()[:12]
	workDir := filepath.Join(s.jobsDir, taskID)
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return domain.Task{}, fmt.Errorf("criar pasta de trabalho do agente: %w", err)
	}

	if strings.TrimSpace(req.SnapshotBase64) != "" {
		rawSnap, decErr := decodeBase64Payload(req.SnapshotBase64)
		if decErr != nil {
			return domain.Task{}, fmt.Errorf("captura da prancheta inválida: %w", decErr)
		}
		snapPath := filepath.Join(workDir, "input-page.png")
		if err := os.WriteFile(snapPath, rawSnap, 0o640); err != nil {
			return domain.Task{}, fmt.Errorf("salvar captura de referência: %w", err)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	status := domain.StatusRunning
	if preflight.NeedsConfirmation && !req.ConfirmBeforeRun {
		status = domain.StatusNeedsConfirmation
	}

	requestedMode := req.Mode
	if requestedMode == "" {
		requestedMode = domain.ModeAuto
	}

	task := domain.Task{
		ID:                  taskID,
		DocumentID:          doc.ID,
		PageIndex:           req.PageIndex,
		RequestedMode:       requestedMode,
		ResolvedMode:        preflight.ResolvedMode,
		Prompt:              preflight.EffectivePrompt,
		ConfirmationReason:  preflight.Reason,
		Status:              status,
		WorkDir:             workDir,
		InstructionsVersion: "rural-conecta-instagram · R3",
		ContextSummary:      fmt.Sprintf("%s · Pág %d (%s)", doc.Title, req.PageIndex+1, preflight.ResolvedMode),
		StartedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.store.CreateAgentTask(ctx, task); err != nil {
		return domain.Task{}, err
	}

	if status == domain.StatusNeedsConfirmation {
		s.appendLogSync(ctx, task.ID, 1, "system", "needs_confirmation", "Aguardando confirmação: "+task.ConfirmationReason)
		s.broadcast("agent:task", task)
		return task, nil
	}

	if err := s.launchExecution(task, doc, offer, ""); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// ConfirmTask proceeds with executing a task paused in StatusNeedsConfirmation.
func (s *Service) ConfirmTask(ctx context.Context, taskID string) (domain.Task, error) {
	task, err := s.store.GetAgentTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if task.Status != domain.StatusNeedsConfirmation {
		return domain.Task{}, fmt.Errorf("tarefa %s não está aguardando confirmação (estado atual: %s)", taskID, task.Status)
	}

	doc, err := s.store.GetDocument(ctx, task.DocumentID)
	if err != nil {
		return domain.Task{}, err
	}
	var offer *library.DatedOffer
	if strings.TrimSpace(doc.OfferID) != "" {
		offer, _ = s.store.GetDatedOffer(ctx, doc.OfferID)
	}

	task.Status = domain.StatusRunning
	task.ConfirmationReason = ""
	if err := s.store.UpdateAgentTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	s.appendLogSync(ctx, task.ID, 2, "system", "confirmed", "Confirmação recebida. Iniciando Codex CLI...")
	s.broadcast("agent:task", task)

	if err := s.launchExecution(task, doc, offer, ""); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// CancelTask cancels the active Codex CLI task (if running) or discards a pending confirmation task.
func (s *Service) CancelTask(ctx context.Context, taskID string) error {
	task, err := s.store.GetAgentTask(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status == domain.StatusNeedsConfirmation {
		task.Status = domain.StatusCanceled
		task.ErrorCode = "canceled"
		task.ErrorMessage = "Pedido cancelado antes de executar."
		task.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		if err := s.store.UpdateAgentTask(ctx, task); err != nil {
			return err
		}
		s.broadcast("agent:task", task)
		return nil
	}

	s.activeMu.Lock()
	activeMatch := s.activeTaskID == taskID
	cancel := s.activeCancel
	s.activeMu.Unlock()

	if activeMatch && cancel != nil {
		cancel()
		return nil
	}
	return errors.New("essa tarefa não está em execução no momento")
}

// RecoverTaskArtifact checks workDir of an interrupted/failed/canceled task for an already generated image,
// imports it safely without re-invoking Codex CLI, and applies it to the document.
func (s *Service) RecoverTaskArtifact(ctx context.Context, taskID string) (domain.Task, error) {
	task, err := s.store.GetAgentTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if strings.TrimSpace(task.WorkDir) == "" {
		return domain.Task{}, errors.New("pasta de trabalho da tarefa não encontrada")
	}

	exclude := map[string]bool{
		filepath.Clean(filepath.Join(task.WorkDir, "input-page.png")): true,
	}
	foundImg := codex.FindGeneratedImage(task.WorkDir, task.OutputImagePath, exclude)
	if foundImg == "" {
		return domain.Task{}, errors.New("nenhuma imagem gerada válida foi encontrada na pasta dessa tarefa")
	}

	updatedTask, err := s.finalizeWithImage(ctx, task, foundImg, "Artefato recuperado da pasta de trabalho sem nova chamada ao Codex.")
	if err != nil {
		return domain.Task{}, err
	}
	return updatedTask, nil
}

// ResumeTask continues an interrupted/failed task session via `codex exec resume <threadID>`
// after ensuring an artifact isn't already waiting on disk (unless followUp refinement is explicitly given).
func (s *Service) ResumeTask(ctx context.Context, taskID string, followUp string) (domain.Task, error) {
	task, err := s.store.GetAgentTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if strings.TrimSpace(task.ThreadID) == "" {
		return domain.Task{}, errors.New("essa tarefa não registrou ID de sessão no Codex para retomada")
	}

	exclude := map[string]bool{
		filepath.Clean(filepath.Join(task.WorkDir, "input-page.png")): true,
	}
	if strings.TrimSpace(followUp) == "" {
		if existing := codex.FindGeneratedImage(task.WorkDir, task.OutputImagePath, exclude); existing != "" {
			return s.finalizeWithImage(ctx, task, existing, "Artefato já existente recuperado antes de retomar a chamada.")
		}
	}

	doc, err := s.store.GetDocument(ctx, task.DocumentID)
	if err != nil {
		return domain.Task{}, err
	}
	var offer *library.DatedOffer
	if strings.TrimSpace(doc.OfferID) != "" {
		offer, _ = s.store.GetDatedOffer(ctx, doc.OfferID)
	}

	task.Status = domain.StatusRunning
	task.ErrorCode = ""
	task.ErrorMessage = ""
	task.FinishedAt = ""
	if err := s.store.UpdateAgentTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	s.broadcast("agent:task", task)

	if err := s.launchExecution(task, doc, offer, strings.TrimSpace(followUp)); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (s *Service) launchExecution(task domain.Task, doc document.Document, offer *library.DatedOffer, resumeFollowUp string) error {
	s.activeMu.Lock()
	if s.activeTaskID != "" {
		s.activeMu.Unlock()
		return errors.New("já existe outra execução do agente ativa")
	}
	runCtx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	s.activeTaskID = task.ID
	s.activeCancel = cancel
	s.activeMu.Unlock()

	go s.runTaskAsync(runCtx, cancel, task, doc, offer, resumeFollowUp)
	return nil
}

func (s *Service) runTaskAsync(
	runCtx context.Context,
	cancel context.CancelFunc,
	task domain.Task,
	doc document.Document,
	offer *library.DatedOffer,
	resumeFollowUp string,
) {
	defer func() {
		cancel()
		s.activeMu.Lock()
		if s.activeTaskID == task.ID {
			s.activeTaskID = ""
			s.activeCancel = nil
		}
		s.activeMu.Unlock()
	}()

	bgCtx := context.Background()
	targetOutput := filepath.Join(task.WorkDir, "result.png")
	snapPath := filepath.Join(task.WorkDir, "input-page.png")
	var inputImages []string
	if _, err := os.Stat(snapPath); err == nil {
		inputImages = append(inputImages, snapPath)
	}

	var prompt string
	if strings.TrimSpace(resumeFollowUp) != "" {
		prompt = fmt.Sprintf("%s\n\nCertifique-se de salvar a imagem PNG final em: %s", strings.TrimSpace(resumeFollowUp), targetOutput)
	} else if strings.TrimSpace(task.ThreadID) != "" && resumeFollowUp == "" && task.Status == domain.StatusRunning && task.UpdatedAt != task.StartedAt {
		prompt = fmt.Sprintf("Retome o pedido anterior desta sessão e salve a imagem PNG final exatamente em: %s", targetOutput)
	} else {
		var profilePtr *library.CompanyProfile
		if p, pErr := s.store.GetCompanyProfile(bgCtx, library.DefaultCompanyID); pErr == nil {
			profilePtr = &p
		}
		skillText := s.loadSkillExcerpt()
		var annotations []document.AgentAnnotation
		var page document.Page
		if task.PageIndex >= 0 && task.PageIndex < len(doc.Pages) {
			page = doc.Pages[task.PageIndex]
			for _, a := range page.Annotations {
				if !a.Resolved {
					annotations = append(annotations, a)
				}
			}
		}
		prompt = domain.BuildGenerationPrompt(domain.PromptBuildInput{
			Document:          doc,
			Page:              page,
			Profile:           profilePtr,
			Offer:             offer,
			Mode:              task.ResolvedMode,
			Prompt:            task.Prompt,
			Annotations:       annotations,
			SnapshotAttached:  len(inputImages) > 0,
			TargetOutputPath:  targetOutput,
			SkillInstructions: skillText,
		})
	}

	existingLogs, _ := s.store.ListAgentLogs(bgCtx, task.ID)
	seq := len(existingLogs)
	var logMu sync.Mutex
	appendStreamLog := func(stream, eventType, text string) {
		logMu.Lock()
		seq++
		currentSeq := seq
		logMu.Unlock()
		s.appendLogSync(bgCtx, task.ID, currentSeq, stream, eventType, text)
	}

	outcome := s.runner.RunExec(runCtx, codex.RunRequest{
		WorkDir:          task.WorkDir,
		Prompt:           prompt,
		InputImages:      inputImages,
		TargetOutputPath: targetOutput,
		ThreadID:         task.ThreadID,
	}, appendStreamLog)

	if outcome.ThreadID != "" {
		task.ThreadID = outcome.ThreadID
	}
	if outcome.OutputImagePath != "" {
		task.OutputImagePath = outcome.OutputImagePath
		task.ArtifactRecoverable = outcome.ArtifactRecoverable
	}

	s.activeMu.Lock()
	closing := s.shuttingDown
	s.activeMu.Unlock()
	if closing {
		task.Status = domain.StatusInterruptedUncertain
		task.ErrorCode = "interrupted"
		task.ErrorMessage = "Aplicativo encerrado durante execução. Confira se há artefato recuperável antes de repetir."
		task.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.store.UpdateAgentTask(bgCtx, task)
		return
	}

	if outcome.Canceled {
		task.Status = domain.StatusCanceled
		task.ErrorCode = outcome.ErrorCode
		task.ErrorMessage = outcome.ErrorMessage
		task.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.store.UpdateAgentTask(bgCtx, task)
		s.broadcast("agent:task", task)
		return
	}

	if outcome.ErrorCode != "" {
		task.Status = domain.StatusFailed
		task.ErrorCode = outcome.ErrorCode
		task.ErrorMessage = outcome.ErrorMessage
		task.SummaryText = outcome.LastMessage
		task.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.store.UpdateAgentTask(bgCtx, task)
		s.broadcast("agent:task", task)
		return
	}

	summary := outcome.LastMessage
	if summary == "" {
		summary = "Imagem gerada pelo Codex CLI e aplicada na peça."
	}
	finalTask, err := s.finalizeWithImage(bgCtx, task, outcome.OutputImagePath, summary)
	if err != nil {
		task.Status = domain.StatusFailed
		task.ErrorCode = "import_apply_error"
		task.ErrorMessage = err.Error()
		task.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = s.store.UpdateAgentTask(bgCtx, task)
		appendStreamLog("error", "import_apply_error", err.Error())
		s.broadcast("agent:task", task)
		return
	}

	appendStreamLog("system", "applied", fmt.Sprintf("Aplicado na peça (v%d) e salvo na Biblioteca (%s).", finalTask.ResultRevision, finalTask.OutputAssetURL))
}

func (s *Service) finalizeWithImage(ctx context.Context, task domain.Task, imagePath string, summary string) (domain.Task, error) {
	asset, err := s.importGeneratedFile(ctx, task, imagePath)
	if err != nil {
		return task, err
	}
	updatedDoc, err := s.applyAssetToDocument(ctx, task, asset)
	if err != nil {
		return task, err
	}

	task.Status = domain.StatusCompleted
	task.OutputImagePath = imagePath
	task.OutputAssetID = asset.ID
	task.OutputAssetURL = asset.AssetURL
	task.ArtifactRecoverable = true
	task.ResultDocumentID = updatedDoc.ID
	task.ResultRevision = updatedDoc.Revision
	task.SummaryText = summary
	task.ErrorCode = ""
	task.ErrorMessage = ""
	task.FinishedAt = time.Now().UTC().Format(time.RFC3339)

	if err := s.store.UpdateAgentTask(ctx, task); err != nil {
		return task, err
	}
	s.broadcast("agent:task", task)
	return task, nil
}

func (s *Service) importGeneratedFile(ctx context.Context, task domain.Task, imagePath string) (library.LibraryAsset, error) {
	if s.libraryDir == "" || s.libraryDir == "." {
		return library.LibraryAsset{}, errors.New("diretório da biblioteca gerenciada não configurado")
	}
	raw, err := os.ReadFile(imagePath)
	if err != nil {
		return library.LibraryAsset{}, fmt.Errorf("ler imagem gerada: %w", err)
	}
	cfg, imgFmt, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return library.LibraryAsset{}, fmt.Errorf("arquivo gerado não é PNG/JPEG válido: %w", err)
	}
	ext := "png"
	mime := "image/png"
	if imgFmt == "jpeg" {
		ext = "jpg"
		mime = "image/jpeg"
	}

	shortID := uuid.NewString()[:8]
	fileName := fmt.Sprintf("codex-%s.%s", shortID, ext)
	relPath := filepath.ToSlash(filepath.Join("generated", fileName))
	destPath := filepath.Join(s.libraryDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
		return library.LibraryAsset{}, fmt.Errorf("criar pasta generated: %w", err)
	}
	tmp := destPath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o640); err != nil {
		return library.LibraryAsset{}, fmt.Errorf("gravar imagem gerada: %w", err)
	}
	if err := os.Rename(tmp, destPath); err != nil {
		_ = os.Remove(tmp)
		return library.LibraryAsset{}, fmt.Errorf("mover imagem gerada: %w", err)
	}

	sum := sha256.Sum256(raw)
	fmtTag := "other"
	if cfg.Width == 1080 && cfg.Height == 1920 {
		fmtTag = "story"
	} else if cfg.Width == 1080 && cfg.Height == 1350 {
		fmtTag = "feed"
	}

	asset := library.LibraryAsset{
		ID:             "gen-" + shortID,
		CompanyID:      library.DefaultCompanyID,
		RelPath:        relPath,
		ThumbRelPath:   relPath,
		SourceRelPath:  "codex-agent/" + task.ID,
		FileName:       fileName,
		MimeType:       mime,
		MediaKind:      "image",
		Classification: "cena",
		SubCategory:    "gerado_codex",
		Format:         fmtTag,
		Title:          fmt.Sprintf("Cena Codex (%s)", shortID),
		Width:          cfg.Width,
		Height:         cfg.Height,
		Bytes:          int64(len(raw)),
		SHA256:         hex.EncodeToString(sum[:]),
		MetadataJSON:   "{}",
		ImportedAt:     time.Now().UTC().Format(time.RFC3339),
		AssetURL:       "/library-assets/" + relPath,
		ThumbURL:       "/library-assets/" + relPath,
	}
	if err := s.store.UpsertLibraryAsset(ctx, asset); err != nil {
		return library.LibraryAsset{}, err
	}
	return asset, nil
}

func (s *Service) applyAssetToDocument(ctx context.Context, task domain.Task, asset library.LibraryAsset) (document.Document, error) {
	doc, err := s.store.GetDocument(ctx, task.DocumentID)
	if err != nil {
		return document.Document{}, err
	}
	// Protect approved revisions cleanly: open new draft revision first without ever mutating approved history.
	if doc.Status == "approved" {
		doc, err = s.store.StartDraftFromApproved(ctx, doc.ID)
		if err != nil {
			return document.Document{}, err
		}
	}
	if task.PageIndex < 0 || task.PageIndex >= len(doc.Pages) {
		return document.Document{}, fmt.Errorf("página %d inválida no documento", task.PageIndex)
	}

	page := doc.Pages[task.PageIndex]
	if task.ResolvedMode == domain.ModeFullPiece {
		// Full piece replacement places the regenerated artwork on the target page while keeping
		// every other page untouched and keeping earlier revisions in document_revisions.
		page.Layers = []document.Layer{
			{
				ID:        "codex-full-" + uuid.NewString()[:8],
				Name:      "Arte editada pelo Agente",
				Kind:      "image",
				X:         0,
				Y:         0,
				Width:     doc.WorkspaceWidth,
				Height:    doc.WorkspaceHeight,
				ScaleX:    1,
				ScaleY:    1,
				Opacity:   1,
				Visible:   true,
				Locked:    false,
				ZIndex:    1,
				AssetKey:  asset.ID,
				AssetURL:  asset.AssetURL,
				ObjectFit: "cover",
				PositionX: 0.5,
				PositionY: 0.5,
			},
		}
	} else {
		// Background layer mode replaces the lowest non-logo background photo layer (or inserts one at zIndex 1),
		// keeping all editable text, shape, and logo layers intact on the page.
		replacedIdx := -1
		for idx, l := range page.Layers {
			if l.Kind == "image" && l.ObjectFit != "contain" && !strings.Contains(strings.ToLower(l.Name), "logo") {
				replacedIdx = idx
				break
			}
		}
		if replacedIdx >= 0 {
			page.Layers[replacedIdx].AssetKey = asset.ID
			page.Layers[replacedIdx].AssetURL = asset.AssetURL
			page.Layers[replacedIdx].ObjectFit = "cover"
		} else {
			for i := range page.Layers {
				page.Layers[i].ZIndex++
			}
			bgLayer := document.Layer{
				ID:        "codex-bg-" + uuid.NewString()[:8],
				Name:      "Cena gerada (Agente)",
				Kind:      "image",
				X:         0,
				Y:         0,
				Width:     doc.WorkspaceWidth,
				Height:    doc.WorkspaceHeight,
				ScaleX:    1,
				ScaleY:    1,
				Opacity:   1,
				Visible:   true,
				Locked:    false,
				ZIndex:    1,
				AssetKey:  asset.ID,
				AssetURL:  asset.AssetURL,
				ObjectFit: "cover",
				PositionX: 0.5,
				PositionY: 0.5,
			}
			page.Layers = append([]document.Layer{bgLayer}, page.Layers...)
		}
	}

	for i := range page.Annotations {
		page.Annotations[i].Resolved = true
	}
	doc.Pages[task.PageIndex] = page

	note := fmt.Sprintf("agente Codex (%s): pág %d", task.ResolvedMode, task.PageIndex+1)
	return s.store.SaveDocument(ctx, doc, note)
}

func (s *Service) appendLogSync(ctx context.Context, taskID string, seq int, stream, eventType, text string) {
	entry := domain.LogEntry{
		ID:        uuid.NewString(),
		TaskID:    taskID,
		Seq:       seq,
		Stream:    stream,
		EventType: eventType,
		Text:      text,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	_ = s.store.AppendAgentLog(ctx, entry)
	s.broadcast("agent:log", entry)
}

func (s *Service) broadcast(name string, payload any) {
	if s.emit != nil {
		s.emit(name, payload)
	}
}

func (s *Service) loadSkillExcerpt() string {
	if s.skillsDir == "" {
		return ""
	}
	skillPath := filepath.Join(s.skillsDir, "rural-conecta-instagram", "SKILL.md")
	raw, err := os.ReadFile(skillPath)
	if err != nil {
		return ""
	}
	txt := string(raw)
	if len(txt) > 2200 {
		txt = txt[:2200]
	}
	return txt
}

func decodeBase64Payload(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("empty payload")
	}
	if idx := strings.Index(trimmed, "base64,"); idx >= 0 {
		trimmed = trimmed[idx+len("base64,"):]
	}
	return base64.StdEncoding.DecodeString(trimmed)
}
