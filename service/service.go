package service

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kubegems.io/configer/client"
)

type ConfigService struct {
	clients map[string]client.ConfigClientIface
	InfoGetter
	db *gorm.DB
}

func NewConfigService(infoGetter InfoGetter, db *gorm.DB) *ConfigService {
	return &ConfigService{
		clients:    make(map[string]client.ConfigClientIface),
		InfoGetter: infoGetter,
		db:         db,
	}
}

type ResponseStruct struct {
	Message   string
	Data      interface{}
	ErrorData interface{}
}

func OK(ctx *gin.Context, data interface{}) {
	ctx.JSON(200, ResponseStruct{
		Message:   "OK",
		Data:      data,
		ErrorData: nil,
	})
}

func NotOK(ctx *gin.Context, err error) {
	ctx.JSON(400, ResponseStruct{
		Message:   err.Error(),
		Data:      nil,
		ErrorData: err.Error(),
	})
}

func (cs *ConfigService) ClientOf(item *client.ConfigItem) (string, client.ConfigClientIface, error) {
	clusterName := cs.InfoGetter.ClusterNameOf(item.Tenant, item.Project, item.Environment)
	if client, ok := cs.clients[clusterName]; ok {
		return clusterName, client, nil
	}
	addr, uname, password, err := cs.InfoGetter.NacosInfoOf(clusterName)
	if err != nil {
		return clusterName, nil, err
	}
	rt := cs.InfoGetter.RoundTripperOf(clusterName)
	// TODO: adapt for more service
	client, err := client.NewNacosService(addr, uname, password, rt)
	if err != nil {
		return clusterName, nil, err
	}
	cs.clients[clusterName] = client
	return clusterName, client, nil
}

func paramOrQuery(c *gin.Context, key string) string {
	if c.Param(key) != "" {
		return c.Param(key)
	}
	return c.Query(key)
}

func buildConfigItemFromReq(c *gin.Context) *client.ConfigItem {
	item := &client.ConfigItem{}
	c.ShouldBindJSON(item)
	rev, _ := strconv.ParseInt(c.Query("rev"), 10, 64)
	item.Tenant = paramOrQuery(c, "tenant")
	item.Project = paramOrQuery(c, "project")
	item.Environment = paramOrQuery(c, "environment")
	item.Application = paramOrQuery(c, "application")
	item.Key = paramOrQuery(c, "key")
	item.Rev = rev
	return item
}

func (cs *ConfigService) withItem(ctx *gin.Context, item *client.ConfigItem, f func(ctx *gin.Context, cli client.ConfigClientIface) error) error {
	clusterName, client, err := cs.ClientOf(item)
	cs.setAuditData(ctx, clusterName, item.Tenant, item.Project, item.Environment, item.Application)

	if err != nil {
		NotOK(ctx, err)
		return err
	}
	return f(ctx, client)
}

func (cs *ConfigService) BaseInfo(c *gin.Context) {
	item := buildConfigItemFromReq(c)
	_, client, err := cs.ClientOf(item)
	if err != nil {
		NotOK(c, err)
		return
	}
	data, err := client.BaseInfo(c, item)
	if err != nil {
		NotOK(c, err)
		return
	}
	OK(c, data)
}

func (cs *ConfigService) Get(c *gin.Context) {
	item := buildConfigItemFromReq(c)
	if err := cs.withItem(c, item, func(ctx *gin.Context, cli client.ConfigClientIface) error {
		return cli.Get(ctx, item)
	}); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, item)
}

func (cs *ConfigService) Pub(c *gin.Context) {
	item := buildConfigItemFromReq(c)
	if err := cs.withItem(c, item, func(ctx *gin.Context, cli client.ConfigClientIface) error {
		c.Set("audit_subject", map[string]string{
			"action": "发布",
			"module": "配置项",
			"name":   item.Key,
		})
		if e := cli.Pub(c, item); e != nil {
			return e
		} else {
			return UpsertConfigItem(item, cs.db, cs.Username(c))
		}
	}); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, item)
}

