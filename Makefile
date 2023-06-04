TAG ?= 0.0.3


.PHONY: install-tools
install-tools:
	go install github.com/google/addlicense@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/client9/misspell/cmd/misspell@latest
	go install github.com/pavius/impi/cmd/impi@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.47.3

.PHONY: build-images
build-images:
	docker build -t logzio/instrumentation-detector:$(TAG)  -f detectors/Dockerfile . --build-arg SERVICE_NAME=detectors
	docker build -t logzio/instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor

.PHONY: push-images
push-images:
	docker push logzio/instrumentation-detector:$(TAG)
	docker push logzio/instrumentor:$(TAG)

.PHONY: build-images-agents
build-images-agents:
	docker build -t logzio/otel-agent-dotnet:$(TAG) -f agents/dotnet/Dockerfile agents/dotnet
	docker build -t logzio/otel-agent-java:$(TAG) -f agents/java/Dockerfile agents/java
	docker build -t logzio/otel-agent-nodejs:$(TAG) -f agents/nodejs/Dockerfile agents/nodejs
	docker build -t logzio/otel-agent-python:$(TAG) -f agents/python/Dockerfile agents/python

.PHONY: push-images-agents
push-images-agents:
	docker push logzio/otel-agent-dotnet:$(TAG)
	docker push logzio/otel-agent-java:$(TAG)
	docker push logzio/otel-agent-nodejs:$(TAG)
	docker push logzio/otel-agent-python:$(TAG)

.PHONY: build-push-all-latest
build-push-all-latest:
	TAG=latest make build-images
	TAG=latest make push-images
	TAG=latest make build-images-agents
	TAG=latest make push-images-agents

.PHONY: build-push-all-tag
build-push-all-tag:
	make build-images
	make push-images
	make build-images-agents
	make push-images-agents

.PHONY: kubectl-deploy
kubectl-deploy:
	kubectl apply -f deploy/kubernetes-manifests
	kubectl apply -f deploy/services-demo

.PHONY: kubectl-clean
kubectl-clean:
	kubectl delete -f deploy/kubernetes-manifests
	kubectl delete -f deploy/services-demo