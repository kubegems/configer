package client

import (
	"context"
)

/*
	key-value based configuration center
*/

type ConfigItem struct {
	Tenant      string
	Project     string
	Application string
	Environment string
	Key         string
	Value       string
	Rev         int64
}

type HistoryVersion struct {
	Rev            string `json:"rev"`
	Version        string `json:"version"`
	LastUpdateTime string `json:"last_update_time"`
}

type ListOptions struct {
	ConfigItem
	Page int
	Size int
}

type ConfigClientIface interface {
	Get(ctx context.Context, item *ConfigItem) error
	Pub(ctx context.Context, item *ConfigItem) error
	Delete(ctx context.Context, item *ConfigItem) error
	List(ctx context.Context, opts *ListOptions) ([]*ConfigItem, error)
	History(ctx context.Context, item *ConfigItem) ([]*HistoryVersion, error)
}

var _ ConfigClientIface = &NacosService{}
var _ ConfigClientIface = &EtcdService{}
