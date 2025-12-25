package configfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Config struct {
	EnableAuthOnOptions    bool     `json:"enable_auth_on_options"`
	TokenExtractors        []string `json:"token_extractors"`
	OAuthJsonWebKeySetsURL string   `json:"oauth_json_web_key_sets_url"`
	OAuthIssuer            string   `json:"oauth_issuer"`
	OAuthAudience          string   `json:"oauth_audience"`
}

func ParseFile(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}

	if err := c.Validate(); err != nil {
		return Config{}, err
	}

	return c, nil
}

func (c Config) Validate() error {
	for _, ex := range c.TokenExtractors {
		switch ex {
		case "headers", "params":
			// ok
		default:
			return fmt.Errorf("token_extractors: unsupported value %q (allowed: headers, params)", ex)
		}
	}

	// Allow empty OAuth fields when auth is not used, but keep a sanity check for partially-configured auth.
	hasAny := c.OAuthJsonWebKeySetsURL != "" || c.OAuthIssuer != "" || c.OAuthAudience != ""
	hasAll := c.OAuthJsonWebKeySetsURL != "" && c.OAuthIssuer != "" && c.OAuthAudience != ""
	if hasAny && !hasAll {
		return errors.New("oauth config incomplete: require oauth_json_web_key_sets_url, oauth_issuer, oauth_audience together")
	}

	return nil
}


