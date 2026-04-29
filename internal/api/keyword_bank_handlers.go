package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/repositories"
)

// Note: handleGetKeywordBanks is defined in autolinking_routes.go

// handleGetKeywordBank returns a single keyword bank by ID
func (r *Router) handleGetKeywordBank(c *gin.Context) {
	ctx := c.Request.Context()
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}
	
	if r.handler == nil || r.handler.keywordBankRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Service not available",
		})
		return
	}
	
	bank, err := r.handler.keywordBankRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Keyword bank not found",
		})
		return
	}
	
	c.JSON(http.StatusOK, bank)
}

// handleCreateKeywordBank creates a new keyword bank
func (r *Router) handleCreateKeywordBank(c *gin.Context) {
	ctx := c.Request.Context()
	
	var input struct {
		Name        string   `json:"name" binding:"required"`
		URL         string   `json:"url" binding:"required,url"`
		Keywords    []string `json:"keywords" binding:"required,min=1"`
		Description string   `json:"description"`
		IsActive    bool     `json:"is_active"`
	}
	
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid input: " + err.Error(),
		})
		return
	}
	
	if r.handler == nil || r.handler.keywordBankRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Service not available",
		})
		return
	}
	
	bank := &repositories.KeywordBank{
		Name:        input.Name,
		URL:         input.URL,
		Keywords:    input.Keywords,
		Description: input.Description,
		IsActive:    input.IsActive,
	}
	
	createdBank, err := r.handler.keywordBankRepo.Create(ctx, bank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create keyword bank: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, createdBank)
}

// handleUpdateKeywordBank updates an existing keyword bank
func (r *Router) handleUpdateKeywordBank(c *gin.Context) {
	ctx := c.Request.Context()
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}
	
	var input struct {
		Name        string   `json:"name" binding:"required"`
		URL         string   `json:"url" binding:"required,url"`
		Keywords    []string `json:"keywords" binding:"required,min=1"`
		Description string   `json:"description"`
		IsActive    bool     `json:"is_active"`
	}
	
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid input: " + err.Error(),
		})
		return
	}
	
	if r.handler == nil || r.handler.keywordBankRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Service not available",
		})
		return
	}
	
	bank := &repositories.KeywordBank{
		ID:          id,
		Name:        input.Name,
		URL:         input.URL,
		Keywords:    input.Keywords,
		Description: input.Description,
		IsActive:    input.IsActive,
	}
	
	err = r.handler.keywordBankRepo.Update(ctx, bank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update keyword bank: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, bank)
}

// handlePatchKeywordBank partially updates a keyword bank (e.g., toggle active status)
func (r *Router) handlePatchKeywordBank(c *gin.Context) {
	ctx := c.Request.Context()
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}
	
	var input struct {
		IsActive *bool `json:"is_active"`
	}
	
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid input: " + err.Error(),
		})
		return
	}
	
	if r.handler == nil || r.handler.keywordBankRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Service not available",
		})
		return
	}
	
	// Get existing bank
	bank, err := r.handler.keywordBankRepo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Keyword bank not found",
		})
		return
	}
	
	// Update only provided fields
	if input.IsActive != nil {
		bank.IsActive = *input.IsActive
	}
	
	err = r.handler.keywordBankRepo.Update(ctx, bank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update keyword bank: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, bank)
}

// handleDeleteKeywordBank deletes a keyword bank
func (r *Router) handleDeleteKeywordBank(c *gin.Context) {
	ctx := c.Request.Context()
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID",
		})
		return
	}
	
	if r.handler == nil || r.handler.keywordBankRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Service not available",
		})
		return
	}
	
	err = r.handler.keywordBankRepo.Delete(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete keyword bank: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Keyword bank deleted successfully",
	})
}
