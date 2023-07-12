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
	javaVolumeName               = "agentdir-java"
	javaMountPath                = "/agent"
	otelResourceAttributesEnvVar = "OTEL_RESOURCE_ATTRIBUTES"
	otelResourceAttrPatteern     = "service.name=%s,k8s.pod.name=%s"
	javaOptsEnvVar               = "JAVA_OPTS"
	javaToolOptionsEnvVar        = "JAVA_TOOL_OPTIONS"
	javaToolOptionsPattern       = "-javaagent:/agent/opentelemetry-javaagent-all.jar " +
		"-Dotel.traces.sampler=always_on -Dotel.traces.exporter=otlp -Dotel.metrics.exporter=none -Dotel.exporter.otlp.traces.endpoint=http://%s:%d"
)

var java = &javaPatcher{}

type javaPatcher struct{}

func (j *javaPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) {
	// Check if volume already exists
	volumeExists := false
	for _, vol := range podSpec.Spec.Volumes {
		if vol.Name == javaVolumeName {
			volumeExists = true
			break
		}
	}

	// If not, add volume
	if !volumeExists {
		podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
			Name: javaVolumeName,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})
	}

	// add detected language annotation
	podSpec.Annotations[LogzioLanguageAnnotation] = "java"
	podSpec.Annotations[tracesInstrumentedAnnotation] = "true"
	// Add security context
	securityContext := &v1.SecurityContext{
		RunAsUser:    podSpec.Spec.SecurityContext.RunAsUser,
		RunAsGroup:   podSpec.Spec.SecurityContext.RunAsGroup,
		RunAsNonRoot: podSpec.Spec.SecurityContext.RunAsNonRoot,
	}

	// Check if init container already exists
	initContainerExists := false
	for _, initContainer := range podSpec.Spec.InitContainers {
		if initContainer.Name == javaInitContainerName {
			initContainerExists = true
			break
		}
	}

	// If not, add init container
	if !initContainerExists {
		podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
			Name:            javaInitContainerName,
			Image:           javaAgentImage,
			Command:         []string{"cp", "/javaagent.jar", "/agent/opentelemetry-javaagent-all.jar"},
			SecurityContext: securityContext,
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      javaVolumeName,
					MountPath: javaMountPath,
				},
			},
		})
	}

	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.JavaProgrammingLanguage, container.Name) {
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

			idx := getIndexOfEnv(container.Env, javaToolOptionsEnvVar)
			if idx == -1 {
				container.Env = append(container.Env, v1.EnvVar{
					Name:  javaToolOptionsEnvVar,
					Value: fmt.Sprintf(javaToolOptionsPattern, LogzioMonitoringService, consts.OTLPPort),
				})
			} else {
				container.Env[idx].Value = container.Env[idx].Value + " " + fmt.Sprintf(javaToolOptionsPattern, LogzioMonitoringService, consts.OTLPPort)
			}
			idx = getIndexOfEnv(container.Env, javaOptsEnvVar)
			if idx == -1 {
				container.Env = append(container.Env, v1.EnvVar{
					Name:  javaOptsEnvVar,
					Value: fmt.Sprintf(javaToolOptionsPattern, LogzioMonitoringService, consts.OTLPPort),
				})
			} else {
				container.Env[idx].Value = container.Env[idx].Value + " " + fmt.Sprintf(javaToolOptionsPattern, LogzioMonitoringService, consts.OTLPPort)
			}
			// calculate active service name
			activeServiceName := calculateActiveServiceName(podSpec, &container, instrumentation)
			container.Env = append(container.Env, v1.EnvVar{
				Name:  otelResourceAttributesEnvVar,
				Value: fmt.Sprintf(otelResourceAttrPatteern, activeServiceName, PodNameEnvValue),
			})
			// update the corresponding crd
			for _, service := range instrumentation.Spec.Languages {
				if service.ContainerName == container.Name {
					service.ActiveServiceName = activeServiceName
				}
			}

			// Check if volume mount already exists
			volumeMountExists := false
			for _, volumeMount := range container.VolumeMounts {
				if volumeMount.Name == javaVolumeName {
					volumeMountExists = true
					break
				}
			}

			// If not, add volume mount
			if !volumeMountExists {
				container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
					MountPath: javaMountPath,
					Name:      javaVolumeName,
				})
			}
		}
		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (j *javaPatcher) UnPatch(podSpec *v1.PodTemplateSpec) error {
	// remove the language annotations
	delete(podSpec.Annotations, LogzioLanguageAnnotation)
	delete(podSpec.Annotations, tracesInstrumentedAnnotation)
	// remove the init container
	var newInitContainers []v1.Container
	for _, container := range podSpec.Spec.InitContainers {
		if container.Name != javaInitContainerName {
			newInitContainers = append(newInitContainers, container)
		}
	}
	podSpec.Spec.InitContainers = newInitContainers

	// remove the environment variables
	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		var newEnv []v1.EnvVar
		for _, env := range container.Env {
			if env.Name != NodeIPEnvName && env.Name != PodNameEnvVName && env.Name != otelResourceAttributesEnvVar {
				if env.Name == javaOptsEnvVar {
					env.Value = strings.Replace(env.Value, fmt.Sprintf(javaToolOptionsPattern, LogzioMonitoringService, consts.OTLPPort), "", -1)
					// if the value is empty, don't add it
					if env.Value == "" {
						continue
					}
				}
				if env.Name == javaToolOptionsEnvVar {
					env.Value = strings.Replace(env.Value, fmt.Sprintf(javaToolOptionsPattern, LogzioMonitoringService, consts.OTLPPort), "", -1)
					// if the value is empty, don't add it
					if env.Value == "" {
						continue
					}
				}
				newEnv = append(newEnv, env)
			}
		}
		container.Env = newEnv
		modifiedContainers = append(modifiedContainers, container)
	}
	podSpec.Spec.Containers = modifiedContainers
	return nil
}

func (j *javaPatcher) IsTracesInstrumented(podSpec *v1.PodTemplateSpec) bool {
	// check if the pod is already traces instrumented
	for key, value := range podSpec.Annotations {
		if key == tracesInstrumentedAnnotation && strings.ToLower(value) == "true" {
			return true
		}
	}
	return false
}
