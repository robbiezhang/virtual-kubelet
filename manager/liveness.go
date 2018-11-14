package manager

import (
	"context"

	"github.com/virtual-kubelet/virtual-kubelet/log"

	"go.opencensus.io/trace"
	"k8s.io/kubernetes/pkg/kubelet/prober/results"
)

type LivenessUpdate struct {
	namespace string
	pod       string
}

type LivenessManager interface {
	Start(ctx context.Context)
	GetLivenessUpdates() <-chan *LivenessUpdate
}

type livenessManager struct {
	rm             *ResourceManager
	resultsManager results.Manager
	updates        chan *LivenessUpdate
}

func NewLivenessManager(rm *ResourceManager, resultsManager results.Manager) LivenessManager {
	return &livenessManager{
		rm:             rm
		resultsManager: resultsManager
		updates:        make(chan *LivenessUpdate, 20)
	} 
}

func (m *livenessManager) Start(ctx context.Context) {
	go func(){
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-m.resultsManager.Updates():
				m.updatePodLiveness(ctx, &update)
			}
		}
	}()
}

func (m *livenessManager) GetLivenessUpdate() <-chan *LivenessUpdate {
	return m.updates
}

func (m *livenessManager) updatePodLiveness(ctx context.Context, update *results.Update) {
	ctx, span := trace.StartSpan(ctx, "livenessManager.updatePodLiveness")
	defer span.End()
	logger := log.G(ctx).WithField("method", "livenessManager.updatePodLiveness")
	logger.Debugf("Get update: %s", convertUpdateToString(update))

	if update.Result == results.Failure {
		for pod := range m.rm.GetPods() {
			if pod.UID == uid {
				logger = logger.WithField("namespace", pod.Namespace).WithField("pod", pod.Name)

				if pod.Status.Phase == corev1.PodSucceeded ||
					pod.Status.Phase == corev1.PodFailed ||
					pod.Status.Reason == podStatusReasonProviderFailed ||
					pod.DeletionTimestamp != nil {
					span.Annotate(nil, "Pod is terminated")
					logger.Debug("Pod is terminated. No update")
					return
				}

				span.Annotate(nil, "Find pod")
				logger.Debugf("Find pod with UID '%s'", update.PodUID)
				m.updates <- &LivenessUpdate{namespace: pod.Namespace, pod: pod.Name}
			}
		}
	}
}

func convertUpdateToString(u *results.Update) string {
	return fmt.Sprintf("PodUID '%s' ContainerID '%s' Result '%s'", u.PodUID, u.ContainerID.String(), u.Result)
}