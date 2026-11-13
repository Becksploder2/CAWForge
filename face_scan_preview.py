"""Face scan alignment preview. Does not claim to be a WWE UV texture."""
import argparse
from pathlib import Path
import cv2
import numpy as np
from PIL import Image

def main():
    p=argparse.ArgumentParser()
    p.add_argument("source")
    p.add_argument("destination")
    p.add_argument('--template',help='Optional existing 820x820 face UV layout')
    a=p.parse_args()
    # Flatten transparent renders onto a neutral backdrop for face detection.
    rgba=cv2.imread(a.source,cv2.IMREAD_UNCHANGED)
    if rgba is None: raise ValueError("Could not open source photo")
    if len(rgba.shape)==2:
        src=cv2.cvtColor(rgba,cv2.COLOR_GRAY2BGR)
    elif rgba.shape[2]==4:
        alpha=rgba[:,:,3:4].astype(np.float32)/255.0
        src=np.uint8(np.clip(rgba[:,:,:3].astype(np.float32)*alpha+235.0*(1-alpha),0,255))
    else:
        src=rgba[:,:,:3]
    h,w=src.shape[:2]
    if min(h,w)<240: raise ValueError("Use a photo at least 240 pixels on each side")
    cascade=cv2.CascadeClassifier(cv2.data.haarcascades+"haarcascade_frontalface_default.xml")
    if cascade.empty(): raise RuntimeError("OpenCV frontal-face detector unavailable")
    gray=cv2.cvtColor(src,cv2.COLOR_BGR2GRAY)
    faces=cascade.detectMultiScale(gray,scaleFactor=1.08,minNeighbors=6,minSize=(70,70))
    if len(faces)==0:
        # A second detector pass handles some makeup-heavy or rendered faces.
        enhanced=cv2.createCLAHE(clipLimit=2.0,tileGridSize=(8,8)).apply(gray)
        faces=cascade.detectMultiScale(enhanced,scaleFactor=1.05,minNeighbors=3,minSize=(60,60))
    if len(faces)>1:
        # Use the largest candidate, typically the foreground portrait.
        faces=sorted(faces,key=lambda f:int(f[2])*int(f[3]),reverse=True)
    if len(faces)==0:
        raise ValueError("No face detected automatically, even after enhanced detection. Try a closer head-and-shoulders crop. Manual face placement is not yet available.")
    x,y,fw,fh=map(int,faces[0])
    # Crop with generous forehead and chin room; never stretch the face.
    side=int(round(max(fw*1.95,fh*1.72)))
    cx=x+fw/2
    cy=y+fh*0.59
    x0=int(round(cx-side/2));y0=int(round(cy-side/2))
    x1=x0+side;y1=y0+side
    # Preserve the face geometry even when the source portrait is narrow.
    # Fill missing crop space with a soft reflection of the photo edges,
    # rather than rejecting otherwise valid front-facing photos.
    pad_left=max(0,-x0)
    pad_top=max(0,-y0)
    pad_right=max(0,x1-w)
    pad_bottom=max(0,y1-h)
    if any((pad_left,pad_top,pad_right,pad_bottom)):
        # Never reflect a portrait: that duplicates eyes, eyebrows and mouths.
        # Sample cheek skin and fill missing canvas with a matching skin tone.
        cheek_y=int(np.clip(y+fh*0.68,0,h-1))
        cheek_xs=[int(np.clip(x+fw*0.16,0,w-1)),int(np.clip(x+fw*0.84,0,w-1))]
        samples=[]
        for px in cheek_xs:
            patch=src[max(0,cheek_y-8):min(h,cheek_y+9),max(0,px-8):min(w,px+9)]
            if patch.size: samples.append(np.median(patch.reshape(-1,3),axis=0))
        skin=np.uint8(np.clip(np.median(samples,axis=0),0,255)) if samples else np.array([180,180,180],dtype=np.uint8)
        expanded=np.empty((h+pad_top+pad_bottom,w+pad_left+pad_right,3),dtype=np.uint8)
        expanded[:]=skin
        expanded[pad_top:pad_top+h,pad_left:pad_left+w]=src
        # Blend only within the newly added border, never across the face.
        mask=np.zeros((expanded.shape[0],expanded.shape[1]),dtype=np.uint8)
        mask[pad_top:pad_top+h,pad_left:pad_left+w]=255
        soft=cv2.GaussianBlur(mask,(0,0),sigmaX=max(2,side*0.018))
        weight=(soft.astype(np.float32)/255.0)[:,:,None]
        expanded=np.uint8(np.clip(expanded.astype(np.float32)*weight+skin[None,None,:]*(1-weight),0,255))
        src=expanded
        x0+=pad_left
        x1+=pad_left
        y0+=pad_top
        y1+=pad_top
    crop=src[y0:y1,x0:x1]
    if crop.shape[0]!=side or crop.shape[1]!=side:
        raise RuntimeError("Internal face crop size mismatch")
    preview=cv2.resize(crop,(820,820),interpolation=cv2.INTER_LANCZOS4)
    target=Path(a.destination);target.parent.mkdir(parents=True,exist_ok=True)
    if a.template:
        # The selected UV is an alignment reference, NOT a background.
        # Never leave the template owner's eyes, brows, nose or mouth visible.
        guide=cv2.imread(a.template,cv2.IMREAD_COLOR)
        if guide is None: raise ValueError("Cannot read selected UV template")
        # Build a new skin-only background sampled from the source cheeks.
        # This is still a preview; proper UV landmarks need calibration.
        cheek_y=int(np.clip(y+fh*0.66,0,h-1))
        skin_samples=[]
        for frac in (0.18,0.82):
            px=int(np.clip(x+fw*frac,0,w-1))
            patch=rgba[max(0,cheek_y-6):min(h,cheek_y+7),max(0,px-6):min(w,px+7)]
            if patch.ndim==3 and patch.shape[2]>=3 and patch.size:
                skin_samples.append(np.median(patch[:,:,:3].reshape(-1,patch.shape[2])[:,:3],axis=0))
        skin=np.uint8(np.clip(np.median(skin_samples,axis=0),0,255)) if skin_samples else np.array([175,175,175],dtype=np.uint8)
        base=np.empty((820,820,3),dtype=np.uint8)
        base[:]=skin
        # Source-only face patch: center it with broad, soft transitions.
        face_x=int(round((x+fw/2-x0)*820/side))
        face_y=int(round((y+fh/2-y0)*820/side))
        half_w=int(round(fw*0.67*820/side))
        half_h=int(round(fh*0.79*820/side))
        xx0=max(0,face_x-half_w);xx1=min(820,face_x+half_w)
        yy0=max(0,face_y-half_h);yy1=min(820,face_y+half_h)
        if xx1-xx0<20 or yy1-yy0<20: raise ValueError("Detected face too small for texture transfer")
        face_patch=preview[yy0:yy1,xx0:xx1]
        dst_w=560;dst_h=590
        face_patch=cv2.resize(face_patch,(dst_w,dst_h),interpolation=cv2.INTER_LANCZOS4)
        # Use a full rectangular skin region with a narrow feathered border.
        # An elliptical mask makes the face look like a circular sticker.
        feather_x=max(18,int(dst_w*0.09))
        feather_y=max(18,int(dst_h*0.09))
        dx=np.minimum(np.arange(dst_w),np.arange(dst_w)[::-1]).astype(np.float32)
        dy=np.minimum(np.arange(dst_h),np.arange(dst_h)[::-1]).astype(np.float32)
        wx=np.clip(dx/feather_x,0,1)
        wy=np.clip(dy/feather_y,0,1)
        wx=wx*wx*(3-2*wx)
        wy=wy*wy*(3-2*wy)
        mask=(wy[:,None]*wx[None,:])[:,:,None]
        top=(820-dst_h)//2;left=(820-dst_w)//2
        region=base[top:top+dst_h,left:left+dst_w].astype(np.float32)
        base[top:top+dst_h,left:left+dst_w]=np.uint8(np.clip(face_patch.astype(np.float32)*mask+region*(1-mask),0,255))
        preview=base
        print("Source-face preview created; UV template used as reference only, not copied.")
    cv2.imwrite(str(target),preview)
    print("820x820 aligned face-scan PREVIEW saved. NOT game-ready UV: hair/ears and template calibration still require work.")
if __name__=="__main__":
    main()
