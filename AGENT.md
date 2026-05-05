# AGENT.md - AI Assistant Context

This document provides context and guidance for AI assistants working with this codebase.

## Project Overview

**external-dns-webhook-technitium** is a Kubernetes webhook provider that bridges external-dns and Technitium DNS Server. It enables automatic DNS record management for Kubernetes workloads by watching Services and Ingresses and synchronizing their DNS requirements to a self-hosted Technitium DNS server.

## Architecture

### High-Level Flow
1. **external-dns** (from kubernetes-sigs) watches Kubernetes resources
2. When it detects services/ingresses with DNS annotations, it calls our **webhook**
3. The **webhook** translates external-dns requests to Technitium API calls
4. **Technitium DNS Server** manages the actual DNS records

### Key Components

#### 1. Technitium API Client (`internal/technitium/`)
- **client.go**: HTTP client for Technitium REST API
- **types.go**: Data structures matching Technitium's API schema
- Handles authentication (token or username/password)
- Manages zones and records (create, update, delete)

#### 2. Webhook Provider (`internal/webhook/`)
- **provider.go**: Implements external-dns provider interface
- **server.go**: HTTP server exposing webhook endpoints
- Transforms between external-dns and Technitium data models
- Handles domain filtering and record type mapping

#### 3. Main Application (`cmd/webhook/`)
- **main.go**: Entry point with configuration and setup
- Signal handling for graceful shutdown
- Logging configuration
- Environment variable and flag parsing

## Key Design Decisions

### 1. API Client Implementation
- Uses standard `net/http` instead of generated clients
- Simple, maintainable code without external dependencies
- Token authentication preferred over username/password
- Automatic retry and error handling for common scenarios

### 2. Provider Interface
- Implements `sigs.k8s.io/external-dns/provider.Provider`
- Methods:
  - `Records()`: Fetch all DNS records from Technitium
  - `ApplyChanges()`: Apply create/update/delete operations
  - `AdjustEndpoints()`: Pre-process endpoints (currently no-op)

### 3. Supported Record Types
Currently supports:
- **A**: IPv4 addresses
- **AAAA**: IPv6 addresses
- **CNAME**: Canonical names
- **TXT**: Text records

Future: MX, SRV records

### 4. Zone Management
- Automatically creates zones if they don't exist
- Extracts zone from DNS name by finding longest matching zone
- Filters zones based on domain filter configuration

## Development Workflow

### Local Development Setup

