<#
.SYNOPSIS
    打包這個專案的兩個服務映像，並驗證後端映像的檔案配置。

.DESCRIPTION
    這支腳本把「怎麼把映像做出來」與「怎麼知道做出來的映像是對的」合併成一步。
    後者不是錦上添花：這個專案有四種「建置成功但映像不能用」的失敗模式，而它們
    全部沒有錯誤訊息（理由見下面「為什麼要驗證檔案配置」）。

    這裡刻意用 docker build 而非 `docker compose build`，原因見 Invoke-Build 的說明。

    這支腳本同時要能在 Windows PowerShell 5.1 與 PowerShell 7 底下執行，因此它與
    docker-common.ps1 都必須存成 **UTF-8 with BOM**（5.1 讀不含 BOM 的 .ps1 會把
    中文解讀成亂碼，並連帶讓整支腳本剖析失敗）。改完請確認檔案仍帶 BOM。

.PARAMETER Target
    只打包其中一個服務（預設兩個都建）。

.PARAMETER NoCache
    不使用層快取。只在改到 Dockerfile 前半段（基底映像、apk、npm ci）時需要。

.PARAMETER SkipVerify
    跳過映像內容驗證。只在除錯這支腳本本身時用 —— 跳過之後，
    「建置成功」不再保證任何與執行期有關的事。

.EXAMPLE
    ./deploy/build-images.ps1

.EXAMPLE
    ./deploy/build-images.ps1 -Target backend -NoCache
#>
[CmdletBinding()]
param(
    [ValidateSet('all', 'backend', 'files_server')]
    [string]$Target = 'all',

    [switch]$NoCache,

    [switch]$SkipVerify
)

# 共用的底層（輸出格式、Invoke-Native、找 docker、Enter-RepoRoot）集中在這一份，
# 理由見該檔的檔頭 —— 底下每一段都是「新增腳本時會再踩一次」的陷阱。
. (Join-Path $PSScriptRoot 'docker-common.ps1')

# ---------------------------------------------------------------------------
# 前置檢查
# ---------------------------------------------------------------------------

<#
  這些檔案**不是**建置的必要條件（映像裡不含設定檔），但缺了它們就無法啟動服務，
  而症狀是「容器起來了、立刻因為設定檔找不到而退出」。在這裡先講清楚，比啟動失敗
  之後才回來查便宜。

  掛載位置刻意寫出來：兩個服務讀設定檔的路徑不同（backend 讀
  config/config.conf、files_server 讀 config.conf），照抄成同一個是這個專案最容易
  犯的部署錯誤。
#>
function Assert-ConfigFiles {
    Write-Step '檢查部署用的設定檔'

    $required = @(
        @{ Path = 'backend/config/config.conf'; Mount = '/app/config/config.conf'; Template = 'backend/config/config.conf.example' },
        @{ Path = 'files_server/config.conf'; Mount = '/app/config.conf'; Template = 'files_server/config.conf.example' },
        @{ Path = 'deploy/settings.conf'; Mount = '（不掛載，是 compose 的變數來源）'; Template = 'deploy/settings.conf.example' }
    )

    foreach ($item in $required) {
        if (Test-Path -LiteralPath $item.Path) {
            Write-Ok "$($item.Path) -> $($item.Mount)"
        }
        else {
            Write-Note "缺少 $($item.Path)（啟動服務時會需要）"
            Write-Note "  範本：$($item.Template)"
        }
    }
}

# ---------------------------------------------------------------------------
# 建置
# ---------------------------------------------------------------------------

<#
  為什麼用 docker build 而不是 `docker compose build`
  ------------------------------------------------
  compose 會先插值整份檔案，而 mysql 與 redis 的密碼寫成 `${VAR:?...}`：沒有
  deploy/settings.conf 時 `docker compose build` 會在**開始建置之前**就結束。也就是
  「打包映像」被迫依賴一份含明文密碼的部署設定檔，而映像裡根本不需要那些值。

  映像名稱沿用 compose 的預設值（<project>-<service>，project 名稱取自 compose 檔頭的
  `name: forum`）。這支腳本做出來的映像因此會被 `docker compose up` 直接沿用，不會
  因為「沒有映像」而重建 —— 兩邊的名字必須一致，否則會出現「以為是同一個版本、
  實際上是剛剛重建的」。
