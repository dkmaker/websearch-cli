---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 8
cached: false
---

# OpenTelemetry Auto-Instrumentation for FastAPI with SQLAlchemy and Celery

The search results provide comprehensive guidance for FastAPI auto-instrumentation but have limited specific details on SQLAlchemy async sessions and Celery worker integration. Here's what's available:

## Core FastAPI Setup

To set up automatic instrumentation for FastAPI, install the necessary packages:

```bash
pip install opentelemetry-api opentelemetry-sdk
pip install opentelemetry-instrumentation-fastapi
pip install opentelemetry-instrumentation-asgi
pip install opentelemetry-exporter-otlp
```

Then configure and instrument your app:

```python
from fastapi import FastAPI
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter

# Configure resource
resource = Resource(attributes={
    "service.name": "fastapi-service",
    "service.version": "1.0.0",
    "deployment.environment": "production"
})

# Set up OTLP exporter
otlp_exporter = OTLPSpanExporter(endpoint="http://localhost:4318")
span_processor = BatchSpanProcessor(otlp_exporter)

# Initialize tracer provider
tracer_provider = TracerProvider(resource=resource)
tracer_provider.add_span_processor(span_processor)

# Instrument FastAPI
app = FastAPI()
FastAPIInstrumentor.instrument_app(app)
```

## SQLAlchemy Database Instrumentation

The search results reference SQLAlchemy auto-instrumentation but don't provide specific code for async sessions. The general approach involves installing the SQLAlchemy instrumentation package and instrumenting database operations alongside FastAPI.

## Production Tuning

For high-throughput applications with Celery workers, configure the batch processor appropriately:

```python
span_processor = BatchSpanProcessor(
    otlp_exporter,
    max_queue_size=4096,
    schedule_delay_millis=3000,
    max_export_batch_size=1024
)
```

## Limitations

The search results do not provide comprehensive instructions for:
- **SQLAlchemy async session instrumentation** specifically
- **Celery worker distributed tracing** configuration
- **Context propagation** between async tasks and workers

You may need to consult OpenTelemetry's official SQLAlchemy and Celery instrumentation documentation for complete async and distributed worker setup, as these are beyond the scope of the current search results.