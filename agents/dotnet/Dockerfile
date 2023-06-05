FROM alpine:3
ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/v0.5.0/opentelemetry-dotnet-instrumentation-linux-musl.zip /tmp/
ADD init.sh /tmp
RUN chmod +x /tmp/init.sh
ENTRYPOINT [ "/tmp/init.sh"]