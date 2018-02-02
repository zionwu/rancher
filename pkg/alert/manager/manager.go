package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/common/model"
	alertconfig "github.com/rancher/rancher/pkg/alert/config"

	"github.com/prometheus/alertmanager/dispatch"
	"github.com/prometheus/alertmanager/types"
	"github.com/rancher/types/apis/core/v1"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

type Manager struct {
	nodeClient v1.NodeInterface
	svcClient  v1.ServiceInterface
	podClient  v1.PodInterface
}

func NewManager(cluster *config.ClusterContext) *Manager {
	return &Manager{
		nodeClient: cluster.Core.Nodes(""),
		svcClient:  cluster.Core.Services(""),
		podClient:  cluster.Core.Pods("cattle-alerting"),
	}
}

//TODO: optimized this
func (m *Manager) getAlertManagerEndpoint() string {

	selector := labels.NewSelector()
	r, _ := labels.NewRequirement("app", selection.Equals, []string{"alertmanager"})
	selector.Add(*r)
	pods, err := m.podClient.List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		logrus.Errorf("Error occured while get pod: %v", err)
		return ""
	}

	if len(pods.Items) == 0 {
		return ""
	}

	node, err := m.nodeClient.Get(pods.Items[0].Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("Error occured while list node: %v", err)
		return ""
	}

	//TODO: check correct way to make call to alertManager
	if len(node.Status.Addresses) == 0 {
		return ""
	}
	ip := node.Status.Addresses[0].Address
	svc, err := m.svcClient.GetNamespaced("cattle-alerting", "alertmanager", metav1.GetOptions{})
	if err != nil {
		//logrus.Errorf("Error occured while get svc : %v", err)
		return ""
	}
	port := svc.Spec.Ports[0].NodePort
	url := "http://" + ip + ":" + strconv.Itoa(int(port))
	return url
}

func (m *Manager) ReloadConfiguration() error {
	url := m.getAlertManagerEndpoint()
	resp, err := http.Post(url+"/-/reload", "text/html", nil)
	//logrus.Infof("Reload  configuration for %s", url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) GetDefaultConfig() *alertconfig.Config {
	config := alertconfig.Config{}

	resolveTimeout, _ := model.ParseDuration("5m")
	config.Global = &alertconfig.GlobalConfig{
		SlackAPIURL:    "slack_api_url",
		ResolveTimeout: resolveTimeout,
		SMTPRequireTLS: false,
	}

	slackConfigs := []*alertconfig.SlackConfig{}
	initSlackConfig := &alertconfig.SlackConfig{
		Channel: "#alert",
	}
	slackConfigs = append(slackConfigs, initSlackConfig)

	receivers := []*alertconfig.Receiver{}
	initReceiver := &alertconfig.Receiver{
		Name:         "rancherlabs",
		SlackConfigs: slackConfigs,
	}
	receivers = append(receivers, initReceiver)

	config.Receivers = receivers

	groupWait, _ := model.ParseDuration("1m")
	groupInterval, _ := model.ParseDuration("0m")
	repeatInterval, _ := model.ParseDuration("1h")

	config.Route = &alertconfig.Route{
		Receiver:       "rancherlabs",
		GroupWait:      &groupWait,
		GroupInterval:  &groupInterval,
		RepeatInterval: &repeatInterval,
	}

	return &config
}

func (m *Manager) GetAlertList() ([]*dispatch.APIAlert, error) {

	url := m.getAlertManagerEndpoint()
	res := struct {
		Data   []*dispatch.APIAlert `json:"data"`
		Status string               `json:"status"`
	}{}

	req, err := http.NewRequest(http.MethodGet, url+"/api/v1/alerts", nil)
	if err != nil {
		return nil, err
	}
	//q := req.URL.Query()
	//q.Add("filter", fmt.Sprintf("{%s}", filter))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	requestBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(requestBytes, &res); err != nil {
		return nil, err
	}

	return res.Data, nil
}

func (m *Manager) GetState(alertID string, apiAlerts []*dispatch.APIAlert) string {

	for _, a := range apiAlerts {
		if string(a.Labels["alert_id"]) == alertID {
			if a.Status.State == "suppressed" {
				return "muted"
			} else {
				return "alerting"
			}
		}
	}

	return "active"
}

func (m *Manager) AddSilenceRule(alertID string) error {

	url := m.getAlertManagerEndpoint()
	matchers := []*model.Matcher{}
	m1 := &model.Matcher{
		Name:    "alert_id",
		Value:   alertID,
		IsRegex: false,
	}
	matchers = append(matchers, m1)

	now := time.Now()
	endsAt := now.AddDate(100, 0, 0)
	silence := model.Silence{
		Matchers:  matchers,
		StartsAt:  now,
		EndsAt:    endsAt,
		CreatedAt: now,
		CreatedBy: "rancherlabs",
		Comment:   "silence",
	}

	silenceData, err := json.Marshal(silence)
	if err != nil {
		return err
	}

	resp, err := http.Post(url+"/api/v1/silences", "application/json", bytes.NewBuffer(silenceData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil

}

func (m *Manager) RemoveSilenceRule(alertID string) error {
	url := m.getAlertManagerEndpoint()
	res := struct {
		Data   []*types.Silence `json:"data"`
		Status string           `json:"status"`
	}{}

	req, err := http.NewRequest(http.MethodGet, url+"/api/v1/silences", nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("filter", fmt.Sprintf("{%s, %s}", "alert_id="+alertID))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	requestBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(requestBytes, &res); err != nil {
		return err
	}

	if res.Status != "success" {
		return fmt.Errorf("Failed to get silence rules for alert")
	}

	for _, s := range res.Data {
		if s.Status.State == types.SilenceStateActive {
			delReq, err := http.NewRequest(http.MethodDelete, url+"/api/v1/silence/"+s.ID, nil)
			if err != nil {
				return err
			}

			delResp, err := client.Do(delReq)
			defer delResp.Body.Close()

			_, err = ioutil.ReadAll(delResp.Body)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
func (m *Manager) SendAlert(alertId, text, title, severity string) error {
	url := m.getAlertManagerEndpoint()

	alertList := model.Alerts{}
	a := &model.Alert{}
	a.Labels = map[model.LabelName]model.LabelValue{}
	a.Labels[model.LabelName("alert_id")] = model.LabelValue(alertId)
	a.Labels[model.LabelName("text")] = model.LabelValue(text)
	a.Labels[model.LabelName("title")] = model.LabelValue(title)
	a.Labels[model.LabelName("severity")] = model.LabelValue(severity)

	alertList = append(alertList, a)

	alertData, err := json.Marshal(alertList)
	if err != nil {
		return err
	}

	resp, err := http.Post(url+"/api/alerts", "application/json", bytes.NewBuffer(alertData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}
