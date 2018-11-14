package manager

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// NewEventRecorder creates an event recorder
func NewEventRecorder() record.EventRecorder {
	return record.NewBroadcaster().NewRecorder(runtime.NewScheme(), v1.EventSource{Component: "virtual-kubelet"})
}