package wailsapp

import (
	"context"

	domain "github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/agent"
	agentusecase "github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/usecase/agent"
)

// AgentBridge exposes contextual Codex CLI generation and recovery use cases to Wails 3 UI.
type AgentBridge struct {
	svc *agentusecase.Service
}

// NewAgentBridge wraps agentusecase.Service for Wails 3 IPC binding.
func NewAgentBridge(svc *agentusecase.Service) *AgentBridge {
	return &AgentBridge{svc: svc}
}

func (b *AgentBridge) CheckAgentHealth() domain.CLIHealth {
	return b.svc.CheckHealth(context.Background())
}

func (b *AgentBridge) ListAgentTasks(documentID string) ([]domain.Task, error) {
	return b.svc.ListTasks(context.Background(), documentID)
}

func (b *AgentBridge) GetAgentTask(taskID string) (domain.TaskDetail, error) {
	return b.svc.GetTask(context.Background(), taskID)
}

func (b *AgentBridge) StartAgentTask(req domain.StartTaskRequest) (domain.Task, error) {
	return b.svc.StartTask(context.Background(), req)
}

func (b *AgentBridge) ConfirmAgentTask(taskID string) (domain.Task, error) {
	return b.svc.ConfirmTask(context.Background(), taskID)
}

func (b *AgentBridge) CancelAgentTask(taskID string) error {
	return b.svc.CancelTask(context.Background(), taskID)
}

func (b *AgentBridge) RecoverAgentArtifact(taskID string) (domain.Task, error) {
	return b.svc.RecoverTaskArtifact(context.Background(), taskID)
}

func (b *AgentBridge) ResumeAgentTask(taskID string, followUp string) (domain.Task, error) {
	return b.svc.ResumeTask(context.Background(), taskID, followUp)
}
