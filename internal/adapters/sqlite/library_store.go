package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
)

// GetCompanyProfile fetches the persisted profile for companyID.
func (s *Store) GetCompanyProfile(ctx context.Context, companyID string) (library.CompanyProfile, error) {
	var p library.CompanyProfile
	var customEdited int
	err := s.db.QueryRowContext(ctx, `
		SELECT
			id, brand, profile_name, instagram_handle, instagram_url,
			site_url, cameras_url, phone_display, phone_e164, whatsapp_link,
			region, client_audience, approved_visual_version, approval_date,
			creative_base_dir, palette_json, bio_text, custom_edited, updated_at
		FROM company_profiles
		WHERE id = ?
	`, companyID).Scan(
		&p.ID, &p.Brand, &p.ProfileName, &p.InstagramHandle, &p.InstagramURL,
		&p.SiteURL, &p.CamerasURL, &p.PhoneDisplay, &p.PhoneE164, &p.WhatsAppLink,
		&p.Region, &p.ClientAudience, &p.ApprovedVisualVersion, &p.ApprovalDate,
		&p.CreativeBaseDir, &p.PaletteJSON, &p.BioText, &customEdited, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return library.CompanyProfile{}, sql.ErrNoRows
		}
		return library.CompanyProfile{}, fmt.Errorf("query company profile %s: %w", companyID, err)
	}
	p.CustomEdited = customEdited != 0
	return p, nil
}

