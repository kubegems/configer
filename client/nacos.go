package client

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

/*
	Nacos version requred: 2.0.3 +
	mapping:
	   	tenant = /t/{gems.tenant}/{gems.project}
	   	group  = {gems.application}
	   	application = {gems.application}
	   	dataID = {gems.key}
*/

const (
	LOGIN_PATH      = "/nacos/v1/auth/login"
	CONFIG_PATH     = "/nacos/v1/cs/configs"
	LISTENER_PATH   = "/nacos/v1/cs/configs/listener"
	HISTORY_PATH    = "/nacos/v1/cs/history"
	TENANT_PATH     = "/nacos/v1/console/namespaces"
	USER_PATH       = "/nacos/v1/auth/users"
	PERM_PATH       = "/nacos/v1/auth/permissions"
	ROLE_PATH       = "/nacos/v1/auth/roles"
	DEFAULT_PAGE    = 1
	DEFAULT_SIZE    = 500
	TenantCacheTime = time.Minute * 10
	UsersCacheTime  = time.Minute * 30
	PermsCacheTime  = time.Minute * 30
	RolesCacheTime  = time.Minute * 30
)

type traverseFunc func(page, size int) error

type NacosService struct {
	client    *http.Client
	addr      string
	username  string
	password  string
	authInfo  *AuthInfo
	lastLogin time.Time
	once      sync.Once

	baseRoundTripper http.RoundTripper
	tenants          []*NacosNamespace
	users            []*NacosUser
	perms            []*NacosPerm
	roles            []*NacosRole

	tenantsLastUpdateTime *time.Time
	usersLastUpdateTime   *time.Time
	permsLastUpdateTime   *time.Time
	rolesLastUpdateTime   *time.Time
	syncLock              sync.Mutex
}

type roundTripWrapper struct {
	proxy             func(*http.Request) (*url.URL, error)
	innerRoundWrapper http.RoundTripper
}

func (rw roundTripWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	wrapperdReq, _ := rw.proxy(req)
	req.URL = wrapperdReq
	req.Header.Add("namespace", "nacos")
	req.Header.Add("service", "nacos-client")
	req.Header.Add("port", "8848")
	return rw.innerRoundWrapper.RoundTrip(req)
}

func NewNacosService(addr, username, password string, baseRoundTripper http.RoundTripper) (*NacosService, error) {
	nacos := &NacosService{
		client:           &http.Client{},
		username:         username,
		password:         password,
		addr:             addr,
		once:             sync.Once{},
		baseRoundTripper: baseRoundTripper,
		tenants:          []*NacosNamespace{},
		users:            []*NacosUser{},
		perms:            []*NacosPerm{},
		roles:            []*NacosRole{},
		syncLock:         sync.Mutex{},
	}
	if err := nacos.login(); err != nil {
		return nil, err
	}
	fn := func(r *http.Request) (*url.URL, error) {
		if time.Now().Unix()-nacos.lastLogin.Unix() >= nacos.authInfo.TokenTTL {
			nacos.login()
		}
		q := r.URL.Query()
		q.Add("accessToken", nacos.authInfo.AccessToken)
		r.URL.RawQuery = q.Encode()
		return r.URL, nil
	}
	if baseRoundTripper != nil {
		roundTripper := roundTripWrapper{
			proxy:             fn,
			innerRoundWrapper: baseRoundTripper,
		}
		nacos.client.Transport = roundTripper
	}
	return nacos, nil
}

func (nacos *NacosService) getHistory(ctx context.Context, mapper *NacosDataMapper) (string, error) {
	resp, err := nacos.client.Get(nacos.urlFor(mapper, HISTORY_PATH))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get history failed, code is %d", resp.StatusCode)
	}
	data := &NacosConfigItem{}
	if err := json.NewDecoder(resp.Body).Decode(data); err != nil {
		return "", err
	}
	return data.Content, nil
}

func (nacos *NacosService) BaseInfo(ctx context.Context, item *ConfigItem) (map[string]string, error) {
	baseMap := map[string]string{
		"provider": "nacos",
	}
	mapper, err := mapperForNacos(item)
	if err != nil {
		return baseMap, err
	}
	baseMap["nacos_tenant"] = mapper.TenantID()
	baseMap["nacos_group"] = mapper.Group()
	return baseMap, nil
}

