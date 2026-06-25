package fabric

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	elog "github.com/eluv-io/log-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
	"github.com/qluvio/elv-mcp/tasks"
)

//
// ────────────────────────────────────────────────────────────────
//   Slim output structures (LLM‑friendly, minimal, deterministic)
// ────────────────────────────────────────────────────────────────
//

type SlimClip struct {
    QID          string `json:"id"`
    DisplayTitle string `json:"title"`
    ReleaseDate  string `json:"release_date"`
    Start        int64  `json:"start_ms"`
    End          int64  `json:"end_ms"`
    VideoURL     string `json:"video_url"`
    ImageURL     string `json:"image_url"`
    Score        string `json:"score"`
}

type SlimResponse struct {
    Summary    string     `json:"summary"`
    Confidence float64    `json:"confidence"`
    Clips      []SlimClip `json:"clips"`
}

//
// ────────────────────────────────────────────────────────────────
//   Slimming helpers
// ────────────────────────────────────────────────────────────────
//

func slimResult(result *tasks.ClipResponse) []SlimClip {
    out := make([]SlimClip, 0, len(result.Contents))

    for _, c := range result.Contents {
        out = append(out, SlimClip{
            QID:          c.QID,
            DisplayTitle: c.DisplayTitle,
            ReleaseDate:  c.ReleaseDate,
            Start:        c.StartTime,
            End:          c.EndTime,
            VideoURL:     c.VideoURL,
            ImageURL:     c.ImageURL,
            Score:        c.Score,
        })
    }

    return out
}

func buildSummary(result *tasks.ClipResponse, terms string) string {
    count := len(result.Contents)
    if count == 0 {
        return fmt.Sprintf("No clips found for %q.", terms)
    }

    first := result.Contents[0]
    return fmt.Sprintf(
        "%d clips found for %q. Top score %s.",
        count,
        terms,
        first.Score,
    )
}

func computeConfidence(result *tasks.ClipResponse) float64 {
    if len(result.Contents) == 0 {
        return 0.0
    }
    score, _ := strconv.ParseFloat(result.Contents[0].Score, 64)
    return score
}

//
// ────────────────────────────────────────────────────────────────
//   Main Search Worker
// ────────────────────────────────────────────────────────────────
//

func SearchWorker(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args SearchClipsArgs,
    cfg *config.Config,
) (*mcp.CallToolResult, *tasks.ClipResponse, error) {

    if strings.TrimSpace(args.Terms) == "" {
        return runtime.ToolError("Invalid request: search terms must not be empty", nil)
    }

    tf, ok := runtime.TenantFromContext(ctx)
    if !ok {
        return runtime.ToolError("no tenant configuration found for this user", nil)
    }

    // State token for search
    sTok, err := auth.Auth.FetchStateChannel(cfg, tf)
    if err != nil {
        return runtime.ToolError("Failed to fetch state token", err)
    }

    searchURL, err := BuildSearchURL(cfg, tf, args, sTok)
    if err != nil {
        return runtime.ToolError("Failed to build search URL (likely configuration or arguments issue)", err)
    }

    runtime.Log.Info("search URL built", "url", searchURL)

    body, httpResp, err := runtime.HTTPGet(ctx, searchURL, map[string]string{
        "Accept": "application/json",
    })

    var httpStatus string
    var statusCode int
    if httpResp != nil {
        httpStatus = httpResp.Status
        statusCode = httpResp.StatusCode
    }
    runtime.Log.Info("search API response", "http_status", httpStatus, "error", err)

    if err != nil {
        switch {
        case errors.Is(err, context.Canceled):
            return runtime.ToolError("Search aborted: the request was cancelled", err)
        case errors.Is(err, context.DeadlineExceeded):
            return runtime.ToolError("Search timed out while waiting for the Eluvio Search API", err)
        default:
            detail := ""
            if statusCode != 0 {
                detail = fmt.Sprintf(" (HTTP %d)", statusCode)
            }
            return runtime.ToolError(fmt.Sprintf("Search failed%s while calling Eluvio Search API", detail), err)
        }
    }

    if len(strings.TrimSpace(string(body))) == 0 {
        empty := &tasks.ClipResponse{}
        return buildSearchResultResponse(args, empty)
    }

    runtime.Log.Debug("raw search API response", "body", string(body))

    var result tasks.ClipResponse
    if err := json.Unmarshal(body, &result); err != nil {
        return runtime.ToolError("Search failed: could not decode Eluvio Search API response", err)
    }

    // Extract raw_score → Score
    var raw map[string]interface{}
    if err := json.Unmarshal(body, &raw); err != nil {
        return runtime.ToolError("Search failed: could not decode raw JSON", err)
    }

    contentsRaw, _ := raw["contents"].([]interface{})

    for i := range result.Contents {
        itemRaw, _ := contentsRaw[i].(map[string]interface{})

        sources, _ := itemRaw["sources"].([]interface{})
        if len(sources) == 0 {
            continue
        }

        firstSource, _ := sources[0].(map[string]interface{})
        rawScore := firstSource["raw_score"].(float64)
        result.Contents[i].Score = fmt.Sprintf("%.3f", rawScore)
    }

    // Enrich clips
    showThumbs := config.BoolOrDefault(args.Thumbnails, true)

    for i := range result.Contents {
        if err := enrichClip(&result.Contents[i], cfg, tf, showThumbs); err != nil {
			return runtime.ToolError("Failed to enrich clip data", err)
		}
    }

    return buildSearchResultResponse(args, &result)
}

