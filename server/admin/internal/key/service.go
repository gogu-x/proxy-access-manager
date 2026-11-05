package key

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gogu-x/proxy-access-manager/conf"
	"github.com/gogu-x/proxy-access-manager/model"
	"github.com/google/uuid"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed clash-template.yaml
var clashTemplate string

type ProxyGateway interface {
	Present(context.Context, model.ProxyKey) (bool, error)
	Add(context.Context, model.ProxyKey) error
	Remove(context.Context, model.ProxyKey) error
	XrayStat(context.Context, model.ProxyKey, string) (*int64, error)
	HysteriaTraffic(context.Context) (map[string]Traffic, error)
	Kick(context.Context, model.ProxyKey) error
}
type Service struct {
	Store  Repository
	Proxy  ProxyGateway
	Config Config
	mu     sync.Mutex
}

func NewService(s Repository, p ProxyGateway, c Config) *Service {
	return &Service{s, p, c, sync.Mutex{}}
}
func (s *Service) Decorate(k model.ProxyKey) model.ProxyKey {
	k.Active = k.IsActive()
	if s.Config.Domain != "" {
		host := s.Config.Domain
		label := url.QueryEscape(k.Name)
		k.HysteriaLink = "hysteria2://" + k.ID + "@" + host + ":" + strconv.Itoa(s.Config.HysteriaPort) + "/?sni=" + url.QueryEscape(host) + "#" + label
		k.VlessLink = "vless://" + k.ID + "@" + host + ":" + strconv.Itoa(s.Config.XrayPort) + "?type=tcp&security=tls&flow=xtls-rprx-vision&sni=" + url.QueryEscape(host) + "#" + label
		k.SubscriptionURL = "https://" + host + "/sub/" + subscriptionToken(k.ID)
	}
	return k
}

