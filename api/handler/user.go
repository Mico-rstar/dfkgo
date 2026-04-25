package handler

import (
	"dfkgo/api/response"
	"dfkgo/entity"
	"dfkgo/errcode"

	usersvc "dfkgo/service/user"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *usersvc.UserService
}

func NewUserHandler(userService *usersvc.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	uid := getUserID(c)
	profile, err := h.userService.GetProfile(uid)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OK(c, profile)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	uid := getUserID(c)
	var req entity.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(c, errcode.ErrInternal)
		return
	}
	if err := h.userService.UpdateProfile(uid, &req); err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OKWithMsg(c, "用户信息更新成功", nil)
}

func (h *UserHandler) InitAvatarUpload(c *gin.Context) {
	uid := getUserID(c)
	var req entity.AvatarUploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(c, errcode.ErrInvalidMimeType)
		return
	}
	resp, err := h.userService.InitAvatarUpload(uid, req.MimeType, req.FileSize)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OK(c, resp)
}

func (h *UserHandler) AvatarUploadCallback(c *gin.Context) {
	uid := getUserID(c)
	var req entity.AvatarUploadCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(c, errcode.ErrInternal)
		return
	}
	avatarUrl, err := h.userService.AvatarUploadCallback(uid, req.ObjectKey)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OK(c, entity.AvatarUploadCallbackResponse{AvatarUrl: avatarUrl})
}

func (h *UserHandler) FetchAvatar(c *gin.Context) {
	uid := getUserID(c)
	avatarUrl, err := h.userService.FetchAvatar(uid)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OK(c, entity.FetchAvatarResponse{AvatarUrl: avatarUrl})
}

func getUserID(c *gin.Context) uint64 {
	userID, _ := c.Get("userID")
	return userID.(uint64)
}
