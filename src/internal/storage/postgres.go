package database

import (
    "database/sql"
    "errors"
    "fmt"
    "os"
    "pr-reviewer/src/internal/domain/models"
    "time"

    _ "github.com/lib/pq"
)

var (
    ErrTeamExists     = errors.New("TEAM_EXISTS")
    ErrPRExists       = errors.New("PR_EXISTS")
    ErrPRMerged       = errors.New("PR_MERGED")
    ErrNotAssigned    = errors.New("NOT_ASSIGNED")
    ErrNoCandidate    = errors.New("NO_CANDIDATE")
    ErrNotFound       = errors.New("NOT_FOUND")
)

type DB struct {
    *sql.DB
}

func New(connectionString string) (*DB, error) {
    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        return nil, err
    }

    if err := db.Ping(); err != nil {
        return nil, err
    }

    if shouldResetDB() {
        if err := resetDatabase(db); err != nil {
            return nil, fmt.Errorf("failed to reset database: %v", err)
        }
        fmt.Println("Database reset completed")
    } else {
        if err := initTables(db); err != nil {
            return nil, err
        }
    }

    return &DB{db}, nil
}

func shouldResetDB() bool {
    return os.Getenv("RESET_DB_ON_STARTUP") == "true"
}

func resetDatabase(db *sql.DB) error {
    dropQueries := []string{
        "DROP TABLE IF EXISTS pr_reviewers CASCADE",
        "DROP TABLE IF EXISTS pull_requests CASCADE", 
        "DROP TABLE IF EXISTS users CASCADE",
        "DROP TABLE IF EXISTS teams CASCADE",
    }

    for _, query := range dropQueries {
        if _, err := db.Exec(query); err != nil {
            return fmt.Errorf("failed to drop table: %v", err)
        }
    }

    return initTables(db)
}

