//go:build windows
package main
import (
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "strconv"
 "syscall"
 "unsafe"
 "runtime"
 "sync"
 "io"
 "image"
 "image/draw"
 "image/png"
 "image/color"
 _ "image/jpeg"

 _ "image/gif"
 "encoding/json"
 "math"
 "sort"
 "time"
 "bytes"
 "net/http"
 "context"
 "crypto/sha256"
 "archive/zip"
 "errors"
 "encoding/hex"
 "regexp"
)
var (u=syscall.NewLazyDLL("user32.dll"); k=syscall.NewLazyDLL("kernel32.dll"); c=syscall.NewLazyDLL("comdlg32.dll");
 reg=u.NewProc("RegisterClassExW"); create=u.NewProc("CreateWindowExW"); show=u.NewProc("ShowWindow"); update=u.NewProc("UpdateWindow"); getmsg=u.NewProc("GetMessageW"); trans=u.NewProc("TranslateMessage"); dispatch=u.NewProc("DispatchMessageW"); def=u.NewProc("DefWindowProcW"); post=u.NewProc("PostQuitMessage"); settext=u.NewProc("SetWindowTextW"); gettext=u.NewProc("GetWindowTextW"); send=u.NewProc("SendMessageW"); msgbox=u.NewProc("MessageBoxW"); instance=k.NewProc("GetModuleHandleW"); openfile=c.NewProc("GetOpenFileNameW"); savefile=c.NewProc("GetSaveFileNameW") )
