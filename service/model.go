package service

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"kubegems.io/configer/client"
)

type ConfigItem struct {
	Tenant      string `gorm:"type:varchar(255);uniqueIndex:idx_config_item_tenant_project_environment_key"`
	Project     string `gorm:"type:varchar(255);uniqueIndex:idx_config_item_tenant_project_environment_key"`
	Environment string `gorm:"type:varchar(255);uniqueIndex:idx_config_item_tenant_project_environment_key"`
	Key         string `gorm:"type:varchar(255);uniqueIndex:idx_config_item_tenant_project_environment_key"`
	Application string `gorm:"type:varchar(255)"`
	// Value          string    `gorm:"type:longtext"`
	LastUpdateTime time.Time `gorm:"autoUpdateTime"`
	CreatedTime    time.Time `gorm:"autoCreateTime"`
	LastUpdateUser string    `gorm:"type:varchar(255)"`
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&ConfigItem{})
}

func UpsertConfigItem(item *client.ConfigItem, db *gorm.DB, username string) error {
	dbitem := &ConfigItem{
		Tenant:      item.Tenant,
		Project:     item.Project,
		Environment: item.Environment,
		Key:         item.Key,
		Application: item.Application,
		// Value:       item.Value,
		LastUpdateUser: username,
	}
	e := db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(dbitem).Error
	if e != nil {
		return e
	}
	item.LastUpdateUser = username
	item.LastModifiedTime = dbitem.LastUpdateTime.Format(time.RFC3339)
	item.CreatedTime = dbitem.CreatedTime.Format(time.RFC3339)
	return nil
}

func DeleteConfigItem(item *client.ConfigItem, db *gorm.DB) error {
	return db.Delete(&ConfigItem{}, ConfigItem{
		Tenant:      item.Tenant,
		Project:     item.Project,
		Environment: item.Environment,
		Key:         item.Key,
	}).Error
}

func FillDates(conditem *client.ConfigItem, items []*client.ConfigItem, db *gorm.DB) error {
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
			continue
		}
		item.LastModifiedTime = dbitem.LastUpdateTime.Format(time.RFC3339)
		item.CreatedTime = dbitem.CreatedTime.Format(time.RFC3339)
		item.LastUpdateUser = dbitem.LastUpdateUser
	}
	return nil
}
