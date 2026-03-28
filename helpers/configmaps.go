package helpers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func IsConfigMapManagedByController(configmap *corev1.ConfigMap) bool {
	return IsManagedByPropagationController(configmap.Labels)
}

func EnabledPropagationFromConfigMapAnnotation(configmap *corev1.ConfigMap) bool {
	return hasPropagationEnabledAnnotation(configmap.Annotations)
}

func GetPropagationNamespaceFromConfigMapAnnotation(configmap *corev1.ConfigMap) []string {
	return getPropagationNamespacesFromAnnotations(configmap.Annotations)
}

func EnsureConfigMapInNamespace(ctx context.Context, client ctrlclient.Client,
	configmap *corev1.ConfigMap, targetNamespace string) error {
	logger := log.FromContext(ctx).WithName("EnsureConfigMapInNamespace").WithValues(
		"configmap", configmap.Name, "namespace", targetNamespace)

	configMapInNamespace := &corev1.ConfigMap{}
	exists, err := getObjectIfExists(ctx, client, ctrlclient.ObjectKey{
		Name:      configmap.Name,
		Namespace: targetNamespace,
	}, configMapInNamespace)
	if err != nil {
		logger.Error(err, "Failed to get ConfigMap in namespace")
		return err
	}

	if !exists {
		newConfigMap := copyConfigMapToNamespace(configmap, targetNamespace)
		return client.Create(ctx, newConfigMap)
	}

	logger.Info("ConfigMap already exists in namespace")
	return nil
}

func copyConfigMapToNamespace(configmap *corev1.ConfigMap, targetNamespace string) *corev1.ConfigMap {
	newConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmap.Name,
			Namespace: targetNamespace,
			Labels:    propagatedResourceLabels(configmap.Namespace),
		},
		Data: configmap.Data,
	}

	return newConfigMap
}
