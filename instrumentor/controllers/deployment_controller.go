/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Credits: https://github.com/keyval-dev/odigos
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	v1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common/consts"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	instAppDepOwnerKey = ".metadata.deployment.controller"
)

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile is responsible for creating InstrumentedApplication objects for every Deployment.
// In addition, Reconcile patch the deployment according to the discovered language and keeps the `instrumented` field
// of InstrumentedApplication up to date with the deployment spec.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dep appsv1.Deployment
	err := r.Get(ctx, req.NamespacedName, &dep)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching deployment object")
		return ctrl.Result{}, err
	}

	err = r.instrumentDeployment(ctx, req, logger, dep)
	if err != nil {
		if apierrors.IsConflict(err) {
			logger.V(0).Info("Conflict encountered and ignored during update")
		} else {
			logger.Error(err, "Encountered an error while trying to instrument deployment")
		}
	}

	return ctrl.Result{}, nil
}

func (r *DeploymentReconciler) instrumentDeployment(ctx context.Context, req ctrl.Request, logger logr.Logger, dep appsv1.Deployment) error {
	if shouldSkip(dep.Annotations, dep.Namespace) {
		logger.V(5).Info("skipped instrumentation for deployment")
		return nil
	}
	err := syncInstrumentedApps(ctx, &req, r.Client, r.Scheme, dep.Status.ReadyReplicas, &dep, &dep.Spec.Template, instAppDepOwnerKey)
	if err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index InstrumentedApps by owner for fast lookup
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.InstrumentedApplication{}, instAppDepOwnerKey, func(rawObj client.Object) []string {
		instApp := rawObj.(*v1.InstrumentedApplication)
		owner := metav1.GetControllerOf(instApp)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != appsv1.SchemeGroupVersion.String() || owner.Kind != consts.SupportedResourceDeployment {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Owns(&v1.InstrumentedApplication{}).
		Complete(r)
}
