package httpserver

import (
	"context"
	"eduseal/pkg/helpers"
	"fmt"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/lithammer/shortuuid/v4"
)

func (s *Service) middlewareDuration(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		c.Next()
		duration := time.Since(t)
		c.Set("duration", duration)
	}
}

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

func (s *Service) middlewareAuthLog(ctx context.Context) gin.HandlerFunc {
	ctx, span := s.tp.Start(ctx, "httpserver:middlewareAuthLog")
	defer span.End()

	log := s.logger.New("http")
	return func(c *gin.Context) {
		u, _ := c.Get("user")
		c.Next()
		log.Info("auth", "user", u, "req_id", c.GetString("req_id"))
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

func (s *Service) middlewareClientCertAuth(ctx context.Context) gin.HandlerFunc {
	ctx, span := s.tp.Start(ctx, "httpserver:middlewareClientCertAuth")
	defer span.End()

	log := s.logger.New("http")
	return func(c *gin.Context) {
		clientCertSHA1 := c.Request.Header.Get("X-SSL-Client-SHA1")
		log.Info("clientCertSHA1", "clientCertSHA1", clientCertSHA1)
		fmt.Println("clientCertSHA1", clientCertSHA1)
		c.Next()
	}
}

// middlewareJWTAuth middleware to require authentication
func (s *Service) middlewareJWTAuth(ctx context.Context) gin.HandlerFunc {
	ctx, span := s.tp.Start(ctx, "httpserver:middlewareJWTAuth")
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
		organizationID, ok := claims["organization_id"]
		if !ok {
			details := "organization_id not found in claims"
			log.Debug(details)
			err := helpers.Error{
				Title:   "unauthorized",
				Details: details,
			}
			renderContent(c, 401, gin.H{"data": nil, "error": err})
			c.Abort()
			return
		}

		organizationIDStr, ok := organizationID.(string)
		if !ok {
			details := "organization_id not a string"
			log.Debug(details)
			err := helpers.Error{
				Title:   "unauthorized",
				Details: details,
			}
			renderContent(c, 401, gin.H{"data": nil, "error": err})
			c.Abort()
			return
		}

		accessService, ok := s.config.APIGW.JWTAuth.Access[organizationIDStr]
		if !ok {
			details := "organization_id not found in config"
			log.Debug(details)
			err := helpers.Error{
				Title:   "unauthorized",
				Details: details,
			}
			renderContent(c, 401, gin.H{"data": nil, "error": err})
			c.Abort()
			return
		}

		allowed := false
		for _, accessClaim := range claims["requested_access"].([]any) {
			ac := accessClaim.(map[string]any)
			if ac["type"] == accessService {
				allowed = true
				break
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

		c.Next()
	}
}
