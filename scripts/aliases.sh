#!/bin/bash
# CCG Aliases - Add these to your ~/.bashrc or ~/.zshrc

# Quick aliases for CCG
alias cq='./ccg -p qwen'      # Quick Qwen
alias cd='./ccg -p deepseek'  # Quick DeepSeek
alias cc='./ccg -p claude'    # Quick Claude

# Default ccg uses Qwen
alias ccg='./ccg'

echo "CCG aliases configured:"
echo "  cq - Use Qwen3-Coder"
echo "  cd - Use DeepSeek"
echo "  cc - Use Claude"
echo ""
echo "Usage: cq 'Write hello world'"