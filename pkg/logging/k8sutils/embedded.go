package k8sutils

import (
	v1beta2 "k8s.io/api/apps/v1beta2"
	v1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	EmbeddedESName     = "elasticsearch"
	EmbeddedKibanaName = "kibana"
	esImage            = "quay.io/pires/docker-elasticsearch-kubernetes:5.6.2"
	kibanaImage        = "kibana:5.6.4"
)

func NewESServiceAccount(namespace string) *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedESName,
			Namespace: namespace,
		},
	}
}

func NewKibanaServiceAccount(namespace string) *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedKibanaName,
			Namespace: namespace,
		},
	}
}

func NewESRole(namespace string) *rbacv1beta1.Role {
	return &rbacv1beta1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedESName,
			Namespace: namespace,
		},
		Rules: []rbacv1beta1.PolicyRule{
			{
				APIGroups: []string{rbacv1beta1.APIGroupAll},
				Resources: []string{"endpoints"},
				Verbs:     []string{rbacv1beta1.VerbAll},
			},
		},
	}
}

func NewKibanaRole(namespace string) *rbacv1beta1.Role {
	return &rbacv1beta1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedKibanaName,
			Namespace: namespace,
		},
		Rules: []rbacv1beta1.PolicyRule{
			{
				APIGroups: []string{rbacv1beta1.APIGroupAll},
				Resources: []string{"endpoints"},
				Verbs:     []string{rbacv1beta1.VerbAll},
			},
		},
	}
}

func NewESRoleBinding(namespace string) *rbacv1beta1.RoleBinding {
	return &rbacv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedESName,
			Namespace: namespace,
		},
		RoleRef: rbacv1beta1.RoleRef{
			Name:     EmbeddedESName,
			Kind:     "Role",
			APIGroup: rbacv1beta1.GroupName,
		},
		Subjects: []rbacv1beta1.Subject{
			{
				Kind:      rbacv1beta1.ServiceAccountKind,
				Name:      EmbeddedESName,
				Namespace: namespace,
			},
		},
	}
}

func NewKibanaRoleBinding(namespace string) *rbacv1beta1.RoleBinding {
	return &rbacv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedKibanaName,
			Namespace: namespace,
		},
		RoleRef: rbacv1beta1.RoleRef{
			Name:     EmbeddedKibanaName,
			Kind:     "Role",
			APIGroup: rbacv1beta1.GroupName,
		},
		Subjects: []rbacv1beta1.Subject{
			{
				Kind:      rbacv1beta1.ServiceAccountKind,
				Name:      EmbeddedKibanaName,
				Namespace: namespace,
			},
		},
	}
}

func NewESService(namespace string) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      EmbeddedESName,
			Labels: map[string]string{
				"k8s-app": EmbeddedESName,
			},
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeNodePort,
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:       "http",
					Port:       9200,
					TargetPort: intstr.FromInt(9200),
					NodePort:   30032, //todo
				},
				v1.ServicePort{
					Name:       "tcp",
					Port:       9300,
					TargetPort: intstr.FromInt(9300),
					NodePort:   30033, //todo
				},
			},
			Selector: map[string]string{
				"k8s-app": EmbeddedESName,
			},
		},
	}
}

func NewKibanaService(namespace string) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      EmbeddedKibanaName,
			Labels: map[string]string{
				"k8s-app": EmbeddedKibanaName,
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:       "http",
					Port:       5601,
					TargetPort: intstr.FromInt(5601),
					NodePort:   30034, //todo
				},
			},
			Type: v1.ServiceTypeNodePort,
			Selector: map[string]string{
				"k8s-app": EmbeddedKibanaName,
			},
		},
	}
}

func NewESDeployment(namespace string) *v1beta2.Deployment {
	deployment := &v1beta2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      EmbeddedESName,
			Labels: map[string]string{
				"k8s-app": EmbeddedESName,
			},
		},
		Spec: v1beta2.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": EmbeddedESName,
				},
			},
			Replicas: int32Ptr(1),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Labels: map[string]string{
						"k8s-app": EmbeddedESName,
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: EmbeddedESName,
					InitContainers: []v1.Container{
						{
							Name:            "init-sysctl",
							Image:           "busybox",
							ImagePullPolicy: v1.PullIfNotPresent,
							Command:         []string{"sysctl", "-w", "vm.max_map_count=262144"},
							SecurityContext: &v1.SecurityContext{
								Privileged: boolPtr(true),
							},
						},
					},
					Containers: []v1.Container{
						{
							Name: EmbeddedESName,
							SecurityContext: &v1.SecurityContext{
								Capabilities: &v1.Capabilities{
									Add: []v1.Capability{"IPC_LOCK"},
								},
							},
							Image: esImage,
							Env: []v1.EnvVar{
								{
									Name:  "KUBERNETES_CA_CERTIFICATE_FILE",
									Value: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
								},
								{
									Name: "NAMESPACE",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "CLUSTER_NAME",
									Value: "myesdb",
								},
								{
									Name:  "DISCOVERY_SERVICE",
									Value: EmbeddedESName,
								},
								{
									Name:  "NODE_MASTER",
									Value: "true",
								},
								{
									Name:  "NODE_DATA",
									Value: "true",
								},
								{
									Name:  "HTTP_ENABLE",
									Value: "true",
								},
							},
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									Protocol:      v1.ProtocolTCP,
									ContainerPort: 9200,
								},
								{
									Name:          "tcp",
									Protocol:      v1.ProtocolTCP,
									ContainerPort: 9300,
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									//CPU is always requested as an absolute quantity, never as a relative quantity; 0.1 is the same amount of CPU on a single-core, dual-core, or 48-core machine
									v1.ResourceCPU: *resource.NewMilliQuantity(int64(2000), resource.DecimalSI),
									//Limits and requests for memory are measured in bytes.
									v1.ResourceMemory: *resource.NewQuantity(int64(4*1024*1024*1024), resource.DecimalSI), // unit is byte
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									MountPath: "/data",
									Name:      "storage",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "storage",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
					},
					RestartPolicy: v1.RestartPolicyAlways,
				},
			},
		},
	}

	return deployment
}

func NewKibanaDeployment(namespace string) *v1beta2.Deployment {
	deployment := &v1beta2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      EmbeddedKibanaName,
		},
		Spec: v1beta2.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": EmbeddedKibanaName,
				},
			},
			Replicas: int32Ptr(1),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Labels: map[string]string{
						"k8s-app": EmbeddedKibanaName,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  EmbeddedKibanaName,
							Image: kibanaImage,
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									Protocol:      v1.ProtocolTCP,
									ContainerPort: 5601,
								},
							},
							Env: []v1.EnvVar{
								{
									Name:  "ELASTICSEARCH_URL",
									Value: "http://" + EmbeddedESName + "." + namespace + ":9200",
								},
							},
						},
					},
					RestartPolicy: v1.RestartPolicyAlways,
				},
			},
		},
	}

	return deployment
}

func int32Ptr(i int32) *int32 { return &i }

func boolPtr(b bool) *bool { return &b }
