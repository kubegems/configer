package client

import (
	"context"
	"crypto/md5"
	"encoding/hex"
)

/*
	key-value based configuration center
*/

type ConfigItem struct {
	Tenant           string `json:"tenant"`
	Project          string `json:"project"`
	Application      string `json:"application"`
	Environment      string `json:"environment"`
	Key              string `json:"key"`
	Value            string `json:"value"`
	Rev              int64  `json:"rev"`
	CreatedTime      string `json:"createdTime"`
	LastModifiedTime string `json:"lastModifiedTime"`
	LastUpdateUser   string `json:"lastUpdateUser"`
}

type HistoryVersion struct {
	Rev            string `json:"rev"`
	Version        string `json:"version"`
	LastUpdateTime string `json:"last_update_time"`
}

type ListOptions struct {
	ConfigItem
	Page int `json:"page"`
	Size int `json:"size"`
}

type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ConfigClientIface interface {
	BaseInfo(ctx context.Context, item *ConfigItem) (map[string]string, error)
	Get(ctx context.Context, item *ConfigItem) error
	Pub(ctx context.Context, item *ConfigItem) error
	Delete(ctx context.Context, item *ConfigItem) error
	List(ctx context.Context, opts *ListOptions) ([]*ConfigItem, error)
	History(ctx context.Context, item *ConfigItem) ([]*HistoryVersion, error)
	Accounts(item *ConfigItem) ([]Account, error)
	Listener(ctx context.Context, item *ConfigItem) (map[string]string, error)
}

var _ ConfigClientIface = &NacosService{}
var _ ConfigClientIface = &EtcdService{}

const salt = "kubegems "

func GenPassword(uname string) string {
	str := salt + uname
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
