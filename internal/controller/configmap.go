package controller

import (
	"context"
	"fmt"
	chaosv1 "github.com/maliciousbucket/pluck/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	configMapField    = ""
	envConfigMapField = ""
)

func (r *TestRunJobReconciler) getScriptConfigMapForJob(ctx context.Context, testRunJob *chaosv1.TestRunJob) (string, error) {
	var configMapVersion string
	configMapName := testRunJob.Spec.ScriptConfigMap
	foundConfigMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: testRunJob.Namespace}, foundConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("configMap %s not found", configMapName)
		} else {
			return "", err
		}
	}
	configMapVersion = foundConfigMap.ResourceVersion

	return configMapVersion, nil
}

func (r *TestRunJobReconciler) getEnvConfigMapForJob(ctx context.Context, testRunJob *chaosv1.TestRunJob) (string, error) {
	if testRunJob.Spec.EnvConfigMap == "" && (testRunJob.Spec.Env == nil || len(testRunJob.Spec.Env) == 0) {
		return "", nil
	}
	var configMapVersion string
	foundConfigMap := &corev1.ConfigMap{}
	configMapName := testRunJob.Spec.EnvConfigMap
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: testRunJob.Namespace}, foundConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.createEnvConfigMapForJob(ctx, testRunJob)
		}
		return "", err
	}
	if testRunJob.Spec.Env != nil && len(testRunJob.Spec.Env) > 0 {
		for _, env := range testRunJob.Spec.Env {
			foundConfigMap.Data[env.Name] = env.Value
		}
	}
	err = r.Update(ctx, foundConfigMap)
	if err != nil {
		return "", err
	}

	configMapVersion = foundConfigMap.ResourceVersion
	return configMapVersion, nil
}

func (r *TestRunJobReconciler) createEnvConfigMapForJob(ctx context.Context, testRunJob *chaosv1.TestRunJob) (string, error) {
	if testRunJob.Spec.Env == nil || len(testRunJob.Spec.Env) == 0 {
		return "", nil
	}
	var configMapVersion string
	configMapName := fmt.Sprintf("%s-env", testRunJob.Spec.ScriptConfigMap)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: testRunJob.Namespace,
		},
	}
	data := make(map[string]string)
	for _, env := range testRunJob.Spec.Env {
		data[env.Name] = env.Value
	}
	configMap.Data = data

	err := r.Create(ctx, configMap)
	if err != nil {
		return configMapVersion, err
	}
	testRunJob.Spec.EnvConfigMap = configMap.Name
	configMapVersion = configMap.ResourceVersion

	return configMapVersion, nil
}
