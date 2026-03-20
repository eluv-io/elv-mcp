package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/qluvio/elv-mcp-experiment/auth"
	"github.com/qluvio/elv-mcp-experiment/builder"
	"github.com/qluvio/elv-mcp-experiment/types"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func SearchClips(ctx context.Context, req *mcp.CallToolRequest, args types.SearchClipsArgs, cfg *types.Config) (*mcp.CallToolResult, any, error) {
	// Basic argument validation up-front
	if strings.TrimSpace(args.Terms) == "" {
		return toolError("Invalid request: search terms must not be empty", nil)
	}

	sTok, err := auth.FetchStateChannel(cfg.SCToken)
	if err != nil {
		return toolError("Failed to fetch state token", err)
	}
	cfg.SCToken = sTok

	searchURL, err := builder.BuildSearchURL(cfg, args, cfg.SCToken)
	if err != nil {
		return toolError("Failed to build search URL (likely configuration or arguments issue)", err)
	}

	log.Printf("search URL: %s", searchURL)

	result, httpResp, err := searchClips(ctx, searchURL, "")
	log.Printf("search result: %v", result)

	var httpStatus string
	var statusCode int
	if httpResp != nil {
		httpStatus = httpResp.Status
		statusCode = httpResp.StatusCode
	}
	log.Printf("search API response: httpStatus=%q err=%v", httpStatus, err)

	// Handle common context-related errors separately so they show up clearly to the user
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return toolError("Search aborted: the request was cancelled", err)
		case errors.Is(err, context.DeadlineExceeded):
			return toolError("Search timed out while waiting for the Eluvio Search API", err)
		default:
			detail := ""
			if statusCode != 0 {
				detail = fmt.Sprintf(" (HTTP %d)", statusCode)
			}
			return toolError(fmt.Sprintf("Search failed%s while calling Eluvio Search API", detail), err)
		}
	}

	// Safety check: result should not be nil if err == nil, but guard anyway
	if result == nil {
		return toolError("Search failed: received an empty response from Eluvio Search API", nil)
	}

	showThumbs := types.BoolOrDefault(args.Thumbnails, true)

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d clip(s) for %q\n\n", len(result.Contents), args.Terms)

	for i, c := range result.Contents {
		// Compute start/end seconds from ms (fallback to empty if not provided)
		startStr := ""
		endStr := ""
		if c.StartTime > 0 {
			startStr = fmt.Sprintf("%.3f", float64(c.StartTime)/1000.0)
		}
		if c.EndTime > 0 {
			endStr = fmt.Sprintf("%.3f", float64(c.EndTime)/1000.0)
		}

		eTok, err := auth.FetchEditorSigned(cfg, c.QLibID, c.QID)
		if err != nil {
			return toolError("Failed to fetch editor token", err)
		}
		c.ESToken = eTok

		// Build the clip URL now that we guarantee `editorToken` is valid
		video := builder.BuildVideoURL(c.VideoURL, c.ESToken, startStr, endStr, cfg)
		result.Contents[i].VideoURL = video

		var thumb string
		if showThumbs {
			stok, err := auth.FetchStateChannel(cfg.SCToken)
			if err != nil {
				return toolError("Failed to fetch state token", err)
			}
			cfg.SCToken = stok

			// Now safely build thumbnail URL
			thumb := builder.BuildNodeThumbURL(c.ImageURL, cfg.SCToken, cfg)
			if thumb != "" {
				result.Contents[i].ImageURL = thumb
			}
		}

		// Structured, minimal text summary
		fmt.Fprintf(&sb, "%d) %s → %s\n", i+1, c.Start, c.End)
		fmt.Fprintf(&sb, "   clip: %s\n", video)
		if showThumbs && thumb != "" {
			fmt.Fprintf(&sb, "   thumbnail: %s\n", thumb)
		}
		sb.WriteString("\n")
	}

	combined := types.ClipResponse{
		Description: "When the tool returns clip results, the assistant must output every clip exactly as provided." +
			"Each clip must include an inline thumbnail displayed as a clickable link." +
			"The assistant must use the exact format: \n [![MovieTitle or description](THUMBNAIL_URL)](VIDEO_URL)." +
			"The title is a short descriptive label created by the assistant." +
			"The thumbnail URL must be the image_url from the tool result," +
			"and the video URL must be the url from the tool result. No raw URLs may appear anywhere in the output." +
			"No thumbnail may be omitted.\nEach clip must be shown in its own separate block." +
			"The block must contain a clip number, the clickable thumbnail, and the start and end times." +
			"Only one clickable thumbnail may appear per paragraph" +
			"and each clip block must be separated by a blank line to ensure rendering stability." +
			"The assistant must output clips in the exact order returned by the tool" +
			"and must not skip or reorder any of them. The assistant must not modify, shorten, rewrite," +
			"or hide the URLs." +
			"They must be used exactly as returned, including authorization tokens." +
			"The assistant must not provide commentary, explanations," +
			"or apologies about formatting, URL length, or rendering behavior." +
			"Only the required clip blocks should be produced." +
			"If a thumbnail fails to render," +
			"the assistant must automatically resend that specific clip block without rerunning the tool." +
			"The assistant must not alter any other clips when doing so.",
		Contents: result.Contents,
	}

	// Try to pretty-print the JSON, but don't fail the whole tool if this breaks
	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Printf("failed to marshal search result as JSON: %v", err)
		// Fallback: just return the human-readable summary.
		content := []mcp.Content{
			&mcp.TextContent{Text: sb.String()},
			&mcp.TextContent{Text: "Raw JSON: <unavailable due to internal JSON encoding error>"},
		}
		return &mcp.CallToolResult{Content: content}, result, nil
	}

	content := []mcp.Content{
		&mcp.TextContent{Text: sb.String()},
		&mcp.TextContent{Text: "Raw JSON:\n" + string(prettyJSON)},
	}
	return &mcp.CallToolResult{Content: content}, combined, nil
}

