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

package controller

import (
	"context"
	"fmt"
	"github.com/robfig/cron"
	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ref "k8s.io/client-go/tools/reference"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	chaosv1 "github.com/maliciousbucket/pluck/api/v1"
)

// TestRunJobReconciler reconciles a TestRunJob object
type TestRunJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Clock
}

var (
	scheduledTimeAnnotation = "chaos.galah-monitoring.io/scheduled-at"
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (_ realClock) Now() time.Time { return time.Now() }

// +kubebuilder:rbac:groups=chaos.galah-monitoring.io,resources=testrunjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.galah-monitoring.io,resources=testrunjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.galah-monitoring.io,resources=testrunjobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TestRunJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *TestRunJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var testRunJob chaosv1.TestRunJob
	if err := r.Get(ctx, req.NamespacedName, &testRunJob); err != nil {
		logger.Error(err, "unable to fetch test run job")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var childJobs kbatch.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		logger.Error(err, "unable to list child jobs")
		return ctrl.Result{}, err
	}

	var activeJobs []*kbatch.Job
	var successfulJobs []*kbatch.Job
	var failedJobs []*kbatch.Job
	var mostRecentTime *time.Time

	for i, job := range childJobs.Items {
		_, finishedType := isJobFinished(&job)
		switch finishedType {
		case "":
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case kbatch.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case kbatch.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}
		scheduledTime, err := getScheduledTimeForJob(&job)
		if err != nil {
			logger.Error(err, "unable to parse scheduled time for job", &job)
			continue
		}
		if scheduledTime != nil {
			if mostRecentTime == nil || mostRecentTime.Before(*scheduledTime) {
				mostRecentTime = scheduledTime
			}
		}
	}

	if mostRecentTime != nil {
		testRunJob.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		testRunJob.Status.LastScheduleTime = nil
	}

	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.Scheme, activeJob)
		if err != nil {
			logger.Error(err, "unable to get reference to active job", "job", activeJob)
			continue
		}
		testRunJob.Status.Active = append(testRunJob.Status.Active, *jobRef)
	}

	logger.V(1).Info("job count", "active jobs", len(activeJobs), "successful jobs", len(successfulJobs), "failed jobs", len(failedJobs))

	if err := r.Status().Update(ctx, &testRunJob); err != nil {
		logger.Error(err, "unable to update test run job status")
		return ctrl.Result{}, err
	}

	if testRunJob.Spec.FailedJobsHistoryLimit != nil {
		sort.Slice(failedJobs, func(i, j int) bool {
			if failedJobs[i].Status.StartTime == nil {
				return failedJobs[j].Status.StartTime != nil
			}
			return failedJobs[i].Status.StartTime.Before(failedJobs[j].Status.StartTime)
		})
		for i, job := range failedJobs {
			if int32(i) >= int32(len(failedJobs))-*testRunJob.Spec.FailedJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
				logger.Error(err, "unable to delete old failed job", "job", job)
			} else {
				logger.V(0).Info("deleted old failed job", "job", job)
			}
		}
	}

	if testRunJob.Spec.SuccessfulJobsHistoryLimit != nil {
		sort.Slice(successfulJobs, func(i, j int) bool {
			if successfulJobs[i].Status.StartTime == nil {
				return successfulJobs[j].Status.StartTime != nil
			}
			return successfulJobs[i].Status.StartTime.Before(successfulJobs[j].Status.StartTime)
		})
		for i, job := range successfulJobs {
			if int32(i) >= int32(len(successfulJobs))-*testRunJob.Spec.SuccessfulJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
				logger.Error(err, "unable to delete old successful job", "job", job)
			} else {
				logger.V(0).Info("deleted old successful job", "job", job)
			}
		}
	}

	if testRunJob.Spec.Suspend != nil && *testRunJob.Spec.Suspend {
		logger.V(1).Info("test run job suspended, skipping")
		return ctrl.Result{}, nil
	}

	missedRun, nextRun, err := getNextSchedule(&testRunJob, r.Now())
	if err != nil {
		logger.Error(err, "unable to get next cron schedule")
		return ctrl.Result{}, nil
	}

	scheduleResult := ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())}
	logger = logger.WithValues("now", r.Now(), "next run", nextRun)

	if missedRun.IsZero() {
		logger.V(1).Info("no upcoming scheduled times, sleeping until next run")
		return scheduleResult, nil
	}

	logger = logger.WithValues("missed run", missedRun)
	tooLate := false
	if testRunJob.Spec.StartingDeadlineSeconds != nil {
		tooLate = missedRun.Add(time.Duration(*testRunJob.Spec.StartingDeadlineSeconds) * time.Second).Before(r.Now())
	}
	if tooLate {
		logger.V(1).Info("missed deadline for last run, skipping")
		return scheduleResult, nil
	}

	return ctrl.Result{}, nil
}

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = chaosv1.GroupVersion.String()
	jobImage    = "bitnami/kubectl"
)

func isJobFinished(job *kbatch.Job) (bool, kbatch.JobConditionType) {
	for _, c := range job.Status.Conditions {
		if (c.Type == kbatch.JobComplete || c.Type == kbatch.JobFailed) && c.Status == corev1.ConditionTrue {
			return true, c.Type
		}

	}
	return false, ""
}

func getScheduledTimeForJob(job *kbatch.Job) (*time.Time, error) {
	timeRaw := job.Annotations[scheduledTimeAnnotation]
	if len(timeRaw) == 0 {
		return nil, nil
	}
	timeParsed, err := time.Parse(time.RFC3339, timeRaw)
	if err != nil {
		return nil, err
	}
	return &timeParsed, nil
}

func getNextSchedule(job *chaosv1.TestRunJob, now time.Time) (lastMissed time.Time, next time.Time, err error) {
	jobSched, err := job.GetScheduleString()
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	sched, err := cron.ParseStandard(jobSched)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	var earliestTime time.Time
	if job.Status.LastScheduleTime != nil {
		earliestTime = job.Status.LastScheduleTime.Time
	} else {
		earliestTime = job.ObjectMeta.CreationTimestamp.Time
	}

	if job.Spec.StartingDeadlineSeconds != nil {
		schedulingDeadline := now.Add(-time.Second * time.Duration(*job.Spec.StartingDeadlineSeconds))

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}
	if earliestTime.After(now) {
		return time.Time{}, sched.Next(now), nil
	}

	starts := 0
	for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
		lastMissed = t
		starts++
		if starts > 100 {
			return time.Time{}, time.Time{}, fmt.Errorf("too many missed start times (> 100)")
		}
	}
	return lastMissed, sched.Next(now), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TestRunJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1.TestRunJob{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ConfigMap{}).
		//Watches(
		//	&corev1.ConfigMap{},
		//	handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
		//	builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		//).
		For(&chaosv1.TestRunJob{}).
		Owns(&corev1.ConfigMap{}).
		//Watches()
		Complete(r)

}
