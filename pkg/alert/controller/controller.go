package controller

import (
	"github.com/rancher/rancher/pkg/alert/controller/configsyner"
	"github.com/rancher/rancher/pkg/alert/controller/deploy"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
)

const (
	alertManager = "alertmanager"
)

func Register(cluster *config.ClusterContext) {
	clusterAlertClient := cluster.Management.Management.ClusterAlerts(cluster.ClusterName)
	clusterAlertLifecycle := &ClusterAlertLifecycle{}
	clusterAlertClient.AddLifecycle("cluster-alert-init-controller", clusterAlertLifecycle)

	projectAlertClient := cluster.Management.Management.ProjectAlerts("")
	projectAlertLifecycle := &ProjectAlertLifecycle{}
	projectAlertClient.AddLifecycle("project-alert-init-controller", projectAlertLifecycle)

	deployer := deploy.NewDeployer(cluster)
	clusterAlertClient.AddClusterScopedHandler("cluster-alert-deployer", cluster.ClusterName, deployer.ClusterSync)
	projectAlertClient.AddClusterScopedHandler("project-alert-deployer", cluster.ClusterName, deployer.ProjectSync)

	configSyner := configsyner.NewConfigSyner(cluster)
	clusterAlertClient.AddClusterScopedHandler("cluster-config-syner", cluster.ClusterName, configSyner.ClusterSync)
	projectAlertClient.AddClusterScopedHandler("project-config-syner", cluster.ClusterName, configSyner.ProjectSync)

}

type ClusterAlertLifecycle struct {
}

func (l *ClusterAlertLifecycle) Create(obj *v3.ClusterAlert) (*v3.ClusterAlert, error) {
	obj.Status.State = "active"

	return obj, nil
}

func (l *ClusterAlertLifecycle) Updated(obj *v3.ClusterAlert) (*v3.ClusterAlert, error) {
	return obj, nil
}

func (l *ClusterAlertLifecycle) Remove(obj *v3.ClusterAlert) (*v3.ClusterAlert, error) {
	return obj, nil
}

type ProjectAlertLifecycle struct {
}

func (l *ProjectAlertLifecycle) Create(obj *v3.ProjectAlert) (*v3.ProjectAlert, error) {
	obj.Status.State = "active"

	return obj, nil
}

func (l *ProjectAlertLifecycle) Updated(obj *v3.ProjectAlert) (*v3.ProjectAlert, error) {
	return obj, nil
}

func (l *ProjectAlertLifecycle) Remove(obj *v3.ProjectAlert) (*v3.ProjectAlert, error) {
	return obj, nil
}
