<#
.SYNOPSIS
    把本專案產出的映像重新標記到一個 registry namespace 下，例如
    forum-backend:latest -> 0collapse0/forum-backend:latest。

.DESCRIPTION
    build-images.ps1 產生的映像名稱是 compose 的預設值（forum-backend、
    forum-files_server），那個名字沒有 namespace，因此不能直接 push 到
    Docker Hub 或任何需要「使用者名/倉庫名」格式的 registry。這支腳本補上
    那個前綴。

    為什麼是「重新標記」而不是「改名」
    ------------------------------
    docker tag 是加上第二個標籤，來源標籤會留著。刻意保留，因為：

      - 本機的 compose 仍然認得 forum-backend（compose 檔裡沒有 image 欄位，
        它用的就是那個名字），刪掉來源會讓 `docker compose up` 立刻變成
        「沒有映像」而重建一次。
      - 沒有後果：同一個映像 ID 掛幾個標籤不佔額外空間。

    命名空間格式在這裡擋掉，而不是留給 push
    --------------------------------------
    registry 的 namespace 必須是小寫、且只能含英數字與 . _ -。寫成
    0Collapse0 或含空格的話，`docker push` 才會回一句與真正原因無關的
    `invalid reference format`。所以格式錯誤在這裡就回錯，並說明是哪一條
    不符合。

.PARAMETER Prefix
    要加在映像名稱前面的 namespace，例如 0collapse0。留空則在啟動時詢問。

.PARAMETER NoPrompt
    沒有 -Prefix 時直接報錯而不詢問。給 CI 或任何 stdin 不是終端機的情境 ——
    Read-Host 在那裡不會停在等待輸入，而會立刻拿到空字串。

.PARAMETER Push
    重新標記之後順便 push。沒有它就只印出 push 指令。

.PARAMETER Force
    目標標籤已經存在、但指向不同映像時覆寫它。沒有它時會停下來，因為覆寫之後
    registry 上那份就是舊的版本，而症狀是「push 上去的東西不對」。

.EXAMPLE
    ./deploy/tag-images.ps1
    （詢問 namespace，例如輸入 0collapse0）

.EXAMPLE
    ./deploy/tag-images.ps1 -Prefix 0collapse0 -Push

.EXAMPLE
    ./deploy/tag-images.ps1 -Prefix 0collapse0 -NoPrompt
#>
[CmdletBinding()]
param(
    [string]$Prefix,

    [switch]$NoPrompt,

    [switch]$Push,

    [switch]$Force
)

. (Join-Path $PSScriptRoot 'docker-common.ps1')

# 只處理本專案產出的映像。用前綴比對而不是把兩個名稱寫死：之後若新增一個服務
# （或某個實驗映像），這個腳本會自動一起處理，而漏掉一個的症狀是「其中一個映像
# 沒被 push」，沒有任何錯誤訊息。
$SourcePrefix = 'forum-'

# ---------------------------------------------------------------------------
# 命名空間
# ---------------------------------------------------------------------------

<#
  三條規則，各對應一種真實的 registry 錯誤：

    1. 只能有小寫英數字與 . _ -，且必須以英數字開頭與結尾。
    2. 不能含 '/'。namespace 是一段，不是路徑 —— 含斜線會被解讀成
       registry 主機名，而那會讓 push 試圖連到一台不存在的主機。
    3. 長度上限。Docker Hub 的 namespace 上限是 30 個字元。
#>
function Assert-Prefix([string]$Value, [switch]$AllowEmpty) {
    if ([string]::IsNullOrWhiteSpace($Value)) {
        if ($AllowEmpty) { return '' }
        throw 'namespace 不能是空的。請用 -Prefix 傳入（例如 -Prefix 0collapse0），或不要加 -NoPrompt 讓腳本詢問。'
    }

    $prefix = $Value.Trim()

    if ($prefix.Contains('/')) {
        throw "namespace 不能包含 '/'：$prefix。只給 namespace（例如 0collapse0），不要給 registry 主機名或路徑。"
    }

    if ($prefix -cmatch '[A-Z]') {
        throw "namespace 必須全部小寫：$prefix。registry 不接受大寫字母，而 push 的錯誤訊息只會說 invalid reference format。"
    }

    if ($prefix -notmatch '^[a-z0-9][a-z0-9._-]*[a-z0-9]$') {
        $bad = ($prefix.ToCharArray() | Where-Object { $_ -notmatch '[a-z0-9._-]' } | Select-Object -Unique) -join ' '
        $hint = if ($bad) { "違規字元：$bad" } else { '必須以英數字開頭與結尾' }
        throw "namespace 只允許小寫英數字與 . _ -：$prefix（$hint）"
    }

    if ($prefix.Length -gt 30) {
        throw "namespace 長度 $($prefix.Length) 超過 Docker Hub 的 30 字元上限：$prefix"
    }

    return $prefix
}

