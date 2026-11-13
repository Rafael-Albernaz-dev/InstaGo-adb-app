CREATE TABLE IF NOT EXISTS agent_tasks (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL DEFAULT '',
  page_index INTEGER NOT NULL DEFAULT 0,
  requested_mode TEXT NOT NULL DEFAULT 'auto',
  resolved_mode TEXT NOT NULL DEFAULT 'background_layer',
  prompt TEXT NOT NULL DEFAULT '',
  confirmation_reason TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  thread_id TEXT NOT NULL DEFAULT '',
  work_dir TEXT NOT NULL DEFAULT '',
  output_image_path TEXT NOT NULL DEFAULT '',
  output_asset_id TEXT NOT NULL DEFAULT '',
  output_asset_url TEXT NOT NULL DEFAULT '',
  artifact_recoverable INTEGER NOT NULL DEFAULT 0,
  instructions_version TEXT NOT NULL DEFAULT '',
  context_summary TEXT NOT NULL DEFAULT '',
  result_document_id TEXT NOT NULL DEFAULT '',
  result_revision INTEGER NOT NULL DEFAULT 0,
  summary_text TEXT NOT NULL DEFAULT '',
  error_code TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_agent_tasks_doc
  ON agent_tasks(document_id, started_at DESC);

CREATE TABLE IF NOT EXISTS agent_task_logs (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  seq INTEGER NOT NULL,
  stream TEXT NOT NULL,
  event_type TEXT NOT NULL DEFAULT '',
  text TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (task_id) REFERENCES agent_tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_task_logs_seq
  ON agent_task_logs(task_id, seq ASC);
