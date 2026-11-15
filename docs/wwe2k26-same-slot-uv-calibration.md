# WWE 2K26 attire UV calibration — same-slot five-region test

Status: **in-game garment assignments confirmed; seam pixel pairs NOT confirmed**.

## Test conditions
User applied all five numbered calibration textures simultaneously to ONE WWE 2K26 attire slot, and supplied front/back screenshots of the cropped top and shorts. Treat the screenshots as one consistent test, not separate garments or sessions.

| Calibration color | Confirmed rendered garment region | Visible evidence |
| --- | --- | --- |
| Red | Cropped top front | Front-facing top has red numbered grid, including black center reference |
| Blue | Cropped top back | Back-facing top has blue numbered grid, including black center reference |
| Green | Shorts front | Front-facing briefs/shorts panel has green numbered grid |
| Purple | Shorts back | Rear panel has purple numbered grid |
| Gold | Waistband front AND back | Gold grid wraps across waist on both views |

## Observations for actual seam work
- Front top is red; rear top is blue. They meet at the left/right side of the top and near the armholes. The side boundary is not a straight 2D image column.
- Shorts front is green and shorts back is purple. Side seams wrap around hips; grid warps toward the leg openings and crotch.
- Gold waistband is visible on both sides, with a front black center mark. The black reference marks are fiducials, not evidence of UV pixel coordinates.
- Screenshots prove assignment and orientation, **not** a correspondence such as red pixel (x,y) equals blue pixel (u,v). Do not infer such pairs from garment colors alone.
- The current baseline export in CAWForge copies the original imported PNG unchanged. The procedural tileSeam/exportSeamPanels feature is NOT WWE 2K26 UV-aware. Do not describe either as calibrated auto-connect.

## Calibration implementation requirements
1. Store per-garment UV masks or polygons in source-image coordinates for the actual WWE attire texture layout.
2. Record paired seam samples with source (u,v), destination (u,v), orientation, and garment identity. Require at least two samples per connected edge; more for curved edges.
3. For each confirmed pair, transfer a narrow strip of pixels from one edge to its paired edge with an adjustable feather width. Respect source artwork and avoid copying across nonadjacent panels.
4. Preview both edges and export a separate stitched PNG. Keep original import unchanged; preserve alpha outside artwork.
5. Reject or mark uncalibrated regions explicitly rather than applying generic left-right tiling.

## Next minimal in-game test
Use the existing numbered grids; request only **close-up LEFT and RIGHT side views** of this same attire, ideally showing the top side seam and shorts hip seam. These will help read adjacent grid IDs. A front/back rerun is unnecessary. For pixel-accurate stitching, also obtain the exact flat UV source sheet used for this calibration and its dimensions.

## Provenance
User-provided WWE 2K26 screenshots in ChatGPT conversation, November 2026. No coordinates have been invented or approved.

## Additional side-view evidence — November 7, 2026
User supplied two in-game side-angle screenshots after the initial same-slot calibration.

- **Top side view**: red front panel meets blue back panel beneath the arm; grid lines bend around the torso and the side transition is not a straight source-image edge. The shorts in this view are plain gray, so this image should not be used to infer shorts seam coordinates.
- **Shorts side/rear three-quarter view**: purple rear panel extends around the hip, with a narrow visible green front-side strip near the leg opening; the gold waistband is not prominent in this screenshot. Top is also visible, but prioritize the shorts edge here.
- These are visual confirmations of **rendered garment adjacency only**. No pixel-exact UV pairs have been established. Do not build a blind edge-blending algorithm based on these images alone.
- Next input for precise implementation: original flat numbered calibration PNGs as imported into the game (with image dimensions) and, if possible, a close view of the other side for orientation.
- Keep user testing minimal: do not ask to repeat the successful front/back or side tests.

Image provenance: user-provided WWE 2K26 screenshot attachments in the same conversation; image files are not committed to this repository.
