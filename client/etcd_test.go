package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"google.golang.org/grpc"
)

var (
	etcdServers *MockServers
	etcdAddrs   []string
)

func startMockEtcdServer(useRealServer bool) {
	if useRealServer {
		etcdAddrs = []string{"127.0.0.1:2379"}
		return
	}

	svrs, err := StartMockServers(3)
	if err != nil {
		panic(err)
	}
	addrs := []string{}
	for _, serv := range svrs.Servers {
		addrs = append(addrs, serv.Address)
	}
	etcdServers = svrs
	etcdAddrs = addrs
}

func stopMockEtcdServer(useRealServer bool) {
	if !useRealServer {
		etcdServers.Stop()
	}
}

func TestNewEtcdService(t *testing.T) {

	type args struct {
		endpoints []string
		username  string
		password  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test new etcdservice success",
			args: args{
				endpoints: etcdAddrs,
				username:  "root",
				password:  "root",
			},
			wantErr: false,
		}, {
			name: "test new etcdservice failed with empty endpoints",
			args: args{
				endpoints: []string{},
				username:  "root",
				password:  "root",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEtcdService(tt.args.endpoints, tt.args.username, tt.args.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEtcdService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestEtcdService_Get(t *testing.T) {
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
			name:    "test get success",
			args:    args{ctx: context.Background(), item: &ConfigItem{Tenant: "t1", Project: "p1", Application: "a1", Environment: "e1", Key: "k2"}},
			wantErr: false,
		},
		{
			name:    "test get failed with no tenant",
			args:    args{ctx: context.Background(), item: &ConfigItem{Project: "proj1", Application: "app1", Environment: "dev", Key: "not exist key"}},
			wantErr: true,
		},
		{
			name:    "test get failed with no key",
			args:    args{ctx: context.Background(), item: &ConfigItem{Tenant: "ten1", Project: "proj1", Application: "app1", Environment: "dev", Key: "not exist key"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := NewEtcdService(etcdAddrs, "root", "root")
			if err := e.Get(tt.args.ctx, tt.args.item); (err != nil) != tt.wantErr {
				t.Errorf("EtcdService.Get() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEtcdService_Pub(t *testing.T) {
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
			name:    "test pub success",
			args:    args{ctx: context.Background(), item: &ConfigItem{Tenant: "ten1", Project: "proj1", Application: "app1", Environment: "dev", Key: "config"}},
			wantErr: false,
		},
		{
			name:    "test pub failed with no tenant",
			args:    args{ctx: context.Background(), item: &ConfigItem{Project: "proj1", Application: "app1", Environment: "dev", Key: "not exist key"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := NewEtcdService(etcdAddrs, "root", "root")
			if err := e.Pub(tt.args.ctx, tt.args.item); (err != nil) != tt.wantErr {
				t.Errorf("EtcdService.Pub() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEtcdService_Delete(t *testing.T) {
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
			name:    "test delete success",
			args:    args{ctx: context.Background(), item: &ConfigItem{Tenant: "ten1", Project: "proj1", Application: "app1", Environment: "dev", Key: "config"}},
			wantErr: false,
		},
		{
			name:    "test delete failed with no tenant",
			args:    args{ctx: context.Background(), item: &ConfigItem{Project: "proj1", Application: "app1", Environment: "dev", Key: "not exist key"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := NewEtcdService(etcdAddrs, "root", "root")
			if err := e.Delete(tt.args.ctx, tt.args.item); (err != nil) != tt.wantErr {
				t.Errorf("EtcdService.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEtcdService_History(t *testing.T) {
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
			name: "test history success",
			args: args{ctx: context.Background(), item: &ConfigItem{Tenant: "t1", Project: "p1", Application: "a1", Environment: "e1", Key: "k3"}},
		},
	}
	e, err := NewEtcdService(etcdAddrs, "root", "root")
	if err != nil {
		t.Error(e)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := e.History(tt.args.ctx, tt.args.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("EtcdService.History() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			bts, _ := json.Marshal(g)
			t.Log(string(bts))
		})
	}
}

func TestEtcdService_List(t *testing.T) {
	type args struct {
		ctx  context.Context
		opts *ListOptions
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test list success",
			args: args{
				ctx: context.Background(),
				opts: &ListOptions{
					ConfigItem: ConfigItem{
						Tenant:      "ten1",
						Project:     "proj1",
						Application: "app1",
						Environment: "dev",
						Key:         "config",
					},
					Page: 1,
					Size: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "test list success in project",
			args: args{
				ctx: context.Background(),
				opts: &ListOptions{
					ConfigItem: ConfigItem{
						Tenant:  "ten1",
						Project: "proj1",
					},
					Page: 1,
					Size: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "test list failed with no tenant",
			args: args{
				ctx: context.Background(),
				opts: &ListOptions{
					ConfigItem: ConfigItem{
						Project:     "proj1",
						Application: "app1",
						Environment: "dev",
						Key:         "config",
					},
					Page: 1,
					Size: 10,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := NewEtcdService(etcdAddrs, "root", "root")
			if _, err := e.List(tt.args.ctx, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("EtcdService.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestEtcdService_Accounts(t *testing.T) {
	e := &EtcdService{}
	got, err := e.Accounts(&ConfigItem{
		Tenant:      "t1",
		Project:     "p1",
		Environment: "e1",
	})
	if err != nil {
		t.Error(err)
	}
	if got[0].Username != "kubegems/t1/p1/e1-r" {
		t.Error("username is not right")
	}
	if got[0].Password != "10393af3163a9c82ce2189a5ce267467" {
		t.Error("password is not right")
	}
	if got[1].Username != "kubegems/t1/p1/e1-rw" {
		t.Error("username is not right")
	}
}

// Refrence: https://github.com/etcd-io/etcd/blob/main/client/v3/mock/mockserver/mockserver.go

// MockServer provides a mocked out grpc server of the etcdserver interface.
type MockServer struct {
	ln         net.Listener
	Network    string
	Address    string
	GrpcServer *grpc.Server
}

// MockServers provides a cluster of mocket out gprc servers of the etcdserver interface.
type MockServers struct {
	mu      sync.RWMutex
	Servers []*MockServer
	wg      sync.WaitGroup
}

// StartMockServers creates the desired count of mock servers
// and starts them.
func StartMockServers(count int) (ms *MockServers, err error) {
	return startMockServersTcp(count)
}

func startMockServersTcp(count int) (ms *MockServers, err error) {
	addrs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		addrs = append(addrs, "localhost:0")
	}
	return startMockServers("tcp", addrs)
}

func startMockServers(network string, addrs []string) (ms *MockServers, err error) {
	ms = &MockServers{
		Servers: make([]*MockServer, len(addrs)),
		wg:      sync.WaitGroup{},
	}
	defer func() {
		if err != nil {
			ms.Stop()
		}
	}()
	for idx, addr := range addrs {
		ln, err := net.Listen(network, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to listen %v", err)
		}
		ms.Servers[idx] = &MockServer{ln: ln, Network: network, Address: ln.Addr().String()}
		ms.StartAt(idx)
	}
	return ms, nil
}

// StartAt restarts mock server at given index.
func (ms *MockServers) StartAt(idx int) (err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.Servers[idx].ln == nil {
		ms.Servers[idx].ln, err = net.Listen(ms.Servers[idx].Network, ms.Servers[idx].Address)
		if err != nil {
			return fmt.Errorf("failed to listen %v", err)
		}
	}

	svr := grpc.NewServer()
	pb.RegisterKVServer(svr, &mockKVServer{})
	pb.RegisterAuthServer(svr, &mockAuthServer{})
	ms.Servers[idx].GrpcServer = svr

	ms.wg.Add(1)
	go func(svr *grpc.Server, l net.Listener) {
		svr.Serve(l)
	}(ms.Servers[idx].GrpcServer, ms.Servers[idx].ln)
	return nil
}

// StopAt stops mock server at given index.
func (ms *MockServers) StopAt(idx int) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.Servers[idx].ln == nil {
		return
	}

	ms.Servers[idx].GrpcServer.Stop()
	ms.Servers[idx].GrpcServer = nil
	ms.Servers[idx].ln = nil
	ms.wg.Done()
}

// Stop stops the mock server, immediately closing all open connections and listeners.
func (ms *MockServers) Stop() {
	for idx := range ms.Servers {
		ms.StopAt(idx)
	}
	ms.wg.Wait()
}

type mockKVServer struct{}

func (m *mockKVServer) Range(ctx context.Context, req *pb.RangeRequest) (*pb.RangeResponse, error) {
	var lastRev int64
	if req.Revision == 0 {
		lastRev = 20
	} else {
		lastRev = req.Revision
	}
	if strings.HasSuffix(string(req.Key), "not exist key") {
		return &pb.RangeResponse{Kvs: []*mvccpb.KeyValue{}}, nil
	}
	kv := &mvccpb.KeyValue{
		Key:            []byte("kubegems/ten1/proj1/dev/config"),
		Value:          []byte("content " + strconv.Itoa(int(lastRev))),
		ModRevision:    lastRev,
		CreateRevision: 1,
		Version:        lastRev,
	}
	return &pb.RangeResponse{
		Header: &pb.ResponseHeader{},
		Kvs:    []*mvccpb.KeyValue{kv},
	}, nil
}

func (m *mockKVServer) Put(context.Context, *pb.PutRequest) (*pb.PutResponse, error) {
	return &pb.PutResponse{}, nil
}

func (m *mockKVServer) DeleteRange(context.Context, *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error) {
	return &pb.DeleteRangeResponse{}, nil
}

func (m *mockKVServer) Txn(context.Context, *pb.TxnRequest) (*pb.TxnResponse, error) {
	return &pb.TxnResponse{}, nil
}

func (m *mockKVServer) Compact(context.Context, *pb.CompactionRequest) (*pb.CompactionResponse, error) {
	return &pb.CompactionResponse{}, nil
}

type mockAuthServer struct{}

func (a *mockAuthServer) AuthEnable(context.Context, *pb.AuthEnableRequest) (*pb.AuthEnableResponse, error) {
	return &pb.AuthEnableResponse{}, nil
}

func (a *mockAuthServer) AuthDisable(context.Context, *pb.AuthDisableRequest) (*pb.AuthDisableResponse, error) {
	return &pb.AuthDisableResponse{}, nil
}

func (a *mockAuthServer) AuthStatus(context.Context, *pb.AuthStatusRequest) (*pb.AuthStatusResponse, error) {
	return &pb.AuthStatusResponse{}, nil
}

func (a *mockAuthServer) Authenticate(context.Context, *pb.AuthenticateRequest) (*pb.AuthenticateResponse, error) {
	return &pb.AuthenticateResponse{}, nil
}

func (a *mockAuthServer) UserAdd(context.Context, *pb.AuthUserAddRequest) (*pb.AuthUserAddResponse, error) {
	return &pb.AuthUserAddResponse{}, nil
}
func (a *mockAuthServer) UserGet(context.Context, *pb.AuthUserGetRequest) (*pb.AuthUserGetResponse, error) {
	return &pb.AuthUserGetResponse{}, nil
}
func (a *mockAuthServer) UserList(context.Context, *pb.AuthUserListRequest) (*pb.AuthUserListResponse, error) {
	return &pb.AuthUserListResponse{}, nil
}
func (a *mockAuthServer) UserDelete(context.Context, *pb.AuthUserDeleteRequest) (*pb.AuthUserDeleteResponse, error) {
	return &pb.AuthUserDeleteResponse{}, nil
}
func (a *mockAuthServer) UserChangePassword(context.Context, *pb.AuthUserChangePasswordRequest) (*pb.AuthUserChangePasswordResponse, error) {
	return &pb.AuthUserChangePasswordResponse{}, nil
}
func (a *mockAuthServer) UserGrantRole(context.Context, *pb.AuthUserGrantRoleRequest) (*pb.AuthUserGrantRoleResponse, error) {
	return &pb.AuthUserGrantRoleResponse{}, nil
}
func (a *mockAuthServer) UserRevokeRole(context.Context, *pb.AuthUserRevokeRoleRequest) (*pb.AuthUserRevokeRoleResponse, error) {
	return &pb.AuthUserRevokeRoleResponse{}, nil
}
func (a *mockAuthServer) RoleAdd(context.Context, *pb.AuthRoleAddRequest) (*pb.AuthRoleAddResponse, error) {
	return &pb.AuthRoleAddResponse{}, nil
}
func (a *mockAuthServer) RoleGet(context.Context, *pb.AuthRoleGetRequest) (*pb.AuthRoleGetResponse, error) {
	return &pb.AuthRoleGetResponse{}, nil
}
func (a *mockAuthServer) RoleList(context.Context, *pb.AuthRoleListRequest) (*pb.AuthRoleListResponse, error) {
	return &pb.AuthRoleListResponse{}, nil
}
func (a *mockAuthServer) RoleDelete(context.Context, *pb.AuthRoleDeleteRequest) (*pb.AuthRoleDeleteResponse, error) {
	return &pb.AuthRoleDeleteResponse{}, nil
}
func (a *mockAuthServer) RoleGrantPermission(context.Context, *pb.AuthRoleGrantPermissionRequest) (*pb.AuthRoleGrantPermissionResponse, error) {
	return &pb.AuthRoleGrantPermissionResponse{}, nil
}
func (a *mockAuthServer) RoleRevokePermission(context.Context, *pb.AuthRoleRevokePermissionRequest) (*pb.AuthRoleRevokePermissionResponse, error) {
	return &pb.AuthRoleRevokePermissionResponse{}, nil
}
