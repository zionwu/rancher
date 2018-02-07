package watcher

import (
	"fmt"
	"strings"
	"time"

	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/core/v1"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SysComponentWatcher struct {
	componentStatuses  v1.ComponentStatusInterface
	clusterAlertLister v3.ClusterAlertLister
	alertManager       *manager.Manager
	clusterName        string
}

func NewSysComponentWatcher(cluster *config.ClusterContext, manager *manager.Manager) *SysComponentWatcher {

	return &SysComponentWatcher{
		componentStatuses:  cluster.Core.ComponentStatuses(""),
		clusterAlertLister: cluster.Management.Management.ClusterAlerts(cluster.ClusterName).Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}
}

func (w *SysComponentWatcher) Watch(stopc <-chan struct{}) {
	tickChan := time.NewTicker(time.Second * 60).C
	for {
		select {
		case <-stopc:
			return
		case <-tickChan:
			clusterAlerts, err := w.clusterAlertLister.List("", labels.NewSelector())
			if err != nil {
				logrus.Errorf("Error occured while getting project alerts: %v", err)
				continue
			}

			statuses, err := w.componentStatuses.List(metav1.ListOptions{})
			if err != nil {
				logrus.Errorf("Error occured while getting component statuses: %v", err)
				continue
			}
			for _, alert := range clusterAlerts {
				if alert.Spec.TargetSystemService.Condition != "" {
					w.checkComponentHealthy(statuses, alert)
				}
			}
		}

	}
}

func (w *SysComponentWatcher) checkComponentHealthy(statuses *v1.ComponentStatusList, alert *v3.ClusterAlert) {
	alertId := alert.Namespace + "-" + alert.Name
	for _, cs := range statuses.Items {
		if strings.HasPrefix(cs.Name, alert.Spec.TargetSystemService.Condition) {
			for _, cond := range cs.Conditions {
				if cond.Type == corev1.ComponentHealthy {
					if cond.Status == corev1.ConditionFalse {
						title := fmt.Sprintf("The system component %s is not running", alert.Spec.TargetSystemService.Condition)
						desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Logs*: %s", alert.Spec.DisplayName, w.clusterName, cond.Message)

						if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
							logrus.Errorf("Failed to send alert: %v", err)
						}
						return
					}

				}

			}

		}
	}

}
