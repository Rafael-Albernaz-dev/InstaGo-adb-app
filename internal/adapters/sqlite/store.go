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

// ExportRecord stores verified raster export metadata.
type ExportRecord struct {
	ID         string `json:"id"`
	DocumentID string `json:"documentId"`
	Revision   int    `json:"revision"`
	Format     string `json:"format"`
	FilePath   string `json:"filePath"`
	SHA256     string `json:"sha256"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Bytes      int64  `json:"bytes"`
	CreatedAt  string `json:"createdAt"`
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
	return doc, nil
}

// ListDocuments returns all persisted documents ordered by most recently updated.
func (s *Store) ListDocuments(ctx context.Context) ([]document.Document, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT payload_json FROM documents ORDER BY updated_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var docs []document.Document
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		var doc document.Document
		if err := json.Unmarshal([]byte(payload), &doc); err != nil {
			return nil, fmt.Errorf("decode document: %w", err)
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// SaveDocument validates and atomically persists the document and a new revision row.
func (s *Store) SaveDocument(ctx context.Context, doc document.Document, note string) (document.Document, error) {
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
	var existingCreated string
	err = tx.QueryRowContext(ctx, "SELECT revision, created_at FROM documents WHERE id = ?", doc.ID).Scan(&existingRev, &existingCreated)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if doc.Revision < 1 {
			doc.Revision = 1
		}
		doc.CreatedAt = now
	case err != nil:
		return document.Document{}, fmt.Errorf("check existing document: %w", err)
	default:
		doc.Revision = existingRev + 1
		doc.CreatedAt = existingCreated
	}
	doc.UpdatedAt = now

	payloadBytes, err := json.Marshal(doc)
	if err != nil {
		return document.Document{}, fmt.Errorf("encode document: %w", err)
	}
	payload := string(payloadBytes)

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO documents (
			id, title, company_id, offer_ref, format,
			workspace_width, workspace_height, export_width, export_height, export_scale,
			revision, status, payload_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			company_id = excluded.company_id,
			offer_ref = excluded.offer_ref,
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
		doc.ID, doc.Title, doc.CompanyID, doc.OfferRef, string(doc.Format),
		doc.WorkspaceWidth, doc.WorkspaceHeight, doc.ExportWidth, doc.ExportHeight, doc.ExportScale,
		doc.Revision, doc.Status, payload, doc.CreatedAt, doc.UpdatedAt,
	); err != nil {
		return document.Document{}, fmt.Errorf("upsert document: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO document_revisions (document_id, revision, note, payload_json, created_at)
		VALUES (?, ?, ?, ?, ?);
	`, doc.ID, doc.Revision, note, payload, now); err != nil {
		return document.Document{}, fmt.Errorf("insert revision: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return document.Document{}, fmt.Errorf("commit save document: %w", err)
	}
	return doc, nil
}

// RecordExport saves a verified raster export entry.
func (s *Store) RecordExport(ctx context.Context, rec ExportRecord) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO exports (id, document_id, revision, format, file_path, sha256, width, height, bytes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, rec.ID, rec.DocumentID, rec.Revision, rec.Format, rec.FilePath, rec.SHA256, rec.Width, rec.Height, rec.Bytes, rec.CreatedAt); err != nil {
		return fmt.Errorf("record export: %w", err)
	}
	return nil
}

// ListExports returns recent export records for a document.
func (s *Store) ListExports(ctx context.Context, documentID string) ([]ExportRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, revision, format, file_path, sha256, width, height, bytes, created_at
		FROM exports
		WHERE document_id = ?
		ORDER BY created_at DESC
		LIMIT 20;
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("list exports: %w", err)
	}
	defer rows.Close()

	items := make([]ExportRecord, 0)
	for rows.Next() {
		var r ExportRecord
		if err := rows.Scan(&r.ID, &r.DocumentID, &r.Revision, &r.Format, &r.FilePath, &r.SHA256, &r.Width, &r.Height, &r.Bytes, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan export: %w", err)
		}
		items = append(items, r)
	}
	return items, rows.Err()
}
