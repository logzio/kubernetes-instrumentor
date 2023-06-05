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

package v1alpha1

import (
	"github.com/logzio/kubernetes-instrumentor/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstrumentedApplicationSpec defines the desired state of InstrumentedApplication
type InstrumentedApplicationSpec struct {
	Languages                []common.LanguageByContainer    `json:"languages,omitempty"`
	Applications             []common.ApplicationByContainer `json:"applications,omitempty"`
	Enabled                  *bool                           `json:"enabled,omitempty"`
	LogType                  string                          `json:"logType"`
	WaitingForDataCollection bool                            `json:"waitingForDataCollection"`
}

// InstrumentedApplicationStatus defines the observed state of InstrumentedApplication
type InstrumentedApplicationStatus struct {
	InstrumentationDetection InstrumentationStatus `json:"instrumentationDetection,omitempty"`
	TracesInstrumented       bool                  `json:"tracesInstrumented"`
	MetricsInstrumented      bool                  `json:"metricsInstrumented"`
	AppDetected              bool                  `json:"appDetected"`
}

type InstrumentationStatus struct {
	Phase InstrumentationPhase `json:"phase,omitempty"`
}

type InstrumentationPhase string

const (
	PendingInstrumentationDetectionPhase   InstrumentationPhase = "Pending"
	RunningInstrumentationDetectionPhase   InstrumentationPhase = "Running"
	CompletedInstrumentationDetectionPhase InstrumentationPhase = "Completed"
	ErrorInstrumentationDetectionPhase     InstrumentationPhase = "Error"
)

// InstrumentedApplication is the Schema for the instrumented applications API
type InstrumentedApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumentedApplicationSpec   `json:"spec,omitempty"`
	Status InstrumentedApplicationStatus `json:"status,omitempty"`
}

// InstrumentedApplicationList contains a list of InstrumentedApplication
type InstrumentedApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentedApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstrumentedApplication{}, &InstrumentedApplicationList{})
}
