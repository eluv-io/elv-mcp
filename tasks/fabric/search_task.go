package fabric

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// SearchClipsTask implements the Task interface for the "search_clips"
// MCP tool.
//
// This task is responsible for:
//
//   - Exposing the tool name used by the LLM ("search_clips")
//   - Providing the human-readable description of the tool
//   - Registering the tool with the MCP server
//   - Wiring the tool to the underlying business logic (SearchWorker)
//
// The task does NOT contain business logic itself. Instead, it delegates
// execution to fabric.SearchWorker, keeping registration concerns separate
// from functional concerns.
//
// This design mirrors an object-oriented command pattern: each task is a
// self-contained unit that knows how to describe and register itself.
type SearchClipsTask struct{}

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

func NewSearchClipsTask() *SearchClipsTask {
	return &SearchClipsTask{}
}

func init() {
	tasks.Register(NewSearchClipsTask())
}

// Name returns the MCP tool name exposed to the LLM.
func (SearchClipsTask) Name() string {
	return "search_clips"
}

// Description returns the human-readable description of the tool.
// This is what the LLM sees when deciding whether to call the tool.
func (SearchClipsTask) Description() string {
	return "Search Eluvio for video clips matching a user-provided text query.\n\n" +
		"Use this tool whenever the user asks to search for clips, find clips, look up clips, retrieve clips, or browse clips matching a phrase or keyword.\n\n" +
		"Required parameter:\n" +
		"  • terms — a non-empty plain text search string from the user's request (example: \"birthday cake\").\n" +
		"    Always provide `terms` as a simple string, not an object, array, or nested structure.\n\n" +
		"Optional parameters:\n" +
		"  • limit — maximum number of results.\n" +
		"  • search_fields — fields to search.\n" +
		"  • display_fields — fields to return.\n" +
		"  • semantic — enable semantic search.\n" +
		"  • start — pagination offset.\n" +
		"  • max_total — cap on total matches.\n" +
		"  • debug — verbose output.\n" +
		"  • clips, clips_include_source_tags, thumbnails — formatting controls.\n\n" +
		"Rules:\n" +
		"  • If the user requests clip search and a query exists, call this tool.\n" +
		"  • Do not answer from memory when clip search is requested.\n" +
		"  • Do not explain how to call the tool.\n" +
		"  • Do not output command-line syntax.\n" +
		"  • Never call this tool with empty `terms`.\n\n" +
		"Example call:\n" +
		"  {\"terms\": \"birthday cake\", \"limit\": 5}"
}

// Register wires this task into the MCP server by calling AddTool.
// It binds the tool metadata to the SearchWorker implementation.
func (SearchClipsTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        SearchClipsTask{}.Name(),
			Description: SearchClipsTask{}.Description(),
		}, func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args SearchClipsArgs,
		) (*mcp.CallToolResult, *tasks.ClipResponse, error) {
			return SearchWorker(ctx, req, args, cfg)
		})
}
