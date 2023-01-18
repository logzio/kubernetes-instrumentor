package patch

import (
	"fmt"
	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/common/consts"
	v1 "k8s.io/api/core/v1"
	"strings"
)

const (
	nodeAgentImage       = "logzio/otel-agent-nodejs:v0.0.1-test"
	nodeVolumeName       = "agentdir-nodejs"
	nodeMountPath        = "/agent-nodejs"
	nodeEnvNodeDebug     = "OTEL_NODEJS_DEBUG"
	nodeEnvTraceExporter = "OTEL_TRACES_EXPORTER"
	nodeEnvEndpoint      = "OTEL_EXPORTER_OTLP_ENDPOINT"
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

	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
		Name:    "copy-nodejs-agent",
		Image:   nodeAgentImage,
		Command: []string{"cp", "-a", "/autoinstrumentation/.", fmt.Sprintf("/%s/", nodeMountPath)},
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
				Name:  nodeEnvEndpoint,
				Value: fmt.Sprintf("%s:%d", LogzioMonitoringService, consts.OTLPPort),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvServiceName,
				Value: calculateAppName(podSpec, &container, instrumentation),
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  nodeEnvNodeOptions,
				Value: fmt.Sprintf("--require /%s/autoinstrumentation.js", nodeMountPath),
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

	// remove the init container that copies the agent
	var newInitContainers []v1.Container
	for _, container := range podSpec.Spec.InitContainers {
		if container.Name != "copy-nodejs-agent" {
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
			if envVar.Name != NodeIPEnvName && envVar.Name != nodeEnvNodeDebug && envVar.Name != nodeEnvTraceExporter && envVar.Name != nodeEnvEndpoint && envVar.Name != nodeEnvServiceName {
				if envVar.Name == nodeEnvNodeOptions {
					envVar.Value = strings.Replace(envVar.Value, fmt.Sprintf("--require /%s/autoinstrumentation.js", nodeMountPath), "", -1)
				}
				newEnv = append(newEnv, envVar)
			}
		}
		podSpec.Spec.Containers[i].Env = newEnv
	}
}

func (n *nodeJsPatcher) IsInstrumented(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) bool {
	// TODO: Deep comparison
	for _, c := range podSpec.Spec.InitContainers {
		if c.Name == "copy-nodejs-agent" {
			return true
		}
	}
	return false
}
