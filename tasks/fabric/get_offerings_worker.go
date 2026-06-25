package fabric

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/eluv-io/errors-go"
	elog "github.com/eluv-io/log-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
)

// BuildOfferingsURL constructs the Fabric struct API URL for retrieving offerings metadata.
//
// Example output:
//
//	{ApiUrl}/q/{contentID}/struct/meta/user/offerings
//
// This mirrors the style and conventions of BuildPublicMetaURL.
func BuildOfferingsURL(cfg *config.Config, contentID, token string) (string, error) {
	if strings.TrimSpace(contentID) == "" {
		return "", fmt.Errorf("content_id cannot be empty")
	}

	base := strings.TrimRight(cfg.ApiUrl, "/")

	u, err := url.Parse(fmt.Sprintf(
		"%s/q/%s/struct/meta/user/offerings",
		base,
		url.PathEscape(contentID),
	))
	if err != nil {
		return "", err
	}

	// Optional query parameters.
	q := u.Query()

	// Some gateways accept token via query as well as header.
	if strings.TrimSpace(token) != "" {
		q.Set("authorization", token)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// GetOfferingsWorker performs the HTTP call to Fabric to retrieve offerings metadata
// and summarizes it into a compact, user-facing structure.
func GetOfferingsWorker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args GetOfferingsArgs,
	cfg *config.Config,
) (*mcp.CallToolResult, any, error) {
	log := elog.Get("/fabric/get_offerings")

	contentID := strings.TrimSpace(args.ContentID)
	if contentID == "" {
		return runtime.MCPError(errors.E("content.get_offerings", errors.K.Invalid,
			"reason", "missing required field 'content_id'"))
	}

	tf, ok := runtime.TenantFromContext(ctx)
	if !ok {
		return runtime.MCPError(
			errors.E("content.get_offerings", errors.K.Permission,
				"reason", "no tenant configuration found for this user"),
		)
	}

	token, err := auth.Auth.FetchEditorSigned(cfg, tf, "", contentID)
	if err != nil {
		log.Error("failed to fetch editor-signed token", "error", err)
		return runtime.MCPError(errors.E("content.get_offerings", errors.K.Unavailable,
			"reason", "failed to fetch editor-signed token", "error", err))
	}

	url, err := BuildOfferingsURL(cfg, contentID, token)
	if err != nil {
		log.Error("failed to build offerings URL", "error", err)
		return runtime.MCPError(errors.E("content.get_offerings", errors.K.Unavailable,
			"reason", "failed to build offerings URL", "error", err))
	}

	log.Debug("requesting offerings metadata", "url", url, "content_id", contentID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error("failed to create HTTP request", "error", err)
		return runtime.MCPError(errors.E("content.get_offerings", errors.K.Unavailable,
			"reason", "failed to create HTTP request", "error", err))
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Error("HTTP request failed", "error", err)
		return runtime.MCPError(errors.E("content.get_offerings", errors.K.Unavailable,
			"reason", "HTTP request failed", "error", err))
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var raw map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			log.Error("failed to decode offerings response JSON", "error", err)
			return runtime.MCPError(errors.E("content.get_offerings", errors.K.Unavailable,
				"reason", "failed to decode response JSON", "error", err))
		}

		log.Debug("decoded offerings response", "keys", len(raw))

		summary, err := SummarizeOfferings(raw)
		if err != nil {
			log.Error("failed to summarize offerings", "error", err)
			return runtime.MCPError(errors.E("content.get_offerings", errors.K.Unavailable,
				"reason", "failed to summarize offerings", "error", err))
		}

		res := &mcp.CallToolResult{
			IsError: false,
		}
		return res, GetOfferingsResult{Offerings: summary}, nil

	case http.StatusBadRequest:
		return runtime.MCPError(errors.E("content.get_offerings", errors.K.Invalid,
			"reason", "bad request to Fabric struct API", "status", resp.StatusCode))

	case http.StatusUnauthorized, http.StatusForbidden:
		return runtime.MCPError(errors.E("content.get_offerings", errors.K.Permission,
			"reason", "unauthorized to access Fabric struct API", "status", resp.StatusCode))

	case http.StatusNotFound, http.StatusConflict:
		return runtime.MCPError(errors.E("content.get_offerings", errors.K.Exist,
			"reason", "content object or struct path not found", "status", resp.StatusCode))

	default:
		return runtime.MCPError(errors.E("content.get_offerings", errors.K.Unavailable,
			"reason", "unexpected Fabric response status", "status", resp.StatusCode))
	}
}

// HELPER FUNCTIONS

