package manager

import (
	"k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/prober"
	"k8s.io/kubernetes/pkg/kubelet/prober/results"
)

type ProberManager interface {
	ReadinessManager
	LivenessManager

	// AddPod creates new probe workers for every container probe.
	AddPod(pod *v1.Pod)
	// RemovePod handles cleaning up the removed pod state, including terminating probe workers and deleting cached results.
	RemovePod(pod *v1.Pod)
}

type proberManager struct {
	readinessManager ReadinessManager
	livenessManager  LivenessManager
	proberManager    prober.Manager
}

// NewProberManager creates a probe manager
func NewProberManager(rm *ResourceManager) ProberManager {
	readinessManager := NewReadinessManager(rm)
	livenessResults := results.NewManager()
	livenessManager := NewLivenessManager(rm, livenessResults)
	proberManager := prober.NewManager(
		readinessManager,
		livenessResults,
		nil,
		container.NewRefManager(),
		NewEventRecorder())

	return &proberManager{
		livenessManager: livenessManager
		readinessManager: readinessManager
		proberManager: proberManager
	}
}

// Start implements the LivenessManager interface
func (m *proberManager) Start(ctx context.Context) {
	m.livenessManager.Start(ctx)
}

// GetLivenessUpdates implements the LivenessManager interface
func (m *proberManager) GetLivenessUpdates() <-chan *LivenessUpdate {
	return m.livenessManager.GetLivenessUpdates()
}

// GetLivenessUpdates implements the LivenessManager interface
func (m *proberManager) GetLivenessUpdates() <-chan *LivenessUpdate {
	return m.livenessManager.GetLivenessUpdates()
}

// GetPodContainersReadiness implements the ReadinessManager interface
func (m *proberManager) GetPodContainersReadiness(ctx context.Context, namespace, pod string) map[string]bool {
	return m.readinessManager.GetPodContainersReadiness(ctx, namespace, pod)
}

// AddPod implements the ProberManager interface
func (m *proberManager) AddPod(pod *v1.Pod) {
	m.proberManager.AddPod(pod)
}

// RemovePod implements the ProberManager interface
func (m *proberManager) RemovePod(pod *v1.Pod) {
	m.proberManager.RemovePod(pod)
}