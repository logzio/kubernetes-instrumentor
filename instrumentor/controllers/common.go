package controllers

import (
	"context"
	"errors"
	"github.com/go-logr/logr"
	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common/consts"
	"github.com/logzio/kubernetes-instrumentor/common/utils"
	"github.com/logzio/kubernetes-instrumentor/instrumentor/patch"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	IgnoredNamespaces = []string{"kube-system", "local-path-storage", "istio-system", "linkerd", consts.DefaultNamespace}
	SkipAnnotation    = "logzio.io/skip"
)

func shouldSkip(annotations map[string]string, namespace string) bool {
	for k, v := range annotations {
		if k == SkipAnnotation && v == "true" {
			return true
		}
	}

	for _, ns := range IgnoredNamespaces {
		if namespace == ns {
			return true
		}
	}

	return false
}

func syncInstrumentedApps(ctx context.Context, req *ctrl.Request, c client.Client, scheme *runtime.Scheme,
	readyReplicas int32, object client.Object, podTemplateSpec *v1.PodTemplateSpec, ownerKey string) error {

	logger := log.FromContext(ctx)
	instApps, err := getInstrumentedApps(ctx, req, c, ownerKey)
	if err != nil {
		logger.Error(err, "error finding InstrumentedApp objects")
		return err
	}

	if len(instApps.Items) == 0 {
		if readyReplicas == 0 {
			logger.V(0).Info("not enough ready replicas, waiting for pods to be ready")
			return nil
		}

		instrumentedApp := apiV1.InstrumentedApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
			},
			Spec: apiV1.InstrumentedApplicationSpec{
				WaitingForDataCollection: !isDataCollectionReady(ctx, c),
			},
		}

		err = ctrl.SetControllerReference(object, &instrumentedApp, scheme)
		if err != nil {
			logger.Error(err, "error creating InstrumentedApp object")
			return err
		}

		err = c.Create(ctx, &instrumentedApp)
		if err != nil {
			logger.Error(err, "error creating InstrumentedApp object")
			return err
		}

		instrumentedApp.Status = apiV1.InstrumentedApplicationStatus{
			LangDetection: apiV1.LangDetectionStatus{
				Phase: apiV1.PendingLangDetectionPhase,
			},
		}
		err = c.Status().Update(ctx, &instrumentedApp)
		if err != nil {
			logger.Error(err, "error creating InstrumentedApp object")
		}

		return nil
	}

	if len(instApps.Items) > 1 {
		return errors.New("found more than one InstrumentedApp")
	}

	// If lang not detected yet - nothing to do
	instApp := instApps.Items[0]
	if len(instApp.Spec.Languages) == 0 || instApp.Status.LangDetection.Phase != apiV1.CompletedLangDetectionPhase {
		return nil
	}

	// if instrumentation conditions are met
	if shouldInstrument(ctx, &instApp, c, logger) {
		// Compute .status.instrumented field
		instrumneted, err := patch.IsInstrumented(podTemplateSpec, &instApp)
		if err != nil {
			logger.Error(err, "error computing instrumented status")
			return err
		}
		if instrumneted != instApp.Status.Instrumented {
			logger.V(0).Info("updating .status.instrumented", "instrumented", instrumneted)
			instApp.Status.Instrumented = instrumneted
			err = c.Status().Update(ctx, &instApp)
			if err != nil {
				logger.Error(err, "error computing instrumented status")
				return err
			}
		}

		// If not instrumented - patch deployment
		if !instrumneted {
			err = patch.ModifyObject(podTemplateSpec, &instApp)
			if err != nil {
				logger.Error(err, "error patching deployment / statefulset")
				return err
			}

			err = c.Update(ctx, object)
			if err != nil {
				logger.Error(err, "error instrumenting application")
				return err
			}
		}
	}

	return nil
}

// TODO implement or delete
func shouldInstrument(ctx context.Context, instApp *apiV1.InstrumentedApplication, c client.Client, logger logr.Logger) bool {
	return true
}

func isDataCollectionReady(ctx context.Context, c client.Client) bool {
	logger := log.FromContext(ctx)
	var collectorGroups apiV1.CollectorsGroupList
	err := c.List(ctx, &collectorGroups, client.InNamespace(utils.GetCurrentNamespace()))
	if err != nil {
		logger.Error(err, "error getting collectors groups, skipping instrumentation")
		return false
	}

	for _, cg := range collectorGroups.Items {
		if cg.Spec.Role == apiV1.CollectorsGroupRoleDataCollection && cg.Status.Ready {
			return true
		}
	}

	return false
}

func getInstrumentedApps(ctx context.Context, req *ctrl.Request, c client.Client, ownerKey string) (*apiV1.InstrumentedApplicationList, error) {
	var instrumentedApps apiV1.InstrumentedApplicationList
	err := c.List(ctx, &instrumentedApps, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: req.Name})
	if err != nil {
		return nil, err
	}

	return &instrumentedApps, nil
}
