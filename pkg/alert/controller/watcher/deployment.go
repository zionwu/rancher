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

type DeploymentWatcher struct {
	deploymentLister   v1beta2.DeploymentLister
	projectAlertLister v3.ProjectAlertLister
	alertManager       *manager.Manager
	clusterName        string
}

func NewDeploymentWatcher(cluster *config.ClusterContext, manager *manager.Manager) *DeploymentWatcher {

	return &DeploymentWatcher{
		deploymentLister:   cluster.Apps.Deployments("").Controller().Lister(),
		projectAlertLister: cluster.Management.Management.ProjectAlerts("").Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}

}

func (w *DeploymentWatcher) Watch(stopc <-chan struct{}) {
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
				if alert.Spec.TargetWorkload.Type == "deployment" {

					if alert.Spec.TargetWorkload.ID != "" {
						parts := strings.Split(alert.Spec.TargetWorkload.ID, ":")
						ns := parts[0]
						id := parts[1]
						dep, err := w.deploymentLister.Get(ns, id)
						if err != nil {
							continue
						}
						w.checkUnavailble(dep, alert)

					} else if alert.Spec.TargetWorkload.Selector != nil {
						//TODO: should check if the deployment in the same project as the alert

						selector := labels.NewSelector()
						for key, value := range alert.Spec.TargetWorkload.Selector {
							r, _ := labels.NewRequirement(key, selection.Equals, []string{value})
							selector.Add(*r)
						}
						deps, err := w.deploymentLister.List("", selector)
						if err != nil {
							continue
						}
						for _, dep := range deps {
							w.checkUnavailble(dep, alert)
						}
					}
				}
			}

		}
	}
}

func (w *DeploymentWatcher) checkUnavailble(deployment *appsv1beta2.Deployment, alert *v3.ProjectAlert) {
	alertId := alert.Namespace + "-" + alert.Name
	percentage := alert.Spec.TargetWorkload.UnavailablePercentage

	if percentage == 0 {
		return
	}

	availableThreshold := (100 - int32(percentage)) * (*deployment.Spec.Replicas) / 100

	if deployment.Status.AvailableReplicas <= availableThreshold {
		title := fmt.Sprintf("The deployment %s has unavailable replicas over %s%%", deployment.Name, strconv.Itoa(percentage))
		desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Available Replicas*: %s\n*Desired Replicas*: %s", alert.Spec.DisplayName, w.clusterName, strconv.Itoa(int(deployment.Status.AvailableReplicas)),
			strconv.Itoa(int(*deployment.Spec.Replicas)))

		if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
			logrus.Debugf("Failed to send alert: %v", err)
		}
	}

}
