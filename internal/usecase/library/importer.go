package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/seed"
)

type brandDataJSON struct {
	Brand                 string `json:"brand"`
	InstagramHandle       string `json:"instagramHandle"`
	InstagramURL          string `json:"instagramUrl"`
	Site                  string `json:"site"`
	CamerasPage           string `json:"camerasPage"`
	PhoneDisplay          string `json:"phoneDisplay"`
	PhoneE164             string `json:"phoneE164"`
	WhatsAppLink          string `json:"whatsappLink"`
	Region                string `json:"region"`
	ApprovedVisualVersion string `json:"approvedVisualVersion"`
	ApprovalDate          string `json:"approvalDate"`
	ClientAudience        string `json:"clientAudience"`
	ProfileName           string `json:"profileName"`
	BioFile               string `json:"bioFile"`
	PrimaryPalette        string `json:"primaryPalette"`
}

type catalogEntryJSON struct {
	OriginalFile string `json:"originalFile"`
	File         string `json:"file"`
	Category     string `json:"category"`
	UsedAsScene  []any  `json:"usedAsSceneIn"`
	Dimensions   *struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"dimensions"`
	SHA256 string `json:"sha256"`
}

type piecesFileJSON struct {
	Posts []struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Caption string `json:"caption"`
		Pages   []struct {
			Title string `json:"title"`
		} `json:"pages"`
	} `json:"posts"`
	Highlights []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Stories []struct {
			Title string `json:"title"`
			CTA   string `json:"cta"`
		} `json:"stories"`
	} `json:"highlights"`
}

type publishedManifestJSON struct {
	Account    string `json:"account"`
	Status     string `json:"status"`
	Date       string `json:"date"`
	LastVerify string `json:"executionDate"`
	Posts      []struct {
		ID           string `json:"id"`
		Position     int    `json:"positionInApprovedGrid"`
		Title        string `json:"title"`
		Pin          bool   `json:"pin"`
		Status       string `json:"status"`
		InstagramURL string `json:"instagramUrl"`
		CaptionFile  string `json:"captionFile"`
		Images       []struct {
			Page   int    `json:"page"`
			File   string `json:"file"`
			SHA256 string `json:"sha256"`
		} `json:"images"`
	} `json:"posts"`
	Highlights []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Position int    `json:"positionInApprovedHighlights"`
		Status   string `json:"status"`
		Stories  []struct {
			Page             int    `json:"page"`
			Title            string `json:"title"`
			File             string `json:"file"`
			PublishedPostURL string `json:"publishedPostUrl"`
			Status           string `json:"status"`
		} `json:"stories"`
	} `json:"highlights"`
}

type importBundle struct {
	Profile library.CompanyProfile
	Assets  []library.LibraryAsset
	Offers  []library.DatedOffer
	History []library.PublicationHistoryItem
	Summary library.ImportSummary
}

func (s *Service) buildImportBundle(sourceDir string, existingProfile *library.CompanyProfile) (importBundle, error) {
	cleanSource := filepath.Clean(sourceDir)
	info, err := os.Stat(cleanSource)
	if err != nil || !info.IsDir() {
		return importBundle{}, fmt.Errorf("creative base directory not found: %s", cleanSource)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	profile, err := parseCompanyProfile(cleanSource, existingProfile, now)
	if err != nil {
		return importBundle{}, err
	}

	pieces, captionsByPiece := parsePiecesAndCaptions(cleanSource)
	assetsList := make([]library.LibraryAsset, 0, 220)
	seenSourceRel := make(map[string]struct{})

	type thumbJob struct {
		idx     int
		absPath string
		hashHex string
		width   int
	}
	thumbJobs := make([]thumbJob, 0, 128)

	addManagedFile := func(opts assetImportOpts) error {
		relNorm := filepath.ToSlash(filepath.Clean(opts.SourceRelPath))
		if _, exists := seenSourceRel[relNorm]; exists {
			return nil
		}
		absSource := filepath.Join(cleanSource, filepath.FromSlash(relNorm))
		st, statErr := os.Stat(absSource)
		if statErr != nil || st.IsDir() {
			return nil
		}
		seenSourceRel[relNorm] = struct{}{}

		destAbs := filepath.Join(s.libraryDir, filepath.FromSlash(opts.DestRelPath))
		hashHex, sizeBytes, copyErr := copyFileAtomicAndHash(absSource, destAbs)
		if copyErr != nil {
			return fmt.Errorf("copy %s: %w", relNorm, copyErr)
		}

		width, height := opts.HintWidth, opts.HintHeight
		if opts.MediaKind == "image" {
			if w, h, ok := readImageDimensions(destAbs); ok {
				width, height = w, h
			}
		}

		assetURL := "/library-assets/" + strings.TrimPrefix(filepath.ToSlash(opts.DestRelPath), "/")
		thumbURL := ""
		if opts.MediaKind == "image" {
			thumbURL = assetURL
		}

		idx := len(assetsList)
		assetsList = append(assetsList, library.LibraryAsset{
			ID:               opts.ID,
			CompanyID:        profile.ID,
			RelPath:          filepath.ToSlash(opts.DestRelPath),
			ThumbRelPath:     "",
			SourceRelPath:    relNorm,
			FileName:         filepath.Base(relNorm),
			MimeType:         detectMimeType(relNorm),
			MediaKind:        opts.MediaKind,
			Classification:   opts.Classification,
			SubCategory:      opts.SubCategory,
			Format:           opts.Format,
			Title:            opts.Title,
			Caption:          opts.Caption,
			PieceID:          opts.PieceID,
			PageNumber:       opts.PageNumber,
			Width:            width,
			Height:           height,
			Bytes:            sizeBytes,
			SHA256:           hashHex,
			IsUniqueApproved: opts.IsUniqueApproved,
			IsEditableModel:  opts.IsEditableModel,
			LinkedDocumentID: opts.LinkedDocumentID,
			MetadataJSON:     opts.MetadataJSON,
			ImportedAt:       now,
			AssetURL:         assetURL,
			ThumbURL:         thumbURL,
		})
		if opts.MediaKind == "image" && (width == 0 || width > 360) {
			thumbJobs = append(thumbJobs, thumbJob{
				idx:     idx,
				absPath: destAbs,
				hashHex: hashHex,
				width:   width,
			})
		}
		return nil
	}

	// 1. Import 70 unique approved arts from 03-banners-prontos/{timeline,stories-destaques,capas-destaques}.
	if err := importUniqueArts(cleanSource, pieces, captionsByPiece, addManagedFile); err != nil {
		return importBundle{}, err
	}

	// 2. Import 12 auxiliary feed covers from 03-banners-prontos/capas-timeline-copias.
	if err := importAuxiliaryCovers(cleanSource, addManagedFile); err != nil {
		return importBundle{}, err
	}

	// 3. Import 05-biblioteca/catalogo-assets.json entries (scenes, logos, editorial covers, fonts, historical items).
	if err := importCatalogAssets(cleanSource, addManagedFile); err != nil {
		return importBundle{}, err
	}

	// 4. Ensure brand fonts and logos under 02-identidade are also managed.
	if err := importIdentityFiles(cleanSource, addManagedFile); err != nil {
		return importBundle{}, err
	}

	// 5. Import key reference documents (V5 HTML marked strictly as reference, brand guide, prompts, skill).
	if err := importReferenceDocuments(cleanSource, s.skillsDir, s.libraryDir, profile.ID, now, &assetsList, addManagedFile); err != nil {
		return importBundle{}, err
	}

	// Generate JPEG thumbnails concurrently across a bounded worker pool (deduplicated by hash).
	if len(thumbJobs) > 0 {
		uniqueJobs := make([]thumbJob, 0, len(thumbJobs))
		seenHash := make(map[string]struct{}, len(thumbJobs))
		for _, job := range thumbJobs {
			if _, ok := seenHash[job.hashHex]; ok {
				continue
			}
			seenHash[job.hashHex] = struct{}{}
			uniqueJobs = append(uniqueJobs, job)
		}

		workers := min(max(runtime.NumCPU(), 2), 4)
		jobsCh := make(chan thumbJob, len(uniqueJobs))
		for _, job := range uniqueJobs {
			jobsCh <- job
		}
		close(jobsCh)

		var wg sync.WaitGroup
		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for job := range jobsCh {
					_, _ = s.ensureThumbnail(job.absPath, job.hashHex, job.width)
				}
			}()
		}
		wg.Wait()

		for _, job := range thumbJobs {
			thumbRel := filepath.ToSlash(filepath.Join(".thumbs", job.hashHex+".jpg"))
			if st, err := os.Stat(filepath.Join(s.libraryDir, filepath.FromSlash(thumbRel))); err == nil && st.Size() > 0 {
				assetsList[job.idx].ThumbRelPath = thumbRel
				assetsList[job.idx].ThumbURL = "/library-assets/" + strings.TrimPrefix(thumbRel, "/")
			}
		}
	}

	offers := parseDatedOffers(cleanSource, profile.ID)
	history := parsePublicationHistory(cleanSource, profile.ID, captionsByPiece)

	summary := buildImportSummary(profile.ID, cleanSource, assetsList, offers, history, s.HasRuralSkill(), now)

	return importBundle{
		Profile: profile,
		Assets:  assetsList,
		Offers:  offers,
		History: history,
		Summary: summary,
	}, nil
}

type assetImportOpts struct {
	ID               string
	SourceRelPath    string
	DestRelPath      string
	MediaKind        string
	Classification   string
	SubCategory      string
	Format           string
	Title            string
	Caption          string
	PieceID          string
	PageNumber       int
	HintWidth        int
	HintHeight       int
	IsUniqueApproved bool
	IsEditableModel  bool
	LinkedDocumentID string
	MetadataJSON     string
}

func parseCompanyProfile(sourceDir string, existing *library.CompanyProfile, now string) (library.CompanyProfile, error) {
	raw, err := os.ReadFile(filepath.Join(sourceDir, "01-contexto", "DADOS-DA-MARCA.json"))
	if err != nil {
		return library.CompanyProfile{}, fmt.Errorf("read DADOS-DA-MARCA.json: %w", err)
	}
	var bd brandDataJSON
	if err := json.Unmarshal(raw, &bd); err != nil {
		return library.CompanyProfile{}, fmt.Errorf("parse DADOS-DA-MARCA.json: %w", err)
	}

	paletteRaw := "{}"
	if bd.PrimaryPalette != "" {
		if b, err := os.ReadFile(filepath.Join(sourceDir, filepath.FromSlash(bd.PrimaryPalette))); err == nil {
			paletteRaw = strings.TrimSpace(string(b))
		}
	}

	bioText := ""
	if bd.BioFile != "" {
		if b, err := os.ReadFile(filepath.Join(sourceDir, filepath.FromSlash(bd.BioFile))); err == nil {
			bioText = strings.TrimSpace(string(b))
		}
	}

	profile := library.CompanyProfile{
		ID:                    library.DefaultCompanyID,
		Brand:                 defaultIfEmpty(bd.Brand, "Rural Conecta"),
		ProfileName:           bd.ProfileName,
		InstagramHandle:       defaultIfEmpty(bd.InstagramHandle, "ruralconectamg"),
		InstagramURL:          bd.InstagramURL,
		SiteURL:               bd.Site,
		CamerasURL:            bd.CamerasPage,
		PhoneDisplay:          bd.PhoneDisplay,
		PhoneE164:             bd.PhoneE164,
		WhatsAppLink:          bd.WhatsAppLink,
		Region:                bd.Region,
		ClientAudience:        bd.ClientAudience,
		ApprovedVisualVersion: bd.ApprovedVisualVersion,
		ApprovalDate:          bd.ApprovalDate,
		CreativeBaseDir:       sourceDir,
		PaletteJSON:           paletteRaw,
		BioText:               bioText,
		CustomEdited:          false,
		UpdatedAt:             now,
	}

	// Preserve operator edits to editable contact fields on reimport.
	if existing != nil && existing.CustomEdited {
		profile.Brand = existing.Brand
		profile.ProfileName = existing.ProfileName
		profile.InstagramHandle = existing.InstagramHandle
		profile.InstagramURL = existing.InstagramURL
		profile.SiteURL = existing.SiteURL
		profile.CamerasURL = existing.CamerasURL
		profile.PhoneDisplay = existing.PhoneDisplay
		profile.PhoneE164 = existing.PhoneE164
		profile.WhatsAppLink = existing.WhatsAppLink
		profile.Region = existing.Region
		profile.ClientAudience = existing.ClientAudience
		profile.BioText = existing.BioText
		profile.CustomEdited = true
	}

	return profile, nil
}

func parsePiecesAndCaptions(sourceDir string) (piecesFileJSON, map[string]string) {
	var pieces piecesFileJSON
	if raw, err := os.ReadFile(filepath.Join(sourceDir, "04-textos", "textos-e-composicoes-por-peca.json")); err == nil {
		_ = json.Unmarshal(raw, &pieces)
	}

	captions := make(map[string]string)
	for _, p := range pieces.Posts {
		if p.ID != "" && strings.TrimSpace(p.Caption) != "" {
			captions[p.ID] = strings.TrimSpace(p.Caption)
		}
	}

	legendasDir := filepath.Join(sourceDir, "04-textos", "legendas")
	if entries, err := os.ReadDir(legendasDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
				continue
			}
			raw, readErr := os.ReadFile(filepath.Join(legendasDir, entry.Name()))
			if readErr != nil {
				continue
			}
			baseName := strings.TrimSuffix(entry.Name(), ".txt")
			text := strings.TrimSpace(string(raw))
			captions[baseName] = text
			if parts := strings.SplitN(baseName, "-", 2); len(parts) == 2 {
				captions[parts[1]] = text
			}
		}
	}
	return pieces, captions
}

func importUniqueArts(sourceDir string, pieces piecesFileJSON, captions map[string]string, add func(assetImportOpts) error) error {
	// 1a. Timeline (38 pages across 12 posts)
	timelineDir := filepath.Join(sourceDir, "03-banners-prontos", "timeline")
	postDirs, err := readSortedDirs(timelineDir)
	if err == nil {
		for _, dirName := range postDirs {
			pieceKey := dirName
			if parts := strings.SplitN(dirName, "-", 2); len(parts) == 2 {
				pieceKey = parts[1]
			}
			caption := captions[dirName]
			if caption == "" {
				caption = captions[pieceKey]
			}
			files, _ := readSortedImageFiles(filepath.Join(timelineDir, dirName))
			for idx, fileName := range files {
				pageNum := idx + 1
				title := resolvePostPageTitle(pieces, pieceKey, pageNum, fileName)
				isEditable := fileName == "02-combo-02.jpg"
				linkedDoc := ""
				if isEditable {
					linkedDoc = seed.RepresentativeV5ComboID
				}
				rel := filepath.ToSlash(filepath.Join("03-banners-prontos", "timeline", dirName, fileName))
				if err := add(assetImportOpts{
					ID:               "art-feed-" + strings.TrimSuffix(fileName, filepath.Ext(fileName)),
					SourceRelPath:    rel,
					DestRelPath:      filepath.ToSlash(filepath.Join("artes-unicas", "timeline", dirName, fileName)),
					MediaKind:        "image",
					Classification:   "arte_unica",
					SubCategory:      "timeline",
					Format:           "feed",
					Title:            title,
					Caption:          caption,
					PieceID:          dirName,
					PageNumber:       pageNum,
					IsUniqueApproved: true,
					IsEditableModel:  isEditable,
					LinkedDocumentID: linkedDoc,
					MetadataJSON:     fmt.Sprintf(`{"postKey":%q,"conversion":"%s"}`, pieceKey, ternStr(isEditable, "editable_seed_r0", "referencia_raster")),
				}); err != nil {
					return err
				}
			}
		}
	}

	// 1b. Stories-destaques (24 stories across 8 highlights)
	storiesDir := filepath.Join(sourceDir, "03-banners-prontos", "stories-destaques")
	storyDirs, err := readSortedDirs(storiesDir)
	if err == nil {
		for hIdx, dirName := range storyDirs {
			files, _ := readSortedImageFiles(filepath.Join(storiesDir, dirName))
			for idx, fileName := range files {
				pageNum := idx + 1
				title, cta := resolveStoryTitle(pieces, hIdx, pageNum, fileName)
				rel := filepath.ToSlash(filepath.Join("03-banners-prontos", "stories-destaques", dirName, fileName))
				if err := add(assetImportOpts{
					ID:               "art-story-" + strings.TrimSuffix(fileName, filepath.Ext(fileName)),
					SourceRelPath:    rel,
					DestRelPath:      filepath.ToSlash(filepath.Join("artes-unicas", "stories-destaques", dirName, fileName)),
					MediaKind:        "image",
					Classification:   "arte_unica",
					SubCategory:      "stories-destaques",
					Format:           "story",
					Title:            title,
					Caption:          cta,
					PieceID:          dirName,
					PageNumber:       pageNum,
					IsUniqueApproved: true,
					MetadataJSON:     fmt.Sprintf(`{"highlightDir":%q,"conversion":"referencia_raster"}`, dirName),
				}); err != nil {
					return err
				}
			}
		}
	}

	// 1c. Capas-destaques (8 covers)
	coversDir := filepath.Join(sourceDir, "03-banners-prontos", "capas-destaques")
	coverFiles, err := readSortedImageFiles(coversDir)
	if err == nil {
		for idx, fileName := range coverFiles {
			rel := filepath.ToSlash(filepath.Join("03-banners-prontos", "capas-destaques", fileName))
			if err := add(assetImportOpts{
				ID:               "art-cover-" + strings.TrimSuffix(fileName, filepath.Ext(fileName)),
				SourceRelPath:    rel,
				DestRelPath:      filepath.ToSlash(filepath.Join("artes-unicas", "capas-destaques", fileName)),
				MediaKind:        "image",
				Classification:   "arte_unica",
				SubCategory:      "capas-destaques",
				Format:           "cover",
				Title:            "Capa destaque: " + strings.TrimSuffix(fileName, filepath.Ext(fileName)),
				PieceID:          "capas-destaques",
				PageNumber:       idx + 1,
				IsUniqueApproved: true,
				MetadataJSON:     `{"conversion":"referencia_raster"}`,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func importAuxiliaryCovers(sourceDir string, add func(assetImportOpts) error) error {
	auxDir := filepath.Join(sourceDir, "03-banners-prontos", "capas-timeline-copias")
	files, err := readSortedImageFiles(auxDir)
	if err != nil {
		return nil
	}
	for idx, fileName := range files {
		rel := filepath.ToSlash(filepath.Join("03-banners-prontos", "capas-timeline-copias", fileName))
		if err := add(assetImportOpts{
			ID:               "aux-cover-" + strings.TrimSuffix(fileName, filepath.Ext(fileName)),
			SourceRelPath:    rel,
			DestRelPath:      filepath.ToSlash(filepath.Join("capas-auxiliares", fileName)),
			MediaKind:        "image",
			Classification:   "capa_auxiliar",
			SubCategory:      "capas-timeline-copias",
			Format:           "feed",
			Title:            "Capa auxiliar (cópia feed): " + fileName,
			PieceID:          "capas-timeline-copias",
			PageNumber:       idx + 1,
			IsUniqueApproved: false,
			MetadataJSON:     `{"auxiliaryCopy":true,"note":"Cópia auxiliar de capa; não conta como post novo"}`,
		}); err != nil {
			return err
		}
	}
	return nil
}

func importCatalogAssets(sourceDir string, add func(assetImportOpts) error) error {
	raw, err := os.ReadFile(filepath.Join(sourceDir, "05-biblioteca", "catalogo-assets.json"))
	if err != nil {
		return nil
	}
	var entries []catalogEntryJSON
	if err := json.Unmarshal(raw, &entries); err != nil {
		return fmt.Errorf("parse catalogo-assets.json: %w", err)
	}

	for idx, entry := range entries {
		rel := filepath.ToSlash(filepath.Clean(entry.File))
		if rel == "" || strings.HasPrefix(rel, "..") {
			continue
		}
		classif, mediaKind, format := mapCatalogCategory(entry.Category, rel)
		w, h := 0, 0
		if entry.Dimensions != nil {
			w, h = entry.Dimensions.Width, entry.Dimensions.Height
		}
		idBase := strings.ReplaceAll(strings.TrimSuffix(rel, filepath.Ext(rel)), "/", "-")
		if err := add(assetImportOpts{
			ID:             fmt.Sprintf("cat-%03d-%s", idx+1, sanitizeID(idBase)),
			SourceRelPath:  rel,
			DestRelPath:    filepath.ToSlash(filepath.Join("catalogo", rel)),
			MediaKind:      mediaKind,
			Classification: classif,
			SubCategory:    entry.Category,
			Format:         format,
			Title:          entry.OriginalFile,
			HintWidth:      w,
			HintHeight:     h,
			MetadataJSON:   fmt.Sprintf(`{"catalogCategory":%q,"usedInCount":%d}`, entry.Category, len(entry.UsedAsScene)),
		}); err != nil {
			return err
		}
	}
	return nil
}

func importIdentityFiles(sourceDir string, add func(assetImportOpts) error) error {
	fontDir := filepath.Join(sourceDir, "02-identidade", "fontes")
	if entries, err := os.ReadDir(fontDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".woff2" && ext != ".woff" && ext != ".ttf" && ext != ".otf" {
				continue
			}
			rel := filepath.ToSlash(filepath.Join("02-identidade", "fontes", entry.Name()))
			if err := add(assetImportOpts{
				ID:             "font-" + sanitizeID(strings.TrimSuffix(entry.Name(), ext)),
				SourceRelPath:  rel,
				DestRelPath:    filepath.ToSlash(filepath.Join("identidade", "fontes", entry.Name())),
				MediaKind:      "font",
				Classification: "fonte",
				SubCategory:    "tipografia",
				Format:         "other",
				Title:          "Fonte local: " + entry.Name(),
				MetadataJSON:   `{"status":"intacta"}`,
			}); err != nil {
				return err
			}
		}
	}

	logosRoot := filepath.Join(sourceDir, "02-identidade", "logos")
	_ = filepath.WalkDir(logosRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !isImageExt(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		sub := "marca"
		if strings.Contains(relSlash, "parceiros-e-plataformas") {
			sub = "logo_terceiro"
		}
		return add(assetImportOpts{
			ID:             "logo-" + sanitizeID(strings.TrimSuffix(relSlash, filepath.Ext(relSlash))),
			SourceRelPath:  relSlash,
			DestRelPath:    filepath.ToSlash(filepath.Join("identidade", relSlash)),
			MediaKind:      "image",
			Classification: "marca",
			SubCategory:    sub,
			Format:         "other",
			Title:          d.Name(),
			MetadataJSON:   `{"origin":"02-identidade/logos"}`,
		})
	})

	return nil
}

func importReferenceDocuments(
	sourceDir string,
	skillsDir string,
	libraryDir string,
	companyID string,
	now string,
	assetsList *[]library.LibraryAsset,
	add func(assetImportOpts) error,
) error {
	refs := []struct {
		id    string
		rel   string
		sub   string
		title string
		note  string
	}{
		{"ref-v5-html", "06-mockup-aprovado/v5/index.html", "v5_referencia", "Mockup V5 HTML (somente referência — não convertido automaticamente)", "V5 original preservada intacta"},
		{"ref-guia-uso", "02-identidade/GUIA-DE-USO.md", "guia_marca", "Guia de uso de marcas, logos e fontes", "Referência da identidade visual"},
		{"ref-ofertas-md", "01-contexto/OFERTAS-APROVADAS-2026-11-11.md", "ofertas_datadas", "Ofertas aprovadas em 11/11/2026 (referência datada)", "Preços datados de referência"},
		{"ref-prompt-foto", "08-prompts-e-modelos/MODELO-DE-PROMPT-FOTOGRAFICO.md", "prompts", "Modelo de prompt fotográfico", "Referência para novas cenas"},
		{"ref-brand-agents", "AGENTS.md", "skill_regras", "Regras vigentes da marca (AGENTS.md)", "Regras locais da Rural Conecta"},
	}

	for _, r := range refs {
		if err := add(assetImportOpts{
			ID:             r.id,
			SourceRelPath:  r.rel,
			DestRelPath:    filepath.ToSlash(filepath.Join("referencias", filepath.Base(r.rel))),
			MediaKind:      "document",
			Classification: "referencia",
			SubCategory:    r.sub,
			Format:         "other",
			Title:          r.title,
			Caption:        r.note,
			MetadataJSON:   fmt.Sprintf(`{"readOnlyReference":true,"note":%q}`, r.note),
		}); err != nil {
			return err
		}
	}

	// Copy the imported Rural Conecta skill if available in workspace skillsDir or creative base.
	skillCandidates := []string{}
	if strings.TrimSpace(skillsDir) != "" {
		skillCandidates = append(skillCandidates, filepath.Join(skillsDir, "rural-conecta-instagram", "SKILL.md"))
	}
	skillCandidates = append(skillCandidates, filepath.Join(sourceDir, ".agents", "skills", "rural-conecta-instagram", "SKILL.md"))
	for _, cand := range skillCandidates {
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			destRel := filepath.ToSlash(filepath.Join("referencias", "rural-conecta-SKILL.md"))
			destAbs := filepath.Join(libraryDir, filepath.FromSlash(destRel))
			hashHex, sizeBytes, copyErr := copyFileAtomicAndHash(cand, destAbs)
			if copyErr != nil {
				return fmt.Errorf("copy skill %s: %w", cand, copyErr)
			}
			*assetsList = append(*assetsList, library.LibraryAsset{
				ID:             "ref-rural-skill",
				CompanyID:      companyID,
				RelPath:        destRel,
				SourceRelPath:  cand,
				FileName:       "rural-conecta-SKILL.md",
				MimeType:       "text/markdown; charset=utf-8",
				MediaKind:      "document",
				Classification: "referencia",
				SubCategory:    "skill_regras",
				Format:         "other",
				Title:          "Skill Rural Conecta (SKILL.md)",
				Caption:        "Regras criativas e diretrizes de marca importadas",
				Bytes:          sizeBytes,
				SHA256:         hashHex,
				MetadataJSON:   `{"readOnlyReference":true,"skill":true}`,
				ImportedAt:     now,
				AssetURL:       "/library-assets/" + destRel,
			})
			break
		}
	}

	return nil
}

func parseDatedOffers(sourceDir string, companyID string) []library.DatedOffer {
	mdPath := filepath.Join(sourceDir, "01-contexto", "OFERTAS-APROVADAS-2026-11-11.md")
	raw, err := os.ReadFile(mdPath)
	if err != nil {
		return make([]library.DatedOffer, 0)
	}
	text := string(raw)
	offers := make([]library.DatedOffer, 0, 16)

	section := ""
	tableRowRe := regexp.MustCompile(`^\|([^|]+)\|([^|]+)\|(?:\s*([^|]+)\|)?(?:\s*([^|]+)\|)?$`)

	for _, rawLine := range strings.Split(text, "\n") {
		line := strings.TrimSpace(rawLine)
		switch {
		case strings.HasPrefix(line, "| Fibra "):
			section = "fibra"
			continue
		case strings.HasPrefix(line, "| Rádio ") || strings.HasPrefix(line, "| Radio "):
			section = "radio"
			continue
		case strings.HasPrefix(line, "| Combo "):
			section = "combo"
			continue
		case strings.HasPrefix(line, "| Serviço ") || strings.HasPrefix(line, "| Servico "):
			section = "ruralcam"
			continue
		case line == "" || strings.HasPrefix(line, "|---"):
			continue
		}

		m := tableRowRe.FindStringSubmatch(line)
		if len(m) == 0 {
			continue
		}
		col1 := strings.TrimSpace(m[1])
		col2 := strings.TrimSpace(m[2])
		col3 := strings.TrimSpace(m[3])
		col4 := strings.TrimSpace(m[4])

		switch section {
		case "fibra":
			offers = append(offers, library.DatedOffer{
				ID:              "offer-fibra-" + sanitizeID(col1),
				CompanyID:       companyID,
				Category:        "internet_fibra",
				Title:           "Fibra " + col1,
				PromoPriceCents: 9999,
				PromoPeriod:     "Nos 3 primeiros meses",
				AfterPriceCents: parseBRLCents(col2),
				InstallFeeCents: 24990,
				SpeedSummary:    col1 + " (fibra)",
				ServicesSummary: "Internet rural via fibra",
				Restrictions:    "Cobertura e disponibilidade mediante consulta. Referência antiga de 11/11/2026.",
				CampaignName:    "Black Friday (referência 11/11/2026)",
				ValidUntil:      "2026-11-30",
				CheckedAt:       "2026-11-11",
				SourceRef:       "01-contexto/OFERTAS-APROVADAS-2026-11-11.md",
				ReadOnly:        true,
			})
		case "radio":
			offers = append(offers, library.DatedOffer{
				ID:              "offer-radio-" + sanitizeID(col1),
				CompanyID:       companyID,
				Category:        "internet_radio",
				Title:           "Rádio " + col1,
				PromoPriceCents: 9999,
				PromoPeriod:     "Nos 3 primeiros meses",
				AfterPriceCents: parseBRLCents(col2),
				InstallFeeCents: 24990,
				SpeedSummary:    col1 + " (rádio)",
				ServicesSummary: "Internet rural via rádio",
				Restrictions:    "Cobertura e disponibilidade mediante consulta. Referência antiga de 11/11/2026.",
				CampaignName:    "Black Friday (referência 11/11/2026)",
				ValidUntil:      "2026-11-30",
				CheckedAt:       "2026-11-11",
				SourceRef:       "01-contexto/OFERTAS-APROVADAS-2026-11-11.md",
				ReadOnly:        true,
			})
		case "combo":
			offers = append(offers, library.DatedOffer{
				ID:              "offer-combo-" + sanitizeID(col1),
				CompanyID:       companyID,
				Category:        "combo",
				Title:           "Combo " + col1,
				PromoPriceCents: 9999,
				PromoPeriod:     "Nos 3 primeiros meses",
				AfterPriceCents: parseBRLCents(col4),
				InstallFeeCents: 24990,
				SpeedSummary:    col2 + " (fibra / rádio)",
				ServicesSummary: col3,
				Restrictions:    "RuralTV + HBO somente no Combo Completo. Referência antiga de 11/11/2026.",
				CampaignName:    "Black Friday (referência 11/11/2026)",
				ValidUntil:      "2026-11-30",
				CheckedAt:       "2026-11-11",
				SourceRef:       "01-contexto/OFERTAS-APROVADAS-2026-11-11.md",
				ReadOnly:        true,
			})
		case "ruralcam":
			restrict := "Instalação grátis para clientes Rural Conecta; R$ 250,00 para novos clientes. 7 dias de gravação na Black Friday para câmeras."
			if strings.Contains(strings.ToLower(col1), "alarme") && !strings.Contains(strings.ToLower(col1), "câmera") {
				restrict = "Somente alarmes (não inclui câmera). Instalação grátis para clientes; R$ 250,00 novos clientes."
			}
			offers = append(offers, library.DatedOffer{
				ID:              "offer-cam-" + sanitizeID(col1),
				CompanyID:       companyID,
				Category:        "ruralcam",
				Title:           "RuralCam — " + col1,
				PromoPriceCents: parseBRLCents(col2),
				PromoPeriod:     "Mensalidade",
				AfterPriceCents: parseBRLCents(col2),
				InstallFeeCents: 25000,
				SpeedSummary:    "",
				ServicesSummary: col1,
				Restrictions:    restrict,
				CampaignName:    "Black Friday (referência 11/11/2026)",
				ValidUntil:      "2026-11-30",
				CheckedAt:       "2026-11-11",
				SourceRef:       "01-contexto/OFERTAS-APROVADAS-2026-11-11.md",
				ReadOnly:        true,
			})
		}
	}

	return offers
}

func parsePublicationHistory(sourceDir string, companyID string, captions map[string]string) []library.PublicationHistoryItem {
	manifestPath := filepath.Join(sourceDir, "07-agente-publicacao", "execucao-2026-11-11", "publicacao-v5-com-links-reais.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return make([]library.PublicationHistoryItem, 0)
	}
	var man publishedManifestJSON
	if err := json.Unmarshal(raw, &man); err != nil {
		return make([]library.PublicationHistoryItem, 0)
	}

	observed := defaultIfEmpty(man.LastVerify, man.Date)
	items := make([]library.PublicationHistoryItem, 0, len(man.Posts)+len(man.Highlights))

	for _, p := range man.Posts {
		assetIDs := make([]string, 0, len(p.Images))
		for _, img := range p.Images {
			base := strings.TrimSuffix(filepath.Base(img.File), filepath.Ext(img.File))
			assetIDs = append(assetIDs, "art-feed-"+base)
		}
		assetJSON, _ := json.Marshal(assetIDs)
		caption := captions[p.ID]
		if caption == "" && p.CaptionFile != "" {
			if b, err := os.ReadFile(filepath.Join(sourceDir, filepath.FromSlash(p.CaptionFile))); err == nil {
				caption = strings.TrimSpace(string(b))
			}
		}
		items = append(items, library.PublicationHistoryItem{
			ID:           "hist-post-" + p.ID,
			CompanyID:    companyID,
			Kind:         "feed_post",
			PieceKey:     p.ID,
			Title:        p.Title,
			Position:     p.Position,
			Pinned:       p.Pin,
			Status:       p.Status,
			InstagramURL: p.InstagramURL,
			Caption:      caption,
			PageCount:    len(p.Images),
			AssetIDsJSON: string(assetJSON),
			Notes:        "Registro passado (11/11/2026) — somente consulta; importação não gera republicação.",
			ObservedAt:   observed,
			SourceRef:    "07-agente-publicacao/execucao-2026-11-11/publicacao-v5-com-links-reais.json",
			ReadOnly:     true,
		})
	}

	for _, h := range man.Highlights {
		assetIDs := make([]string, 0, len(h.Stories))
		for _, st := range h.Stories {
			base := strings.TrimSuffix(filepath.Base(st.File), filepath.Ext(st.File))
			assetIDs = append(assetIDs, "art-story-"+base)
		}
		assetJSON, _ := json.Marshal(assetIDs)
		items = append(items, library.PublicationHistoryItem{
			ID:           "hist-highlight-" + h.ID,
			CompanyID:    companyID,
			Kind:         "story_highlight",
			PieceKey:     h.ID,
			Title:        "Destaque: " + h.Name,
			Position:     h.Position,
			Pinned:       false,
			Status:       h.Status,
			InstagramURL: "",
			Caption:      fmt.Sprintf("%d Stories vinculados ao destaque %s.", len(h.Stories), h.Name),
			PageCount:    len(h.Stories),
			AssetIDsJSON: string(assetJSON),
			Notes:        "Destaque concluído pelo usuário em 11/11/2026 (somente consulta; pendente apenas conferir 23 links por toque e URL limpa da bio conforme ESTADO.md).",
			ObservedAt:   observed,
			SourceRef:    "07-agente-publicacao/execucao-2026-11-11/aplicacao-android/ESTADO.md",
			ReadOnly:     true,
		})
	}

	return items
}

func buildImportSummary(
	companyID string,
	sourceDir string,
	assets []library.LibraryAsset,
	offers []library.DatedOffer,
	history []library.PublicationHistoryItem,
	skillLoaded bool,
	now string,
) library.ImportSummary {
	var uniqueArts, auxCovers, editableModels, refOnly, fonts int
	for _, a := range assets {
		switch a.Classification {
		case "arte_unica":
			uniqueArts++
			if a.IsEditableModel {
				editableModels++
			} else {
				refOnly++
			}
		case "capa_auxiliar":
			auxCovers++
		case "fonte":
			fonts++
		case "referencia":
			refOnly++
		}
	}
	return library.ImportSummary{
		RunID:                "imp-" + uuid.NewString()[:8],
		CompanyID:            companyID,
		SourceDir:            sourceDir,
		UniqueArtsCount:      uniqueArts,
		AuxiliaryCoversCount: auxCovers,
		TotalAssetsCount:     len(assets),
		OffersCount:          len(offers),
		HistoryCount:         len(history),
		EditableModelsCount:  editableModels,
		ReferenceOnlyCount:   refOnly,
		FontsCount:           fonts,
		SkillLoaded:          skillLoaded,
		Status:               "imported",
		CreatedAt:            now,
	}
}

func (s *Service) ensureThumbnail(imageAbsPath string, hashHex string, width int) (string, bool) {
	if width > 0 && width <= 360 {
		return "", false
	}
	ext := strings.ToLower(filepath.Ext(imageAbsPath))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return "", false
	}
	thumbRel := filepath.ToSlash(filepath.Join(".thumbs", hashHex+".jpg"))
	thumbAbs := filepath.Join(s.libraryDir, filepath.FromSlash(thumbRel))
	if st, err := os.Stat(thumbAbs); err == nil && st.Size() > 0 {
		return thumbRel, true
	}

	f, err := os.Open(imageAbsPath)
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()

	srcImg, _, err := image.Decode(f)
	if err != nil {
		return "", false
	}
	bounds := srcImg.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 360 || srcH <= 0 {
		return "", false
	}

	dstW := 320
	dstH := max(1, (srcH*dstW)/srcW)
	dst := downscaleImageFast(srcImg, dstW, dstH)

	if err := os.MkdirAll(filepath.Dir(thumbAbs), 0o750); err != nil {
		return "", false
	}
	tmpPath := thumbAbs + ".tmp-" + uuid.NewString()[:8]
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return "", false
	}
	if err := jpeg.Encode(out, dst, &jpeg.Options{Quality: 80}); err != nil {
		_ = out.Close()
		_ = os.Remove(tmpPath)
		return "", false
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", false
	}
	if err := os.Rename(tmpPath, thumbAbs); err != nil {
		_ = os.Remove(tmpPath)
		return "", false
	}
	return thumbRel, true
}

func downscaleImageFast(srcImg image.Image, dstW, dstH int) *image.RGBA {
	bounds := srcImg.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))

	switch img := srcImg.(type) {
	case *image.YCbCr:
		for y := 0; y < dstH; y++ {
			sy := bounds.Min.Y + (y*srcH)/dstH
			rowOff := y * dst.Stride
			for x := 0; x < dstW; x++ {
				sx := bounds.Min.X + (x*srcW)/dstW
				yi := img.YOffset(sx, sy)
				ci := img.COffset(sx, sy)
				r, g, b := color.YCbCrToRGB(img.Y[yi], img.Cb[ci], img.Cr[ci])
				off := rowOff + x*4
				dst.Pix[off] = r
				dst.Pix[off+1] = g
				dst.Pix[off+2] = b
				dst.Pix[off+3] = 255
			}
		}
	case *image.RGBA:
		for y := 0; y < dstH; y++ {
			sy := bounds.Min.Y + (y*srcH)/dstH
			rowOff := y * dst.Stride
			for x := 0; x < dstW; x++ {
				sx := bounds.Min.X + (x*srcW)/dstW
				sOff := img.PixOffset(sx, sy)
				dOff := rowOff + x*4
				copy(dst.Pix[dOff:dOff+4], img.Pix[sOff:sOff+4])
			}
		}
	case *image.NRGBA:
		for y := 0; y < dstH; y++ {
			sy := bounds.Min.Y + (y*srcH)/dstH
			rowOff := y * dst.Stride
			for x := 0; x < dstW; x++ {
				sx := bounds.Min.X + (x*srcW)/dstW
				sOff := img.PixOffset(sx, sy)
				dOff := rowOff + x*4
				a := uint16(img.Pix[sOff+3])
				if a == 255 {
					copy(dst.Pix[dOff:dOff+4], img.Pix[sOff:sOff+4])
				} else if a == 0 {
					dst.Pix[dOff], dst.Pix[dOff+1], dst.Pix[dOff+2], dst.Pix[dOff+3] = 255, 255, 255, 255
				} else {
					dst.Pix[dOff] = uint8((uint16(img.Pix[sOff])*a + 255*(255-a)) / 255)
					dst.Pix[dOff+1] = uint8((uint16(img.Pix[sOff+1])*a + 255*(255-a)) / 255)
					dst.Pix[dOff+2] = uint8((uint16(img.Pix[sOff+2])*a + 255*(255-a)) / 255)
					dst.Pix[dOff+3] = 255
				}
			}
		}
	default:
		for y := 0; y < dstH; y++ {
			sy := bounds.Min.Y + (y*srcH)/dstH
			for x := 0; x < dstW; x++ {
				sx := bounds.Min.X + (x*srcW)/dstW
				c := color.RGBAModel.Convert(srcImg.At(sx, sy)).(color.RGBA)
				dst.SetRGBA(x, y, c)
			}
		}
	}
	return dst
}

func copyFileAtomicAndHash(srcPath string, dstPath string) (string, int64, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = src.Close() }()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o750); err != nil {
		return "", 0, err
	}
	tmpPath := dstPath + ".tmp-" + uuid.NewString()[:8]
	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return "", 0, err
	}

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(dst, hasher), src)
	if err != nil {
		_ = dst.Close()
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	if err := os.Rename(tmpPath, dstPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	return hex.EncodeToString(hasher.Sum(nil)), written, nil
}

func readImageDimensions(path string) (int, int, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return 0, 0, false
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

func mapCatalogCategory(cat string, rel string) (classification string, mediaKind string, format string) {
	ext := strings.ToLower(filepath.Ext(rel))
	mediaKind = "image"
	if ext == ".woff2" || ext == ".woff" || ext == ".ttf" || ext == ".otf" {
		mediaKind = "font"
	} else if ext == ".js" || ext == ".json" || ext == ".md" || ext == ".txt" || ext == ".html" {
		mediaKind = "document"
	}
	format = "other"

	switch cat {
	case "marca", "logo_terceiro":
		return "marca", mediaKind, format
	case "cena_timeline":
		return "cena", mediaKind, "feed"
	case "cena_story":
		return "cena", mediaKind, "story"
	case "cena_site_ou_banco", "cena_legado_aprovado_v5":
		return "cena", mediaKind, format
	case "capa_editorial":
		return "capa_editorial", mediaKind, "cover"
	case "fonte":
		return "fonte", "font", format
	case "historico_fora_da_composicao_final":
		return "historico", mediaKind, format
	default:
		return "referencia", mediaKind, format
	}
}

func resolvePostPageTitle(pieces piecesFileJSON, pieceKey string, pageNum int, fallback string) string {
	for _, p := range pieces.Posts {
		if p.ID == pieceKey {
			if pageNum-1 >= 0 && pageNum-1 < len(p.Pages) && strings.TrimSpace(p.Pages[pageNum-1].Title) != "" {
				return fmt.Sprintf("%s · Pág. %d: %s", p.Title, pageNum, p.Pages[pageNum-1].Title)
			}
			return fmt.Sprintf("%s · Pág. %d", p.Title, pageNum)
		}
	}
	return fallback
}

func resolveStoryTitle(pieces piecesFileJSON, highlightIdx int, pageNum int, fallback string) (string, string) {
	if highlightIdx >= 0 && highlightIdx < len(pieces.Highlights) {
		h := pieces.Highlights[highlightIdx]
		if pageNum-1 >= 0 && pageNum-1 < len(h.Stories) {
			st := h.Stories[pageNum-1]
			return fmt.Sprintf("%s · %d: %s", h.Name, pageNum, st.Title), st.CTA
		}
		return fmt.Sprintf("%s · Story %d", h.Name, pageNum), ""
	}
	return fallback, ""
}

func readSortedDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func readSortedImageFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && isImageExt(e.Name()) {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func isImageExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}

func detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".woff2":
		return "font/woff2"
	case ".webp":
		return "image/webp"
	case ".md":
		return "text/markdown; charset=utf-8"
	}
	if m := mime.TypeByExtension(ext); m != "" {
		return m
	}
	return "application/octet-stream"
}

func parseBRLCents(raw string) int64 {
	cleaned := strings.ReplaceAll(raw, "R$", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimSpace(cleaned)
	var digits strings.Builder
	for _, r := range cleaned {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	if digits.Len() == 0 {
		return 0
	}
	val, err := strconv.ParseInt(digits.String(), 10, 64)
	if err != nil {
		return 0
	}
	return val
}

func sanitizeID(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func defaultIfEmpty(val, fallback string) string {
	if strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}

func ternStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
