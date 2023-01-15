package process

import (
	"fmt"
	"github.com/fntlnz/mountinfo"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

type Details struct {
	ProcessID int
	ExeName   string
	CmdLine   string
}

func FindAllInContainer(podUID string, containerName string) ([]Details, []Details, error) {
	proc, err := os.Open("/proc")
	if err != nil {
		return nil, nil, err
	}

	var detectedContainers []Details
	var detectedApps []Details
	for {
		dirs, err := proc.Readdir(15)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
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
				return nil, nil, err
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

					cmdLine, err := ioutil.ReadFile(path.Join("/proc", dname, "cmdline"))
					var cmd string
					if err != nil {
						log.Println("Error reading cmdline")
						cmd = ""
					} else {
						cmd = string(cmdLine)
					}

					detectedContainers = append(detectedContainers, Details{
						ProcessID: pid,
						ExeName:   exeName,
						CmdLine:   cmd,
					})

					detectedApps = append(detectedApps, Details{
						ProcessID: pid,
						ExeName:   exeName,
						CmdLine:   cmd,
					})
				}
			}
		}
	}

	log.Println("No processes found")
	return detectedContainers, detectedApps, nil
}
