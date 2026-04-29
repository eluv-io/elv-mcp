//go:build integration
// +build integration

package integration

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/tasks/tagstore"
)

// -----------------------------------------------------------------------------
// Test constants (from base.go)
// -----------------------------------------------------------------------------
//
// const TagstoreTestTenant = "urc-content-ops"
// const IntegrationTestQID = "iq__47cbSU6ygSF5Zaoc6RfCyS4E1Ppr"
//
// These are imported automatically because this file shares the same package.
// -----------------------------------------------------------------------------

// Unique track names for integration tests.
const (
	testTrackCreateDelete = "integration_test_track_create_delete"
	testTrackDuplicate    = "integration_test_track_duplicate"
)

// -----------------------------------------------------------------------------
// Test 1 — Create a track, then delete it (with verified delete)
// -----------------------------------------------------------------------------

func TestTagStore_CreateThenDeleteTrack(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Cleanup guard
	deleted := false

	// Cleanup fallback — only runs if verified delete did NOT run
	defer func() {
		if deleted {
			return
		}
		tagstore.TagStoreDeleteTrackWorker(
			ctx,
			&mcp.CallToolRequest{},
			tagstore.TagStoreDeleteTrackArgs{
				QID:   IntegrationTestQID,
				Track: testTrackCreateDelete,
			},
			cfg,
		)
	}()

	// ---------------------- CREATE ----------------------
	createArgs := tagstore.TagStoreCreateTrackArgs{
		QID:   IntegrationTestQID,
		Track: testTrackCreateDelete,
	}

	res, payload, err := tagstore.TagStoreCreateTrackWorker(
		ctx,
		&mcp.CallToolRequest{},
		createArgs,
		cfg,
	)
	if err != nil {
		t.Fatalf("unexpected error creating track: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error on create: %+v", res)
	}

	out, ok := payload.(*tagstore.TagStoreCreateTrackResult)
	if !ok {
		t.Fatalf("expected TagStoreCreateTrackResult, got %T", payload)
	}
	if !out.Created {
		t.Fatalf("expected Created=true, got false")
	}
	if out.Track != testTrackCreateDelete {
		t.Fatalf("unexpected track name: %s", out.Track)
	}

	// ---------------------- DELETE (verified) ----------------------
	deleteArgs := tagstore.TagStoreDeleteTrackArgs{
		QID:   IntegrationTestQID,
		Track: testTrackCreateDelete,
	}

	delRes, delPayload, delErr := tagstore.TagStoreDeleteTrackWorker(
		ctx,
		&mcp.CallToolRequest{},
		deleteArgs,
		cfg,
	)
	if delErr != nil {
		t.Fatalf("unexpected error deleting track: %v", delErr)
	}
	if delRes.IsError {
		t.Fatalf("tool returned error on delete: %+v", delRes)
	}

	delOut, ok := delPayload.(*tagstore.TagStoreDeleteTrackResult)
	if !ok {
		t.Fatalf("expected TagStoreDeleteTrackResult, got %T", delPayload)
	}
	if !delOut.Deleted {
		t.Fatalf("expected Deleted=true, got false")
	}

	// Mark cleanup as done
	deleted = true
}

// -----------------------------------------------------------------------------
// Test 2 — Create a track twice → second creation must fail with Conflict
// -----------------------------------------------------------------------------

func TestTagStore_CreateDuplicateTrack(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

	// Cleanup guard
	deleted := false

	// Cleanup fallback
	defer func() {
		if deleted {
			return
		}
		tagstore.TagStoreDeleteTrackWorker(
			ctx,
			&mcp.CallToolRequest{},
			tagstore.TagStoreDeleteTrackArgs{
				QID:   IntegrationTestQID,
				Track: testTrackDuplicate,
			},
			cfg,
		)
	}()

	// ---------------------- FIRST CREATE ----------------------
	firstArgs := tagstore.TagStoreCreateTrackArgs{
		QID:   IntegrationTestQID,
		Track: testTrackDuplicate,
	}

	res, payload, err := tagstore.TagStoreCreateTrackWorker(
		ctx,
		&mcp.CallToolRequest{},
		firstArgs,
		cfg,
	)
	if err != nil {
		t.Fatalf("unexpected error creating track: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error on first create: %+v", res)
	}

	out, ok := payload.(*tagstore.TagStoreCreateTrackResult)
	if !ok {
		t.Fatalf("expected TagStoreCreateTrackResult, got %T", payload)
	}
	if !out.Created {
		t.Fatalf("expected Created=true on first create")
	}

	// ---------------------- SECOND CREATE (should fail) ----------------------
	secondArgs := tagstore.TagStoreCreateTrackArgs{
		QID:   IntegrationTestQID,
		Track: testTrackDuplicate,
	}

	res2, payload2, err2 := tagstore.TagStoreCreateTrackWorker(
		ctx,
		&mcp.CallToolRequest{},
		secondArgs,
		cfg,
	)

	if err2 == nil {
		t.Fatalf("expected conflict error on duplicate create, got nil")
	}
	if res2 == nil || !res2.IsError {
		t.Fatalf("expected MCP error result on duplicate create")
	}
	if payload2 != nil {
		t.Fatalf("payload must be nil on duplicate create")
	}

	// ---------------------- DELETE (verified) ----------------------
	deleteArgs := tagstore.TagStoreDeleteTrackArgs{
		QID:   IntegrationTestQID,
		Track: testTrackDuplicate,
	}

	delRes, delPayload, delErr := tagstore.TagStoreDeleteTrackWorker(
		ctx,
		&mcp.CallToolRequest{},
		deleteArgs,
		cfg,
	)
	if delErr != nil {
		t.Fatalf("unexpected error deleting track: %v", delErr)
	}
	if delRes.IsError {
		t.Fatalf("tool returned error on delete: %+v", delRes)
	}

	delOut, ok := delPayload.(*tagstore.TagStoreDeleteTrackResult)
	if !ok {
		t.Fatalf("expected TagStoreDeleteTrackResult, got %T", delPayload)
	}
	if !delOut.Deleted {
		t.Fatalf("expected Deleted=true, got false")
	}

	deleted = true
}

