package admin

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/dimitarkovachev/wedding/internal/store"
)

// AdminStore defines the store operations needed by the admin handler.
type AdminStore interface {
	GetAllInvites(ctx context.Context) (map[string]store.InviteRecord, error)
	ReplaceAllInvites(ctx context.Context, invites map[string]store.InviteRecord) error
}

type Handler struct {
	store AdminStore
}

func NewHandler(s AdminStore) *Handler {
	return &Handler{store: s}
}

var _ ServerInterface = (*Handler)(nil)

func (h *Handler) GetAdminInvites(c *gin.Context) {
	invites, err := h.store.GetAllInvites(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("failed to get all invites")
		c.JSON(http.StatusInternalServerError, Error{Message: "internal error"})
		return
	}

	c.JSON(http.StatusOK, invites)
}

func (h *Handler) PutAdminInvites(c *gin.Context) {
	var invites map[string]store.InviteRecord
	if err := c.ShouldBindJSON(&invites); err != nil {
		c.JSON(http.StatusBadRequest, Error{Message: "invalid request body"})
		return
	}

	if err := h.store.ReplaceAllInvites(c.Request.Context(), invites); err != nil {
		log.WithError(err).Error("failed to replace invites")
		c.JSON(http.StatusInternalServerError, Error{Message: "internal error"})
		return
	}

	c.JSON(http.StatusOK, invites)
}
