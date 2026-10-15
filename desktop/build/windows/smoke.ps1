# smoke.ps1 -Release <folder> -Build <build.xml>: the Windows release proof on a Windows machine with WebView2 (CI's
# windows-latest). 1. The portable exe from go-pob-windows-amd64.zip boots, makes a New build, opens a pob:// link through
# the /api proxy, opens Build as a file and runs the 1-worker power report on it (desktop/e2e/webview.js), with the peak
# working set of go-pob and its WebView2 processes. 2. The installer installs silently, registers pob:// and the .xml
# Open with entry, and a pob:// link started through the shell launches the installed app onto that build. 3. The
# uninstaller removes both. Fails on any step.
param(
    [Parameter(Mandatory)] [string] $Release,
    [Parameter(Mandatory)] [string] $Build,
    [string] $Link = 'pob://pobbin/h-M3H3It2fDw',
    [int] $Seconds = 300
)
$ErrorActionPreference = 'Stop'

$work = Join-Path ([IO.Path]::GetTempPath()) "go-pob-smoke-$PID"
New-Item -ItemType Directory -Force $work | Out-Null
$probeSrc = Join-Path $PSScriptRoot '..\..\e2e\webview.js'
$log = Join-Path $env:APPDATA 'go-pob\settings\probe-log.json'
Remove-Item -Force -ErrorAction SilentlyContinue $log

function Write-Probe([string] $phase) {
    $path = Join-Path $work "probe-$phase.js"
    "window.__SMOKE = { phase: `"$phase`", link: `"$Link`" };`n" + (Get-Content -Raw $probeSrc) | Set-Content -Encoding utf8 $path
    $path
}

# Wait-App waits for every go-pob process to exit, sampling the summed working set of go-pob and msedgewebview2.
function Wait-App {
    $deadline = (Get-Date).AddSeconds($Seconds)
    $peak = 0
    while (Get-Process -Name go-pob -ErrorAction SilentlyContinue) {
        if ((Get-Date) -gt $deadline) {
            Stop-Process -Name go-pob -Force
            throw "go-pob still running after $Seconds s"
        }
        $sum = (Get-Process -Name go-pob, msedgewebview2 -ErrorAction SilentlyContinue | Measure-Object WorkingSet64 -Sum).Sum
        if ($sum -gt $peak) { $peak = $sum }
        Start-Sleep -Milliseconds 500
    }
    $peak
}

function Read-Log([string] $phase) {
    if (-not (Test-Path $log)) { throw "phase ${phase}: the probe wrote no log (it never ran)" }
    $entries = Get-Content -Raw $log | ConvertFrom-Json
    $entries | ForEach-Object { $_ | ConvertTo-Json -Compress -Depth 5 }
    Remove-Item $log
    if (-not ($entries | Where-Object { $_.done -eq $phase })) { throw "phase ${phase} failed" }
}

Write-Output '--- portable exe'
Expand-Archive -Force (Join-Path $Release 'go-pob-windows-amd64.zip') (Join-Path $work 'portable')
$env:GOPOB_DESKTOP_PROBE = Write-Probe 'main'
$env:GOPOB_DESKTOP_OPEN = (Resolve-Path $Build).Path
$app = Start-Process -FilePath (Join-Path $work 'portable\go-pob.exe') -PassThru `
    -RedirectStandardError (Join-Path $work 'main.err') -RedirectStandardOutput (Join-Path $work 'main.out')
$null = $app.Handle # keeps the exit code readable after the process ends
$peak = Wait-App
$app.WaitForExit()
Get-Content (Join-Path $work 'main.err') | Where-Object { $_ -notmatch 'probe: ' } | Select-Object -Last 20
Read-Log 'main'
Write-Output ('exit {0}; peak working set, go-pob + msedgewebview2: {1:N0} MB' -f $app.ExitCode, ($peak / 1MB))
if ($app.ExitCode -ne 0) { throw "go-pob exited $($app.ExitCode)" }
Remove-Item Env:GOPOB_DESKTOP_OPEN

Write-Output '--- installer'
$setup = Start-Process -FilePath (Join-Path $Release 'go-pob-amd64-installer.exe') -ArgumentList '/S' -Wait -PassThru
if ($setup.ExitCode -ne 0) { throw "installer exited $($setup.ExitCode)" }
$installed = Join-Path $env:LOCALAPPDATA 'Programs\go-pob\go-pob.exe'
if (-not (Test-Path $installed)) { throw "no $installed" }
Write-Output ('pob:// command: ' + (Get-ItemProperty 'HKCU:\Software\Classes\pob\shell\open\command').'(default)')
Write-Output ('.xml Open with: ' + ((Get-Item 'HKCU:\Software\Classes\.xml\OpenWithProgids').GetValueNames() -join ', '))

$env:GOPOB_DESKTOP_PROBE = Write-Probe 'link'
Start-Process $Link
$deadline = (Get-Date).AddSeconds(30)
while (-not (Get-Process -Name go-pob -ErrorAction SilentlyContinue)) {
    if ((Get-Date) -gt $deadline) { throw "$Link launched no go-pob" }
    Start-Sleep -Milliseconds 200
}
Write-Output ('launched: ' + (Get-Process -Name go-pob | Select-Object -First 1).Path)
$null = Wait-App
Read-Log 'link'

Write-Output '--- uninstaller'
Start-Process -FilePath (Join-Path $env:LOCALAPPDATA 'Programs\go-pob\uninstall.exe') -ArgumentList '/S' -Wait
$deadline = (Get-Date).AddSeconds(30)
while ((Test-Path 'HKCU:\Software\Classes\pob') -or (Test-Path $installed)) {
    if ((Get-Date) -gt $deadline) { throw 'the uninstaller left pob:// or the exe behind' }
    Start-Sleep -Milliseconds 500
}
Write-Output 'pob:// and the exe removed'
