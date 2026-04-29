package fabric_test

import (
	"encoding/json"
	"testing"

	"github.com/qluvio/elv-mcp/tasks/fabric"
)

func TestSummarizeOfferings_Basic(t *testing.T) {
    // log := elog.Get("/test/get_offerings")

    rawJSON := []byte(`{
        "offerings": {
            "default": {
                "ready": true,
                "drm_optional": false,
                "media_struct": {
                    "duration_rat": { "float": 514.36 },
                    "streams": {
                        "audio": {
                            "codec_type": "audio",
                            "codec_name": "aac",
                            "label": "Audio",
                            "language": "en",
                            "channels": 2,
                            "channel_layout": "stereo",
                            "bit_rate": 303437,
                            "default_for_media_type": true
                        },
                        "video": {
                            "codec_type": "video",
                            "codec_name": "h264",
                            "width": 1920,
                            "height": 1080,
                            "rate": "25",
                            "aspect_ratio": "16/9",
                            "bit_rate": 9117864,
                            "default_for_media_type": true
                        }
                    }
                },
                "playout": {
                    "drm_keys": {
                        "219945860bc16c6ba354d800fdb17271": {},
                        "2d272a6a74f10d7628f1885eeac66ec7": {}
                    },
                    "playout_formats": {
                        "dash-clear": {
                            "drm": null,
                            "protocol": { "type": "ProtoDash" }
                        },
                        "dash-widevine": {
                            "drm": { "type": "DrmWidevine" },
                            "protocol": { "type": "ProtoDash" }
                        }
                    },
                    "streams": {
                        "audio": {
                            "encryption_schemes": {
                                "aes-128": { "type": "EncSchemeAes128" }
                            }
                        },
                        "video": {
                            "encryption_schemes": {
                                "cenc": { "type": "EncSchemeCenc" }
                            }
                        }
                    }
                }
            }
        }
    }`)

    var raw map[string]any
    if err := json.Unmarshal(rawJSON, &raw); err != nil {
        t.Fatalf("failed to unmarshal test JSON: %v", err)
    }

    out, err := fabric.SummarizeOfferings(raw)
    if err != nil {
        t.Fatalf("SummarizeOfferings returned error: %v", err)
    }

    if len(out) != 1 {
        t.Fatalf("expected 1 offering, got %d", len(out))
    }

    def, ok := out["default"]
    if !ok {
        t.Fatalf("expected 'default' offering key")
    }

    if def.DurationSeconds <= 0 {
        t.Errorf("expected positive duration_seconds, got %f", def.DurationSeconds)
    }
    if def.DurationHuman == "" {
        t.Errorf("expected non-empty duration_human")
    }
    if def.Video == nil {
        t.Fatalf("expected primary video summary")
    }
    if len(def.AudioTracks) != 1 {
        t.Errorf("expected 1 audio track, got %d", len(def.AudioTracks))
    }
    if len(def.DRM.Schemes) == 0 {
        t.Errorf("expected at least one DRM scheme")
    }
    if len(def.PlayoutFormats) != 2 {
        t.Errorf("expected 2 playout formats, got %d", len(def.PlayoutFormats))
    }
    if !def.Ready {
        t.Errorf("expected ready=true")
    }
}

