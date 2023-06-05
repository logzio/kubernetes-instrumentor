package inspectors

import (
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/detectors/process"
	"regexp"
)

type ApplicationInspector struct{}

var Application = &ApplicationInspector{}

/*
Returns an Application name if that name exists in either exe or command line
*/
func (appInspector *ApplicationInspector) Inspect(process *process.Details) (string, bool) {

	detectedApps := make(map[string]bool)

	for _, applicationType := range common.Applications {
		match, _ := regexp.MatchString("\\b"+string(applicationType)+"\\b", process.ExeName)

		if match {
			detectedApps[string(applicationType)] = true
		}
	}

	for _, applicationType := range common.Applications {
		match, _ := regexp.MatchString("\\b"+string(applicationType)+"\\b", process.CmdLine)

		if match {
			detectedApps[string(applicationType)] = true
		}
	}

	return findAppMatch(detectedApps)
}

func findAppMatch(apps map[string]bool) (string, bool) {
	res := ""
	for key := range apps {
		if value, exists := common.ProcessNameToType[key]; exists {
			res = value
			return res, true
		}

	}

	return res, false
}