// UpsertCompanyProfile inserts or updates a company profile.
func (s *Store) UpsertCompanyProfile(ctx context.Context, p library.CompanyProfile) (library.CompanyProfile, error) {
	if err := p.Validate(); err != nil {
		return library.CompanyProfile{}, err
	}
	if strings.TrimSpace(p.UpdatedAt) == "" {
		p.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	customInt := 0
	if p.CustomEdited {
		customInt = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO company_profiles (
			id, brand, profile_name, instagram_handle, instagram_url,
			site_url, cameras_url, phone_display, phone_e164, whatsapp_link,
			region, client_audience, approved_visual_version, approval_date,
			creative_base_dir, palette_json, bio_text, custom_edited, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			brand = excluded.brand,
			profile_name = excluded.profile_name,
			instagram_handle = excluded.instagram_handle,
			instagram_url = excluded.instagram_url,
			site_url = excluded.site_url,
			cameras_url = excluded.cameras_url,
			phone_display = excluded.phone_display,
			phone_e164 = excluded.phone_e164,
			whatsapp_link = excluded.whatsapp_link,
			region = excluded.region,
			client_audience = excluded.client_audience,
			approved_visual_version = excluded.approved_visual_version,
			approval_date = excluded.approval_date,
			creative_base_dir = excluded.creative_base_dir,
			palette_json = excluded.palette_json,
			bio_text = excluded.bio_text,
			custom_edited = excluded.custom_edited,
			updated_at = excluded.updated_at
	`,
		p.ID, p.Brand, p.ProfileName, p.InstagramHandle, p.InstagramURL,
		p.SiteURL, p.CamerasURL, p.PhoneDisplay, p.PhoneE164, p.WhatsAppLink,
		p.Region, p.ClientAudience, p.ApprovedVisualVersion, p.ApprovalDate,
		p.CreativeBaseDir, p.PaletteJSON, p.BioText, customInt, p.UpdatedAt,
	)
	if err != nil {
		return library.CompanyProfile{}, fmt.Errorf("upsert company profile %s: %w", p.ID, err)
	}
	return p, nil
}

// SaveLibraryImport atomically replaces the company's imported library catalog,
// dated offers, and operational history while preserving editable documents and exports.
func (s *Store) SaveLibraryImport(
	ctx context.Context,
	profile library.CompanyProfile,
	assets []library.LibraryAsset,
	offers []library.DatedOffer,
	history []library.PublicationHistoryItem,
	summary library.ImportSummary,
) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin import tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	customInt := 0
	if profile.CustomEdited {
		customInt = 1
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO company_profiles (
			id, brand, profile_name, instagram_handle, instagram_url,
			site_url, cameras_url, phone_display, phone_e164, whatsapp_link,
			region, client_audience, approved_visual_version, approval_date,
			creative_base_dir, palette_json, bio_text, custom_edited, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			brand = excluded.brand,
			profile_name = excluded.profile_name,
			instagram_handle = excluded.instagram_handle,
			instagram_url = excluded.instagram_url,
			site_url = excluded.site_url,
			cameras_url = excluded.cameras_url,
			phone_display = excluded.phone_display,
			phone_e164 = excluded.phone_e164,
			whatsapp_link = excluded.whatsapp_link,
			region = excluded.region,
			client_audience = excluded.client_audience,
			approved_visual_version = excluded.approved_visual_version,
			approval_date = excluded.approval_date,
			creative_base_dir = excluded.creative_base_dir,
			palette_json = excluded.palette_json,
			bio_text = excluded.bio_text,
			custom_edited = excluded.custom_edited,
			updated_at = excluded.updated_at
	`,
		profile.ID, profile.Brand, profile.ProfileName, profile.InstagramHandle, profile.InstagramURL,
		profile.SiteURL, profile.CamerasURL, profile.PhoneDisplay, profile.PhoneE164, profile.WhatsAppLink,
		profile.Region, profile.ClientAudience, profile.ApprovedVisualVersion, profile.ApprovalDate,
		profile.CreativeBaseDir, profile.PaletteJSON, profile.BioText, customInt, profile.UpdatedAt,
	); err != nil {
		return fmt.Errorf("upsert profile in import: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM library_assets WHERE company_id = ?", profile.ID); err != nil {
		return fmt.Errorf("clear library_assets: %w", err)
	}
	assetStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO library_assets (
			id, company_id, rel_path, thumb_rel_path, source_rel_path,
			file_name, mime_type, media_kind, classification, sub_category,
			format, title, caption, piece_id, page_number, width, height, bytes,
			sha256, is_unique_approved, is_editable_model, linked_document_id,
			metadata_json, imported_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare asset insert: %w", err)
	}
	defer func() { _ = assetStmt.Close() }()

	for _, a := range assets {
		if _, err := assetStmt.ExecContext(ctx,
			a.ID, a.CompanyID, a.RelPath, a.ThumbRelPath, a.SourceRelPath,
			a.FileName, a.MimeType, a.MediaKind, a.Classification, a.SubCategory,
			a.Format, a.Title, a.Caption, a.PieceID, a.PageNumber, a.Width, a.Height, a.Bytes,
			a.SHA256, boolToInt(a.IsUniqueApproved), boolToInt(a.IsEditableModel), a.LinkedDocumentID,
			a.MetadataJSON, a.ImportedAt,
		); err != nil {
			return fmt.Errorf("insert asset %s: %w", a.ID, err)
		}
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM dated_offers WHERE company_id = ?", profile.ID); err != nil {
		return fmt.Errorf("clear dated_offers: %w", err)
	}
	offerStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO dated_offers (
			id, company_id, category, title, promo_price_cents, promo_period,
			after_price_cents, install_fee_cents, speed_summary, services_summary,
			restrictions, campaign_name, valid_until, checked_at, source_ref, read_only
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare offer insert: %w", err)
	}
	defer func() { _ = offerStmt.Close() }()

	for _, o := range offers {
		if _, err := offerStmt.ExecContext(ctx,
			o.ID, o.CompanyID, o.Category, o.Title, o.PromoPriceCents, o.PromoPeriod,
			o.AfterPriceCents, o.InstallFeeCents, o.SpeedSummary, o.ServicesSummary,
			o.Restrictions, o.CampaignName, o.ValidUntil, o.CheckedAt, o.SourceRef, boolToInt(o.ReadOnly),
		); err != nil {
			return fmt.Errorf("insert offer %s: %w", o.ID, err)
		}
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM publication_history WHERE company_id = ?", profile.ID); err != nil {
		return fmt.Errorf("clear publication_history: %w", err)
	}
	histStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO publication_history (
			id, company_id, kind, piece_key, title, position, pinned, status,
			instagram_url, caption, page_count, asset_ids_json, notes,
			observed_at, source_ref, read_only
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare history insert: %w", err)
	}
	defer func() { _ = histStmt.Close() }()

	for _, h := range history {
		if _, err := histStmt.ExecContext(ctx,
			h.ID, h.CompanyID, h.Kind, h.PieceKey, h.Title, h.Position, boolToInt(h.Pinned), h.Status,
			h.InstagramURL, h.Caption, h.PageCount, h.AssetIDsJSON, h.Notes,
			h.ObservedAt, h.SourceRef, boolToInt(h.ReadOnly),
		); err != nil {
			return fmt.Errorf("insert history %s: %w", h.ID, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO import_runs (
			id, company_id, source_dir, unique_arts_count, auxiliary_covers_count,
			total_assets_count, offers_count, history_count, editable_models_count,
			reference_only_count, fonts_count, skill_loaded, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		summary.RunID, summary.CompanyID, summary.SourceDir, summary.UniqueArtsCount, summary.AuxiliaryCoversCount,
		summary.TotalAssetsCount, summary.OffersCount, summary.HistoryCount, summary.EditableModelsCount,
		summary.ReferenceOnlyCount, summary.FontsCount, boolToInt(summary.SkillLoaded), summary.Status, summary.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert import_run: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit import tx: %w", err)
	}
	return nil
}

// ListLibraryAssets returns assets for a company filtered by optional query/classification/format/subCategory.
func (s *Store) ListLibraryAssets(ctx context.Context, companyID string, filter library.LibraryFilter) ([]library.LibraryAsset, error) {
	clauses := []string{"company_id = ?"}
	args := []any{companyID}

	if c := strings.TrimSpace(filter.Classification); c != "" && c != "all" {
		clauses = append(clauses, "classification = ?")
		args = append(args, c)
	}
	if f := strings.TrimSpace(filter.Format); f != "" && f != "all" {
		clauses = append(clauses, "format = ?")
		args = append(args, f)
	}
	if sc := strings.TrimSpace(filter.SubCategory); sc != "" && sc != "all" {
		clauses = append(clauses, "sub_category = ?")
		args = append(args, sc)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		clauses = append(clauses, "(LOWER(title) LIKE ? OR LOWER(file_name) LIKE ? OR LOWER(caption) LIKE ? OR LOWER(piece_id) LIKE ? OR LOWER(sub_category) LIKE ?)")
		args = append(args, like, like, like, like, like)
	}

	query := fmt.Sprintf(`
		SELECT
			id, company_id, rel_path, thumb_rel_path, source_rel_path,
			file_name, mime_type, media_kind, classification, sub_category,
			format, title, caption, piece_id, page_number, width, height, bytes,
			sha256, is_unique_approved, is_editable_model, linked_document_id,
			metadata_json, imported_at
		FROM library_assets
		WHERE %s
		ORDER BY is_unique_approved DESC, classification ASC, sub_category ASC, piece_id ASC, page_number ASC, file_name ASC
	`, strings.Join(clauses, " AND "))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list library_assets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]library.LibraryAsset, 0)
	for rows.Next() {
		var a library.LibraryAsset
		var isUnique, isEditable int
		if err := rows.Scan(
			&a.ID, &a.CompanyID, &a.RelPath, &a.ThumbRelPath, &a.SourceRelPath,
			&a.FileName, &a.MimeType, &a.MediaKind, &a.Classification, &a.SubCategory,
			&a.Format, &a.Title, &a.Caption, &a.PieceID, &a.PageNumber, &a.Width, &a.Height, &a.Bytes,
			&a.SHA256, &isUnique, &isEditable, &a.LinkedDocumentID,
			&a.MetadataJSON, &a.ImportedAt,
		); err != nil {
			return nil, fmt.Errorf("scan library_asset: %w", err)
		}
		a.IsUniqueApproved = isUnique != 0
		a.IsEditableModel = isEditable != 0
		if a.RelPath != "" {
			a.AssetURL = "/library-assets/" + strings.TrimPrefix(a.RelPath, "/")
		}
		if a.ThumbRelPath != "" {
			a.ThumbURL = "/library-assets/" + strings.TrimPrefix(a.ThumbRelPath, "/")
		} else if a.MediaKind == "image" {
			a.ThumbURL = a.AssetURL
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetLibraryAsset fetches a single managed asset by ID.
func (s *Store) GetLibraryAsset(ctx context.Context, id string) (library.LibraryAsset, error) {
	var a library.LibraryAsset
	var isUnique, isEditable int
	err := s.db.QueryRowContext(ctx, `
		SELECT
			id, company_id, rel_path, thumb_rel_path, source_rel_path,
			file_name, mime_type, media_kind, classification, sub_category,
			format, title, caption, piece_id, page_number, width, height, bytes,
			sha256, is_unique_approved, is_editable_model, linked_document_id,
			metadata_json, imported_at
		FROM library_assets
		WHERE id = ?
	`, id).Scan(
		&a.ID, &a.CompanyID, &a.RelPath, &a.ThumbRelPath, &a.SourceRelPath,
		&a.FileName, &a.MimeType, &a.MediaKind, &a.Classification, &a.SubCategory,
		&a.Format, &a.Title, &a.Caption, &a.PieceID, &a.PageNumber, &a.Width, &a.Height, &a.Bytes,
		&a.SHA256, &isUnique, &isEditable, &a.LinkedDocumentID,
		&a.MetadataJSON, &a.ImportedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return library.LibraryAsset{}, sql.ErrNoRows
		}
		return library.LibraryAsset{}, fmt.Errorf("query library_asset %s: %w", id, err)
	}
	a.IsUniqueApproved = isUnique != 0
	a.IsEditableModel = isEditable != 0
	if a.RelPath != "" {
		a.AssetURL = "/library-assets/" + strings.TrimPrefix(a.RelPath, "/")
	}
	if a.ThumbRelPath != "" {
		a.ThumbURL = "/library-assets/" + strings.TrimPrefix(a.ThumbRelPath, "/")
	} else if a.MediaKind == "image" {
		a.ThumbURL = a.AssetURL
	}
	return a, nil
}

// ListDatedOffers returns all imported read-only dated offers for a company.
func (s *Store) ListDatedOffers(ctx context.Context, companyID string) ([]library.DatedOffer, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id, company_id, category, title, promo_price_cents, promo_period,
			after_price_cents, install_fee_cents, speed_summary, services_summary,
			restrictions, campaign_name, valid_until, checked_at, source_ref, read_only
		FROM dated_offers
		WHERE company_id = ?
		ORDER BY category ASC, promo_price_cents ASC, after_price_cents ASC, id ASC
	`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list dated_offers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]library.DatedOffer, 0)
	for rows.Next() {
		var o library.DatedOffer
		var ro int
		if err := rows.Scan(
			&o.ID, &o.CompanyID, &o.Category, &o.Title, &o.PromoPriceCents, &o.PromoPeriod,
			&o.AfterPriceCents, &o.InstallFeeCents, &o.SpeedSummary, &o.ServicesSummary,
			&o.Restrictions, &o.CampaignName, &o.ValidUntil, &o.CheckedAt, &o.SourceRef, &ro,
		); err != nil {
			return nil, fmt.Errorf("scan dated_offer: %w", err)
		}
		o.ReadOnly = ro != 0
		items = append(items, o)
	}
	return items, rows.Err()
}

// ListPublicationHistory returns read-only historical publication records.
func (s *Store) ListPublicationHistory(ctx context.Context, companyID string) ([]library.PublicationHistoryItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id, company_id, kind, piece_key, title, position, pinned, status,
			instagram_url, caption, page_count, asset_ids_json, notes,
			observed_at, source_ref, read_only
		FROM publication_history
		WHERE company_id = ?
		ORDER BY kind ASC, position ASC, title ASC
	`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list publication_history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]library.PublicationHistoryItem, 0)
	for rows.Next() {
		var h library.PublicationHistoryItem
		var pinned, ro int
		if err := rows.Scan(
			&h.ID, &h.CompanyID, &h.Kind, &h.PieceKey, &h.Title, &h.Position, &pinned, &h.Status,
			&h.InstagramURL, &h.Caption, &h.PageCount, &h.AssetIDsJSON, &h.Notes,
			&h.ObservedAt, &h.SourceRef, &ro,
		); err != nil {
			return nil, fmt.Errorf("scan publication_history: %w", err)
		}
		h.Pinned = pinned != 0
		h.ReadOnly = ro != 0
		items = append(items, h)
	}
	return items, rows.Err()
}

// GetLatestImportSummary returns the most recent base import run summary, or nil if none.
func (s *Store) GetLatestImportSummary(ctx context.Context, companyID string) (*library.ImportSummary, error) {
	var sum library.ImportSummary
	var skillLoaded int
	err := s.db.QueryRowContext(ctx, `
		SELECT
			id, company_id, source_dir, unique_arts_count, auxiliary_covers_count,
			total_assets_count, offers_count, history_count, editable_models_count,
			reference_only_count, fonts_count, skill_loaded, status, created_at
		FROM import_runs
		WHERE company_id = ?
		ORDER BY created_at DESC, rowid DESC
		LIMIT 1
	`, companyID).Scan(
		&sum.RunID, &sum.CompanyID, &sum.SourceDir, &sum.UniqueArtsCount, &sum.AuxiliaryCoversCount,
		&sum.TotalAssetsCount, &sum.OffersCount, &sum.HistoryCount, &sum.EditableModelsCount,
		&sum.ReferenceOnlyCount, &sum.FontsCount, &skillLoaded, &sum.Status, &sum.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest import_run: %w", err)
	}
	sum.SkillLoaded = skillLoaded != 0
	return &sum, nil
}

// SnapshotDB writes an atomic, self-contained SQLite copy at destPath via VACUUM INTO.
func (s *Store) SnapshotDB(ctx context.Context, destPath string) error {
	clean := filepath.Clean(destPath)
	if err := os.MkdirAll(filepath.Dir(clean), 0o750); err != nil {
		return fmt.Errorf("create snapshot dir: %w", err)
	}
	_ = os.Remove(clean)
	escaped := strings.ReplaceAll(clean, "'", "''")
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", escaped)); err != nil {
		return fmt.Errorf("vacuum snapshot: %w", err)
	}
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
