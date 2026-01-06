#!/bin/bash

# Build and restart server
go build -o bin/quic-server ./cmd/server
pkill -9 -f "quic-server"
sleep 1
./bin/quic-server -c config/server.yaml > /tmp/qd6.log 2>&1 &
sleep 5

# Test API
curl -X POST http://localhost:8475/api/file/upload/init \
  -H 'Content-Type: application/json' \
  -d '{"filename":"test.txt","file_size":1024}' \
  --max-time 5

# Show logs
echo ""
echo "=== RECENT DEBUG LOGS ==="
grep "processTask\|InitUpload\|SubmitTask" /tmp/qd6.log | tail -50
