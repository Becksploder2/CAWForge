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
  $configFile = Join-Path $PSScriptRoot 'ai-config.json'
  if (Test-Path $configFile) {
    try {
      $saved = Get-Content $configFile -Raw | ConvertFrom-Json
      if ($saved.ai_home -and (Test-Path $saved.ai_home)) { $AIHome = $saved.ai_home }
    } catch { Write-Host 'Existing AI config unreadable; selecting a default location.' -ForegroundColor Yellow }
  }
}
if (-not $AIHome) {
  $candidate = 'E:\CAWForge-AI'
  if (Test-Path 'E:\') { $AIHome = $candidate }
  else { $AIHome = Join-Path $env:LOCALAPPDATA 'CAWForge-AI' }
}
$AIHome = [System.IO.Path]::GetFullPath($AIHome)
Write-Host "AI installation directory: $AIHome"
$free = FreeGB $AIHome
if ($free -lt 5 -and -not (Test-Path (Join-Path $AIHome '.venv\Scripts\python.exe'))) { throw "Only $free GB free on AI drive. At least 5 GB is needed for a new environment (more for model downloads). Rerun with -AIHome 'D:\CAWForge-AI' to choose another drive." }
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
  $version = & $py -c 'import sys; print(sys.version_info.major,sys.version_info.minor,sep=chr(46))'
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
Write-Host '[1/6] Checking Python package installer...'
& $py -m pip --version
Assert-Success 'pip readiness'
& $py -c 'import torch; assert torch.cuda.is_available(); print(torch.__version__)'
if ($LASTEXITCODE -ne 0) {
  Write-Host '[2/6] Installing or repairing CUDA PyTorch...'
  & $py -m pip install --no-cache-dir 'torch==2.5.1' 'torchvision==0.20.1' --index-url https://download.pytorch.org/whl/cu121
  Assert-Success 'CUDA PyTorch installation'
} else { Write-Host 'Keeping existing working CUDA PyTorch.' }
Write-Host '[3/6] Installing or repairing generative AI libraries...'
& $py -m pip install -r (Join-Path $PSScriptRoot 'sd_requirements.txt')
Assert-Success 'AI dependencies installation'
Write-Host '[4/6] Installing or repairing face scan, cutout and render libraries...'
& $py -m pip install 'rembg==2.0.67' 'onnxruntime>=1.18,<2' 'numpy<2' 'opencv-python-headless>=4.8,<4.12' 'Pillow>=10,<13'
Assert-Success 'Background removal dependencies installation'
& $py -c 'import sys, numpy, onnxruntime, cv2, PIL, torch, diffusers, transformers, accelerate, safetensors; from rembg import remove, new_session; print(sys.executable)'
Assert-Success 'Full dependency import verification'
Write-Host '[5/6] Checking AI backend...'
& $py (Join-Path $PSScriptRoot 'sd_backend.py') --check
Assert-Success 'GPU readiness check'
Write-Host '[6/6] Checking model files...'
if (-not $SkipModelDownload) {
  @'
from huggingface_hub import snapshot_download
snapshot_download("stable-diffusion-v1-5/stable-diffusion-v1-5", allow_patterns=["*.json","*.txt","*.safetensors","tokenizer/*","scheduler/*","feature_extractor/*"])
'@ | & $py -
  Assert-Success 'Stable Diffusion model download'
}
$config = @{ python = $py; ai_home = $AIHome; hf_home = $env:HF_HOME; temp = $env:TEMP; torch_home = $env:TORCH_HOME }
[System.IO.File]::WriteAllText((Join-Path $PSScriptRoot 'ai-config.json'), ($config | ConvertTo-Json), [System.Text.UTF8Encoding]::new($false))
Write-Host 'CAWForge AI setup complete. All required Python modules verified. Configuration saved in ai-config.json.' -ForegroundColor Green
