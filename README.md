# goddns - 强大的动态 DNS 客户端

[goddns](./goddns) 是一个用 Go 编写的轻量级且功能强大的动态 DNS (DDNS) 客户端。它自动更新 Cloudflare DNS 记录，支持 IPv6，具备跨平台能力和丰富的日志输出。

## 特性
- **Cloudflare 集成**：自动更新 Cloudflare DNS 记录。
- **IPv6 支持**：原生支持 IPv6，支持多平台接口获取。
- **多平台适配**：Linux 使用 netlink，FreeBSD/macOS 使用 ioctl/ifconfig。
- **代理支持**：支持 HTTP(S)/SOCKS5 代理。
- **IP 缓存**：避免重复 API 调用。
- **彩色日志**：终端下日志分级彩色显示，支持文件输出。
- **配置灵活**：JSON 配置，支持多种 IP 获取方式。
- **版本管理**：支持编译时注入版本信息。

## 快速开始

### 构建

#### 基础构建
```bash
go build -o goddns ./cmd/goddns
```

#### 带版本信息构建
```bash
# 简单版本构建
go build -ldflags "-X main.version=v1.0.0" -o goddns ./cmd/goddns

# 完整版本信息构建
go build -ldflags "-X main.version=v1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o goddns ./cmd/goddns
```

#### 使用脚本自动化构建
创建 `build.sh` 脚本：
```bash
#!/bin/bash
VERSION=${1:-"dev"}
COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "Building goddns ${VERSION} (${COMMIT})"

go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" -o goddns ./cmd/goddns

echo "Build completed. Run './goddns -v' to check version."
```

使用方法：
```bash
chmod +x build.sh
./build.sh v1.0.0    # 构建指定版本
./build.sh           # 构建开发版本
```

### 运行
```bash
./goddns run -f config.json
# -i 可选，忽略缓存强制更新
```

### 显示版本
```bash
./goddns -v
```

版本输出示例：
```
goddns v1.0.0
commit: a1b2c3d4e5f67890
built: 2024-01-15T10:30:45Z
```

## 配置示例
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

## 字段说明
- **provider**：DNS 服务商，目前仅支持 cloudflare
- **get_ip.interface**：本地网卡名，优先使用
- **get_ip.urls/get_ip.url**：外部检测 IPv6 的 API 列表
- **work_dir**：缓存文件目录
- **log_output**：日志输出路径或 shell
- **provider_options.api_token**：Cloudflare API Token
- **provider_options.zone_id**：Cloudflare 区域 ID
- **provider_options.domain.zone/record**：主域名/子域名
- **proxy**：可选，支持 http/https/socks5

## 自动运行

### systemd 定时
创建 `/etc/systemd/system/goddns.service`：
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
```bash
crontab -e
# 添加：
*/5 * * * * /usr/local/bin/goddns run -f /etc/goddns/config.json >> /var/log/goddns-cron.log 2>&1
```

## 目录结构
- `cmd/goddns/`：主程序入口
- `internal/config/`：配置与缓存
- `internal/log/`：日志
- `internal/platform/ifaddr/`：平台相关网络工具
- `internal/provider/cloudflare/`：Cloudflare API

## 构建参数说明
### ldflags 参数详解
- `-X main.version`：设置程序版本号
- `-X main.commit`：设置 Git 提交哈希
- `-X main.buildDate`：设置构建时间（UTC格式）

### 环境变量支持
可以在构建时使用环境变量：
```bash
export VERSION=v1.0.0
export COMMIT=$(git rev-parse HEAD)
export BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" -o goddns ./cmd/goddns
```

## 许可证
请见 LICENSE 文件。

如需更多帮助或反馈建议，请提交 issue。