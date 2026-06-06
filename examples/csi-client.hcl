# Run on user cluster nodes via Helm, or standalone with:
#   kks-csi -c examples/csi-client.hcl
#
# When installed with charts/kloud-csi, settings come from env vars instead.
driver {
  name     = "storage.csi.kloud.team"
  endpoint = "unix:///var/lib/kubelet/plugins/storage.csi.kloud.team/csi.sock"
  mode     = "all"
}

client {
  server_url   = "https://csi.kloud.team"
  access_token = "REPLACE_WITH_CLUSTER_ACCESS_TOKEN"
}
