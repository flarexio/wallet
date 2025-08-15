package http

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/golang-jwt/jwt/v5"

	"github.com/flarexio/wallet/conf"
)

var (
	issuer   string
	audience string
	keyFn    jwt.Keyfunc
)

func Init(cfg conf.JWTConfig) error {
	issuer = cfg.Issuer
	audience = cfg.Audience

	if cfg.JWKsURL == "" {
		return errors.New("JWKURL is required for JWT verification")
	}

	fn, err := fetchEd25519Keyfunc(cfg.JWKsURL)
	if err != nil {
		return err
	}

	keyFn = fn

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

func fetchEd25519Keyfunc(jwkURL string) (jwt.Keyfunc, error) {
	resp, err := http.Get(jwkURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var set JWKSet
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, err
	}

	if len(set.Keys) == 0 {
		return nil, errors.New("no keys in JWKS")
	}

	// 支援多把 key，依 kid 選擇
	keys := make(map[string]ed25519.PublicKey)
	for _, k := range set.Keys {
		if k.Kty == "OKP" && k.Crv == "Ed25519" && k.X != "" {
			pub, err := base64.RawURLEncoding.DecodeString(k.X)
			if err != nil {
				return nil, err
			}

			keys[k.Kid] = ed25519.PublicKey(pub)
		}
	}

	if len(keys) == 0 {
		return nil, errors.New("no Ed25519 key found in JWKS")
	}

	return func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if pub, ok := keys[kid]; ok {
			return pub, nil
		}

		for _, pub := range keys {
			return pub, nil
		}

		return nil, errors.New("no matching Ed25519 key found")
	}, nil
}
