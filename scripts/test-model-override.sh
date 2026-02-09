#!/bin/bash

# Test script for dynamic model switching

echo "Starting CCG service..."
./ccg start

echo "Waiting for service to start..."
sleep 2

echo "Testing model override with /model/ path..."

# Test 1: Override to use OpenAI GPT-5
echo -e "\n1. Testing OpenAI GPT-5 override:"
curl -X POST http://localhost:3456/model/openai:gpt-5/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-sonnet",
    "messages": [{"role": "user", "content": "Say hello"}],
    "stream": false
  }' -v 2>&1 | grep -E "(< HTTP|selected_model)"

# Test 2: Override to use a different provider
echo -e "\n2. Testing provider switch override:"
curl -X POST http://localhost:3456/model/anthropic:claude-3-opus/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5",
    "messages": [{"role": "user", "content": "What is 2+2?"}],
    "stream": false
  }' -v 2>&1 | grep -E "(< HTTP|selected_model)"

# Test 3: Normal request without override
echo -e "\n3. Testing normal request (no override):"
curl -X POST http://localhost:3456/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-sonnet",
    "messages": [{"role": "user", "content": "What is the capital of France?"}],
    "stream": false
  }' -v 2>&1 | grep -E "(< HTTP|selected_model)"

echo -e "\nTests complete!"