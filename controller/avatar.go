package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUserAvatarImage serves the locally stored avatar image for a user.
// Public by design (like Gravatar): the response is only an avatar image,
// and being auth-free lets browsers cache it across sessions.
func GetUserAvatarImage(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		c.Status(http.StatusBadRequest)
		return
	}

	avatar, err := model.GetUserAvatar(userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusNotFound)
		} else {
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	etag := `"` + avatar.Hash + `"`
	c.Header("ETag", etag)
	c.Header("Cache-Control", "public, max-age=86400")
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}

	mimeType := avatar.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	c.Data(http.StatusOK, mimeType, avatar.Data)
}
