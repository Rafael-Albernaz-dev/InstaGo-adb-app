ALTER TABLE documents ADD COLUMN kind TEXT NOT NULL DEFAULT 'piece';
ALTER TABLE documents ADD COLUMN piece_type TEXT NOT NULL DEFAULT 'feed_single';
ALTER TABLE documents ADD COLUMN approved_revision INTEGER NOT NULL DEFAULT 0;
ALTER TABLE documents ADD COLUMN offer_id TEXT NOT NULL DEFAULT '';

ALTER TABLE document_revisions ADD COLUMN status TEXT NOT NULL DEFAULT 'draft';

ALTER TABLE exports ADD COLUMN page_index INTEGER NOT NULL DEFAULT 0;
ALTER TABLE exports ADD COLUMN bundle_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS export_bundles (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  document_status TEXT NOT NULL,
  format TEXT NOT NULL,
  piece_type TEXT NOT NULL,
  bundle_dir TEXT NOT NULL,
  caption_path TEXT NOT NULL,
  manifest_path TEXT NOT NULL,
  page_count INTEGER NOT NULL,
  file_paths_json TEXT NOT NULL,
  is_draft_warning INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_export_bundles_doc
  ON export_bundles(document_id, created_at DESC);
