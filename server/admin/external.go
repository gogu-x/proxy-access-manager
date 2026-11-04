package admin

import (
	"github.com/gogu-x/proxy-access-manager/admin/internal"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// NewAdminService 创建管理服务 Actor。
func NewAdminService(database *mongo.Database) *internal.AdminService {
	return internal.NewAdminService(database)
}
