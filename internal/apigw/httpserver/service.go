package httpserver

import (
	"context"
	"crypto/tls"
	"eduseal/internal/apigw/apiv1"
	"eduseal/pkg/helpers"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
	"net/http"
	"time"

	// Swagger
	_ "eduseal/docs/apigw"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Service is the service object for httpserver
type Service struct {
	config    *model.Cfg
	logger    *logger.Log
	server    *http.Server
	apiv1     Apiv1
	gin       *gin.Engine
	tlsConfig *tls.Config
	tp        *trace.Tracer
}

// New creates a new httpserver service
func New(ctx context.Context, config *model.Cfg, api *apiv1.Client, tp *trace.Tracer, log *logger.Log) (*Service, error) {
	s := &Service{
		config: config,
		logger: log,
		apiv1:  api,
		tp:     tp,
		server: &http.Server{
			ReadHeaderTimeout: 2 * time.Second,
		},
	}

	switch s.config.Common.Production {
	case true:
		gin.SetMode(gin.ReleaseMode)
	case false:
		gin.SetMode(gin.DebugMode)
	}

	apiValidator, err := helpers.NewValidator()
	if err != nil {
		return nil, err
	}
	binding.Validator = &defaultValidator{
		Validate: apiValidator,
	}

	s.gin = gin.New()
	s.server.Handler = s.gin
	s.server.Addr = config.APIGW.APIServer.Addr
	s.server.ReadTimeout = 5 * time.Second
	s.server.WriteTimeout = 30 * time.Second
	s.server.IdleTimeout = 90 * time.Second

	// Middlewares
	s.gin.Use(s.middlewareRequestID(ctx))
	s.gin.Use(s.middlewareDuration(ctx))
	s.gin.Use(s.middlewareLogger(ctx))
	s.gin.Use(s.middlewareCrash(ctx))
	problem404, err := helpers.Problem404()
	if err != nil {
		return nil, err
	}
	s.gin.NoRoute(func(c *gin.Context) { c.JSON(http.StatusNotFound, problem404) })

	rgRoot := s.gin.Group("/")
	s.regEndpoint(ctx, rgRoot, http.MethodGet, "health", s.endpointHealth)

	rgDocs := rgRoot.Group("/swagger")
	rgDocs.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	rgAPIv1 := rgRoot.Group("api/v1")

	rgPDF := rgAPIv1.Group("/pdf")
	if s.config.APIGW.JWTAuth.Enabled {
		rgPDF.Use(s.middlewareJWTAuth(ctx))
	}
	s.regEndpoint(ctx, rgPDF, http.MethodPost, "/sign", s.endpointSignPDF)
	s.regEndpoint(ctx, rgPDF, http.MethodGet, "/:transaction_id", s.endpointGetSignedPDF)
	s.regEndpoint(ctx, rgPDF, http.MethodPost, "/validate", s.endpointValidatePDF)
	s.regEndpoint(ctx, rgPDF, http.MethodPut, "/revoke/:transaction_id", s.endpointPDFRevoke)

	// Run http server
	go func() {
		s.logger.Info("ListenAndServe", "addr", s.config.APIGW.APIServer.Addr)
		s.logger.Info("TLS enabled", "enabled", s.config.APIGW.APIServer.TLS.Enabled)
		if s.config.APIGW.APIServer.TLS.Enabled {
			s.logger.Info("TLS enabled")
			s.applyTLSConfig(ctx)

			err := s.server.ListenAndServeTLS(s.config.APIGW.APIServer.TLS.CertFilePath, s.config.APIGW.APIServer.TLS.KeyFilePath)
			if err != nil {
				s.logger.Error(err, "listen_and_server_tls")
			}
		} else {
			err = s.server.ListenAndServe()
			s.logger.Info("TLS disabled")
			if err != nil {
				s.logger.Error(err, "listen_and_server")
			}
		}
	}()

	s.logger.Info("started")

	return s, nil
}

func (s *Service) regEndpoint(ctx context.Context, rg *gin.RouterGroup, method, path string, handler func(context.Context, *gin.Context) (any, error)) {
	rg.Handle(method, path, func(c *gin.Context) {
		res, err := handler(ctx, c)
		if err != nil {
			renderContent(c, 400, gin.H{"error": helpers.NewErrorFromError(err)})
			return
		}

		renderContent(c, 200, res)
	})
}

func renderContent(c *gin.Context, code int, data any) {
	switch c.NegotiateFormat(gin.MIMEJSON, "*/*") {
	case gin.MIMEJSON:
		c.JSON(code, data)
	case "*/*": // curl
		c.JSON(code, data)
	default:
		c.JSON(406, gin.H{"error": helpers.NewErrorDetails("not_acceptable", "Accept header is invalid. It should be \"application/json\".")})
	}
}

// Close closing httpserver
func (s *Service) Close(ctx context.Context) error {
	s.logger.Info("Quit")
	return nil
}
