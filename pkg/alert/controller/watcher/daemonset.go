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

type DaemonsetWatcher struct {
	daemonsetLister    v1beta2.DaemonSetLister
	projectAlertLister v3.ProjectAlertLister
	alertManager       *manager.Manager
	clusterName        string
}

func StartDaemonsetWatcher(cluster *config.ClusterContext, manager *manager.Manager) {

	client := cluster.Apps.DaemonSets("")
	watcher := &DaemonsetWatcher{
		daemonsetLister:    client.Controller().Lister(),
		projectAlertLister: cluster.Management.Management.ProjectAlerts("").Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}
	client.AddHandler("daemonset-alert-watcher", watcher.Watch)
}

func (w *DaemonsetWatcher) Watch(key string, daemonset *appsv1beta2.DaemonSet) error {
	logrus.Infof("daemon key %s", key)
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
	ds, err := w.daemonsetLister.Get(daemonset.Namespace, daemonset.Name)
	if err != nil {
		return err
	}

	for _, alert := range pAlerts {
		if alert.Spec.TargetWorkload.Type == "daemonset" {
			if alert.Spec.TargetWorkload.ID != "" {
				parts := strings.Split(alert.Spec.TargetWorkload.ID, ":")
				ns := parts[0]
				id := parts[1]
				if id == ds.Name && ns == ds.Namespace {
					w.checkUnavailble(ds, alert)
				}

			} else if alert.Spec.TargetWorkload.Selector != nil {
				ssLabel := labels.Set(ds.GetLabels())

				selector := labels.NewSelector()
				for key, value := range alert.Spec.TargetWorkload.Selector {
					r, _ := labels.NewRequirement(key, selection.Equals, []string{value})
					selector.Add(*r)
				}
				if selector.Matches(ssLabel) {
					w.checkUnavailble(ds, alert)
				}
			}
		}
	}

	return nil
}

func (w *DaemonsetWatcher) checkUnavailble(ds *appsv1beta2.DaemonSet, alert *v3.ProjectAlert) {
	alertId := alert.Namespace + "-" + alert.Name
	percentage := alert.Spec.TargetWorkload.UnavailablePercentage

	if percentage == 0 {
		return
	}

	availableThreshold := (100 - int32(percentage)) * (ds.Status.DesiredNumberScheduled) / 100

	if ds.Status.NumberAvailable <= availableThreshold {
		title := fmt.Sprintf("The daemonset %s has unavailable replicas over %s", ds.Name, percentage)
		//TODO: get reason or log
		desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Ready Replicas*: %s\n*Desired Replicas*: %s", alert.Spec.DisplayName, w.clusterName, ds.Status.NumberAvailable,
			ds.Status.DesiredNumberScheduled)

		if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
			logrus.Errorf("Failed to send alert: %v", err)
		}
	}

}
