# deploy/helm

The Helm umbrella chart `sms-manager`, with one subchart per deployable service. Each subchart should set resources, probes (`/livez`/`/readyz`), PodDisruptionBudgets, topology spread, non-root securityContext, ServiceMonitor, HPA/KEDA, and NetworkPolicies, with per-environment values files — see "Add a new service's subchart" below for what the placeholder has today versus what a real Phase 1 service needs to add.

## Layout

```
deploy/helm/sms-manager/
├── Chart.yaml              # umbrella chart metadata + dependencies: one entry per subchart
├── values.yaml             # one top-level key per subchart, e.g. `gateway-api:`
├── templates/               # umbrella-level templates only (NOTES.txt, etc.) — rare
└── charts/
    └── gateway-api/          # placeholder subchart (SMM-22) — proves the pattern
        ├── Chart.yaml
        ├── values.yaml
        ├── values.schema.json   # optional: validates values at install/template time
        └── templates/
            ├── _helpers.tpl
            ├── deployment.yaml
            ├── service.yaml
            ├── serviceaccount.yaml
            ├── poddisruptionbudget.yaml
            └── NOTES.txt
```

`gateway-api` is **not a real service**. It renders a public "hello world"
image (`nginxinc/nginx-unprivileged` — chosen specifically because it starts
cleanly under a non-root, read-only-root-filesystem, all-capabilities-dropped
securityContext; see the image comment in `values.yaml` for why that matters)
behind a `Deployment`/`Service` purely to prove the umbrella/subchart wiring.
Replace it with (or add alongside it) your real service's chart using the
recipe below.

## Add a new service's subchart (Phase 1 recipe)

Say you're adding the chart for `dispatcher`:

1. **Scaffold the subchart directory**, mirroring `gateway-api`:
   ```bash
   mkdir -p deploy/helm/sms-manager/charts/dispatcher/templates
   ```

2. **`charts/dispatcher/Chart.yaml`** — copy `gateway-api/Chart.yaml` and
   change `name`/`description`. Keep `apiVersion: v2`, `type: application`.

3. **`charts/dispatcher/values.yaml`** — copy `gateway-api/values.yaml` as a
   starting point. At minimum set:
   - `image.repository` / `image.tag` to the real GHCR image
     (`ghcr.io/amirandarabi/sms-manager/dispatcher`)
   - `service.port` (what the Service exposes) and `service.targetPort`
     (what the container actually listens on) to the service's real ports —
     keep them as two separate values, don't collapse them back to one. The
     gateway in particular terminates HTTP/3 and will not listen on 80.
   - `securityContext.runAsUser`/`podSecurityContext.fsGroup` to match
     whatever non-root uid your actual image runs as (check the image's
     Dockerfile/docs — don't just copy `101`, that's specific to
     nginx-unprivileged)
   - `probes.livenessPath` / `probes.readinessPath` to `/livez` / `/readyz`
     (ARCHITECTURE §14 — every service must expose these) and tune
     `probes.initialDelaySeconds`/`periodSeconds`/`timeoutSeconds`/
     `failureThreshold` for the service's real startup time
   - `resources.requests`/`limits` sized for the real workload
   - keep `securityContext`/`podSecurityContext` as non-root,
     `drop: ["ALL"]`, `readOnlyRootFilesystem: true`,
     `seccompProfile: {type: RuntimeDefault}` — do not relax these without an
     ADR. If your image needs scratch space beyond `/tmp`, add more
     `emptyDir` volumes rather than loosening `readOnlyRootFilesystem`.
   - `serviceAccount.automountServiceAccountToken` stays `false` unless the
     service actually calls the Kubernetes API

