package consts

import "errors"

const (
	InstrumentationDetectionContainerAnnotationKey = "logzio/instrumentation-detection-pod"
	CurrentNamespaceEnvVar                         = "CURRENT_NS"
	DefaultNamespace                               = "monitoring"
	DefaultLogzioConfigurationName                 = "logzio-config"
	OTLPPort                                       = 4317
	OTLPHttpPort                                   = 4318
	DefaultMonitoringNamespace                     = "monitoring"
	KubeSystemNamespace                            = "kube-system"
	GateKeeperSystemNamespace                      = "gatekeeper-system"
	LocalPathStorageNamespace                      = "local-path-storage"
)

var (
	PodsNotFoundErr = errors.New("could not find a ready pod")
)

var IgnoredNamespaces = []string{"kube-system", "local-path-storage", "istio-system", "linkerd", "gatekeeper-system", DefaultNamespace}
