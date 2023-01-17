#!/bin/sh
mkdir /tmp/otel
cd /tmp/otel
tar xvf /tmp/opentelemetry-dotnet-instrumentation-linux-musl.zip
mv /tmp/otel/* /agent
chmod -R 777 /agent