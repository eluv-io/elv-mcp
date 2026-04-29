package fabric

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/eluv-io/errors-go"
	elog "github.com/eluv-io/log-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
)

//
// ────────────────────────────────────────────────────────────────
//   Logging
// ────────────────────────────────────────────────────────────────
//
// We keep a dedicated logger for the fabric package.
// All debug logs are structured and contextual, matching the style
// used across the rest of the MCP workers.
//
var Log = elog.Get("/fabric")

//
// ────────────────────────────────────────────────────────────────
//   Domain model
// ────────────────────────────────────────────────────────────────
//
// These structs represent the *internal* representation of image
// search results. They are intentionally minimal and stable.
// The Slim* structs are the LLM‑facing, deterministic output.
//
type ImageHit struct {
    QID        string
    Similarity float64

    FrameIndex int64
    FPS        float64
    Offering   string

    Timestamp float64
    ImageURL  string
}

type ImageSearchResult struct {
    Hits []ImageHit
}

//
// ────────────────────────────────────────────────────────────────
//   Slim output structures (LLM‑friendly, deterministic)
// ────────────────────────────────────────────────────────────────
//
type SlimImageItem struct {
    QID        string  `json:"id"`
    Similarity float64 `json:"similarity"`

    FrameIndex int64   `json:"frame_index"`
    FPS        float64 `json:"fps"`
    Offering   string  `json:"offering"`

    Timestamp float64 `json:"timestamp"`
    ImageURL  string  `json:"image_url"`
}

type SlimImageResponse struct {
    Summary    string          `json:"summary"`
    Confidence float64         `json:"confidence"`
    Items      []SlimImageItem `json:"items"`
}

//
// ────────────────────────────────────────────────────────────────
//   Main SearchImages Worker
// ────────────────────────────────────────────────────────────────
//
// This worker mirrors the structure of SearchWorker (clips):
//   • Validate arguments
//   • Resolve tenant + collection
//   • Fetch state token
//   • Dispatch to text or image search
//   • Normalize + slim results
//   • Return MCP‑friendly JSON
//
// All error paths use structured errors via errors.E and return
// MCPError for consistency with the rest of the codebase.
//
func SearchImagesWorker(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args SearchImagesArgs,
    cfg *config.Config,
) (*mcp.CallToolResult, any, error) {

    Log.Debug("SearchImagesWorker invoked",
        "query", args.Query,
        "image_path", args.ImagePath,
        "collection_id", args.CollectionID,
    )

    // Input validation: exactly one of query or image_path must be provided.
    if strings.TrimSpace(args.Query) == "" && strings.TrimSpace(args.ImagePath) == "" {
        return runtime.MCPError(
            errors.E("search_images", errors.K.Invalid, "reason", "either query or image_path must be provided"),
        )
    }

    // Tenant resolution is mandatory for all fabric operations.
    tf, ok := runtime.TenantFromContext(ctx)
    if !ok {
        return runtime.MCPError(
            errors.E("search_images", errors.K.Internal, "reason", "no tenant configuration found for this user"),
        )
    }

    collectionID, err := resolveCollectionID(args, tf)
    if err != nil {
        return runtime.MCPError(
            errors.E("search_images", errors.K.Invalid, "reason", err.Error()),
        )
    }

    Log.Debug("Resolved collection ID", "collection_id", collectionID)

    // State token is required for all search-ng calls.
    sTok, err := auth.Auth.FetchStateChannel(cfg, tf)
    if err != nil {
        return runtime.MCPError(
            errors.E("search_images", errors.K.Internal, "reason", "failed to fetch state token"),
        )
    }

    var result *ImageSearchResult

    // Dispatch to the appropriate search mode.
    switch {
    case strings.TrimSpace(args.ImagePath) != "":
        Log.Debug("Performing image-based search", "image_path", args.ImagePath)
        result, err = searchImagesByImage(ctx, cfg, collectionID, args.ImagePath, sTok)

    case strings.TrimSpace(args.Query) != "":
        Log.Debug("Performing text-based search", "query", args.Query)
        result, err = searchImagesByText(ctx, cfg, collectionID, args.Query, sTok)
    }

    // Standardized error handling consistent with other workers.
    if err != nil {
        switch {
        case errors.Is(err, context.Canceled):
            return runtime.MCPError(
                errors.E("search_images", errors.K.Timeout, "reason", "image search aborted: request was cancelled"),
            )
        case errors.Is(err, context.DeadlineExceeded):
            return runtime.MCPError(
                errors.E("search_images", errors.K.Timeout, "reason", "image search timed out"),
            )
        default:
            return runtime.MCPError(
                errors.E("search_images", errors.K.IO, "reason", "image search failed", err),
            )
        }
    }

    // Empty result is still a valid success path.
    if result == nil || len(result.Hits) == 0 {
        Log.Debug("No image hits found")
        empty := &ImageSearchResult{Hits: nil}
        return buildSearchImagesResultResponse(args, empty)
    }

    Log.Debug("Image search completed", "hits", len(result.Hits))
    return buildSearchImagesResultResponse(args, result)
}