// SummarizeOfferings extracts and summarizes all offerings from the raw Fabric response.
//
// The raw response is expected to contain a list of offering objects whose keys are offering
// names (e.g. "default", "hd") and whose values are offering metadata objects.
func SummarizeOfferings(raw map[string]any) (map[string]OfferingSummary, error) {
	out := make(map[string]OfferingSummary)

	offeringsMap := raw

	elog.Debug("summarizing offerings", "count", len(offeringsMap))

	for name, v := range offeringsMap {
		offObj, ok := v.(map[string]any)
		if !ok {
			elog.Warn("skipping offering with non-object value", "name", name)
			continue
		}

		summary, err := summarizeSingleOffering(name, offObj)
		if err != nil {
			elog.Warn("failed to summarize offering", "name", name, "error", err)
			continue
		}

		out[name] = summary
	}

	return out, nil
}

// summarizeSingleOffering isolates the logic that decides which fields to expose
// in the result for a single offering. This is intentionally centralized and
// documented so that future changes (e.g. adding more fields) are easy and safe.
func summarizeSingleOffering(name string, off map[string]any) (OfferingSummary, error) {
	elog.Debug("summarizing single offering", "name", name)

	mediaStruct := getMap(off, "media_struct")

	durationSeconds := extractDurationSeconds(mediaStruct)
	durationHuman := formatDurationHuman(durationSeconds)

	videoTracks := extractVideoTracks(mediaStruct)
	primaryVideo := selectPrimaryVideoTrack(videoTracks)

	audioTracks := extractAudioTracks(mediaStruct)
	subtitleTracks := extractSubtitleTracks(mediaStruct)

	drm := extractDRMInfo(off)
	playoutFormats := extractPlayoutFormats(off)

	ready := getBool(off, "ready")

	summary := OfferingSummary{
		DurationSeconds: durationSeconds,
		DurationHuman:   durationHuman,
		Video:           primaryVideo,
		VideoTracks:     videoTracks,
		AudioTracks:     audioTracks,
		SubtitleTracks:  subtitleTracks,
		DRM:             drm,
		PlayoutFormats:  playoutFormats,
		Ready:           ready,
	}

	elog.Debug("summarized offering",
		"name", name,
		"duration_seconds", durationSeconds,
		"video_tracks", len(videoTracks),
		"audio_tracks", len(audioTracks),
		"subtitle_tracks", len(subtitleTracks),
		"drm_schemes", len(drm.Schemes),
		"playout_formats", len(playoutFormats),
		"ready", ready,
	)

	return summary, nil
}

// extractDurationSeconds attempts to derive a float duration in seconds from media_struct.
//
// Priority:
// 1. media_struct.duration_rat.float
// 2. media_struct.streams.video.duration.float (or first video stream)
// 3. 0.0 if nothing usable is found.
func extractDurationSeconds(mediaStruct map[string]any) float64 {
	if mediaStruct == nil {
		elog.Warn("media_struct missing; duration will be 0")
		return 0
	}

	// 1) Try media_struct.duration_rat.float
	if dur := getMap(mediaStruct, "duration_rat"); dur != nil {
		if f, ok := getFloat(dur, "float"); ok && f > 0 {
			return f
		}
	}

	// 2) Try first video stream duration.float
	streams := getMap(mediaStruct, "streams")

	for _, v := range streams {
		stream, ok := v.(map[string]any)
		if !ok {
			continue
		}
		if codecType, _ := getString(stream, "codec_type"); codecType == "video" {
			if dur := getMap(stream, "duration"); dur != nil {
				if f, ok := getFloat(dur, "float"); ok && f > 0 {
					return f
				}
			}
			// Only need the first video stream for duration.
			break
		}
	}

	elog.Warn("could not determine duration; defaulting to 0")
	return 0
}

