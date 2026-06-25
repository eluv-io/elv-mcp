package fabric

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
	"github.com/qluvio/elv-mcp/tasks"
)

// RefreshURLWorker is the entrypoint for the refresh_clips MCP tool.
func RefreshURLWorker(
	ctx context.Context, req *mcp.CallToolRequest, args RefreshClipsArgs, cfg *config.Config,
) (*mcp.CallToolResult, *tasks.ClipResponse, error) {

	if len(args.Contents) == 0 {
		return runtime.ToolError("Invalid request: contents must not be empty", nil)
	}

	tf, ok := runtime.TenantFromContext(ctx)
	if !ok {
		return runtime.ToolError("no tenant configuration found for this user", nil)
	}

	first := args.Contents[0]

	tf.Mu.Lock()
	tf.SCToken = ""
	tf.ESToken = ""
	tf.Mu.Unlock()

	// NOTE: in real-life this uses Auth that is defined in auth_fabric.go
	// in tests Auth can be replaced with a Mock
	newStateToken, err := auth.Auth.FetchStateChannel(cfg, tf)
	if err != nil {
		return runtime.ToolError("Failed to fetch state token", err)
	}

	// NOTE: in real-life this uses Auth that is defined in auth_fabric.go
	// in tests Auth can be replaced with a Mock
	newEditorToken, err := auth.Auth.FetchEditorSigned(cfg, tf, first.QLibID, first.QID)
	if err != nil {
		return runtime.ToolError("Failed to fetch editor token", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Refreshed %d clip(s)\n\n", len(args.Contents))

	for i := range args.Contents {
		c := &args.Contents[i]

		args.Contents[i].VideoURL, _ = ReplaceQueryToken(c.VideoURL, "ath", newEditorToken)
		args.Contents[i].ImageURL, _ = ReplaceQueryToken(c.ImageURL, "authorization", newStateToken)

		fmt.Fprintf(&sb, "%d) %s → %s\n", i+1, c.Start, c.End)
		fmt.Fprintf(&sb, "   clip: %s\n", args.Contents[i].VideoURL)
		fmt.Fprintf(&sb, "   thumbnail: %s\n\n", args.Contents[i].ImageURL)
	}

	combined := &tasks.ClipResponse{
		Description: argsDescription,
		Contents:    args.Contents,
	}

	prettyJSON, err := json.MarshalIndent(combined, "", "  ")
	if err != nil {
		runtime.Log.Warn("failed to marshal refresh result as JSON", "error", err)
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

const argsDescription = "CRITICAL: Treat `url` and `image_url` as opaque signed strings. Copy them byte-for-byte exactly as provided (including every query parameter like `authorization=` / `ath=`). Do NOT trim, shorten, reformat, re-encode, or remove anything; any change breaks the URL. " +
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
	"The assistant must not alter any other clips when doing so."

// ReplaceQueryToken replaces the value of a query parameter in a URL.
func ReplaceQueryToken(rawURL, key, newValue string) (string, bool) {
	if rawURL == "" || key == "" {
		return rawURL, false
	}

	// Make sure to look for both "?key=" and "&key=" to find the parameter in the URL
	// and not just key-like substrings in the path or other query parameters. This ensures we only replace the intended parameter.
	needle := "?" + key + "="
	i := strings.Index(rawURL, needle)
	if i < 0 {
		needle = "&" + key + "="
		i = strings.Index(rawURL, needle)
		if i < 0 {
			return rawURL, false
		}
	}

	start := i + len(needle)

	end := strings.Index(rawURL[start:], "&")
	if end < 0 {
		return rawURL[:start] + newValue, true
	}
	end = start + end

	return rawURL[:start] + newValue + rawURL[end:], true
}
