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
	"context"
	"fmt"
	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var testrunjoblog = logf.Log.WithName("testrunjob-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&TestRunJob{}).
		WithValidator(&TestRunJobCustomValidator{}).
		WithDefaulter(&TestRunJobCustomDefaulter{
			DefaultSuspend:                    false,
			DefaultRunOnce:                    true,
			DefaultTestRunCount:               0,
			DefaultSuccessfulJobsHistoryLimit: 3,
			DefaultFailedJobsHistoryLimit:     1,
			DefaultTestRunHistoryLimit:        3,
		}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-chaos-galah-monitoring-io-v1-testrunjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.galah-monitoring.io,resources=testrunjobs,verbs=create;update,versions=v1,name=mtestrunjob.kb.io,admissionReviewVersions=v1

type TestRunJobCustomDefaulter struct {
	DefaultSuspend                    bool
	DefaultRunOnce                    bool
	DefaultTestRunCount               int32
	DefaultSuccessfulJobsHistoryLimit int32
	DefaultFailedJobsHistoryLimit     int32
	DefaultTestRunHistoryLimit        int32
}

var _ webhook.CustomDefaulter

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (d *TestRunJobCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	testRunJob, ok := obj.(*TestRunJob)
	if !ok {
		return fmt.Errorf("expected a TestRunJob but got a %T", obj)
	}

	testrunjoblog.Info("Defaulting for TestRunJob", "name", testRunJob.Name)
	return nil

}

func (d *TestRunJobCustomDefaulter) applyDefaults(ctx context.Context, testRunJob *TestRunJob) {
	if testRunJob.Spec.Suspend == nil {
		testRunJob.Spec.Suspend = new(bool)
		*testRunJob.Spec.Suspend = d.DefaultSuspend
	}

	if testRunJob.Spec.RunOnce == nil {
		testRunJob.Spec.RunOnce = new(bool)
		*testRunJob.Spec.RunOnce = d.DefaultRunOnce
	}

	if testRunJob.Spec.SuccessfulJobsHistoryLimit == nil {
		testRunJob.Spec.SuccessfulJobsHistoryLimit = new(int32)
		*testRunJob.Spec.SuccessfulJobsHistoryLimit = d.DefaultSuccessfulJobsHistoryLimit
	}

	if testRunJob.Spec.FailedJobsHistoryLimit == nil {
		testRunJob.Spec.FailedJobsHistoryLimit = new(int32)
		*testRunJob.Spec.FailedJobsHistoryLimit = d.DefaultFailedJobsHistoryLimit
	}

	if testRunJob.Spec.TestRunHistoryLimit == nil {
		testRunJob.Spec.TestRunHistoryLimit = new(int32)
		*testRunJob.Spec.TestRunHistoryLimit = d.DefaultTestRunHistoryLimit
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-chaos-galah-monitoring-io-v1-testrunjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.galah-monitoring.io,resources=testrunjobs,verbs=create;update,versions=v1,name=vtestrunjob.kb.io,admissionReviewVersions=v1

type TestRunJobCustomValidator struct{}

var _ webhook.CustomValidator = &TestRunJobCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *TestRunJobCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	testRunJob, ok := obj.(*TestRunJob)
	if !ok {
		return nil, fmt.Errorf("expected TestRunJob but got a %T", obj)
	}
	testrunjoblog.Info("validate create", "name", testRunJob.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, validateTestRunJob(testRunJob)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *TestRunJobCustomValidator) ValidateUpdate(ctx context.Context, newObj, oldObj runtime.Object) (admission.Warnings, error) {
	testRunJob, ok := newObj.(*TestRunJob)
	if !ok {
		return nil, fmt.Errorf("expected a TestRunJob but got a %T", newObj)
	}
	testrunjoblog.Info("validate update", "name", testRunJob.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *TestRunJobCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	testRunJob, ok := obj.(*TestRunJob)
	if !ok {
		return nil, fmt.Errorf("expected a TestRunJob but got a %T", obj)
	}
	testrunjoblog.Info("validate delete", "name", testRunJob.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

func validateTestRunJob(job *TestRunJob) error {
	var allErrs field.ErrorList
	if err := validateTestRunJobName(job); err != nil {
		allErrs = append(allErrs, err)
	}

	if err := validateScheduleFormat(job.Spec.Schedule); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "chaos.galah-monitoring.io", Kind: "TestRunJob"},
		job.Name, allErrs)
}

func validateScheduleFormat(schedule CronSchedule) *field.Error {
	sched, err := schedule.GetScheduleString()
	if err != nil {
		return field.Invalid(field.NewPath("spec"), schedule, err.Error())
	}
	if _, err = cron.ParseStandard(sched); err != nil {
		return field.Invalid(field.NewPath("spec"), schedule, err.Error())
	}
	return nil
}

func validateTestRunJobName(testRunJob *TestRunJob) *field.Error {
	if len(testRunJob.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		return field.Invalid(field.NewPath("metadata").Child("name"), testRunJob.ObjectMeta.Name, "name must be no more than 52 characters")
	}
	return nil
}
