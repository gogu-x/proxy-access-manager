package model

import "time"

type ProxyKey struct {
	ID              string           `json:"id" bson:"_id"`
	Name            string           `json:"name" bson:"name"`
	Enabled         bool             `json:"enabled" bson:"enabled"`
	ExpiresAt       int64            `json:"expires_at" bson:"expires_at"`
	QuotaBytes      int64            `json:"quota_bytes" bson:"quota_bytes"`
	UsedBytes       int64            `json:"used_bytes" bson:"used_bytes"`
	XraySynced      bool             `json:"xray_synced" bson:"xray_synced"`
	SyncError       string           `json:"sync_error" bson:"sync_error"`
	CreatedAt       int64            `json:"created_at" bson:"created_at"`
	XrayUpLast      int64            `json:"-" bson:"xray_up_last"`
	XrayDownLast    int64            `json:"-" bson:"xray_down_last"`
	HystUpLast      int64            `json:"-" bson:"hyst_up_last"`
	HystDownLast    int64            `json:"-" bson:"hyst_down_last"`
	UsageDaily      map[string]int64 `json:"-" bson:"usage_daily,omitempty"`
	Active          bool             `json:"active" bson:"-"`
	HysteriaLink    string           `json:"hysteria_link,omitempty" bson:"-"`
	VlessLink       string           `json:"vless_link,omitempty" bson:"-"`
	SubscriptionURL string           `json:"subscription_url,omitempty" bson:"-"`
}

func (k ProxyKey) IsActive() bool {
	return k.Enabled && k.ExpiresAt > time.Now().Unix() && (k.QuotaBytes == 0 || k.UsedBytes < k.QuotaBytes)
}
