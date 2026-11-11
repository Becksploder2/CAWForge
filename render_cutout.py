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


def studio_light(image, strength):
    """Non-generative softbox/key-fill treatment using the existing pixels."""
    if strength<=0:
        return image
    rgba=np.array(image.convert("RGBA"),dtype=np.uint8,copy=True)
    alpha=rgba[:,:,3].astype(np.float32)/255.0
    ys,xs=np.where(alpha>0.5)
    if len(xs)<100:
        return image
    h,w=alpha.shape
    x0,x1=xs.min(),xs.max()
    y0,y1=ys.min(),ys.max()
    sw=max(1,x1-x0)
    sh=max(1,y1-y0)
    yy,xx=np.mgrid[0:h,0:w].astype(np.float32)
    # Broad softbox from upper left, weaker fill from right.
    key=np.exp(-(((xx-(x0+0.20*sw))/(0.95*sw))**2+
                 ((yy-(y0+0.12*sh))/(1.05*sh))**2)*1.25)
    fill=np.exp(-(((xx-(x0+0.96*sw))/(1.20*sw))**2+
                  ((yy-(y0+0.48*sh))/(1.35*sh))**2)*1.6)
    # Local edge glow, capped so dark hair is not outlined artificially.
    interior=cv2.distanceTransform((alpha>0.5).astype(np.uint8),cv2.DIST_L2,3)
    rim=np.exp(-interior/12.0)*(alpha>0.5)
    rim*=np.clip((xx-x0)/sw,0,1)
    # Apply only luminance changes: hue, logos and tattoos remain unchanged.
    rgb=rgba[:,:,:3].copy()
    lab=cv2.cvtColor(rgb,cv2.COLOR_RGB2LAB)
    lum=lab[:,:,0].astype(np.float32)
    light=(key*12.0+fill*4.5+rim*5.0-3.0)*strength
    # Protect extreme highlights from clipping.
    light*=np.clip((255.0-lum)/65.0,0,1)
    lab[:,:,0]=np.uint8(np.clip(lum+light,0,255))
    rgba[:,:,:3]=cv2.cvtColor(lab,cv2.COLOR_LAB2RGB)
    return Image.fromarray(rgba,"RGBA")


def remove_blue_backdrop_leak(image):
    """Conservatively reject partially transparent blue-gray backdrop spill.
    Protect opaque clothing, dark hair, and fine edges."""
    rgba=np.array(image.convert("RGBA"),dtype=np.uint8,copy=True)
    alpha=rgba[:,:,3]
    hsv=cv2.cvtColor(rgba[:,:,:3],cv2.COLOR_RGB2HSV)
    h,w=alpha.shape
    yy=np.arange(h)[:,None]
    # Background leakage is blue-gray, semi-transparent, and below the hair.
    suspect=((hsv[:,:,0]>=95)&(hsv[:,:,0]<=124)&
             (hsv[:,:,1]>=42)&(hsv[:,:,1]<=135)&
             (hsv[:,:,2]<125)&(alpha>12)&(alpha<245)&
             (yy>int(h*0.34)))
    # Remove only substantial connected spill regions, not isolated pixels.
    count,labels,stats,_=cv2.connectedComponentsWithStats(suspect.astype(np.uint8),8)
    for idx in range(1,count):
        area=int(stats[idx,cv2.CC_STAT_AREA])
        if area>=60:
            rgba[:,:,3][labels==idx]=0
    return Image.fromarray(rgba,"RGBA")


def material_detail(image, strength):
    """Selective luminance finishing from existing pixels, without synthesis."""
    if strength<=0:
        return image
    rgba=np.array(image.convert("RGBA"),dtype=np.uint8,copy=True)
    rgb=rgba[:,:,:3]
    hsv=cv2.cvtColor(rgb,cv2.COLOR_RGB2HSV)
    lab=cv2.cvtColor(rgb,cv2.COLOR_RGB2LAB)
    lum=lab[:,:,0].astype(np.float32)
    # Broad skin color range; avoid dark tattoos and saturated gear.
    skin=((hsv[:,:,0]>=4)&(hsv[:,:,0]<=24)&
          (hsv[:,:,1]>=35)&(hsv[:,:,1]<=190)&(hsv[:,:,2]>=80))
    # Dark, low-saturation material includes hair; avoid global skin sharpening.
    dark=((hsv[:,:,2]<115)&(hsv[:,:,1]<105))
    # Colorful and bright gear gets stronger local detail.
    gear=((hsv[:,:,1]>115)&(hsv[:,:,2]>65)&(~skin))
    soft=cv2.GaussianBlur(lum,(0,0),1.15)
    detail=lum-soft
    gain=np.zeros_like(lum)
    gain[skin]=0.48
    gain[dark]=0.85
    gain[gear]=0.95
    # No invented pores: only amplify detail already present in source.
    lum=np.clip(lum+np.clip(detail,-12,12)*gain*strength,0,255)
    lab[:,:,0]=lum.astype(np.uint8)
    rgba[:,:,:3]=cv2.cvtColor(lab,cv2.COLOR_LAB2RGB)
    return Image.fromarray(rgba,"RGBA")


def main():
    p=argparse.ArgumentParser()
    p.add_argument("source")
    p.add_argument("destination")
    p.add_argument("--realism",type=int,default=0)
    p.add_argument("--studio",action="store_true",help="Apply non-generative studio lighting")
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
    output=remove_blue_backdrop_leak(output)
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
    if args.studio:
        output=studio_light(output,level)
        output=material_detail(output,level)
    if not args.no_crop:
        output=crop_to_subject(output)
    output=clear_hidden_background(output)
    target=Path(args.destination)
    target.parent.mkdir(parents=True,exist_ok=True)
    output.save(target,"PNG")
    print(str(target))

if __name__=="__main__":
    main()
