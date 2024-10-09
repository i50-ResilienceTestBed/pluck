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
	log.Printf("Creatign k6 map: %s", testRunJob.Name)

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
	//privileged := true
	//nonRoot := false

	//k6Run := &k6.TestRun{
	//	TypeMeta: metav1.TypeMeta{
	//		Kind:       "TestRun",
	//		APIVersion: "k6.io/v1alpha1",
	//	},
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name:      name,
	//		Namespace: testRunJob.Namespace,
	//	},
	//k6Spec := &k6.TestRunSpec{
	//	Script: k6.K6Script{
	//		ConfigMap: k6.K6Configmap{
	//			Name: testRunJob.Spec.TestName,
	//			File: file,
	//		},
	//	},
	//	Parallelism: 0,
	//	Separate:    false,
	//	Arguments:   args,
	//	Ports:       nil,
	//	Initializer: nil,
	//	Runner: k6.Pod{
	//		Image:           k6Image,
	//		ImagePullPolicy: "IfNotPresent",
	//		Metadata: k6.PodMetadata{
	//			Annotations: annotations,
	//			Labels:      make(map[string]string),
	//		},
	//		Resources:          corev1.ResourceRequirements{},
	//		ServiceAccountName: testRunJob.Spec.ServiceAccount,
	//		SecurityContext:    corev1.PodSecurityContext{},
	//		ContainerSecurityContext: corev1.SecurityContext{
	//			Privileged:               &privileged,
	//			RunAsNonRoot:             &nonRoot,
	//			ReadOnlyRootFilesystem:   nil,
	//			AllowPrivilegeEscalation: nil,
	//			ProcMount:                nil,
	//			SeccompProfile:           nil,
	//			AppArmorProfile:          nil,
	//		},
	//		EnvFrom: envFrom,
	//	},
	//	Cleanup: "post",
	//}

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
				Image:           k6Image,
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

	//wrapper := &TestRunWrapper{
	//	APIVersion: k6ApiVersion,
	//	Kind:       "TestRun",
	//	Metadata: metav1.ObjectMeta{
	//		Name:      name,
	//		Namespace: testRunJob.Namespace,
	//	},
	//	Spec: spec,
	//}

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
		//BinaryData: map[string][]byte{
		//	`k6.yaml`: bytes,
		//},
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

//type Wrapper struct {
//	APIVersion string            `yaml:"apiVersion"`
//	Kind       string            `yaml:"kind"`
//	Metadata   metav1.ObjectMeta `yaml:"metadata"`
//	Spec       struct {
//		Parallelism int `yaml:"parallelism"`
//		Script      struct {
//			ConfigMap struct {
//				Name string `yaml:"name"`
//				File string `yaml:"file"`
//			} `yaml:"configMap"`
//		} `yaml:"script"`
//		Runner struct {
//			Image    string `yaml:"image"`
//			Metadata struct {
//				Labels      map[string]string `yaml:"labels"`
//				Annotations map[string]string `yaml:"annotations"`
//			} `yaml:"metadata"`
//			SecurityContext struct {
//				RunAsNonRoot bool `yaml:"runAsNonRoot"`
//			} `yaml:"securityContext"`
//			Resources struct {
//				Limits struct {
//					CPU    string `yaml:"cpu,omitempty"`
//					Memory string `yaml:"memory,omitempty"`
//				} `yaml:"limits"`
//				Requests struct {
//					CPU    string `yaml:"cpu,omitempty"`
//					Memory string `yaml:"memory,omitempty"`
//				} `yaml:"requests,omitempty"`
//			} `yaml:"resources"`
//		} `yaml:"runner"`
//	} `yaml:"spec"`
//}

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
