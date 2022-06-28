package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

/*
test Etcd Version: v3.5.2
*/

type EtcdService struct {
	cli *clientv3.Client

	users []string
	roles []string
}

type PreAcionDone string

const preAcionDone PreAcionDone = "pre_action_done"

func NewEtcdService(endpoints []string, username, password string) (*EtcdService, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		Username:    username,
		Password:    password,
		DialTimeout: 5 * 1e9,
	})
	if err != nil {
		return nil, err
	}
	return &EtcdService{
		cli:   cli,
		users: []string{},
		roles: []string{},
	}, nil
}

type Rev struct {
	Version        int64
	CreateRevision int64
	ModRevision    int64
}

func (e *EtcdService) get(ctx context.Context, item *ConfigItem, opts ...clientv3.OpOption) (*Rev, error) {
	mapper, err := mapperForEtcd(item)
	if err != nil {
		return nil, err
	}
	if err := e.preAction(ctx, mapper); err != nil {
		return nil, err
	}
	resp, err := e.cli.Get(ctx, mapper.Key(), opts...)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) != 1 {
		return nil, fmt.Errorf("key %s not found or more than one key found", mapper.Key())
	}
	kv := resp.Kvs[0]
	item.Value = string(kv.Value)
	rev := &Rev{
		Version:        kv.Version,
		ModRevision:    kv.ModRevision,
		CreateRevision: kv.CreateRevision,
	}
	return rev, nil
}

func (e *EtcdService) Get(ctx context.Context, item *ConfigItem) error {
	_, err := e.get(ctx, item, clientv3.WithLimit(1), clientv3.WithRev(item.Rev))
	return err
}

func (e *EtcdService) BaseInfo(ctx context.Context, item *ConfigItem) (map[string]string, error) {
	return map[string]string{
		"provider": "etcd",
	}, nil
}

func (e *EtcdService) Pub(ctx context.Context, item *ConfigItem) error {
	mapper, err := mapperForEtcd(item)
	if err != nil {
		return err
	}
	if err := e.preAction(ctx, mapper); err != nil {
		return err
	}
	_, err = e.cli.Put(ctx, mapper.Key(), item.Value)
	return err
}

func (e *EtcdService) Delete(ctx context.Context, item *ConfigItem) error {
	mapper, err := mapperForEtcd(item)
	if err != nil {
		return err
	}
	if err := e.preAction(ctx, mapper); err != nil {
		return err
	}
	_, err = e.cli.Delete(ctx, mapper.Key())
	return err
}

func (e *EtcdService) Listener(ctx context.Context, item *ConfigItem) (map[string]string, error) {
	return map[string]string{}, nil
}

func (e *EtcdService) History(ctx context.Context, item *ConfigItem) ([]*HistoryVersion, error) {
	latestRev, err := e.get(ctx, item)
	if err != nil {
		return nil, err
	}
	flag := 10
	version := latestRev.Version - 1
	lastRev := latestRev.ModRevision - 1
	ret := []*HistoryVersion{}
	nctx := context.WithValue(ctx, preAcionDone, true)
	for version > 0 && flag > 0 && lastRev >= latestRev.CreateRevision {
		rev, err := e.get(nctx, item, clientv3.WithRev(lastRev))
		if err != nil {
			return nil, err
		}
		ret = append(ret, &HistoryVersion{
			Version: strconv.Itoa(int(version)),
			Rev:     strconv.Itoa(int(lastRev)),
		})
		flag--
		version--
		lastRev = rev.ModRevision - 1
	}
	return ret, nil
}

func (e *EtcdService) List(ctx context.Context, opts *ListOptions) ([]*ConfigItem, error) {
	mapper, err := mapperForEtcd(&opts.ConfigItem)
	if err != nil {
		return nil, err
	}
	if err := e.preAction(ctx, mapper); err != nil {
		return nil, err
	}
	end := clientv3.GetPrefixRangeEnd(mapper.ListKey())
	resp, err := e.cli.Get(ctx, mapper.ListKey(), clientv3.WithRange(end))
	if err != nil {
		return nil, err
	}
	ret := []*ConfigItem{}
	for _, kv := range resp.Kvs {
		cfg, err := e.convert(kv)
		if err != nil {
			continue
		}
		ret = append(ret, cfg)
	}
	return ret, nil
}

