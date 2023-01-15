package common

type ApplicationByContainer struct {
	ContainerName string      `json:"containerName"`
	Application   Application `json:"application"`
	//ProcessName   string      `json:"processName,omitempty"`
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
