package nodewatcher

import (
	"fmt"
	"time"

	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/core/v1"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

type NodeWatcher struct {
	nodeLister         v1.NodeLister
	clusterAlertLister v3.ClusterAlertLister
	alertManager       *manager.Manager
	clusterName        string
}

func NewWatcher(cluster *config.ClusterContext, manager *manager.Manager) *NodeWatcher {

	return &NodeWatcher{
		nodeLister:         cluster.Core.Nodes("").Controller().Lister(),
		clusterAlertLister: cluster.Management.Management.ClusterAlerts(cluster.ClusterName).Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}
}

func (w *NodeWatcher) Watch(stopc <-chan struct{}) {
	tickChan := time.NewTicker(time.Second * 10).C
	for {
		select {
		case <-stopc:
			return
		case <-tickChan:

			clusterAlerts, err := w.clusterAlertLister.List("", labels.NewSelector())
			if err != nil {
				logrus.Infof("Failed to get cluster alerts: %v", err)
				continue
			}

			for _, alert := range clusterAlerts {
				if alert.Spec.TargetNode.Condition != "notready" {
					if alert.Spec.TargetNode.ID != "" {
						node, err := w.nodeLister.Get("", alert.Spec.TargetNode.ID)
						if err != nil {
							continue
						}
						w.checkNodeReady(node, alert)

					} else if alert.Spec.TargetNode.Selector != nil {
						selector := labels.NewSelector()
						for key, value := range alert.Spec.TargetNode.Selector {
							r, _ := labels.NewRequirement(key, selection.Equals, []string{value})
							selector.Add(*r)
						}

						nodes, err := w.nodeLister.List("", selector)
						if err != nil {
							continue
						}
						for _, node := range nodes {
							w.checkNodeReady(node, alert)
						}
					}
				}
			}
		}
	}
}

func (w *NodeWatcher) checkNodeReady(node *corev1.Node, alert *v3.ClusterAlert) {
	alertId := alert.Namespace + "-" + alert.Name
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionFalse {

				title := fmt.Sprintf("The kubelet on the node %s is not healthy", node.Name)
				desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Logs*: %s", alert.Spec.DisplayName, w.clusterName, cond.Message)

				if err := w.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
					logrus.Errorf("Failed to send alert: %v", err)
				}
				return
			}
		}

	}

}
