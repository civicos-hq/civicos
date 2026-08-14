package communities

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Gin's radix router historically panicked on a static segment sharing a
// position with a wildcard. Registering /geocode alongside /:id is exactly
// that shape, and a panic here means the service does not start at all.
func TestGeocodeRouteDoesNotConflictWithIDRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/communities")
	g.GET("/:id", func(c *gin.Context) { c.String(200, "id:"+c.Param("id")) })
	g.GET("/geocode", func(c *gin.Context) { c.String(200, "geocode") })

	for _, tc := range []struct{ path, want string }{
		{"/communities/geocode", "geocode"},
		{"/communities/abc-123", "id:abc-123"},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != http.StatusOK || w.Body.String() != tc.want {
			t.Fatalf("%s → %d %q, want %q", tc.path, w.Code, w.Body.String(), tc.want)
		}
	}
}
