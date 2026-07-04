# Load remotes.env into the current session (gitignored secrets file).
param(
    [string]$Path = ""
)

$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$file = if ($Path) { $Path } else { Join-Path $root "remotes.env" }

if (-not (Test-Path $file)) {
    return
}

Get-Content $file | ForEach-Object {
    $line = $_.Trim()
    if (-not $line -or $line.StartsWith("#")) { return }
    if ($line -match '^\s*([A-Za-z_][A-Za-z0-9_]*)=(.*)$') {
        $name = $matches[1]
        $value = $matches[2].Trim().Trim('"').Trim("'")
        if ($value -and $value -notmatch '^(your-|YOUR_)') {
            Set-Item -Path "env:$name" -Value $value
        }
    }
}
