param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\git-safe",
    [string]$Repo = "asd2003ru/git-safe"
)

$ErrorActionPreference = "Stop"

function Fail([string]$Message) {
    Write-Error "git-safe install error: $Message"
    exit 1
}

function Resolve-Arch {
    switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()) {
        "x64" { "amd64"; break }
        "arm64" { "arm64"; break }
        default { Fail "unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)" }
    }
}

function Add-ToUserPath([string]$Directory) {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $parts = @()
    if ($currentPath) {
        $parts = $currentPath -split ";"
    }

    if ($parts -contains $Directory) {
        return
    }

    $newPath = if ($currentPath) { "$currentPath;$Directory" } else { $Directory }
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $env:Path = "$env:Path;$Directory"
}

$arch = Resolve-Arch
$archiveName = "git-safe-windows-$arch.zip"
$downloadUrl = "https://github.com/$Repo/releases/latest/download/$archiveName"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) "git-safe-install-$([System.Guid]::NewGuid())"

New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
    $archivePath = Join-Path $tempDir $archiveName
    Write-Host "Downloading $downloadUrl"
    Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath

    Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force

    $binaryName = "git-safe-windows-$arch.exe"
    $sourcePath = Join-Path $tempDir $binaryName
    if (-not (Test-Path $sourcePath)) {
        Fail "archive does not contain $binaryName"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path $sourcePath -Destination (Join-Path $InstallDir "git-safe.exe") -Force
    Add-ToUserPath -Directory $InstallDir

    Write-Host "Installed git-safe to $InstallDir\git-safe.exe"
    Write-Host "Restart your terminal if git-safe is not available in the current session."
}
finally {
    Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
