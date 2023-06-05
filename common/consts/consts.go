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

package consts

import "errors"

const (
	InstrumentationDetectionContainerAnnotationKey = "logzio/instrumentation-detection-pod"
	CurrentNamespaceEnvVar                         = "CURRENT_NS"
	DefaultNamespace                               = "monitoring"
	OTLPPort                                       = 4317
	OTLPHttpPort                                   = 4318
	ApplicationTypeAnnotation                      = "logz.io/application_type"
	SkipAppDetectionAnnotation                     = "logz.io/skip_app_detection"
	SupportedResourceDeployment                    = "Deployment"
	SupportedResourceStatefulSet                   = "StatefulSet"
)

var (
	PodsNotFoundErr = errors.New("could not find a ready pod")
)

var IgnoredNamespaces = []string{"kube-system", "local-path-storage", "istio-system", "linkerd", "gatekeeper-system", DefaultNamespace}
