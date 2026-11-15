# Render Creator — lifelike promotional render specification

## User goal
Make WWE 2K26 CAW renders more lifelike while preserving **exact character identity**, clothing, tattoos, hair, facial features, glasses, braces, proportions, and logos.

## Two explicit modes
1. **Faithful Cutout (existing):** remove backdrop, preserve original RGB. Label plain-background fallback as non-AI. Export transparent PNG.
2. **Lifelike Promotional Render (new target):** true image-to-image refinement, not a color/contrast filter. Must use a working, compatible image generation or image editing backend. If missing, show a clear actionable error and leave source unchanged.

## Generation constraints
- Use imported full-body WWE 2K26 screenshot as the authoritative character reference; no cropped limbs.
- Preserve face, hair, makeup, freckles, body proportions, piercings, tattoos, glasses, braces, every attire color/logo/material, boots, and accessories. Do not beautify or redesign.
- Improve physically plausible skin, pores, hair strand detail, fabric, stitching, highlights, shadows, and soft three-point studio lighting.
- Transparent alpha, no scenery, lettering, watermarks, or extra people.
- Save output separately; never overwrite screenshot or faithful cutout.
- Compare output with input before offering to replace any existing render; allow user to reject results.
- Do not claim AI mode is working until a real image-edit backend is bundled and validated.

## Hardware
Target NVIDIA GeForce RTX 3050 Laptop GPU, VRAM unknown. Prefer memory-efficient inference or optional cloud backend. Validate memory budget and preserve a non-AI cutout fallback.

## Acceptance tests
- Input: front-facing full-body CAW screenshot with hair and attire details.
- Output: full-body PNG, transparent background, preserved tattoos/logos and recognizable face.
- Error: missing model/helper yields clear message; no fabricated photorealistic render.
