package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	log "github.com/sirupsen/logrus"

	"github.com/dimitarkovachev/wedding/internal/store"
)

// Handler implements the generated ServerInterface.
type Handler struct {
	store store.InviteStore
}

func NewHandler(s store.InviteStore) *Handler {
	return &Handler{store: s}
}

var _ ServerInterface = (*Handler)(nil)

func (h *Handler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

func (h *Handler) GetInvite(c *gin.Context, id openapi_types.UUID) {
	idStr := id.String()
	logger := log.WithField("invite_id", idStr)

	rec, err := h.store.GetInvite(c.Request.Context(), idStr)
	if err != nil {
		logger.WithError(err).Error("failed to get invite")
		c.JSON(http.StatusInternalServerError, Error{Message: "internal error"})
		return
	}
	if rec == nil {
		c.JSON(http.StatusNotFound, Error{Message: "invite not found"})
		return
	}

	logger.Info("invite viewed")
	c.JSON(http.StatusOK, recordToInvite(rec))
}

func (h *Handler) PutInvite(c *gin.Context, id openapi_types.UUID) {
	idStr := id.String()
	logger := log.WithField("invite_id", idStr)

	var body InviteUpdate
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, Error{Message: "invalid request body"})
		return
	}

	if !body.IsAccepted {
		c.JSON(http.StatusBadRequest, Error{Message: "only accepted=true updates are allowed"})
		return
	}

	var additional []string
	if body.Additional != nil {
		additional = *body.Additional
	}

	rec, err := h.store.UpdateInvite(c.Request.Context(), idStr, body.IsAccepted, additional)
	if err != nil {
		logger.WithError(err).Error("failed to update invite")
		c.JSON(http.StatusBadRequest, Error{Message: err.Error()})
		return
	}
	if rec == nil {
		c.JSON(http.StatusNotFound, Error{Message: "invite not found"})
		return
	}

	logger.Info("invite accepted")
	c.JSON(http.StatusOK, recordToInvite(rec))
}

func recordToInvite(r *store.InviteRecord) Invite {
	inv := Invite{
		People:          r.People,
		AdditionalCount: r.AdditionalCount,
		IsAccepted:      r.Accepted,
		IsOpened:        len(r.ViewedAt) > 0,
	}
	if len(r.Additional) > 0 {
		inv.Additional = &r.Additional
	}
	return inv
}
