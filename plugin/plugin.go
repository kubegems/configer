package plugin

import "fmt"

type ConfigServerProvider string

const (
	NacosProvider  ConfigServerProvider = "nacos"
	EtcdProvider   ConfigServerProvider = "etcd"
	ConsulProvider ConfigServerProvider = "consul"
)

type ConfigServerPlugin struct {
	Provider  ConfigServerProvider
	External  bool
	Endpoints []string
	Username  string
	Password  string
}

func (plugin *ConfigServerPlugin) CheckConnection() error {
	return nil
}

func (plugin *ConfigServerPlugin) Install() error {
	if plugin.External {
		return fmt.Errorf("external config server not support to install")
	}
	// todo: apply plugin.kubegems.io
	return nil
}

func (plugin *ConfigServerPlugin) Uninstall() error {
	if plugin.External {
		return fmt.Errorf("external config server not support to uninstall")
	}
	// todo: delete plugin.kubegems.io
	return nil
}
