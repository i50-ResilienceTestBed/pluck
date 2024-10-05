package controller

import (
	chaosv1 "github.com/maliciousbucket/pluck/api/v1"
	kbatch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newJob(testRunJob *chaosv1.TestRunJob) (*kbatch.Job, error) {
	k6Map, err := createTestRunConfigMap(testRunJob, *testRunJob.Spec.TestRunCount)
	if err != nil {
		return nil, err
	}

	job := &kbatch.Job{
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       kbatch.JobSpec{},
	}

	return nil, nil
}

func getCommand(script string) string {
	return ""
}
