package httpserver

import (
	"context"
	"eduseal/pkg/helpers"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/lithammer/shortuuid/v4"
)

func (s *Service) middlewareRequestID(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := shortuuid.New()
		c.Set("req_id", id)
		c.Header("req_id", id)
		c.Next()
	}
}

func (s *Service) middlewareLogger(ctx context.Context) gin.HandlerFunc {
	log := s.logger.New("http")
	return func(c *gin.Context) {
		c.Next()
		log.Info("request", "status", c.Writer.Status(), "url", c.Request.URL.String(), "method", c.Request.Method, "req_id", c.GetString("req_id"))
	}
}

func (s *Service) middlewareCrash(ctx context.Context) gin.HandlerFunc {
	log := s.logger.New("http")
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Debug("crash", "error", r)
				status := c.Writer.Status()
				log.Trace("crash", "error", r, "status", status, "url", c.Request.URL.Path, "method", c.Request.Method)
				err := helpers.Error{
					Title:   "internal_server_error",
					Details: r,
				}
				renderContent(c, 500, gin.H{"data": nil, "error": err.Error()})
			}
		}()
		c.Next()
	}
}

// middlewareJWTAuth middleware to require authentication
func (s *Service) middlewareJWTAuth(ctx context.Context) gin.HandlerFunc {
	ctx, span := s.tracer.Start(ctx, "httpserver:middlewareJWTAuth")
	defer span.End()

	log := s.logger.New("middlewareJWTAuth")
	log.Debug("middlewareJWTAuth", "enabled", s.config.APIGW.JWTAuth.Enabled)
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

		if tokenString == "" {
			details := "Authorization header not found"
			log.Debug(details)
			err := helpers.Error{
				Title:   "unauthorized",
				Details: details,
			}
			renderContent(c, 401, gin.H{"data": nil, "error": err})
			c.Abort()
			return
		}
		tokenString, found := strings.CutPrefix(tokenString, "Bearer ")
		if !found {
			log.Debug("no bearer prefix found")
		}

		options := keyfunc.Options{
			Ctx: ctx,
			RefreshErrorHandler: func(err error) {
				log.Error(err, "There was an error with the jwt.KeyFunc")
			},
			RefreshInterval:   time.Hour,
			RefreshRateLimit:  time.Minute * 5,
			RefreshTimeout:    time.Second * 10,
			RefreshUnknownKID: true,
		}

		jwks, err := keyfunc.Get(s.config.APIGW.JWTAuth.JWKURL, options)
		if err != nil {
			details := "Faild to create JWKS from resource at the given URL"
			log.Debug(details)
			err := helpers.Error{
				Title:   "unauthorized",
				Details: details,
			}
			renderContent(c, 401, gin.H{"data": nil, "error": err})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, jwks.Keyfunc)
		if err != nil {
			details := "Failed to parse token"
			log.Debug(details)
			err := helpers.Error{
				Title:   "unauthorized",
				Details: details,
			}
			renderContent(c, 401, gin.H{"data": nil, "error": err})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			details := "claims can't be cast to jwt.MapClaims"
			log.Debug(details)
			err := helpers.Error{
				Title:   "unauthorized",
				Details: details,
			}
			renderContent(c, 401, gin.H{"data": nil, "error": err})
			c.Abort()
			return
		}

		if !token.Valid {
			details := "token not valid"
			log.Debug(details)
			err := helpers.Error{
				Title:   "unauthorized",
				Details: details,
			}
			renderContent(c, 401, gin.H{"data": nil, "error": err})
			c.Abort()
			return
		}

		// Check if the requested access is allowed
		var accessService string
		var accessFound bool

		log.Debug("JWT claims", "claims", claims)

		organizationID, ok := claims["organization_id"]
		if ok {
			organizationIDStr, ok := organizationID.(string)
			if ok {
				accessService, accessFound = s.config.APIGW.JWTAuth.Access[organizationIDStr]
			}
		}

		// Fallback: check if the client certificate CN (from "common_name" claim) is in the access map with value "cn"
		if !accessFound {
			if cn, ok := claims["common_name"].(string); ok {
				if v, cnOK := s.config.APIGW.JWTAuth.Access[cn]; cnOK && v == "cn" {
					accessService = v
					accessFound = true
				}
			}
		}

		if !accessFound {
			details := "access not found in config (no matching organization_id or client certificate CN)"
			log.Debug(details)
			err := helpers.Error{
				Title:   "unauthorized",
				Details: details,
			}
			renderContent(c, 401, gin.H{"data": nil, "error": err})
			c.Abort()
			return
		}

		// CN-based access is authorized by the mTLS certificate itself, skip requested_access check
		if accessService != "cn" {
			allowed := false
			if requestedAccess, ok := claims["requested_access"].([]any); ok {
				for _, accessClaim := range requestedAccess {
					ac := accessClaim.(map[string]any)
					if ac["type"] == accessService {
						allowed = true
						break
					}
				}
			}
			if !allowed {
				details := "requested access not allowed"
				log.Debug(details)
				err := helpers.Error{
					Title:   "unauthorized",
					Details: details,
				}
				renderContent(c, 401, gin.H{"data": nil, "error": err})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
