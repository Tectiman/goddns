# goddns - 强大的动态 DNS 客户端

`goddns` 是一个用 Go 编写的轻量级且功能强大的动态 DNS (DDNS) 客户端。它旨在自动更新您的 Cloudflare DNS 记录，特别是支持 IPv6，并通过 Linux netlink 进行 IP 地址检索。

## 特性

*   **Cloudflare 集成**: 与 Cloudflare API 无缝集成以更新 DNS 记录。
*   **IPv6 支持**: 专门为 IPv6 环境设计，确保您的 IPv6 地址始终保持最新。
*   **Linux Netlink 驱动**: 利用 Linux netlink 机制准确获取本地网络接口的 IP 地址。
*   **代理支持**: 可以通过 HTTP(S) 或 SOCKS5 代理路由所有 Cloudflare API 请求。
*   **IP 地址缓存**: 缓存上次更新的 IP 地址以避免不必要的 API 调用和速率限制。
*   **可配置性**: 通过简单的 JSON 配置文件轻松配置提供商、IP 获取方法和域信息。

## 构建

### 简单构建

```bash
go build -o goddns ./cmd/goddns
```

### 带版本信息构建

您可以在构建时嵌入版本、提交和构建日期信息：

```bash
# 示例: 设置版本、提交和构建日期
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=1.2.3 -X main.commit=$(git rev-parse --short HEAD) -X 'main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o goddns ./cmd/goddns
```

## 使用

### 显示版本信息

```bash
./goddns -v
# 或
./goddns --version
```

### 运行

```bash
./goddns run -f /path/to/config.json [-i]
```
`-i` 参数用于忽略缓存并强制更新 IP。

## 配置

`goddns` 通过 `config.json` 文件进行配置。以下是一个示例配置片段及其解释：

```json
{
    "provider": "cloudflare",
    "get_ip": {
        "interface": "enp6s18",
        "url": "https://ipv6.icanhazip.com"
    },
    "work_dir": "",
    "provider_options": {
        "api_token": "enc:YOUR_ENCRYPTED_API_TOKEN",
        "zone_id": "enc:YOUR_ENCRYPTED_ZONE_ID",
        "proxied": false,
        "ttl": 180,
        "domain": {
            "zone": "yourdomain.com",
            "record": "sub"
        }
    },
    "proxy": "socks5://127.0.0.1:1080"
}
```

### 配置字段说明

*   **`provider`**: (字符串) DNS 服务提供商，目前支持 `"cloudflare"`。
*   **`get_ip`**: (对象) IP 地址获取配置。
    *   **`interface`**: (字符串) 用于获取 IP 地址的网络接口名称（例如 `"enp6s18"`）。
    *   **`url`**: (字符串) 用于外部 IP 地址检测的 URL（例如 `"https://ipv6.icanhazip.com"`）。
*   **`work_dir`**: (字符串) 工作目录，用于存放 `cache.lastip` 文件。如果为空，则使用可执行文件所在的目录。
*   **`provider_options`**: (对象) 特定于提供商的配置选项。
    *   **`api_token`**: (字符串) Cloudflare API 令牌。建议使用加密形式（`enc:YOUR_ENCRYPTED_API_TOKEN`）。
    *   **`zone_id`**: (字符串) Cloudflare 区域 ID。建议使用加密形式（`enc:YOUR_ENCRYPTED_ZONE_ID`）。
    *   **`proxied`**: (布尔值) 是否将 DNS 记录设置为 Cloudflare 代理（CDN）。
    *   **`ttl`**: (整数) DNS 记录的生存时间 (TTL)，单位为秒。
    *   **`domain`**: (对象) 域信息。
        *   **`zone`**: (字符串) 您的主域名（例如 `"contactofen.site"`）。
        *   **`record`**: (字符串) 要更新的子域记录（例如 `"dev"`）。
*   **`proxy`**: (字符串，可选) 代理服务器配置。
    *   必须包含方案。支持的方案：`http://host:port`、`https://host:port` 或 `socks5://host:port`、`socks5h://host:port`。
    *   如果设置了 SOCKS5 代理，程序将对所有 Cloudflare API 请求使用它。
    *   如果 `proxy` 为空或省略，则使用直接连接。

## 项目结构

该项目采用以下结构组织：

*   [`cmd/goddns/`](cmd/goddns/main.go): `main` 包和命令行入口点。
*   [`internal/config/`](internal/config/config.go): 配置解析和缓存辅助函数。
*   [`internal/netlink/`](internal/netlink/netlinkutil.go): Linux netlink 交互和 IPv6 选择逻辑。
*   [`internal/platform/netlinkutil/`](internal/platform/netlinkutil/netlinkutil.go): 包含平台特定的 netlink 工具。
*   [`internal/provider/cloudflare/`](internal/provider/cloudflare/cloudflare.go): Cloudflare API 实现。
*   [`internal/log/`](internal/log/log.go): 日志辅助函数。

## 许可证

该项目根据 [LICENSE](goddns/LICENSE) 文件中的条款获得许可。
