import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { NodeSDK } from '@opentelemetry/sdk-node';
import { BasicTracerProvider, SimpleSpanProcessor} from '@opentelemetry/sdk-trace-base';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { Resource } from "@opentelemetry/resources";
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';

const process = require("process");


console.log("Instrumenting Node.js application");
console.log("Active config: ", process.env.OTEL_SERVICE_NAME, process.env.OTEL_TRACES_EXPORTER, process.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT, process.env.OTEL_EXPORTER_OTLP_TRACES_PROTOCOL);

const otlpEndpoint = "http://" + process.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT || 'http://localhost:4317';
const exporter = new OTLPTraceExporter({
    url: otlpEndpoint
});

const provider = new BasicTracerProvider({
    resource: new Resource({
        [SemanticResourceAttributes.SERVICE_NAME]:
        process.env.OTEL_SERVICE_NAME,
    }),
});

console.log("Adding OTLP exporter");
// export spans to opentelemetry collector
provider.addSpanProcessor(new SimpleSpanProcessor(exporter));
provider.register();

console.log("Registering Node.js auto-instrumentations");
const sdk = new NodeSDK({
    traceExporter: exporter,
    instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start()
console.log("Tracing initialized");

process.on("SIGTERM", () => {
    sdk
        .shutdown()
        .then(
            () => console.log("SDK shut down successfully"),
            (err) => console.log("Error shutting down SDK", err)
        )
        .finally(() => process.exit(0));
});



