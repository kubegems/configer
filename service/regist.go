package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type InfoGetter interface {
	ClusterNameOf(tenant, project, environment string) (clusterName string)
	NacosInfoOf(clusterName string) (addr, username, password string, err error)
	RoundTripperOf(clusterName string) (rt http.RoundTripper)
	Username(c *gin.Context) string
}

type ConfigerHandler struct {
	*ConfigService
	db *gorm.DB
}

type Plugin struct {
	Handler ConfigerHandler
}

func (p *Plugin) InitDatabase() error {
	return Migrate(p.Handler.db)
}

func NewPlugin(infoGetter InfoGetter, db *gorm.DB) (*Plugin, error) {
	handler := &ConfigerHandler{
		ConfigService: NewConfigService(infoGetter, db),
		db:            db,
	}
	return &Plugin{
		Handler: *handler,
	}, nil
}

func (h *ConfigerHandler) RegistRouter(rg *gin.RouterGroup) {
	// list configs
	rg.GET("/configer/tenant/:tenant/project/:project/environment/:environment", h.List)
	// base info
	rg.GET("/configer/tenant/:tenant/project/:project/environment/:environment/baseinfo", h.BaseInfo)
	// get accounts
	rg.GET("/configer/tenant/:tenant/project/:project/environment/:environment/accounts", h.AccountInfo)
	// get config item detail
	rg.GET("/configer/tenant/:tenant/project/:project/environment/:environment/key/:key", h.Get)
	// publish config item
	rg.POST("/configer/tenant/:tenant/project/:project/environment/:environment/key/:key", h.Pub)
	// delete config item
	rg.DELETE("/configer/tenant/:tenant/project/:project/environment/:environment/key/:key", h.Delete)
	// get config item history
	rg.GET("/configer/tenant/:tenant/project/:project/environment/:environment/key/:key/history", h.History)
	// show config item listener
	rg.GET("/configer/tenant/:tenant/project/:project/environment/:environment/key/:key/listener", h.Listener)

	// sync backend data to database
	rg.POST("/configer/tenant/:tenant/project/:project/environment/:environment/action/backup", h.SyncBackend2Database)
	rg.POST("/configer/tenant/:tenant/project/:project/environment/:environment/action/restore", h.SyncDatabase2Backend)

}
