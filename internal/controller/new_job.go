package controller

import (
	chaosv1 "github.com/maliciousbucket/pluck/api/v1"
	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var (
	kubectlContainerName = "kubectl"
	applyK6TestRunArg    = "kubectl delete -f /tmp/k6.yaml; kubectl apply -f /tmp/k6.yaml"
)

func constructJobForTestRunJob(testRunJob *chaosv1.TestRunJob, scheduledTime time.Time, scriptVer, envVer, k6Map string) (*kbatch.Job, error) {

	job := &kbatch.Job{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: kbatch.JobSpec{
			Parallelism: nil,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  kubectlContainerName,
							Image: testRunJob.Spec.Image,
							Command: []string{
								"/bin/bash",
							},
							Args: []string{
								"-c",
								applyK6TestRunArg,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "k6-yaml",
									MountPath: "/tmp/",
								},
							},
						},
					},
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					ServiceAccountName: testRunJob.Spec.ServiceAccount,
					Volumes: []corev1.Volume{
						{
							Name: "k6-yaml",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: k6Map,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for k, v := range testRunJob.Spec.JobTemplate.Annotations {
		job.Annotations[k] = v
	}

	job.Annotations[scheduledTimeAnnotation] = scheduledTime.Format(time.RFC3339)
	job.Annotations[scriptVersionAnnotation] = scriptVer
	if envVer != "" {
		job.Annotations[envVersionAnnotation] = envVer
	}

	for k, v := range testRunJob.Spec.JobTemplate.Labels {
		job.Labels[k] = v
	}

	return job, nil
}

func getCommand(script string) string {
	return ""
}
