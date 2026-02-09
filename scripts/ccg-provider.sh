#!/bin/bash
# Unified provider configuration script for CCG

set_provider() {
    local provider="$1"
    
    # Clear existing Anthropic environment variables
    unset ANTHROPIC_BASE_URL
    unset ANTHROPIC_AUTH_TOKEN
    unset ANTHROPIC_MODEL
    unset ANTHROPIC_SMALL_FAST_MODEL
    unset ANTHROPIC_API_KEY
    
    case "$provider" in
        qwen|q)
            export ANTHROPIC_BASE_URL="https://dashscope-intl.aliyuncs.com/api/v2/apps/claude-code-proxy"
            export ANTHROPIC_AUTH_TOKEN="${QWEN_API_KEY:-sk-bleh}"
            echo "🟢 Switched to Qwen3-Coder"
            ;;
        deepseek|d)
            export ANTHROPIC_BASE_URL="https://api.deepseek.com/anthropic"
            export ANTHROPIC_AUTH_TOKEN="${DEEPSEEK_API_KEY:-your-api-key}"
            export ANTHROPIC_MODEL="deepseek-chat"
            export ANTHROPIC_SMALL_FAST_MODEL="deepseek-chat"
            echo "🟢 Switched to DeepSeek"
            ;;
        claude|c)
            export ANTHROPIC_API_KEY="${CLAUDE_API_KEY:-your-claude-key}"
            echo "🟢 Switched to Claude (Anthropic)"
            ;;
        *)
            echo "Usage: source ccg-provider.sh [provider]"
            echo "Providers: qwen|q, deepseek|d, claude|c"
            return 1
            ;;
    esac
    
    echo "Usage: claude 'your prompt'"
}

# If called with argument, set provider
if [ -n "$1" ]; then
    set_provider "$1"
fi