#>
function Invoke-Build {
    Write-Step '打包映像'

    $common = @('build', '--progress=plain')
    if ($NoCache) { $common += '--no-cache' }

    if ($Target -in @('all', 'files_server')) {
        # 這個映像的建置情境是 files_server/ 自己（它只認得自己的那份 .dockerignore）。
        $code = Invoke-Native -FilePath 'docker' `
            -Arguments ($common + @('-t', 'forum-files_server', '-f', 'files_server/Dockerfile', 'files_server'))
        if ($code -ne 0) { throw 'files_server 映像建置失敗' }
        Write-Ok 'forum-files_server'
    }

    if ($Target -in @('all', 'backend')) {
        # 建置情境是最後那個「.」，也就是倉庫根（backend/Dockerfile 同時需要
        # backend/ 的 Go 程式碼與 frontend/ 的 TypeScript），理由見它的檔頭。
        $code = Invoke-Native -FilePath 'docker' `
            -Arguments ($common + @('-t', 'forum-backend', '-f', 'backend/Dockerfile', '.'))
        if ($code -ne 0) { throw 'backend 映像建置失敗' }
        Write-Ok 'forum-backend'
    }
}

# ---------------------------------------------------------------------------
# 驗證
# ---------------------------------------------------------------------------

<#
  為什麼要驗證檔案配置
  ------------------
  這個專案有四種「建置成功、容器也起來了、但功能是壞的」情況，而它們全部不會
  產生任何錯誤訊息：

    1. /asset/ 與映像裡的位置不一致 → PWA 圖示、logo、字型全部 404，症狀只是
       「安裝了圖示但打開是空白」。前端路徑由後端在啟動時推測（frontendAssetRoot），
       而那組推測的候選與 Dockerfile 的 COPY 目的地是兩處獨立的事實 ——
       沒有任何機制保證它們相同。
    2. /app/logs 是 root 擁有 → compose 掛在那裡的 named volume 會**繼承**映像裡
       的擁有者，於是非 root 的行程在啟動後第一次寫 server.log 才失敗。
    3. 前端產物沒進到映像 → 每一頁都是 404，而 distroless 沒有 shell，
       「進去看一眼」這個診斷手段不存在（見 backend/Dockerfile 的檔頭）。
    4. 設定檔掛載點不存在 → 容器啟動時才會以「工作目錄不存在」結束。

  這四件事在 build 的輸出裡完全看不出來，因此只能在這裡檢查。

  為什麼用 docker cp 而不是 exec 進去看：distroless 沒有 /bin/sh，exec 任何東西都
  以 "exec: ...: no such file or directory" 結束 —— 那正是它先被引發的原因。
  docker cp 是唯一能在不啟動行程的情況下讀取映像檔案的手段。
#>
function Test-BackendImage {
    Write-Step '驗證 forum-backend 的檔案配置'

    $container = New-ProbeContainer 'forum-backend'
    try {
        # 取的是各有各的理由的代表，而不是把 16 個 HTML 殼列一份 —— 那是
        # frontend_shells_test.go 的工作，在腳本裡抄一份只會多一個會漂移的來源。
        $required = @(
            @{ Path = '/app/forum'; Why = '後端執行檔' },
            @{ Path = '/app/frontend/dist/forum-profile.html'; Why = '前端殼（啟動時會為它算 CSP 雜湊）' },
            @{ Path = '/app/frontend/dist/forum-manifest.json'; Why = 'PWA manifest（路徑被 <link rel=manifest> 綁死）' },
            @{ Path = '/app/frontend/dist/service-worker.js'; Why = 'service worker（位置改掉等於沒有）' },
            @{ Path = '/app/frontend/asset/logo.png'; Why = '/asset/ 的一手素材；位置不一致時只有這裡會 404' },
            @{ Path = '/app/config'; Why = '設定檔掛載點' },
            @{ Path = '/app/logs'; Why = 'server.log 掛載點' }
        )

        $ok = $true
        foreach ($item in $required) {
            if (Test-ImagePath $container $item.Path) {
                Write-Ok "$($item.Path)  $($item.Why)"
            }
            else {
                Fail "$($item.Path) 不在映像裡（$($item.Why)）"
                $ok = $false
            }
        }

        if (-not (Assert-MountPointOwnership $container)) { $ok = $false }

        # -check 證明的是「執行檔在這個映像裡真的跑得起來」：distroless 的非 root
        # 環境、唯讀的設定檔掛載、CA 憑證在不在 —— 這些都是啟動時才會出現的問題。
        if (Test-Path -LiteralPath 'backend/config/config.conf') {
            $conf = (Resolve-Path 'backend/config/config.conf').Path
            $code = Invoke-Native -FilePath 'docker' -Quiet -Arguments @(
                'run', '--rm', '-v', "${conf}:/app/config/config.conf:ro",
                'forum-backend', '/app/forum', '-check'
            )
            if ($code -eq 0) { Write-Ok '/app/forum -check 通過（設定檔可讀、行程可執行）' }
            else { Fail '/app/forum -check 未通過'; $ok = $false }
        }
        else {
            Write-Note '跳過 -check（backend/config/config.conf 不存在）'
        }

        if (-not $ok) { throw 'forum-backend 的檔案配置驗證失敗' }
    }
    finally {
        Remove-ProbeContainer $container
    }
}

