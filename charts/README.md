# Helm Charts

This directory contains Helm charts for deploying the external-dns webhook for Technitium DNS.

## Available Charts

### external-dns-webhook-technitium

The main Helm chart for deploying the webhook provider.

**Quick Start:**

```bash
# Install the chart
helm install my-webhook ./external-dns-webhook-technitium \
  --namespace external-dns \
  --create-namespace

# Customize values
helm install my-webhook ./external-dns-webhook-technitium \
  --namespace external-dns \
  --create-namespace \
  --set technitium.url=http://my-technitium:5380 \
  --set technitium.password=mysecret
```

**See:** [external-dns-webhook-technitium/README.md](./external-dns-webhook-technitium/README.md) for full documentation.

## Chart Structure

```
external-dns-webhook-technitium/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default configuration values
├── README.md               # Chart documentation
├── .helmignore            # Files to ignore when packaging
└── templates/
    ├── _helpers.tpl       # Template helpers
    ├── serviceaccount.yaml
    ├── configmap.yaml
    ├── secret.yaml
    ├── service.yaml
    ├── deployment.yaml
    └── NOTES.txt          # Post-install notes
```

## Development

### Linting

```bash
helm lint external-dns-webhook-technitium
```

### Template Rendering

```bash
helm template my-webhook external-dns-webhook-technitium \
  --namespace external-dns
```

### Package

```bash
helm package external-dns-webhook-technitium
```

## Usage with Just

The project includes Just recipes for Helm operations:

```bash
# Install via Helm
just helm-install

# Upgrade via Helm
just helm-upgrade

# Uninstall
just helm-uninstall
```
