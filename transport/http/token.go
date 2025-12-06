package http

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/flarexio/wallet/conf"
)

var (
	issuer   string
	audience string
	jwksURL  string

	cachedKeys map[string]ed25519.PublicKey
	cacheMu    sync.RWMutex
)

func Init(ctx context.Context, cfg conf.JWTConfig) error {
	issuer = cfg.Issuer
	audience = cfg.Audience
	jwksURL = cfg.JWKsURL

	if jwksURL == "" {
		return errors.New("JWKURL is required for JWT verification")
	}

	refreshJWKS()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				refreshJWKS()
			}
		}
	}()

	return nil
}

type Claims struct {
	jwt.RegisteredClaims
	Roles []string `json:"roles"`
}

func (c *Claims) Map() map[string]any {
	return map[string]any{
		"sub":   c.Subject,
		"roles": c.Roles,
	}
}

type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
}

type JWKSet struct {
	Keys []JWK `json:"keys"`
}

func refreshJWKS() error {
	set, err := fetchJWKSet(jwksURL)
	if err != nil {
		return err
	}

	keys := make(map[string]ed25519.PublicKey)
	for _, k := range set.Keys {
		if k.Kty == "OKP" && k.Crv == "Ed25519" && k.X != "" {
			pub, err := base64.RawURLEncoding.DecodeString(k.X)
			if err != nil {
				return err
			}

			keys[k.Kid] = ed25519.PublicKey(pub)
		}
	}

	if len(keys) == 0 {
		return errors.New("no Ed25519 key found in JWKS")
	}

	cacheMu.Lock()
	cachedKeys = keys
	cacheMu.Unlock()

	return nil
}

func fetchJWKSet(url string) (*JWKSet, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var set *JWKSet
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, err
	}

	if len(set.Keys) == 0 {
		return nil, errors.New("no keys in JWKS")
	}

	return set, nil
}

func KeyFunc(token *jwt.Token) (any, error) {
	cacheMu.RLock()
	keys := cachedKeys
	cacheMu.RUnlock()

	if len(keys) == 0 {
		return nil, errors.New("no cached keys available")
	}

	kid, _ := token.Header["kid"].(string)
	if pub, ok := keys[kid]; ok {
		return pub, nil
	}

	for _, pub := range keys {
		return pub, nil
	}

	return nil, errors.New("no matching Ed25519 key found")
}
