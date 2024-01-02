package serviceNameDetector

import (
	"github.com/logzio/kubernetes-instrumentor/detectors/process"
	"strings"
)

func DetectServiceName(processes []process.Details) string {
	for _, p := range processes {
		for key, value := range p.Env {
			if key == "OTEL_RESOURCE_ATTRIBUTES" {
				attributes := strings.Split(value, ",")
				for _, attr := range attributes {
					if strings.HasPrefix(attr, "service.name=") {
						return strings.TrimPrefix(attr, "service.name=")
					}
				}
			}
			if key == "OTEL_SERVICE_NAME" {
				return value
			}
		}
	}
	return ""
}
