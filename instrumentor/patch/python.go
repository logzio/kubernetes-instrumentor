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
	"fmt"
	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/common/consts"
	v1 "k8s.io/api/core/v1"
)

const (
	pythonVolumeName        = "agentdir-python"
	pythonMountPath         = "/otel-auto-instrumentation"
	envOtelTracesExporter   = "OTEL_TRACES_EXPORTER"
	envOtelMetricsExporter  = "OTEL_METRICS_EXPORTER"
	envValOtelHttpExporter  = "otlp_proto_http"
	envLogCorrelation       = "OTEL_PYTHON_LOG_CORRELATION"
	pythonInitContainerName = "copy-python-agent"
)

var python = &pythonPatcher{}

type pythonPatcher struct{}

func (p *pythonPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) {
	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
		Name: pythonVolumeName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	})
	// add annotations
	podSpec.Annotations[LogzioLanguageAnnotation] = "python"
	podSpec.Annotations[RemoveInitContainerAnnotaion] = "true"
	podSpec.Annotations[annotationInstrumentedApp] = "true"
	// Add security context, run as privileged to allow the agent to copy files to the shared container volume
	runAsNonRoot := false
	root := int64(0)
	securityContext := &v1.SecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		RunAsUser:    &root,
		RunAsGroup:   &root,
	}
	// Add init container that copies the agent
	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
		Name:            pythonInitContainerName,
		Image:           pythonAgentName,
		Command:         []string{"cp", "-a", "/autoinstrumentation/.", "/otel-auto-instrumentation/"},
		SecurityContext: securityContext,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      pythonVolumeName,
				MountPath: pythonMountPath,
			},
		},
	})
	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.PythonProgrammingLanguage, container.Name) {
			container.Env = append([]v1.EnvVar{{
				Name: NodeIPEnvName,
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			},
				{
					Name: PodNameEnvVName,
					ValueFrom: &v1.EnvVarSource{
						FieldRef: &v1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				},
			}, container.Env...)

			container.Env = append(container.Env, v1.EnvVar{
				Name:  "PYTHONPATH",
				Value: "/otel-auto-instrumentation/opentelemetry/instrumentation/auto_instrumentation:/otel-auto-instrumentation",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
				Value: fmt.Sprintf("http://%s:%d", LogzioMonitoringService, consts.OTLPHttpPort),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  "OTEL_RESOURCE_ATTRIBUTES",
				Value: fmt.Sprintf("service.name=%s,k8s.pod.name=%s", calculateAppName(podSpec, &container, instrumentation), PodNameEnvValue),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  envOtelTracesExporter,
				Value: envValOtelHttpExporter,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  envOtelMetricsExporter,
				Value: "",
			})

			container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				MountPath: pythonMountPath,
				Name:      pythonVolumeName,
			})
		}
		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (p *pythonPatcher) UnPatch(podSpec *v1.PodTemplateSpec) {
	// remove annotations
	delete(podSpec.Annotations, LogzioLanguageAnnotation)
	delete(podSpec.Annotations, annotationInstrumentedApp)
	// remove the python volume
	var newVolumes []v1.Volume
	for _, volume := range podSpec.Spec.Volumes {
		if volume.Name != pythonVolumeName {
			newVolumes = append(newVolumes, volume)
		}
	}
	podSpec.Spec.Volumes = newVolumes

	// remove the python init container
	var newInitContainers []v1.Container
	for _, container := range podSpec.Spec.InitContainers {
		if container.Name != pythonInitContainerName {
			newInitContainers = append(newInitContainers, container)
		}
	}
	podSpec.Spec.InitContainers = newInitContainers

	// remove the environment variables from the containers
	for i, container := range podSpec.Spec.Containers {
		var newVolumeMounts []v1.VolumeMount
		for _, volumeMount := range container.VolumeMounts {
			if volumeMount.Name != pythonVolumeName {
				newVolumeMounts = append(newVolumeMounts, volumeMount)
			}
		}
		container.VolumeMounts = newVolumeMounts
		var newEnv []v1.EnvVar
		for _, env := range container.Env {
			if env.Name != NodeIPEnvName && env.Name != PodNameEnvVName && env.Name != envLogCorrelation && env.Name != "PYTHONPATH" && env.Name != "OTEL_EXPORTER_OTLP_ENDPOINT" && env.Name != "OTEL_RESOURCE_ATTRIBUTES" && env.Name != envOtelTracesExporter && env.Name != envOtelMetricsExporter {
				newEnv = append(newEnv, env)
			}
		}
		podSpec.Spec.Containers[i].Env = newEnv
	}
}

func (p *pythonPatcher) IsInstrumented(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) bool {
	// check if the pod is already instrumented
	for key, value := range podSpec.Annotations {
		if key == annotationInstrumentedApp && value == "true" {
			return true
		}
	}
	return false
}
