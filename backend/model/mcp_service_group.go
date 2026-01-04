package model

import (
	"encoding/json"
	"errors"

	"github.com/burugo/thing"
)

type MCPServiceGroup struct {
	thing.BaseModel

	UserID         int64  `db:"user_id,index:idx_group_owner" json:"user_id"`
	Name           string `db:"name,index:idx_group_owner" json:"name"`
	DisplayName    string `db:"display_name" json:"display_name"`
	Description    string `db:"description" json:"description"`
	ServiceIDsJSON string `db:"service_ids_json" json:"service_ids_json"`
	Enabled        bool   `db:"enabled" json:"enabled"`
}

var MCPServiceGroupDB *thing.Thing[*MCPServiceGroup]

func MCPServiceGroupInit() error {
	var err error
	MCPServiceGroupDB, err = thing.Use[*MCPServiceGroup]()
	return err
}

func (g *MCPServiceGroup) TableName() string {
	return "mcp_service_groups"
}

func (g *MCPServiceGroup) GetServiceIDs() []int64 {
	var ids []int64
	if g.ServiceIDsJSON == "" {
		return ids
	}
	_ = json.Unmarshal([]byte(g.ServiceIDsJSON), &ids)
	return ids
}

func (g *MCPServiceGroup) SetServiceIDs(ids []int64) {
	bytes, _ := json.Marshal(ids)
	g.ServiceIDsJSON = string(bytes)
}

func GetMCPServiceGroupsByUserID(userID int64) ([]*MCPServiceGroup, error) {
	return MCPServiceGroupDB.Where("user_id = ?", userID).Order("id DESC").Fetch(0, 1000)
}

func GetMCPServiceGroupByName(name string, userID int64) (*MCPServiceGroup, error) {
	groups, err := MCPServiceGroupDB.Where("name = ? AND user_id = ?", name, userID).Fetch(0, 1)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, errors.New("group_not_found")
	}
	return groups[0], nil
}

func GetMCPServiceGroupByID(id int64, userID int64) (*MCPServiceGroup, error) {
	group, err := MCPServiceGroupDB.ByID(id)
	if err != nil {
		return nil, err
	}
	if group.UserID != userID {
		return nil, errors.New("unauthorized")
	}
	return group, nil
}

func (g *MCPServiceGroup) Insert() error {
	if g.UserID == 0 || g.Name == "" || g.DisplayName == "" {
		return errors.New("missing_required_fields")
	}
	return MCPServiceGroupDB.Save(g)
}

func (g *MCPServiceGroup) Update() error {
	if g.ID == 0 {
		return errors.New("empty_id")
	}
	return MCPServiceGroupDB.Save(g)
}

func (g *MCPServiceGroup) Delete() error {
	if g.ID == 0 {
		return errors.New("empty_id")
	}
	return MCPServiceGroupDB.SoftDelete(g)
}

func (g *MCPServiceGroup) ContainsServiceName(name string) bool {
	ids := g.GetServiceIDs()
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		svc, err := GetServiceByID(id)
		if err != nil {
			continue
		}
		if svc.Name == name {
			return true
		}
	}
	return false
}

func (g *MCPServiceGroup) GetServiceByName(name string) (*MCPService, error) {
	ids := g.GetServiceIDs()
	for _, id := range ids {
		svc, err := GetServiceByID(id)
		if err != nil {
			continue
		}
		if svc.Name == name {
			return svc, nil
		}
	}
	return nil, errors.New("service_not_in_group")
}
