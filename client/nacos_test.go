package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var (
	server          *httptest.Server
	nacosRealServer = false
	etcdRealServer  = false
)

func setup() {
	startNacosMockServer(nacosRealServer)
	startMockEtcdServer(etcdRealServer)
}

func teardown() {
	stopNacosMockServer(nacosRealServer)
	stopMockEtcdServer(etcdRealServer)
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func startNacosMockServer(useRealServer bool) {
	if useRealServer {
		server = &httptest.Server{
			URL: "http://localhost:8848/nacos",
		}
		return
	}
	successFn := func(w http.ResponseWriter, r *http.Request) {
		respData := struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			Code:    200,
			Message: "success",
		}
		ret, _ := json.Marshal(respData)
		w.Write(ret)
	}

	notAllowFn := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	flag := false
	// mock login
	http.HandleFunc(LOGIN_PATH, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			passwd := r.URL.Query().Get("password")
			switch passwd {
			// mock login error
			case "error":
				w.WriteHeader(403)
				w.Write([]byte("unknown user!"))
			// mock login with error response
			case "errorresp":
				w.Write([]byte("error resp"))
			// mock login success
			default:
				var ttl int64
				if flag {
					ttl = 10
				} else {
					flag = true
					ttl = -1
				}
				respData := AuthInfo{
					AccessToken: "test123",
					TokenTTL:    ttl,
					GlobalAdmin: true,
				}
				ret, _ := json.Marshal(respData)
				w.Write(ret)
			}
		default:
			notAllowFn(w, r)
		}
	}))

	// mock list tenant and create tenant
	http.HandleFunc(TENANT_PATH, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			respData := struct {
				Code    int              `json:"code"`
				Message string           `json:"message"`
				Data    []NacosNamespace `json:"data"`
			}{
				Data: []NacosNamespace{
					{
						Namespace:         "kubegems_ten1_proj1",
						NamepsaceShowName: "kubegems/ten1/proj1",
						Quota:             200,
						ConfigCount:       0,
						Type:              0,
					},
					{
						Namespace:         "kubegems_ten3_proj3",
						NamepsaceShowName: "kubegems/ten3/proj3",
						Quota:             200,
						ConfigCount:       0,
						Type:              0,
					},
					{
						Namespace:         "kubegems_ten4_proj4",
						NamepsaceShowName: "kubegems/ten4/proj4",
						Quota:             200,
						ConfigCount:       0,
						Type:              0,
					},
				},
			}
			ret, _ := json.Marshal(respData)
			w.Write(ret)
		case http.MethodPost:
			switch {
			case strings.Contains(r.URL.Query().Get("namespaceName"), "error"):
				w.WriteHeader(400)
			default:
				successFn(w, r)
			}
		default:
			notAllowFn(w, r)
		}
	}))

	// mock user
	http.HandleFunc(USER_PATH, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			datas := [][]NacosUser{}
			datas = append(datas, []NacosUser{
				{
					Username: "kubegems/ten1/proj1:dev:r",
				},
				{
					Username: "kubegems/ten1/proj1:dev:rw",
				},
				{
					Username: "kubegems/ten2/proj2:dev:r",
				},
			})
			datas = append(datas, []NacosUser{
				{
					Username: "kubegems/ten3/proj3:dev:rw",
				},
				{
					Username: "kubegems/ten3/proj3:dev:r",
				},
			})
			datas = append(datas, []NacosUser{
				{
					Username: "kubegems/ten2/proj2:dev:rw",
				},
				{
					Username: "kubegems/ten4/proj4:dev:r",
				},
				{
					Username: "kubegems/ten4/proj4:dev:rw",
				},
			})
			var page int
			if r.URL.Query().Get("pageNo") == "1" {
				page = 1
			} else {
				page = 2
			}
			respData := NacosListStruct{
				TotalCount:     6,
				PageNumber:     page,
				PagesAvailable: 3,
				PageItems:      datas[page-1],
			}
			bts, _ := json.Marshal(respData)
			w.Write(bts)
		case http.MethodPost:
			switch {
			case strings.Contains(r.FormValue("username"), "error"):
				w.WriteHeader(400)
			default:
				successFn(w, r)
			}
		default:
			notAllowFn(w, r)
		}
	})

	// mock roles
	http.HandleFunc(ROLE_PATH, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			respData := NacosListStruct{
				TotalCount:     4,
				PageNumber:     1,
				PagesAvailable: 1,
				PageItems: []NacosRole{
					{
						Username: "kubegems/ten1/proj1:dev:r",
						Role:     "kubegems/ten1/proj1:dev:r",
					},
					{
						Username: "kubegems/ten1/proj1:dev:rw",
						Role:     "kubegems/ten1/proj1:dev:rw",
					},
					{
						Username: "kubegems/ten2/proj1:dev:rw",
						Role:     "kubegems/ten2/proj1:dev:rw",
					},
					{
						Username: "kubegems/ten2/proj2:dev:rw",
						Role:     "kubegems/ten2/proj2:dev:rw",
					},
					{
						Username: "kubegems/ten4/proj4:dev:rw",
						Role:     "kubegems/ten4/proj4:dev:rw",
					},
					{
						Username: "kubegems/ten4/proj4:dev:r",
						Role:     "kubegems/ten4/proj4:dev:r",
					},
					{
						Username: "kubegems/ten3/proj3:dev:rw",
						Role:     "kubegems/ten3/proj3:dev:rw",
					},
				},
			}
			bts, _ := json.Marshal(respData)
			w.Write(bts)
		case http.MethodPost:
			switch {
			case strings.Contains(r.Form.Get("username"), "error"):
				w.WriteHeader(400)
			default:
				successFn(w, r)
			}
		default:
			notAllowFn(w, r)
		}
	})

	// mock perms
	http.HandleFunc(PERM_PATH, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			respData := NacosListStruct{
				TotalCount:     6,
				PageNumber:     1,
				PagesAvailable: 1,
				PageItems: []NacosPerm{
					{
						Role:     "kubegems/ten2/proj1:dev:r",
						Resource: "kubegems_ten2_proj1:dev",
						Action:   "r",
					},
					{
						Role:     "kubegems/ten2/proj2:dev:r",
						Resource: "kubegems_ten2_proj2:dev",
						Action:   "r",
					},
					{
						Role:     "kubegems/ten2/proj2:dev:rw",
						Resource: "kubegems_ten2_proj2:dev",
						Action:   "rw",
					},
					{
						Role:     "kubegems/ten1/proj1:dev:rw",
						Resource: "kubegems_ten1_proj1:dev",
						Action:   "rw",
					},
					{
						Role:     "kubegems/ten1/proj2:dev:rw",
						Resource: "kubegems_ten1_proj2:dev",
						Action:   "rw",
					},
					{
						Role:     "kubegems/ten4/proj4:dev:rw",
						Resource: "kubegems_ten4_proj4:dev",
						Action:   "rw",
					},
					{
						Role:     "kubegems/ten4/proj4:dev:r",
						Resource: "kubegems_ten4_proj4:dev",
						Action:   "r",
					},
				},
			}
			bts, _ := json.Marshal(respData)
			w.Write(bts)
		case http.MethodPost:
			switch {
			case strings.Contains(r.Form.Get("username"), "error"):
				w.WriteHeader(400)
			default:
				successFn(w, r)
			}
		default:
			notAllowFn(w, r)
		}
	})

	// mock config
	http.HandleFunc(CONFIG_PATH, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			datas := []NacosConfigItem{
				{
					Content: "test config",
					DataID:  "config",
					Group:   "dev",
					Tenant:  "kubegems_ten1_proj1",
					Md5:     "",
				},
				{
					Content: "prod config",
					DataID:  "config",
					Group:   "prod",
					Tenant:  "kubegems_ten1_proj1",
					Md5:     "",
				},
			}
			if r.URL.Query().Get("group") == "error" {
				datas = append(datas, NacosConfigItem{
					Content: "prod config",
					DataID:  "config",
					Group:   "prod",
					Tenant:  "error tenant",
					Md5:     "",
				})

			}

			respData := NacosListStruct{
				TotalCount:     10,
				PageNumber:     1,
				PagesAvailable: 1,
				PageItems:      datas,
			}
			bts, _ := json.Marshal(respData)
			w.Write(bts)
		case http.MethodPost:
			switch {
			case strings.Contains(r.URL.Query().Get("username"), "error"):
				w.WriteHeader(400)
			default:
				successFn(w, r)
			}
		case http.MethodDelete:
			successFn(w, r)
		default:
			notAllowFn(w, r)
		}
	})

	// mock history
	http.HandleFunc(HISTORY_PATH, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			var retdata []byte
			c := NacosConfigItem{
				ID:               "123",
				Content:          "test config",
				DataID:           "config",
				Group:            "dev",
				Tenant:           "kubegems_ten1_proj1",
				Md5:              "",
				CreatedTime:      "2010-05-05T00:00:00.000+08:00",
				LastModifiedTime: "2010-05-05T00:00:00.000+08:00",
			}
			switch {
			case r.URL.Query().Get("nid") != "":
				datas := []NacosConfigItem{c}
				respData := NacosListStruct{
					TotalCount:     1,
					PageNumber:     1,
					PagesAvailable: 1,
					PageItems:      datas,
				}
				retdata, _ = json.Marshal(respData)
			default:
				retdata, _ = json.Marshal(c)

			}
			w.Write(retdata)
		default:
			notAllowFn(w, r)
		}
	})

	s := httptest.NewServer(http.DefaultServeMux)
	server = s
}