func ptr(s string)*uint16{return syscall.StringToUTF16Ptr(s)}
type WNDCLASSEX struct {Size uint32; Style uint32; WndProc uintptr; ClsExtra int32; WndExtra int32; Instance uintptr; Icon uintptr; Cursor uintptr; Background uintptr; MenuName *uint16; ClassName *uint16; IconSm uintptr}
type MSG struct {Hwnd uintptr; Message uint32; Wparam uintptr; Lparam uintptr; Time uint32; PtX int32; PtY int32; Private uint32}
type OPENFILENAME struct {Size uint32; Owner uintptr; Instance uintptr; Filter *uint16; CustomFilter *uint16; MaxCustom uint32; FilterIndex uint32; File *uint16; MaxFile uint32; FileTitle *uint16; MaxFileTitle uint32; InitialDir *uint16; Title *uint16; Flags uint32; FileOffset uint16; FileExtension uint16; DefExt *uint16; CustData uintptr; Hook uintptr; TemplateName *uint16; Reserved uintptr; Reserved2 uint32; FlagsEx uint32}
const (WS_CHILD=0x40000000;WS_VISIBLE=0x10000000;WS_BORDER=0x800000;WS_OVERLAPPEDWINDOW=0x00CF0000;CBS_DROPDOWNLIST=3;CB_ADDSTRING=0x143;CB_SETCURSEL=0x14E;CB_GETCURSEL=0x147;WM_COMMAND=0x111;WM_DESTROY=2;WM_HSCROLL=0x114;TBM_SETRANGE=0x406;TBM_SETPOS=0x405;TBM_GETPOS=0x400;SW_SHOW=5;ES_MULTILINE=4;ES_AUTOVSCROLL=0x40;WS_VSCROLL=0x200000;ES_READONLY=0x800;WS_TABSTOP=0x10000)
var hwnd uintptr
var themeIndex=5
var themeBitmap uintptr
var themeBackdrop uintptr
var themeCustomPath string
var themePicker bool
const STM_SETIMAGE_THEME=0x0172
const IMAGE_BITMAP_THEME=0
const LR_LOADFROMFILE_THEME=0x0010
const HWND_BOTTOM=1
// Game-extracted artwork is loaded only if the user has installed it locally.
// Otherwise the existing clearly labeled inspired background is used.
func themePath() string {
 exe,_:=os.Executable();base:=filepath.Dir(exe)
 if themeIndex==6{return themeCustomPath}
 if themeIndex>=1&&themeIndex<=5 {
  game:=fmt.Sprintf("2k%d",21+themeIndex)
  for _,ext:=range []string{".bmp",".png",".jpg",".jpeg"} {
   candidate:=filepath.Join(base,"assets","backgrounds","originals",game+ext)
   if _,err:=os.Stat(candidate);err==nil{return candidate}
  }
 }
 return filepath.Join(base,"assets","backgrounds",fmt.Sprintf("theme_%d.bmp",themeIndex))
}
func convertThemeToBMP(path string)(string,error){
 ext:=strings.ToLower(filepath.Ext(path))
 if ext==".bmp" {return path,nil}
 f,err:=os.Open(path);if err!=nil{return "",err};defer f.Close()
 im,_,err:=image.Decode(f);if err!=nil{return "",err}
 const width=1100;const height=740
 row:=((width*3+3)/4)*4
 data:=make([]byte,54+row*height);copy(data[:2],[]byte{'B','M'})
 put:=func(i int,v uint32){data[i]=byte(v);data[i+1]=byte(v>>8);data[i+2]=byte(v>>16);data[i+3]=byte(v>>24)}
 put(2,uint32(len(data)));put(10,54);put(14,40);put(18,width);put(22,height);data[26]=1;data[28]=24
 b:=im.Bounds()
 for y:=0;y<height;y++{for x:=0;x<width;x++{
  sx:=b.Min.X+x*b.Dx()/width;sy:=b.Min.Y+(height-1-y)*b.Dy()/height
  r,g,bb,_:=im.At(sx,sy).RGBA();i:=54+y*row+x*3
  data[i]=byte(bb>>8);data[i+1]=byte(g>>8);data[i+2]=byte(r>>8)
 }}
 out:=filepath.Join(os.TempDir(),fmt.Sprintf("cawforge_theme_%d.bmp",time.Now().UnixNano()))
 return out,os.WriteFile(out,data,0644)
}
var themeConvertedPath string
func loadTheme(){
 if themeBackdrop==0{saveTheme();return}
 path:=themePath();if path==""{return}
 converted,err:=convertThemeToBMP(path)
 if err!=nil{alert("Could not read background: "+err.Error());return}
 bmp,_,_:=u.NewProc("LoadImageW").Call(0,uintptr(unsafe.Pointer(ptr(converted))),IMAGE_BITMAP_THEME,0,0,LR_LOADFROMFILE_THEME)
 if bmp==0{if converted!=path{os.Remove(converted)};alert("Cannot load background: "+path);return}
 send.Call(themeBackdrop,STM_SETIMAGE_THEME,IMAGE_BITMAP_THEME,bmp)
 if themeBitmap!=0{syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteObject").Call(themeBitmap)}
 if themeConvertedPath!=""{os.Remove(themeConvertedPath);themeConvertedPath=""}
 if converted!=path{themeConvertedPath=converted}
 themeBitmap=bmp
 u.NewProc("SetWindowPos").Call(themeBackdrop,HWND_BOTTOM,0,0,0,0,0x0001|0x0002|0x0010)
 saveTheme()
}
func saveTheme(){exe,_:=os.Executable();b,_:=json.Marshal(map[string]interface{}{"theme":themeIndex,"custom":themeCustomPath});_ = os.WriteFile(filepath.Join(filepath.Dir(exe),"appearance.json"),b,0644)}
func restoreTheme(){exe,_:=os.Executable();b,e:=os.ReadFile(filepath.Join(filepath.Dir(exe),"appearance.json"));if e!=nil{return};var v struct{Theme int `json:"theme"`;Custom string `json:"custom"`};if json.Unmarshal(b,&v)==nil&&v.Theme>=0&&v.Theme<=6{themeIndex=v.Theme;themeCustomPath=v.Custom}}

var pickerBusy bool
var pickerResult string
var pickerErr string
var pickerMu sync.Mutex
const WM_PICKER_DONE=0x8001
const WM_DROPFILES=0x233
var shell=syscall.NewLazyDLL("shell32.dll")
var dragAccept=shell.NewProc("DragAcceptFiles")
var dragQuery=shell.NewProc("DragQueryFileW")
var dragFinish=shell.NewProc("DragFinish")
var postMessage=u.NewProc("PostMessageW")
var ctrls=map[int]uintptr{}
var gender="F"
var genderMode="Auto"
var template=10
var active="Head"
var imagePath=""
var templateLibraryDir string
const WM_CAL_DONE=0x8002
const WM_MATCH_DONE=0x8003
const WM_TIMER=0x0113
const MORPH_PREVIEW_TIMER=901
var morphResult map[string]int
var matchMessage string
var matchID string
var matchMu sync.Mutex
var matchRunning bool
var matchQueued bool
var faceTextureStatus string
var faceTexturePreviewPath string
var previewBitmap uintptr
var previewBmpPath string
var textureBusy bool
var currentTab=0
var graphicBusy bool
var graphicStatus string
const WM_GRAPHIC_DONE=0x8005
const WM_AI_DONE=0x8006
var aiBusy bool
var aiResult string
var aiOutput string
const WM_TEXTURE_DONE=0x8004
const WM_TEXTURE_PREVIEW_READY=0x8008
const WM_EXPORT_DONE=0x8009
var exportStatus string
var texturePreviewResult struct{path string;err string}
const SS_BITMAP=0x0000000E
const STM_SETIMAGE=0x0172
const IMAGE_BITMAP=0
const LR_LOADFROMFILE=0x0010
type Fingerprint struct { ID string `json:"id"`; File string `json:"file"`; Features []float64 `json:"features"`; Size int64 `json:"size,omitempty"`; Modified int64 `json:"modified,omitempty"` }
type Index struct { Version int `json:"version"`; Entries []Fingerprint `json:"entries"` }
var index Index
var calibrationMessage string
var calMu sync.Mutex
var calibrationRunning bool
func feature(path string)([]float64,error){
 f,err:=os.Open(path);if err!=nil{return nil,err};defer f.Close()
 im,_,err:=image.Decode(f);if err!=nil{return nil,err}
 b:=im.Bounds();if b.Dx()<32||b.Dy()<32{return nil,fmt.Errorf("image too small")}
 // Center crop is only a coarse image descriptor, NOT facial landmark detection.
 side:=b.Dx();if b.Dy()<side{side=b.Dy()};x0:=b.Min.X+(b.Dx()-side)/2;y0:=b.Min.Y+(b.Dy()-side)/2
 out:=make([]float64,256);var sum float64
 for y:=0;y<16;y++{for x:=0;x<16;x++{
   var v float64
   for dy:=0;dy<3;dy++{for dx:=0;dx<3;dx++{
    xx:=x0+(x*3+dx)*side/48;yy:=y0+(y*3+dy)*side/48
    r,g,bb,_:=im.At(xx,yy).RGBA();v+=0.2126*float64(r)+0.7152*float64(g)+0.0722*float64(bb)
   }}
   out[y*16+x]=v/9;sum+=v/9
 }}
 mean:=sum/256;var variance float64
 for i:=range out{out[i]-=mean;variance+=out[i]*out[i]}
 sd:=math.Sqrt(variance/256);if sd<1{sd=1};for i:=range out{out[i]/=sd}
 return out,nil
}
func scanIndex()(Index,string){
 next:=Index{Version:2,Entries:[]Fingerprint{}};skipped:=0;reused:=0
 old:=Index{}
 if raw,err:=os.ReadFile(filepath.Join(templateLibraryDir,"calibration_index.json"));err==nil{_ = json.Unmarshal(raw,&old)}
 cache:=map[string]Fingerprint{}
 for _,entry:=range old.Entries{cache[entry.ID]=entry}
 for _,g:=range []string{"M","F"}{for n:=1;n<=21;n++{if g=="M"&&n==21{continue}
  id:=fmt.Sprintf("%s%d",g,n)
  for _,ext:=range []string{".png",".jpg",".jpeg",".gif"}{
   path:=filepath.Join(templateLibraryDir,id+ext)
   if _,err:=os.Stat(path);err!=nil{
    folder:="Male";if g=="F"{folder="Female"}
    path=filepath.Join(templateLibraryDir,folder,id+ext)
    if _,err:=os.Stat(path);err!=nil{continue}
   }
   stat,err:=os.Stat(path);if err!=nil{skipped++;continue}
   if previous,ok:=cache[id];ok && previous.File==path && previous.Size==stat.Size() && previous.Modified==stat.ModTime().UnixNano() && len(previous.Features)==256{
    next.Entries=append(next.Entries,previous);reused++;break
   }
   feat,err:=feature(path);if err!=nil{skipped++;continue}
   next.Entries=append(next.Entries,Fingerprint{ID:id,File:path,Features:feat,Size:stat.Size(),Modified:stat.ModTime().UnixNano()});break
  }
 }}
 raw,_:=json.MarshalIndent(next,"","  ");_ = os.WriteFile(filepath.Join(templateLibraryDir,"calibration_index.json"),raw,0644)
 if len(next.Entries)==0{return next,"Library is empty. Add M1-M20 or F1-F20 reference screenshots; no calibration performed."};return next,fmt.Sprintf("Ready: %d/40 templates (%d loaded from cache, %d unreadable).",len(next.Entries),reused,skipped)
}
func beginCalibration(){
 if calibrationRunning{return};calibrationRunning=true;set(41,"Indexing available M1-M20 and F1-F20 screenshots...")
 go func(){next,msg:=scanIndex();calMu.Lock();index=next;calibrationMessage=msg;calMu.Unlock();postMessage.Call(hwnd,WM_CAL_DONE,0,0)}()
}
func matchPhoto(){
 if imagePath==""{set(41,"Import a reference photo first.");return}
 if matchRunning{matchQueued=true;return}
 calMu.Lock();entries:=append([]Fingerprint(nil),index.Entries...);calMu.Unlock()
 count:=0;for _,e:=range entries{if strings.HasPrefix(e.ID,gender){count++}}
 if count==0{set(41,"No verified "+gender+" templates installed. Add reference images to assets/templates; matching unavailable.");return}
 matchRunning=true;set(41,"Comparing photo to "+strconv.Itoa(count)+" installed references in background...")
 photo:=imagePath;selectedGender:=gender;started:=time.Now()
 go func(){
  feat,err:=feature(photo);message:="";best:=""
  if err!=nil{message="Cannot analyze image: "+err.Error()}else{
   type candidate struct{id string;score float64};var matches []candidate
   for _,e:=range entries{if !strings.HasPrefix(e.ID,selectedGender)||len(e.Features)!=len(feat){continue};var sum float64;for i:=range feat{d:=feat[i]-e.Features[i];sum+=d*d};matches=append(matches,candidate{e.ID,sum})}
   sort.Slice(matches,func(i,j int)bool{return matches[i].score<matches[j].score})
   if len(matches)==0{message="No readable template references for selected section."}else{best=matches[0].id;var labels []string;for i:=0;i<len(matches)&&i<3;i++{labels=append(labels,matches[i].id)};message="Visual candidates: "+strings.Join(labels,", ")+". Comparison "+time.Since(started).Round(time.Millisecond).String()+". Experimental, not landmark matching."}
  }
  var estimated map[string]int
  if best!=""{for _,e:=range entries{if e.ID==best{estimated=suggestedMorph(feat,e.Features);break}}}
  matchMu.Lock();matchMessage=message;matchID=best;morphResult=estimated;matchMu.Unlock();postMessage.Call(hwnd,WM_MATCH_DONE,0,0)
 }()
}


func clampSlider(v float64)int {if v<0{return 0};if v>100{return 100};return int(math.Round(v))}
func suggestedMorph(photo,reference []float64)map[string]int{
 out:=map[string]int{}
 // Each setting derives from the actual photo/template luminance-profile
 // differences. These are heuristic estimates, not calibrated game sliders.
 for _,group:=range groups {for _,name:=range group.items {
  key:=group.name+" / "+name
  hash:=uint32(2166136261)
  for _,ch:=range key {hash=(hash^uint32(ch))*16777619}
  region:=int(hash%uint32(len(photo)-18))
  delta:=0.0
  for j:=0;j<18;j++ {delta+=(photo[region+j]-reference[region+j])/18}
  out[key]=clampSlider(50+delta*13)
 }}
 return out
}
func generateTexture(photoPath,outPath string)error{
 f,err:=os.Open(photoPath);if err!=nil{return err};defer f.Close()
 im,_,err:=image.Decode(f);if err!=nil{return err}
 b:=im.Bounds();w,h:=b.Dx(),b.Dy()
 if w<100||h<100{return fmt.Errorf("photo too small")}
 // Face-only elliptical matte: discard the original photograph background.
 // This is a centered-photo fallback, not a landmark/segmentation model.
 cropW:=int(float64(w)*0.62);cropH:=int(float64(h)*0.78)
 if cropW>cropH*3/4{cropW=cropH*3/4}
 if cropW<1{cropW=1};if cropH>h{cropH=h}
 cx:=b.Min.X+w/2;cy:=b.Min.Y+h/2
 x0:=cx-cropW/2;y0:=cy-cropH/2
 if x0<b.Min.X{x0=b.Min.X};if y0<b.Min.Y{y0=b.Min.Y}
 if x0+cropW>b.Max.X{x0=b.Max.X-cropW};if y0+cropH>b.Max.Y{y0=b.Max.Y-cropH}
 // Sample central cheek regions to create a seamless surrounding skin field.
 sample:=[]image.Point{{cx-cropW/6,cy},{cx+cropW/6,cy},{cx,cy+cropH/5}}
 var sr,sg,sb uint64
 for _,p:=range sample{r,g,bb,_:=im.At(p.X,p.Y).RGBA();sr+=uint64(r>>8);sg+=uint64(g>>8);sb+=uint64(bb>>8)}
 bg:=color.RGBA{uint8(sr/3),uint8(sg/3),uint8(sb/3),255}
 dst:=image.NewRGBA(image.Rect(0,0,820,820))
 for y:=0;y<820;y++{for x:=0;x<820;x++{
  // Oval excludes outer photographic backdrop; soft edges blend into skin.
  fx:=(float64(x)-106)/608;fy:=(float64(y)-25)/770
  nx:=(fx-0.5)/0.5;ny:=(fy-0.5)/0.5
  radius:=math.Sqrt(nx*nx+ny*ny)
  alpha:=math.Max(0,math.Min(1,(0.88-radius)/0.16))
  if fx<0||fx>=1||fy<0||fy>=1||alpha<=0{dst.SetRGBA(x,y,bg);continue}
  sx:=x0+int(fx*float64(cropW));sy:=y0+int(fy*float64(cropH))
  r,g,bb,_:=im.At(sx,sy).RGBA()
  dst.SetRGBA(x,y,color.RGBA{uint8(alpha*float64(r>>8)+(1-alpha)*float64(bg.R)),uint8(alpha*float64(g>>8)+(1-alpha)*float64(bg.G)),uint8(alpha*float64(bb>>8)+(1-alpha)*float64(bg.B)),255})
 }}
 o,err:=os.Create(outPath);if err!=nil{return err};defer o.Close()
 return png.Encode(o,dst)
}

const faceTexturePrompt = `Create an 820x820 photorealistic WWE 2K26 face texture from the imported reference photograph. Preserve the person's real facial proportions, eye placement, nose shape, lips, makeup, natural skin tone, pores and recognizable details. Perfectly frontal, symmetrical alignment, neutral expression, even neutral studio lighting. No hair, no ears, no glasses, no accessories, no dark jawline or chin shadow, no harsh face boundary, no text or background objects. Surround the face with seamlessly blended matching skin texture, maintaining natural realistic pores. Align eyes, nose, mouth, forehead and jaw to the selected in-game face template; prioritize accurate template placement and smooth transitions over stylization. Do not invent or change the person's facial features.`
func writeFacePrompt(path string)error{
 folder:=filepath.Dir(path)
 return os.WriteFile(filepath.Join(folder,"WWE_Face_Texture_Generation_Prompt.txt"),[]byte(faceTexturePrompt+"\r\n"),0644)
}
// Windows STATIC image preview uses an RGB BMP generated from the exported PNG.
// BMP is a temporary preview file; the user-facing output remains PNG.
func writePreviewBMP(path string) (string,error){
 f,err:=os.Open(path);if err!=nil{return "",err};defer f.Close()
 im,_,err:=image.Decode(f);if err!=nil{return "",err}
 const side=320
 row:=((side*3+3)/4)*4
 data:=make([]byte,54+row*side)
 copy(data[:2],[]byte{'B','M'})
 put32:=func(i int,v uint32){data[i]=byte(v);data[i+1]=byte(v>>8);data[i+2]=byte(v>>16);data[i+3]=byte(v>>24)}
 put32(2,uint32(len(data)));put32(10,54);put32(14,40);put32(18,side);put32(22,side)
 data[26]=1;data[28]=24
 b:=im.Bounds()
 for y:=0;y<side;y++{for x:=0;x<side;x++{
  sx:=b.Min.X+x*b.Dx()/side;sy:=b.Min.Y+(side-1-y)*b.Dy()/side
  r,g,bb,_:=im.At(sx,sy).RGBA()
  i:=54+y*row+x*3;data[i]=byte(bb>>8);data[i+1]=byte(g>>8);data[i+2]=byte(r>>8)
 }}
 out:=filepath.Join(os.TempDir(),"wwe_morph_preview_"+strconv.FormatInt(time.Now().UnixNano(),10)+".bmp")
 return out,os.WriteFile(out,data,0644)
}
func showTexturePreview(path string){
 // Convert image off the UI thread; only Win32 bitmap assignment stays on UI.
 go func(){
  bmpPath,err:=writePreviewBMP(path)
  matchMu.Lock()
  texturePreviewResult.path=bmpPath
  texturePreviewResult.err=""
  if err!=nil{texturePreviewResult.err=err.Error()}
  matchMu.Unlock()
  postMessage.Call(hwnd,WM_TEXTURE_PREVIEW_READY,0,0)
 }()
}
func applyTexturePreview(){
 matchMu.Lock()
 bmpPath,errText:=texturePreviewResult.path,texturePreviewResult.err
 matchMu.Unlock()
 if errText!=""{set(203,"Texture saved, but preview failed: "+errText);return}
 load:=u.NewProc("LoadImageW")
 bmp,_,_:=load.Call(0,uintptr(unsafe.Pointer(ptr(bmpPath))),IMAGE_BITMAP,0,0,LR_LOADFROMFILE)
 if bmp==0{set(203,"Texture saved, but Windows could not load the preview.");os.Remove(bmpPath);return}
 send.Call(ctrls[205],STM_SETIMAGE,IMAGE_BITMAP,bmp)
 if previewBitmap!=0{syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteObject").Call(previewBitmap)}
 if previewBmpPath!=""{os.Remove(previewBmpPath)}
 previewBitmap=bmp;previewBmpPath=bmpPath
}


func runAITexture(){
 if aiBusy{return}
 if imagePath==""{set(215,"Import a reference photo before running local AI.");return}
 exe,err:=os.Executable();if err!=nil{set(215,"Cannot locate application: "+err.Error());return}
 dir:=filepath.Dir(exe)
 workflow:=strings.TrimSpace(txt(211))
 if workflow==""{workflow=filepath.Join(dir,"workflow_api.json")}
 tmpl:=strings.TrimSpace(txt(23))
 if tmpl==""{tmpl=gender+strconv.Itoa(template)}
 photo:=imagePath
 aiBusy=true
 set(215,"Checking local AI dependencies in background...")
 go func(){
  var result,outputPath string
  engine:=filepath.Join(dir,"local-engine.exe")
  if _,err:=os.Stat(engine);err!=nil{
   result="Local AI is unavailable: local-engine.exe is not installed. Standard face texture generation remains available."
  }else if _,err:=os.Stat(workflow);err!=nil{
   result="Local AI workflow not found. Standard face texture generation remains available."
  }else{
   home,_:=os.UserHomeDir()
   if home==""{home=os.TempDir()}
   outdir:=filepath.Join(home,"Documents","CAWForge","FaceTextures")
   if err:=os.MkdirAll(outdir,0755);err!=nil{
    result="Cannot create AI output folder: "+err.Error()
   }else{
    out:=filepath.Join(outdir,fmt.Sprintf("WWE_Face_%s_%d.png",tmpl,time.Now().UnixNano()))
    ctx,cancel:=context.WithTimeout(context.Background(),3*time.Minute)
    defer cancel()
    cmd:=exec.CommandContext(ctx,engine,photo,tmpl,workflow,out)
    output,err:=cmd.CombinedOutput()
    if ctx.Err()==context.DeadlineExceeded{
     result="Local AI timed out after 3 minutes. The worker was stopped; the main window remains usable."
    }else if err!=nil{
     detail:=string(output);if len(detail)>600{detail=detail[:600]}
     result="Local AI failed: "+err.Error()+" "+detail
    }else{
     result="Saved 820 x 820: "+out
     outputPath=out
    }
   }
  }
  matchMu.Lock();aiResult=result;aiOutput=outputPath;matchMu.Unlock()
  postMessage.Call(hwnd,WM_AI_DONE,0,0)
 }()
}
func minInt(a,b int)int{if a<b{return a};return b}

func traceTexture(event string){
 dir:=filepath.Join(os.TempDir(),"CAWForge-Diagnostics")
 if err:=os.MkdirAll(dir,0755);err!=nil{return}
 f,err:=os.OpenFile(filepath.Join(dir,"texture-events.log"),os.O_CREATE|os.O_WRONLY|os.O_APPEND,0644)
 if err!=nil{return}
 defer f.Close()
 fmt.Fprintf(f,"%s %s\n",time.Now().Format(time.RFC3339Nano),event)
}
func startTexture(){
 // Avoid synchronous file logging on the UI thread.
 if imagePath==""{alert("Import a reference photograph first.");return}
 if textureBusy{return}
 textureBusy=true
 set(203,"Generating 820 x 820 texture in background...")
 photo:=imagePath
 go func(){
  home,_:=os.UserHomeDir()
  if home==""{home=os.TempDir()}
  dir:=filepath.Join(home,"Documents","CAWForge","FaceTextures")
  err:=os.MkdirAll(dir,0755)
  path:=filepath.Join(dir,strings.TrimSuffix(filepath.Base(photo),filepath.Ext(photo))+"_face_texture_820.png")
  traceTexture("Texture generation started")
  if err==nil{err=generateTexture(photo,path)}
  if err!=nil{traceTexture("Texture error: "+err.Error())}else{traceTexture("Texture generation completed")}
  if err==nil{_ = writeFacePrompt(path)}
  matchMu.Lock()
  if err!=nil{faceTextureStatus="Texture failed: "+err.Error();faceTexturePreviewPath=""}else{faceTextureStatus="Saved: "+path;faceTexturePreviewPath=path}
  matchMu.Unlock()
  traceTexture("Posting texture completion event")
  postMessage.Call(hwnd,WM_TEXTURE_DONE,0,0)
 }()
}

func proceduralGraphic(kind,description string) (string,error) {
 desc:=strings.ToLower(description)
 width,height:=1024,1024
 if kind=="Entrance background"{width,height=1280,720}
 if kind=="Wrestling sign"{width,height=1024,512}
 img:=image.NewRGBA(image.Rect(0,0,width,height))
 // Optional transparent base for wearable graphics and logos.
 transparent:=kind=="Attire pattern"||kind=="Wrestler logo"
 primary:=color.RGBA{205,45,113,255}
 secondary:=color.RGBA{255,206,230,255}
 switch {
 case strings.Contains(desc,"burnt orange")||strings.Contains(desc,"texas"):primary=color.RGBA{191,87,0,255};secondary=color.RGBA{255,255,255,255}
 case strings.Contains(desc,"purple"):primary=color.RGBA{101,38,159,255};secondary=color.RGBA{220,194,255,255}
 case strings.Contains(desc,"blue"):primary=color.RGBA{29,81,193,255};secondary=color.RGBA{230,240,255,255}
 case strings.Contains(desc,"green"):primary=color.RGBA{30,136,91,255};secondary=color.RGBA{227,255,234,255}
 case strings.Contains(desc,"red"):primary=color.RGBA{180,28,49,255};secondary=color.RGBA{255,230,235,255}
 case strings.Contains(desc,"black"):primary=color.RGBA{24,24,29,255};secondary=color.RGBA{198,198,205,255}
 }
 plaid:=strings.Contains(desc,"plaid")||strings.Contains(desc,"tartan")
 stripes:=strings.Contains(desc,"stripe")||strings.Contains(desc,"pinstripe")
 checker:=strings.Contains(desc,"check")||strings.Contains(desc,"checker")
 star:=strings.Contains(desc,"star")
 if !plaid&&!stripes&&!checker&&!star {stripes=true}
 for y:=0;y<height;y++{for x:=0;x<width;x++{
  v:=primary
  if !transparent{
   if (x+y)/90%2==0{v=secondary}
  }
  if plaid{
   if x%160<27||y%160<27{v=secondary}
   if x%160<7||y%160<7{v=primary}
  }else if stripes{
   if (x+y/5)%180<30{v=secondary}
  }else if checker{
   if (x/95+y/95)%2==0{v=secondary}
  }else if star{
   dx:=float64((x%220)-110);dy:=float64((y%220)-110)
   a:=math.Atan2(dy,dx);r:=math.Hypot(dx,dy)
   limit:=42+22*math.Cos(5*a)
   if r<limit{v=secondary}
  }
  if transparent{
   // Transparent outside a central design field for use as an overlay.
   margin:=width/20
   if x<margin||x>=width-margin||y<margin||y>=height-margin {v.A=0}
  }
  img.SetRGBA(x,y,v)
 }}
 exe,_:=os.Executable()
 dir:=filepath.Join(filepath.Dir(exe),"WWE_Morph_Exports")
 if err:=os.MkdirAll(dir,0755);err!=nil{return "",err}
 safe:=strings.ToLower(strings.ReplaceAll(kind," ","_"))
 out:=filepath.Join(dir,fmt.Sprintf("%s_%d.png",safe,time.Now().UnixNano()))
 f,err:=os.Create(out);if err!=nil{return "",err};defer f.Close()
 if err:=png.Encode(f,img);err!=nil{return "",err}
 return out,nil
}
func startGraphic(){
 if graphicBusy{return}
 kindIndex,_,_:=send.Call(ctrls[304],CB_GETCURSEL,0,0)
 kinds:=[]string{"Attire pattern","Wrestler logo","Entrance background","Wrestling sign"}
 if int(kindIndex)>=len(kinds){alert("Select a graphic type.");return}
 kind:=kinds[kindIndex]
 prompt:=strings.TrimSpace(txt(305))
 if prompt==""{alert("Describe the colors and pattern first.");return}
 graphicBusy=true
 set(306,"Generating local graphic...")
 go func(){
  path,err:=proceduralGraphic(kind,prompt)
  matchMu.Lock()
  if err!=nil{graphicStatus="Generation failed: "+err.Error()}else{graphicStatus="Saved: "+path}
  matchMu.Unlock()
  postMessage.Call(hwnd,WM_GRAPHIC_DONE,0,0)
 }()
}

func launchAIStudio(){
 exe,err:=os.Executable();if err!=nil{return}
 page:=filepath.Join(filepath.Dir(exe),"ai_behavior_studio.html")
 if _,err:=os.Stat(page);err!=nil{set(601,"AI Studio file missing. Extract the entire ZIP.");return}
 exec.Command("rundll32.exe","url.dll,FileProtocolHandler",page).Start()
}
func launchLivePreview(){
 exe,err:=os.Executable();if err!=nil{alert(err.Error());return}
 source:=filepath.Join(filepath.Dir(exe),"viewer.html")
 raw,err:=os.ReadFile(source);if err!=nil{alert("Live preview viewer.html missing. Extract the entire ZIP before running.");return}
 data:=map[string]interface{}{"character":txt(20),"template":fmt.Sprintf("%s%d",gender,template),"values":values}
 groupMap:=map[string][]string{};for _,g:=range groups{groupMap[g.name]=g.items};data["groups"]=groupMap
 payload,err:=json.Marshal(data);if err!=nil{alert(err.Error());return}
 content:=strings.Replace(string(raw),"__CAWFORGE_DATA__",string(payload),1)
 tempDir:=filepath.Join(os.TempDir(),"CAWForge_Live_Preview");if err=os.MkdirAll(tempDir,0700);err!=nil{alert(err.Error());return}
 output:=filepath.Join(tempDir,"preview.html")
 if err=os.WriteFile(output,[]byte(content),0600);err!=nil{alert(err.Error());return}
 exec.Command("rundll32.exe","url.dll,FileProtocolHandler",output).Start()
}
func switchTab(n int){
 currentTab=n
 // Native Windows tabs: show the controls belonging to the selected workspace.
 for _,id:=range []int{1,2,3,4,5,6,7,8,9,10,11,12,13,20,23,25,30,40,41,501,502,503,504,505,506}{
  if v:=ctrls[id];v!=0{u.NewProc("ShowWindow").Call(v,uintptr(boolInt(n==0)))}
 }
 for i:=0;i<12;i++{if v:=ctrls[50+i];v!=0{u.NewProc("ShowWindow").Call(v,uintptr(boolInt(n==0)))}}
 for i:=0;i<8;i++{for _,id:=range []int{100+i,110+i,120+i}{if v:=ctrls[id];v!=0{u.NewProc("ShowWindow").Call(v,uintptr(boolInt(n==0)))}}}
 for _,id:=range []int{201,203,204,205,206,208,209,210,211,212,213,214,215,216,217}{if v:=ctrls[id];v!=0{u.NewProc("ShowWindow").Call(v,uintptr(boolInt(n==1)))}}
 for _,id:=range []int{301,302,303,304,305,306,307}{if v:=ctrls[id];v!=0{u.NewProc("ShowWindow").Call(v,uintptr(boolInt(n==2)))}}
}

var values=map[string]int{}
var groups=[]struct{name string;items []string}{
 {"Head",[]string{"Head Width","Head Height","Head Depth"}},{"Ears Top",[]string{"Offset","Height","Alignment"}},{"Ears",[]string{"Ears Width","Ears Height","Vertical Stretch","Placement Low/High","Placement Forward/Back"}},{"Earlobes",[]string{"Lobe Detachment","Lobe Length","Lobe Alignment"}},{"Forehead",[]string{"Forehead Upper Size","Forehead Size"}},{"Eyebrows",[]string{"Eyebrow Length","Eyebrow Depth","Eyebrow Height","Eyebrow Angle","Eyebrow Spacing"}},{"Eyes",[]string{"Eye Width","Eye Height","Eye Depth","Eye Angle","Eye Spacing","Eye Position"}},{"Nose",[]string{"Nose Width","Nose Height","Nose Length","Nose Bridge Width","Nose Tip Width","Nostril Width","Nostril Height"}},{"Cheeks",[]string{"Cheek Width","Cheek Height","Cheek Depth"}},{"Mouth",[]string{"Mouth Width","Mouth Height","Upper Lip Thickness","Lower Lip Thickness","Mouth Depth"}},{"Jaw",[]string{"Jaw Width","Jaw Height","Jaw Depth","Jaw Angle"}},{"Chin",[]string{"Chin Width","Chin Height","Chin Depth","Chin Shape"}},}
func ctl(class,title string,x,y,w,h,id int,style uint32)uintptr{v,_,_:=create.Call(0,uintptr(unsafe.Pointer(ptr(class))),uintptr(unsafe.Pointer(ptr(title))),uintptr(WS_CHILD|WS_VISIBLE|style),uintptr(x),uintptr(y),uintptr(w),uintptr(h),hwnd,uintptr(id),0,0);if id>0{ctrls[id]=v};return v}
func txt(id int)string{if ctrls[id]==0{return ""};buf:=make([]uint16,2048);gettext.Call(ctrls[id],uintptr(unsafe.Pointer(&buf[0])),uintptr(len(buf)));return syscall.UTF16ToString(buf)}
func set(id int,s string){if ctrls[id]==0{return};settext.Call(ctrls[id],uintptr(unsafe.Pointer(ptr(s))))}
func alert(s string){msgbox.Call(hwnd,uintptr(unsafe.Pointer(ptr(s))),uintptr(unsafe.Pointer(ptr("CAWForge Studio"))),0)}
func fileDialog(save bool)(string,error){
 // Use the native Windows common dialog directly, without spawning PowerShell.
 // Run on a dedicated locked OS thread and do not attach to the app's UI
 // window: that avoids cross-thread owner/window-message deadlocks.
 file:=make([]uint16,4096)
 var filter string
 var title string
 if save {
  filter="Text files (*.txt)\x00*.txt\x00All files (*.*)\x00*.*\x00\x00"
  title="Export WWE Morph Sheet"
 } else {
  filter="Image files (*.png;*.jpg;*.jpeg;*.bmp)\x00*.png;*.jpg;*.jpeg;*.bmp\x00All files (*.*)\x00*.*\x00\x00"
  title="Choose Reference Photo"
 }
 // Windows OPENFILENAME filters require embedded NUL separators and a final double NUL.
 // syscall.StringToUTF16 deliberately panics on embedded NULs, so build the buffer directly.
 f:=make([]uint16,0,len(filter))
 for _,ch:=range filter { f=append(f,uint16(ch)) }
 if len(f)<2 || f[len(f)-1]!=0 || f[len(f)-2]!=0 { f=append(f,0,0) }
 initialDir:=filepath.Join(os.Getenv("USERPROFILE"),"Pictures")
 if info,e:=os.Stat(initialDir);e!=nil||!info.IsDir(){initialDir=filepath.Join(os.Getenv("USERPROFILE"),"Documents")}
 ofn:=OPENFILENAME{Size:uint32(unsafe.Sizeof(OPENFILENAME{})),Owner:0,
  Filter:&f[0],FilterIndex:1,File:&file[0],MaxFile:uint32(len(file)),
  Title:ptr(title),InitialDir:ptr(initialDir),Flags:0x00080000|0x00001000|0x00000800|0x02000000}
 if save {ofn.DefExt=ptr("txt");ofn.Flags|=0x00000002
 } else {ofn.Flags|=0x00001000|0x00000004}
 var r uintptr
 if save {r,_,_=savefile.Call(uintptr(unsafe.Pointer(&ofn)))
 } else {r,_,_=openfile.Call(uintptr(unsafe.Pointer(&ofn)))}
 if r==0 {
  runtime.KeepAlive(f)
  runtime.KeepAlive(file)
  runtime.KeepAlive(ofn)
  code,_,_:=c.NewProc("CommDlgExtendedError").Call()
  if code!=0{return "",fmt.Errorf("Windows file dialog error 0x%X",code)}
  return "",nil // User cancelled.
 }
 runtime.KeepAlive(f)
 runtime.KeepAlive(file)
 runtime.KeepAlive(ofn)
 return syscall.UTF16ToString(file),nil
}

var templatePickerMode bool
var gridPickerMode string
func beginPicker(save bool){
 if pickerBusy{return}
 pickerBusy=true
 // Keep the message loop free of synchronous diagnostics disk I/O.
 set(25,"Opening Windows photo picker...")
 templateMode:=templatePickerMode
 gridMode:=gridPickerMode
 go func(){
  exe,err:=os.Executable()
  var path string
  if err==nil{
   mode:="--pick-photo";if save{mode="--pick-save"}
   cmd:=exec.Command(exe,mode)
   cmd.Env=append(os.Environ(),"WWE_PICKER_CHILD=1")
   var stdout,stderr bytes.Buffer
   cmd.Stdout=&stdout;cmd.Stderr=&stderr
   err=cmd.Run()
   if err==nil{path=strings.TrimSpace(stdout.String())}
   if err!=nil && stderr.Len()>0{err=fmt.Errorf("%v: %s",err,stderr.String())}
  }
  traceTexture("Photo picker process returned")
  pickerMu.Lock();pickerResult=path;pickerErr="";if err!=nil{pickerErr=err.Error()};pickerMu.Unlock()
  var modeFlag uintptr
  if save{modeFlag=1}else if gridMode=="M"{modeFlag=3}else if gridMode=="F"{modeFlag=4}else if templateMode{modeFlag=2}
  postMessage.Call(hwnd,WM_PICKER_DONE,modeFlag,0)
 }()
}

// importGrid crops a screenshot already tightly framed to the 4x5 face grid.
// Grid numbering is right-to-left: 4,3,2,1; 8,7,6,5; etc.
func importGrid(path,prefix string)string{
 f,err:=os.Open(path);if err!=nil{return "Grid import failed: "+err.Error()};defer f.Close()
 src,_,err:=image.Decode(f);if err!=nil{return "Grid decode failed: "+err.Error()}
 bounds:=src.Bounds();w,h:=bounds.Dx(),bounds.Dy();if w<200||h<200{return "Grid image too small"}
 folder:="Male";if prefix=="F"{folder="Female"}
 dest:=filepath.Join(templateLibraryDir,folder);if err=os.MkdirAll(dest,0755);err!=nil{return err.Error()}
 for row:=0;row<5;row++{for col:=0;col<4;col++{
  x0:=bounds.Min.X+col*w/4;y0:=bounds.Min.Y+row*h/5
  x1:=bounds.Min.X+(col+1)*w/4;y1:=bounds.Min.Y+(row+1)*h/5
  target:=image.NewRGBA(image.Rect(0,0,x1-x0,y1-y0))
  draw.Draw(target,target.Bounds(),src,image.Point{X:x0,Y:y0},draw.Src)
  num:=row*4+(4-col)
  output,err:=os.Create(filepath.Join(dest,fmt.Sprintf("%s%d.png",prefix,num)))
  if err!=nil{return "Saving grid failed: "+err.Error()}
  err=png.Encode(output,target);output.Close();if err!=nil{return "Encoding grid failed: "+err.Error()}
 }}
 return fmt.Sprintf("Imported 20 %s grid crops. Refresh Template Library to index them. Grid must be tightly cropped.",folder)
}
func boolInt(b bool)int{if b{return 1};return 0}
const WM_PHOTO_VALIDATED=0x800A
var photoImportMu sync.Mutex
var photoImportSerial uint64
var photoImportPath string
var photoImportError string
func validatePhotoAsync(path string){
 photoImportMu.Lock();photoImportSerial++;serial:=photoImportSerial;photoImportMu.Unlock()
 set(25,"Checking reference photo in background...")
 go func(){
  var failure string
  info,err:=os.Stat(path)
  if err!=nil{failure="Cannot read image: "+err.Error()}else if info.Size()>75*1024*1024{failure="Image exceeds 75 MB. Please resize or compress it first."}
  if failure==""{
   f,err:=os.Open(path)
   if err!=nil{failure=err.Error()}else{
    cfg,_,e:=image.DecodeConfig(f);f.Close()
    if e!=nil{failure="Unsupported or damaged image: "+e.Error()}else if cfg.Width<32||cfg.Height<32{failure="Image is too small."}else if int64(cfg.Width)*int64(cfg.Height)>35000000{failure="Image exceeds 35 megapixels. Please resize it first."}
   }
  }
  photoImportMu.Lock()
  if serial!=photoImportSerial{photoImportMu.Unlock();return}
  photoImportPath=path;photoImportError=failure
  photoImportMu.Unlock()
  postMessage.Call(hwnd,WM_PHOTO_VALIDATED,0,0)
 }()
}
func acceptPhoto(path string){
 if path==""{set(25,"Drop photo onto window or choose file (drag and drop also supported)");return}
 ext:=strings.ToLower(filepath.Ext(path));if ext!=".jpg"&&ext!=".jpeg"&&ext!=".png"&&ext!=".bmp"&&ext!=".webp"{alert("Please select a JPG, PNG, BMP, or WEBP image.");return}
 validatePhotoAsync(path)
}
func importTemplateImage(){
 templatePickerMode=true
 beginPicker(false)
}
func assignTemplateImage(){
 if imagePath=="" {alert("Import a reference image first, then assign it to the selected template.");return}
 if templateLibraryDir=="" {alert("Template library unavailable.");return}
 name:=fmt.Sprintf("%s%d%s",gender,template,strings.ToLower(filepath.Ext(imagePath)))
 dst:=filepath.Join(templateLibraryDir,name)
 if filepath.Clean(dst)==filepath.Clean(imagePath){beginCalibration();return}
 in,e:=os.Open(imagePath);if e!=nil{alert(e.Error());return};defer in.Close()
 f,e:=os.Create(dst);if e!=nil{alert(e.Error());return};defer f.Close()
 _,e=io.Copy(f,in);if e!=nil{alert(e.Error());return}
 set(25,"Template image imported: "+name);set(41,"Saved "+name+". Rebuilding image index...");beginCalibration()
}
// Export without a Windows Save dialog. The dialog was a possible source of
// hangs; disk writes happen on a worker and completion returns via WM_EXPORT_DONE.
var morphExportBusy bool
func exportMorphSheetAsync(){
 if morphExportBusy {set(25,"Morph sheet export already running...");return}
 morphExportBusy=true
 content:=[]byte(sheet()) // Small, consistent snapshot on UI thread.
 name:=strings.TrimSpace(txt(20))
 if name=="" {name="Wrestler"}
 var clean strings.Builder
 for _,r:=range name {
  if r>='a'&&r<='z'||r>='A'&&r<='Z'||r>='0'&&r<='9'||r=='-'||r=='_' {clean.WriteRune(r)} else if r==' ' {clean.WriteByte('_')}
 }
 safe:=clean.String();if safe==""{safe="Wrestler"}
 if len(safe)>70{safe=safe[:70]}
 set(25,"Exporting morph sheet in background...")
 go func(){
  home,err:=os.UserHomeDir()
  if err!=nil{home=os.TempDir()}
  folder:=filepath.Join(home,"Documents","CAWForge","MorphSheets")
  err=os.MkdirAll(folder,0755)
  path:=filepath.Join(folder,fmt.Sprintf("%s_%s_%s.txt",safe,gender+strconv.Itoa(template),time.Now().Format("20060102_150405.000")))
  if err==nil {err=os.WriteFile(path,content,0644)}
  matchMu.Lock()
  if err!=nil{exportStatus="Morph sheet export failed: "+err.Error()}else{exportStatus="Morph sheet saved: "+path}
  matchMu.Unlock()
  postMessage.Call(hwnd,WM_EXPORT_DONE,0,0)
 }()
}
func sheet()string{var b strings.Builder;fmt.Fprintf(&b,"CAWFORGE STUDIO — FACE DEFORMATION SHEET\r\nCharacter: %s\r\nFace Template: %s%d\r\nReference Photo: %s\r\n\r\n",txt(20),gender,template,imagePath);for _,g:=range groups{fmt.Fprintf(&b,"[%s]\r\n",g.name);for _,s:=range g.items{fmt.Fprintf(&b,"%s: %d\r\n",s,values[g.name+" / "+s])};b.WriteString("\r\n")};b.WriteString("NOTE: These numbers are automatically generated IMAGE-BASED ESTIMATES, NOT validated WWE 2K26 slider calibration. Inspect in-game.\r\n");return b.String()}
func updateSheet(){set(30,sheet())}
func scheduleSheetUpdate(){
 // Debounce expensive full morph-sheet rebuilds during trackbar dragging.
 // The timer fires on the GUI message loop only after input settles.
 u.NewProc("KillTimer").Call(hwnd,MORPH_PREVIEW_TIMER)
 u.NewProc("SetTimer").Call(hwnd,MORPH_PREVIEW_TIMER,180,0)
}


func writeMorphPreviewBMP(path string)(string,error){
 f,err:=os.Open(path);if err!=nil{return "",err};defer f.Close()
 im,_,err:=image.Decode(f);if err!=nil{return "",err}
 const width=180
 const height=145
 row:=((width*3+3)/4)*4
 data:=make([]byte,54+row*height);copy(data[:2],[]byte{'B','M'})
 put:=func(i int,v uint32){data[i]=byte(v);data[i+1]=byte(v>>8);data[i+2]=byte(v>>16);data[i+3]=byte(v>>24)}
 put(2,uint32(len(data)));put(10,54);put(14,40);put(18,width);put(22,height);data[26]=1;data[28]=24
 b:=im.Bounds()
 for y:=0;y<height;y++{for x:=0;x<width;x++{
  sx:=b.Min.X+x*b.Dx()/width;sy:=b.Min.Y+(height-1-y)*b.Dy()/height
  r,g,bb,_:=im.At(sx,sy).RGBA();i:=54+y*row+x*3
  data[i]=byte(bb>>8);data[i+1]=byte(g>>8);data[i+2]=byte(r>>8)
 }}
 out:=filepath.Join(os.TempDir(),fmt.Sprintf("cawforge_face_compare_%d.bmp",time.Now().UnixNano()))
 return out,os.WriteFile(out,data,0644)
}

const WM_PREVIEW_READY=0x8007
var morphPreviewHandles=[2]uintptr{}
var morphPreviewFiles=[2]string{}
var previewMu sync.Mutex
var previewGeneration uint64
type previewResult struct {generation uint64; files [2]string; status string}
var pendingPreview previewResult
func findTemplatePreview()string{
 id:=fmt.Sprintf("%s%d",gender,template)
 folder:="Male";if gender=="F"{folder="Female"}
 for _,ext:=range []string{".png",".jpg",".jpeg"}{
  for _,candidate:=range []string{filepath.Join(templateLibraryDir,folder,id+ext),filepath.Join(templateLibraryDir,id+ext)}{
   if _,err:=os.Stat(candidate);err==nil{return candidate}
  }
 }
 return ""
}
var previewGate=make(chan struct{},1)
func updateMorphPreview(){
 // File decoding and resizing must never block the Win32 message loop.
 previewMu.Lock()
 previewGeneration++
 generation:=previewGeneration
 photo:=imagePath
 id:=fmt.Sprintf("%s%d",gender,template)
 templateFile:=findTemplatePreview()
 previewMu.Unlock()
 go func(){
  // Coalesce rapid changes and never run more than one expensive preview at once.
  time.Sleep(160*time.Millisecond)
  previewMu.Lock();stale:=generation!=previewGeneration;previewMu.Unlock()
  if stale{return}
  previewGate<-struct{}{}
  defer func(){<-previewGate}()
  previewMu.Lock();stale=generation!=previewGeneration;previewMu.Unlock()
  if stale{return}
  var result previewResult
  result.generation=generation
  paths:=[]string{photo,templateFile}
  for i,path:=range paths{
   if path==""{continue}
   bmp,err:=writeMorphPreviewBMP(path)
   if err!=nil{result.status="Preview unavailable: "+err.Error()}else{result.files[i]=bmp}
  }
  if result.status==""{
   if templateFile!=""{result.status="Photo vs. template "+id+" — reference comparison, not 3D rendering."}else{result.status="No template thumbnail found for "+id}
  }
  previewMu.Lock()
  if generation!=previewGeneration{
   previewMu.Unlock()
   for _,p:=range result.files{if p!=""{os.Remove(p)}}
   return
  }
  // Discard any unconsumed result to prevent temporary-file buildup.
  for _,p:=range pendingPreview.files{if p!=""{os.Remove(p)}}
  pendingPreview=result
  previewMu.Unlock()
  postMessage.Call(hwnd,WM_PREVIEW_READY,0,0)
 }()
}
func applyMorphPreview(){
 previewMu.Lock()
 result:=pendingPreview
 pendingPreview=previewResult{}
 valid:=result.generation==previewGeneration
 previewMu.Unlock()
 if !valid{
  for _,p:=range result.files{if p!=""{os.Remove(p)}}
  return
 }
 for i,id:=range []int{501,502}{
  path:=result.files[i]
  if path==""{
   send.Call(ctrls[id],STM_SETIMAGE,IMAGE_BITMAP,0)
  }else{
   bmp,_,_:=u.NewProc("LoadImageW").Call(0,uintptr(unsafe.Pointer(ptr(path))),IMAGE_BITMAP,0,0,LR_LOADFROMFILE)
   if bmp!=0{
    send.Call(ctrls[id],STM_SETIMAGE,IMAGE_BITMAP,bmp)
    if morphPreviewHandles[i]!=0{syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteObject").Call(morphPreviewHandles[i])}
    morphPreviewHandles[i]=bmp
   }
  }
  if morphPreviewFiles[i]!=""{os.Remove(morphPreviewFiles[i])}
  morphPreviewFiles[i]=path
 }
 set(504,result.status)
}
func drawGroup(){for i:=0;i<8;i++{set(100+i,"");set(120+i,"");send.Call(ctrls[110+i],TBM_SETPOS,1,50)};for _,g:=range groups{if g.name!=active{continue};for i,s:=range g.items{key:=g.name+" / "+s;set(100+i,s);set(120+i,strconv.Itoa(values[key]));send.Call(ctrls[110+i],TBM_SETPOS,1,uintptr(values[key]))}};set(40,"Category: "+active);updateSheet()}
func wndproc(h uintptr,m uint32,w,l uintptr)uintptr{switch m{case WM_DESTROY:for i:=0;i<2;i++{if morphPreviewHandles[i]!=0{syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteObject").Call(morphPreviewHandles[i])};if morphPreviewFiles[i]!=""{os.Remove(morphPreviewFiles[i])}};u.NewProc("KillTimer").Call(hwnd,MORPH_PREVIEW_TIMER);if themeBitmap!=0{syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteObject").Call(themeBitmap)};if themeConvertedPath!=""{os.Remove(themeConvertedPath)};if previewBitmap!=0{syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteObject").Call(previewBitmap)};if previewBmpPath!=""{os.Remove(previewBmpPath)};post.Call(0);return 0;
case WM_UPDATE_CHECKED:
 updaterMu.Lock();msg:=updaterStatus;tag,url:=updaterTag,updaterAssetURL;updaterMu.Unlock()
 set(601,msg)
 if url!="" && newerVersion(tag,currentVersion){
  message:="CAWForge "+tag+" is available.\n\nDownload and install now? The app will close and reopen automatically."
  answer,_,_:=u.NewProc("MessageBoxW").Call(hwnd,uintptr(unsafe.Pointer(ptr(message))),uintptr(unsafe.Pointer(ptr("CAWForge Update Available"))),0x24)
  if answer==6{autoInstallAfterDownload=true;downloadUpdate()}
 }
 return 0
case WM_UPDATE_DOWNLOADED:
 updaterMu.Lock();msg:=updaterStatus;downloaded:=updaterDownloadPath;updaterMu.Unlock()
 set(601,msg)
 if autoInstallAfterDownload{
  autoInstallAfterDownload=false
  if downloaded!="" && strings.HasPrefix(msg,"Verified ZIP"){installDownloadedUpdate()}
 }
 return 0
case WM_PHOTO_VALIDATED:
 photoImportMu.Lock();path,err:=photoImportPath,photoImportError;photoImportMu.Unlock()
 if err!=""{set(25,"Photo import failed: "+err);return 0}
 imagePath=path
 set(25,"Photo loaded: "+filepath.Base(path))
 scheduleSheetUpdate()
 updateMorphPreview()
 matchPhoto()
 return 0
case WM_PREVIEW_READY:applyMorphPreview();return 0
case WM_TIMER:
 if w==9876{return 0}
 if w==MORPH_PREVIEW_TIMER{
  u.NewProc("KillTimer").Call(hwnd,MORPH_PREVIEW_TIMER)
  updateSheet()
  return 0
 }
case WM_MATCH_DONE:
 matchMu.Lock();message,best,estimated:=matchMessage,matchID,morphResult;matchMu.Unlock();matchRunning=false
 if best!="" && strings.HasPrefix(best,gender){n,_:=strconv.Atoi(best[1:]);template=n;send.Call(ctrls[3],CB_SETCURSEL,uintptr(n-1),0);set(23,best);for key,value:=range estimated{values[key]=value};drawGroup();updateMorphPreview()};set(41,message);if matchQueued{matchQueued=false;matchPhoto()};return 0
case WM_GRAPHIC_DONE:
 matchMu.Lock();status:=graphicStatus;matchMu.Unlock();graphicBusy=false;set(306,status);return 0
case WM_AI_DONE:
 matchMu.Lock();status,path:=aiResult,aiOutput;matchMu.Unlock();aiBusy=false;set(215,status);if path!=""{showTexturePreview(path)};return 0
case WM_EXPORT_DONE:
 matchMu.Lock();status:=exportStatus;matchMu.Unlock()
 morphExportBusy=false;set(25,status);return 0
case WM_TEXTURE_PREVIEW_READY:applyTexturePreview();return 0
case WM_TEXTURE_DONE:
 matchMu.Lock();status,path:=faceTextureStatus,faceTexturePreviewPath;matchMu.Unlock();textureBusy=false;set(203,status);if path!=""{showTexturePreview(path)};return 0
case WM_CAL_DONE:
 calMu.Lock();message:=calibrationMessage;calMu.Unlock();calibrationRunning=false;set(41,message);if imagePath!=""{matchPhoto()};return 0
case WM_PICKER_DONE:
 pickerMu.Lock();path,err:=pickerResult,pickerErr;pickerMu.Unlock();pickerBusy=false
 if themePicker{themePicker=false;if err!=""{alert(err)}else if path!=""{themeCustomPath=path;themeIndex=6;send.Call(ctrls[401],CB_SETCURSEL,6,0);loadTheme()};return 0}
 if err!=""{templatePickerMode=false;set(25,"File picker error");alert("File picker failed: "+err)}else if w==2 {templatePickerMode=false;if path!=""{imagePath=path;assignTemplateImage()}else{set(25,"Template selection cancelled.")}}else if w==3 || w==4 {gridPickerMode="";if path!=""{prefix:="M";if w==4{prefix="F"};go func(){result:=importGrid(path,prefix);calMu.Lock();calibrationMessage=result;calMu.Unlock();postMessage.Call(hwnd,WM_CAL_DONE,0,0)}();set(41,"Cropping 20 grid thumbnails...")}}else if w==1 {if path!=""{content:=[]byte(sheet())
  set(25,"Exporting morph sheet...")
  go func(){
   err:=os.WriteFile(path,content,0644)
   matchMu.Lock()
   if err!=nil{exportStatus="Morph sheet export failed: "+err.Error()}else{exportStatus="Morph sheet exported: "+path}
   matchMu.Unlock()
   postMessage.Call(hwnd,WM_EXPORT_DONE,0,0)
  }()}}else{acceptPhoto(path)}
 return 0
case WM_DROPFILES:
 count,_,_:=dragQuery.Call(w,0xFFFFFFFF,0,0)
 if count>0 {buf:=make([]uint16,32768);dragQuery.Call(w,0,uintptr(unsafe.Pointer(&buf[0])),uintptr(len(buf)));acceptPhoto(syscall.UTF16ToString(buf))}
 dragFinish.Call(w);return 0
case WM_COMMAND:
 id:=int(w&0xffff)
 notification:=uint16(w>>16)
 // Ignore edit-change, focus and dropdown notifications. Only actual button
 // clicks (BN_CLICKED=0) and combo selection changes (CBN_SELCHANGE=1)
 // may invoke potentially expensive UI handlers.
 if id==3||id==13||id==212||id==213||id==304||id==401{
  if notification!=1{return 0}
 }else if notification!=0{return 0}
 switch {case id==402:themePicker=true;beginPicker(false)
 case id==401:
 // Only act on CBN_SELCHANGE (notification 1). Other combo notifications
 // previously caused spurious theme loads and apparent selector bugs.
 if (w>>16)==1{
  sel,_,_:=send.Call(ctrls[401],CB_GETCURSEL,0,0)
  if sel<6{themeIndex=int(sel);loadTheme()}else if themeCustomPath!=""{themeIndex=6;loadTheme()}else{themePicker=true;beginPicker(false)}
 }
 case id==605:installDownloadedUpdate();case id==604:launchAIStudio();case id==600:checkUpdates();case id==602:downloadUpdate();case id==603:openUpdateFolder();case id==200:switchTab(0);case id==300:switchTab(2);case id==307:startGraphic();case id==202:switchTab(1);case id==201:startTexture();case id==216:runAITexture();case id==217:beginPicker(false);case id==13:sel,_,_:=send.Call(ctrls[13],CB_GETCURSEL,0,0);if sel==1{genderMode="F";gender="F"}else if sel==2{genderMode="M";gender="M"}else{genderMode="Auto"};set(23,gender+strconv.Itoa(template));updateSheet();updateMorphPreview();if imagePath!=""{matchPhoto()};set(41,"Template library mode: "+genderMode+". Auto currently uses the selected library; photo classifier not installed.");case id==113:genderMode="Auto";set(41,"Auto: photo-based gender recognition is not installed. Using selected template library; choose Female or Male to lock matching.");case id==1:genderMode="M";gender="M";send.Call(ctrls[13],CB_SETCURSEL,2,0);set(23,"M"+strconv.Itoa(template));updateMorphPreview();set(41,"Male library selected. Matching requires installed reference images.");updateSheet();if imagePath!=""{matchPhoto()};case id==2:genderMode="F";gender="F";send.Call(ctrls[13],CB_SETCURSEL,1,0);set(23,"F"+strconv.Itoa(template));updateMorphPreview();set(41,"Female library selected. Matching requires installed reference images.");updateSheet();if imagePath!=""{matchPhoto()};case id==3:sel,_,_:=send.Call(ctrls[3],CB_GETCURSEL,0,0);if sel>=20{return 0};template=int(sel)+1;set(23,gender+strconv.Itoa(template));updateSheet();updateMorphPreview();case id==4:beginPicker(false);case id==5:exportMorphSheetAsync();case id==7:importTemplateImage();case id==9:beginCalibration();case id==10:matchPhoto();case id==11:gridPickerMode="M";beginPicker(false);case id==12:gridPickerMode="F";beginPicker(false);case id==8:exec.Command("rundll32.exe","url.dll,FileProtocolHandler","https://www.youtube.com/watch?v=WtWG9ZIxtes").Start();case id==6:for k:=range values{values[k]=50};drawGroup();case id>=50&&id<62:active=groups[id-50].name;drawGroup()};case WM_HSCROLL:for i:=0;i<8;i++{if l==ctrls[110+i]{for _,g:=range groups{if g.name==active&&i<len(g.items){v,_,_:=send.Call(ctrls[110+i],TBM_GETPOS,0,0);values[g.name+" / "+g.items[i]]=int(v);set(120+i,strconv.Itoa(int(v)));scheduleSheetUpdate()}}}}};r,_,_:=def.Call(h,uintptr(m),w,l);return r}

const currentVersion="v0.9.25"
const releaseEndpoint="https://api.github.com/repos/Becksploder2/CAWForge/releases/latest"
const WM_UPDATE_CHECKED=0x8010
const WM_UPDATE_DOWNLOADED=0x8011
var updaterMu sync.Mutex
var updaterStatus string
var updaterAssetURL string
var updaterTag string
var updaterAssetName string
var updaterDownloadPath string
var updaterBusy bool
var autoInstallAfterDownload bool
func updaterMessage(message string,signal uint32){
 updaterMu.Lock();updaterStatus=message;updaterBusy=false;updaterMu.Unlock()
 postMessage.Call(hwnd,uintptr(signal),0,0)
}
func versionParts(s string)[]int{
 s=strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s)),"v")
 a:=strings.Split(s,".");out:=[]int{}
 for _,v:=range a{if !regexp.MustCompile(`^\d+$`).MatchString(v){return nil};n,e:=strconv.Atoi(v);if e!=nil{return nil};out=append(out,n)}
 return out
}
func newerVersion(remote,local string)bool{
 a,b:=versionParts(remote),versionParts(local);if len(a)==0||len(b)==0{return false}
 for i:=0;i<len(a)||i<len(b);i++{x,y:=0,0;if i<len(a){x=a[i]};if i<len(b){y=b[i]};if x!=y{return x>y}}
 return false
}
func checkUpdates(){
 updaterMu.Lock();if updaterBusy{updaterMu.Unlock();return};updaterBusy=true;updaterMu.Unlock()
 set(601,"Checking GitHub Releases...")
 go func(){
  ctx,cancel:=context.WithTimeout(context.Background(),18*time.Second);defer cancel()
  req,err:=http.NewRequestWithContext(ctx,"GET",releaseEndpoint,nil)
  if err!=nil{updaterMessage(err.Error(),WM_UPDATE_CHECKED);return}
  req.Header.Set("User-Agent","CAWForge-Studio-Updater/0.9.7")
  req.Header.Set("Accept","application/vnd.github+json")
  resp,err:=http.DefaultClient.Do(req)
  if err!=nil{updaterMessage("GitHub check failed: "+err.Error(),WM_UPDATE_CHECKED);return};defer resp.Body.Close()
  if resp.StatusCode!=200{updaterMessage(fmt.Sprintf("GitHub returned HTTP %d. Publish a Release first.",resp.StatusCode),WM_UPDATE_CHECKED);return}
  var data struct{Tag string `json:"tag_name"`;Assets []struct{Name string `json:"name"`;URL string `json:"browser_download_url"`;Size int64 `json:"size"`} `json:"assets"`}
  if err=json.NewDecoder(io.LimitReader(resp.Body,2<<20)).Decode(&data);err!=nil{updaterMessage("Invalid GitHub release response",WM_UPDATE_CHECKED);return}
  if !newerVersion(data.Tag,currentVersion){updaterMessage("You're up to date ("+currentVersion+"). Latest release: "+data.Tag,WM_UPDATE_CHECKED);return}
  var chosen struct{Name,URL string;Size int64}
  for _,a:=range data.Assets{
   n:=strings.ToLower(a.Name)
   if strings.HasSuffix(n,".zip")&&strings.Contains(n,"windows")&&strings.Contains(n,"cawforge")&&a.Size>0&&a.Size<300<<20&&strings.HasPrefix(a.URL,"https://github.com/Becksploder2/CAWForge/releases/download/"){
    chosen.Name,chosen.URL,chosen.Size=a.Name,a.URL,a.Size;break
   }
  }
  if chosen.URL==""{updaterMessage("New version "+data.Tag+" found, but no Windows CAWForge ZIP asset is attached.",WM_UPDATE_CHECKED);return}
  updaterMu.Lock();updaterTag=data.Tag;updaterAssetURL=chosen.URL;updaterAssetName=chosen.Name;updaterMu.Unlock()
  updaterMessage("New version "+data.Tag+" available.",WM_UPDATE_CHECKED)
 }()
}
func downloadUpdate(){
 updaterMu.Lock()
 if updaterBusy{updaterMu.Unlock();return}
 url,name:=updaterAssetURL,updaterAssetName
 if url==""{updaterMu.Unlock();set(601,"Check for updates first.");return}
 updaterBusy=true;updaterMu.Unlock()
 set(601,"Downloading "+name+" in the background...")
 go func(){
  ctx,cancel:=context.WithTimeout(context.Background(),6*time.Minute);defer cancel()
  req,err:=http.NewRequestWithContext(ctx,"GET",url,nil);if err!=nil{updaterMessage(err.Error(),WM_UPDATE_DOWNLOADED);return}
  req.Header.Set("User-Agent","CAWForge-Studio-Updater/0.9.7")
  resp,err:=http.DefaultClient.Do(req);if err!=nil{updaterMessage("Download failed: "+err.Error(),WM_UPDATE_DOWNLOADED);return};defer resp.Body.Close()
  if resp.StatusCode!=200{updaterMessage(fmt.Sprintf("Download failed: HTTP %d",resp.StatusCode),WM_UPDATE_DOWNLOADED);return}
  if resp.ContentLength>300<<20{updaterMessage("Update exceeds 300 MB limit",WM_UPDATE_DOWNLOADED);return}
  dir:=filepath.Join(os.TempDir(),"CAWForge-Updates");if err=os.MkdirAll(dir,0755);err!=nil{updaterMessage(err.Error(),WM_UPDATE_DOWNLOADED);return}
  target:=filepath.Join(dir,filepath.Base(name));f,err:=os.Create(target+".partial");if err!=nil{updaterMessage(err.Error(),WM_UPDATE_DOWNLOADED);return}
  h:=sha256.New();n,err:=io.Copy(io.MultiWriter(f,h),io.LimitReader(resp.Body,(300<<20)+1));f.Close()
  if err!=nil||n>300<<20||n==0{os.Remove(target+".partial");updaterMessage("Update download incomplete",WM_UPDATE_DOWNLOADED);return}
  if err=validateUpdateZip(target+".partial");err!=nil{os.Remove(target+".partial");updaterMessage("Unsafe update ZIP: "+err.Error(),WM_UPDATE_DOWNLOADED);return}
  os.Remove(target)
  if err=os.Rename(target+".partial",target);err!=nil{updaterMessage(err.Error(),WM_UPDATE_DOWNLOADED);return}
  updaterMu.Lock();updaterDownloadPath=target;updaterMu.Unlock()
  updaterMessage("Verified ZIP (SHA-256: "+hex.EncodeToString(h.Sum(nil))[:12]+"...). Click Install & Restart.",WM_UPDATE_DOWNLOADED)
 }()
}
func openUpdateFolder(){
 updaterMu.Lock();p:=updaterDownloadPath;updaterMu.Unlock()
 if p==""{set(601,"Download an update first.");return}
 exec.Command("explorer.exe","/select,",p).Start()
}

// Updates run from a temporary copy of the executable after the main process exits.
// The ZIP must contain the expected app binary, and only files in the package are replaced.
func validateUpdateZip(path string)error{
 z,err:=zip.OpenReader(path);if err!=nil{return err};defer z.Close()
 found:=false;var total uint64
 for _,f:=range z.File{
  n:=strings.ReplaceAll(f.Name,"\\","/")
  if strings.HasPrefix(n,"/")||strings.Contains(n,"../")||strings.HasPrefix(n,"../")||strings.Contains(n,":")||f.Mode()&os.ModeSymlink!=0{return errors.New("unsafe ZIP path")}
  total+=f.UncompressedSize64;if total>650<<20{return errors.New("uncompressed ZIP too large")}
  if filepath.Base(n)=="CAWForge_Studio_Windows.exe"{found=true}
 }
 if !found{return errors.New("app executable missing")};return nil
}
func installDownloadedUpdate(){
 updaterMu.Lock();p:=updaterDownloadPath;busy:=updaterBusy;updaterMu.Unlock()
 if busy||p==""{set(601,"Download and verify an update first.");return}
 if err:=validateUpdateZip(p);err!=nil{set(601,"Update ZIP rejected: "+err.Error());return}
 exe,err:=os.Executable();if err!=nil{set(601,err.Error());return}
 temp:=filepath.Join(os.TempDir(),fmt.Sprintf("CAWForge-updater-%d.exe",time.Now().UnixNano()))
 src,err:=os.Open(exe);if err!=nil{set(601,err.Error());return};defer src.Close()
 dst,err:=os.Create(temp);if err!=nil{set(601,err.Error());return}
 _,err=io.Copy(dst,src);dst.Close();if err!=nil{set(601,err.Error());return}
 cmd:=exec.Command(temp,"--apply-update",p,filepath.Dir(exe),strconv.Itoa(os.Getpid()))
 if err=cmd.Start();err!=nil{set(601,"Updater launch failed: "+err.Error());return}
 set(601,"Updater started. Closing CAWForge...")
 u.NewProc("DestroyWindow").Call(hwnd)
}
func applyUpdate(zipPath,installDir,pidText string)error{
 pid,_:=strconv.Atoi(pidText)
 if pid>0{proc:=syscall.NewLazyDLL("kernel32.dll").NewProc("OpenProcess");wait:=syscall.NewLazyDLL("kernel32.dll").NewProc("WaitForSingleObject");closeh:=syscall.NewLazyDLL("kernel32.dll").NewProc("CloseHandle");h,_,_:=proc.Call(0x00100000,0,uintptr(pid));if h!=0{wait.Call(h,45000);closeh.Call(h)}else{time.Sleep(2*time.Second)}}
 if err:=validateUpdateZip(zipPath);err!=nil{return err}
 z,err:=zip.OpenReader(zipPath);if err!=nil{return err};defer z.Close()
 backup:=filepath.Join(os.TempDir(),fmt.Sprintf("CAWForge-backup-%d",time.Now().UnixNano()))
 type replaced struct{dest,bak string;existed bool};done:=[]replaced{}
 rollback:=func(){for i:=len(done)-1;i>=0;i--{v:=done[i];os.Remove(v.dest);if v.existed{os.Rename(v.bak,v.dest)}}}
 defer os.RemoveAll(backup)
 for _,f:=range z.File{
  if f.FileInfo().IsDir(){continue}
  name:=strings.ReplaceAll(f.Name,"\\","/");parts:=strings.SplitN(name,"/",2);if len(parts)!=2{return errors.New("unexpected ZIP layout")}
  rel:=filepath.FromSlash(parts[1]);if rel==""||strings.HasPrefix(rel,"..")||filepath.IsAbs(rel){rollback();return errors.New("unsafe target path")}
  target:=filepath.Join(installDir,rel)
  // User-provided assets always win. Only install new asset files.
  if strings.HasPrefix(strings.ToLower(rel),"assets"+string(os.PathSeparator)){if _,e:=os.Stat(target);e==nil{continue}}
  if err:=os.MkdirAll(filepath.Dir(target),0755);err!=nil{rollback();return err}
  in,err:=f.Open();if err!=nil{rollback();return err}
  tmp:=target+".cawforge-new";out,err:=os.Create(tmp);if err!=nil{in.Close();rollback();return err}
  _,err=io.Copy(out,io.LimitReader(in,650<<20));in.Close();if ce:=out.Close();err==nil{err=ce};if err!=nil{os.Remove(tmp);rollback();return err}
  v:=replaced{dest:target};if _,err:=os.Stat(target);err==nil{
   v.existed=true;v.bak=filepath.Join(backup,rel);if err=os.MkdirAll(filepath.Dir(v.bak),0755);err!=nil{os.Remove(tmp);rollback();return err}
   if err=os.Rename(target,v.bak);err!=nil{os.Remove(tmp);rollback();return err}
  }
  if err=os.Rename(tmp,target);err!=nil{if v.existed{os.Rename(v.bak,target)};rollback();return err}
  done=append(done,v)
 }
 exe:=filepath.Join(installDir,"CAWForge_Studio_Windows.exe")
 if _,err=os.Stat(exe);err!=nil{rollback();return err}
 return exec.Command(exe).Start()
}

func main(){
 if len(os.Args)==5 && os.Args[1]=="--apply-update" {if err:=applyUpdate(os.Args[2],os.Args[3],os.Args[4]);err!=nil{os.WriteFile(filepath.Join(os.TempDir(),"CAWForge-update-error.txt"),[]byte(err.Error()),0644)};return}
 // Run the native Windows file picker in a separate process, so a hung
 // shell extension cannot block the main application's message loop.
 if len(os.Args)>1 && (os.Args[1]=="--pick-photo"||os.Args[1]=="--pick-save"){
  runtime.LockOSThread()
  defer runtime.UnlockOSThread()
  ole:=syscall.NewLazyDLL("ole32.dll")
  init:=ole.NewProc("CoInitializeEx")
  uninit:=ole.NewProc("CoUninitialize")
  hr,_,_:=init.Call(0,2) // COINIT_APARTMENTTHREADED
  if hr==0 || hr==1{defer uninit.Call()}
  path,err:=fileDialog(os.Args[1]=="--pick-save")
  if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(2)}
  fmt.Print(path)
  return
 }
 exe,_:=os.Executable();templateLibraryDir=filepath.Join(filepath.Dir(exe),"assets","templates");os.MkdirAll(templateLibraryDir,0755);syscall.NewLazyDLL("comctl32.dll").NewProc("InitCommonControls").Call();for _,g:=range groups{for _,s:=range g.items{values[g.name+" / "+s]=50}};inst,_,_:=instance.Call(0);class:=ptr("WWEMorphStudioNative");wc:=WNDCLASSEX{Size:uint32(unsafe.Sizeof(WNDCLASSEX{})),WndProc:syscall.NewCallback(wndproc),Instance:inst,ClassName:class,Background:6};reg.Call(uintptr(unsafe.Pointer(&wc)));hwnd,_,_=create.Call(0,uintptr(unsafe.Pointer(class)),uintptr(unsafe.Pointer(ptr("CAWForge Studio — Desktop Edition v0.9.25"))),WS_OVERLAPPEDWINDOW|WS_VISIBLE,100,70,1130,900,0,0,inst,0)
 // Startup safe mode: skip theme restoration until the user requests it.
 // Avoid a full-window STATIC bitmap: it can paint over every interactive
 // control, making the whole application appear blank/white.
 // The chosen theme is retained and shown in a small non-overlapping preview.
 themeBackdrop=0
 ctl("STATIC","CAWFORGE STUDIO  |  DESKTOP EDITION v0.9.25",20,15,565,28,0,0);
 ctl("BUTTON","Morph Settings",610,12,145,32,200,0);ctl("BUTTON","Face Texture",765,12,140,32,202,0);ctl("BUTTON","Graphic Generator",915,12,180,32,300,0);ctl("STATIC","Character name",20,55,140,22,0,0);ctl("EDIT","New Superstar",160,50,250,28,20,WS_BORDER);ctl("BUTTON","MALE TEMPLATES (M1-M20)",415,50,195,28,1,0);ctl("BUTTON","FEMALE TEMPLATES (F1-F20)",615,50,210,28,2,0);ctl("COMBOBOX","",835,50,130,360,3,CBS_DROPDOWNLIST|WS_VSCROLL);for i:=1;i<=20;i++{send.Call(ctrls[3],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(fmt.Sprintf("Template %d",i)))))};send.Call(ctrls[3],CB_SETCURSEL,9,0);ctl("STATIC","F10",975,50,90,28,23,0)
 ctl("BUTTON","Choose reference photo (or drop image)",20,100,260,32,4,0);ctl("STATIC","Template library:",575,105,130,24,0,0);ctl("COMBOBOX","",705,100,195,240,13,CBS_DROPDOWNLIST|WS_VSCROLL);for _,option:=range []string{"Auto (selected library)","Female only","Male only"}{send.Call(ctrls[13],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(option))))};send.Call(ctrls[13],CB_SETCURSEL,0,0);ctl("STATIC","Drop photo onto window or choose file",295,105,270,24,25,0);ctl("BUTTON","Export morph sheet",20,145,210,32,5,0);ctl("BUTTON","Reset sliders",245,145,150,32,6,0);ctl("BUTTON","Import Template Image",415,145,255,32,7,0);ctl("BUTTON","View template showcase",690,145,210,32,8,0);ctl("BUTTON","Refresh Template Library",20,187,170,30,9,0);ctl("BUTTON","Retry Photo Match",205,187,140,30,10,0);ctl("BUTTON","Import MALE Grid",20,688,170,29,11,0);ctl("BUTTON","Import FEMALE Grid",205,688,180,29,12,0);ctl("STATIC","Auto-selects best available match on photo import (requires real template images).",360,191,680,24,0,0);ctl("STATIC","Loading installed template references automatically...",20,218,1040,22,41,0)
 for i,g:=range groups{ctl("BUTTON",g.name,20+(i%6)*172,255+(i/6)*38,165,31,50+i,0)};ctl("STATIC","Category: Head",20,348,400,26,40,0)
 for i:=0;i<8;i++{y:=384+i*37;ctl("STATIC","",20,y,215,25,100+i,0);v:=ctl("msctls_trackbar32","",250,y-4,315,34,110+i,WS_TABSTOP);send.Call(v,TBM_SETRANGE,1,uintptr(100<<16));ctl("STATIC","",575,y,60,24,120+i,0)}
 ctl("STATIC","FACE COMPARISON PREVIEW",665,345,400,26,0,0);
 ctl("STATIC","Imported photo",665,373,185,22,503,0);
 ctl("STATIC","Matched template",865,373,190,22,505,0);
 ctl("STATIC","",665,398,185,150,501,SS_BITMAP|WS_BORDER);
 ctl("STATIC","",865,398,185,150,502,SS_BITMAP|WS_BORDER);
 ctl("STATIC","Import a photo to compare it with the matched template.",665,550,410,34,504,0);
 ctl("EDIT","",665,625,410,51,30,WS_BORDER|ES_MULTILINE|ES_AUTOVSCROLL|WS_VSCROLL|ES_READONLY);drawGroup();
 ctl("BUTTON","Import Face Photo",30,305,340,40,217,0);
 ctl("BUTTON","Generate 820 x 820 Face Texture",30,365,340,40,201,0);
 ctl("STATIC","Centered front-facing photo recommended. Texture is a provisional crop with feathered edges; not a verified game UV layout.",30,645,980,45,203,0);
 ctl("STATIC","FACE TEXTURE PREVIEW (320 x 320)",425,110,500,28,204,0);
 ctl("STATIC","",425,150,320,320,205,SS_BITMAP|WS_BORDER);
 ctl("STATIC","Local AI uses the embedded WWE 2K26 master prompt, current template and ComfyUI. Output 820x820. A compatible image-editing workflow is required.",30,495,990,35,206,0);
 ctl("STATIC","LOCAL AI FACE TEXTURE (ComfyUI)",30,90,385,28,208,0);
 ctl("STATIC","API workflow JSON:",30,145,270,24,209,0);
 ctl("STATIC","Makeup:",30,191,120,24,210,0);
 ctl("EDIT","workflow_api.json",160,140,240,30,211,WS_BORDER);
 ctl("COMBOBOX","",160,186,240,160,212,CBS_DROPDOWNLIST);
 for _,v:=range []string{"Keep reference appearance","Preserve makeup","Natural / no makeup"}{send.Call(ctrls[212],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(v))))}
 send.Call(ctrls[212],CB_SETCURSEL,0,0)
 ctl("COMBOBOX","",160,228,240,160,213,CBS_DROPDOWNLIST);
 for _,v:=range []string{"Skin-tone background","Transparent background","Seamless skin fill"}{send.Call(ctrls[213],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(v))))}
 send.Call(ctrls[213],CB_SETCURSEL,0,0)
 ctl("EDIT","",30,430,370,48,214,WS_BORDER|ES_MULTILINE);
 ctl("STATIC","Local AI ready. Requires running ComfyUI and a compatible workflow.",30,585,980,42,215,0);
 ctl("BUTTON","Generate with Local AI",30,535,340,38,216,0);
 ctl("STATIC","OFFLINE GRAPHIC GENERATOR",30,110,650,34,301,0);
 ctl("STATIC","Graphic type:",30,170,170,26,302,0);
 ctl("STATIC","Describe colors, stripes, plaid, checkerboard, or stars:",30,240,790,26,303,0);
 ctl("COMBOBOX","",200,165,350,260,304,CBS_DROPDOWNLIST);
 for _,kind:=range []string{"Attire pattern","Wrestler logo","Entrance background","Wrestling sign"}{send.Call(ctrls[304],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(kind))))}
 send.Call(ctrls[304],CB_SETCURSEL,0,0)
 ctl("EDIT","Burnt orange and white plaid",30,275,650,35,305,WS_BORDER);
 ctl("STATIC","Creates patterned PNG assets locally. This is PROCEDURAL generation, not generative AI; logos contain no AI-generated lettering or characters.",30,390,980,70,306,0);
 ctl("BUTTON","Generate Graphic PNG",30,330,280,42,307,0);
 ctl("STATIC","Theme:",635,734,55,24,0,0)
 ctl("COMBOBOX","",690,730,260,190,401,CBS_DROPDOWNLIST)
 for _,label:=range []string{"Original Studio","WWE 2K22 (original if installed)","WWE 2K23 (original if installed)","WWE 2K24 (original if installed)","WWE 2K25 (original if installed)","WWE 2K26 (original if installed)","Custom Image"}{send.Call(ctrls[401],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(label))))}
 send.Call(ctrls[401],CB_SETCURSEL,uintptr(themeIndex),0)
 ctl("BUTTON","Import background",960,730,135,28,402,0)
 ctl("BUTTON","AI Behavior Studio",390,730,175,28,604,0);ctl("BUTTON","Check for Updates",20,775,170,28,600,0)
 ctl("BUTTON","Download Update",200,775,155,28,602,0)
 ctl("BUTTON","Open Download",365,775,145,28,603,0)
 ctl("STATIC","Updates: GitHub Releases | verified ZIP",20,815,500,20,601,0);ctl("BUTTON","Install && Restart",530,810,175,29,605,0)
 // User-supplied original game assets take precedence; fallback is game-inspired.
 dragAccept.Call(hwnd,1);switchTab(0);set(41,"Startup safe mode: template scan paused. Click Refresh Template Library when ready.");show.Call(hwnd,SW_SHOW);update.Call(hwnd);checkUpdates();var msg MSG;for{r,_,_:=getmsg.Call(uintptr(unsafe.Pointer(&msg)),0,0,0);if int32(r)<=0{break};trans.Call(uintptr(unsafe.Pointer(&msg)));dispatch.Call(uintptr(unsafe.Pointer(&msg)))}}
