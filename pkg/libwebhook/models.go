package libwebhook

import (
	"encoding/json"
	"github.com/monzo/typhon"
	"gopkg.in/dealancer/validate.v2"
	"io/ioutil"
	"os"
	"strings"
)

type GenericResponse struct {
	Message interface{} `json:"message"`
	Error   *string     `json:"error,omitempty"`
}

func ParseConfig(path string) (config Config, err error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return Config{}, err
	}
	err = json.Unmarshal(content, &config)
	if err != nil {
		return Config{}, err
	}
	err = validate.Validate(config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}

type Config struct {
	EnableAuthOnOptions    bool     `json:"enable_auth_on_options"`
	TokenExtractors        []string `json:"token_extractors" validate:"> 	one_of=headers,params"` // allowed values are "headers" and "params"
	OAuthJsonWebKeySetsUrl string   `json:"oauth_json_web_key_sets_url"`
	OAuthIssuer            string   `json:"oauth_issuer"`
	OAuthAudience          string   `json:"oauth_audience"`
	//TODO: add more fields here if you want to make the app more configurable
}

type Service func(app *App) typhon.Service
type Validator func(request typhon.Request) (interface{}, error)
type ValidatorWithService typhon.Filter

type Route struct {
	Path           string               `json:"-"`
	Method         string               `json:"method"`
	CurlExample    string               `json:"curl_example"`
	Validator      *Validator           `json:"-"`
	TokenValidator ValidatorWithService `json:"-"`
	Service        Service              `json:"-"`
}

type Module interface {
	Version() string
	Namespace() string
	Routes() map[string]Route
	LongPath(route Route) string
}

func DefaultLongPath(module Module, route Route) string {
	return "/" + strings.Join([]string{module.Version(), module.Namespace(), route.Path}, "/")
}
