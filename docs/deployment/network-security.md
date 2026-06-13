# Network Security

NetworkPolicy support is optional because enforcement depends on the Kubernetes CNI.

The chart models:

- ingress/controller traffic to frontends and API Gateway
- API Gateway to internal services
- services to PostgreSQL, Redis, Kafka/Redpanda, MQTT, and OTLP
- DNS egress
- optional controlled external egress

Production values enable NetworkPolicies and default-deny ingress. Review generated manifests against the target CNI before enforcing default-deny egress.

