package handlers

import (
    "net/http"
    "pr-reviewer/src/internal/storage"
    "pr-reviewer/src/internal/domain/models"

    "github.com/gin-gonic/gin"
)

type TeamHandler struct {
    db *database.DB
}

func NewTeamHandler(db *database.DB) *TeamHandler {
    return &TeamHandler{db: db}
}

func (h *TeamHandler) AddTeam(c *gin.Context) {
    var team models.Team
    if err := c.ShouldBindJSON(&team); err != nil {
        c.JSON(http.StatusBadRequest, createErrorResponse(models.CodeInvalidRequest, err.Error()))
        return
    }

    err := h.db.CreateTeam(team)
    if err != nil {
        switch err {
        case database.ErrTeamExists:
            c.JSON(http.StatusBadRequest, createErrorResponse(models.CodeTeamExists, "team_name already exists"))
        default:
            c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        }
        return
    }

    c.JSON(http.StatusCreated, gin.H{"team": team})
}

func (h *TeamHandler) GetTeam(c *gin.Context) {
    teamName := c.Query("team_name")
    if teamName == "" {
        c.JSON(http.StatusBadRequest, createErrorResponse(models.CodeInvalidRequest, "team_name is required"))
        return
    }

    team, err := h.db.GetTeam(teamName)
    if err != nil {
        if err == database.ErrNotFound {
            c.JSON(http.StatusNotFound, createErrorResponse(models.CodeNotFound, "resource not found"))
        } else {
            c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        }
        return
    }

    c.JSON(http.StatusOK, team)
}

func createErrorResponse(code models.ErrorCodes, message string) models.ErrorResponse {
    var resp models.ErrorResponse
    resp.Error.Code = code
    resp.Error.Message = message
    return resp
}