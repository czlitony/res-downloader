#!/bin/bash
# 下载各平台 ffmpeg 二进制文件
# 用法: ./scripts/download-ffmpeg.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FFMPEG_DIR="$SCRIPT_DIR/../ffmpeg-bin"

mkdir -p "$FFMPEG_DIR"
echo "FFmpeg 二进制文件将下载到: $FFMPEG_DIR"

# Windows amd64
WIN_AMD64_URL="https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
WIN_AMD64_DIR="$FFMPEG_DIR/windows-amd64"

echo -e "\n[1/4] 下载 Windows amd64..."
if [ ! -f "$WIN_AMD64_DIR/ffmpeg.exe" ]; then
    mkdir -p "$WIN_AMD64_DIR"
    curl -L "$WIN_AMD64_URL" -o "$FFMPEG_DIR/ffmpeg-win.zip"
    unzip -q "$FFMPEG_DIR/ffmpeg-win.zip" -d "$FFMPEG_DIR"
    EXTRACTED=$(find "$FFMPEG_DIR" -maxdepth 1 -type d -name "ffmpeg-*-essentials*" | head -1)
    if [ -n "$EXTRACTED" ]; then
        cp "$EXTRACTED/bin/ffmpeg.exe" "$WIN_AMD64_DIR/"
        rm -rf "$EXTRACTED"
    fi
    rm -f "$FFMPEG_DIR/ffmpeg-win.zip"
    echo "  完成: $WIN_AMD64_DIR/ffmpeg.exe"
else
    echo "  已存在，跳过"
fi

# macOS universal
MAC_URL="https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip"
MAC_DIR="$FFMPEG_DIR/darwin-universal"

echo -e "\n[2/4] 下载 macOS universal..."
if [ ! -f "$MAC_DIR/ffmpeg" ]; then
    mkdir -p "$MAC_DIR"
    curl -L "$MAC_URL" -o "$FFMPEG_DIR/ffmpeg-mac.zip" || {
        echo "  下载失败，请手动从 https://evermeet.cx/ffmpeg/ 下载"
    }
    if [ -f "$FFMPEG_DIR/ffmpeg-mac.zip" ]; then
        unzip -q "$FFMPEG_DIR/ffmpeg-mac.zip" -d "$MAC_DIR"
        rm -f "$FFMPEG_DIR/ffmpeg-mac.zip"
        chmod +x "$MAC_DIR/ffmpeg"
        echo "  完成: $MAC_DIR/ffmpeg"
    fi
else
    echo "  已存在，跳过"
fi

# Linux amd64
LINUX_AMD64_URL="https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz"
LINUX_AMD64_DIR="$FFMPEG_DIR/linux-amd64"

echo -e "\n[3/4] 下载 Linux amd64..."
if [ ! -f "$LINUX_AMD64_DIR/ffmpeg" ]; then
    mkdir -p "$LINUX_AMD64_DIR"
    curl -L "$LINUX_AMD64_URL" -o "$FFMPEG_DIR/ffmpeg-linux-amd64.tar.xz"
    tar -xf "$FFMPEG_DIR/ffmpeg-linux-amd64.tar.xz" -C "$FFMPEG_DIR"
    EXTRACTED=$(find "$FFMPEG_DIR" -maxdepth 1 -type d -name "ffmpeg-*-amd64-static" | head -1)
    if [ -n "$EXTRACTED" ]; then
        cp "$EXTRACTED/ffmpeg" "$LINUX_AMD64_DIR/"
        rm -rf "$EXTRACTED"
    fi
    rm -f "$FFMPEG_DIR/ffmpeg-linux-amd64.tar.xz"
    chmod +x "$LINUX_AMD64_DIR/ffmpeg"
    echo "  完成: $LINUX_AMD64_DIR/ffmpeg"
else
    echo "  已存在，跳过"
fi

# Linux arm64
LINUX_ARM64_URL="https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-arm64-static.tar.xz"
LINUX_ARM64_DIR="$FFMPEG_DIR/linux-arm64"

echo -e "\n[4/4] 下载 Linux arm64..."
if [ ! -f "$LINUX_ARM64_DIR/ffmpeg" ]; then
    mkdir -p "$LINUX_ARM64_DIR"
    curl -L "$LINUX_ARM64_URL" -o "$FFMPEG_DIR/ffmpeg-linux-arm64.tar.xz"
    tar -xf "$FFMPEG_DIR/ffmpeg-linux-arm64.tar.xz" -C "$FFMPEG_DIR"
    EXTRACTED=$(find "$FFMPEG_DIR" -maxdepth 1 -type d -name "ffmpeg-*-arm64-static" | head -1)
    if [ -n "$EXTRACTED" ]; then
        cp "$EXTRACTED/ffmpeg" "$LINUX_ARM64_DIR/"
        rm -rf "$EXTRACTED"
    fi
    rm -f "$FFMPEG_DIR/ffmpeg-linux-arm64.tar.xz"
    chmod +x "$LINUX_ARM64_DIR/ffmpeg"
    echo "  完成: $LINUX_ARM64_DIR/ffmpeg"
else
    echo "  已存在，跳过"
fi

echo -e "\n下载完成！目录结构:"
cat << 'EOF'
ffmpeg-bin/
├── windows-amd64/
│   └── ffmpeg.exe
├── darwin-universal/
│   └── ffmpeg
├── linux-amd64/
│   └── ffmpeg
└── linux-arm64/
    └── ffmpeg
EOF

echo -e "\n打包时将对应平台的 ffmpeg 复制到输出目录即可"
