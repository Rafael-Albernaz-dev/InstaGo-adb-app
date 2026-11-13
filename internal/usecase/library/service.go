package library

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
)

// Service coordinates company profile management, base import, managed library browsing,
// integrity checks, and portable .zip backups.
type Service struct {
	store           *sqlite.Store
	libraryDir      string
	backupsDir      string
	exportsDir      string
	creativeBaseDir string
	skillsDir       string
	appVersion      string
	openDirFn       func(dir string) error
	importMu        sync.Mutex
}

// NewService constructs the R1 library & profile use-case service.
func NewService(
	store *sqlite.Store,
	libraryDir string,
	backupsDir string,
	exportsDir string,
	creativeBaseDir string,
	skillsDir string,
	appVersion string,
) *Service {
	return &Service{
		store:           store,
		libraryDir:      filepath.Clean(libraryDir),
		backupsDir:      filepath.Clean(backupsDir),
		exportsDir:      filepath.Clean(exportsDir),
		creativeBaseDir: filepath.Clean(creativeBaseDir),
		skillsDir:       skillsDir,
		appVersion:      appVersion,
		openDirFn:       defaultOpenDir,
	}
}

// SetOpenDirFunc overrides the OS folder opener (primarily for deterministic unit tests).
func (s *Service) SetOpenDirFunc(fn func(dir string) error) {
	if fn != nil {
		s.openDirFn = fn
	}
}

// LibraryDir returns the managed library root directory.
func (s *Service) LibraryDir() string {
	return s.libraryDir
}

