package watcher

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rancher/norman/controller"
	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/core/v1"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type PodWatcher struct {
	podLister          v1.PodLister
	alertManager       *manager.Manager
	projectAlertLister v3.ProjectAlertLister
	clusterName        string
	podRestartTrack    map[string][]restartTrack
}

type restartTrack struct {
	Count int32
	Time  time.Time
}

func NewPodWatcher(cluster *config.ClusterContext, manager *manager.Manager) *PodWatcher {
	client := cluster.Management.Management.ProjectAlerts("")

	podWatcher := &PodWatcher{
		podLister:          cluster.Core.Pods("").Controller().Lister(),
		projectAlertLister: client.Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
		podRestartTrack:    map[string][]restartTrack{},
	}

	projectAlertLifecycle := &ProjectAlertLifecycle{
		podWatcher: podWatcher,
	}
	client.AddClusterScopedLifecycle("project-alert-podtarget", cluster.ClusterName, projectAlertLifecycle)

	return podWatcher
}

type ProjectAlertLifecycle struct {
	podWatcher *PodWatcher
}

func (l *ProjectAlertLifecycle) Create(obj *v3.ProjectAlert) (*v3.ProjectAlert, error) {
	l.podWatcher.podRestartTrack[obj.Namespace+":"+obj.Name] = make([]restartTrack, 0)
	return obj, nil
}

func (l *ProjectAlertLifecycle) Updated(obj *v3.ProjectAlert) (*v3.ProjectAlert, error) {
	return obj, nil
}

func (l *ProjectAlertLifecycle) Remove(obj *v3.ProjectAlert) (*v3.ProjectAlert, error) {
	l.podWatcher.podRestartTrack[obj.Namespace+":"+obj.Name] = nil
	return obj, nil
}

func (w *PodWatcher) Watch(stopc <-chan struct{}) {
	tickChan := time.NewTicker(time.Second * 30).C
	for {
		select {
		case <-stopc:
			return
		case <-tickChan:
			projectAlerts, err := w.projectAlertLister.List("", labels.NewSelector())
			if err != nil {
				logrus.Infof("Failed to get project alerts: %v", err)
				continue
			}

			pAlerts := []*v3.ProjectAlert{}
			for _, alert := range projectAlerts {
				if controller.ObjectInCluster(w.clusterName, alert) {
					pAlerts = append(pAlerts, alert)
				}
			}

			for _, alert := range pAlerts {

				if alert.Spec.TargetPod.ID != "" {
					parts := strings.Split(alert.Spec.TargetPod.ID, ":")
					ns := parts[0]
					podId := parts[1]
					newPod, err := w.podLister.Get(ns, podId)
					if err != nil {
						//TODO: what to do when pod not found
						logrus.Infof("Failed to get pod %s: %v", podId, err)
						continue
					}

					switch alert.Spec.TargetPod.Condition {
					case "notrunning":
						w.checkPodRunning(newPod, alert)
					case "notscheduled":
						w.checkPodScheduled(newPod, alert)
					case "restarts":
						w.checkPodRestarts(newPod, alert)
					}
				}
			}

		}
	}
}

func (w *PodWatcher) checkPodRestarts(pod *corev1.Pod, alert *v3.ProjectAlert) {

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Running == nil {
			curCount := containerStatus.RestartCount
			preCount := w.getRestartTimeFromTrack(alert, curCount)

			if curCount-preCount >= int32(alert.Spec.TargetPod.RestartTimes) {
				alertId := alert.Namespace + "-" + alert.Name
				details := ""
				if containerStatus.State.Waiting != nil {
					details = containerStatus.State.Waiting.Message
				}
				title := fmt.Sprintf("The Pod %s restarts %s in 5 mins", pod.Name, strconv.Itoa(alert.Spec.TargetPod.RestartTimes))
				desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Namespace*: %s\n*Container Name*: %s\n*Logs*: %s", alert.Spec.DisplayName, w.clusterName, pod.Namespace, containerStatus.Name, details)

				if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
					logrus.Errorf("Error occured while getting pod %s: %v", alert.Spec.TargetPod.ID, err)
				}
			}

			return
		}
	}

}

func (w *PodWatcher) getRestartTimeFromTrack(alert *v3.ProjectAlert, curCount int32) int32 {
	tracks := w.podRestartTrack[alert.Namespace+":"+alert.Name]
	now := time.Now()

	if len(tracks) == 0 {
		tracks = append(tracks, restartTrack{Count: curCount, Time: now})
		w.podRestartTrack[alert.Namespace+":"+alert.Name] = tracks
		return curCount
	}

	for i, track := range tracks {
		if now.Sub(track.Time).Seconds() < float64(alert.Spec.TargetPod.RestartIntervalSeconds) {
			tracks = tracks[i:]
			tracks = append(tracks, restartTrack{Count: curCount, Time: now})
			w.podRestartTrack[alert.Namespace+":"+alert.Name] = tracks
			return track.Count
		}
	}

	w.podRestartTrack[alert.Namespace+":"+alert.Name] = []restartTrack{}
	return curCount
}

func (w *PodWatcher) checkPodRunning(pod *corev1.Pod, alert *v3.ProjectAlert) {
	if !w.checkPodScheduled(pod, alert) {
		return
	}

	alertId := alert.Namespace + "-" + alert.Name
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Running == nil {
			//TODO: need to consider all the cases
			details := ""
			if containerStatus.State.Waiting != nil {
				details = containerStatus.State.Waiting.Message
			}

			if containerStatus.State.Terminated != nil {
				details = containerStatus.State.Terminated.Message
			}

			title := fmt.Sprintf("The Pod %s is not running", pod.Name)
			desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Namespace*: %s\n*Container Name*: %s\n*Logs*: %s", alert.Spec.DisplayName, w.clusterName, pod.Namespace, containerStatus.Name, details)

			if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
				logrus.Errorf("Error occured while send alert %s: %v", alert.Spec.TargetPod.ID, err)
			}
			return
		}
	}
}

func (w *PodWatcher) checkPodScheduled(pod *corev1.Pod, alert *v3.ProjectAlert) bool {

	alertId := alert.Namespace + "-" + alert.Name
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
			details := condition.Message

			title := fmt.Sprintf("The Pod %s is not scheduled", pod.Name)
			desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Namespace*: %s\n*Pod Name*: %s\n*Logs*: %s", alert.Spec.DisplayName, w.clusterName, pod.Namespace, pod.Name, details)

			if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
				logrus.Errorf("Error occured while getting pod %s: %v", alert.Spec.TargetPod.ID, err)
			}
			return false
		}
	}

	return true

}
