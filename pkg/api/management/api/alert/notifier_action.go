package alert

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/rancher/norman/parse"
	"github.com/rancher/norman/types"
)

func NotifierCollectionFormatter(apiContext *types.APIContext, collection *types.GenericCollection) {
	collection.AddAction(apiContext, "send")
}

func NotifierActionHandler(actionName string, action *types.Action, apiContext *types.APIContext) error {
	switch actionName {
	case "send":
		return testNotifier(actionName, action, apiContext)
	}

	return errors.Errorf("bad action %v", actionName)

}

func testNotifier(actionName string, action *types.Action, apiContext *types.APIContext) error {

	actionInput, err := parse.ReadBody(apiContext.Request)
	if err != nil {
		return err
	}
	slackConfigInterface, exist := actionInput["slackConfig"]
	if exist {
		slackConfig, ok := slackConfigInterface.(map[string]interface{})
		if ok {
			url := slackConfig["url"].(string)
			if err := testSlack(url); err != nil {
				return err
			}
			return nil
		}
	}

	smtpConfigInterface, exist := actionInput["smtpConfig"]
	if exist {
		smtpConfig, ok := smtpConfigInterface.(map[string]interface{})
		if ok {
			host := smtpConfig["host"].(string)
			port := smtpConfig["port"].(string)
			password := smtpConfig["password"].(string)
			username := smtpConfig["username"].(string)
			tls := smtpConfig["tls"].(bool)
			if err := testEmail(host, port, password, username, tls); err != nil {
				return err
			}
			return nil
		}
	}

	webhookConfigInterface, exist := actionInput["webhookConfig"]
	if exist {
		webhookConfig, ok := webhookConfigInterface.(map[string]interface{})
		if ok {
			url := webhookConfig["url"].(string)
			if err := testWebhook(url); err != nil {
				return err
			}
			return nil
		}
	}

	pagerdutyConfigInterface, exist := actionInput["pagerdutyConfig"]
	if exist {
		pagerdutyConfig, ok := pagerdutyConfigInterface.(map[string]interface{})
		if ok {
			key := pagerdutyConfig["serviceKey"].(string)
			if err := testPagerduty(key); err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

type pagerDutyMessage struct {
	RoutingKey  string `json:"routing_key,omitempty"`
	ServiceKey  string `json:"service_key,omitempty"`
	DedupKey    string `json:"dedup_key,omitempty"`
	IncidentKey string `json:"incident_key,omitempty"`
	EventType   string `json:"event_type,omitempty"`
	Description string `json:"description,omitempty"`
}

func hashKey(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func testPagerduty(key string) error {

	msg := &pagerDutyMessage{
		ServiceKey:  key,
		EventType:   "trigger",
		IncidentKey: hashKey("key"),
		Description: "test pagerduty service key",
	}

	url := "https://events.pagerduty.com/generic/2010-04-15/create_event.json"

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(msg); err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status code is not 200")
	}

	return nil
}

func testWebhook(url string) error {
	alertList := model.Alerts{}
	a := &model.Alert{}
	a.Labels = map[model.LabelName]model.LabelValue{}
	a.Labels[model.LabelName("test_msg")] = model.LabelValue("test webhook")

	alertList = append(alertList, a)

	alertData, err := json.Marshal(alertList)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(alertData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status code is not 200")
	}

	return nil
}

func testSlack(url string) error {
	req := struct {
		Text string `json:"text"`
	}{}

	req.Text = "webhook validation"

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status code is not 200")
	}

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if string(res) != "ok" {
		return fmt.Errorf("http response is not ok")
	}

	return nil
}
func testEmail(host, port, password, username string, requireTLS bool) error {
	var c *smtp.Client
	smartHost := host + ":" + port

	if requireTLS {
		conn, err := tls.Dial("tcp", smartHost, &tls.Config{ServerName: host})
		if err != nil {
			return err
		}
		c, err = smtp.NewClient(conn, smartHost)
		if err != nil {
			return err
		}

	} else {
		// Connect to the SMTP smarthost.
		c, err := smtp.Dial(smartHost)
		if err != nil {
			return err
		}
		defer c.Quit()
	}
	if ok, mech := c.Extension("AUTH"); ok {
		auth, err := auth(mech, username, password)
		if err != nil {
			return err
		}
		if auth != nil {
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("%T failed: %s", auth, err)
			}
		}
	}

	return nil
}

func auth(mechs string, username, password string) (smtp.Auth, error) {

	for _, mech := range strings.Split(mechs, " ") {
		switch mech {
		case "LOGIN":
			if password == "" {
				continue
			}

			return &loginAuth{username, password}, nil
		}
	}
	return nil, fmt.Errorf("smtp server does not support login auth")
}

type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

// Used for AUTH LOGIN. (Maybe password should be encrypted)
func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch strings.ToLower(string(fromServer)) {
		case "username:":
			return []byte(a.username), nil
		case "password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unexpected server challenge")
		}
	}
	return nil, nil
}
