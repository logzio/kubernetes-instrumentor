.PHONY: build-images
build-images:
	docker build -t logzio/instrumentation-detector:$(TAG)  -f detectors/Dockerfile . --build-arg SERVICE_NAME=detectors
	docker build -t logzio/instrumentor:$(TAG) . --build-arg SERVICE_NAME=instrumentor

.PHONY: push-images
push-images:
	docker push logzio/instrumentation-detector:$(TAG)
	docker push logzio/instrumentor:$(TAG)
