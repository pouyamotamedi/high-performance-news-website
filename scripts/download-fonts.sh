#!/bin/bash
# Download self-hosted fonts for the news website
# This eliminates Google Fonts dependency for better performance

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
FONTS_DIR="$PROJECT_ROOT/web/static/fonts"

echo "📥 Downloading self-hosted fonts..."

# Create fonts directory
mkdir -p "$FONTS_DIR"

# Download Inter font (variable font for Latin)
echo "Downloading Inter font..."
curl -L -o "$FONTS_DIR/inter-var-latin.woff2" \
    "https://fonts.gstatic.com/s/inter/v13/UcCO3FwrK3iLTeHuS_fvQtMwCp50KnMw2boKoduKmMEVuLyfAZ9hjp-Ek-_EeA.woff2"

# Download Inter static weights as fallback
curl -L -o "$FONTS_DIR/inter-regular.woff2" \
    "https://fonts.gstatic.com/s/inter/v13/UcCO3FwrK3iLTeHuS_fvQtMwCp50KnMw2boKoduKmMEVuGKYAZ9hjp-Ek-_EeA.woff2"

curl -L -o "$FONTS_DIR/inter-medium.woff2" \
    "https://fonts.gstatic.com/s/inter/v13/UcCO3FwrK3iLTeHuS_fvQtMwCp50KnMw2boKoduKmMEVuI6fAZ9hjp-Ek-_EeA.woff2"

curl -L -o "$FONTS_DIR/inter-semibold.woff2" \
    "https://fonts.gstatic.com/s/inter/v13/UcCO3FwrK3iLTeHuS_fvQtMwCp50KnMw2boKoduKmMEVuGKYAZ9hjp-Ek-_EeA.woff2"

curl -L -o "$FONTS_DIR/inter-bold.woff2" \
    "https://fonts.gstatic.com/s/inter/v13/UcCO3FwrK3iLTeHuS_fvQtMwCp50KnMw2boKoduKmMEVuFuYAZ9hjp-Ek-_EeA.woff2"

curl -L -o "$FONTS_DIR/inter-extrabold.woff2" \
    "https://fonts.gstatic.com/s/inter/v13/UcCO3FwrK3iLTeHuS_fvQtMwCp50KnMw2boKoduKmMEVuDyYAZ9hjp-Ek-_EeA.woff2"

# Download Vazir font for Persian/Farsi
echo "Downloading Vazir font..."
curl -L -o "$FONTS_DIR/vazir-regular.woff2" \
    "https://cdn.jsdelivr.net/gh/AlirezaShamsoshoara/Vazir-Font@v30.1.0/dist/Vazir-Regular.woff2" || \
curl -L -o "$FONTS_DIR/vazir-regular.woff2" \
    "https://cdn.rawgit.com/AlirezaShamsoshoara/Vazir-Font/v30.1.0/dist/Vazir-Regular.woff2" || \
echo "⚠️  Could not download Vazir font. Please download manually from https://github.com/rastikerdar/vazir-font"

curl -L -o "$FONTS_DIR/vazir-medium.woff2" \
    "https://cdn.jsdelivr.net/gh/AlirezaShamsoshoara/Vazir-Font@v30.1.0/dist/Vazir-Medium.woff2" 2>/dev/null || true

curl -L -o "$FONTS_DIR/vazir-bold.woff2" \
    "https://cdn.jsdelivr.net/gh/AlirezaShamsoshoara/Vazir-Font@v30.1.0/dist/Vazir-Bold.woff2" 2>/dev/null || true

# Download Amiri font for Arabic
echo "Downloading Amiri font..."
curl -L -o "$FONTS_DIR/amiri-regular.woff2" \
    "https://fonts.gstatic.com/s/amiri/v27/J7aRnpd8CGxBHqUpvrIw74NL.woff2"

curl -L -o "$FONTS_DIR/amiri-bold.woff2" \
    "https://fonts.gstatic.com/s/amiri/v27/J7acnpd8CGxBHp2VkZY4xJ9CGyAa.woff2"

echo ""
echo "✅ Fonts downloaded successfully!"
echo ""
echo "📁 Fonts directory: $FONTS_DIR"
echo ""
echo "📊 Downloaded files:"
ls -la "$FONTS_DIR"/*.woff2 2>/dev/null || echo "No woff2 files found"
echo ""
echo "💡 Note: If any fonts failed to download, you may need to download them manually."
echo "   - Inter: https://fonts.google.com/specimen/Inter"
echo "   - Vazir: https://github.com/rastikerdar/vazir-font"
echo "   - Amiri: https://fonts.google.com/specimen/Amiri"
