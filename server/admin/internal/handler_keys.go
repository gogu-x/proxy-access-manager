package internal

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gogu-x/proxy-access-manager/admin/internal/key"
)

func (a *AdminService) registerKeyRoutes(r *gin.RouterGroup) {
	keys := r.Group("/keys", requireRole("admin"))
	keys.GET("", a.listKeys)
	keys.GET("/:id/usage", a.keyUsage)
	keys.POST("", a.createKey)
	keys.POST("/:id/revoke", a.changeKey)
	keys.POST("/:id/enable", a.changeKey)
	keys.POST("/:id/renew", a.changeKey)
	keys.PUT("/:id/bandwidth", a.updateBandwidth)
	keys.DELETE("/:id", a.deleteKey)
	r.POST("/reconcile", requireRole("admin"), a.reconcileKeys)
}
func (a *AdminService) updateBandwidth(c *gin.Context) {
	var req struct {
		UpMbps   int `json:"up_mbps"`
		DownMbps int `json:"down_mbps"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请求无效"})
		return
	}
	item, err := a.keys.UpdateBandwidth(c.Request.Context(), c.Param("id"), req.UpMbps, req.DownMbps)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, key.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": item})
}
func (a *AdminService) deleteKey(c *gin.Context) {
	if err := a.keys.Delete(c.Request.Context(), c.Param("id")); err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, key.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
func (a *AdminService) keyUsage(c *gin.Context) {
	granularity := c.DefaultQuery("granularity", "day")
	points, err := a.keys.UsageHistory(c.Request.Context(), c.Param("id"), granularity)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, key.ErrNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, key.ErrInvalidGranularity) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": points})
}
func (a *AdminService) listKeys(c *gin.Context) {
	items, err := a.keys.List(c.Request.Context())
	if err != nil {
		c.JSON(503, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true, "data": items})
}
func (a *AdminService) createKey(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Days    int    `json:"days"`
		QuotaGB int    `json:"quota_gb"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求无效"})
		return
	}
	item, err := a.keys.Create(c.Request.Context(), strings.TrimSpace(req.Name), req.Days, req.QuotaGB)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"ok": true, "data": item})
}
func (a *AdminService) changeKey(c *gin.Context) {
	var req struct {
		Days int `json:"days"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "请求无效"})
			return
		}
	}
	action := c.FullPath()
	action = action[strings.LastIndex(action, "/")+1:]
	item, err := a.keys.Change(c.Request.Context(), c.Param("id"), action, req.Days)
	if err != nil {
		status := 400
		if errors.Is(err, key.ErrNotFound) {
			status = 404
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true, "data": item})
}
func (a *AdminService) reconcileKeys(c *gin.Context) {
	if err := a.keys.Reconcile(c.Request.Context()); err != nil {
		c.JSON(503, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
func (a *AdminService) hysteriaAuth(c *gin.Context) {
	var req struct {
		Auth string `json:"auth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"ok": false, "id": ""})
		return
	}
	item, err := a.keys.Store.Get(c.Request.Context(), req.Auth)
	if err != nil || !item.IsActive() {
		c.JSON(200, gin.H{"ok": false, "id": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "id": item.ID})
}
