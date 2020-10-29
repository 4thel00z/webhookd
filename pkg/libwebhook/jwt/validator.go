package jwt

import (
	"context"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/monzo/typhon"
	"log"
	"net/http"
	"webhookd/pkg/libwebhook"
	"strings"
)

type Option func(j *Validator)
type Validator struct {

	// If you are too lazy to scope check in your request handlers, you can do it here
	ScopeChecker ScopeChecker

	// Callback for an unsuccessful Errorcheck
	// Default value: OnScopeInsufficient
	ScopeCheckErrorHandler ErrorHandler

	// The function that will return the Key to validate the JWT.
	// It can be either a shared secret or a public key.
	// Default value: ValidationKeyGetterFromEnv()
	ValidationKeyGetter jwt.Keyfunc

	// The name of the property in the request where the user information
	// from the JWT will be stored.
	// Default value: "user"
	UserProperty string

	// The function that will be called when there's an error validating the token
	// Default value: OnError
	ErrorHandler ErrorHandler

	// The function that will be called when there is no token set
	// Default value: OnEmptyToken
	EmptyTokenHandler EmptyTokenHandler

	// A boolean indicating if the credentials are required or not
	// Default value: false
	CredentialsOptional bool

	// A function that extracts the token from the request
	// Default: FromAuthHeader (i.e., from Authorization header as bearer token)
	Extractor TokenExtractor

	// Debug flag turns on debugging output
	// Default: false
	Debug bool

	// When set, all requests with the OPTIONS method will use authentication
	// Default: false
	EnableAuthOnOptions bool

	// When set, the middleware verifies that tokens are signed with the specific signing algorithm
	// If the signing method is not constant the ValidationKeyGetter callback can be used to implement additional checks
	// Important to avoid security issues described here: https://auth0.com/blog/2015/03/31/critical-vulnerabilities-in-json-web-token-libraries/
	// Default: jwt.SigningMethodRS256
	SigningMethod jwt.SigningMethod
}

func WithScopeChecker(s ScopeChecker) Option {
	return func(j *Validator) {
		j.ScopeChecker = s
	}
}

func WithEnvScopeChecker(scope string) Option {
	return func(j *Validator) {
		j.ScopeChecker = CheckOAuthScopeFromEnv(scope)
	}
}

func WithScopeCheckErrorHandler(e ErrorHandler) Option {
	return func(j *Validator) {
		j.ScopeCheckErrorHandler = e
	}
}

func WithValidationKeyGetter(getter jwt.Keyfunc) Option {
	return func(j *Validator) {
		j.ValidationKeyGetter = getter
	}
}
func WithEnvValidationKeyGetter() Option {
	return func(j *Validator) {
		j.ValidationKeyGetter = ValidationKeyGetterFromEnv()
	}
}

func WithUserProperty(u string) Option {
	return func(j *Validator) {
		j.UserProperty = u
	}
}

func WithDebug() Option {
	return func(j *Validator) {
		j.Debug = true
	}
}

func WithSigningMethod(s jwt.SigningMethod) Option {
	return func(j *Validator) {
		j.SigningMethod = s
	}
}

func WithTokenExtractors(extractors ...TokenExtractor) Option {
	return func(j *Validator) {
		j.Extractor = FromFirst(extractors...)
	}
}

func WithTokenExtractor(extractor TokenExtractor) Option {
	return func(j *Validator) {
		j.Extractor = extractor
	}
}

func WithEmptyTokenHandler(e EmptyTokenHandler) Option {
	return func(j *Validator) {
		j.EmptyTokenHandler = e
	}
}

func WithCredentialsOptional(o bool) Option {
	return func(j *Validator) {
		j.CredentialsOptional = o
	}
}

// New constructs a new Secure instance with supplied 
func New(options ...Option) *Validator {

	j := &Validator{
		UserProperty:           DefaultUserProperty,
		ErrorHandler:           OnError,
		Extractor:              FromAuthHeader,
		ScopeCheckErrorHandler: OnScopeInsufficient,
		SigningMethod:          jwt.SigningMethodRS256,
		ValidationKeyGetter:    ValidationKeyGetterFromEnv(),
	}

	for _, option := range options {
		option(j)
	}

	return j
}

