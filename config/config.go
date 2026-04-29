package config

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"

	elog "github.com/eluv-io/log-go"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/eluv-io/errors-go"
)

var log = elog.Get("/types")

const defaultFabricApiUrl = "https://main.net955305.contentfabric.io"
const defaultAITaggerUrl = "https://ai.contentfabric.io/tagging-live"
const defaultTagstoreUrl = "https://ai.contentfabric.io/tagstore"

// TenantFabric holds per-tenant fabric credentials and cached tokens.
type TenantFabric struct {
	PkStr       string
	QLibIndexID string
	QIndexID    string
	SearchCollectionID string

	// Pre-parsed at load time
	PrivateKey *ecdsa.PrivateKey

	// Cached tokens (protected by Mu)
	Mu      sync.RWMutex // Mu protects the token fields below from concurrent access
	SCToken string
	ESToken string
}

// Tenant is a named group of users sharing fabric credentials.
type Tenant struct {
	ID     string
	Fabric *TenantFabric
}

// TenantRegistry maps OAuth subject IDs (sub claim) to their Tenant.
type TenantRegistry struct {
	byUser map[string]*Tenant
}

// Lookup returns the Tenant for the given OAuth subject, or false if unknown.
func (r *TenantRegistry) Lookup(sub string) (*Tenant, bool) {
	t, ok := r.byUser[sub]
	return t, ok
}

// yamlConfig mirrors the config.yaml structure.
type yamlConfig struct {
	Log     *yamlLogConfig `yaml:"log"`
	Server  yamlServer     `yaml:"server"`
	Fabric  yamlFabric     `yaml:"fabric"`
	Tenants []yamlTenant   `yaml:"tenants"`
}

// yamlLogConfig maps the log section with proper yaml tags.
// The elog.Config struct uses json tags which yaml.v2 doesn't read,
// so we mirror it here with yaml tags and convert.
type yamlLogConfig struct {
	Level     string                    `yaml:"level"`
	Formatter string                    `yaml:"formatter"`
	File      *elog.LumberjackConfig    `yaml:"file"`
	Named     map[string]*yamlLogConfig `yaml:"named"`
}

func (y *yamlLogConfig) toLogConfig() *elog.Config {
	if y == nil {
		return nil
	}
	c := &elog.Config{
		Level:   y.Level,
		Handler: y.Formatter,
		File:    y.File,
	}
	if len(y.Named) > 0 {
		c.Named = make(map[string]*elog.Config, len(y.Named))
		for k, v := range y.Named {
			c.Named[k] = v.toLogConfig()
		}
	}
	return c
}

type yamlServer struct {
	OAuthIssuer string `yaml:"oauth_issuer"`
	ResourceURL string `yaml:"resource_url"`
}

// yamlFabric holds shared infrastructure URLs (not per-tenant).
type yamlFabric struct {
	SearchIdxUrl string `yaml:"search_base_url"`
	ImgBaseUrl   string `yaml:"image_base_url"`
	VidBaseUrl   string `yaml:"vid_base_url"`
	EthUrl       string `yaml:"eth_url"`
	QSpaceID     string `yaml:"qspace_id"`
	ApiUrl       string `yaml:"api_url"`       // optional, defaults to https://main.net955305.contentfabric.io
	AITaggerURL  string `yaml:"ai_tagger_url"` // optional, defaults to https://ai.contentfabric.io/tagging-live
	TagStoreUrl  string `yaml:"tagstore_url"`  // optional, defaults to https://ai.contentfabric.io/tagstore
}

type yamlTenantFabric struct {
	PrivateKey  string `yaml:"private_key"`
	QLibIndexID string `yaml:"qlibid_index"`
	QIndexID    string `yaml:"qid_index"`
	SearchCollectionID string `yaml:"search_collection_id"`
}

type yamlTenant struct {
	ID     string           `yaml:"id"`
	Users  []string         `yaml:"users"`
	Fabric yamlTenantFabric `yaml:"fabric"`
}

// Config holds all server-wide configuration. Per-tenant settings live in Tenants.
type Config struct {
	// Shared fabric infrastructure
	SearchIdxUrl string
	ImgBaseUrl   string
	VidBaseUrl   string
	EthUrl       string
	QSpaceID     string
	ApiUrl       string
	AITaggerUrl  string
	TagStoreUrl  string

	// OAuth configuration
	OAuthIssuer string // Ory issuer URL (e.g. https://eloquent-carson-yt726m2tf6.projects.oryapis.com)
	ResourceURL string // This server's public URL (for protected resource metadata)

	// Tenant registry (user → tenant config)
	Tenants *TenantRegistry
}

