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
	"strings"
)

const (
	SkipAnnotation             = "logz.io/skip"
	LogTypeAnnotation          = "logz.io/application_type"
	TracesInstrumentAnnotation = "logz.io/traces_instrument"
)

func shouldSkip(annotations map[string]string, namespace string) bool {
	for k, v := range annotations {
		if k == SkipAnnotation && strings.ToLower(v) == "true" {
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
	err := c.Get(ctx, req.NamespacedName, object)
	if err != nil {
		logger.Error(err, "error getting kubernetes objects")
		return err
	}
	instApps, err := getInstrumentedApps(ctx, req, c, ownerKey)
	if err != nil {
		logger.Error(err, "error finding InstrumentedApp objects")
		return err
	}
	// if no InstrumentedApp found - create one
	if len(instApps.Items) == 0 {
		if readyReplicas == 0 {
			return nil
		}

		instrumentedApp := apiV1.InstrumentedApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
			},
			Spec: apiV1.InstrumentedApplicationSpec{
				WaitingForDataCollection: !isDataCollectionReady(ctx, c),
				LogType:                  "",
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
	// if more than one InstrumentedApp found - return error
	if len(instApps.Items) > 1 {
		return errors.New("found more than one InstrumentedApp")
	}
	// if InstrumentedApp found - run process
	instApp := instApps.Items[0]
	if instApp.Status.InstrumentationDetection.Phase != apiV1.CompletedInstrumentationDetectionPhase {
		return nil
	}
	// detect log type
	err = processLogType(ctx, podTemplateSpec, &instApp, logger, c, object)
	if err != nil {
		return err
	}
	// instrumentation detection process
	if shouldInstrument(podTemplateSpec) {
		err = processInstrumentedApps(ctx, podTemplateSpec, &instApp, logger, c, object)
		if err != nil {
			return err
		}
	}
	if shouldRollBackTraces(podTemplateSpec) {
		err = processRollback(ctx, podTemplateSpec, &instApp, logger, c, object)
		if err != nil {
			return err
		}
	}

	if len(instApp.Spec.Applications) == 0 || instApp.Status.InstrumentationDetection.Phase != apiV1.CompletedInstrumentationDetectionPhase {
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

func processLogType(ctx context.Context, podTemplateSpec *v1.PodTemplateSpec, instApp *apiV1.InstrumentedApplication, logger logr.Logger, c client.Client, object client.Object) error {
	key := client.ObjectKeyFromObject(instApp)
	if err := c.Get(ctx, key, instApp); err != nil {
		return err
	}
	annotations := podTemplateSpec.GetAnnotations()
	newLogType := ""
	if annotations[LogTypeAnnotation] != "" {
		newLogType = annotations[LogTypeAnnotation]
	}
	// Update only if there is a change
	if instApp.Spec.LogType != newLogType {
		instApp.Spec.LogType = newLogType
		if err := c.Update(ctx, instApp); err != nil {
			return err
		}
	}
	return nil
}

func processRollback(ctx context.Context, podTemplateSpec *v1.PodTemplateSpec, instApp *apiV1.InstrumentedApplication, logger logr.Logger, c client.Client, object client.Object) error {
	objectKey := client.ObjectKeyFromObject(object)
	if err := c.Get(ctx, objectKey, object); err != nil {
		return err
	}
	instAppKey := client.ObjectKeyFromObject(instApp)
	if err := c.Get(ctx, instAppKey, instApp); err != nil {
		return err
	}
	instrumented, err := patch.IsTracesInstrumented(podTemplateSpec, instApp)
	if err != nil {
		return err
	}
	if instrumented != instApp.Status.TracesInstrumented {
		instApp.Status.TracesInstrumented = instrumented
		err = c.Status().Update(ctx, instApp)
		if err != nil {
			return err
		}
	}
	annotations := podTemplateSpec.GetAnnotations()
	if instrumented && strings.ToLower(annotations[TracesInstrumentAnnotation]) == "rollback" {
		logger.V(0).Info("Rolling back instrumentation", "object", object)
		err = patch.RollbackPatch(podTemplateSpec, instApp)
		if err != nil {
			return err
		}
		err = c.Update(ctx, object)
		if err != nil {
			return err
		}
		// update crd active service names due to rollback
		for i := range instApp.Spec.Languages {
			instApp.Spec.Languages[i].ActiveServiceName = ""
		}
		err = c.Update(ctx, instApp)
		if err != nil {
			return err
		}
		instApp.Status.TracesInstrumented = false
		err = c.Status().Update(ctx, instApp)
		if err != nil {
			return err
		}
		logger.V(0).Info("Successfully rolled back instrumentation, changing instrumented app status to not instrumented")
	}
	return nil
}

func processInstrumentedApps(ctx context.Context, podTemplateSpec *v1.PodTemplateSpec, instApp *apiV1.InstrumentedApplication, logger logr.Logger, c client.Client, object client.Object) error {
	objectKey := client.ObjectKeyFromObject(object)
	if err := c.Get(ctx, objectKey, object); err != nil {
		return err
	}
	instAppKey := client.ObjectKeyFromObject(instApp)
	if err := c.Get(ctx, instAppKey, instApp); err != nil {
		return err
	}
	instrumented, err := patch.IsTracesInstrumented(podTemplateSpec, instApp)
	if err != nil {
		return err
	}
	if instrumented != instApp.Status.TracesInstrumented {
		logger.V(0).Info("updating .status.instrumented", "instrumented", instrumented)
		instApp.Status.TracesInstrumented = instrumented
		err = c.Status().Update(ctx, instApp)
		if err != nil {
			return err
		}
	}
	// If not instrumented - patch deployment
	if !instrumented {
		logger.V(0).Info("Instrumenting pod")
		err = patch.ModifyObject(podTemplateSpec, instApp)
		if err != nil {
			return err
		}
		err = c.Update(ctx, object)
		if err != nil {
			return err
		}
		err = c.Update(ctx, instApp)
		if err != nil {
			return err
		}
		// instApp.Status.TracesInstrumented is a part of the status in the custom resource definition
		instApp.Status.TracesInstrumented = true
		err = c.Status().Update(ctx, instApp)
		if err != nil {
			return err
		}

	}
	// if the app is instrumented update the active service name
	if instrumented {
		err = patch.UpdateActiveServiceName(podTemplateSpec, instApp)
		if err != nil {
			return err
		}
		err = c.Update(ctx, object)
		if err != nil {
			return err
		}
		err = c.Update(ctx, instApp)
		if err != nil {
			return err
		}
	}
	logger.V(0).Info("Successfully instrumented pod: " + podTemplateSpec.GetName())
	return nil
}

func processDetectedApps(ctx context.Context, req *ctrl.Request, c client.Client, podTemplateSpec *v1.PodTemplateSpec, instApp apiV1.InstrumentedApplication, logger logr.Logger, object client.Object) error {
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

	return nil
}

func shouldRollBackTraces(podTemplateSpec *v1.PodTemplateSpec) bool {
	annotations := podTemplateSpec.GetAnnotations()
	if val, exists := annotations[TracesInstrumentAnnotation]; exists && strings.ToLower(val) == "rollback" {
		return true
	}
	return false
}

func shouldInstrument(podSpec *v1.PodTemplateSpec) bool {
	annotations := podSpec.GetAnnotations()
	if val, exists := annotations[consts.SkipAppDetectionAnnotation]; exists && strings.ToLower(val) == "true" {
		return false
	}
	// if logz.io/instrument is set to "true" - instrument the app
	if val, exists := annotations[TracesInstrumentAnnotation]; exists && strings.ToLower(val) == "true" {
		return true
	} else {
		return false
	}
}

func shouldDetectApps(podSpec *v1.PodTemplateSpec, logger logr.Logger) bool {
	annotations := podSpec.GetAnnotations()
	if val, exists := annotations[consts.SkipAppDetectionAnnotation]; exists && strings.ToLower(val) == "true" {
		logger.V(0).Info("skipping app detection, skip annotation was set")
		return false
	}

	if _, exists := annotations[consts.ApplicationTypeAnnotation]; exists {
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
