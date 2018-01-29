package controller

import (
	"github.com/pkg/errors"
	"github.com/rancher/types/apis/apps/v1beta2"
	"github.com/rancher/types/apis/core/v1"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	alertManager = "alertmanager"
	alertConfig  = `global:
  slack_api_url: 'slack.com'
templates:
  - '/etc/alertmanager-templates/*.tmpl'
route:
  receiver: 'rancherlas'
  group_wait: 3m
  group_interval: 3m
  repeat_interval: 1h
receivers:
  - name: 'rancherlas'
    slack_configs:
    - channel: '#alert'`
)

func Register(cluster *config.ClusterContext) {
	clusterAlertClient := cluster.Management.Management.ClusterAlerts("")
	clusterAlertLifecycle := &ClusterAlertLifecycle{}
	clusterAlertClient.AddLifecycle("cluster-alert-init-controller", clusterAlertLifecycle)

	projectAlertClient := cluster.Management.Management.ProjectAlerts("")
	projectAlertLifecycle := &ProjectAlertLifecycle{}
	projectAlertClient.AddLifecycle("project-alert-init-controller", projectAlertLifecycle)

	deployer := &Deployer{
		nsClient:     cluster.Core.Namespaces(""),
		appsClient:   cluster.Apps.Deployments(""),
		secretClient: cluster.Core.Secrets(""),
		svcClient:    cluster.Core.Services(""),
	}

	clusterAlertClient.AddClusterScopedHandler("cluster-alert-deployer", cluster.ClusterName, deployer.clusterSync)
	projectAlertClient.AddClusterScopedHandler("project-alert-deployer", cluster.ClusterName, deployer.projectSync)
	//clusterAlertClient.AddClusterScopedHandler("project-aPo

}

type Deployer struct {
	nsClient     v1.NamespaceInterface
	appsClient   v1beta2.DeploymentInterface
	secretClient v1.SecretInterface
	svcClient    v1.ServiceInterface
}

func (d *Deployer) projectSync(key string, alert *v3.ProjectAlert) error {
	return d.deploy()
}

func (d *Deployer) clusterSync(key string, alert *v3.ClusterAlert) error {
	return d.deploy()
}

func (d *Deployer) deploy() error {

	//TODO: cleanup resources while there is not any alert configured
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cattle-alerting",
		},
	}
	if _, err := d.nsClient.Create(ns); err != nil && !apierrors.IsAlreadyExists(err) {
		logrus.Errorf("Error occured while create ns: %v", err)
		return errors.Wrapf(err, "Creating ns")
	}

	secret := getSecret()
	if _, err := d.secretClient.Create(secret); err != nil && !apierrors.IsAlreadyExists(err) {
		logrus.Errorf("Error occured while create secret: %v", err)
		return errors.Wrapf(err, "Creating secret")
	}

	deployment := getDeployment()
	if _, err := d.appsClient.Create(deployment); err != nil && !apierrors.IsAlreadyExists(err) {
		logrus.Errorf("Error occured while create deployment: %v", err)
		return errors.Wrapf(err, "Creating deployment")
	}

	service := getService()
	if _, err := d.svcClient.Create(service); err != nil && !apierrors.IsAlreadyExists(err) {
		logrus.Errorf("Error occured while create service: %v", err)
		return errors.Wrapf(err, "Creating service")
	}

	return nil
}

func getSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "cattle-alerting",
			Name:      "alertmanager",
		},
		Data: map[string][]byte{
			"config.yml": []byte(alertConfig),
		},
	}
}

func getService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "cattle-alerting",
			Name:      "alertmanager",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"app": "alertmanager",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "alertmanager",
					Port: 9093,
				},
			},
		},
	}
}

func getDeployment() *appsv1beta2.Deployment {
	replicas := int32(1)
	return &appsv1beta2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "cattle-alerting",
			Name:      "alertmanager",
		},
		Spec: appsv1beta2.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "alertmanager"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "alertmanager"},
					Name:   "alertmanager",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "alertmanager",
							Image: "prom/alertmanager:v0.11.0",
							Args:  []string{"-config.file=/etc/alertmanager/config.yml", "-storage.path=/alertmanager"},
							Ports: []corev1.ContainerPort{
								{
									Name:          "alertmanager",
									ContainerPort: 9093,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "alertmanager",
									MountPath: "/alertmanager",
								},
								{
									Name:      "config-volume",
									MountPath: "/etc/alertmanager",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "alertmanager",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "alertmanager",
								},
							},
						},
					},
				},
			},
		},
	}
}

type ClusterAlertLifecycle struct {
}

func (l *ClusterAlertLifecycle) Create(obj *v3.ClusterAlert) (*v3.ClusterAlert, error) {
	obj.Status.State = "active"
	return obj, nil
}

func (l *ClusterAlertLifecycle) Updated(obj *v3.ClusterAlert) (*v3.ClusterAlert, error) {
	return obj, nil
}

func (l *ClusterAlertLifecycle) Remove(obj *v3.ClusterAlert) (*v3.ClusterAlert, error) {
	return obj, nil
}

type ProjectAlertLifecycle struct {
}

func (l *ProjectAlertLifecycle) Create(obj *v3.ProjectAlert) (*v3.ProjectAlert, error) {
	obj.Status.State = "active"
	return obj, nil
}

func (l *ProjectAlertLifecycle) Updated(obj *v3.ProjectAlert) (*v3.ProjectAlert, error) {
	return obj, nil
}

func (l *ProjectAlertLifecycle) Remove(obj *v3.ProjectAlert) (*v3.ProjectAlert, error) {
	return obj, nil
}
