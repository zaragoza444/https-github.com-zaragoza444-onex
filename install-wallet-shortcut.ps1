$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Wsh = New-Object -ComObject WScript.Shell
$Desktop = [Environment]::GetFolderPath("Desktop")
$Shortcut = $Wsh.CreateShortcut((Join-Path $Desktop "Shiva Wallet.lnk"))
$Shortcut.TargetPath = Join-Path $Root "run-shiva-wallet.bat"
$Shortcut.WorkingDirectory = $Root
$Shortcut.Description = "Shiva Wallet - bridge to local Shiva blockchain"
$Icon = Join-Path $Root "shiva-icon.ico"
if (Test-Path $Icon) { $Shortcut.IconLocation = $Icon }
$Shortcut.Save()
Write-Host "Created: $($Shortcut.FullName)"
