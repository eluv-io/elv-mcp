package taggers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eluv-io/errors-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	elog "github.com/eluv-io/log-go"
	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
	async "github.com/qluvio/elv-mcp/tasks/async"
)

var Log = elog.Get("/taggers")

// -----------------------------------------------------------------------------
// Internal request/response structures for Tagger API
// -----------------------------------------------------------------------------

// StartJobsRequest mirrors the Tagger StartJobsRequest schema.
type StartJobsRequest struct {
	Options *TaggerOptions `json:"options,omitempty"`
	Jobs    []TagJobSpec   `json:"jobs"`
}

// StartTaggingResponse mirrors the Tagger StartTaggingResponse schema.
type StartTaggingResponse struct {
	Jobs []StartStatus `json:"jobs"`
}

// StartStatus mirrors the Tagger StartStatus schema.
type StartStatus struct {
	JobID   string  `json:"job_id"`
	Model   string  `json:"model"`
	Stream  string  `json:"stream"`
	Started bool    `json:"started"`
	Message string  `json:"message"`
	Error   *string `json:"error,omitempty"`
}

// StatusResponse mirrors the Tagger StatusResponse schema.
type StatusResponse struct {
	Jobs []JobStatus `json:"jobs"`
}

// JobStatus mirrors the Tagger JobStatus schema.
type JobStatus struct {
	JobID           string   `json:"job_id"`
	Status          string   `json:"status"`
	TimeRunning     float64  `json:"time_running"`
	TaggingProgress string   `json:"tagging_progress"`
	MissingTags     []string `json:"missing_tags"`
	Failed          []string `json:"failed"`
	Model           string   `json:"model"`
	Stream          string   `json:"stream"`
}

const POLLING_INTERVAL = 5 * time.Second

// -----------------------------------------------------------------------------
// Public handler entrypoint
// -----------------------------------------------------------------------------

func TagContentWorker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args TagContentArgs,
	cfg *config.Config,
) (*mcp.CallToolResult, any, error) {

	// ---------------------- VALIDATION ----------------------
	if strings.TrimSpace(args.QID) == "" {
		return runtime.MCPError(
			errors.E("tag_content", errors.K.Invalid, "reason", "qid is required"),
		)
	}
	if len(args.Jobs) == 0 {
		return runtime.MCPError(
			errors.E("tag_content", errors.K.Invalid, "reason", "at least one job is required"),
		)
	}

	tf, ok := runtime.TenantFromContext(ctx)
	if !ok {
		return runtime.MCPError(
			errors.E("tag_content", errors.K.Permission, "reason", "tenant not found in context"),
		)
	}

	Log.Debug("Start tagging", "Jobs", args.Jobs, "Options", args.Options, "Synchronous", args.Synchronous)

	// ---------------------- STEP 1: START JOBS ----------------------
	startStatuses, err := startTaggerJobs(ctx, args, cfg, tf)
	if err != nil {
		return runtime.MCPError(err)
	}
	Log.Debug("Jobs started", "StartStatuses", startStatuses)

	// ---------------------- STEP 2: FIRST POLL ----------------------
	jobs, done, err := pollStarted(ctx, args.QID, cfg, tf)
	if err != nil {
		return runtime.MCPError(err)
	}
	Log.Debug("Initial poll", "Jobs", jobs, "Done", done)

	// ---------------------- STEP 3: EARLY EXIT ----------------------
	if done {
		Log.Debug("Job already completed after first poll")
		return &mcp.CallToolResult{}, &TagContentSyncResult{Jobs: jobs}, nil
	}

	// ---------------------- SYNC MODE ----------------------
	if args.Synchronous {
		Log.Debug("Entering synchronous polling loop")

		for {
			jobs, done, err := pollTaggerStatus(ctx, args.QID, cfg, tf)
			if err != nil {
				return runtime.MCPError(err)
			}
			if done {
				Log.Debug("Synchronous polling loop completed", "Jobs", jobs)
				return &mcp.CallToolResult{}, &TagContentSyncResult{Jobs: jobs}, nil
			}

			time.Sleep(POLLING_INTERVAL)
		}
	}

	// ---------------------- ASYNC MODE ----------------------
	Log.Debug("Spawning async polling task")

	taskID := async.StartAsyncTask(ctx, func(taskCtx context.Context) (any, error) {
		Log.Debug("Async polling loop started", "QID", args.QID)

		for {
			select {
			case <-taskCtx.Done():
				return nil, taskCtx.Err()
			default:
			}

			jobs, done, err := pollTaggerStatus(taskCtx, args.QID, cfg, tf)
			if err != nil {
				return nil, err
			}
			if done {
				Log.Debug("Async polling loop completed", "Jobs", jobs)
				return jobs, nil
			}

			time.Sleep(POLLING_INTERVAL)
		}
	})

	return &mcp.CallToolResult{}, &TagContentAsyncResult{TaskID: taskID}, nil
}

