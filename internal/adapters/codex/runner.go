package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/agent"
)

// RunRequest parameterizes a single `codex exec` or `codex exec resume` invocation.
type RunRequest struct {
	WorkDir          string
	Prompt           string
	InputImages      []string
	TargetOutputPath string
	ThreadID         string
}

// RunOutcome summarizes the final CLI execution state, recovered thread ID, output image, and diagnostics.
type RunOutcome struct {
	ThreadID            string
	LastMessage         string
	OutputImagePath     string
	ArtifactRecoverable bool
	ErrorCode           string
	ErrorMessage        string
	Canceled            bool
}

// LogCallback receives human-readable live terminal log lines parsed from stdout/stderr.
type LogCallback func(stream, eventType, text string)

// Runner executes Codex CLI commands cleanly via os/exec without shell interpolation or credential access.
type Runner struct {
	binary string
}

// NewRunner creates a Codex CLI adapter using binaryPath (defaults to "codex" in PATH).
func NewRunner(binaryPath string) *Runner {
	bin := strings.TrimSpace(binaryPath)
	if bin == "" {
		if custom := strings.TrimSpace(os.Getenv("INSTAGO_CODEX_BIN")); custom != "" {
			bin = custom
		} else {
			bin = "codex"
		}
	}
	return &Runner{binary: bin}
}

// CheckHealth detects the Codex CLI executable, version, and active ChatGPT login status.
// It never reads auth files or extracts tokens/credentials.
func (r *Runner) CheckHealth(ctx context.Context) agent.CLIHealth {
	resolved, err := exec.LookPath(r.binary)
	if err != nil {
		return agent.CLIHealth{
			Available:    false,
			ErrorMessage: fmt.Sprintf("Codex CLI (%q) não encontrado no PATH.", r.binary),
		}
	}

	health := agent.CLIHealth{
		Available:  true,
		BinaryPath: resolved,
	}

	verCtx, cancelVer := context.WithTimeout(ctx, 8*time.Second)
	defer cancelVer()
	verOut, verErr := exec.CommandContext(verCtx, resolved, "--version").CombinedOutput()
	if verErr != nil {
		health.ErrorMessage = fmt.Sprintf("Falha ao consultar versão do Codex CLI: %v", verErr)
	} else {
		health.Version = strings.TrimSpace(string(verOut))
	}

	authCtx, cancelAuth := context.WithTimeout(ctx, 8*time.Second)
	defer cancelAuth()
	authOut, authErr := exec.CommandContext(authCtx, resolved, "login", "status").CombinedOutput()
	statusText := strings.TrimSpace(string(authOut))
	health.AuthStatusText = statusText
	if authErr == nil && strings.Contains(strings.ToLower(statusText), "logged in") {
		health.Authenticated = true
	} else {
		health.Authenticated = false
		if health.ErrorMessage == "" {
			if statusText != "" {
				health.ErrorMessage = fmt.Sprintf("Login Codex não ativo: %s", statusText)
			} else {
				health.ErrorMessage = "Login Codex (ChatGPT) não confirmado. Execute `codex login` no terminal."
			}
		}
	}

	return health
}

