package resources

import (
  "time"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
)

// newDeployment
func NewDeployment(labels map[string]string, name, namespace, image, config_name string) *appsv1.Deployment {
  updateLabels := map[string]string{
		"app": labels["app"],
		"controller": labels["controller"],
		"last_restart": time.Now().Format(layout),
	}
  var replicas int32 = 1

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
      Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: updateLabels,
				},
        Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/fluent-bit/etc/fluent-bit.conf",
                  SubPath:   "fluent-bit.conf",
								},
                {
									Name:      "lua",
									MountPath: "/fluent-bit/etc/fluent-bit-metrics.lua",
                  SubPath:   "fluent-bit-metrics.lua",
								},
							},
						},
					},
          ServiceAccountName: "logfilter-controller",
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: config_name,
									},
								},
							},
						},
						{
							Name: "lua",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "fluent-bit-metrics-lua",
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
