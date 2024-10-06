/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestRunJobSpec defines the desired state of TestRunJob
type TestRunJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//+kubebuilder:validation:MinLength=0
	TestName string `json:"testName"`
	//+kubebuilder:validation:MinLength=0
	ServiceAccountName string `json:"serviceAccount"`
	//+kubebuilder:validation:MinLength=0
	ScriptConfigMap string `json:"scriptConfigMap,omitempty"`
	//+kubebuilder:validation:MinLength=0
	EnvConfigMap string `json:"envConfigMap,omitempty"`

	//+optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	//+kubebuilder:validation:MinLength=0
	Args string `json:"args"`

	Schedule CronSchedule `json:"schedule"`

	// +kubebuilder:validation:Minimum=0
	// +optional
	TestRunCount *int32 `json:"testRunCount,omitempty"`

	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// +optional
	RunOnce *bool `json:"runOnce,omitempty"`

	//// Specifies the job that will be created when executing a CronJob.
	//JobTemplate batchv1.JobTemplateSpec `json:"jobTemplate"`
	JobTemplate JobTemplate `json:"jobTemplate"`

	// +kubebuilder:validation:Minimum=0
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +optional
	TestRunHistoryLimit *int32 `json:"testRunJobHistoryLimit,omitempty"`
}

// TestRunJobStatus defines the observed state of TestRunJob
type TestRunJobStatus struct {

	// A list of pointers to currently running jobs.
	// +optional
	Active []corev1.ObjectReference `json:"active,omitempty"`

	//TODO: Replace with K6 type

	// +optional
	CurrentTestRun corev1.ObjectReference `json:"currentTestRun,omitempty"`

	// Information when was the last time the job was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// +optional
	FinishedAt metav1.Time `json:"finishedAt,omitempty"`
}

type CronSchedule struct {
	// +optional
	Minute *ConField `json:"minute,omitempty"`
	// +optional
	Hour *ConField `json:"hour,omitempty"`
	// +optional
	DayOfMonth *ConField `json:"dayOfMonth,omitempty"`
	// +optional
	Month *ConField `json:"month,omitempty"`
	// +optional
	DayOfWeek *ConField `json:"dayOfWeek,omitempty"`
}

type ConField string

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TestRunJob is the Schema for the testrunjobs API
type TestRunJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestRunJobSpec   `json:"spec,omitempty"`
	Status TestRunJobStatus `json:"status,omitempty"`
}

func (s *CronSchedule) GetScheduleString() (string, error) {

	scheduleParts := []string{"*", "*", "*", "*", "*"}
	if s.Minute != nil {
		scheduleParts[0] = string(*s.Minute)
	}
	if s.Hour != nil {
		scheduleParts[1] = string(*s.Hour)
	}
	if s.DayOfMonth != nil {
		scheduleParts[2] = string(*s.DayOfMonth)
	}
	if s.Month != nil {
		scheduleParts[3] = string(*s.Month)
	}
	if s.DayOfWeek != nil {
		scheduleParts[4] = string(*s.DayOfWeek)
	}
	return strings.Join(scheduleParts, " "), nil

}

type JobTemplate struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// +kubebuilder:object:root=true

// TestRunJobList contains a list of TestRunJob
type TestRunJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestRunJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestRunJob{}, &TestRunJobList{})
}

//type TestRun struct {
//	Id         string      `json:"id"`
//	TestRunJob string      `json:"testRunJob"`
//	K6TestRun  string      `json:"k6TestRun,omitempty"`
//	StartedAt  metav1.Time `json:"startedAt,omitempty"`
//	FinishedAt metav1.Time `json:"finishedAt,omitempty"`
//	Succeeded  bool        `json:"succeeded,omitempty"`
//}
