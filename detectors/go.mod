module github.com/logzio/kubernetes-instrumentor/detectors

go 1.18

require (
	github.com/fntlnz/mountinfo v0.0.0-20171106231217-40cb42681fad
	github.com/logzio/kubernetes-instrumentor/common v0.0.0
)

replace github.com/logzio/kubernetes-instrumentor/common => ./../common