// HasRuralSkill reports whether the imported Rural Conecta skill is available locally.
func (s *Service) HasRuralSkill() bool {
	candidates := []string{}
	if strings.TrimSpace(s.skillsDir) != "" {
		candidates = append(candidates, filepath.Join(s.skillsDir, "rural-conecta-instagram", "SKILL.md"))
	}
	if strings.TrimSpace(s.creativeBaseDir) != "" {
		candidates = append(candidates, filepath.Join(s.creativeBaseDir, ".agents", "skills", "rural-conecta-instagram", "SKILL.md"))
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

// EnsureInitialized imports the configured creative base automatically on first launch
// if no import run exists yet and the creative base directory is present.
func (s *Service) EnsureInitialized(ctx context.Context) error {
	s.importMu.Lock()
	defer s.importMu.Unlock()

	if err := os.MkdirAll(s.libraryDir, 0o750); err != nil {
		return fmt.Errorf("create library dir: %w", err)
	}
	if err := os.MkdirAll(s.backupsDir, 0o750); err != nil {
		return fmt.Errorf("create backups dir: %w", err)
	}

	summary, err := s.store.GetLatestImportSummary(ctx, library.DefaultCompanyID)
	if err != nil {
		return err
	}
	if summary != nil {
		return nil
	}

	if st, err := os.Stat(s.creativeBaseDir); err == nil && st.IsDir() {
		_, err := s.importBaseLocked(ctx, s.creativeBaseDir)
		return err
	}

	// Ensure a minimal default profile exists even if creativeBaseDir is absent.
	_, err = s.store.GetCompanyProfile(ctx, library.DefaultCompanyID)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.store.UpsertCompanyProfile(ctx, library.CompanyProfile{
			ID:              library.DefaultCompanyID,
			Brand:           "Rural Conecta",
			ProfileName:     "Rural Conecta | Internet Rural Paracatu-MG",
			InstagramHandle: "ruralconectamg",
			InstagramURL:    "https://www.instagram.com/ruralconectamg/",
			SiteURL:         "https://ruralconectamg.com.br/",
			CreativeBaseDir: s.creativeBaseDir,
			PaletteJSON:     "{}",
		})
	}
	return err
}

// ImportBase scans sourceDir, copies managed files into libraryDir with SHA-256 verification,
// parses dated offers and operational history as read-only records, and persists the import atomically.
func (s *Service) ImportBase(ctx context.Context, sourceDir string) (library.ImportSummary, error) {
	s.importMu.Lock()
	defer s.importMu.Unlock()
	return s.importBaseLocked(ctx, sourceDir)
}

func (s *Service) importBaseLocked(ctx context.Context, sourceDir string) (library.ImportSummary, error) {
	targetDir := strings.TrimSpace(sourceDir)
	if targetDir == "" {
		targetDir = s.creativeBaseDir
	}
	targetDir = filepath.Clean(targetDir)

	var existingPtr *library.CompanyProfile
	if existing, err := s.store.GetCompanyProfile(ctx, library.DefaultCompanyID); err == nil {
		existingPtr = &existing
	}

	bundle, err := s.buildImportBundle(targetDir, existingPtr)
	if err != nil {
		return library.ImportSummary{}, err
	}

	if err := s.store.SaveLibraryImport(
		ctx,
		bundle.Profile,
		bundle.Assets,
		bundle.Offers,
		bundle.History,
		bundle.Summary,
	); err != nil {
		return library.ImportSummary{}, err
	}

	s.creativeBaseDir = targetDir
	return bundle.Summary, nil
}

// GetOverview returns the active profile, latest import summary, filtered assets,
// dated offers, operational history, and per-classification totals.
func (s *Service) GetOverview(ctx context.Context, filter library.LibraryFilter) (library.LibraryOverview, error) {
	if err := s.EnsureInitialized(ctx); err != nil {
		return library.LibraryOverview{}, err
	}

	profile, err := s.store.GetCompanyProfile(ctx, library.DefaultCompanyID)
	if err != nil {
		return library.LibraryOverview{}, err
	}
	summary, err := s.store.GetLatestImportSummary(ctx, library.DefaultCompanyID)
	if err != nil {
		return library.LibraryOverview{}, err
	}
	allAssets, err := s.store.ListLibraryAssets(ctx, library.DefaultCompanyID, library.LibraryFilter{})
	if err != nil {
		return library.LibraryOverview{}, err
	}
	filteredAssets, err := s.store.ListLibraryAssets(ctx, library.DefaultCompanyID, filter)
	if err != nil {
		return library.LibraryOverview{}, err
	}
	offers, err := s.store.ListDatedOffers(ctx, library.DefaultCompanyID)
	if err != nil {
		return library.LibraryOverview{}, err
	}
	history, err := s.store.ListPublicationHistory(ctx, library.DefaultCompanyID)
	if err != nil {
		return library.LibraryOverview{}, err
	}

	counts := map[string]int{
		"all":            len(allAssets),
		"arte_unica":     0,
		"capa_auxiliar":  0,
		"cena":           0,
		"marca":          0,
		"capa_editorial": 0,
		"fonte":          0,
		"historico":      0,
		"referencia":     0,
	}
	for _, a := range allAssets {
		counts[a.Classification]++
	}

	return library.LibraryOverview{
		Profile:       profile,
		Summary:       summary,
		Assets:        filteredAssets,
		Offers:        offers,
		History:       history,
		CountsByClass: counts,
		LibraryDir:    s.libraryDir,
		BackupsDir:    s.backupsDir,
	}, nil
}

// UpdateProfile persists operator edits to the company profile.
func (s *Service) UpdateProfile(ctx context.Context, profile library.CompanyProfile) (library.CompanyProfile, error) {
	if strings.TrimSpace(profile.ID) == "" {
		profile.ID = library.DefaultCompanyID
	}
	profile.CustomEdited = true
	profile.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(profile.CreativeBaseDir) != "" {
		profile.CreativeBaseDir = filepath.Clean(profile.CreativeBaseDir)
		s.creativeBaseDir = profile.CreativeBaseDir
	}
	return s.store.UpsertCompanyProfile(ctx, profile)
}

// VerifyIntegrity checks every managed asset in libraryDir against its recorded SHA-256 hash.
func (s *Service) VerifyIntegrity(ctx context.Context) (library.IntegrityReport, error) {
	assetsList, err := s.store.ListLibraryAssets(ctx, library.DefaultCompanyID, library.LibraryFilter{})
	if err != nil {
		return library.IntegrityReport{}, err
	}

	report := library.IntegrityReport{
		CheckedAt:   time.Now().UTC().Format(time.RFC3339),
		TotalAssets: len(assetsList),
		Issues:      make([]library.IntegrityIssue, 0),
	}

	for _, a := range assetsList {
		abs := filepath.Clean(filepath.Join(s.libraryDir, filepath.FromSlash(a.RelPath)))
		if !strings.HasPrefix(abs, s.libraryDir+string(filepath.Separator)) {
			report.CorruptedCount++
			report.Issues = append(report.Issues, library.IntegrityIssue{
				AssetID:        a.ID,
				Title:          a.Title,
				RelPath:        a.RelPath,
				ExpectedSHA256: a.SHA256,
				Reason:         "read_error",
			})
			continue
		}
		actual, err := hashFileSHA256(abs)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				report.MissingCount++
				report.Issues = append(report.Issues, library.IntegrityIssue{
					AssetID:        a.ID,
					Title:          a.Title,
					RelPath:        a.RelPath,
					ExpectedSHA256: a.SHA256,
					Reason:         "missing",
				})
			} else {
				report.CorruptedCount++
				report.Issues = append(report.Issues, library.IntegrityIssue{
					AssetID:        a.ID,
					Title:          a.Title,
					RelPath:        a.RelPath,
					ExpectedSHA256: a.SHA256,
					Reason:         "read_error",
				})
			}
			continue
		}
		if actual != a.SHA256 {
			report.CorruptedCount++
			report.Issues = append(report.Issues, library.IntegrityIssue{
				AssetID:        a.ID,
				Title:          a.Title,
				RelPath:        a.RelPath,
				ExpectedSHA256: a.SHA256,
				ActualSHA256:   actual,
				Reason:         "sha256_mismatch",
			})
			continue
		}
		report.OKCount++
	}

	return report, nil
}

