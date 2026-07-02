package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: rdb,
	}
}

// HealthCheck checks the health of database and redis connections
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "UP"
	redisStatus := "UP"
	hasError := false

	// Check DB
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			dbStatus = "DOWN"
			hasError = true
		}
	} else {
		dbStatus = "NOT_CONFIGURED"
		hasError = true
	}

	// Check Redis
	if h.redis != nil {
		if _, err := h.redis.Ping(ctx).Result(); err != nil {
			redisStatus = "DOWN"
			hasError = true
		}
	} else {
		redisStatus = "NOT_CONFIGURED"
		hasError = true
	}

	status := http.StatusOK
	if hasError {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status": dbStatus, // main status mirrors DB
		"details": gin.H{
			"database": dbStatus,
			"redis":    redisStatus,
		},
	})
}
