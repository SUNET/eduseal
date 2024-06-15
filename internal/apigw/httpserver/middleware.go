package httpserver

import (
	"context"
	"eduseal/pkg/helpers"
	"errors"
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
	//ctx, span := s.tp.Start(ctx, "httpserver:middlewareJWTAuth")
	//defer span.End()

	log := s.logger.New("middlewareJWTAuth")
	log.Debug("middlewareJWTAuth", "enabled", s.config.APIGW.JWTAuth.Enabled)
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

		if tokenString == "" {
			log.Debug("Authorization header not found")
			//c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			//status := c.Writer.Status()
			//log.Trace("crash", "error", r, "status", status, "url", c.Request.URL.Path, "method", c.Request.Method)
			renderContent(c, 401, gin.H{"data": nil, "error": helpers.NewError("unauthorized")})
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

		jwks, err := keyfunc.Get("https://auth-test.sunet.se/.well-known/jwks.json", options)
		if err != nil {
			log.Error(err, "Faild to create JWKS from resource at the given URL")
			//c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			renderContent(c, 401, gin.H{"data": nil, "error": helpers.NewError("unauthorized")})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, jwks.Keyfunc)
		if err != nil {
			log.Error(err, "tokenParse")
			//c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			renderContent(c, 401, gin.H{"data": nil, "error": helpers.NewError("unauthorized")})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Error(errors.New("claims can't be cast to jwt.MapClaims"), "middlewareJWTAuth")
			// c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			renderContent(c, 401, gin.H{"data": nil, "error": helpers.NewError("unauthorized")})
			c.Abort()
			return
		}

		if !token.Valid {
			log.Debug("token not valid")
			// c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			renderContent(c, 401, gin.H{"data": nil, "error": helpers.NewError("unauthorized")})
			c.Abort()
			return
		}

		// Check if the requested access is allowed
		organizationID, ok := claims["organization_id"]
		if !ok {
			log.Debug("organization_id not found in claims")
			renderContent(c, 401, gin.H{"data": nil, "error": helpers.NewError("unauthorized")})
			c.Abort()
			return
		}

		organizationIDStr, ok := organizationID.(string)
		if !ok {
			log.Debug("organization_id not a string")
			renderContent(c, 401, gin.H{"data": nil, "error": helpers.NewError("unauthorized")})
			c.Abort()
			return
		}

		accessService, ok := s.config.APIGW.JWTAuth.Access[organizationIDStr]
		if !ok {
			log.Debug("organization_id not found in config")
			//			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			renderContent(c, 401, gin.H{"data": nil, "error": helpers.NewError("unauthorized")})
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
			log.Debug("requested access not allowed")
			//c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			renderContent(c, 401, gin.H{"data": nil, "error": helpers.NewError("unauthorized")})
			c.Abort()
			return
		}

		c.Next()
	}
}
