package controller

import (
	"context"
	"fmt"
	chaosv1 "github.com/maliciousbucket/pluck/api/v1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"strings"
)

var (
	k6Image              = "docker.io/grafana/k6"
	testNameAnnotation   = "chaos.galah-monitoring.io/test"
	testNumberAnnotation = "chaos.galah-monitoring.io/test-run"
	k6ApiVersion         = `k6.io/v1alpha1`
)

func createK6TestRunForJob(testRunJob *chaosv1.TestRunJob, count int32) *TestRunWrapper {
	envFrom := []corev1.EnvFromSource{}
	log.Printf("Creating k6 map: %s", testRunJob.Name)

	if testRunJob.Spec.EnvConfigMap != "" {
		envFrom = append(envFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: testRunJob.Spec.EnvConfigMap},
			},
		})
	}
	args := addK6Tag(testRunJob.Spec.Args, testRunJob.Spec.TestName, testRunJob.Status.TestRunCount)
	name := fmt.Sprintf("%s-%d", testRunJob.Name, count)
	annotations := annotationsForK6(testRunJob.Name, count)
	file := fmt.Sprintf("%s.js", testRunJob.Spec.TestName)
	image := k6Image
	if testRunJob.Spec.Image != "" {
		image = testRunJob.Spec.Image
	}

	spec := &TestRunWrapper{
		APIVersion: k6ApiVersion,
		Kind:       "TestRun",
		Metadata: WrapperMeta{
			Name:      name,
			Namespace: testRunJob.Namespace,
		},
		Spec: &SpecWrapper{
			Parallelism: 1,
			Script: ScriptWrapper{
				ConfigMap: struct {
					Name string `yaml:"name"`
					File string `yaml:"file"`
				}{Name: testRunJob.Spec.TestName, File: file},
			},
			Runner: RunnerWrapper{
				Image:           image,
				ImagePullPolicy: "IfNotPresent",
				Metadata: struct {
					Labels      map[string]string `yaml:"labels"`
					Annotations map[string]string `yaml:"annotations"`
				}{
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: annotations,
				},
				SecurityContext: struct {
					RunAsNonRoot bool `yaml:"runAsNonRoot"`
				}{
					RunAsNonRoot: false,
				},
			},
			Arguments: args,
			Cleanup:   "post",
		},
	}
	log.Printf("Creating K6 TestRun %s", spec.Metadata.Name)
	return spec
}

func annotationsForK6(name string, count int32) map[string]string {
	annotations := make(map[string]string)
	annotations[testNameAnnotation] = name
	annotations[testNumberAnnotation] = fmt.Sprintf("%d", count)
	return annotations
}

func (r *TestRunJobReconciler) getOrCreateTestRunMap(ctx context.Context, testRunJob *chaosv1.TestRunJob, count int32) (*corev1.ConfigMap, bool, error) {
	foundConfigMap := &corev1.ConfigMap{}
	name := fmt.Sprintf("%s-%d", testRunJob.Name, count)
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: testRunJob.Namespace}, foundConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.createTestRunConfigMap(testRunJob, count)
		}
		return nil, false, err
	}
	return foundConfigMap, false, nil
}

func (r *TestRunJobReconciler) createTestRunConfigMap(testRunJob *chaosv1.TestRunJob, count int32) (*corev1.ConfigMap, bool, error) {
	k6Run := createK6TestRunForJob(testRunJob, count)
	if k6Run == nil {
		return nil, false, fmt.Errorf("could not create K6 TestRun")
	}

	bytes, err := yaml.Marshal(k6Run)
	if err != nil {
		return nil, false, err
	}

	fmt.Println(k6Run)

	name := fmt.Sprintf("%s-%d", testRunJob.Name, count)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testRunJob.Namespace,
		},
		Immutable: nil,
		Data: map[string]string{
			`k6.yaml`: string(bytes),
		},
	}
	fmt.Println(string(bytes))
	return configMap, true, nil

}

func addK6Tag(args, name string, count int32) string {
	tag := fmt.Sprintf("--tag test-id=%s-%d", name, count)
	result := strings.Join([]string{tag, args}, " ")
	result = strings.TrimSpace(result)
	return result

}

type TestRunWrapper struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   WrapperMeta  `yaml:"metadata"`
	Spec       *SpecWrapper `yaml:"spec"`
}

type WrapperMeta struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type SpecWrapper struct {
	Parallelism int           `yaml:"parallelism,omitempty"`
	Script      ScriptWrapper `yaml:"script"`
	Runner      RunnerWrapper `yaml:"runner"`
	Arguments   string        `yaml:"arguments"`
	Cleanup     string        `yaml:"cleanup"`
}

type RunnerWrapper struct {
	Image           string `yaml:"image"`
	ImagePullPolicy string `yaml:"imagePullPolicy"`
	Metadata        struct {
		Labels      map[string]string `yaml:"labels"`
		Annotations map[string]string `yaml:"annotations"`
	}
	SecurityContext struct {
		RunAsNonRoot bool `yaml:"runAsNonRoot"`
	} `yaml:"securityContext"`
	Resources struct {
		Limits struct {
			CPU    string `yaml:"cpu,omitempty"`
			Memory string `yaml:"memory,omitempty"`
		} `yaml:"limits,omitempty"`
		Requests struct {
			CPU    string `yaml:"cpu,omitempty"`
			Memory string `yaml:"memory,omitempty"`
		} `yaml:"requests,omitempty"`
	} `yaml:"resources,omitempty"`
}

type ScriptWrapper struct {
	ConfigMap struct {
		Name string `yaml:"name"`
		File string `yaml:"file"`
	} `yaml:"configMap"`
}
