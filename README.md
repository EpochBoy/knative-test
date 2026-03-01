# knative-test

Knative Serving test function for EpochCloud serverless platform.

## What it does

A Fibonacci calculator that demonstrates:

- Scale-to-zero (Knative pod management)
- Cold start behavior with Linkerd sidecar
- JSON API endpoint (`POST /api/fib`)
- HTML dashboard (`GET /?n=42`)
- Health/readiness probes (`/healthz`, `/readyz`)

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/?n=42` | HTML page with Fibonacci calculator |
| POST | `/api/fib` | JSON API: `{"n": 42}` → `{"n": 42, "result": "267914296", "time_ms": 0}` |
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe |

## Deployment

Deployed as a Knative Service on EpochCloud:

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: fibonacci
  namespace: default
spec:
  template:
    spec:
      containers:
        - image: harbor.epoch.engineering/epochcloud/knative-test:0.1.0
          ports:
            - containerPort: 8080
```

URL: `https://fn-fibonacci-default.epoch.engineering`

## Build

Built automatically by Argo Workflows CI pipeline on push to main.
