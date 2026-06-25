package taggers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/qluvio/elv-mcp/config"
)

// -----------------------------------------------------------------------------
// API response structures
// -----------------------------------------------------------------------------

type apiTaggerModel struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	TagTracks   []struct {
		Name string `json:"name"`
	} `json:"tag_tracks"`
	Dependencies []string `json:"dependencies"`
}

type apiTaggerModelsResponse struct {
	Models []apiTaggerModel `json:"models"`
}

// -----------------------------------------------------------------------------
// Static aliases (merged with dynamic aliases)
// -----------------------------------------------------------------------------

var staticHumanAliases = map[string]string{
        // ------------------------------------------------------------
	// LLaVA — vision-language, multimodal understanding
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

// -----------------------------------------------------------------------------
// Dynamic caches
// -----------------------------------------------------------------------------

var (
	cachedModels       []string
	cachedDependencies map[string][]string // model -> []model
	cachedAliases      map[string]string   // human name -> model
	cachedTrackToModel map[string]string   // track -> model

	loadOnce sync.Once
	loadErr  error

	// testInjectedModelsJSON is used only in tests to bypass the HTTP call in
	// loadModelsFromAPI. When non-nil, the loader will decode this JSON instead
	// of calling the Tagger /models endpoint.
	testInjectedModelsJSON []byte
)

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------
const CharacterModelName = "characters"
const ChaptersModelName = "chapters"

// -----------------------------------------------------------------------------
// Public API
// -----------------------------------------------------------------------------

// GetSupportedModels returns the list of models loaded from the API.
// If the API failed, it returns an empty slice.
func GetSupportedModels(cfg *config.Config) []string {
	loadOnce.Do(func() {
		loadErr = loadModelsFromAPI(cfg)
	})

	out := make([]string, len(cachedModels))
	copy(out, cachedModels)
	return out
}

// GetModelDependencies returns model → []model dependencies.
// If the API failed, it returns an empty map.
func GetModelDependencies(cfg *config.Config) map[string][]string {
	GetSupportedModels(cfg)

	out := make(map[string][]string, len(cachedDependencies))
	for k, v := range cachedDependencies {
		cp := append([]string(nil), v...)
		out[k] = cp
	}
	return out
}

// GetModelForTrack returns the model that produces a given track.
// If the API failed or the track is unknown, returns "".
func GetModelForTrack(cfg *config.Config, trackName string) string {
	GetSupportedModels(cfg)

	if cachedTrackToModel == nil {
		return ""
	}
	if m, ok := cachedTrackToModel[trackName]; ok {
		return m
	}
	return ""
}

// NormalizeModelName maps humanized names to technical identifiers.
// If the API failed, only static aliases are available.
func NormalizeModelName(cfg *config.Config, name string) string {
	GetSupportedModels(cfg)

	name = strings.ToLower(strings.TrimSpace(name))
	if mapped, ok := cachedAliases[name]; ok {
		return mapped
	}
	return name
}

// DescribeSupportedModels returns a formatted description block.
// If the API failed, the list will be empty.
func DescribeSupportedModels(cfg *config.Config) string {
	models := GetSupportedModels(cfg)

	var b strings.Builder
	b.WriteString("Supported models:\n")
	for _, m := range models {
		b.WriteString(fmt.Sprintf("  - `%s`\n", m))
	}
	b.WriteString("\nHumanized model names are also accepted and automatically mapped to the correct identifiers.\n\n")
	return b.String()
}

// -----------------------------------------------------------------------------
// Internal dynamic loader
// -----------------------------------------------------------------------------

func loadModelsFromAPI(cfg *config.Config) error {
	var apiResp apiTaggerModelsResponse

	// Test mode: if a JSON payload has been injected, use it instead of HTTP.
	if testInjectedModelsJSON != nil {
		if err := json.Unmarshal(testInjectedModelsJSON, &apiResp); err != nil {
			return fmt.Errorf("failed to decode injected test models: %w", err)
		}
	} else {
		// Normal mode: call the real API.
		base := strings.TrimRight(cfg.AITaggerUrl, "/")
		url := base + "/models"

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to fetch models: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %s", resp.Status)
		}

		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return fmt.Errorf("failed to decode models: %w", err)
		}
	}

	// From here down, keep your existing code exactly as it is:
	models := make([]string, len(apiResp.Models))
	trackToModel := make(map[string]string)
	depsByModel := make(map[string][]string)
	aliases := make(map[string]string)

	// First pass: model names, track→model, dynamic aliases
	for i, m := range apiResp.Models {
		models[i] = m.Name

		for _, tt := range m.TagTracks {
			trackToModel[tt.Name] = m.Name
		}

		aliases[strings.ToLower(m.Description)] = m.Name
		for _, tt := range m.TagTracks {
			aliases[strings.ReplaceAll(tt.Name, "_", " ")] = m.Name
		}
	}

	// Second pass: resolve dependencies (track names) to model names
	for _, m := range apiResp.Models {
		var modelDeps []string
		for _, depTrack := range m.Dependencies {
			if depModel, ok := trackToModel[depTrack]; ok {
				modelDeps = append(modelDeps, depModel)
			}
		}
		depsByModel[m.Name] = modelDeps
	}

	for k, v := range staticHumanAliases {
		aliases[k] = v
	}

	cachedModels = models
	cachedTrackToModel = trackToModel
	cachedDependencies = depsByModel
	cachedAliases = aliases

	return nil
}
