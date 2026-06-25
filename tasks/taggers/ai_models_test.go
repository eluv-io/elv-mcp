package taggers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks/taggers"
)

// -----------------------------------------------------------------------------
// Test helpers
// -----------------------------------------------------------------------------

func Init_mock_models() {
    taggers.TestInjectModelsJSON([]byte(`{
      "models": [
        {"name": "asr", "description": "Speech to Text", "type": "audio",
         "tag_tracks": [{"name": "speech_to_text"}, {"name": "auto_captions"}],
         "dependencies": []},

        {"name": "speaker", "description": "Speaker Identification", "type": "audio",
         "tag_tracks": [{"name": "speaker_detection"}, {"name": "transcription"}],
         "dependencies": ["auto_captions"]},

        {"name": "chapters", "description": "Chapter Generation", "type": "processor",
         "tag_tracks": [{"name": "chapter"}],
         "dependencies": ["auto_captions"]},

         {"name": "characters", "description": "Character Identification", "type": "processor",
         "tag_tracks": [{"name": "character"}],
         "dependencies": ["auto_captions"]},


        {"name": "shot", "description": "Shot Detection", "type": "video",
         "tag_tracks": [{"name": "shot_detection"}],
         "dependencies": []},

        {"name": "vertical_video", "description": "Focus Identification", "type": "video",
         "tag_tracks": [{"name": "vertical_video"}, {"name": "focus_detection"}],
         "dependencies": ["shot_detection"]}
      ]
    }`))
}


// mockModelsResponse returns a realistic mock of the Tagger /models API.
func mockModelsResponse() taggersTestAPIResponse {
    return taggersTestAPIResponse{
        Models: []taggersTestModel{
            {
                Name:        "shot",
                Description: "Shot Detection",
                TagTracks:   []string{"shot_detection"},
                Dependencies: []string{},
            },
            {
                Name:        "asr",
                Description: "Speech to Text",
                TagTracks:   []string{"speech_to_text"},
                Dependencies: []string{},
            },
            {
                Name:        "chapters",
                Description: "Chapter Generation",
                TagTracks:   []string{"chapter"},
                Dependencies: []string{"speech_to_text"}, // depends on ASR
            },
            {
                Name:        "vertical_video",
                Description: "Focus Identification",
                TagTracks:   []string{"vertical_video"},
                Dependencies: []string{"shot_detection"}, // depends on SHOT
            },
        },
    }
}

// Structures used only for test JSON encoding.
type taggersTestAPIResponse struct {
    Models []taggersTestModel `json:"models"`
}

type taggersTestModel struct {
    Name         string   `json:"name"`
    Description  string   `json:"description"`
    TagTracks    []string `json:"tag_tracks"`
    Dependencies []string `json:"dependencies"`
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

func TestDynamicModelLoadingAndDependencies(t *testing.T) {
    // Mock Tagger API server
    mockData := mockModelsResponse()

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/models" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }

        w.Header().Set("Content-Type", "application/json")

        // Convert test struct to real API format
        type apiModel struct {
            Name         string `json:"name"`
            Description  string `json:"description"`
            Type         string `json:"type"`
            TagTracks    []struct{ Name string `json:"name"` } `json:"tag_tracks"`
            Dependencies []string `json:"dependencies"`
        }

        var apiResp struct {
            Models []apiModel `json:"models"`
        }

        for _, m := range mockData.Models {
            var tracks []struct{ Name string `json:"name"` }
            for _, tt := range m.TagTracks {
                tracks = append(tracks, struct{ Name string `json:"name"` }{Name: tt})
            }

            apiResp.Models = append(apiResp.Models, apiModel{
                Name:         m.Name,
                Description:  m.Description,
                Type:         "video",
                TagTracks:    tracks,
                Dependencies: m.Dependencies,
            })
        }

        json.NewEncoder(w).Encode(apiResp)
    }))
    defer srv.Close()

    cfg := &config.Config{AITaggerUrl: srv.URL}

    // -------------------------------------------------------------------------
    // Validate GetSupportedModels
    // -------------------------------------------------------------------------

    models := taggers.GetSupportedModels(cfg)
    if len(models) != len(mockData.Models) {
        t.Fatalf("expected %d models, got %d", len(mockData.Models), len(models))
    }

    // Ensure copy semantics
    m1 := taggers.GetSupportedModels(cfg)
    m2 := taggers.GetSupportedModels(cfg)
    m1[0] = "modified"
    if m2[0] == "modified" {
        t.Fatalf("GetSupportedModels returned a non-copy slice")
    }

    // -------------------------------------------------------------------------
    // Validate GetModelForTrack
    // -------------------------------------------------------------------------

    if taggers.GetModelForTrack(cfg, "shot_detection") != "shot" {
        t.Fatalf("expected shot_detection → shot")
    }
    if taggers.GetModelForTrack(cfg, "speech_to_text") != "asr" {
        t.Fatalf("expected speech_to_text → asr")
    }
    if taggers.GetModelForTrack(cfg, "chapter") != "chapters" {
        t.Fatalf("expected chapter → chapters")
    }

    // -------------------------------------------------------------------------
    // Validate GetModelDependencies (track → model → model)
    // -------------------------------------------------------------------------

    deps := taggers.GetModelDependencies(cfg)

    // chapters depends on ASR (via speech_to_text)
    if len(deps["chapters"]) != 1 || deps["chapters"][0] != "asr" {
        t.Fatalf("expected chapters → [asr], got %#v", deps["chapters"])
    }

    // vertical_video depends on shot (via shot_detection)
    if len(deps["vertical_video"]) != 1 || deps["vertical_video"][0] != "shot" {
        t.Fatalf("expected vertical_video → [shot], got %#v", deps["vertical_video"])
    }

    // shot and asr have no dependencies
    if len(deps["shot"]) != 0 {
        t.Fatalf("expected shot → [], got %#v", deps["shot"])
    }
    if len(deps["asr"]) != 0 {
        t.Fatalf("expected asr → [], got %#v", deps["asr"])
    }

    // -------------------------------------------------------------------------
    // Validate NormalizeModelName (dynamic + static aliases)
    // -------------------------------------------------------------------------

    if taggers.NormalizeModelName(cfg, "Shot Detection") != "shot" {
        t.Fatalf("expected alias 'Shot Detection' → shot")
    }
    if taggers.NormalizeModelName(cfg, "Focus Identification") != "vertical_video" {
        t.Fatalf("expected alias 'Focus Identification' → vertical_video")
    }

    // static alias still works
    if taggers.NormalizeModelName(cfg, "speech-to-text") != "asr" {
        t.Fatalf("expected static alias speech-to-text → asr")
    }

    // unknown passes through
    if taggers.NormalizeModelName(cfg, "unknown-model") != "unknown-model" {
        t.Fatalf("expected unknown-model to pass through unchanged")
    }

    // -------------------------------------------------------------------------
    // Validate DescribeSupportedModels
    // -------------------------------------------------------------------------

    desc := taggers.DescribeSupportedModels(cfg)
    for _, m := range models {
        if !strings.Contains(desc, m) {
            t.Fatalf("expected description to contain %q", m)
        }
    }
}