// -----------------------------------------------------------------------------
// Test 3 — List tracks (after creating one)
// -----------------------------------------------------------------------------
func TestTagStore_ListTracks(t *testing.T) {
    cfg := loadIntegrationConfig(t)
    ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

    const trackName = "integration_test_list_tracks"

    // Cleanup guard
    deleted := false

    // Cleanup fallback
    defer func() {
        if deleted {
            return
        }
        tagstore.TagStoreDeleteTrackWorker(
            ctx,
            &mcp.CallToolRequest{},
            tagstore.TagStoreDeleteTrackArgs{
                QID:   IntegrationTestQID,
                Track: trackName,
            },
            cfg,
        )
    }()

    // ---------------------- CREATE ----------------------
    createArgs := tagstore.TagStoreCreateTrackArgs{
        QID:   IntegrationTestQID,
        Track: trackName,
    }

    res, payload, err := tagstore.TagStoreCreateTrackWorker(
        ctx,
        &mcp.CallToolRequest{},
        createArgs,
        cfg,
    )
    if err != nil {
        t.Fatalf("unexpected error creating track: %v", err)
    }
    if res.IsError {
        t.Fatalf("tool returned error on create: %+v", res)
    }

    out, ok := payload.(*tagstore.TagStoreCreateTrackResult)
    if !ok {
        t.Fatalf("expected TagStoreCreateTrackResult, got %T", payload)
    }
    if !out.Created {
        t.Fatalf("expected Created=true, got false")
    }

    // ---------------------- LIST ----------------------
    listArgs := tagstore.TagStoreListTracksArgs{
        QID: IntegrationTestQID,
    }

    listRes, listPayload, listErr := tagstore.TagStoreListTracksWorker(
        ctx,
        &mcp.CallToolRequest{},
        listArgs,
        cfg,
    )
    if listErr != nil {
        t.Fatalf("unexpected error listing tracks: %v", listErr)
    }
    if listRes.IsError {
        t.Fatalf("tool returned error on list: %+v", listRes)
    }

    listOut, ok := listPayload.(*tagstore.TagStoreListTracksResult)
    if !ok {
        t.Fatalf("expected TagStoreListTracksResult, got %T", listPayload)
    }

    found := false
    for _, tr := range listOut.Tracks {
        if tr.Name == trackName {
            found = true
            if tr.QID == "" || tr.ID == "" {
                t.Fatalf("expected full track metadata, got %+v", tr)
            }
            break
        }
    }
    if !found {
        t.Fatalf("expected track %q to appear in list, got %+v", trackName, listOut.Tracks)
    }

    // ---------------------- DELETE (verified) ----------------------
    deleteArgs := tagstore.TagStoreDeleteTrackArgs{
        QID:   IntegrationTestQID,
        Track: trackName,
    }

    delRes, delPayload, delErr := tagstore.TagStoreDeleteTrackWorker(
        ctx,
        &mcp.CallToolRequest{},
        deleteArgs,
        cfg,
    )
    if delErr != nil {
        t.Fatalf("unexpected error deleting track: %v", delErr)
    }
    if delRes.IsError {
        t.Fatalf("tool returned error on delete: %+v", delRes)
    }

    delOut, ok := delPayload.(*tagstore.TagStoreDeleteTrackResult)
    if !ok {
        t.Fatalf("expected TagStoreDeleteTrackResult, got %T", delPayload)
    }
    if !delOut.Deleted {
        t.Fatalf("expected Deleted=true, got false")
    }

    deleted = true
}


func TestTagStore_ListExistingTracks(t *testing.T) {
    cfg := loadIntegrationConfig(t)
    ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

    const trackName = "integration_test_list_tracks"

    // ---------------------- LIST ----------------------
    listArgs := tagstore.TagStoreListTracksArgs{
        QID: IntegrationTestQID,
    }

    listRes, listPayload, listErr := tagstore.TagStoreListTracksWorker(
        ctx,
        &mcp.CallToolRequest{},
        listArgs,
        cfg,
    )
    if listErr != nil {
        t.Fatalf("unexpected error listing tracks: %v", listErr)
    }
    if listRes.IsError {
        t.Fatalf("tool returned error on list: %+v", listRes)
    }

    listOut, ok := listPayload.(*tagstore.TagStoreListTracksResult)
    if !ok {
        t.Fatalf("expected TagStoreListTracksResult, got %T", listPayload)
    }

	println("Existing tracks:", len(listOut.Tracks))
	println("Track names:")
	for _, tr := range listOut.Tracks {
		println("- " + tr.Name)
	}    
}