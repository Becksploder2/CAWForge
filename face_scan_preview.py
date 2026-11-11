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
        src=cv2.copyMakeBorder(
            src,pad_top,pad_bottom,pad_left,pad_right,
            cv2.BORDER_REFLECT_101
        )
        x0+=pad_left
        x1+=pad_left
        y0+=pad_top
        y1+=pad_top
    crop=src[y0:y1,x0:x1]
    if crop.shape[0]!=side or crop.shape[1]!=side:
        raise RuntimeError("Internal face crop size mismatch")
    preview=cv2.resize(crop,(820,820),interpolation=cv2.INTER_LANCZOS4)
    target=Path(a.destination);target.parent.mkdir(parents=True,exist_ok=True)
    cv2.imwrite(str(target),preview)
    print("820x820 aligned face-scan PREVIEW saved. NOT game-ready UV: hair/ears and template calibration still require work.")
if __name__=="__main__":
    main()
