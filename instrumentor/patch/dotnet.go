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
	"github.com/logzio/kubernetes-instrumentor/common/consts"
	"strings"

	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common"
	v1 "k8s.io/api/core/v1"
)

const (
	enableProfilingEnvVar = "COR_ENABLE_PROFILING"
	profilerEndVar        = "COR_PROFILER"
	profilerId            = "{918728DD-259F-4A6A-AC2B-B85E1B658318}"
	profilerPathEnv       = "COR_PROFILER_PATH"
	profilerPath          = "/agent/linux-musl-x64/OpenTelemetry.AutoInstrumentation.ClrProfiler.Native.so"
	serviceNameEnv        = "OTEL_SERVICE_NAME"
	collectorUrlEnv       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	tracerHomeEnv         = "OTEL_DOTNET_AUTO_HOME"
	exportTypeEnv         = "OTEL_TRACES_EXPORTER"
	exportType            = "otlp"
	exportProtocolEnv     = "OTEL_EXPORTER_OTLP_PROTOCOL"
	exportProtocol        = "grpc"
	tracerHome            = "/agent"
	mountPath             = "/agent"
	dotnetVolumeName      = "agentdir-dotnet"
	startupHookEnv        = "DOTNET_STARTUP_HOOKS"
	startupHook           = "/agent/net/OpenTelemetry.AutoInstrumentation.StartupHook.dll"
	additonalDepsEnv      = "DOTNET_ADDITIONAL_DEPS"
	additonalDeps         = "/agent/AdditionalDeps"
	sharedStoreEnv        = "DOTNET_SHARED_STORE"
	sharedStore           = "/agent/store"
	resourceAttrEnv       = "OTEL_RESOURCE_ATTRIBUTES"
	resourceAttr          = "logz.io/language=dotnet"
	metricsExporterEnv    = "OTEL_METRICS_EXPORTER"
	logsExporterEnv       = "OTEL_LOGS_EXPORTER"
)

var dotNet = &dotNetPatcher{}

type dotNetPatcher struct{}

func (d *dotNetPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) {
	// Check if volume already exists
	volumeExists := false
	for _, vol := range podSpec.Spec.Volumes {
		if vol.Name == dotnetVolumeName {
			volumeExists = true
			break
		}
	}

	// If not, add volume
	if !volumeExists {
		podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
			Name: dotnetVolumeName,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})
	}

	// add annotations
	podSpec.Annotations[LogzioLanguageAnnotation] = "dotnet"
	podSpec.Annotations[tracesInstrumentedAnnotation] = "true"

	// Add security context, run as privileged to allow the agent to copy files to the shared container volume
	runAsNonRoot := false
	root := int64(0)
	securityContext := &v1.SecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		RunAsUser:    &root,
		RunAsGroup:   &root,
	}

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
				Name:  tracerHomeEnv,
				Value: tracerHome,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  collectorUrlEnv,
				Value: fmt.Sprintf("http://%s:%d", LogzioMonitoringService, consts.OTLPPort),
			})
			// calculate active service name
			activeServiceName := calculateServiceName(podSpec, &container, instrumentation)
			container.Env = append(container.Env, v1.EnvVar{
				Name:  serviceNameEnv,
				Value: activeServiceName,
			})
			// update the corresponding crd
			for i := range instrumentation.Spec.Languages {
				if instrumentation.Spec.Languages[i].ContainerName == container.Name {
					instrumentation.Spec.Languages[i].ActiveServiceName = activeServiceName
				}
			}

			container.Env = append(container.Env, v1.EnvVar{
				Name:  exportTypeEnv,
				Value: exportType,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  exportProtocolEnv,
				Value: exportProtocol,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  startupHookEnv,
				Value: startupHook,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  additonalDepsEnv,
				Value: additonalDeps,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  sharedStoreEnv,
				Value: sharedStore,
			})
			container.Env = append(container.Env, v1.EnvVar{
				Name:  resourceAttrEnv,
				Value: resourceAttr,
			})
			container.Env = append(container.Env, v1.EnvVar{
				Name:  metricsExporterEnv,
				Value: "none",
			})
			container.Env = append(container.Env, v1.EnvVar{
				Name:  logsExporterEnv,
				Value: "none",
			})

			// Check if volume mount already exists
			volumeMountExists := false
			for _, volumeMount := range container.VolumeMounts {
				if volumeMount.Name == dotnetVolumeName {
					volumeMountExists = true
					break
				}
			}
			// Check if init container already exists
			initContainerExists := false
			for _, initContainer := range podSpec.Spec.InitContainers {
				if initContainer.Name == dotnetInitContainerName {
					initContainerExists = true
					break
				}
			}

			// If not, add init container
			if !initContainerExists {
				podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
					Name:            dotnetInitContainerName,
					Image:           dotnetAgentName,
					Command:         []string{"/bin/sh", "-c", "/init.sh"},
					SecurityContext: securityContext,
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      dotnetVolumeName,
							MountPath: mountPath,
						},
					},
				})
			}

			// If not, add volume mount
			if !volumeMountExists {
				container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
					MountPath: mountPath,
					Name:      dotnetVolumeName,
				})
			}
		}
		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (d *dotNetPatcher) UnPatch(podSpec *v1.PodTemplateSpec) error {
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
		var newEnv []v1.EnvVar
		for _, env := range container.Env {
			if env.Name != resourceAttrEnv && env.Name != metricsExporterEnv && env.Name != logsExporterEnv && env.Name != NodeIPEnvName && env.Name != enableProfilingEnvVar && env.Name != profilerEndVar && env.Name != profilerPathEnv && env.Name != serviceNameEnv && env.Name != collectorUrlEnv && env.Name != tracerHomeEnv && env.Name != exportTypeEnv && env.Name != exportProtocolEnv && env.Name != startupHookEnv && env.Name != additonalDepsEnv && env.Name != sharedStoreEnv {
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
	return nil
}

func (d *dotNetPatcher) IsTracesInstrumented(podSpec *v1.PodTemplateSpec) bool {
	// check if the pod is already instrumented
	for key, value := range podSpec.Annotations {
		if key == tracesInstrumentedAnnotation && strings.ToLower(value) == "true" {
			return true
		}
	}
	return false
}

func (d *dotNetPatcher) UpdateServiceNameEnv(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) {
	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		// calculate active service name
		serviceName := calculateServiceName(podSpec, &container, instrumentation)
		if shouldUpdateServiceName(instrumentation, common.DotNetProgrammingLanguage, container.Name, serviceName) {
			// remove old env
			var newEnv []v1.EnvVar
			for _, env := range container.Env {
				if env.Name != serviceNameEnv {
					newEnv = append(newEnv, env)
				}
			}
			newEnv = append(newEnv, v1.EnvVar{
				Name:  serviceNameEnv,
				Value: serviceName,
			})
			container.Env = newEnv
			// update the corresponding crd
			for j := range instrumentation.Spec.Languages {
				if instrumentation.Spec.Languages[j].ContainerName == container.Name {
					instrumentation.Spec.Languages[j].ActiveServiceName = serviceName
				}
			}
			modifiedContainers = append(modifiedContainers, container)
		}
	}
	if len(modifiedContainers) > 0 {
		podSpec.Spec.Containers = modifiedContainers
	}
}
