param([switch]$SkipModelDownload)
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
$python = 'python'
try { & $python --version | Out-Null } catch { throw 'Install Python 3.10 or 3.11 (64-bit), then rerun this setup.' }
if (!(Test-Path '.venv/Scripts/python.exe')) { & $python -m venv .venv }
$py = Join-Path $PSScriptRoot '.venv/Scripts/python.exe'
& $py -m pip install --upgrade pip
# CUDA 12.1 wheels; driver must support this runtime.
& $py -m pip install torch torchvision --index-url https://download.pytorch.org/whl/cu121
& $py -m pip install -r sd_requirements.txt
if (!$SkipModelDownload) {
  & $py -c "from huggingface_hub import snapshot_download; snapshot_download('stable-diffusion-v1-5/stable-diffusion-v1-5', allow_patterns=['*.json','*.txt','*.safetensors','tokenizer/*','scheduler/*','feature_extractor/*'])"
}
& $py sd_backend.py --check
if ($LASTEXITCODE -ne 0) { throw 'Stable Diffusion environment check failed. See output above.' }
Write-Host 'CAWForge Stable Diffusion environment prepared. Open CAWForge and select Local AI.' -ForegroundColor Green
