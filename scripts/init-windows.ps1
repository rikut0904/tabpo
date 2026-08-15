$ErrorActionPreference = 'Stop'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Goが必要です: https://go.dev/dl/'
}

if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    throw 'Node.js/npmが必要です: https://nodejs.org/'
}

go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0

$goBin = go env GOPATH
if ($goBin -is [array]) {
    $goBin = $goBin[0]
}
$goBin = Join-Path $goBin 'bin'

Write-Host 'Windowsの初期化が完了しました。'
Write-Host "Wails CLI: $goBin\wails.exe"
Write-Host 'PATHにGoのbinを追加していない場合は、次を実行してください:'
Write-Host ('$env:Path += "' + $goBin + '"')
