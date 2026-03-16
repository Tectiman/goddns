# goddns - 强大的动态 DNS 客户端

[goddns](./goddns) 是一个用 Go 编写的轻量级且功能强大的动态 DNS (DDNS) 客户端。它自动更新 DNS 记录，支持多域名多服务商，支持 IPv6，具备跨平台能力和丰富的日志输出。

## v2.0 重构说明

**v2.0 进行了重大重构**，从单域名单服务商模式升级为支持**多域名多服务商**的现代化架构。

### 主要变化

- ✅ **多域名支持**：一条配置可更新多个 DNS 记录
- ✅ **多服务商支持**：支持 Cloudflare、阿里云 DNS
- ✅ **记录级代理控制**：每个记录可独立配置是否使用代理
- ✅ **get_ip 直连**：IP 获取功能始终直连，不受代理影响
- ✅ **并发更新**：多个域名并行更新，提高效率

## 平台支持说明
- **多平台适配**：Linux 使用 netlink，FreeBSD/openBSD 使用 ioctl。
- **macOS 支持说明**：由于没有 mac 设备进行测试，暂时未支持 macOS，欢迎有能力的开发者提交 PR 以完善 macOS 兼容性。

## 特性
- **多域名支持**：单次运行可更新多个 DNS 记录
- **Cloudflare 集成**：自动更新 Cloudflare DNS 记录
- **阿里云 DNS**：自动更新阿里云 DNS 记录
- **IPv6 支持**：原生支持 IPv6，支持多平台接口获取
- **代理支持**：支持 HTTP(S)/SOCKS5 代理，支持记录级控制（仅 Cloudflare）
- **IP 缓存**：避免重复 API 调用
- **彩色日志**：终端下日志分级彩色显示，支持文件输出
- **配置灵活**：JSON 配置，模块化设计
- **版本管理**：支持编译时注入版本信息

## 快速开始

### 构建

#### 基础构建
```bash
go build -o goddns ./cmd/goddns
```

#### 带版本信息构建
```bash
# 简单版本构建
go build -ldflags "-X main.version=v2.0.0" -o goddns ./cmd/goddns

# 完整版本信息构建
go build -ldflags "-X main.version=v2.0.0 -X main.commit=$(git rev-parse HEAD) -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o goddns ./cmd/goddns
```

#### 使用脚本自动化构建
```bash
chmod +x build.sh
./build.sh v2.0.0    # 构建指定版本
./build.sh           # 构建开发版本
```

### 运行
```bash
./goddns run -f config.json
# -i 可选，忽略缓存强制更新
```

### 显示版本
```bash
./goddns version
```

## 配置示例

```json
{
    "general": {
        "get_ip": {
            "interface": "enp6s18",
            "urls": [
                "https://ipv6.icanhazip.com",
                "https://6.ipw.cn"
            ]
        },
        "work_dir": "/var/lib/goddns",
        "log_output": "shell",
        "proxy": ""
    },
    "records": [
        {
            "provider": "cloudflare",
            "zone": "example.com",
            "record": "dev",
            "ttl": 180,
            "proxied": false,
            "use_proxy": false,
            "cloudflare": {
                "api_token": "${CLOUDFLARE_API_TOKEN}",
                "zone_id": "${CLOUDFLARE_ZONE_ID:-}"
            }
        },
        {
            "provider": "aliyun",
            "zone": "example.cn",
            "record": "www",
            "ttl": 600,
            "use_proxy": false,
            "aliyun": {
                "access_key_id": "${ALIYUN_ACCESS_KEY_ID}",
                "access_key_secret": "${ALIYUN_ACCESS_KEY_SECRET}"
            }
        }
    ]
}
```

## 环境变量

goddns 支持使用环境变量配置敏感信息，推荐在生产环境中使用。