function Resolve-Prefix {
    if (-not [string]::IsNullOrWhiteSpace($Prefix)) {
        return Assert-Prefix $Prefix
    }

    if ($NoPrompt) {
        # 這裡刻意不給「空值就跳過」的路徑：整支腳本的目的就是重新標記，
        # 靜靜地什麼都不做會讓映像以沒有 namespace 的名字被推上去 —— 而那個錯誤
        # 要到部署別的機器時才會浮現。
        throw '缺少 -Prefix。-NoPrompt 的意思是「不要詢問」，不是「沒有就略過」。'
    }

    Write-Step '輸入 registry namespace'
    Write-Note '例如 0collapse0 —— 結果會是 0collapse0/forum-backend:latest'
    Write-Note '只有小寫英數字與 . _ -（不要包含 registry 主機名或斜線）'
    Write-Host 'namespace: ' -NoNewline -ForegroundColor White

    return Assert-Prefix (Read-Host)
}

# ---------------------------------------------------------------------------
# 重新標記
# ---------------------------------------------------------------------------

function Get-SourceImages {
    # 輸出格式用 | 分隔而不是空白：名稱與標籤裡的空白雖然不可能出現，但用一個
    # 明確分隔符就不必去猜。
    $result = Invoke-NativeCapture -FilePath 'docker' -Arguments @(
        'images', '--format', '{{.Repository}}|{{.Tag}}'
    )
    if ($result.ExitCode -ne 0) {
        # 刻意不讓這個情況落到下面的「找不到任何 forum-* 映像」：那一句的意思會是
        # 「映像列表是空的」，而實際上是「docker 失敗了」。
        throw "docker images 失敗（結束碼 $($result.ExitCode)）"
    }

    $images = @()
    foreach ($line in $result.Output) {
        if (-not $line) { continue }
        $parts = $line -split '\|', 2
        if ($parts.Count -lt 2) { continue }
        # <none> 是被 intermediate step 或手動 untag 留下的無名映像，
        # 它沒有可保留的來源標籤，而重新標記一個無名的東西沒有任何意義。
        if ($parts[0] -eq '<none>' -or $parts[1] -eq '<none>') { continue }
        if (-not $parts[0].StartsWith($SourcePrefix)) { continue }
        $images += [pscustomobject]@{ Name = $parts[0]; Tag = $parts[1] }
    }
    return $images
}

<#
  為什麼要逐一比對映像 ID，而不是直接 tag 然後看有沒有出錯
  ------------------------------------------------------
  docker tag 對「目標已存在」是**成功的**（它只是把標籤指過去）。因此
  「重新執行一次」會靜靜地蓋掉別的版本，而症狀是 registry 上那份不是你以為的
  版本。所以在標記之前先比對 ID，並在指向不同映像時停下來。
#>
function Get-ImageId([string]$Reference) {
    # 目標不存在時 docker 會寫 stderr 並回非 0 —— 這正是「不存在」的答案，不是
    # 錯誤，因此走 Invoke-NativeCapture 並且壓掉 stderr（理由見該函式）。
    $result = Invoke-NativeCapture -SuppressErrorOutput -FilePath 'docker' -Arguments @(
        'image', 'inspect', '--format', '{{.Id}}', $Reference
    )
    if ($result.ExitCode -ne 0 -or -not $result.Output) { return $null }
    # @() 不可省：外部程式的輸出只有一行時，PowerShell 給的是**字串**而不是陣列，
    # 而 $string[0] 取到的是第一個字元。少了 @()，映像 ID 會變成單一字元，
    # 於是下面每一次比較都在比字元 —— 而那會讓「目標已經是同一個映像」這個判斷
    # 永遠不成立，於是每次執行都覆寫一次別的版本。
    return (@($result.Output)[0] | Out-String).Trim()
}