func (nacos *NacosService) Get(ctx context.Context, item *ConfigItem) error {
	mapper, err := mapperForNacos(item)
	if err != nil {
		return err
	}
	if err := nacos.preAction(mapper); err != nil {
		return err
	}
	if mapper.Rev() != "" {
		content, err := nacos.getHistory(ctx, mapper)
		if err != nil {
			return err
		}
		item.Value = content
		return nil

	}
	resp, err := nacos.client.Get(nacos.urlFor(mapper, CONFIG_PATH))
	if err != nil {
		return err
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	item.Value = string(content)
	return nil
}

func (nacos *NacosService) Pub(ctx context.Context, item *ConfigItem) error {
	mapper, err := mapperForNacos(item)
	if err != nil {
		return err
	}
	if err := nacos.preAction(mapper); err != nil {
		return err
	}
	resp, err := nacos.client.PostForm(
		nacos.urlFor(mapper, CONFIG_PATH),
		url.Values{"content": []string{item.Value}},
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("create config failed, code is %d", resp.StatusCode)
	}
	return nil
}

func (nacos *NacosService) Delete(ctx context.Context, item *ConfigItem) error {
	mapper, err := mapperForNacos(item)
	if err != nil {
		return err
	}
	if err := nacos.preAction(mapper); err != nil {
		return err
	}
	req, _ := http.NewRequest(http.MethodDelete, nacos.urlFor(mapper, CONFIG_PATH), nil)
	resp, err := nacos.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete config failed, code is %d", resp.StatusCode)
	}
	return nil
}

func (nacos *NacosService) History(ctx context.Context, item *ConfigItem) ([]*HistoryVersion, error) {
	mapper, err := mapperForNacos(item)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Add("search", "accurate")
	q.Add("group", mapper.Group())
	q.Add("dataId", mapper.DataID())
	q.Add("tenant", mapper.TenantID())
	uri := nacos.addr + HISTORY_PATH + "?" + q.Encode()
	resp, err := nacos.client.Get(uri)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get history of config failed, code is %d", resp.StatusCode)
	}
	versions := &[]*NacosConfigItem{}
	data := NacosListStruct{
		PageItems: versions,
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode history of config failed, %s", err.Error())
	}
	ret := make([]*HistoryVersion, len(*versions))
	for idx, ver := range *versions {
		ret[idx] = &HistoryVersion{
			Rev:            ver.ID,
			Version:        ver.ID,
			LastUpdateTime: ver.LastModifiedTime,
		}
	}
	return ret, nil
}

func (nacos *NacosService) List(ctx context.Context, opts *ListOptions) ([]*ConfigItem, error) {
	q := url.Values{}
	mapper, err := mapperForNacos(&opts.ConfigItem)
	if err != nil {
		return nil, err
	}
	q.Add("tenant", mapper.TenantID())
	var (
		page, size int
	)
	page = opts.Page
	if opts.Page <= 0 {
		page = 1
	}
	size = opts.Size
	if opts.Size <= 0 {
		size = 10
	}
	q.Add("group", opts.Environment)
	q.Add("search", "accurate")
	q.Add("dataId", opts.Key)
	q.Add("pageNo", strconv.Itoa(page))
	q.Add("pageSize", strconv.Itoa(size))
	if opts.Application != "" {
		q.Add("appName", opts.Application)
	}
	req, _ := http.NewRequest(http.MethodGet, nacos.addr+CONFIG_PATH, nil)
	req.URL.RawQuery = q.Encode()
	resp, err := nacos.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list config failed, code is %d", resp.StatusCode)
	}
	respData := NacosListStruct{
		PageItems: &[]*NacosConfigItem{},
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, err
	}
	ret := []*ConfigItem{}
	cmlist := respData.PageItems.(*[]*NacosConfigItem)
	for _, item := range *cmlist {
		kitem := &ConfigItem{
			Tenant:           opts.ConfigItem.Tenant,
			Project:          opts.ConfigItem.Project,
			Application:      item.AppName,
			Environment:      item.Group,
			Key:              item.DataID,
			Value:            item.Content,
			CreatedTime:      item.CreatedTime,
			LastModifiedTime: item.LastModifiedTime,
		}
		ret = append(ret, kitem)
	}
	return ret, nil
}