// RunExec starts a non-interactive `codex exec` or `codex exec resume` process, streams JSONL events,
// handles cancellation cleanly, and verifies generated image artifacts on disk.
func (r *Runner) RunExec(ctx context.Context, req RunRequest, onLog LogCallback) RunOutcome {
	resolved, err := exec.LookPath(r.binary)
	if err != nil {
		msg := fmt.Sprintf("Codex CLI (%q) não encontrado.", r.binary)
		emitLog(onLog, "error", "cli_missing", msg)
		return RunOutcome{ErrorCode: "cli_missing", ErrorMessage: msg}
	}

	workDir := filepath.Clean(req.WorkDir)
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		msg := fmt.Sprintf("Falha ao preparar pasta de trabalho do agente: %v", err)
		emitLog(onLog, "error", "workdir_error", msg)
		return RunOutcome{ErrorCode: "workdir_error", ErrorMessage: msg}
	}

	lastMsgFile := filepath.Join(workDir, ".codex-last-message.txt")
	_ = os.Remove(lastMsgFile)

	args := buildExecArgs(req, lastMsgFile)
	cmd := exec.Command(resolved, args...)
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(req.Prompt)
	configureCommandProcess(cmd)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		msg := fmt.Sprintf("Falha ao abrir stdout do Codex CLI: %v", err)
		return RunOutcome{ErrorCode: "process_error", ErrorMessage: msg}
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		msg := fmt.Sprintf("Falha ao abrir stderr do Codex CLI: %v", err)
		return RunOutcome{ErrorCode: "process_error", ErrorMessage: msg}
	}

	emitLog(onLog, "system", "command_start", fmt.Sprintf("$ %s %s", filepath.Base(resolved), strings.Join(args, " ")))
	if err := cmd.Start(); err != nil {
		msg := fmt.Sprintf("Erro ao iniciar Codex CLI: %v", err)
		emitLog(onLog, "error", "start_error", msg)
		return RunOutcome{ErrorCode: "process_error", ErrorMessage: msg}
	}

	var outcome RunOutcome
	outcome.ThreadID = strings.TrimSpace(req.ThreadID)

	var stateMu sync.Mutex
	var diagLines []string
	var wg sync.WaitGroup
	wg.Add(2)

	// Cancel watcher terminates the process group immediately when ctx is canceled.
	doneWait := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			emitLog(onLog, "system", "cancel_signal", "Solicitando cancelamento do processo Codex...")
			killCommandProcess(cmd)
		case <-doneWait:
		}
	}()

	go func() {
		defer wg.Done()
		readJSONLStream(stdoutPipe, func(line string) {
			evType, threadID, summary, streamKind := parseJSONLEvent(line)
			stateMu.Lock()
			if threadID != "" {
				outcome.ThreadID = threadID
			}
			if summary != "" {
				diagLines = append(diagLines, summary)
			}
			stateMu.Unlock()
			if summary != "" {
				emitLog(onLog, streamKind, evType, summary)
			}
		})
	}()

	go func() {
		defer wg.Done()
		readLineStream(stderrPipe, func(line string) {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				return
			}
			stateMu.Lock()
			diagLines = append(diagLines, trimmed)
			stateMu.Unlock()
			emitLog(onLog, "stderr", "stderr", trimmed)
		})
	}()

	wg.Wait()
	waitErr := cmd.Wait()
	close(doneWait)

	if rawLast, readErr := os.ReadFile(lastMsgFile); readErr == nil {
		outcome.LastMessage = strings.TrimSpace(string(rawLast))
	}

	exclude := make(map[string]bool)
	for _, imgPath := range req.InputImages {
		exclude[filepath.Clean(imgPath)] = true
	}
	exclude[filepath.Clean(lastMsgFile)] = true

	foundImg := FindGeneratedImage(workDir, req.TargetOutputPath, exclude)
	if foundImg != "" {
		outcome.OutputImagePath = foundImg
		outcome.ArtifactRecoverable = true
	}

	if ctx.Err() != nil {
		outcome.Canceled = true
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			outcome.ErrorCode = "timeout"
			if foundImg != "" {
				outcome.ErrorMessage = "Tempo limite atingido, mas uma imagem válida já havia sido salva na pasta de trabalho."
			} else {
				outcome.ErrorMessage = "Tempo limite de execução do Codex atingido."
			}
		} else {
			outcome.ErrorCode = "canceled"
			if foundImg != "" {
				outcome.ErrorMessage = "Geração cancelada pelo operador, mas uma imagem válida já havia sido salva na pasta de trabalho."
			} else {
				outcome.ErrorMessage = "Geração cancelada pelo operador."
			}
		}
		emitLog(onLog, "system", outcome.ErrorCode, outcome.ErrorMessage)
		return outcome
	}

	if waitErr != nil {
		joinedDiag := strings.Join(diagLines, "\n") + "\n" + outcome.LastMessage
		code, friendly := classifyProcessError(waitErr, joinedDiag)
		outcome.ErrorCode = code
		outcome.ErrorMessage = friendly
		emitLog(onLog, "error", code, friendly)
		return outcome
	}

	if foundImg == "" {
		outcome.ErrorCode = "no_artifact"
		outcome.ErrorMessage = "O Codex encerrou sem gerar uma imagem PNG/JPEG válida na pasta de trabalho."
		emitLog(onLog, "error", "no_artifact", outcome.ErrorMessage)
		return outcome
	}

	emitLog(onLog, "system", "artifact_ready", fmt.Sprintf("Imagem validada em %s", foundImg))
	return outcome
}

func buildExecArgs(req RunRequest, lastMsgFile string) []string {
	var args []string
	if strings.TrimSpace(req.ThreadID) != "" {
		args = []string{
			"exec", "resume",
			"--json",
			"--skip-git-repo-check",
			"-o", lastMsgFile,
		}
		for _, img := range req.InputImages {
			if strings.TrimSpace(img) != "" {
				args = append(args, "-i", img)
			}
		}
		args = append(args, strings.TrimSpace(req.ThreadID), "-")
		return args
	}

	args = []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--color", "never",
		"-s", "workspace-write",
		"-C", req.WorkDir,
		"--add-dir", req.WorkDir,
		"-o", lastMsgFile,
	}
	for _, img := range req.InputImages {
		if strings.TrimSpace(img) != "" {
			args = append(args, "-i", img)
		}
	}
	args = append(args, "-")
	return args
}

