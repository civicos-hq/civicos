package response

import "github.com/gin-gonic/gin"

// Success sends a standard success envelope.
func Success(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
	})
}

// Error sends a structured error — internal details never leak to the client.
func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"success": false,
		"code":    code,
		"message": message,
	})
}