func initTables(db *sql.DB) error {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS teams (
            team_name VARCHAR(255) PRIMARY KEY,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,

        `CREATE TABLE IF NOT EXISTS users (
            user_id VARCHAR(255) PRIMARY KEY,
            username VARCHAR(255) NOT NULL,
            team_name VARCHAR(255) REFERENCES teams(team_name) ON DELETE CASCADE,
            is_active BOOLEAN DEFAULT TRUE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,

        `CREATE TABLE IF NOT EXISTS pull_requests (
            pull_request_id VARCHAR(255) PRIMARY KEY,
            pull_request_name VARCHAR(255) NOT NULL,
            author_id VARCHAR(255) REFERENCES users(user_id),
            status VARCHAR(50) DEFAULT 'OPEN',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            merged_at TIMESTAMP
        )`,

        `CREATE TABLE IF NOT EXISTS pr_reviewers (
            pull_request_id VARCHAR(255) REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
            reviewer_id VARCHAR(255) REFERENCES users(user_id),
            assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (pull_request_id, reviewer_id)
        )`,

        `CREATE INDEX IF NOT EXISTS idx_users_team_active ON users(team_name, is_active)`,
        `CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active)`,
        `CREATE INDEX IF NOT EXISTS idx_pr_status ON pull_requests(status)`,
        `CREATE INDEX IF NOT EXISTS idx_pr_author ON pull_requests(author_id)`,
        `CREATE INDEX IF NOT EXISTS idx_reviewers_pr_id ON pr_reviewers(pull_request_id)`,
        `CREATE INDEX IF NOT EXISTS idx_reviewers_user_id ON pr_reviewers(reviewer_id)`,
    }

    for _, query := range queries {
        if _, err := db.Exec(query); err != nil {
            return fmt.Errorf("failed to create table/index: %v", err)
        }
    }

    return nil
}

func (db *DB) CreateTeam(team models.Team) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    var exists bool
    err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", team.TeamName).Scan(&exists)
    if err != nil {
        return err
    }
    if exists {
        return ErrTeamExists
    }

    _, err = tx.Exec("INSERT INTO teams (team_name) VALUES ($1)", team.TeamName)
    if err != nil {
        return err
    }

    for _, member := range team.Members {
        _, err = tx.Exec(`
            INSERT INTO users (user_id, username, team_name, is_active) 
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (user_id) 
            DO UPDATE SET username = $2, team_name = $3, is_active = $4
        `, member.UserID, member.Username, team.TeamName, member.IsActive)
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}

func (db *DB) GetTeam(teamName string) (*models.Team, error) {
    var team models.Team
    team.TeamName = teamName

    rows, err := db.Query(`
        SELECT user_id, username, is_active 
        FROM users 
        WHERE team_name = $1
    `, teamName)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var member models.TeamMember
        if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
            return nil, err
        }
        team.Members = append(team.Members, member)
    }

    if len(team.Members) == 0 {
        return nil, ErrNotFound
    }

    return &team, nil
}

func (db *DB) SetUserActive(userID string, isActive bool) (*models.User, error) {
    var user models.User

    err := db.QueryRow(`
        UPDATE users SET is_active = $1 
        WHERE user_id = $2 
        RETURNING user_id, username, team_name, is_active
    `, isActive, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)

    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, err
    }

    return &user, nil
}

func (db *DB) CreatePullRequest(pr models.CreatePRRequest) (*models.PullRequest, error) {
    tx, err := db.Begin()
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    var exists bool
    err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)", pr.PullRequestID).Scan(&exists)
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, ErrPRExists
    }

    var teamName string
    var authorExists bool
    err = tx.QueryRow("SELECT team_name, EXISTS(SELECT 1 FROM users WHERE user_id = $1) FROM users WHERE user_id = $1", pr.AuthorID).Scan(&teamName, &authorExists)
    if err == sql.ErrNoRows || !authorExists {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, err
    }

    _, err = tx.Exec(`
        INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) 
        VALUES ($1, $2, $3, 'OPEN')
    `, pr.PullRequestID, pr.PullRequestName, pr.AuthorID)
    if err != nil {
        return nil, err
    }

    rows, err := tx.Query(`
        SELECT user_id FROM users 
        WHERE team_name = $1 
        AND user_id != $2 
        AND is_active = true 
        LIMIT 2
    `, teamName, pr.AuthorID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var reviewers []string
    for rows.Next() {
        var reviewerID string
        if err := rows.Scan(&reviewerID); err != nil {
            return nil, err
        }
        reviewers = append(reviewers, reviewerID)
    }

    for _, reviewerID := range reviewers {
        _, err = tx.Exec(`
            INSERT INTO pr_reviewers (pull_request_id, reviewer_id) 
            VALUES ($1, $2)
        `, pr.PullRequestID, reviewerID)
        if err != nil {
            return nil, err
        }
    }

    var result models.PullRequest
    var createdAt time.Time
    err = tx.QueryRow(`
        SELECT pull_request_id, pull_request_name, author_id, status, created_at 
        FROM pull_requests 
        WHERE pull_request_id = $1
    `, pr.PullRequestID).Scan(&result.PullRequestID, &result.PullRequestName, &result.AuthorID, &result.Status, &createdAt)
    if err != nil {
        return nil, err
    }
    result.CreatedAt = createdAt
    result.AssignedReviewers = reviewers

    if err := tx.Commit(); err != nil {
        return nil, err
    }

    return &result, nil
}

func (db *DB) MergePullRequest(prID string) (*models.PullRequest, error) {
    tx, err := db.Begin()
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    var currentStatus string
    err = tx.QueryRow("SELECT status FROM pull_requests WHERE pull_request_id = $1", prID).Scan(&currentStatus)
    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, err
    }

    if currentStatus == "MERGED" {
        return db.getPullRequest(tx, prID)
    }

    _, err = tx.Exec(`
        UPDATE pull_requests 
        SET status = 'MERGED', merged_at = CURRENT_TIMESTAMP 
        WHERE pull_request_id = $1
    `, prID)
    if err != nil {
        return nil, err
    }

    pr, err := db.getPullRequest(tx, prID)
    if err != nil {
        return nil, err
    }

    return pr, tx.Commit()
}

func (db *DB) ReassignReviewer(prID, oldUserID string) (*models.PullRequest, string, error) {
    tx, err := db.Begin()
    if err != nil {
        return nil, "", err
    }
    defer tx.Rollback()

    var status, authorID string
    err = tx.QueryRow(`
        SELECT status, author_id FROM pull_requests WHERE pull_request_id = $1
    `, prID).Scan(&status, &authorID)
    if err == sql.ErrNoRows {
        return nil, "", ErrNotFound
    }
    if err != nil {
        return nil, "", err
    }

    if status == "MERGED" {
        return nil, "", ErrPRMerged
    }

    var isAssigned bool
    err = tx.QueryRow(`
        SELECT EXISTS(SELECT 1 FROM pr_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2)
    `, prID, oldUserID).Scan(&isAssigned)
    if err != nil {
        return nil, "", err
    }
    if !isAssigned {
        return nil, "", ErrNotAssigned
    }

    var teamName string
    err = tx.QueryRow("SELECT team_name FROM users WHERE user_id = $1", authorID).Scan(&teamName)
    if err != nil {
        return nil, "", err
    }

    var newUserID string
    err = tx.QueryRow(`
        SELECT u.user_id FROM users u
        WHERE u.team_name = $1 
        AND u.user_id != $2 
        AND u.is_active = true 
        AND u.user_id NOT IN (
            SELECT reviewer_id FROM pr_reviewers WHERE pull_request_id = $3
        )
        LIMIT 1
    `, teamName, authorID, prID).Scan(&newUserID)

    if err == sql.ErrNoRows {
        return nil, "", ErrNoCandidate
    }
    if err != nil {
        return nil, "", err
    }

    _, err = tx.Exec(`
        UPDATE pr_reviewers 
        SET reviewer_id = $1 
        WHERE pull_request_id = $2 AND reviewer_id = $3
    `, newUserID, prID, oldUserID)
    if err != nil {
        return nil, "", err
    }

    pr, err := db.getPullRequest(tx, prID)
    if err != nil {
        return nil, "", err
    }

    if err := tx.Commit(); err != nil {
        return nil, "", err
    }

    return pr, newUserID, nil
}

func (db *DB) GetUserPullRequests(userID string) (*models.UserPRsResponse, error) {
    var exists bool
    err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", userID).Scan(&exists)
    if err != nil {
        return nil, err
    }
    if !exists {
        return nil, ErrNotFound
    }

    var response models.UserPRsResponse
    response.UserID = userID

    rows, err := db.Query(`
        SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
        FROM pull_requests pr
        JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
        WHERE prr.reviewer_id = $1
    `, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var pr models.PullRequestShort
        if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
            return nil, err
        }
        response.PullRequests = append(response.PullRequests, pr)
    }

    return &response, nil
}

func (db *DB) getPullRequest(tx *sql.Tx, prID string) (*models.PullRequest, error) {
    var pr models.PullRequest
    var createdAt time.Time
    var mergedAt sql.NullTime

    err := tx.QueryRow(`
        SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
        FROM pull_requests 
        WHERE pull_request_id = $1
    `, prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt)
    if err != nil {
        return nil, err
    }

    pr.CreatedAt = createdAt
    if mergedAt.Valid {
        pr.MergedAt = mergedAt.Time
    }

    rows, err := tx.Query(`
        SELECT reviewer_id FROM pr_reviewers WHERE pull_request_id = $1
    `, prID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var reviewerID string
        if err := rows.Scan(&reviewerID); err != nil {
            return nil, err
        }
        pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
    }

    return &pr, nil
}

func (db *DB) GetSystemStats() (*models.SystemStats, error) {
    var stats models.SystemStats

    err := db.QueryRow("SELECT COUNT(*) FROM teams").Scan(&stats.TotalTeams)
    if err != nil {
        return nil, err
    }

    err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
    if err != nil {
        return nil, err
    }

    err = db.QueryRow("SELECT COUNT(*) FROM pull_requests").Scan(&stats.TotalPRs)
    if err != nil {
        return nil, err
    }

    err = db.QueryRow("SELECT COUNT(*) FROM pull_requests WHERE status = 'OPEN'").Scan(&stats.TotalOpenPRs)
    if err != nil {
        return nil, err
    }

    err = db.QueryRow("SELECT COUNT(*) FROM pull_requests WHERE status = 'MERGED'").Scan(&stats.TotalMergedPRs)
    if err != nil {
        return nil, err
    }

    err = db.QueryRow("SELECT COUNT(*) FROM pr_reviewers").Scan(&stats.TotalReviews)
    if err != nil {
        return nil, err
    }

    if stats.TotalPRs > 0 {
        stats.AvgReviewsPerPR = float64(stats.TotalReviews) / float64(stats.TotalPRs)
    }

    return &stats, nil
}

func (db *DB) GetTopReviewers(limit int) ([]models.TopReviewer, error) {
    query := `
        SELECT u.user_id, u.username, COUNT(pr.reviewer_id) as review_count
        FROM users u
        LEFT JOIN pr_reviewers pr ON u.user_id = pr.reviewer_id
        GROUP BY u.user_id, u.username
        ORDER BY review_count DESC
        LIMIT $1
    `

    rows, err := db.Query(query, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var reviewers []models.TopReviewer
    for rows.Next() {
        var reviewer models.TopReviewer
        err := rows.Scan(&reviewer.UserID, &reviewer.Username, &reviewer.Count)
        if err != nil {
            return nil, err
        }
        reviewers = append(reviewers, reviewer)
    }

    return reviewers, nil
}

func (db *DB) GetUserStats() ([]models.UserStats, error) {
    query := `
        SELECT 
            u.user_id,
            u.username,
            u.team_name,
            u.is_active,
            COUNT(DISTINCT pr_author.pull_request_id) as prs_count,
            COUNT(DISTINCT pr_reviewers.pull_request_id) as reviews_count
        FROM users u
        LEFT JOIN pull_requests pr_author ON u.user_id = pr_author.author_id
        LEFT JOIN pr_reviewers pr_reviewers ON u.user_id = pr_reviewers.reviewer_id
        GROUP BY u.user_id, u.username, u.team_name, u.is_active
        ORDER BY reviews_count DESC, prs_count DESC
    `

    rows, err := db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var userStats []models.UserStats
    for rows.Next() {
        var stat models.UserStats
        err := rows.Scan(&stat.UserID, &stat.Username, &stat.TeamName, &stat.IsActive, &stat.PRsCount, &stat.ReviewsCount)
        if err != nil {
            return nil, err
        }
        userStats = append(userStats, stat)
    }

    return userStats, nil
}

func (db *DB) GetPRStats() ([]models.PRStats, error) {
    query := `
        SELECT 
            pr.pull_request_id,
            pr.pull_request_name,
            pr.author_id,
            u.username as author_name,
            pr.status,
            COUNT(prr.reviewer_id) as reviewers_count,
            pr.created_at,
            pr.merged_at
        FROM pull_requests pr
        LEFT JOIN users u ON pr.author_id = u.user_id
        LEFT JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
        GROUP BY pr.pull_request_id, pr.pull_request_name, pr.author_id, u.username, pr.status, pr.created_at, pr.merged_at
        ORDER BY pr.created_at DESC
    `

    rows, err := db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var prStats []models.PRStats
    for rows.Next() {
        var stat models.PRStats
        var mergedAt sql.NullTime
        err := rows.Scan(&stat.PullRequestID, &stat.PullRequestName, &stat.AuthorID, &stat.AuthorName, &stat.Status, &stat.ReviewersCount, &stat.CreatedAt, &mergedAt)
        if err != nil {
            return nil, err
        }
        if mergedAt.Valid {
            stat.MergedAt = mergedAt.Time
        }
        prStats = append(prStats, stat)
    }

    return prStats, nil
}