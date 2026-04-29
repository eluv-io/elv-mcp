// tagger_chapters_worker.go
package taggers

import (
	"context"
	"fmt"
	"strings"

	"github.com/eluv-io/errors-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
	async "github.com/qluvio/elv-mcp/tasks/async"
)

// -----------------------------------------------------------------------------
// Public handler entrypoint
// -----------------------------------------------------------------------------

// TagChaptersWorker orchestrates dependency resolution and execution of the
// `chapters` model. It delegates all Tagger start/poll logic to TagContentWorker,
// which is the single source of truth for Tagger execution (sync and async).
//
// High‑level flow:
//
//  1. Validate input and resolve tenant.
//  2. Resolve dependencies for the `chapters` model.
//  3. Build TagContentArgs for the chapters model.
//  4. SYNC MODE:
//     - Call TagContentWorker synchronously.
//     - Extract TagContentSyncResult.
//     - Wrap into ChaptersTaggingSyncResult.
//  5. ASYNC MODE:
//     - Spawn an async task.
//     - Inside the task: resolve dependencies, call TagContentWorker
//     synchronously, and return ChaptersTaggingSyncResult.
//     - Immediate return contains only a task_id.
func TagChaptersWorker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args ChaptersTaggingArgs,
	cfg *config.Config,
) (*mcp.CallToolResult, any, error) {

	// ---------------------- VALIDATION ----------------------
	if strings.TrimSpace(args.QID) == "" {
		return runtime.MCPError(
			errors.E("tag_chapters", errors.K.Invalid, "reason", "qid is required"),
		)
	}

	_, ok := runtime.TenantFromContext(ctx)
	if !ok {
		return runtime.MCPError(
			errors.E("tag_chapters", errors.K.Permission, "reason", "tenant not found in context"),
		)
	}

	// ---------------------- SYNC MODE ----------------------
	if args.Synchronous {
		Log.Debug("TagChapters - synchronous mode", "QID", args.QID, "AutoRunDependencies", args.AutoRunDependencies)

		// 1. Resolve dependencies synchronously
		autoRan, err := resolveChaptersDependencies(ctx, args, cfg, ChaptersModelName)
		if err != nil {
			return runtime.MCPError(err)
		}

		// 2. Build TagContentArgs for chapters model
		tagArgs := TagContentArgs{
			QID:         args.QID,
			Synchronous: true,
			Options:     args.Options,
			Jobs: []TagJobSpec{
				{Model: ChaptersModelName},
			},
		}

		// 3. Call TagContentWorker synchronously
		//
		//    This follows the canonical strategy:
		//      - start jobs
		//      - first poll
		//      - long‑running polling (sync)
		//      - error wrapping
		Log.Debug("TagChapters - starting chapters tagging", "QID", args.QID)

		res2, payload2, err := TagContentWorker(ctx, &mcp.CallToolRequest{}, tagArgs, cfg)
		if err != nil {
			return runtime.MCPError(err)
		}
		if res2 != nil && res2.IsError {
			return runtime.MCPError(
				errors.E("tag_chapters", errors.K.Unavailable, "reason", "TagContentWorker returned error"),
			)
		}

		syncRes, ok := payload2.(*TagContentSyncResult)
		if !ok {
			return runtime.MCPError(
				errors.E("tag_chapters", errors.K.Invalid,
					"reason", fmt.Sprintf("unexpected payload from TagContentWorker: %T", payload2)),
			)
		}

		if autoRan == nil {
			autoRan = []string{}
		}

		// 4. Wrap into ChaptersTaggingSyncResult
		return &mcp.CallToolResult{}, &ChaptersTaggingSyncResult{
			Jobs:                syncRes.Jobs,
			AutoRanDependencies: autoRan,
		}, nil
	}

	// ---------------------- ASYNC MODE ----------------------
	//
	// The async task performs:
	//   - dependency resolution
	//   - synchronous TagContentWorker call
	//   - returns ChaptersTaggingSyncResult
	//
	// The immediate return contains only a task_id.
	Log.Debug("TagChapters - async mode", "QID", args.QID, "AutoRunDependencies", args.AutoRunDependencies)

	taskID := async.StartAsyncTask(ctx, func(taskCtx context.Context) (any, error) {
		Log.Debug("TagChapters - async task started", "QID", args.QID)

		// 1. Resolve dependencies inside async task
		autoRan, err := resolveChaptersDependencies(taskCtx, args, cfg, ChaptersModelName)
		if err != nil {
			return nil, err
		}

		// 2. Build TagContentArgs for chapters model
		tagArgs := TagContentArgs{
			QID:         args.QID,
			Synchronous: true, // async task runs chapters tagging synchronously
			Options:     args.Options,
			Jobs: []TagJobSpec{
				{Model: ChaptersModelName},
			},
		}

		// 3. Call TagContentWorker synchronously inside async task
		Log.Debug("TagChapters - async task starting chapters tagging", "QID", args.QID)

		res2, payload2, err := TagContentWorker(taskCtx, &mcp.CallToolRequest{}, tagArgs, cfg)
		if err != nil {
			return nil, err
		}
		if res2 != nil && res2.IsError {
			return nil, errors.E("tag_chapters", errors.K.Unavailable,
				"reason", "TagContentWorker returned error in async task")
		}

		syncRes, ok := payload2.(*TagContentSyncResult)
		if !ok {
			return nil, errors.E("tag_chapters", errors.K.Invalid,
				"reason", fmt.Sprintf("unexpected payload from TagContentWorker: %T", payload2))
		}

		if autoRan == nil {
			autoRan = []string{}
		}

		// 4. Return final result
		return &ChaptersTaggingSyncResult{
			Jobs:                syncRes.Jobs,
			AutoRanDependencies: autoRan,
		}, nil
	})

	return &mcp.CallToolResult{}, &ChaptersTaggingAsyncResult{TaskID: taskID}, nil
}

