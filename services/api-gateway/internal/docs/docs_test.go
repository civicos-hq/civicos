package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDocsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r)

	cases := []struct {
		path   string
		expect string
	}{
		{"/docs", "swagger-ui"},
		{"/docs/openapi/identity.yaml", "CivicOS Identity Service"},
		{"/docs/openapi/community.yaml", "CivicOS Community Service"},
		{"/docs/openapi/organization.yaml", "CivicOS Organization Service"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", tc.path, w.Code)
		}
		if !strings.Contains(w.Body.String(), tc.expect) {
			t.Fatalf("%s: response missing %q", tc.path, tc.expect)
		}
	}
}