### 支持的环境变量

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `CLOUDFLARE_API_TOKEN` | Cloudflare API Token | `your_api_token_here` |
| `CLOUDFLARE_ZONE_ID` | Cloudflare Zone ID（可选） | `abc123xyz` |
| `ALIYUN_ACCESS_KEY_ID` | 阿里云 AccessKey ID | `LTAI1234567890` |
| `ALIYUN_ACCESS_KEY_SECRET` | 阿里云 AccessKey Secret | `your_secret_here` |

### 使用方式

在配置文件中引用环境变量：

```json
{
    "records": [
        {
            "provider": "cloudflare",
            "cloudflare": {
                "api_token": "${CLOUDFLARE_API_TOKEN}"
            }
        }
    ]
}
```

运行前设置环境变量：

```bash
# 设置环境变量
export CLOUDFLARE_API_TOKEN="your_token_here"
export ALIYUN_ACCESS_KEY_ID="LTAI1234567890"
export ALIYUN_ACCESS_KEY_SECRET="your_secret_here"

# 运行 goddns
./goddns run -f config.json
```

### 环境变量默认值

支持设置默认值（当环境变量未设置时使用默认值）：

```json
{
    "cloudflare": {
        "zone_id": "${CLOUDFLARE_ZONE_ID:-}"
    }
}
```

`${VAR:-default}` - 如果 VAR 未设置，使用 default 值

## 配置字段说明

### general（全局配置）

| 字段 | 说明 |
|------|------|
| `get_ip.interface` | 本地网卡名，优先使用此方式获取 IP |
| `get_ip.urls` | 外部 IP 检测 API 列表（当 interface 不可用时使用） |
| `work_dir` | 缓存文件目录 |
| `log_output` | 日志输出路径，使用 `shell` 表示输出到终端 |
| `proxy` | 全局代理配置（可选），格式：`socks5://127.0.0.1:1080` |

### records（记录数组）

| 字段 | 说明 |
|------|------|
| `provider` | DNS 服务商，支持 `cloudflare` 或 `aliyun` |
| `zone` | 主域名（如 `example.com`） |
| `record` | 子域名/记录名（如 `dev`、`www`、`@`） |
| `ttl` | DNS 记录 TTL（可选） |
| `proxied` | Cloudflare 代理模式（可选，默认 false，仅 Cloudflare） |
| `use_proxy` | 是否使用全局代理（可选，默认 false，仅 Cloudflare） |

### Cloudflare 配置

| 字段 | 说明 |
|------|------|
| `cloudflare.api_token` | Cloudflare API Token |
| `cloudflare.zone_id` | Cloudflare Zone ID（可选，留空自动获取） |
| `cloudflare.ttl` | TTL（可选，覆盖记录级 ttl） |
| `cloudflare.proxied` | 代理模式（可选，覆盖记录级 proxied） |

### 阿里云配置

| 字段 | 说明 |
|------|------|
| `aliyun.access_key_id` | 阿里云 AccessKey ID |
| `aliyun.access_key_secret` | 阿里云 AccessKey Secret |
| `aliyun.ttl` | TTL（可选，覆盖记录级 ttl） |

## 服务商对比

| 特性 | Cloudflare | 阿里云 |
|------|------------|--------|
| IPv6 支持 | ✅ | ✅ |
| 代理支持 | ✅ | ❌ |
| 自动获取 ZoneID | ✅ | ✅ |
| API 认证 | API Token | AccessKey |

## 代理说明

- **get_ip 始终直连**：获取 IP 地址时不使用代理
- **记录级代理控制**：每个 DNS 记录可独立配置 `use_proxy`
- **仅 Cloudflare 支持代理**：阿里云 DNS API 不支持代理
- **代理优先级**：记录级 `use_proxy: true` + 全局 `proxy` 配置

## 自动运行

### systemd 定时

#### 方式一：在配置文件中引用环境变量

