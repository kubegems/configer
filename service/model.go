package service

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"kubegems.io/configer/client"
)

type ConfigItem struct {
	Tenant         string    `gorm:"type:varchar(192);uniqueIndex:idx_config_item_tenant_project_environment_key"`
	Project        string    `gorm:"type:varchar(192);uniqueIndex:idx_config_item_tenant_project_environment_key"`
	Environment    string    `gorm:"type:varchar(192);uniqueIndex:idx_config_item_tenant_project_environment_key"`
	Key            string    `gorm:"type:varchar(192);uniqueIndex:idx_config_item_tenant_project_environment_key"`
	Application    string    `gorm:"type:varchar(255)"`
	Value          string    `gorm:"type:longtext"`
	LastUpdateTime time.Time `gorm:"autoUpdateTime"`
	CreatedTime    time.Time `gorm:"autoCreateTime"`
	LastUpdateUser string    `gorm:"type:varchar(255)"`
}

func (item *ConfigItem) ToClientConfigItem() *client.ConfigItem {
	return &client.ConfigItem{
		Tenant:           item.Tenant,
		Project:          item.Project,
		Environment:      item.Environment,
		Key:              item.Key,
		Application:      item.Application,
		Value:            item.Value,
		LastModifiedTime: item.LastUpdateTime.Format(time.RFC3339),
		CreatedTime:      item.CreatedTime.Format(time.RFC3339),
		LastUpdateUser:   item.LastUpdateUser,
	}
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&ConfigItem{})
}

func UpsertConfigItem(item *client.ConfigItem, db *gorm.DB, username string) error {
	var (
		existOne ConfigItem
		err      error
		exist    bool
	)
	dbitem := &ConfigItem{
		Tenant:      item.Tenant,
		Project:     item.Project,
		Environment: item.Environment,
		Key:         item.Key,
		Application: item.Application,
		Value:       item.Value,
	}
	if username != "" {
		dbitem.LastUpdateUser = username
	}
	cond := ConfigItem{
		Tenant:      item.Tenant,
		Project:     item.Project,
		Environment: item.Environment,
		Key:         item.Key,
	}
	err = db.Find(&existOne, cond).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		exist = true
	}
	if exist {
		if existOne.Application != item.Application || existOne.Value != item.Value {
			existOne.Application = item.Application
			existOne.Value = item.Value
			existOne.LastUpdateUser = dbitem.LastUpdateUser
			err = db.Model(&cond).Updates(existOne).Error
			dbitem = &existOne
		}
	} else {
		err = db.Create(dbitem).Error
	}
	if err != nil {
		return err
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