//
// ────────────────────────────────────────────────────────────────
//   Collection resolution
// ────────────────────────────────────────────────────────────────
//
// Mirrors the clip worker logic: explicit > tenant default > error.
//
func resolveCollectionID(args SearchImagesArgs, tf *config.TenantFabric) (string, error) {
    if strings.TrimSpace(args.CollectionID) != "" {
        return args.CollectionID, nil
    }
    if tf != nil && strings.TrimSpace(tf.SearchCollectionID) != "" {
        return tf.SearchCollectionID, nil
    }
    return "", errors.E("collection_id is required and no default is configured for this tenant")
}

//
// ────────────────────────────────────────────────────────────────
//   HTTP + mapping: text search
// ────────────────────────────────────────────────────────────────
//
// These helpers intentionally avoid runtime.HTTPGet because search-ng
// requires POST bodies and multipart uploads. Using net/http directly
// keeps the logic explicit and testable.
//
type rawImageSearchResponse struct {
    Meta struct {
        Count int `json:"count"`
    } `json:"meta"`
    Results []struct {
        QID        string  `json:"qid"`
        Similarity float64 `json:"similarity"`
        MatchInfo  struct {
            FPS      string  `json:"fps"`
            FPSFloat float64 `json:"fps_float"`
            FrameIdx int64   `json:"frame_idx"`
            Offering string  `json:"offering"`
        } `json:"match_info"`
    } `json:"results"`
}

func searchImagesByText(
    ctx context.Context,
    cfg *config.Config,
    collectionID string,
    query string,
    stateToken string,
) (*ImageSearchResult, error) {

    url := fmt.Sprintf("%s/search-ng/collections/%s/search/text", cfg.SearchIdxUrl, collectionID)
    Log.Debug("POST text search", "url", url)

    body := map[string]string{"query": query}
    payload, err := json.Marshal(body)
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+stateToken)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Read entire body so we can log it AND decode it
    rawBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    Log.Debug("Image search text response",
        "status", resp.StatusCode,
        "body", string(rawBody),
    )

    var raw rawImageSearchResponse
    if err := json.Unmarshal(rawBody, &raw); err != nil {
        Log.Error("Cannot decode search images answer", "error", err)
        return nil, err
    }

    return mapRawToImageSearchResult(&raw), nil
}

//
// ────────────────────────────────────────────────────────────────
//   HTTP + mapping: image search (multipart upload)
// ────────────────────────────────────────────────────────────────
//

