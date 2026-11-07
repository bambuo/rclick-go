#!/bin/bash
# 使用 mygo 构建 Go 主程序，嵌入 Swift FinderSync 扩展与原生 helper，
# 完成 ad-hoc 签名，并可选生成 DMG。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GO="${GO:-$(command -v go 2>/dev/null || echo "$HOME/Library/Go/1.27.3/bin/go")}"
VERSION="${VERSION:-$(plutil -extract version raw -o - "$ROOT/mygo.json" 2>/dev/null || echo 3.0.0)}"
BUILD_ID="${BUILD_ID:-$(date +%Y%m%d%H%M)}"
ARCH="${ARCH:-arm64}"
OUT="${OUT:-$ROOT/build}"
MIN_MACOS="${MIN_MACOS:-15.0}"
SKIP_DMG="${SKIP_DMG:-0}"
REGISTER="${REGISTER:-1}"
INSTALL_DIR="${INSTALL_DIR:-}"

cd "$ROOT"

echo "==> Building Go app ($VERSION $BUILD_ID)"
"$GO" tool mygo build -skip-dmg -skip-notarize -sign - .

APP="$(find "$OUT" -maxdepth 3 -name "RClick.app" -type d | head -1)"
if [ -z "$APP" ]; then
  echo "error: RClick.app not found under $OUT" >&2
  exit 1
fi
echo "    app: $APP"

echo "==> Building native helper"
mkdir -p "$ROOT/native/helper/bin"
xcrun swiftc -O -parse-as-library -swift-version 5 \
  -target "${ARCH}-apple-macos${MIN_MACOS}" \
  -framework AppKit \
  -o "$ROOT/native/helper/bin/RClickHelper" \
  "$ROOT/native/helper/RClickHelper.swift"
cp "$ROOT/native/helper/bin/RClickHelper" "$APP/Contents/MacOS/RClickHelper"
chmod +x "$APP/Contents/MacOS/RClickHelper"

echo "==> Building FinderSync extension"
APPEX="$APP/Contents/PlugIns/RClickFinderSync.appex"
rm -rf "$APPEX"
mkdir -p "$APPEX/Contents/MacOS" "$APPEX/Contents/Resources"

xcrun swiftc -O -parse-as-library -swift-version 5 \
  -target "${ARCH}-apple-macos${MIN_MACOS}" -module-name RClickFinderSync \
  -framework FinderSync -framework Cocoa \
  -Xlinker -e -Xlinker _NSExtensionMain \
  -o "$APPEX/Contents/MacOS/RClickFinderSync" \
  "$ROOT/native/finder/FinderSyncExt.swift" \
  "$ROOT/native/finder/SharedConfigStore.swift" \
  "$ROOT/native/finder/models/RCBase.swift" \
  "$ROOT/native/finder/models/OpenWithApp.swift" \
  "$ROOT/native/finder/models/RCAction.swift" \
  "$ROOT/native/finder/models/NewFile.swift" \
  "$ROOT/native/finder/models/CommonDir.swift" \
  "$ROOT/native/finder/models/QuickCommand.swift" \
  "$ROOT/native/finder/shared/Constants.swift" \
  "$ROOT/native/finder/shared/AppLogger.swift" \
  "$ROOT/native/finder/shared/Utils.swift" \
  "$ROOT/native/finder/shared/Messager.swift"

cp "$ROOT"/native/finder/resources/* "$APPEX/Contents/Resources/"
for lproj in "$ROOT"/native/finder/*.lproj; do
  [ -d "$lproj" ] || continue
  cp -R "$lproj" "$APPEX/Contents/Resources/"
done

cat > "$APPEX/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key><string>en</string>
  <key>CFBundleDisplayName</key><string>RClickFinderSync</string>
  <key>CFBundleExecutable</key><string>RClickFinderSync</string>
  <key>CFBundleIdentifier</key><string>cn.wflixu.RClick.FinderSyncExt</string>
  <key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
  <key>CFBundleLocalizations</key><array><string>en</string><string>zh-Hans</string></array>
  <key>CFBundleName</key><string>RClickFinderSync</string>
  <key>CFBundlePackageType</key><string>XPC!</string>
  <key>CFBundleShortVersionString</key><string>$VERSION</string>
  <key>CFBundleVersion</key><string>$BUILD_ID</string>
  <key>CFBundleSupportedPlatforms</key><array><string>MacOSX</string></array>
  <key>LSMinimumSystemVersion</key><string>$MIN_MACOS</string>
  <key>NSExtension</key>
  <dict>
    <key>NSExtensionAttributes</key>
    <dict>
      <key>NSExtensionActivationRule</key>
      <dict>
        <key>NSExtensionActivationSupportsFileWithMaxCount</key>
        <integer>100</integer>
      </dict>
    </dict>
    <key>NSExtensionPointIdentifier</key><string>com.apple.FinderSync</string>
    <key>NSExtensionPrincipalClass</key><string>RClickFinderSync.FinderSyncExt</string>
  </dict>
</dict>
</plist>
PLIST

echo "==> Signing"
codesign --force --sign - --identifier cn.wflixu.RClick.FinderSyncExt \
  --entitlements "$ROOT/native/finder/FinderSyncExt.entitlements" "$APPEX"
codesign --force --sign - "$APP/Contents/MacOS/RClickHelper"
# 只签主 App，不要使用 --deep：重签嵌套代码会丢失扩展的沙盒权限。
codesign --force --sign - "$APP"
codesign --verify --deep --strict --verbose=2 "$APP"

if [ -n "$INSTALL_DIR" ]; then
  echo "==> Installing to $INSTALL_DIR"
  mkdir -p "$INSTALL_DIR"
  TARGET="$INSTALL_DIR/RClick.app"
  rm -rf "$TARGET"
  ditto "$APP" "$TARGET"
  # Remove the build-tree registration so only the installed copy remains.
  if [ "$REGISTER" = "1" ]; then
    pluginkit -r "$APPEX" >/dev/null 2>&1 || true
  fi
  APP="$TARGET"
  APPEX="$APP/Contents/PlugIns/RClickFinderSync.appex"
  echo "    installed: $APP"
fi

echo "==> Registering with Launch Services and pluginkit"
if [ "$REGISTER" = "1" ]; then
  LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/LaunchServices.framework/Versions/A/Support/lsregister"
  "$LSREGISTER" -f "$APP" >/dev/null 2>&1 || true
  pluginkit -a "$APPEX" >/dev/null 2>&1 || true
  pluginkit -e use -i cn.wflixu.RClick.FinderSyncExt >/dev/null 2>&1 || true
else
  echo "    skipped (REGISTER=0)"
fi

DMG=""
if [ "$SKIP_DMG" != "1" ]; then
  echo "==> Building DMG"
  DMG="$OUT/RClick-$VERSION-$ARCH.dmg"
  STAGE="$(mktemp -d)"
  mkdir -p "$STAGE/RClick"
  ditto "$APP" "$STAGE/RClick/RClick.app"
  ln -s /Applications "$STAGE/RClick/Applications"
  hdiutil create -volname "RClick $VERSION" -srcfolder "$STAGE/RClick" -ov -format UDZO "$DMG" >/dev/null
  rm -rf "$STAGE"
fi

echo "APP=$APP"
[ -n "$DMG" ] && echo "DMG=$DMG"
echo "Next: open '$APP' and enable RClickFinderSync in System Settings > General > Login Items & Extensions > Finder Extensions."