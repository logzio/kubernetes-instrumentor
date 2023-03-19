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
	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/common/consts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	nodeVolumeName       = "agentdir-nodejs"
	nodeMountPath        = "/agent-nodejs"
	nodeEnvNodeDebug     = "OTEL_NODEJS_DEBUG"
	nodeEnvTraceExporter = "OTEL_TRACES_EXPORTER"
	nodeEnvTraceProtocol = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"
	nodeEnvEndpoint      = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	nodeEnvServiceName   = "OTEL_SERVICE_NAME"
	nodeEnvNodeOptions   = "NODE_OPTIONS"
)

var nodeJs = &nodeJsPatcher{}

type nodeJsPatcher struct{}

func (n *nodeJsPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) {
	podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, v1.Volume{
		Name: nodeVolumeName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	})
	// add detected language annotation
	podSpec.Annotations[LogzioLanguageAnnotation] = "javascript"
	podSpec.Annotations[tracesInstrumentedAnnotation] = "true"

	// Add security context
	securityContext := &v1.SecurityContext{
		RunAsUser:    podSpec.Spec.SecurityContext.RunAsUser,
		RunAsGroup:   podSpec.Spec.SecurityContext.RunAsGroup,
		RunAsNonRoot: podSpec.Spec.SecurityContext.RunAsNonRoot,
	}
	// Add init container that copies the agent
	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
		Name:            nodeInitContainerName,
		Image:           nodeAgentImage,
		Command:         []string{"cp", "-a", "/autoinstrumentation/.", fmt.Sprintf("%s/", nodeMountPath)},
		SecurityContext: securityContext,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      nodeVolumeName,
				MountPath: nodeMountPath,
			},
		},
	})

	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.JavascriptProgrammingLanguage, container.Name) {
			container.Env = append([]v1.EnvVar{{
				Name: NodeIPEnvName,
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			}}, container.Env...)

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvNodeDebug,
				Value: "true",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvTraceExporter,
				Value: "otlp",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvTraceProtocol,
				Value: "grpc",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvEndpoint,
				Value: fmt.Sprintf("%s:%d", LogzioMonitoringService, consts.OTLPPort),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvServiceName,
				Value: calculateAppName(podSpec, &container, instrumentation),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvNodeOptions,
				Value: fmt.Sprintf("--require %s/autoinstrumentation.js", nodeMountPath),
			})

			container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				MountPath: nodeMountPath,
				Name:      nodeVolumeName,
			})
		}
		modifiedContainers = append(modifiedContainers, container)
	}
	podSpec.Spec.Containers = modifiedContainers
}

func (n *nodeJsPatcher) UnPatch(podSpec *v1.PodTemplateSpec) {
	// remove the detected language annotation
	delete(podSpec.Annotations, LogzioLanguageAnnotation)
	delete(podSpec.Annotations, tracesInstrumentedAnnotation)

	// remove the init container that copies the agent
	var newInitContainers []v1.Container
	for _, container := range podSpec.Spec.InitContainers {
		if container.Name != nodeInitContainerName {
			newInitContainers = append(newInitContainers, container)
		}
	}
	podSpec.Spec.InitContainers = newInitContainers

	// remove the volume for the agent
	var newVolumes []v1.Volume
	for _, volume := range podSpec.Spec.Volumes {
		if volume.Name != nodeVolumeName {
			newVolumes = append(newVolumes, volume)
		}
	}
	podSpec.Spec.Volumes = newVolumes

	// remove environment variables from containers
	for i, container := range podSpec.Spec.Containers {
		var newVolumeMounts []v1.VolumeMount
		for _, volumeMount := range container.VolumeMounts {
			if volumeMount.Name != nodeVolumeName {
				newVolumeMounts = append(newVolumeMounts, volumeMount)
			}
		}
		container.VolumeMounts = newVolumeMounts
		var newEnv []v1.EnvVar
		for _, envVar := range container.Env {
			if envVar.Name != NodeIPEnvName && envVar.Name != nodeEnvNodeDebug && envVar.Name != nodeEnvTraceExporter && envVar.Name != nodeEnvEndpoint && envVar.Name != nodeEnvTraceProtocol && envVar.Name != nodeEnvServiceName {
				if envVar.Name == nodeEnvNodeOptions {
					envVar.Value = strings.Replace(envVar.Value, fmt.Sprintf("--require %s/autoinstrumentation.js", nodeMountPath), "", -1)
				}
				newEnv = append(newEnv, envVar)
			}
		}
		podSpec.Spec.Containers[i].Env = newEnv
	}
}
func (n *nodeJsPatcher) RemoveInitContainer(podSpec *v1.PodTemplateSpec) {
	var newInitContainers []v1.Container
	for _, container := range podSpec.Spec.InitContainers {
		if container.Name != nodeInitContainerName {
			newInitContainers = append(newInitContainers, container)
		}
	}
	podSpec.Spec.InitContainers = newInitContainers
}
func (n *nodeJsPatcher) ShouldRemoveInitContainer(podSpec *v1.PodTemplateSpec, ctx context.Context, object client.Object) bool {
	removeAnnotation := false
	initContainerTerminated := false
	for key, value := range podSpec.Annotations {
		if key == RemoveInitContainerAnnotaion && value == "true" {
			removeAnnotation = true
		}
	}
	//check for the state of the init container
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	if object.GetNamespace() != "" && object.GetName() != "" {
		// get the pod
		pods, err := k8sClient.CoreV1().Pods(object.GetNamespace()).List(ctx, metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, object.GetName()) {
				// check the state of the init container
				initContainers := pod.Spec.InitContainers
				for _, container := range initContainers {
					if container.Name == nodeInitContainerName {
						for _, containerStatus := range pod.Status.InitContainerStatuses {
							if containerStatus.Name == container.Name {
								if containerStatus.State.Terminated != nil {
									initContainerTerminated = true
								}
							}
						}
					}
				}
			}
		}
	}
	return removeAnnotation && initContainerTerminated
}

func (n *nodeJsPatcher) IsTracesInstrumented(podSpec *v1.PodTemplateSpec) bool {
	// check if the pod is already instrumented
	for key, value := range podSpec.Annotations {
		if key == tracesInstrumentedAnnotation && value == "true" {
			return true
		}
	}
	return false
}

func (n *nodeJsPatcher) IsMetricsInstrumented(podSpec *v1.PodTemplateSpec) bool {
	// check if the pod is already instrumented
	for key, value := range podSpec.Annotations {
		if key == metricsInstrumentedAnnotation && value == "true" {
			return true
		}
	}
	return false
}
