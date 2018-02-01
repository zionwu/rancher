package logging

import (
	"net"
	"os"
	"text/template"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"net/url"

	loggingconfig "github.com/rancher/rancher/pkg/logging/config"
	"github.com/rancher/rancher/pkg/logging/k8sutils"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"
)

const (
	ProjectIDAnnotation = "field.cattle.io/projectId"
)

const (
	elasticsearch = "elasticsearch"
	splunk        = "splunk"
	kafka         = "kafka"
	embedded      = "embedded"
	syslog        = "syslog"
)

var (
	basePath          = "/tmp"
	clusterConfigPath = basePath + "/cluster.conf"
	projectConfigPath = basePath + "/project.conf"
)

type WrapClusterLogging struct {
	CurrentTarget string
	v3.ClusterLoggingSpec
	WrapSyslog
	WrapSplunk
	WrapKafka
	WrapEmbedded
	WrapElasticsearch
}

type WrapProjectLogging struct {
	CurrentTarget string
	v3.ProjectLoggingSpec
	GrepNamespace string
	WrapSyslog
	WrapSplunk
	WrapKafka
	WrapElasticsearch
}

type WrapSyslog struct {
	Host string
	Port string
}

type WrapSplunk struct {
	Server string
	Scheme string
}

type WrapKafka struct {
	Zookeeper string
}

type WrapEmbedded struct {
	DateFormat string
}

type WrapElasticsearch struct {
	DateFormat string
}

func toWrapClusterLogging(currentTarget string, clusterLogging v3.ClusterLoggingSpec) (*WrapClusterLogging, error) {
	wp := WrapClusterLogging{
		CurrentTarget:      currentTarget,
		ClusterLoggingSpec: clusterLogging,
	}
	if clusterLogging.SyslogConfig != nil {
		u, err := url.Parse(clusterLogging.SyslogConfig.Endpoint)
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			return nil, errors.Wrapf(err, "parse endpoint failed %s", clusterLogging.SyslogConfig.Endpoint)
		}
		wp.WrapSyslog = WrapSyslog{
			Host: host,
			Port: port,
		}
	}

	if clusterLogging.SplunkConfig != nil {
		u, err := url.Parse(clusterLogging.SplunkConfig.Endpoint)
		if err != nil {
			return nil, errors.Wrapf(err, "parse endpoint failed %s", clusterLogging.SplunkConfig.Endpoint)
		}
		wp.WrapSplunk = WrapSplunk{
			Server: u.Host,
			Scheme: u.Scheme,
		}
	}

	if clusterLogging.KafkaConfig != nil {
		u, err := url.Parse(clusterLogging.KafkaConfig.ZookeeperEndpoint)
		if err != nil {
			return nil, errors.Wrapf(err, "parse endpoint failed %s", clusterLogging.SplunkConfig.Endpoint)
		}
		wp.WrapKafka = WrapKafka{
			Zookeeper: u.Host,
		}
	}

	if clusterLogging.ElasticsearchConfig != nil {
		wp.WrapElasticsearch.DateFormat = getDateFormat(clusterLogging.ElasticsearchConfig.DateFormat)
	}

	if clusterLogging.EmbeddedConfig != nil {
		wp.WrapEmbedded.DateFormat = getDateFormat(clusterLogging.EmbeddedConfig.DateFormat)
	}
	return &wp, nil
}

func getDateFormat(dateformat string) string {
	ToRealMap := map[string]string{
		"YYYY.MM.DD": "%Y.%m.%d",
		"YYYY.MM":    "%Y.%m.",
		"YYYY":       "%Y.",
	}
	if _, ok := ToRealMap[dateformat]; ok {
		return ToRealMap[dateformat]
	}
	return "%Y.%m.%d"
}

