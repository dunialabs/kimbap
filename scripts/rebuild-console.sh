#!/bin/bash
# Rebuild kimbap with latest webui and restart server
set -e
cd /Users/hyeonwoo/Desktop/projects/kimbap
go build -o bin/kimbap ./cmd/kimbap
pkill -f "bin/kimbap serve" 2>/dev/null || true
sleep 0.5
./bin/kimbap serve --console --port 8080 > /tmp/kimbap-server.log 2>&1 &
sleep 1.5
echo "Console ready at http://localhost:8080/console"
