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
	v1 "k8s.io/api/core/v1"
)

const (
	enableProfilingEnvVar = "CORECLR_ENABLE_PROFILING"
	profilerEndVar        = "CORECLR_PROFILER"
	profilerId            = "{918728DD-259F-4A6A-AC2B-B85E1B658318}"
	profilerPathEnv       = "CORECLR_PROFILER_PATH"
	profilerPath          = "/agent/OpenTelemetry.AutoInstrumentation.ClrProfiler.Native.so"
	intergationEnv        = "OTEL_INTEGRATIONS"
	intergations          = "/agent/integrations.json"
	conventionsEnv        = "OTEL_CONVENTION"
	serviceNameEnv        = "OTEL_SERVICE"
	convetions            = "OpenTelemetry"
	collectorUrlEnv       = "OTEL_TRACE_AGENT_URL"
	tracerHomeEnv         = "OTEL_DOTNET_TRACER_HOME"
	exportTypeEnv         = "OTEL_EXPORTER"
	tracerHome            = "/agent"
	dotnetVolumeName      = "agentdir-dotnet"
)

var dotNet = &dotNetPatcher{}

type dotNetPatcher struct{}

func (d *dotNetPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) {
	//podAnnotations := podSpec.GetAnnotations()
	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
		Name: dotnetVolumeName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	})
	// add annotations
	podSpec.Annotations[LogzioLanguageAnnotation] = "dotnet"
	podSpec.Annotations[tracesInstrumentedAnnotation] = "true"
	// Add security context, run as privileged to allow the agent to copy files to the shared container volume
	runAsNonRoot := false
	root := int64(0)
	// Add security context
	securityContext := &v1.SecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		RunAsUser:    &root,
		RunAsGroup:   &root,
	}
	// Add init container that copies the agent
	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
		Name:            dotnetInitContainerName,
		Image:           dotnetAgentName,
		SecurityContext: securityContext,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      dotnetVolumeName,
				MountPath: tracerHome,
			},
		},
	})

	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.DotNetProgrammingLanguage, container.Name) {
			container.Env = append([]v1.EnvVar{{
				Name: NodeIPEnvName,
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			}}, container.Env...)

			container.Env = append(container.Env, v1.EnvVar{
				Name:  enableProfilingEnvVar,
				Value: "1",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  profilerEndVar,
				Value: profilerId,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  profilerPathEnv,
				Value: profilerPath,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  intergationEnv,
				Value: intergations,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  tracerHomeEnv,
				Value: tracerHome,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  conventionsEnv,
				Value: convetions,
			})

			// Currently .NET instrumentation only support zipkin format, we should move to OTLP when support is added
			container.Env = append(container.Env, v1.EnvVar{
				Name:  collectorUrlEnv,
				Value: fmt.Sprintf("http://%s:9411/api/v2/spans", LogzioMonitoringService),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  serviceNameEnv,
				Value: calculateAppName(podSpec, &container, instrumentation),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  exportTypeEnv,
				Value: "Zipkin",
			})

			container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				MountPath: tracerHome,
				Name:      dotnetVolumeName,
			})
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (d *dotNetPatcher) UnPatch(podSpec *v1.PodTemplateSpec) {
	// remove the empty directory volume
	var newVolumes []v1.Volume
	for _, volume := range podSpec.Spec.Volumes {
		if volume.Name != dotnetVolumeName {
			newVolumes = append(newVolumes, volume)
		}
	}
	podSpec.Spec.Volumes = newVolumes

	// remove the init container
	var newInitContainers []v1.Container
	for _, container := range podSpec.Spec.InitContainers {
		if container.Name != dotnetInitContainerName {
			newInitContainers = append(newInitContainers, container)
		}
	}
	podSpec.Spec.InitContainers = newInitContainers

	// remove the environment variables
	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		//var newVolumeMounts []v1.VolumeMount
		//for _, volumeMount := range container.VolumeMounts {
		//	if volumeMount.Name != dotnetVolumeName {
		//		newVolumeMounts = append(newVolumeMounts, volumeMount)
		//	}
		//}
		//container.VolumeMounts = newVolumeMounts
		var newEnv []v1.EnvVar
		for _, env := range container.Env {
			if env.Name != NodeIPEnvName && env.Name != enableProfilingEnvVar && env.Name != profilerEndVar && env.Name != profilerPathEnv && env.Name != intergationEnv && env.Name != conventionsEnv && env.Name != serviceNameEnv && env.Name != collectorUrlEnv && env.Name != tracerHomeEnv && env.Name != exportTypeEnv {
				newEnv = append(newEnv, env)
			}
		}
		container.Env = newEnv
		modifiedContainers = append(modifiedContainers, container)
	}
	podSpec.Spec.Containers = modifiedContainers

	// remove the annotations
	delete(podSpec.Annotations, LogzioLanguageAnnotation)
	delete(podSpec.Annotations, tracesInstrumentedAnnotation)
}

func (d *dotNetPatcher) IsTracesInstrumented(podSpec *v1.PodTemplateSpec) bool {
	// check if the pod is already instrumented
	for key, value := range podSpec.Annotations {
		if key == tracesInstrumentedAnnotation && value == "true" {
			return true
		}
	}
	return false
}
