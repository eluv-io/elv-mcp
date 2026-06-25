package auth

import (
	"context"
	"time"

	"github.com/eluv-io/common-go/format/eat"
	"github.com/eluv-io/errors-go"
	elvclient "github.com/qluvio/elv-client-go"
	"github.com/qluvio/elv-mcp/config"
)

// -----------------------------------------------------------------------------
// Token Fetching Logic (unchanged)
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// Real Provider Wiring
// -----------------------------------------------------------------------------

type realAuthProvider struct{}

func (realAuthProvider) FetchStateChannel(cfg *config.Config, tf *config.TenantFabric) (string, error) {
	return FetchStateChannel(cfg, tf)
}

func (realAuthProvider) FetchEditorSigned(cfg *config.Config, tf *config.TenantFabric, qlibID, qid string) (string, error) {
	return FetchEditorSigned(cfg, tf, qlibID, qid)
}

func (realAuthProvider) GetQLibId(cfg *config.Config, tf *config.TenantFabric, QID string) (string, string, error) {
	return GetQLibId(cfg, tf, QID)
}

func init() {
	Auth = realAuthProvider{}
}

// -----------------------------------------------------------------------------
// Token Fetching Logic (unchanged)
// -----------------------------------------------------------------------------

// FetchStateChannel returns a valid state channel token for the given tenant,
// using the cached value if it still has more than 12 hours of life remaining.
func FetchStateChannel(cfg *config.Config, tf *config.TenantFabric) (string, error) {
	tf.Mu.RLock()
	tok := tf.SCToken
	tf.Mu.RUnlock()

    if tok != "" {
        valid, err := validateExp(tok)
        if err != nil {
            return "", err
        }
        if valid {
            return tok, nil
        }
    }

    newTok, err := generateStateChannel(cfg, tf)
    if err != nil {
        return "", err
    }

    tf.Mu.Lock()
    tf.SCToken = newTok
    tf.Mu.Unlock()

    return newTok, nil
}

// FetchEditorSigned returns a valid editor-signed token for the given tenant,
// using the cached value if it still has more than 12 hours of life remaining.
// If QlibID is empty, it will be resolved automatically by querying the Fabric
// metadata API using a state channel token.
func FetchEditorSigned(cfg *config.Config, tf *config.TenantFabric, QLibID, QID string) (string, error) {
	tf.Mu.RLock()
	tok := tf.ESToken
	tf.Mu.RUnlock()

	if tok != "" {
		valid, err := validateExp(tok)
		if err != nil {
			return "", err
		}
		if valid {
			return tok, nil
		}
	}

	// If qlibID is not provided, resolve it dynamically.
	if QLibID == "" {
		qlibId, s, err := GetQLibId(cfg, tf, QID)
		if err != nil {
			return s, err
		}
		QLibID = qlibId
	}

	newTok, err := generateEditorSigned(cfg, tf, QLibID, QID)
	if err != nil {
		return "", err
	}

    tf.Mu.Lock()
    tf.ESToken = newTok
    tf.Mu.Unlock()

    return newTok, nil
}

func GetQLibId(cfg *config.Config, tf *config.TenantFabric, QID string) (string, string, error) {
	scTok, err := FetchStateChannel(cfg, tf)
	if err != nil {
		return "", "", errors.E("fetch editor-signed token", errors.K.Unavailable,
			"reason", "failed to fetch state channel token", "error", err)
	}

	client, err := elvclient.NewElvClientFromConfigURL(cfg.ApiUrl+"/config", scTok)
	if err != nil {
		return "", "", errors.E("fetch editor-signed token", errors.K.Unavailable,
			"reason", "failed to create elv client", "error", err)
	}

	qlibId, err := client.ContentObjectLibraryID(context.Background(), QID)
	if err != nil {
		return "", "", errors.E("fetch editor-signed token", errors.K.Unavailable,
			"reason", "failed to resolve qlib_id", "error", err)
	}
	return qlibId, "", nil
}

func validateExp(token string) (bool, error) {
    tk, err := eat.Parse(token)
    if err != nil {
        return false, errors.E("validate expiration", errors.K.Unavailable, "error", err)
    }

    if tk.Expires.UnixMilli()-time.Now().UTC().UnixMilli() > 720*time.Minute.Milliseconds() {
        return true, nil
    }
    return false, nil
}
