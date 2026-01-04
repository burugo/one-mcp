package handler

import (
	"net/http"
	"one-mcp/backend/common"
	"one-mcp/backend/common/i18n"
	"one-mcp/backend/model"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type groupPayload struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	Description    string `json:"description"`
	ServiceIDsJSON string `json:"service_ids_json"`
	Enabled        *bool  `json:"enabled"`
}

func GetGroups(c *gin.Context) {
	userID := c.GetInt64("user_id")
	groups, err := model.GetMCPServiceGroupsByUserID(userID)
	if err != nil {
		common.RespError(c, http.StatusInternalServerError, "failed to fetch groups", err)
		return
	}
	common.RespSuccess(c, groups)
}

func CreateGroup(c *gin.Context) {
	lang := c.GetString("lang")
	var payload groupPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_param", lang), err)
		return
	}
	if strings.TrimSpace(payload.Name) == "" || strings.TrimSpace(payload.DisplayName) == "" {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_param", lang))
		return
	}

	userID := c.GetInt64("user_id")
	group := &model.MCPServiceGroup{
		UserID:         userID,
		Name:           strings.TrimSpace(payload.Name),
		DisplayName:    strings.TrimSpace(payload.DisplayName),
		Description:    strings.TrimSpace(payload.Description),
		ServiceIDsJSON: strings.TrimSpace(payload.ServiceIDsJSON),
		Enabled:        true,
	}
	if payload.Enabled != nil {
		group.Enabled = *payload.Enabled
	}

	if err := group.Insert(); err != nil {
		common.RespError(c, http.StatusInternalServerError, "failed to create group", err)
		return
	}
	common.RespSuccess(c, group)
}

func UpdateGroup(c *gin.Context) {
	lang := c.GetString("lang")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_param", lang))
		return
	}

	var payload groupPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.RespError(c, http.StatusBadRequest, i18n.Translate("invalid_param", lang), err)
		return
	}

	userID := c.GetInt64("user_id")
	group, err := model.GetMCPServiceGroupByID(id, userID)
	if err != nil {
		common.RespError(c, http.StatusNotFound, "group not found", err)
		return
	}

	if strings.TrimSpace(payload.Name) != "" {
		group.Name = strings.TrimSpace(payload.Name)
	}
	if strings.TrimSpace(payload.DisplayName) != "" {
		group.DisplayName = strings.TrimSpace(payload.DisplayName)
	}
	if payload.Description != "" {
		group.Description = strings.TrimSpace(payload.Description)
	}
	if payload.ServiceIDsJSON != "" {
		group.ServiceIDsJSON = strings.TrimSpace(payload.ServiceIDsJSON)
	}
	if payload.Enabled != nil {
		group.Enabled = *payload.Enabled
	}

	if err := group.Update(); err != nil {
		common.RespError(c, http.StatusInternalServerError, "failed to update group", err)
		return
	}
	common.RespSuccess(c, group)
}

func DeleteGroup(c *gin.Context) {
	lang := c.GetString("lang")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.RespErrorStr(c, http.StatusBadRequest, i18n.Translate("invalid_param", lang))
		return
	}

	userID := c.GetInt64("user_id")
	group, err := model.GetMCPServiceGroupByID(id, userID)
	if err != nil {
		common.RespError(c, http.StatusNotFound, "group not found", err)
		return
	}

	if err := group.Delete(); err != nil {
		common.RespError(c, http.StatusInternalServerError, "failed to delete group", err)
		return
	}

	common.RespSuccess(c, nil)
}
