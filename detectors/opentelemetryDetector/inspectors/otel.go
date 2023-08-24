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

package inspectors

import (
	"github.com/logzio/kubernetes-instrumentor/detectors/process"
	"log"
	"strings"
)

type openTelemetryInspector struct{}

var OpenTelemetry = &openTelemetryInspector{}

const (
	otelStr          = "OTEL"
	otlpStr          = "OTLP"
	opentelemetryStr = "opentelemetry"
	heliosStr        = "helios"
)

func (o *openTelemetryInspector) Inspect(p *process.Details) bool {
	if otelInEnv(p.Env) || otelInCmdLine(p.CmdLine) || otelInDeps(p.Dependencies) {
		return true
	}
	return false
}

func otelInEnv(env map[string]string) bool {
	for envKey := range env {
		if strings.Contains(envKey, otelStr) || strings.Contains(envKey, otlpStr) {
			return true
		}
	}
	return false

}

func otelInCmdLine(cmdLine string) bool {
	return strings.Contains(cmdLine, otelStr) || strings.Contains(cmdLine, otlpStr)
}

func otelInDeps(deps map[string]string) bool {
	for dep := range deps {
		if strings.Contains(dep, opentelemetryStr) || strings.Contains(dep, heliosStr) {
			log.Printf("Found at least one OpenTelemetry dependency: %s", dep)
			return true
		}
	}
	return false
}
