package service

import (
	"crypto/sha1"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"kubegems.io/configer/sregistry"
)

type RegistryService struct {
	cliSets   map[string]sregistry.ServiceRegistryClientIfe
	proxySets map[string]*sregistry.ReverseProxyService
	InfoGetter
	lock sync.Locker
}

func NewRegistryService(infoGetter InfoGetter) *RegistryService {
	return &RegistryService{
		InfoGetter: infoGetter,
		cliSets:    map[string]sregistry.ServiceRegistryClientIfe{},
		lock:       &sync.Mutex{},
	}
}

func (sh *RegistryService) cliOf(c *gin.Context) (sregistry.ServiceRegistryClientIfe, error) {
	clusterName := sh.ClusterNameOf(c.Param("tenant"), c.Param("project"), c.Param("environment"))
	cli, exist := sh.cliSets[clusterName]
	if exist {
		return cli, nil
	}
	sh.lock.Lock()
	defer sh.lock.Unlock()
	addr, uname, password, err := sh.NacosInfoOf(clusterName)
	if err != nil {
		return nil, err
	}
	cli, err = sregistry.NewNacosServiceRegistryClient(addr, uname, password, sh.RoundTripperOf(clusterName))
	if err != nil {
		return nil, err
	}
	sh.cliSets[clusterName] = cli
	return cli, nil
}

func (sh *RegistryService) do(c *gin.Context, q sregistry.MetaSetter, handler func(c *gin.Context, cli sregistry.ServiceRegistryClientIfe)) {
	hash := sha1.New()
	b := "kubegems_" + c.Param("tenant") + "_" + c.Param("project")
	hash.Write([]byte(b))
	c.Param("environment")
	q.SetGroupName(c.Param("environment"))
	q.SetNamespace(fmt.Sprintf("%x", hash.Sum(nil)))
	if err := c.Bind(q); err != nil {
		NotOK(c, err)
		c.Abort()
		return
	}
	cli, err := sh.cliOf(c)
	if err != nil {
		NotOK(c, err)
		c.Abort()
		return
	}
	handler(c, cli)
}

func (h *RegistryService) ListService(c *gin.Context) {
	q := &sregistry.ServiceListQuery{}
	h.do(c, q, func(c *gin.Context, cli sregistry.ServiceRegistryClientIfe) {
		ret := map[string]interface{}{}
		if err := cli.ListServices(q, &ret); err != nil {
			NotOK(c, err)
			return
		}
		OK(c, ret)
	})
}

func (h *RegistryService) GetService(c *gin.Context) {
	q := &sregistry.ServiceQuery{}
	h.do(c, q, func(c *gin.Context, cli sregistry.ServiceRegistryClientIfe) {
		ret := map[string]interface{}{}
		if err := cli.RetrieveService(q, &ret); err != nil {
			NotOK(c, err)
			return
		}
		OK(c, ret)
	})
}

func (h *RegistryService) ListInstances(c *gin.Context) {
	q := &sregistry.ListInstanceQuery{}
	h.do(c, q, func(c *gin.Context, cli sregistry.ServiceRegistryClientIfe) {
		ret := map[string]interface{}{}
		if err := cli.ListInstances(q, &ret); err != nil {
			NotOK(c, err)
			return
		}
		OK(c, ret)
	})
}

func (h *RegistryService) GetInstance(c *gin.Context) {
	q := &sregistry.RetrieveInstanceQuery{}
	h.do(c, q, func(c *gin.Context, cli sregistry.ServiceRegistryClientIfe) {
		ret := ""
		if err := cli.RetrieveInstance(q, &ret); err != nil {
			NotOK(c, err)
			return
		}
		OK(c, ret)
	})
}

func (h *RegistryService) DeleteInstance(c *gin.Context) {
	q := &sregistry.DeRegistInstanceQuery{}
	h.do(c, q, func(c *gin.Context, cli sregistry.ServiceRegistryClientIfe) {
		ret := ""
		if err := cli.DeRegistInstance(q, &ret); err != nil {
			NotOK(c, err)
			return
		}
		OK(c, ret)
	})
}

func (h *RegistryService) ModifyInstance(c *gin.Context) {
	q := &sregistry.RegistInstanceQuery{}
	h.do(c, q, func(c *gin.Context, cli sregistry.ServiceRegistryClientIfe) {
		ret := ""
		if err := cli.ModifyInstance(q, &ret); err != nil {
			NotOK(c, err)
			return
		}
		OK(c, ret)
	})
}

func (h *RegistryService) RegistInstance(c *gin.Context) {
	q := &sregistry.RegistInstanceQuery{}
	h.do(c, q, func(c *gin.Context, cli sregistry.ServiceRegistryClientIfe) {
		ret := ""
		if err := cli.RegistInstance(q, &ret); err != nil {
			NotOK(c, err)
			return
		}
		OK(c, ret)
	})
}

// reverse proxy

func (sh *RegistryService) Proxy(c *gin.Context, q sregistry.MetaSetter) {
	hash := sha1.New()
	b := "kubegems_" + c.Param("tenant") + "_" + c.Param("project")
	hash.Write([]byte(b))
	c.Param("environment")
	q.SetGroupName(c.Param("environment"))
	q.SetNamespace(fmt.Sprintf("%x", hash.Sum(nil)))
	if err := c.Bind(q); err != nil {
		NotOK(c, err)
		c.Abort()
		return
	}
	p, err := sh.proxyOf(c)
	if err != nil {
		NotOK(c, err)
		c.Abort()
		return
	}
	c.Request.URL.RawQuery = q.AsQuery()
	c.Request.URL.Path = p.GetRealPath(c.Request.URL.Path, c.Request.Method)
	p.ServeHTTP(c.Writer, c.Request)
}

func (sh *RegistryService) proxyOf(c *gin.Context) (*sregistry.ReverseProxyService, error) {
	clusterName := sh.ClusterNameOf(c.Param("tenant"), c.Param("project"), c.Param("environment"))
	proxyInstance, exist := sh.proxySets[clusterName]
	if exist {
		return proxyInstance, nil
	}
	sh.lock.Lock()
	defer sh.lock.Unlock()
	addr, uname, password, err := sh.NacosInfoOf(clusterName)
	if err != nil {
		return nil, err
	}
	proxyInstance, err = sregistry.NewReverseProxyService(addr, uname, password, sh.RoundTripperOf(clusterName))
	if err != nil {
		return nil, err
	}
	sh.proxySets[clusterName] = proxyInstance
	return proxyInstance, nil
}