type backupManifest struct {
	AppName       string            `json:"appName"`
	AppVersion    string            `json:"appVersion"`
	CompanyID     string            `json:"companyId"`
	CreatedAt     string            `json:"createdAt"`
	ArchivedFiles int               `json:"archivedFiles"`
	FilesSHA256   map[string]string `json:"filesSha256"`
}

// CreateBackup exports an atomic portable .zip containing a SQLite snapshot, managed library,
// exports directory, and a SHA-256 manifest.
func (s *Service) CreateBackup(ctx context.Context) (library.BackupResult, error) {
	if err := os.MkdirAll(s.backupsDir, 0o750); err != nil {
		return library.BackupResult{}, fmt.Errorf("create backups dir: %w", err)
	}

	now := time.Now().UTC()
	fileName := fmt.Sprintf("instago-backup-%s.zip", now.Format("20060102-150405"))
	destZip := filepath.Join(s.backupsDir, fileName)
	tmpZip := destZip + ".tmp-" + uuid.NewString()[:8]

	tmpDBDir, err := os.MkdirTemp("", "instago-db-snap-*")
	if err != nil {
		return library.BackupResult{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDBDir) }()

	snapDBPath := filepath.Join(tmpDBDir, "instago.db")
	if err := s.store.SnapshotDB(ctx, snapDBPath); err != nil {
		return library.BackupResult{}, err
	}

	zf, err := os.OpenFile(tmpZip, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return library.BackupResult{}, fmt.Errorf("create temp backup zip: %w", err)
	}

	zw := zip.NewWriter(zf)
	filesSHA := make(map[string]string)

	addToZip := func(archivePath string, diskPath string) error {
		f, err := os.Open(diskPath)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

		info, err := f.Stat()
		if err != nil {
			return err
		}
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(archivePath)
		hdr.Method = zip.Deflate

		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		h := sha256.New()
		if _, err := io.Copy(io.MultiWriter(w, h), f); err != nil {
			return err
		}
		filesSHA[hdr.Name] = hex.EncodeToString(h.Sum(nil))
		return nil
	}

	cleanupOnErr := func(cause error) (library.BackupResult, error) {
		_ = zw.Close()
		_ = zf.Close()
		_ = os.Remove(tmpZip)
		return library.BackupResult{}, cause
	}

	if err := addToZip("database/instago.db", snapDBPath); err != nil {
		return cleanupOnErr(fmt.Errorf("archive database snapshot: %w", err))
	}

	walkAndArchive := func(rootDir string, prefix string) error {
		if st, err := os.Stat(rootDir); err != nil || !st.IsDir() {
			return nil
		}
		return filepath.WalkDir(rootDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err
			}
			return addToZip(filepath.ToSlash(filepath.Join(prefix, rel)), path)
		})
	}

	if err := walkAndArchive(s.libraryDir, "library"); err != nil {
		return cleanupOnErr(fmt.Errorf("archive library: %w", err))
	}
	if err := walkAndArchive(s.exportsDir, "exports"); err != nil {
		return cleanupOnErr(fmt.Errorf("archive exports: %w", err))
	}

	manifest := backupManifest{
		AppName:       "InstaGo",
		AppVersion:    s.appVersion,
		CompanyID:     library.DefaultCompanyID,
		CreatedAt:     now.Format(time.RFC3339),
		ArchivedFiles: len(filesSHA) + 1,
		FilesSHA256:   filesSHA,
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return cleanupOnErr(err)
	}
	mw, err := zw.Create("manifest.json")
	if err != nil {
		return cleanupOnErr(err)
	}
	if _, err := mw.Write(manifestBytes); err != nil {
		return cleanupOnErr(err)
	}

	if err := zw.Close(); err != nil {
		_ = zf.Close()
		_ = os.Remove(tmpZip)
		return library.BackupResult{}, fmt.Errorf("close zip writer: %w", err)
	}
	if err := zf.Close(); err != nil {
		_ = os.Remove(tmpZip)
		return library.BackupResult{}, fmt.Errorf("close zip file: %w", err)
	}
	if err := os.Rename(tmpZip, destZip); err != nil {
		_ = os.Remove(tmpZip)
		return library.BackupResult{}, fmt.Errorf("commit backup zip: %w", err)
	}

	zipHash, err := hashFileSHA256(destZip)
	if err != nil {
		return library.BackupResult{}, err
	}
	st, err := os.Stat(destZip)
	if err != nil {
		return library.BackupResult{}, err
	}

	return library.BackupResult{
		FilePath:      destZip,
		FileName:      fileName,
		Bytes:         st.Size(),
		SHA256:        zipHash,
		ArchivedFiles: manifest.ArchivedFiles,
		CreatedAt:     manifest.CreatedAt,
	}, nil
}

