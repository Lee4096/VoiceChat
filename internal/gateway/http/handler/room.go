package handler

import (
	"net/http"
	"strconv"

	"voicechat/internal/room"
	"voicechat/pkg/utils"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	roomService *room.Service
}

func NewRoomHandler(roomService *room.Service) *RoomHandler {
	return &RoomHandler{roomService: roomService}
}

type CreateRoomRequest struct {
	Name string `json:"name" binding:"required"`
}

type RoomResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	OwnerID   string `json:"owner_id"`
	CreatedAt string `json:"created_at"`
}

func (h *RoomHandler) Create(c *gin.Context) {
	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "无效的请求: "+err.Error(), http.StatusBadRequest)
		return
	}

	userID, _ := c.Get("user_id")
	ctx := c.Request.Context()

	r, err := h.roomService.Create(ctx, req.Name, userID.(string))
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "创建房间失败", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, RoomResponse{
		ID:        r.ID,
		Name:      r.Name,
		OwnerID:   r.OwnerID,
		CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *RoomHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	ctx := c.Request.Context()
	rooms, err := h.roomService.List(ctx, limit, offset)
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "获取房间列表失败", http.StatusInternalServerError)
		return
	}

	var response []RoomResponse
	for _, r := range rooms {
		response = append(response, RoomResponse{
			ID:        r.ID,
			Name:      r.Name,
			OwnerID:   r.OwnerID,
			CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.JSON(http.StatusOK, gin.H{"rooms": response})
}

func (h *RoomHandler) Get(c *gin.Context) {
	roomID := c.Param("id")
	ctx := c.Request.Context()

	r, err := h.roomService.GetByID(ctx, roomID)
	if err != nil {
		if err == room.ErrRoomNotFound {
			utils.NewErrorResponse(c, utils.ErrCodeRoomNotFound, "房间不存在", http.StatusNotFound)
		} else {
			utils.NewErrorResponse(c, utils.ErrCodeInternal, "获取房间失败", http.StatusInternalServerError)
		}
		return
	}

	members, _ := h.roomService.GetMembers(ctx, roomID)

	c.JSON(http.StatusOK, gin.H{
		"room": RoomResponse{
			ID:        r.ID,
			Name:      r.Name,
			OwnerID:   r.OwnerID,
			CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
		"members": members,
	})
}

func (h *RoomHandler) Join(c *gin.Context) {
	roomID := c.Param("id")
	userID, _ := c.Get("user_id")
	ctx := c.Request.Context()

	_, err := h.roomService.GetByID(ctx, roomID)
	if err != nil {
		if err == room.ErrRoomNotFound {
			utils.NewErrorResponse(c, utils.ErrCodeRoomNotFound, "房间不存在", http.StatusNotFound)
		} else {
			utils.NewErrorResponse(c, utils.ErrCodeInternal, "获取房间失败", http.StatusInternalServerError)
		}
		return
	}

	member, err := h.roomService.Join(ctx, roomID, userID.(string))
	if err != nil {
		if err == room.ErrRoomFull {
			utils.NewErrorResponse(c, utils.ErrCodeRoomFull, "房间已满", http.StatusForbidden)
		} else {
			utils.NewErrorResponse(c, utils.ErrCodeInternal, "加入房间失败", http.StatusInternalServerError)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"member": member})
}

func (h *RoomHandler) Leave(c *gin.Context) {
	roomID := c.Param("id")
	userID, _ := c.Get("user_id")
	ctx := c.Request.Context()

	err := h.roomService.Leave(ctx, roomID, userID.(string))
	if err != nil {
		utils.NewErrorResponse(c, utils.ErrCodeInternal, "离开房间失败", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已离开房间"})
}
