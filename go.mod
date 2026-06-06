module github.com/KubelanCloud/kks-csi-plugin

go 1.24.5

require (
	github.com/container-storage-interface/spec v1.9.0
	github.com/hashicorp/hcl/v2 v2.24.0
	github.com/spf13/cobra v1.9.1
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.68.1
	google.golang.org/protobuf v1.34.2
	k8s.io/mount-utils v0.33.3
	k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738
)

require (
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/zclconf/go-cty v1.16.3 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
)

replace k8s.io/kubernetes => github.com/kubernetes/kubernetes v1.33.3