func toWrapProjectLogging(currentTarget, grepNamespace string, projectLogging v3.ProjectLoggingSpec) (*WrapProjectLogging, error) {
	wp := WrapProjectLogging{
		CurrentTarget:      currentTarget,
		ProjectLoggingSpec: projectLogging,
		GrepNamespace:      grepNamespace,
	}
	if projectLogging.SyslogConfig != nil {
		u, err := url.Parse(projectLogging.SyslogConfig.Endpoint)
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			return nil, errors.Wrapf(err, "parse endpoint failed %s", projectLogging.SyslogConfig.Endpoint)
		}
		wp.WrapSyslog = WrapSyslog{
			Host: host,
			Port: port,
		}
	}

	if projectLogging.SplunkConfig != nil {
		u, err := url.Parse(projectLogging.SplunkConfig.Endpoint)
		if err != nil {
			return nil, errors.Wrapf(err, "parse endpoint failed %s", projectLogging.SplunkConfig.Endpoint)
		}
		wp.WrapSplunk = WrapSplunk{
			Server: u.Host,
			Scheme: u.Scheme,
		}
	}
	if projectLogging.KafkaConfig != nil {
		u, err := url.Parse(projectLogging.KafkaConfig.ZookeeperEndpoint)
		if err != nil {
			return nil, errors.Wrapf(err, "parse endpoint failed %s", projectLogging.SplunkConfig.Endpoint)
		}
		wp.WrapKafka = WrapKafka{
			Zookeeper: u.Host,
		}
	}

	if projectLogging.ElasticsearchConfig != nil {
		wp.WrapElasticsearch.DateFormat = getDateFormat(projectLogging.ElasticsearchConfig.DateFormat)
	}
	return &wp, nil
}

func generateConfigFile(configPath, templatePath string, conf map[string]interface{}) error {
	w, err := os.Create(configPath)
	var t *template.Template
	t, err = template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	return t.Execute(w, conf)
}

func createFluentd(corev1 typedv1.CoreV1Interface, rbacv1beta1 rbacv1beta1.RbacV1beta1Interface, appv1beta2 appsv1beta2.AppsV1beta2Interface) (err error) {
	serviceAccount := k8sutils.NewFluentdServiceAccount(loggingconfig.FluentdName, loggingconfig.LoggingNamespace)
	serviceAccount, err = corev1.ServiceAccounts(loggingconfig.LoggingNamespace).Create(serviceAccount)
	defer func() {
		if err != nil {
			corev1.ServiceAccounts(loggingconfig.LoggingNamespace).Delete(serviceAccount.Name, &metav1.DeleteOptions{})
		}
	}()
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	projectRoleBind := k8sutils.NewFluentdRoleBinding(loggingconfig.FluentdName, loggingconfig.LoggingNamespace)
	projectRoleBind, err = rbacv1beta1.ClusterRoleBindings().Create(projectRoleBind)
	defer func() {
		if err != nil {
			rbacv1beta1.ClusterRoleBindings().Delete(projectRoleBind.Name, &metav1.DeleteOptions{})
		}
	}()
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	daemonset := k8sutils.NewFluentdDaemonset(loggingconfig.FluentdName, loggingconfig.LoggingNamespace, loggingconfig.FluentdName)
	daemonset, err = appv1beta2.DaemonSets(loggingconfig.LoggingNamespace).Create(daemonset)
	defer func() {
		if err != nil {
			appv1beta2.DaemonSets(loggingconfig.LoggingNamespace).Delete(daemonset.Name, &metav1.DeleteOptions{})
		}
	}()
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func removeFluentd(corev1 typedv1.CoreV1Interface, rbacv1beta1 rbacv1beta1.RbacV1beta1Interface, appv1beta2 appsv1beta2.AppsV1beta2Interface) {
	deleteOp := metav1.DeletePropagationBackground
	err := appv1beta2.DaemonSets(loggingconfig.LoggingNamespace).Delete(loggingconfig.FluentdName, &metav1.DeleteOptions{PropagationPolicy: &deleteOp})
	if err != nil && !apierrors.IsNotFound(err) {
		logrus.Errorf("delete DaemonSets %s failed", loggingconfig.FluentdName)
	}
	err = corev1.ServiceAccounts(loggingconfig.LoggingNamespace).Delete(loggingconfig.FluentdName, &metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		logrus.Errorf("delete ServiceAccount %s failed", loggingconfig.FluentdName)
	}
	err = rbacv1beta1.ClusterRoleBindings().Delete(loggingconfig.FluentdName, &metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		logrus.Errorf("delete ClusterRoleBindings %s failed", loggingconfig.FluentdName)
	}
}

func updateConfigMap(configPath, loggingName, level string, corev1 typedv1.CoreV1Interface) error {
	configMap, err := k8sutils.BuildConfigMap(configPath, loggingconfig.LoggingNamespace, loggingName, level)
	if err != nil {
		logrus.Errorf("BuildConfigMap failed %v", err)
		return errors.Wrap(err, "BuildConfigMap failed")
	}
	existConfig, err := corev1.ConfigMaps(loggingconfig.LoggingNamespace).Get(loggingName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = corev1.ConfigMaps(loggingconfig.LoggingNamespace).Create(configMap)
		} else {
			return err
		}
	} else {
		newConfigMap := existConfig.DeepCopy()
		newConfigMap.Data = configMap.Data
		_, err = corev1.ConfigMaps(loggingconfig.LoggingNamespace).Update(newConfigMap)
	}
	return err
}

