CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS documents (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  company_id TEXT NOT NULL,
  offer_ref TEXT NOT NULL DEFAULT '',
  format TEXT NOT NULL,
  workspace_width REAL NOT NULL,
  workspace_height REAL NOT NULL,
  export_width INTEGER NOT NULL,
  export_height INTEGER NOT NULL,
  export_scale REAL NOT NULL,
  revision INTEGER NOT NULL,
  status TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS document_revisions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  document_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE,
  UNIQUE (document_id, revision)
);

CREATE TABLE IF NOT EXISTS exports (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  format TEXT NOT NULL,
  file_path TEXT NOT NULL,
  sha256 TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  bytes INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);
