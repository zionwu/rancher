package watcher

import (
	"fmt"

	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

type EventLifecycle struct {
	eventLister        v3.ClusterEventLister
	clusterAlertLister v3.ClusterAlertLister
	alertManager       *manager.Manager
	clusterName        string
}

func StartEventWatcher(cluster *config.ClusterContext, manager *manager.Manager) {
	client := cluster.Management.Management.ClusterEvents(cluster.ClusterName)

	eventLifecycle := &EventLifecycle{
		eventLister:        client.Controller().Lister(),
		clusterAlertLister: cluster.Management.Management.ClusterAlerts(cluster.ClusterName).Controller().Lister(),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}

	client.AddLifecycle("cluster-event-alert", eventLifecycle)
}

func (l *EventLifecycle) Create(obj *v3.ClusterEvent) (*v3.ClusterEvent, error) {
	clusterAlerts, err := l.clusterAlertLister.List("", labels.NewSelector())
	if err != nil {
		logrus.Infof("Failed to get cluster alerts: %v", err)
		return obj, nil
	}
	for _, alert := range clusterAlerts {
		alertId := alert.Namespace + "-" + alert.Name
		target := alert.Spec.TargetEvent
		if target.ResourceKind != "" {
			if target.Type == obj.Event.Type && target.ResourceKind == obj.Event.InvolvedObject.Kind {

				title := fmt.Sprintf("A % event of %s occurred", target.Type, target.ResourceKind)
				//TODO: how to set unit for display for Quantity
				desc := fmt.Sprintf("*Alert Name*: %s\n*Cluster Name*: %s\n*Target*: %s\n*Event Message*: %s", obj.Event.InvolvedObject.Name, obj.Event.Message)

				if err := l.alertManager.SendAlert(alertId, desc, title, alert.Spec.Severity); err != nil {
					logrus.Errorf("Failed to send alert: %v", err)
				}

			}

		}
	}

	return obj, nil
}

func (l *EventLifecycle) Updated(obj *v3.ClusterEvent) (*v3.ClusterEvent, error) {
	return obj, nil
}

func (l *EventLifecycle) Remove(obj *v3.ClusterEvent) (*v3.ClusterEvent, error) {
	return obj, nil
}
