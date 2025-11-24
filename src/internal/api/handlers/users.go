package handlers

import (
    "net/http"
    "pr-reviewer/src/internal/storage"
    "pr-reviewer/src/internal/domain/models"

    "github.com/gin-gonic/gin"
)

type UserHandler struct {
    db *database.DB
}

func NewUserHandler(db *database.DB) *UserHandler {
    return &UserHandler{db: db}
}

func (h *UserHandler) SetIsActive(c *gin.Context) {
    var req models.SetActiveRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, createErrorResponse(models.CodeInvalidRequest, err.Error()))
        return
    }

    user, err := h.db.SetUserActive(req.UserID, req.IsActive)
    if err != nil {
        if err == database.ErrNotFound {
            c.JSON(http.StatusNotFound, createErrorResponse(models.CodeNotFound, "resource not found"))
        } else {
            c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        }
        return
    }

    c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *UserHandler) GetReview(c *gin.Context) {
    userID := c.Query("user_id")
    if userID == "" {
        c.JSON(http.StatusBadRequest, createErrorResponse(models.CodeInvalidRequest, "user_id is required"))
        return
    }

    response, err := h.db.GetUserPullRequests(userID)
    if err != nil {
        if err == database.ErrNotFound {
            c.JSON(http.StatusNotFound, createErrorResponse(models.CodeNotFound, "resource not found"))
        } else {
            c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        }
        return
    }

    c.JSON(http.StatusOK, response)
}