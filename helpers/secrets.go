package helpers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func IsSecretManagedByController(secret *corev1.Secret) bool {
	return IsManagedByPropagationController(secret.Labels)
}

func EnabledPropagationFromSecretAnnotation(secret *corev1.Secret) bool {
	return hasPropagationEnabledAnnotation(secret.Annotations)
}

func GetPropagationNamespaceFromSecretAnnotation(secret *corev1.Secret) []string {
	return getPropagationNamespacesFromAnnotations(secret.Annotations)
}

func EnsureSecretInNamespace(ctx context.Context, client ctrlclient.Client,
	secret *corev1.Secret, targetNamespace string) error {
	logger := log.FromContext(ctx).WithName("EnsureSecretInNamespace").WithValues(
		"secret", secret.Name, "namespace", targetNamespace)

	secretInNamespace := &corev1.Secret{}
	exists, err := getObjectIfExists(ctx, client, ctrlclient.ObjectKey{
		Name:      secret.Name,
		Namespace: targetNamespace,
	}, secretInNamespace)
	if err != nil {
		logger.Error(err, "Failed to get Secret in namespace")
		return err
	}

	if !exists {
		newSecret := copySecretToNamespace(secret, targetNamespace)
		return client.Create(ctx, newSecret)
	}

	logger.Info("Secret already exists in namespace")
	return nil
}

func copySecretToNamespace(secret *corev1.Secret, targetNamespace string) *corev1.Secret {
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: targetNamespace,
			Labels:    propagatedResourceLabels(secret.Namespace),
		},
		Type: secret.Type,
		Data: secret.Data,
	}

	return newSecret
}
