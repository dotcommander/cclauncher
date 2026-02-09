#!/bin/bash

# Test json_schema transformation with ccg

echo "Testing json_schema transformation..."

# Start ccg if not running
./ccg status > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "Starting ccg..."
    ./ccg start
    sleep 2
fi

# Test with a compatible model (should use json_schema)
echo -e "\n1. Testing with gpt-4o (compatible model):"
curl -s -X POST http://localhost:3456/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-key" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "extract user profile from: John Doe, john@example.com, 30 years old, likes coding"}
    ]
  }' | jq '.response_format' 2>/dev/null || echo "Request sent (no direct response inspection)"

# Test with an incompatible model (should fall back to json_object)
echo -e "\n2. Testing with gpt-3.5-turbo (incompatible model):"
curl -s -X POST http://localhost:3456/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "extract user information"}
    ]
  }' | jq '.response_format' 2>/dev/null || echo "Request sent (no direct response inspection)"

echo -e "\nCheck the ccg logs for transformation details:"
echo "tail -f ~/.claude-code-router/ccg.log"