package watcher

import (
	"fmt"

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

func StartNodeWatcher(cluster *config.ClusterContext, manager *manager.Manager) {

	nodeClient := cluster.Core.Nodes("")
	watcher := &NodeWatcher{
		nodeLister:         nodeClient.Controller().Lister(),
		clusterAlertLister: cluster.Management.Management.ClusterAlerts(cluster.ClusterName).Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}
	nodeClient.AddHandler("node-alert-watcher", watcher.Watch)
}

func (w *NodeWatcher) Watch(key string, node *corev1.Node) error {
	logrus.Infof("node key %s", key)

	clusterAlerts, err := w.clusterAlertLister.List("", labels.NewSelector())
	if err != nil {
		logrus.Infof("Failed to get cluster alerts: %v", err)
		return nil
	}

	newNode, err := w.nodeLister.Get("", node.Name)
	if err != nil {
		return err
	}

	for _, alert := range clusterAlerts {
		if alert.Spec.TargetNode.Condition != "notready" {
			//TODO: check key format
			if alert.Spec.TargetNode.ID != "" && alert.Spec.TargetNode.ID == node.Name {
				w.checkNodeReady(newNode, alert)

			} else if alert.Spec.TargetNode.Selector != nil {
				nodeLabel := labels.Set(newNode.GetLabels())

				selector := labels.NewSelector()
				for key, value := range alert.Spec.TargetNode.Selector {
					r, _ := labels.NewRequirement(key, selection.Equals, []string{value})
					selector.Add(*r)
				}
				if selector.Matches(nodeLabel) {
					w.checkNodeReady(newNode, alert)
				}

			}
		}
	}

	return nil
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
