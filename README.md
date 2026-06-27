# MiniSnap

MiniSnap 是一个超轻量级的自托管内容发布工具，灵感来源于“贴一下就能分享”的体验。它让你可以把 Markdown 或原始 HTML 文本瞬间变成可分享的网页，同时完全掌控数据。

## 功能概览

- ✅ 自定义登录页，支持会话保持（用户名固定为 `admin`，密码来自环境变量）
- ✅ 支持 Markdown 与原始 HTML 渲染，统一经 HTML 消毒（剥离 `<script>`、内联事件、`javascript:` 链接）
- ✅ 内容储存为纯文件（`content/<slug>.json`），无需数据库
- ✅ 自动生成唯一 slug，并提供查看、编辑链接
- ✅ 可编辑历史内容（`/{slug}/edit`）
- ✅ 后台内容列表与搜索，快速定位历史内容
- ✅ 可选描述字段，丰富内容库摘要
- ✅ 可在后台内容库中删除条目
- ✅ 健康检查端点 `GET /healthz`
- ✅ Markdown 内容页支持亮/暗主题临时切换
- ✅ 页面展示发布时间及最近更新时间

### 安全特性

- **HTML 消毒**：所有渲染产物经 [bluemonday](https://github.com/microcosm-cc/bluemonday) 白名单过滤。Markdown 走严格策略；原始 HTML 在此基础上保留 `<style>` 块与 `style`/`class` 属性、放开结构交互（`<details>`）与媒体（`<video>`/`<audio>`/`<picture>`），但始终剥离 `<script>`、`on*` 事件处理器、`javascript:` 链接，并限制 `<iframe>`/`<form>` 等高风险元素。
- **登录加固**：密码使用恒定时间比较，避免侧信道；基于 IP 的失败计数限流（默认 5 次/分钟触发锁定）。
- **运行时加固**：HTTP server 设置读写/空闲超时；监听 `SIGINT`/`SIGTERM` 实现优雅关停，排空在途连接。

## 快速开始

### 1. 准备环境变量

复制根目录的 `.env.example` 到 `.env` 并修改密码：

```pwsh
Copy-Item .env.example .env
# 然后编辑 .env，更新 ADMIN_PASSWORD
```

MiniSnap 在启动时会自动加载同目录下的 `.env`。若 `.env` 不存在，则回退到系统环境变量。

### 2. 本地运行

```pwsh
go run ./cmd/server
```

> 若需临时覆盖 `.env` 中的配置，可通过命令行参数或环境变量实现，例如 `$env:ADMIN_PASSWORD = "devpass"`。

服务默认监听 `:8080`。首次访问 `http://localhost:8080/login` 输入用户名 `admin` 搭配环境变量指定的密码即可进入后台；会话采用安全 Cookie 维持，可在右上角随时登出。

### 3. Docker 运行

无需本地构建，可直接使用发布在 Docker Hub 的镜像：

```pwsh
docker pull evarle/minisnap:latest
docker run --rm -p 8080:8080 --env-file .env evarle/minisnap:latest
```

后台与内容目录会在容器内 `/app/content`。若希望持久化，请挂载卷：

```pwsh
docker run --rm -p 8080:8080 --env-file .env -v ${PWD}/content:/app/content evarle/minisnap:latest
```

若确实需要自定义镜像，可使用 `docker build -t yourname/minisnap:tag .` 本地构建。

> 镜像内置的默认密码为 `devpass`，强烈建议通过 `.env` 或 `-e ADMIN_PASSWORD=...` 覆盖。



## 配置项

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `ADMIN_PASSWORD` | `devpass` (Docker) / _(必填)_ (本地) | 后台登录密码 |
| `BIND_ADDR` | `:8080` | HTTP 监听地址 |
| `CONTENT_DIR` | `content` | 内容存储目录 |

## 目录结构

```
cmd/server       # 可执行入口
internal/config  # 配置加载
internal/content # 内容存储与渲染
internal/server  # HTTP server 与路由
templates        # HTML 模板
content          # 已发布内容（运行时生成）
```

## 开发说明

- **运行环境**：Go 1.22+
- **模板系统**：HTML 模板位于 `templates/*.tmpl`
- **Markdown 引擎**：使用 `github.com/yuin/goldmark` 提供 GitHub 风格渲染
- **HTML 消毒**：使用 `github.com/microcosm-cc/bluemonday` 对渲染产物做白名单过滤（见上方“安全特性”）
- **存储方式**：文件系统，每个条目对应一个 JSON 文件；列表读取会跳过损坏条目并记日志，单条坏数据不影响整库可用性
- **会话管理**：基于安全 HTTP Cookie，支持登录状态保持
- **构建优化**：Docker 多阶段构建，最终镜像约 20MB

### 常用命令

仓库内提供 `Makefile`，可快速执行常见操作：

```pwsh
make build          # 编译生成 bin/minisnap
make test           # 运行 go test ./...
make lint           # 检查 gofmt + go vet
make docker-build   # 构建本地 Docker 镜像（默认标签 evarle/minisnap:latest）
make docker-push    # 推送镜像至 Docker Hub（需先 docker login）
make docker-run     # 以当前目录内容启动容器
```

可通过环境变量覆盖默认镜像标签，例如：`make docker-build IMAGE=myname/minisnap TAG=v1`。

### 调试技巧

- **本地热重载**：配合 [air](https://github.com/cosmtrek/air) 等工具可以监听文件改动自动重启进程。
- **日志输出**：服务使用 `slog`，临时调试可在关键路径（如 `internal/server`）增加 `slog.Debug` 日志，运行时通过 `GODEBUG=slogtostderr=1` 输出到标准错误。
- **断点调试**：通过 `dlv`（[Delve](https://github.com/go-delve/delve)）运行：

```pwsh
dlv debug ./cmd/server -- -admin-password=devpass -content-dir=content
```

- **查看存档内容**：所有条目以 JSON 存储，可直接在 `content/<slug>.json` 中查看/编辑原始数据。

### 运行测试

```pwsh
go test ./...
```

上述命令也可通过 `make test` 快速执行。

### 持续集成

仓库包含 GitHub Actions 工作流 `.github/workflows/ci.yml`，在 `push` 或 `pull request` 时自动执行：

- `gofmt` 检查（格式不符直接失败）
- `go vet` 静态分析
- `go test ./...` 单元测试
- `go build ./cmd/server` 编译验证
- `docker build`（仅验证 Dockerfile 是否可构建）

当推送到 `main` 且仓库配置了 Secrets `DOCKERHUB_USERNAME` 与 `DOCKERHUB_TOKEN`（建议使用 Docker Hub Access Token）时，CI 会额外执行
`docker/build-push-action`，将镜像发布到 `evarle/minisnap:latest` 与 `evarle/minisnap:<commit-sha>`。

> 配置方式：进入 GitHub 仓库 Settings → Secrets and variables → Actions，分别新增
> `DOCKERHUB_USERNAME`（Docker Hub 用户名）与 `DOCKERHUB_TOKEN`（Docker Hub Access Token）。

默认使用 Go 1.22。若需扩展（如多架构构建、Helm 打包等），可在现有工作流基础上新增作业或步骤。

## 使用示例

### 发布 Markdown 内容
1. 访问 `http://localhost:8080/login` 登录后台
2. 在文本框中输入 Markdown 内容
3. 可选填写描述信息便于后续查找
4. 点击发布，系统会自动生成短链接

### 管理已发布内容
- 访问 `/admin/library` 查看所有已发布内容
- 使用搜索功能快速定位特定内容
- 点击 "分享" 复制链接，点击 "删除" 移除内容
- 点击标题进入编辑页面修改内容

### 生产环境部署
```pwsh
# 1. 获取官方镜像
docker pull evarle/minisnap:latest

# 2. 创建数据目录
mkdir -p ./data/content

# 3. 运行容器（数据持久化）
docker run -d --name minisnap \
  -p 8080:8080 \
  -e ADMIN_PASSWORD=your-secure-password \
  -v ./data/content:/app/content \
  --restart unless-stopped \
  evarle/minisnap:latest
```

## 后续拓展想法

- 私钥/Token 级别编辑链接
- 内容版本历史
- API 支持与 CLI 客户端
- 批量导入/导出功能
- 自定义主题支持

欢迎根据需求自行扩展。祝玩得开心 🎉
