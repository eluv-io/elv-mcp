package fabric_test

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
	"github.com/qluvio/elv-mcp/tasks"
	"github.com/qluvio/elv-mcp/tasks/fabric"
)

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

func TestReplaceQueryToken(t *testing.T) {
	url := "https://example.com/video?ath=OLD&x=1"
	newURL, ok := fabric.ReplaceQueryToken(url, "ath", "NEW")

	if !ok {
		t.Fatalf("expected replacement")
	}
	if newURL != "https://example.com/video?ath=NEW&x=1" {
		t.Fatalf("unexpected result: %s", newURL)
	}
}

func TestReplaceQueryToken_NoMatch(t *testing.T) {
	url := "https://example.com/video?x=1"
	newURL, ok := fabric.ReplaceQueryToken(url, "ath", "NEW")

	if ok {
		t.Fatalf("expected no replacement")
	}
	if newURL != url {
		t.Fatalf("URL should remain unchanged")
	}
}

func TestRefreshToken_EmptyContents(t *testing.T) {
	cfg := &config.Config{}
	args := fabric.RefreshClipsArgs{Contents: []tasks.ClipItem{}}

	res, _, err := fabric.RefreshURLWorker(context.Background(), &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected MCP error result")
	}
}

func TestRefreshURLWorker_PopulatesDescription(t *testing.T) {
	// minimal args
	args := fabric.RefreshClipsArgs{
		Contents: []tasks.ClipItem{
			{VideoURL: "x?ath=old", ImageURL: "y?authorization=old"},
		},
	}

	cfg := &config.Config{}
	tenant := &config.TenantFabric{}
	ctx := runtime.WithTenant(context.Background(), tenant)

	// mock Auth
	prev := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prev }()

	_, resp, err := fabric.RefreshURLWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Description != argsDescription {
		t.Fatalf("Description mismatch.\nGot:\n%s\n\nExpected:\n%s",
			resp.Description, argsDescription)
	}
}
