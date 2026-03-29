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

func Test_ownerReferences(t *testing.T) {
	tests := map[string]struct {
		labels   map[string]string
		expected bool
	}{
		"managed by controller": {
			labels: map[string]string{
				ManagedByLabel: ManagedByValue,
			},
			expected: true,
		},
		"not managed by controller": {
			labels:   map[string]string{},
			expected: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := IsManagedByPropagationController(test.labels)
			assert.Equal(t, test.expected, result)
		})
	}
}

func Test_propagationEnabled(t *testing.T) {
	tests := map[string]struct {
		annotations map[string]string
		expected    bool
	}{
		"has propagation enabled": {
			annotations: map[string]string{
				PropagationEnableAnnotationKey: "true",
			},
			expected: true,
		},
		"has propagation enabled ignore case": {
			annotations: map[string]string{
				PropagationEnableAnnotationKey: "True",
			},
			expected: true,
		},
		"has propagation disabled": {
			annotations: map[string]string{
				PropagationEnableAnnotationKey: "random-value",
			},
			expected: false,
		},
		"has propagation annotation empty": {
			annotations: map[string]string{},
			expected:    false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := hasPropagationEnabledAnnotation(test.annotations)
			assert.Equal(t, test.expected, result)
		})
	}
}

func Test_propagationNamespaces(t *testing.T) {
	tests := map[string]struct {
		annotations map[string]string
		expected    int
	}{
		"namespaces not specified": {
			annotations: map[string]string{},
			expected:    0,
		},
		"single namespace target": {
			annotations: map[string]string{
				PropagationNamespaceAnnotationKey: "default",
			},
			expected: 1,
		},
		"multiple namespace target": {
			annotations: map[string]string{
				PropagationNamespaceAnnotationKey: "default,kube-system,kube-public",
			},
			expected: 3,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := len(getPropagationNamespacesFromAnnotations(test.annotations))
			assert.Equal(t, test.expected, result)
		})
	}
}

func Test_getObjectIfExists(t *testing.T) {
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
		client  ctrlclient.Client
		key     ctrlclient.ObjectKey
		obj     ctrlclient.Object
		exists  bool
		wantErr error
	}{
		"object exists": {
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(existing).
				Build(),
			key:     ctrlclient.ObjectKey{Name: "existing-configmap", Namespace: "default"},
			obj:     &corev1.ConfigMap{},
			exists:  true,
			wantErr: nil,
		},
		"object not found": {
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			key:     ctrlclient.ObjectKey{Name: "non-existing-configmap", Namespace: "default"},
			obj:     &corev1.ConfigMap{},
			exists:  false,
			wantErr: nil,
		},
		"unexpected error": {
			client: &utils.ErrorClient{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
				GetErr: errors.New("object not found"),
			},
			key:     ctrlclient.ObjectKey{Name: "non-existing-configmap", Namespace: "default"},
			obj:     &corev1.ConfigMap{},
			exists:  false,
			wantErr: errors.New("object not found"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			exists, err := getObjectIfExists(context.Background(), test.client, test.key, test.obj)
			assert.Equal(t, test.wantErr, err)
			assert.Equal(t, test.exists, exists)
		})
	}
}

func Test_propagateResourceLabels(t *testing.T) {
	tests := map[string]struct {
		sourceNamespace string
		expected        map[string]string
	}{
		"resource labels are set": {
			sourceNamespace: "default",
			expected: map[string]string{
				ManagedByLabel:    ManagedByValue,
				AppNamespaceLabel: "default",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := propagatedResourceLabels(test.sourceNamespace)
			assert.Equal(t, test.expected[AppNamespaceLabel], result[AppNamespaceLabel])
			assert.Equal(t, test.expected[ManagedByLabel], result[ManagedByLabel])
		})
	}
}
