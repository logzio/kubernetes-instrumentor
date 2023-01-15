package common

type ApplicationByContainer struct {
	ContainerName string      `json:"containerName"`
	Application   Application `json:"application"`
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
