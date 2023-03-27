package sregistry

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/pkg/errors"

	"github.com/go-resty/resty/v2"
	"kubegems.io/configer/utils"
)

// 当前直接转发请求
const (
	QueryServicePath        = "/nacos/v1/ns/service"
	QueryServiceListPath    = "/nacos/v1/ns/service/list"
	ServiceInstancePath     = "/nacos/v1/ns/instance"
	ServiceInstanceListPath = "/nacos/v1/ns/instance/list"
)

type NacosServiceRegistryClient struct {
	cli  *resty.Client
	addr string
}

func NewNacosServiceRegistryClient(addr, username, password string, baseRoundTripper http.RoundTripper) (*NacosServiceRegistryClient, error) {
	tokenProvider := utils.NewNacosRoundTripper(addr, username, password, baseRoundTripper)
	cli := resty.New()
	cli.SetTransport(tokenProvider.GetRoundTripper())
	cli.Debug = true
	return &NacosServiceRegistryClient{
		addr: addr,
		cli:  cli,
	}, nil
}

func (nacos *NacosServiceRegistryClient) RetrieveService(query *ServiceQuery, ret interface{}) error {
	resp, err := nacos.cli.R().SetResult(ret).Get(nacos.addr + QueryServicePath + "?" + query.AsQuery())
	if resp.StatusCode() >= 400 {
		return errors.Wrap(err, resp.Status())
	}
	return err
}

func (nacos *NacosServiceRegistryClient) ListServices(query *ServiceListQuery, ret interface{}) error {
	resp, err := nacos.cli.R().SetResult(ret).Get(nacos.addr + QueryServiceListPath + "?" + query.AsQuery())
	if resp.StatusCode() >= 400 {
		return errors.Wrap(err, resp.Status())
	}
	return err
}

func (nacos *NacosServiceRegistryClient) RegistInstance(data *RegistInstanceQuery, ret interface{}) error {
	resp, err := nacos.cli.R().Post(nacos.addr + ServiceInstancePath + "?" + data.AsQuery())
	if resp.StatusCode() >= 400 {
		return errors.Wrap(err, resp.Status())
	}
	return err
}

func (nacos *NacosServiceRegistryClient) DeRegistInstance(instance *DeRegistInstanceQuery, ret interface{}) error {
	resp, err := nacos.cli.R().Delete(nacos.addr + ServiceInstancePath + "?" + instance.AsQuery())
	if resp.StatusCode() >= 400 {
		return errors.Wrap(err, resp.Status())
	}
	return err
}

func (nacos *NacosServiceRegistryClient) ModifyInstance(data *RegistInstanceQuery, ret interface{}) error {
	resp, err := nacos.cli.R().Put(nacos.addr + ServiceInstancePath + "?" + data.AsQuery())
	if resp.StatusCode() >= 400 {
		return errors.Wrap(err, resp.Status())
	}
	return err
}

func (nacos *NacosServiceRegistryClient) ListInstances(query QueryIface, ret interface{}) error {
	resp, err := nacos.cli.R().Get(nacos.addr + ServiceInstanceListPath + "?" + query.AsQuery())
	if resp.StatusCode() >= 400 {
		return errors.Wrap(err, resp.Status())
	}
	return err
}

func (nacos *NacosServiceRegistryClient) RetrieveInstance(query *RetrieveInstanceQuery, ret interface{}) error {
	nacos.cli.R().Get(nacos.addr + ServiceInstancePath + "?" + query.AsQuery())
	return nil
}

var _ ServiceRegistryClientIfe = &NacosServiceRegistryClient{}

type NamespaceGroupNameBase struct {
	GroupName   string `form:"groupName"`
	NamespaceID string `form:"namespaceId"`
}

func (s *NamespaceGroupNameBase) SetGroupName(groupname string) {
	s.GroupName = groupname
}

func (s *NamespaceGroupNameBase) SetNamespace(namespace string) {
	s.NamespaceID = namespace
}

type ServiceQuery struct {
	ServiceName string `form:"serviceName" binding:"required"`
	NamespaceGroupNameBase
}

func (s *NamespaceGroupNameBase) query() url.Values {
	v := url.Values{}
	v.Add("groupName", s.GroupName)
	v.Add("namespaceId", s.NamespaceID)
	return v
}

func (s *ServiceQuery) AsQuery() string {
	v := s.query()
	v.Add("serviceName", s.ServiceName)
	return v.Encode()
}

type ServiceListQuery struct {
	PageNo   int `form:"pageNo" binding:"required"`
	PageSize int `form:"pageSize" binding:"required"`
	NamespaceGroupNameBase
}

