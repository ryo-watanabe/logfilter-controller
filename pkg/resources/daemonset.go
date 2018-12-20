package resources

import (
  "time"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
)

const layout = "2006-01-02-15-04-05"
// newDaemonset
func NewDaemonSet(labels map[string]string, name, namespace, image string) *appsv1.DaemonSet {
	updateLabels := map[string]string{
		"app": labels["app"],
		"controller": labels["controller"],
		"last_restart": time.Now().Format(layout),
	}
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
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
									Name:      "varlog",
									MountPath: "/var/log",
								},
                {
									Name:      "rkelog",
									MountPath: "/var/lib/rancher/rke/log",
								},
                {
									Name:      "varlibdockercontainers",
									MountPath: "/var/lib/docker/containers",
                  ReadOnly:  true
								},
								{
									Name:      "config",
									MountPath: "/fluent-bit/etc/fluent-bit.conf",
                  SubPath:   "fluent-bit.conf"
								},
                {
									Name:      "lua",
									MountPath: "/fluent-bit/etc/funcs.lua",
                  SubPath:   "funcs.conf"
								},
							},
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{ "kill", "-SIGUSR1", "1" },
									},
								},
							},
						},
					},
          Tolerations: []corev1.Toleration{
            {
              Effect: "NoExecute",
              Key: "node-role.kubernetes.io/etcd",
              Value: "true",
            },
            {
              Effect: "NoSchedule",
              Key: "node-role.kubernetes.io/controlplane",
              Value: "true",
            },
          }
					Volumes: []corev1.Volume{
            {
							Name: "varlog",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
                  Path: "/var/log"
								},
							},
						},
            {
							Name: "rkelog",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
                  Path: "/var/lib/rancher/rke/log"
								},
							},
						},
            {
							Name: "varlibdockercontainers",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
                  Path: "/var/lib/docker/containers"
								},
							},
						},
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "fluentbit-config",
									},
								},
							},
						},
						{
							Name: "lua",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "fluentbit-lua",
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
