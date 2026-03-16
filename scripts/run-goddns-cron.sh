#!/bin/bash
# goddns crontab 运行脚本
# 
# 使用方法：
# 1. 复制此文件到 /etc/goddns/run-goddns.sh
# 2. 修改环境变量配置
# 3. 设置执行权限：chmod +x /etc/goddns/run-goddns.sh
# 4. 添加到 crontab：*/5 * * * * /etc/goddns/run-goddns.sh

# 设置 PATH 环境变量（cron 环境中可能需要）
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# 配置路径
GODDNS_BIN="/usr/local/bin/goddns"
GODDNS_CONFIG="/etc/goddns/config.json"
GODDNS_ENV="/etc/goddns/goddns.env"
LOG_FILE="/var/log/goddns-cron.log"

# 检查 goddns 是否存在
if [ ! -x "$GODDNS_BIN" ]; then
    echo "$(date): Error - goddns binary not found at $GODDNS_BIN" >> "$LOG_FILE"
    exit 1
fi

# 检查配置文件是否存在
if [ ! -f "$GODDNS_CONFIG" ]; then
    echo "$(date): Error - config file not found at $GODDNS_CONFIG" >> "$LOG_FILE"
    exit 1
fi

# 加载环境变量文件（如果存在）
if [ -f "$GODDNS_ENV" ]; then
    # set -a 使后续 source 的变量自动导出为环境变量
    set -a
    source "$GODDNS_ENV"
    set +a
fi

# 运行 goddns
# -f 指定配置文件
# -i 可选：忽略缓存强制更新
"$GODDNS_BIN" run -f "$GODDNS_CONFIG"

# 记录退出状态
EXIT_CODE=$?
if [ $EXIT_CODE -eq 0 ]; then
    echo "$(date): goddns executed successfully" >> "$LOG_FILE"
else
    echo "$(date): goddns failed with exit code $EXIT_CODE" >> "$LOG_FILE"
fi

exit $EXIT_CODE
