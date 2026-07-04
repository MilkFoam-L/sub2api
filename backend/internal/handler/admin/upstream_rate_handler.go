package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type UpstreamRateHandler struct {
	svc *service.UpstreamRateService
}

func NewUpstreamRateHandler(svc *service.UpstreamRateService) *UpstreamRateHandler {
	return &UpstreamRateHandler{svc: svc}
}

type upstreamRateSourceRequest struct {
	Name                string  `json:"name" binding:"required,max=100"`
	SourceType          string  `json:"source_type" binding:"required,oneof=sub2api newapi"`
	BaseURL             string  `json:"base_url" binding:"required,max=500"`
	AuthMode            string  `json:"auth_mode" binding:"omitempty,oneof=none bearer_token"`
	Token               *string `json:"token"`
	ClearToken          bool    `json:"clear_token"`
	RechargeMultiplier  float64 `json:"recharge_multiplier"`
	SyncIntervalSeconds int     `json:"sync_interval_seconds" binding:"omitempty,min=15,max=86400"`
	Enabled             *bool   `json:"enabled"`
	UseForScheduling    *bool   `json:"use_for_scheduling"`
}

type upstreamRateBindingRequest struct {
	SourceID         int64    `json:"source_id" binding:"required"`
	UpstreamGroupKey string   `json:"upstream_group_key" binding:"required,max=200"`
	TargetType       string   `json:"target_type" binding:"required,oneof=account group"`
	TargetID         int64    `json:"target_id" binding:"required"`
	Mode             string   `json:"mode" binding:"omitempty,oneof=first avg min max"`
	Offset           float64  `json:"offset"`
	ClampMin         *float64 `json:"clamp_min"`
	ClampMax         *float64 `json:"clamp_max"`
	Enabled          *bool    `json:"enabled"`
}

