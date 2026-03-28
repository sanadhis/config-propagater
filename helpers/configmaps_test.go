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

func Test_CMCheckOwner(t *testing.T) {
	tests := map[string]struct {
		configmap *corev1.ConfigMap
		expected  bool
	}{
		"managed by controller": {
			configmap: &corev1.ConfigMap{
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
			configmap: &corev1.ConfigMap{
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
			result := IsConfigMapManagedByController(test.configmap)
			assert.Equal(t, test.expected, result)
		})
	}
}

func Test_CMHasPropagationEnabled(t *testing.T) {
	tests := map[string]struct {
		configmap *corev1.ConfigMap
		expected  bool
	}{
		"propagation enabled": {
			configmap: &corev1.ConfigMap{
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
			configmap: &corev1.ConfigMap{
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
			configmap: &corev1.ConfigMap{
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
			result := EnabledPropagationFromConfigMapAnnotation(test.configmap)
			assert.Equal(t, test.expected, result)
		})
	}
}

func Test_CMgetPropagationNamespaces(t *testing.T) {
	tests := map[string]struct {
		configmap *corev1.ConfigMap
		expected  int
	}{
		"propagation namespaces defined": {
			configmap: &corev1.ConfigMap{
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
			configmap: &corev1.ConfigMap{
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
			result := len(GetPropagationNamespaceFromConfigMapAnnotation(test.configmap))
			assert.Equal(t, test.expected, result)
		})
	}
}

func Test_EnsureCMExist(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add corev1 to scheme: %v", err)
	}

	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-configmap",
			Namespace: "default",
		},
	}

	tests := map[string]struct {
		client          ctrlclient.Client
		configmap       *corev1.ConfigMap
		targetNamespace string
		wantErr         error
	}{
		"configmap does not exist in target namespace": {
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(existing).
				Build(),
			configmap:       existing,
			targetNamespace: "kube-system",
			wantErr:         nil,
		},
		"object exists": {
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(existing).
				Build(),
			configmap:       existing,
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
			configmap:       existing,
			targetNamespace: "not-existing",
			wantErr:         errors.New("object not found"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := EnsureConfigMapInNamespace(context.Background(), test.client, test.configmap, test.targetNamespace)
			assert.Equal(t, test.wantErr, err)
		})
	}
}

func Test_copyConfigMapToNamespace(t *testing.T) {
	tests := map[string]struct {
		configmap       *corev1.ConfigMap
		targetNamespace string
		expectedLabes   map[string]string
	}{
		"new configmap in target namespace": {
			configmap: &corev1.ConfigMap{
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
			result := copyConfigMapToNamespace(test.configmap, test.targetNamespace)
			assert.Equal(t, test.targetNamespace, result.Namespace)
			assert.Equal(t, test.expectedLabes[ManagedByLabel], result.Labels[ManagedByLabel])
			assert.Equal(t, test.expectedLabes[AppNamespaceLabel], "default")
		})
	}
}
