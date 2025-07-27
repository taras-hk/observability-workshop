# Observability Demo Project

This project demonstrates modern observability practices using a microservices architecture. It covers three main pillars of observability:
- **Logging** (structured vs unstructured)
- **Metrics** (Prometheus)
- **Distributed Tracing** (OpenTelemetry + Jaeger)

## Problem Overview

The project simulates a subscription management system to demonstrate the importance of proper observability practices. It focuses on the evolution from poor to excellent practices in two key areas:

### 1. Structured vs Unstructured Logging

#### Unstructured Logging Example (V1 - Bad)
```go
log.Printf("subscription %s not found", id)
```
Problems:
- Hard to parse
- No standardized format
- Difficult to search and analyze
- Limited context

#### Structured Logging Example (V3 - Best)
```go
logger.Error().
    Str("subscription_id", id).
    Str("user_id", userId).
    Str("operation", "get_subscription").
    Msg("Subscription not found")
```
Benefits:
- Machine-parseable JSON format
- Consistent structure
- Rich context
- Easy to analyze and search
- Better error tracking

### 2. Metrics Evolution: Basic → Dimensional → SLI-Focused

#### Basic Metrics (V1 - Bad)
```go
// Just basic counters with no context
requests := prometheus.NewCounter(prometheus.CounterOpts{
    Name: "requests",  // Too generic
    Help: "requests",  // Unhelpful
})
```
Problems:
- No labels/dimensions
- No business context
- Can't differentiate performance by endpoint
- No SLI/SLO alignment

#### Better Metrics (V2 - Improved)
```go
requests := prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "service_requests_total",
    Help: "Total HTTP requests",
}, []string{"method", "endpoint"})  // Some labels but inconsistent
```
Improvements but still issues:
- Some labels added
- Better naming
- Still inconsistent labeling
- Missing business metrics

#### Best Practice Metrics (V3 - Excellent)
```go
requests := prometheus.NewCounterVec(prometheus.CounterOpts{
    Name: "subscription_service_http_requests_total",
    Help: "Total HTTP requests (SLI: Request Rate)",
}, []string{"method", "endpoint", "status_class"})  // Consistent, SLI-focused
```
Best practices:
- Consistent labeling strategy
- SLI/SLO aligned metrics
- Rich business dimensions
- Proper histogram buckets
- Clear error classification

## Tracing Evolution (V1 → V2 → V3)

### Terrible Tracing (V1 - Poor)
```go
// No context propagation, manual timing
func handleRequest(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    log.Printf("Starting operation")
    // ... business logic
    log.Printf("Completed in %v", time.Since(start))
}
```
Problems:
- No context propagation (breaks distributed tracing)
- Manual timing instead of spans
- No span attributes or metadata
- Fragmented traces
- No error context

### Better Tracing (V2 - Improved)
```go
// Basic context propagation, some attributes
func (t *TracingV2) InstrumentHandler(handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := t.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
        ctx, span := t.tracer.Start(ctx, r.Method+" "+r.URL.Path)
        defer span.End()
        
        span.SetAttributes(
            semconv.HTTPMethod(r.Method),
            semconv.HTTPTarget(r.URL.Path),
        )
        
        handler(w, r.WithContext(ctx))
    }
}
```
Improvements but limitations:
- Basic context propagation working
- Some HTTP semantic attributes
- Limited business context
- Inconsistent error handling

### Excellent Tracing (V3 - Production-Ready)
```go
// Full semantic conventions, rich business context
func (t *TracingV3) InstrumentHandler(handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := t.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
        spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
        ctx, span := t.tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
        defer span.End()

        // Rich semantic attributes
        span.SetAttributes(
            semconv.HTTPMethod(r.Method),
            semconv.HTTPTarget(r.URL.Path),
            attribute.String("user.id", r.Header.Get("X-User-ID")),
            attribute.String("tenant.id", r.Header.Get("X-Tenant-ID")),
        )
        
        // Span events for important milestones
        span.AddEvent("request.started")
        
        // Business context via baggage
        ctx = t.AddBusinessContext(ctx, userID, tenantID, sessionID)
        
        handler(w, r.WithContext(ctx))
        
        // Performance monitoring
        if duration > threshold {
            span.AddEvent("slow_request", trace.WithAttributes(
                attribute.Int64("duration_ms", duration.Milliseconds()),
            ))
        }
    }
}
```
Best practices:
- Full context propagation with baggage
- Rich semantic conventions (HTTP, DB, business)
- Comprehensive span attributes
- Span events and annotations
- Performance monitoring
- Business context tracking
- Proper error categorization

