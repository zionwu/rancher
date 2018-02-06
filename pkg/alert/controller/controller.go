package controller

import (
	"context"

	"github.com/rancher/rancher/pkg/alert/controller/configsyncer"
	"github.com/rancher/rancher/pkg/alert/controller/deploy"
	"github.com/rancher/rancher/pkg/alert/controller/statesyncer"
	"github.com/rancher/rancher/pkg/alert/controller/watcher"
	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	sysWatcher := watcher.NewSysComponentWatcher(cluster, alertmanager)
	go sysWatcher.Watch(ctx.Done())

	watcher.StartNodeWatcher(cluster, alertmanager)
	watcher.StartPodWatcher(cluster, alertmanager)
	watcher.StartDeploymentWatcher(cluster, alertmanager)
	watcher.StartDaemonsetWatcher(cluster, alertmanager)
	watcher.StartStatefulsetWatcher(cluster, alertmanager)

	initClusterPreCanAlerts(clusterAlertClient, cluster.ClusterName)

}

func initClusterPreCanAlerts(clusterAlertClient v3.ClusterAlertInterface, clusterName string) {
	etcdRule := &v3.ClusterAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusteralert-etcd",
		},
		Spec: v3.ClusterAlertSpec{
			ClusterName: clusterName,
			AlertCommonSpec: v3.AlertCommonSpec{
				DisplayName:           "Alert for etcd",
				Description:           "Pre-can Alert for etcd component",
				Severity:              "critical",
				InitialWaitSeconds:    180,
				RepeatIntervalSeconds: 3600,
			},
			TargetSystemService: v3.TargetSystemService{
				Condition: "etcd",
			},
		},
	}

	if _, err := clusterAlertClient.Create(etcdRule); err != nil {
		logrus.Infof("Failed to create pre-can rules for etcd: %v", err)
	}

	cmRule := &v3.ClusterAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusteralert-controllermanager",
		},
		Spec: v3.ClusterAlertSpec{
			ClusterName: clusterName,
			AlertCommonSpec: v3.AlertCommonSpec{
				DisplayName:           "Alert for controller-manager",
				Description:           "Pre-can Alert for controller-manager component",
				Severity:              "critical",
				InitialWaitSeconds:    180,
				RepeatIntervalSeconds: 3600,
			},
			TargetSystemService: v3.TargetSystemService{
				Condition: "controller-manager",
			},
		},
	}

	if _, err := clusterAlertClient.Create(cmRule); err != nil {
		logrus.Infof("Failed to create pre-can rules for controller manager: %v", err)
	}

	schedulerRule := &v3.ClusterAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusteralert-scheduler",
		},
		Spec: v3.ClusterAlertSpec{
			ClusterName: clusterName,
			AlertCommonSpec: v3.AlertCommonSpec{
				DisplayName:           "Alert for scheduler",
				Description:           "Pre-can Alert for scheduler component",
				Severity:              "critical",
				InitialWaitSeconds:    180,
				RepeatIntervalSeconds: 3600,
			},
			TargetSystemService: v3.TargetSystemService{
				Condition: "scheduler",
			},
		},
	}

	if _, err := clusterAlertClient.Create(schedulerRule); err != nil {
		logrus.Infof("Failed to create pre-can rules for scheduler: %v", err)
	}

	dnsRule := &v3.ClusterAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusteralert-dns",
		},
		Spec: v3.ClusterAlertSpec{
			ClusterName: clusterName,
			AlertCommonSpec: v3.AlertCommonSpec{
				DisplayName:           "Alert for dns",
				Description:           "Pre-can Alert for dns component",
				Severity:              "critical",
				InitialWaitSeconds:    180,
				RepeatIntervalSeconds: 3600,
			},
			TargetSystemService: v3.TargetSystemService{
				Condition: "dns",
			},
		},
	}

	if _, err := clusterAlertClient.Create(dnsRule); err != nil {
		logrus.Infof("Failed to create pre-can rules for dns: %v", err)
	}

	networkRule := &v3.ClusterAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusteralert-network",
		},
		Spec: v3.ClusterAlertSpec{
			ClusterName: clusterName,
			AlertCommonSpec: v3.AlertCommonSpec{
				DisplayName:           "Alert for network",
				Description:           "Pre-can Alert for network component",
				Severity:              "critical",
				InitialWaitSeconds:    180,
				RepeatIntervalSeconds: 3600,
			},
			TargetSystemService: v3.TargetSystemService{
				Condition: "network",
			},
		},
	}

	if _, err := clusterAlertClient.Create(networkRule); err != nil {
		logrus.Infof("Failed to create pre-can rules for network: %v", err)
	}

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
