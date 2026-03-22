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
	"github.com/joho/godotenv"

	"github.com/eluv-io/common-go/format/types"
	"github.com/eluv-io/errors-go"
)

// Config holds all environment-driven configuration used by the server.
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

// LoadConfig returns a POINTER (*Config) so we share the same instance
func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file")
	}

	// 1. Parse Private Key
	pkStr := os.Getenv("PRIVATE_KEY")
	if strings.HasPrefix(pkStr, "0x") {
		pkStr = pkStr[2:]
	}
	privateKey, err := crypto.HexToECDSA(pkStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PRIVATE_KEY: %w", err)
	}

	qlibIndexID := os.Getenv("QLIBID_INDEX")
	qidIndexID := os.Getenv("QID_INDEX")

	// 2. Return Pointer (&Config)
	oauthIssuer := os.Getenv("OAUTH_ISSUER")
	if oauthIssuer == "" {
		oauthIssuer = "https://eloquent-carson-yt726m2tf6.projects.oryapis.com"
	}
	resourceURL := os.Getenv("RESOURCE_URL")
	if resourceURL == "" {
		resourceURL = "https://localhost:8080"
	}

	cfg := &Config{
		QlibTest:     types.QLibID(qlibIndexID),
		QIDTest:      types.QID(qidIndexID),
		QLibIndexID:  qlibIndexID,
		QIndexID:     qidIndexID,
		SearchIdxUrl: os.Getenv("SEARCH_BASE_URL"),
		ImgBaseUrl:   os.Getenv("IMAGE_BASE_URL"),
		VidBaseUrl:   os.Getenv("VID_BASE_URL"),
		EthUrl:       os.Getenv("ETH_URL"),
		PkStr:        os.Getenv("PRIVATE_KEY"),
		PkStrTest:    privateKey,
		QSpaceID:     os.Getenv("QSPACE_ID"),
		OAuthIssuer: oauthIssuer,
		ResourceURL: resourceURL,
	}

	if cfg.QLibIndexID == "" || cfg.QIndexID == "" || cfg.SearchIdxUrl == "" {
		return cfg, errors.E("config", errors.K.Invalid, "reason", "missing env variables")
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
