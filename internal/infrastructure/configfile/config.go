package configfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server ServerConfig `json:"server"`
	DB     DBConfig     `json:"db"`

	EnableAuthOnOptions    bool     `json:"enable_auth_on_options"`
	TokenExtractors        []string `json:"token_extractors"`
	OAuthJsonWebKeySetsURL string   `json:"oauth_json_web_key_sets_url"`
	OAuthIssuer            string   `json:"oauth_issuer"`
	OAuthAudience          string   `json:"oauth_audience"`
}

type ServerConfig struct {
	Addr string `json:"addr"`
}

type DBConfig struct {
	Driver string `json:"driver"` // sqlite | postgres | memory
	DSN    string `json:"dsn"`

	MaxOpenConns           int `json:"max_open_conns"`
	MaxIdleConns           int `json:"max_idle_conns"`
	ConnMaxLifetimeSeconds int `json:"conn_max_lifetime_seconds"`
	ConnMaxIdleTimeSeconds int `json:"conn_max_idle_time_seconds"`

	SQLitePragmas map[string]string `json:"sqlite_pragmas"`
}

func Default() Config {
	// Keep auth defaults “off” unless explicitly configured.
	return Config{
		Server: ServerConfig{
			Addr: "0.0.0.0:1337",
		},
		DB: DBConfig{
			Driver: "sqlite",
			DSN:    ":memory:",
		},
	}
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

	c.ApplyDefaults()
	if err := c.Validate(); err != nil {
		return Config{}, err
	}

	return c, nil
}

func (c *Config) ApplyDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = "0.0.0.0:1337"
	}

	if c.DB.Driver == "" {
		c.DB.Driver = "sqlite"
	}
	if c.DB.DSN == "" && strings.EqualFold(c.DB.Driver, "sqlite") {
		c.DB.DSN = ":memory:"
	}

	// Pragmas default to nil unless the user provides them (repo layer will apply its own defaults).
	if c.DB.SQLitePragmas == nil {
		c.DB.SQLitePragmas = map[string]string{}
	}
}

func (c *Config) ApplyEnv(prefix string) error {
	// Server
	if v := os.Getenv(prefix + "ADDR"); v != "" {
		c.Server.Addr = v
	}
	if v := os.Getenv(prefix + "SERVER_ADDR"); v != "" {
		c.Server.Addr = v
	}

	// DB
	if v := os.Getenv(prefix + "DB_DRIVER"); v != "" {
		c.DB.Driver = v
	}
	if v := os.Getenv(prefix + "DB_DSN"); v != "" {
		c.DB.DSN = v
	}
	if v := os.Getenv(prefix + "DB_MAX_OPEN_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("%sDB_MAX_OPEN_CONNS: %w", prefix, err)
		}
		c.DB.MaxOpenConns = n
	}
	if v := os.Getenv(prefix + "DB_MAX_IDLE_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("%sDB_MAX_IDLE_CONNS: %w", prefix, err)
		}
		c.DB.MaxIdleConns = n
	}
	if v := os.Getenv(prefix + "DB_CONN_MAX_LIFETIME_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("%sDB_CONN_MAX_LIFETIME_SECONDS: %w", prefix, err)
		}
		c.DB.ConnMaxLifetimeSeconds = n
	}
	if v := os.Getenv(prefix + "DB_CONN_MAX_IDLE_TIME_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("%sDB_CONN_MAX_IDLE_TIME_SECONDS: %w", prefix, err)
		}
		c.DB.ConnMaxIdleTimeSeconds = n
	}

	// SQLite pragmas via env, as JSON object string to avoid a huge list of env vars.
	// Example: WEBHOOKD_SQLITE_PRAGMAS='{"busy_timeout":"5000","foreign_keys":"ON"}'
	if v := os.Getenv(prefix + "SQLITE_PRAGMAS"); v != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return fmt.Errorf("%sSQLITE_PRAGMAS: invalid JSON object: %w", prefix, err)
		}
		if c.DB.SQLitePragmas == nil {
			c.DB.SQLitePragmas = map[string]string{}
		}
		for k, val := range m {
			c.DB.SQLitePragmas[k] = val
		}
	}

	// Auth
	if v := os.Getenv(prefix + "ENABLE_AUTH_ON_OPTIONS"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("%sENABLE_AUTH_ON_OPTIONS: %w", prefix, err)
		}
		c.EnableAuthOnOptions = b
	}
	if v := os.Getenv(prefix + "TOKEN_EXTRACTORS"); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			out = append(out, p)
		}
		c.TokenExtractors = out
	}
	if v := os.Getenv(prefix + "OAUTH_JSON_WEB_KEY_SETS_URL"); v != "" {
		c.OAuthJsonWebKeySetsURL = v
	}
	if v := os.Getenv(prefix + "OAUTH_ISSUER"); v != "" {
		c.OAuthIssuer = v
	}
	if v := os.Getenv(prefix + "OAUTH_AUDIENCE"); v != "" {
		c.OAuthAudience = v
	}

	c.ApplyDefaults()
	return c.Validate()
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

	// DB config
	driver := strings.ToLower(strings.TrimSpace(c.DB.Driver))
	switch driver {
	case "", "sqlite", "postgres", "memory":
		// ok (empty will be defaulted)
	default:
		return fmt.Errorf("db.driver: unsupported value %q (allowed: sqlite, postgres, memory)", c.DB.Driver)
	}
	if strings.EqualFold(driver, "postgres") && strings.TrimSpace(c.DB.DSN) == "" {
		return errors.New("db.dsn: required when db.driver is postgres")
	}
	if c.DB.MaxOpenConns < 0 || c.DB.MaxIdleConns < 0 || c.DB.ConnMaxLifetimeSeconds < 0 || c.DB.ConnMaxIdleTimeSeconds < 0 {
		return errors.New("db: pool settings must be >= 0")
	}

	return nil
}
