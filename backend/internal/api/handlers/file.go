package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"one-cmp/backend/internal/common"
	"one-cmp/backend/internal/domain/model"
	"one-cmp/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// GetAllFiles gets a paginated list of files.
func GetAllFiles(c *gin.Context) {
	// TODO: Add permission check - assuming admin for now
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	files, err := service.FindAllFiles(p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    files,
	})
}

// SearchFiles searches for files by keyword.
func SearchFiles(c *gin.Context) {
	// TODO: Add permission check - assuming admin for now
	keyword := c.Query("keyword")
	files, err := service.FindFilesByKeyword(keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    files,
	})
}

// UploadFile handles file uploads.
func UploadFile(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的表单: " + err.Error()})
		return
	}
	// Get user from context (set by Auth middleware)
	id := c.GetInt("id")
	if id == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "用户未登录或无效"})
		return
	}
	// For service func, we need the user model (or just ID if service fetches it)
	user := &model.User{Id: id}

	files := form.File["file"]
	var links []string
	var firstError error
	for _, file := range files {
		link, err := service.UploadAndRecordFile(user, file)
		if err != nil {
			common.SysError(fmt.Sprintf("Failed to upload file %s: %s", file.Filename, err.Error()))
			if firstError == nil {
				firstError = err
			}
			continue
		}
		// Construct full URL - Assuming UploadPath doesn't end with /
		fullLink := common.ServerAddress + "/upload/" + link // Construct URL relative to server address
		links = append(links, fullLink)
	}

	if len(links) == 0 && firstError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "文件上传失败: " + firstError.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    links,
	})
}

// DeleteFile handles deleting a file.
func DeleteFile(c *gin.Context) {
	fileIDStr := c.Param("id")
	fileID, err := strconv.Atoi(fileIDStr)
	if err != nil || fileID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的文件 ID"})
		return
	}
	// TODO: Add permission check (Admin or Owner) using user from context

	if err := service.DeleteFileRecord(fileID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "文件未找到"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "文件删除失败: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "文件删除成功"})
}

// DownloadFile handles file downloads.
func DownloadFile(c *gin.Context) {
	link := c.Param("link")
	// TODO: Add permission checks

	fileRecord, err := service.FindFileByLink(link)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "文件记录未找到"})
		return
	}

	fullPath := filepath.Join(common.UploadPath, link)
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(common.UploadPath)) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的文件路径"})
		return
	}
	c.FileAttachment(fullPath, fileRecord.Filename)
}
