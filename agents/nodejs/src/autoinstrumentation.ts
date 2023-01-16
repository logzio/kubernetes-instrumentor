// @ts-ignore
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
// @ts-ignore
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';

// @ts-ignore
import { NodeSDK } from '@opentelemetry/sdk-node';

const sdk = new NodeSDK({
    autoDetectResources: true,
    instrumentations: [getNodeAutoInstrumentations()],
    traceExporter: new OTLPTraceExporter(),
});

sdk.start();