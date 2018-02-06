package alert

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/pkg/errors"
	"github.com/rancher/norman/types"

	"github.com/rancher/norman/parse"
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

	emailConfigInterface, exist := actionInput["smtpConfig"]
	if exist {
		emailConfig, ok := emailConfigInterface.(map[string]interface{})
		if ok {
			host := emailConfig["host"].(string)
			port := emailConfig["port"].(string)
			password := emailConfig["password"].(string)
			username := emailConfig["username"].(string)
			if err := testEmail(host, port, password, username); err != nil {
				return err
			}
			return nil
		}
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
func testEmail(host, port, password, username string) error {
	var c *smtp.Client
	smartHost := host + ":" + port

	if port == "465" || port == "587" {
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
