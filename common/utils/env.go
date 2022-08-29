package utils

import (
	"github.com/logzio/kubernetes-instrumentor/common/consts"
	"os"
)

func getEnvVarOrDefault(envKey string, defaultVal string) string {
	val, exists := os.LookupEnv(envKey)
	if exists {
		return val
	}

	return defaultVal
}

func GetCurrentNamespace() string {
	return getEnvVarOrDefault(consts.CurrentNamespaceEnvVar, consts.DefaultNamespace)
}
