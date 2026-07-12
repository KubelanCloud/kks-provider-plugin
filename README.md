# kks-provider-plugin

Combined provider plugin for Kubernetes user clusters.

This repository now contains a single plugin and Helm chart that bundles:

- KloudLB controller + speaker logic
- Kloud CSI controller + node logic

The Helm chart deploys one DaemonSet (`kks-provider-plugin-provider`) that runs all required LB and CSI containers in the same pod on each node.

## Helm Chart

Chart path: `charts/kks-provider-plugin`

Example install:

```bash
helm install kks-provider-plugin ./charts/kks-provider-plugin \
  --namespace kube-system \
  --create-namespace \
  --set lb.serverURL=https://lb.example.kloud.team \
  --set lb.accessToken="$LB_TOKEN" \
  --set csi.serverURL=https://csi.example.kloud.team \
  --set csi.accessToken="$CSI_TOKEN"
```

## Binary Commands

The container/binary entrypoint is `kks-provider` and exposes:

- `kks-provider csi`
- `kks-provider lb-controller`
- `kks-provider lb-speaker`

Each command also starts a Prometheus metrics endpoint on `/metrics` by default:

- `kks-provider csi` on `:10080`
- `kks-provider lb-controller` on `:10081`
- `kks-provider lb-speaker` on `:10082`

You can override the bind address with `--metrics-bind-address` (or disable metrics with `--metrics-bind-address=off`).

## Image

Default image repository is:

`ghcr.io/kubelancloud/kks-provider-plugin`
