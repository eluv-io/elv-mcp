package taggers

import (
	"fmt"
	"strings"
)

// -----------------------------------------------------------------------------
// Model metadata and helpers
// -----------------------------------------------------------------------------

// PENDING: In the future, this metadata can be loaded dynamically from the Tagger
// defaultTaggerModels is the current hardcoded list of supported models.
// In the future, this can be populated from the Tagger /list endpoint.
var defaultTaggerModels = []string{
	"llava",
	"asr",
	"euro_asr",
	"caption",
	"shot",
	"celeb",
	"ocr",
	"logo",
	"vertical_video",
	"speaker",
	"chapters",
}

// GetSupportedModels returns the list of supported model identifiers.
// A copy is returned to avoid accidental mutation by callers.
func GetSupportedModels() []string {
	out := make([]string, len(defaultTaggerModels))
	copy(out, defaultTaggerModels)
	return out
}

// humanModelAliases maps humanized model names to technical identifiers.
var humanModelAliases = map[string]string{
	// ------------------------------------------------------------
	// LLaVA — vision‑language, multimodal understanding
	// ------------------------------------------------------------
	"vision-language":          "llava",
	"multimodal":               "llava",
	"image question answering": "llava",
	"visual qa":                "llava",
	"image understanding":      "llava",

	// ------------------------------------------------------------
	// ASR — English speech recognition
	// ------------------------------------------------------------
	"speech-to-text":        "asr",
	"speech recognition":    "asr",
	"transcription":         "asr",
	"english transcription": "asr",

	// ------------------------------------------------------------
	// Euro ASR — multilingual European speech recognition
	// ------------------------------------------------------------
	"multilingual asr":           "euro_asr",
	"european asr":               "euro_asr",
	"multilingual transcription": "euro_asr",
	"euro speech recognition":    "euro_asr",

	// ------------------------------------------------------------
	// Caption — image captioning
	// ------------------------------------------------------------
	"image captioning":   "caption",
	"caption generation": "caption",
	"describe image":     "caption",

	// ------------------------------------------------------------
	// Shot detection
	// ------------------------------------------------------------
	"shot detection":          "shot",
	"shot boundary detection": "shot",
	"scene change detection":  "shot",

	// ------------------------------------------------------------
	// Celeb — celebrity / face recognition
	// ------------------------------------------------------------
	"celebrity recognition": "celeb",
	"face recognition":      "celeb",
	"face identification":   "celeb",

	// ------------------------------------------------------------
	// OCR — text extraction from images/video
	// ------------------------------------------------------------
	"ocr":              "ocr",
	"text extraction":  "ocr",
	"text recognition": "ocr",
	"read text":        "ocr",

	// ------------------------------------------------------------
	// Logo detection
	// ------------------------------------------------------------
	"logo detection":  "logo",
	"brand detection": "logo",
	"brand logo":      "logo",

	// ------------------------------------------------------------
	// Vertical video detection
	// ------------------------------------------------------------
	"vertical video":  "vertical_video",
	"portrait video":  "vertical_video",
	"vertical format": "vertical_video",

	// ------------------------------------------------------------
	// Speaker diarization
	// ------------------------------------------------------------
	"speaker diarization":  "speaker",
	"who spoke when":       "speaker",
	"speaker segmentation": "speaker",

	// ------------------------------------------------------------
	// Chapters segmentation
	// ------------------------------------------------------------
	"chapter detection":    "chapters",
	"chapter segmentation": "chapters",
	"content chapters":     "chapters",
}

// NormalizeModelName maps humanized names to technical identifiers.
// If no mapping is found, the input is returned lowercased and trimmed.
func NormalizeModelName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if mapped, ok := humanModelAliases[name]; ok {
		return mapped
	}
	return name
}

// DescribeSupportedModels returns a formatted description block for the
// supported models, suitable for inclusion in an MCP tool description.
func DescribeSupportedModels(models []string) string {
	var b strings.Builder

	b.WriteString("Supported models:\n")
	for _, m := range models {
		b.WriteString(fmt.Sprintf("  - `%s`\n", m))
	}
	b.WriteString("\nHumanized model names are also accepted and automatically mapped to the correct identifiers.\n\n")

	return b.String()
}

// -----------------------------------------------------------------------------
// Model dependency registry
// -----------------------------------------------------------------------------

const CharacterModelName = "character"
const ChaptersModelName = "chapters"

var ModelDependencies = map[string][]string{
	CharacterModelName: {"celeb"},
	ChaptersModelName:  {"speaker"},
}
