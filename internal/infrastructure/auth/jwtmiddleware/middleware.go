package jwtmiddleware

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
)

type TokenContextKey struct{}

type TokenInfo struct {
	Raw    string
	Claims jwt.MapClaims
}

type Config struct {
	EnableAuthOnOptions bool
	TokenExtractors     []string

	JWKSURL  string
	Issuer   string
	Audience string

	// Optional overrides
	HTTPClient *http.Client
	CacheTTL   time.Duration
}

func (c Config) Valid() bool {
	return c.JWKSURL != "" && c.Issuer != "" && c.Audience != ""
}

type Middleware struct {
	cfg Config

	mu        sync.RWMutex
	cachedAt  time.Time
	cachedTTL time.Duration
	keys      map[string]*rsa.PublicKey // kid -> key
}

func New(cfg Config) *Middleware {
	ttl := cfg.CacheTTL
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &Middleware{
		cfg:       cfg,
		keys:      map[string]*rsa.PublicKey{},
		cachedTTL: ttl,
	}
}

func (m *Middleware) Huma(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(hctx huma.Context, next func(huma.Context)) {
		if !m.cfg.EnableAuthOnOptions && strings.EqualFold(hctx.Method(), http.MethodOptions) {
			next(hctx)
			return
		}

		if !m.cfg.Valid() {
			// Config missing: treat as server misconfiguration.
			writeAuthErr(api, hctx, http.StatusServiceUnavailable, "auth not configured")
			return
		}

		raw, err := extractToken(hctx, m.cfg.TokenExtractors)
		if err != nil {
			writeAuthErr(api, hctx, http.StatusUnauthorized, err.Error())
			return
		}
		if raw == "" {
			writeAuthErr(api, hctx, http.StatusUnauthorized, "missing bearer token")
			return
		}

		claims := jwt.MapClaims{}
		parsed, err := jwt.ParseWithClaims(raw, claims, m.keyFunc(hctx.Context()), jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
		if err != nil {
			writeAuthErr(api, hctx, http.StatusUnauthorized, "invalid token")
			return
		}
		if !parsed.Valid {
			writeAuthErr(api, hctx, http.StatusUnauthorized, "invalid token")
			return
		}

		if !verifyIssuer(claims, m.cfg.Issuer) {
			writeAuthErr(api, hctx, http.StatusUnauthorized, "invalid issuer")
			return
		}
		if !verifyAudience(claims, m.cfg.Audience) {
			writeAuthErr(api, hctx, http.StatusUnauthorized, "invalid audience")
			return
		}

		info := TokenInfo{Raw: raw, Claims: claims}
		next(huma.WithValue(hctx, TokenContextKey{}, info))
	}
}

func writeAuthErr(api huma.API, ctx huma.Context, status int, msg string) {
	se := huma.NewError(status, msg)
	_ = huma.WriteErr(api, ctx, status, msg, se)
}

func (m *Middleware) keyFunc(reqCtx context.Context) jwt.Keyfunc {
	return func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing kid")
		}

		// Fast path: cached key.
		if key := m.getCachedKey(kid); key != nil {
			return key, nil
		}

		// Refresh JWKS and try again.
		if err := m.refreshKeys(reqCtx); err != nil {
			return nil, err
		}
		if key := m.getCachedKey(kid); key != nil {
			return key, nil
		}
		return nil, errors.New("unknown kid")
	}
}

func (m *Middleware) getCachedKey(kid string) *rsa.PublicKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if time.Since(m.cachedAt) > m.cachedTTL {
		return nil
	}
	return m.keys[kid]
}

type jwks struct {
	Keys []struct {
		Kid string   `json:"kid"`
		X5c []string `json:"x5c"`
	} `json:"keys"`
}

func (m *Middleware) refreshKeys(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Another goroutine may have refreshed it while we waited.
	if time.Since(m.cachedAt) <= m.cachedTTL && len(m.keys) > 0 {
		return nil
	}

	client := m.cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.cfg.JWKSURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("jwks fetch failed: status %d", resp.StatusCode)
	}

	var doc jwks
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return err
	}

	newKeys := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kid == "" || len(k.X5c) == 0 {
			continue
		}
		key, err := rsaPublicKeyFromX5C(k.X5c[0])
		if err != nil {
			continue
		}
		newKeys[k.Kid] = key
	}

	if len(newKeys) == 0 {
		return errors.New("jwks contained no usable keys")
	}

	m.keys = newKeys
	m.cachedAt = time.Now()
	return nil
}

func rsaPublicKeyFromX5C(certBase64 string) (*rsa.PublicKey, error) {
	pemCert := "-----BEGIN CERTIFICATE-----\n" + certBase64 + "\n-----END CERTIFICATE-----\n"
	block, _ := pem.Decode([]byte(pemCert))
	if block == nil {
		return nil, errors.New("invalid cert pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	pk, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not rsa public key")
	}
	return pk, nil
}

func extractToken(hctx huma.Context, extractors []string) (string, error) {
	if len(extractors) == 0 {
		extractors = []string{"headers"}
	}
	for _, ex := range extractors {
		switch ex {
		case "headers":
			raw := hctx.Header("Authorization")
			if raw == "" {
				continue
			}
			parts := strings.Fields(raw)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return "", errors.New("authorization header format must be Bearer {token}")
			}
			return parts[1], nil
		case "params":
			if v := hctx.Query("access_token"); v != "" {
				return v, nil
			}
			if v := hctx.Query("token"); v != "" {
				return v, nil
			}
		default:
			// ignore unknown extractor values
		}
	}
	return "", nil
}

func verifyIssuer(claims jwt.MapClaims, expected string) bool {
	iss, _ := claims["iss"].(string)
	return iss == expected
}

func verifyAudience(claims jwt.MapClaims, expected string) bool {
	// RFC allows aud to be string or array of strings.
	switch aud := claims["aud"].(type) {
	case string:
		return aud == expected
	case []any:
		for _, v := range aud {
			if s, ok := v.(string); ok && s == expected {
				return true
			}
		}
		return false
	default:
		return false
	}
}