func stopNacosMockServer(useRealServer bool) {
	if !useRealServer {
		server.Close()
	}
}

func TestNewNacosService(t *testing.T) {
	type args struct {
		addr     string
		username string
		password string
	}
	tests := []struct {
		name    string
		args    args
		want    *NacosService
		wantErr bool
	}{
		{
			name: "test new NacosService success",
			args: args{
				addr:     server.URL,
				username: "nacos",
				password: "pass",
			},
			wantErr: false,
		}, {
			name: "test new NacosService with error password",
			args: args{
				addr:     server.URL,
				username: "nacos",
				password: "error",
			},
			wantErr: true,
		}, {
			name: "test new NacosService with error host",
			args: args{
				addr:     "http://error.com",
				username: "nacos",
				password: "error",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNacosService(tt.args.addr, tt.args.username, tt.args.password, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNacosService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestNacosService_Get(t *testing.T) {
	type args struct {
		ctx      context.Context
		item     *ConfigItem
		addr     string
		username string
		password string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test get with tenant not exist",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Tenant:      "ten2",
					Project:     "proj2",
					Environment: "dev",
					Application: "app2",
					Key:         "config",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: false,
		}, {
			name: "test get with tenant exists",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Tenant:      "ten1",
					Project:     "proj1",
					Environment: "dev",
					Application: "app3",
					Key:         "config",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: false,
		}, {
			name: "test get success",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Tenant:      "ten4",
					Project:     "proj3",
					Environment: "dev",
					Application: "app3",
					Key:         "config",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: false,
		}, {
			name: "test get with err args, empty tenant",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Project:     "proj1",
					Environment: "dev",
					Application: "app1",
					Key:         "config",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: true,
		}, {
			name: "test get with error",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Tenant:      "error",
					Project:     "proj1",
					Environment: "dev",
					Application: "app1",
					Key:         "config",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: true,
		}, {
			name: "test get history",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Tenant:      "ten1",
					Project:     "proj1",
					Environment: "dev",
					Application: "app1",
					Key:         "config",
					Rev:         123,
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: false,
		},
	}

	nacos, err := NewNacosService(server.URL, "nacos", "nacos", nil)
	if err != nil {
		t.Errorf("NewNacosService() error = %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := nacos.Get(tt.args.ctx, tt.args.item); (err != nil) != tt.wantErr {
				t.Errorf("NacosService.Get() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNacosService_Pub(t *testing.T) {
	type args struct {
		ctx      context.Context
		item     *ConfigItem
		addr     string
		username string
		password string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test pub with valid args",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Tenant:      "ten1",
					Project:     "proj1",
					Environment: "dev",
					Application: "app1",
					Key:         "config",
					Value:       "config content",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: false,
		}, {
			name: "test pub with valid args",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Tenant:      "ten2",
					Project:     "proj1",
					Environment: "dev",
					Application: "app1",
					Key:         "config",
					Value:       "config content",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: false,
		}, {
			name: "test pub with err args, empty tenant",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Project:     "proj1",
					Environment: "dev",
					Application: "app1",
					Key:         "config",
					Value:       "config content",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: true,
		},
	}
	nacos, err := NewNacosService(server.URL, "nacos", "nacos", nil)
	if err != nil {
		t.Errorf("NewNacosService() error = %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := nacos.Pub(tt.args.ctx, tt.args.item); (err != nil) != tt.wantErr {
				t.Errorf("NacosService.Get() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNacosService_Delete(t *testing.T) {
	type args struct {
		ctx      context.Context
		item     *ConfigItem
		addr     string
		username string
		password string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test delete with valid args",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Tenant:      "ten1",
					Project:     "proj1",
					Environment: "dev",
					Application: "app1",
					Key:         "config",
					Value:       "config content",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: false,
		}, {
			name: "test delete with valid args",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Tenant:      "ten2",
					Project:     "proj1",
					Environment: "dev",
					Application: "app1",
					Key:         "config",
					Value:       "config content",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: false,
		}, {
			name: "test delete with err args, empty tenant",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Project:     "proj1",
					Environment: "dev",
					Application: "app1",
					Key:         "config",
					Value:       "config content",
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: true,
		},
	}
	nacos, err := NewNacosService(server.URL, "nacos", "nacos", nil)
	if err != nil {
		t.Errorf("NewNacosService() error = %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := nacos.Delete(tt.args.ctx, tt.args.item); (err != nil) != tt.wantErr {
				t.Errorf("NacosService.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNacosService_List(t *testing.T) {
	type args struct {
		ctx      context.Context
		opts     *ListOptions
		addr     string
		username string
		password string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test list with valid args",
			args: args{
				ctx: context.Background(),
				opts: &ListOptions{
					ConfigItem: ConfigItem{
						Tenant:      "ten1",
						Project:     "proj1",
						Environment: "dev",
						Application: "app1",
						Key:         "config",
						Value:       "config content",
					},
					Page: 1,
					Size: 2,
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: false,
		}, {
			name: "test list with valid args",
			args: args{
				ctx: context.Background(),
				opts: &ListOptions{
					ConfigItem: ConfigItem{
						Tenant:      "ten2",
						Project:     "proj1",
						Environment: "dev",
						Application: "app1",
						Key:         "config",
						Value:       "config content",
					},
					Page: -1,
					Size: -2,
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: false,
		}, {
			name: "test list with err args, empty tenant",
			args: args{
				ctx: context.Background(),
				opts: &ListOptions{
					ConfigItem: ConfigItem{
						Project:     "proj1",
						Environment: "dev",
						Application: "app1",
						Key:         "config",
						Value:       "config content",
					},
					Page: 1,
					Size: 2,
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: true,
		}, {
			name: "test list with err, group  error",
			args: args{
				ctx: context.Background(),
				opts: &ListOptions{
					ConfigItem: ConfigItem{
						Tenant:      "ten1",
						Project:     "proj1",
						Environment: "error",
						Application: "app1",
						Key:         "config",
						Value:       "config content",
					},
					Page: 1,
					Size: 2,
				},
				addr:     server.URL,
				username: "nacos",
				password: "nacos",
			},
			wantErr: true,
		},
	}
	nacos, err := NewNacosService(server.URL, "nacos", "nacos", nil)
	if err != nil {
		t.Errorf("NewNacosService() error = %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := nacos.List(tt.args.ctx, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("NacosService.List() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNacosService_History(t *testing.T) {
	type args struct {
		ctx  context.Context
		item *ConfigItem
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test history",
			args: args{
				ctx: context.Background(),
				item: &ConfigItem{
					Tenant:      "ten1",
					Project:     "proj1",
					Environment: "dev",
					Application: "app1",
					Key:         "config",
					Value:       "config content",
				},
			},
		},
	}
	nacos, err := NewNacosService(server.URL, "nacos", "nacos", nil)
	if err != nil {
		t.Errorf("NewNacosService() error = %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := nacos.History(tt.args.ctx, tt.args.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("NacosService.History() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			bts, _ := json.Marshal(g)

			t.Log(string(bts))
		})
	}
}

func TestNacosService_Accounts(t *testing.T) {
	nacos := &NacosService{}
	got, err := nacos.Accounts(&ConfigItem{
		Tenant:      "t1",
		Project:     "p1",
		Environment: "e1",
	})
	if err != nil {
		t.Error(err)
	}
	if got[0].Username != "kubegems/t1/p1:e1:r" {
		t.Error("username is not right")
	}
	if got[0].Password != "5096658f1599e2f9b1e677b2838177d2" {
		t.Log(got[0].Password)
		t.Error("password is not right")
	}
	if got[1].Username != "kubegems/t1/p1:e1:rw" {
		t.Error("username is not right")
	}
}