func (cs *ConfigService) Delete(c *gin.Context) {
	item := buildConfigItemFromReq(c)
	if err := cs.withItem(c, item, func(ctx *gin.Context, cli client.ConfigClientIface) error {
		c.Set("audit_subject", map[string]string{
			"action": "删除",
			"module": "配置项",
			"name":   item.Key,
		})
		if e := cli.Delete(c, item); e != nil {
			return e
		} else {
			return DeleteConfigItem(item, cs.db)
		}
	}); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, item)
}

func (cs *ConfigService) List(c *gin.Context) {
	item := buildConfigItemFromReq(c)
	page, perr := strconv.Atoi(c.Query("page"))
	if perr != nil {
		page = 1
	}
	size, serr := strconv.Atoi(c.Query("size"))
	if serr != nil {
		size = 10
	}
	cs.withItem(c, item, func(ctx *gin.Context, cli client.ConfigClientIface) error {
		data, err := cli.List(c, &client.ListOptions{
			ConfigItem: *item,
			Page:       page,
			Size:       size,
		})
		FillDates(item, data, cs.db)
		if err != nil {
			NotOK(ctx, err)
		} else {
			OK(ctx, data)
		}
		return err
	})
}

func (cs *ConfigService) History(c *gin.Context) {
	item := buildConfigItemFromReq(c)
	cs.withItem(c, item, func(ctx *gin.Context, cli client.ConfigClientIface) error {
		data, err := cli.History(c, item)
		if err != nil {
			NotOK(ctx, err)
		} else {
			OK(ctx, data)
		}
		return err
	})
}

func (cs *ConfigService) Listener(c *gin.Context) {
	item := buildConfigItemFromReq(c)
	cs.withItem(c, item, func(ctx *gin.Context, cli client.ConfigClientIface) error {
		data, err := cli.Listener(c, item)
		if err != nil {
			NotOK(ctx, err)
		} else {
			OK(ctx, data)
		}
		return err
	})
}

func (cs *ConfigService) AccountInfo(c *gin.Context) {
	item := buildConfigItemFromReq(c)
	cs.withItem(c, item, func(ctx *gin.Context, cli client.ConfigClientIface) error {
		data, err := cli.Accounts(item)
		if err != nil {
			NotOK(ctx, err)
		} else {
			OK(ctx, data)
		}
		return err
	})
}

func (cs *ConfigService) setAuditData(c *gin.Context, clusterName, tenant, project, environment, application string) {
	auditExtraDatas := map[string]string{
		"tenant":      tenant,
		"project":     project,
		"environment": environment,
		"cluster":     clusterName,
	}
	if application != "" {
		auditExtraDatas["application"] = application
	}
	c.Set("audit_extra_datas", auditExtraDatas)
}

func (cs *ConfigService) SyncBackend2Database(c *gin.Context) {
	item := buildConfigItemFromReq(c)
	if err := cs.withItem(c, item, func(ctx *gin.Context, cli client.ConfigClientIface) error {
		c.Set("audit_subject", map[string]string{
			"action": "备份",
			"module": "环境下的配置项",
			"name":   item.Environment,
		})

		datas, err := cli.List(c, &client.ListOptions{
			ConfigItem: *item,
			Page:       1,
			Size:       1000,
		})
		if err != nil {
			return err
		}
		return SyncBackend2Database(item, datas, cs.db)
	}); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, item)
}

func (cs *ConfigService) SyncDatabase2Backend(c *gin.Context) {
	item := buildConfigItemFromReq(c)
	if err := cs.withItem(c, item, func(ctx *gin.Context, cli client.ConfigClientIface) error {
		c.Set("audit_subject", map[string]string{
			"action": "恢复",
			"module": "环境下的配置项",
			"name":   item.Environment,
		})
		return SyncDatabase2Backend(item, cs.db, cli)
	}); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, item)
}