func InitData(corev1Client typedv1.CoreV1Interface) error {
	initNamespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: loggingconfig.LoggingNamespace,
			Labels: map[string]string{
				"cluster": "logging",
			},
		},
	}

	_, err := corev1Client.Namespaces().Create(&initNamespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	initClusterConfigMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: loggingconfig.ClusterLoggingName,
			Labels: map[string]string{
				"logging-level": "cluster",
			},
		},
		Data: map[string]string{
			"cluster.conf": "",
		},
	}
	_, err = corev1Client.ConfigMaps(loggingconfig.LoggingNamespace).Create(&initClusterConfigMap)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	initProjectConfigMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: loggingconfig.ProjectLoggingName,
			Labels: map[string]string{
				"logging-level": "project",
			},
		},
		Data: map[string]string{
			"project.conf": "",
		},
	}

	_, err = corev1Client.ConfigMaps(loggingconfig.LoggingNamespace).Create(&initProjectConfigMap)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func isAllLoggingDisable(clusterLoggingClient v3.ClusterLoggingInterface, projectLoggingClient v3.ProjectLoggingInterface, excludeClusterLogging, excludeProjectLogging string) (bool, error) {
	clusterLogging, err := clusterLoggingClient.List(metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	existClusterLogging := len(clusterLogging.Items)
	for _, v := range clusterLogging.Items {
		if v.Name == excludeClusterLogging {
			existClusterLogging = existClusterLogging - 1
		}
	}

	projectLogging, err := projectLoggingClient.List(metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	existProjectLogging := len(projectLogging.Items)
	for _, v := range projectLogging.Items {
		if v.Name == excludeProjectLogging {
			existProjectLogging = existProjectLogging - 1
		}
	}
	return existClusterLogging == 0 && existProjectLogging == 0, nil

}

func removeAllLogging(corev1 typedv1.CoreV1Interface, rbacv1beta1 rbacv1beta1.RbacV1beta1Interface, appv1beta2 appsv1beta2.AppsV1beta2Interface, clusterLoggingClient v3.ClusterLoggingInterface, projectLoggingClient v3.ProjectLoggingInterface, excludeClusterLogging, excludeProjectLogging string) error {
	allDisabled, err := isAllLoggingDisable(clusterLoggingClient, projectLoggingClient, excludeClusterLogging, excludeProjectLogging)
	if err != nil {
		return err
	}

	if allDisabled {
		removeFluentd(corev1, rbacv1beta1, appv1beta2)
	}
	return nil
}
