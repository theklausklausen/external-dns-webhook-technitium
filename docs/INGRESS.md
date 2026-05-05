# Ingress Setup Guide

This guide explains how to use Ingress resources with external-dns and Technitium DNS.

## Prerequisites

The NGINX Ingress Controller is automatically enabled when you run `just start`. If you need to enable it manually:

```bash
minikube addons enable ingress --profile=external-dns
```

## Technitium Web UI Ingress

Access the Technitium web interface via Ingress instead of port-forwarding.

### Deploy Technitium Ingress

```bash
just deploy-technitium-ingress
```

### Access Technitium UI

```bash
# Get your minikube IP
just minikube-ip

# Access via browser (replace IP if different)
# http://dns.192.168.49.2.nip.io
```

**Default Credentials**: admin / admin

### Configuration

The ingress is configured in `deploy/technitium/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: technitium-dns-ingress
  namespace: external-dns
  annotations:
    external-dns.alpha.kubernetes.io/hostname: dns.192.168.49.2.nip.io
spec:
  ingressClassName: nginx
  rules:
  - host: dns.192.168.49.2.nip.io
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: technitium-dns
            port:
              number: 5380
```

## Test Application Ingress

Deploy a test application with Ingress to verify external-dns creates DNS records.

### Deploy Test Service with Ingress

```bash
just test-service-with-ingress
```

This creates:
1. A test nginx deployment
2. A LoadBalancer service with `external-dns.alpha.kubernetes.io/hostname: test.example.com`
3. An Ingress with host `test.192.168.49.2.nip.io`

### Verify DNS Record Creation

```bash
# Watch webhook logs
just logs-webhook

# Watch external-dns logs
just logs-external-dns

# Check Technitium UI for created records
# http://dns.192.168.49.2.nip.io
```

### Access Test Application

```bash
# Access via browser
# http://test.192.168.49.2.nip.io
```

## How external-dns Works with Ingress

external-dns automatically creates DNS records for Ingress resources:

1. **Ingress without annotation**: Uses the host from `spec.rules[].host`
2. **Ingress with annotation**: Uses `external-dns.alpha.kubernetes.io/hostname`

### Example: Automatic Host Detection

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app
spec:
  rules:
  - host: myapp.example.com  # external-dns creates record automatically
    http:
      paths:
      - path: /
        backend:
          service:
            name: my-app
            port:
              number: 80
```

### Example: Custom Hostname

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app
  annotations:
    external-dns.alpha.kubernetes.io/hostname: custom.example.com
spec:
  rules:
  - host: myapp.example.com
    http:
      paths:
      - path: /
        backend:
          service:
            name: my-app
            port:
              number: 80
```

## Using nip.io for Local Development

[nip.io](https://nip.io) provides wildcard DNS for any IP address. It's perfect for local development:

- `dns.192.168.49.2.nip.io` → resolves to `192.168.49.2`
- `test.192.168.49.2.nip.io` → resolves to `192.168.49.2`
- `anything.192.168.49.2.nip.io` → resolves to `192.168.49.2`

This allows you to access services without modifying `/etc/hosts` or running a local DNS server.

## Custom Domain Configuration

To use your own domain instead of nip.io:

### 1. Update Ingress Hostnames

Edit `deploy/technitium/ingress.yaml` and `deploy/test/test-ingress.yaml`:

```yaml
spec:
  rules:
  - host: dns.yourdomain.com  # Change this
```

### 2. Configure DNS

Point your domain to the minikube IP:

```bash
# Get minikube IP
just minikube-ip

# Add A record in your DNS provider:
# dns.yourdomain.com → <minikube-ip>
```

### 3. Update Annotations

Ensure the hostname annotation matches:

```yaml
metadata:
  annotations:
    external-dns.alpha.kubernetes.io/hostname: dns.yourdomain.com
```

### 4. Redeploy

```bash
kubectl apply -f deploy/technitium/ingress.yaml
kubectl apply -f deploy/test/test-ingress.yaml
```

## Troubleshooting

### Ingress Not Working

**Check NGINX Ingress Controller**:
```bash
kubectl get pods -n ingress-nginx
```

If not running, enable it:
```bash
minikube addons enable ingress --profile=external-dns
```

**Check Ingress Resources**:
```bash
kubectl get ingress -A
kubectl describe ingress technitium-dns-ingress -n external-dns
```

### DNS Records Not Created

**Check external-dns logs**:
```bash
just logs-external-dns
```

**Verify external-dns is watching Ingress**:
```bash
just logs-external-dns | grep -i ingress
```

**Check webhook logs**:
```bash
just logs-webhook
```

### Wrong Minikube IP

If your minikube IP is different from 192.168.49.2:

```bash
# Get actual IP
MINIKUBE_IP=$(minikube ip --profile=external-dns)

# Update ingress files with correct IP
sed -i "s/192.168.49.2/$MINIKUBE_IP/g" deploy/technitium/ingress.yaml
sed -i "s/192.168.49.2/$MINIKUBE_IP/g" deploy/test/test-ingress.yaml

# Redeploy
kubectl apply -f deploy/technitium/ingress.yaml
kubectl apply -f deploy/test/test-ingress.yaml
```

### Can't Access via Browser

**Test connectivity**:
```bash
curl -v http://dns.192.168.49.2.nip.io
```

**Check service endpoints**:
```bash
kubectl get endpoints -n external-dns
```

**Test service directly**:
```bash
kubectl port-forward -n external-dns service/technitium-dns 5380:5380
```

## Advanced Configuration

### TLS/HTTPS

To enable TLS for Ingress:

```yaml
metadata:
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - dns.yourdomain.com
    secretName: technitium-tls
```

Requires [cert-manager](https://cert-manager.io) to be installed.

### Multiple Hosts

```yaml
spec:
  rules:
  - host: dns.yourdomain.com
    http:
      paths:
      - path: /
        backend:
          service:
            name: technitium-dns
            port:
              number: 5380
  - host: dns-admin.yourdomain.com
    http:
      paths:
      - path: /
        backend:
          service:
            name: technitium-dns
            port:
              number: 5380
```

### Custom Annotations

```yaml
metadata:
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
```

## Quick Reference

```bash
# Deploy Technitium ingress
just deploy-technitium-ingress

# Deploy test service with ingress
just test-service-with-ingress

# Show all URLs
just show-urls

# Get minikube IP
just minikube-ip

# View all ingress resources
kubectl get ingress -A

# Delete test resources
just delete-test-service
```

## Resources

- [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/)
- [external-dns Ingress Support](https://github.com/kubernetes-sigs/external-dns/blob/master/docs/tutorials/nginx-ingress.md)
- [nip.io](https://nip.io)
- [Kubernetes Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)
