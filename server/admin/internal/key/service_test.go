package key

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/gogu-x/proxy-access-manager/model"
)

type memoryRepository struct{ items map[string]model.ProxyKey }

func (m *memoryRepository) Get(_ context.Context, id string) (model.ProxyKey, error) {
	item, ok := m.items[id]
	if !ok {
		return item, ErrNotFound
	}
	return item, nil
}
func (m *memoryRepository) List(context.Context) ([]model.ProxyKey, error) {
	out := []model.ProxyKey{}
	for _, item := range m.items {
		out = append(out, item)
	}
	return out, nil
}
func (m *memoryRepository) Create(_ context.Context, item model.ProxyKey) error {
	m.items[item.ID] = item
	return nil
}
func (m *memoryRepository) Delete(_ context.Context, id string) error {
	if _, ok := m.items[id]; !ok {
		return ErrNotFound
	}
	delete(m.items, id)
	return nil
}
func (m *memoryRepository) SetEnabled(_ context.Context, id string, enabled bool) error {
	item := m.items[id]
	item.Enabled = enabled
	m.items[id] = item
	return nil
}
func (m *memoryRepository) Renew(_ context.Context, id string, days int) error {
	item := m.items[id]
	item.ExpiresAt += int64(days) * 86400
	m.items[id] = item
	return nil
}
func (m *memoryRepository) SetBandwidth(_ context.Context, id string, up, down int) error {
	item, ok := m.items[id]
	if !ok {
		return ErrNotFound
	}
	item.BandwidthUpMbps = up
	item.BandwidthDownMbps = down
	m.items[id] = item
	return nil
}
func (m *memoryRepository) RecordUsage(context.Context, model.ProxyKey, *int64, *int64, *int64, *int64) error {
	return nil
}
func (m *memoryRepository) SetSync(_ context.Context, id string, synced bool, message string) error {
	item := m.items[id]
	item.XraySynced = synced
	item.SyncError = message
	m.items[id] = item
	return nil
}

type fakeGateway struct {
	present                bool
	added, removed, kicked int
	kickErr                error
}

