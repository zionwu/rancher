package watcher

import (
	"fmt"
	"strings"

	"github.com/rancher/norman/controller"
	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/apps/v1beta2"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	appsv1beta2 "k8s.io/api/apps/v1beta2"
)

type StatefulsetWatcher struct {
	statefulsetLister  v1beta2.StatefulSetLister
	projectAlertLister v3.ProjectAlertLister
	alertManager       *manager.Manager
	clusterName        string
}

func StartStatefulsetWatcher(cluster *config.ClusterContext, manager *manager.Manager) {

	client := cluster.Apps.StatefulSets("")
	watcher := &StatefulsetWatcher{
		statefulsetLister:  client.Controller().Lister(),
		projectAlertLister: cluster.Management.Management.ProjectAlerts("").Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}
	client.AddHandler("statefulset-alert-watcher", watcher.Watch)

}

func (w *StatefulsetWatcher) Watch(key string, statefulset *appsv1beta2.StatefulSet) error {
	logrus.Infof("statefulset key %s", key)
	projectAlerts, err := w.projectAlertLister.List("", labels.NewSelector())
	if err != nil {
		logrus.Infof("Failed to get project alerts: %v", err)
		return nil
	}

	pAlerts := []*v3.ProjectAlert{}
	for _, alert := range projectAlerts {
		if controller.ObjectInCluster(w.clusterName, alert) {
			pAlerts = append(pAlerts, alert)
		}
	}
	ss, err := w.statefulsetLister.Get(statefulset.Namespace, statefulset.Name)
	if err != nil {
		return err
	}

	for _, alert := range pAlerts {
		if alert.Spec.TargetWorkload.Type == "statefulset" {
			if alert.Spec.TargetWorkload.ID != "" {
				parts := strings.Split(alert.Spec.TargetWorkload.ID, ":")
				ns := parts[0]
				id := parts[1]
				if id == ss.Name && ns == ss.Namespace {
					w.checkUnavailble(ss, alert)
				}

			} else if alert.Spec.TargetWorkload.Selector != nil {
				ssLabel := labels.Set(ss.GetLabels())

				selector := labels.NewSelector()
				for key, value := range alert.Spec.TargetWorkload.Selector {
					r, _ := labels.NewRequirement(key, selection.Equals, []string{value})
					selector.Add(*r)
				}
				if selector.Matches(ssLabel) {
					w.checkUnavailble(ss, alert)
				}
			}
		}
	}

	return nil
}

func (w *StatefulsetWatcher) checkUnavailble(ss *appsv1beta2.StatefulSet, alert *v3.ProjectAlert) {
	alertId := alert.Namespace + "-" + alert.Name
	percentage := alert.Spec.TargetWorkload.UnavailablePercentage

	if percentage == 0 {
		return
	}

	availableThreshold := (100 - int32(percentage)) * (*ss.Spec.Replicas) / 100

	if ss.Status.ReadyReplicas <= availableThreshold {
		title := fmt.Sprintf("The stateful set %s has unavailable replicas over %s", ss.Name, percentage)
		//TODO: get reason or log
		desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Ready Replicas*: %s\n*Desired Replicas*: %s", alert.Spec.DisplayName, w.clusterName, ss.Status.ReadyReplicas,
			ss.Spec.Replicas)

		if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
			logrus.Errorf("Failed to send alert: %v", err)
		}
	}

}
