package library

import (
	"errors"
	"fmt"
	"strings"
)

const DefaultCompanyID = "rural-conecta"

// CompanyProfile stores the brand identity, contacts, and creative source configuration.
type CompanyProfile struct {
	ID                    string `json:"id"`
	Brand                 string `json:"brand"`
	ProfileName           string `json:"profileName"`
	InstagramHandle       string `json:"instagramHandle"`
	InstagramURL          string `json:"instagramUrl"`
	SiteURL               string `json:"siteUrl"`
	CamerasURL            string `json:"camerasUrl"`
	PhoneDisplay          string `json:"phoneDisplay"`
	PhoneE164             string `json:"phoneE164"`
	WhatsAppLink          string `json:"whatsappLink"`
	Region                string `json:"region"`
	ClientAudience        string `json:"clientAudience"`
	ApprovedVisualVersion string `json:"approvedVisualVersion"`
	ApprovalDate          string `json:"approvalDate"`
	CreativeBaseDir       string `json:"creativeBaseDir"`
	PaletteJSON           string `json:"paletteJson"`
	BioText               string `json:"bioText"`
	CustomEdited          bool   `json:"customEdited"`
	UpdatedAt             string `json:"updatedAt"`
}

// Validate checks minimal required fields for a company profile.
func (p CompanyProfile) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("profile id is required")
	}
	if strings.TrimSpace(p.Brand) == "" {
		return errors.New("brand name is required")
	}
	if strings.TrimSpace(p.InstagramHandle) == "" {
		return errors.New("instagram handle is required")
	}
	return nil
}

