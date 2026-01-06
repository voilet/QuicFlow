#!/bin/bash

# Test complete upload flow

# Prepare the test data first
CHUNK1="Hello, this is chunk 1 of the test file. "
CHUNK1_LEN=${#CHUNK1}
CHUNK2="This is chunk 2."
CHUNK2_LEN=${#CHUNK2}
TOTAL=$((CHUNK1_LEN + CHUNK2_LEN))

echo "=== Test data ==="
echo "Chunk 1: $CHUNK1_LEN bytes"
echo "Chunk 2: $CHUNK2_LEN bytes"
echo "Total: $TOTAL bytes"
echo ""

echo "=== Step 1: Initialize upload (file_size=$TOTAL) ==="
RESP=$(curl -s -X POST http://localhost:8475/api/file/upload/init \
  -H 'Content-Type: application/json' \
  -d "{\"filename\":\"test-upload.txt\",\"file_size\":$TOTAL}")

echo "$RESP" | python3 -m json.tool 2>/dev/null || echo "$RESP"

# Extract task_id
TASK_ID=$(echo "$RESP" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['task_id'])" 2>/dev/null)

if [ -z "$TASK_ID" ]; then
  echo "Failed to get task_id"
  exit 1
fi

echo "Task ID: $TASK_ID"
echo ""

# Wait a bit for the worker to start
sleep 2

echo "=== Step 2: Upload chunk 1 (offset 0, $CHUNK1_LEN bytes) ==="
curl -s -X POST "http://localhost:8475/api/file/upload/chunk?task_id=$TASK_ID&offset=0&sequence=0" \
  --data-binary "$CHUNK1" \
  | python3 -m json.tool 2>/dev/null || echo "Upload chunk 1 failed"

sleep 1

echo ""
echo "=== Step 3: Upload chunk 2 (offset $CHUNK1_LEN, $CHUNK2_LEN bytes) ==="
curl -s -X POST "http://localhost:8475/api/file/upload/chunk?task_id=$TASK_ID&offset=$CHUNK1_LEN&sequence=1" \
  --data-binary "$CHUNK2" \
  | python3 -m json.tool 2>/dev/null || echo "Upload chunk 2 failed"

sleep 1

echo ""
echo "=== Step 4: Complete upload ==="
curl -s -X POST http://localhost:8475/api/file/upload/complete \
  -H 'Content-Type: application/json' \
  -d "{\"task_id\":\"$TASK_ID\",\"checksum\":\"\"}" \
  | python3 -m json.tool 2>/dev/null || echo "Complete upload failed"

echo ""
echo ""
echo "=== Server debug logs ==="
grep "processUpload\|UploadChunk\|CompleteUpload\|processTask\|doneChan" /tmp/qd-fixed2.log | tail -30