type upstreamRateSourceResponse struct {
	ID                  int64   `json:"id"`
	Name                string  `json:"name"`
	SourceType          string  `json:"source_type"`
	BaseURL             string  `json:"base_url"`
	AuthMode            string  `json:"auth_mode"`
	TokenConfigured     bool    `json:"token_configured"`
	RechargeMultiplier  float64 `json:"recharge_multiplier"`
	SyncIntervalSeconds int     `json:"sync_interval_seconds"`
	Enabled             bool    `json:"enabled"`
	UseForScheduling    bool    `json:"use_for_scheduling"`
	LastSyncAt          *string `json:"last_sync_at"`
	LastSyncStatus      string  `json:"last_sync_status"`
	LastError           string  `json:"last_error"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

func (h *UpstreamRateHandler) ListSources(c *gin.Context) {
	items, err := h.svc.ListSources(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]upstreamRateSourceResponse, 0, len(items))
	for _, item := range items {
		out = append(out, sourceToResponse(item))
	}
	response.Success(c, out)
}

func (h *UpstreamRateHandler) GetSource(c *gin.Context) {
	id, ok := parsePositiveID(c, "id")
	if !ok {
		return
	}
	item, err := h.svc.GetSource(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sourceToResponse(item))
}

func (h *UpstreamRateHandler) CreateSource(c *gin.Context) {
	var req upstreamRateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	useForScheduling := false
	if req.UseForScheduling != nil {
		useForScheduling = *req.UseForScheduling
	}
	token := ""
	if req.Token != nil {
		token = *req.Token
	}
	created, err := h.svc.CreateSource(c.Request.Context(), service.UpstreamRateCreateSourceParams{
		Name: req.Name, SourceType: req.SourceType, BaseURL: req.BaseURL, AuthMode: req.AuthMode, Token: token,
		RechargeMultiplier: req.RechargeMultiplier, SyncIntervalSeconds: req.SyncIntervalSeconds, Enabled: enabled, UseForScheduling: useForScheduling,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, sourceToResponse(created))
}

func (h *UpstreamRateHandler) UpdateSource(c *gin.Context) {
	id, ok := parsePositiveID(c, "id")
	if !ok {
		return
	}
	var req upstreamRateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	updated, err := h.svc.UpdateSource(c.Request.Context(), id, service.UpstreamRateUpdateSourceParams{
		Name: &req.Name, SourceType: &req.SourceType, BaseURL: &req.BaseURL, AuthMode: &req.AuthMode, Token: req.Token, ClearToken: req.ClearToken,
		RechargeMultiplier: &req.RechargeMultiplier, SyncIntervalSeconds: &req.SyncIntervalSeconds, Enabled: req.Enabled, UseForScheduling: req.UseForScheduling,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, sourceToResponse(updated))
}

func (h *UpstreamRateHandler) DeleteSource(c *gin.Context) {
	id, ok := parsePositiveID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteSource(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *UpstreamRateHandler) TestSource(c *gin.Context) {
	id, ok := parsePositiveID(c, "id")
	if !ok {
		return
	}
	result, err := h.svc.TestSource(c.Request.Context(), id)
	if err != nil {
		response.Success(c, result)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamRateHandler) SyncSource(c *gin.Context) {
	id, ok := parsePositiveID(c, "id")
	if !ok {
		return
	}
	result, err := h.svc.SyncSource(c.Request.Context(), id)
	if err != nil {
		response.Success(c, result)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamRateHandler) ListBindings(c *gin.Context) {
	items, err := h.svc.ListBindings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *UpstreamRateHandler) CreateBinding(c *gin.Context) {
	var req upstreamRateBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item, err := h.svc.CreateBinding(c.Request.Context(), bindingParamsFromRequest(req, enabled))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *UpstreamRateHandler) UpdateBinding(c *gin.Context) {
	id, ok := parsePositiveID(c, "id")
	if !ok {
		return
	}
	var req upstreamRateBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item, err := h.svc.UpdateBinding(c.Request.Context(), id, bindingParamsFromRequest(req, enabled))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, item)
}

func (h *UpstreamRateHandler) DeleteBinding(c *gin.Context) {
	id, ok := parsePositiveID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeleteBinding(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *UpstreamRateHandler) Overview(c *gin.Context) {
	items, err := h.svc.Overview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *UpstreamRateHandler) Health(c *gin.Context) {
	items, err := h.svc.Health(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *UpstreamRateHandler) LatestSnapshots(c *gin.Context) {
	id, ok := parsePositiveID(c, "id")
	if !ok {
		return
	}
	items, err := h.svc.LatestSnapshots(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func bindingParamsFromRequest(req upstreamRateBindingRequest, enabled bool) service.UpstreamRateBindingParams {
	return service.UpstreamRateBindingParams{SourceID: req.SourceID, UpstreamGroupKey: req.UpstreamGroupKey, TargetType: req.TargetType, TargetID: req.TargetID, Mode: req.Mode, Offset: req.Offset, ClampMin: req.ClampMin, ClampMax: req.ClampMax, Enabled: enabled}
}

func sourceToResponse(source *service.UpstreamRateSource) upstreamRateSourceResponse {
	resp := upstreamRateSourceResponse{ID: source.ID, Name: source.Name, SourceType: source.SourceType, BaseURL: source.BaseURL, AuthMode: source.AuthMode, TokenConfigured: source.TokenConfigured, RechargeMultiplier: source.RechargeMultiplier, SyncIntervalSeconds: source.SyncIntervalSeconds, Enabled: source.Enabled, UseForScheduling: source.UseForScheduling, LastSyncStatus: source.LastSyncStatus, LastError: source.LastError, CreatedAt: formatTimeRFC3339(source.CreatedAt), UpdatedAt: formatTimeRFC3339(source.UpdatedAt)}
	if source.LastSyncAt != nil {
		v := source.LastSyncAt.UTC().Format(time.RFC3339)
		resp.LastSyncAt = &v
	}
	return resp
}

func parsePositiveID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid id")
		return 0, false
	}
	return id, true
}

func formatTimeRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
