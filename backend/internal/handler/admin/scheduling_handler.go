package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type SchedulingHandler struct {
	settingService *service.SettingService
	logService     *service.SchedulingLogService
}

func NewSchedulingHandler(settingService *service.SettingService) *SchedulingHandler {
	return &SchedulingHandler{
		settingService: settingService,
		logService:     service.DefaultSchedulingLogService,
	}
}

func (h *SchedulingHandler) SetLogService(logService *service.SchedulingLogService) {
	if logService != nil {
		h.logService = logService
	}
}

func (h *SchedulingHandler) GetConfig(c *gin.Context) {
	cfg, err := h.settingService.GetGatewaySchedulingConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gatewaySchedulingToDTO(cfg))
}

func (h *SchedulingHandler) UpdateConfig(c *gin.Context) {
	var req dto.GatewaySchedulingSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	base, err := h.settingService.GetGatewaySchedulingConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	cfg, err := applyGatewaySchedulingDTO(base, &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.settingService.UpdateGatewaySchedulingConfig(c.Request.Context(), cfg); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gatewaySchedulingToDTO(cfg))
}

func (h *SchedulingHandler) ListLogs(c *gin.Context) {
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			response.BadRequest(c, "limit must be a non-negative integer")
			return
		}
		if parsed > 0 {
			limit = parsed
		}
	}
	if limit > 200 {
		limit = 200
	}
	response.Success(c, h.logService.List(limit))
}
