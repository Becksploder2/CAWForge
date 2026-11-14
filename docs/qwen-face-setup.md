# CAWForge Qwen Face Texture — setup status and next steps

CAWForge's Face Texture Generate button invokes `qwen_backend.py` with two input images: the WWE 2K26 face UV template and the source face photograph. It is **not ready for use** until a working local ComfyUI Qwen-Image-Edit workflow is installed and configured.

## One-time setup
1. Install and start ComfyUI locally (default API address `http://127.0.0.1:8188`).
2. In ComfyUI, install an actual **Qwen-Image-Edit** workflow and compatible model files. Choose a model/workflow compatible with your hardware. The RTX 3050 laptop with 4 GB VRAM may not run Qwen locally at a usable speed, even with offloading. Do not assume Stable Diffusion's existing installer installs Qwen.
3. Ensure the workflow accepts **two images**: (a) the UV template and (b) the reference face photo, and contains a positive text prompt and SaveImage node. Export it in **API format**, not the normal ComfyUI UI JSON. Save as `qwen_face_api.json` beside CAWForge.exe.
4. Copy `qwen_face_config.example.json` to `qwen_face_config.json` beside CAWForge.exe. Open the exported API workflow JSON and replace the placeholder node IDs:
   - `template_node`: LoadImage node for the face UV template
   - `photo_node`: LoadImage node for the reference photo
   - `prompt_node`: node with the positive literal text prompt
   - `workflow`: `qwen_face_api.json`
   - `server`: `http://127.0.0.1:8188`
5. With ComfyUI running, use `python qwen_setup_check.py` in the CAWForge folder to check configuration. This does **not** download models or prove the model fits in VRAM.
6. In CAWForge, import the face photo and select the actual UV template, then press Generate. Inspect the resulting 820×820 PNG carefully for facial identity, seams, hair, ears and template alignment before using it in WWE 2K26.

## Troubleshooting
- **qwen_backend.py missing**: use the latest complete Windows test ZIP, not just the executable.
- **qwen_face_config.json missing**: copy and edit the example file.
- **ComfyUI not reachable**: start ComfyUI and check the API port.
- **Workflow missing / invalid node IDs**: export the API-format workflow and update the config.
- **Out of memory / hangs**: Qwen model may be too large for 4 GB VRAM; use a smaller compatible variant, offloading, or a different machine.
- **Generated image not game-ready**: Qwen is an image editor, not a calibrated UV solver. Inspect alignment and use calibrated template landmarks.

Render Creator continues to use its separate existing pipeline. No Qwen model weights or ComfyUI installation are bundled in the Windows ZIP.
