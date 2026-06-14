# Kubernetes Troubleshooting

Start with:

```bash
make k8s-status
kubectl -n tradeops describe pod <pod>
kubectl -n tradeops logs <pod> --tail=120
```

Common issues:

- `ImagePullBackOff` or `ErrImagePull`: run `make k8s-load-images`, or rerun `make k8s-deploy-local` with Docker running. Local Kind deployments use locally built `tradeops/*` images and do not require Docker Hub pulls.
- `CreateContainerConfigError` with `runAsNonRoot` and a named user: rebuild the image after the v3.0.0 runtime fix and run `make validate-container-users`. Backend images must report numeric `10001:10001` from `docker inspect`.
- `CrashLoopBackOff` for Redpanda: inspect `kubectl logs -n tradeops deployment/tradeops-tradeops-redpanda --previous`. Local Redpanda uses `redpanda start` with single-node listener settings and must advertise the Kubernetes service DNS name, not `localhost`.
- Helm `pending-install` or `pending-upgrade`: run `make k8s-status`. The local deploy script can recover a pending local release with `K8S_RECOVER_PENDING=true` while preserving the namespace and cluster. Set `K8S_RECOVER_PENDING=false` to inspect manually.
- Missing Secret keys: verify `POSTGRES_PASSWORD`, `JWT_SECRET`, and identity refresh secret.
- External dependency DNS failure: verify service hostnames and NetworkPolicies.
- Migration job failed: inspect job logs and confirm schema compatibility.
- Ingress unavailable: use port-forwarding first, then verify ingress class and controller.
- WebSocket failures: confirm API Gateway ingress route supports connection upgrades and timeout settings.
