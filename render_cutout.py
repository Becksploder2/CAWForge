"""Pixel-preserving CAW foreground cutout using rembg/U2Net.
Usage: python render_cutout.py input.png output.png --realism 50
The model predicts only alpha; original RGB values are never regenerated.
"""
import argparse
from pathlib import Path
import numpy as np
from PIL import Image
import cv2
from rembg import remove, new_session

def refine_mask(mask):
    """Remove disconnected debris while retaining the model's soft hair edges."""
    solid=(mask>=96).astype(np.uint8)
    count,labels,stats,_=cv2.connectedComponentsWithStats(solid,8)
    if count<=1:
        return mask
    largest=1+int(np.argmax(stats[1:,cv2.CC_STAT_AREA]))
    if stats[largest,cv2.CC_STAT_AREA]<0.01*mask.size:
        return mask
    # Include the original feathered pixels touching the main component.
    keep=(labels==largest).astype(np.uint8)
    keep=cv2.dilate(keep,np.ones((5,5),np.uint8),iterations=1)
    result=np.where(keep>0,mask,0).astype(np.uint8)
    # Very light feather only on the transition, not the interior.
    softened=cv2.GaussianBlur(result,(3,3),0.55)
    transition=(result>0)&(result<255)
    result[transition]=softened[transition]
    return result


def clear_hidden_background(image):
    """Zero RGB under fully transparent pixels without changing visible pixels."""
    pixels=np.array(image.convert("RGBA"),dtype=np.uint8,copy=True)
    pixels[pixels[:,:,3]==0,:3]=0
    return Image.fromarray(pixels,"RGBA")


def crop_to_subject(image):
    alpha=np.asarray(image.getchannel("A"))
    ys,xs=np.where(alpha>24)
    if len(xs)==0:
        return image
    left,right=int(xs.min()),int(xs.max())+1
    top,bottom=int(ys.min()),int(ys.max())+1
    subject_w,subject_h=right-left,bottom-top
    pad_x=max(12,int(subject_w*0.07))
    pad_y=max(12,int(subject_h*0.05))
    return image.crop((max(0,left-pad_x),max(0,top-pad_y),
                       min(image.width,right+pad_x),min(image.height,bottom+pad_y)))


def main():
    p=argparse.ArgumentParser()
    p.add_argument("source")
    p.add_argument("destination")
    p.add_argument("--realism",type=int,default=0)
    p.add_argument("--no-crop",action="store_true",help="Keep original canvas dimensions")
    args=p.parse_args()
    source=Image.open(args.source).convert("RGBA")
    w,h=source.size
    if w<64 or h<64: raise ValueError("Source image too small")
    a=np.asarray(source.getchannel("A"))
    if np.mean(a<250)>0.01:
        # Existing meaningful transparency must never be re-segmented.
        output=source.copy()
        print("Existing transparency detected; preserving original alpha")
    else:
        session=new_session("u2netp")
        cut=remove(source,session=session,only_mask=True,post_process_mask=False)
        mask=np.asarray(cut.convert("L"),dtype=np.uint8)
        opaque=np.mean(mask>=245)
        visible=np.mean(mask>16)
        if opaque<0.015 or visible<0.035 or visible>0.80:
            raise RuntimeError(f"Segmentation mask failed safety check: opaque={opaque:.3f}, visible={visible:.3f}; source unchanged")
        mask=refine_mask(mask)
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
    if not args.no_crop:
        output=crop_to_subject(output)
    output=clear_hidden_background(output)
    target=Path(args.destination)
    target.parent.mkdir(parents=True,exist_ok=True)
    output.save(target,"PNG")
    print(str(target))

if __name__=="__main__":
    main()
