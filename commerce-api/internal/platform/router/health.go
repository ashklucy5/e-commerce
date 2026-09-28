package router

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthDependencies struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func registerHealthRoutes(
	engine *gin.Engine,
	deps HealthDependencies,
) {
	engine.GET(
		"/health/live",
		func(c *gin.Context) {
			c.JSON(
				http.StatusOK,
				gin.H{
					"status": "ok",
				},
			)
		},
	)

	engine.GET(
		"/health/ready",
		func(c *gin.Context) {

			ctx, cancel := context.WithTimeout(
				c.Request.Context(),
				2*time.Second,
			)
			defer cancel()

			postgresStatus := "ok"
			redisStatus := "ok"

			if err := deps.DB.Ping(ctx); err != nil {
				postgresStatus = "unavailable"
			}

			if err := deps.Redis.Ping(ctx).Err(); err != nil {
				redisStatus = "unavailable"
			}

			if postgresStatus != "ok" ||
				redisStatus != "ok" {

				c.JSON(
					http.StatusServiceUnavailable,
					gin.H{
						"status": "not_ready",
						"dependencies": gin.H{
							"postgres": postgresStatus,
							"redis":    redisStatus,
						},
					},
				)

				return
			}

			c.JSON(
				http.StatusOK,
				gin.H{
					"status": "ready",
					"dependencies": gin.H{
						"postgres": postgresStatus,
						"redis":    redisStatus,
					},
				},
			)
		},
	)
}
