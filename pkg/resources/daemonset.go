package resources

import (
  "time"
  "strings"

  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  appsv1 "k8s.io/api/apps/v1"
  corev1 "k8s.io/api/core/v1"
)

const layout = "2006-01-02-15-04-05"
// newDaemonset
func NewDaemonSet(labels map[string]string, name, namespace, image, tolerations, node_selector, config_name string) *appsv1.DaemonSet {
	updateLabels := map[string]string{
		"app": labels["app"],
		"controller": labels["controller"],
		"last_restart": time.Now().Format(layout),
	}

  tols := []corev1.Toleration{}
  if tolerations != "" {
    tlist := strings.Split(tolerations,",")
    for _, t := range tlist {
      if t == "etcd" {
        tols = append(tols, corev1.Toleration{
          Effect: "NoExecute",
          Key: "node-role.kubernetes.io/etcd",
          Value: "true",
        })
      }
      if t == "controlplane" {
        tols = append(tols, corev1.Toleration{
          Effect: "NoSchedule",
          Key: "node-role.kubernetes.io/controlplane",
          Value: "true",
        })
      }
    }
  }

  nselector := map[string]string{}
  if node_selector != "" {
    nselector["node-role.kubernetes.io/" + node_selector] = "true"
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
              Env: []corev1.EnvVar{
                {
                  Name: "HOSTNAME",
                  ValueFrom: &corev1.EnvVarSource{
                    FieldRef: &corev1.ObjectFieldSelector{
                      FieldPath: "spec.nodeName",
                    },
                  },
                },
              },
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
                  ReadOnly:  true,
								},
                {
									Name:      "proc",
									MountPath: "/host/proc",
                  ReadOnly:  true,
								},
                {
									Name:      "cgroup",
									MountPath: "/host/sys/fs/cgroup",
                  ReadOnly:  true,
								},
                {
									Name:      "tmp",
									MountPath: "/tmp",
								},
								{
									Name:      "config",
									MountPath: "/fluent-bit/etc/fluent-bit.conf",
                  SubPath:   "fluent-bit.conf",
								},
                {
									Name:      "lua",
									MountPath: "/fluent-bit/etc/funcs.lua",
                  SubPath:   "funcs.lua",
								},
                {
									Name:      "chk-proc",
									MountPath: "/fluent-bit/etc/chk_proc.sh",
                  SubPath:   "chk_proc.sh",
								},
							},
						},
					},
          Tolerations: tols,
          NodeSelector: nselector,
          ServiceAccountName: "logfilter-controller",
					Volumes: []corev1.Volume{
            {
							Name: "varlog",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
                  Path: "/var/log",
								},
							},
						},
            {
							Name: "rkelog",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
                  Path: "/var/lib/rancher/rke/log",
								},
							},
						},
            {
							Name: "varlibdockercontainers",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
                  Path: "/var/lib/docker/containers",
								},
							},
						},
            {
							Name: "proc",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
                  Path: "/proc",
								},
							},
						},
            {
							Name: "cgroup",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
                  Path: "/sys/fs/cgroup",
								},
							},
						},
            {
							Name: "tmp",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
                  Path: "/tmp",
								},
							},
						},
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
										Name: "fluentbit-lua",
									},
								},
							},
						},
            {
              Name: "chk-proc",
              VolumeSource: corev1.VolumeSource{
                ConfigMap: &corev1.ConfigMapVolumeSource{
                  LocalObjectReference: corev1.LocalObjectReference{
                    Name: "chk-proc",
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
