# Stable Diffusion in CAWForge (Windows test integration)

Run `Setup-StableDiffusion.ps1` in PowerShell from the extracted CAWForge folder. The script creates a private `.venv`, installs CUDA-enabled PyTorch and the Diffusers stack, and downloads Stable Diffusion 1.5 from Hugging Face on first use. This is a **large download**, not an offline bundle. Internet access is needed for setup. You must comply with the model's license and terms.

The Windows application invokes `.venv\\Scripts\\python.exe` when available, then `python` as fallback. Keep `sd_backend.py` beside the executable. Face Texture Local AI and Attire Pattern both use this backend. They do **not** require the older local-engine.exe or attire-engine.exe when SD is installed.

**Limitations:** This is an experimental image-to-image/text-to-image pipeline, not a trained WWE face-UV generator. Face results are resized to 820×820 but accurate UV alignment, identity, ear/hair removal, and logo continuity are **not guaranteed**. Attire results are flat concept images, **not** in-game UV atlases. Do not describe them as game-ready. A compatible NVIDIA CUDA GPU is required; smaller laptop GPUs may encounter out-of-memory errors.

If Windows blocks the setup script, use `powershell -ExecutionPolicy Bypass -File .\\Setup-StableDiffusion.ps1` after reviewing the script. This setup does not install anything into the WWE game directory.
