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

func Register(ctx context.Context, cluster *config.ClusterContext) {
	alertmanager := manager.NewManager(cluster)

	clusterAlerts := cluster.Management.Management.ClusterAlerts(cluster.ClusterName)
	projectAlerts := cluster.Management.Management.ProjectAlerts("")
	notifiers := cluster.Management.Management.Notifiers("")
	projects := cluster.Management.Management.Projects("")

	projectLifecycle := &ProjectLifecycle{
		projectAlerts: projectAlerts,
		clusterName:   cluster.ClusterName,
	}
	projects.AddLifecycle("project-precan-alert-controller", projectLifecycle)

	deployer := deploy.NewDeployer(cluster, alertmanager)
	clusterAlerts.AddClusterScopedHandler("cluster-alert-deployer", cluster.ClusterName, deployer.ClusterSync)
	projectAlerts.AddClusterScopedHandler("project-alert-deployer", cluster.ClusterName, deployer.ProjectSync)

	configSyncer := configsyner.NewConfigSyncer(cluster, alertmanager)
	clusterAlerts.AddClusterScopedHandler("cluster-config-syncer", cluster.ClusterName, configSyncer.ClusterSync)
	projectAlerts.AddClusterScopedHandler("project-config-syncer", cluster.ClusterName, configSyncer.ProjectSync)
	notifiers.AddClusterScopedHandler("notifier-config-syncer", cluster.ClusterName, configSyncer.NotifierSync)

	stateSyncer := statesyncer.NewStateSyncer(cluster, alertmanager)
	go stateSyncer.Run(ctx.Done())

	sysWatcher := watcher.NewSysComponentWatcher(cluster, alertmanager)
	go sysWatcher.Watch(ctx.Done())

	podWatcher := watcher.NewPodWatcher(cluster, alertmanager)
	go podWatcher.Watch(ctx.Done())

	nodeWatcher := watcher.NewNodeWatcher(cluster, alertmanager)
	go nodeWatcher.Watch(ctx.Done())

	ssWatcher := watcher.NewStatefulsetWatcher(cluster, alertmanager)
	go ssWatcher.Watch(ctx.Done())

	depWatcher := watcher.NewDeploymentWatcher(cluster, alertmanager)
	go depWatcher.Watch(ctx.Done())

	dsWatcher := watcher.NewDaemonsetWatcher(cluster, alertmanager)
	go dsWatcher.Watch(ctx.Done())

	watcher.StartEventWatcher(cluster, alertmanager)

	initClusterPreCanAlerts(clusterAlerts, cluster.ClusterName)

}

func initClusterPreCanAlerts(clusterAlerts v3.ClusterAlertInterface, clusterName string) {
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

	if _, err := clusterAlerts.Create(etcdRule); err != nil {
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

	if _, err := clusterAlerts.Create(cmRule); err != nil {
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

	if _, err := clusterAlerts.Create(schedulerRule); err != nil {
		logrus.Infof("Failed to create pre-can rules for scheduler: %v", err)
	}

	nodeRule := &v3.ClusterAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusteralert-node-mem",
		},
		Spec: v3.ClusterAlertSpec{
			ClusterName: clusterName,
			AlertCommonSpec: v3.AlertCommonSpec{
				DisplayName:           "Alert for Node Memory Usage",
				Description:           "Pre-can Alert for node mem usage",
				Severity:              "critical",
				InitialWaitSeconds:    180,
				RepeatIntervalSeconds: 3600,
			},
			TargetNode: v3.TargetNode{
				Condition:    "mem",
				MemThreshold: 70,
				Selector: map[string]string{
					"node": "node",
				},
			},
		},
	}

	if _, err := clusterAlerts.Create(nodeRule); err != nil {
		logrus.Infof("Failed to create pre-can rules for node: %v", err)
	}

	eventRule := &v3.ClusterAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusteralert-deploment-event",
		},
		Spec: v3.ClusterAlertSpec{
			ClusterName: clusterName,
			AlertCommonSpec: v3.AlertCommonSpec{
				DisplayName:           "Alert for Warning Event of Deployment",
				Description:           "Pre-can Alert for warning event",
				Severity:              "critical",
				InitialWaitSeconds:    180,
				RepeatIntervalSeconds: 3600,
			},
			TargetEvent: v3.TargetEvent{
				Type:         "Warning",
				ResourceKind: "Deployemnt",
			},
		},
	}

	if _, err := clusterAlerts.Create(eventRule); err != nil {
		logrus.Infof("Failed to create pre-can rules for event: %v", err)
	}

}

type ProjectLifecycle struct {
	projectAlerts v3.ProjectAlertInterface
	clusterName   string
}

func (l *ProjectLifecycle) Create(obj *v3.Project) (*v3.Project, error) {
	deploymentAlert := &v3.ProjectAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "projectalert-deploment",
			Namespace: obj.Name,
		},
		Spec: v3.ProjectAlertSpec{
			ProjectName: l.clusterName + ":" + obj.Name,
			AlertCommonSpec: v3.AlertCommonSpec{
				DisplayName:           "Alert for Deployment",
				Description:           "Pre-can Alert for Deployment",
				Severity:              "critical",
				InitialWaitSeconds:    180,
				RepeatIntervalSeconds: 3600,
			},
			TargetWorkload: v3.TargetWorkload{
				Type: "deployment",
				Selector: map[string]string{
					"app": "deployment",
				},
				UnavailablePercentage: 50,
			},
		},
	}

	if _, err := l.projectAlerts.Create(deploymentAlert); err != nil {
		logrus.Infof("Failed to create pre-can rules for deployment: %v", err)
	}

	dsAlert := &v3.ProjectAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "projectalert-daemonset",
			Namespace: obj.Name,
		},
		Spec: v3.ProjectAlertSpec{
			ProjectName: l.clusterName + ":" + obj.Name,
			AlertCommonSpec: v3.AlertCommonSpec{
				DisplayName:           "Alert for Daemonset",
				Description:           "Pre-can Alert for Daemonset",
				Severity:              "critical",
				InitialWaitSeconds:    180,
				RepeatIntervalSeconds: 3600,
			},
			TargetWorkload: v3.TargetWorkload{
				Type: "daemonset",
				Selector: map[string]string{
					"app": "daemonset",
				},
				UnavailablePercentage: 50,
			},
		},
	}

	if _, err := l.projectAlerts.Create(dsAlert); err != nil {
		logrus.Infof("Failed to create pre-can rules for daemonset: %v", err)
	}

	ssAlert := &v3.ProjectAlert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "projectalert-statefuleset",
			Namespace: obj.Name,
		},
		Spec: v3.ProjectAlertSpec{
			ProjectName: l.clusterName + ":" + obj.Name,
			AlertCommonSpec: v3.AlertCommonSpec{
				DisplayName:           "Alert for StatefulSet",
				Description:           "Pre-can Alert for StatefulSet",
				Severity:              "critical",
				InitialWaitSeconds:    180,
				RepeatIntervalSeconds: 3600,
			},
			TargetWorkload: v3.TargetWorkload{
				Type: "statefulset",
				Selector: map[string]string{
					"app": "statefulset",
				},
				UnavailablePercentage: 50,
			},
		},
	}

	if _, err := l.projectAlerts.Create(ssAlert); err != nil {
		logrus.Infof("Failed to create pre-can rules for daemonset: %v", err)
	}

	return obj, nil
}

func (l *ProjectLifecycle) Updated(obj *v3.Project) (*v3.Project, error) {
	return obj, nil
}

func (l *ProjectLifecycle) Remove(obj *v3.Project) (*v3.Project, error) {
	return obj, nil
}
