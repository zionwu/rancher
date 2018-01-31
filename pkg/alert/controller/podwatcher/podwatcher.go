package podwatcher

import (
	"fmt"
	"strings"
	"time"

	"github.com/rancher/norman/controller"
	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/core/v1"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

type PodWatcher struct {
	podLister          v1.PodLister
	alertManager       *manager.Manager
	projectAlertLister v3.ProjectAlertLister
	clusterName        string
}

func NewWatcher(cluster *config.ClusterContext, manager *manager.Manager) *PodWatcher {
	return &PodWatcher{
		podLister:          cluster.Core.Pods("").Controller().Lister(),
		projectAlertLister: cluster.Management.Management.ProjectAlerts("").Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}

}

func (w *PodWatcher) Watch(stopc <-chan struct{}) {
	tickChan := time.NewTicker(time.Second * 10).C

	for {
		select {
		case <-stopc:
			return
		case <-tickChan:
			projectAlerts, err := w.projectAlertLister.List("", labels.NewSelector())
			if err != nil {
				logrus.Errorf("Error occured while getting project alerts: %v", err)
				continue
			}

			pAlerts := []*v3.ProjectAlert{}
			for _, alert := range projectAlerts {
				if controller.ObjectInCluster(w.clusterName, alert) {
					pAlerts = append(pAlerts, alert)
				}
			}

			for _, alert := range pAlerts {
				alertId := alert.Namespace + "-" + alert.Name

				if alert.Spec.TargetPod.ID != "" {
					parts := strings.Split(alert.Spec.TargetPod.ID, ":")
					ns := parts[0]
					podId := parts[1]
					pod, err := w.podLister.Get(ns, podId)
					if err != nil {
						logrus.Errorf("Error occured while getting pod %s: %v", alert.Spec.TargetPod.ID, err)
						continue
					}

					switch alert.Spec.TargetPod.Condition {
					case "notrunning":
						for _, containerStatus := range pod.Status.ContainerStatuses {
							if containerStatus.State.Running == nil {
								//TODO: need to consider all the cases
								reason := ""
								details := ""

								if containerStatus.State.Waiting != nil {
									reason = containerStatus.State.Waiting.Reason
									details = containerStatus.State.Waiting.Message
								}

								if containerStatus.State.Terminated != nil {
									reason = containerStatus.State.Terminated.Reason
									details = containerStatus.State.Terminated.Message
								}

								title := fmt.Sprintf("The Pod %s is not running", pod.Name)
								desc := fmt.Sprintf("*Summary*: The container `%s` is not running due to %s.\n*Cluster Name*: %s\n*Namespace*: %s\n*Details*: %s", containerStatus.Name, reason, w.clusterName, ns, details)

								if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
									logrus.Errorf("Error occured while getting pod %s: %v", alert.Spec.TargetPod.ID, err)
								}
							}
						}

					case "notschduled":
					case "restarts":

					}

				}

			}
		}

	}
}
