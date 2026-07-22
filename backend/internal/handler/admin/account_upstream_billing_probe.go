package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type upstreamBillingProbeEnabledRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

type upstreamBillingProbeBatchRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required"`
}

type upstreamBillingProbeSettingsRequest struct {
	Enabled                *bool `json:"enabled"`
	IntervalMinutes        *int  `json:"interval_minutes"`
	BalanceEnabled         *bool `json:"balance_enabled"`
	BalanceIntervalMinutes *int  `json:"balance_interval_minutes"`
}

func (h *AccountHandler) SetUpstreamBalanceProbeEnabled(c *gin.Context) {
	if h.upstreamBillingProbe == nil {
		response.ErrorFrom(c, service.ErrUpstreamBillingProbeUnavailable)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req upstreamBillingProbeEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.upstreamBillingProbe.SetBalanceAccountEnabled(c.Request.Context(), id, *req.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"account_id": id, "enabled": *req.Enabled})
}
func (h *AccountHandler) ProbeUpstreamBalance(c *gin.Context) {
	if h.upstreamBillingProbe == nil {
		response.ErrorFrom(c, service.ErrUpstreamBillingProbeUnavailable)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	snap, err := h.upstreamBillingProbe.ProbeBalanceAccount(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, service.UpstreamBalanceProbeResult{AccountID: id, Snapshot: snap})
}
func (h *AccountHandler) ProbeUpstreamBalanceBatch(c *gin.Context) {
	if h.upstreamBillingProbe == nil {
		response.ErrorFrom(c, service.ErrUpstreamBillingProbeUnavailable)
		return
	}
	var req upstreamBillingProbeBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.AccountIDs) == 0 || len(req.AccountIDs) > service.UpstreamBillingProbeMaxBatchSize {
		response.BadRequest(c, "account_ids must contain between 1 and 20 items")
		return
	}
	response.Success(c, gin.H{"results": h.upstreamBillingProbe.ProbeBalanceAccounts(c.Request.Context(), req.AccountIDs)})
}

func (h *AccountHandler) GetUpstreamBillingProbeSettings(c *gin.Context) {
	if h.upstreamBillingProbe == nil {
		response.ErrorFrom(c, service.ErrUpstreamBillingProbeUnavailable)
		return
	}
	settings, err := h.upstreamBillingProbe.GetSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *AccountHandler) UpdateUpstreamBillingProbeSettings(c *gin.Context) {
	if h.upstreamBillingProbe == nil {
		response.ErrorFrom(c, service.ErrUpstreamBillingProbeUnavailable)
		return
	}
	var req upstreamBillingProbeSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	current, err := h.upstreamBillingProbe.GetSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	if req.IntervalMinutes != nil {
		current.IntervalMinutes = *req.IntervalMinutes
	}
	if req.BalanceEnabled != nil {
		current.BalanceEnabled = *req.BalanceEnabled
	}
	if req.BalanceIntervalMinutes != nil {
		current.BalanceIntervalMinutes = *req.BalanceIntervalMinutes
	}
	if err := h.upstreamBillingProbe.UpdateSettings(c.Request.Context(), current); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	settings, err := h.upstreamBillingProbe.GetSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *AccountHandler) SetUpstreamBillingProbeEnabled(c *gin.Context) {
	if h.upstreamBillingProbe == nil {
		response.ErrorFrom(c, service.ErrUpstreamBillingProbeUnavailable)
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req upstreamBillingProbeEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.upstreamBillingProbe.SetAccountEnabled(c.Request.Context(), accountID, *req.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"account_id": accountID, "enabled": *req.Enabled})
}

func (h *AccountHandler) ProbeUpstreamBilling(c *gin.Context) {
	if h.upstreamBillingProbe == nil {
		response.ErrorFrom(c, service.ErrUpstreamBillingProbeUnavailable)
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	snapshot, err := h.upstreamBillingProbe.ProbeAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, service.UpstreamBillingProbeResult{AccountID: accountID, Snapshot: snapshot})
}

func (h *AccountHandler) ProbeUpstreamBillingBatch(c *gin.Context) {
	if h.upstreamBillingProbe == nil {
		response.ErrorFrom(c, service.ErrUpstreamBillingProbeUnavailable)
		return
	}
	var req upstreamBillingProbeBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if len(req.AccountIDs) == 0 || len(req.AccountIDs) > service.UpstreamBillingProbeMaxBatchSize {
		response.BadRequest(c, "account_ids must contain between 1 and 20 items")
		return
	}
	seen := make(map[int64]struct{}, len(req.AccountIDs))
	accountIDs := make([]int64, 0, len(req.AccountIDs))
	for _, accountID := range req.AccountIDs {
		if accountID <= 0 {
			response.BadRequest(c, "account_ids must contain positive IDs")
			return
		}
		if _, exists := seen[accountID]; exists {
			continue
		}
		seen[accountID] = struct{}{}
		accountIDs = append(accountIDs, accountID)
	}
	response.Success(c, gin.H{"results": h.upstreamBillingProbe.ProbeAccounts(c.Request.Context(), accountIDs)})
}
