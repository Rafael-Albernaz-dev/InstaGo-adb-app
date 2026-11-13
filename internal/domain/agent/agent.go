package agent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/document"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
)

// Status represents the explicit lifecycle state of an agent task.
type Status string

const (
	StatusPending              Status = "pending"
	StatusRunning              Status = "running"
	StatusNeedsConfirmation    Status = "needs_confirmation"
	StatusCompleted            Status = "completed"
	StatusFailed               Status = "failed"
	StatusCanceled             Status = "canceled"
	StatusInterruptedUncertain Status = "interrupted_uncertain"
)

// DeliveryMode defines how a generated/edited image integrates back into the editable document.
type DeliveryMode string

const (
	ModeAuto            DeliveryMode = "auto"
	ModeBackgroundLayer DeliveryMode = "background_layer"
	ModeFullPiece       DeliveryMode = "full_piece"
)

// CLIHealth reports local Codex CLI availability and ChatGPT login status without exposing secrets.
type CLIHealth struct {
	Available      bool   `json:"available"`
	BinaryPath     string `json:"binaryPath"`
	Version        string `json:"version"`
	Authenticated  bool   `json:"authenticated"`
	AuthStatusText string `json:"authStatusText"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
}

// Task stores the metadata and outcome of a single Codex CLI generation/edit request.
type Task struct {
	ID                  string       `json:"id"`
	DocumentID          string       `json:"documentId"`
	PageIndex           int          `json:"pageIndex"`
	RequestedMode       DeliveryMode `json:"requestedMode"`
	ResolvedMode        DeliveryMode `json:"resolvedMode"`
	Prompt              string       `json:"prompt"`
	ConfirmationReason  string       `json:"confirmationReason,omitempty"`
	Status              Status       `json:"status"`
	ThreadID            string       `json:"threadId,omitempty"`
	WorkDir             string       `json:"workDir,omitempty"`
	OutputImagePath     string       `json:"outputImagePath,omitempty"`
	OutputAssetID       string       `json:"outputAssetId,omitempty"`
	OutputAssetURL      string       `json:"outputAssetUrl,omitempty"`
	ArtifactRecoverable bool         `json:"artifactRecoverable,omitempty"`
	InstructionsVersion string       `json:"instructionsVersion,omitempty"`
	ContextSummary      string       `json:"contextSummary,omitempty"`
	ResultDocumentID    string       `json:"resultDocumentId,omitempty"`
	ResultRevision      int          `json:"resultRevision,omitempty"`
	SummaryText         string       `json:"summaryText,omitempty"`
	ErrorCode           string       `json:"errorCode,omitempty"`
	ErrorMessage        string       `json:"errorMessage,omitempty"`
	StartedAt           string       `json:"startedAt"`
	UpdatedAt           string       `json:"updatedAt"`
	FinishedAt          string       `json:"finishedAt,omitempty"`
}

// LogEntry captures one live terminal-style line streamed from Codex CLI or the agent runner.
type LogEntry struct {
	ID        string `json:"id"`
	TaskID    string `json:"taskId"`
	Seq       int    `json:"seq"`
	Stream    string `json:"stream"` // system | stdout | agent | tool | stderr | error
	EventType string `json:"eventType,omitempty"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

// TaskDetail bundles a Task with all persisted streaming logs.
type TaskDetail struct {
	Task Task       `json:"task"`
	Logs []LogEntry `json:"logs"`
}

// StartTaskRequest carries the operator's agent prompt and optional rendered page PNG snapshot.
type StartTaskRequest struct {
	DocumentID         string       `json:"documentId"`
	PageIndex          int          `json:"pageIndex"`
	Mode               DeliveryMode `json:"mode"`
	Prompt             string       `json:"prompt"`
	SnapshotBase64     string       `json:"snapshotBase64,omitempty"`
	ConfirmBeforeRun   bool         `json:"confirmBeforeRun,omitempty"`
	IncludeAnnotations bool         `json:"includeAnnotations"`
}

// PreflightResult summarizes prompt validation, ambiguity/commercial checks, and resolved mode.
type PreflightResult struct {
	Valid             bool         `json:"valid"`
	NeedsConfirmation bool         `json:"needsConfirmation"`
	Reason            string       `json:"reason,omitempty"`
	ErrorMessage      string       `json:"errorMessage,omitempty"`
	ResolvedMode      DeliveryMode `json:"resolvedMode"`
	EffectivePrompt   string       `json:"effectivePrompt"`
}

// PromptBuildInput provides all verified domain context needed to assemble a clean Codex prompt.
type PromptBuildInput struct {
	Document          document.Document
	Page              document.Page
	Profile           *library.CompanyProfile
	Offer             *library.DatedOffer
	Mode              DeliveryMode
	Prompt            string
	Annotations       []document.AgentAnnotation
	SnapshotAttached  bool
	TargetOutputPath  string
	SkillInstructions string
}

var brlPriceRe = regexp.MustCompile(`(?i)R\$\s*([0-9]{1,3}(?:\.[0-9]{3})*(?:,[0-9]{2})|[0-9]+(?:,[0-9]{2})?)`)

// ResolveDeliveryMode decides whether the agent output should feed a background photo layer
// or regenerate a full piece revision based on explicit preference, page structure, and prompt text.
func ResolveDeliveryMode(prompt string, requested DeliveryMode, page document.Page) DeliveryMode {
	switch requested {
	case ModeBackgroundLayer, ModeFullPiece:
		return requested
	}

	lower := strings.ToLower(strings.TrimSpace(prompt))

	fullPieceSignals := []string{
		"arte inteira", "banner inteiro", "peça inteira", "peca inteira",
		"refazer arte", "refazer a arte", "editar o banner", "corrigir o layout",
		"mudar o texto", "alterar o preço", "alterar preco", "remover texto da imagem",
		"na arte pronta", "arte completa",
	}
	for _, sig := range fullPieceSignals {
		if strings.Contains(lower, sig) {
			return ModeFullPiece
		}
	}

	bgSignals := []string{
		"fundo", "cenário", "cenario", "cena", "foto", "fotografia",
		"paisagem", "varanda", "família", "familia", "sítio", "sitio",
		"fazenda", "câmera", "camera", "roteador", "lavoura", "pasto",
		"trocar a foto", "gerar uma foto", "nova imagem", "imagem de fundo",
	}
	for _, sig := range bgSignals {
		if strings.Contains(lower, sig) {
			return ModeBackgroundLayer
		}
	}

	hasTextOrShapeOverlay := false
	imageCount := 0
	for _, l := range page.Layers {
		if l.Kind == "text" || l.Kind == "rect" {
			hasTextOrShapeOverlay = true
		}
		if l.Kind == "image" {
			imageCount++
		}
	}
	if hasTextOrShapeOverlay {
		return ModeBackgroundLayer
	}
	if imageCount == 1 {
		return ModeFullPiece
	}
	return ModeBackgroundLayer
}

// AnalyzePreflight validates prompt readiness and flags commercial/pricing conflicts before running Codex.
func AnalyzePreflight(
	prompt string,
	requested DeliveryMode,
	doc document.Document,
	pageIndex int,
	offer *library.DatedOffer,
	offerCheck document.OfferCheckResult,
) PreflightResult {
	var page document.Page
	if pageIndex >= 0 && pageIndex < len(doc.Pages) {
		page = doc.Pages[pageIndex]
	}

	trimmedPrompt := strings.TrimSpace(prompt)
	var annotationNotes []string
	for _, a := range page.Annotations {
		if !a.Resolved && strings.TrimSpace(a.Note) != "" {
			annotationNotes = append(annotationNotes, strings.TrimSpace(a.Note))
		}
	}

	effective := trimmedPrompt
	if effective == "" && len(annotationNotes) > 0 {
		effective = "Aplicar na página as marcações indicadas: " + strings.Join(annotationNotes, "; ")
	}
	if effective == "" {
		return PreflightResult{
			Valid:        false,
			ErrorMessage: "Escreva o que deseja pedir ao agente ou adicione uma nota nas marcações da arte.",
		}
	}

	resolved := ResolveDeliveryMode(effective, requested, page)

	if offerCheck.Status == "conflict" {
		return PreflightResult{
			Valid:             true,
			NeedsConfirmation: true,
			Reason:            strings.Join(offerCheck.Messages, " "),
			ResolvedMode:      resolved,
			EffectivePrompt:   effective,
		}
	}

	combinedText := effective + " " + strings.Join(annotationNotes, " ")
	citedCents := extractPriceCents(combinedText)
	if len(citedCents) > 0 {
		if offer == nil {
			if resolved == ModeFullPiece {
				return PreflightResult{
					Valid:             true,
					NeedsConfirmation: true,
					Reason:            "O pedido cita valores em R$, mas esta peça não está vinculada a nenhuma oferta oficial cadastrada. Confirme para continuar sem inventar condições.",
					ResolvedMode:      resolved,
					EffectivePrompt:   effective,
				}
			}
		} else {
			for _, cents := range citedCents {
				if cents != offer.PromoPriceCents && cents != offer.AfterPriceCents && cents != offer.InstallFeeCents {
					return PreflightResult{
						Valid:             true,
						NeedsConfirmation: true,
						Reason:            fmt.Sprintf("O pedido menciona %s, valor diferente da oferta vinculada %q. Confirme se deseja usar esse valor mesmo assim.", formatBRL(cents), offer.Title),
						ResolvedMode:      resolved,
						EffectivePrompt:   effective,
					}
				}
			}
		}
	}

	return PreflightResult{
		Valid:           true,
		ResolvedMode:    resolved,
		EffectivePrompt: effective,
	}
}

// BuildGenerationPrompt builds the exact contextual instruction sent to Codex CLI via stdin.
func BuildGenerationPrompt(in PromptBuildInput) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Você está atuando no estúdio criativo local InstaGo para gerar/editar uma imagem para Instagram.\n")
	if in.Profile != nil {
		fmt.Fprintf(&b, "Empresa: %s (%s) · Região: %s · Público: %s\n",
			in.Profile.Brand, in.Profile.InstagramHandle, in.Profile.Region, in.Profile.ClientAudience)
		fmt.Fprintf(&b, "Contatos oficiais: Telefone/WhatsApp %s | Site %s\n",
			in.Profile.PhoneDisplay, in.Profile.SiteURL)
	}
	fmt.Fprintf(&b, "Identidade visual obrigatória: paleta #3D6A2F (verde marca), #98B432 (destaque/CTA lima), #374F1E (texto escuro), #F6F7EE (creme claro); estilo fotográfico/fotorrealista em ambiente rural brasileiro crível, nunca cartoon/desenho animado.\n\n")

	if strings.TrimSpace(in.SkillInstructions) != "" {
		b.WriteString("Diretrizes da skill local:\n")
		b.WriteString(strings.TrimSpace(in.SkillInstructions))
		b.WriteString("\n\n")
	}

	formatDesc := fmt.Sprintf("Feed retrato %dx%d px (proporção 4:5)", in.Document.ExportWidth, in.Document.ExportHeight)
	if in.Document.Format == document.FormatStory {
		formatDesc = fmt.Sprintf("Story vertical %dx%d px (proporção 9:16), deixando áreas seguras no topo e na base", in.Document.ExportWidth, in.Document.ExportHeight)
	}
	fmt.Fprintf(&b, "Peça aberta: %q (página %d/%d — %q) · Formato de saída: %s.\n",
		in.Document.Title, in.Page.Index+1, maxInt(1, len(in.Document.Pages)), in.Page.Title, formatDesc)

	if in.Offer != nil {
		fmt.Fprintf(&b, "Oferta vinculada (dados factuais, não inventar preço/benefício diferente): %q · Promoção: %s (%s) · Após promoção: %s · Instalação: %s · Detalhes: %s %s\n",
			in.Offer.Title,
			formatBRL(in.Offer.PromoPriceCents),
			in.Offer.PromoPeriod,
			formatBRL(in.Offer.AfterPriceCents),
			formatBRL(in.Offer.InstallFeeCents),
			in.Offer.SpeedSummary,
			in.Offer.ServicesSummary,
		)
	}

	if in.Mode == ModeBackgroundLayer {
		b.WriteString("\nModo de entrega: APENAS FOTO/CENA DE FUNDO EDITÁVEL.\n")
		b.WriteString("- Gere ou ajuste somente a fotografia/cena de fundo usando a ferramenta nativa image_gen.\n")
		b.WriteString("- NÃO escreva textos, preços, selos, botões nem desenhe logomarcas por cima da imagem (textos e logos ficarão em camadas editáveis no editor).\n")
		b.WriteString("- Deixe uma área limpa de respiro visual compatível com textos por cima e mantenha pessoas/rostos/equipamentos inteiros sem cortes.\n")
	} else {
		b.WriteString("\nModo de entrega: AJUSTE COMPLETO DA ARTE DA PÁGINA.\n")
		b.WriteString("- Use a ferramenta nativa image_gen para gerar/editar a peça respeitando exatamente o formato pedido.\n")
		b.WriteString("- Preserve intacto tudo o que não foi pedido para alterar nas marcações e mantenha alta legibilidade para público leigo e idoso.\n")
	}

	if in.SnapshotAttached {
		b.WriteString("\nUma captura visual da página atual foi anexada (-i). ")
		if len(in.Annotations) > 0 {
			b.WriteString("As áreas ou pinos numerados em laranja na captura são apenas guias visuais do operador para mostrar onde agir — NÃO desenhe esses marcadores laranjas na imagem final!\n")
		} else {
			b.WriteString("Use-a como referência visual da composição atual.\n")
		}
	}

	if len(in.Annotations) > 0 {
		b.WriteString("\nMarcações visuais nesta página:\n")
		for i, a := range in.Annotations {
			if a.Kind == "rect" {
				fmt.Fprintf(&b, "- Marcação #%d [Área em %.0f,%.0f tamanho %.0fx%.0f na prancheta %.0fx%.0f]: %s\n",
					i+1, a.X, a.Y, a.Width, a.Height, in.Document.WorkspaceWidth, in.Document.WorkspaceHeight, strings.TrimSpace(a.Note))
			} else {
				fmt.Fprintf(&b, "- Marcação #%d [Ponto em %.0f,%.0f na prancheta %.0fx%.0f]: %s\n",
					i+1, a.X, a.Y, in.Document.WorkspaceWidth, in.Document.WorkspaceHeight, strings.TrimSpace(a.Note))
			}
		}
	}

	fmt.Fprintf(&b, "\nPedido do operador:\n%s\n\n", strings.TrimSpace(in.Prompt))
	fmt.Fprintf(&b, "Requisito obrigatório de arquivo:\n")
	fmt.Fprintf(&b, "- Salve a imagem PNG final gerada exatamente em: %s\n", in.TargetOutputPath)
	fmt.Fprintf(&b, "- Só finalize após confirmar que o arquivo %s existe e está legível; responda de forma breve em português descrevendo o resultado gerado.\n", in.TargetOutputPath)

	return b.String()
}

func extractPriceCents(text string) []int64 {
	matches := brlPriceRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]int64, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		raw := strings.ReplaceAll(m[1], ".", "")
		parts := strings.SplitN(raw, ",", 2)
		whole, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		cents := whole * 100
		if len(parts) == 2 && len(parts[1]) == 2 {
			dec, decErr := strconv.ParseInt(parts[1], 10, 64)
			if decErr == nil {
				cents += dec
			}
		}
		out = append(out, cents)
	}
	return out
}

func formatBRL(cents int64) string {
	whole := cents / 100
	rem := cents % 100
	return fmt.Sprintf("R$ %d,%02d", whole, rem)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
