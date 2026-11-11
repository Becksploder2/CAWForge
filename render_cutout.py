"""Pixel-preserving CAW foreground cutout using rembg/U2Net.
Usage: python render_cutout.py input.png output.png --realism 50
The model predicts only alpha; original RGB values are never regenerated.
"""
import argparse
from pathlib import Path
import numpy as np
from PIL import Image
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
    # Identity-safe photographic finishing: deterministic, local and non-generative.
    # Keep alpha and silhouette unchanged; only refine original RGB.
    level=max(0,min(100,args.realism))/100
    if level>0:
        alpha=output.getchannel("A")
        rgb=output.convert("RGB")
        # Work on luminance to avoid recoloring gear, tattoos or makeup.
        import cv2
        arr=np.asarray(rgb,dtype=np.uint8)
        lab=cv2.cvtColor(arr,cv2.COLOR_RGB2LAB)
        l,ac,bc=cv2.split(lab)
        # Clearly visible but identity-safe local luminance enhancement.
        # 50% should already show a difference; 100% is the strongest finish.
        clahe=cv2.createCLAHE(clipLimit=1.8+1.5*level,tileGridSize=(8,8))
        local=clahe.apply(l)
        l=cv2.addWeighted(l,1-0.88*level,local,0.88*level,0)
        refined=cv2.cvtColor(cv2.merge((l,ac,bc)),cv2.COLOR_LAB2RGB)
        # Mild detail enhancement over a bilateral-smoothed reference.
        # Never synthesize or move facial features, logos, or tattoos.
        smooth=cv2.bilateralFilter(refined,7,28,28)
        detail=cv2.addWeighted(refined,1+0.48*level,smooth,-0.48*level,0)
        # Blend original pixels back to reduce harsh halos and noise.
        finished=cv2.addWeighted(arr,1-0.92*level,detail,0.92*level,0)
        output=Image.fromarray(finished,"RGB").convert("RGBA")
        output.putalpha(alpha)
    target=Path(args.destination)
    target.parent.mkdir(parents=True,exist_ok=True)
    output.save(target,"PNG")
    print(str(target))

if __name__=="__main__":
    main()
