package probe

import (
	_ "k8s.io/client-go/tools/record"
	_ "k8s.io/client-go/tools/reference"
	_ "k8s.io/kubernetes/pkg/kubelet/container"
	_ "k8s.io/kubernetes/pkg/kubelet/prober"
	_ "k8s.io/kubernetes/pkg/kubelet/prober/results"
	_ "k8s.io/kubernetes/pkg/kubelet/status"
)
