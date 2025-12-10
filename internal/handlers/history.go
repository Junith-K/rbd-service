package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/rbd-service/internal/repository"
)

type HistoryHandler struct {
	historyRepo *repository.HistoryRepository
}

func NewHistoryHandler() *HistoryHandler {
	return &HistoryHandler{
		historyRepo: repository.NewHistoryRepository(),
	}
}

// GetHistory retrieves history between current user and a friend
func (h *HistoryHandler) GetHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	friendUserID := c.Param("friendUserId")
	if friendUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "friendUserId is required"})
		return
	}

	// Parse pagination params
	page := 1
	limit := 50
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	history, total, err := h.historyRepo.GetHistoryBetweenUsers(c.Request.Context(), userID, friendUserID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   total,
	})
}
