package auth

import (
	"time"

	"github.com/eluv-io/common-go/format/eat"
	"github.com/eluv-io/errors-go"

	"github.com/qluvio/elv-mcp/types"
)

func FetchStateChannel(cfg *types.Config, token string) (string, error) {
	if token != "" {
		valid, err := validateExp(token)
		if err != nil {
			return "", err
		}
		if !valid {
			return generateStateChannel(cfg)
		}
		return token, nil
	}

	return generateStateChannel(cfg)
}

func FetchEditorSigned(cfg *types.Config, QLibID, QID string) (string, error) {
	if cfg.ESToken != "" {
		valid, err := validateExp(cfg.ESToken)
		if err != nil {
			return "", err
		}
		if !valid {
			return generateEditorSigned(cfg, QLibID, QID)
		}
		return cfg.ESToken, nil
	}
	return generateEditorSigned(cfg, QLibID, QID)
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
