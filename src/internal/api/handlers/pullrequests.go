package handlers

import (
    "net/http"
    "pr-reviewer/src/internal/storage"
    "pr-reviewer/src/internal/domain/models"

    "github.com/gin-gonic/gin"
)

type PRHandler struct {
    db *database.DB
}

func NewPRHandler(db *database.DB) *PRHandler {
    return &PRHandler{db: db}
}

func (h *PRHandler) CreatePR(c *gin.Context) {
    var req models.CreatePRRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, createErrorResponse(models.CodeInvalidRequest, err.Error()))
        return
    }

    pr, err := h.db.CreatePullRequest(req)
    if err != nil {
        switch err {
        case database.ErrPRExists:
            c.JSON(http.StatusConflict, createErrorResponse(models.CodePRExists, "PR id already exists"))
        case database.ErrNotFound:
            c.JSON(http.StatusNotFound, createErrorResponse(models.CodeNotFound, "resource not found"))
        default:
            c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        }
        return
    }

    c.JSON(http.StatusCreated, gin.H{"pr": pr})
}

func (h *PRHandler) MergePR(c *gin.Context) {
    var req models.MergePRRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, createErrorResponse(models.CodeInvalidRequest, err.Error()))
        return
    }

    pr, err := h.db.MergePullRequest(req.PullRequestID)
    if err != nil {
        if err == database.ErrNotFound {
            c.JSON(http.StatusNotFound, createErrorResponse(models.CodeNotFound, "resource not found"))
        } else {
            c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        }
        return
    }

    c.JSON(http.StatusOK, gin.H{"pr": pr})
}

func (h *PRHandler) Reassign(c *gin.Context) {
    var req models.ReassignRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, createErrorResponse(models.CodeInvalidRequest, err.Error()))
        return
    }

    pr, newUserID, err := h.db.ReassignReviewer(req.PullRequestID, req.OldUserID)
    if err != nil {
        switch err {
        case database.ErrNotFound:
            c.JSON(http.StatusNotFound, createErrorResponse(models.CodeNotFound, "resource not found"))
        case database.ErrPRMerged:
            c.JSON(http.StatusConflict, createErrorResponse(models.CodePRMerged, "cannot reassign on merged PR"))
        case database.ErrNotAssigned:
            c.JSON(http.StatusConflict, createErrorResponse(models.CodeNotAssigned, "reviewer is not assigned to this PR"))
        case database.ErrNoCandidate:
            c.JSON(http.StatusConflict, createErrorResponse(models.CodeNoCandidate, "no active replacement candidate in team"))
        default:
            c.JSON(http.StatusInternalServerError, createErrorResponse(models.CodeInternalError, err.Error()))
        }
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "pr":          pr,
        "replaced_by": newUserID,
    })
}