"""Pixel-preserving CAW foreground cutout using rembg/U2Net.
Usage: python render_cutout.py input.png output.png --realism 50
The model predicts only alpha; original RGB values are never regenerated.
"""
import argparse
from pathlib import Path
import numpy as np
from PIL import Image, ImageEnhance, ImageFilter
from rembg import remove, new_session

def main():
    p=argparse.ArgumentParser()
    p.add_argument("source")
    p.add_argument("destination")
    p.add_argument("--realism",type=int,default=0)
    args=p.parse_args()
    source=Image.open(args.source).convert("RGBA")
    w,h=source.size
    if w<64 or h<64: raise ValueError("Source image too small")
    a=np.asarray(source.getchannel("A"))
    if np.mean(a<250)>0.01:
        # Existing meaningful transparency must never be re-segmented.
        output=source
        print("Existing transparency detected; preserving original pixels")
    else:
        session=new_session("u2netp")
        cut=remove(source,session=session,only_mask=True,post_process_mask=True)
        mask=np.asarray(cut.convert("L"),dtype=np.uint8)
        opaque=np.mean(mask>=245)
        visible=np.mean(mask>16)
        if opaque<0.015 or visible<0.035 or visible>0.80:
            raise RuntimeError(f"Segmentation mask failed safety check: opaque={opaque:.3f}, visible={visible:.3f}; source unchanged")
        # Do not touch original color channels. Alpha alone is replaced.
        output=source.copy()
        output.putalpha(Image.fromarray(mask,"L"))
    # Realism slider is a conservative photographic finishing pass,
    # NOT a generative face/outfit transformation.
    level=max(0,min(100,args.realism))/100
    if level>0:
        alpha=output.getchannel("A")
        rgb=output.convert("RGB")
        rgb=ImageEnhance.Contrast(rgb).enhance(1+0.10*level)
        rgb=ImageEnhance.Color(rgb).enhance(1-0.07*level)
        rgb=ImageEnhance.Sharpness(rgb).enhance(1+0.35*level)
        if level>0.6:
            rgb=rgb.filter(ImageFilter.UnsharpMask(radius=1.0,percent=int(35*(level-0.6)/0.4),threshold=3))
        output=rgb.convert("RGBA")
        output.putalpha(alpha)
    target=Path(args.destination)
    target.parent.mkdir(parents=True,exist_ok=True)
    output.save(target,"PNG")
    print(str(target))

if __name__=="__main__":
    main()
