package patch

import (
	"fmt"
	apiV1 "github.com/logzio/kubernetes-instrumentor/api/v1alpha1"
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/common/consts"
	v1 "k8s.io/api/core/v1"
)

const (
	pythonAgentName         = "logzio/otel-agent-python:v0.0.1-test"
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
	// add detected language annotation
	podSpec.Annotations[LogzioLanguageAnnotation] = "python"
	podSpec.Spec.InitContainers = append(podSpec.Spec.InitContainers, v1.Container{
		Name:    pythonInitContainerName,
		Image:   pythonAgentName,
		Command: []string{"cp", "-a", "/autoinstrumentation/.", "/otel-auto-instrumentation/"},
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
				Name:  envLogCorrelation,
				Value: "true",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  "PYTHONPATH",
				Value: "/otel-auto-instrumentation/opentelemetry/instrumentation/auto_instrumentation:/otel-auto-instrumentation",
			})

			container.Env = append(container.Env, v1.EnvVar{
				Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
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

func (p *pythonPatcher) Unpatch(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) {
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
		if shouldPatch(instrumentation, common.PythonProgrammingLanguage, container.Name) {
			var newEnv []v1.EnvVar
			for _, env := range container.Env {
				if env.Name != NodeIPEnvName && env.Name != PodNameEnvVName && env.Name != envLogCorrelation && env.Name != "PYTHONPATH" && env.Name != "OTEL_EXPORTER_OTLP_ENDPOINT" {
					newEnv = append(newEnv, env)
				}
			}
			podSpec.Spec.Containers[i].Env = newEnv
		}
	}
}

func (p *pythonPatcher) IsInstrumented(podSpec *v1.PodTemplateSpec, instrumentation *apiV1.InstrumentedApplication) bool {
	// TODO: Deep comparison
	for _, c := range podSpec.Spec.InitContainers {
		if c.Name == pythonInitContainerName {
			return true
		}
	}
	return false
}
