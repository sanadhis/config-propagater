package helpers

import (
	"context"
	"errors"
	"testing"

	"github.com/sanadhis/config-propagator/test/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_secretCheckOwner(t *testing.T) {
	tests := map[string]struct {
		secret   *corev1.Secret
		expected bool
	}{
		"managed by controller": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
					Labels: map[string]string{
						ManagedByLabel: ManagedByValue,
					},
				},
			},
			expected: true,
		},
		"not managed by controller": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
			},
			expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := IsSecretManagedByController(test.secret)
			assert.Equal(t, test.expected, result)
		})
	}
}

func Test_secretHasPropagationEnabled(t *testing.T) {
	tests := map[string]struct {
		secret   *corev1.Secret
		expected bool
	}{
		"propagation enabled": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
					Annotations: map[string]string{
						PropagationEnableAnnotationKey: "true",
					},
				},
			},
			expected: true,
		},
		"propagation disabled": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
					Annotations: map[string]string{
						PropagationEnableAnnotationKey: "false",
					},
				},
			},
			expected: false,
		},
		"propagation disabled since it's empty": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
			},
			expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := EnabledPropagationFromSecretAnnotation(test.secret)
			assert.Equal(t, test.expected, result)
		})
	}
}

func Test_secretGetPropagationNamespaces(t *testing.T) {
	tests := map[string]struct {
		secret   *corev1.Secret
		expected int
	}{
		"propagation namespaces defined": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
					Annotations: map[string]string{
						PropagationNamespaceAnnotationKey: "default,other-namespace",
					},
				},
			},
			expected: 2,
		},
		"propagation namespaces not defined": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
			},
			expected: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := len(GetPropagationNamespaceFromSecretAnnotation(test.secret))
			assert.Equal(t, test.expected, result)
		})
	}
}

func Test_EnsureSecretExist(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add corev1 to scheme: %v", err)
	}

	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "default",
		},
	}

	tests := map[string]struct {
		client          ctrlclient.Client
		secret          *corev1.Secret
		targetNamespace string
		wantErr         error
	}{
		"secret does not exist in target namespace": {
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(existing).
				Build(),
			secret:          existing,
			targetNamespace: "kube-system",
			wantErr:         nil,
		},
		"object exists": {
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(existing).
				Build(),
			secret:          existing,
			targetNamespace: "default",
			wantErr:         nil,
		},
		"unexpected error": {
			client: &utils.ErrorClient{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
				Err: errors.New("object not found"),
			},
			secret:          existing,
			targetNamespace: "not-existing",
			wantErr:         errors.New("object not found"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := EnsureSecretInNamespace(context.Background(), test.client, test.secret, test.targetNamespace)
			assert.Equal(t, test.wantErr, err)
		})
	}
}

func Test_copySecretToNamespace(t *testing.T) {
	tests := map[string]struct {
		secret          *corev1.Secret
		targetNamespace string
		expectedLabes   map[string]string
	}{
		"new secret in target namespace": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
			},
			targetNamespace: "kube-system",
			expectedLabes: map[string]string{
				ManagedByLabel:    ManagedByValue,
				AppNamespaceLabel: "default",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := copySecretToNamespace(test.secret, test.targetNamespace)
			assert.Equal(t, test.targetNamespace, result.Namespace)
			assert.Equal(t, test.expectedLabes[ManagedByLabel], result.Labels[ManagedByLabel])
			assert.Equal(t, test.expectedLabes[AppNamespaceLabel], "default")
		})
	}
}
