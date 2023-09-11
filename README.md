# Logz.io kubernetes instrumentor
**This project is in Beta and subject to breaking changes**
**Important note:** This microservice is designed to be deployed alongside `logzio-monitoring` [helm chart](https://github.com/logzio/logzio-helm/tree/master/charts/logzio-monitoring).
language detector and auto instrumentation microservice for kubernetes.  Inspired by odigos.io

### Getting Started
`deploy/kubernetes-manifests` folder contains the kubernetes manifests for the microservice. You can go to the folder and run `kubectl apply -f .` to deploy the microservice.
The following will be deployed:
- instrumented applications custom resource definition
- `logzio-instrumentor` service
- `logzio-instrumentor` deployment
- service account for the deployment
- cluster role and cluster role binding for the service account used by the deployment
- leader election role and role binding for the service account used by the deployment

The `logzio-instrumetor` microservice can be deployed to your cluster to discover applications, inject opentelemetry instrumentation, add log types and more. You can control the discovery process with annotations.
- `logz.io/traces_instrument = true` - will instrument the application with opentelemetry
- `logz.io/traces_instrument = rollback` - will delete the opentelemetry instrumentation
- `logz.io/service-name = <string>` - will set active service name for your opentelemetry instrumentation
- `logz.io/application_type = <string>` - will set log type to send to logz.io (**dependent on logz.io fluentd helm chart**)
- `logz.io/skip = true` - will skip the application from instrumentation or app detection

### Configuration for `logzio-instrumentor` container
To configure the `logzio-instrumentor` container, you can use the following arguments and apply in the deployment manifest (`deploy/kubernetes-manifests/deployment.yaml`):
#### Arguments
- `instrumentation-detector-tag`: The container tag to use for language detection, with a default value of `latest`.
- `instrumentation-detector-image`: The container image to use for language detection, with a default value of `logzio/instrumentation-detector`.
- `delete-detection-pods`: A flag that enables automatic termination of detection pods, with a default value of `true`.
- `metrics-bind-address`: The address the metrics endpoint binds to, with a default value of `:8080`.
- `health-probe-bind-address`: The address the health probe endpoint binds to, with a default value of `:8081`.
- `leader-elect`: A flag that enables leader election for the controller manager, with a default value of false.
#### Environment variables
- `MONITORING_SERVICE_ENDPOINT`: The endpoint of the monitoring service (ex: `logzio-monitoring-otel-collector.monitoring.svc.cluster.local`).

### 
### Development
Build:
```
export TAG=<your-tag>>
make build-images
```
Deploy Images:
```
export TAG=<your-tag>>
make push-images
```

## Change log
* v1.0.7
    - Add opentelemetry dependency detection in dependency files for: `nodejs`, `python`, `dotnet`  
* v1.0.6
    - Use pointers for instapp
    - Minimize k8s client `Get()` calls to avoid mismatching objects while the dynamic update
    - Add metrics env vars to Python instrumentation (it breaks otherwise)
* v1.0.5
    - remove `JAVA_OPTS` `JAVA_TOOL_OPTIONS` `NODE_OPTIONS` if they are empty
    - fix crd client updates
    - added `ActiveServiceName` to custom resource definition
    - handle `ActiveServiceName` updates
* v1.0.4
    - fix log type condition
    - change calculate app name logic
* v1.0.3
    - Add support for opentelemetry detection
    - `nodejs`: check for existing `NODE_OPTIONS`
    - `python`: remove metrics exporter
* v1.0.2
    - Add support for setting service name using logz.io/service-name annotation
* v1.0.0 - Initial release
    - language detector and auto instrumentation microservice for kubernetes

