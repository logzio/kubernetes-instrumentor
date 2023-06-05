#!/bin/sh
mkdir /tmp/otel
cd /tmp/otel
unzip /tmp/opentelemetry-dotnet-instrumentation-linux-musl.zip -d /tmp/otel/
mv /tmp/otel/ /agent
chmod -R 777 /agent
