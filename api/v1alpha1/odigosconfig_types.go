package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type InstrumentationMode string

// TODO delete if not needed
//const (
//	OptInInstrumentationMode  InstrumentationMode = "OPT_IN"
//	OptOutInstrumentationMode InstrumentationMode = "OPT_OUT"
//)

// LogzioConfigurationSpec defines the desired state of LogzioConfiguration
type LogzioConfigurationSpec struct {
	InstrumentationMode InstrumentationMode `json:"instrumentationMode"`
}

// LogzioConfiguration is the Schema for the logzio configuration
type LogzioConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LogzioConfigurationSpec `json:"spec,omitempty"`
}

// LogzioConfigurationList contains a list of LogzioConfiguration
type LogzioConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LogzioConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LogzioConfiguration{}, &LogzioConfigurationList{})
}
