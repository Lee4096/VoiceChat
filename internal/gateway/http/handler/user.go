package handler

import (
	"net/http"

	"voicechat/internal/user"
	"voicechat/pkg/utils"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *user.Service
}

func NewUserHandler(userService *user.Service) *UserHandler {
	return &UserHandler{userService: userService}
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	Provider  string `json:"provider"`
	CreatedAt string `json:"created_at"`
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	ctx := c.Request.Context()

	u, err := h.userService.GetByID(ctx, userID.(string))
	if err != nil {
		if err == user.ErrUserNotFound {
			utils.NewErrorResponse(c, utils.ErrCodeInternal, "用户不存在", http.StatusNotFound)
		} else {
			utils.NewErrorResponse(c, utils.ErrCodeInternal, "获取用户失败", http.StatusInternalServerError)
		}
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Avatar:    u.Avatar,
		Provider:  u.Provider,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}
