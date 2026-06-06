package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	hcl "github.com/hashicorp/hcl/v2/hclsimple"
)

const (
	CSIModeAll        = "all"
	CSIModeController = "controller"
	CSIModeNode       = "node"
)

type Config struct {
	Driver *DriverConf `hcl:"driver,block"`
	Client *ClientConf `hcl:"client,block"`
}

type DriverConf struct {
	Name     string `hcl:"name,optional"`
	Endpoint string `hcl:"endpoint,optional"`
	NodeID   string `hcl:"node_id,optional"`
	Mode     string `hcl:"mode,optional"`
}

type ClientConf struct {
	ServerURL      string `hcl:"server_url,optional"`
	AccessToken    string `hcl:"access_token,optional"`
	Token          string `hcl:"token,optional"`
	TimeoutSeconds int    `hcl:"timeout_seconds,optional"`
}

func Load(filename string) (*Config, error) {
	cfg := &Config{}
	if err := hcl.DecodeFile(filename, nil, cfg); err != nil {
		return nil, err
	}
	if err := cfg.normalize(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadClient loads driver config from file and/or env vars used by the Helm chart.
func LoadClient(filename string) (*Config, error) {
	var cfg *Config
	if filename != "" {
		if _, err := os.Stat(filename); err == nil {
			loaded, err := Load(filename)
			if err != nil {
				return nil, err
			}
			cfg = loaded
		}
	}
	if cfg == nil {
		cfg = &Config{
			Driver: &DriverConf{},
			Client: &ClientConf{},
		}
		if err := cfg.normalize(); err != nil {
			return nil, err
		}
	}
	ApplyEnvOverrides(cfg)
	ApplyClientTimeoutFromEnv(cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Client == nil || strings.TrimSpace(c.Client.ServerURL) == "" {
		return fmt.Errorf("client.server_url is required")
	}
	if strings.TrimSpace(c.Client.BearerToken()) == "" {
		return fmt.Errorf("client.access_token is required")
	}
	return nil
}

func (c *Config) normalize() error {
	if c.Driver == nil {
		c.Driver = &DriverConf{}
	}
	if c.Client == nil {
		c.Client = &ClientConf{}
	}

	if c.Driver.Name == "" {
		c.Driver.Name = "storage.csi.kloud.team"
	}
	if c.Driver.Endpoint == "" {
		c.Driver.Endpoint = "unix:///var/lib/kubelet/plugins/storage.csi.kloud.team/csi.sock"
	}
	if c.Driver.NodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("driver.node_id is empty and hostname lookup failed: %w", err)
		}
		c.Driver.NodeID = hostname
	}
	if c.Driver.Mode == "" {
		c.Driver.Mode = CSIModeAll
	}

	mode := strings.ToLower(strings.TrimSpace(c.Driver.Mode))
	switch mode {
	case CSIModeAll, CSIModeController, CSIModeNode:
		c.Driver.Mode = mode
	default:
		return fmt.Errorf("driver.mode must be one of: all, controller, node")
	}

	c.Client.ServerURL = strings.TrimRight(strings.TrimSpace(c.Client.ServerURL), "/")
	if c.Client.TimeoutSeconds <= 0 {
		c.Client.TimeoutSeconds = 30
	}
	if strings.TrimSpace(c.Client.AccessToken) == "" && strings.TrimSpace(c.Client.Token) != "" {
		c.Client.AccessToken = strings.TrimSpace(c.Client.Token)
	}

	return nil
}

func (c *ClientConf) Timeout() time.Duration {
	return time.Duration(c.TimeoutSeconds) * time.Second
}

func (c *ClientConf) BearerToken() string {
	if strings.TrimSpace(c.AccessToken) != "" {
		return strings.TrimSpace(c.AccessToken)
	}
	return strings.TrimSpace(c.Token)
}
