package logging

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"

	loggingconfig "github.com/rancher/rancher/pkg/logging/config"
	"github.com/rancher/rancher/pkg/logging/generator"
)

type ProjectLoggingLifecycle struct {
	clusterName          string
	management           config.ManagementContext
	appv1beta2           appsv1beta2.AppsV1beta2Interface
	corev1               typedv1.CoreV1Interface
	rbacv1beta1          rbacv1beta1.RbacV1beta1Interface
	projectLoggingClient v3.ProjectLoggingInterface
	clusterLoggingClient v3.ClusterLoggingInterface
}

func RegisterProjectLogging(cluster *config.ClusterContext) {
	projectLoggingClient := cluster.Management.Management.ProjectLoggings("")
	lifecycle := &ProjectLoggingLifecycle{
		clusterName:          cluster.ClusterName,
		rbacv1beta1:          cluster.K8sClient.RbacV1beta1(),
		management:           *cluster.Management,
		appv1beta2:           cluster.K8sClient.AppsV1beta2(),
		corev1:               cluster.K8sClient.CoreV1(),
		projectLoggingClient: projectLoggingClient,
		clusterLoggingClient: cluster.Management.Management.ClusterLoggings(""),
	}

	projectLoggingClient.AddClusterScopedLifecycle("project-logging-controller", cluster.ClusterName, lifecycle)
}

func (c *ProjectLoggingLifecycle) Create(obj *v3.ProjectLogging) (*v3.ProjectLogging, error) {
	err := createOrUpdateProjectConfigMap(c.projectLoggingClient, c.corev1, "")
	if err != nil {
		return nil, err
	}

	return obj, createFluentd(c.corev1, c.rbacv1beta1, c.appv1beta2)
}

func (c *ProjectLoggingLifecycle) Remove(obj *v3.ProjectLogging) (*v3.ProjectLogging, error) {
	err := createOrUpdateProjectConfigMap(c.projectLoggingClient, c.corev1, obj.Name)
	if err != nil {
		logrus.Errorf("before remove project logging update configmap error %s", err)
		return nil, err
	}
	return obj, removeAllLogging(c.corev1, c.rbacv1beta1, c.appv1beta2, c.clusterLoggingClient, c.projectLoggingClient, "", obj.Name)
}

func (c *ProjectLoggingLifecycle) Updated(obj *v3.ProjectLogging) (*v3.ProjectLogging, error) {
	return obj, createOrUpdateProjectConfigMap(c.projectLoggingClient, c.corev1, "")
}

func createOrUpdateProjectConfigMap(projectLoggingClient v3.ProjectLoggingInterface, corev1 typedv1.CoreV1Interface, exclude string) error {
	err := generateProjectConfigFile(projectLoggingClient, corev1, exclude)
	if err != nil {
		return errors.Wrap(err, "generate project config file failed")
	}
	return updateConfigMap(projectConfigPath, loggingconfig.ProjectLoggingName, "project", corev1)
}

func generateProjectConfigFile(projectLoggingClient v3.ProjectLoggingInterface, corev1 typedv1.CoreV1Interface, exclude string) error {
	projectLoggings, err := projectLoggingClient.List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "list project logging failed")
	}

	ns, err := corev1.Namespaces().List(metav1.ListOptions{})
	var wl []WrapProjectLogging
	for _, v := range projectLoggings.Items {
		if v.Name == exclude {
			continue
		}
		var grepNamespace []string
		for _, v2 := range ns.Items {
			nsProjectName := v2.Annotations[ProjectIDAnnotation]
			if nsProjectName == v.Spec.ProjectName {
				grepNamespace = append(grepNamespace, v2.Name)
			}
		}

		formatgrepNamespace := fmt.Sprintf("(%s)", strings.Join(grepNamespace, "|"))
		currentTarget := getProjectTarget(&v.Spec)
		projectLogging, err := toWrapProjectLogging(currentTarget, formatgrepNamespace, v.Spec)
		if err != nil {
			return err
		}
		wl = append(wl, *projectLogging)

	}
	conf := make(map[string]interface{})
	conf["projectTargets"] = wl
	return generator.GenerateConfigFile(projectConfigPath, generator.ProjectTemplate, "project", conf)
}

func getProjectTarget(spec *v3.ProjectLoggingSpec) string {
	if spec.ElasticsearchConfig != nil {
		return "elasticsearch"
	} else if spec.SplunkConfig != nil {
		return "splunk"
	} else if spec.KafkaConfig != nil {
		return "kafka"
	} else if spec.SyslogConfig != nil {
		return "syslog"
	}
	return "none"
}
