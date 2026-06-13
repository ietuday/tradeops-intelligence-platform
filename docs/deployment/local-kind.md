# Local Kind Deployment

Prerequisites:

- Docker
- Kind
- kubectl
- Helm

Run:

```bash
make k8s-create-local
make k8s-deploy-local
make k8s-smoke
make k8s-status
```

Access:

```bash
kubectl -n tradeops port-forward svc/api-gateway 8080:8080
kubectl -n tradeops port-forward svc/trading-dashboard-react 4300:8080
kubectl -n tradeops port-forward svc/shell-angular 4200:8080
```

Cleanup:

```bash
make k8s-destroy-local
DELETE_KIND_CLUSTER=true make k8s-destroy-local
```

The first cleanup removes only the `tradeops` namespace. Cluster deletion requires the explicit `DELETE_KIND_CLUSTER=true` flag.

