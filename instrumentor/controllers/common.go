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
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

const (
	SkipAnnotation    = "logz.io/skip"
	LogTypeAnnotation = "logz.io/application_type"
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

	if len(instApps.Items) > 1 {
		return errors.New("found more than one InstrumentedApp")
	}

	// if lang not detected - stay in function and check for app detection
	instApp := instApps.Items[0]
	if instApp.Status.InstrumentationDetection.Phase != apiV1.CompletedInstrumentationDetectionPhase {
		return nil
	}
	// detect log type
	err = processLogType(ctx, req, podTemplateSpec, instApp, logger, c, object)
	if err != nil {
		return err
	}
	// instrumentation detection process
	if shouldInstrument(podTemplateSpec, logger) {
		err = processInstrumentedApps(ctx, podTemplateSpec, instApp, logger, c, object)
		if err != nil {
			logger.Error(err, "Encountered an error while trying to process instrumented apps")
			return err
		}
	}
	if shouldRollBackTraces(podTemplateSpec, logger, instApp) {
		err = processRollback(ctx, podTemplateSpec, instApp, logger, c, object)
		if err != nil {
			logger.Error(err, "Encountered an error while trying to process rollback")
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

func processLogType(ctx context.Context, req *ctrl.Request, podTemplateSpec *v1.PodTemplateSpec, instApp apiV1.InstrumentedApplication, logger logr.Logger, c client.Client, object client.Object) error {
	logger.V(0).Info("checking for log type annotation")
	annotations := podTemplateSpec.GetAnnotations()
	if annotations == nil {
		logger.V(0).Info("No annotations found")
		return nil
	}
	if annotations[LogTypeAnnotation] != "" {
		logger.V(0).Info("found log type annotation", "logtype", annotations[LogTypeAnnotation])
		instApp.Spec.LogType = annotations[LogTypeAnnotation]
	}
	err := c.Update(ctx, &instApp)
	if err != nil {
		logger.Error(err, "error updating InstrumentedApp object with log type")
		return err
	}
	return nil
}

func processRollback(ctx context.Context, podTemplateSpec *v1.PodTemplateSpec, instApp apiV1.InstrumentedApplication, logger logr.Logger, c client.Client, object client.Object) error {
	instrumented, err := patch.IsTracesInstrumented(podTemplateSpec, &instApp)
	if err != nil {
		logger.Error(err, "Error computing instrumented status")
		return err
	}
	if instrumented != instApp.Status.TracesInstrumented {
		logger.V(0).Info("Updating .status.instrumented", "instrumented", instrumented)
		instApp.Status.TracesInstrumented = instrumented
		err = c.Status().Update(ctx, &instApp)
		if err != nil {
			logger.Error(err, "Error updating instrumented status")
			return err
		}
	}
	annotations := podTemplateSpec.GetAnnotations()
	if instrumented && annotations[patch.TracesInstrumentAnnotation] == "rollback" {
		err = c.Get(ctx, client.ObjectKey{Namespace: object.GetNamespace(), Name: object.GetName()}, object)
		if err != nil {
			logger.Error(err, "Error getting object")
			return err
		}
		err := patch.RollbackPatch(podTemplateSpec, &instApp)
		if err != nil {
			logger.Error(err, "Error unpatching deployment / statefulset")
			return err
		}

		logger.V(0).Info("PodSpec after patching:", "podSpec", podTemplateSpec)
		// Define an exponential backoff configuration
		backoff := wait.Backoff{
			Duration: time.Second * 2, // Initial delay
			Factor:   2,               // Factor by which the delay is exponentially increased
			Jitter:   0.1,             // Jitter to introduce some random variation in the delay
			Steps:    5,               // Number of steps to retry
		}
		// The error variable to collect all errors encountered
		var lastErr error
		// Retry logic with exponential backoff
		retryErr := wait.ExponentialBackoff(backoff, func() (bool, error) {
			updateErr := c.Update(ctx, object)
			if updateErr != nil {
				// Save the error encountered
				lastErr = updateErr
				logger.Error(updateErr, "error instrumenting application, retrying...")
				// Return false to indicate a retry should happen
				return false, nil
			}
			// Return true to indicate the function was successful
			return true, nil
		})

		if retryErr != nil || lastErr != nil {
			if retryErr != nil {
				logger.Error(lastErr, "error after retrying")
			}
			return lastErr
		}

		logger.V(0).Info("Successfully rolled back instrumentation, changing instrumented app status to not instrumented")
		instApp.Status.TracesInstrumented = false
	}
	return nil
}

func processInstrumentedApps(ctx context.Context, podTemplateSpec *v1.PodTemplateSpec, instApp apiV1.InstrumentedApplication, logger logr.Logger, c client.Client, object client.Object) error {
	instrumented, err := patch.IsTracesInstrumented(podTemplateSpec, &instApp)
	if err != nil {
		logger.Error(err, "error computing instrumented status")
		return err
	}
	if instrumented != instApp.Status.TracesInstrumented {
		logger.V(0).Info("updating .status.instrumented", "instrumented", instrumented)
		instApp.Status.TracesInstrumented = instrumented
		err = c.Status().Update(ctx, &instApp)
		if err != nil {
			logger.Error(err, "error computing traces instrumented status")
			return err
		}
	}
	// If not instrumented - patch deployment
	if !instrumented {
		err = c.Get(ctx, client.ObjectKey{Namespace: object.GetNamespace(), Name: object.GetName()}, object)
		if err != nil {
			logger.Error(err, "Error getting object")
			return err
		}
		logger.V(0).Info("Instrumenting pod: " + podTemplateSpec.GetName())
		err = patch.ModifyObject(podTemplateSpec, &instApp)
		if err != nil {
			logger.Error(err, "error patching deployment / statefulset")
			return err
		}
		// Define an exponential backoff configuration
		backoff := wait.Backoff{
			Duration: time.Second * 2, // Initial delay
			Factor:   2,               // Factor by which the delay is exponentially increased
			Jitter:   0.1,             // Jitter to introduce some random variation in the delay
			Steps:    5,               // Number of steps to retry
		}
		// The error variable to collect all errors encountered
		var lastErr error
		// Retry logic with exponential backoff
		retryErr := wait.ExponentialBackoff(backoff, func() (bool, error) {
			updateErr := c.Update(ctx, object)
			if updateErr != nil {
				// Save the error encountered
				lastErr = updateErr
				logger.Error(updateErr, "error instrumenting application, retrying...")
				// Return false to indicate a retry should happen
				return false, nil
			}
			// Return true to indicate the function was successful
			return true, nil
		})

		if retryErr != nil || lastErr != nil {
			if retryErr != nil {
				logger.Error(lastErr, "error after retrying")
			}
			return lastErr
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

	return nil
}

func shouldRollBackTraces(podTemplateSpec *v1.PodTemplateSpec, logger logr.Logger, instApp apiV1.InstrumentedApplication) bool {
	annotations := podTemplateSpec.GetAnnotations()
	if annotations[patch.TracesInstrumentAnnotation] == "rollback" && instApp.Status.TracesInstrumented {
		logger.V(0).Info("rollback annotation detected for traces instrumented app")
		return true
	}
	return false
}

func shouldInstrument(podSpec *v1.PodTemplateSpec, logger logr.Logger) bool {
	annotations := podSpec.GetAnnotations()
	logger.V(0).Info("Checking if should instrument", "pod spec", podSpec, "pod name", podSpec.GetName())
	if val, exists := annotations[patch.SkipAppDetectionAnnotation]; exists && val == "true" {
		logger.V(0).Info("skipping instrumentation, skip annotation was set")
		return false
	}
	// if logz.io/instrument is set to "true" - instrument the app
	if annotations[patch.TracesInstrumentAnnotation] == "true" {
		return true
	} else {
		logger.V(0).Info("skipping instrumentation according to `logz.io/instrument` annotation")
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
