# kks-provider-plugin

Combined provider plugin for Kubernetes user clusters.

This repository now contains a single plugin and Helm chart that bundles:

- KloudLB controller + speaker logic
- Kloud CSI controller + node logic

The Helm chart deploys one DaemonSet (`kks-provider-plugin-provider`) that runs all required LB and CSI containers in the same pod on each node.

## Helm Chart

Chart URL: `https://github.com/kubelancloud/kks-provider-plugin/releases/download/v1.0.0/kks-provider-plugin-1.0.0.tgz`

Example install:

```bash
helm install kks-provider-plugin https://github.com/kubelancloud/kks-provider-plugin/releases/download/v1.0.0/kks-provider-plugin-1.0.0.tgz \
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

For cluster-wide pod/node resource metrics (for `kubectl top` and autoscaling), deploy Kubernetes metrics-server once per cluster, for example:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/high-availability-1.21+.yaml
```

## Image

Default image repository is:

`ghcr.io/kubelancloud/kks-provider-plugin`
