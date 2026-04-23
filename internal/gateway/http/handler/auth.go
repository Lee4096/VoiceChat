package handler

import (
	"net/http"

	"voicechat/internal/auth"
	"voicechat/internal/user"
	"voicechat/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	jwtService      *auth.JWTService
	oauthService   *auth.OAuth2Service
	passwordService *auth.PasswordService
	userService    *user.Service
}

func NewAuthHandler(jwtService *auth.JWTService, oauthService *auth.OAuth2Service, passwordService *auth.PasswordService, userService *user.Service) *AuthHandler {
	return &AuthHandler{
		jwtService:      jwtService,
		oauthService:   oauthService,
		passwordService: passwordService,
		userService:    userService,
	}
}

type LoginResponse struct {
	Token string    `json:"token"`
	User  *LoginUser `json:"user"`
}

type LoginUser struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required,min=2"`
}

type PasswordLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "无效的请求: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := c.Request.Context()

	exists, err := h.passwordService.UserExists(ctx, req.Email)
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "检查用户失败", http.StatusInternalServerError)
		return
	}
	if exists {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "用户已存在", http.StatusBadRequest)
		return
	}

	userID, err := h.passwordService.Register(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "注册失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	u, err := h.userService.GetByID(ctx, userID)
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "获取用户失败", http.StatusInternalServerError)
		return
	}

	token, err := h.jwtService.GenerateToken(u.ID, u.Email)
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "生成令牌失败", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, LoginResponse{
		Token: token,
		User: &LoginUser{
			ID:     u.ID,
			Email:  u.Email,
			Name:   u.Name,
			Avatar: u.Avatar,
		},
	})
}

func (h *AuthHandler) PasswordLogin(c *gin.Context) {
	var req PasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "无效的请求: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := c.Request.Context()

	userID, err := h.passwordService.Login(ctx, req.Email, req.Password)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			utils.NewErrorResponse(c, utils.ErrCodeAuthInvalidToken, "邮箱或密码错误", http.StatusUnauthorized)
		} else {
			utils.NewErrorResponse(c, utils.ErrCodeInternal, "登录失败", http.StatusInternalServerError)
		}
		return
	}

	u, err := h.userService.GetByID(ctx, userID)
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "获取用户失败", http.StatusInternalServerError)
		return
	}

	token, err := h.jwtService.GenerateToken(u.ID, u.Email)
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "生成令牌失败", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User: &LoginUser{
			ID:     u.ID,
			Email:  u.Email,
			Name:   u.Name,
			Avatar: u.Avatar,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	provider := c.Param("provider")

	var code string
	if provider == "github" {
		code = c.Query("code")
	} else if provider == "google" {
		code = c.Query("code")
	} else {
		utils.NewErrorResponse(c, utils.ErrCodeAuthOAuthFailed, "不支持的认证提供商", http.StatusBadRequest)
		return
	}

	ctx := c.Request.Context()
	var oauthUser *auth.OAuthUser
	var err error

	if provider == "github" {
		oauthUser, err = h.oauthService.GitHubCallback(ctx, code)
	} else if provider == "google" {
		oauthUser, err = h.oauthService.GoogleCallback(ctx, code)
	}

	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeAuthOAuthFailed, "OAuth认证失败: "+err.Error(), http.StatusUnauthorized)
		return
	}

	u, err := h.userService.Create(ctx, oauthUser.Email, oauthUser.Name, "", provider)
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "创建用户失败", http.StatusInternalServerError)
		return
	}

	token, err := h.jwtService.GenerateToken(u.ID, u.Email)
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "生成令牌失败", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User: &LoginUser{
			ID:     u.ID,
			Email:  u.Email,
			Name:   u.Name,
			Avatar: u.Avatar,
		},
	})
}

func (h *AuthHandler) GetLoginURL(c *gin.Context) {
	provider := c.Param("provider")

	var url string
	if provider == "github" {
		url = h.oauthService.GitHubLoginURL()
	} else if provider == "google" {
		url = h.oauthService.GoogleLoginURL()
	} else {
		utils.NewErrorResponse(c, utils.ErrCodeAuthOAuthFailed, "不支持的认证提供商", http.StatusBadRequest)
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		utils.NewErrorResponse(c, utils.ErrCodeAuthInvalidToken, "缺少令牌", http.StatusUnauthorized)
		return
	}

	token = token[7:]

	newToken, err := h.jwtService.RefreshToken(token)
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeAuthTokenExpired, "令牌刷新失败", http.StatusUnauthorized)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": newToken})
}
