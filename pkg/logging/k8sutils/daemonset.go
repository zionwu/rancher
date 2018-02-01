package k8sutils

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"k8s.io/api/apps/v1beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	loggingconfig "github.com/rancher/rancher/pkg/logging/config"
)

func BuildConfigMap(configPath, namespace, name, level string) (*v1.ConfigMap, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, errors.Wrapf(err, "find cluster logging configuration file failed")
	}
	defer file.Close()
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Wrapf(err, "read cluster logging configuration file failed")
	}
	configFile := string(buf)

	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"logging-level": level,
			},
		},
		Data: map[string]string{
			level + ".conf": configFile,
		},
	}, nil
}

func NewFluentdServiceAccount(name, namespace string) *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func NewFluentdRoleBinding(name, namespace string) *v1beta1.ClusterRoleBinding {
	return &v1beta1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Subjects: []v1beta1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: namespace,
			},
		},
		RoleRef: v1beta1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func NewFluentdDaemonset(name, namespace, clusterName string) *v1beta2.DaemonSet {
	privileged := true
	terminationGracePeriodSeconds := int64(30)
	return &v1beta2.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"k8s-app": "fluentd-logging",
				// "version":                       "v1",
				// "kubernetes.io/cluster-service": "true",
			},
		},
		Spec: v1beta2.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": "fluentd-logging",
					// "version":                       "v1",
					// "kubernetes.io/cluster-service": "true",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Labels: map[string]string{
						"k8s-app": "fluentd-logging",
						// "version":                       "v1",
						// "kubernetes.io/cluster-service": "true",
					},
				},
				Spec: v1.PodSpec{
					Tolerations: []v1.Toleration{{
						Key:    "node-role.kubernetes.io/master",
						Effect: v1.TaintEffectNoSchedule,
					}},
					Containers: []v1.Container{
						{
							Name:            "fluentd-helper",
							Image:           "micheliac/fluentd-helper:v0.0.0.1",
							Command:         []string{"fluentd-helper"},
							Args:            []string{"--watched-file-list", "/fluentd/etc/config/cluster", "--watched-file-list", "/fluentd/etc/config/project"},
							ImagePullPolicy: v1.PullAlways,
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "clusterlogging",
									MountPath: "/fluentd/etc/config/cluster",
								},
								{
									Name:      "projectlogging",
									MountPath: "/fluentd/etc/config/project",
								},
							},
						},
						{
							Name:            "fluentd",
							Image:           "micheliac/fluentd-base:v0.0.0.2", //todo: change images
							ImagePullPolicy: v1.PullIfNotPresent,
							Command:         []string{"fluentd"},
							Args:            []string{"-c", "/fluentd/etc/config/fluent.conf"},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "varlibdockercontainers",
									MountPath: "/var/lib/docker/containers",
								},
								{
									Name:      "varlogcontainers",
									MountPath: "/var/log/containers",
								},
								{
									Name:      "varlogpods",
									MountPath: "/var/log/pods",
								},
								{
									Name:      "fluentdlog",
									MountPath: "/fluentd/etc/log",
								},
								{
									Name:      "clusterlogging",
									MountPath: "/fluentd/etc/config/cluster",
								},
								{
									Name:      "projectlogging",
									MountPath: "/fluentd/etc/config/project",
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
							},
						},
					},
					ServiceAccountName:            loggingconfig.FluentdName,
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Volumes: []v1.Volume{
						{
							Name: "varlibdockercontainers",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/docker/containers",
								},
							},
						},
						{
							Name: "varlogcontainers",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/log/containers",
								},
							},
						},
						{
							Name: "varlogpods",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/log/pods",
								},
							},
						},
						{
							Name: "fluentdlog",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/log/fluentd",
								},
							},
						},
						{
							Name: "clusterlogging",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: loggingconfig.ClusterLoggingName,
									},
									Items: []v1.KeyToPath{
										{
											Key:  "cluster.conf",
											Path: "cluster.conf",
										},
									},
								},
							},
						},
						{
							Name: "projectlogging",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: loggingconfig.ProjectLoggingName,
									},
									Items: []v1.KeyToPath{
										{
											Key:  "project.conf",
											Path: "project.conf",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
