#!/bin/bash

# Gen-Code Service Test Script
# 测试代码生成服务的功能

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "🚀 Gen-Code Service Test Script"
echo "================================"
echo ""

# 1. Health Check
echo "1️⃣  Testing health check..."
curl -s "${BASE_URL}/health" | jq .
echo "✅ Health check passed"
echo ""

# 2. Create a task
echo "2️⃣  Creating a new code generation task..."
RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/generate" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "创建一个简单的Go语言Hello World程序，包含main.go和README.md文件",
    "repo_name": "test-hello-go",
    "model": "deepseek"
  }')

echo "$RESPONSE" | jq .

TASK_ID=$(echo "$RESPONSE" | jq -r '.task_id')
echo "✅ Task created with ID: $TASK_ID"
echo ""

# 3. Get task status
echo "3️⃣  Checking task status..."
sleep 2
curl -s "${BASE_URL}/api/v1/task/${TASK_ID}" | jq .
echo ""

# 4. Subscribe to SSE
echo "4️⃣  Subscribing to task status updates (SSE)..."
echo "Press Ctrl+C to stop watching"
echo ""

curl -N -s "${BASE_URL}/api/v1/status/${TASK_ID}" | while IFS= read -r line; do
  if [[ $line == data:* ]]; then
    echo "$line" | sed 's/^data: //' | jq -c .
  fi
done

echo ""
echo "✅ Test completed!"