func readJSONLStream(r io.Reader, handle func(string)) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			handle(line)
		}
	}
}

func readLineStream(r io.Reader, handle func(string)) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 16*1024), 1024*1024)
	for scanner.Scan() {
		handle(scanner.Text())
	}
}

func emitLog(cb LogCallback, stream, eventType, text string) {
	if cb == nil || strings.TrimSpace(text) == "" {
		return
	}
	cb(stream, eventType, text)
}

func parseJSONLEvent(line string) (eventType, threadID, summary, stream string) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		return "raw", "", line, "stdout"
	}

	eventType, _ = payload["type"].(string)
	if tid, ok := payload["thread_id"].(string); ok && tid != "" {
		threadID = tid
	}

	switch eventType {
	case "thread.started":
		return eventType, threadID, fmt.Sprintf("Sessão Codex iniciada (%s)", threadID), "system"
	case "turn.started":
		return eventType, threadID, "Processando pedido...", "system"
	case "turn.completed":
		return eventType, threadID, "Turno do agente concluído.", "system"
	case "turn.failed", "error":
		errText := extractNestedText(payload)
		if errText == "" {
			errText = line
		}
		return eventType, threadID, errText, "error"
	}

	if item, ok := payload["item"].(map[string]any); ok {
		itemType, _ := item["type"].(string)
		switch itemType {
		case "agent_message", "reasoning":
			txt := extractNestedText(item)
			if txt != "" {
				return eventType + ":" + itemType, threadID, txt, "agent"
			}
		case "command_execution":
			cmdStr, _ := item["command"].(string)
			status, _ := item["status"].(string)
			if cmdStr != "" {
				return eventType + ":" + itemType, threadID, fmt.Sprintf("[tool:%s] %s", status, cmdStr), "tool"
			}
		case "image_generation", "image_generation_call":
			status, _ := item["status"].(string)
			return eventType + ":" + itemType, threadID, fmt.Sprintf("[image_gen] %s", status), "tool"
		default:
			txt := extractNestedText(item)
			if txt != "" {
				return eventType + ":" + itemType, threadID, fmt.Sprintf("[%s] %s", itemType, txt), "tool"
			}
		}
	}

	if txt := extractNestedText(payload); txt != "" {
		return eventType, threadID, txt, "stdout"
	}
	return eventType, threadID, line, "stdout"
}

func extractNestedText(m map[string]any) string {
	for _, key := range []string{"message", "text", "error", "delta", "prompt", "revised_prompt"} {
		if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		if sub, ok := m[key].(map[string]any); ok {
			if inner := extractNestedText(sub); inner != "" {
				return inner
			}
		}
	}
	if content, ok := m["content"].([]any); ok {
		var parts []string
		for _, c := range content {
			if cm, ok := c.(map[string]any); ok {
				if t := extractNestedText(cm); t != "" {
					parts = append(parts, t)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}
	return ""
}

func classifyProcessError(waitErr error, diag string) (string, string) {
	lower := strings.ToLower(diag)
	switch {
	case strings.Contains(lower, "quota") || strings.Contains(lower, "rate limit") || strings.Contains(lower, "429") || strings.Contains(lower, "usage limit"):
		return "quota_limit", "Limite de uso/quota do Codex/ChatGPT atingido. Aguarde ou verifique sua conta antes de tentar novamente."
	case strings.Contains(lower, "not logged in") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "authentication") || strings.Contains(lower, "login required"):
		return "auth_required", "Autenticação do Codex CLI não está ativa. Faça login no Codex CLI com sua conta ChatGPT."
	default:
		return "process_error", fmt.Sprintf("Codex CLI finalizou com erro (%v).", waitErr)
	}
}

// FindGeneratedImage checks preferredPath first, then searches workDir for the newest valid PNG/JPEG image
// created outside excludePaths.
func FindGeneratedImage(workDir string, preferredPath string, excludePaths map[string]bool) string {
	if strings.TrimSpace(preferredPath) != "" && !excludePaths[filepath.Clean(preferredPath)] {
		if isValidImageFile(preferredPath) {
			return filepath.Clean(preferredPath)
		}
	}

	var bestPath string
	var bestTime time.Time

	_ = filepath.WalkDir(workDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		cleaned := filepath.Clean(path)
		if excludePaths[cleaned] {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(cleaned))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			return nil
		}
		if !isValidImageFile(cleaned) {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		if bestPath == "" || info.ModTime().After(bestTime) {
			bestPath = cleaned
			bestTime = info.ModTime()
		}
		return nil
	})
	return bestPath
}

func isValidImageFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return false
	}
	return cfg.Width > 0 && cfg.Height > 0
}
