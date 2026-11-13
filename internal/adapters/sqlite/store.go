package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// ErrApprovedDocumentImmutable indicates an attempt to mutate an approved revision directly.
var ErrApprovedDocumentImmutable = errors.New("documento aprovado está protegido; crie uma nova versão em rascunho para editar")

// ExportRecord stores verified raster export metadata.
type ExportRecord struct {
	ID         string `json:"id"`
	DocumentID string `json:"documentId"`
	Revision   int    `json:"revision"`
	Format     string `json:"format"`
	PageIndex  int    `json:"pageIndex"`
	BundleID   string `json:"bundleId,omitempty"`
	FilePath   string `json:"filePath"`
	SHA256     string `json:"sha256"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Bytes      int64  `json:"bytes"`
	CreatedAt  string `json:"createdAt"`
}

// ExportBundleRecord summarizes a multi-page or single-page export folder (images + caption + manifest).
type ExportBundleRecord struct {
	ID             string   `json:"id"`
	DocumentID     string   `json:"documentId"`
	Revision       int      `json:"revision"`
	DocumentStatus string   `json:"documentStatus"`
	Format         string   `json:"format"`
	PieceType      string   `json:"pieceType"`
	BundleDir      string   `json:"bundleDir"`
	CaptionPath    string   `json:"captionPath"`
	ManifestPath   string   `json:"manifestPath"`
	PageCount      int      `json:"pageCount"`
	FilePaths      []string `json:"filePaths"`
	IsDraftWarning bool     `json:"isDraftWarning"`
	CreatedAt      string   `json:"createdAt"`
}

// RevisionSummary lists a persisted historical snapshot without transferring full JSON.
type RevisionSummary struct {
	DocumentID string `json:"documentId"`
	Revision   int    `json:"revision"`
	Note       string `json:"note"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
}

// RevisionRecord contains a full historical document snapshot for comparison or restoration.
type RevisionRecord struct {
	RevisionSummary
	Document document.Document `json:"document"`
}

// Store persists InstaGo documents, revisions, and verified exports in SQLite.
type Store struct {
	db     *sql.DB
	dbPath string
}

// Open initializes SQLite at dbPath and runs embedded schema migrations.
func Open(ctx context.Context, dbPath string) (*Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, errors.New("dbPath is required")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o750); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("sqlite %s: %w", pragma, err)
		}
	}

	s := &Store{db: db, dbPath: dbPath}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Path returns the resolved SQLite file path.
func (s *Store) Path() string {
	return s.dbPath
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix := strings.SplitN(entry.Name(), "_", 2)[0]
		ver, err := strconv.Atoi(prefix)
		if err != nil {
			return fmt.Errorf("invalid migration filename %q: %w", entry.Name(), err)
		}
		var exists int
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version = ?", ver).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %d: %w", ver, err)
		}
		if exists > 0 {
			continue
		}
		raw, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx, string(raw)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec migration %s: %w", entry.Name(), err)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)", ver, entry.Name(), now); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// GetDocument fetches the latest persisted state of a document.
func (s *Store) GetDocument(ctx context.Context, id string) (document.Document, error) {
	var payload string
	err := s.db.QueryRowContext(ctx, "SELECT payload_json FROM documents WHERE id = ?", id).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return document.Document{}, sql.ErrNoRows
		}
		return document.Document{}, fmt.Errorf("query document %s: %w", id, err)
	}
	var doc document.Document
	if err := json.Unmarshal([]byte(payload), &doc); err != nil {
		return document.Document{}, fmt.Errorf("decode document %s: %w", id, err)
	}
	doc.NormalizeDefaults()
	return doc, nil
}