## Architecture

### Services

1. **Subscription Service** (`subscription-service/`)
   - Manages subscription lifecycle
   - Handles CRUD operations
   - Connects to Payment Service
   - Port: 8082

2. **Payment Service** (`payment-service/`)
   - Processes payments
   - Validates subscription plans
   - Port: 8081

### Observability Stack

1. **ELK Stack**
   - Elasticsearch (port: 9200)
   - Logstash (port: 5044)
   - Kibana (port: 5601)
   - Handles log aggregation and analysis

2. **Prometheus** (port: 9090)
   - Metrics collection
   - Alert rules
   - Key metrics:
     - Request counts
     - Error rates
     - Response times
     - Queue lengths

3. **Jaeger** (port: 16686)
   - Distributed tracing
   - Request flow visualization
   - Performance analysis
   - Service dependencies

### Core Modules

1. **pkg/observability/**
   - Shared observability utilities
   - **Logging Evolution:** V1 (unstructured) → V2 (contextual) → V3 (structured)
   - **Metrics Evolution:** V1 (basic counters) → V2 (some labels) → V3 (SLI-focused)
   - **Tracing Evolution:** V1 (fragmented) → V2 (basic context) → V3 (semantic conventions)
   - Configuration management

## Building and Running

### Prerequisites
- Go 1.20+
- Docker and Docker Compose
- Make
- curl
- jq (for JSON processing)

### Quick Start
```bash
# Start all services
make start-all
```

### Infrastructure Commands
```bash
make start-all     # Start all services (Jaeger, Prometheus, ELK)
make stop-all      # Stop all services
```

### Observability Evolution Simulations
```bash
make simulate-v1     # Demonstrate poor observability practices
make simulate-v2     # Demonstrate improved observability practices  
make simulate-v3     # Demonstrate excellent observability practices
```

### Alerts Testing Evolution (V1 → V2 → V3)
```bash
make test-alerts-v1         # Basic business errors (poor practices)
make test-alerts-v2         # Business-aware patterns (improved practices)
make test-alerts-v3         # SLO breach + business alerts (excellent practices)
```

## Observability UIs

After starting the services, access:
- Kibana: http://localhost:5601 (logs)
- Prometheus: http://localhost:9090 (metrics)
- Jaeger UI: http://localhost:16686 (traces)

### Kibana Setup

When accessing Kibana for the first time, you'll need to set up an index pattern to view the logs:

1. Open Kibana at http://localhost:5601
2. Navigate to **Stack Management** → **Index Patterns** (or **Data Views** in newer versions)
3. Click **Create index pattern** (or **Create data view**)
4. Enter the index pattern: `logstash-*`
5. Click **Next step**
6. Select the timestamp field: `@timestamp`
7. Click **Create index pattern**

Once the index pattern is created, you can:
- Go to **Discover** to view and search logs
- Use the time picker to filter logs by time range
- Create filters to compare structured vs unstructured logs
- Build visualizations and dashboards in **Visualize** and **Dashboard**


## Stopping Simulations and Services

```bash
# Stop specific simulations (run in background)
pkill -f simulate_v1_observability    # Stop V1 simulation
pkill -f simulate_v2_observability    # Stop V2 simulation  
pkill -f simulate_v3_observability    # Stop V3 simulation
pkill -f test_alerts                  # Stop alert testing scripts

# Stop all background simulations
pkill -f simulate                     # Stop all simulation scripts
pkill -f test_alerts                  # Stop all alert testing scripts

# Stop infrastructure services
make stop-all                         # Stop Docker services (Jaeger, Prometheus, ELK)
```

## Quick Start Guide

```bash
# 1. Start the observability infrastructure
make start-all

# 2. Wait for services to be ready (automatically done by start-all)
# Access points will be displayed:
# - Jaeger UI: http://localhost:16686
# - Prometheus: http://localhost:9090  
# - Kibana: http://localhost:5601

# 3. Set up Kibana index pattern (see Kibana Setup section above)

# 4. Run observability simulations
make simulate-v1    # Poor practices
make simulate-v2    # Improved practices
make simulate-v3    # Excellent practices

# 5. Test alert scenarios
make test-alerts-v1 # Basic errors
make test-alerts-v2 # Business patterns
make test-alerts-v3 # Production SLO testing

# 6. Explore the differences in observability UIs
# - Compare log quality in Kibana
# - Compare metrics in Prometheus
# - Compare traces in Jaeger

# 7. Clean up
pkill -f simulate   # Stop simulations
make stop-all       # Stop infrastructure
```