type EtcdMapper struct {
	item *ConfigItem
}

func mapperForEtcd(item *ConfigItem) (*EtcdMapper, error) {
	if item.Tenant == "" || item.Project == "" {
		return nil, fmt.Errorf("tenant and project must be specified")
	}
	return &EtcdMapper{
		item: item,
	}, nil
}

func (c *EtcdMapper) Key() string {
	return fmt.Sprintf("kubegems/%s/%s/%s/%s", c.item.Tenant, c.item.Project, c.item.Environment, c.item.Key)
}

func (c *EtcdMapper) ListKey() string {
	if c.item.Environment == "" {
		return fmt.Sprintf("kubegems/%s/%s", c.item.Tenant, c.item.Project)
	} else {
		return fmt.Sprintf("kubegems/%s/%s/%s", c.item.Tenant, c.item.Project, c.item.Environment)
	}
}

func (c *EtcdMapper) NsPrefix() string {
	return fmt.Sprintf("kubegems/%s/%s/%s", c.item.Tenant, c.item.Project, c.item.Environment)
}

func (e *EtcdService) convert(kv *mvccpb.KeyValue) (*ConfigItem, error) {
	k := string(kv.Key)
	seps := strings.Split(k, "/")
	if len(seps) != 5 {
		return nil, fmt.Errorf("invalid key")
	}
	return &ConfigItem{
		Key:         seps[4],
		Value:       string(kv.Value),
		Tenant:      seps[1],
		Project:     seps[2],
		Environment: seps[3],
	}, nil
}

func (e *EtcdService) preAction(ctx context.Context, mapper *EtcdMapper) error {
	if ctx.Value(preAcionDone) != nil {
		return nil
	}
	rUser, rwUser, rRole, rwRole := e.userRolesFor(mapper)
	var newRUser, newRWUser, newRRole, newRWRole bool
	if !contains(e.users, rUser) {
		e.cli.Auth.UserAdd(ctx, rUser, GenPassword(rUser))
		newRUser = true
	}
	if !contains(e.users, rwUser) {
		e.cli.Auth.UserAdd(ctx, rwUser, GenPassword(rwUser))
		newRWUser = true
	}
	if !contains(e.roles, rRole) {
		e.cli.Auth.RoleAdd(ctx, rRole)
		newRRole = true
	}
	if !contains(e.roles, rwRole) {
		e.cli.Auth.RoleAdd(ctx, rwRole)
		newRWRole = true
	}

	if newRUser || newRRole {
		e.cli.Auth.RoleGrantPermission(ctx, rRole, mapper.NsPrefix(), clientv3.GetPrefixRangeEnd(mapper.Key()), clientv3.PermissionType(clientv3.PermRead))
		e.cli.Auth.UserGrantRole(ctx, rUser, rRole)
	}
	if newRWUser || newRWRole {
		e.cli.Auth.RoleGrantPermission(ctx, rwRole, mapper.NsPrefix(), clientv3.GetPrefixRangeEnd(mapper.Key()), clientv3.PermissionType(clientv3.PermReadWrite))
		e.cli.Auth.UserGrantRole(ctx, rwUser, rwRole)
	}
	return nil
}

func (e *EtcdService) userRolesFor(mapper *EtcdMapper) (rUser, rwUser, rRole, rwRole string) {
	rUser = mapper.NsPrefix() + "-r"
	rwUser = mapper.NsPrefix() + "-rw"
	rRole = rUser
	rwRole = rwUser
	return
}

func (e *EtcdService) Accounts(item *ConfigItem) ([]Account, error) {
	mapper, err := mapperForEtcd(item)
	if err != nil {
		return nil, err
	}
	rUser, rwUser, _, _ := e.userRolesFor(mapper)
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
