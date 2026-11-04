package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gogu-x/proxy-access-manager/admin/internal/auth"
	"github.com/gogu-x/proxy-access-manager/admin/internal/key"
	"github.com/gogu-x/proxy-access-manager/admin/internal/store"
	"github.com/gogu-x/proxy-access-manager/conf"
	"github.com/gogu-x/tree"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// AdminService 保留 ops 管理 Actor 的服务生命周期。
type AdminService struct {
	auth *auth.Service
	keys *key.Service
	http *http.Server
	stop context.CancelFunc
}

func NewAdminService(database *mongo.Database) *AdminService {
	config := key.LoadConfig()
	return &AdminService{auth: auth.NewService(store.NewMongoRepository(database)), keys: key.NewService(key.NewMongoRepository(database), key.NewProxy(config), config)}
}
func (a *AdminService) Name() string { return "proxy-access-manager-http" }
func (a *AdminService) OnInit(_ tree.Context) {
	if err := a.auth.Bootstrap(context.Background()); err != nil {
		panic("bootstrap admin: " + err.Error())
	}
	gin.SetMode(gin.ReleaseMode)
	a.http = &http.Server{Addr: fmt.Sprintf("%s:%v", conf.Bind, conf.Addr), Handler: a.newRouter(), ReadHeaderTimeout: 5 * time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	a.stop = cancel
	go a.keys.Run(ctx)
	go func() {
		if err := a.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			println("[proxy-access-manager] http server:", err.Error())
		}
	}()
	println("[proxy-access-manager] HTTP server listening on", conf.Addr)
}
func (a *AdminService) HandleMessage(_ tree.Context, _ interface{}) {}
func (a *AdminService) OnStop(_ tree.Context) {
	if a.stop != nil {
		a.stop()
	}
	if a.http != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = a.http.Shutdown(ctx)
		cancel()
	}
}