// formatDurationHuman converts seconds into "HH:MM:SS.mmm" format.
//
// This is intentionally isolated so that future changes to formatting are localized.
func formatDurationHuman(seconds float64) string {
	if seconds <= 0 {
		return "00:00:00.000"
	}

	msTotal := int64(math.Round(seconds * 1000))
	d := time.Duration(msTotal) * time.Millisecond

	h := int64(d.Hours())
	m := int64(d.Minutes()) % 60
	s := int64(d.Seconds()) % 60
	ms := msTotal % 1000

	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

// extractVideoTracks returns all video tracks found in media_struct.streams.
//
// NOTE: This function is intentionally minimal but structured so that additional
// fields can be added later without touching the worker.
func extractVideoTracks(mediaStruct map[string]any) []VideoTrackSummary {
	var tracks []VideoTrackSummary

	if mediaStruct == nil {
		return tracks
	}

	streams := getMap(mediaStruct, "streams")
	if streams == nil {
		return tracks
	}

	for _, v := range streams {
		stream, ok := v.(map[string]any)
		if !ok {
			continue
		}
		codecType, _ := getString(stream, "codec_type")
		if codecType != "video" {
			continue
		}

		codec, _ := getString(stream, "codec_name")
		width := getInt(stream, "width")
		height := getInt(stream, "height")
		frameRate, _ := getString(stream, "rate")
		aspectRatio, _ := getString(stream, "aspect_ratio")
		hdr := stream["hdr"]
		bitrate := getInt64(stream, "bit_rate")
		def := getBool(stream, "default_for_media_type")

		tracks = append(tracks, VideoTrackSummary{
			Codec:               codec,
			Width:               width,
			Height:              height,
			FrameRate:           frameRate,
			AspectRatio:         aspectRatio,
			HDR:                 hdr,
			Bitrate:             bitrate,
			DefaultForMediaType: def,
		})
	}

	elog.Debug("extracted video tracks", "count", len(tracks))
	return tracks
}

// selectPrimaryVideoTrack chooses a single primary video track using a deterministic
// priority order:
//
// 1. default_for_media_type == true
// 2. highest resolution (width * height)
// 3. highest bitrate
// 4. first in slice order (as a final tie-breaker)
//
// If no tracks are present, returns nil.
func selectPrimaryVideoTrack(tracks []VideoTrackSummary) *VideoSummary {
	if len(tracks) == 0 {
		elog.Warn("no video tracks found; primary video will be nil")
		return nil
	}

	// Copy indices for deterministic selection.
	indices := make([]int, len(tracks))
	for i := range tracks {
		indices[i] = i
	}

	sort.SliceStable(indices, func(i, j int) bool {
		a := tracks[indices[i]]
		b := tracks[indices[j]]

		// 1) default_for_media_type
		if a.DefaultForMediaType != b.DefaultForMediaType {
			return a.DefaultForMediaType && !b.DefaultForMediaType
		}

		// 2) resolution
		resA := a.Width * a.Height
		resB := b.Width * b.Height
		if resA != resB {
			return resA > resB
		}

		// 3) bitrate
		if a.Bitrate != b.Bitrate {
			return a.Bitrate > b.Bitrate
		}

		// 4) stable by original order (indices already preserve that).
		return indices[i] < indices[j]
	})

	primary := tracks[indices[0]]
	elog.Debug("selected primary video track",
		"codec", primary.Codec,
		"width", primary.Width,
		"height", primary.Height,
		"bitrate", primary.Bitrate,
		"default_for_media_type", primary.DefaultForMediaType,
	)

	return &VideoSummary{
		Codec:               primary.Codec,
		Width:               primary.Width,
		Height:              primary.Height,
		FrameRate:           primary.FrameRate,
		AspectRatio:         primary.AspectRatio,
		HDR:                 primary.HDR,
		Bitrate:             primary.Bitrate,
		DefaultForMediaType: primary.DefaultForMediaType,
	}
}

// extractAudioTracks returns all audio tracks found in media_struct.streams.
func extractAudioTracks(mediaStruct map[string]any) []AudioTrackSummary {
	var tracks []AudioTrackSummary

	if mediaStruct == nil {
		return tracks
	}

	streams := getMap(mediaStruct, "streams")
	if streams == nil {
		return tracks
	}

	for _, v := range streams {
		stream, ok := v.(map[string]any)
		if !ok {
			continue
		}
		codecType, _ := getString(stream, "codec_type")
		if codecType != "audio" {
			continue
		}

		label, _ := getString(stream, "label")
		lang, _ := getString(stream, "language")
		channels := getInt(stream, "channels")
		layout, _ := getString(stream, "channel_layout")
		codec, _ := getString(stream, "codec_name")
		bitrate := getInt64(stream, "bit_rate")
		def := getBool(stream, "default_for_media_type")

		tracks = append(tracks, AudioTrackSummary{
			Label:               label,
			Language:            lang,
			Channels:            channels,
			Layout:              layout,
			Codec:               codec,
			Bitrate:             bitrate,
			DefaultForMediaType: def,
		})
	}

	elog.Debug("extracted audio tracks", "count", len(tracks))
	return tracks
}

// extractSubtitleTracks returns all subtitle tracks found in media_struct.streams.
//
// NOTE: This is intentionally minimal but clearly structured so that additional
// fields (e.g. bitrate, duration) can be added later if we decide to move toward
// a richer representation (Option C).
func extractSubtitleTracks(mediaStruct map[string]any) []SubtitleTrackSummary {
	var tracks []SubtitleTrackSummary

	if mediaStruct == nil {
		return tracks
	}

	streams := getMap(mediaStruct, "streams")
	if streams == nil {
		return tracks
	}

	for _, v := range streams {
		stream, ok := v.(map[string]any)
		if !ok {
			continue
		}
		codecType, _ := getString(stream, "codec_type")
		if codecType != "subtitle" {
			continue
		}

		label, _ := getString(stream, "label")
		lang, _ := getString(stream, "language")
		codec, _ := getString(stream, "codec_name")

		var forcedPtr *bool
		if forced, ok := getBoolOk(stream, "forced"); ok {
			forcedPtr = &forced
		}

		var hiPtr *bool
		if hi, ok := getBoolOk(stream, "hearing_impaired"); ok {
			hiPtr = &hi
		}

		def := getBool(stream, "default_for_media_type")

		tracks = append(tracks, SubtitleTrackSummary{
			Label:               label,
			Language:            lang,
			Codec:               codec,
			Forced:              forcedPtr,
			HearingImpaired:     hiPtr,
			DefaultForMediaType: def,
		})
	}

	elog.Debug("extracted subtitle tracks", "count", len(tracks))
	return tracks
}

// extractDRMInfo summarizes DRM configuration for an offering.
//
// It merges schemes from playout.drm_keys and streams.*.encryption_schemes.
func extractDRMInfo(off map[string]any) DRMInfo {
	drmOptional := getBool(off, "drm_optional")

	schemesSet := make(map[string]struct{})

	// From playout.drm_keys
	playout := getMap(off, "playout")
	if playout != nil {
		drmKeys := getMap(playout, "drm_keys")
		for scheme := range drmKeys {
			schemesSet[scheme] = struct{}{}
		}
	}

	// From streams.*.encryption_schemes
	if playout != nil {
		streams := getMap(playout, "streams")
		for _, v := range streams {
			stream, ok := v.(map[string]any)
			if !ok {
				continue
			}
			encSchemes := getMap(stream, "encryption_schemes")
			for scheme := range encSchemes {
				schemesSet[scheme] = struct{}{}
			}
		}
	}

	var schemes []string
	for s := range schemesSet {
		schemes = append(schemes, s)
	}
	sort.Strings(schemes)

	elog.Debug("extracted DRM info", "optional", drmOptional, "schemes", schemes)

	return DRMInfo{
		Optional: drmOptional,
		Schemes:  schemes,
	}
}

// extractPlayoutFormats summarizes playout_formats for an offering.
func extractPlayoutFormats(off map[string]any) []PlayoutFormatSummary {
	var formats []PlayoutFormatSummary

	playout := getMap(off, "playout")
	if playout == nil {
		return formats
	}

	pf := getMap(playout, "playout_formats")
	if pf == nil {
		return formats
	}

	for name, v := range pf {
		obj, ok := v.(map[string]any)
		if !ok {
			continue
		}

		drmObj := getMap(obj, "drm")
		var drmType string
		if drmObj != nil {
			if t, ok := getString(drmObj, "type"); ok {
				switch t {
				case "DrmWidevine":
					drmType = "widevine"
				case "DrmFairplay":
					drmType = "fairplay"
				case "DrmSampleAes":
					drmType = "sample-aes"
				default:
					drmType = t
				}
			}
		}

		protoObj := getMap(obj, "protocol")
		var proto string
		if protoObj != nil {
			if t, ok := getString(protoObj, "type"); ok {
				switch t {
				case "ProtoDash":
					proto = "dash"
				case "ProtoHls":
					proto = "hls"
				default:
					proto = t
				}
			}
		}

		formats = append(formats, PlayoutFormatSummary{
			Name:     name,
			DRM:      drmType,
			Protocol: proto,
		})
	}

	elog.Debug("extracted playout formats", "count", len(formats))
	return formats
}

// --- small helpers for safe map access ---

func getMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getString(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}

func getBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getBoolOk(m map[string]any, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b, true
		}
	}
	return false, false
}

func getInt(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	if v, ok := m[key]; ok {
		switch vv := v.(type) {
		case float64:
			return int(vv)
		case int:
			return vv
		case int64:
			return int(vv)
		}
	}
	return 0
}

func getInt64(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	if v, ok := m[key]; ok {
		switch vv := v.(type) {
		case float64:
			return int64(vv)
		case int:
			return int64(vv)
		case int64:
			return vv
		}
	}
	return 0
}

func getFloat(m map[string]any, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f, true
		}
	}
	return 0, false
}
