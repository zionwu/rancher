package logging

import (
	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	appsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"

	loggingconfig "github.com/rancher/rancher/pkg/logging/config"
	"github.com/rancher/rancher/pkg/logging/k8sutils"
)

type ClusterLoggingLifecycle struct {
	management           config.ManagementContext
	appv1beta2           appsv1beta2.AppsV1beta2Interface
	corev1               typedv1.CoreV1Interface
	rbacv1beta1          rbacv1beta1.RbacV1beta1Interface
	clusterLoggingClient v3.ClusterLoggingInterface
	projectLoggingClient v3.ProjectLoggingInterface
}

func RegisterClusterLogging(cluster *config.ClusterContext) {
	clusterloggingClient := cluster.Management.Management.ClusterLoggings("")
	lifecycle := &ClusterLoggingLifecycle{
		rbacv1beta1:          cluster.K8sClient.RbacV1beta1(),
		management:           *cluster.Management,
		appv1beta2:           cluster.K8sClient.AppsV1beta2(),
		corev1:               cluster.K8sClient.CoreV1(),
		clusterLoggingClient: clusterloggingClient,
		projectLoggingClient: cluster.Management.Management.ProjectLoggings(""),
	}

	clusterloggingClient.AddClusterScopedLifecycle("cluster-logging-controller", cluster.ClusterName, lifecycle)
}

func (c *ClusterLoggingLifecycle) Create(obj *v3.ClusterLogging) (*v3.ClusterLogging, error) {
	target := getClusterTarget(&obj.Spec)
	if target == "embedded" {
		c.createEmbeddedTarget(loggingconfig.LoggingNamespace)
	}

	err := createOrUpdateClusterConfigMap(c.clusterLoggingClient, c.corev1, "")
	if err != nil {
		return nil, err
	}

	return obj, createFluentd(c.corev1, c.rbacv1beta1, c.appv1beta2)
}

func (c *ClusterLoggingLifecycle) Remove(obj *v3.ClusterLogging) (*v3.ClusterLogging, error) {
	err := createOrUpdateClusterConfigMap(c.clusterLoggingClient, c.corev1, obj.Name)
	if err != nil {
		return nil, err
	}
	return obj, removeAllLogging(c.corev1, c.rbacv1beta1, c.appv1beta2, c.clusterLoggingClient, c.projectLoggingClient)
}

func (c *ClusterLoggingLifecycle) Updated(obj *v3.ClusterLogging) (*v3.ClusterLogging, error) {
	target := getClusterTarget(&obj.Spec)
	if target == "embedded" {
		c.createEmbeddedTarget(loggingconfig.LoggingNamespace)
	}

	return obj, createOrUpdateClusterConfigMap(c.clusterLoggingClient, c.corev1, "")
}

func generateClusterConfigFile(clusterLoggingClient v3.ClusterLoggingInterface, exclude string) error {
	clusterLoggingList, err := clusterLoggingClient.List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	var clusterlogging v3.ClusterLogging
	if len(clusterLoggingList.Items) != 0 && clusterLoggingList.Items[0].Name != exclude {
		clusterlogging = clusterLoggingList.Items[0]
	}
	currentTarget := getClusterTarget(&clusterlogging.Spec)
	conf := make(map[string]interface{})
	wpClusterlogging, err := toWrapClusterLogging(currentTarget, clusterlogging.Spec)
	if err != nil {
		return err
	}
	conf["clusterTarget"] = wpClusterlogging
	return generateConfigFile(clusterConfigPath, clusterTemplatePath, conf)
}

func createOrUpdateClusterConfigMap(clusterLoggingClient v3.ClusterLoggingInterface, corev1 typedv1.CoreV1Interface, exclude string) error {
	err := generateClusterConfigFile(clusterLoggingClient, exclude)
	if err != nil {
		return errors.Wrap(err, "generate cluster config file failed")
	}

	return updateConfigMap(clusterConfigPath, loggingconfig.ClusterLoggingName, "cluster", corev1)
}