创建 `/etc/systemd/system/goddns.service`：
```ini
[Unit]
Description=Dynamic DNS client for GodDNS
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/goddns run -f /etc/goddns/config.json
User=nobody
Group=nogroup
Environment="CLOUDFLARE_API_TOKEN=your_token_here"
Environment="ALIYUN_ACCESS_KEY_ID=LTAI1234567890"
Environment="ALIYUN_ACCESS_KEY_SECRET=your_secret_here"
```

#### 方式二：使用 EnvironmentFile（推荐）

创建 `/etc/goddns/goddns.env`：
```bash
# /etc/goddns/goddns.env
CLOUDFLARE_API_TOKEN="your_token_here"
CLOUDFLARE_ZONE_ID=""
ALIYUN_ACCESS_KEY_ID="LTAI1234567890"
ALIYUN_ACCESS_KEY_SECRET="your_secret_here"
```

设置文件权限（仅 root 可读）：
```bash
chmod 600 /etc/goddns/goddns.env
```

创建 `/etc/systemd/system/goddns.service`：
```ini
[Unit]
Description=Dynamic DNS client for GodDNS
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/goddns run -f /etc/goddns/config.json
User=nobody
Group=nogroup
EnvironmentFile=/etc/goddns/goddns.env
```

创建 `/etc/systemd/system/goddns.timer`：
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

启用：
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now goddns.timer
```

### cron 定时

#### 方式一：在 crontab 中设置环境变量

```bash
crontab -e

# 添加环境变量和任务
CLOUDFLARE_API_TOKEN="your_token_here"
ALIYUN_ACCESS_KEY_ID="LTAI1234567890"
ALIYUN_ACCESS_KEY_SECRET="your_secret_here"

*/5 * * * * /usr/local/bin/goddns run -f /etc/goddns/config.json >> /var/log/goddns-cron.log 2>&1
```

#### 方式二：使用脚本加载环境变量（推荐）

创建 `/etc/goddns/run-goddns.sh`：
```bash
#!/bin/bash
# /etc/goddns/run-goddns.sh

# 加载环境变量文件
if [ -f /etc/goddns/goddns.env ]; then
    set -a
    source /etc/goddns/goddns.env
    set +a
fi

# 运行 goddns
/usr/local/bin/goddns run -f /etc/goddns/config.json
```

设置脚本权限：
```bash
chmod +x /etc/goddns/run-goddns.sh
```

配置 crontab：
```bash
crontab -e
# 添加：
*/5 * * * * /etc/goddns/run-goddns.sh >> /var/log/goddns-cron.log 2>&1
```

#### 方式三：直接在 crontab 中使用 env 命令

```bash
crontab -e
# 添加：
*/5 * * * * env CLOUDFLARE_API_TOKEN="your_token" /usr/local/bin/goddns run -f /etc/goddns/config.json >> /var/log/goddns-cron.log 2>&1
```

> **注意**：cron 环境与交互式 shell 不同，可能需要：
> - 使用绝对路径
> - 在脚本中设置 `PATH` 环境变量
> - 确保环境变量文件权限正确（`chmod 600`）

## 目录结构
```
goddns/
├── cmd/goddns/           # 主程序入口
│   ├── main.go
│   └── cmd.go
├── internal/
│   ├── config/           # 配置管理
│   ├── log/              # 日志系统
│   ├── platform/ifaddr/  # 平台相关网络工具
│   └── provider/         # DNS 服务商实现
│       ├── cloudflare/   # Cloudflare API
│       └── aliyun/       # 阿里云 DNS API
└── config.example.json   # 配置示例
```

## 构建参数

### ldflags 参数
- `-X main.version`：设置程序版本号
- `-X main.commit`：设置 Git 提交哈希
- `-X main.buildDate`：设置构建时间（UTC 格式）

### 环境变量
```bash
export VERSION=v2.0.0
export COMMIT=$(git rev-parse HEAD)
export BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" -o goddns ./cmd/goddns
```

## 许可证
请见 LICENSE 文件。

如需更多帮助或反馈建议，请提交 issue。
