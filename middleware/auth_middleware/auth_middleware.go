package auth_middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/needsomesleeptd/annotater-core/models"
	auth_service "github.com/needsomesleeptd/annotater-core/service"
	auth_utils "github.com/needsomesleeptd/annotater-core/utilsPorts/authUtils"
	response "github.com/needsomesleeptd/http-server/lib/api"
	"github.com/sirupsen/logrus"
)

type contextKeyRole struct{}
type contextKeyID struct{}

func FromIncomingContextRole(ctx context.Context) (models.Role, bool) {
	role, ok := ctx.Value(contextKeyRole{}).(models.Role)
	return role, ok
}

func FromIncomingContextID(ctx context.Context) (uint64, bool) {
	id, ok := ctx.Value(contextKeyID{}).(uint64)
	return id, ok
}

var (
	UserIDContextKey = "contextKeyRole{}"
	RoleContextKey   = "contextKeyID{}"
)

func NewJwtAuthMiddleware(loggerSrc *logrus.Logger, secretSrc string, tokenHandlerSrc auth_utils.ITokenHandler) JwtAuthMiddleware {
	return JwtAuthMiddleware{
		secret:       secretSrc,
		tokenHandler: tokenHandlerSrc,
		logger:       loggerSrc,
	}
}

type JwtAuthMiddleware struct {
	logger       *logrus.Logger
	secret       string
	tokenHandler auth_utils.ITokenHandler
}

func (m *JwtAuthMiddleware) MiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			m.logger.Info("user with no token came")
			render.JSON(w, r, response.Error("Error in parsing token"))
			render.Status(r, http.StatusBadRequest)
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")

		payload, err := m.tokenHandler.ParseToken(token, auth_service.SECRET)
		if err != nil {
			if err == auth_utils.ErrParsingToken {
				m.logger.Info("user with invalid jwt came")
				render.JSON(w, r, response.Error(err.Error()))
				render.Status(r, http.StatusBadRequest)
			} else {
				m.logger.Info("user with invalid jwt came")
				render.JSON(w, r, response.Error(err.Error()))
				render.Status(r, http.StatusUnauthorized)
			}
			return
		}
		ctx := context.WithValue(r.Context(), UserIDContextKey, payload.ID)
		ctx = context.WithValue(ctx, RoleContextKey, payload.Role)

		m.logger.WithFields(
			logrus.Fields{
				"src":    "JwtAuthMiddleware.MiddleFunc",
				"userID": payload.ID,
				"role":   payload.Role}).
			Info("successfully authorized")

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
