package handlers

import (
    "net/http"
    "strconv"
    "pr-reviewer/src/internal/storage"
    "pr-reviewer/src/internal/domain/models"

    "github.com/gin-gonic/gin"
)

type StatsHandler struct {
    db *database.DB
}

func NewStatsHandler(db *database.DB) *StatsHandler {
    return &StatsHandler{db: db}
}

// GetSystemStats возвращает общую статистику системы
// @Summary Получить общую статистику системы
// @Tags Statistics
// @Produce json
// @Success 200 {object} models.StatsResponse
// @Router /stats/system [get]
func (h *StatsHandler) GetSystemStats(c *gin.Context) {
    systemStats, err := h.db.GetSystemStats()
    if err != nil {
        c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        return
    }

    topReviewers, err := h.db.GetTopReviewers(5)
    if err != nil {
        c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        return
    }

    response := models.StatsResponse{
        SystemStats:  *systemStats,
        TopReviewers: topReviewers,
    }

    c.JSON(http.StatusOK, response)
}

// GetUserStats возвращает статистику по пользователям
// @Summary Получить статистику по пользователям
// @Tags Statistics
// @Produce json
// @Success 200 {object} models.StatsResponse
// @Router /stats/users [get]
func (h *StatsHandler) GetUserStats(c *gin.Context) {
    userStats, err := h.db.GetUserStats()
    if err != nil {
        c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        return
    }

    response := models.StatsResponse{
        UserStats: userStats,
    }

    c.JSON(http.StatusOK, response)
}

// GetPRStats возвращает статистику по PR
// @Summary Получить статистику по pull requests
// @Tags Statistics
// @Produce json
// @Success 200 {object} models.StatsResponse
// @Router /stats/prs [get]
func (h *StatsHandler) GetPRStats(c *gin.Context) {
    prStats, err := h.db.GetPRStats()
    if err != nil {
        c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        return
    }

    response := models.StatsResponse{
        PRStats: prStats,
    }

    c.JSON(http.StatusOK, response)
}

// GetTopReviewers возвращает топ ревьюверов
// @Summary Получить топ ревьюверов
// @Tags Statistics
// @Produce json
// @Param limit query int false "Количество возвращаемых записей" default(10)
// @Success 200 {object} models.StatsResponse
// @Router /stats/top-reviewers [get]
func (h *StatsHandler) GetTopReviewers(c *gin.Context) {
    limitStr := c.DefaultQuery("limit", "10")
    limit, err := strconv.Atoi(limitStr)
    if err != nil || limit <= 0 {
        limit = 10
    }
    if limit > 50 {
        limit = 50
    }

    topReviewers, err := h.db.GetTopReviewers(limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        return
    }

    response := models.StatsResponse{
        TopReviewers: topReviewers,
    }

    c.JSON(http.StatusOK, response)
}