func (j *Validator) Middleware(r typhon.Request, service typhon.Service) typhon.Response {
	if !j.EnableAuthOnOptions {
		if r.Method == "OPTIONS" {
			return service(r)
		}
	}

	// Use the specified token extractor to extract a token from the request
	token, err := j.Extractor(r)

	// If debugging is turned on, log the outcome
	if err != nil {
		j.logf("Error extracting JWT: %v", err)
	} else {
		j.logf("Token extracted: %s", token)
	}

	// If an error occurs, call the error handler and return an error
	if err != nil {
		return j.ErrorHandler(r, fmt.Sprintf("error extracting token: %e", err))
	}

	// If the token is empty...
	if token == "" {
		// Check if it was required
		if j.CredentialsOptional {
			j.logf("  No credentials found (CredentialsOptional=true)")
			// No error, just no token (and that is ok given that CredentialsOptional is true)
			return service(r)
		}

		// If we get here, the required token is missing
		errorMsg := "Required authorization token not found"
		j.logf("  Error: No credentials found (CredentialsOptional=false)")
		return j.ErrorHandler(r, errorMsg)

	}

	// Now parse the token
	parsedToken, err := jwt.Parse(token, j.ValidationKeyGetter)

	// Check if there was an error in parsing...
	if err != nil {
		j.logf("Error parsing token:%s", err.Error())
		return j.ErrorHandler(r, fmt.Sprintf("Error parsing token: %s", err.Error()))

	}

	if j.SigningMethod != nil && j.SigningMethod.Alg() != parsedToken.Header["alg"] {
		message := fmt.Sprintf("Expected %s signing method but token specified %s",
			j.SigningMethod.Alg(),
			parsedToken.Header["alg"])
		j.logf("Error validating token algorithm: %s", message)
		return j.ErrorHandler(r, fmt.Sprintf("Error validating token algorithm: %s", message))
	}

	// Check if the parsed token is valid...
	if !parsedToken.Valid {
		j.logf("Token is invalid")
		return j.ErrorHandler(r, "The token isn't valid")
	}

	if j.ScopeChecker != nil {
		valid := j.ScopeChecker(token)
		if !valid {
			return j.ScopeCheckErrorHandler(r, "scope insufficient")
		}

	}

	j.logf("JWT: %v", parsedToken)

	r.Context = context.WithValue(r.Context, j.UserProperty, parsedToken)
	return service(r)
}

func (j *Validator) logf(format string, args ...interface{}) {
	if j.Debug {
		log.Printf(format, args...)
	}
}

// FromFirst returns a function that runs multiple token extractors and takes the
// first token it finds
func FromFirst(extractors ...TokenExtractor) TokenExtractor {
	return func(r typhon.Request) (string, error) {
		for _, ex := range extractors {
			token, err := ex(r)
			if err != nil {
				return "", err
			}
			if token != "" {
				return token, nil
			}
		}
		return "", nil
	}
}

func OnError(r typhon.Request, errMsg string) typhon.Response {
	response := r.Response(libwebhook.GenericResponse{
		Message: nil,
		Error:   &errMsg,
	})

	response.StatusCode = http.StatusUnauthorized
	return response
}
func OnScopeInsufficient(r typhon.Request, errMsg string) typhon.Response {
	response := r.Response(libwebhook.GenericResponse{
		Message: nil,
		Error:   &errMsg,
	})

	response.StatusCode = http.StatusForbidden
	return response
}

func FromAuthHeader(r typhon.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", nil // No error, just no token
	}

	// TODO: Make this a bit more robust, parsing-wise
	authHeaderParts := strings.Fields(authHeader)
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return "", errors.New("authorization header format must be Bearer {token}")
	}

	return authHeaderParts[1], nil
}

// TokenExtractorFromParameter returns a TokenExtractor that extracts the token from the specified
// query string parameter
func TokenExtractorFromParameter(param string) TokenExtractor {
	return func(r typhon.Request) (string, error) {
		return r.URL.Query().Get(param), nil
	}
}
