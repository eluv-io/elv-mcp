package auth

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	elog "github.com/eluv-io/log-go"
	"github.com/golang-jwt/jwt/v5"
)

var jlog = elog.Get("/auth/jwt")

// JWKSVerifier fetches and caches the JWKS from an OIDC issuer, then uses it
// to verify JWT signatures and validate the issuer claim.
type JWKSVerifier struct {
	issuer  string
	jwksURL string

	mu      sync.RWMutex
	keys    map[string]crypto.PublicKey
	fetched time.Time
}

// NewJWKSVerifier creates a verifier for the given OIDC issuer URL.
// It lazily fetches the JWKS on first use.
func NewJWKSVerifier(issuer string) *JWKSVerifier {
	return &JWKSVerifier{issuer: issuer}
}

// VerifyJWT parses a JWT, verifies its signature against the issuer's JWKS,
// and validates that the "iss" claim matches the configured issuer.
func (v *JWKSVerifier) VerifyJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.NewParser(
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(v.issuer),
	).Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("token missing kid header")
		}

		key, err := v.getKey(kid)
		if err != nil {
			return nil, err
		}
		return key, nil
	})
	if err != nil {
		return nil, fmt.Errorf("JWT verification failed: %w", err)
	}

	return token, nil
}

const jwksCacheTTL = 10 * time.Minute

func (v *JWKSVerifier) getKey(kid string) (crypto.PublicKey, error) {
	v.mu.RLock()
	if key, ok := v.keys[kid]; ok && time.Since(v.fetched) < jwksCacheTTL {
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()

	// Refresh JWKS
	if err := v.fetchJWKS(); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok := v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %q not found in JWKS from %s", kid, v.issuer)
	}
	return key, nil
}

func (v *JWKSVerifier) fetchJWKS() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(v.fetched) < 5*time.Second {
		return nil
	}

	// Discover JWKS URL if not yet known
	if v.jwksURL == "" {
		url, err := discoverJWKSURL(v.issuer)
		if err != nil {
			return err
		}
		v.jwksURL = url
	}

	jlog.Debug("fetching JWKS", "url", v.jwksURL)

	resp, err := http.Get(v.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JWKS response: %w", err)
	}

	var jwks struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("failed to parse JWKS: %w", err)
	}

	keys := make(map[string]crypto.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Use != "" && k.Use != "sig" {
			continue
		}
		pub, err := k.publicKey()
		if err != nil {
			jlog.Warn("skipping JWKS key", "kid", k.Kid, "error", err)
			continue
		}
		keys[k.Kid] = pub
	}

	v.keys = keys
	v.fetched = time.Now()
	jlog.Debug("JWKS loaded", "keys", len(keys))
	return nil
}

func discoverJWKSURL(issuer string) (string, error) {
	discoveryURL := issuer + "/.well-known/openid-configuration"
	resp, err := http.Get(discoveryURL)
	if err != nil {
		return "", fmt.Errorf("OIDC discovery failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC discovery returned %s", resp.Status)
	}

	var doc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("failed to parse OIDC discovery: %w", err)
	}
	if doc.JWKSURI == "" {
		return "", fmt.Errorf("OIDC discovery has no jwks_uri")
	}
	return doc.JWKSURI, nil
}

// jwk represents a single JSON Web Key.
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	// RSA
	N string `json:"n"`
	E string `json:"e"`
	// EC
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func (k *jwk) publicKey() (crypto.PublicKey, error) {
	switch k.Kty {
	case "RSA":
		return k.rsaPublicKey()
	case "EC":
		return k.ecPublicKey()
	default:
		return nil, fmt.Errorf("unsupported key type: %s", k.Kty)
	}
}

func (k *jwk) rsaPublicKey() (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA n: %w", err)
	}
	eb, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA e: %w", err)
	}

	n := new(big.Int).SetBytes(nb)
	e := 0
	for _, b := range eb {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

func (k *jwk) ecPublicKey() (*ecdsa.PublicKey, error) {
	var curve elliptic.Curve
	switch k.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported EC curve: %s", k.Crv)
	}

	xb, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, fmt.Errorf("invalid EC x: %w", err)
	}
	yb, err := base64.RawURLEncoding.DecodeString(k.Y)
	if err != nil {
		return nil, fmt.Errorf("invalid EC y: %w", err)
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xb),
		Y:     new(big.Int).SetBytes(yb),
	}, nil
}
