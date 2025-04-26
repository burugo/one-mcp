package service

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"time"

	"one-cmp/backend/internal/common"
	"one-cmp/backend/internal/library/db"
	"one-cmp/backend/internal/domain/model"
	"gorm.io/gorm"
)

// UploadAndRecordFile uploads a file and creates a record in the DB.
func UploadAndRecordFile(user *model.User, file *multipart.FileHeader) (string, error) {
	// TODO: Add permission check based on user.Role
	// if user.Role < model.RoleAdminUser && common.FileUploadPermission == common.RoleAdminUser { ... }

	link, err := common.UploadFile(file) // common.UploadFile handles saving to disk
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	fileRecord := model.File{
		UserId:    user.Id,
		Filename:  file.Filename,
		Link:      link,
		CreatedAt: time.Now().Unix(),
	}

	if err := db.DB.Create(&fileRecord).Error; err != nil {
		// Attempt to clean up the saved file if DB record fails
		diskPath := filepath.Join(common.UploadPath, link)
		_ = common.DeleteFile(diskPath) // Ignore error during cleanup
		return "", fmt.Errorf("failed to save file record: %w", err)
	}

	return link, nil
}

// FindFilesForUser finds files for a specific user, paginated.
func FindFilesForUser(userId int, p int) ([]*model.File, error) {
	// TODO: Add permission check (only self or admin can list)
	var files []*model.File
	query := db.DB.Where("user_id = ?", userId)
	files, err := common.Paginate[model.File](query, p, "id desc")
	return files, err
}

// FindAllFiles finds all files, paginated (for admin).
func FindAllFiles(p int) ([]*model.File, error) {
	// TODO: Ensure caller is Admin
	var files []*model.File
	files, err := common.Paginate[model.File](db.DB, p, "id desc")
	return files, err
}

// FindFilesByKeyword searches files by keyword (for admin).
func FindFilesByKeyword(keyword string) ([]*model.File, error) {
	// TODO: Ensure caller is Admin
	var files []*model.File
	likeKeyword := keyword + "%"
	err := db.DB.Where("filename LIKE ?", likeKeyword).Find(&files).Error
	return files, err
}

// DeleteFileRecord deletes a file record and the file from disk.
func DeleteFileRecord(fileId int) error {
	var fileRecord model.File
	if err := db.DB.First(&fileRecord, fileId).Error; err != nil {
		return fmt.Errorf("failed to find file record %d: %w", fileId, err)
	}

	// TODO: Add permission check (Admin or Owner)

	// Delete DB record first
	if err := db.DB.Delete(&fileRecord).Error; err != nil {
		return fmt.Errorf("failed to delete file record %d: %w", fileId, err)
	}

	// Delete the actual file from disk
	filePath := filepath.Join(common.UploadPath, fileRecord.Link)
	if err := common.DeleteFile(filePath); err != nil {
		// Log error but don't fail the operation if DB entry was deleted
		common.SysError(fmt.Sprintf("Failed to delete file %s from disk for record %d: %s", filePath, fileId, err.Error()))
	}
	return nil
}

// FindFileByLink finds a file record by its link.
func FindFileByLink(link string) (*model.File, error) {
	var fileRecord model.File
	if err := db.DB.Where("link = ?", link).First(&fileRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to find file by link %s: %w", link, err)
	}
	return &fileRecord, nil
} 