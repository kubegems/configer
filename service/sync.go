package service

import (
	"context"

	"gorm.io/gorm"
	"kubegems.io/configer/client"
)

/*
由于配置后端可能会不稳定，所以将配置数据同步备份在数据库中，以防数据丢失
*/

// 将配置后端中的数据内容同步到数据库中
func SyncBackend2Database(conditem *client.ConfigItem, items []*client.ConfigItem, db *gorm.DB) error {
	dbitems := []ConfigItem{}
	db.Find(&dbitems, ConfigItem{
		Tenant:      conditem.Tenant,
		Project:     conditem.Project,
		Environment: conditem.Environment,
	})
	dbitemsMap := map[string]ConfigItem{}
	for _, dbitem := range dbitems {
		dbitemsMap[dbitem.Key] = dbitem
	}
	for _, item := range items {
		dbitem, exist := dbitemsMap[item.Key]
		if !exist {
			if e := UpsertConfigItem(item, db, "syncer_service"); e != nil {
				return e
			}
		} else if dbitem.Value != item.Value || dbitem.Application != item.Application {
			if e := UpsertConfigItem(item, db, "syncer_service"); e != nil {
				return e
			}
		}
	}
	return nil
}

// 将数据库中的数据，同步到配置后端中
func SyncDatabase2Backend(conditem *client.ConfigItem, db *gorm.DB, cli client.ConfigClientIface) error {
	dbitems := []ConfigItem{}
	db.Find(&dbitems, ConfigItem{
		Tenant:      conditem.Tenant,
		Project:     conditem.Project,
		Environment: conditem.Environment,
	})
	cfgItems, err := cli.List(context.Background(), &client.ListOptions{
		ConfigItem: *conditem,
		Page:       1,
		Size:       1000,
	})
	if err != nil {
		return err
	}
	cfgItemsMap := map[string]*client.ConfigItem{}
	for _, cfgItem := range cfgItems {
		cfgItemsMap[cfgItem.Key] = cfgItem
	}
	for _, dbitem := range dbitems {
		if dbitem.Value == "" {
			continue
		}
		cfgItem, exist := cfgItemsMap[dbitem.Key]
		if exist {
			if cfgItem.Value == dbitem.Value {
				continue
			}
		}
		if e := cli.Pub(context.Background(), dbitem.ToClientConfigItem()); e != nil {
			return e
		}
	}
	return nil
}
