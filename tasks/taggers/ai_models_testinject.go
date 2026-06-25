package taggers

import "sync"

// This file exists solely to support unit testing of the model‑loading logic.
// The production code loads model metadata from the Tagger /models API, but
// several tests need to validate behavior that depends on specific model graphs
// (model → track → model dependencies). Mocking the entire HTTP endpoint in
// every test would be noisy, repetitive, and brittle.
//
// Instead, tests can inject a controlled JSON payload that represents the
// /models response. This allows the tests to exercise the real parsing and
// dependency‑resolution logic without performing network calls or duplicating
// implementation details.
//
// The build tag ensures this code is never included in production builds.

// TestInjectModelsJSON replaces the model metadata normally fetched from the
// Tagger API with a test‑provided JSON payload. It also resets all internal
// caches so each test starts from a clean state.
func TestInjectModelsJSON(data []byte) {
    testInjectedModelsJSON = data

    // Reset all cached state so the loader runs again.
    cachedModels = nil
    cachedDependencies = nil
    cachedAliases = nil
    cachedTrackToModel = nil

    loadOnce = sync.Once{}
    loadErr = nil
}
