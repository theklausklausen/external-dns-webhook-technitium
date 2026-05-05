# Quick Start Guide

Get started with external-dns-webhook-technitium in minutes!

## Prerequisites Check

```bash
# Verify prerequisites
just install-deps
```

You need:
- Go 1.21+
- Docker
- minikube
- kubectl
- just

## 🚀 Quick Start (5 minutes)

### 1. Clone and Initialize

```bash
git clone https://github.com/klausklausen/external-dns-webhook-technitium.git
cd external-dns-webhook-technitium
just init
```

### 2. Start Everything

```bash
just start
```

This single command will:
- ✅ Start minikube cluster
- ✅ Build Docker image
- ✅ Deploy Technitium DNS server
- ✅ Deploy webhook
- ✅ Deploy external-dns

Wait 2-3 minutes for everything to be ready.

### 3. Verify Installation

```bash
just status
```

All pods should show `Running` status.

### 4. Test DNS Record Creation

```bash
# Create a test service with DNS annotation
just test-service

# Watch webhook logs (in a new terminal)
just logs-webhook

# Watch external-dns logs
just logs-external-dns
```

### 5. Access Technitium UI

```bash
# Port forward Technitium UI (in a new terminal)
just port-forward-technitium

# Open browser to http://localhost:5380
# Login: admin / admin
```

You should see DNS records created automatically!

## 🎯 What Just Happened?

1. **external-dns** is watching Kubernetes Services and Ingresses
2. When it finds a service with `external-dns.alpha.kubernetes.io/hostname` annotation
3. It calls the **webhook** to create/update/delete DNS records
4. The **webhook** translates these to **Technitium API** calls
5. **Technitium** stores and serves the DNS records

## 📝 Create Your Own Service

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: my-app
  annotations:
    external-dns.alpha.kubernetes.io/hostname: myapp.example.com
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: my-app
EOF
```

Check the logs to see the DNS record being created:
```bash
just logs-webhook
```

## 🔍 Debugging

### View Logs
```bash
just logs-webhook           # Webhook logs
just logs-external-dns      # external-dns logs
just logs-technitium        # Technitium logs
```

### Check Status
```bash
just status                 # All components
kubectl get pods -n external-dns
kubectl get services -n external-dns
```

### Access Webhook API Directly
```bash
kubectl port-forward -n external-dns service/external-dns-webhook-technitium 8888:8888

# In another terminal
curl http://localhost:8888/healthz
curl http://localhost:8888/records
```

## 🧹 Cleanup

### Clean Kubernetes Resources
```bash
just clean
```

### Delete Everything (including minikube)
```bash
just clean-all
```

## 🔧 Common Commands

```bash
# Restart webhook
just restart-webhook

# Rebuild and redeploy webhook
just redeploy-webhook

# View all available commands
just --list

# Get environment info
just info
```

## 📚 Next Steps

- Read [README.md](README.md) for detailed documentation
- Check [DEVELOPMENT.md](docs/DEVELOPMENT.md) for development guide
- Review [API.md](docs/API.md) for Technitium API reference
- See [AGENT.md](AGENT.md) for AI assistant context

## 🆘 Troubleshooting

### Pods not starting?
```bash
kubectl describe pod -n external-dns <pod-name>
```

### Webhook can't connect to Technitium?
```bash
kubectl get configmap external-dns-webhook-technitium -n external-dns -o yaml
```

### DNS records not created?
1. Check service has annotation: `kubectl get service <name> -o yaml | grep external-dns`
2. Check external-dns logs: `just logs-external-dns`
3. Check webhook logs: `just logs-webhook`

### Need to restart everything?
```bash
just restart
```

## 💡 Tips

- Use `just status` frequently to check component health
- Keep logs open in separate terminals while testing
- Access Technitium UI to verify DNS records visually
- Use dry-run mode for testing: Set `DRY_RUN=true` in ConfigMap

## 🎓 Learning Resources

- [external-dns Documentation](https://github.com/kubernetes-sigs/external-dns)
- [Technitium DNS Server](https://technitium.com/dns/)
- [Kubernetes Services](https://kubernetes.io/docs/concepts/services-networking/service/)

---

**Need Help?** Check the [README.md](README.md) or open an issue!
