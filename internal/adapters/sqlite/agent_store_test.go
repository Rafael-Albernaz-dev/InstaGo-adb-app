package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/agent"
)

func TestAgentTaskStoreLifecycleAndRecovery(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = store.Close() }()

	task := agent.Task{
		ID:            "task-1",
		DocumentID:    "doc-1",
		PageIndex:     0,
		RequestedMode: agent.ModeAuto,
		ResolvedMode:  agent.ModeBackgroundLayer,
		Prompt:        "Trocar fundo",
		Status:        agent.StatusRunning,
		ThreadID:      "thread-abc",
		WorkDir:       "/tmp/agent-work",
	}
	if err := store.CreateAgentTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := store.AppendAgentLog(ctx, agent.LogEntry{
		ID:        "log-1",
		TaskID:    "task-1",
		Seq:       1,
		Stream:    "system",
		EventType: "started",
		Text:      "Iniciando codex exec...",
	}); err != nil {
		t.Fatalf("append log: %v", err)
	}

	recovered, err := store.RecoverInterruptedAgentTasks(ctx)
	if err != nil {
		t.Fatalf("recover interrupted tasks: %v", err)
	}
	if recovered != 1 {
		t.Fatalf("expected 1 recovered interrupted task, got %d", recovered)
	}

	got, err := store.GetAgentTask(ctx, "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Status != agent.StatusInterruptedUncertain {
		t.Fatalf("expected interrupted_uncertain, got %s", got.Status)
	}

	logs, err := store.ListAgentLogs(ctx, "task-1")
	if err != nil || len(logs) != 1 || logs[0].Text != "Iniciando codex exec..." {
		t.Fatalf("unexpected logs: %+v, err=%v", logs, err)
	}

	list, err := store.ListAgentTasks(ctx, "doc-1", 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("unexpected listed tasks: %+v, err=%v", list, err)
	}
}
