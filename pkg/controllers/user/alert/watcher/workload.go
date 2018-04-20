package watcher

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rancher/norman/controller"
	"github.com/rancher/rancher/pkg/controllers/user/alert/manager"
	"github.com/rancher/rancher/pkg/controllers/user/workload"
	"github.com/rancher/rancher/pkg/ticker"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	syncInterval = 30 * time.Second
)

type WorkloadWatcher struct {
	workloadController workload.CommonController
	alertManager       *manager.Manager
	projectAlertLister v3.ProjectAlertLister
	clusterName        string
}

func StartWorkloadWatcher(ctx context.Context, cluster *config.UserContext, manager *manager.Manager) {

	d := &WorkloadWatcher{
		projectAlertLister: cluster.Management.Management.ProjectAlerts("").Controller().Lister(),
		workloadController: workload.NewWorkloadController(cluster.UserOnlyContext(), nil),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}

	go d.watch(ctx, syncInterval)
}

func (w *WorkloadWatcher) watch(ctx context.Context, interval time.Duration) {
	for range ticker.Context(ctx, interval) {
		err := w.watchRule()
		if err != nil {
			logrus.Infof("Failed to watch deployment", err)
		}
	}
}

func (w *WorkloadWatcher) watchRule() error {
	if w.alertManager.IsDeploy == false {
		return nil
	}

	projectAlerts, err := w.projectAlertLister.List("", labels.NewSelector())
	if err != nil {
		return err
	}

	pAlerts := []*v3.ProjectAlert{}
	for _, alert := range projectAlerts {
		if controller.ObjectInCluster(w.clusterName, alert) {
			pAlerts = append(pAlerts, alert)
		}
	}

	for _, alert := range pAlerts {
		if alert.Status.AlertState == "inactive" {
			continue
		}

		if alert.Spec.TargetWorkload == nil {
			continue
		}

		if alert.Spec.TargetWorkload.WorkloadID != "" {

			wl, err := w.workloadController.GetByWorkloadID(alert.Spec.TargetWorkload.WorkloadID)
			if err != nil || wl == nil {
				logrus.Warnf("Fail to get workload for %s, %v", alert.Spec.TargetWorkload.WorkloadID, err)
				continue
			}

			w.checkWorkloadCondition(wl, alert)

		} else if alert.Spec.TargetWorkload.Selector != nil {
			wls, err := w.workloadController.GetWorkloadsMatchingSelector("", alert.Spec.TargetWorkload.Selector)
			if err != nil {
				logrus.Warnf("Fail to list workload: %v", err)
				continue
			}
			for _, wl := range wls {
				w.checkWorkloadCondition(wl, alert)
			}
		}

	}

	return nil
}

func (w *WorkloadWatcher) checkWorkloadCondition(wl *workload.Workload, alert *v3.ProjectAlert) {

	if wl.Kind == workload.JobType || wl.Kind == workload.CronJobType {
		return
	}

	alertID := alert.Namespace + "-" + alert.Name
	percentage := alert.Spec.TargetWorkload.AvailablePercentage

	if percentage == 0 {
		return
	}

	availableThreshold := int32(percentage) * (wl.Status.Replicas) / 100

	if wl.Status.AvailableReplicas <= availableThreshold {
		title := fmt.Sprintf("The workload %s has available replicas less than  %s%%", wl.Name, strconv.Itoa(percentage))
		desc := fmt.Sprintf("Alert Name: %s\n Severity: %s\n Cluster Name: %s\n Available Replicas: %s\n Desired Replicas: %s", alert.Spec.DisplayName, alert.Spec.Severity, w.clusterName, strconv.Itoa(int(wl.Status.AvailableReplicas)),
			strconv.Itoa(int(wl.Status.Replicas)))

		if err := w.alertManager.SendAlert(alertID, desc, title, alert.Spec.Severity); err != nil {
			logrus.Debugf("Failed to send alert: %v", err)
		}
	}

}
