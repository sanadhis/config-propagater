package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/sanadhis/config-propagator/helpers"
	"github.com/sanadhis/config-propagator/test/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Test_secretControllerReconcile(t *testing.T) {
	secretToAllNs := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Annotations: map[string]string{
				helpers.PropagationEnableAnnotationKey: "true",
			},
		},
	}
	secretToSelectedNs := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Annotations: map[string]string{
				helpers.PropagationEnableAnnotationKey:    "true",
				helpers.PropagationNamespaceAnnotationKey: "default,ns2",
			},
		},
	}
	secretSkipped1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Annotations: map[string]string{
				helpers.PropagationEnableAnnotationKey: "false",
			},
		},
	}
	secretSkipped2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels: map[string]string{
				helpers.ManagedByLabel: helpers.ManagedByValue,
			},
		},
	}
	defaultReconcileRequest := reconcile.Request{
		NamespacedName: client.ObjectKey{
			Name:      "foo",
			Namespace: "default",
		},
	}

	tests := map[string]struct {
		client           client.Client
		reconcileRequest reconcile.Request
		wantErr          error
	}{
		"secret is propagated to all namespaces": {
			client: fake.NewClientBuilder().
				WithObjects(secretToAllNs).
				Build(),
			reconcileRequest: defaultReconcileRequest,
			wantErr:          nil,
		},
		"secret is propagated to selected namespaces": {
			client: fake.NewClientBuilder().
				WithObjects(secretToSelectedNs).
				Build(),
			reconcileRequest: defaultReconcileRequest,
			wantErr:          nil,
		},
		"secret is skipped 1": {
			client: fake.NewClientBuilder().
				WithObjects(secretSkipped1).
				Build(),
			reconcileRequest: defaultReconcileRequest,
			wantErr:          nil,
		},
		"secret is skipped 2": {
			client: fake.NewClientBuilder().
				WithObjects(secretSkipped2).
				Build(),
			reconcileRequest: defaultReconcileRequest,
			wantErr:          nil,
		},
		"secret is not found": {
			client: &utils.ErrorClient{
				Client: fake.NewClientBuilder().
					Build(),
				GetErr: client.IgnoreNotFound(errors.New("secret not found")),
			},
			reconcileRequest: defaultReconcileRequest,
			wantErr:          client.IgnoreNotFound(errors.New("secret not found")),
		},
		"general error when getting secret": {
			client: &utils.ErrorClient{
				Client: fake.NewClientBuilder().
					Build(),
				GetErr: errors.New("failed to get secret"),
			},
			reconcileRequest: defaultReconcileRequest,
			wantErr:          errors.New("failed to get secret"),
		},
		"general error when listing namespaces": {
			client: &utils.ErrorClient{
				Client: fake.NewClientBuilder().
					WithObjects(secretToAllNs).
					Build(),
				ListErr: errors.New("failed to list namespaces"),
			},
			reconcileRequest: defaultReconcileRequest,
			wantErr:          errors.New("failed to list namespaces"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			r := &SecretController{
				Client: test.client,
				Scheme: test.client.Scheme(),
			}
			_, err := r.Reconcile(context.Background(), test.reconcileRequest)
			assert.Equal(t, test.wantErr, err)
		})
	}
}