function Test-FilesServerImage {
    Write-Step '驗證 forum-files_server 的檔案配置'

    $container = New-ProbeContainer 'forum-files_server'
    try {
        $ok = $true
        foreach ($path in @('/app/files_server', '/app/storage')) {
            if (Test-ImagePath $container $path) { Write-Ok $path }
            else { Fail "$path 不在映像裡"; $ok = $false }
        }
        if (-not $ok) { throw 'forum-files_server 的檔案配置驗證失敗' }

        # storage 必須是 volume：否則容器每次重建都會失去所有上傳的檔案，而症狀是
        # 「貼文裡的圖片全都 404」。這個映像刻意**不**設 healthcheck ——
        # 它沒有可探測的端點，加一個會讓「Redis 掛掉時降級為不驗證媒體 token」
        # 變成「容器被判定為不健康並重啟」，與該服務刻意採取的策略衝突。
        Write-Note '這個映像刻意沒有 healthcheck（理由見 files_server/Dockerfile）'
    }
    finally {
        Remove-ProbeContainer $container
    }
}

function Test-ImagePath([string]$Container, [string]$Path) {
    $temp = Join-Path ([System.IO.Path]::GetTempPath()) ([Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $temp | Out-Null
    try {
        $code = Invoke-Native -FilePath 'docker' -Quiet -Arguments @('cp', "$($Container):$($Path)", $temp)
        return ($code -eq 0)
    }
    finally {
        Remove-Item -LiteralPath $temp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function New-ProbeContainer([string]$Image) {
    $name = 'forum-build-verify-' + [Guid]::NewGuid().ToString('N').Substring(0, 8)
    $code = Invoke-Native -FilePath 'docker' -Quiet -Arguments @('create', '--name', $name, $Image)
    if ($code -ne 0) { throw "無法為 $Image 建立驗證用的容器" }
    return $name
}

function Remove-ProbeContainer([string]$Name) {
    Invoke-Native -FilePath 'docker' -Quiet -Arguments @('rm', '-f', $Name) | Out-Null
}

<#
  named volume 會繼承映像裡那個目錄的擁有者，因此掛載點必須已經是 nonroot（65532）。
  這一項要讀 tar 的 uid/gid，docker cp 看不到 —— 所以改用 docker export。

  tar 是 Windows 10 1803 之後內建的 bsdtar。找不到時只警告、不回傳失敗：那台機器
  沒有 tar 不是映像的問題，而讓打包腳本因為主機少一個選用工具而失敗會更糟。
  但也不會假裝驗過了 —— 訊息會說明這一項沒有被檢查。

  回傳值是「有沒有發現問題」，而不是「檢查過沒有問題」—— 兩者的差別在跳過時
  才是真的，而那正是最容易把警告讀成通過的地方。

  這裡刻意不走 Invoke-Native：那一個把輸出送到主控台，而這裡需要把 tar 的清單
  收成變數。stderr 的處理照樣要降 ErrorActionPreference（見 Invoke-Native 的說明）。
#>
function Assert-MountPointOwnership([string]$Container) {
    if (-not (Get-Command tar -ErrorAction SilentlyContinue)) {
        Write-Note '跳過掛載點擁有權檢查：這台機器找不到 tar'
        return $true
    }

    $tarPath = Join-Path ([System.IO.Path]::GetTempPath()) "$Container.tar"
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        & docker export $Container -o $tarPath | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Write-Note '跳過掛載點擁有權檢查：docker export 失敗'
            return $true
        }

        # tar -tv 的開頭是 <mode> <links> <uid> <gid>，之後才是 size、日期與路徑。
        # **刻意不去數日期有幾個欄位**：那個數量取決於 locale（英文是
        # "2026-10-04 03:18" 兩個 token，繁中可能是 "10月 04 03:18" 三個），
        # 而寫死欄位數的解析會在別的語系上安靜地全部失配 —— 症狀是「擁有權檢查
        # 說這個目錄不存在」，而它明明存在。
        # 因此只取前三欄之後的餘字串，再用「結尾是這個路徑」來認。
        $found = @{}
        foreach ($line in (tar -tvf $tarPath)) {
            $parts = $line.Trim() -split '\s+', 5
            if ($parts.Count -lt 5) { continue }
            if ($parts[4] -notmatch 'app/(?<dir>config|logs)/$') { continue }
            $found[$Matches.dir] = @{ Uid = $parts[2]; Gid = $parts[3] }
        }
    }
    finally {
        $ErrorActionPreference = $previous
        Remove-Item -LiteralPath $tarPath -Force -ErrorAction SilentlyContinue
    }

    $ok = $true
    foreach ($dir in @('config', 'logs')) {
        if (-not $found.ContainsKey($dir)) {
            Fail "app/$dir/ 不在匯出的檔案清單裡（擁有權無從確認）"
            $ok = $false
        }
        elseif ($found[$dir].Uid -eq '65532' -and $found[$dir].Gid -eq '65532') {
            Write-Ok "app/$dir/ 擁有權 65532:65532（named volume 會繼承它）"
        }
        else {
            Fail "app/$dir/ 擁有權是 $($found[$dir].Uid):$($found[$dir].Gid)，應為 65532:65532 —— 非 root 行程會無法寫入"
            $ok = $false
        }
    }
    return $ok
}

# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------

function Show-Summary {
    Write-Step '完成'
    Invoke-Native -FilePath 'docker' -Arguments @('images', '--format', '    {{.Repository}}:{{.Tag}}  {{.Size}}', 'forum-*') | Out-Null

    Write-Host ''
    Write-Host '啟動服務（compose 需要 deploy/settings.conf 才有密碼可插值，' -ForegroundColor Cyan
    Write-Host '否則會以「MYSQL_ROOT_PASSWORD 必須在 settings.conf 設定」結束）：' -ForegroundColor Cyan
    Write-Host '    docker compose --env-file deploy/settings.conf up -d' -ForegroundColor White
    Write-Host ''
    Write-Host '部署前先驗證設定（這份輸出一致最值得先看）：' -ForegroundColor Cyan
    Write-Host '    docker compose --env-file deploy/settings.conf run --rm backend /app/forum -check' -ForegroundColor White
    Write-Host ''
    Write-Host 'backend 讀 /app/config/config.conf、files_server 讀 /app/config.conf ——' -ForegroundColor DarkGray
    Write-Host '兩個掛載路徑不同，compose 檔裡已經分開寫好了。' -ForegroundColor DarkGray
    Write-Host ''
    Write-Host '要推到 registry（Docker Hub 或任何需要 namespace 的）請接著執行：' -ForegroundColor Cyan
    Write-Host '    ./deploy/tag-images.ps1' -ForegroundColor White
}

# 呼叫放在檔案最後：PowerShell 的函式必須先被定義才存在於執行期。
# Push-Location / Pop-Location 讓「從任何目錄執行」與「離開時不改變呼叫者的
# 工作目錄」兩件事同時成立（後者在被 dot-source 進另一支腳本時才看得出差別）。
Enter-RepoRoot
try {
    Resolve-Docker
    Assert-DockerRunning
    Assert-ConfigFiles
    Invoke-Build

    if ($SkipVerify) {
        Write-Step '已跳過映像內容驗證（-SkipVerify）'
    }
    else {
        if ($Target -in @('all', 'backend')) { Test-BackendImage }
        if ($Target -in @('all', 'files_server')) { Test-FilesServerImage }
    }

    Show-Summary
}
finally {
    Pop-Location
}
