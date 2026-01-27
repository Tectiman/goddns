
# goddns - 强大的动态 DNS 客户端

[goddns](file:///home/jasper/code_project/Go/goddns/goddns) 是一个用 Go 编写的轻量级且功能强大的动态 DNS (DDNS) 客户端。它旨在自动更新您的 Cloudflare DNS 记录，特别是支持 IPv6，并通过 Linux netlink 进行 IP 地址检索。

## 特性

*   **Cloudflare 集成**: 与 Cloudflare API 无缝集成以更新 DNS 记录。
*   **IPv6 支持**: 专门为 IPv6 环境设计，确保您的 IPv6 地址始终保持最新。
*   **Linux Netlink 驱动**: 利用 Linux netlink 机制准确获取本地网络接口的 IP 地址。
*   **代理支持**: 可以通过 HTTP(S) 或 SOCKS5 代理路由所有 Cloudflare API 请求。
*   **IP 地址缓存**: 缓存上次更新的 IP 地址以避免不必要的 API 调用和速率限制。
*   **灵活的日志配置**: 支持将日志输出到控制台和/或文件，便于调试和生产环境监控。
*   **可配置性**: 通过简单的 JSON 配置文件轻松配置提供商、IP 获取方法和域信息。

## 构建

### 简单构建

```bash
go build -o goddns ./cmd/goddns
带版本信息构建
您可以在构建时嵌入版本、提交和构建日期信息：

bash
# 示例: 设置版本、提交和构建日期
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=1.2.3 -X main.commit=$(git rev-parse --short HEAD) -X 'main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o goddns ./cmd/goddns
使用
显示版本信息
bash
./goddns -v
# 或
./goddns --version
运行
bash
./goddns run -f /path/to/config.json [-i]
-i 参数用于忽略缓存并强制更新 IP。

配置
goddns 通过 config.json 文件进行配置。以下是一个示例配置片段及其解释：

json
{
    "provider": "cloudflare",
    "get_ip": {
        "interface": "enp6s18",
        "url": "https://ipv6.icanhazip.com"
    },
    "work_dir": "",
    "log_output": "./goddns.log",
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
## 配置说明

goddns 通过 JSON 格式的配置文件进行配置。以下是完整的配置选项说明：

```json
{
    "provider": "cloudflare",
    "get_ip": {
        "interface": "enp6s18",
        "urls": [
            "https://ipv6.icanhazip.com",
            "https://6.ipw.cn"
        ]
    },
    "work_dir": "/var/lib/goddns",
    "log_output": "/var/log/goddns.log",
    "provider_options": {
        "api_token": "YOUR_API_TOKEN",
        "zone_id": "YOUR_ZONE_ID",
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

### 配置字段详细说明
* provider: (字符串) DNS 服务提供商，目前支持 "cloudflare"。
* get_ip: (对象) IP 地址获取配置。
* interface: (字符串) 用于获取 IP 地址的网络接口名称（例如 "enp6s18"）。
* url: (字符串) 用于外部 IP 地址检测的 URL（例如 "https://ipv6.icanhazip.com"）。
* work_dir: (字符串，可选) 工作目录，用于存放 cache.lastip 文件。如果为空，则使用可执行文件所在的目录。建议设置为绝对路径，如 "/var/lib/goddns"
* log_output: (字符串，可选) 日志输出配置。可选值：
    - 留空或设置为 "shell": 日志输出到终端
    - 文件路径 (如 "/var/log/goddns.log"): 日志输出到指定文件
* provider_options: (对象) 特定于提供商的配置选项。
* api_token: (字符串) Cloudflare API 令牌。
* zone_id: (字符串) Cloudflare 区域 ID。
* proxied: (布尔值) 是否将 DNS 记录设置为 Cloudflare 代理（CDN）。
* ttl: (整数) DNS 记录的生存时间 (TTL)，单位为秒。
* domain: (对象) 域信息。
* zone: (字符串) 您的主域名（例如 "contactofen.site"）。
* record: (字符串) 要更新的子域记录（例如 "dev"）。
* proxy: (字符串，可选) 代理服务器配置。必须包含方案。支持的方案：http://host:port、https://host:port 或 socks5://host:port、socks5h://host:port。
    如果设置了 SOCKS5 代理，程序将对所有 Cloudflare API 请求使用它。
    如果 proxy 为空或省略，则使用直接连接。
## 自动运行配置

### 使用 systemd timer (推荐)

1. 创建 systemd 服务文件 `/etc/systemd/system/goddns.service`：

```ini
[Unit]
Description=Dynamic DNS client for Cloudflare
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/goddns run -f /etc/goddns/config.json
User=nobody
Group=nogroup
```

2. 创建 timer 文件 `/etc/systemd/system/goddns.timer`：

```ini
[Unit]
Description=Run goddns every 5 minutes

[Timer]
OnBootSec=5min
OnUnitActiveSec=5min
Persistent=true

[Install]
WantedBy=timers.target
```

3. 启用并启动 timer：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now goddns.timer
```

### 使用 cron

1. 编辑 crontab：

```bash
crontab -e
```

2. 添加以下行（每5分钟运行一次）：

```bash
*/5 * * * * /usr/local/bin/goddns run -f /etc/goddns/config.json >> /var/log/goddns-cron.log 2>&1
```

## 项目结构
该项目采用以下结构组织：

cmd/goddns/: main 包和命令行入口点。
internal/config/: 配置解析和缓存辅助函数。
internal/netlink/: Linux netlink 交互和 IPv6 选择逻辑。
internal/platform/netlinkutil/: 包含平台特定的 netlink 工具。
internal/provider/cloudflare/: Cloudflare API 实现。
internal/log/: 日志辅助函数。
许可证
该项目根据 LICENSE 文件中的条款获得许可。
