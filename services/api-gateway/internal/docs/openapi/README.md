# Embedded OpenAPI specs

These files are byte-for-byte mirrors of the canonical specs at
`docs/api/openapi-*.yaml` (repo root). They live here so `go:embed` can
bundle them into the api-gateway binary — Go's embed directive can't reach
files outside the module.

## Editing

Edit the canonical files in `docs/api/`, then re-sync:

```bash
cp docs/api/openapi-*.yaml services/api-gateway/internal/docs/openapi/
```

## What serves them

- `GET /docs` — Swagger UI with a service picker
- `GET /docs/openapi/identity.yaml`
- `GET /docs/openapi/community.yaml`
- `GET /docs/openapi/organization.yaml`

Handler: `services/api-gateway/internal/docs/docs.go`.
