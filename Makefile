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
