# CAWForge attire generation: local engine integration

The native Windows app now checks for `attire-engine.exe` beside CAWForge.exe before trying the cloud image API. The worker interface is:

```text
attire-engine.exe --piece top --prompt "..." --output "C:\\...\\top_concept.png"
attire-engine.exe --piece tights --prompt "..." --output "C:\\...\\tights_concept.png"
attire-engine.exe --piece waistband --prompt "..." --output "C:\\...\\waistband_concept.png"
```

The helper must write a real PNG at least 512x512 for each part. CAWForge validates all three. This is a worker **integration contract**, not a bundled AI engine or model. Until an actual compatible inference worker and model are packaged, text-to-image generation remains unavailable without `OPENAI_API_KEY`.

The existing Import Existing Attire PNG > Export Seam Baseline path needs **no** AI model or API key. A matching `<name>_seams.json` alongside the imported PNG enables the opt-in calibrated seam stitcher, but no verified WWE 2K26 seam-pair mapping has yet been produced. Never claim automatic UV mapping is complete.

Target user hardware: NVIDIA GeForce RTX 3050 Laptop GPU; dedicated VRAM capacity still unknown. Choose a memory-efficient compatible inference stack and validate its license before packaging. User expects a standalone Windows EXE and performs in-game testing.
