CREATE TABLE IF NOT EXISTS company_profiles (
  id TEXT PRIMARY KEY,
  brand TEXT NOT NULL,
  profile_name TEXT NOT NULL DEFAULT '',
  instagram_handle TEXT NOT NULL,
  instagram_url TEXT NOT NULL DEFAULT '',
  site_url TEXT NOT NULL DEFAULT '',
  cameras_url TEXT NOT NULL DEFAULT '',
  phone_display TEXT NOT NULL DEFAULT '',
  phone_e164 TEXT NOT NULL DEFAULT '',
  whatsapp_link TEXT NOT NULL DEFAULT '',
  region TEXT NOT NULL DEFAULT '',
  client_audience TEXT NOT NULL DEFAULT '',
  approved_visual_version TEXT NOT NULL DEFAULT '',
  approval_date TEXT NOT NULL DEFAULT '',
  creative_base_dir TEXT NOT NULL DEFAULT '',
  palette_json TEXT NOT NULL DEFAULT '{}',
  bio_text TEXT NOT NULL DEFAULT '',
  custom_edited INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS library_assets (
  id TEXT PRIMARY KEY,
  company_id TEXT NOT NULL,
  rel_path TEXT NOT NULL,
  thumb_rel_path TEXT NOT NULL DEFAULT '',
  source_rel_path TEXT NOT NULL,
  file_name TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  media_kind TEXT NOT NULL,
  classification TEXT NOT NULL,
  sub_category TEXT NOT NULL DEFAULT '',
  format TEXT NOT NULL DEFAULT 'other',
  title TEXT NOT NULL,
  caption TEXT NOT NULL DEFAULT '',
  piece_id TEXT NOT NULL DEFAULT '',
  page_number INTEGER NOT NULL DEFAULT 0,
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  bytes INTEGER NOT NULL DEFAULT 0,
  sha256 TEXT NOT NULL,
  is_unique_approved INTEGER NOT NULL DEFAULT 0,
  is_editable_model INTEGER NOT NULL DEFAULT 0,
  linked_document_id TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  imported_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_library_assets_company_class
  ON library_assets(company_id, classification);

CREATE INDEX IF NOT EXISTS idx_library_assets_sha256
  ON library_assets(sha256);

CREATE TABLE IF NOT EXISTS dated_offers (
  id TEXT PRIMARY KEY,
  company_id TEXT NOT NULL,
  category TEXT NOT NULL,
  title TEXT NOT NULL,
  promo_price_cents INTEGER NOT NULL DEFAULT 0,
  promo_period TEXT NOT NULL DEFAULT '',
  after_price_cents INTEGER NOT NULL DEFAULT 0,
  install_fee_cents INTEGER NOT NULL DEFAULT 0,
  speed_summary TEXT NOT NULL DEFAULT '',
  services_summary TEXT NOT NULL DEFAULT '',
  restrictions TEXT NOT NULL DEFAULT '',
  campaign_name TEXT NOT NULL DEFAULT '',
  valid_until TEXT NOT NULL DEFAULT '',
  checked_at TEXT NOT NULL,
  source_ref TEXT NOT NULL,
  read_only INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS publication_history (
  id TEXT PRIMARY KEY,
  company_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  piece_key TEXT NOT NULL,
  title TEXT NOT NULL,
  position INTEGER NOT NULL DEFAULT 0,
  pinned INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  instagram_url TEXT NOT NULL DEFAULT '',
  caption TEXT NOT NULL DEFAULT '',
  page_count INTEGER NOT NULL DEFAULT 0,
  asset_ids_json TEXT NOT NULL DEFAULT '[]',
  notes TEXT NOT NULL DEFAULT '',
  observed_at TEXT NOT NULL,
  source_ref TEXT NOT NULL,
  read_only INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS import_runs (
  id TEXT PRIMARY KEY,
  company_id TEXT NOT NULL,
  source_dir TEXT NOT NULL,
  unique_arts_count INTEGER NOT NULL DEFAULT 0,
  auxiliary_covers_count INTEGER NOT NULL DEFAULT 0,
  total_assets_count INTEGER NOT NULL DEFAULT 0,
  offers_count INTEGER NOT NULL DEFAULT 0,
  history_count INTEGER NOT NULL DEFAULT 0,
  editable_models_count INTEGER NOT NULL DEFAULT 0,
  reference_only_count INTEGER NOT NULL DEFAULT 0,
  fonts_count INTEGER NOT NULL DEFAULT 0,
  skill_loaded INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL
);
