package http

import (
	"net/http"
	"time"

	"voicechat/internal/auth"
	"voicechat/internal/gateway/http/handler"
	"voicechat/internal/gateway/http/middleware"
	"voicechat/internal/repository/postgres"
	"voicechat/internal/repository/redis"
	"voicechat/internal/room"
	"voicechat/internal/signaling"
	"voicechat/internal/user"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg           Config
	logger        Logger
	pg            *postgres.DB
	redis         *redis.Client
	signaling     *signaling.Server
	jwtService    *auth.JWTService
	oauthSvc      *auth.OAuth2Service
	passwordSvc   *auth.PasswordService
	userService   *user.Service
	roomService   *room.Service
	engine        *gin.Engine
	rateLimiter   *middleware.RateLimiter
}

type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
}

type Config struct {
	Port         int
	ReadTimeout  int
	WriteTimeout int
}

func NewServer(cfg Config, logger Logger, pg *postgres.DB, rd *redis.Client, signal *signaling.Server) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	s := &Server{
		cfg:       cfg,
		logger:    logger,
		pg:        pg,
		redis:     rd,
		signaling: signal,
		rateLimiter: middleware.NewRateLimiter(100, time.Minute),
	}

	s.setupServices()
	s.setupRoutes(engine)

	return s
}

func (s *Server) setupServices() {
	jwtCfg := struct {
		Secret     string
		Expiration int
	}{
		Secret:     "fireredchat-secret",
		Expiration: 86400,
	}

	oauthCfg := struct {
		GitHub struct {
			ClientID     string
			ClientSecret string
			CallbackURL  string
		}
		Google struct {
			ClientID     string
			ClientSecret string
			CallbackURL  string
		}
	}{}

	s.jwtService = auth.NewJWTService(jwtCfg)
	s.oauthSvc = auth.NewOAuth2Service(oauthCfg)
	s.passwordSvc = auth.NewPasswordService(s.pg.Pool())
	s.userService = user.NewService(s.pg.Pool())
	s.roomService = room.NewService(s.pg.Pool())
}

func (s *Server) setupRoutes(e *gin.Engine) {
	e.Use(middleware.CORSMiddleware())
	e.Use(gin.Logger())

	e.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	authHandler := handler.NewAuthHandler(s.jwtService, s.oauthSvc, s.passwordSvc, s.userService)
	roomHandler := handler.NewRoomHandler(s.roomService)
	userHandler := handler.NewUserHandler(s.userService)
	healthHandler := handler.NewHealthHandler()

	api := e.Group("/api/v1")
	api.Use(middleware.RateLimitMiddleware(s.rateLimiter))
	{
		auth := api.Group("/auth")
		{
			auth.GET("/login/:provider", authHandler.GetLoginURL)
			auth.GET("/callback/:provider", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login/password", authHandler.PasswordLogin)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		rooms := api.Group("/rooms")
		rooms.Use(middleware.AuthMiddleware(s.jwtService))
		{
			rooms.GET("", roomHandler.List)
			rooms.POST("", roomHandler.Create)
			rooms.GET("/:id", roomHandler.Get)
			rooms.POST("/:id/join", roomHandler.Join)
			rooms.POST("/:id/leave", roomHandler.Leave)
		}

		users := api.Group("/users")
		users.Use(middleware.AuthMiddleware(s.jwtService))
		{
			users.GET("/me", userHandler.GetCurrentUser)
		}

		api.GET("/health", healthHandler.Health)
	}

	s.engine = e
}

func (s *Server) Run(ctx interface{}) error {
	addr := ":8080"
	s.logger.Info("HTTP server starting on %s", addr)
	return s.engine.Run(addr)
}

func (s *Server) Close() error {
	return nil
}
