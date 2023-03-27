package sregistry

type QueryIface interface {
	AsQuery() string
}

type ServiceRegistryClientIfe interface {
	// 创建服务
	// CreateService()

	// 删除服务
	// DeleteService()

	// 获取服务详情
	RetrieveService(query *ServiceQuery, ret interface{}) error

	// 列出服务
	ListServices(query *ServiceListQuery, ret interface{}) error

	// 更新服务
	// UpdateService()

	// 注册实例到服务
	RegistInstance(instance *RegistInstanceQuery, ret interface{}) error

	// 注销实例
	DeRegistInstance(instance *DeRegistInstanceQuery, ret interface{}) error

	// 修改实例
	ModifyInstance(instance *RegistInstanceQuery, ret interface{}) error

	// 列出实例
	ListInstances(query QueryIface, ret interface{}) error

	// 获取实例详情
	RetrieveInstance(query *RetrieveInstanceQuery, ret interface{}) error
}
