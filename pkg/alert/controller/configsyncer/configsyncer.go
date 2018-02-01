package configsyner

import (
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/rancher/norman/controller"
	alertconfig "github.com/rancher/rancher/pkg/alert/config"
	"github.com/rancher/rancher/pkg/alert/manager"
	"github.com/rancher/types/apis/core/v1"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func NewConfigSyncer(cluster *config.ClusterContext, manager *manager.Manager) *ConfigSyncer {
	return &ConfigSyncer{
		secretClient:       cluster.Core.Secrets("cattle-alerting"),
		clusterAlertLister: cluster.Management.Management.ClusterAlerts(cluster.ClusterName).Controller().Lister(),
		projectAlertLister: cluster.Management.Management.ProjectAlerts("").Controller().Lister(),
		notifierLister:     cluster.Management.Management.Notifiers(cluster.ClusterName).Controller().Lister(),
		clusterName:        cluster.ClusterName,
		alertManager:       manager,
	}

}

type ConfigSyncer struct {
	secretClient       v1.SecretInterface
	projectAlertLister v3.ProjectAlertLister
	clusterAlertLister v3.ClusterAlertLister
	notifierLister     v3.NotifierLister
	clusterName        string
	alertManager       *manager.Manager
}

func (d *ConfigSyncer) ProjectSync(key string, alert *v3.ProjectAlert) error {
	return d.sync()
}

func (d *ConfigSyncer) ClusterSync(key string, alert *v3.ClusterAlert) error {
	return d.sync()
}

func (d *ConfigSyncer) NotifierSync(key string, alert *v3.Notifier) error {
	return d.sync()
}

func (d *ConfigSyncer) sync() error {
	logrus.Info("start sync config")

	clusterAlerts, err := d.clusterAlertLister.List("", labels.NewSelector())
	if err != nil {
		logrus.Errorf("Error occured while getting cluster alerts: %v", err)
		return errors.Wrapf(err, "Creating cluster alerts")
	}

	projectAlerts, err := d.projectAlertLister.List("", labels.NewSelector())
	if err != nil {
		logrus.Errorf("Error occured while getting project alerts: %v", err)
		return errors.Wrapf(err, "Creating project alerts")
	}

	pAlerts := []*v3.ProjectAlert{}
	for _, alert := range projectAlerts {
		if controller.ObjectInCluster(d.clusterName, alert) {
			pAlerts = append(pAlerts, alert)
		}
	}

	notifiers, err := d.notifierLister.List("", labels.NewSelector())
	if err != nil {
		logrus.Errorf("Error occured while getting project notifier: %v", err)
		return errors.Wrapf(err, "Creating project alerts")
	}

	config := d.alertManager.GetDefaultConfig()
	config.Global.PagerdutyURL = "https://events.pagerduty.com/generic/2010-04-15/create_event.json"

	d.addClusterAlert2Config(config, clusterAlerts, notifiers)
	d.addProjectAlert2Config(config, pAlerts, notifiers)

	data, err := yaml.Marshal(config)
	logrus.Infof("after updating notifier: %s", string(data))
	if err != nil {
		logrus.Errorf("Error occured while marshal: %v", err)
		return errors.Wrapf(err, "Marshal secrets")
	}

	configSecret, err := d.secretClient.Get("alertmanager", metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("Error occured getting secret: %v", err)
		return errors.Wrapf(err, "Get secrets")
	}

	configSecret.Data["config.yml"] = data
	_, err = d.secretClient.Update(configSecret)
	if err != nil {
		logrus.Errorf("Error occured while update secret: %v", err)
		return errors.Wrapf(err, "Update secrets")
	}

	go func() {
		for i := 0; i < 10; i++ {
			d.alertManager.ReloadConfiguration()
			time.Sleep(10 * time.Second)
		}

	}()

	return nil
}

func (d *ConfigSyncer) getNotifier(id string, notifiers []*v3.Notifier) *v3.Notifier {

	for _, n := range notifiers {
		//TODO: check if this is correct
		if d.clusterName+":"+n.Name == id {
			return n
		}
	}

	return nil
}

func (d *ConfigSyncer) addProjectAlert2Config(config *alertconfig.Config, alerts []*v3.ProjectAlert, notifiers []*v3.Notifier) {
	for _, alert := range alerts {
		if alert.Status.State == "inactive" {
			continue
		}

		id := alert.Namespace + "-" + alert.Name

		receiver := &alertconfig.Receiver{Name: id}
		config.Receivers = append(config.Receivers, receiver)
		d.addRecipients(notifiers, receiver, alert.Spec.Recipients)

		d.addRoute(config, id, alert.Spec.InitialWaitSeconds, alert.Spec.RepeatIntervalSeconds)
	}
}

func (d *ConfigSyncer) addClusterAlert2Config(config *alertconfig.Config, alerts []*v3.ClusterAlert, notifiers []*v3.Notifier) {
	for _, alert := range alerts {
		if alert.Status.State == "inactive" {
			continue
		}

		id := alert.Namespace + "-" + alert.Name

		receiver := &alertconfig.Receiver{Name: id}
		config.Receivers = append(config.Receivers, receiver)
		d.addRecipients(notifiers, receiver, alert.Spec.Recipients)

		d.addRoute(config, id, alert.Spec.InitialWaitSeconds, alert.Spec.RepeatIntervalSeconds)
	}
}

func (d *ConfigSyncer) addRoute(config *alertconfig.Config, id string, initalWait, repeatInterval int) {
	routes := config.Route.Routes
	if routes == nil {
		routes = []*alertconfig.Route{}
	}

	match := map[string]string{}
	match["alert_id"] = id
	route := &alertconfig.Route{
		Receiver: id,
		Match:    match,
	}

	gw := model.Duration(time.Duration(initalWait) * time.Second)
	route.GroupWait = &gw
	ri := model.Duration(time.Duration(repeatInterval) * time.Second)
	route.RepeatInterval = &ri

	routes = append(routes, route)
	config.Route.Routes = routes
}

func (d *ConfigSyncer) addRecipients(notifiers []*v3.Notifier, receiver *alertconfig.Receiver, recipients []v3.Recipient) {
	for _, r := range recipients {
		if r.NotifierId != "" {
			notifier := d.getNotifier(r.NotifierId, notifiers)
			if notifier == nil {
				logrus.Errorf("Can not find the notifier %s", r.NotifierId)
				continue
			}
			if notifier.Spec.PagerdutyConfig != nil {
				pagerduty := &alertconfig.PagerdutyConfig{
					ServiceKey:  alertconfig.Secret(notifier.Spec.PagerdutyConfig.ServiceKey),
					Description: "{{ (index .Alerts 0).Labels.description}}",
				}
				if r.Recipient != "" {
					pagerduty.ServiceKey = alertconfig.Secret(r.Recipient)
				}
				receiver.PagerdutyConfigs = append(receiver.PagerdutyConfigs, pagerduty)

			} else if notifier.Spec.WebhookConfig != nil {
				webhook := &alertconfig.WebhookConfig{
					URL: notifier.Spec.WebhookConfig.URL,
				}
				if r.Recipient != "" {
					webhook.URL = r.Recipient
				}
				receiver.WebhookConfigs = append(receiver.WebhookConfigs, webhook)
			} else if notifier.Spec.SlackConfig != nil {
				slack := &alertconfig.SlackConfig{
					APIURL:  alertconfig.Secret(notifier.Spec.SlackConfig.URL),
					Channel: notifier.Spec.SlackConfig.DefaultRecipient,
					Text:    "{{ (index .Alerts 0).Labels.text}}\n",
					Title:   "{{ (index .Alerts 0).Labels.title}}\n",
					//Pretext: "Alert From Rancher",
					Color: `{{ if eq (index .Alerts 0).Labels.severity "critical" }}danger{{ else if eq (index .Alerts 0).Labels.severity "warning" }}warning{{ else }}good{{ end }}`,
				}
				if r.Recipient != "" {
					slack.Channel = r.Recipient
				}
				receiver.SlackConfigs = append(receiver.SlackConfigs, slack)

			} else if notifier.Spec.SmtpConfig != nil {
				header := map[string]string{}
				header["Subject"] = "Alert from Rancher: {{ (index .Alerts 0).Labels.description}}"
				email := &alertconfig.EmailConfig{
					Smarthost:    notifier.Spec.SmtpConfig.Host + ":" + strconv.Itoa(notifier.Spec.SmtpConfig.Port),
					AuthPassword: alertconfig.Secret(notifier.Spec.SmtpConfig.Password),
					AuthUsername: notifier.Spec.SmtpConfig.Username,
					RequireTLS:   &notifier.Spec.SmtpConfig.TLS,
					To:           notifier.Spec.SmtpConfig.DefaultRecipient,
					Headers:      header,
				}
				if r.Recipient != "" {
					email.To = r.Recipient
				}
				receiver.EmailConfigs = append(receiver.EmailConfigs, email)
			}

		}
	}

}