function Invoke-Tag {
    param(
        [Parameter(Mandatory = $true)][object]$Image,
        [Parameter(Mandatory = $true)][string]$Namespace,
        [switch]$Force
    )

    $source = "$($Image.Name):$($Image.Tag)"
    $target = "$Namespace/$($Image.Name):$($Image.Tag)"

    $sourceId = Get-ImageId $source
    if (-not $sourceId) {
        Fail "$source 不存在（映像列表與實際狀態不一致？）"
        return $false
    }

    $targetId = Get-ImageId $target
    if ($targetId -eq $sourceId) {
        Write-Ok "$target  已經指向同一個映像，略過"
        return $true
    }

    if ($targetId) {
        if (-not $Force) {
            Fail "$target 已經存在而且指向**另一個**映像。確認要覆寫請加 -Force"
            return $false
        }
        Write-Note "覆寫既有的 $target"
    }

    # 括號不可省：寫成 `if (Invoke-Native ... -ne 0)` 時，PowerShell 會把 -ne 當成
    # Invoke-Native 的一個參數名（於是回「A parameter cannot be found that matches
    # parameter name 'ne'」），而那個訊息完全不會指向真正的原因。
    if ((Invoke-Native -Quiet -FilePath 'docker' -Arguments @('tag', $source, $target)) -ne 0) {
        Fail "docker tag $source $target 失敗"
        return $false
    }

    # 標記之後回頭確認，而不是假設它成功了：這是這個腳本唯一能證明「namespace
    # 加對了」的地方，而打錯一個字元的症狀是 push 到錯誤的倉庫。
    if ((Get-ImageId $target) -ne $sourceId) {
        Fail "$target 的映像 ID 與來源不符 —— docker tag 沒有生效"
        return $false
    }

    Write-Ok "$source -> $target"
    return $true
}

# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------

function Show-Summary([string]$Namespace, [object[]]$Tagged) {
    Write-Step '完成'
    Invoke-Native -FilePath 'docker' -Arguments @(
        'images', '--format', '    {{.Repository}}:{{.Tag}}  {{.Size}}', "$Namespace/*"
    ) | Out-Null

    Write-Host ''
    if ($Push) {
        Write-Host "已 push $($Tagged.Count) 個映像。" -ForegroundColor Green
    }
    else {
        Write-Host '還沒有 push。要推送請執行：' -ForegroundColor Cyan
        foreach ($image in $Tagged) {
            Write-Host "    docker push $Namespace/$($image.Name):$($image.Tag)" -ForegroundColor White
        }
        Write-Host ''
        Write-Host '或直接重跑這支腳本並加上 -Push。' -ForegroundColor DarkGray
    }

    Write-Host ''
    Write-Host '來源標籤（forum-backend 等）刻意保留：compose 檔沒有 image 欄位，' -ForegroundColor DarkGray
    Write-Host '它用的就是那個名字，刪掉會讓 docker compose up 變成「沒有映像」。' -ForegroundColor DarkGray
}

# 呼叫放在檔案最後：PowerShell 的函式必須先被定義才存在於執行期。
Enter-RepoRoot
try {
    Resolve-Docker
    Assert-DockerRunning

    $namespace = Resolve-Prefix
    Write-Step "重新標記到 $namespace/*"

    $sources = Get-SourceImages
    if ($sources.Count -eq 0) {
        throw "找不到任何 $SourcePrefix* 映像。請先執行 deploy/build-images.ps1。"
    }
    Write-Note "來源：$($sources.Count) 個（$SourcePrefix*）"

    $tagged = @()
    $failed = $false
    foreach ($image in $sources) {
        if (Invoke-Tag -Image $image -Namespace $namespace -Force:$Force) {
            $tagged += $image
        }
        else {
            $failed = $true
        }
    }

    if ($failed) {
        # 刻意在 push 之前就停下：把一半的映像推上去之後再失敗，會留下一個
        # namespace 裡新舊版本各半的狀態，而那比全部沒推更難診斷。
        throw '有映像沒有重新標記成功，因此沒有 push（否則會留下新舊各半的 namespace）'
    }

    if ($Push) {
        Write-Step "推送 $namespace/*"
        foreach ($image in $tagged) {
            if ((Invoke-Native -FilePath 'docker' -Arguments @('push', "$namespace/$($image.Name):$($image.Tag)")) -ne 0) {
                throw "push $namespace/$($image.Name):$($image.Tag) 失敗（多半是還沒 docker login）"
            }
        }
    }

    Show-Summary $namespace $tagged
}
finally {
    Pop-Location
}