//
// ────────────────────────────────────────────────────────────────
//   Final slim JSON output
// ────────────────────────────────────────────────────────────────
//

func BuildSearchResultResponse(args SearchClipsArgs, result *tasks.ClipResponse) (*mcp.CallToolResult, *tasks.ClipResponse, error) {
    return buildSearchResultResponse(args, result)
}

func buildSearchResultResponse(args SearchClipsArgs, result *tasks.ClipResponse) (*mcp.CallToolResult, *tasks.ClipResponse, error) {
    // Remove heavy metadata
    for i := range result.Contents {
        stripMeta(&result.Contents[i])
    }

    slim := slimResult(result)

    resp := SlimResponse{
        Summary:    buildSummary(result, args.Terms),
        Confidence: computeConfidence(result),
        Clips:      slim,
    }

    jsonBytes, err := json.Marshal(resp)
    if err != nil {
        return runtime.ToolError("Failed to encode JSON", err)
    }

    content := []mcp.Content{
        &mcp.TextContent{Text: string(jsonBytes)},
    }

    elog.Debug("Slim result", "json", string(jsonBytes))

    return &mcp.CallToolResult{Content: content}, result, nil
}

//
// ────────────────────────────────────────────────────────────────
//   Metadata helpers
// ────────────────────────────────────────────────────────────────
//

func promoteMeta(c *tasks.ClipItem) {
    if len(c.Meta) == 0 {
        return
    }

    pub, _ := c.Meta["public"].(map[string]interface{})
    if pub == nil {
        return
    }
    am, _ := pub["asset_metadata"].(map[string]interface{})
    if am == nil {
        return
    }

    if c.DisplayTitle == "" {
        c.DisplayTitle = metaString(am, "display_title")
    }
    if c.IPTitleID == "" {
        c.IPTitleID = metaString(am, "ip_title_id")
    }
    if c.ReleaseDate == "" {
        if info, ok := am["info"].(map[string]interface{}); ok {
            c.ReleaseDate = metaString(info, "release_date")
        }
    }
}

func enrichClip(
    c *tasks.ClipItem,
    cfg *config.Config,
    tf *config.TenantFabric,
    showThumbs bool,
) error {

    // Promote metadata
    promoteMeta(c)

    // Compute start/end seconds
    startStr := ""
    endStr := ""
    if c.StartTime > 0 {
        startStr = fmt.Sprintf("%.3f", float64(c.StartTime)/1000.0)
    }
    if c.EndTime > 0 {
        endStr = fmt.Sprintf("%.3f", float64(c.EndTime)/1000.0)
    }

    // Editor token
    eTok, err := auth.Auth.FetchEditorSigned(cfg, tf, c.QLibID, c.QID)
    if err != nil {
        return err
    }
    c.ESToken = eTok

    // Signed video URL
    c.VideoURL = BuildVideoURL(c.VideoURL, c.ESToken, startStr, endStr, cfg)

    // Thumbnails
    if showThumbs {
        stok, err := auth.Auth.FetchStateChannel(cfg, tf)
        if err != nil {
            return err
        }
        thumb := BuildNodeThumbURL(c.ImageURL, stok, cfg)
        if thumb != "" {
            c.ImageURL = thumb
        }
    } else {
        c.ImageURL = ""
    }

    return nil
}


func stripMeta(c *tasks.ClipItem) {
    c.Meta = nil
}

func metaString(m map[string]interface{}, key string) string {
    if v, ok := m[key].(string); ok {
        return v
    }
    return ""
}
