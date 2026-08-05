package server

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

const siteRateLimitPerMinute = 300

func siteRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:               siteRateLimitPerMinute,
		Expiration:        time.Minute,
		LimiterMiddleware: limiter.SlidingWindow{},
		LimitReached: func(c fiber.Ctx) error {
			if strings.HasPrefix(c.Path(), "/api/v1/") {
				return sendError(c, fiber.StatusTooManyRequests, "rate_limited", "too many requests")
			}
			return c.SendStatus(fiber.StatusTooManyRequests)
		},
	})
}
