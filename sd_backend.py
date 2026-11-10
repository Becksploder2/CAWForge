"""CAWForge Stable Diffusion 1.5 backend. Requires torch, diffusers, transformers, accelerate, safetensors, Pillow.
Run with: python sd_backend.py --mode face --input photo.png --output face.png
or: python sd_backend.py --mode attire --piece top --prompt "navy gold honeycomb" --output top.png
The first run downloads the official Diffusers model; subsequent runs use its cache.
"""
import argparse
from pathlib import Path
from PIL import Image, ImageOps
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

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--check", action="store_true")
    ap.add_argument("--mode", choices=["face", "attire"])
    ap.add_argument("--input")
    ap.add_argument("--piece", choices=["top", "tights", "waistband"])
    ap.add_argument("--prompt", default="")
    ap.add_argument("--output", required=True)
    ap.add_argument("--strength", type=float, default=0.30)
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
    if args.mode == "face":
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
        result.resize((820, 820), Image.Resampling.LANCZOS).save(out, "PNG")
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