4. **Templates** — copy and adapt from `gateway-api/templates/`:
   - `_helpers.tpl` — rename the `define` blocks (`dispatcher.fullname`, etc.)
   - `deployment.yaml` / `service.yaml` — update as needed for the service's
     actual ports/env/volumes. Keep `containerPort` wired to
     `service.targetPort`, not `service.port`.
   - `serviceaccount.yaml` — keep (gated by
     `.Values.serviceAccount.create`)
   - `poddisruptionbudget.yaml` — keep (gated by
     `.Values.podDisruptionBudget.enabled`)
   - Add what the service actually needs beyond the placeholder, per
     ARCHITECTURE §14 and the engineering rules in `CLAUDE.md`:
     - `ServiceMonitor` (Prometheus scrape config)
     - `HPA` for APIs, or a KEDA `ScaledObject` (Kafka consumer lag) for
       consumers
     - `NetworkPolicy`
     - topology spread constraints (already wired via
       `.Values.topologySpreadConstraints` in the placeholder — populate it)
   - Secrets: reference `ExternalSecret`s only — never inline a Kubernetes
     `Secret` with real values in a committed chart.

5. **Register the subchart in the umbrella chart**:
   - Add a `dependencies:` entry in `deploy/helm/sms-manager/Chart.yaml`:
     ```yaml
     dependencies:
       - name: dispatcher
         version: 0.1.0
         repository: "file://charts/dispatcher"
         condition: dispatcher.enabled
     ```
     (the subchart under `charts/` is auto-discovered by Helm regardless of
     this entry — `helm lint` doesn't require it. It's needed so
     `condition: dispatcher.enabled` works: a `condition` can only gate a
     *declared* dependency, which is how `gateway-api.enabled` toggles the
     placeholder subchart on/off. No `helm dependency update`/network fetch
     is needed for local `file://` repos — Helm reads the chart directory
     directly.)
   - Add a top-level `dispatcher:` block to
     `deploy/helm/sms-manager/values.yaml`, namespacing that subchart's
     values (set `enabled: true`, override anything that differs from the
     subchart's own defaults).

6. **Per-environment values** — once real environments exist, add
   `values-dev.yaml` / `values-staging.yaml` / `values-prod.yaml` next to the
   umbrella `values.yaml` (or per-subchart, if only one service needs
   per-env overrides) and layer them with `helm template/install -f
   values.yaml -f values-<env>.yaml`. Prod-specific secrets still come from
   `ExternalSecrets`, never from a values file.

7. **Verify before opening a PR**:
   ```bash
   helm lint deploy/helm/sms-manager
   helm lint deploy/helm/sms-manager/charts/dispatcher   # standalone, optional
   helm template deploy/helm/sms-manager
   ```
   If `kubeconform` is available locally, also pipe the rendered output
   through it to catch schema errors (`helm template deploy/helm/sms-manager
   | kubeconform -strict -summary`).

   **`helm lint`/`helm template` only prove the YAML renders — they are
   structurally blind to whether the container can actually start under the
   securityContext you set** (readOnlyRootFilesystem + non-root uid +
   dropped capabilities is a common combination that silently breaks images
   not built for it — e.g. anything that tries to write to a root-owned
   directory or bind a privileged port). Before opening the PR, actually
   start the container under the same constraints, e.g.:
   ```bash
   docker run --rm --user <runAsUser>:<runAsUser> --read-only \
     --cap-drop ALL <repository>:<tag>
   ```
   or `helm install --dry-run=server`/a real install against your local dev
   cluster (Docker Desktop's built-in Kubernetes, context `docker-desktop` —
   never a shared/remote cluster from a local run) and confirm the pod
   reaches `Running`/`Ready`, not just that the manifest parses.

8. ArgoCD wiring (pointing an `Application`/`ApplicationSet` at this chart) is
   out of scope here — see `deploy/argocd/README.md` and the sibling story
   that adds the ApplicationSet.

## Local verification

```bash
helm lint deploy/helm/sms-manager
helm template deploy/helm/sms-manager
```

Both must pass with no errors before merging any change under
`deploy/helm/`. Neither command needs a live cluster. If you do want to test
against a cluster locally (e.g. `helm install --dry-run --debug`), target
whatever local dev cluster you use (e.g. Docker Desktop's built-in
Kubernetes, context `docker-desktop`) — never a shared/remote cluster from a
local run.
