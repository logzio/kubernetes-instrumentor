FROM alpine:3
ARG DOTNET_OTEL_VERSION=v0.7.0
ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/$DOTNET_OTEL_VERSION/opentelemetry-dotnet-instrumentation-linux-musl.zip /tmp/
ADD init.sh /tmp
RUN chmod +x /tmp/init.sh
ENTRYPOINT [ "/tmp/init.sh"]