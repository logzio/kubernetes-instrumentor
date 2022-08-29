package patch

import (
	"fmt"
	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common"
	v1 "k8s.io/api/core/v1"
)

const (
	NodeIPEnvName           = "NODE_IP"
	PodNameEnvVName         = "POD_NAME"
	PodNameEnvValue         = "$(POD_NAME)"
	LogzioMonitoringService = "logzio-monitoring-otel-collector.monitoring.svc.cluster.local"
)

type CollectorInfo struct {
	Hostname string
	Port     int
}

type Patcher interface {
	Patch(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication)
	IsInstrumented(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) bool
}

var patcherMap = map[common.ProgrammingLanguage]Patcher{
	common.JavaProgrammingLanguage:       java,
	common.PythonProgrammingLanguage:     python,
	common.DotNetProgrammingLanguage:     dotNet,
	common.JavascriptProgrammingLanguage: nodeJs,
	common.GoProgrammingLanguage:         golang,
}

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
