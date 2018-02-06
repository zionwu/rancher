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

type DeploymentWatcher struct {
	deploymentLister   v1beta2.DeploymentLister
	projectAlertLister v3.ProjectAlertLister
	alertManager       *manager.Manager
	clusterName        string
}

func StartDeploymentWatcher(cluster *config.ClusterContext, manager *manager.Manager) {

	dClient := cluster.Apps.Deployments("")
	watcher := &DeploymentWatcher{
		deploymentLister:   dClient.Controller().Lister(),
		projectAlertLister: cluster.Management.Management.ProjectAlerts("").Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}
	dClient.AddHandler("deployment-alert-watcher", watcher.Watch)

}

func (w *DeploymentWatcher) Watch(key string, deployment *appsv1beta2.Deployment) error {
	logrus.Infof("deployment key %s", key)
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

	dep, err := w.deploymentLister.Get(deployment.Namespace, deployment.Name)
	if err != nil {
		return err
	}

	for _, alert := range pAlerts {
		if alert.Spec.TargetWorkload.Type == "deployment" {

			if alert.Spec.TargetWorkload.ID != "" {
				parts := strings.Split(alert.Spec.TargetWorkload.ID, ":")
				ns := parts[0]
				id := parts[1]
				if id == dep.Name && ns == dep.Namespace {
					w.checkUnavailble(dep, alert)
				}

			} else if alert.Spec.TargetWorkload.Selector != nil {
				//TODO: should check if the deployment in the same project as the alert
				nodeLabel := labels.Set(dep.GetLabels())

				selector := labels.NewSelector()
				for key, value := range alert.Spec.TargetWorkload.Selector {
					r, _ := labels.NewRequirement(key, selection.Equals, []string{value})
					selector.Add(*r)
				}
				if selector.Matches(nodeLabel) {
					w.checkUnavailble(dep, alert)
				}
			}
		}
	}

	return nil
}

func (w *DeploymentWatcher) checkUnavailble(deployment *appsv1beta2.Deployment, alert *v3.ProjectAlert) {
	alertId := alert.Namespace + "-" + alert.Name
	percentage := alert.Spec.TargetWorkload.UnavailablePercentage

	if percentage == 0 {
		return
	}

	availableThreshold := (100 - int32(percentage)) * (*deployment.Spec.Replicas) / 100

	if deployment.Status.AvailableReplicas <= availableThreshold {
		title := fmt.Sprintf("The deployment %s has unavailable replicas over %s", deployment.Name, percentage)
		desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Available Replicas*: %s\n*Desired Replicas*: %s", alert.Spec.DisplayName, w.clusterName, deployment.Status.AvailableReplicas,
			deployment.Spec.Replicas)

		if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
			logrus.Errorf("Failed to send alert: %v", err)
		}
	}

}
