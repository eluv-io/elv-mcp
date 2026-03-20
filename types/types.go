package types

// SearchClipsArgs is the input structure for the search_clips MCP tool.
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

type ClipItem struct {
	QID       string `json:"id"`
	QLibID    string `json:"qlib_id"`
	VideoURL  string `json:"url"`
	ImageURL  string `json:"image_url"`
	Start     string `json:"start"`
	End       string `json:"end"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

type ClipResponse struct {
	Description string     `json:"description"`
	Contents    []ClipItem `json:"contents"`
}

type RefreshClipsArgs struct {
	Contents []ClipItem `json:"contents"`
}
