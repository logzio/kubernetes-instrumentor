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
	// if instrumented with easyConnect, we don't want to report it as an opentelemetry process.
	// this is because easyConnect is a tool that instruments the application with opentelemetry.
	// we want to report the application as an opentelemetry process only if it is not instrumented with easyConnect.
	if easyConnectInEnv(p.Env) {
		return false
	}
	if otelInEnv(p.Env) || otelInCmdLine(p.CmdLine) || otelInDeps(p.Dependencies) {
		return true
	}
	return false
}

func easyConnectInEnv(env map[string]string) bool {
	for key, value := range env {
		if key == "OTEL_RESOURCE_ATTRIBUTES" && strings.Contains(value, "easy.connect.version") {
			log.Printf("found easy connect env var:\nkey: %s\nvalue: %s\n", key, value)
			return true
		}
	}
	return false

}

func otelInEnv(env map[string]string) bool {
	for envKey := range env {
		if strings.Contains(envKey, otelStr) || strings.Contains(envKey, otlpStr) {
			log.Printf("found opentelemetry env var:\nkey: %s\nvalue: %s\n", envKey, env[envKey])
			return true
		}
	}
	return false

}

func otelInCmdLine(cmdLine string) bool {
	if strings.Contains(cmdLine, otelStr) || strings.Contains(cmdLine, otlpStr) {
		log.Printf("found opentelemetry refrence in cmdline: %s", cmdLine)
		return true
	}
	return false
}

func otelInDeps(deps map[string]string) bool {
	detected := false
	for dep := range deps {
		if strings.Contains(dep, opentelemetryStr) || strings.Contains(dep, heliosStr) {
			log.Printf("Found opentelemetry dependency: %s", dep)
			detected = true
		}
	}
	return detected
}
