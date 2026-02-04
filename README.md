

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

## 快速开始

### 构建
```bash
go build -o goddns ./cmd/goddns
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

### 字段说明
- `provider`：DNS 服务商，目前仅支持 cloudflare
- `get_ip.interface`：本地网卡名，优先使用
- `get_ip.urls`/`get_ip.url`：外部检测 IPv6 的 API 列表
- `work_dir`：缓存文件目录
- `log_output`：日志输出路径或 shell
- `provider_options.api_token`：Cloudflare API Token
- `provider_options.zone_id`：Cloudflare 区域 ID
- `provider_options.domain.zone`/`record`：主域名/子域名
- `proxy`：可选，支持 http/https/socks5

## 自动运行

### systemd 定时
1. `/etc/systemd/system/goddns.service`
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
2. `/etc/systemd/system/goddns.timer`
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
3. 启用
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

## 许可证
请见 LICENSE 文件。

---

如需更多帮助或反馈建议，请提交 issue。
