package models

import (
	"time"
)

type ErrorCodes string

const (
	CodeTeamExists     ErrorCodes = "TEAM_EXISTS"
	CodePRExists       ErrorCodes = "PR_EXISTS"
	CodePRMerged       ErrorCodes = "PR_MERGED"
	CodeNotAssigned    ErrorCodes = "NOT_ASSIGNED"
	CodeNoCandidate    ErrorCodes = "NO_CANDIDATE"
	CodeNotFound       ErrorCodes = "NOT_FOUND"
	CodeInvalidRequest ErrorCodes = "INVALID_REQUEST"
	CodeInternalError  ErrorCodes = "INTERNAL_ERROR"
)

type ErrorResponse struct {
	Error struct {
		Code    ErrorCodes `json:"code"`
		Message string     `json:"message"`
	} `json:"error"`
}

type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Team struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type PullRequest struct {
	PullRequestID     string    `json:"pull_request_id"`
	PullRequestName   string    `json:"pull_request_name"`
	AuthorID          string    `json:"author_id"`
	Status            string    `json:"status"`
	AssignedReviewers []string  `json:"assigned_reviewers"`
	CreatedAt         time.Time `json:"createdAt,omitempty"`
	MergedAt          time.Time `json:"mergedAt,omitempty"`
}

type PullRequestShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

type SetActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type ReassignRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

type UserPRsResponse struct {
	UserID       string             `json:"user_id"`
	PullRequests []PullRequestShort `json:"pull_requests"`
}

type UserStats struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	TeamName     string `json:"team_name"`
	IsActive     bool   `json:"is_active"`
	PRsCount     int    `json:"prs_count"`
	ReviewsCount int    `json:"reviews_count"`
}

type PRStats struct {
	PullRequestID   string    `json:"pull_request_id"`
	PullRequestName string    `json:"pull_request_name"`
	AuthorID        string    `json:"author_id"`
	AuthorName      string    `json:"author_name"`
	Status          string    `json:"status"`
	ReviewersCount  int       `json:"reviewers_count"`
	CreatedAt       time.Time `json:"created_at"`
	MergedAt        time.Time `json:"merged_at,omitempty"`
}

type SystemStats struct {
	TotalTeams      int     `json:"total_teams"`
	TotalUsers      int     `json:"total_users"`
	TotalPRs        int     `json:"total_prs"`
	TotalOpenPRs    int     `json:"total_open_prs"`
	TotalMergedPRs  int     `json:"total_merged_prs"`
	TotalReviews    int     `json:"total_reviews"`
	AvgReviewsPerPR float64 `json:"avg_reviews_per_pr"`
}

type TopReviewer struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Count    int    `json:"count"`
}

type StatsResponse struct {
	SystemStats  SystemStats   `json:"system_stats"`
	TopReviewers []TopReviewer `json:"top_reviewers,omitempty"`
	UserStats    []UserStats   `json:"user_stats,omitempty"`
	PRStats      []PRStats     `json:"pr_stats,omitempty"`
}
