package consts

import "errors"

const (
	LangDetectionContainerAnnotationKey = "logzio/lang-detection-pod"
	CurrentNamespaceEnvVar              = "CURRENT_NS"
	DefaultNamespace                    = "monitoring"
	DefaultLogzioConfigurationName      = "logzio-config"
	OTLPPort                            = 4317
	OTLPHttpPort                        = 4318
)

var (
	PodsNotFoundErr = errors.New("could not find a ready pod")
)
