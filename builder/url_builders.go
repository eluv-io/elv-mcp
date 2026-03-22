package builder

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/qluvio/elv-mcp/types"
)

// BuildNodeThumbURL builds: {baseURL}/t/{token}{imageURL}
// imageURL is expected to start with /q/ as returned by the search API.
func BuildNodeThumbURL(imageURL, token string, cfg *types.Config) string {
	if imageURL == "" {
		return ""
	}

	base := strings.TrimRight(cfg.ImgBaseUrl, "/")
	path := imageURL
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if token == "" {
		return base + path
	}
	return base + "/t/" + token + path
}

// BuildSearchURL constructs the search Index API URL using config and args.
func BuildSearchURL(cfg *types.Config, args types.SearchClipsArgs, token string) (string, error) {
	if strings.TrimSpace(args.Terms) == "" {
		return "", fmt.Errorf("terms cannot be empty")
	}

	u, err := url.Parse(fmt.Sprintf(
		"%s/search/qlibs/%s/q/%s/rep/search",
		cfg.SearchIdxUrl,
		url.PathEscape(cfg.QLibIndexID),
		url.PathEscape(cfg.QIndexID),
	))
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("terms", args.Terms)

	// Only include semantic if caller provided it.
	// This preserves original behavior when Semantic is the empty string
	// (the parameter is effectively absent).
	if strings.TrimSpace(args.Semantic) != "" {
		q.Set("semantic", args.Semantic)
	}

	if len(args.SearchFields) > 0 {
		q.Set("search_fields", strings.Join(args.SearchFields, ","))
	}
	if len(args.DisplayFields) > 0 {
		q.Set("display_fields", strings.Join(args.DisplayFields, ","))
	}

	// Always request metadata via select
	q.Set("select", "/public/asset_metadata/display_title,/public/asset_metadata/info/release_date,/public/asset_metadata/ip_title_id")

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

	clips := types.BoolOrDefault(args.Clips, true)
	if clips {
		q.Set("clips", "true")
	}

	includeTags := types.BoolOrDefault(args.ClipsIncludeSourceTags, true)
	if includeTags {
		q.Set("clips_include_source_tags", "true")
	}

	if args.Debug {
		q.Set("debug", "true")
	}

	// Add token as query too (some gateways accept either header or query)
	if strings.TrimSpace(token) != "" {
		q.Set("authorization", token)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// BuildVideoURL builds the final clip URL with start/end and auth.
func BuildVideoURL(videoURL, token, start, end string, cfg *types.Config) string {
	hash := extractHash(videoURL)
	if hash == "" {
		// If we can't find a hash, return the original URL unchanged
		return videoURL
	}

	u, err := url.Parse(cfg.VidBaseUrl)
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

// extractHash tries to pull the content hash (hq__ or iq__) out of the URL.
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
