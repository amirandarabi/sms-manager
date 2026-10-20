# api/proto

Protobuf contracts — the source of truth for every service's public API (REST via Vanguard, gRPC, Connect over HTTP/3).
Holds `buf.yaml`, `buf.gen.yaml`, and per-domain `.proto` files (e.g. `sms/v1`). Changes here go through `buf lint` and `buf breaking` in CI before code generation (`task gen`). Filled in by SMM-3.
