param([switch]$SkipModelDownload, [string]$AIHome = "")
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
function Assert-Success([string]$Stage) { if ($LASTEXITCODE -ne 0) { throw "$Stage failed (exit code $LASTEXITCODE)" } }
function FreeGB([string]$Path) {
  $root = [System.IO.Path]::GetPathRoot([System.IO.Path]::GetFullPath($Path))
  $drive = [System.IO.DriveInfo]::new($root)
  if (-not $drive.IsReady) { throw "Drive $root is not ready" }
  return [math]::Round($drive.AvailableFreeSpace / 1GB, 1)
}
if (-not $AIHome) {
  $candidate = 'E:\CAWForge-AI'
  if (Test-Path 'E:\') { $AIHome = $candidate }
  else { $AIHome = Join-Path $env:LOCALAPPDATA 'CAWForge-AI' }
}
$AIHome = [System.IO.Path]::GetFullPath($AIHome)
Write-Host "AI installation directory: $AIHome"
$free = FreeGB $AIHome
if ($free -lt 20) { throw "Only $free GB free on AI drive. At least 20 GB is recommended. Rerun with -AIHome 'D:\CAWForge-AI' to choose another drive." }
foreach ($sub in @('','Cache','Models','Temp')) { New-Item -ItemType Directory -Force -Path (Join-Path $AIHome $sub) | Out-Null }
$env:TEMP = Join-Path $AIHome 'Temp'
$env:TMP = $env:TEMP
$env:PIP_CACHE_DIR = Join-Path $AIHome 'Cache'
$env:HF_HOME = Join-Path $AIHome 'Models'
$env:HF_HUB_CACHE = Join-Path $env:HF_HOME 'hub'
$env:TORCH_HOME = Join-Path $AIHome 'Models\torch'
$pyLauncher = Get-Command py -ErrorAction SilentlyContinue
if (-not $pyLauncher) { throw 'Python launcher missing. Install Python 3.11 (64-bit) from python.org.' }
$installed = & py -0p 2>$null | Out-String
if ($installed -notmatch '3\.11') {
  if (-not (Get-Command winget -ErrorAction SilentlyContinue)) { throw 'Python 3.11 is required. Install it from python.org and rerun.' }
  & winget install --id Python.Python.3.11 --exact --source winget --accept-source-agreements --accept-package-agreements
  Assert-Success 'Python 3.11 installation'
}
& py -3.11 --version
Assert-Success 'Python 3.11 detection (restart installer if launcher registration is delayed)'
$envPath = Join-Path $AIHome '.venv'
$py = Join-Path $envPath 'Scripts\python.exe'
$recreate = $true
if (Test-Path $py) {
  $version = & $py -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")'
  if ($LASTEXITCODE -eq 0 -and $version -eq '3.11') { $recreate = $false }
}
if ($recreate) {
  if (Test-Path $envPath) {
    $backup = Join-Path $AIHome ('.venv-backup-' + (Get-Date -Format 'yyyyMMdd-HHmmss'))
    Move-Item $envPath $backup
    Write-Host "Old environment backed up to $backup"
  }
  & py -3.11 -m venv $envPath
  Assert-Success 'Python environment creation'
}
& $py -m pip install --upgrade pip --no-cache-dir
Assert-Success 'pip upgrade'
& $py -m pip install --no-cache-dir 'torch==2.5.1' 'torchvision==0.20.1' --index-url https://download.pytorch.org/whl/cu121
Assert-Success 'CUDA PyTorch installation'
& $py -m pip install --no-cache-dir -r (Join-Path $PSScriptRoot 'sd_requirements.txt')
Assert-Success 'AI dependencies installation'
& $py (Join-Path $PSScriptRoot 'sd_backend.py') --check
Assert-Success 'GPU readiness check'
if (-not $SkipModelDownload) {
  & $py -c "from huggingface_hub import snapshot_download; snapshot_download('stable-diffusion-v1-5/stable-diffusion-v1-5', allow_patterns=['*.json','*.txt','*.safetensors','tokenizer/*','scheduler/*','feature_extractor/*'])"
  Assert-Success 'Stable Diffusion model download'
}
$config = @{ python = $py; ai_home = $AIHome; hf_home = $env:HF_HOME; temp = $env:TEMP; torch_home = $env:TORCH_HOME }
$config | ConvertTo-Json | Set-Content -Path (Join-Path $PSScriptRoot 'ai-config.json') -Encoding UTF8
Write-Host 'CAWForge AI setup complete. Configuration saved in ai-config.json.' -ForegroundColor Green
