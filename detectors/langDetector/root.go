package langDetector

import (
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/detectors/langDetector/inspectors"
	"github.com/logzio/kubernetes-instrumentor/detectors/process"
)

type inspector interface {
	Inspect(process *process.Details) (common.ProgrammingLanguage, bool)
}

var inspectorsList = []inspector{inspectors.Java, inspectors.Python, inspectors.DotNet, inspectors.NodeJs}

// DetectLanguage returns a list of all the detected languages in the process-app list
// For go applications the process-app path is also returned, in all other languages the value is empty
func DetectLanguage(processes []process.Details) ([]common.ProgrammingLanguage, string) {
	var result []common.ProgrammingLanguage
	processName := ""
	for _, p := range processes {
		for _, i := range inspectorsList {
			inspectionResult, detected := i.Inspect(&p)
			if detected {
				result = append(result, inspectionResult)
				break
			}
		}
	}

	return result, processName
}
