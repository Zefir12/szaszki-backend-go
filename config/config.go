package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

type ConfigValues struct {
	// You can add extra keys as needed
	Extra          map[string]interface{} `json:"-"`
	WS_PORT        IntOrString            `json:"WS_PORT"`
	GRPC_PORT      IntOrString            `json:"GRPC_PORT"`
	NODE_GRPC_ADDR string                 `json:"NODE_GRPC_ADDR"`
}

type Config struct {
	cache   *ConfigValues
	loading bool
	mu      sync.Mutex
}

// Singleton instance
var Instance = &Config{}

func (c *Config) Get() (*ConfigValues, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cache != nil {
		return c.cache, nil
	}
	if c.loading {
		return nil, errors.New("config is already loading, please wait")
	}

	c.loading = true
	defer func() { c.loading = false }()

	isProd := c.isProduction()
	fmt.Printf("🔧 Loading config (%s mode)...\n", map[bool]string{true: "PROD", false: "DEV"}[isProd])

	var cfg *ConfigValues
	var err error

	if isProd {
		cfg, err = c.fetchProdConfig()
	} else {
		cfg = &ConfigValues{}
	}

	if err != nil {
		fmt.Println("❌ Config load failed:", err)
		return nil, err
	}

	c.cache = cfg
	fmt.Printf("✅ Config loaded (%s): %+v\n", map[bool]string{true: "PROD", false: "DEV"}[isProd], cfg)
	return cfg, nil
}

func (c *Config) Reload() (*ConfigValues, error) {
	c.mu.Lock()
	c.cache = nil
	c.mu.Unlock()
	return c.Get()
}

// --- Helpers ---

func (c *Config) isProduction() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if os.Getenv("GO_ENV") == "production" {
		return true
	}
	if os.Getenv("CONTAINER") != "" || os.Getenv("DOCKER") != "" {
		return true
	}
	return false
}

func (c *Config) fetchProdConfig() (*ConfigValues, error) {
	serverURLsEnv := os.Getenv("ENV_SERVERS_URL")
	password := os.Getenv("ENV_PASSWORD")
	appID := os.Getenv("APP_ID")

	if serverURLsEnv == "" || password == "" || appID == "" {
		return nil, errors.New("ENV_SERVERS_URL, ENV_PASSWORD, and APP_ID must be set")
	}

	// Split and clean URLs
	serverURLs := []string{}
	for _, url := range strings.Split(serverURLsEnv, ",") {
		if trimmed := strings.TrimSpace(url); trimmed != "" {
			serverURLs = append(serverURLs, trimmed)
		}
	}
	if len(serverURLs) == 0 {
		return nil, errors.New("no valid server URLs provided in ENV_SERVERS_URL")
	}

	var lastErr error
	for _, serverURL := range serverURLs {
		url := fmt.Sprintf("%s/env", serverURL)
		body := fmt.Sprintf(`{"password":"%s","appId":%s}`, password, appID)

		resp, err := http.Post(url, "application/json", io.NopCloser(strings.NewReader(body)))
		if err != nil {
			lastErr = fmt.Errorf("failed to reach %s: %w", serverURL, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("server %s returned %d %s", serverURL, resp.StatusCode, resp.Status)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response from %s: %w", serverURL, err)
			continue
		}

		var cfg ConfigValues
		if err := json.Unmarshal(data, &cfg); err != nil {
			lastErr = fmt.Errorf("invalid config from %s: %w", serverURL, err)
			continue
		}

		return &cfg, nil
	}

	return nil, fmt.Errorf("all server URLs failed: %v", lastErr)
}

// small helper to turn string into io.Reader
type byteReader string

func (b byteReader) Read(p []byte) (int, error) {
	n := copy(p, b)
	if n < len(b) {
		return n, nil
	}
	return n, io.EOF
}

type IntOrString int

func (i *IntOrString) UnmarshalJSON(b []byte) error {
	// Try int
	var intVal int
	if err := json.Unmarshal(b, &intVal); err == nil {
		*i = IntOrString(intVal)
		return nil
	}

	// Try string
	var strVal string
	if err := json.Unmarshal(b, &strVal); err == nil {
		n, err := strconv.Atoi(strVal)
		if err != nil {
			return err
		}
		*i = IntOrString(n)
		return nil
	}

	return fmt.Errorf("VPSID must be string or int, got: %s", string(b))
}
