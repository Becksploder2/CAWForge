"""CAWForge Stable Diffusion 1.5 backend. Requires torch, diffusers, transformers, accelerate, safetensors, Pillow.
Run with: python sd_backend.py --mode face --input photo.png --output face.png
or: python sd_backend.py --mode attire --piece top --prompt "navy gold honeycomb" --output top.png
The first run downloads the official Diffusers model; subsequent runs use its cache.
"""
import argparse
import json
import numpy as np
from pathlib import Path
from PIL import Image, ImageOps
from collections import deque
import torch
from diffusers import StableDiffusionPipeline, StableDiffusionImg2ImgPipeline

MODEL = "stable-diffusion-v1-5/stable-diffusion-v1-5"

def load_pipeline(img2img):
    cls = StableDiffusionImg2ImgPipeline if img2img else StableDiffusionPipeline
    if not torch.cuda.is_available():
        raise RuntimeError("CUDA GPU unavailable; install a CUDA-enabled PyTorch build")
    pipe = cls.from_pretrained(MODEL, torch_dtype=torch.float16, variant="fp16", use_safetensors=True)
    pipe.enable_model_cpu_offload()
    pipe.enable_vae_slicing()
    return pipe

def fit_template(image, template_id, landmarks_file):
    """Warp source facial landmarks to calibrated template positions.

    JSON: {"templates":{"F10":{"target":[[x,y],...]}},
           "source":[[x,y],...]} with at least three corresponding
    points, all coordinates in 820x820 pixels. Missing calibration is an
    explicit error, never silently treated as a fitted result.
    """
    if not landmarks_file:
        raise RuntimeError("Template fitting requires --landmarks JSON with calibrated source and target points")
    with open(landmarks_file, encoding="utf-8") as f:
        spec = json.load(f)
    source = np.asarray(spec.get("source", []), dtype=np.float32)
    target = np.asarray(spec.get("templates", {}).get(template_id, {}).get("target", []), dtype=np.float32)
    if source.ndim != 2 or source.shape[1:] != (2,) or len(source) < 3 or source.shape != target.shape:
        raise RuntimeError(f"Template {template_id} has no valid corresponding landmark calibration")
    if not (np.isfinite(source).all() and np.isfinite(target).all()):
        raise RuntimeError("Landmarks contain nonfinite values")
    if np.any(source < 0) or np.any(source > 819) or np.any(target < 0) or np.any(target > 819):
        raise RuntimeError("Landmarks must be within the 820x820 canvas")
    # Least-squares affine mapping, source -> target, then invert for PIL.
    a = np.column_stack((source, np.ones(len(source), dtype=np.float32)))
    params, _, rank, _ = np.linalg.lstsq(a, target, rcond=None)
    if rank < 3:
        raise RuntimeError("Landmarks are collinear; cannot calibrate template")
    forward = np.eye(3)
    forward[:2, :] = params.T
    inv = np.linalg.inv(forward)
    residual = np.linalg.norm(a @ params - target, axis=1)
    if float(np.max(residual)) > 28:
        raise RuntimeError(f"Template landmarks cannot be aligned by a single affine transform (max error {float(np.max(residual)):.1f}px)")
    return image.transform((820, 820), Image.Transform.AFFINE,
        (inv[0,0], inv[0,1], inv[0,2], inv[1,0], inv[1,1], inv[1,2]),
        resample=Image.Resampling.BICUBIC)

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--check", action="store_true")
    ap.add_argument("--mode", choices=["face", "attire", "render"])
    ap.add_argument("--input")
    ap.add_argument("--piece", choices=["top", "tights", "waistband"])
    ap.add_argument("--prompt", default="")
    ap.add_argument("--output")
    ap.add_argument("--strength", type=float, default=0.30)
    ap.add_argument("--realism", type=int, default=65)
    ap.add_argument("--template", default="")
    ap.add_argument("--landmarks", default="")
    args = ap.parse_args()
    if args.check:
        print("CUDA available:", torch.cuda.is_available())
        if not torch.cuda.is_available():
            raise SystemExit("CUDA-enabled PyTorch or NVIDIA GPU unavailable")
        return
    if not args.mode or not args.output:
        ap.error("--mode and --output are required unless --check is used")
    out = Path(args.output)
    out.parent.mkdir(parents=True, exist_ok=True)
    if args.mode == "render":
        if not args.input:
            ap.error("--input required for render mode")
        original = Image.open(args.input).convert("RGBA")
        rgb = original.convert("RGB")
        # Conservative border-connected background removal; preserves the generated foreground.
        # For complex backdrops users should use the existing cutout tool instead.
        init = ImageOps.contain(rgb, (512, 768), method=Image.Resampling.LANCZOS)
        canvas = Image.new("RGB", (512, 768), (127, 127, 127))
        canvas.paste(init, ((512-init.width)//2, (768-init.height)//2))
        pipe = load_pipeline(True)
        strength = max(0.12, min(0.48, 0.12 + args.realism / 280))
        result = pipe(
            prompt="professional photorealistic full body studio promotional photograph of the same professional wrestler, exact outfit and accessories, natural skin, realistic fabric, neutral plain gray studio background, detailed facial structure",
            negative_prompt="different clothes, changed logos, extra fingers, duplicate limbs, cropped feet, cartoon, illustration, text, watermark",
            image=canvas, strength=strength, guidance_scale=6, num_inference_steps=28
        ).images[0]
        # Preserve original framing; generated image is opaque by design.
        # Do not falsely claim a transparent render.
        result.save(out, "PNG")
    elif args.mode == "face":
        if not args.input:
            ap.error("--input required for face mode")
        img = Image.open(args.input).convert("RGB")
        img = ImageOps.fit(img, (512, 512), method=Image.Resampling.LANCZOS)
        pipe = load_pipeline(True)
        prompt = ("flat frontal neutral expression human face texture for a 3D character UV map, "
                  "even lighting, realistic pores, natural skin, no hairstyle, no ears, "
                  "no portrait framing, seamless skin edges, " + args.prompt)
        result = pipe(prompt=prompt, negative_prompt="hair, ears, smile, teeth, portrait, shadows, frame, cartoon, text", image=img,
                      strength=max(0.05, min(args.strength, 0.8)), guidance_scale=6.5, num_inference_steps=25).images[0]
        result = result.resize((820, 820), Image.Resampling.LANCZOS)
        # Explicit, calibrated landmark fitting. No guessed WWE template coordinates.
        if args.template:
            result = fit_template(result, args.template, args.landmarks)
        result.save(out, "PNG")
    else:
        if not args.piece:
            ap.error("--piece required for attire mode")
        pipe = load_pipeline(False)
        prompt = (f"flat 2D seamless professional wrestling {args.piece} fabric graphic design, "
                  "orthographic texture artwork only, no garment or person, no model, "
                  "no text, no logos, no shadows, " + args.prompt)
        result = pipe(prompt=prompt, negative_prompt="person, body, clothing mockup, shirt, pants, mannequin, words, letters, watermark",
                      width=512, height=512, guidance_scale=7, num_inference_steps=25).images[0]
        result.save(out, "PNG")
    print(str(out))

if __name__ == "__main__":
    main()
