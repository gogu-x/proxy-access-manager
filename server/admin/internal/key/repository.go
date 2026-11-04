package key

import (
	"context"
	"errors"
	"time"

	"github.com/gogu-x/proxy-access-manager/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("key 不存在")

type Repository interface {
	Get(context.Context, string) (model.ProxyKey, error)
	List(context.Context) ([]model.ProxyKey, error)
	Create(context.Context, model.ProxyKey) error
	SetEnabled(context.Context, string, bool) error
	Renew(context.Context, string, int) error
	RecordUsage(context.Context, model.ProxyKey, *int64, *int64, *int64, *int64) error
	SetSync(context.Context, string, bool, string) error
}
type MongoRepository struct{ collection *mongo.Collection }

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{db.Collection("proxy_keys")}
}
func (r *MongoRepository) Get(ctx context.Context, id string) (model.ProxyKey, error) {
	var key model.ProxyKey
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&key)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return key, ErrNotFound
	}
	return key, err
}
func (r *MongoRepository) List(ctx context.Context) ([]model.ProxyKey, error) {
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := []model.ProxyKey{}
	err = cursor.All(ctx, &items)
	return items, err
}
func (r *MongoRepository) Create(ctx context.Context, k model.ProxyKey) error {
	_, err := r.collection.InsertOne(ctx, k)
	return err
}
func (r *MongoRepository) SetEnabled(ctx context.Context, id string, enabled bool) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"enabled": enabled}})
	if err == nil && result.MatchedCount == 0 {
		return ErrNotFound
	}
	return err
}
func (r *MongoRepository) Renew(ctx context.Context, id string, days int) error {
	k, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	expires := k.ExpiresAt
	if expires < time.Now().Unix() {
		expires = time.Now().Unix()
	}
	expires += int64(days) * 86400
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id, "expires_at": k.ExpiresAt}, bson.M{"$set": bson.M{"expires_at": expires}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("续期冲突，请重试")
	}
	return nil
}
func (r *MongoRepository) SetSync(ctx context.Context, id string, synced bool, message string) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"xray_synced": synced, "sync_error": message}})
	return err
}
func (r *MongoRepository) RecordUsage(ctx context.Context, k model.ProxyKey, xup, xdown, hup, hdown *int64) error {
	vals := []*int64{xup, xdown, hup, hdown}
	old := [4]int64{k.XrayUpLast, k.XrayDownLast, k.HystUpLast, k.HystDownLast}
	next := old
	used := k.UsedBytes
	for i, v := range vals {
		if v == nil {
			continue
		}
		if *v >= old[i] {
			used += *v - old[i]
		} else {
			used += *v
		}
		next[i] = *v
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -29).Format("20060102")
	obsolete := bson.M{}
	for day := range k.UsageDaily {
		if day < cutoff {
			obsolete["usage_daily."+day] = ""
		}
	}
	if used == k.UsedBytes && next == old && len(obsolete) == 0 {
		return nil
	}
	update := bson.M{"$set": bson.M{"used_bytes": used, "xray_up_last": next[0], "xray_down_last": next[1], "hyst_up_last": next[2], "hyst_down_last": next[3]}}
	if len(obsolete) > 0 {
		update["$unset"] = obsolete
	}
	if delta := used - k.UsedBytes; delta > 0 {
		update["$inc"] = bson.M{"usage_daily." + time.Now().UTC().Format("20060102"): delta}
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": k.ID}, update)
	return err
}
