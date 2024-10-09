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
	corev1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"log"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	chaosv1 "github.com/maliciousbucket/pluck/api/v1"

	batchv1 "k8s.io/api/batch/v1"
)

var _ = Describe("TestRunJob Controller", func() {
	const (
		TestRunJobName      = "test-testrunjob"
		TestRunJobNamespace = "default"
		JobName             = "test-job"
		ScriptName          = "test-script"
		ConfigMapName       = "test-configmap"

		timeout  = time.Second * 15
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		testrunjob := &chaosv1.TestRunJob{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind TestRunJob")
			err := k8sClient.Get(ctx, typeNamespacedName, testrunjob)
			if err != nil && errors.IsNotFound(err) {
				resource := &chaosv1.TestRunJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &chaosv1.TestRunJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance TestRunJob")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &TestRunJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})

	Context("When updating TestRunJob status", func() {
		//It("Should first have everything setup", func() {
		//
		//})

		It("Should increase TestRunJob Status.Active count when new Jobs are created", func() {

			By("Creating the script Config Map")
			ctx := context.Background()
			file := fmt.Sprintf("%s.js", ScriptName)
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ConfigMapName,
					Namespace: TestRunJobNamespace,
				},
				Data: map[string]string{
					file: "b",
				},
			}
			Expect(k8sClient.Create(ctx, configMap)).To(Succeed())
			By("Creating the Job Role")
			roleName := fmt.Sprintf("k6-%s", TestRunJobNamespace)
			role := rbacV1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName,
					Namespace: TestRunJobNamespace,
				},
				Rules: []rbacV1.PolicyRule{
					{
						APIGroups: []string{"k6.io"},
						Resources: []string{"testruns"},
						Verbs: []string{
							"create",
							"delete",
							"get",
							"list",
							"patch",
							"update",
							"watch",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &role)).To(Succeed())
			By("Creating the Job Service Account")
			accountName := fmt.Sprintf("k6-%s", TestRunJobNamespace)
			account := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      accountName,
					Namespace: TestRunJobNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, account)).To(Succeed())

			By("Creating the Job Role Binding")
			roleBinding := rbacV1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      role.Name,
					Namespace: TestRunJobNamespace,
				},
				RoleRef: rbacV1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     role.Name,
				},
				Subjects: []rbacV1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      account.Name,
						Namespace: TestRunJobNamespace,
					},
				},
			}
			Expect(k8sClient.Create(ctx, &roleBinding)).To(Succeed())

			By("By crating a new TestRunJob")
			//ctx := context.Background()
			//accountName := fmt.Sprintf("k6-%s", TestRunJobNamespace)
			minute := chaosv1.CronField("1")
			testrunjob := &chaosv1.TestRunJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "chaos.galah-monitoring.io/v1",
					Kind:       "TestRunJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      TestRunJobName,
					Namespace: TestRunJobNamespace,
				},
				Spec: chaosv1.TestRunJobSpec{
					TestName:        ScriptName,
					ServiceAccount:  accountName,
					ScriptConfigMap: ScriptName,
					EnvConfigMap:    "",
					Env:             nil,
					Args:            "--tag testid=pluck-test",
					Schedule: chaosv1.CronSchedule{
						Minute: &minute,
					},
					Image: "docker.io/grafana/k6",
					//TestRunCount:            nil,
					StartingDeadlineSeconds: nil,
					Suspend:                 nil,
					RunOnce:                 nil,
					JobTemplate: chaosv1.JobTemplate{
						Labels: map[string]string{
							"foo": "bar",
						},
					},
					SuccessfulJobsHistoryLimit: nil,
					FailedJobsHistoryLimit:     nil,
					TestRunHistoryLimit:        nil,
				},
			}
			log.Printf("Job Base: %+v", testrunjob)
			Expect(k8sClient.Create(ctx, testrunjob)).To(Succeed())

			lookupKey := types.NamespacedName{Name: TestRunJobName, Namespace: TestRunJobNamespace}
			createdTestrunJob := &chaosv1.TestRunJob{}
			afg := k8sClient.Get(ctx, lookupKey, createdTestrunJob)
			log.Printf("TestrunJob GET: %+v", afg)

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, lookupKey, createdTestrunJob)).To(Succeed())
				g.Expect(createdTestrunJob.Spec.TestName).To(Equal(ScriptName))
			}, timeout, interval).Should(Succeed())

			log.Printf("Job: %+v", createdTestrunJob)

			Expect(createdTestrunJob.Spec.Schedule.GetScheduleString()).To(Equal("1 * * * *"))

			By("Checking the TestRunJob has zero active Jobs")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, lookupKey, createdTestrunJob)).To(Succeed())
				g.Expect(createdTestrunJob.Status.Active).To(HaveLen(0))
			}, duration, interval).Should(Succeed())

			log.Printf("Job 2: %+v", createdTestrunJob)

			By("Creating a new job")
			testJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      JobName,
					Namespace: TestRunJobNamespace,
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test-container",
									Image: "test-image",
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			}

			kind := reflect.TypeOf(chaosv1.TestRunJob{}).Name()
			gvk := chaosv1.GroupVersion.WithKind(kind)

			controllerRef := metav1.NewControllerRef(createdTestrunJob, gvk)
			testJob.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
			Expect(k8sClient.Create(ctx, testJob)).To(Succeed())
			log.Println("ok")
			log.Printf("Job 3: %+v", createdTestrunJob)
			log.Printf("Job 3 status: %+v", createdTestrunJob.Status)
			log.Printf("Test Job 3: %+v", testJob)

			testJob.Status.Active = 2
			Expect(k8sClient.Status().Update(ctx, testJob)).To(Succeed())
			log.Println(testJob.Status)
			By("By Checking that the TestRunJob has one active job")
			//Eventually(func() ([]string, error) {
			//	err := k8sClient.Get(ctx, lookupKey, createdTestrunJob)
			//	if err != nil {
			//		return nil, err
			//	}
			//	names := []string{}
			//	for _, job := range createdTestrunJob.Status.Active {
			//		names = append(names, job.Name)
			//	}
			//	return names, nil
			//}, timeout, interval).Should(ConsistOf(JobName))
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, lookupKey, createdTestrunJob)).To(Succeed(), "should Get the TestRunJob")
				log.Printf("Job 4: %+v", createdTestrunJob)
				log.Printf("Job 4 status: %+v", createdTestrunJob.Status)
				g.Expect(createdTestrunJob.Status.Active).To(HaveLen(1), "should have exactly one active job")
				g.Expect(createdTestrunJob.Status.Active[0].Name).To(Equal(JobName), "the wrong job is active")

			}, timeout, interval).Should(Succeed(), "Should list the active job", JobName)
		})
	})
})
