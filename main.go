package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ------------------------------------------------------------------
// Environment variables
// ------------------------------------------------------------------
var BaseURL, QLibID, QID, AuthToken, NodeURL, AuthTokenObj, BaseURLVid string

func loadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Printf(".env not found or could not be loaded: %v (will rely on existing environment)", err)
	}
	BaseURL = os.Getenv("SEARCH_BASE_URL")
	QLibID = os.Getenv("QLIBID_INDEX")
	QID = os.Getenv("QID_INDEX")
	AuthToken = os.Getenv("INDEX_AUTH_TOKEN")
	NodeURL = os.Getenv("IMAGE_BASE_URL")
	AuthTokenObj = os.Getenv("QAUTH_TOKEN")
	BaseURLVid = os.Getenv("VID_BASE_URL")
}

// ------------------------------------------------------------------
// Tool request / response payloads
// ------------------------------------------------------------------

type SearchClipsArgs struct {
	Terms                  string   `json:"terms"` // required
	SearchFields           []string `json:"search_fields,omitempty"`
	DisplayFields          []string `json:"display_fields,omitempty"`
	Semantic               string   `json:"semantic,omitempty"`
	Start                  int      `json:"start,omitempty"`                     // default 0
	Limit                  int      `json:"limit,omitempty"`                     // default 20
	MaxTotal               int      `json:"max_total,omitempty"`                 // default 100
	Debug                  bool     `json:"debug,omitempty"`                     // default false
	Clips                  *bool    `json:"clips,omitempty"`                     // default true
	ClipsIncludeSourceTags *bool    `json:"clips_include_source_tags,omitempty"` // default true
	Thumbnails             *bool    `json:"thumbnails,omitempty"`                // default true
}

