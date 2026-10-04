<#
.SYNOPSIS
    兩個部署腳本共用的底層：輸出格式、呼叫外部程式、找 docker、確認倉庫根。

.DESCRIPTION
    這份檔案存在的理由是「同一個陷阱會在每個部署腳本裡各踩一次」。
    底下每一段都是那種陷阱：

      - 外部程式的 stderr + $ErrorActionPreference：Windows PowerShell 5.1 下會
        中止整支腳本（Invoke-Native 的說明）。
      - 檔案編碼：5.1 讀不含 BOM 的 .ps1 會把中文解讀成亂碼，並連帶把單引號吃進
        亂碼字元裡（症狀是「未預期的 '{' 語彙基元」，而錯誤訊息本身也是亂碼）。
        因此**本檔與引用它的兩個腳本都必須存成 UTF-8 with BOM**。
      - docker.exe 不在 PATH（Docker Desktop 裝在 Windows 上時）。
      - 相對路徑依賴呼叫時的工作目錄（Enter-RepoRoot 的說明）。

    把這些集中在一處之後，新增一個部署腳本只要 dot-source 這份檔案就自動正確 ——
    否則每一個新腳本都會在前三個陷阱裡選一個踩下去，而症狀全部指向錯誤的方向。

    使用方式：`. (Join-Path $PSScriptRoot 'docker-common.ps1')`
#>

$ErrorActionPreference = 'Stop'

# ---------------------------------------------------------------------------
# 輸出
# ---------------------------------------------------------------------------

function Write-Step([string]$Message) {
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Write-Ok([string]$Message) {
    Write-Host "    [OK] $Message" -ForegroundColor Green
}

function Write-Note([string]$Message) {
    Write-Host "    $Message" -ForegroundColor DarkGray
}

function Fail([string]$Message) {
    Write-Host "    [FAIL] $Message" -ForegroundColor Red
}

# ---------------------------------------------------------------------------
# 呼叫外部程式
# ---------------------------------------------------------------------------

<#
  為什麼每一個外部程式都要走這裡
  ----------------------------
  Windows PowerShell 5.1 與 PowerShell 7 對「外部程式寫 stderr」的處理不同，而
  這個專案的部署者兩種終端機都會用：

  5.1 在 $ErrorActionPreference='Stop' 下，只要外部程式寫了 stderr 就會把它包成
  NativeCommandError 並**中止整支腳本** —— 連 `2>$null` 都一樣（已實測）。而 docker
  的進度、警告、拉取訊息全部走 stderr，也就是「幾乎每一次呼叫」都會踩到。
  7 之後才有 $PSNativeCommandUseErrorActionPreference 這種明確控制。

  因此 Invoke-Native 在呼叫期間把 ErrorActionPreference 降成 Continue，並回傳
  結束碼讓呼叫端自己判斷 —— $LASTEXITCODE 是唯一在兩者之間行為一致的做法。

  輸出用 Out-Host 送出去而不是回傳：外部程式的 stdout 會變成這個函式的輸出，
  而呼叫端把結束碼收成變數時會把整份建置日誌一起收進去。
#>
function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$Arguments = @(),

        # 丟掉輸出而不是送到主控台。用於「只需要知道成功或失敗」的呼叫：
        # docker create 會把剛建立起來的容器名稱印在 stdout，不擋掉的話它會
        # 混在這支腳本自己的 [OK] 訊息中間。
        [switch]$Quiet
    )

    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        if ($Quiet) {
            & $FilePath @Arguments | Out-Null
        }
        else {
            & $FilePath @Arguments | Out-Host
        }
        return $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previous
    }
}

# ---------------------------------------------------------------------------
# 找 docker
# ---------------------------------------------------------------------------

<#
  Invoke-NativeCapture 是 Invoke-Native 的「把輸出收進來」版本，回傳
  @{ ExitCode; Output }。存在的理由只有一個：少數地方真的需要讀外部程式的
  stdout（列出映像、查映像 ID），而那些地方若自己直接呼叫外部程式，就必須各自
  重寫一次 ErrorActionPreference 的降級 —— 而 Invoke-Native 的檔頭說明那個降級
  為什麼是必要的，也就是它「被抄一次就會被抄錯一次」的內容。

  分成兩個函式而不是加一個 -Capture 開關：兩者的回傳型別不同（一個是 int，
  一個是物件），而讓同一個函式依參數回傳不同型別，會讓每個呼叫端都得先確認
  自己拿到的是哪一種。
#>
function Invoke-NativeCapture {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$Arguments = @(),

        # 丟掉 stderr。**只**用在「寫 stderr 是預期答案」的情況 —— 例如查一個
        # 刻意可能不存在的映像。不擋掉的話，那行 "No such image" 會出現在腳本自己的
        # [OK] 訊息中間，看起來像失敗而實際上是「它不存在」這個預期結果。
        #
        # 不要用它在真正會失敗的地方：那會把失敗的原因藏起來，而 Invoke-NativeCapture
        # 的存在意義就是讓呼叫端拿得到結束碼與輸出。
        [switch]$SuppressErrorOutput
    )

    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = if ($SuppressErrorOutput) { & $FilePath @Arguments 2>$null } else { & $FilePath @Arguments }
        return [pscustomobject]@{ ExitCode = $LASTEXITCODE; Output = $output }
    }
    finally {
        $ErrorActionPreference = $previous
    }
}

