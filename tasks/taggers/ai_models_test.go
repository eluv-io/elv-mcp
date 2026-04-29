package taggers_test

import (
	"strings"
	"testing"

	"github.com/qluvio/elv-mcp/tasks/taggers"
)

func TestGetSupportedModels_ReturnsCopy(t *testing.T) {
	models1 := taggers.GetSupportedModels()
	models2 := taggers.GetSupportedModels()

	if len(models1) == 0 {
		t.Fatalf("expected non-empty model list")
	}

	// Mutate the returned slice and ensure original is unaffected
	models1[0] = "modified"

	models3 := taggers.GetSupportedModels()
	if models3[0] == "modified" {
		t.Fatalf("GetSupportedModels returned a slice that was not a copy")
	}

	// Ensure two calls return equal content but different backing arrays
	if len(models1) != len(models2) {
		t.Fatalf("expected same length slices")
	}
}

func TestNormalizeModelName_TechnicalNames(t *testing.T) {
	tests := []string{"asr", "celeb", "logo", "shot", "llava", "ocr"}

	for _, m := range tests {
		out := taggers.NormalizeModelName(m)
		if out != m {
			t.Fatalf("expected %q to normalize to itself, got %q", m, out)
		}
	}
}

func TestNormalizeModelName_HumanizedAliases(t *testing.T) {
	cases := map[string]string{
		"speech-to-text":     "asr",
		"Speech Recognition": "asr",
		"transcription":      "asr",

		"celebrity recognition": "celeb",
		"face recognition":      "celeb",

		"logo detection": "logo",

		"shot detection":          "shot",
		"Shot Boundary Detection": "shot",

		"vision-language": "llava",
		"multimodal":      "llava",

		"ocr":             "ocr",
		"text extraction": "ocr",
	}

	for input, expected := range cases {
		out := taggers.NormalizeModelName(input)
		if out != expected {
			t.Fatalf("NormalizeModelName(%q) = %q, expected %q", input, out, expected)
		}
	}
}

func TestNormalizeModelName_UnknownPassThrough(t *testing.T) {
	out := taggers.NormalizeModelName("unknown-model")
	if out != "unknown-model" {
		t.Fatalf("expected unknown model to pass through unchanged, got %q", out)
	}
}

func TestDescribeSupportedModels_ContainsAllModels(t *testing.T) {
	models := taggers.GetSupportedModels()
	desc := taggers.DescribeSupportedModels(models)

	for _, m := range models {
		if !strings.Contains(desc, m) {
			t.Fatalf("expected description to contain model %q", m)
		}
	}

	if !strings.Contains(desc, "Humanized model names") {
		t.Fatalf("expected description to mention humanized model names")
	}
}