// pollStarted performs the *initial* status check immediately after starting
// Tagger jobs. It is used by asynchronous mode to validate that the Tagger
// accepted the job request and to detect any immediate terminal states before
// spawning a long‑running async polling task.
//
// This function differs from the normal polling loop in two important ways:
//
//  1. It is called exactly once, synchronously, right after startTaggerJobs.
//     This ensures that the caller receives immediate feedback about:
//     - invalid model requests
//     - permission errors
//     - content not eligible for tagging
//     - jobs that fail instantly
//     - jobs that complete almost instantly
//
//  2. It does *not* loop. It simply returns the first observed job statuses
//     and a boolean indicating whether all jobs are already in a terminal
//     state ("completed", "failed", "stopped", "cancelled", "error",
//     "succeeded").
//
// If `done` is true, the async worker does not need to be started and the
// caller can return a synchronous result immediately.
//
// If `done` is false, the async worker will take over and continue polling
// until the jobs reach a terminal state.
func pollStarted(
	ctx context.Context,
	qid string,
	cfg *config.Config,
	tf *config.TenantFabric,
) ([]TagJobStatus, bool, error) {

	Log.Debug("pollStarted: first status check", "QID", qid)

	jobs, done, err := pollTaggerStatus(ctx, qid, cfg, tf)
	if err != nil {
		Log.Debug("pollStarted: error", "Error", err)
		return nil, false, err
	}

	Log.Debug("pollStarted: result", "Jobs", jobs, "Done", done)
	return jobs, done, nil
}

// -----------------------------------------------------------------------------
// Tagger API calls
// -----------------------------------------------------------------------------

func startTaggerJobs(
	ctx context.Context,
	args TagContentArgs,
	cfg *config.Config,
	tf *config.TenantFabric,
) ([]StartStatus, error) {

	token, err := auth.Auth.FetchEditorSigned(cfg, tf, "", args.QID)
	if err != nil {
		return nil, errors.E("tag_content", errors.K.Permission,
			"reason", "failed to fetch editor-signed token", "error", err)
	}

	// Normalize model names (technical + humanized)
	jobs := make([]TagJobSpec, len(args.Jobs))
	for i, j := range args.Jobs {
		jobs[i] = j
		jobs[i].Model = NormalizeModelName(j.Model)
	}

	reqBody := StartJobsRequest{
		Options: args.Options,
		Jobs:    jobs,
	}

	bodyBytes, err := json.Marshal(reqBody)

	if err != nil {
		return nil, errors.E("tag_content", errors.K.Invalid,
			"reason", "failed to marshal tagger start request", "error", err)
	}

	url := fmt.Sprintf("%s/%s/tag?authorization=%s", strings.TrimRight(cfg.AITaggerUrl, "/"), args.QID, token)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errors.E("tag_content", errors.K.Invalid,
			"reason", "failed to build tagger start request", "error", err)
	}
	// httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	Log.Debug("Start tagger", "Request Body", string(bodyBytes))

	resp, err := http.DefaultClient.Do(httpReq)

	Log.Debug("Start tagger", "Response", resp)

	if err != nil {
		return nil, errors.E("tag_content", errors.K.Unavailable,
			"reason", "tagger start request failed", "error", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.E("tag_content", errors.K.Unavailable,
			"reason", "tagger returned non-200 on start", "status", resp.Status)
	}

	var parsed StartTaggingResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, errors.E("tag_content", errors.K.Invalid,
			"reason", "failed to parse tagger start response", "error", err)
	}

	return parsed.Jobs, nil
}

func pollTaggerStatus(
	ctx context.Context,
	qid string,
	cfg *config.Config,
	tf *config.TenantFabric,
) ([]TagJobStatus, bool, error) {

	token, err := auth.Auth.FetchEditorSigned(cfg, tf, "", qid)
	if err != nil {
		return nil, false, errors.E("tag_content", errors.K.Permission,
			"reason", "failed to fetch editor-signed token", "error", err)
	}

	url := fmt.Sprintf("%s/%s/job-status?authorization=%s", strings.TrimRight(cfg.AITaggerUrl, "/"), qid, token)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, errors.E("tag_content", errors.K.Invalid,
			"reason", "failed to build tagger status request", "error", err)
	}
	// httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)

	if err != nil {
		return nil, false, errors.E("tag_content", errors.K.Unavailable,
			"reason", "tagger status request failed", "error", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, errors.E("tag_content", errors.K.Unavailable,
			"reason", "tagger returned non-200 on status", "status", resp.Status)
	}

	var parsed StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, false, errors.E("tag_content", errors.K.Invalid,
			"reason", "failed to parse tagger status response", "error", err)
	}

	jobs := make([]TagJobStatus, len(parsed.Jobs))
	allDone := true

	for i, j := range parsed.Jobs {
		jobs[i] = TagJobStatus{
			Model:           j.Model,
			Status:          j.Status,
			TimeRunning:     j.TimeRunning,
			TaggingProgress: j.TaggingProgress,
			MissingTags:     j.MissingTags,
			Failed:          j.Failed,
		}
		Log.Debug("Tagger Job Status", "Response", jobs[i])
		// Heuristic: consider jobs done when status is "completed" or "failed".
		if j.Status != "completed" && j.Status != "failed" && j.Status != "stopped" && j.Status != "cancelled" && j.Status != "error" && j.Status != "succeeded" {
			allDone = false
		}
	}

	return jobs, allDone, nil
}
