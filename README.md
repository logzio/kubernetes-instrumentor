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