func (nacos *NacosService) Accounts(item *ConfigItem) ([]Account, error) {
	mapper, err := mapperForNacos(item)
	if err != nil {
		return nil, err
	}
	rUser, rwUser, _, _, _ := nacos.userRolesFor(mapper)
	return []Account{
		{
			Username: rUser,
			Password: GenPassword(rUser),
		},
		{
			Username: rwUser,
			Password: GenPassword(rwUser),
		},
	}, nil
}

func (nacos *NacosService) Listener(ctx context.Context, item *ConfigItem) (map[string]string, error) {
	mapper, err := mapperForNacos(item)
	if err != nil {
		return nil, err
	}
	resp, err := nacos.client.Get(nacos.urlFor(mapper, LISTENER_PATH))
	if err != nil {
		return nil, err
	}
	respData := &ListenerResp{
		ListenersGroupkeyStatus: map[string]string{},
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, err
	}
	return respData.ListenersGroupkeyStatus, nil
}

type ListenerResp struct {
	CollectStatus           int               `json:"collectStatus"`
	ListenersGroupkeyStatus map[string]string `json:"lisentersGroupkeyStatus"`
}

type AuthInfo struct {
	AccessToken string `json:"accessToken"`
	TokenTTL    int64  `json:"tokenTtl"`
	GlobalAdmin bool   `json:"globalAdmin"`
}

type PubConfig struct {
	Tenant  string `json:"tenant"`
	DataId  string `json:"dataId"`
	Group   string `json:"group"`
	Content string `json:"content"`
}

type NacosNamespace struct {
	Namespace         string `json:"namespace"`
	NamepsaceShowName string `json:"namespaceShowName"`
	Quota             int    `json:"quota"`
	ConfigCount       int    `json:"configCount"`
	Type              int    `json:"type"`
}

type NacosUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type NacosPerm struct {
	Role     string `json:"role"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type NacosRole struct {
	Role     string `json:"role"`
	Username string `json:"username"`
}

type NacosListStruct struct {
	TotalCount     int         `json:"totalCount"`
	PageNumber     int         `json:"pageNumber"`
	PagesAvailable int         `json:"pagesAvailable"`
	PageItems      interface{} `json:"pageItems"`
}

type ListNacosConfigItemResp struct {
	PageItems []*NacosConfigItem `json:"pageItems"`
}

type NacosConfigItem struct {
	ID               string `json:"id"`
	DataID           string `json:"dataId"`
	Group            string `json:"group"`
	Content          string `json:"content"`
	Md5              string `json:"md5"`
	Tenant           string `json:"tenant"`
	AppName          string `json:"appName"`
	Type             string `json:"type"`
	CreatedTime      string `json:"createdTime"`
	LastModifiedTime string `json:"lastModifiedTime"`
}

func (nacos *NacosService) preAction(mapper *NacosDataMapper) error {
	/*
		每次操作前, 需要确保外围数据存在
		1. 是否存在租户, 不存在就创建
		2. 是否存在用户, 不存在就创建
		3. 是否存在权限, 不存在就创建
		4. 是否存在资源权限绑定，不存在就创建

		五分钟缓存过期
	*/
	var (
		existTenant, existRUser, existRWUser, existRRole, existRWRole, existRPerm, existRWPerm bool
	)
	rUser, rwUser, rRole, rwRole, resource := nacos.userRolesFor(mapper)
	tenantName, tenantID := mapper.Tenant(), mapper.TenantID()
	now := time.Now()
	if now.Sub(*nacos.tenantsLastUpdateTime).Nanoseconds() > TenantCacheTime.Nanoseconds() {
		existTenant = false
	} else {
		for _, exist := range nacos.tenants {
			if tenantName == exist.NamepsaceShowName && tenantID == exist.Namespace {
				existTenant = true
				break
			}
		}
	}

	if now.Sub(*nacos.usersLastUpdateTime).Nanoseconds() > UsersCacheTime.Nanoseconds() {
		existRUser = false
		existRWUser = false
	} else {
		for _, user := range nacos.users {
			if user.Username == rwUser {
				existRWUser = true
				break
			}
		}
		for _, user := range nacos.users {
			if user.Username == rUser {
				existRUser = true
				break
			}
		}
	}

	if now.Sub(*nacos.rolesLastUpdateTime).Nanoseconds() > RolesCacheTime.Nanoseconds() {
		existRRole = false
		existRWRole = false
	} else {
		for _, role := range nacos.roles {
			if role.Role == rRole && role.Username == rUser {
				existRRole = true
				break
			}
		}
		for _, role := range nacos.roles {
			if role.Role == rwRole && role.Username == rwUser {
				existRWRole = true
				break
			}
		}
	}

	if now.Sub(*nacos.permsLastUpdateTime).Nanoseconds() > PermsCacheTime.Nanoseconds() {
		existRPerm = false
		existRWPerm = false
	} else {
		for _, perm := range nacos.perms {
			if perm.Resource == resource && perm.Role == rRole && perm.Action == "r" {
				existRPerm = true
				break
			}
		}
		for _, perm := range nacos.perms {
			if perm.Resource == resource && perm.Role == rwRole && perm.Action == "rw" {
				existRWPerm = true
				break
			}
		}
	}

	if existTenant && existRUser && existRWUser && existRRole && existRWRole && existRPerm && existRWPerm {
		return nil
	}

	nacos.syncLock.Lock()
	defer nacos.syncLock.Unlock()
	if !existTenant {
		tenantList, err := nacos.listTenant()
		if err != nil {
			return fmt.Errorf("list tenant failed, %s", err)
		}
		for _, ten := range tenantList {
			if ten.Namespace == tenantID && ten.NamepsaceShowName == tenantName {
				nacos.tenants = tenantList
				goto TenantCheckDone
			}
		}
		if err := nacos.createTenant(mapper.TenantID(), tenantName); err != nil {
			return fmt.Errorf("create tenant failed, %s", err)
		}
		newTenantList, err := nacos.listTenant()
		if err != nil {
			return fmt.Errorf("list tenant failed, %s", err)
		}
		nacos.tenants = newTenantList
		nacos.tenantsLastUpdateTime = &now
	}
TenantCheckDone:

	if !existRUser || !existRWUser {
		var extr, extrw bool
		userList, err := nacos.listUsers()
		if err != nil {
			return fmt.Errorf("list users failed, %s", err)
		}
		for _, user := range userList {
			if user.Username == rUser {
				extr = true
			}
			if user.Username == rwUser {
				extrw = true
			}
			if extr && extrw {
				goto UserCheckDone
			}
		}
		if !extr {
			if err := nacos.createUser(rUser); err != nil {
				return fmt.Errorf("create read user failed, %s", err)
			}
		}
		if !extrw {
			if err := nacos.createUser(rwUser); err != nil {
				return fmt.Errorf("create operator user failed, %s", err)
			}
		}
		newUserList, err := nacos.listUsers()
		if err != nil {
			return fmt.Errorf("list user failed, %s", err)
		}
		nacos.users = newUserList
		nacos.usersLastUpdateTime = &now
	}
UserCheckDone:

	if !existRRole || !existRWRole {
		var extr, extrw bool
		roleList, err := nacos.listRoles()
		if err != nil {
			return fmt.Errorf("list role failed, %s", err)
		}
		for _, role := range roleList {
			if role.Role == rRole && role.Username == rUser {
				extr = true
			}
			if role.Role == rwRole && role.Username == rwUser {
				extrw = true
			}
			if extr && extrw {
				goto RoleCheckDone
			}
		}
		if !extr {
			if err := nacos.createRole(rRole, rUser); err != nil {
				return fmt.Errorf("create read role failed, %s", err)
			}
		}
		if !extrw {
			if err := nacos.createRole(rwRole, rwUser); err != nil {
				return fmt.Errorf("create operator role failed, %s", err)
			}
		}
		newRoleList, err := nacos.listRoles()
		if err != nil {
			return fmt.Errorf("list role failed, %s", err)
		}
		nacos.roles = newRoleList
		nacos.rolesLastUpdateTime = &now
	}
RoleCheckDone:

	if !existRPerm || !existRWPerm {
		var extr, extrw bool
		permList, err := nacos.listPerms()
		if err != nil {
			return fmt.Errorf("list perm failed, %s", err)
		}
		for _, perm := range permList {
			if perm.Role == rRole && perm.Action == "r" && perm.Resource == resource {
				extr = true
			}
			if perm.Role == rwRole && perm.Action == "rw" && perm.Resource == resource {
				extrw = true
			}
			if extr && extrw {
				goto PermCheckDone
			}
		}
		if !extr {
			if err := nacos.createPerm(rRole, resource, "r"); err != nil {
				return fmt.Errorf("create read perm failed, %s", err)
			}
		}
		if !extrw {
			if err := nacos.createPerm(rwRole, resource, "rw"); err != nil {
				return fmt.Errorf("create operator perm failed, %s", err)
			}
		}
		newPermList, err := nacos.listPerms()
		if err != nil {
			return fmt.Errorf("list perm failed, %s", err)
		}
		nacos.perms = newPermList
		nacos.permsLastUpdateTime = &now
	}
PermCheckDone:
	return nil
}

func (nacos *NacosService) listUsers() ([]*NacosUser, error) {
	ret := []*NacosUser{}
	err := nacos.listFunc(USER_PATH, func() interface{} { return &[]*NacosUser{} }, func(ptr interface{}) {
		list := ptr.(*[]*NacosUser)
		ret = append(ret, *list...)
	})
	return ret, err
}

func (nacos *NacosService) listPerms() ([]*NacosPerm, error) {
	ret := []*NacosPerm{}
	nacos.listFunc(PERM_PATH, func() interface{} { return &[]*NacosPerm{} }, func(ptr interface{}) {
		list := ptr.(*[]*NacosPerm)
		ret = append(ret, *list...)
	})
	return ret, nil
}

func (nacos *NacosService) listRoles() ([]*NacosRole, error) {
	ret := []*NacosRole{}
	nacos.listFunc(ROLE_PATH, func() interface{} { return &[]*NacosRole{} }, func(ptr interface{}) {
		list := ptr.(*[]*NacosRole)
		ret = append(ret, *list...)
	})
	return ret, nil
}

func (nacos *NacosService) listFunc(path string, f1 func() interface{}, f2 func(interface{})) error {
	var f traverseFunc
	f = func(page, size int) error {
		req, _ := http.NewRequest(http.MethodGet, nacos.addr+path, nil)
		q := url.Values{}
		q.Add("pageNo", strconv.Itoa(page))
		q.Add("pageSize", strconv.Itoa(size))
		req.URL.RawQuery = q.Encode()
		resp, err := nacos.client.Do(req)
		if err != nil {
			return nil
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to list users, code is %d", resp.StatusCode)
		}
		ptr := f1()
		respData := NacosListStruct{
			PageItems: ptr,
		}
		if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
			return err
		}
		f2(ptr)
		if page < respData.PagesAvailable {
			return f(page+1, size)
		}
		return nil
	}
	return f(DEFAULT_PAGE, DEFAULT_SIZE)
}

func (nacos *NacosService) listTenant() ([]*NacosNamespace, error) {
	resp, err := nacos.client.Get(nacos.addr + TENANT_PATH)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list nacos namespace, code is %d", resp.StatusCode)
	}
	nsList := []*NacosNamespace{}
	respData := struct {
		Code    int                `json:"code"`
		Message string             `json:"message"`
		Data    *[]*NacosNamespace `json:"data"`
	}{
		Data: &nsList,
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, fmt.Errorf("failed to decode nacos namespace resp, err is %v", err)
	}
	return nsList, nil
}

func (nacos *NacosService) postForm(path string, data url.Values) error {
	resp, err := nacos.client.PostForm(nacos.addr+path, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to post data, code is %d", resp.StatusCode)
	}
	return nil
}

func (nacos *NacosService) createUser(uname string) error {
	return nacos.postForm(USER_PATH, url.Values{
		"username": {uname},
		"password": {GenPassword(uname)},
	})
}

func (nacos *NacosService) createRole(rolename, username string) error {
	return nacos.postForm(ROLE_PATH, url.Values{
		"username": {username},
		"role":     {rolename},
	})
}

func (nacos *NacosService) createPerm(role, resource, action string) error {
	return nacos.postForm(PERM_PATH, url.Values{
		"role":     {role},
		"resource": {resource},
		"action":   {action},
	})
}

func (nacos *NacosService) createTenant(tenantid, tenantname string) error {
	q := url.Values{}
	q.Add("customNamespaceId", tenantid)
	q.Add("namespaceName", tenantname)
	q.Add("namespaceDesc", tenantname)
	req, _ := http.NewRequest(http.MethodPost, nacos.addr+TENANT_PATH+"?"+q.Encode(), nil)
	resp, err := nacos.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create nacos namespace, code is %d", resp.StatusCode)
	}
	return nil
}

func (nacos *NacosService) userRolesFor(mapper *NacosDataMapper) (rUser, rwUser, rRole, rwRole, resource string) {
	rUser = mapper.Tenant() + ":" + mapper.Environment() + ":r"
	rwUser = mapper.Tenant() + ":" + mapper.Environment() + ":rw"
	rRole = mapper.Tenant() + ":" + mapper.Environment() + ":r"
	rwRole = mapper.Tenant() + ":" + mapper.Environment() + ":rw"
	resource = mapper.TenantID() + ":" + mapper.Environment()
	return
}

func (nacos *NacosService) urlFor(mapper *NacosDataMapper, path string) string {
	q := url.Values{}
	q.Add("tenant", mapper.TenantID())
	q.Add("group", mapper.Group())
	q.Add("dataId", mapper.DataID())
	switch path {
	case HISTORY_PATH:
		q.Add("nid", mapper.Rev())
	default:
		q.Add("appName", mapper.Application())
	}
	u := nacos.addr + path + "?" + q.Encode()
	return u
}

func (nacos *NacosService) login() error {
	q := url.Values{}
	q.Add("username", nacos.username)
	q.Add("password", nacos.password)
	req, _ := http.NewRequest(http.MethodPost, nacos.addr+LOGIN_PATH, nil)
	req.URL.RawQuery = q.Encode()
	cli := http.Client{}
	if nacos.baseRoundTripper != nil {
		cli.Transport = nacos.baseRoundTripper
	}
	req.Header.Add("namespace", "nacos")
	req.Header.Add("service", "nacos-client")
	req.Header.Add("port", "8848")
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to login nacos, code is %d", resp.StatusCode)
	}
	authResponse := AuthInfo{}
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return fmt.Errorf("failed to decode login response, %s", err)
	}
	nacos.authInfo = &authResponse
	nacos.lastLogin = time.Now()
	return nil
}

type NacosDataMapper struct {
	item *ConfigItem
}

func mapperForNacos(item *ConfigItem) (*NacosDataMapper, error) {
	if item.Tenant == "" || item.Project == "" {
		return nil, fmt.Errorf("tenant and project must be set")
	}
	return &NacosDataMapper{
		item: item,
	}, nil
}

func (c *NacosDataMapper) TenantID() string {
	hash := sha1.New()
	b := "kubegems_" + c.item.Tenant + "_" + c.item.Project
	hash.Write([]byte(b))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (c *NacosDataMapper) Tenant() string {
	return fmt.Sprintf("kubegems/%s/%s", c.item.Tenant, c.item.Project)
}

func (c *NacosDataMapper) Group() string {
	return c.item.Environment
}

func (c *NacosDataMapper) Application() string {
	return c.item.Application
}

func (c *NacosDataMapper) Environment() string {
	return c.item.Environment
}

func (c *NacosDataMapper) DataID() string {
	return c.item.Key
}

func (c *NacosDataMapper) Rev() string {
	if c.item.Rev == 0 {
		return ""
	}
	return strconv.FormatInt(c.item.Rev, 10)
}
