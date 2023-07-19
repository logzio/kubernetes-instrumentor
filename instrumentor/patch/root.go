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

package patch

import (
	"context"
	"fmt"
	"os"

	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common"
	v1 "k8s.io/api/core/v1"
)

const (
	NodeIPEnvName                = "NODE_IP"
	PodNameEnvVName              = "POD_NAME"
	PodNameEnvValue              = "$(POD_NAME)"
	LogzioLanguageAnnotation     = "logz.io/instrumentation-language"
	LogzioServiceAnnotationName  = "logz.io/service-name"
	tracesInstrumentedAnnotation = "logz.io/traces-instrumented"
	pythonInitContainerName      = "copy-python-agent"
	nodeInitContainerName        = "copy-nodejs-agent"
	javaInitContainerName        = "copy-java-agent"
	dotnetInitContainerName      = "copy-dotnet-agent"
)

var (
	LogzioMonitoringService = os.Getenv("MONITORING_SERVICE_ENDPOINT")
	dotnetAgentName         = os.Getenv("DOTNET_AGENT_IMAGE")
	pythonAgentName         = os.Getenv("PYTHON_AGENT_IMAGE")
	nodeAgentImage          = os.Getenv("NODEJS_AGENT_IMAGE")
	javaAgentImage          = os.Getenv("JAVA_AGENT_IMAGE")
)

type Patcher interface {
	Patch(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication)
	UnPatch(podSpec *v1.PodTemplateSpec) error
	IsTracesInstrumented(podSpec *v1.PodTemplateSpec) bool
	UpdateServiceNameEnv(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication)
}

var patcherMap = map[common.ProgrammingLanguage]Patcher{
	common.JavaProgrammingLanguage:       java,
	common.PythonProgrammingLanguage:     python,
	common.DotNetProgrammingLanguage:     dotNet,
	common.JavascriptProgrammingLanguage: nodeJs,
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

func UpdateActiveServiceName(original *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) error {
	for _, l := range getLangsInResult(instrumentation) {
		p, exists := patcherMap[l]
		if !exists {
			return fmt.Errorf("unable to find patcher for lang %s", l)
		}

		p.UpdateServiceNameEnv(original, instrumentation)
	}

	return nil
}

func RollbackPatch(original *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) error {
	for _, l := range getLangsInResult(instrumentation) {
		p, exists := patcherMap[l]
		if !exists {
			return fmt.Errorf("unable to find patcher for lang %s", l)
		}
		p.UnPatch(original)
	}
	return nil
}

func IsTracesInstrumented(original *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) (bool, error) {
	instrumented := true
	for _, l := range getLangsInResult(instrumentation) {
		p, exists := patcherMap[l]
		if !exists {
			return false, fmt.Errorf("unable to find patcher for lang %s", l)
		}

		instrumented = instrumented && p.IsTracesInstrumented(original)
	}

	return instrumented, nil
}

func getLangsInResult(instrumentation *apiV1.InstrumentedApplication) []common.ProgrammingLanguage {
	langMap := make(map[common.ProgrammingLanguage]interface{})
	for _, c := range instrumentation.Spec.Languages {
		langMap[c.Language] = nil
	}

	var langs []common.ProgrammingLanguage
	for l := range langMap {
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

func shouldUpdateServiceName(instrumentation *apiV1.InstrumentedApplication, lang common.ProgrammingLanguage, containerName string, serviceName string) bool {
	for _, l := range instrumentation.Spec.Languages {
		if l.ContainerName == containerName && l.Language == lang {
			// the active service name is different from the calculated service name
			if serviceName != "" && l.ActiveServiceName != "" && l.ActiveServiceName != serviceName {
				return true
			}
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

func calculateServiceName(podSpec *v1.PodTemplateSpec, currentContainer *v1.Container, instrumentation *apiV1.InstrumentedApplication) string {
	if podSpec.Annotations[LogzioServiceAnnotationName] != "" {
		return podSpec.Annotations[LogzioServiceAnnotationName]
	}
	if len(podSpec.Spec.Containers) > 1 {
		return currentContainer.Name
	}
	return instrumentation.ObjectMeta.OwnerReferences[0].Name + "-" + currentContainer.Name
}

func getApplicationFromDetectionResult(instrumentedApplication *apiV1.InstrumentedApplication) string {
	var detectedApp = ""
	if len(instrumentedApplication.Spec.Applications) > 0 {
		detectedApp = string(instrumentedApplication.Spec.Applications[0].Application)
	}

	return detectedApp
}

func IsDetected(ctx context.Context, original *v1.PodTemplateSpec, instrumentedApp *apiV1.InstrumentedApplication) (bool, error) {
	isDetected := true
	app := getApplicationFromDetectionResult(instrumentedApp)
	if app != "" {
		p, exists := annotationPatcherMap[app]
		if !exists {
			return false, fmt.Errorf("unable to find patcher for %s", app)
		}

		isDetected = isDetected && p.shouldPatch(original.Annotations, original.Namespace)
	}

	return isDetected, nil
}

func init() {
	addAnnotationPatcher()
}

func addAnnotationPatcher() {
	for _, app := range common.ProcessNameToType {
		annotationPatcherMap[app] = *AnnotationPatcherInst
	}
}
