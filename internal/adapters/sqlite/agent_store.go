package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/agent"
)

// CreateAgentTask persists a new agent task record.
func (s *Store) CreateAgentTask(ctx context.Context, task agent.Task) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(task.StartedAt) == "" {
		task.StartedAt = now
	}
	if strings.TrimSpace(task.UpdatedAt) == "" {
		task.UpdatedAt = now
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_tasks (
			id, document_id, page_index, requested_mode, resolved_mode, prompt,
			confirmation_reason, status, thread_id, work_dir, output_image_path,
			output_asset_id, output_asset_url, artifact_recoverable, instructions_version,
			context_summary, result_document_id, result_revision, summary_text,
			error_code, error_message, started_at, updated_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`,
		task.ID, task.DocumentID, task.PageIndex, string(task.RequestedMode), string(task.ResolvedMode),
		task.Prompt, task.ConfirmationReason, string(task.Status), task.ThreadID, task.WorkDir,
		task.OutputImagePath, task.OutputAssetID, task.OutputAssetURL, boolToInt(task.ArtifactRecoverable),
		task.InstructionsVersion, task.ContextSummary, task.ResultDocumentID, task.ResultRevision,
		task.SummaryText, task.ErrorCode, task.ErrorMessage, task.StartedAt, task.UpdatedAt, task.FinishedAt,
	)
	if err != nil {
		return fmt.Errorf("create agent_task: %w", err)
	}
	return nil
}

// UpdateAgentTask persists updated fields of an existing agent task.
func (s *Store) UpdateAgentTask(ctx context.Context, task agent.Task) error {
	task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		UPDATE agent_tasks
		SET document_id = ?,
			page_index = ?,
			requested_mode = ?,
			resolved_mode = ?,
			prompt = ?,
			confirmation_reason = ?,
			status = ?,
			thread_id = ?,
			work_dir = ?,
			output_image_path = ?,
			output_asset_id = ?,
			output_asset_url = ?,
			artifact_recoverable = ?,
			instructions_version = ?,
			context_summary = ?,
			result_document_id = ?,
			result_revision = ?,
			summary_text = ?,
			error_code = ?,
			error_message = ?,
			updated_at = ?,
			finished_at = ?
		WHERE id = ?;
	`,
		task.DocumentID, task.PageIndex, string(task.RequestedMode), string(task.ResolvedMode),
		task.Prompt, task.ConfirmationReason, string(task.Status), task.ThreadID, task.WorkDir,
		task.OutputImagePath, task.OutputAssetID, task.OutputAssetURL, boolToInt(task.ArtifactRecoverable),
		task.InstructionsVersion, task.ContextSummary, task.ResultDocumentID, task.ResultRevision,
		task.SummaryText, task.ErrorCode, task.ErrorMessage, task.UpdatedAt, task.FinishedAt, task.ID,
	)
	if err != nil {
		return fmt.Errorf("update agent_task: %w", err)
	}
	return nil
}

// GetAgentTask returns one task by ID.
func (s *Store) GetAgentTask(ctx context.Context, taskID string) (agent.Task, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id, document_id, page_index, requested_mode, resolved_mode, prompt,
			confirmation_reason, status, thread_id, work_dir, output_image_path,
			output_asset_id, output_asset_url, artifact_recoverable, instructions_version,
			context_summary, result_document_id, result_revision, summary_text,
			error_code, error_message, started_at, updated_at, finished_at
		FROM agent_tasks
		WHERE id = ?;
	`, taskID)
	t, err := scanAgentTask(row)
	if err != nil {
		return agent.Task{}, err
	}
	return t, nil
}

// ListAgentTasks lists recent agent tasks, optionally filtered by documentID.
func (s *Store) ListAgentTasks(ctx context.Context, documentID string, limit int) ([]agent.Task, error) {
	if limit <= 0 {
		limit = 30
	}
	query := `
		SELECT
			id, document_id, page_index, requested_mode, resolved_mode, prompt,
			confirmation_reason, status, thread_id, work_dir, output_image_path,
			output_asset_id, output_asset_url, artifact_recoverable, instructions_version,
			context_summary, result_document_id, result_revision, summary_text,
			error_code, error_message, started_at, updated_at, finished_at
		FROM agent_tasks`
	var args []any
	if strings.TrimSpace(documentID) != "" {
		query += ` WHERE document_id = ?`
		args = append(args, documentID)
	}
	query += ` ORDER BY started_at DESC, rowid DESC LIMIT ?;`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list agent_tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]agent.Task, 0)
	for rows.Next() {
		t, scanErr := scanAgentTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

// AppendAgentLog inserts a sequential terminal log line for an agent task.
func (s *Store) AppendAgentLog(ctx context.Context, entry agent.LogEntry) error {
	if strings.TrimSpace(entry.CreatedAt) == "" {
		entry.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_task_logs (id, task_id, seq, stream, event_type, text, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, entry.ID, entry.TaskID, entry.Seq, entry.Stream, entry.EventType, entry.Text, entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert agent_task_log: %w", err)
	}
	return nil
}

// ListAgentLogs returns ordered streaming log lines for a task.
func (s *Store) ListAgentLogs(ctx context.Context, taskID string) ([]agent.LogEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, seq, stream, event_type, text, created_at
		FROM agent_task_logs
		WHERE task_id = ?
		ORDER BY seq ASC, rowid ASC;
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list agent_task_logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]agent.LogEntry, 0)
	for rows.Next() {
		var l agent.LogEntry
		if err := rows.Scan(&l.ID, &l.TaskID, &l.Seq, &l.Stream, &l.EventType, &l.Text, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan agent_task_log: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// RecoverInterruptedAgentTasks marks any task left running or pending from an abrupt shutdown
// as interrupted_uncertain so the operator can inspect/recover its workDir before retrying.
func (s *Store) RecoverInterruptedAgentTasks(ctx context.Context) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		UPDATE agent_tasks
		SET status = ?,
			error_code = 'interrupted',
			error_message = 'Execução interrompida antes da conclusão (app fechado ou reiniciado). Verifique se o arquivo foi salvo antes de repetir.',
			updated_at = ?,
			finished_at = ?
		WHERE status IN (?, ?);
	`, string(agent.StatusInterruptedUncertain), now, now, string(agent.StatusRunning), string(agent.StatusPending))
	if err != nil {
		return 0, fmt.Errorf("recover interrupted agent_tasks: %w", err)
	}
	return res.RowsAffected()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAgentTask(row rowScanner) (agent.Task, error) {
	var t agent.Task
	var reqMode, resMode, status string
	var recoverable int
	err := row.Scan(
		&t.ID, &t.DocumentID, &t.PageIndex, &reqMode, &resMode, &t.Prompt,
		&t.ConfirmationReason, &status, &t.ThreadID, &t.WorkDir, &t.OutputImagePath,
		&t.OutputAssetID, &t.OutputAssetURL, &recoverable, &t.InstructionsVersion,
		&t.ContextSummary, &t.ResultDocumentID, &t.ResultRevision, &t.SummaryText,
		&t.ErrorCode, &t.ErrorMessage, &t.StartedAt, &t.UpdatedAt, &t.FinishedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return agent.Task{}, err
		}
		return agent.Task{}, fmt.Errorf("scan agent_task: %w", err)
	}
	t.RequestedMode = agent.DeliveryMode(reqMode)
	t.ResolvedMode = agent.DeliveryMode(resMode)
	t.Status = agent.Status(status)
	t.ArtifactRecoverable = recoverable != 0
	return t, nil
}
