package main

import (
    "log"
    "os"
    "time"
    "pr-reviewer/src/internal/storage"
    "pr-reviewer/src/internal/api/handlers"

    "github.com/gin-gonic/gin"
)

type Config struct {
    DBHost     string
    DBPort     string
    DBUser     string
    DBPassword string
    DBName     string
    Port       string
    ResetDB    bool
}

func loadConfig() Config {
    return Config{
        DBHost:     getEnv("DB_HOST", "postgres"),
        DBPort:     getEnv("DB_PORT", "5432"),
        DBUser:     getEnv("DB_USER", "postgres"),
        DBPassword: getEnv("DB_PASSWORD", "postgres"),
        DBName:     getEnv("DB_NAME", "pr_reviewer"),
        Port:       getEnv("PORT", "8080"),
        ResetDB:    getEnv("RESET_DB_ON_STARTUP", "true") == "true",
    }
}

func main() {
    config := loadConfig()

    connStr := "host=" + config.DBHost + " port=" + config.DBPort + " user=" + config.DBUser +
        " password=" + config.DBPassword + " dbname=" + config.DBName + " sslmode=disable"

    log.Printf("Connecting to database: %s@%s:%s/%s", config.DBUser, config.DBHost, config.DBPort, config.DBName)

    var db *database.DB
    var err error

    maxAttempts := 10
    for i := 0; i < maxAttempts; i++ {
        db, err = database.New(connStr)
        if err == nil {
            break
        }
        log.Printf("Failed to connect to database (attempt %d/%d): %v", i+1, maxAttempts, err)
        if i < maxAttempts-1 {
            waitTime := time.Duration(i+1) * 2 * time.Second
            log.Printf("Retrying in %v...", waitTime)
            time.Sleep(waitTime)
        }
    }

    if err != nil {
        log.Fatal("Failed to connect to database after", maxAttempts, "attempts:", err)
    }
    defer db.Close()

    log.Println("Successfully connected to database")

    teamHandler := handlers.NewTeamHandler(db)
    userHandler := handlers.NewUserHandler(db)
    prHandler := handlers.NewPRHandler(db)
	statsHandler := handlers.NewStatsHandler(db)

    router := gin.Default()

    router.Use(func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    })

    router.POST("/team/add", teamHandler.AddTeam)
    router.GET("/team/get", teamHandler.GetTeam)

    router.POST("/users/setIsActive", userHandler.SetIsActive)
    router.GET("/users/getReview", userHandler.GetReview)

    router.POST("/pullRequest/create", prHandler.CreatePR)
    router.POST("/pullRequest/merge", prHandler.MergePR)
    router.POST("/pullRequest/reassign", prHandler.Reassign)

	router.GET("/stats/system", statsHandler.GetSystemStats)
    router.GET("/stats/users", statsHandler.GetUserStats)
    router.GET("/stats/prs", statsHandler.GetPRStats)
    router.GET("/stats/top-reviewers", statsHandler.GetTopReviewers)

    log.Printf("Server starting on :%s", config.Port)
    if err := router.Run(":" + config.Port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}

func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}