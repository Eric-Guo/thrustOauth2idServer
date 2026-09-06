package routers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-dev-frame/sponge/pkg/gin/middleware/auth"
)

const sessionErrorKey = "error"

// VerifyRailsSessionUserIDIs returns a middleware that verifies the rails session
// contains a warden user id.
func VerifyRailsSessionUserIDIs(userID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("rails_session")
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "rails_session missing"})
			return
		}
		session, ok := v.(map[string]any)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "invalid rails_session"})
			return
		}
		uidVal, ok := auth.UserIDFromSession(session)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "user id not found in session"})
			return
		}
		var uid int64
		switch v := uidVal.(type) {
		case int64:
			uid = v
		case int:
			uid = int64(v)
		case float64:
			uid = int64(v)
		case string:
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "invalid user id in session"})
				return
			}
			uid = parsed
		default:
			c.AbortWithStatusJSON(401, gin.H{sessionErrorKey: "invalid user id type in session"})
			return
		}
		if uid != userID {
			c.AbortWithStatusJSON(403, gin.H{sessionErrorKey: "forbidden"})
			return
		}
		c.Next()
	}
}
