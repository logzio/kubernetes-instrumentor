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
	"strings"

	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/common/consts"
	v1 "k8s.io/api/core/v1"
)

const (
	pythonVolumeName                   = "agentdir-python"
	pythonMountPath                    = "/otel-auto-instrumentation"
	envOtelTracesExporter              = "OTEL_TRACES_EXPORTER"
	envOtelMetricsExporter             = "OTEL_METRICS_EXPORTER"
	envValOtelOtlpExporter             = "otlp"
	envLogCorrelation                  = "OTEL_PYTHON_LOG_CORRELATION"
	envOtelExporterOTLPTracesProtocol  = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"
	envOtelExporterOTLPMetricsProtocol = "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL"
	httpProtoProtocol                  = "http/protobuf"
)

var python = &pythonPatcher{}

type pythonPatcher struct{}

func (p *pythonPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) {
	// Check if volume already exists
	volumeExists := false
	for _, vol := range podSpec.Spec.Volumes {
		if vol.Name == pythonVolumeName {
			volumeExists = true
			break
		}
	}

	// If not, add volume
	if !volumeExists {
		podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
			Name: pythonVolumeName,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})
	}

	// add annotations
	podSpec.Annotations[LogzioLanguageAnnotation] = "python"
	podSpec.Annotations[tracesInstrumentedAnnotation] = "true"
	// Add security context, run as privileged to allow the agent to copy files to the shared container volume
	runAsNonRoot := false
	root := int64(0)
	securityContext := &v1.SecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		RunAsUser:    &root,
		RunAsGroup:   &root,
	}

	// Check if init container already exists
	initContainerExists := false
	for _, initContainer := range podSpec.Spec.InitContainers {
		if initContainer.Name == pythonInitContainerName {
			initContainerExists = true
			break
		}
	}

	// If not, add init container
	if !initContainerExists {
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
	}

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
				Name:  envOtelExporterOTLPTracesProtocol,
				Value: httpProtoProtocol,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  envOtelExporterOTLPMetricsProtocol,
				Value: httpProtoProtocol,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
				Value: fmt.Sprintf("http://%s:%d", LogzioMonitoringService, consts.OTLPHttpPort),
			})
			// calculate active service name
			activeServiceName := calculateServiceName(podSpec, &container, instrumentation)
			container.Env = append(container.Env, v1.EnvVar{
				Name:  "OTEL_RESOURCE_ATTRIBUTES",
				Value: fmt.Sprintf("service.name=%s,k8s.pod.name=%s", activeServiceName, PodNameEnvValue),
			})
			// update the corresponding crd
			for i := range instrumentation.Spec.Languages {
				if instrumentation.Spec.Languages[i].ContainerName == container.Name {
					instrumentation.Spec.Languages[i].ActiveServiceName = activeServiceName
				}
			}

			container.Env = append(container.Env, v1.EnvVar{
				Name:  envOtelTracesExporter,
				Value: envValOtelOtlpExporter,
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  envOtelMetricsExporter,
				Value: envValOtelOtlpExporter,
			})

			// Check if volume mount already exists
			volumeMountExists := false
			for _, volumeMount := range container.VolumeMounts {
				if volumeMount.Name == pythonVolumeName {
					volumeMountExists = true
					break
				}
			}

			// If not, add volume mount
			if !volumeMountExists {
				container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
					MountPath: pythonMountPath,
					Name:      pythonVolumeName,
				})
			}
		}
		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (p *pythonPatcher) UnPatch(podSpec *v1.PodTemplateSpec) error {
	// remove annotations
	delete(podSpec.Annotations, LogzioLanguageAnnotation)
	delete(podSpec.Annotations, tracesInstrumentedAnnotation)
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
		var newEnv []v1.EnvVar
		for _, env := range container.Env {
			if env.Name != NodeIPEnvName && env.Name != PodNameEnvVName && env.Name != envLogCorrelation && env.Name != "PYTHONPATH" && env.Name != "OTEL_EXPORTER_OTLP_ENDPOINT" && env.Name != "OTEL_RESOURCE_ATTRIBUTES" && env.Name != envOtelTracesExporter && env.Name != envOtelExporterOTLPTracesProtocol && env.Name != envOtelExporterOTLPMetricsProtocol && env.Name != envOtelMetricsExporter && env.Name != httpProtoProtocol {
				newEnv = append(newEnv, env)
			}
		}
		podSpec.Spec.Containers[i].Env = newEnv
	}
	return nil
}

func (p *pythonPatcher) IsTracesInstrumented(podSpec *v1.PodTemplateSpec) bool {
	for key, value := range podSpec.Annotations {
		if key == tracesInstrumentedAnnotation && strings.ToLower(value) == "true" {
			return true
		}
	}
	return false
}
func (p *pythonPatcher) UpdateServiceNameEnv(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) {
	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		serviceName := calculateServiceName(podSpec, &container, instrumentation)
		if shouldUpdateServiceName(instrumentation, common.PythonProgrammingLanguage, container.Name, serviceName) {
			// remove old env
			var newEnv []v1.EnvVar
			for _, env := range container.Env {
				if env.Name != "OTEL_RESOURCE_ATTRIBUTES" {
					newEnv = append(newEnv, env)
				}
			}
			// calculate active service name
			newEnv = append(newEnv, v1.EnvVar{
				Name:  "OTEL_RESOURCE_ATTRIBUTES",
				Value: fmt.Sprintf("service.name=%s,k8s.pod.name=%s", serviceName, PodNameEnvValue),
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
