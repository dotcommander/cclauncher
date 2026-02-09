#!/bin/bash

# CCG Global Installation Script

set -e

echo "🚀 Installing CCG as a global tool..."

# Determine installation directory
INSTALL_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.config/ccg"

# Create directories if they don't exist
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"

# Build CCG
echo "📦 Building CCG..."
go build -o ccg-binary cmd/ccg/main.go

# Install binary
echo "📂 Installing to $INSTALL_DIR/ccg..."
cp ccg-binary "$INSTALL_DIR/ccg"
chmod +x "$INSTALL_DIR/ccg"

# Create global config file with API key placeholders
cat > "$CONFIG_DIR/providers.env" << 'EOF'
# CCG Provider Configuration
# Set your API keys here or in your shell environment

# Qwen (default provider)
# Uses default key if not set
# QWEN_API_KEY=sk-bleh

# DeepSeek
# Get your key from https://platform.deepseek.com/
# DEEPSEEK_API_KEY=sk-your-deepseek-key

# Claude (Anthropic)
# Get your key from https://console.anthropic.com/
# CLAUDE_API_KEY=sk-ant-your-claude-key
EOF

# Create shell integration script
cat > "$CONFIG_DIR/ccg-init.sh" << 'EOF'
# CCG Shell Integration
# Source this in your .bashrc/.zshrc: source ~/.config/ccg/ccg-init.sh

# Load provider configuration
if [ -f "$HOME/.config/ccg/providers.env" ]; then
    set -a
    source "$HOME/.config/ccg/providers.env"
    set +a
fi

# Convenience aliases
alias cq='ccg -p qwen'      # Quick Qwen
alias cds='ccg -p deepseek'  # Quick DeepSeek (renamed to avoid conflict with cd)
alias ccl='ccg -p claude'    # Quick Claude

# Show current provider function
ccg-status() {
    echo "CCG Providers:"
    echo "  cq  - Qwen3-Coder (default)"
    echo "  cds - DeepSeek"
    echo "  ccl - Claude"
    echo ""
    echo "Usage: ccg 'your prompt' or cq 'your prompt'"
}
EOF

# Add to PATH if not already there
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "⚠️  Add $INSTALL_DIR to your PATH:"
    echo ""
    
    # Detect shell
    if [ -n "$ZSH_VERSION" ]; then
        SHELL_RC="$HOME/.zshrc"
        echo "Add to your ~/.zshrc:"
    elif [ -n "$BASH_VERSION" ]; then
        SHELL_RC="$HOME/.bashrc"
        echo "Add to your ~/.bashrc:"
    else
        SHELL_RC="your shell config"
        echo "Add to your shell config:"
    fi
    
    echo ""
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo "  source \$HOME/.config/ccg/ccg-init.sh"
    echo ""
fi

echo "✅ CCG installed successfully!"
echo ""
echo "📋 Next steps:"
echo "1. Add to your shell config (if not already done):"
echo "   export PATH=\"\$HOME/.local/bin:\$PATH\""
echo "   source \$HOME/.config/ccg/ccg-init.sh"
echo ""
echo "2. Edit ~/.config/ccg/providers.env to add your API keys"
echo ""
echo "3. Reload your shell or run:"
echo "   source ~/.config/ccg/ccg-init.sh"
echo ""
echo "4. Use CCG from anywhere:"
echo "   ccg 'Write hello world'"
echo "   cq 'Write a Go function'    # Qwen"
echo "   cds 'Explain recursion'     # DeepSeek"
echo "   ccl 'Analyze this code'     # Claude"