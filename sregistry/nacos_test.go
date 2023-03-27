package sregistry

import (
	"os"
	"testing"
)

var nacos *NacosServiceRegistryClient

const (
	s1 = "test"
	// sha1(kubegems_test_test)
	n1 = "59bd042678a226516a50e3badb0ac726b91cf393"

	s2 = "test2"
	// sha1(kubegems_test_test_1)
	n2 = "2d237ac8af508173667ad349d4a563b7a95b893f"
)

func getServiceQuery(serviceName, nsId string) *ServiceQuery {
	return &ServiceQuery{
		ServiceName: serviceName,
		NamespaceGroupNameBase: NamespaceGroupNameBase{
			NamespaceID: nsId,
		},
	}
}

func listServiceQuery(nsId string) *ServiceListQuery {
	return &ServiceListQuery{
		PageNo:   1,
		PageSize: 10,
		NamespaceGroupNameBase: NamespaceGroupNameBase{
			NamespaceID: nsId,
		},
	}
}

func registInstanceQuery(serviceName, nsId string) *RegistInstanceQuery {
	return &RegistInstanceQuery{
		Weight: 0.4,
		InstanceCommon: InstanceCommon{
			IP:          "172.16.1.1",
			Port:        9999,
			ServiceName: serviceName,
			NamespaceGroupNameBase: NamespaceGroupNameBase{
				NamespaceID: nsId,
			},
		},
	}

}

func modifyInstanceQuery(serviceName, nsId string) *RegistInstanceQuery {
	return &RegistInstanceQuery{
		Weight: 0.5,
		InstanceCommon: InstanceCommon{
			IP:          "172.16.1.1",
			Port:        9999,
			ServiceName: serviceName,
			NamespaceGroupNameBase: NamespaceGroupNameBase{
				NamespaceID: nsId,
			},
		},
	}
}

func retrieveInstanceQuery(serviceName, nsId string) *RetrieveInstanceQuery {
	return &RetrieveInstanceQuery{
		InstanceCommon: InstanceCommon{
			IP:          "172.16.1.1",
			Port:        9999,
			ServiceName: serviceName,
			NamespaceGroupNameBase: NamespaceGroupNameBase{
				NamespaceID: nsId,
			},
		},
	}
}

func deleteInstanceQuery(serviceName, nsId string) *DeRegistInstanceQuery {
	return &DeRegistInstanceQuery{
		InstanceCommon: InstanceCommon{
			IP:          "172.16.1.1",
			Port:        9999,
			ServiceName: serviceName,
			NamespaceGroupNameBase: NamespaceGroupNameBase{
				NamespaceID: nsId,
			},
		},
	}
}

func listInstanceQuery(serviceName, nsId string) *ListInstanceQuery {
	return &ListInstanceQuery{
		ServiceName: serviceName,
		NamespaceGroupNameBase: NamespaceGroupNameBase{
			NamespaceID: nsId,
		},
	}
}

func TestMain(m *testing.M) {
	n, err := NewNacosServiceRegistryClient("http://127.0.0.1:8848", "nacos", "nacos", nil)
	if err != nil {
		panic(err)
	}
	nacos = n
	os.Exit(m.Run())
}

func TestNacosServiceRegistryClient_All(t *testing.T) {
	registInstanceRet := map[string]interface{}{}
	if e := nacos.RegistInstance(registInstanceQuery(s2, n2), registInstanceRet); e != nil {
		t.Error(e)
	}
	serviceRet := map[string]interface{}{}
	if e := nacos.RetrieveService(getServiceQuery(s2, n2), serviceRet); e != nil {
		t.Error(e)
	}
	listServiceRet := map[string]interface{}{}
	if e := nacos.ListServices(listServiceQuery(n2), listServiceRet); e != nil {
		t.Error(e)
	}
	modifyInstanceRet := map[string]interface{}{}
	if e := nacos.ModifyInstance(modifyInstanceQuery(s2, n2), modifyInstanceRet); e != nil {
		t.Error(e)
	}
	listInstanceRet := map[string]interface{}{}
	if e := nacos.ListInstances(listInstanceQuery(s2, n2), listInstanceRet); e != nil {
		t.Error(e)
	}
	deleteInstanceRet := map[string]interface{}{}
	if e := nacos.DeRegistInstance(deleteInstanceQuery(s2, n2), deleteInstanceRet); e != nil {
		t.Error(e)
	}
}

func TestNacosServiceRegistryClient_RetrieveService(t *testing.T) {
	ret := map[string]interface{}{}
	if err := nacos.RetrieveService(getServiceQuery(s1, n1), ret); err != nil {
		t.Error(err)
	}
}

func TestNacosServiceRegistryClient_ListServices(t *testing.T) {
	ret := map[string]interface{}{}
	if err := nacos.ListServices(listServiceQuery(n1), ret); err != nil {
		t.Error(err)
	}
}

func TestNacosServiceRegistryClient_RegistInstance(t *testing.T) {
	ret := map[string]interface{}{}
	if err := nacos.RegistInstance(registInstanceQuery(s1, n1), ret); err != nil {
		t.Error(err)
	}
}

func TestNacosServiceRegistryClient_ModifyInstance(t *testing.T) {
	ret := map[string]interface{}{}
	if err := nacos.ModifyInstance(modifyInstanceQuery(s1, n1), ret); err != nil {
		t.Error(err)
	}
}

func TestNacosServiceRegistryClient_RetrieveInstance(t *testing.T) {
	ret := map[string]interface{}{}
	if err := nacos.RetrieveInstance(retrieveInstanceQuery(s1, n1), ret); err != nil {
		t.Error(err)
	}
}

func TestNacosServiceRegistryClient_ListInstances(t *testing.T) {
	ret := map[string]interface{}{}
	if err := nacos.ListInstances(listInstanceQuery(s1, n1), ret); err != nil {
		t.Error(err)
	}
}

func TestNacosServiceRegistryClient_DeRegistInstance(t *testing.T) {
	ret := map[string]interface{}{}
	if err := nacos.DeRegistInstance(deleteInstanceQuery(s1, n1), ret); err != nil {
		t.Error(err)
	}
}
