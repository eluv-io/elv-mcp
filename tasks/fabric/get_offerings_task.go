package fabric

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
)

// GetOfferingsArgs defines the input arguments for the content.get_offerings tool.
type GetOfferingsArgs struct {
    // ContentID is the content hash, ID, or write token of the content object.
    // It will be used both as the QID for authorization and as the qhit in the struct URL.
    ContentID string `json:"content_id"`
}

// GetOfferingsResult is the summarized view of all offerings for a content object.
//
// The result is a map keyed by offering name (e.g. "default", "hd", "sd").
// Each value is a compact, user-facing summary of that offering.
type GetOfferingsResult struct {
    Offerings map[string]OfferingSummary `json:"offerings"`
}

// OfferingSummary is the compact, user-facing summary of a single offering.
type OfferingSummary struct {
    DurationSeconds float64                     `json:"duration_seconds"`
    DurationHuman   string                      `json:"duration_human"`
    Video           *VideoSummary               `json:"video,omitempty"`
    VideoTracks     []VideoTrackSummary         `json:"video_tracks,omitempty"`
    AudioTracks     []AudioTrackSummary         `json:"audio_tracks,omitempty"`
    SubtitleTracks  []SubtitleTrackSummary      `json:"subtitle_tracks,omitempty"`
    DRM             DRMInfo                     `json:"drm"`
    PlayoutFormats  []PlayoutFormatSummary      `json:"playout_formats,omitempty"`
    Ready           bool                        `json:"ready"`
}

// VideoSummary describes the primary video track for an offering.
type VideoSummary struct {
    Codec              string `json:"codec"`
    Width              int    `json:"width"`
    Height             int    `json:"height"`
    FrameRate          string `json:"frame_rate"`
    AspectRatio        string `json:"aspect_ratio"`
    HDR                any    `json:"hdr"` // passthrough; may be null or a struct
    Bitrate            int64  `json:"bitrate"`
    DefaultForMediaType bool  `json:"default_for_media_type"`
}

// VideoTrackSummary describes a single video track (all video tracks are listed here).
type VideoTrackSummary struct {
    Codec               string `json:"codec"`
    Width               int    `json:"width"`
    Height              int    `json:"height"`
    FrameRate           string `json:"frame_rate"`
    AspectRatio         string `json:"aspect_ratio"`
    HDR                 any    `json:"hdr"`
    Bitrate             int64  `json:"bitrate"`
    DefaultForMediaType bool   `json:"default_for_media_type"`
}

// AudioTrackSummary describes a single audio track.
type AudioTrackSummary struct {
    Label               string `json:"label"`
    Language            string `json:"language"`
    Channels            int    `json:"channels"`
    Layout              string `json:"layout"`
    Codec               string `json:"codec"`
    Bitrate             int64  `json:"bitrate"`
    DefaultForMediaType bool   `json:"default_for_media_type"`
}

// SubtitleTrackSummary describes a single subtitle track.
//
// NOTE: This is intentionally minimal but extensible. Additional fields (e.g. bitrate,
// duration, etc.) can be added later if needed without breaking existing consumers.
type SubtitleTrackSummary struct {
    Label               string `json:"label"`
    Language            string `json:"language"`
    Codec               string `json:"codec"`
    Forced              *bool  `json:"forced,omitempty"`
    HearingImpaired     *bool  `json:"hearing_impaired,omitempty"`
    DefaultForMediaType bool   `json:"default_for_media_type"`
}

// DRMInfo summarizes DRM configuration for an offering.
type DRMInfo struct {
    Optional bool     `json:"optional"`
    Schemes  []string `json:"schemes"`
}

// PlayoutFormatSummary describes a single playout format (e.g. dash-clear, hls-fairplay).
type PlayoutFormatSummary struct {
    Name     string `json:"name"`
    DRM      string `json:"drm"`      // e.g. "widevine", "fairplay", "sample-aes", or ""
    Protocol string `json:"protocol"` // e.g. "dash", "hls"
}

// GetOfferingsTask implements the MCP task definition for content.get_offerings.
type GetOfferingsTask struct{}

// NewGetOfferingsTask constructs a new GetOfferingsTask.
func NewGetOfferingsTask() *GetOfferingsTask {
    return &GetOfferingsTask{}
}

// Name returns the MCP tool name.
func (GetOfferingsTask) Name() string {
    return "content.get_offerings"
}

// Description returns a human-readable description of the tool.
func (GetOfferingsTask) Description() string {
    var b strings.Builder
    b.WriteString("Retrieve and summarize the available offerings for a content object from Fabric.\n\n")
    b.WriteString("Required parameters:\n")
    b.WriteString("- content_id (string): A content hash, ID, or write token.\n")
    b.WriteString("  - The hash of a finalized content object.\n")
    b.WriteString("  - A content ID (resolved to the latest version).\n")
    b.WriteString("  - The write token of a draft content object.\n\n")
    b.WriteString("Behavior:\n")
    b.WriteString("- Resolves the library ID (qlibid) for the given content_id using the Fabric metadata API (via auth).\n")
    b.WriteString("- Uses content_id as the qhit.\n")
    b.WriteString("- Issues a GET request to /q/{content_id}/struct/meta/offerings.\n")
    b.WriteString("- Parses the 'offerings' object and returns a compact summary per offering.\n\n")
    b.WriteString("Each offering summary includes:\n")
    b.WriteString("- duration_seconds and duration_human\n")
    b.WriteString("- primary video track and all video tracks\n")
    b.WriteString("- all audio tracks\n")
    b.WriteString("- all subtitle tracks (label, language, codec, forced, hearing_impaired, default_for_media_type)\n")
    b.WriteString("- DRM info (optional flag and list of schemes)\n")
    b.WriteString("- playout formats (name, DRM type, protocol)\n")
    b.WriteString("- readiness flag\n\n")
    b.WriteString("Rules:\n")
    b.WriteString("- Fails with an Invalid error if content_id is missing or empty.\n")
    b.WriteString("- Fails with a Permission error if the tenant is missing from context.\n")
    b.WriteString("- Fails with an Unavailable error if authorization or Fabric API calls fail.\n")
    b.WriteString("- Ignores mezzanine-internal details (mez_prep_specs, billing_items, part_hashes, xc, etc.).\n")
    return b.String()
}

// Register registers the content.get_offerings tool with the MCP runtime.
func (GetOfferingsTask) Register(server *mcp.Server, cfg *config.Config) {
    mcp.AddTool(
        server,
        &mcp.Tool{
            Name:        GetOfferingsTask{}.Name(),
            Description: GetOfferingsTask{}.Description(),
        },
        func(
            ctx context.Context,
            req *mcp.CallToolRequest,
            args GetOfferingsArgs,
        ) (*mcp.CallToolResult, any, error) {
            return GetOfferingsWorker(ctx, req, args, cfg)
        },
    )
}
