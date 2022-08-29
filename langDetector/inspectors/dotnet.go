package inspectors

import (
	"fmt"
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/langDetector/process"
	"io/ioutil"
	"strings"
)

type dotnetInspector struct{}

const (
	aspnet = "ASPNET"
	dotnet = "DOTNET"
)

var dotNet = &dotnetInspector{}

func (d *dotnetInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/environ", p.ProcessID))
	if err == nil {
		environ := string(data)
		if strings.Contains(environ, aspnet) || strings.Contains(environ, dotnet) {
			return common.DotNetProgrammingLanguage, true
		}
	}

	return "", false
}
