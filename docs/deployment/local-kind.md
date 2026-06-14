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
make k8s-status
make k8s-smoke
```

`make k8s-deploy-local` builds the first-party application images, loads them into the `tradeops-local` Kind cluster, validates the Helm chart, installs or upgrades the release, and waits for workloads. Manual `kind load docker-image` is no longer required for the standard local flow.

Useful split commands:

```bash
make k8s-build-images
make k8s-load-images
make validate-container-users
```

Backend application images declare `USER 10001:10001` so Kubernetes can verify `runAsNonRoot`. Named Docker users such as `USER tradeops` are intentionally avoided because kubelet cannot prove they are non-root from image metadata.

Local Redpanda runs as a single-node demo broker with in-cluster advertised address `tradeops-tradeops-redpanda:9092`. Production values should use externally operated Kafka/Redpanda brokers instead of this local dependency.

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
