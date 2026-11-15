# Calibrated Seam Stitching — first implementation

The Windows app's **Export Seam Baseline** button now supports an optional calibrated seam map alongside the imported PNG.

For an imported file `attire.png`, put `attire_seams.json` in the same folder. The JSON format is:

```json
{
  "width": 1024,
  "height": 1024,
  "edges": [
    {
      "name": "top-left-side",
      "width": 4,
      "points": [
        {"ax": 100, "ay": 200, "bx": 800, "by": 200},
        {"ax": 105, "ay": 300, "bx": 795, "by": 300}
      ]
    }
  ]
}
```

**IMPORTANT: These numbers are illustrative, not WWE 2K26 UV coordinates. Do not use them as an actual mapping.**

- `width`/`height` must match the imported texture exactly.
- `ax,ay` and `bx,by` are corresponding **pixel coordinates** on the two sides of a single seam.
- Each edge needs at least two pairs; `width` is a 1–32 pixel blend radius.
- The engine interpolates paired samples, blends their source colors locally, and retains the original image's alpha. It skips fully transparent pixels.
- With no sidecar mapping, the existing baseline export remains unchanged and clearly says no calibration was applied.
- An invalid map produces an error without overwriting the original baseline.
- This is a **first implementation**, not a verified game-ready WWE UV mapper. No actual WWE 2K26 seam coordinates have yet been established, and there is no visual coordinate editor or automatic detection in the UI.
- The existing generic `tileSeam` graphic generator output is NOT UV-calibrated.

## Testing checklist
1. Import a PNG with a matching, deliberately measured sidecar map.
2. Export baseline and confirm both `_SEAM_BASELINE.png` and `_CALIBRATED_SEAMS.png` exist.
3. Confirm only local seam bands change; confirm transparent pixels stay transparent.
4. Verify in-game on a duplicate attire slot before using on final designs.
