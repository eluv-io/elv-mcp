package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	yaml := `
log:
  level: info
  formatter: text

server:
  oauth_issuer: "https://issuer.example"
  resource_url: "https://resource.example"

fabric:
  search_base_url: "http://search"
  image_base_url: "http://img"
  vid_base_url: "http://vid"
  eth_url: "http://eth"
  qspace_id: "qs"

tenants:
  - id: test-tenant
    users:
      - "user|abc123"
    fabric:
      private_key: "0x1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd"
      qlibid_index: "qlib123"
      qid_index: "qid123"
      search_collection_id: "qid123"
`

	if err := os.WriteFile("config.yaml", []byte(yaml), 0o644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}
	defer os.Remove("config.yaml")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.SearchIdxUrl != "http://search" {
		t.Fatalf("unexpected SearchIdxUrl: %s", cfg.SearchIdxUrl)
	}

	// Tenant lookup and private key parsed?
	tenant, ok := cfg.Tenants.Lookup("user|abc123")
	if !ok {
		t.Fatalf("expected to find tenant for user|abc123")
	}
	if tenant.Fabric.QLibIndexID != "qlib123" {
		t.Fatalf("unexpected QLibIndexID: %s", tenant.Fabric.QLibIndexID)
	}
	if tenant.Fabric.QIndexID != "qid123" {
		t.Fatalf("unexpected QIndexID: %s", tenant.Fabric.QIndexID)
	}
	if tenant.Fabric.SearchCollectionID != "qid123" {
		t.Fatalf("unexpected SearchCollectionID: %s", tenant.Fabric.SearchCollectionID)
	}
	if tenant.Fabric.PrivateKey == nil {
		t.Fatalf("expected PrivateKey to be parsed")
	}

	if cfg.OAuthIssuer != "https://issuer.example" {
		t.Fatalf("unexpected OAuthIssuer: %s", cfg.OAuthIssuer)
	}
	if cfg.ResourceURL != "https://resource.example" {
		t.Fatalf("unexpected ResourceURL: %s", cfg.ResourceURL)
	}
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "missing search_base_url",
			yaml: `
server:
  oauth_issuer: "https://issuer.example"
  resource_url: "https://resource.example"
fabric:
  search_base_url: ""
tenants:
  - id: t
    users: ["user|abc"]
    fabric:
      private_key: "0x1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd"
      qlibid_index: "qlib"
      qid_index: "qid"
`,
		},
		{
			name: "missing oauth_issuer",
			yaml: `
server:
  resource_url: "https://resource.example"
fabric:
  search_base_url: "http://search"
`,
		},
		{
			name: "missing resource_url",
			yaml: `
server:
  oauth_issuer: "https://issuer.example"
fabric:
  search_base_url: "http://search"
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile("config.yaml", []byte(tc.yaml), 0o644); err != nil {
				t.Fatalf("failed to write config.yaml: %v", err)
			}
			defer os.Remove("config.yaml")

			_, err := LoadConfig()
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}
