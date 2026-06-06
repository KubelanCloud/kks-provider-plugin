# kks-csi-plugin

Kloud CSI driver for Kubernetes user clusters. This plugin runs inside each user cluster and implements the [Container Storage Interface (CSI)](https://github.com/container-storage-interface/spec) so workloads can use persistent volumes backed by Kloud storage.

Storage operations are forwarded to the **kks management CSI server** over HTTP. The driver does not talk to storage hardware directly; it translates Kubernetes CSI calls into REST API requests against the management plane.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  User Kubernetes cluster                                    │
│                                                             │
│  ┌──────────────────┐     ┌─────────────────────────────┐  │
│  │ CSI sidecars       │     │ kks-csi-plugin              │  │
│  │ (provisioner,       │────▶│ controller / node modes     │  │
│  │  attacher, etc.)   │     └──────────────┬──────────────┘  │
│  └──────────────────┘                      │ HTTP            │
│                                            ▼                 │
└────────────────────────────────────────────┼─────────────────┘
                                             │
                                             ▼
                              ┌──────────────────────────────┐
                              │ kks management CSI server    │
                              │ (Kloud control plane)        │
                              └──────────────────────────────┘
```

The Helm chart deploys two workloads:

| Component | Kind | Role |
|-----------|------|------|
| **Controller** | Deployment | Handles volume create/delete and publish/unpublish via CSI controller RPCs |
| **Node** | DaemonSet | Stages and publishes volumes on each node (format, mount, bind-mount) |

Standard CSI sidecars are bundled with each workload:

- **csi-provisioner** and **csi-attacher** on the controller
- **csi-node-driver-registrar** on each node
- **liveness-probe** on both

## Features

- Dynamic provisioning via a `StorageClass` (optional, enabled by default)
- Controller publish/unpublish (attach/detach)
- Node stage/unstage and publish/unpublish (mount operations)
- Block volumes are not supported
- Volume expansion is not supported

Driver name: `storage.csi.kloud.team`

## Prerequisites

- Kubernetes **1.28+**
- Network access from the user cluster to the kks management CSI server
- A cluster **CSI access token** and **server URL** from your Kloud cluster details

## Install with Helm

Install the chart into each user cluster (typically `kube-system`):

```bash
helm install kloud-csi ./charts/kloud-csi \
  --namespace kube-system \
  --create-namespace \
  --set serverURL=https://csi.example.kloud.team \
  --set accessToken="YOUR_CLUSTER_CSI_ACCESS_TOKEN"
```

### Using an existing secret

If you already have a secret containing the access token:

```bash
kubectl create secret generic kloud-csi-credentials \
  --namespace kube-system \
  --from-literal=access-token="YOUR_CLUSTER_CSI_ACCESS_TOKEN"

helm install kloud-csi ./charts/kloud-csi \
  --namespace kube-system \
  --set serverURL=https://csi.example.kloud.team \
  --set existingSecret=kloud-csi-credentials
```

### Verify the install

```bash
kubectl get pods -n kube-system -l app.kubernetes.io/name=kloud-csi
kubectl get csidriver storage.csi.kloud.team
kubectl get storageclass kloud-csi
```

## Helm chart

Chart path: [`charts/kloud-csi`](charts/kloud-csi)

### Required values

| Value | Description |
|-------|-------------|
| `serverURL` | Base URL of the kks management CSI server |
| `accessToken` | Cluster CSI access token (required unless `existingSecret` is set) |

Helm fails at render time if `serverURL` or credentials are missing.

### Common values

| Value | Default | Description |
|-------|---------|-------------|
| `image.repository` | `ghcr.io/KubelanCloud/kks-csi-plugin` | Driver container image |
| `image.tag` | `latest` | Image tag (defaults to chart `appVersion` if empty) |
| `existingSecret` | `""` | Use an existing secret instead of creating one |
| `existingSecretAccessTokenKey` | `access-token` | Key in the secret holding the token |
| `driver.name` | `storage.csi.kloud.team` | CSI driver name |
| `storageClass.enabled` | `true` | Create a `StorageClass` |
| `storageClass.name` | `kloud-csi` | StorageClass name |
| `storageClass.isDefault` | `false` | Mark as the default StorageClass |
| `storageClass.reclaimPolicy` | `Delete` | `Delete` or `Retain` |
| `storageClass.volumeBindingMode` | `WaitForFirstConsumer` | Volume binding mode |
| `controller.replicas` | `1` | Controller deployment replicas |
| `rbac.create` | `true` | Create RBAC for controller and node |

See [`charts/kloud-csi/values.yaml`](charts/kloud-csi/values.yaml) for the full list, including sidecar image versions and resource limits.

### Chart resources

The chart creates:

- `CSIDriver` — registers the driver with Kubernetes
- `StorageClass` — optional, for dynamic provisioning
- `Deployment` — controller + sidecars
- `DaemonSet` — node plugin + registrar on every node
- `ConfigMap` — minimal HCL stub (settings come from env vars)
- `Secret` — access token (unless `existingSecret` is used)
- RBAC — service accounts and roles for controller and node

### Upgrade and uninstall

```bash
helm upgrade kloud-csi ./charts/kloud-csi \
  --namespace kube-system \
  --reuse-values \
  --set serverURL=https://csi.example.kloud.team

helm uninstall kloud-csi --namespace kube-system
```

## Using persistent volumes

With the default StorageClass installed, create a PVC:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-data
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: kloud-csi
  resources:
    requests:
      storage: 10Gi
```

Then mount it in a pod as usual.

## Configuration reference

The driver accepts configuration from an HCL file, environment variables, or both. Environment variables take precedence and are what the Helm chart uses.

### HCL file

Example: [`examples/csi-client.hcl`](examples/csi-client.hcl)

```hcl
driver {
  name     = "storage.csi.kloud.team"
  endpoint = "unix:///var/lib/kubelet/plugins/storage.csi.kloud.team/csi.sock"
  mode     = "all"   # all | controller | node
}

client {
  server_url   = "https://csi.example.kloud.team"
  access_token = "YOUR_CLUSTER_CSI_ACCESS_TOKEN"
}
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `KKS_CSI_SERVER_URL` | Management CSI server base URL |
| `KKS_CSI_ACCESS_TOKEN` | Cluster access token |
| `KKS_CSI_DRIVER_MODE` | `controller`, `node`, or `all` |
| `KKS_CSI_NODE_ID` | Node identifier (set automatically on node pods) |
| `KKS_CSI_DRIVER_NAME` | CSI driver name override |
| `KKS_CSI_DRIVER_ENDPOINT` | gRPC socket path |
| `KKS_CSI_CLIENT_TIMEOUT_SECONDS` | HTTP client timeout (default: 30) |

## Standalone / development

Build and run locally:

```bash
go build -o kks-csi .
./kks-csi -c examples/csi-client.hcl
```

Or with Docker:

```bash
docker build -t kks-csi .
docker run --rm -v "$(pwd)/examples/csi-client.hcl:/csi.hcl:ro" kks-csi -c /csi.hcl
```

Container images are published to `ghcr.io/KubelanCloud/kks-csi-plugin` on pushes to `main`.

## Development

```bash
go test ./...
```

## License

See repository license terms.
