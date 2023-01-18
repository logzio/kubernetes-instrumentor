package controllers

import (
	"context"
	"errors"
	"github.com/go-logr/logr"
	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common/consts"
	"github.com/logzio/kubernetes-instrumentor/instrumentor/patch"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	SkipAnnotation = "logz.io/skip"
)

func shouldSkip(annotations map[string]string, namespace string) bool {
	for k, v := range annotations {
		if k == SkipAnnotation && v == "true" {
			return true
		}
	}

	for _, ns := range consts.IgnoredNamespaces {
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
			logger.Error(err, "error creating InstrumentedApp object, failed to set controller reference")
			return err
		}

		err = c.Create(ctx, &instrumentedApp)
		if err != nil {
			logger.Error(err, "error creating InstrumentedApp object")
			return err
		}

		instrumentedApp.Status = apiV1.InstrumentedApplicationStatus{
			InstrumentationDetection: apiV1.InstrumentationStatus{
				Phase: apiV1.PendingInstrumentationDetectionPhase,
			},
		}
		err = c.Status().Update(ctx, &instrumentedApp)
		if err != nil {
			logger.Error(err, "error updating InstrumentedApp object with phase")
		}

		return nil
	}

	if len(instApps.Items) > 1 {
		return errors.New("found more than one InstrumentedApp")
	}

	// if lang not detected - stay in function and check for app detection
	instApp := instApps.Items[0]
	if instApp.Status.InstrumentationDetection.Phase != apiV1.CompletedInstrumentationDetectionPhase {
		return nil
	}

	if shouldInstrument(podTemplateSpec, logger) {
		err = processInstrumentedApps(ctx, podTemplateSpec, instApp, logger, c, object)
		if err != nil {
			logger.Error(err, "Encountered an error while trying to process instrumented apps")
			return err
		}
	}

	if len(instApp.Spec.Applications) == 0 || instApp.Status.InstrumentationDetection.Phase != apiV1.CompletedInstrumentationDetectionPhase {
		logger.V(0).Info("No new applications detected or app detection is still in progress", "container", instApp.Name, "detectedapp", instApp.Spec.Applications, "appstatus", instApp.Status.InstrumentationDetection.Phase)
		return nil
	}

	if shouldDetectApps(podTemplateSpec, logger) {
		err = processDetectedApps(ctx, req, c, podTemplateSpec, instApp, logger, object)
		if err != nil {
			logger.Error(err, "Encountered an error while trying to process detected apps")
		}
	}

	return err
}

func processInstrumentedApps(ctx context.Context, podTemplateSpec *v1.PodTemplateSpec, instApp apiV1.InstrumentedApplication, logger logr.Logger, c client.Client, object client.Object) error {
	instrumented, err := patch.IsInstrumented(podTemplateSpec, &instApp)
	if err != nil {
		logger.Error(err, "error computing instrumented status")
		return err
	}
	if instrumented != instApp.Status.Instrumented {
		logger.V(0).Info("updating .status.instrumented", "instrumented", instrumented)
		instApp.Status.Instrumented = instrumented
		err = c.Status().Update(ctx, &instApp)
		if err != nil {
			logger.Error(err, "error computing instrumented status")
			return err
		}
	}
	annotations := podTemplateSpec.GetAnnotations()
	// If logz.io/instrument is set to "rollback" and the app is instrumented, then rollback the instrumentation
	if instrumented && annotations[patch.InstrumentAnnotation] == "rollback" {
		err = patch.RollbackPatch(podTemplateSpec, &instApp)
		if err != nil {
			logger.Error(err, "error unpatching deployment / statefulset")
			return err
		}
		err = c.Update(ctx, object)
		if err != nil {
			logger.Error(err, "error updating application")
			return err
		}
	}

	// If not instrumented - patch deployment
	if !instrumented {
		logger.V(0).Info("Instrumenting pod: " + podTemplateSpec.GetName())
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
	return nil
}

func processDetectedApps(ctx context.Context, req *ctrl.Request, c client.Client, podTemplateSpec *v1.PodTemplateSpec, instApp apiV1.InstrumentedApplication, logger logr.Logger, object client.Object) error {
	logger.V(0).Info("Starting app detection")
	detected, err := patch.IsDetected(ctx, podTemplateSpec, &instApp)
	if err != nil {
		logger.Error(err, "error computing instrumented app status for annotation patching")
		return err
	}

	if detected != instApp.Status.AppDetected {
		instApp.Status.AppDetected = detected
		c.Get(ctx, req.NamespacedName, &instApp)
		instApp.Status.AppDetected = detected
		err = c.Status().Update(ctx, &instApp)
		if err != nil {
			logger.Error(err, "Error computing instrumented app status for annotation patching")
		}
	}

	if detected {
		err = patch.ModifyObjectWithAnnotation(ctx, &instApp, object)
		if err != nil {
			logger.Error(err, "error patching deployment / statefulset with annotation")
			return err
		}
	}
	return nil
}

func shouldInstrument(podSpec *v1.PodTemplateSpec, logger logr.Logger) bool {
	annotations := podSpec.GetAnnotations()
	logger.V(0).Info("Checking if should instrument", "annotations", annotations)
	if val, exists := annotations[patch.SkipAppDetectionAnnotation]; exists && val == "true" {
		logger.V(0).Info("skipping instrumentation, skip annotation was set")
		return false
	}
	if annotations[patch.InstrumentAnnotation] == "true" {
		return true
	} else {
		logger.V(0).Info("skipping instrumentation, `logz.io/instrument` Is not set to true")
		return false
	}
}

func shouldDetectApps(podSpec *v1.PodTemplateSpec, logger logr.Logger) bool {
	annotations := podSpec.GetAnnotations()
	if val, exists := annotations[patch.SkipAppDetectionAnnotation]; exists && val == "true" {
		logger.V(0).Info("skipping app detection, skip annotation was set")
		return false
	}

	if _, exists := annotations[patch.ApplicationTypeAnnotation]; exists {
		logger.V(0).Info("skipping app detection, application type annotation already exists")
		return false
	}

	return true
}

func isDataCollectionReady(ctx context.Context, c client.Client) bool {
	return true
}

func getInstrumentedApps(ctx context.Context, req *ctrl.Request, c client.Client, ownerKey string) (*apiV1.InstrumentedApplicationList, error) {
	var instrumentedApps apiV1.InstrumentedApplicationList
	err := c.List(ctx, &instrumentedApps, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: req.Name})
	if err != nil {
		return nil, err
	}

	return &instrumentedApps, nil
}
