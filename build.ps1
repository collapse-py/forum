param(
    [ValidateSet('debug','release')]
    [string]$Variant = 'debug',
    [string]$BackendUrl = 'https://forum.isme.ccwu.cc'
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$frontend = Join-Path $root 'frontend'
$android = Join-Path $frontend 'android'

if (-not $env:JAVA_HOME -or -not (Test-Path (Join-Path $env:JAVA_HOME 'bin\java.exe'))) {
    $jdk = 'C:\Program Files\Java\jdk-21.0.10'
    if (Test-Path (Join-Path $jdk 'bin\java.exe')) { $env:JAVA_HOME = $jdk }
}

if (-not $env:ANDROID_HOME -or -not (Test-Path $env:ANDROID_HOME)) {
    $sdk = 'C:\Program Files (x86)\Android\android-sdk'
    if (Test-Path $sdk) { $env:ANDROID_HOME = $sdk }
}

if (-not $env:JAVA_HOME) { throw 'JAVA_HOME not found' }
if (-not $env:ANDROID_HOME) { throw 'ANDROID_HOME not found' }

$env:PATH = "$env:JAVA_HOME\bin;$env:ANDROID_HOME\platform-tools;$env:PATH"
$env:CAPACITOR_SERVER_URL = $BackendUrl

Push-Location $frontend
try {
    npm run build:android
    npx cap sync android
} finally {
    Pop-Location
}

Push-Location $android
try {
    if ($Variant -eq 'release') {
        .\gradlew assembleRelease
        $apk = 'app\build\outputs\apk\release\app-release.apk'
    } else {
        .\gradlew assembleDebug
        $apk = 'app\build\outputs\apk\debug\app-debug.apk'
    }
} finally {
    Pop-Location
}

$apkPath = Join-Path $android $apk
Write-Host "APK: $apkPath"
if (Test-Path $apkPath) { Get-Item $apkPath } else { throw 'APK not found' }
