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

package common

type ApplicationByContainer struct {
	ContainerName string      `json:"containerName"`
	Application   Application `json:"application"`
	LogType       string      `json:"logType"`
}

type Application string

var Applications = []Application{
	"kafka-server",
	"mysql",
	"nginx",
}

var ProcessNameToType = map[string]string{
	"kafka-server": "kafka_server",
	"nginx":        "nginx",
	"mysql":        "mysql",
}
