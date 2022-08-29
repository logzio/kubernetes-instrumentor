package inspectors

import (
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/langDetector/process"
	"strings"
)

type javaInspector struct{}

var java = &javaInspector{}

const processName = "java"

func (j *javaInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(p.ExeName, processName) || strings.Contains(p.CmdLine, processName) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}
