param([switch]$SkipModelDownload)
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

function Assert-Success([string]$Stage) {
  if ($LASTEXITCODE -ne 0) { throw "$Stage failed (exit code $LASTEXITCODE)" }
}

# Use a known-compatible Python instead of silently inheriting Python 3.14.
$pyLauncher = Get-Command py -ErrorAction SilentlyContinue
if (-not $pyLauncher) {
  throw 'Python Launcher (py.exe) missing. Install Python 3.11 (64-bit) from python.org, then rerun setup.'
}
& py -3.11 --version 2>$null
if ($LASTEXITCODE -ne 0) {
  Write-Host 'Python 3.11 is required. Trying Windows Package Manager (winget)...'
  if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    throw 'Python 3.11 not installed and winget unavailable. Install Python 3.11 (64-bit) from https://www.python.org/downloads/release/python-3119/'
  }
  & winget install --id Python.Python.3.11 --exact --source winget --accept-source-agreements --accept-package-agreements
  Assert-Success 'Python 3.11 installation'
  & py -3.11 --version
  Assert-Success 'Python 3.11 detection (reopen PowerShell if launcher registration is delayed)'
}

$envPath = Join-Path $PSScriptRoot '.venv'
$py = Join-Path $envPath 'Scripts/python.exe'
$recreate = $true
if (Test-Path $py) {
  $version = & $py -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")'
  if ($LASTEXITCODE -eq 0 -and $version -eq '3.11') { $recreate = $false }
}
if ($recreate) {
  if (Test-Path $envPath) {
    $backup = Join-Path $PSScriptRoot ('.venv-backup-' + (Get-Date -Format 'yyyyMMdd-HHmmss'))
    Move-Item $envPath $backup
    Write-Host "Old Python environment preserved at $backup"
  }
  & py -3.11 -m venv $envPath
  Assert-Success 'Virtual environment creation'
}
& $py -m pip install --upgrade pip
Assert-Success 'pip upgrade'
& $py -m pip install 'torch==2.5.1' 'torchvision==0.20.1' --index-url https://download.pytorch.org/whl/cu121
Assert-Success 'CUDA PyTorch installation'
& $py -m pip install -r sd_requirements.txt
Assert-Success 'Diffusers dependencies installation'
& $py sd_backend.py --check
Assert-Success 'GPU readiness check'
if (-not $SkipModelDownload) {
  & $py -c "from huggingface_hub import snapshot_download; snapshot_download('stable-diffusion-v1-5/stable-diffusion-v1-5', allow_patterns=['*.json','*.txt','*.safetensors','tokenizer/*','scheduler/*','feature_extractor/*'])"
  Assert-Success 'Stable Diffusion model download'
}
Write-Host 'CAWForge Stable Diffusion setup complete. You can now launch CAWForge.' -ForegroundColor Green
