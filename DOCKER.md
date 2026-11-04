# Docker 镜像打包

本项目与 ops 一样，分别以 `server/` 和 `frontend/` 为构建上下文，生成后端和前端两个镜像。Jenkins 使用根目录 `Jenkinsfile`，从当前任务配置的仓库检出代码，按分支及提交号打标签；`PUSH=true` 时使用 Jenkins 凭据 `dockerhub-xiaoguyun` 推送。

本机可执行：

```bash
docker build -f server/Dockerfile -t proxy-access-manager-backend:local server
docker build -f frontend/Dockerfile -t proxy-access-manager-web:local frontend
```

后端镜像只包含代理访问管理服务，不包含 Xray。运行时需要让容器内有兼容的 `xray` 可执行文件，并用 `--xray-bin` 指向它；同时配置管理端到 MongoDB、Xray API 和 Hysteria2 统计 API 的网络连通性。服务启动配置使用 `--admin-password`、`--jwt-secret`、`--mongo-url`、`--domain`、`--hysteria-secret` 等参数，详见 `go run . --help`。不要把这些值写进镜像。
