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

package process

import (
	"encoding/json"
	"fmt"
	"github.com/fntlnz/mountinfo"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

type Details struct {
	ProcessID    int
	ExeName      string
	CmdLine      string
	Env          map[string]string
	Dependencies map[string]string
}

func findFiles(rootPath string, targetFiles []string) []string {
	var foundFiles []string

	files, err := os.ReadDir(rootPath)
	if err != nil {
		return foundFiles
	}

	for _, file := range files {
		fullPath := path.Join(rootPath, file.Name())
		if file.IsDir() {
			foundFiles = append(foundFiles, findFiles(fullPath, targetFiles)...)
		} else {
			for _, target := range targetFiles {
				if file.Name() == target {
					foundFiles = append(foundFiles, fullPath)
				}
			}
		}
	}
	return foundFiles
}

func extractDependencies(pid int) map[string]string {
	basepath := path.Join("/proc", strconv.Itoa(pid), "root")
	// List of target dependency files
	targetFiles := []string{"go.mod", "package.json", "requirements.txt"}
	// Find all matching files recursively
	matchingFiles := findFiles(basepath, targetFiles)
	log.Println("Found dependency files: ", matchingFiles)

	files := map[string]func(string) map[string]string{
		"package.json":     extractNodejsDeps,
		"requirements.txt": extractPythonDeps,
	}

	allDeps := make(map[string]string)
	for _, filepath := range matchingFiles {
		handler, ok := files[path.Base(filepath)]
		if ok {
			for k, v := range handler(filepath) {
				allDeps[k] = v
			}
		}
	}

	return allDeps
}

func extractNodejsDeps(filepath string) map[string]string {
	deps := make(map[string]string)
	data, err := os.ReadFile(filepath)
	if err != nil {
		return deps
	}

	var jsonData map[string]interface{}
	if err = json.Unmarshal(data, &jsonData); err != nil {
		return deps
	}

	if dependencies, ok := jsonData["dependencies"].(map[string]interface{}); ok {
		for pkg, ver := range dependencies {
			deps[pkg] = ver.(string)
		}
	}
	return deps
}

func extractPythonDeps(filepath string) map[string]string {
	deps := make(map[string]string)
	data, err := os.ReadFile(filepath)
	if err != nil {
		return deps
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "==")
		if len(parts) == 2 {
			deps[parts[0]] = parts[1]
		}
	}
	return deps
}

func FindAllInContainer(podUID string, containerName string) ([]Details, error) {
	proc, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}

	var detectedContainers []Details
	for {
		dirs, err := proc.Readdir(15)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for _, di := range dirs {
			if !di.IsDir() {
				continue
			}

			dname := di.Name() // && dname != "1" - to dir we look for
			if dname[0] < '0' || dname[0] > '9' {
				continue
			}

			pid, err := strconv.Atoi(dname)
			if err != nil {
				return nil, err
			}

			mi, err := mountinfo.GetMountInfo(path.Join("/proc", dname, "mountinfo"))
			if err != nil {
				log.Println("Error getting mount info", dname)
				continue
			}

			for _, m := range mi {
				root := m.Root
				if strings.Contains(root, fmt.Sprintf("%s/containers/%s", podUID, containerName)) {
					exeName, err := os.Readlink(path.Join("/proc", dname, "exe"))
					if err != nil {
						// Read link may fail if target process-app runs not as root
						log.Println("Error reading links")
						exeName = ""
					}

					cmdLine, err := os.ReadFile(path.Join("/proc", dname, "cmdline"))
					var cmd string
					if err != nil {
						log.Println("Error reading cmdline")
						cmd = ""
					} else {
						cmd = string(cmdLine)
					}

					// Read environment variables
					envFilePath := path.Join("/proc", strconv.Itoa(pid), "environ")
					envBytes, err := os.ReadFile(envFilePath)
					if err != nil {
						log.Println("Error reading env file", envFilePath)
						return nil, err
					}

					env := make(map[string]string)
					for _, line := range strings.Split(string(envBytes), "\x00") {
						if line == "" {
							continue
						}
						parts := strings.SplitN(line, "=", 2)
						if len(parts) != 2 {
							continue // Skip malformed entries
						}
						env[parts[0]] = parts[1]
					}
					// Add dependencies
					deps := extractDependencies(pid)
					detectedContainers = append(detectedContainers, Details{
						ProcessID:    pid,
						ExeName:      exeName,
						CmdLine:      cmd,
						Env:          env,
						Dependencies: deps,
					})
				}
			}
		}
	}
	log.Printf("Detected containers: %+v", detectedContainers)
	return detectedContainers, nil
}
