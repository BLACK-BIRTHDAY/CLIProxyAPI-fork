package management

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/magpie"
	log "github.com/sirupsen/logrus"
)

func (h *Handler) effectiveProxyURL() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cfg == nil {
		return ""
	}
	return strings.TrimSpace(h.cfg.ProxyURL)
}

// GetMagpieStatus reports process, install, configuration, and quota metrics for MAGPIE.
func (h *Handler) GetMagpieStatus(c *gin.Context) {
	proxyURL := h.effectiveProxyURL()
	dataDir := magpie.ResolveDataDir("")
	status, err := magpie.GetFullStatus(c.Request.Context(), dataDir, proxyURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

// StartMagpie launches the managed AIStudio2API process.
func (h *Handler) StartMagpie(c *gin.Context) {
	proxyURL := h.effectiveProxyURL()
	dataDir := magpie.ResolveDataDir("")
	cfg, err := magpie.LoadGatewayConfig(dataDir, proxyURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	st, errStart := magpie.StartBackend(c.Request.Context(), dataDir, cfg, proxyURL)
	if errStart != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errStart.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  st,
	})
}

// StopMagpie stops the managed AIStudio2API process.
func (h *Handler) StopMagpie(c *gin.Context) {
	dataDir := magpie.ResolveDataDir("")
	if err := magpie.StopBackend(c.Request.Context(), dataDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RestartMagpie stops and restarts the managed AIStudio2API process.
func (h *Handler) RestartMagpie(c *gin.Context) {
	proxyURL := h.effectiveProxyURL()
	dataDir := magpie.ResolveDataDir("")
	cfg, err := magpie.LoadGatewayConfig(dataDir, proxyURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	st, errRestart := magpie.RestartBackend(c.Request.Context(), dataDir, cfg, proxyURL)
	if errRestart != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errRestart.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  st,
	})
}

// GetMagpieQuota returns accounts, cooldowns, and status from the MAGPIE daemon.
func (h *Handler) GetMagpieQuota(c *gin.Context) {
	dataDir := magpie.ResolveDataDir("")
	quota, err := magpie.GetQuotaSummary(c.Request.Context(), dataDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, quota)
}

// ToggleMagpieAccount enables or disables a MAGPIE account.
func (h *Handler) ToggleMagpieAccount(c *gin.Context) {
	accountID := c.Param("accountId")
	if strings.TrimSpace(accountID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountId is required"})
		return
	}

	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enabled boolean field is required"})
		return
	}

	proxyURL := h.effectiveProxyURL()
	dataDir := magpie.ResolveDataDir("")
	cfg, err := magpie.LoadGatewayConfig(dataDir, proxyURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	res, errToggle := magpie.ToggleAccount(c.Request.Context(), dataDir, cfg, proxyURL, accountID, *req.Enabled)
	if errToggle != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errToggle.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeleteMagpieAccount deletes a MAGPIE account and its storage state.
func (h *Handler) DeleteMagpieAccount(c *gin.Context) {
	accountID := c.Param("accountId")
	if strings.TrimSpace(accountID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountId is required"})
		return
	}
	dataDir := magpie.ResolveDataDir("")
	if err := magpie.DeleteAccount(c.Request.Context(), dataDir, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// LoginMagpieAccount triggers interactive Camoufox browser login to add a new MAGPIE account.
func (h *Handler) LoginMagpieAccount(c *gin.Context) {
	proxyURL := h.effectiveProxyURL()
	dataDir := magpie.ResolveDataDir("")
	cfg, err := magpie.LoadGatewayConfig(dataDir, proxyURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	res, errLogin := magpie.LoginNewAccount(c.Request.Context(), dataDir, cfg, proxyURL)
	if errLogin != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errLogin.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// ReloginMagpieAccount triggers interactive Camoufox browser re-login for an existing MAGPIE account.
func (h *Handler) ReloginMagpieAccount(c *gin.Context) {
	accountID := c.Param("accountId")
	if strings.TrimSpace(accountID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountId is required"})
		return
	}

	proxyURL := h.effectiveProxyURL()
	dataDir := magpie.ResolveDataDir("")
	cfg, err := magpie.LoadGatewayConfig(dataDir, proxyURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	res, errRelogin := magpie.ReloginAccount(c.Request.Context(), dataDir, cfg, proxyURL, accountID)
	if errRelogin != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errRelogin.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// ExportMagpieAccounts exports all accounts into an Account State Bundle.
func (h *Handler) ExportMagpieAccounts(c *gin.Context) {
	dataDir := magpie.ResolveDataDir("")
	bundle, err := magpie.ExportAccountsBundle(dataDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=magpie-accounts-%s.json", time.Now().Format("20060102-150405")))
	c.JSON(http.StatusOK, bundle)
}

// ImportMagpieAccounts imports an Account State Bundle into <data_dir>/auth/.
func (h *Handler) ImportMagpieAccounts(c *gin.Context) {
	var bundle magpie.MagpieAccountBundle
	if err := c.ShouldBindJSON(&bundle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid bundle JSON: %v", err)})
		return
	}

	proxyURL := h.effectiveProxyURL()
	dataDir := magpie.ResolveDataDir("")
	cfg, err := magpie.LoadGatewayConfig(dataDir, proxyURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result, errImport := magpie.ImportAccountsBundle(c.Request.Context(), dataDir, &bundle, cfg, proxyURL)
	if errImport != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errImport.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// SaveMagpieConfig updates and syncs MAGPIE configuration.
func (h *Handler) SaveMagpieConfig(c *gin.Context) {
	proxyURL := h.effectiveProxyURL()
	dataDir := magpie.ResolveDataDir("")
	curCfg, err := magpie.LoadGatewayConfig(dataDir, proxyURL)
	if err != nil {
		curCfg = magpie.NewDefaultConfig()
	}

	rawBody, errRead := c.GetRawData()
	if errRead != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	if errUnmarshal := json.Unmarshal(rawBody, curCfg); errUnmarshal != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid config JSON: %v", errUnmarshal)})
		return
	}

	if errSave := magpie.SaveGatewayConfig(dataDir, curCfg); errSave != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save gateway config: %v", errSave)})
		return
	}

	if errApply := magpie.ApplyMagpieConfig(c.Request.Context(), dataDir, curCfg, proxyURL); errApply != nil {
		log.Warnf("[MAGPIE] ApplyMagpieConfig encountered warning/error: %v", errApply)
	}

	c.JSON(http.StatusOK, curCfg)
}
