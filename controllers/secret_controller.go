package controllers

import (
	"context"

	"github.com/sanadhis/config-propagator/helpers"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type SecretController struct {
	client.Client
	Scheme           *runtime.Scheme
	ConcurrencyLimit int
}

func (r *SecretController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	o := &corev1.Secret{}
	if err := r.Get(ctx, req.NamespacedName, o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Failed to get Secret resource",
			"secret", o.Name, "namespace", o.Namespace)
		return ctrl.Result{}, err
	}

	if o.GetDeletionTimestamp() != nil {
		logger.Info("Got deletion timestamp for Secret, skipping",
			"secret", o.Name, "namespace", o.Namespace)
		return ctrl.Result{}, r.handleDeletion(ctx, o)
	}

	if !helpers.EnabledPropagationFromSecretAnnotation(o) {
		logger.V(2).Info("Secret does not have propagation annotation, skipping",
			"secret", o.Name, "namespace", o.Namespace)
		return ctrl.Result{}, nil
	}

	if helpers.IsSecretManagedByController(o) {
		logger.V(2).Info("Secret is managed by the controller, skipping to avoid loops",
			"secret", o.Name, "namespace", o.Namespace)
		return ctrl.Result{}, nil
	}

	var namespaces []string
	if len(helpers.GetPropagationNamespaceFromSecretAnnotation(o)) > 0 {
		namespaces = helpers.GetPropagationNamespaceFromSecretAnnotation(o)
	} else {
		var err error
		namespaces, err = helpers.GetAllNamespaces(ctx, r.Client)
		if namespaces == nil || err != nil {
			logger.Error(err, "Failed to list namespaces")
			return ctrl.Result{}, err
		}
	}

	var g errgroup.Group
	if r.ConcurrencyLimit > 0 {
		g.SetLimit(r.ConcurrencyLimit)
	} else {
		// Default to a reasonable concurrency limit if not set
		g.SetLimit(10)
	}

	for _, ns := range namespaces {
		g.Go(func() error {
			if ns == o.Namespace {
				logger.Info("Secret is in the same namespace as the Namespace, skipping",
					"secret", o.Name, "namespace", ns)
				return nil
			}

			if ok, err := helpers.VerifyNamespaceExists(ctx, r.Client, ns); !ok || err != nil {
				logger.Info("[WARNING] Namespace does not exist or failed to get, skipping",
					"namespace", ns)
				return nil
			}

			if err := helpers.EnsureSecretInNamespace(ctx, r.Client, o, ns); err != nil {
				logger.Error(err, "Failed to ensure Secret in namespace",
					"secret", o.Name, "namespace", ns)
				return err
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SecretController) handleDeletion(ctx context.Context, secret *corev1.Secret) error {
	return nil
}

func (r *SecretController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Complete(r)
}
