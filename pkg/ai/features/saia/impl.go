package saia

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ConfigMap(namespace, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-saia-config", name),
			Namespace: namespace,
		},
		Data: map[string]string{
			"SAIA_FEATURE_ENABLED": "true",
		},
	}
}

func Secret(namespace, name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-saia-secret", name),
			Namespace: namespace,
		},
		StringData: map[string]string{
			"API_TOKEN": "replace-me",
		},
	}
}

func Deployment(namespace, name string) *appsv1.Deployment {
	labels := map[string]string{"app": "saia"}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-saia", name),
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "saia",
							Image: "docker.io/splunk/saia:latest",
							Env: []corev1.EnvVar{
								{Name: "SAIA_CONFIG", ValueFrom: &corev1.EnvVarSource{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										Key: "SAIA_FEATURE_ENABLED",
										LocalObjectReference: corev1.LocalObjectReference{
											Name: fmt.Sprintf("%s-saia-config", name),
										},
									},
								}},
							},
						},
					},
				},
			},
		},
	}
}

func Service(namespace, name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-saia-svc", name),
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "saia"},
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 8080,
				},
			},
		},
	}
}

func pointer(i int32) *int32 {
	return &i
}
