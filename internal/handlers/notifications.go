package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/rbd-service/internal/models"
	"github.com/yourusername/rbd-service/internal/services"
)

type NotificationHandler struct {
	notificationService *services.NotificationService
}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{
		notificationService: services.NewNotificationService(),
	}
}

// TriggerNotification triggers a notification to a friend
func (h *NotificationHandler) TriggerNotification(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.TriggerNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.notificationService.TriggerNotification(c.Request.Context(), userID, req.TargetUserID)
	if err != nil {
		// Check if it's a cooldown error
		if strings.HasPrefix(err.Error(), "cooldown_active:") {
			parts := strings.Split(err.Error(), ":")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "cooldown_active",
				"availableAt": parts[1],
			})
			return
		}

		if err.Error() == "user_muted" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_muted"})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CheckCooldown checks if there's an active cooldown
func (h *NotificationHandler) CheckCooldown(c *gin.Context) {
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

	response, err := h.notificationService.CheckCooldown(c.Request.Context(), userID, friendUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}
