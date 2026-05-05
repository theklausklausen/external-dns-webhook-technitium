# Development Guide

This guide provides detailed information for developers working on the external-dns-webhook-technitium project.

## Prerequisites

Before you begin, ensure you have the following tools installed:

- **Go 1.21+**: [Installation Guide](https://go.dev/doc/install)
- **Docker**: [Installation Guide](https://docs.docker.com/get-docker/)
- **minikube**: [Installation Guide](https://minikube.sigs.k8s.io/docs/start/)
- **kubectl**: [Installation Guide](https://kubernetes.io/docs/tasks/tools/)
- **just**: [Installation Guide](https://github.com/casey/just#installation)

Verify installations:
```bash
just install-deps
```

## Getting Started

### 1. Clone and Setup

```bash
git clone https://github.com/klausklausen/external-dns-webhook-technitium.git
cd external-dns-webhook-technitium
just init
```

### 2. Start Development Environment

```bash
just start
```

This will:
1. Start a minikube cluster with the profile "external-dns"
2. Build the Docker image inside minikube's Docker daemon
3. Deploy Technitium DNS server
4. Deploy the webhook
5. Deploy external-dns

### 3. Verify Setup

```bash
just status
```

Expected output should show all pods in `Running` state.

## Development Workflow

### Making Code Changes

1. **Edit the code** in your preferred editor
2. **Build the Docker image**:
   ```bash
   just docker-build
   ```
3. **Redeploy the webhook**:
   ```bash
   just redeploy-webhook
   ```
4. **View logs**:
   ```bash
   just logs-webhook
   ```

### Testing Changes

#### Create a Test Service

```bash
just test-service
```

This creates a test nginx deployment with a LoadBalancer service that has DNS annotations.

#### Verify DNS Record Creation

1. Check webhook logs:
   ```bash
   just logs-webhook
   ```

2. Check external-dns logs:
   ```bash
   just logs-external-dns
   ```

3. Access Technitium UI:
   ```bash
   just port-forward-technitium
   # Open http://localhost:5380 in your browser
   # Login: admin / admin
   ```

#### Clean Up Test Service

```bash
just delete-test-service
```

## Project Structure Deep Dive

```
external-dns-webhook-technitium/
├── cmd/
│   └── webhook/
│       └── main.go                 # Application entry point
├── internal/
│   ├── technitium/
│   │   ├── client.go              # Technitium API client
│   │   ├── types.go               # API type definitions
│   │   └── client_test.go         # Unit tests (TODO)
│   └── webhook/
│       ├── provider.go            # external-dns provider implementation
│       ├── server.go              # HTTP server
│       └── provider_test.go       # Unit tests (TODO)
├── deploy/
│   ├── kubernetes/
│   │   └── webhook-deployment.yaml  # Webhook K8s manifests
│   ├── technitium/
│   │   └── deployment.yaml          # Technitium K8s manifests
│   └── external-dns/
│       └── deployment.yaml          # external-dns K8s manifests
├── docs/
│   ├── DEVELOPMENT.md             # This file
│   └── API.md                     # Technitium API reference
├── Dockerfile                      # Multi-stage Docker build
├── justfile                        # Task automation
├── go.mod                          # Go module definition
├── go.sum                          # Go module checksums
├── README.md                       # Main documentation
└── AGENT.md                        # AI assistant context
```

## Code Architecture

### Technitium Client (`internal/technitium/`)

The client provides a Go interface to the Technitium DNS Server API:

- **Authentication**: Supports both token and username/password auth
- **Zone Management**: Create, list zones
- **Record Management**: Add, update, delete records (A, AAAA, CNAME, TXT)
- **Health Checks**: Verify connectivity to Technitium

Key functions:
```go
NewClient(baseURL, token string) *Client
NewClientWithAuth(baseURL, username, password string) (*Client, error)
ListZones() ([]Zone, error)
GetRecords(zone string) ([]Record, error)
CreateZone(zone string) error
AddRecord(zone, name, recordType string, ttl int, value string) error
DeleteRecord(zone, name, recordType, value string) error
```

### Webhook Provider (`internal/webhook/`)

Implements the external-dns provider interface:

```go
type Provider interface {
    Records(ctx context.Context) ([]*endpoint.Endpoint, error)
    ApplyChanges(ctx context.Context, changes *plan.Changes) error
    AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error)
}
```

Key responsibilities:
- Convert between external-dns endpoints and Technitium records
- Filter domains based on configuration
- Handle record lifecycle (create, update, delete)
- Extract zone information from DNS names

### HTTP Server (`internal/webhook/server.go`)

Exposes webhook endpoints for external-dns:

- `GET /` - Service information
- `GET /healthz` - Health check
- `GET /readyz` - Readiness check
- `GET /records` - List all DNS records
- `POST /records` - Apply DNS changes
- `POST /adjustendpoints` - Adjust endpoints

## Building and Testing

### Build Binary Locally

```bash
just build
```

Output: `bin/webhook`

### Run Locally (Outside Kubernetes)

```bash
# Start Technitium locally (or use port-forward)
just port-forward-technitium

# In another terminal
export TECHNITIUM_URL=http://localhost:5380
export TECHNITIUM_USER=admin
export TECHNITIUM_PASSWORD=admin
./bin/webhook
```

### Run Tests

```bash
just test
```

### Build Docker Image

```bash
just docker-build
```

The Dockerfile uses multi-stage builds:
1. **Build stage**: Compiles Go binary
2. **Runtime stage**: Minimal Alpine image with binary

## Kubernetes Deployment

### Manual Deployment Steps

```bash
# Deploy Technitium
kubectl apply -f deploy/technitium/deployment.yaml

# Wait for Technitium
kubectl wait --for=condition=ready pod -l app=technitium-dns -n external-dns --timeout=300s

# Deploy Webhook
kubectl apply -f deploy/kubernetes/webhook-deployment.yaml

# Deploy external-dns
kubectl apply -f deploy/external-dns/deployment.yaml
```

### Configuration

Edit `deploy/kubernetes/webhook-deployment.yaml` to change configuration:

```yaml
data:
  TECHNITIUM_URL: "http://technitium-dns.external-dns.svc.cluster.local:5380"
  LOG_LEVEL: "debug"  # Change to debug for more verbose logging
```

Apply changes:
```bash
kubectl apply -f deploy/kubernetes/webhook-deployment.yaml
kubectl rollout restart deployment/external-dns-webhook-technitium -n external-dns
```

## Debugging

### View Logs

```bash
# Webhook logs
just logs-webhook

# External-dns logs
just logs-external-dns

# Technitium logs
just logs-technitium

# Follow logs with tail
kubectl logs -f deployment/external-dns-webhook-technitium -n external-dns
```

### Access Pod Shell

```bash
kubectl exec -it deployment/external-dns-webhook-technitium -n external-dns -- sh
```

### Port Forward Services

```bash
# Technitium UI
just port-forward-technitium

# Webhook API
kubectl port-forward -n external-dns service/external-dns-webhook-technitium 8888:8888

# Test webhook directly
curl http://localhost:8888/
curl http://localhost:8888/healthz
curl http://localhost:8888/records
```

### Common Issues

**Issue**: Webhook can't connect to Technitium
```bash
# Check Technitium is running
kubectl get pods -n external-dns -l app=technitium-dns

# Check service exists
kubectl get service technitium-dns -n external-dns

# Check webhook configuration
kubectl get configmap external-dns-webhook-technitium -n external-dns -o yaml
```

**Issue**: Records not being created
```bash
# Check service has annotation
kubectl get service -A -o yaml | grep external-dns

# Check external-dns is watching
just logs-external-dns | grep -i watch

# Check webhook is receiving requests
just logs-webhook | grep -i record
```

**Issue**: Permission denied errors
```bash
# Check RBAC
kubectl get clusterrole external-dns -o yaml
kubectl get clusterrolebinding external-dns -o yaml
kubectl get serviceaccount external-dns -n external-dns
```

## Adding New Features

### Adding a New Record Type

1. **Update client** (`internal/technitium/client.go`):
   ```go
   func (c *Client) AddRecord(zone, name, recordType string, ttl int, value string) error {
       // ... existing code ...
       switch recordType {
       case "A", "AAAA":
           params.Set("ipAddress", value)
       case "CNAME":
           params.Set("cname", value)
       case "TXT":
           params.Set("text", value)
       case "MX":  // New type
           params.Set("exchange", value)
           params.Set("preference", "10")  // Could be extracted from value
       default:
           return fmt.Errorf("unsupported record type: %s", recordType)
       }
       // ... rest of code ...
   }
   ```

2. **Update provider** (`internal/webhook/provider.go`):
   ```go
   func isSupportedRecordType(recordType string) bool {
       supportedTypes := map[string]bool{
           "A":     true,
           "AAAA":  true,
           "CNAME": true,
           "TXT":   true,
           "MX":    true,  // Add new type
       }
       return supportedTypes[recordType]
   }
   ```

3. **Test**:
   ```bash
   just docker-build
   just redeploy-webhook
   # Create service with MX record annotation
   ```

### Adding Metrics

Consider adding Prometheus metrics:

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    recordsCreated = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "webhook_records_created_total",
            Help: "Total number of DNS records created",
        },
    )
)

func init() {
    prometheus.MustRegister(recordsCreated)
}
```

Add metrics endpoint to server:
```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

s.router.Handle("/metrics", promhttp.Handler())
```

## Performance Optimization

### Caching

Consider adding zone caching to reduce API calls:

```go
type CachedClient struct {
    client    *Client
    zoneCache map[string]time.Time
    cacheTTL  time.Duration
}
```

### Connection Pooling

The HTTP client already uses connection pooling, but you can tune it:

```go
httpClient: &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
},
```

## Contributing

### Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Add comments for exported functions
- Keep functions small and focused

### Commit Messages

Use conventional commit format:
```
feat: add support for MX records
fix: handle connection timeout gracefully
docs: update development guide
chore: update dependencies
```

### Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Update documentation
6. Submit pull request

## Resources

- [external-dns Documentation](https://github.com/kubernetes-sigs/external-dns)
- [Technitium API Documentation](https://github.com/TechnitiumSoftware/DnsServer/blob/master/APIDOCS.md)
- [Go Documentation](https://go.dev/doc/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
