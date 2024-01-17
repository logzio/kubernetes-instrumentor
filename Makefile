TAG ?= v1.0.10
AWS_ECR_REGISTRY=public.ecr.aws/logzio

.PHONY: build-push-images-multiarch-ecr
build-push-images-multiarch-ecr:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(AWS_ECR_REGISTRY)/instrumentation-detector:$(TAG) -f detectors/Dockerfile . --build-arg SERVICE_NAME=detectors --push
	docker buildx build --platform linux/amd64,linux/arm64 -t $(AWS_ECR_REGISTRY)/instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor --push


.PHONY: build-push-images-agents-multiarch-ecr
build-push-images-agents-multiarch-ecr:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(AWS_ECR_REGISTRY)/otel-agent-dotnet:$(TAG) -f agents/dotnet/Dockerfile agents/dotnet --push
	docker buildx build --platform linux/amd64,linux/arm64 -t $(AWS_ECR_REGISTRY)/otel-agent-java:$(TAG) -f agents/java/Dockerfile agents/java --push
	docker buildx build --platform linux/amd64,linux/arm64 -t $(AWS_ECR_REGISTRY)/otel-agent-nodejs:$(TAG) -f agents/nodejs/Dockerfile agents/nodejs --push
	docker buildx build --platform linux/amd64,linux/arm64 -t $(AWS_ECR_REGISTRY)/otel-agent-python:$(TAG) -f agents/python/Dockerfile agents/python --push


.PHONY: build-push-images-multiarch
build-push-images-multiarch:
	docker buildx build --platform linux/amd64,linux/arm64 -t logzio/instrumentation-detector:$(TAG) -f detectors/Dockerfile . --build-arg SERVICE_NAME=detectors --push
	docker buildx build --platform linux/amd64,linux/arm64 -t logzio/instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor --push

.PHONY: build-push-images-agents-multiarch
build-push-images-agents-multiarch:
	docker buildx build --platform linux/amd64,linux/arm64 -t logzio/otel-agent-dotnet:$(TAG) -f agents/dotnet/Dockerfile agents/dotnet --push
	docker buildx build --platform linux/amd64,linux/arm64 -t logzio/otel-agent-java:$(TAG) -f agents/java/Dockerfile agents/java --push
	docker buildx build --platform linux/amd64,linux/arm64 -t logzio/otel-agent-nodejs:$(TAG) -f agents/nodejs/Dockerfile agents/nodejs --push
	docker buildx build --platform linux/amd64,linux/arm64 -t logzio/otel-agent-python:$(TAG) -f agents/python/Dockerfile agents/python --push

.PHONY: build-push-images-amd
build-push-images-amd:
	docker build -t logzio/instrumentation-detector:$(TAG) -f detectors/Dockerfile . --build-arg SERVICE_NAME=detectors
	docker build -t logzio/instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor
	docker push logzio/instrumentation-detector:$(TAG)
	docker push logzio/instrumentor:$(TAG)

.PHONY: build-push-images-agents-amd
build-push-images-agents-amd:
	docker build -t logzio/otel-agent-dotnet:$(TAG) -f agents/dotnet/Dockerfile agents/dotnet
	docker build -t logzio/otel-agent-java:$(TAG) -f agents/java/Dockerfile agents/java
	docker build -t logzio/otel-agent-nodejs:$(TAG) -f agents/nodejs/Dockerfile agents/nodejs
	docker build -t logzio/otel-agent-python:$(TAG) -f agents/python/Dockerfile agents/python
	docker push logzio/otel-agent-dotnet:$(TAG)
	docker push logzio/otel-agent-java:$(TAG)
	docker push logzio/otel-agent-nodejs:$(TAG)
	docker push logzio/otel-agent-python:$(TAG)

