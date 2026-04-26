package handler

import (
	"dfkgo/api/response"
	"dfkgo/entity"
	"dfkgo/errcode"

	filesvc "dfkgo/service/file"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	fileService *filesvc.FileService
}

func NewUploadHandler(fileService *filesvc.FileService) *UploadHandler {
	return &UploadHandler{fileService: fileService}
}

func (h *UploadHandler) InitUpload(c *gin.Context) {
	uid := c.GetUint64("userID")
	var req entity.UploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(c, errcode.ErrFileTypeNotSupported)
		return
	}
	resp, err := h.fileService.InitUpload(uid, &req)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}
	if resp.Hit {
		response.OKWithMsg(c, "文件已存在", resp)
	} else {
		response.OK(c, resp)
	}
}

func (h *UploadHandler) UploadCallback(c *gin.Context) {
	uid := c.GetUint64("userID")
	var req entity.UploadCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(c, errcode.ErrFileIDNotFound)
		return
	}
	resp, err := h.fileService.UploadCallback(uid, req.FileId)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OKWithMsg(c, "文件上传完成", resp)
}