<#
  為什麼要自己找：Docker Desktop 安裝在 Windows 上時，docker.exe 與
  docker-compose.exe 放在「使用者自己的程式目錄」底下，而**不在 PATH 裡** ——
  只有從開始選單啟動桌面程式的那個環境變數才有它。症狀是
  `docker : 無法將「docker」辨識為 cmdlet`，而使用者會以為自己沒裝 Docker。

  連帶的細節：同一個目錄裡還有 docker-credential-desktop.exe，而憑證協助程式必須在
  PATH 上，否則建置會在拉基底映像時以
  `error getting credentials - exec: "docker-credential-desktop": executable file
  not found in %PATH%` 失敗。因此這裡加的是整個目錄，不是只加 docker.exe。
#>
function Resolve-Docker {
    if (Get-Command docker -ErrorAction SilentlyContinue) { return }

    $candidates = @(
        (Join-Path $env:LOCALAPPDATA 'Programs\DockerDesktop\resources\bin'),
        (Join-Path $env:ProgramFiles 'Docker\Docker\resources\bin')
    ) | Where-Object { $_ -and (Test-Path -LiteralPath $_) }

    if ($candidates.Count -eq 0) {
        throw '找不到 docker。請安裝 Docker Desktop，或把 docker.exe 所在目錄加入 PATH。'
    }

    Write-Step "docker 不在 PATH，改用 $($candidates[0])"
    $env:Path = "$($candidates[0]);$env:Path"
}

<#
  為什麼用 docker info 而不是 docker version
  ----------------------------------------
  `docker version --format` 的模板根節點是「client 與 server 合併」的結構，
  欄位路徑寫錯時（例如寫成 {{.ServerVersion}} 而不是 {{.Server.Version}}）它回的是
  一個模板錯誤、結束碼非 0 —— 而那與「連不上 daemon」在結果上完全無法區分。

  這個差別實際上咬過一次：模板打錯的那一版，錯誤訊息寫的是「Docker daemon 沒有在
  跑」，於是診斷方向整個跑掉（真的 daemon 是活的）。所以這裡改用只做一件事的
  `docker info`，並且在失敗時把結束碼與「docker 自己說的話已經印在上面」一起帶出來
  —— 不把一個不確定的原因講成確定的原因。
#>
function Assert-DockerRunning {
    $result = Invoke-NativeCapture -FilePath 'docker' -Arguments @('info', '--format', '{{.ServerVersion}}')
    if ($result.ExitCode -ne 0) {
        throw ("無法與 Docker daemon 通訊（docker info 的結束碼是 {0}；docker 自己說的話印在上面）。" -f $result.ExitCode) +
              "`n若那段訊息是「cannot connect to the Docker daemon」，請先啟動 Docker Desktop，" +
              "等 tray 圖示停止顯示啟動中再重跑。"
    }
    # @() 不可省：輸出一行時 PowerShell 給的是字串而不是陣列，而 $string[0] 是
    # 第一個**字元** —— 少了它會印出「Docker daemon 2 可用」。
    Write-Ok "Docker daemon $((@($result.Output)[0] | Out-String).Trim()) 可用"
}

# ---------------------------------------------------------------------------
# 位置
# ---------------------------------------------------------------------------

<#
  部署腳本用到的每一個路徑都是相對於**倉庫根**的，而 docker build 的建置情境也必須
  是那裡（backend/Dockerfile 同時需要 backend/ 與 frontend/）。因此腳本自己切過去，
  不假設呼叫時的工作目錄 —— 從 deploy/ 底下執行是最自然的使用方式，而假設 cwd
  的症狀是兩個都不指向真正原因的回報：

    - docker: path "files_server" not found
    - 部署腳本: 缺少 backend/config/config.conf（而它明明就存在）

  兩者都沒有說「你是在錯誤的目錄下執行的」。

  標記檢查同理：確認這裡確實是倉庫根。少了它，腳本被複製到別處時的症狀會是上面
  那兩句其中之一。

  呼叫端必須自己配一組 try/finally 與 Pop-Location：dot-source 進另一支腳本時，
  不還原工作目錄會改掉呼叫者的狀態。
#>
function Enter-RepoRoot {
    $repoRoot = Split-Path -Parent $PSScriptRoot

    foreach ($marker in @('docker-compose.yml', 'backend/Dockerfile', 'files_server/Dockerfile')) {
        if (-not (Test-Path -LiteralPath (Join-Path $repoRoot $marker))) {
            throw "找不到 $marker —— 這支腳本必須留在倉庫根的 deploy/ 底下（現在位於 $repoRoot）"
        }
    }

    Push-Location $repoRoot
}