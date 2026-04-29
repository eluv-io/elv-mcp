// tagger_character_worker.go
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

// TagCharactersWorker orchestrates dependency resolution and execution of the
// `character` model. It delegates all Tagger start/poll logic to
// TagContentWorker, which is now the single source of truth for Tagger
// execution (sync and async).
//
// High‑level flow:
//
//  1. Validate input and resolve tenant.
//  2. Resolve dependencies for the `character` model.
//  3. Build TagContentArgs for the character model.
//  4. SYNC MODE:
//     - Call TagContentWorker synchronously.
//     - Extract TagContentSyncResult.
//     - Wrap into CharacterTaggingSyncResult.
//  5. ASYNC MODE:
//     - Spawn an async task.
//     - Inside the task: resolve dependencies, call TagContentWorker
//     synchronously, and return CharacterTaggingSyncResult.
//     - Immediate return contains only a task_id.
func TagCharactersWorker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args CharacterTaggingArgs,
	cfg *config.Config,
) (*mcp.CallToolResult, any, error) {

	// ---------------------- VALIDATION ----------------------
	if strings.TrimSpace(args.QID) == "" {
		return runtime.MCPError(
			errors.E("tag_characters", errors.K.Invalid, "reason", "qid is required"),
		)
	}

	_, ok := runtime.TenantFromContext(ctx)
	if !ok {
		return runtime.MCPError(
			errors.E("tag_characters", errors.K.Permission, "reason", "tenant not found in context"),
		)
	}

	// ---------------------- SYNC MODE ----------------------
	if args.Synchronous {
		Log.Debug("TagCharacters - synchronous mode", "QID", args.QID, "AutoRunDependencies", args.AutoRunDependencies)

		// 1. Resolve dependencies synchronously
		autoRan, err := resolveModelDependencies(ctx, args, cfg, CharacterModelName)
		if err != nil {
			return runtime.MCPError(err)
		}

		// 2. Build TagContentArgs for character model
		tagArgs := TagContentArgs{
			QID:         args.QID,
			Synchronous: true,
			Options:     args.Options,
			Jobs: []TagJobSpec{
				{Model: CharacterModelName},
			},
		}

		// 3. Call TagContentWorker synchronously
		//
		//    This is the *only* correct way to run Tagger jobs now that
		//    runTaggerSync has been removed. TagContentWorker handles:
		//      - starting jobs
		//      - first poll
		//      - long‑running polling
		//      - error wrapping
		Log.Debug("TagCharacters - starting character tagging", "QID", args.QID)

		res2, payload2, err := TagContentWorker(ctx, &mcp.CallToolRequest{}, tagArgs, cfg)
		if err != nil {
			return runtime.MCPError(err)
		}
		if res2 != nil && res2.IsError {
			return runtime.MCPError(
				errors.E("tag_characters", errors.K.Unavailable, "reason", "TagContentWorker returned error"),
			)
		}

		syncRes, ok := payload2.(*TagContentSyncResult)
		if !ok {
			return runtime.MCPError(
				errors.E("tag_characters", errors.K.Invalid,
					"reason", fmt.Sprintf("unexpected payload from TagContentWorker: %T", payload2)),
			)
		}

		// 4. Wrap into CharacterTaggingSyncResult
		return &mcp.CallToolResult{}, &CharacterTaggingSyncResult{
			Jobs:                syncRes.Jobs,
			AutoRanDependencies: autoRan,
		}, nil
	}

	// ---------------------- ASYNC MODE ----------------------
	//
	// The async task performs:
	//   - dependency resolution
	//   - synchronous TagContentWorker call
	//   - returns CharacterTaggingSyncResult
	//
	// The immediate return contains only a task_id.
	Log.Debug("TagCharacters - async mode", "QID", args.QID, "AutoRunDependencies", args.AutoRunDependencies)

	taskID := async.StartAsyncTask(ctx, func(taskCtx context.Context) (any, error) {
		Log.Debug("TagCharacters - async task started", "QID", args.QID)

		// 1. Resolve dependencies inside async task
		autoRan, err := resolveModelDependencies(taskCtx, args, cfg, CharacterModelName)
		if err != nil {
			return nil, err
		}

		// 2. Build TagContentArgs for character model
		tagArgs := TagContentArgs{
			QID:         args.QID,
			Synchronous: true, // async task runs character tagging synchronously
			Options:     args.Options,
			Jobs: []TagJobSpec{
				{Model: CharacterModelName},
			},
		}

		// 3. Call TagContentWorker synchronously inside async task
		Log.Debug("TagCharacters - async task starting character tagging", "QID", args.QID)

		res2, payload2, err := TagContentWorker(taskCtx, &mcp.CallToolRequest{}, tagArgs, cfg)
		if err != nil {
			return nil, err
		}
		if res2 != nil && res2.IsError {
			return nil, errors.E("tag_characters", errors.K.Unavailable,
				"reason", "TagContentWorker returned error in async task")
		}

		syncRes, ok := payload2.(*TagContentSyncResult)
		if !ok {
			return nil, errors.E("tag_characters", errors.K.Invalid,
				"reason", fmt.Sprintf("unexpected payload from TagContentWorker: %T", payload2))
		}

		// 4. Return final result
		return &CharacterTaggingSyncResult{
			Jobs:                syncRes.Jobs,
			AutoRanDependencies: autoRan,
		}, nil
	})

	return &mcp.CallToolResult{}, &CharacterTaggingAsyncResult{TaskID: taskID}, nil
}

// -----------------------------------------------------------------------------
// Dependency resolution
// -----------------------------------------------------------------------------

func resolveModelDependencies(
	ctx context.Context,
	args CharacterTaggingArgs,
	cfg *config.Config,
	targetModel string,
) ([]string, error) {

	deps := ModelDependencies[targetModel]
	if len(deps) == 0 {
		Log.Debug("TagCharacters - no dependencies for model", "Model", targetModel)
		return nil, nil
	}

	Log.Debug("TagCharacters - resolving dependencies", "QID", args.QID, "TargetModel", targetModel, "Dependencies", deps)

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
		return nil, errors.E("tag_characters", errors.K.Unavailable,
			"reason", "tag_status worker returned error")
	}

	summaries, ok := payload.(TagStatusSummaryResponse)
	if !ok {
		return nil, errors.E("tag_characters", errors.K.Invalid,
			"reason", fmt.Sprintf("unexpected payload type from tag_status: %T", payload))
	}

	statuses := summaries.Statuses // tag_status response is wrapped in an extra struct; unwrap it

	// 2. Determine missing dependencies
	statusByModel := make(map[string]TagStatusSummary, len(statuses))
	for _, s := range statuses {
		statusByModel[s.Model] = s
	}

	var missing []string
	for _, dep := range deps {
		s, found := statusByModel[dep]
		if !found || s.PercentComplete < 1.0 {
			missing = append(missing, dep)
		}
	}

	if len(missing) == 0 {
		return nil, nil
	}

	// 3. Auto‑run disabled → error
	if !args.AutoRunDependencies {
		return nil, errors.E("tag_characters", errors.K.Invalid,
			"reason", fmt.Sprintf(
				"character tagging requires the following models to be fully tagged: %v; "+
					"these dependencies were not satisfied and auto_run_dependencies was not specified",
				missing,
			),
		)
	}

	// 4. Auto‑run dependencies
	var autoRan []string

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
			return nil, errors.E("tag_characters", errors.K.Unavailable,
				"reason", fmt.Sprintf("failed to auto‑run dependency model %q", dep),
				"error", err,
			)
		}

		autoRan = append(autoRan, dep)
	}

	return autoRan, nil
}
