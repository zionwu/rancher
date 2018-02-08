package watcher

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

func NewStatefulsetWatcher(cluster *config.ClusterContext, manager *manager.Manager) *StatefulsetWatcher {
	return &StatefulsetWatcher{
		statefulsetLister:  cluster.Apps.StatefulSets("").Controller().Lister(),
		projectAlertLister: cluster.Management.Management.ProjectAlerts("").Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}

}

func (w *StatefulsetWatcher) Watch(stopc <-chan struct{}) {
	tickChan := time.NewTicker(time.Second * 30).C
	for {
		select {
		case <-stopc:
			return
		case <-tickChan:
			projectAlerts, err := w.projectAlertLister.List("", labels.NewSelector())
			if err != nil {
				continue
			}

			pAlerts := []*v3.ProjectAlert{}
			for _, alert := range projectAlerts {
				if controller.ObjectInCluster(w.clusterName, alert) {
					pAlerts = append(pAlerts, alert)
				}
			}

			for _, alert := range pAlerts {
				if alert.Status.State == "inactive" {
					continue
				}
				if alert.Spec.TargetWorkload.Type == "statefulset" {
					if alert.Spec.TargetWorkload.ID != "" {
						parts := strings.Split(alert.Spec.TargetWorkload.ID, ":")
						ns := parts[0]
						id := parts[1]
						ss, err := w.statefulsetLister.Get(ns, id)
						if err != nil {
							continue
						}
						w.checkUnavailble(ss, alert)

					} else if alert.Spec.TargetWorkload.Selector != nil {

						selector := labels.NewSelector()
						for key, value := range alert.Spec.TargetWorkload.Selector {
							r, _ := labels.NewRequirement(key, selection.Equals, []string{value})
							selector.Add(*r)
						}
						ss, err := w.statefulsetLister.List("", selector)
						if err != nil {
							continue
						}
						for _, s := range ss {
							w.checkUnavailble(s, alert)
						}

					}
				}
			}

		}
	}
}

func (w *StatefulsetWatcher) checkUnavailble(ss *appsv1beta2.StatefulSet, alert *v3.ProjectAlert) {
	alertId := alert.Namespace + "-" + alert.Name
	percentage := alert.Spec.TargetWorkload.UnavailablePercentage

	if percentage == 0 {
		return
	}

	availableThreshold := (100 - int32(percentage)) * (*ss.Spec.Replicas) / 100

	if ss.Status.ReadyReplicas <= availableThreshold {
		title := fmt.Sprintf("The stateful set %s has unavailable replicas over %s%%", ss.Name, strconv.Itoa(percentage))
		//TODO: get reason or log
		desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Ready Replicas*: %s\n*Desired Replicas*: %s", alert.Spec.DisplayName, w.clusterName, strconv.Itoa(int(ss.Status.ReadyReplicas)),
			strconv.Itoa(int(*ss.Spec.Replicas)))

		if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
			logrus.Debugf("Failed to send alert: %v", err)
		}
	}

}