// -----------------------------------------------------------------------------
// Dependency resolution for chapters
// -----------------------------------------------------------------------------

// resolveChaptersDependencies ensures that all required dependency models for the
// given target model are fully tagged for the specified QID.
//
// It uses the Tagger tag_status worker to inspect the current model status,
// and optionally auto‑runs missing dependencies via TagContentWorker when
// ChaptersTaggingArgs.AutoRunDependencies is true.
//
// Returns the list of dependency model names that were automatically started
// by this call (AutoRanDependencies), or an error if dependencies are missing
// and auto‑run is not enabled.
func resolveChaptersDependencies(
	ctx context.Context,
	args ChaptersTaggingArgs,
	cfg *config.Config,
	targetModel string,
) ([]string, error) {

	deps := ModelDependencies[targetModel]
	if len(deps) == 0 {
		Log.Debug("TagChapters - no dependencies for model", "Model", targetModel)
		return nil, nil
	}

	Log.Debug("TagChapters - resolving dependencies", "QID", args.QID, "TargetModel", targetModel, "Dependencies", deps)

	// 1. Query tag_status
	res, payload, err := TaggerTagStatusWorker(
		ctx,
		&mcp.CallToolRequest{},
		TaggerTagStatusArgs{QID: args.QID},
		cfg,
	)
	if err != nil {
		return nil, err
	}
	if res != nil && res.IsError {
		return nil, errors.E("tag_chapters", errors.K.Unavailable,
			"reason", "tag_status worker returned error")
	}

	summaries, ok := payload.(TagStatusSummaryResponse)
	if !ok {
		return nil, errors.E("tag_chapters", errors.K.Invalid,
			"reason", fmt.Sprintf("unexpected payload type from tag_status: %T", payload))
	}

	// 2. Determine missing dependencies
	statusByModel := make(map[string]TagStatusSummary, len(summaries.Statuses))
	for _, s := range summaries.Statuses {
		statusByModel[s.Model] = s
	}

	var missing []string
	for _, dep := range deps {
		s, found := statusByModel[dep]
		if !found || s.PercentComplete < 1.0 {
			missing = append(missing, dep)
		}
	}

	var autoRan []string

	if len(missing) == 0 {
		return autoRan, nil
	}

	// 3. Auto‑run disabled → error
	if !args.AutoRunDependencies {
		return nil, errors.E("tag_chapters", errors.K.Invalid,
			"reason", fmt.Sprintf(
				"chapters tagging requires the following models to be fully tagged: %v; "+
					"these dependencies were not satisfied and auto_run_dependencies was not specified",
				missing,
			),
		)
	}

	// 4. Auto‑run dependencies
	

	for _, dep := range missing {
		depArgs := TagContentArgs{
			QID:         args.QID,
			Synchronous: true,
			Options:     args.Options,
			Jobs: []TagJobSpec{
				{Model: dep},
			},
		}

		_, _, err := TagContentWorker(ctx, &mcp.CallToolRequest{}, depArgs, cfg)
		if err != nil {
			return nil, errors.E("tag_chapters", errors.K.Unavailable,
				"reason", fmt.Sprintf("failed to auto‑run dependency model %q", dep),
				"error", err,
			)
		}

		autoRan = append(autoRan, dep)
	}

	return autoRan, nil
}
