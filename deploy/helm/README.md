# deploy/helm

The Helm umbrella chart `sms-manager`, with one subchart per deployable service. Each subchart sets resources, probes (`/livez`/`/readyz`), PodDisruptionBudgets, topology spread, non-root securityContext, ServiceMonitor, HPA/KEDA, and NetworkPolicies, with per-environment values files.
