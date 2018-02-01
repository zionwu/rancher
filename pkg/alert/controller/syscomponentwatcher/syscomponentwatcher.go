package syscomponentwatcher

import (
	"strings"
	"time"

	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/core/v1"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type SysComponentWatcher struct {
	componentStatuses  v1.ComponentStatusInterface
	clusterAlertLister v3.ClusterAlertLister
	alertManager       *manager.Manager
	clustertName       string
}

func NewSysComponentWatcher(cluster *config.ClusterContext, manager *manager.Manager) *SysComponentWatcher {

	return &SysComponentWatcher{
		componentStatuses:  cluster.Core.ComponentStatuses(""),
		clusterAlertLister: cluster.Management.Management.ClusterAlerts(cluster.ClusterName).Controller().Lister(),
		alertManager:       manager,
		clustertName:       cluster.ClusterName,
	}
}

func (w *SysComponentWatcher) Watch(stopc <-chan struct{}) {
	tickChan := time.NewTicker(time.Second * 10).C
	for {
		select {
		case <-stopc:
			return
		case <-tickChan:
			clusterAlerts, err := w.clusterAlertLister.List("", labels.NewSelector())
			if err != nil {
				logrus.Errorf("Error occured while getting project alerts: %v", err)
				continue
			}

			for _, alert := range clusterAlerts {
				if alert.Spec.TargetSystemService.Condition != "" {
					statues, err := w.componentStatuses.List(metav1.ListOptions{})
					if err != nil {
						logrus.Errorf("Error occured while getting project alerts: %v", err)
						continue
					}

					for _, cs := range statues.Items {
						if strings.HasPrefix(cs.Name, alert.Spec.TargetSystemService.Condition) {

						}
					}
				}

			}
		}

	}
}
