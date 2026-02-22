package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// NewOpenAPIValidator creates a Gin middleware that validates incoming requests
// against the provided OpenAPI 3 spec. Invalid requests are rejected with 400.
func NewOpenAPIValidator(spec *openapi3.T) (gin.HandlerFunc, error) {
	// Reason: clear servers so the router matches paths without a server URL prefix
	spec.Servers = nil

	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		return nil, fmt.Errorf("creating openapi router: %w", err)
	}

	return validatorHandler(router), nil
}

func validatorHandler(router routers.Router) gin.HandlerFunc {
	return func(c *gin.Context) {
		route, pathParams, err := router.FindRoute(c.Request)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"message": "route not found in API specification",
			})
			return
		}

		input := &openapi3filter.RequestValidationInput{
			Request:    c.Request,
			PathParams: pathParams,
			Route:      route,
			Options: &openapi3filter.Options{
				// Reason: skip auth validation since this API has no auth
				AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			},
		}

		if err := openapi3filter.ValidateRequest(c.Request.Context(), input); err != nil {
			log.WithError(err).WithField("path", c.Request.URL.Path).Warn("request validation failed")

			msg := sanitizeValidationError(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"message": msg,
			})
			return
		}

		c.Next()
	}
}

func sanitizeValidationError(err error) string {
	msg := err.Error()
	// Reason: kin-openapi wraps errors verbosely; trim to the useful part
	if idx := strings.Index(msg, "Schema:"); idx >= 0 {
		msg = strings.TrimSpace(msg[idx:])
	}
	return msg
}
