# 代理访问管理（Proxy Access Manager）

这是一个独立项目，前后端从 `/data/g1/ops/frontend` 与 `/data/g1/ops/server` 的源码复制后修改。服务保留 ops 的 Go Actor 生命周期、`admin/internal` 认证与仓储、Gin 路由、Vue Router、Pinia、Axios 客户端和 `AppLayout`；移除了主机、项目和容器业务，加入 Hysteria2 / Xray 密钥管理。

## 本机启动

需要可访问的 MongoDB、Xray 和 Hysteria2 服务。管理员账户名为 `admin`，首次运行时沿用 ops 的认证服务创建账户。

```bash
cd server
go run . --admin-password '请设置强密码' \
  --jwt-secret '请设置至少32字节的随机签名密钥' \
  --mongo-url 'mongodb://127.0.0.1:27017' \
  --domain 'secure3.cn' \
  --hysteria-secret '与 deploy/hysteria.yaml 中 trafficStats.secret 相同的密钥'
```

有 MongoDB 认证时再添加 `--mongo-username`、`--mongo-password`；监听地址和端口可用 `--bind`、`--addr` 指定。启动参数中的密码会出现在本机进程参数中，应限制服务器进程查看权限。管理员密码仅用于首次创建账户，JWT 密钥应在重启后保持不变。

后端默认监听 `127.0.0.1:10001`。前端在另一终端执行：

```bash
cd frontend && npm ci && npm run dev
```

打开 `http://127.0.0.1:5173/`，使用 `admin` 和 `--admin-password` 指定的初始密码登录。部署到容器时添加 `--bind 0.0.0.0`，并限制后端端口只被前端容器访问。

密钥存储在 MongoDB 的 `keymanager.proxy_keys` 集合；为兼容已有数据，数据库名仍保留 `keymanager`。管理员与刷新令牌沿用复制的 ops 认证仓储。旧版独立实现使用 SQLite；本次结构调整没有自动导入旧 SQLite 文件。若旧库中已有实际 key，请先迁移再切换流量。本工作区未发现旧版 `data/keys.db`。

## 代理接入

服务器配置示例保存在 `deploy/hysteria.yaml` 和 `deploy/xray.json`。复制到服务器前，替换 Hysteria2 流量统计密钥，并确认代理访问管理服务、Hysteria2 与 Xray 共享本机网络；容器或分机部署需要调整 `127.0.0.1` 地址。Xray 示例的 `clients` 为空，启用后旧共享 UUID 不再可用；迁移期间应先保留旧用户，验证新用户后再移除。两份示例均不包含原共享凭据。


Hysteria2 服务端保留现有 TLS、带宽和混淆配置，另配置：

```yaml
auth:
  type: http
  http:
    url: http://127.0.0.1:10001/auth/hysteria
trafficStats:
  listen: 127.0.0.1:9999
  secret: 请设置独立随机令牌
```

将相同令牌通过后端启动参数 `--hysteria-secret` 指定。如果管理服务在容器中，回调地址须改为 Hysteria 可达的容器地址，统计 API 地址通过 `--hysteria-stats` 配置。

Xray 的 VLESS 入站需要稳定的 `tag`，并开启本机 API 和用户统计：

```json
"api": { "tag": "api", "listen": "127.0.0.1:10085", "services": ["HandlerService", "StatsService"] },
"stats": {},
"policy": { "levels": { "0": { "statsUserUplink": true, "statsUserDownlink": true } } }
```

服务调用已安装的 `xray api inbounduser/adu/rmu/stats` 命令；可执行文件路径通过启动参数 `--xray-bin /usr/local/bin/xray` 指定，接口地址由 `--xray-api` 指定。管理服务运行环境（包括容器内）必须有兼容的 Xray 可执行文件；只提供远程 Xray API 地址仍不足以执行当前命令。简化 `api.listen` 需要 Xray 1.8.12 或更新版本，还需确认实际版本支持 `inbounduser` 命令。

## 使用与边界

- 首页展示用户数、可用数、累计流量、同步异常、流量排行及最近创建；独立的用户管理页支持创建、启用、停用，详情中可编辑续期天数、删除用户、查看额度和复制每个用户专属的 Clash 订阅地址。删除时先停用凭据并清理 Xray/Hysteria，再移除数据库记录；清理失败时保留已停用记录以便重试。管理接口要求 ops 的管理员角色。
- 列表、创建、续期、启用、停用和立即同步统一在管理页面操作；后端命令行只用于启动服务。
- Hysteria2 在建立连接时调用认证接口；Xray 对新连接检查用户列表。后台启动时采集一次双协议流量，随后每分钟采集并同步到期或超额状态；管理页的“立即同步”可随时触发采集。额度可能在采样间隔内短时超出。
- 用户详情可切换最近 60 分钟的分钟曲线、最近 24 小时的小时曲线、最近 30 天的天曲线。流量增量按 UTC 时间桶保存，分钟、小时、天数据分别保留约 60 分钟、24 小时、30 天；旧的日汇总继续显示，旧累计流量不会反推分钟或小时曲线。桌面端可以拖动详情抽屉左边缘调整宽度。
- Hysteria2 会对停用用户调用 `/kick`；Xray 动态删除是否立即断开已有连接取决于实际版本，当前不承诺即时断线。
- 订阅地址由 `--domain` 生成，形如 `https://secure3.cn/sub/<令牌>`；前端 Nginx 已转发 `/sub/` 到后端。若外层还有 Nginx，也须将该路径转发到代理访问管理服务。Hysteria2、VLESS 节点使用同一个用户 key，VLESS 默认 TCP/TLS/Vision。停用、过期或超额后订阅返回 404；已下载配置依赖代理认证继续管控。签名令牌由 `--jwt-secret` 派生，更换该密钥会让旧订阅地址失效。交付前请核对现网参数。
- 订阅 YAML 使用 `server/admin/internal/key/clash-template.yaml`，保留现网 Clash 文件的代理组、DNS、TUN 和规则，仅在两个节点替换用户 key、域名和端口。模板通过 Go embed 打入后端镜像；修改模板后需重新构建后端。原文件中的共享密码与 UUID 未写入模板。该模板包含 FlClash 的本地设置，导入时可能覆盖客户端已有的 TUN、DNS 和监听配置。
- Hysteria2 订阅节点默认使用上行 `20 Mbps`、下行 `100 Mbps`，详情页可按用户修改并持久化。FlClash/Mihomo 节点使用平级的 `up`、`down` 字段，用户刷新订阅后生效；这是客户端的带宽参数，不能作为防止用户修改配置的服务端限速。Hysteria2 原生客户端的同类配置才使用嵌套的 `bandwidth.up/down`。服务器 `deploy/hysteria.yaml` 不设置全局带宽，以免影响全部用户。

前端镜像内的 Nginx 将后端服务解析为 `proxy-access-manager:10001`，部署时应使用相同的服务名或网络别名。

## 镜像打包

打包方式与 ops 项目一致：根目录 `Jenkinsfile` 分别构建 Go 后端与 Vue 前端镜像，可选推送；本机命令见 `DOCKER.md`。

## 结构

- `server/main.go`：保留 ops 的 Actor 启动方式，只接受服务启动参数。
- `server/admin/internal`：复制 ops 的 AdminService、认证服务、Gin 路由，并加入密钥处理函数。
- `server/admin/internal/key`：密钥业务服务、Mongo 仓储、Xray/Hysteria 接入。
- `frontend/src/api`、`stores`、`router`、`components`、`views`：沿用 ops 的前端分层与布局，新增密钥 API 和页面。
