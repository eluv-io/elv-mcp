package tasks

type ClipItem struct {
	QID       string `json:"id"`
	QLibID    string `json:"qlib_id"`
	VideoURL  string `json:"url"`
	ImageURL  string `json:"image_url"`
	Start     string `json:"start"`
	End       string `json:"end"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	ESToken   string `json:"esat"`

	// Metadata extracted from the select query parameter response
	DisplayTitle string `json:"display_title,omitempty"`
	ReleaseDate  string `json:"release_date,omitempty"`
	IPTitleID    string `json:"ip_title_id,omitempty"`
	Score		 string `json:"score,omitempty"`

	// Meta captures nested metadata returned by the search API
	Meta map[string]interface{} `json:"meta,omitempty"`
}

type ClipResponse struct {
	Description string     `json:"description"`
	Contents    []ClipItem `json:"contents"`
}
