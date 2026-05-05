# external-dns-webhook-technitium Helm Chart

A Helm chart for deploying the external-dns webhook provider for Technitium DNS to Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- A running Technitium DNS server

## Installation

### Add the repository (if published)

```bash
helm repo add external-dns-webhook-technitium https://yourusername.github.io/external-dns-webhook-technitium
helm repo update
```

### Install from local chart

```bash
helm install external-dns-webhook-technitium ./charts/external-dns-webhook-technitium \
  --namespace external-dns \
  --create-namespace
```

### Install with custom values

```bash
helm install external-dns-webhook-technitium ./charts/external-dns-webhook-technitium \
  --namespace external-dns \
  --create-namespace \
  --set technitium.url=http://my-technitium:5380 \
  --set technitium.username=admin \
  --set technitium.password=mypassword
```

## Configuration

The following table lists the configurable parameters of the chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `external-dns-webhook-technitium` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag | `latest` |
| `technitium.url` | Technitium DNS server URL | `http://technitium-dns:5380` |
| `technitium.username` | Technitium username | `admin` |
| `technitium.password` | Technitium password | `admin` |
| `technitium.token` | Technitium API token (alternative to username/password) | `""` |
| `technitium.domainFilter` | List of domains to manage | `[]` |
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8888` |
| `resources.limits.cpu` | CPU limit | `200m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `50m` |
| `resources.requests.memory` | Memory request | `64Mi` |
| `logLevel` | Log level (debug, info, warn, error) | `info` |
| `logFormat` | Log format (text, json) | `text` |

### Example: Custom values file

Create a `values.yaml` file:

```yaml
technitium:
  url: "http://technitium-dns.dns-system.svc.cluster.local:5380"
  username: "admin"
  password: "supersecret"
  domainFilter:
    - example.com
    - test.com

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

logLevel: "debug"
```

Install with the custom values:

```bash
helm install external-dns-webhook-technitium ./charts/external-dns-webhook-technitium \
  --namespace external-dns \
  --create-namespace \
  -f values.yaml
```

## Usage with external-dns

After installing the webhook, configure external-dns to use it:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-dns
spec:
  template:
    spec:
      containers:
      - name: external-dns
        image: registry.k8s.io/external-dns/external-dns:v0.15.0
        args:
        - --source=service
        - --source=ingress
        - --provider=webhook
        - --webhook-provider-url=http://external-dns-webhook-technitium.external-dns.svc.cluster.local:8888
```

## Upgrading

```bash
helm upgrade external-dns-webhook-technitium ./charts/external-dns-webhook-technitium \
  --namespace external-dns
```

## Uninstalling

```bash
helm uninstall external-dns-webhook-technitium --namespace external-dns
```

## Troubleshooting

### Check pod status

```bash
kubectl get pods -n external-dns -l app.kubernetes.io/name=external-dns-webhook-technitium
```

### View logs

```bash
kubectl logs -n external-dns -l app.kubernetes.io/name=external-dns-webhook-technitium --tail=100 -f
```

### Test the webhook

```bash
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://external-dns-webhook-technitium.external-dns.svc.cluster.local:8888/healthz
```

## License

MIT License - see LICENSE file for details