type clipItem struct {
	VideoURL  string `json:"url"`
	ImageURL  string `json:"image_url"`
	Start     string `json:"start"`
	End       string `json:"end"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

type clipResponse struct {
	Contents []clipItem `json:"contents"`
}

// searchClipsResult is the structured object returned to the LLM per MCP call.
// It includes a per-call Description plus the list of clip Contents.
type searchClipsResult struct {
	Description string     `json:"description"`
	Contents    []clipItem `json:"contents"`
}

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

func boolOrDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

// buildNodeThumbURL builds:  {nodeURL}/q/{imageURL}{sep}authorization={token}
// - imageURL is used AS-IS (no rewriting, no escaping, no leading slash added)
// - sep is "&" if imageURL already has '?', otherwise "?"
// buildNodeThumbURL builds:  {nodeURL}/q/{imageURL}{sep}authorization={token}
// If imageURL already starts with "q/" or "/q/", we do NOT add another "/q/".
func buildNodeThumbURL(nodeURL, imageURL, token string) string {
	if nodeURL == "" || imageURL == "" {
		return ""
	}

	nodeURL = strings.TrimRight(nodeURL, "/")

	// Ensure we only add one /q/
	var final string
	if strings.HasPrefix(imageURL, "q/") || strings.HasPrefix(imageURL, "/q/") {
		final = nodeURL + "/" + strings.TrimLeft(imageURL, "/")
	} else {
		final = nodeURL + "/q/" + strings.TrimLeft(imageURL, "/")
	}

	// If auth is empty or already present, don't append again
	if token == "" || strings.Contains(imageURL, "authorization=") || strings.Contains(final, "authorization=") {
		return final
	}

	sep := "?"
	if strings.Contains(imageURL, "?") {
		sep = "&"
	}
	return final + sep + "authorization=" + url.QueryEscape(token)
}

// ------------------------------------------------------------------
// Search API
// ------------------------------------------------------------------

func buildSearchURL(args SearchClipsArgs) (string, error) {
	if strings.TrimSpace(BaseURL) == "" ||
		strings.TrimSpace(QLibID) == "" ||
		strings.TrimSpace(QID) == "" {
		return "", fmt.Errorf("missing BASE_URL or QLIB_ID or CONTENT_ID")
	}
	if strings.TrimSpace(args.Terms) == "" {
		return "", fmt.Errorf("terms cannot be empty")
	}

	base := strings.TrimRight(BaseURL, "/")
	u, err := url.Parse(fmt.Sprintf("%s/search/qlibs/%s/q/%s/rep/search",
		base, url.PathEscape(QLibID), url.PathEscape(QID)))
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("terms", args.Terms)

	// ✅ Turn on semantic search (recommended by Eluvio docs)
	q.Set("semantic", args.Semantic)

	if len(args.SearchFields) > 0 {
		q.Set("search_fields", strings.Join(args.SearchFields, ","))
	}
	if len(args.DisplayFields) > 0 {
		q.Set("display_fields", strings.Join(args.DisplayFields, ","))
	}

	if args.Start > 0 {
		q.Set("start", fmt.Sprint(args.Start))
	} else {
		q.Set("start", "0")
	}
	if args.Limit > 0 {
		q.Set("limit", fmt.Sprint(args.Limit))
	} else {
		q.Set("limit", "20")
	}
	if args.MaxTotal > 0 {
		q.Set("max_total", fmt.Sprint(args.MaxTotal))
	} else {
		q.Set("max_total", "100")
	}

	clips := true
	if args.Clips != nil {
		clips = *args.Clips
	}
	if clips {
		q.Set("clips", "true")
	}

	includeTags := true
	if args.ClipsIncludeSourceTags != nil {
		includeTags = *args.ClipsIncludeSourceTags
	}
	if includeTags {
		q.Set("clips_include_source_tags", "true")
	}

	if args.Debug {
		q.Set("debug", "true")
	}

	// Add token as query too (some gateways accept either header or query)
	if strings.TrimSpace(AuthToken) != "" {
		q.Set("authorization", AuthToken)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func buildVideoURL(videoURL, token, start, end string) string {
	hash := extractHash(videoURL)
	if hash == "" {
		// If we can't find a hash, return the original URL unchanged
		return videoURL
	}

	base := strings.TrimSpace(BaseURLVid)
	if base == "" {
		return videoURL
	}

	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return videoURL
	}

	// Build query manually to preserve exact ordering and values
	type kv struct{ k, v string }
	params := []kv{
		{"p", ""},
		{"net", "main"},
		{"cid", hash},
		{"mt", "v"},
	}
	if token != "" {
		params = append(params, kv{"ath", token})
	}
	params = append(params, kv{"ct", "h"})
	if start != "" {
		params = append(params, kv{"start", start})
	}
	if end != "" {
		params = append(params, kv{"end", end})
	}

	var b strings.Builder
	for i, pair := range params {
		if i == 0 {
			b.WriteByte('?')
		} else {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(pair.k))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(pair.v))
	}

	u.RawQuery = "" // ensure we fully control the query string
	return u.String() + b.String()
}

func extractHash(videoURL string) string {
	// Find the start of the hash ("hq__" or "iq__")
	start := strings.Index(videoURL, "hq__")
	if start == -1 {
		start = strings.Index(videoURL, "iq__")
	}
	if start == -1 {
		return ""
	}

	// Cut off anything after "/rep/"
	hash := videoURL[start:]
	if i := strings.Index(hash, "/rep/"); i != -1 {
		hash = hash[:i]
	}
	if i := strings.Index(hash, "?"); i != -1 {
		hash = hash[:i]
	}
	return hash
}

func doGET(ctx context.Context, urlStr string) (*clipResponse, *http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(AuthToken) != "" {
		req.Header.Set("Authorization", "Bearer "+AuthToken)
	}

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp, fmt.Errorf("search API returned %s", resp.Status)
	}

	var out clipResponse
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ------------------------------------------------------------------
// main: MCP server over stdio
// ------------------------------------------------------------------

func main() {
	loadEnv()
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "eluvio-search-mcp",
		Version: "1.0.0",
	}, nil)

	// Tool: search_clips (with NodeURL-based thumbnails)
	mcp.AddTool[SearchClipsArgs](server, &mcp.Tool{
		Name:        "search_clips",
		Description: "Searches the Eluvio Search API and returns video clip results with thumbnail images. The tool provides a thumbnail_url field for each clip. The assistant should always display these thumbnails inline as images in the chat and should not output raw URLs unless necessary. Thumbnails should appear first, followed by the clip’s link or details.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SearchClipsArgs) (*mcp.CallToolResult, any, error) {
		searchURL, err := buildSearchURL(args)
		log.Println("search URL: %s", searchURL)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		result, httpResp, err := doGET(ctx, searchURL)
		log.Println("search API response:", result, httpResp, err)
		if err != nil {
			var detail string
			if httpResp != nil {
				detail = fmt.Sprintf(" (HTTP %d)", httpResp.StatusCode)
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Search failed%s: %v", detail, err)}},
				IsError: true,
			}, nil, nil
		}

		showThumbs := boolOrDefault(args.Thumbnails, true)
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

			// Build clip URL and persist it into the result
			video := buildVideoURL(c.VideoURL, AuthTokenObj, startStr, endStr)
			result.Contents[i].VideoURL = video

			// Build thumbnail URL (if requested) and persist in the result
			var thumb string
			if showThumbs {
				thumb = buildNodeThumbURL(NodeURL, c.ImageURL, AuthToken)
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

		combined := searchClipsResult{
			Description: "Searches the Eluvio Search API and returns video clip results with thumbnail images. The tool provides a thumbnail_url field for each clip. The assistant should always display these thumbnails inline as images in the chat and should not output raw URLs unless necessary. A thumbnail image URL embedded inside a clickable link, formatted exactly like this: [![Title or description](THUMBNAIL_URL)](VIDEO_URL) ",
			Contents:    result.Contents,
		}

		prettyJSON, _ := json.MarshalIndent(combined, "", "  ")
		content := []mcp.Content{
			&mcp.TextContent{Text: sb.String()},
			&mcp.TextContent{Text: "Raw JSON:\n" + string(prettyJSON)},
		}
		return &mcp.CallToolResult{Content: content}, combined, nil
	})

	sseHandler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server { return server }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", sseHandler)
	log.Println("MCP server listening on http://localhost:8080/mcp")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}

	//if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
	//	log.Printf("Server failed: %v", err)
	//} else {
	//	log.Printf("Server exited")
	//}
}
