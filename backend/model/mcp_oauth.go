package model

import (
	"errors"
	"fmt"

	"github.com/burugo/thing"
)

var ErrMCPOAuthNotFound = errors.New("mcp_oauth_not_found")

type MCPOAuth struct {
	thing.BaseModel
	ServiceID                    int64  `db:"service_id,unique" json:"service_id"`
	ClientID                     string `db:"client_id" json:"-"`
	EncryptedClientSecret        string `db:"encrypted_client_secret" json:"-"`
	EncryptedToken               string `db:"encrypted_token" json:"-"`
	AuthServerMetadataURL        string `db:"auth_server_metadata_url" json:"-"`
	ProtectedResourceMetadataURL string `db:"protected_resource_metadata_url" json:"-"`
}

func (o *MCPOAuth) TableName() string {
	return "mcp_oauth"
}

var MCPOAuthDB *thing.Thing[*MCPOAuth]

func MCPOAuthInit() error {
	var err error
	MCPOAuthDB, err = thing.Use[*MCPOAuth]()
	if err != nil {
		return fmt.Errorf("failed to initialize MCPOAuthDB: %w", err)
	}
	return nil
}

func GetMCPOAuthByServiceID(serviceID int64) (*MCPOAuth, error) {
	records, err := MCPOAuthDB.Where("service_id = ?", serviceID).Fetch(0, 1)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, ErrMCPOAuthNotFound
	}
	return records[0], nil
}

func SaveMCPOAuth(oauth *MCPOAuth) error {
	return MCPOAuthDB.Save(oauth)
}

func DeleteMCPOAuthByServiceID(serviceID int64) error {
	oauth, err := GetMCPOAuthByServiceID(serviceID)
	if err != nil {
		if errors.Is(err, ErrMCPOAuthNotFound) {
			return nil
		}
		return err
	}
	return MCPOAuthDB.Delete(oauth)
}
