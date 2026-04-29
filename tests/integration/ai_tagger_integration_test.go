//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	auth "github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/runtime"
	async "github.com/qluvio/elv-mcp/tasks/async"
	"github.com/qluvio/elv-mcp/tasks/taggers"
)

// -----------------------------------------------------------------------------
// Existing test (unchanged)
// -----------------------------------------------------------------------------

func TestGetQLibId_Integration(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	tf, ok := runtime.TenantFromContext(ctx)
	if !ok {
		t.Fatalf("no tenant configuration found for this user")
	}

	qlibId, user_error, err := auth.GetQLibId(cfg, tf, IntegrationTestQID)
	if err != nil {
		t.Fatalf("unexpected error: %v, user error: %v", err, user_error)
	}

	if qlibId == "" {
		t.Fatalf("expected non-empty qlibId, got empty string")
	}

	if user_error != "" {
		t.Fatalf("expected no user error, got: %s", user_error)
	}
}

// -----------------------------------------------------------------------------
// Tagger Integration Tests
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Single‑model synchronous (ASR)
// -----------------------------------------------------------------------------

func TestTagger_Sync_ASR(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := []string{"auto_captions"}

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	args := taggers.TagContentArgs{
		QID:         IntegrationTestQID,
		Synchronous: true,
		Jobs: []taggers.TagJobSpec{
			{Model: "asr"},
		},
	}

	res, result, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	syncRes := result.(*taggers.TagContentSyncResult)
	if len(syncRes.Jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if syncRes.Jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Single‑model asynchronous (ASR)
// -----------------------------------------------------------------------------

func TestTagger_Async_ASR(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := []string{"auto_captions"}

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	args := taggers.TagContentArgs{
		QID:         IntegrationTestQID,
		Synchronous: false,
		Jobs: []taggers.TagJobSpec{
			{Model: "asr"},
		},
	}

	_, result, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	asyncRes := result.(*taggers.TagContentAsyncResult)
	if asyncRes.TaskID == "" {
		t.Fatalf("expected non-empty task ID")
	}

	waitForTaskCompletion(t, asyncRes.TaskID)

	snap := mustGetTaskSnapshot(t, asyncRes.TaskID)
	jobs := snap.Result["result"].([]taggers.TagJobStatus)

	if len(jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Multi‑model synchronous (ASR + OCR)
// -----------------------------------------------------------------------------

func TestTagger_Sync_ASR_SHOT(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := []string{"auto_captions", "shot_detection"}

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	args := taggers.TagContentArgs{
		QID:         IntegrationTestQID,
		Synchronous: true,
		Jobs: []taggers.TagJobSpec{
			{Model: "asr"},
			{Model: "shot"},
		},
	}

	res, result, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	syncRes := result.(*taggers.TagContentSyncResult)
	if len(syncRes.Jobs) == 0 {
		t.Fatalf("expected at least one job")
	}

	if syncRes.Jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Multi‑model asynchronous (ASR + OCR)
// -----------------------------------------------------------------------------

func TestTagger_Async_ASR_SHOT(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := []string{"auto_captions", "shot_detection"}

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	args := taggers.TagContentArgs{
		QID:         IntegrationTestQID,
		Synchronous: false,
		Jobs: []taggers.TagJobSpec{
			{Model: "asr"},
			{Model: "shot"},
		},
	}

	_, result, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	asyncRes := result.(*taggers.TagContentAsyncResult)
	if asyncRes.TaskID == "" {
		t.Fatalf("expected non-empty task ID")
	}

	waitForTaskCompletion(t, asyncRes.TaskID)

	snap := mustGetTaskSnapshot(t, asyncRes.TaskID)
	jobs := snap.Result["result"].([]taggers.TagJobStatus)

	if len(jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Stop Tagging — Synchronous (all jobs)
// -----------------------------------------------------------------------------

func TestTagger_Stop_Not_Running(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Immediately stop all jobs
	stopArgs := taggers.TaggerStopArgs{
		QID: IntegrationTestQID,
	}

	res, stopResult, err := taggers.TaggerStopWorker(ctx, &mcp.CallToolRequest{}, stopArgs, cfg)

	if err == nil {
		t.Fatalf("Stopping tagging must fail")
	}
	if !res.IsError {
		t.Fatalf("Tool must return error: %+v", res)
	}

	syncRes := stopResult.(*taggers.TaggerStopResult)
	if len(syncRes.Jobs) != 1 {
		t.Fatalf("expected exactly one stopped job")
	}
	if syncRes.Jobs[0].Message != "No running jobs found for qid: "+string(stopArgs.QID) {
		t.Fatalf("expected specific error message, got %s", syncRes.Jobs[0].Message)
	}
}

func TestTagger_Stop_All(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := []string{"celebrity_detection"}

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	// Start tagging asynchronously so there is something to stop
	startArgs := taggers.TagContentArgs{
		QID:         IntegrationTestQID,
		Synchronous: false,
		Jobs: []taggers.TagJobSpec{
			{Model: "celeb"},
		},
		Options: &taggers.TaggerOptions{
			Replace: true,
		},
	}

	// PENDING ANDREA - remove after debug
	b, err := json.MarshalIndent(startArgs, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))

	_, startResult, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, startArgs, cfg)
	if err != nil {
		t.Fatalf("unexpected error starting tagging: %v", err)
	}

	startAsync := startResult.(*taggers.TagContentAsyncResult)
	if startAsync.TaskID == "" {
		t.Fatalf("expected non-empty task ID from start")
	}

	// Immediately stop all jobs
	stopArgs := taggers.TaggerStopArgs{
		QID: IntegrationTestQID,
	}

	time.Sleep(2 * time.Second) // wait to ensure the job has started before stopping

	res, stopResult, err := taggers.TaggerStopWorker(ctx, &mcp.CallToolRequest{}, stopArgs, cfg)
	if err != nil {
		t.Fatalf("unexpected error stopping tagging: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	syncRes := stopResult.(*taggers.TaggerStopResult)
	if len(syncRes.Jobs) == 0 {
		t.Fatalf("expected at least one stopped job")
	}
	if syncRes.Jobs[0].JobID == "" {
		t.Fatalf("expected non-empty job_id")
	}
	if syncRes.Jobs[0].Message == "" {
		t.Fatalf("expected non-empty message")
	}
}

// -----------------------------------------------------------------------------
// Stop Tagging — Synchronous (specific model)
// -----------------------------------------------------------------------------

func TestTagger_Stop_Model(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := []string{"optical_character_recognition", "celebrity_detection"}

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	// Start tagging asynchronously for a specific model
	startArgs := taggers.TagContentArgs{
		QID:         IntegrationTestQID,
		Synchronous: false,
		Jobs: []taggers.TagJobSpec{
			{Model: "ocr"},
			{Model: "celeb"},
		},
		Options: &taggers.TaggerOptions{
			Replace: true,
		},
	}

	_, startResult, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, startArgs, cfg)
	if err != nil {
		t.Fatalf("unexpected error starting tagging: %v", err)
	}

	startAsync := startResult.(*taggers.TagContentAsyncResult)
	if startAsync.TaskID == "" {
		t.Fatalf("expected non-empty task ID from start")
	}

	time.Sleep(5 * time.Second) // wait to ensure the job has started before stopping

	// Stop only ASR jobs
	stopArgs := taggers.TaggerStopArgs{
		QID:   IntegrationTestQID,
		Model: "celeb",
	}

	res, stopResult, err := taggers.TaggerStopWorker(ctx, &mcp.CallToolRequest{}, stopArgs, cfg)
	if err != nil {
		t.Fatalf("unexpected error stopping tagging: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	syncRes := stopResult.(*taggers.TaggerStopResult)
	if len(syncRes.Jobs) == 0 {
		t.Fatalf("expected at least one stopped job")
	}
	if syncRes.Jobs[0].JobID == "" {
		t.Fatalf("expected non-empty job_id")
	}
	if syncRes.Jobs[0].Message == "" {
		t.Fatalf("expected non-empty message")
	}

	waitForTaskCompletion(t, startAsync.TaskID)
	// test that the stopped model is not complete, but the other model is complete
	snap := mustGetTaskSnapshot(t, startAsync.TaskID)
	result := snap.Result["result"].([]taggers.TagJobStatus)
	if len(result) != 2 {
		t.Fatalf("expected status for 2 jobs, got %d", len(result))
	}
	for _, job := range result {
		// PENDING ANDREA - need to understand how this is signaled excatly. It'd definitively not asynch.StatusFailed, we need to check the result
		if job.Model == "celeb" && job.Status != "cancelled" {
			t.Fatalf("expected celeb job to be stopped, got status %s", job.Status)
		}
		if job.Model == "ocr" && job.Status != "succeeded" {
			t.Fatalf("expected ocr job to be complete, got status %s", job.Status)
		}
	}

}

// -----------------------------------------------------------------------------
// List Models — Integration Tests
// -----------------------------------------------------------------------------

func TestTagger_ListModels_Success(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	res, payload, err := taggers.TaggerListModelsWorker(
		ctx,
		&mcp.CallToolRequest{},
		taggers.ListModelsArgs{},
		cfg,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	out, ok := payload.(*taggers.ModelsResponse)
	if !ok {
		t.Fatalf("expected *ModelsResponse payload, got %T", payload)
	}

	if len(out.Models) == 0 {
		t.Fatalf("expected at least one model")
	}

	println("Available models:")
	for _, model := range out.Models {
		println("- " + model.Name)
	}
	first := out.Models[0]

	if first.Name == "" {
		t.Fatalf("expected non-empty model name")
	}
	if first.Type == "" {
		t.Fatalf("expected non-empty model type")
	}
	if len(first.TagTracks) == 0 {
		t.Fatalf("expected at least one tag track")
	}
	if first.TagTracks[0].Name == "" {
		t.Fatalf("expected non-empty tag track name")
	}
}

func TestTagger_ListModels_InvalidURL(t *testing.T) {
	cfg := loadIntegrationConfig(t)

	// Force an invalid URL
	cfg.AITaggerUrl = "http://127.0.0.1:9" // guaranteed closed port

	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	res, payload, err := taggers.TaggerListModelsWorker(
		ctx,
		&mcp.CallToolRequest{},
		taggers.ListModelsArgs{},
		cfg,
	)

	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if res == nil {
		t.Fatalf("CallToolResult must not be nil on error")
	}
	if !res.IsError {
		t.Fatalf("IsError must be true on error")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

// -----------------------------------------------------------------------------
// Tag Status — Summary (all models)
// -----------------------------------------------------------------------------

func TestTagger_TagStatus_Summary(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	args := taggers.TaggerTagStatusArgs{
		QID: IntegrationTestQID,
	}

	// Tracks created by this test
	createdTracks := []string{"auto_captions"}

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	// Start tagging asynchronously so there is status to retrieve
	startArgs := taggers.TagContentArgs{
		QID:         IntegrationTestQID,
		Synchronous: false,
		Jobs: []taggers.TagJobSpec{
			{Model: "asr"},
		},
	}
	_, _, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, startArgs, cfg)

	if err != nil {
		t.Fatalf("unexpected error starting tagging: %v", err)
	}

	time.Sleep(2 * time.Second) // wait to ensure the job has started before checking status

	res, payload, err := taggers.TaggerTagStatusWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	summaries, ok := payload.(taggers.TagStatusSummaryResponse)
	if !ok {
		t.Fatalf("expected TagStatusSummaryResponse, got %T", payload)
	}

	if len(summaries.Statuses) == 0 {
		t.Fatalf("expected at least one model summary")
	}

	first := summaries.Statuses[0]
	if first.Model == "" {
		t.Fatalf("expected non-empty model")
	}
	if first.Track == "" {
		t.Fatalf("expected non-empty track")
	}
	if first.PercentComplete < 0 {
		t.Fatalf("expected percent_complete >= 0")
	}
}

// -----------------------------------------------------------------------------
// Tag Status — Model Detail
// -----------------------------------------------------------------------------

func TestTagger_TagStatus_ModelDetail(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := []string{"shot_detection"}

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	// Start tagging asynchronously so there is status to retrieve
	startArgs := taggers.TagContentArgs{
		QID:         IntegrationTestQID,
		Synchronous: true,
		Jobs: []taggers.TagJobSpec{
			{Model: "shot"},
		},
	}
	_, _, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, startArgs, cfg)

	if err != nil {
		t.Fatalf("unexpected error starting tagging: %v", err)
	}

	// Choose a model that is known to exist in your environment
	args := taggers.TaggerTagStatusArgs{
		QID:   IntegrationTestQID,
		Model: "shot",
	}

	res, payload, err := taggers.TaggerTagStatusWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	detail, ok := payload.(taggers.TagStatusModelDetail)
	if !ok {
		t.Fatalf("expected TagStatusModelDetail, got %T", payload)
	}

	if detail.Summary.Model == "" {
		t.Fatalf("expected non-empty model")
	}
	if detail.Summary.Track == "" {
		t.Fatalf("expected non-empty track")
	}
	if detail.Summary.PercentComplete < 0 {
		t.Fatalf("expected percent_complete >= 0")
	}

	// Jobs list may be empty depending on environment, but must decode cleanly
	for _, job := range detail.Jobs {
		if job.TimeRan == "" {
			t.Fatalf("expected non-empty time_ran")
		}
	}
}

// -----------------------------------------------------------------------------
// Tag Status — Invalid QID
// -----------------------------------------------------------------------------

func TestTagger_TagStatus_InvalidQID(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	args := taggers.TaggerTagStatusArgs{
		QID: "iq__DOES_NOT_EXIST",
	}

	res, payload, err := taggers.TaggerTagStatusWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err == nil {
		t.Fatalf("expected error for invalid qid")
	}
	if res == nil {
		t.Fatalf("CallToolResult must not be nil on error")
	}
	if !res.IsError {
		t.Fatalf("IsError must be true on error")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

// -----------------------------------------------------------------------------
// Character Tagging — Integration Tests
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Character Tagging — Synchronous (celeb already complete)
// -----------------------------------------------------------------------------

func TestTagCharacters_Sync_Success(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	args := taggers.CharacterTaggingArgs{
		QID:         IntegrationTestQID,
		Synchronous: true,
	}

	res, payload, err := taggers.TagCharactersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	out, ok := payload.(*taggers.CharacterTaggingSyncResult)
	if !ok {
		t.Fatalf("expected CharacterTaggingSyncResult, got %T", payload)
	}

	if len(out.Jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if out.Jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Character Tagging — Synchronous (auto-run celeb)
// -----------------------------------------------------------------------------

func TestTagCharacters_Sync_AutoRunDependencies(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	args := taggers.CharacterTaggingArgs{
		QID:                 IntegrationTestQID,
		Synchronous:         true,
		AutoRunDependencies: true,
	}

	res, payload, err := taggers.TagCharactersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	out, ok := payload.(*taggers.CharacterTaggingSyncResult)
	if !ok {
		t.Fatalf("expected CharacterTaggingSyncResult, got %T", payload)
	}

	// Auto-run may or may not occur depending on environment,
	// but the field must decode cleanly.
	if out.AutoRanDependencies == nil {
		t.Fatalf("expected auto_ran_dependencies field to be present")
	}

	if len(out.Jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if out.Jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Character Tagging — Asynchronous (celeb already complete)
// -----------------------------------------------------------------------------

func TestTagCharacters_Async_Success(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	args := taggers.CharacterTaggingArgs{
		QID:         IntegrationTestQID,
		Synchronous: false,
	}

	_, payload, err := taggers.TagCharactersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	asyncRes, ok := payload.(*taggers.CharacterTaggingAsyncResult)
	if !ok {
		t.Fatalf("expected CharacterTaggingAsyncResult, got %T", payload)
	}
	if asyncRes.TaskID == "" {
		t.Fatalf("expected non-empty task ID")
	}

	waitForTaskCompletion(t, asyncRes.TaskID)

	snap := mustGetTaskSnapshot(t, asyncRes.TaskID)
	result := snap.Result["result"].(*taggers.CharacterTaggingSyncResult)

	if len(result.Jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if result.Jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Character Tagging — Asynchronous (auto-run celeb)
// -----------------------------------------------------------------------------

func TestTagCharacters_Async_AutoRunDependencies(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	args := taggers.CharacterTaggingArgs{
		QID:                 IntegrationTestQID,
		Synchronous:         false,
		AutoRunDependencies: true,
	}

	_, payload, err := taggers.TagCharactersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	asyncRes, ok := payload.(*taggers.CharacterTaggingAsyncResult)
	if !ok {
		t.Fatalf("expected CharacterTaggingAsyncResult, got %T", payload)
	}
	if asyncRes.TaskID == "" {
		t.Fatalf("expected non-empty task ID")
	}

	waitForTaskCompletion(t, asyncRes.TaskID)

	snap := mustGetTaskSnapshot(t, asyncRes.TaskID)
	result := snap.Result["result"].(*taggers.CharacterTaggingSyncResult)

	if result.AutoRanDependencies == nil {
		t.Fatalf("expected auto_ran_dependencies field to be present")
	}

	if len(result.Jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if result.Jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Character Tagging — Missing Dependencies Without AutoRun
// -----------------------------------------------------------------------------

func TestTagCharacters_MissingDependencies_NoAutoRun(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	args := taggers.CharacterTaggingArgs{
		QID:         IntegrationTestQID,
		Synchronous: true,
		// AutoRunDependencies = false
	}

	res, payload, err := taggers.TagCharactersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if res == nil {
		t.Fatalf("CallToolResult must not be nil on error")
	}
	if !res.IsError {
		t.Fatalf("IsError must be true on error")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

// -----------------------------------------------------------------------------
// Character Tagging — Invalid QID
// -----------------------------------------------------------------------------

func TestTagCharacters_InvalidQID(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	args := taggers.CharacterTaggingArgs{
		QID: "iq__DOES_NOT_EXIST",
		Synchronous: true,
	}

	res, payload, err := taggers.TagCharactersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err == nil {
		t.Fatalf("expected error for invalid qid")
	}
	if res == nil {
		t.Fatalf("CallToolResult must not be nil on error")
	}
	if !res.IsError {
		t.Fatalf("IsError must be true on error")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

// -----------------------------------------------------------------------------
// Chapters Tagging — Integration Tests
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Chapters Tagging — Synchronous (speaker already complete)
// -----------------------------------------------------------------------------

func TestTagChapters_Sync_Success(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := append(taggers.ModelDependencies[taggers.ChaptersModelName], "chapter")

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	args := taggers.ChaptersTaggingArgs{
		QID:         IntegrationTestQID,
		Synchronous: true,
	}

	res, payload, err := taggers.TagChaptersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	out, ok := payload.(*taggers.ChaptersTaggingSyncResult)
	if !ok {
		t.Fatalf("expected ChaptersTaggingSyncResult, got %T", payload)
	}

	if len(out.Jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if out.Jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Chapters Tagging — Synchronous (auto-run speaker)
// -----------------------------------------------------------------------------

func TestTagChapters_Sync_AutoRunDependencies(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := append(taggers.ModelDependencies[taggers.ChaptersModelName], "chapter")

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	args := taggers.ChaptersTaggingArgs{
		QID:                 IntegrationTestQID,
		Synchronous:         true,
		AutoRunDependencies: true,
	}

	res, payload, err := taggers.TagChaptersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}

	out, ok := payload.(*taggers.ChaptersTaggingSyncResult)
	if !ok {
		t.Fatalf("expected ChaptersTaggingSyncResult, got %T", payload)
	}

	// Auto-run may or may not occur depending on environment,
	// but the field must decode cleanly.
	if out.AutoRanDependencies == nil {
		t.Fatalf("expected auto_ran_dependencies field to be present")
	}

	if len(out.Jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if out.Jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Chapters Tagging — Asynchronous (speaker already complete)
// -----------------------------------------------------------------------------

func TestTagChapters_Async_Success(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := append(taggers.ModelDependencies[taggers.ChaptersModelName], "chapter")

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	args := taggers.ChaptersTaggingArgs{
		QID:         IntegrationTestQID,
		Synchronous: false,
		Options: &taggers.TaggerOptions{
			Replace: false,
		},
	}

	_, payload, err := taggers.TagChaptersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	asyncRes, ok := payload.(*taggers.ChaptersTaggingAsyncResult)
	if !ok {
		t.Fatalf("expected ChaptersTaggingAsyncResult, got %T", payload)
	}
	if asyncRes.TaskID == "" {
		t.Fatalf("expected non-empty task ID")
	}

	waitForTaskCompletion(t, asyncRes.TaskID)

	snap := mustGetTaskSnapshot(t, asyncRes.TaskID)
	result := snap.Result["result"].(*taggers.ChaptersTaggingSyncResult)

	if len(result.Jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if result.Jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Chapters Tagging — Asynchronous (auto-run speaker)
// -----------------------------------------------------------------------------

func TestTagChapters_Async_AutoRunDependencies(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := append(taggers.ModelDependencies[taggers.ChaptersModelName], "chapter")

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	args := taggers.ChaptersTaggingArgs{
		QID:                 IntegrationTestQID,
		Synchronous:         false,
		AutoRunDependencies: true,
	}

	_, payload, err := taggers.TagChaptersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	asyncRes, ok := payload.(*taggers.ChaptersTaggingAsyncResult)
	if !ok {
		t.Fatalf("expected ChaptersTaggingAsyncResult, got %T", payload)
	}
	if asyncRes.TaskID == "" {
		t.Fatalf("expected non-empty task ID")
	}

	waitForTaskCompletion(t, asyncRes.TaskID)

	snap := mustGetTaskSnapshot(t, asyncRes.TaskID)
	result := snap.Result["result"].(*taggers.ChaptersTaggingSyncResult)

	if result.AutoRanDependencies == nil {
		t.Fatalf("expected auto_ran_dependencies field to be present")
	}

	if len(result.Jobs) == 0 {
		t.Fatalf("expected at least one job")
	}
	if result.Jobs[0].Status == "" {
		t.Fatalf("expected non-empty status")
	}
}

// -----------------------------------------------------------------------------
// Chapters Tagging — Missing Dependencies Without AutoRun
// -----------------------------------------------------------------------------

func TestTagChapters_MissingDependencies_NoAutoRun(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Tracks created by this test
	createdTracks := []string{"chapter"}

	// Always clean up what we create
	defer deleteTracksBestEffort(t, ctx, cfg, createdTracks...)

	args := taggers.ChaptersTaggingArgs{
		QID:         IntegrationTestQID,
		Synchronous: true,
		// AutoRunDependencies = false
	}

	res, payload, err := taggers.TagChaptersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if res == nil {
		t.Fatalf("CallToolResult must not be nil on error")
	}
	if !res.IsError {
		t.Fatalf("IsError must be true on error")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

// -----------------------------------------------------------------------------
// Chapters Tagging — Invalid QID
// -----------------------------------------------------------------------------

func TestTagChapters_InvalidQID(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	args := taggers.ChaptersTaggingArgs{
		QID: "iq__DOES_NOT_EXIST",
		Synchronous: true,
	}

	res, payload, err := taggers.TagChaptersWorker(
		ctx,
		&mcp.CallToolRequest{},
		args,
		cfg,
	)

	if err == nil {
		t.Fatalf("expected error for invalid qid")
	}
	if res == nil {
		t.Fatalf("CallToolResult must not be nil on error")
	}
	if !res.IsError {
		t.Fatalf("IsError must be true on error")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

// -----------------------------------------------------------------------------
// Task helpers (same pattern as search integration tests)
// -----------------------------------------------------------------------------

func waitForTaskCompletion(t *testing.T, taskID string) {
	t.Helper()

	deadline := time.Now().Add(180 * time.Second)

	for {
		snap, ok := async.GetSnapshot(taskID)
		if !ok {
			t.Fatalf("task not found: %s", taskID)
		}

		if snap.Status == async.StatusCompleted {
			return
		}
		if snap.Status == async.StatusFailed {
			t.Fatalf("task failed: %v", snap.Error)
		}

		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for task to complete")
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func mustGetTaskSnapshot(t *testing.T, taskID string) async.Snapshot {
	t.Helper()

	snap, ok := async.GetSnapshot(taskID)
	if !ok {
		t.Fatalf("task not found: %s", taskID)
	}
	return snap
}