func (c *ClusterLoggingLifecycle) createEmbeddedTarget(namespace string) error {
	// create es deployment
	existESDep, err := c.appv1beta2.Deployments(namespace).List(metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", k8sutils.EmbeddedESName).String()})
	if err != nil {
		return errors.Wrapf(err, "get deployment %s fail", k8sutils.EmbeddedESName)
	}
	if len(existESDep.Items) == 0 {
		// create service account, role and rolebinding
		sc := k8sutils.NewESServiceAccount(namespace)
		role := k8sutils.NewESRole(namespace)
		roleBind := k8sutils.NewESRoleBinding(namespace)

		defer func() {
			if err != nil {
				c.corev1.ServiceAccounts(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
			}
		}()
		_, err = c.corev1.ServiceAccounts(namespace).Create(sc)
		if err != nil {
			return errors.Wrapf(err, "create service account %s fail", k8sutils.EmbeddedESName)
		}

		defer func() {
			if err != nil {
				c.rbacv1beta1.Roles(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
			}
		}()
		_, err = c.rbacv1beta1.Roles(namespace).Create(role)
		if err != nil {
			return errors.Wrapf(err, "create role %s fail", k8sutils.EmbeddedESName)
		}

		defer func() {
			if err != nil {
				c.rbacv1beta1.RoleBindings(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
			}
		}()
		_, err = c.rbacv1beta1.RoleBindings(namespace).Create(roleBind)
		if err != nil {
			return errors.Wrapf(err, "create role %s fail", k8sutils.EmbeddedESName)
		}

		defer func() {
			if err != nil {
				c.corev1.Services(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
			}
		}()
		// create service and deployment
		newService := k8sutils.NewESService(namespace)
		_, err = c.corev1.Services(namespace).Create(newService)
		if err != nil {
			return errors.Wrapf(err, "create service %s fail", k8sutils.EmbeddedESName)
		}

		defer func() {
			if err != nil {
				c.appv1beta2.Deployments(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
			}
		}()
		esDeployment := k8sutils.NewESDeployment(namespace)
		_, err = c.appv1beta2.Deployments(namespace).Create(esDeployment)
		if err != nil {
			return errors.Wrapf(err, "create deployment %s fail", k8sutils.EmbeddedESName)
		}
	} else {
		logrus.Info("Embedded Elasticsearch already deployed")
	}

	// create kibana deployment
	// var existKibanaDep *v1beta1.DeploymentList
	existKibanaDep, err := c.appv1beta2.Deployments(namespace).List(metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", k8sutils.EmbeddedKibanaName).String()})
	if err != nil {
		return errors.Wrapf(err, "get deployment %s fail", k8sutils.EmbeddedKibanaName)
	}
	if len(existKibanaDep.Items) == 0 {
		// create service account, role and rolebinding
		sc := k8sutils.NewKibanaServiceAccount(namespace)
		role := k8sutils.NewKibanaRole(namespace)
		roleBind := k8sutils.NewKibanaRoleBinding(namespace)

		defer func() {
			if err != nil {
				c.corev1.ServiceAccounts(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
			}
		}()
		_, err = c.corev1.ServiceAccounts(namespace).Create(sc)
		if err != nil {
			return errors.Wrapf(err, "create service account  %s fail", k8sutils.EmbeddedKibanaName)
		}

		defer func() {
			if err != nil {
				c.rbacv1beta1.Roles(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
			}
		}()
		_, err = c.rbacv1beta1.Roles(namespace).Create(role)
		if err != nil {

			return errors.Wrapf(err, "create role %s fail", k8sutils.EmbeddedKibanaName)
		}

		defer func() {
			if err != nil {
				c.rbacv1beta1.RoleBindings(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
			}
		}()
		_, err = c.rbacv1beta1.RoleBindings(namespace).Create(roleBind)
		if err != nil {
			return errors.Wrapf(err, "create role %s fail", k8sutils.EmbeddedKibanaName)
		}

		defer func() {
			if err != nil {
				c.corev1.Services(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
			}
		}()
		newService := k8sutils.NewKibanaService(namespace)
		_, err = c.corev1.Services(namespace).Create(newService)
		if err != nil {
			return errors.Wrapf(err, "create service %s fail", k8sutils.EmbeddedKibanaName)
		}

		defer func() {
			if err != nil {
				c.appv1beta2.Deployments(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
			}
		}()
		kibanaDeployment := k8sutils.NewKibanaDeployment(namespace)
		_, err = c.appv1beta2.Deployments(namespace).Create(kibanaDeployment)
		if err != nil {
			return errors.Wrapf(err, "create deployment %s fail", k8sutils.EmbeddedKibanaName)
		}
	} else {
		logrus.Info("Embedded Kibana already deployed")
	}
	return nil
}

func (c *ClusterLoggingLifecycle) DeleteEmbeddedTarget(namespace string) error {
	//service account
	err := c.corev1.ServiceAccounts(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete service account %s fail", k8sutils.EmbeddedESName)
	}
	err = c.corev1.ServiceAccounts(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete service account %s fail", k8sutils.EmbeddedKibanaName)
	}

	//role
	err = c.rbacv1beta1.Roles(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete role %s fail", k8sutils.EmbeddedESName)
	}
	err = c.rbacv1beta1.Roles(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete role %s fail", k8sutils.EmbeddedKibanaName)
	}

	//rolebinding
	err = c.rbacv1beta1.RoleBindings(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete role %s fail", k8sutils.EmbeddedESName)
	}
	err = c.rbacv1beta1.RoleBindings(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete role %s fail", k8sutils.EmbeddedKibanaName)
	}

	//service
	err = c.corev1.Services(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete service %s fail", k8sutils.EmbeddedESName)
	}
	err = c.corev1.Services(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete service %s fail", k8sutils.EmbeddedKibanaName)
	}

	//deployment
	err = c.appv1beta2.Deployments(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete deployment %s fail", k8sutils.EmbeddedESName)
	}

	err = c.appv1beta2.Deployments(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete deployment %s fail", k8sutils.EmbeddedKibanaName)
	}
	return nil
}

func getClusterTarget(spec *v3.ClusterLoggingSpec) string {
	if spec.EmbeddedConfig != nil {
		return "embedded"
	} else if spec.ElasticsearchConfig != nil {
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
