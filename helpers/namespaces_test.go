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

func Test_getAllNamespaces(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add corev1 to scheme: %v", err)
	}

	existing := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kube-system",
				},
			},
		},
	}

	tests := map[string]struct {
		client   ctrlclient.Client
		expected int
		wantErr  error
	}{
		"there are 2 namespaces": {
			client: &utils.ErrorClient{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					WithLists(existing).
					Build(),
				ListErr: nil,
			},
			expected: 2,
			wantErr:  nil,
		},
		"unexpected error": {
			client: &utils.ErrorClient{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
				ListErr: errors.New("failed to list namespaces"),
			},
			expected: 0,
			wantErr:  errors.New("failed to list namespaces"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			namespaces, err := GetAllNamespaces(context.Background(), test.client)
			assert.Equal(t, test.expected, len(namespaces))
			assert.Equal(t, test.wantErr, err)
		})
	}
}

func Test_verifyNamespaceExists(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add corev1 to scheme: %v", err)
	}

	existing := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "existing-namespace",
		},
	}

	tests := map[string]struct {
		client    ctrlclient.Client
		namespace string
		wantErr   error
	}{
		"namespace does not exist": {
			client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(existing).
				Build(),
			namespace: "existing-namespace",
			wantErr:   nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ok, err := VerifyNamespaceExists(context.Background(), test.client, test.namespace)
			assert.Equal(t, true, ok)
			assert.Equal(t, test.wantErr, err)
		})
	}
}
