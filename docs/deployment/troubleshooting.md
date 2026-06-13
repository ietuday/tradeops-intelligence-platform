# Kubernetes Troubleshooting

Start with:

```bash
make k8s-status
kubectl -n tradeops describe pod <pod>
kubectl -n tradeops logs <pod> --tail=120
```

Common issues:

- Missing Secret keys: verify `POSTGRES_PASSWORD`, `JWT_SECRET`, and identity refresh secret.
- External dependency DNS failure: verify service hostnames and NetworkPolicies.
- Migration job failed: inspect job logs and confirm schema compatibility.
- Ingress unavailable: use port-forwarding first, then verify ingress class and controller.
- WebSocket failures: confirm API Gateway ingress route supports connection upgrades and timeout settings.

