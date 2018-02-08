package alert

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/rancher/norman/parse"
	"github.com/rancher/norman/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NotifierCollectionFormatter(apiContext *types.APIContext, collection *types.GenericCollection) {
	collection.AddAction(apiContext, "send")
}

func NotifierFormatter(apiContext *types.APIContext, resource *types.RawResource) {
	resource.AddAction(apiContext, "send")
}

func (h *Handler) NotifierActionHandler(actionName string, action *types.Action, apiContext *types.APIContext) error {
	switch actionName {
	case "send":

		return h.testNotifier(actionName, action, apiContext)
	}

	return errors.Errorf("bad action %v", actionName)

}

func (h *Handler) testNotifier(actionName string, action *types.Action, apiContext *types.APIContext) error {
	actionInput, err := parse.ReadBody(apiContext.Request)
	if err != nil {
		return err
	}

	msg := ""
	msgInf, exist := actionInput["message"]
	if exist {
		msg = msgInf.(string)
	}

	if apiContext.ID != "" {
		parts := strings.Split(apiContext.ID, ":")
		ns := parts[0]
		id := parts[1]
		notifier, err := h.Management.Management.Notifiers("").GetNamespaced(ns, id, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if notifier.Spec.SlackConfig.URL != "" {
			if err := testSlack(notifier.Spec.SlackConfig.URL, msg); err != nil {
				return err
			}
			return nil
		}

		if notifier.Spec.SmtpConfig.Host != "" {
			s := notifier.Spec.SmtpConfig
			if err := testEmail(s.Host, s.Password, s.Username, int(s.Port), s.TLS, msg, s.DefaultRecipient); err != nil {
				return err
			}
			return nil
		}

		if notifier.Spec.PagerdutyConfig.ServiceKey != "" {
			if err := testPagerduty(notifier.Spec.PagerdutyConfig.ServiceKey, msg); err != nil {
				return err
			}
			return nil
		}

		if notifier.Spec.WebhookConfig.URL != "" {
			if err := testWebhook(notifier.Spec.WebhookConfig.URL, msg); err != nil {
				return err
			}
			return nil
		}

	} else {

		slackConfigInterface, exist := actionInput["slackConfig"]
		if exist {
			slackConfig, ok := slackConfigInterface.(map[string]interface{})
			if ok {
				url := slackConfig["url"].(string)
				if err := testSlack(url, msg); err != nil {
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
				port, _ := smtpConfig["port"].(json.Number).Int64()
				password := smtpConfig["password"].(string)
				username := smtpConfig["username"].(string)
				receiver := smtpConfig["defaultRecipient"].(string)
				tls := smtpConfig["tls"].(bool)
				if err := testEmail(host, password, username, int(port), tls, msg, receiver); err != nil {
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
				if err := testWebhook(url, msg); err != nil {
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
				if err := testPagerduty(key, msg); err != nil {
					return err
				}
				return nil
			}
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

func testPagerduty(key, msg string) error {
	if msg == "" {
		msg = "test pagerduty service key"
	}

	pd := &pagerDutyMessage{
		ServiceKey:  key,
		EventType:   "trigger",
		IncidentKey: hashKey("key"),
		Description: msg,
	}

	url := "https://events.pagerduty.com/generic/2010-04-15/create_event.json"

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(pd); err != nil {
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

func testWebhook(url, msg string) error {
	if msg == "" {
		msg = "test webhook"
	}
	alertList := model.Alerts{}
	a := &model.Alert{}
	a.Labels = map[model.LabelName]model.LabelValue{}
	a.Labels[model.LabelName("test_msg")] = model.LabelValue(msg)

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

func testSlack(url, msg string) error {
	if msg == "" {
		msg = "test slack webhook"
	}
	req := struct {
		Text string `json:"text"`
	}{}

	req.Text = msg

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

func testEmail(host, password, username string, port int, requireTLS bool, msg, receiver string) error {
	var c *smtp.Client
	smartHost := host + ":" + strconv.Itoa(port)

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

	if msg == "" {
		msg = "smtp server configuation validation"
	}

	addrs, err := mail.ParseAddressList(username)
	if err != nil {
		return fmt.Errorf("parsing from addresses: %s", err)
	}
	if len(addrs) != 1 {
		return fmt.Errorf("must be exactly one from address")
	}
	if err := c.Mail(addrs[0].Address); err != nil {
		return fmt.Errorf("sending mail from: %s", err)
	}
	addrs, err = mail.ParseAddressList(receiver)
	if err != nil {
		return fmt.Errorf("parsing to addresses: %s", err)
	}
	for _, addr := range addrs {
		if err := c.Rcpt(addr.Address); err != nil {
			return fmt.Errorf("sending rcpt to: %s", err)
		}
	}

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		return err
	}
	defer wc.Close()

	buffer := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(buffer)

	fmt.Fprintf(wc, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(wc, "Content-Type: multipart/alternative;  boundary=%s\r\n", multipartWriter.Boundary())
	fmt.Fprintf(wc, "MIME-Version: 1.0\r\n")

	// TODO: Add some useful headers here, such as URL of the alertmanager
	// and active/resolved.
	fmt.Fprintf(wc, "\r\n")

	if len(msg) > 0 {
		// Text template
		w, err := multipartWriter.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=UTF-8"}})
		if err != nil {
			return fmt.Errorf("creating part for text template: %s", err)
		}

		_, err = w.Write([]byte(msg))
		if err != nil {
			return err
		}
	}

	multipartWriter.Close()
	wc.Write(buffer.Bytes())

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
