FROM alpine:3
ARG DOTNET_OTEL_VERSION=v1.2.0
ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/$DOTNET_OTEL_VERSION/opentelemetry-dotnet-instrumentation-linux-musl.zip /tmp/
RUN mkdir /tmp/otel
RUN unzip /tmp/opentelemetry-dotnet-instrumentation-linux-musl.zip -d /tmp/otel/
RUN chmod -R 777 /tmp/otel
COPY init.sh /init.sh
RUN chmod +x /init.sh

