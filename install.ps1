param(
    [string]$ApkPath = "frontend\android\app\build\outputs\apk\debug\app-debug.apk"
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$apk = Join-Path $root $ApkPath
if (-not (Test-Path $apk)) { throw "APK not found: $apk" }

$adb = "C:\Program Files (x86)\Android\android-sdk\platform-tools\adb.exe"
if (-not (Test-Path $adb)) {
    $adb = "C:\Program Files\platform-tools\adb.exe"
}
if (-not (Test-Path $adb)) {
    $cmd = Get-Command adb -ErrorAction SilentlyContinue
    if ($cmd) { $adb = $cmd.Source }
}
if (-not (Test-Path $adb)) { throw "adb not found" }

& $adb devices
& $adb uninstall com.forum.app 2>$null
& $adb install -r $apk
& $adb shell monkey -p com.forum.app -c android.intent.category.LAUNCHER 1
