package main

import (
	"encoding/json"
	"flag"
	"github.com/logzio/kubernetes-instrumentor/common"
	"github.com/logzio/kubernetes-instrumentor/detectors/appDetector"
	"github.com/logzio/kubernetes-instrumentor/detectors/langDetector"
	"github.com/logzio/kubernetes-instrumentor/detectors/process"
	"io/fs"
	"io/ioutil"
	"log"
	"strings"
)

type Args struct {
	PodUID         string
	ContainerNames []string
}

func main() {
	args := parseArgs()
	var containerResults []common.LanguageByContainer
	var detectedAppResults common.ApplicationByContainer
	for _, containerName := range args.ContainerNames {
		processes, detected_apps, err := process.FindAllInContainer(args.PodUID, containerName)
		if err != nil {
			log.Fatalf("could not find processes, error: %s\n", err)
		}

		processResults, processName := langDetector.DetectLanguage(processes)
		log.Printf("detection result: %s\n", processResults)

		detectedAppName := appDetector.DetectApplication(detected_apps)
		log.Printf("detection result: %s\n", detectedAppName)

		if len(processResults) > 0 {
			containerResults = append(containerResults, common.LanguageByContainer{
				ContainerName: containerName,
				Language:      processResults[0],
				ProcessName:   processName,
			})
		}

		// Only one detected app is relevant (the rest is duplicated)
		if len(detectedAppName) > 0 {
			detectedAppResults = common.ApplicationByContainer{
				ContainerName: containerName,
				Application:   common.Application(detectedAppName[0]),
			}
		}
	}

	detectionResult := common.DetectionResult{
		LanguageByContainer:    containerResults,
		ApplicationByContainer: detectedAppResults,
	}

	err := publishDetectionResult(detectionResult)
	if err != nil {
		log.Fatalf("could not publish detection result, error: %s\n", err)
	}
}

func parseArgs() *Args {
	result := Args{}
	var names string
	flag.StringVar(&result.PodUID, "pod-uid", "", "The UID of the target pod")
	flag.StringVar(&names, "container-names", "", "The container names in the target pod")
	flag.Parse()

	result.ContainerNames = strings.Split(names, ",")

	return &result
}

func publishDetectionResult(result common.DetectionResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return ioutil.WriteFile("/dev/detection-result", data, fs.ModePerm)
}
