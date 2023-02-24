package controller

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/wy0917/jlink_dock/model"
	"net/http"
	"os"
)

// GetInfo godoc
//
//	@Summary		Get information
//	@Description	Display the information for current node
//	@Tags			info
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.Config
//	@Failure		500	{object}	model.APIError
//	@Router			/info [get]
func (c *Controller) GetInfo(ctx *gin.Context) {
	config := ctx.MustGet("config").(*model.Config)
	bodyJSON, _ := json.Marshal(config)
	ctx.Data(http.StatusOK, "application/json", bodyJSON)
}

// GetLog godoc
//
//	@Summary		Get running log
//	@Description	Display the log for current node
//	@Tags			log
//	@Accept			json
//	@Produce		plain
//	@Success		200
//	@Failure		500	{object}	model.APIError
//	@Router			/log [get]
func (c *Controller) GetLog(ctx *gin.Context) {
	logFile := ctx.MustGet("log_path").(string)
	dat, err := os.ReadFile(logFile)
	if err != nil {
		ctx.Error(err)
	}
	ctx.Data(http.StatusOK, "text/plain", dat)
}