func replaceQueryToken(rawURL, key, newValue string) (string, bool) {
	if rawURL == "" || key == "" {
		return rawURL, false
	}

	needle := key + "="
	i := strings.Index(rawURL, needle)
	if i < 0 {
		return rawURL, false
	}

	// Start of the token value
	start := i + len(needle)

	// End is next '&' or end-of-string
	end := strings.Index(rawURL[start:], "&")
	if end < 0 {
		// param goes to end
		return rawURL[:start] + newValue, true
	}
	end = start + end

	return rawURL[:start] + newValue + rawURL[end:], true
}

func RefreshToken(ctx context.Context, req *mcp.CallToolRequest, args types.RefreshClipsArgs, cfg *types.Config) (*mcp.CallToolResult, any, error) {

	if len(args.Contents) == 0 {
		return toolError("Invalid request: contents must not be empty", nil)
	}

	first := args.Contents[0]

	// Create State token
	newStateToken, err := auth.FetchStateChannel("")
	if err != nil {
		return toolError("Failed to fetch state token", err)
	}

	// Create Editor token
	cfg.ESToken = "" // force fresh
	newEditorToken, err := auth.FetchEditorSigned(cfg, first.QLibID, first.QID)
	if err != nil {
		return toolError("Failed to fetch editor token", err)
	}

	cfg.SCToken = newStateToken
	cfg.ESToken = newEditorToken

	// Small summary
	var sb strings.Builder
	fmt.Fprintf(&sb, "Refreshed %d clip(s)\n\n", len(args.Contents))

	for i := range args.Contents {
		c := &args.Contents[i]

		// Replace token values in the URLs
		args.Contents[i].VideoURL, _ = replaceQueryToken(c.VideoURL, "ath", newEditorToken)
		args.Contents[i].ImageURL, _ = replaceQueryToken(c.ImageURL, "authorization", newStateToken)

		fmt.Fprintf(&sb, "%d) %s → %s\n", i+1, c.Start, c.End)
		fmt.Fprintf(&sb, "   clip: %s\n", args.Contents[i].VideoURL)
		fmt.Fprintf(&sb, "   thumbnail: %s\n\n", args.Contents[i].ImageURL)
	}

	combined := types.ClipResponse{
		Description: "CRITICAL: Treat `url` and `image_url` as opaque signed strings. Copy them byte-for-byte exactly as provided (including every query parameter like `authorization=` / `ath=`). Do NOT trim, shorten, reformat, re-encode, or remove anything; any change breaks the URL. " +
			"When the tool returns clip results, the assistant must output every clip exactly as provided." +
			"Each clip must include an inline thumbnail displayed as a clickable link." +
			"The assistant must use the exact format: [![MovieTitle or description](THUMBNAIL_URL)](VIDEO_URL)." +
			"The title is a short descriptive label created by the assistant." +
			"The thumbnail URL must be the image_url from the tool result," +
			"and the video URL must be the url from the tool result. No raw URLs may appear anywhere in the output." +
			"No thumbnail may be omitted. Each clip must be shown in its own separate block." +
			"The block must contain a clip number, the clickable thumbnail, and the start and end times." +
			"Only one clickable thumbnail may appear per paragraph and each clip block must be separated by a blank line to ensure rendering stability." +
			"The assistant must output clips in the exact order returned by the tool and must not skip or reorder any of them." +
			"The assistant must not modify, shorten, rewrite, or hide the URLs. They must be used exactly as returned, including authorization tokens." +
			"The assistant must not provide commentary, explanations, or apologies about formatting, URL length, or rendering behavior." +
			"Only the required clip blocks should be produced. If a thumbnail fails to render, the assistant must automatically resend that specific clip block without rerunning the tool." +
			"The assistant must not alter any other clips when doing so.",
		Contents: args.Contents,
	}

	prettyJSON, err := json.MarshalIndent(combined, "", "  ")
	if err != nil {
		log.Printf("failed to marshal refresh result as JSON: %v", err)
		content := []mcp.Content{
			&mcp.TextContent{Text: sb.String()},
			&mcp.TextContent{Text: "Raw JSON: <unavailable due to internal JSON encoding error>"},
		}
		return &mcp.CallToolResult{Content: content}, combined, nil
	}

	content := []mcp.Content{
		&mcp.TextContent{Text: sb.String()},
		&mcp.TextContent{Text: "Raw JSON:\n" + string(prettyJSON)},
	}

	return &mcp.CallToolResult{Content: content}, combined, nil
}
