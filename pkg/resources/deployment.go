package resources

import (
        "time"
        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        appsv1 "k8s.io/api/apps/v1"
        corev1 "k8s.io/api/core/v1"
)

// newDeployment
func NewDeployment(labels map[string]string, name, namespace, image, kafkasecret, kafkasecretpath,
        registrykey, config_name string) *appsv1.Deployment {
        updateLabels := map[string]string{
		"app": labels["app"],
		"controller": labels["controller"],
		"last_restart": time.Now().Format(layout),
	}
        var replicas int32 = 1

        imagepullsecrets := []corev1.LocalObjectReference{}
        if registrykey != "" {
                imagepullsecrets = append(imagepullsecrets, corev1.LocalObjectReference{Name: registrykey})
        }

	deploy := appsv1.Deployment{
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
									Name:      "filter",
									MountPath: "/fluent-bit/filter",
								},
							},
						},
					},
                                        ServiceAccountName: "logfilter-controller",
                                        ImagePullSecrets: imagepullsecrets,
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
							Name: "filter",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "lua-filter-scripts",
									},
								},
							},
						},
					},
				},
			},
		},
	}

        if kafkasecret != "" {
                deploy.Spec.Template.Spec.Containers[0].VolumeMounts = append(
                        deploy.Spec.Template.Spec.Containers[0].VolumeMounts,
                        corev1.VolumeMount{
                                Name:      kafkasecret,
                        	MountPath: kafkasecretpath,
                        },
                )
                deploy.Spec.Template.Spec.Volumes = append(
                        deploy.Spec.Template.Spec.Volumes,
                        corev1.Volume{
                                Name: kafkasecret,
                                VolumeSource: corev1.VolumeSource{
                                        Secret: &corev1.SecretVolumeSource{
                                                SecretName: kafkasecret,
                                        },
                                },
                        },
                )
        }

        return &deploy
}
