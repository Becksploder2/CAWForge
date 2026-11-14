# CAWForge face template calibration

The Face Texture Studio can now apply a **validated affine alignment** to a selected WWE 2K26 template when a calibrated `face_template_landmarks.json` file is installed next to the executable. It does **not** know the actual eye/nose/mouth positions of every WWE template automatically.

## File format

```json
{
  "source": [[250, 310], [570, 310], [410, 445], [325, 570], [495, 570]],
  "templates": {
    "F10": {
      "target": [[250, 310], [570, 310], [410, 445], [325, 570], [495, 570]]
    }
  }
}
```

**The coordinates above are a FORMAT EXAMPLE ONLY, NOT verified template coordinates. Do not install this example as a production calibration.** Source coordinates must describe the *generated texture's* left eye, right eye, nose, left mouth corner, and right mouth corner (in that order). Target coordinates must be measured at corresponding positions on each WWE face template. Source coordinates will vary between generated faces; using one fixed source configuration is only appropriate after the generation workflow consistently normalizes features to those positions.

When calibration is absent, CAWForge still generates the 820x820 texture but does **not** claim that it fits a template. When a calibration file exists but is invalid or lacks the selected template, generation reports an error instead of silently pretending alignment succeeded. This first-pass alignment uses a single affine transform; it does not provide fine-grained nonlinear UV deformation.

## Accuracy roadmap

1. Establish a stable, standardized source face coordinate system by measuring actual outputs.
2. Measure target landmarks for **each** in-game template; do not guess from screenshots.
3. Validate landmark positions on real CAWs, including Template 10.
4. Add dense piecewise warping for different face shapes, identity preservation and masked inpainting.
5. Keep original images and calibrated outputs separately and visually compare them before importing.

Stable Diffusion 1.5 is a general image model. It cannot guarantee identity preservation, neutral expressions, or accurate game UV placement. Those are independent quality checks.
