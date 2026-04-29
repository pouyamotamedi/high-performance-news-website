#!/bin/bash
# Build script for CSS/JS minification and optimization
# Requires: npm install -g clean-css-cli terser

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
STATIC_DIR="$PROJECT_ROOT/web/static"
CSS_DIR="$STATIC_DIR/css"
JS_DIR="$STATIC_DIR/js"
DIST_DIR="$STATIC_DIR/dist"

echo "🔧 Building production assets..."

# Create dist directory
mkdir -p "$DIST_DIR/css" "$DIST_DIR/js"

# Check if tools are installed
if ! command -v cleancss &> /dev/null; then
    echo "⚠️  clean-css-cli not found. Installing..."
    npm install -g clean-css-cli
fi

if ! command -v terser &> /dev/null; then
    echo "⚠️  terser not found. Installing..."
    npm install -g terser
fi

echo "📦 Minifying CSS files..."

# Core CSS bundle (main styles)
cleancss -o "$DIST_DIR/css/core.min.css" \
    "$CSS_DIR/main.css" \
    "$CSS_DIR/responsive-utils.css" \
    "$CSS_DIR/themes.css" \
    "$CSS_DIR/dark-mode-fixes.css" \
    "$CSS_DIR/breaking-ticker.css" \
    "$CSS_DIR/mobile-menu-fallback.css"

# Homepage CSS bundle
cleancss -o "$DIST_DIR/css/homepage.min.css" \
    "$CSS_DIR/homepage.css" \
    "$CSS_DIR/professional-news.css"

# Article CSS bundle
cleancss -o "$DIST_DIR/css/article.min.css" \
    "$CSS_DIR/article.css"

# Category/Tag CSS bundle
cleancss -o "$DIST_DIR/css/category.min.css" \
    "$CSS_DIR/category.css" \
    "$CSS_DIR/tag.css"

# Search CSS
cleancss -o "$DIST_DIR/css/search.min.css" \
    "$CSS_DIR/search.css"

# Admin CSS bundle
cleancss -o "$DIST_DIR/css/admin.min.css" \
    "$CSS_DIR/admin.css" \
    "$CSS_DIR/admin-sidebar.css"

# RTL CSS
cleancss -o "$DIST_DIR/css/rtl.min.css" \
    "$CSS_DIR/rtl.css"

# Fonts CSS
cleancss -o "$DIST_DIR/css/fonts.min.css" \
    "$CSS_DIR/fonts.css"

# Critical CSS (inline in HTML)
cleancss -o "$DIST_DIR/css/critical.min.css" \
    "$CSS_DIR/critical.css"

echo "📦 Minifying JavaScript files..."

# Main JS bundle
terser "$JS_DIR/main.js" "$JS_DIR/theme.js" \
    -o "$DIST_DIR/js/core.min.js" \
    --compress --mangle

# PWA JS
terser "$JS_DIR/pwa.js" \
    -o "$DIST_DIR/js/pwa.min.js" \
    --compress --mangle

# Homepage JS
terser "$JS_DIR/homepage.js" "$JS_DIR/homepage-simple.js" \
    -o "$DIST_DIR/js/homepage.min.js" \
    --compress --mangle 2>/dev/null || \
terser "$JS_DIR/homepage-simple.js" \
    -o "$DIST_DIR/js/homepage.min.js" \
    --compress --mangle

# Article JS
terser "$JS_DIR/article.js" \
    -o "$DIST_DIR/js/article.min.js" \
    --compress --mangle

# Search JS
terser "$JS_DIR/search.js" \
    -o "$DIST_DIR/js/search.min.js" \
    --compress --mangle

# Multilingual JS
terser "$JS_DIR/multilingual.js" \
    -o "$DIST_DIR/js/multilingual.min.js" \
    --compress --mangle

# Admin JS bundle
if [ -d "$JS_DIR/admin" ]; then
    find "$JS_DIR/admin" -name "*.js" -exec cat {} + | \
    terser -o "$DIST_DIR/js/admin.min.js" --compress --mangle
fi

echo "📊 Generating asset manifest..."

# Generate manifest with file sizes and hashes
cat > "$DIST_DIR/manifest.json" << EOF
{
  "version": "$(date +%Y%m%d%H%M%S)",
  "generated": "$(date -Iseconds)",
  "css": {
EOF

# Add CSS files to manifest
first=true
for file in "$DIST_DIR/css"/*.min.css; do
    filename=$(basename "$file")
    size=$(wc -c < "$file")
    hash=$(md5sum "$file" | cut -d' ' -f1 | head -c 8)
    if [ "$first" = true ]; then
        first=false
    else
        echo "," >> "$DIST_DIR/manifest.json"
    fi
    printf '    "%s": {"size": %d, "hash": "%s"}' "$filename" "$size" "$hash" >> "$DIST_DIR/manifest.json"
done

cat >> "$DIST_DIR/manifest.json" << EOF

  },
  "js": {
EOF

# Add JS files to manifest
first=true
for file in "$DIST_DIR/js"/*.min.js; do
    filename=$(basename "$file")
    size=$(wc -c < "$file")
    hash=$(md5sum "$file" | cut -d' ' -f1 | head -c 8)
    if [ "$first" = true ]; then
        first=false
    else
        echo "," >> "$DIST_DIR/manifest.json"
    fi
    printf '    "%s": {"size": %d, "hash": "%s"}' "$filename" "$size" "$hash" >> "$DIST_DIR/manifest.json"
done

cat >> "$DIST_DIR/manifest.json" << EOF

  }
}
EOF

echo ""
echo "✅ Build complete!"
echo ""
echo "📁 Output directory: $DIST_DIR"
echo ""
echo "📊 Bundle sizes:"
echo "   CSS:"
for file in "$DIST_DIR/css"/*.min.css; do
    size=$(wc -c < "$file" | tr -d ' ')
    filename=$(basename "$file")
    printf "   - %-25s %6d bytes\n" "$filename" "$size"
done
echo ""
echo "   JS:"
for file in "$DIST_DIR/js"/*.min.js; do
    size=$(wc -c < "$file" | tr -d ' ')
    filename=$(basename "$file")
    printf "   - %-25s %6d bytes\n" "$filename" "$size"
done
echo ""
echo "💡 To use minified assets in production, update your templates to load from /static/dist/"
