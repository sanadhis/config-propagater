package controllers

import (
	"context"

	"github.com/sanadhis/config-propagator/helpers"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ConfigMapController struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ConfigMapController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	o := &corev1.ConfigMap{}
	if err := r.Get(ctx, req.NamespacedName, o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "Failed to get ConfigMap resource",
			"configmap", o.Name, "namespace", o.Namespace)
		return ctrl.Result{}, err
	}

	if o.GetDeletionTimestamp() != nil {
		logger.Info("Got deletion timestamp for Secret, skipping",
			"configmap", o.Name, "namespace", o.Namespace)
		return ctrl.Result{}, r.handleDeletion(ctx, o)
	}

	if !helpers.EnabledPropagationFromConfigMapAnnotation(o) {
		logger.V(2).Info("Secret does not have propagation annotation, skipping",
			"configmap", o.Name, "namespace", o.Namespace)
		return ctrl.Result{}, nil
	}

	if helpers.IsConfigMapManagedByController(o) {
		logger.V(2).Info("ConfigMap is managed by the controller, skipping to avoid loops",
			"configmap", o.Name, "namespace", o.Namespace)
		return ctrl.Result{}, nil
	}

	var namespaces []string
	if len(helpers.GetPropagationNamespaceFromConfigMapAnnotation(o)) > 0 {
		namespaces = helpers.GetPropagationNamespaceFromConfigMapAnnotation(o)
	} else {
		var err error
		namespaces, err = helpers.GetAllNamespaces(ctx, r.Client)
		if namespaces == nil || err != nil {
			logger.Error(err, "Failed to list namespaces")
			return ctrl.Result{}, err
		}
	}

	for _, ns := range namespaces {
		if ns == o.Namespace {
			logger.Info("ConfigMap is in the same namespace as the Namespace, skipping",
				"configmap", o.Name, "namespace", ns)
			continue
		}
		if ok, err := helpers.VerifyNamespaceExists(ctx, r.Client, ns); !ok || err != nil {
			logger.Info("[WARNING] Namespace does not exist or failed to get, skipping",
				"namespace", ns)
			continue
		}

		if err := helpers.EnsureConfigMapInNamespace(ctx, r.Client, o, ns); err != nil {
			logger.Error(err, "Failed to ensure ConfigMap in namespace",
				"configmap", o.Name, "namespace", ns)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ConfigMapController) handleDeletion(ctx context.Context, configmap *corev1.ConfigMap) error {
	return nil
}

func (r *ConfigMapController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Complete(r)
}
