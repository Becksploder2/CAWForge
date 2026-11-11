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
    src=cv2.imread(a.source)
    if src is None: raise ValueError("Could not open source photo")
    h,w=src.shape[:2]
    if min(h,w)<240: raise ValueError("Use a photo at least 240 pixels on each side")
    cascade=cv2.CascadeClassifier(cv2.data.haarcascades+"haarcascade_frontalface_default.xml")
    if cascade.empty(): raise RuntimeError("OpenCV frontal-face detector unavailable")
    gray=cv2.cvtColor(src,cv2.COLOR_BGR2GRAY)
    faces=cascade.detectMultiScale(gray,scaleFactor=1.08,minNeighbors=6,minSize=(90,90))
    if len(faces)!=1:
        raise ValueError(f"Expected exactly one clear frontal face; detected {len(faces)}. Use a straight-on, well-lit single-person photo.")
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