func searchImagesByImage(
    ctx context.Context,
    cfg *config.Config,
    collectionID string,
    imagePath string,
    stateToken string,
) (*ImageSearchResult, error) {

    url := fmt.Sprintf("%s/search-ng/collections/%s/search/image", cfg.SearchIdxUrl, collectionID)
    Log.Debug("POST image search", "url", url, "image_path", imagePath)

    file, err := os.Open(imagePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var buf bytes.Buffer
    writer := multipart.NewWriter(&buf)

    part, err := writer.CreateFormFile("file", imagePath)
    if err != nil {
        return nil, err
    }
    if _, err := io.Copy(part, file); err != nil {
        return nil, err
    }

    if err := writer.Close(); err != nil {
        return nil, err
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", writer.FormDataContentType())
    req.Header.Set("Authorization", "Bearer "+stateToken)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Read entire body so we can log it AND decode it
    rawBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    Log.Debug("Image search upload response",
        "status", resp.StatusCode,
        "body", string(rawBody),
    )

    var raw rawImageSearchResponse
    if err := json.Unmarshal(rawBody, &raw); err != nil {
        Log.Error("Cannot decode search images answer", "error", err)
        return nil, err
    }

    return mapRawToImageSearchResult(&raw), nil
}


//
// ────────────────────────────────────────────────────────────────
//   Mapping + slimming
// ────────────────────────────────────────────────────────────────
//
// These helpers normalize the raw API response into deterministic,
// LLM‑friendly structures. They intentionally avoid adding fields
// unless they are stable and well‑defined.
//
func mapRawToImageSearchResult(raw *rawImageSearchResponse) *ImageSearchResult {
    out := &ImageSearchResult{
        Hits: make([]ImageHit, 0, len(raw.Results)),
    }

    for _, r := range raw.Results {
        fps := r.MatchInfo.FPSFloat
        if fps == 0 {
            fps = 1 // defensive fallback
        }
        ts := float64(r.MatchInfo.FrameIdx) / fps

        out.Hits = append(out.Hits, ImageHit{
            QID:        r.QID,
            Similarity: r.Similarity,
            FrameIndex: r.MatchInfo.FrameIdx,
            FPS:        fps,
            Offering:   r.MatchInfo.Offering,
            Timestamp:  ts,
            ImageURL:   "", // filled later once frame URL strategy is defined
        })
    }

    return out
}

func slimImageResult(result *ImageSearchResult) []SlimImageItem {
    out := make([]SlimImageItem, 0, len(result.Hits))
    for _, h := range result.Hits {
        out = append(out, SlimImageItem{
            QID:        h.QID,
            Similarity: h.Similarity,
            FrameIndex: h.FrameIndex,
            FPS:        h.FPS,
            Offering:   h.Offering,
            Timestamp:  h.Timestamp,
            ImageURL:   h.ImageURL,
        })
    }
    return out
}

func buildImageSummary(result *ImageSearchResult, args SearchImagesArgs) string {
    count := len(result.Hits)

    target := strings.TrimSpace(args.Query)
    if target == "" && strings.TrimSpace(args.ImagePath) != "" {
        target = "uploaded image"
    }
    if count == 0 {
        return fmt.Sprintf("No images found for %q.", target)
    }

    top := result.Hits[0]
    return fmt.Sprintf(
        "%d images found for %q. Top similarity %.3f.",
        count,
        target,
        top.Similarity,
    )
}

func computeImageConfidence(result *ImageSearchResult) float64 {
    if len(result.Hits) == 0 {
        return 0.0
    }
    return result.Hits[0].Similarity
}

//
// ────────────────────────────────────────────────────────────────
//   Final slim JSON output
// ────────────────────────────────────────────────────────────────
//
// Mirrors buildSearchResultResponse from clip search. The worker
// returns both the slim JSON (for MCP) and the raw result (for tests).
//
func buildSearchImagesResultResponse(args SearchImagesArgs, result *ImageSearchResult) (*mcp.CallToolResult, any, error) {
    slim := slimImageResult(result)

    resp := SlimImageResponse{
        Summary:    buildImageSummary(result, args),
        Confidence: computeImageConfidence(result),
        Items:      slim,
    }

    jsonBytes, err := json.Marshal(resp)
    if err != nil {
        return runtime.MCPError(
            errors.E("search_images", errors.K.Internal, "reason", "failed to encode JSON"),
        )
    }

    content := []mcp.Content{
        &mcp.TextContent{Text: string(jsonBytes)},
    }

    return &mcp.CallToolResult{Content: content}, result, nil
}
