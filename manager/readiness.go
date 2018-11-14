package manager

import (
	"context"
	"fmt"
	"sync"

	"github.com/virtual-kubelet/virtual-kubelet/log"

	"go.opencensus.io/trace"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/status"
)

type ReadinessManager interface {
	// GetPodContainersReadiness returns the container readiness map for the specified pod
	GetPodContainersReadiness(ctx context.Context, namespace, pod string) map[string]bool
}

type readinessManager struct {
	rm        *ResourceManager
	lock      sync.RWMutex
	readiness map[string]map[string]map[string]bool
}

// NewReadinessManager creates a readniess manager
func NewReadinessManager(rm *ResourceManager) *readinessManager {
	return &readinessManager{rm: rm}
}

// GetPodStatus implements the status.PodStatusProvider interface
func (m *readinessManager) GetPodStatus(uid types.UID) (v1.PodStatus, bool) {
	ctc := context.TODO()
	ctx, span := trace.StartSpan(ctx, "readinessManager.GetPodStatus")
	defer span.End()
	logger := log.G(ctx).WithField("method", "readinessManager.GetPodStatus")
	logger.Debugf("Getting pod status with UID '%s'", uid)

	for pod := range m.rm.GetPods() {
		if pod.UID == uid {
			span.Annotate(nil, "Find pod")
			logger.Debugf("Find pod with UID '%s'", uid)
			return pod.Status, true
		}
	}

	span.SetStatus(trace.Status{Code: trace.StatusCodeNotFound, Message: fmt.Sprintf("Unable to find pod with UID '%s'", uid)})
	logger.Debugf("Unable to find pod with UID '%s'", uid)
	return v1.PodStatus{}, false
}

// Start implements the status.Manager interface
func (m *readinessManager) Start() {
	logger := log.G(context.TODO()).WithField("method", "readinessManager.Start")
	logger.Debug("Starting")
}

// SetPodStatus implements the status.Manager interface
func (m *readinessManager) SetPodStatus(pod *v1.Pod, status v1.PodStatus) {
	logger := log.G(context.TODO()).WithField("method", "readinessManager.SetPodStatus")
	logger.WithField("namespace", pod.Namespace).WithField("pod", pod.Name)
	logger.Debugf("Setting pod status:\n'%v'", status)
}

// SetContainerReadiness implements the status.Manager interface
func (m *readinessManager) SetContainerReadiness(podUID types.UID, containerID container.ContainerID, ready bool) {
	ctc := context.TODO()
	ctx, span := trace.StartSpan(ctx, "readinessManager.SetContainerReadiness")
	defer span.End()
	logger := log.G(ctx).WithField("method", "readinessManager.SetContainerReadiness")
	logger.Debugf("Pod with UID '%s', ContainerID '%s', Ready '%v'", podUID, containerID, ready)
	var targetPod *v1.Pod
	for pod := range m.rm.GetPods() {
		if pod.UID == podUID {
			logger.Debugf("Find pod with UID '%s'", uid)
			targetPod = pod
		}
	}

	if targetPod == nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeNotFound, Message: fmt.Sprintf("Unable to find pod with UID '%s'", uid)})
		logger.Debugf("Unable to find pod with UID '%s'", uid)
		return
	}

	logger = logger.WithField("namespace", targetPod.Namespace).WithField("pod", targetPod.Name)

	cid := containerID.String()
	for c := range targetPod.Status.ContainerStatuses {
		if c.ContainerID == cid {
			logger.Debugf("Find container '%s' with ContainerID '%s'", c.Name, cid)

			m.lock.Lock()
			defer m.lock.Unlock()

			ns, ok := m.readiness[targetPod.Namespace]
			if !ok {
				ns = make(map[string]map[string]bool)
			}

			pod, ok := ns[targetPod.Name]
			if !ok {
				pod = make(map[string]bool)
			}

			pod[c.Name] = ready

			span.Annotate(nil, "Container readiness is set")
			return
		}
	}

	span.SetStatus(trace.Status{Code: trace.StatusCodeNotFound, Message: fmt.Sprintf("Unable to find container with ContainerID '%s'", cid)})
	logger.Debugf("Unable to find container with ContainerID '%s'", cid)
}

// TerminatePod implements the status.Manager interface
func (m *readinessManager) TerminatePod(pod *v1.Pod) {
	logger := log.G(context.TODO()).WithField("method", "readinessManager.TerminatePod")
	logger.WithField("namespace", pod.Namespace).WithField("pod", pod.Name)
	logger.Debug("Terminate pod")
}

// RemoveOrphanedStatuses implements the status.Manager interface
func (m *readinessManager) RemoveOrphanedStatuses(podUIDs map[types.UID]bool) {
	logger := log.G(context.TODO()).WithField("method", "readinessManager.RemoveOrphanedStatuses")
	logger.WithField("namespace", pod.Namespace).WithField("pod", pod.Name)
	logger.Debugf("Remove orphanced pod:\n'%v'", podUIDs)
}

// GetPodContainersReadiness implements the ReadinessManager interface
func (m *readinessManager) GetPodContainersReadiness(ctx context.Context, namespace, pod string) map[string]bool {
	ctx, span := trace.StartSpan(ctx, "readinessManager.GetPodContainersReadiness")
	defer span.End()
	logger := log.G(ctx).WithField("method", "readinessManager.GetPodContainersReadiness")
	logger.WithField("namespace", namespace).WithField("pod", pod)

	m.lock.RLock()
	defer m.lock.RUnlock()

	if ns, ok := m.readiness[namespace]; ok {
		if pod, ok := ns[pod]; ok {
			span.Annotate(nil, "Find pod containers readiness")
			logger.Debug("Find pod containers readiness")
			return pod
		}
	}

	return nil
}