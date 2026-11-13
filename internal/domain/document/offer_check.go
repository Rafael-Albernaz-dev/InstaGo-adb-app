package document

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
)

// OfferCheckResult reports consistency between a document's visible prices/text and its linked DatedOffer.
type OfferCheckResult struct {
	HasOffer         bool     `json:"hasOffer"`
	OfferID          string   `json:"offerId,omitempty"`
	OfferTitle       string   `json:"offerTitle,omitempty"`
	ValidUntil       string   `json:"validUntil,omitempty"`
	CheckedAt        string   `json:"checkedAt,omitempty"`
	Status           string   `json:"status"` // none | ok | warning | conflict
	DetectedPrices   []string `json:"detectedPrices"`
	ExpectedPromoBRL string   `json:"expectedPromoBrl,omitempty"`
	ExpectedAfterBRL string   `json:"expectedAfterBrl,omitempty"`
	Messages         []string `json:"messages"`
}

var brlDecimalRe = regexp.MustCompile(`(?:\bR\$\s*)?(\d{1,3}(?:\.\d{3})*|\d+),(\d{2})\b`)

// FormatBRLCents formats integer cents into standard Brazilian currency notation (e.g. 9999 -> "R$ 99,99").
func FormatBRLCents(cents int64) string {
	if cents <= 0 {
		return ""
	}
	whole := cents / 100
	frac := cents % 100
	wholeStr := strconv.FormatInt(whole, 10)
	if len(wholeStr) > 3 {
		var out []byte
		rem := len(wholeStr) % 3
		if rem > 0 {
			out = append(out, wholeStr[:rem]...)
			if len(wholeStr) > rem {
				out = append(out, '.')
			}
		}
		for i := rem; i < len(wholeStr); i += 3 {
			if i > rem {
				out = append(out, '.')
			}
			out = append(out, wholeStr[i:i+3]...)
		}
		wholeStr = string(out)
	}
	return fmt.Sprintf("R$ %s,%02d", wholeStr, frac)
}

// ExtractMonetaryValuesCents returns sorted/deduplicated monetary amounts in cents detected inside text.
func ExtractMonetaryValuesCents(text string) []int64 {
	matches := brlDecimalRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(matches))
	out := make([]int64, 0, len(matches))
	for _, m := range matches {
		wholeClean := strings.ReplaceAll(m[1], ".", "")
		wholeVal, err1 := strconv.ParseInt(wholeClean, 10, 64)
		fracVal, err2 := strconv.ParseInt(m[2], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		cents := wholeVal*100 + fracVal
		if _, exists := seen[cents]; !exists {
			seen[cents] = struct{}{}
			out = append(out, cents)
		}
	}
	return out
}

// CheckOfferConsistency inspects visible text layers across all pages of doc against offer.
func CheckOfferConsistency(doc Document, offer *library.DatedOffer) OfferCheckResult {
	res := OfferCheckResult{
		Status:         "none",
		DetectedPrices: make([]string, 0),
		Messages:       make([]string, 0),
	}
	if offer == nil {
		return res
	}

	res.HasOffer = true
	res.OfferID = offer.ID
	res.OfferTitle = offer.Title
	res.ValidUntil = offer.ValidUntil
	res.CheckedAt = offer.CheckedAt
	res.Status = "ok"
	if offer.PromoPriceCents > 0 {
		res.ExpectedPromoBRL = FormatBRLCents(offer.PromoPriceCents)
	}
	if offer.AfterPriceCents > 0 {
		res.ExpectedAfterBRL = FormatBRLCents(offer.AfterPriceCents)
	}

	allowed := make(map[int64]bool)
	for _, val := range []int64{offer.PromoPriceCents, offer.AfterPriceCents, offer.InstallFeeCents} {
		if val > 0 {
			allowed[val] = true
		}
	}

	var allText strings.Builder
	detectedSeen := make(map[int64]bool)
	mismatches := make([]int64, 0)

	for _, p := range doc.Pages {
		for _, l := range p.Layers {
			if !l.Visible || l.Kind != "text" || strings.TrimSpace(l.Text) == "" {
				continue
			}
			allText.WriteString(l.Text)
			allText.WriteByte('\n')
			for _, cents := range ExtractMonetaryValuesCents(l.Text) {
				if !detectedSeen[cents] {
					detectedSeen[cents] = true
					res.DetectedPrices = append(res.DetectedPrices, FormatBRLCents(cents))
				}
				if len(allowed) > 0 && !allowed[cents] {
					mismatches = append(mismatches, cents)
				}
			}
		}
	}

	if len(mismatches) > 0 {
		res.Status = "conflict"
		for _, bad := range mismatches {
			res.Messages = append(res.Messages, fmt.Sprintf(
				"Valor %s na arte não corresponde aos valores oficiais da oferta %q (%s).",
				FormatBRLCents(bad), offer.Title, formatExpectedSummary(*offer),
			))
		}
	}

	fullLower := strings.ToLower(allText.String())
	if offer.PromoPriceCents > 0 && !detectedSeen[offer.PromoPriceCents] && len(detectedSeen) > 0 {
		if res.Status != "conflict" {
			res.Status = "conflict"
		}
		res.Messages = append(res.Messages, fmt.Sprintf(
			"Preço promocional esperado (%s) não foi encontrado nos textos visíveis da peça.",
			FormatBRLCents(offer.PromoPriceCents),
		))
	}

	if offer.PromoPriceCents > 0 && offer.AfterPriceCents > 0 && !detectedSeen[offer.AfterPriceCents] {
		if res.Status == "ok" {
			res.Status = "warning"
		}
		res.Messages = append(res.Messages, fmt.Sprintf(
			"Condição posterior (%s) não aparece nas camadas visíveis da peça.",
			FormatBRLCents(offer.AfterPriceCents),
		))
	}

	if offer.PromoPeriod != "" && detectedSeen[offer.PromoPriceCents] && !strings.Contains(fullLower, strings.ToLower(offer.PromoPeriod)) {
		if res.Status == "ok" {
			res.Status = "warning"
		}
		res.Messages = append(res.Messages, fmt.Sprintf(
			"Período promocional da oferta (%q) não aparece explicitamente nos textos.",
			offer.PromoPeriod,
		))
	}

	if res.Status == "ok" {
		res.Messages = append(res.Messages, fmt.Sprintf(
			"Preços alinhados com %s (conferência em %s).",
			offer.Title, offer.CheckedAt,
		))
	}

	return res
}

func formatExpectedSummary(offer library.DatedOffer) string {
	var parts []string
	if offer.PromoPriceCents > 0 {
		parts = append(parts, "promo "+FormatBRLCents(offer.PromoPriceCents))
	}
	if offer.AfterPriceCents > 0 {
		parts = append(parts, "depois "+FormatBRLCents(offer.AfterPriceCents))
	}
	if offer.InstallFeeCents > 0 {
		parts = append(parts, "instalação "+FormatBRLCents(offer.InstallFeeCents))
	}
	if len(parts) == 0 {
		return "sem valor monetário cadastrado"
	}
	return strings.Join(parts, ", ")
}
