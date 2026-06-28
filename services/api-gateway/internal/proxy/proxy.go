package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// NewReverseProxy builds a gin handler that proxies to targetURL,
// stripping the provided prefix from the path before forwarding.
func NewReverseProxy(targetURL, stripPrefix string) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic("invalid proxy target: " + targetURL)
	}

	rp := httputil.NewSingleHostReverseProxy(target)

	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"UPSTREAM_ERROR","message":"Upstream service unavailable"}}`))
	}

	return func(c *gin.Context) {
		if stripPrefix != "" {
			c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, stripPrefix)
			if c.Request.URL.Path == "" {
				c.Request.URL.Path = "/"
			}
		}
		c.Request.Header.Del("Origin")
		rp.ServeHTTP(c.Writer, c.Request)
	}
}
