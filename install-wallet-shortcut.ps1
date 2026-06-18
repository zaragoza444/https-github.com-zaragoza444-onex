$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Wsh = New-Object -ComObject WScript.Shell
$Desktop = [Environment]::GetFolderPath("Desktop")
$Icon = Join-Path $Root "onex-icon.ico"
if (-not (Test-Path $Icon)) {
  Write-Host "Generating onex-icon.ico..."
  python (Join-Path $Root "generate_icon.py")
}

function New-OneXShortcut($Name, $Target, $Desc) {
  $Shortcut = $Wsh.CreateShortcut((Join-Path $Desktop "$Name.lnk"))
  $Shortcut.TargetPath = $Target
  $Shortcut.WorkingDirectory = $Root
  $Shortcut.Description = $Desc
  if (Test-Path $Icon) { $Shortcut.IconLocation = $Icon }
  $Shortcut.Save()
  Write-Host "Created: $($Shortcut.FullName)"
}

New-OneXShortcut "OneX Wallet" (Join-Path $Root "run-onex-wallet.bat") "OneX Wallet - bridge + real ledger"
New-OneXShortcut "OneX Blockchain" (Join-Path $Root "run-onex.bat") "OneX node + explorer"
