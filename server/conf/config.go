package conf

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/urfave/cli/v3"
)

var (
	Addr           = "10001"
	Bind           = "127.0.0.1"
	MongoURL       = "mongodb://127.0.0.1:27017"
	MongoUsername  = ""
	MongoPassword  = ""
	JWTSecret      = ""
	AdminPassword  = ""
	AccessTTLMin   = 15
	RefreshTTLDays = 7
	XrayBin        = "xray"
	Domain         = ""
	XrayAPI        = "127.0.0.1:10085"
	XrayTag        = "vless-in"
	XrayPort       = 8088
	HysteriaStats  = "http://127.0.0.1:9999"
	HysteriaSecret = ""
	HysteriaPort   = 443
)

// ConnectionFlags 定义服务启动参数。
func ConnectionFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "addr", Value: Addr, Usage: "HTTP 监听端口"},
		&cli.StringFlag{Name: "bind", Value: Bind, Usage: "HTTP 监听地址"},
		&cli.StringFlag{Name: "mongo-url", Value: MongoURL, Usage: "MongoDB 连接地址"},
		&cli.StringFlag{Name: "mongo-username", Usage: "MongoDB 用户名"},
		&cli.StringFlag{Name: "mongo-password", Usage: "MongoDB 密码"},
		&cli.StringFlag{Name: "admin-password", Usage: "管理员初始密码"},
		&cli.StringFlag{Name: "jwt-secret", Usage: "至少 32 字节的 JWT 签名密钥"},
		&cli.StringFlag{Name: "xray-bin", Value: XrayBin, Usage: "Xray 可执行文件的绝对路径"},
		&cli.StringFlag{Name: "domain", Usage: "订阅地址和代理节点使用的域名"},
		&cli.StringFlag{Name: "xray-api", Value: XrayAPI, Usage: "Xray API 地址"},
		&cli.StringFlag{Name: "xray-tag", Value: XrayTag, Usage: "VLESS 入站 tag"},
		&cli.IntFlag{Name: "xray-port", Value: XrayPort, Usage: "VLESS 节点端口"},
		&cli.StringFlag{Name: "hysteria-stats", Value: HysteriaStats, Usage: "Hysteria2 流量统计接口地址"},
		&cli.StringFlag{Name: "hysteria-secret", Usage: "Hysteria2 流量统计接口密钥"},
		&cli.IntFlag{Name: "hysteria-port", Value: HysteriaPort, Usage: "Hysteria2 节点端口"},
	}
}

// LoadAndApply 从启动参数读取配置，不依赖环境变量。
func LoadAndApply(c *cli.Command) error {
	Addr = strings.TrimSpace(c.String("addr"))
	Bind = strings.TrimSpace(c.String("bind"))
	MongoURL = strings.TrimSpace(c.String("mongo-url"))
	MongoUsername = c.String("mongo-username")
	MongoPassword = c.String("mongo-password")
	AdminPassword = c.String("admin-password")
	JWTSecret = c.String("jwt-secret")
	XrayBin = strings.TrimSpace(c.String("xray-bin"))
	Domain = strings.TrimSpace(c.String("domain"))
	XrayAPI = strings.TrimSpace(c.String("xray-api"))
	XrayTag = strings.TrimSpace(c.String("xray-tag"))
	XrayPort = c.Int("xray-port")
	HysteriaStats = strings.TrimRight(strings.TrimSpace(c.String("hysteria-stats")), "/")
	HysteriaSecret = c.String("hysteria-secret")
	HysteriaPort = c.Int("hysteria-port")
	if Addr == "" || Bind == "" || MongoURL == "" || XrayBin == "" {
		return cli.Exit("--addr、--bind、--mongo-url 和 --xray-bin 不能为空", 1)
	}
	if XrayAPI == "" || XrayTag == "" || XrayPort < 1 || XrayPort > 65535 || HysteriaStats == "" || HysteriaPort < 1 || HysteriaPort > 65535 {
		return cli.Exit("代理接口地址、入站 tag 和端口无效", 1)
	}
	return nil
}

func RandomSecret() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "development-secret-change-me"
	}
	return hex.EncodeToString(buf)
}
