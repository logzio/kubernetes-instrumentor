package patch

import (
	"context"
	"fmt"
	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NodeIPEnvName           = "NODE_IP"
	PodNameEnvVName         = "POD_NAME"
	PodNameEnvValue         = "$(POD_NAME)"
	LogzioMonitoringService = "logzio-monitoring-otel-collector.monitoring.svc.cluster.local"
)

type Patcher interface {
	Patch(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication)
	IsInstrumented(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) bool
}

//type AnnotationPatcher interface {
//	Patch(ctx context.Context, detected *apiV1.AppDetector, object client.Object) error
//	shouldPatch(annotations map[string]string, namespace string) bool
//}

var patcherMap = map[common.ProgrammingLanguage]Patcher{
	common.JavaProgrammingLanguage:       java,
	common.PythonProgrammingLanguage:     python,
	common.DotNetProgrammingLanguage:     dotNet,
	common.JavascriptProgrammingLanguage: nodeJs,
	common.GoProgrammingLanguage:         golang,
}

var annotationPatcherMap = map[string]AnnotationPatcher{}

func ModifyObject(original *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) error {
	for _, l := range getLangsInResult(instrumentation) {
		p, exists := patcherMap[l]
		if !exists {
			return fmt.Errorf("unable to find patcher for lang %s", l)
		}

		p.Patch(original, instrumentation)
	}

	return nil
}

func IsInstrumented(original *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) (bool, error) {
	instrumented := true
	for _, l := range getLangsInResult(instrumentation) {
		p, exists := patcherMap[l]
		if !exists {
			return false, fmt.Errorf("unable to find patcher for lang %s", l)
		}

		instrumented = instrumented && p.IsInstrumented(original, instrumentation)
	}

	return instrumented, nil
}

func getLangsInResult(instrumentation *apiV1.InstrumentedApplication) []common.ProgrammingLanguage {
	langMap := make(map[common.ProgrammingLanguage]interface{})
	for _, c := range instrumentation.Spec.Languages {
		langMap[c.Language] = nil
	}

	var langs []common.ProgrammingLanguage
	for l, _ := range langMap {
		langs = append(langs, l)
	}

	return langs
}

func shouldPatch(instrumentation *apiV1.InstrumentedApplication, lang common.ProgrammingLanguage, containerName string) bool {
	for _, l := range instrumentation.Spec.Languages {
		if l.ContainerName == containerName && l.Language == lang {
			// TODO: Handle CGO
			return true
		}
	}

	return false
}

func getIndexOfEnv(envs []v1.EnvVar, name string) int {
	for i := range envs {
		if envs[i].Name == name {
			return i
		}
	}
	return -1
}

func calculateAppName(podSpace *v1.PodTemplateSpec, currentContainer *v1.Container, instrumentation *apiV1.InstrumentedApplication) string {
	if len(podSpace.Spec.Containers) > 1 {
		return currentContainer.Name
	}

	return instrumentation.ObjectMeta.OwnerReferences[0].Name
}

func getApplicationFromDetectionResult(ctx context.Context, instrumentedApplication *apiV1.InstrumentedApplication) string {
	var detectedApp = ""
	if len(instrumentedApplication.Spec.Applications) > 0 {
		detectedApp = string(instrumentedApplication.Spec.Applications[0].Application)
	}

	return detectedApp
}

func IsDetected(ctx context.Context, original *v1.PodTemplateSpec, instrumentedApp *apiV1.InstrumentedApplication) (bool, error) {
	isDetected := true
	app := getApplicationFromDetectionResult(ctx, instrumentedApp)
	if app != "" {
		p, exists := annotationPatcherMap[app]
		if !exists {
			return false, fmt.Errorf("unable to find patcher for %s", app)
		}

		isDetected = isDetected && p.shouldPatch(original.Annotations, original.Namespace)
	}

	return isDetected, nil
}

func ModifyObjectWithAnnotation(ctx context.Context, detectedApplication *apiV1.InstrumentedApplication, object client.Object) error {
	app := getApplicationFromDetectionResult(ctx, detectedApplication)
	if app != "" {
		p, exists := annotationPatcherMap[app]
		if !exists {
			return fmt.Errorf("unable to find patcher for app %s", app)
		}

		err := p.Patch(ctx, detectedApplication, object)

		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	addAnnotationPatcher()
}

func addAnnotationPatcher() {
	for _, app := range common.ProcessNameToType {
		annotationPatcherMap[app] = *AnnotationPatcherInst
	}
}
