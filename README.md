# external-dns-webhook-technitium

[![Go Version](https://img.shields.io/badge/go-1.21-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

A webhook provider for [external-dns](https://github.com/kubernetes-sigs/external-dns) that integrates with [Technitium DNS Server](https://technitium.com/dns/), enabling automatic DNS record management for Kubernetes services and ingresses.

## Features

- 🔄 Automatic DNS record synchronization from Kubernetes to Technitium DNS
- 🎯 Support for A, AAAA, CNAME, and TXT records
- 🔐 Token and username/password authentication
- 🌐 Domain filtering for selective DNS management
- 📊 Health checks and readiness probes
- 🐳 Docker and Kubernetes-ready
- 🚀 Complete local development environment with minikube

## Quick Start

### Prerequisites

- [Go 1.21+](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/get-docker/)
- [minikube](https://minikube.sigs.k8s.io/docs/start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [just](https://github.com/casey/just) (task runner)

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/klausklausen/external-dns-webhook-technitium.git
   cd external-dns-webhook-technitium
   ```

2. **Initialize Go modules:**
   ```bash
   just init
   ```

3. **Start the complete environment:**
   ```bash
   just start
   ```

   This command will:
   - Start a minikube cluster
   - Build the webhook Docker image
   - Deploy Technitium DNS server
   - Deploy the webhook
   - Deploy external-dns

4. **Check the status:**
   ```bash
   just status
   ```

### Testing the Setup

Create a test service to verify DNS record creation:

```bash
just test-service
```

Check the logs to see DNS records being created:

```bash
# View webhook logs
just logs-webhook

# View external-dns logs
just logs-external-dns
```

Access Technitium UI to verify DNS records:

```bash
just port-forward-technitium
# Then open http://localhost:5380 in your browser
# Default credentials: admin / admin
```

## Configuration

### Environment Variables

The webhook can be configured using the following environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `TECHNITIUM_URL` | `http://localhost:5380` | Technitium DNS server URL |
| `TECHNITIUM_USER` | `admin` | Technitium username |
| `TECHNITIUM_PASSWORD` | `admin` | Technitium password |
| `TECHNITIUM_TOKEN` | - | Technitium API token (overrides user/password) |
| `WEBHOOK_ADDR` | `:8888` | Webhook server listen address |
| `DOMAIN_FILTER` | - | Comma-separated list of domains to manage |
| `DRY_RUN` | `false` | Enable dry-run mode (no changes made) |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | `text` | Log format (text, json) |

### Command Line Flags

All environment variables can also be specified as command line flags:

```bash
./webhook \
  --technitium-url=http://dns.example.com:5380 \
  --technitium-user=admin \
  --technitium-password=secret \
  --domain-filter=example.com,test.com \
  --log-level=debug
```

## Architecture

```
┌─────────────────┐
│   Kubernetes    │
│   (Services/    │
│   Ingresses)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  external-dns   │
│   (watches K8s  │
│    resources)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     Webhook     │
│   (this project)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Technitium    │
│   DNS Server    │
└─────────────────┘
```

1. **external-dns** watches Kubernetes resources (Services, Ingresses) for DNS annotations
2. **Webhook** receives requests from external-dns and translates them to Technitium API calls
3. **Technitium DNS Server** stores and serves the DNS records

## Development

### Project Structure

```
.
├── cmd/
│   └── webhook/          # Main application entry point
├── internal/
│   ├── technitium/       # Technitium API client
│   └── webhook/          # Webhook server and provider
├── deploy/
│   ├── kubernetes/       # Webhook deployment manifests
│   ├── technitium/       # Technitium deployment manifests
│   └── external-dns/     # external-dns deployment manifests
├── docs/                 # Additional documentation
├── Dockerfile            # Multi-stage Docker build
├── justfile              # Task automation
└── README.md             # This file
```

### Common Tasks

```bash
# Build the binary locally
just build

# Run tests
just test

# Build Docker image
just docker-build

# View logs
just logs-webhook
just logs-technitium
just logs-external-dns

# Restart services
just restart-webhook
just restart-external-dns

# Clean up
just clean          # Delete Kubernetes resources
just clean-all      # Delete everything including minikube
```

### Making Changes

1. Make your code changes
2. Rebuild the Docker image: `just docker-build`
3. Redeploy the webhook: `just redeploy-webhook`
4. Check the logs: `just logs-webhook`

## Usage with Kubernetes

### Service Annotation

To create DNS records for a Kubernetes Service, add the `external-dns.alpha.kubernetes.io/hostname` annotation:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    external-dns.alpha.kubernetes.io/hostname: my-app.example.com
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: my-app
```

### Ingress Annotation

For Ingress resources, external-dns automatically uses the hostname from the Ingress spec:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
spec:
  rules:
  - host: my-app.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-service
            port:
              number: 80
```

### Custom TTL

Set a custom TTL for DNS records:

```yaml
metadata:
  annotations:
    external-dns.alpha.kubernetes.io/hostname: my-app.example.com
    external-dns.alpha.kubernetes.io/ttl: "60"
```

## Troubleshooting

### Webhook not connecting to Technitium

Check the webhook configuration:
```bash
kubectl get configmap external-dns-webhook-technitium -n external-dns -o yaml
```

Verify Technitium is running:
```bash
kubectl get pods -n external-dns -l app=technitium-dns
```

### DNS records not being created

Check external-dns logs:
```bash
just logs-external-dns
```

Check webhook logs:
```bash
just logs-webhook
```

Verify the service has the correct annotation:
```bash
kubectl get service <service-name> -o yaml | grep external-dns
```

### Permission issues

Verify RBAC is correctly configured:
```bash
kubectl get clusterrolebinding external-dns
kubectl get serviceaccount external-dns -n external-dns
```

## API Endpoints

The webhook exposes the following endpoints:

- `GET /` - Service information
- `GET /healthz` - Health check
- `GET /readyz` - Readiness check
- `GET /records` - List all DNS records
- `POST /records` - Apply DNS record changes
- `POST /adjustendpoints` - Adjust endpoints before processing

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [external-dns](https://github.com/kubernetes-sigs/external-dns) - The external-dns project
- [Technitium DNS Server](https://technitium.com/dns/) - Self-hosted DNS server
- [kubernetes-sigs](https://github.com/kubernetes-sigs) - Kubernetes Special Interest Groups

## Support

- 📖 [Documentation](docs/)
- 🐛 [Issue Tracker](https://github.com/klausklausen/external-dns-webhook-technitium/issues)
- 💬 [Discussions](https://github.com/klausklausen/external-dns-webhook-technitium/discussions)

## Roadmap

- [ ] Support for additional record types (MX, SRV)
- [ ] Metrics endpoint for Prometheus
- [ ] Helm chart for easier deployment
- [ ] Integration tests
- [ ] Support for multiple Technitium instances
- [ ] TLS support for webhook endpoint
