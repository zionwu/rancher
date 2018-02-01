package controller

import (
	"context"

	"github.com/rancher/rancher/pkg/alert/controller/configsyncer"
	"github.com/rancher/rancher/pkg/alert/controller/deploy"
	"github.com/rancher/rancher/pkg/alert/controller/podwatcher"
	"github.com/rancher/rancher/pkg/alert/controller/statesyncer"
	"github.com/rancher/rancher/pkg/alert/controller/syscomponentwatcher"
	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
)

const (
	alertManager = "alertmanager"
)

func Register(ctx context.Context, cluster *config.ClusterContext) {
	alertmanager := manager.NewManager(cluster)

	clusterAlertClient := cluster.Management.Management.ClusterAlerts(cluster.ClusterName)
	projectAlertClient := cluster.Management.Management.ProjectAlerts("")
	notifierClient := cluster.Management.Management.Notifiers("")

	clusterAlertLifecycle := &ClusterAlertLifecycle{}
	clusterAlertClient.AddLifecycle("cluster-alert-init-controller", clusterAlertLifecycle)

	projectAlertLifecycle := &ProjectAlertLifecycle{}
	projectAlertClient.AddLifecycle("project-alert-init-controller", projectAlertLifecycle)

	deployer := deploy.NewDeployer(cluster, alertmanager)
	clusterAlertClient.AddClusterScopedHandler("cluster-alert-deployer", cluster.ClusterName, deployer.ClusterSync)
	projectAlertClient.AddClusterScopedHandler("project-alert-deployer", cluster.ClusterName, deployer.ProjectSync)

	configSyncer := configsyner.NewConfigSyncer(cluster, alertmanager)
	clusterAlertClient.AddClusterScopedHandler("cluster-config-syncer", cluster.ClusterName, configSyncer.ClusterSync)
	projectAlertClient.AddClusterScopedHandler("project-config-syncer", cluster.ClusterName, configSyncer.ProjectSync)
	notifierClient.AddClusterScopedHandler("notifier-config-syncer", cluster.ClusterName, configSyncer.NotifierSync)

	stateSyncer := statesyncer.NewStateSyncer(cluster, alertmanager)
	go stateSyncer.Run(ctx.Done())

	podWatcher := podwatcher.NewWatcher(cluster, alertmanager)
	go podWatcher.Watch(ctx.Done())

	sysWatcher := syscomponentwatcher.NewSysComponentWatcher(cluster, alertmanager)
	go sysWatcher.Watch(ctx.Done())
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
