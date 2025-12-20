package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/burugo/thing"
	"github.com/mark3labs/mcp-go/mcp"
)

type ToolsCacheEntry struct {
	Tools     []mcp.Tool `json:"tools"`
	FetchedAt time.Time  `json:"fetched_at"`
}

type toolsLocalCacheItem struct {
	value     string
	expiresAt time.Time
}

// ToolsCacheManager caches tool lists separately from health status.
type ToolsCacheManager struct {
	cacheClient thing.CacheClient
	expireTime  time.Duration
	mutex       sync.RWMutex
	local       map[string]toolsLocalCacheItem
}

func NewToolsCacheManager(expireTime time.Duration) *ToolsCacheManager {
	if expireTime <= 0 {
		expireTime = 10 * time.Minute
	}

	return &ToolsCacheManager{
		cacheClient: thing.Cache(),
		expireTime:  expireTime,
		local:       make(map[string]toolsLocalCacheItem),
	}
}

func (tcm *ToolsCacheManager) generateCacheKey(serviceID int64) string {
	return fmt.Sprintf("tools:service:%d", serviceID)
}

func (tcm *ToolsCacheManager) SetServiceTools(serviceID int64, entry *ToolsCacheEntry) {
	if entry == nil {
		return
	}

	tcm.mutex.Lock()
	defer tcm.mutex.Unlock()

	ctx := context.Background()
	cacheKey := tcm.generateCacheKey(serviceID)

	entryCopy := *entry
	entryJSON, err := json.Marshal(&entryCopy)
	if err != nil {
		log.Printf("Error marshaling tools cache for service %d: %v", serviceID, err)
		return
	}

	if tcm.cacheClient == nil {
		tcm.local[cacheKey] = toolsLocalCacheItem{
			value:     string(entryJSON),
			expiresAt: time.Now().Add(tcm.expireTime),
		}
		return
	}

	if err := tcm.cacheClient.Set(ctx, cacheKey, string(entryJSON), tcm.expireTime); err != nil {
		log.Printf("Error setting tools cache for service %d: %v", serviceID, err)
		return
	}
}

func (tcm *ToolsCacheManager) GetServiceTools(serviceID int64) (*ToolsCacheEntry, bool) {
	tcm.mutex.RLock()
	defer tcm.mutex.RUnlock()

	ctx := context.Background()
	cacheKey := tcm.generateCacheKey(serviceID)

	var entryJSON string
	if tcm.cacheClient == nil {
		item, ok := tcm.local[cacheKey]
		if !ok {
			return nil, false
		}
		if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
			delete(tcm.local, cacheKey)
			return nil, false
		}
		entryJSON = item.value
	} else {
		v, err := tcm.cacheClient.Get(ctx, cacheKey)
		if err != nil {
			return nil, false
		}
		entryJSON = v
	}

	var entry ToolsCacheEntry
	if err := json.Unmarshal([]byte(entryJSON), &entry); err != nil {
		log.Printf("Error unmarshaling tools cache for service %d: %v", serviceID, err)
		go tcm.DeleteServiceTools(serviceID)
		return nil, false
	}

	return &entry, true
}

func (tcm *ToolsCacheManager) DeleteServiceTools(serviceID int64) {
	tcm.mutex.Lock()
	defer tcm.mutex.Unlock()

	ctx := context.Background()
	cacheKey := tcm.generateCacheKey(serviceID)

	if tcm.cacheClient == nil {
		delete(tcm.local, cacheKey)
		return
	}

	if err := tcm.cacheClient.Delete(ctx, cacheKey); err != nil {
		log.Printf("Error deleting tools cache for service %d: %v", serviceID, err)
	}
}

var globalToolsCacheManager *ToolsCacheManager
var toolsCacheOnce sync.Once

func GetToolsCacheManager() *ToolsCacheManager {
	toolsCacheOnce.Do(func() {
		globalToolsCacheManager = NewToolsCacheManager(10 * time.Minute)
	})
	return globalToolsCacheManager
}
