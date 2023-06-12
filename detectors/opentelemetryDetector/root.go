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

package opentelemetryDetector

import (
	"github.com/logzio/kubernetes-instrumentor/detectors/opentelemetryDetector/inspectors"
	"github.com/logzio/kubernetes-instrumentor/detectors/process"
)

type inspector interface {
	Inspect(process *process.Details) bool
}

var inspectorsList = []inspector{inspectors.OpenTelemetry}

func DetectApplication(processes []process.Details) bool {
	otelDetected := false
	for _, p := range processes {
		for _, i := range inspectorsList {
			detected := i.Inspect(&p)
			if detected {
				otelDetected = true
				break
			}
		}
	}
	return otelDetected
}