// ListDocuments returns all persisted documents ordered by templates vs pieces and updated time.
func (s *Store) ListDocuments(ctx context.Context) ([]document.Document, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT payload_json FROM documents ORDER BY kind ASC, updated_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	docs := make([]document.Document, 0)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		var doc document.Document
		if err := json.Unmarshal([]byte(payload), &doc); err != nil {
			return nil, fmt.Errorf("decode document: %w", err)
		}
		doc.NormalizeDefaults()
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// SaveDraft persists the active working draft without adding a new numbered revision snapshot.
// If the document does not exist yet, it creates revision 1 cleanly.
// Refuses direct edits when the persisted document is currently marked "approved".
func (s *Store) SaveDraft(ctx context.Context, doc document.Document) (document.Document, error) {
	doc.NormalizeDefaults()
	if err := doc.Validate(); err != nil {
		return document.Document{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return document.Document{}, fmt.Errorf("begin save draft: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339)
	var existingRev int
	var existingApprovedRev int
	var existingStatus string
	var existingCreated string
	err = tx.QueryRowContext(
		ctx,
		"SELECT revision, approved_revision, status, created_at FROM documents WHERE id = ?",
		doc.ID,
	).Scan(&existingRev, &existingApprovedRev, &existingStatus, &existingCreated)

	isNew := false
	switch {
	case errors.Is(err, sql.ErrNoRows):
		isNew = true
		if doc.Revision < 1 {
			doc.Revision = 1
		}
		doc.CreatedAt = now
	case err != nil:
		return document.Document{}, fmt.Errorf("check existing draft document: %w", err)
	default:
		if existingStatus == "approved" {
			return document.Document{}, ErrApprovedDocumentImmutable
		}
		doc.Revision = existingRev
		doc.ApprovedRevision = existingApprovedRev
		doc.CreatedAt = existingCreated
		if doc.Status == "approved" {
			doc.Status = "draft"
		}
	}
	doc.UpdatedAt = now

	payload, err := upsertDocumentRowTx(ctx, tx, doc)
	if err != nil {
		return document.Document{}, err
	}
	if isNew {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO document_revisions (document_id, revision, note, status, payload_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?);
		`, doc.ID, doc.Revision, "rascunho inicial", doc.Status, payload, now); err != nil {
			return document.Document{}, fmt.Errorf("insert initial draft revision: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return document.Document{}, fmt.Errorf("commit save draft: %w", err)
	}
	return doc, nil
}

// SaveDocument validates and atomically persists the document and a new numbered revision snapshot.
// Direct saves on an already-approved document are blocked unless unlocked via StartDraftFromApproved.
func (s *Store) SaveDocument(ctx context.Context, doc document.Document, note string) (document.Document, error) {
	doc.NormalizeDefaults()
	if err := doc.Validate(); err != nil {
		return document.Document{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return document.Document{}, fmt.Errorf("begin save document: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339)
	var existingRev int
	var existingApprovedRev int
	var existingStatus string
	var existingCreated string
	err = tx.QueryRowContext(
		ctx,
		"SELECT revision, approved_revision, status, created_at FROM documents WHERE id = ?",
		doc.ID,
	).Scan(&existingRev, &existingApprovedRev, &existingStatus, &existingCreated)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		if doc.Revision < 1 {
			doc.Revision = 1
		}
		doc.CreatedAt = now
	case err != nil:
		return document.Document{}, fmt.Errorf("check existing document: %w", err)
	default:
		if existingStatus == "approved" {
			return document.Document{}, ErrApprovedDocumentImmutable
		}
		maxRev, err := maxRevisionTx(ctx, tx, doc.ID, existingRev)
		if err != nil {
			return document.Document{}, err
		}
		doc.Revision = maxRev + 1
		doc.ApprovedRevision = existingApprovedRev
		doc.CreatedAt = existingCreated
		// Editing/saving a new version never inherits "approved" automatically (RN-07 / RF-10).
		if doc.Status == "approved" {
			doc.Status = "draft"
		}
	}
	doc.UpdatedAt = now

	payload, err := upsertDocumentRowTx(ctx, tx, doc)
	if err != nil {
		return document.Document{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO document_revisions (document_id, revision, note, status, payload_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?);
	`, doc.ID, doc.Revision, note, doc.Status, payload, now); err != nil {
		return document.Document{}, fmt.Errorf("insert revision: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return document.Document{}, fmt.Errorf("commit save document: %w", err)
	}
	return doc, nil
}

// ApproveDocument snapshots the exact document version as "approved" and binds approvedRevision.
func (s *Store) ApproveDocument(ctx context.Context, doc document.Document, note string) (document.Document, error) {
	doc.NormalizeDefaults()
	if err := doc.Validate(); err != nil {
		return document.Document{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return document.Document{}, fmt.Errorf("begin approve document: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339)
	var existingRev int
	var existingCreated string
	err = tx.QueryRowContext(ctx, "SELECT revision, created_at FROM documents WHERE id = ?", doc.ID).Scan(&existingRev, &existingCreated)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if doc.Revision < 1 {
			doc.Revision = 1
		}
		doc.CreatedAt = now
	case err != nil:
		return document.Document{}, fmt.Errorf("check document for approval: %w", err)
	default:
		maxRev, err := maxRevisionTx(ctx, tx, doc.ID, existingRev)
		if err != nil {
			return document.Document{}, err
		}
		doc.Revision = maxRev + 1
		doc.CreatedAt = existingCreated
	}

	doc.Status = "approved"
	doc.ApprovedRevision = doc.Revision
	doc.UpdatedAt = now
	if strings.TrimSpace(note) == "" {
		note = fmt.Sprintf("versão v%d aprovada", doc.Revision)
	}

	payload, err := upsertDocumentRowTx(ctx, tx, doc)
	if err != nil {
		return document.Document{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO document_revisions (document_id, revision, note, status, payload_json, created_at)
		VALUES (?, ?, ?, 'approved', ?, ?);
	`, doc.ID, doc.Revision, note, payload, now); err != nil {
		return document.Document{}, fmt.Errorf("insert approved revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return document.Document{}, fmt.Errorf("commit approve document: %w", err)
	}
	return doc, nil
}

// StartDraftFromApproved clones the current approved document state into a new unlocked draft revision,
// keeping the approved snapshot intact inside document_revisions.
func (s *Store) StartDraftFromApproved(ctx context.Context, documentID string) (document.Document, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return document.Document{}, fmt.Errorf("begin start draft: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var payload string
	var status string
	var rev int
	var approvedRev int
	err = tx.QueryRowContext(
		ctx,
		"SELECT payload_json, status, revision, approved_revision FROM documents WHERE id = ?",
		documentID,
	).Scan(&payload, &status, &rev, &approvedRev)
	if err != nil {
		return document.Document{}, fmt.Errorf("load document %s: %w", documentID, err)
	}

	var doc document.Document
	if err := json.Unmarshal([]byte(payload), &doc); err != nil {
		return document.Document{}, fmt.Errorf("decode document %s: %w", documentID, err)
	}
	doc.NormalizeDefaults()

	if status != "approved" {
		return doc, nil
	}

	if approvedRev <= 0 {
		approvedRev = rev
	}
	maxRev, err := maxRevisionTx(ctx, tx, documentID, rev)
	if err != nil {
		return document.Document{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	doc.Status = "draft"
	doc.ApprovedRevision = approvedRev
	doc.Revision = maxRev + 1
	doc.UpdatedAt = now

	newPayload, err := upsertDocumentRowTx(ctx, tx, doc)
	if err != nil {
		return document.Document{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO document_revisions (document_id, revision, note, status, payload_json, created_at)
		VALUES (?, ?, ?, 'draft', ?, ?);
	`, doc.ID, doc.Revision, fmt.Sprintf("novo rascunho a partir da v%d aprovada", approvedRev), newPayload, now); err != nil {
		return document.Document{}, fmt.Errorf("insert draft branch revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return document.Document{}, fmt.Errorf("commit start draft: %w", err)
	}
	return doc, nil
}

// ListRevisions returns saved historical snapshots for documentID in descending order.
func (s *Store) ListRevisions(ctx context.Context, documentID string) ([]RevisionSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT document_id, revision, note, COALESCE(status, 'draft'), created_at
		FROM document_revisions
		WHERE document_id = ?
		ORDER BY revision DESC;
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("list revisions: %w", err)
	}
	defer rows.Close()

	out := make([]RevisionSummary, 0)
	for rows.Next() {
		var r RevisionSummary
		if err := rows.Scan(&r.DocumentID, &r.Revision, &r.Note, &r.Status, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan revision: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetRevision loads a full persisted historical snapshot for inspection or comparison.
func (s *Store) GetRevision(ctx context.Context, documentID string, revision int) (RevisionRecord, error) {
	var rec RevisionRecord
	var payload string
	err := s.db.QueryRowContext(ctx, `
		SELECT document_id, revision, note, COALESCE(status, 'draft'), payload_json, created_at
		FROM document_revisions
		WHERE document_id = ? AND revision = ?;
	`, documentID, revision).Scan(&rec.DocumentID, &rec.Revision, &rec.Note, &rec.Status, &payload, &rec.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RevisionRecord{}, sql.ErrNoRows
		}
		return RevisionRecord{}, fmt.Errorf("get revision %d: %w", revision, err)
	}
	if err := json.Unmarshal([]byte(payload), &rec.Document); err != nil {
		return RevisionRecord{}, fmt.Errorf("decode revision %d: %w", revision, err)
	}
	rec.Document.NormalizeDefaults()
	return rec, nil
}

// RestoreRevisionAsDraft copies a historical revision into a new draft revision on the live document.
func (s *Store) RestoreRevisionAsDraft(ctx context.Context, documentID string, targetRevision int) (document.Document, error) {
	current, err := s.GetDocument(ctx, documentID)
	if err != nil {
		return document.Document{}, err
	}
	if current.Status == "approved" {
		return document.Document{}, ErrApprovedDocumentImmutable
	}
	rev, err := s.GetRevision(ctx, documentID, targetRevision)
	if err != nil {
		return document.Document{}, err
	}
	restored := rev.Document
	restored.Status = "draft"
	restored.ApprovedRevision = current.ApprovedRevision
	return s.SaveDocument(ctx, restored, fmt.Sprintf("restaurado da v%d", targetRevision))
}

func maxRevisionTx(ctx context.Context, tx *sql.Tx, documentID string, fallback int) (int, error) {
	var maxRev int
	err := tx.QueryRowContext(
		ctx,
		"SELECT COALESCE(MAX(revision), 0) FROM document_revisions WHERE document_id = ?",
		documentID,
	).Scan(&maxRev)
	if err != nil {
		return 0, fmt.Errorf("query max revision: %w", err)
	}
	if fallback > maxRev {
		maxRev = fallback
	}
	return maxRev, nil
}

func upsertDocumentRowTx(ctx context.Context, tx *sql.Tx, doc document.Document) (string, error) {
	payloadBytes, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("encode document: %w", err)
	}
	payload := string(payloadBytes)

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO documents (
			id, title, company_id, offer_ref, offer_id, kind, piece_type, approved_revision, format,
			workspace_width, workspace_height, export_width, export_height, export_scale,
			revision, status, payload_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			company_id = excluded.company_id,
			offer_ref = excluded.offer_ref,
			offer_id = excluded.offer_id,
			kind = excluded.kind,
			piece_type = excluded.piece_type,
			approved_revision = excluded.approved_revision,
			format = excluded.format,
			workspace_width = excluded.workspace_width,
			workspace_height = excluded.workspace_height,
			export_width = excluded.export_width,
			export_height = excluded.export_height,
			export_scale = excluded.export_scale,
			revision = excluded.revision,
			status = excluded.status,
			payload_json = excluded.payload_json,
			updated_at = excluded.updated_at;
	`,
		doc.ID, doc.Title, doc.CompanyID, doc.OfferRef, doc.OfferID, doc.Kind, doc.PieceType, doc.ApprovedRevision, string(doc.Format),
		doc.WorkspaceWidth, doc.WorkspaceHeight, doc.ExportWidth, doc.ExportHeight, doc.ExportScale,
		doc.Revision, doc.Status, payload, doc.CreatedAt, doc.UpdatedAt,
	); err != nil {
		return "", fmt.Errorf("upsert document: %w", err)
	}
	return payload, nil
}

// RecordExport saves a verified raster export entry.
func (s *Store) RecordExport(ctx context.Context, rec ExportRecord) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO exports (id, document_id, revision, format, page_index, bundle_id, file_path, sha256, width, height, bytes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, rec.ID, rec.DocumentID, rec.Revision, rec.Format, rec.PageIndex, rec.BundleID, rec.FilePath, rec.SHA256, rec.Width, rec.Height, rec.Bytes, rec.CreatedAt); err != nil {
		return fmt.Errorf("record export: %w", err)
	}
	return nil
}

// ListExports returns recent export records for a document.
func (s *Store) ListExports(ctx context.Context, documentID string) ([]ExportRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, revision, format, COALESCE(page_index, 0), COALESCE(bundle_id, ''), file_path, sha256, width, height, bytes, created_at
		FROM exports
		WHERE document_id = ?
		ORDER BY created_at DESC
		LIMIT 40;
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("list exports: %w", err)
	}
	defer rows.Close()

	items := make([]ExportRecord, 0)
	for rows.Next() {
		var r ExportRecord
		if err := rows.Scan(&r.ID, &r.DocumentID, &r.Revision, &r.Format, &r.PageIndex, &r.BundleID, &r.FilePath, &r.SHA256, &r.Width, &r.Height, &r.Bytes, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan export: %w", err)
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// RecordExportBundle persists a multi-page or single-page export folder metadata record.
func (s *Store) RecordExportBundle(ctx context.Context, rec ExportBundleRecord) error {
	filesJSON, err := json.Marshal(rec.FilePaths)
	if err != nil {
		return fmt.Errorf("encode bundle file paths: %w", err)
	}
	draftFlag := 0
	if rec.IsDraftWarning {
		draftFlag = 1
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO export_bundles (
			id, document_id, revision, document_status, format, piece_type,
			bundle_dir, caption_path, manifest_path, page_count, file_paths_json, is_draft_warning, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, rec.ID, rec.DocumentID, rec.Revision, rec.DocumentStatus, rec.Format, rec.PieceType,
		rec.BundleDir, rec.CaptionPath, rec.ManifestPath, rec.PageCount, string(filesJSON), draftFlag, rec.CreatedAt); err != nil {
		return fmt.Errorf("record export bundle: %w", err)
	}
	return nil
}

// ListExportBundles returns recent bundle exports for documentID.
func (s *Store) ListExportBundles(ctx context.Context, documentID string) ([]ExportBundleRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, revision, document_status, format, piece_type,
		       bundle_dir, caption_path, manifest_path, page_count, file_paths_json, is_draft_warning, created_at
		FROM export_bundles
		WHERE document_id = ?
		ORDER BY created_at DESC
		LIMIT 20;
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("list export bundles: %w", err)
	}
	defer rows.Close()

	out := make([]ExportBundleRecord, 0)
	for rows.Next() {
		var r ExportBundleRecord
		var filesJSON string
		var draftFlag int
		if err := rows.Scan(
			&r.ID, &r.DocumentID, &r.Revision, &r.DocumentStatus, &r.Format, &r.PieceType,
			&r.BundleDir, &r.CaptionPath, &r.ManifestPath, &r.PageCount, &filesJSON, &draftFlag, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan export bundle: %w", err)
		}
		r.IsDraftWarning = draftFlag == 1
		if err := json.Unmarshal([]byte(filesJSON), &r.FilePaths); err != nil {
			r.FilePaths = make([]string, 0)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
