package controller

import (
	"fmt"
	k6 "github.com/grafana/k6-operator/api/v1alpha1"
	chaosv1 "github.com/maliciousbucket/pluck/api/v1"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
)

var (
	k6Image              = ""
	testNameAnnotation   = "chaos.galah-monitoring.io/test"
	testNumberAnnotation = "chaos.galah-monitoring.io/test-run"
)

func createK6TestRunForJob(testRunJob *chaosv1.TestRunJob, count int32) *k6.TestRun {
	envFrom := []corev1.EnvFromSource{}

	if testRunJob.Spec.EnvConfigMap != "" {
		envFrom = append(envFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: testRunJob.Spec.EnvConfigMap},
			},
		})
	}

	name := fmt.Sprintf("%s-%d", testRunJob.Name, count)
	annotations := annotationsForK6(testRunJob.Name, count)
	k6Run := &k6.TestRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testRunJob.Namespace,
		},
		Spec: k6.TestRunSpec{
			Script: k6.K6Script{
				ConfigMap: k6.K6Configmap{
					Name: name,
					File: "k6.yaml",
				},
			},
			Parallelism: 0,
			Separate:    false,
			Arguments:   testRunJob.Spec.Args,
			Ports:       nil,
			Initializer: nil,
			Runner: k6.Pod{
				Image:           k6Image,
				ImagePullPolicy: "IFNOTPRESENT",
				Metadata: k6.PodMetadata{
					Annotations: annotations,
					Labels:      nil,
				},
				Resources:                corev1.ResourceRequirements{},
				ServiceAccountName:       "",
				SecurityContext:          corev1.PodSecurityContext{},
				ContainerSecurityContext: corev1.SecurityContext{},
				EnvFrom:                  envFrom,
			},
			Cleanup: "post",
		},
	}
	log.Printf("Creating K6 TestRun %s", k6Run.Name)
	return k6Run
}

func annotationsForK6(name string, count int32) map[string]string {
	annotations := make(map[string]string)
	annotations[testNameAnnotation] = name
	annotations[testNumberAnnotation] = fmt.Sprintf("%d", count)
	return annotations
}

func (r *TestRunJobReconciler) createTestRunConfigMap(testRunJob *chaosv1.TestRunJob, count int32) (*corev1.ConfigMap, error) {
	k6Run := createK6TestRunForJob(testRunJob, count)
	if k6Run == nil {
		return nil, fmt.Errorf("could not create K6 TestRun")
	}
	bytes, err := yaml.Marshal(k6Run)
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s-%d", testRunJob.Name, count)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testRunJob.Namespace,
		},
		Immutable: nil,
		Data: map[string]string{
			"k6.yaml": string(bytes),
		},
	}
	return configMap, nil

}
