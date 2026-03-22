package types

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"gopkg.in/yaml.v2"

	"github.com/eluv-io/common-go/format/types"
	"github.com/eluv-io/errors-go"
)

// yamlConfig mirrors the config.yaml structure.
type yamlConfig struct {
	Server yamlServer `yaml:"server"`
	Fabric yamlFabric `yaml:"fabric"`
	Dev    yamlDev    `yaml:"dev"`
}

type yamlServer struct {
	OAuthIssuer string `yaml:"oauth_issuer"`
	ResourceURL string `yaml:"resource_url"`
}

type yamlFabric struct {
	QLibIndexID  string `yaml:"qlibid_index"`
	QIndexID     string `yaml:"qid_index"`
	SearchIdxUrl string `yaml:"search_base_url"`
	ImgBaseUrl   string `yaml:"image_base_url"`
	VidBaseUrl   string `yaml:"vid_base_url"`
	EthUrl       string `yaml:"eth_url"`
	QSpaceID     string `yaml:"qspace_id"`
}

type yamlDev struct {
	PrivateKey string `yaml:"private_key"`
}

// Config holds all configuration used by the server.
type Config struct {
	// Mu protects the token fields below from concurrent access
	Mu      sync.RWMutex
	ESToken string
	SCToken string

	QLibIndexID  string
	QIndexID     string
	SearchIdxUrl string
	ImgBaseUrl   string
	VidBaseUrl   string
	EthUrl       string
	PkStr        string
	QSpaceID     string
	QlibTest     types.QLibID
	QIDTest      types.QID
	PkStrTest    *ecdsa.PrivateKey

	// OAuth configuration
	OAuthIssuer string // Ory issuer URL (e.g. https://eloquent-carson-yt726m2tf6.projects.oryapis.com)
	ResourceURL string // This server's public URL (for protected resource metadata)
}

// LoadConfig reads config.yaml and returns a shared *Config instance.
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	var yc yamlConfig
	if err := yaml.Unmarshal(data, &yc); err != nil {
		return nil, fmt.Errorf("failed to parse config.yaml: %w", err)
	}

	// Parse Private Key
	pkStr := yc.Dev.PrivateKey
	if strings.HasPrefix(pkStr, "0x") {
		pkStr = pkStr[2:]
	}
	privateKey, err := crypto.HexToECDSA(pkStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dev.private_key: %w", err)
	}

	// Apply defaults for optional OAuth fields
	oauthIssuer := yc.Server.OAuthIssuer
	if oauthIssuer == "" {
		oauthIssuer = "https://confident-dewdney-govmlzzeyi.projects.oryapis.com"
	}
	resourceURL := yc.Server.ResourceURL
	if resourceURL == "" {
		resourceURL = "https://appsvc.svc.eluv.io/mcp"
	}

	cfg := &Config{
		QlibTest:     types.QLibID(yc.Fabric.QLibIndexID),
		QIDTest:      types.QID(yc.Fabric.QIndexID),
		QLibIndexID:  yc.Fabric.QLibIndexID,
		QIndexID:     yc.Fabric.QIndexID,
		SearchIdxUrl: yc.Fabric.SearchIdxUrl,
		ImgBaseUrl:   yc.Fabric.ImgBaseUrl,
		VidBaseUrl:   yc.Fabric.VidBaseUrl,
		EthUrl:       yc.Fabric.EthUrl,
		PkStr:        yc.Dev.PrivateKey,
		PkStrTest:    privateKey,
		QSpaceID:     yc.Fabric.QSpaceID,
		OAuthIssuer:  oauthIssuer,
		ResourceURL:  resourceURL,
	}

	if cfg.QLibIndexID == "" || cfg.QIndexID == "" || cfg.SearchIdxUrl == "" {
		return cfg, errors.E("config", errors.K.Invalid, "reason", "missing required fields in config.yaml")
	}

	return cfg, nil
}

// BoolOrDefault returns the dereferenced bool or a default if nil.
func BoolOrDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func decode(pemEncoded string) *ecdsa.PrivateKey {
	block, _ := pem.Decode([]byte(pemEncoded))
	log.Println(block)
	x509Encoded := block.Bytes
	log.Println(x509Encoded)
	privateKey, _ := x509.ParseECPrivateKey(x509Encoded)
	log.Println(privateKey)

	return privateKey
}
