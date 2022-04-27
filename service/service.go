package service

import (
	"context"
	"fmt"

	"kubegems.io/configer/client"
)

type ConfigService struct {
	cfgClient client.ConfigClientIface
}

func (cs *ConfigService) Get(ctx context.Context, tenant, project, env, key string) (string, error) {
	item := &client.ConfigItem{
		Tenant:      tenant,
		Project:     project,
		Environment: env,
		Key:         key,
	}
	if err := cs.cfgClient.Get(ctx, item); err != nil {
		return "", err
	}
	return item.Value, nil

}

func (cs *ConfigService) Pub(ctx context.Context, tenant, project, env, key, value string) error {
	item := &client.ConfigItem{
		Tenant:      tenant,
		Project:     project,
		Environment: env,
		Key:         key,
		Value:       value,
	}
	if err := cs.cfgClient.Pub(ctx, item); err != nil {
		return err
	}
	return nil

}

func (cs *ConfigService) Delete(ctx context.Context, tenant, project, env, key string) error {
	item := &client.ConfigItem{
		Tenant:      tenant,
		Project:     project,
		Environment: env,
		Key:         key,
	}
	if err := cs.cfgClient.Delete(ctx, item); err != nil {
		return err
	}
	return nil
}

func (cs *ConfigService) List(ctx context.Context, tenant, project, env string) error {
	// TODO:
	items, err := cs.cfgClient.List(ctx, &client.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Println(items)
	return nil
}

func (cs *ConfigService) History(ctx context.Context, tenant, project, env, key string) ([]*client.HistoryVersion, error) {
	item := &client.ConfigItem{
		Tenant:      tenant,
		Project:     project,
		Environment: env,
		Key:         key,
	}
	return cs.cfgClient.History(ctx, item)
}

func (cs *ConfigService) AccountInfo() {
}
