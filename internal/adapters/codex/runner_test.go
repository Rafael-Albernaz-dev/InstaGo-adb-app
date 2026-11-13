package codex

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFakeExecutable(t *testing.T, script string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "codex")
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	return p
}

func writeValidPNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	img.Set(1, 1, color.RGBA{R: 61, G: 106, B: 47, A: 255})
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}

func TestRunnerCheckHealth(t *testing.T) {
	fake := writeFakeExecutable(t, `#!/bin/bash
if [ "$1" = "--version" ]; then
  echo "codex-cli 0.167.2-test"
  exit 0
fi
if [ "$1" = "login" ] && [ "$2" = "status" ]; then
  echo "Logged in using ChatGPT"
  exit 0
fi
exit 1
`)

	r := NewRunner(fake)
	h := r.CheckHealth(context.Background())
	if !h.Available || !h.Authenticated || h.Version != "codex-cli 0.167.2-test" {
		t.Fatalf("unexpected health: %+v", h)
	}

	missing := NewRunner(filepath.Join(t.TempDir(), "missing-codex"))
	mh := missing.CheckHealth(context.Background())
	if mh.Available || mh.Authenticated {
		t.Fatalf("expected missing binary health to be false: %+v", mh)
	}
}

func TestRunnerExecSuccessAndNoArtifactFailure(t *testing.T) {
	workDir := t.TempDir()
	outPath := filepath.Join(workDir, "result.png")
	seedImg := filepath.Join(t.TempDir(), "generated.png")
	writeValidPNG(t, seedImg)

	fakeSuccess := writeFakeExecutable(t, `#!/bin/bash
cat >/dev/null
echo '{"type":"thread.started","thread_id":"thr-123"}'
echo '{"type":"item.completed","item":{"type":"image_generation_call","status":"completed"}}'
echo '{"type":"item.completed","item":{"type":"agent_message","text":"Imagem criada com sucesso."}}'
echo '{"type":"turn.completed"}'
cp "`+seedImg+`" "`+outPath+`"
exit 0
`)

	var logs []string
	r := NewRunner(fakeSuccess)
	res := r.RunExec(context.Background(), RunRequest{
		WorkDir:          workDir,
		Prompt:           "Gerar cena",
		TargetOutputPath: outPath,
	}, func(_, _, text string) {
		logs = append(logs, text)
	})
	if res.ErrorCode != "" || res.ThreadID != "thr-123" || res.OutputImagePath != outPath {
		t.Fatalf("unexpected success outcome: %+v", res)
	}
	if len(logs) == 0 {
		t.Fatal("expected streamed logs from JSONL output")
	}

	// Failure without artifact: agent claims success text, but creates no image
	emptyWorkDir := t.TempDir()
	fakeNoImg := writeFakeExecutable(t, `#!/bin/bash
cat >/dev/null
echo '{"type":"thread.started","thread_id":"thr-empty"}'
echo '{"type":"item.completed","item":{"type":"agent_message","text":"Pronto!"}}'
exit 0
`)
	rNoImg := NewRunner(fakeNoImg)
	noArtifactRes := rNoImg.RunExec(context.Background(), RunRequest{
		WorkDir:          emptyWorkDir,
		Prompt:           "Gerar",
		TargetOutputPath: filepath.Join(emptyWorkDir, "result.png"),
	}, nil)
	if noArtifactRes.ErrorCode != "no_artifact" || noArtifactRes.OutputImagePath != "" {
		t.Fatalf("expected no_artifact when text says done without file, got %+v", noArtifactRes)
	}
}

func TestRunnerCancellationPreservesRecoverableArtifactAndClassifiesQuota(t *testing.T) {
	workDir := t.TempDir()
	outPath := filepath.Join(workDir, "result.png")
	seedImg := filepath.Join(t.TempDir(), "partial.png")
	writeValidPNG(t, seedImg)

	fakeSlow := writeFakeExecutable(t, `#!/bin/bash
cat >/dev/null
echo '{"type":"thread.started","thread_id":"thr-slow"}'
cp "`+seedImg+`" "`+outPath+`"
sleep 5
exit 0
`)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	r := NewRunner(fakeSlow)
	res := r.RunExec(ctx, RunRequest{
		WorkDir:          workDir,
		Prompt:           "Demorado",
		TargetOutputPath: outPath,
	}, nil)
	if !res.Canceled || !res.ArtifactRecoverable || res.OutputImagePath != outPath {
		t.Fatalf("expected canceled with recoverable artifact, got %+v", res)
	}

	fakeQuota := writeFakeExecutable(t, `#!/bin/bash
cat >/dev/null
echo '{"type":"error","message":"Rate limit / quota exceeded (429)"}' >&2
exit 1
`)
	rq := NewRunner(fakeQuota).RunExec(context.Background(), RunRequest{
		WorkDir: t.TempDir(),
		Prompt:  "Teste quota",
	}, nil)
	if rq.ErrorCode != "quota_limit" || !strings.Contains(strings.ToLower(rq.ErrorMessage), "quota") {
		t.Fatalf("expected quota_limit classification, got %+v", rq)
	}
}
