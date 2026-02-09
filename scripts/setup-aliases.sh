#!/bin/bash
# Add these aliases to your ~/.bashrc or ~/.zshrc

cat << 'EOF'

# Claude Code Provider Shortcuts
alias cq='source ~/go/src/ccg/use-qwen.sh && claude'     # Quick Qwen
alias cd='source ~/go/src/ccg/use-deepseek.sh && claude' # Quick DeepSeek  
alias cc='source ~/go/src/ccg/use-claude.sh && claude'   # Quick Claude

# Just switch provider (without running claude)
alias use-qwen='source ~/go/src/ccg/use-qwen.sh'
alias use-deepseek='source ~/go/src/ccg/use-deepseek.sh'
alias use-claude='source ~/go/src/ccg/use-claude.sh'

# Show current provider
alias which-llm='echo "Current: ${ANTHROPIC_BASE_URL:-Claude (default)}"'

EOF

echo ""
echo "To install these aliases, add them to your shell config:"
echo ""
echo "For bash:"
echo "  cat setup-aliases.sh >> ~/.bashrc && source ~/.bashrc"
echo ""
echo "For zsh:"
echo "  cat setup-aliases.sh >> ~/.zshrc && source ~/.zshrc"