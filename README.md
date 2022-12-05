# Logz.io kubernetes instrumentor
**This project is in Beta and subject to breaking changes**

language detector and auto instrumentation microservice for kubernetes.  Inspired by odigos.io

**Important note:** This microservice is designed to be deployed alongside `logzio-monitoring` [helm chart](https://github.com/logzio/logzio-helm/tree/master/charts/logzio-monitoring).

### Development
Build:
```
export TAG=<your-tag>>
make build-images
```


```shell
  helm install -n monitoring \
  --set metricsOrTraces.enabled=true \
  --set logzio-k8s-telemetry.secrets.ListenerHost="https://listener.logz.io:8053" \
  --set logzio-k8s-telemetry.secrets.p8s_logzio_name="yotiii" \
  --set logzio-k8s-telemetry.traces.enabled=true \
  --set logzio-k8s-telemetry.secrets.TracesToken="dNMWmEguAYhCRxQsPTKsMUkyUXdEMcNX" \
  --set logzio-k8s-telemetry.secrets.LogzioRegion="us" \
  --set logzio-k8s-telemetry.spm.enabled=true \
  --set logzio-k8s-telemetry.standaloneCollector.resources.limits.memory=256Mi \
  --set logzio-k8s-telemetry.standaloneCollector.resources.limits.cpu=128m \
  --set logzio-k8s-telemetry.secrets.env_id="yotams" \
  --set logzio-k8s-telemetry.secrets.SpmToken="QVLXSJfEVYnYqkVQaJaXAdlLXembBVYj" \
  logzio-monitoring logzio-helm/logzio-monitoring
```
