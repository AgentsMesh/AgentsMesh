# AgentsMesh Runner Installation Script for Windows
# Usage: irm https://agentsmesh.ai/install.ps1 | iex
#
# For macOS/Linux, use: curl -fsSL https://agentsmesh.ai/install.sh | sh

$ErrorActionPreference = "Stop"

# GitHub release repository
$GITHUB_REPO = "AgentsMesh/AgentsMesh"
$BINARY_NAME = "agentsmesh-runner.exe"

# Colors
function Write-Info { Write-Host "==> " -ForegroundColor Blue -NoNewline; Write-Host $args }
function Write-Success { Write-Host "==> " -ForegroundColor Green -NoNewline; Write-Host $args }
function Write-Warn { Write-Host "==> " -ForegroundColor Yellow -NoNewline; Write-Host $args }
function Write-Err { Write-Host "==> " -ForegroundColor Red -NoNewline; Write-Host $args }

# Print banner (ASCII-only to avoid iex parsing issues with Unicode)
function Show-Banner {
    Write-Host ""
    Write-Host "     _                    _       __  __           _     " -ForegroundColor Cyan
    Write-Host "    / \   __ _  ___ _ __ | |_ ___|  \/  | ___  __| |__  " -ForegroundColor Cyan
    Write-Host "   / _ \ / _' |/ _ \ '_ \| __/ __| |\/| |/ _ \/ _| '_ \ " -ForegroundColor Cyan
    Write-Host "  / ___ \ (_| |  __/ | | | |_\__ \ |  | |  __/\__ \ | | |" -ForegroundColor Cyan
    Write-Host " /_/   \_\__, |\___|_| |_|\__|___/_|  |_|\___||___/_| |_|" -ForegroundColor Cyan
    Write-Host "         |___/                                           " -ForegroundColor Cyan
    Write-Host ""
    Write-Host "              Runner Installation Script" -ForegroundColor White
    Write-Host ""
}

# Detect architecture
function Get-Platform {
    # PROCESSOR_ARCHITEW6432 holds the real OS arch when running as 32-bit process on 64-bit OS (WoW64)
    $arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }

    switch ($arch) {
        "AMD64" { return "windows_amd64" }
        "ARM64" { return "windows_arm64" }
        "x86" {
            throw "Unsupported architecture: x86 (32-bit). AgentsMesh Runner requires 64-bit Windows (x64 or ARM64)."
        }
        default {
            throw "Unsupported architecture: $arch. AgentsMesh Runner supports Windows x64 and ARM64 only. Download manually from: https://github.com/$GITHUB_REPO/releases/latest"
        }
    }
}

# Get latest version from GitHub
function Get-LatestVersion {
    Write-Info "Fetching latest version..."

    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$GITHUB_REPO/releases/latest" -UseBasicParsing
        $version = $release.tag_name -replace "^v", ""
        Write-Info "Latest version: v$version"
        return $version
    }
    catch {
        throw "Failed to fetch latest version: $_"
    }
}

# Get install directory
function Get-InstallDir {
    # Try to use a directory in PATH, or create one in user's local app data
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User") -split ";"

    # Check for common bin directories
    $candidates = @(
        "$env:USERPROFILE\.local\bin",
        "$env:USERPROFILE\bin",
        "$env:LOCALAPPDATA\Programs\agentsmesh"
    )

    foreach ($dir in $candidates) {
        if ($userPath -contains $dir) {
            return $dir
        }
    }

    # Default to LocalAppData
    $installDir = "$env:LOCALAPPDATA\Programs\agentsmesh"
    return $installDir
}

# Add to PATH if needed
function Add-ToPath {
    param([string]$Directory)

    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($userPath -notlike "*$Directory*") {
        Write-Info "Adding $Directory to PATH..."
        $newPath = "$Directory;$userPath"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        $env:PATH = "$Directory;$env:PATH"
        Write-Success "Added to PATH. You may need to restart your terminal."
    }
}

