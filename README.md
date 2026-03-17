# goddns - 动态 DNS 客户端

[![Go Version](https://img.shields.io/badge/go-1.21-blue.svg)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

**goddns** 是一个用 Go 编写的轻量级动态 DNS (DDNS) 客户端，支持多域名、多服务商、IPv6，具备跨平台能力和丰富的日志输出。

---

## 目录

- [特性](#特性)
- [快速开始](#快速开始)
- [配置指南](#配置指南)
- [环境变量](#环境变量)
- [自动运行](#自动运行)
- [平台支持](#平台支持)
- [项目结构](#项目结构)

---

## 特性

| 功能 | 说明 |
|------|------|
| 🌐 **多域名支持** | 一条配置可更新多个 DNS 记录 |
| ☁️ **多服务商** | 支持 Cloudflare、阿里云 DNS |
| 🔒 **安全性** | 强制使用环境变量，禁止明文密钥 |
| 🚀 **并发更新** | 多个域名并行更新，提高效率 |
| 📦 **IPv6 支持** | 原生支持 IPv6，多平台接口获取 |
| 🔄 **IP 缓存** | 避免重复 API 调用 |
| 🎨 **彩色日志** | 终端下日志分级彩色显示，支持文件输出 |
| 🌍 **代理支持** | HTTP(S)/SOCKS5 代理，记录级控制 |

---

## 快速开始

### 1. 构建

```bash
# 基础构建
go build -o goddns ./cmd/goddns

# 带版本信息构建
go build -ldflags "-X main.version=v2.0.0" -o goddns ./cmd/goddns

# 使用构建脚本
chmod +x build.sh
./build.sh v2.0.0
```

### 2. 配置

创建 `config.json`：

```json
{
    "general": {
        "get_ip": {
            "interface": "eth0",
            "urls": ["https://ipv6.icanhazip.com"]
        },
        "work_dir": "/var/lib/goddns",
        "log_output": "shell"
    },
    "records": [
        {
            "provider": "cloudflare",
            "zone": "example.com",
            "record": "dev",
            "cloudflare": {
                "api_token": "${CLOUDFLARE_API_TOKEN}"
            }
        }
    ]
}
```

### 3. 设置环境变量

```bash
export CLOUDFLARE_API_TOKEN="your_api_token_here"
```

### 4. 运行

```bash
# 运行
./goddns run -f config.json

# 忽略缓存强制更新
./goddns run -f config.json -i

# 查看版本
./goddns version
```

---

## 配置指南

### 安全性要求

> ⚠️ **重要**：出于安全考虑，goddns **禁止在配置文件中明文存储密钥信息**。所有敏感信息必须使用环境变量引用。

❌ **错误示例**（会被拒绝执行）：
```json
{
    "cloudflare": {
        "api_token": "your_actual_token_here"
    }
}
```

✅ **正确示例**：
```json
{
    "cloudflare": {
        "api_token": "${CLOUDFLARE_API_TOKEN}",
        "zone_id": "${CLOUDFLARE_ZONE_ID:-}"
    }
}
```

---

### 完整配置示例

```json
{
    "general": {
        "get_ip": {
            "interface": "eth0",
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
            "aliyun": {
                "access_key_id": "${ALIYUN_ACCESS_KEY_ID}",
                "access_key_secret": "${ALIYUN_ACCESS_KEY_SECRET}"
            }
        }
    ]
}
```

---

### 配置字段说明

#### general（全局配置）

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `get_ip.interface` | string | 本地网卡名（优先使用） | `eth0` |
| `get_ip.urls` | []string | 外部 IP 检测 API 列表（降级） | `["https://ipv6.icanhazip.com"]` |
| `work_dir` | string | 缓存文件目录 | `/var/lib/goddns` |
| `log_output` | string | 日志输出，`shell` 表示终端 | `shell` 或文件路径 |
| `proxy` | string | 全局代理（可选） | `socks5://127.0.0.1:1080` |

#### records（记录数组）

| 字段 | 类型 | 说明 | 必填 |
|------|------|------|------|
| `provider` | string | 服务商：`cloudflare` 或 `aliyun` | ✅ |
| `zone` | string | 主域名 | ✅ |
| `record` | string | 子域名/记录名（`@` 表示根域） | ✅ |
| `ttl` | int | DNS 记录 TTL（秒） | ❌ |
| `proxied` | bool | Cloudflare 代理模式 | ❌ |
| `use_proxy` | bool | 是否使用全局代理 | ❌ |

#### Cloudflare 配置

| 字段 | 类型 | 说明 |
|------|------|------|
| `cloudflare.api_token` | string | API Token（环境变量引用） |
| `cloudflare.zone_id` | string | Zone ID（可选，留空自动获取） |
| `cloudflare.ttl` | int | TTL（可选，覆盖记录级） |
| `cloudflare.proxied` | bool | 代理模式（可选，覆盖记录级） |

#### 阿里云配置

| 字段 | 类型 | 说明 |
|------|------|------|
| `aliyun.access_key_id` | string | AccessKey ID（环境变量引用） |
| `aliyun.access_key_secret` | string | AccessKey Secret（环境变量引用） |
| `aliyun.ttl` | int | TTL（可选，覆盖记录级） |

---

### 服务商对比

| 特性 | Cloudflare | 阿里云 |
|------|------------|--------|
| IPv6 支持 | ✅ | ✅ |
| 代理支持 | ✅ | ❌ |
| 自动获取 ZoneID | ✅ | ✅ |
| API 认证 | API Token | AccessKey |

---

## 环境变量

### 支持的环境变量

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `CLOUDFLARE_API_TOKEN` | Cloudflare API Token | `your_token_here` |
| `CLOUDFLARE_ZONE_ID` | Cloudflare Zone ID（可选） | `abc123xyz` |
| `ALIYUN_ACCESS_KEY_ID` | 阿里云 AccessKey ID | `LTAI1234567890` |
| `ALIYUN_ACCESS_KEY_SECRET` | 阿里云 AccessKey Secret | `your_secret_here` |

### 使用方式

```bash
# 设置环境变量
export CLOUDFLARE_API_TOKEN="your_token_here"
export ALIYUN_ACCESS_KEY_ID="LTAI1234567890"
export ALIYUN_ACCESS_KEY_SECRET="your_secret_here"

# 运行
./goddns run -f config.json
```

### 环境变量默认值

支持 `${VAR:-default}` 语法：

```json
{
    "cloudflare": {
        "zone_id": "${CLOUDFLARE_ZONE_ID:-}"
    }
}
```

- `${VAR}` - 使用环境变量值
- `${VAR:-default}` - 未设置或为空时使用默认值
- `${VAR-default}` - 未设置时使用默认值

---

## 自动运行

### systemd 定时（推荐）

#### 1. 创建环境变量文件

```bash
# /etc/goddns/goddns.env
CLOUDFLARE_API_TOKEN="your_token_here"
ALIYUN_ACCESS_KEY_ID="LTAI1234567890"
ALIYUN_ACCESS_KEY_SECRET="your_secret_here"

# 设置权限（仅 root 可读）
chmod 600 /etc/goddns/goddns.env
```

#### 2. 创建 service 文件

```ini
# /etc/systemd/system/goddns.service
[Unit]
Description=Dynamic DNS client
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/goddns run -f /etc/goddns/config.json
EnvironmentFile=/etc/goddns/goddns.env
```

#### 3. 创建 timer 文件

```ini
# /etc/systemd/system/goddns.timer
[Unit]
Description=Run goddns every 5 minutes

[Timer]
OnBootSec=5min
OnUnitActiveSec=5min
Persistent=true

[Install]
WantedBy=timers.target
```

#### 4. 启用

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now goddns.timer
```

---

### cron 定时

#### 方式一：使用脚本（推荐）

```bash
#!/bin/bash
# /etc/goddns/run-goddns.sh

set -a
source /etc/goddns/goddns.env
set +a

/usr/local/bin/goddns run -f /etc/goddns/config.json
```

```bash
chmod +x /etc/goddns/run-goddns.sh
crontab -e
# 添加：*/5 * * * * /etc/goddns/run-goddns.sh >> /var/log/goddns-cron.log 2>&1
```

#### 方式二：直接在 crontab 中设置

```bash
crontab -e
# 添加：
CLOUDFLARE_API_TOKEN="your_token_here"
*/5 * * * * /usr/local/bin/goddns run -f /etc/goddns/config.json >> /var/log/goddns-cron.log 2>&1
```

---

## 平台支持

| 平台 | 状态 | 说明 |
|------|------|------|
| Linux | ✅ | 使用 netlink 接口 |
| FreeBSD | ✅ | 使用 ioctl 接口 |
| OpenBSD | ✅ | 使用 ioctl 接口 |
| macOS | ⚠️ | 暂无支持，欢迎提交 PR |

---

## 项目结构

```
goddns/
├── cmd/goddns/           # 主程序入口
│   ├── main.go
│   └── cmd.go
├── internal/
│   ├── config/           # 配置管理
│   │   ├── config.go
│   │   └── config_test.go
│   ├── log/              # 日志系统
│   │   └── log.go
│   ├── platform/ifaddr/  # 平台相关网络工具
│   │   ├── linux_netlink.go
│   │   ├── freebsd_ioctl.go
│   │   ├── openbsd_ioctl.go
│   │   ├── shared.go
│   │   ├── shared_test.go
│   │   └── util.go
│   └── provider/         # DNS 服务商实现
│       ├── provider.go
│       ├── factory/
│       ├── cloudflare/
│       └── aliyun/
├── config.example.json   # 配置示例
├── .env.example          # 环境变量示例
├── build.sh              # 构建脚本
└── README.md
```

---

## 常见问题

### 1. 如何获取 Cloudflare API Token？

访问 [Cloudflare Dashboard](https://dash.cloudflare.com/profile/api-tokens)，创建具有 `Zone:DNS:Edit` 权限的 Token。

### 2. 如何获取阿里云 AccessKey？

访问 [阿里云 RAM 控制台](https://ram.console.aliyun.com/manage/ak) 创建 AccessKey。

### 3. Zone ID 如何获取？

- **Cloudflare**：Dashboard → Overview 右侧，或留空自动获取
- **阿里云**：不需要

### 4. 代理如何配置？

```json
{
    "general": {
        "proxy": "socks5://127.0.0.1:1080"
    },
    "records": [
        {
            "use_proxy": true
        }
    ]
}
```

> 注意：仅 Cloudflare 支持代理，阿里云不支持。

---

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件。

---

## 贡献

欢迎提交 Issue 和 Pull Request！
