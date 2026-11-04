package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/gogu-x/proxy-access-manager/admin"
	"github.com/gogu-x/proxy-access-manager/conf"
	"github.com/gogu-x/tree"
	"github.com/gogu-x/tree/db/mongorpc"
	"github.com/urfave/cli/v3"
)

func main() {
	command := &cli.Command{
		Name:  "proxy-access-manager",
		Usage: "启动密钥管理服务，密钥操作请在管理页面完成",
		Flags: conf.ConnectionFlags(),
		Action: func(_ context.Context, c *cli.Command) error {
			if err := conf.LoadAndApply(c); err != nil {
				return err
			}
			if conf.AdminPassword == "" || len(conf.JWTSecret) < 32 {
				return errors.New("请通过 --admin-password 和 --jwt-secret 指定管理员初始密码及至少 32 字节的签名密钥")
			}
			database := mongorpc.Connect(conf.MongoURL, conf.MongoUsername, conf.MongoPassword, "keymanager")
			if database == nil {
				return errors.New("MongoDB 连接失败")
			}
			tree.Spawn(admin.NewAdminService(database))
			tree.Default().Start()
			return nil
		},
	}
	if err := command.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
