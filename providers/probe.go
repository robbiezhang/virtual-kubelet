package providers

import (
	_ "k8s.io/kubernetes/pkg/probe/http"
	_ "k8s.io/kubernetes/pkg/probe/tcp"
)