# Download and install
function Install-Runner {
    param(
        [string]$Version,
        [string]$Platform
    )

    $downloadUrl = "https://github.com/$GITHUB_REPO/releases/download/v$Version/agentsmesh-runner_${Version}_${Platform}.zip"
    $installDir = Get-InstallDir

    Write-Info "Downloading from: $downloadUrl"

    # Create temp directory
    $tempDir = Join-Path $env:TEMP "agentsmesh-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

    try {
        # Download
        $zipPath = Join-Path $tempDir "runner.zip"
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing

        # Extract
        Write-Info "Extracting..."
        Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force

        # Find binary
        $binaryPath = Get-ChildItem -Path $tempDir -Filter "agentsmesh-runner.exe" -Recurse | Select-Object -First 1
        if (-not $binaryPath) {
            throw "Binary not found in archive"
        }

        # Create install directory
        if (-not (Test-Path $installDir)) {
            Write-Info "Creating directory: $installDir"
            New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        }

        # Move binary
        Write-Info "Installing to $installDir..."
        $destPath = Join-Path $installDir $BINARY_NAME

        # Remove existing if present
        if (Test-Path $destPath) {
            Remove-Item $destPath -Force
        }

        Move-Item -Path $binaryPath.FullName -Destination $destPath -Force

        # Add to PATH
        Add-ToPath -Directory $installDir

        Write-Success "AgentsMesh Runner v$Version installed successfully!"
        return $destPath
    }
    finally {
        # Cleanup
        Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Verify installation
function Test-Installation {
    param([string]$BinaryPath)

    if (Test-Path $BinaryPath) {
        Write-Host ""
        Write-Success "Installation verified:"
        & $BinaryPath version
        return $true
    }
    return $false
}

# Print next steps (ASCII-only separators for iex compatibility)
function Show-NextSteps {
    Write-Host ""
    Write-Host "------------------------------------------------------------------------" -ForegroundColor DarkGray
    Write-Host ""
    Write-Success "Next steps:"
    Write-Host ""
    Write-Host "  1. Register your runner:" -ForegroundColor White
    Write-Host "     agentsmesh-runner register --server https://agentsmesh.ai --token <YOUR_TOKEN>" -ForegroundColor Blue
    Write-Host ""
    Write-Host "  2. Start the runner:" -ForegroundColor White
    Write-Host "     agentsmesh-runner run" -ForegroundColor Blue
    Write-Host ""
    Write-Host "  Get your registration token from: Settings > Runners > Create Token" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  For more options, run: " -ForegroundColor White -NoNewline
    Write-Host "agentsmesh-runner --help" -ForegroundColor Blue
    Write-Host ""
    Write-Host "------------------------------------------------------------------------" -ForegroundColor DarkGray
}

# Check for Scoop (returns $false if user chooses Scoop instead)
function Test-Scoop {
    if (Get-Command scoop -ErrorAction SilentlyContinue) {
        Write-Host ""
        Write-Warn "Scoop detected! You can also install via:"
        Write-Host "     scoop bucket add agentsmesh https://github.com/AgentsMesh/scoop-bucket" -ForegroundColor Blue
        Write-Host "     scoop install agentsmesh-runner" -ForegroundColor Blue
        Write-Host ""
        $response = Read-Host "Continue with direct installation? [Y/n]"
        if ($response -match "^[nN]") {
            Write-Info "Installation cancelled. Use Scoop to install."
            return $false
        }
    }
    return $true
}

# Main
function Main {
    Show-Banner

    try {
        $platform = Get-Platform
        Write-Info "Detected platform: $platform"

        if (-not (Test-Scoop)) { return }

        $version = Get-LatestVersion
        $binaryPath = Install-Runner -Version $version -Platform $platform

        if (Test-Installation -BinaryPath $binaryPath) {
            Show-NextSteps
        }
    }
    catch {
        Write-Err "Installation failed: $_"
        Write-Host ""
        Write-Host "If the problem persists, download manually from:" -ForegroundColor Gray
        Write-Host "  https://github.com/$GITHUB_REPO/releases/latest" -ForegroundColor Blue
        Write-Host ""
    }
}

Main