// LibraryAsset represents an imported and managed asset inside InstaGo's portable library.
type LibraryAsset struct {
	ID               string `json:"id"`
	CompanyID        string `json:"companyId"`
	RelPath          string `json:"relPath"`
	ThumbRelPath     string `json:"thumbRelPath"`
	SourceRelPath    string `json:"sourceRelPath"`
	FileName         string `json:"fileName"`
	MimeType         string `json:"mimeType"`
	MediaKind        string `json:"mediaKind"`      // image | font | document
	Classification   string `json:"classification"` // arte_unica | capa_auxiliar | cena | marca | capa_editorial | fonte | historico | referencia
	SubCategory      string `json:"subCategory"`
	Format           string `json:"format"` // feed | story | cover | square | other
	Title            string `json:"title"`
	Caption          string `json:"caption"`
	PieceID          string `json:"pieceId"`
	PageNumber       int    `json:"pageNumber"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	Bytes            int64  `json:"bytes"`
	SHA256           string `json:"sha256"`
	IsUniqueApproved bool   `json:"isUniqueApproved"`
	IsEditableModel  bool   `json:"isEditableModel"`
	LinkedDocumentID string `json:"linkedDocumentId"`
	MetadataJSON     string `json:"metadataJson"`
	ImportedAt       string `json:"importedAt"`
	AssetURL         string `json:"assetUrl"`
	ThumbURL         string `json:"thumbUrl"`
}

// DatedOffer represents a read-only commercial offer reference imported with date and source.
type DatedOffer struct {
	ID              string `json:"id"`
	CompanyID       string `json:"companyId"`
	Category        string `json:"category"` // internet_fibra | internet_radio | combo | ruralcam | regra_comercial
	Title           string `json:"title"`
	PromoPriceCents int64  `json:"promoPriceCents"`
	PromoPeriod     string `json:"promoPeriod"`
	AfterPriceCents int64  `json:"afterPriceCents"`
	InstallFeeCents int64  `json:"installFeeCents"`
	SpeedSummary    string `json:"speedSummary"`
	ServicesSummary string `json:"servicesSummary"`
	Restrictions    string `json:"restrictions"`
	CampaignName    string `json:"campaignName"`
	ValidUntil      string `json:"validUntil"`
	CheckedAt       string `json:"checkedAt"`
	SourceRef       string `json:"sourceRef"`
	ReadOnly        bool   `json:"readOnly"`
}

// PublicationHistoryItem records past published pieces purely as read-only operational history.
type PublicationHistoryItem struct {
	ID           string `json:"id"`
	CompanyID    string `json:"companyId"`
	Kind         string `json:"kind"` // feed_post | story_highlight
	PieceKey     string `json:"pieceKey"`
	Title        string `json:"title"`
	Position     int    `json:"position"`
	Pinned       bool   `json:"pinned"`
	Status       string `json:"status"`
	InstagramURL string `json:"instagramUrl"`
	Caption      string `json:"caption"`
	PageCount    int    `json:"pageCount"`
	AssetIDsJSON string `json:"assetIdsJson"`
	Notes        string `json:"notes"`
	ObservedAt   string `json:"observedAt"`
	SourceRef    string `json:"sourceRef"`
	ReadOnly     bool   `json:"readOnly"`
}

// ImportSummary reports counts and conversion status from a base import run.
type ImportSummary struct {
	RunID                string `json:"runId"`
	CompanyID            string `json:"companyId"`
	SourceDir            string `json:"sourceDir"`
	UniqueArtsCount      int    `json:"uniqueArtsCount"`
	AuxiliaryCoversCount int    `json:"auxiliaryCoversCount"`
	TotalAssetsCount     int    `json:"totalAssetsCount"`
	OffersCount          int    `json:"offersCount"`
	HistoryCount         int    `json:"historyCount"`
	EditableModelsCount  int    `json:"editableModelsCount"`
	ReferenceOnlyCount   int    `json:"referenceOnlyCount"`
	FontsCount           int    `json:"fontsCount"`
	SkillLoaded          bool   `json:"skillLoaded"`
	Status               string `json:"status"`
	CreatedAt            string `json:"createdAt"`
}

// LibraryFilter filters listed assets in the library.
type LibraryFilter struct {
	Query          string `json:"query"`
	Classification string `json:"classification"`
	Format         string `json:"format"`
	SubCategory    string `json:"subCategory"`
}

// LibraryOverview aggregates everything needed by the Library and Settings views.
type LibraryOverview struct {
	Profile       CompanyProfile           `json:"profile"`
	Summary       *ImportSummary           `json:"summary"`
	Assets        []LibraryAsset           `json:"assets"`
	Offers        []DatedOffer             `json:"offers"`
	History       []PublicationHistoryItem `json:"history"`
	CountsByClass map[string]int           `json:"countsByClass"`
	LibraryDir    string                   `json:"libraryDir"`
	BackupsDir    string                   `json:"backupsDir"`
}

// IntegrityIssue describes a missing or corrupted file inside the managed library.
type IntegrityIssue struct {
	AssetID        string `json:"assetId"`
	Title          string `json:"title"`
	RelPath        string `json:"relPath"`
	ExpectedSHA256 string `json:"expectedSha256"`
	ActualSHA256   string `json:"actualSha256"`
	Reason         string `json:"reason"` // missing | sha256_mismatch | read_error
}

// IntegrityReport summarizes SHA-256 verification across all managed library files.
type IntegrityReport struct {
	CheckedAt      string           `json:"checkedAt"`
	TotalAssets    int              `json:"totalAssets"`
	OKCount        int              `json:"okCount"`
	MissingCount   int              `json:"missingCount"`
	CorruptedCount int              `json:"corruptedCount"`
	Issues         []IntegrityIssue `json:"issues"`
}

// BackupResult reports a created portable .zip backup archive.
type BackupResult struct {
	FilePath      string `json:"filePath"`
	FileName      string `json:"fileName"`
	Bytes         int64  `json:"bytes"`
	SHA256        string `json:"sha256"`
	ArchivedFiles int    `json:"archivedFiles"`
	CreatedAt     string `json:"createdAt"`
}

// FormatBRL formats integer cents as Brazilian currency (e.g. 9999 -> "R$ 99,99").
func FormatBRL(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	reais := cents / 100
	centavos := cents % 100
	s := fmt.Sprintf("%d", reais)
	if len(s) > 3 {
		var parts []string
		for len(s) > 3 {
			parts = append([]string{s[len(s)-3:]}, parts...)
			s = s[:len(s)-3]
		}
		parts = append([]string{s}, parts...)
		s = strings.Join(parts, ".")
	}
	out := fmt.Sprintf("R$ %s,%02d", s, centavos)
	if neg {
		return "-" + out
	}
	return out
}
