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

package inspectors

import (
	"fmt"
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/detectors/process"
	"io/ioutil"
	"strings"
)

type dotnetInspector struct{}

const (
	aspnet = "ASPNET"
	dotnet = "DOTNET"
)

var DotNet = &dotnetInspector{}

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
