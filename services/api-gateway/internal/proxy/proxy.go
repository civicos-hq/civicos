package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// NewReverseProxy builds a gin handler that proxies to targetURL,
// stripping the provided prefix from the path before forwarding.
func NewReverseProxy(targetURL, stripPrefix string) gin.HandlerFunc {
	return newProxy(targetURL, stripPrefix, 0)
}

// NewStreamingProxy is like NewReverseProxy but flushes immediately on every
// upstream write, so Server-Sent Events reach the browser without buffering.
func NewStreamingProxy(targetURL, stripPrefix string) gin.HandlerFunc {
	return newProxy(targetURL, stripPrefix, -1)
}

// healthClient bounds upstream health probes so a slow or sleeping service
// can't hold the admin dashboard's health panel open indefinitely.
var healthClient = &http.Client{Timeout: 10 * time.Second}

// NewHealthProxy returns a handler that probes the upstream's /health
// server-side. Browsers can't reach the internal services directly (and
// shouldn't need to know their URLs), so the gateway answers on their behalf.
func NewHealthProxy(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, targetURL+"/health", nil)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"status": "down"})
			return
		}
		resp, err := healthClient.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"status": "down"})
			return
		}
		defer resp.Body.Close()
		c.DataFromReader(resp.StatusCode, resp.ContentLength, "application/json", resp.Body, nil)
	}
}

func newProxy(targetURL, stripPrefix string, flushInterval time.Duration) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		panic("invalid proxy target: " + targetURL)
	}

	rp := httputil.NewSingleHostReverseProxy(target)
	rp.FlushInterval = flushInterval

	// SingleHostReverseProxy rewrites req.URL.Host but not req.Host, so the
	// outbound request keeps the client-facing Host header. Shared-edge
	// platforms (Render) dispatch by Host, which would bounce the request
	// straight back to this gateway. Pin Host to the upstream's own name.
	director := rp.Director
	rp.Director = func(req *http.Request) {
		director(req)
		req.Host = target.Host
	}

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
