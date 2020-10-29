package jwt

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/monzo/typhon"
)

type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

type CustomClaims struct {
	Scope string `json:"scope"`
	jwt.StandardClaims
}

type TokenExtractor func(r typhon.Request) (string, error)
type ErrorHandler func(r typhon.Request, errMsg string) typhon.Response
type EmptyTokenHandler typhon.Service
type ScopeChecker func(tokenString string) bool
