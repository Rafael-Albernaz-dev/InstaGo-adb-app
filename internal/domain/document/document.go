package document

import (
	"errors"
	"fmt"
	"strings"
)

// Format identifies supported Instagram aspect ratios and export sizes.
type Format string

const (
	FormatFeed  Format = "feed"
	FormatStory Format = "story"
)

const (
	FeedExportWidth   = 1080
	FeedExportHeight  = 1350
	StoryExportWidth  = 1080
	StoryExportHeight = 1920
	WorkspaceWidth    = 600.0
	ExportScale       = 1.8
)

// PresetDimensions returns workspace and export pixel dimensions for a format.
func PresetDimensions(format Format) (wsW, wsH float64, expW, expH int, scale float64, err error) {
	switch format {
	case FormatFeed:
		return WorkspaceWidth, 750.0, FeedExportWidth, FeedExportHeight, ExportScale, nil
	case FormatStory:
		return WorkspaceWidth, 1066.6667, StoryExportWidth, StoryExportHeight, ExportScale, nil
	default:
		return 0, 0, 0, 0, 0, fmt.Errorf("unsupported format %q", format)
	}
}

// Layer represents one editable visual element on a page.
type Layer struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Kind          string  `json:"kind"` // rect | text | image
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Width         float64 `json:"width"`
	Height        float64 `json:"height"`
	ScaleX        float64 `json:"scaleX"`
	ScaleY        float64 `json:"scaleY"`
	Angle         float64 `json:"angle"`
	Opacity       float64 `json:"opacity"`
	Visible       bool    `json:"visible"`
	Locked        bool    `json:"locked"`
	ZIndex        int     `json:"zIndex"`
	Fill          string  `json:"fill,omitempty"`
	Stroke        string  `json:"stroke,omitempty"`
	StrokeWidth   float64 `json:"strokeWidth,omitempty"`
	Radius        float64 `json:"radius,omitempty"`
	Text          string  `json:"text,omitempty"`
	FontFamily    string  `json:"fontFamily,omitempty"`
	FontSize      float64 `json:"fontSize,omitempty"`
	FontWeight    string  `json:"fontWeight,omitempty"`
	LineHeight    float64 `json:"lineHeight,omitempty"`
	LetterSpacing float64 `json:"letterSpacing,omitempty"`
	TextAlign     string  `json:"textAlign,omitempty"`
	AssetKey      string  `json:"assetKey,omitempty"`
	AssetURL      string  `json:"assetUrl,omitempty"`
	ObjectFit     string  `json:"objectFit,omitempty"`
	PositionX     float64 `json:"positionX,omitempty"` // 0..1 focal point X
	PositionY     float64 `json:"positionY,omitempty"` // 0..1 focal point Y
}

// Page represents a single artboard inside a document (1 page for Story/single post, N for carousel).
type Page struct {
	ID         string  `json:"id"`
	Index      int     `json:"index"`
	Title      string  `json:"title"`
	Background string  `json:"background"`
	Layers     []Layer `json:"layers"`
}

// Document is InstaGo's versioned editable composition schema.
type Document struct {
	ID                string  `json:"id"`
	SchemaVersion     int     `json:"schemaVersion"`
	Title             string  `json:"title"`
	CompanyID         string  `json:"companyId"`
	OfferRef          string  `json:"offerRef"`
	Format            Format  `json:"format"`
	WorkspaceWidth    float64 `json:"workspaceWidth"`
	WorkspaceHeight   float64 `json:"workspaceHeight"`
	ExportWidth       int     `json:"exportWidth"`
	ExportHeight      int     `json:"exportHeight"`
	ExportScale       float64 `json:"exportScale"`
	ReferenceImageURL string  `json:"referenceImageUrl,omitempty"`
	Revision          int     `json:"revision"`
	Status            string  `json:"status"` // draft | in_review | approved
	Pages             []Page  `json:"pages"`
	CanvasJSON        string  `json:"canvasJson,omitempty"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

// Validate enforces structural and export invariants on the document.
func (d Document) Validate() error {
	if strings.TrimSpace(d.ID) == "" {
		return errors.New("document id is required")
	}
	if strings.TrimSpace(d.Title) == "" {
		return errors.New("document title is required")
	}
	if d.SchemaVersion < 1 {
		return errors.New("schemaVersion must be >= 1")
	}
	if _, _, expW, expH, _, err := PresetDimensions(d.Format); err != nil {
		return err
	} else if d.ExportWidth != expW || d.ExportHeight != expH {
		return fmt.Errorf("export dimensions %dx%d do not match format %s (%dx%d)", d.ExportWidth, d.ExportHeight, d.Format, expW, expH)
	}
	if d.WorkspaceWidth <= 0 || d.WorkspaceHeight <= 0 || d.ExportScale <= 0 {
		return errors.New("workspace dimensions and exportScale must be positive")
	}
	if len(d.Pages) == 0 {
		return errors.New("document must contain at least one page")
	}
	seenLayers := make(map[string]struct{})
	for i, p := range d.Pages {
		if strings.TrimSpace(p.ID) == "" {
			return fmt.Errorf("page %d id is required", i)
		}
		if p.Index != i {
			return fmt.Errorf("page %s index must be %d, got %d", p.ID, i, p.Index)
		}
		for _, l := range p.Layers {
			if strings.TrimSpace(l.ID) == "" {
				return fmt.Errorf("page %s contains layer with empty id", p.ID)
			}
			if _, dup := seenLayers[l.ID]; dup {
				return fmt.Errorf("duplicate layer id %q", l.ID)
			}
			seenLayers[l.ID] = struct{}{}
			switch l.Kind {
			case "rect", "text", "image":
			default:
				return fmt.Errorf("unsupported layer kind %q on layer %s", l.Kind, l.ID)
			}
		}
	}
	return nil
}