// OpenFolder opens a safe, allowlisted directory in the OS file manager without accepting arbitrary shell paths.
func (s *Service) OpenFolder(ctx context.Context, scope string, assetID string) (string, error) {
	var targetDir string
	switch scope {
	case "library":
		targetDir = s.libraryDir
	case "backups":
		targetDir = s.backupsDir
	case "exports":
		targetDir = s.exportsDir
	case "creative_base":
		targetDir = s.creativeBaseDir
	case "asset_dir":
		if strings.TrimSpace(assetID) == "" {
			return "", errors.New("assetId is required for asset_dir scope")
		}
		asset, err := s.store.GetLibraryAsset(ctx, assetID)
		if err != nil {
			return "", fmt.Errorf("load asset %s: %w", assetID, err)
		}
		absAsset := filepath.Clean(filepath.Join(s.libraryDir, filepath.FromSlash(asset.RelPath)))
		if !strings.HasPrefix(absAsset, s.libraryDir+string(filepath.Separator)) {
			return "", errors.New("invalid asset path")
		}
		targetDir = filepath.Dir(absAsset)
	default:
		return "", fmt.Errorf("unsupported folder scope: %s", scope)
	}

	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		if scope == "creative_base" {
			return "", fmt.Errorf("creative base directory unavailable: %w", err)
		}
		return "", err
	}
	if err := s.openDirFn(targetDir); err != nil {
		return targetDir, fmt.Errorf("open folder %s: %w", targetDir, err)
	}
	return targetDir, nil
}

func defaultOpenDir(dir string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer", dir)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}

func hashFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
