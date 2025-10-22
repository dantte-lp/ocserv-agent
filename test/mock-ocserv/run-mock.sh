#!/bin/bash
# Mock ocserv для тестирования

echo "🔧 Starting mock ocserv..."

# Создать fake socket
mkdir -p /var/run
touch /var/run/occtl.socket

# Имитация occtl команд
while true; do
    if [ -p /tmp/occtl-pipe ]; then
        read cmd < /tmp/occtl-pipe
        case $cmd in
            "show users")
                echo '{"users": []}'
                ;;
            "show status")
                echo '{"status": "running", "uptime": 12345}'
                ;;
            *)
                echo '{"error": "unknown command"}'
                ;;
        esac
    fi
    sleep 1
done
