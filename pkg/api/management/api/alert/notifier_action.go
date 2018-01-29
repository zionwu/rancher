package alert

import (
	"github.com/pkg/errors"
	"github.com/rancher/norman/types"
	"github.com/sirupsen/logrus"
)

func NotifierCollectionFormatter(apiContext *types.APIContext, collection *types.GenericCollection) {
	collection.AddAction(apiContext, "send")
}

func NotifierActionHandler(actionName string, action *types.Action, apiContext *types.APIContext) error {
	logrus.Infof("do activity action:%s", actionName)

	switch actionName {
	case "send":
		return testNotifier(actionName, action, apiContext)
	}

	return errors.Errorf("bad action %v", actionName)

}

func testNotifier(actionName string, action *types.Action, apiContext *types.APIContext) error {
	return nil
}
