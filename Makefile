.PHONY: build-images
build-images:
	docker build -t logzio/lang-detector:$(TAG)  -f langDetector/Dockerfile . --build-arg SERVICE_NAME=langDetector
	docker build -t logzio/instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor

.PHONY: push-images
push-images:
	docker push logzio/lang-detector:$(TAG)
	docker push logzio/instrumentor:$(TAG)
