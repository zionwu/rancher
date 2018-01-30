package statesyncer

import (
	"time"

	"github.com/rancher/norman/controller"
	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

func NewStateSyncer(cluster *config.ClusterContext, manager *manager.Manager) *StateSyncer {
	return &StateSyncer{
		clusterAlertClient: cluster.Management.Management.ClusterAlerts(cluster.ClusterName),
		projectAlertClient: cluster.Management.Management.ProjectAlerts(""),
		alertManager:       manager,
		clusterName:        cluster.ClusterName,
	}
}

type StateSyncer struct {
	clusterAlertClient v3.ClusterAlertInterface
	projectAlertClient v3.ProjectAlertInterface
	alertManager       *manager.Manager
	clusterName        string
}

func (s *StateSyncer) Run(stopc <-chan struct{}) {

	tickChan := time.NewTicker(time.Second * 10).C

	for {
		select {
		case <-stopc:
			return
		case <-tickChan:
			apiAlerts, err := s.alertManager.GetAlertList()
			if err != nil {
				logrus.Errorf("Error while getting alert list from alertmanager: %v", err)
			} else {
				clusterAlerts, err := s.clusterAlertClient.Controller().Lister().List("", labels.NewSelector())
				if err != nil {
					logrus.Errorf("Error occured while getting cluster alerts: %v", err)
					continue
				}

				projectAlerts, err := s.projectAlertClient.Controller().Lister().List("", labels.NewSelector())
				if err != nil {
					logrus.Errorf("Error occured while getting project alerts: %v", err)
					continue
				}

				pAlerts := []*v3.ProjectAlert{}
				for _, alert := range projectAlerts {
					if controller.ObjectInCluster(s.clusterName, alert) {
						pAlerts = append(pAlerts, alert)
					}
				}

				for _, alert := range clusterAlerts {
					alertId := alert.Namespace + "-" + alert.Name
					state := s.alertManager.GetState(alertId, apiAlerts)
					needUpdate := s.doSync(alertId, alert.Status.State, state)

					if needUpdate {
						alert.Status.State = state
						_, err := s.clusterAlertClient.Update(alert)
						if err != nil {
							logrus.Errorf("Error occurred while updating alert state : %v", err)
						}
					}
				}

				for _, alert := range pAlerts {
					alertId := alert.Namespace + "-" + alert.Name
					state := s.alertManager.GetState(alertId, apiAlerts)
					needUpdate := s.doSync(alertId, alert.Status.State, state)

					if needUpdate {
						alert.Status.State = state
						_, err := s.projectAlertClient.Update(alert)
						if err != nil {
							logrus.Errorf("Error occurred while updating alert state and time: %v", err)
						}
					}
				}
			}
		}
	}

}

//The curState is the state in the CRD status,
//The newState is the state in alert manager side
func (s *StateSyncer) doSync(alertId, curState, newState string) (needUpdate bool) {
	if curState == "inactive" {
		return false
	}

	//only take ation when the state is not the same
	if newState != curState {

		//the alert is muted by user (curState == muted), but it already went away in alertmanager side (newState == active)
		//then we need to remove the silence rule and update the state in CRD
		if curState == "muted" && newState == "active" {
			err := s.alertManager.RemoveSilenceRule(alertId)
			if err != nil {
				logrus.Errorf("Error occurred while remove silence : %v", err)
			}
			return true
		}

		//the alert is unmuted by user, but it is still muted in alertmanager side
		//need to remove the silence rule, but do not have to update the CRD
		if curState == "alerting" && newState == "muted" {
			err := s.alertManager.RemoveSilenceRule(alertId)
			if err != nil {
				logrus.Errorf("Error occurred while remove silence : %v", err)
			}
			return false
		}

		//the alert is muted by user, but it is still alerting in alertmanager side
		//need to add silence rule to alertmanager
		if curState == "muted" && newState == "alerting" {
			err := s.alertManager.AddSilenceRule(alertId)
			if err != nil {
				logrus.Errorf("Error occurred while remove silence : %v", err)
			}
			return false
		}

		return true
	}

	return false

}
