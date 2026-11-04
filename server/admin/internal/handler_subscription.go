package internal

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (a *AdminService) subscription(c *gin.Context) {
	config, found, err := a.keys.Subscription(c.Request.Context(), c.Param("token"))
	if err != nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}
	if !found {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Content-Type", "text/yaml; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="clash.yaml"`)
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.String(http.StatusOK, strings.TrimSpace(config)+"\n")
}
