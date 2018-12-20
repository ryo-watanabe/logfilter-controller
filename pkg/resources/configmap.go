package resources

import (
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  corev1 "k8s.io/api/core/v1"
)

func NewConfigMap(name, namespace string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}
