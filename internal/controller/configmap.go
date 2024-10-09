package controller

import (
	"context"
	"fmt"
	chaosv1 "github.com/maliciousbucket/pluck/api/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	configMapField    = ""
	envConfigMapField = ""
	testNameLabel     = "galah-monitoring.io/test-name"
	testEnvLabel      = "galah-monitoring.io/test-env"
)

func (r *TestRunJobReconciler) getScriptConfigMapForJob(ctx context.Context, testRunJob *chaosv1.TestRunJob, req ctrl.Request) (string, error) {
	var configMapVersion string
	var ownedConfigMaps corev1.ConfigMapList
	logger := log.FromContext(ctx)

	listErr := r.List(ctx, &ownedConfigMaps, client.InNamespace(req.Namespace), client.MatchingLabels{testNameLabel: testRunJob.Spec.TestName})

	if listErr != nil {
		logger.Error(listErr, "Failed to list owned config maps: %v")
		if !errors.IsNotFound(listErr) {
			logger.Error(listErr, "error listing owned config maps: %v")
			return "", listErr
		}
	}
	if len(ownedConfigMaps.Items) > 0 {
		logger.V(0).Info("Found %d owned config maps", "count", len(ownedConfigMaps.Items))
		owned := ownedConfigMaps.Items[len(ownedConfigMaps.Items)-1]
		configMapRef, err := reference.GetReference(r.Scheme, &owned)
		if err != nil {
			return "", err
		}
		if testRunJob.Status.ScriptConfigMap != configMapRef {
			testRunJob.Status.ScriptConfigMap = configMapRef
		} else {
			return owned.ResourceVersion, nil
		}

	}

	configMapName := testRunJob.Spec.ScriptConfigMap
	foundConfigMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: testRunJob.Namespace}, foundConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "configmap not found")
			return "", fmt.Errorf("configMap %s not found", configMapName)
		} else {
			return "", err
		}
	}
	configMapRef, err := reference.GetReference(r.Scheme, foundConfigMap)
	if err != nil {
		return "", err
	}
	testRunJob.Status.ScriptConfigMap = configMapRef
	if foundConfigMap.Labels[testNameLabel] != testRunJob.Spec.TestName {
		logger.V(0).Info("Updating config map %s with name %s", "configmap", configMapName)
		foundConfigMap.Labels[testNameLabel] = testRunJob.Spec.TestName
		if err = r.Update(ctx, foundConfigMap); err != nil {
			return "", err
		}
	}

	configMapVersion = foundConfigMap.ResourceVersion

	return configMapVersion, nil
}

func (r *TestRunJobReconciler) getEnvConfigMapForJob(ctx context.Context, testRunJob *chaosv1.TestRunJob, req ctrl.Request) (string, error) {
	logger := log.FromContext(ctx)
	if testRunJob.Spec.EnvConfigMap == "" && (testRunJob.Spec.Env == nil || len(testRunJob.Spec.Env) == 0) {
		logger.V(0).Info("no env required for job", "testrunjob", testRunJob.Name)
		return "", nil
	}
	var configMapVersion string
	if testRunJob.Status.EnvConfigMap != nil && testRunJob.Spec.EnvConfigMap != "" {
		var ownedConfigMaps corev1.ConfigMapList
		err := r.List(ctx, &ownedConfigMaps, client.InNamespace(req.Namespace), client.MatchingLabels{testEnvLabel: testRunJob.Spec.TestName})
		if err != nil {
			logger.Error(err, "failed to list owned env config maps: %v")
			return "", err
		}
		if len(ownedConfigMaps.Items) > 0 {
			for _, owned := range ownedConfigMaps.Items {
				configMapRef, err := reference.GetReference(r.Scheme, &owned)
				if err != nil {
					logger.Error(err, "failed to get env config map ref")
					return "", err
				}
				if testRunJob.Status.ScriptConfigMap == configMapRef {
					return owned.ResourceVersion, nil
				}
			}
			logger.V(0).Info("owned items < 0", "items", len(ownedConfigMaps.Items))
			return "", err
		}
	}

	foundConfigMap := &corev1.ConfigMap{}
	configMapName := testRunJob.Spec.EnvConfigMap
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: testRunJob.Namespace}, foundConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(0).Info("env config map not found", "configmap", configMapName)
			return r.createEnvConfigMapForJob(ctx, testRunJob)
		}
		return "", err
	}
	if testRunJob.Spec.Env != nil && len(testRunJob.Spec.Env) > 0 {
		for _, env := range testRunJob.Spec.Env {
			foundConfigMap.Data[env.Name] = env.Value
		}
	}
	if foundConfigMap.Labels[testNameLabel] != testRunJob.Spec.TestName {
		foundConfigMap.Labels[testNameLabel] = testRunJob.Spec.TestName

	}
	logger.V(0).Info("updating env config map: %s", "configmap", foundConfigMap.Name)
	err = r.Update(ctx, foundConfigMap)
	if err != nil {
		return "", err
	}

	configMapVersion = foundConfigMap.ResourceVersion
	configMapRef, err := reference.GetReference(r.Scheme, foundConfigMap)
	if err != nil {
		return "", err
	}
	testRunJob.Status.EnvConfigMap = configMapRef

	return configMapVersion, nil
}

func (r *TestRunJobReconciler) createEnvConfigMapForJob(ctx context.Context, testRunJob *chaosv1.TestRunJob) (string, error) {
	logger := log.FromContext(ctx)
	if testRunJob.Spec.Env == nil || len(testRunJob.Spec.Env) == 0 {
		return "", nil
	}

	logger.V(0).Info("creating env config map", "newConfigMap", testRunJob.Spec.EnvConfigMap)

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
	configMap.Labels[testEnvLabel] = testRunJob.Spec.TestName
	logger.Info("creating config map", "configmap", configMap)

	err := r.Create(ctx, configMap)
	if err != nil {
		return "", err
	}
	configMapRef, err := reference.GetReference(r.Scheme, configMap)
	if err != nil {
		return "", err
	}
	testRunJob.Status.EnvConfigMap = configMapRef

	configMapVersion = configMap.ResourceVersion

	return configMapVersion, nil
}
