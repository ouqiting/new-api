package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/plugin"
	"github.com/gin-gonic/gin"
)

type TogglePluginRequest struct {
	PluginId string `json:"plugin_id" binding:"required"`
	Enabled  bool   `json:"enabled"`
}

func GetPlugins(c *gin.Context) {
	plugins := plugin.List()
	common.ApiSuccess(c, plugins)
}

func TogglePlugin(c *gin.Context) {
	var req TogglePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if err := plugin.SetEnabled(req.PluginId, req.Enabled); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, nil)
}

func ReloadPlugins(c *gin.Context) {
	if err := plugin.Reload(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
