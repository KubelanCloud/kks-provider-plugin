package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	envCSIServerURL      = "KKS_CSI_SERVER_URL"
	envCSIAccessToken    = "KKS_CSI_ACCESS_TOKEN"
	envCSIDriverMode     = "KKS_CSI_DRIVER_MODE"
	envCSINodeID         = "KKS_CSI_NODE_ID"
	envCSIDriverName     = "KKS_CSI_DRIVER_NAME"
	envCSIDriverEndpoint = "KKS_CSI_DRIVER_ENDPOINT"
)

// ApplyEnvOverrides applies environment variables used by the Helm chart.
func ApplyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.Driver == nil {
		cfg.Driver = &DriverConf{}
	}
	if cfg.Client == nil {
		cfg.Client = &ClientConf{}
	}

	if v := strings.TrimSpace(os.Getenv(envCSIServerURL)); v != "" {
		cfg.Client.ServerURL = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv(envCSIAccessToken)); v != "" {
		cfg.Client.AccessToken = v
	}
	if v := strings.TrimSpace(os.Getenv(envCSIDriverMode)); v != "" {
		cfg.Driver.Mode = strings.ToLower(v)
	}
	if v := strings.TrimSpace(os.Getenv(envCSINodeID)); v != "" {
		cfg.Driver.NodeID = v
	}
	if v := strings.TrimSpace(os.Getenv(envCSIDriverName)); v != "" {
		cfg.Driver.Name = v
	}
	if v := strings.TrimSpace(os.Getenv(envCSIDriverEndpoint)); v != "" {
		cfg.Driver.Endpoint = v
	}

	if cfg.Client.TimeoutSeconds <= 0 {
		cfg.Client.TimeoutSeconds = 30
	}
	if cfg.Driver.Mode == "" {
		cfg.Driver.Mode = CSIModeNode
	}
	if cfg.Driver.Name == "" {
		cfg.Driver.Name = "storage.csi.kloud.team"
	}
	if cfg.Driver.Endpoint == "" {
		cfg.Driver.Endpoint = "unix:///var/lib/kubelet/plugins/storage.csi.kloud.team/csi.sock"
	}
}

func ApplyClientTimeoutFromEnv(cfg *Config) {
	if cfg == nil || cfg.Client == nil {
		return
	}
	raw := strings.TrimSpace(os.Getenv("KKS_CSI_CLIENT_TIMEOUT_SECONDS"))
	if raw == "" {
		return
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return
	}
	cfg.Client.TimeoutSeconds = seconds
}
