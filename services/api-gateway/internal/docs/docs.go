// Package docs serves the Swagger UI and the embedded OpenAPI specs.
//
// The YAML files under ./openapi/ are byte-for-byte mirrors of docs/api/
// at the repo root. Go's embed directive can only pull files inside the
// module, so we keep a copy here. Re-sync from the repo root with:
//
//	cp docs/api/openapi-*.yaml services/api-gateway/internal/docs/openapi/
package docs

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed openapi/*.yaml
var specs embed.FS

// swaggerHTML renders Swagger UI with a picker for the three services.
// The UI is loaded from a CDN so this handler stays a single self-contained
// file — no npm dependency, no build step.
const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>CivicOS API — Swagger UI</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css">
<style>
  body { margin: 0; font-family: system-ui, sans-serif; }
  header {
    display: flex; align-items: center; gap: 1rem;
    padding: 0.75rem 1.25rem;
    background: #0f172a; color: #f8fafc;
    border-bottom: 1px solid #1e293b;
  }
  header h1 { font-size: 1rem; font-weight: 600; margin: 0; }
  header select {
    padding: 0.35rem 0.6rem;
    background: #1e293b; color: #f8fafc;
    border: 1px solid #334155; border-radius: 6px;
    font-size: 0.9rem; cursor: pointer;
  }
  #swagger-ui { max-width: 1200px; margin: 0 auto; }
</style>
</head>
<body>
<header>
  <h1>CivicOS API Documentation</h1>
  <label for="service-picker" style="opacity:0.7;font-size:0.85rem;">Service:</label>
  <select id="service-picker">
    <option value="/docs/openapi/identity.yaml">identity-service</option>
    <option value="/docs/openapi/community.yaml">community-service</option>
    <option value="/docs/openapi/organization.yaml">organization-service</option>
  </select>
</header>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
<script>
  const params = new URLSearchParams(window.location.search);
  const initial = params.get('spec') || '/docs/openapi/identity.yaml';
  const picker = document.getElementById('service-picker');
  picker.value = initial;

  let ui = SwaggerUIBundle({
    url: initial,
    dom_id: '#swagger-ui',
    deepLinking: true,
    persistAuthorization: true,
    presets: [SwaggerUIBundle.presets.apis],
  });

  picker.addEventListener('change', (e) => {
    const next = e.target.value;
    const url = new URL(window.location.href);
    url.searchParams.set('spec', next);
    window.history.replaceState({}, '', url.toString());
    ui.specActions.updateUrl(next);
    ui.specActions.download(next);
  });
</script>
</body>
</html>`

// RegisterRoutes wires the docs handlers into the gateway router.
func RegisterRoutes(r *gin.Engine) {
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})

	r.GET("/docs/openapi/identity.yaml", serveSpec("openapi/openapi-identity.yaml"))
	r.GET("/docs/openapi/community.yaml", serveSpec("openapi/openapi-community.yaml"))
	r.GET("/docs/openapi/organization.yaml", serveSpec("openapi/openapi-organization.yaml"))
}

func serveSpec(path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := specs.ReadFile(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "spec not found"})
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", data)
	}
}