func (s *ServiceListQuery) AsQuery() string {
	v := s.query()
	v.Add("pageNo", strconv.Itoa(s.PageNo))
	v.Add("pageSize", strconv.Itoa(s.PageSize))
	return v.Encode()
}

type InstanceCommon struct {
	IP          string `form:"ip" binding:"required"`
	Port        int    `form:"port" binding:"required"`
	ServiceName string `form:"serviceName" binding:"required"`
	NamespaceGroupNameBase
}

type RegistInstanceQuery struct {
	InstanceCommon
	ClusterName string  `form:"clusterName"`
	Ephemeral   bool    `form:"ephemeral"`
	Weight      float64 `form:"weight"`
	Enabled     bool    `form:"enabled"`
	Healthy     bool    `form:"healthy"`
	Metadata    string  `form:"metadata"`
}

type DeRegistInstanceQuery struct {
	ClusterName string `form:"clusterName"`
	Ephemeral   bool   `form:"ephemeral"`
	InstanceCommon
}

type RetrieveInstanceQuery struct {
	InstanceCommon
	Cluster     string `form:"cluster"`
	Ephemeral   bool   `form:"ephemeral"`
	HealthyOnly bool   `form:"healthyOnly"`
}

type ListInstanceQuery struct {
	NamespaceGroupNameBase
	ServiceName string `form:"serviceName"`
	Clusters    string `form:"clusters"`
	HealthyOnly bool   `form:"healthyOnly"`
}

func (s *InstanceCommon) query() url.Values {
	v := s.NamespaceGroupNameBase.query()
	v.Set("ip", s.IP)
	v.Set("port", strconv.Itoa(s.Port))
	v.Set("serviceName", s.ServiceName)
	return v
}

func (s *RegistInstanceQuery) AsQuery() string {
	v := s.query()
	v.Set("clusterName", s.ClusterName)
	v.Set("ephemeral", strconv.FormatBool(s.Ephemeral))
	v.Set("enabled", strconv.FormatBool(s.Enabled))
	v.Set("healthy", strconv.FormatBool(s.Healthy))
	if s.Weight != 0 {
		v.Set("weight", strconv.FormatFloat(s.Weight, 'f', -1, 64))
	}
	if s.Metadata != "" {
		v.Set("metadata", s.Metadata)
	}
	return v.Encode()
}

func (s *DeRegistInstanceQuery) AsQuery() string {
	v := s.query()
	v.Set("clusterName", s.ClusterName)
	v.Set("ephemeral", strconv.FormatBool(s.Ephemeral))
	return v.Encode()
}

func (s *RetrieveInstanceQuery) AsQuery() string {
	v := s.query()
	v.Set("healthyOnly", strconv.FormatBool(s.HealthyOnly))
	return v.Encode()
}

func (s *ListInstanceQuery) AsQuery() string {
	v := s.query()
	v.Set("healthyOnly", strconv.FormatBool(s.HealthyOnly))
	v.Set("serviceName", s.ServiceName)
	v.Set("healthyOnly", strconv.FormatBool(s.HealthyOnly))
	v.Set("clusters", s.Clusters)
	return v.Encode()
}

type MetaSetter interface {
	SetGroupName(groupname string)
	SetNamespace(namespace string)
	AsQuery() string
}

// reverse proxy

func NewReverseProxyService(addr, username, password string, baseRoundTripper http.RoundTripper) (*ReverseProxyService, error) {
	tokenProvider := utils.NewNacosRoundTripper(addr, username, password, baseRoundTripper)
	s := &ReverseProxyService{
		proxyInstance: httputil.NewSingleHostReverseProxy(nil),
		rt:            tokenProvider.GetRoundTripper(),
	}
	s.proxyInstance.Transport = s
	s.proxyInstance.ModifyResponse = s.ModifyResponse
	return s, nil
}

type ReverseProxyService struct {
	proxyInstance *httputil.ReverseProxy
	rt            http.RoundTripper
}

func (s *ReverseProxyService) GetRealPath(originPath, method string) string {
	// TODO: change path
	switch originPath {

	}
	return ""
}

func (s *ReverseProxyService) RoundTrip(req *http.Request) (*http.Response, error) {
	// TODO
	return s.rt.RoundTrip(req)
}

func (s *ReverseProxyService) ModifyResponse(resp *http.Response) error {
	// TODO
	return nil
}

func (s *ReverseProxyService) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	s.proxyInstance.ServeHTTP(rw, req)
}