func (f *fakeGateway) Present(context.Context, model.ProxyKey) (bool, error) { return f.present, nil }
func (f *fakeGateway) Add(context.Context, model.ProxyKey) error {
	f.added++
	f.present = true
	return nil
}
func (f *fakeGateway) Remove(context.Context, model.ProxyKey) error {
	f.removed++
	f.present = false
	return nil
}
func (f *fakeGateway) XrayStat(context.Context, model.ProxyKey, string) (*int64, error) {
	return nil, errors.New("no stats")
}
func (f *fakeGateway) HysteriaTraffic(context.Context) (map[string]Traffic, error) { return nil, nil }
func (f *fakeGateway) Kick(context.Context, model.ProxyKey) error {
	f.kicked++
	return f.kickErr
}
func TestCreateAndRevoke(t *testing.T) {
	repo := &memoryRepository{items: map[string]model.ProxyKey{}}
	gateway := &fakeGateway{}
	service := NewService(repo, gateway, Config{Domain: "example.test", XrayPort: 8088, HysteriaPort: 443})
	item, err := service.Create(t.Context(), "张三", 30, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !item.IsActive() || !item.XraySynced || gateway.added != 1 || item.ExpiresAt < time.Now().Unix() {
		t.Fatalf("创建未生效: %+v", item)
	}
	item, err = service.Change(t.Context(), item.ID, "revoke", 0)
	if err != nil {
		t.Fatal(err)
	}
	if item.IsActive() || gateway.removed != 1 || gateway.kicked != 1 {
		t.Fatalf("吊销未生效: %+v", item)
	}
}

func TestSyncStatusesRestoresXrayUser(t *testing.T) {
	repo := &memoryRepository{items: map[string]model.ProxyKey{
		"user-id": {ID: "user-id", Name: "张三", Enabled: true, ExpiresAt: time.Now().Add(time.Hour).Unix(), XraySynced: true},
	}}
	gateway := &fakeGateway{present: false}
	service := NewService(repo, gateway, Config{})
	if err := service.SyncStatuses(t.Context()); err != nil {
		t.Fatal(err)
	}
	if gateway.added != 1 || !repo.items["user-id"].XraySynced {
		t.Fatal("Xray 重启后未补回有效用户")
	}
}

func TestDeleteDisablesBeforeRemovingRecord(t *testing.T) {
	key := model.ProxyKey{ID: "user-id", Name: "张三", Enabled: true, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	repo := &memoryRepository{items: map[string]model.ProxyKey{key.ID: key}}
	gateway := &fakeGateway{present: true, kickErr: errors.New("stats unavailable")}
	service := NewService(repo, gateway, Config{})
	if err := service.Delete(t.Context(), key.ID); err == nil {
		t.Fatal("断开失败时不应删除记录")
	}
	if repo.items[key.ID].Enabled || gateway.removed != 1 {
		t.Fatal("删除失败后用户应保持停用，且 Xray 用户已移除")
	}
	gateway.kickErr = nil
	if err := service.Delete(t.Context(), key.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.items[key.ID]; ok {
		t.Fatal("代理清理成功后记录仍存在")
	}
}

func TestSubscription(t *testing.T) {
	repo := &memoryRepository{items: map[string]model.ProxyKey{}}
	service := NewService(repo, &fakeGateway{}, Config{Domain: "example.test", XrayPort: 8088, HysteriaPort: 443})
	item, err := service.Create(t.Context(), "张三", 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(item.SubscriptionURL, "https://example.test/sub/")
	content, found, err := service.Subscription(t.Context(), token)
	if err != nil || !found || strings.Count(content, item.ID) != 2 {
		t.Fatalf("订阅内容无效: found=%v err=%v", found, err)
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		t.Fatalf("订阅 YAML 无效: %v", err)
	}
	for _, section := range []string{"proxies", "proxy-groups", "dns", "tun", "rules"} {
		if _, ok := document[section]; !ok {
			t.Fatalf("缺少模板配置 %s", section)
		}
	}
	if !strings.Contains(content, `server: "example.test"`) || !strings.Contains(content, `port: 8088`) || !strings.Contains(content, `port: 443`) || strings.Contains(content, "{{") {
		t.Fatal("订阅节点未正确替换")
	}
	if !strings.Contains(content, `up: "20 Mbps"`) || !strings.Contains(content, `down: "100 Mbps"`) {
		t.Fatal("默认 Hysteria 带宽未写入订阅")
	}
	if !strings.Contains(content, `DOMAIN-SUFFIX,google.com,PROXY`) || !strings.Contains(content, `DOMAIN-SUFFIX,openai.com,PROXY`) || !strings.Contains(content, `MATCH,PROXY`) {
		t.Fatal("原模板规则未保留")
	}
	if _, found, _ := service.Subscription(t.Context(), "invalid"); found {
		t.Fatal("无效令牌被接受")
	}
	if _, err := service.Change(t.Context(), item.ID, "revoke", 0); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := service.Subscription(t.Context(), token); found {
		t.Fatal("停用用户仍可下载订阅")
	}
}

func TestUpdateBandwidthChangesSubscription(t *testing.T) {
	repo := &memoryRepository{items: map[string]model.ProxyKey{}}
	service := NewService(repo, &fakeGateway{}, Config{Domain: "example.test", XrayPort: 8088, HysteriaPort: 443})
	item, err := service.Create(t.Context(), "张三", 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateBandwidth(t.Context(), item.ID, 0, 100); err == nil {
		t.Fatal("无效带宽被接受")
	}
	item, err = service.UpdateBandwidth(t.Context(), item.ID, 35, 80)
	if err != nil {
		t.Fatal(err)
	}
	if item.BandwidthUpMbps != 35 || item.BandwidthDownMbps != 80 {
		t.Fatal("用户带宽未保存")
	}
	token := strings.TrimPrefix(item.SubscriptionURL, "https://example.test/sub/")
	content, found, err := service.Subscription(t.Context(), token)
	if err != nil || !found || !strings.Contains(content, `up: "35 Mbps"`) || !strings.Contains(content, `down: "80 Mbps"`) {
		t.Fatal("更新后的带宽未写入订阅")
	}
	var document struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(content), &document); err != nil || len(document.Proxies) != 2 {
		t.Fatalf("订阅 YAML 无效: %v", err)
	}
	hysteria := document.Proxies[1]
	if hysteria["up"] != "35 Mbps" || hysteria["down"] != "80 Mbps" {
		t.Fatal("Hysteria 节点未使用平级 up/down 字段")
	}
	if _, nested := hysteria["bandwidth"]; nested {
		t.Fatal("FlClash 节点不应使用嵌套 bandwidth 字段")
	}
}

func TestSubscriptionsUseDistinctKeys(t *testing.T) {
	repo := &memoryRepository{items: map[string]model.ProxyKey{}}
	service := NewService(repo, &fakeGateway{}, Config{Domain: "example.test", XrayPort: 8088, HysteriaPort: 443})
	first, err := service.Create(t.Context(), "张三", 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(t.Context(), "李四", 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || first.SubscriptionURL == second.SubscriptionURL {
		t.Fatal("不同用户的连接凭据或订阅地址相同")
	}
	for _, item := range []model.ProxyKey{first, second} {
		token := strings.TrimPrefix(item.SubscriptionURL, "https://example.test/sub/")
		content, found, err := service.Subscription(t.Context(), token)
		if err != nil || !found || strings.Count(content, item.ID) != 2 {
			t.Fatalf("用户 %s 的订阅凭据错误: found=%v err=%v", item.Name, found, err)
		}
		other := first.ID
		if item.ID == first.ID {
			other = second.ID
		}
		if strings.Contains(content, other) {
			t.Fatalf("用户 %s 的订阅包含其他用户的凭据", item.Name)
		}
	}
}

func TestUsageHistory(t *testing.T) {
	now := time.Now().UTC()
	repo := &memoryRepository{items: map[string]model.ProxyKey{"user-1": {
		ID:            "user-1",
		UsageDaily:    map[string]int64{now.Format("20060102"): 1024},
		UsageHourly:   map[string]int64{now.Format("2006010215"): 512},
		UsageMinutely: map[string]int64{now.Format("200601021504"): 256},
	}}}
	service := NewService(repo, &fakeGateway{}, Config{})
	for _, tc := range []struct {
		granularity string
		count       int
		bytes       int64
	}{
		{"minute", 60, 256},
		{"hour", 24, 512},
		{"day", 30, 1024},
	} {
		points, err := service.UsageHistory(t.Context(), "user-1", tc.granularity)
		if err != nil || len(points) != tc.count || points[len(points)-1].Bytes != tc.bytes {
			t.Fatalf("%s 流量历史不正确: points=%v err=%v", tc.granularity, points, err)
		}
	}
	if _, err := service.UsageHistory(t.Context(), "user-1", "week"); !errors.Is(err, ErrInvalidGranularity) {
		t.Fatalf("无效粒度未被拒绝: %v", err)
	}
}
