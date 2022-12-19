package appDetector

import (
	"github.com/logzio/kubernetes-instrumentor/detectors/appDetector/inspectors"
	"github.com/logzio/kubernetes-instrumentor/detectors/process"
)

type inspector interface {
	Inspect(process *process.Details) (string, bool)
}

var inspectorsList = []inspector{inspectors.Application}

func DetectApplication(processes []process.Details) []string {
	var result []string
	for _, p := range processes {
		for _, i := range inspectorsList {
			inspectionResult, detected := i.Inspect(&p)
			if detected {
				result = append(result, inspectionResult)
				break
			}
		}
	}

	return result
}