// LoadConfig reads config.yaml and returns a shared *Config instance.
func LoadConfig() (*Config, error) {
	return LoadConfigWithPath("config.yaml")
}

// This should be used for integration tests, pointing to a real config file
// outside the git module
func LoadConfigWithPath(config_path string) (*Config, error) {
	data, err := os.ReadFile(config_path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	var yc yamlConfig
	if err := yaml.Unmarshal(data, &yc); err != nil {
		return nil, fmt.Errorf("failed to parse config.yaml: %w", err)
	}

	if yc.Server.OAuthIssuer == "" {
		return nil, errors.E("config", errors.K.Invalid, "reason", "server.oauth_issuer is required")
	}
	if yc.Server.ResourceURL == "" {
		return nil, errors.E("config", errors.K.Invalid, "reason", "server.resource_url is required")
	}

	if yc.Fabric.SearchIdxUrl == "" {
		return nil, errors.E("config", errors.K.Invalid, "reason", "fabric.search_base_url is required")
	}

	// Build tenant registry
	registry, err := buildTenantRegistry(yc.Tenants)
	if err != nil {
		return nil, fmt.Errorf("failed to build tenant registry: %w", err)
	}

	// Configure logging
	if logCfg := yc.Log.toLogConfig(); logCfg != nil {
		elog.SetDefault(logCfg)
	}

	// Default Fabric API URL if not provided
	apiURL := yc.Fabric.ApiUrl
	if apiURL == "" {
		apiURL = defaultFabricApiUrl
	}

	aiTaggerURL := yc.Fabric.AITaggerURL
	if aiTaggerURL == "" {
		aiTaggerURL = defaultAITaggerUrl
	}

	tagStoreURL := yc.Fabric.TagStoreUrl
	if tagStoreURL == "" {
		tagStoreURL = defaultTagstoreUrl
	}

	return &Config{
		SearchIdxUrl: yc.Fabric.SearchIdxUrl,
		ImgBaseUrl:   yc.Fabric.ImgBaseUrl,
		VidBaseUrl:   yc.Fabric.VidBaseUrl,
		EthUrl:       yc.Fabric.EthUrl,
		QSpaceID:     yc.Fabric.QSpaceID,
		OAuthIssuer:  yc.Server.OAuthIssuer,
		ResourceURL:  yc.Server.ResourceURL,
		ApiUrl:       apiURL,
		AITaggerUrl:  aiTaggerURL,
		TagStoreUrl:  tagStoreURL,
		Tenants:      registry,
	}, nil
}

func buildTenantRegistry(tenants []yamlTenant) (*TenantRegistry, error) {
	r := &TenantRegistry{byUser: make(map[string]*Tenant)}
	for _, yt := range tenants {
		if yt.ID == "" {
			return nil, fmt.Errorf("tenant is missing an id")
		}

		// Parse Private Key
		pkStr := yt.Fabric.PrivateKey
		rawKey := strings.TrimPrefix(pkStr, "0x")
		pk, err := crypto.HexToECDSA(rawKey)
		if err != nil {
			return nil, fmt.Errorf("tenant %q: failed to parse private_key: %w", yt.ID, err)
		}
		if yt.Fabric.QLibIndexID == "" || yt.Fabric.QIndexID == "" {
			return nil, fmt.Errorf("tenant %q: qlibid_index and qid_index are required", yt.ID)
		}

		tf := &TenantFabric{
			PkStr:       pkStr,
			QLibIndexID: yt.Fabric.QLibIndexID,
			QIndexID:    yt.Fabric.QIndexID,
			PrivateKey:  pk,
			SearchCollectionID: yt.Fabric.SearchCollectionID,
		}
		tenant := &Tenant{ID: yt.ID, Fabric: tf}
		for _, raw := range yt.Users {
			sub := strings.TrimSpace(raw)
			if sub == "" {
				log.Warn("tenant config contains empty user sub, skipping", "tenant", yt.ID)
				continue
			}
			if _, exists := r.byUser[sub]; exists {
				return nil, fmt.Errorf("user %q is listed in multiple tenants or multiple times", sub)
			}
			r.byUser[sub] = tenant
		}
	}
	return r, nil
}

// BoolOrDefault returns the dereferenced bool or a default if nil.
func BoolOrDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}