1. **Prerequisites**: minikube, kubectl, docker, go 1.21+, just
2. **Quick start**: `just start` (starts everything)
3. **Development cycle**:
   - Edit code
   - `just docker-build` (builds in minikube's Docker)
   - `just redeploy-webhook` (restarts pod)
   - `just logs-webhook` (view logs)

### Testing Strategy

**Manual Testing**:
```bash
just test-service    # Creates test service
just logs-webhook    # Verify DNS records created
just port-forward-technitium  # Check Technitium UI
```

**Unit Tests**: Currently minimal, room for expansion
**Integration Tests**: TODO - automated end-to-end testing

### Debugging

**Common Issues**:
1. **Compilation errors**: Run `just init` to download dependencies
2. **Connection refused**: Verify Technitium is running with `just status`
3. **Records not created**: Check both external-dns and webhook logs
4. **Permission denied**: Verify RBAC configuration in deploy/external-dns/

**Debugging Commands**:
```bash
just status              # Overall system status
just logs-webhook        # Webhook logs
just logs-external-dns   # external-dns logs
just logs-technitium     # Technitium logs
kubectl describe pod <pod-name> -n external-dns  # Pod details
```

## Code Patterns

### Error Handling
```go
if err != nil {
    log.Errorf("Operation failed: %v", err)
    return fmt.Errorf("context: %w", err)
}
```
- Always wrap errors with context
- Log errors before returning
- Use structured logging (logrus)

### Logging
```go
log.Infof("Action completed: %s", details)
log.Debugf("Debug info: %+v", data)
log.Warnf("Non-fatal issue: %v", warning)
log.Errorf("Error occurred: %v", err)
```
- Info: Important operations
- Debug: Detailed flow information
- Warn: Non-fatal issues
- Error: Failures

### Configuration
- Environment variables preferred
- Command-line flags as alternative
- Sensible defaults for development
- All configurable via Kubernetes ConfigMap/Secret

## Kubernetes Manifests

### Deployment Structure
```
deploy/
├── technitium/      # DNS server (StatefulSet + PVC)
├── kubernetes/      # Webhook (Deployment + Service)
└── external-dns/    # external-dns controller
```

### Resource Organization
- **Namespace**: `external-dns` for all components
- **ServiceAccounts**: Separate for webhook and external-dns
- **RBAC**: ClusterRole for external-dns to watch resources
- **ConfigMap/Secret**: Webhook configuration

## Justfile Commands

The `justfile` provides task automation:

**Setup**:
- `just install-deps` - Check prerequisites
- `just init` - Initialize Go modules
- `just start` - Complete environment setup

**Development**:
- `just build` - Build binary
- `just docker-build` - Build Docker image
- `just test` - Run tests
- `just redeploy-webhook` - Quick redeploy

**Operations**:
- `just status` - Check component status
- `just logs-*` - View logs
- `just restart-*` - Restart components
- `just clean` - Clean up deployments

**Testing**:
- `just test-service` - Create test service
- `just port-forward-technitium` - Access UI

## External Dependencies

### Go Modules
- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/sirupsen/logrus` - Structured logging
- `sigs.k8s.io/external-dns` - external-dns types

### Container Images
- `golang:1.21-alpine` - Build stage
- `alpine:3.19` - Runtime stage
- `technitium/dns-server:latest` - DNS server
- `registry.k8s.io/external-dns/external-dns:v0.14.0` - Controller

## Common Modifications

### Adding a New Record Type

1. Update `internal/technitium/client.go`:
   - Add cases to `AddRecord()`, `DeleteRecord()`
   
2. Update `internal/webhook/provider.go`:
   - Add to `isSupportedRecordType()`
   - Add case to `convertToEndpoint()`

3. Test with appropriate Kubernetes resource

### Changing Default Configuration

1. Update environment variables in `deploy/kubernetes/webhook-deployment.yaml`
2. Update defaults in `cmd/webhook/main.go`
3. Update `README.md` configuration section
4. Redeploy: `just redeploy-webhook`

### Adding New Justfile Recipe

1. Add recipe to `justfile` with `@echo` for user feedback
2. Use variables like `{{NAMESPACE}}` for consistency
3. Add to README.md if it's user-facing
4. Test in clean environment

## Testing Considerations

### Unit Tests
- Mock Technitium client for provider tests
- Test record type conversions
- Test zone extraction logic
- Test error handling paths

### Integration Tests
- Deploy to test cluster
- Create Kubernetes resources
- Verify DNS records in Technitium
- Test cleanup/deletion
- Test edge cases (invalid domains, etc.)

## Performance Considerations

- **Caching**: Consider caching zone list (currently fetched on each operation)
- **Rate Limiting**: Technitium might need rate limiting for large clusters
- **Parallel Operations**: Currently sequential, could be parallelized
- **Connection Pooling**: HTTP client reuses connections

## Security Considerations

- **Credentials**: Stored in Kubernetes Secrets
- **RBAC**: Minimal permissions for each component
- **Non-root**: All containers run as non-root users
- **Read-only FS**: Webhook runs with read-only root filesystem
- **Network Policies**: TODO - isolate components

## Future Enhancements

1. **Metrics**: Prometheus metrics endpoint
2. **Helm Chart**: Package for easier deployment
3. **TLS**: Secure webhook endpoint
4. **HA**: Multiple webhook replicas
5. **Caching**: Zone and record caching
6. **Tests**: Comprehensive test suite
7. **CI/CD**: Automated builds and tests

## Troubleshooting Guide for AI Assistants

When helping with issues:

1. **Check logs first**: `just logs-webhook`, `just logs-external-dns`
2. **Verify connectivity**: Can webhook reach Technitium?
3. **Check configuration**: ConfigMap, Secret values correct?
4. **Validate resources**: Do Services have correct annotations?
5. **Review RBAC**: Does external-dns have required permissions?
6. **Test manually**: Use `kubectl port-forward` to test webhook directly

## References

- [external-dns Documentation](https://github.com/kubernetes-sigs/external-dns)
- [Technitium API Docs](https://github.com/TechnitiumSoftware/DnsServer/blob/master/APIDOCS.md)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Go Best Practices](https://go.dev/doc/effective_go)
