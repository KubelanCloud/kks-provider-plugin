package client

import (
	"github.com/KubelanCloud/kks-provider-plugin/config"
	"github.com/KubelanCloud/kks-provider-plugin/pkg/csi/api"
	"github.com/KubelanCloud/kks-provider-plugin/pkg/csi/provisioner"
)

func NewBackend(cfg *config.ClientConf) provisioner.Backend {
	return api.NewClient(api.ClientConfig{
		BaseURL: cfg.ServerURL,
		Token:   cfg.BearerToken(),
		Timeout: cfg.Timeout(),
	})
}