// subscriptionToken 为每个 key 派生不暴露连接凭据的订阅令牌。
func subscriptionToken(id string) string {
	mac := hmac.New(sha256.New, []byte(conf.JWTSecret))
	mac.Write([]byte("subscription:" + id))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Subscription 按订阅令牌查找有效用户并生成客户端配置。
func (s *Service) Subscription(ctx context.Context, token string) (string, bool, error) {
	if s.Config.Domain == "" || len(token) != 43 {
		return "", false, nil
	}
	items, err := s.Store.List(ctx)
	if err != nil {
		return "", false, err
	}
	for _, item := range items {
		if hmac.Equal([]byte(token), []byte(subscriptionToken(item.ID))) {
			if !item.IsActive() {
				return "", false, nil
			}
			return s.SubscriptionYAML(item), true, nil
		}
	}
	return "", false, nil
}

// SubscriptionYAML 根据用户提供的 Clash 模板替换节点地址与凭据。
func (s *Service) SubscriptionYAML(k model.ProxyKey) string {
	return strings.NewReplacer(
		"{{DOMAIN}}", strconv.Quote(s.Config.Domain),
		"{{USER_KEY}}", strconv.Quote(k.ID),
		"{{XRAY_PORT}}", strconv.Itoa(s.Config.XrayPort),
		"{{HYSTERIA_PORT}}", strconv.Itoa(s.Config.HysteriaPort),
	).Replace(clashTemplate)
}
func (s *Service) List(ctx context.Context) ([]model.ProxyKey, error) {
	keys, err := s.Store.List(ctx)
	for i := range keys {
		keys[i] = s.Decorate(keys[i])
	}
	return keys, err
}
func (s *Service) Get(ctx context.Context, id string) (model.ProxyKey, error) {
	k, err := s.Store.Get(ctx, id)
	return s.Decorate(k), err
}

type UsagePoint struct {
	Date  string `json:"date"`
	Bytes int64  `json:"bytes"`
}

var ErrInvalidGranularity = errors.New("统计粒度无效")

// UsageHistory 按分钟、小时或天返回固定范围的流量数据。
func (s *Service) UsageHistory(ctx context.Context, id, granularity string) ([]UsagePoint, error) {
	var count int
	var interval time.Duration
	var keyFormat, displayFormat string
	switch granularity {
	case "minute":
		count, interval, keyFormat, displayFormat = 60, time.Minute, "200601021504", "2006-01-02T15:04:00Z"
	case "hour":
		count, interval, keyFormat, displayFormat = 24, time.Hour, "2006010215", "2006-01-02T15:00:00Z"
	case "day":
		count, interval, keyFormat, displayFormat = 30, 24*time.Hour, "20060102", "2006-01-02T00:00:00Z"
	default:
		return nil, ErrInvalidGranularity
	}
	k, err := s.Store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	points := make([]UsagePoint, 0, count)
	current := time.Now().UTC().Truncate(interval)
	for offset := count - 1; offset >= 0; offset-- {
		at := current.Add(-time.Duration(offset) * interval)
		var bytes int64
		switch granularity {
		case "minute":
			bytes = k.UsageMinutely[at.Format(keyFormat)]
		case "hour":
			bytes = k.UsageHourly[at.Format(keyFormat)]
		case "day":
			bytes = k.UsageDaily[at.Format(keyFormat)]
		}
		points = append(points, UsagePoint{Date: at.Format(displayFormat), Bytes: bytes})
	}
	return points, nil
}
func (s *Service) Create(ctx context.Context, name string, days, quotaGB int) (model.ProxyKey, error) {
	if name == "" || len([]rune(name)) > 80 || days < 1 || days > 3650 || quotaGB < 0 || quotaGB > 100000 {
		return model.ProxyKey{}, fmt.Errorf("名称、天数或额度无效")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := model.ProxyKey{ID: uuid.NewString(), Name: name, Enabled: true, CreatedAt: time.Now().Unix(), ExpiresAt: time.Now().Unix() + int64(days)*86400, QuotaBytes: int64(quotaGB) * 1024 * 1024 * 1024}
	if err := s.Store.Create(ctx, k); err != nil {
		return k, err
	}
	s.syncOne(ctx, k.ID)
	return s.Get(ctx, k.ID)
}
func (s *Service) Change(ctx context.Context, id, action string, days int) (model.ProxyKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.Store.Get(ctx, id); err != nil {
		return model.ProxyKey{}, err
	}
	var err error
	switch action {
	case "revoke":
		err = s.Store.SetEnabled(ctx, id, false)
	case "enable":
		err = s.Store.SetEnabled(ctx, id, true)
	case "renew":
		if days < 1 || days > 3650 {
			return model.ProxyKey{}, fmt.Errorf("续期天数无效")
		}
		err = s.Store.Renew(ctx, id, days)
	default:
		return model.ProxyKey{}, fmt.Errorf("未知操作")
	}
	if err != nil {
		return model.ProxyKey{}, err
	}
	s.syncOne(ctx, id)
	return s.Get(ctx, id)
}
func (s *Service) syncOne(ctx context.Context, id string) {
	k, err := s.Store.Get(ctx, id)
	if err != nil {
		return
	}
	wanted := k.IsActive()
	synced := k.XraySynced
	failures := []string{}
	present, err := s.Proxy.Present(ctx, k)
	if err != nil {
		failures = append(failures, err.Error())
	} else {
		if wanted && !present {
			err = s.Proxy.Add(ctx, k)
		}
		if !wanted && present {
			err = s.Proxy.Remove(ctx, k)
		}
		if err != nil {
			failures = append(failures, err.Error())
		} else {
			synced = wanted
		}
	}
	if !wanted {
		if err := s.Proxy.Kick(ctx, k); err != nil {
			failures = append(failures, "Hysteria: "+err.Error())
		}
	}
	message := strings.Join(failures, "; ")
	if len(message) > 250 {
		message = message[:250]
	}
	_ = s.Store.SetSync(ctx, id, synced, message)
}
func (s *Service) Reconcile(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys, err := s.Store.List(ctx)
	if err != nil {
		return err
	}
	traffic, _ := s.Proxy.HysteriaTraffic(ctx)
	for _, k := range keys {
		up, _ := s.Proxy.XrayStat(ctx, k, "uplink")
		down, _ := s.Proxy.XrayStat(ctx, k, "downlink")
		var hup, hdown *int64
		if item, ok := traffic[k.ID]; ok {
			hup = &item.TX
			hdown = &item.RX
		}
		if err := s.Store.RecordUsage(ctx, k, up, down, hup, hdown); err != nil {
			return err
		}
		s.syncOne(ctx, k.ID)
	}
	return nil
}

// SyncStatuses 定期同步到期和停用状态，不触发流量统计接口。
func (s *Service) SyncStatuses(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys, err := s.Store.List(ctx)
	if err != nil {
		return err
	}
	for _, k := range keys {
		s.syncOne(ctx, k.ID)
	}
	return nil
}

func (s *Service) Run(ctx context.Context) {
	s.Reconcile(ctx)
	trafficTicker := time.NewTicker(time.Minute)
	defer trafficTicker.Stop()
	statusTicker := time.NewTicker(time.Minute)
	defer statusTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-trafficTicker.C:
			s.Reconcile(ctx)
		case <-statusTicker.C:
			s.SyncStatuses(ctx)
		}
	}
}
