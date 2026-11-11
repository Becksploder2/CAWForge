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
 "encoding/base64"
 "mime/multipart"
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
type RosterEntry struct {Name string `json:"name"`;Picbase string `json:"picbase"`;Alignment string `json:"alignment"`;Group string `json:"group"`;Mugshot string `json:"mugshot,omitempty"`}
func rosterPath()(string,error){home,e:=os.UserHomeDir();if e!=nil{return "",e};return filepath.Join(home,"Documents","CAWForge","roster.json"),nil}
func rosterRead()([]RosterEntry,error){p,e:=rosterPath();if e!=nil{return nil,e};b,e:=os.ReadFile(p);if os.IsNotExist(e){return []RosterEntry{},nil};if e!=nil{return nil,e};var a []RosterEntry;e=json.Unmarshal(b,&a);if e==nil{sort.Slice(a,func(i,j int)bool{return strings.ToLower(a[i].Name)<strings.ToLower(a[j].Name)})};return a,e}
func rosterSave()error{
 name:=strings.TrimSpace(txt(331));if name==""{return fmt.Errorf("enter a wrestler name")}
 a,e:=rosterRead();if e!=nil{return e}
 v:=RosterEntry{Name:name,Picbase:strings.TrimSpace(txt(332)),Alignment:strings.TrimSpace(txt(333)),Group:strings.TrimSpace(txt(342))}
 found:=false;for i:=range a{if strings.EqualFold(a[i].Name,name)||(rosterEditing!=""&&strings.EqualFold(a[i].Name,rosterEditing)){v.Mugshot=a[i].Mugshot;a[i]=v;found=true;break}}
 if !found{a=append(a,v)}
 sort.Slice(a,func(i,j int)bool{return strings.ToLower(a[i].Name)<strings.ToLower(a[j].Name)})
 p,e:=rosterPath();if e!=nil{return e};if e=os.MkdirAll(filepath.Dir(p),0755);e!=nil{return e}
 b,e:=json.MarshalIndent(a,"","  ");if e!=nil{return e}
 tmp:=p+".tmp";if e=os.WriteFile(tmp,b,0600);e!=nil{return e}
 if e=os.Rename(tmp,p);e!=nil{os.Remove(tmp);return e};rosterEditing=name;return rosterRefresh()
}
var rosterPage int
var rosterEditing string
var rosterMugshotPicker bool
var rosterPhotoBitmaps [4]uintptr
var rosterPhotoTemp [4]string
func rosterPhotoPreview(slot int,path string){
 if slot<0||slot>=4{return}
 if rosterPhotoBitmaps[slot]!=0{syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteObject").Call(rosterPhotoBitmaps[slot]);rosterPhotoBitmaps[slot]=0}
 if rosterPhotoTemp[slot]!=""{os.Remove(rosterPhotoTemp[slot]);rosterPhotoTemp[slot]=""}
 if path==""{send.Call(ctrls[380+slot],STM_SETIMAGE,IMAGE_BITMAP,0);return}
 f,e:=os.Open(path);if e!=nil{return};im,_,e:=image.Decode(f);f.Close();if e!=nil{return}
 const width,height=115,110
 row:=((width*3+3)/4)*4;data:=make([]byte,54+row*height)
 copy(data[:2],[]byte{'B','M'})
 put:=func(i int,v uint32){data[i]=byte(v);data[i+1]=byte(v>>8);data[i+2]=byte(v>>16);data[i+3]=byte(v>>24)}
 put(2,uint32(len(data)));put(10,54);put(14,40);put(18,width);put(22,height);data[26]=1;data[28]=24
 b:=im.Bounds();side:=b.Dx();if b.Dy()<side{side=b.Dy()}
 left:=b.Min.X+(b.Dx()-side)/2;top:=b.Min.Y+(b.Dy()-side)/2
 for y:=0;y<height;y++{for x:=0;x<width;x++{
  sx:=left+x*side/width;sy:=top+(height-1-y)*side/height
  rr,gg,bb,_:=im.At(sx,sy).RGBA();i:=54+y*row+x*3;data[i]=byte(bb>>8);data[i+1]=byte(gg>>8);data[i+2]=byte(rr>>8)
 }}
 p:=filepath.Join(os.TempDir(),fmt.Sprintf("cawforge_roster_%d_%d.bmp",slot,time.Now().UnixNano()))
 if os.WriteFile(p,data,0600)!=nil{return}
 bmp,_,_:=u.NewProc("LoadImageW").Call(0,uintptr(unsafe.Pointer(ptr(p))),IMAGE_BITMAP,0,0,LR_LOADFROMFILE)
 if bmp==0{os.Remove(p);return}
 rosterPhotoBitmaps[slot]=bmp;rosterPhotoTemp[slot]=p
 send.Call(ctrls[380+slot],STM_SETIMAGE,IMAGE_BITMAP,bmp)
}
func rosterGalleryRefresh(a []RosterEntry){
 if len(a)==0{rosterPage=0}
 if rosterPage<0{rosterPage=0}
 if rosterPage>0&&rosterPage*4>=len(a){rosterPage=(len(a)-1)/4}
 for i:=0;i<4;i++{
  j:=rosterPage*4+i
  if j<len(a){set(384+i,a[j].Name);rosterPhotoPreview(i,a[j].Mugshot)}else{set(384+i,"(empty)");rosterPhotoPreview(i,"")}
 }
 total:=(len(a)+3)/4;if total<1{total=1}
 set(389,fmt.Sprintf("Gallery page %d / %d",rosterPage+1,total))
}
func rosterGallerySelect(slot int)error{
 a,e:=rosterRead();if e!=nil{return e}
 j:=rosterPage*4+slot;if j<0||j>=len(a){return fmt.Errorf("no wrestler in this slot")}
 v:=a[j];set(331,v.Name);set(332,v.Picbase);set(333,v.Alignment);set(342,v.Group)
 rosterEditing=v.Name
 set(341,"Selected "+v.Name+". Click Edit CAW to change details.")
 return nil
}
func rosterImportMugshot(path string)error{
 name:=strings.TrimSpace(txt(331));if name==""{return fmt.Errorf("select or save a wrestler first")}
 f,e:=os.Open(path);if e!=nil{return e};_,_,e=image.DecodeConfig(f);f.Close();if e!=nil{return fmt.Errorf("invalid image: %w",e)}
 a,e:=rosterRead();if e!=nil{return e}
 idx:=-1;for i:=range a{if strings.EqualFold(a[i].Name,name){idx=i;break}}
 if idx<0{return fmt.Errorf("save wrestler before adding mugshot")}
 home,e:=os.UserHomeDir();if e!=nil{return e}
 dir:=filepath.Join(home,"Documents","CAWForge","RosterPhotos");if e=os.MkdirAll(dir,0755);e!=nil{return e}
 safe:=strings.Map(func(r rune)rune{if strings.ContainsRune(`\\/:*?"<>|`,r){return '_'};return r},name)
 dest:=filepath.Join(dir,safe+strings.ToLower(filepath.Ext(path)))
 data,e:=os.ReadFile(path);if e!=nil{return e}
 if e=os.WriteFile(dest,data,0600);e!=nil{return e}
 a[idx].Mugshot=dest
 p,e:=rosterPath();if e!=nil{return e}
 b,e:=json.MarshalIndent(a,"","  ");if e!=nil{return e}
 if e=os.WriteFile(p,b,0600);e!=nil{return e}
 return rosterRefresh()
}
func rosterRefresh()error{
 a,e:=rosterRead();if e!=nil{return e}
 box:=ctrls[343];if box==0{return nil}
 send.Call(box,0x014B,0,0) // CB_RESETCONTENT
 for _,v:=range a{send.Call(box,0x0143,0,uintptr(unsafe.Pointer(ptr(v.Name))))} // CB_ADDSTRING
 set(345,fmt.Sprintf("%d wrestlers saved locally",len(a)))
 rosterGalleryRefresh(a)
 return nil
}
func rosterViewSelected()error{
 box:=ctrls[343];if box==0{return fmt.Errorf("roster viewer unavailable")}
 index,_,_:=send.Call(box,CB_GETCURSEL,0,0)
 if int32(index)<0{return fmt.Errorf("select a wrestler from the roster list")}
 a,e:=rosterRead();if e!=nil{return e}
 if int(index)>=len(a){return fmt.Errorf("roster changed; click Refresh Roster")}
 v:=a[index]
 set(331,v.Name);set(332,v.Picbase);set(333,v.Alignment);set(342,v.Group)
 rosterEditing=v.Name;set(341,"Viewing "+v.Name+" — click Edit CAW to change details.")
 return nil
}
func rosterLoad()error{
 name:=strings.TrimSpace(txt(331));if name==""{return fmt.Errorf("enter a wrestler name")}
 a,e:=rosterRead();if e!=nil{return e}
 for _,v:=range a{if strings.EqualFold(v.Name,name){set(331,v.Name);set(332,v.Picbase);set(333,v.Alignment);set(342,v.Group);return nil}}
 return fmt.Errorf("wrestler not found: %s",name)
}

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
 imageWorkGate<-struct{}{}
 defer func(){<-imageWorkGate}()
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
 if len(next.Entries)==0{return next,"Template library empty at "+templateLibraryDir+". Add M1-M20 and F1-F21 screenshots, then Rebuild Index; no calibration performed."};return next,fmt.Sprintf("Ready: %d/41 templates (%d loaded from cache, %d unreadable).",len(next.Entries),reused,skipped)
}
func beginCalibration(){
 if calibrationRunning{return};calibrationRunning=true;set(41,"Indexing available M1-M20 and F1-F21 screenshots...")
 go func(){next,msg:=scanIndex();calMu.Lock();index=next;calibrationMessage=msg;calMu.Unlock();postMessage.Call(hwnd,WM_CAL_DONE,0,0)}()
}
func matchPhoto(){
 if imagePath==""{set(41,"Import a reference photo first.");return}
 if matchRunning{matchQueued=true;return}
 calMu.Lock();entries:=append([]Fingerprint(nil),index.Entries...);calMu.Unlock()
 count:=0;for _,e:=range entries{if strings.HasPrefix(e.ID,gender){count++}}
 if count==0{set(41,"No indexed "+gender+" template images. Required: "+filepath.Join(templateLibraryDir,gender+"1.png")+" (through "+gender+"21). Import real WWE 2K26 template screenshots, then Rebuild Index. No placeholder morph estimates applied.");return}
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
func maxInt(a,b int)int{if a>b{return a};return b}
func generateTexture(photoPath,outPath string)error{
 imageWorkGate<-struct{}{}
 defer func(){<-imageWorkGate}()
 f,err:=os.Open(photoPath);if err!=nil{return err};defer f.Close()
 im,_,err:=image.Decode(f);if err!=nil{return err}
 b:=im.Bounds();w,h:=b.Dx(),b.Dy()
 if w<100||h<100{return fmt.Errorf("photo too small")}
 // Centered rectangular crop, not AI face segmentation or expression editing.
 cropW:=int(float64(w)*0.62);cropH:=int(float64(h)*0.78)
 if cropW>cropH*3/4{cropW=cropH*3/4}
 if cropW<1{cropW=1};if cropH>h{cropH=h}
 cx:=b.Min.X+w/2;cy:=b.Min.Y+h/2
 x0:=cx-cropW/2;y0:=cy-cropH/2
 if x0<b.Min.X{x0=b.Min.X};if y0<b.Min.Y{y0=b.Min.Y}
 if x0+cropW>b.Max.X{x0=b.Max.X-cropW};if y0+cropH>b.Max.Y{y0=b.Max.Y-cropH}
 // Sample skin-color candidates from the crop instead of the full photo.
 // Clamping prevents malformed/out-of-bounds sampling on unusual aspect ratios.
 sample:=[]image.Point{{x0+cropW*35/100,y0+cropH*48/100},{x0+cropW*65/100,y0+cropH*48/100},{x0+cropW/2,y0+cropH*63/100}}
 var sr,sg,sb uint64
 for _,p:=range sample{
  px:=maxInt(b.Min.X,minInt(b.Max.X-1,p.X));py:=maxInt(b.Min.Y,minInt(b.Max.Y-1,p.Y))
  r,g,bb,_:=im.At(px,py).RGBA();sr+=uint64(r>>8);sg+=uint64(g>>8);sb+=uint64(bb>>8)
 }
 bg:=color.RGBA{uint8(sr/uint64(len(sample))),uint8(sg/uint64(len(sample))),uint8(sb/uint64(len(sample))),255}
 dst:=image.NewRGBA(image.Rect(0,0,820,820))
 for y:=0;y<820;y++{for x:=0;x<820;x++{
  // Rectangular, edge-feathered crop: no elliptical halo or circular cutoff.
  // The standard mode is deliberately non-generative: it cannot close a smile
  // or reconstruct skin hidden by hair. Use Local AI for those operations.
  fx:=float64(x)/819;fy:=float64(y)/819
  sx:=x0+minInt(cropW-1,int(fx*float64(cropW-1)))
  sy:=y0+minInt(cropH-1,int(fy*float64(cropH-1)))
  r,g,bb,_:=im.At(sx,sy).RGBA()
  edge:=math.Min(math.Min(float64(x),float64(819-x)),math.Min(float64(y),float64(819-y)))
  alpha:=math.Min(1,math.Max(0,edge/115))
  dst.SetRGBA(x,y,color.RGBA{uint8(alpha*float64(r>>8)+(1-alpha)*float64(bg.R)),uint8(alpha*float64(g>>8)+(1-alpha)*float64(bg.G)),uint8(alpha*float64(bb>>8)+(1-alpha)*float64(bg.B)),255})
 }}
 o,err:=os.Create(outPath);if err!=nil{return err};defer o.Close()
 return png.Encode(o,dst)
}

const faceTexturePrompt = `Create an 820x820 photorealistic WWE 2K26 face texture from the imported reference photograph. Preserve the person's real facial proportions, eye placement, nose shape, lips, makeup, natural skin tone, pores and recognizable details. Perfectly frontal, symmetrical alignment, neutral expression, even neutral studio lighting. Completely remove hair including bangs and side hair, remove ears, keep natural eyebrows, close lips into a relaxed neutral expression with no teeth visible, remove glasses and accessories, no dark jawline or chin shadow, no circular or oval vignette, no visible cutout boundary, no text or background objects. Surround the face with seamlessly blended matching skin texture, maintaining natural realistic pores. Align eyes, nose, mouth, forehead and jaw to the selected in-game face template; prioritize accurate template placement and smooth transitions over stylization. Do not invent or change the person's facial features.`
func writeFacePrompt(path string)error{
 folder:=filepath.Dir(path)
 return os.WriteFile(filepath.Join(folder,"WWE_Face_Texture_Generation_Prompt.txt"),[]byte(faceTexturePrompt+"\r\n"),0644)
}
// Windows STATIC image preview uses an RGB BMP generated from the exported PNG.
// BMP is a temporary preview file; the user-facing output remains PNG.
func writePreviewBMP(path string) (string,error){
 imageWorkGate<-struct{}{}
 defer func(){<-imageWorkGate}()
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


type aiRuntimeConfig struct {
 Python string `json:"python"`
 AIHome string `json:"ai_home"`
 HFHome string `json:"hf_home"`
 Temp string `json:"temp"`
 TorchHome string `json:"torch_home"`
}
func stableDiffusionPython(dir string)string{
 configPath:=filepath.Join(dir,"ai-config.json")
 data,err:=os.ReadFile(configPath)
 if err==nil{
  var cfg aiRuntimeConfig
  if e:=json.Unmarshal(bytes.TrimPrefix(data,[]byte{0xEF,0xBB,0xBF}),&cfg);e!=nil{
   return filepath.Join(dir,"CAWForge_AI_CONFIG_INVALID_"+strings.ReplaceAll(e.Error(),":","_"))
  }
  if cfg.Python==""{return filepath.Join(dir,"CAWForge_AI_CONFIG_MISSING_PYTHON")}
  if info,e:=os.Stat(cfg.Python);e==nil&&!info.IsDir(){return cfg.Python}
  return cfg.Python // Explicit error from process launch; never fall back to stale C: Python.
 }
 // Default to the external AI installation if present, even without config.
 external:=filepath.Join("E:\\","CAWForge-AI",".venv","Scripts","python.exe")
 if info,e:=os.Stat(external);e==nil&&!info.IsDir(){return external}
 venv:=filepath.Join(dir,".venv","Scripts","python.exe")
 if info,e:=os.Stat(venv);e==nil&&!info.IsDir(){return venv}
 return "python"
}
func stableDiffusionEnv(dir string)[]string{
 env:=os.Environ()
 data,err:=os.ReadFile(filepath.Join(dir,"ai-config.json"))
 if err!=nil{return env}
 var cfg aiRuntimeConfig
 if json.Unmarshal(bytes.TrimPrefix(data,[]byte{0xEF,0xBB,0xBF}),&cfg)!=nil{return env}
 for _,pair:=range [][2]string{{"HF_HOME",cfg.HFHome},{"TORCH_HOME",cfg.TorchHome},{"TEMP",cfg.Temp},{"TMP",cfg.Temp}}{
  if pair[1]!=""{env=append(env,pair[0]+"="+pair[1])}
 }
 return env
}
func runAITexture(){
 set(215,"Face extraction is not yet implemented. Stable Diffusion portrait generation has been disabled to protect facial accuracy. See Face Texture Guide.")
 alert("CAWForge Face Texture: Stable Diffusion portrait generation is disabled because it changes facial features and leaves hair, ears and accessories. Use Face Texture Guide for preparation. A calibrated face-extraction pipeline is still required.")
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
// Generate an edited face texture from the actual reference photo.
// The old crop mode produced portraits and could never remove hair or smiles.
func generateEditedFaceTexture(photo,out string)error{
 key:=strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
 if key==""{return fmt.Errorf("real face texture generation needs OPENAI_API_KEY; the old portrait-crop exporter has been disabled")}
 f,err:=os.Open(photo);if err!=nil{return err};defer f.Close()
 var body bytes.Buffer
 writer:=multipart.NewWriter(&body)
 writer.WriteField("model","gpt-image-1")
 writer.WriteField("size","1024x1024")
 writer.WriteField("prompt",faceTexturePrompt+" IMPORTANT: This is a flat game face texture, NOT a finished portrait. Remove all hair and ears, no smile or teeth, remove oval vignette and visible cutout edges, keep the real eyes, brows, nose, lips and skin pores. Fill entire canvas with continuous matching skin. No decorative background.")
 part,err:=writer.CreateFormFile("image",filepath.Base(photo));if err!=nil{return err}
 if _,err=io.Copy(part,f);err!=nil{return err}
 if err=writer.Close();err!=nil{return err}
 ctx,cancel:=context.WithTimeout(context.Background(),4*time.Minute);defer cancel()
 req,err:=http.NewRequestWithContext(ctx,"POST","https://api.openai.com/v1/images/edits",&body);if err!=nil{return err}
 req.Header.Set("Authorization","Bearer "+key);req.Header.Set("Content-Type",writer.FormDataContentType())
 resp,err:=http.DefaultClient.Do(req);if err!=nil{return err};defer resp.Body.Close()
 data,err:=io.ReadAll(io.LimitReader(resp.Body,24<<20));if err!=nil{return err}
 if resp.StatusCode!=200{return fmt.Errorf("face image API %s: %s",resp.Status,strings.TrimSpace(string(data[:minInt(len(data),500)])))}
 var result struct{Data []struct{B64JSON string `json:"b64_json"`}`json:"data"`}
 if err=json.Unmarshal(data,&result);err!=nil{return err}
 if len(result.Data)==0||result.Data[0].B64JSON==""{return fmt.Errorf("image API returned no face texture")}
 raw,err:=base64.StdEncoding.DecodeString(result.Data[0].B64JSON);if err!=nil{return err}
 src,_,err:=image.Decode(bytes.NewReader(raw));if err!=nil{return err}
 bounds:=src.Bounds();dst:=image.NewRGBA(image.Rect(0,0,820,820))
 for y:=0;y<820;y++{for x:=0;x<820;x++{
  sx:=bounds.Min.X+x*bounds.Dx()/820;sy:=bounds.Min.Y+y*bounds.Dy()/820
  dst.Set(x,y,src.At(sx,sy))
 }}
 output,err:=os.Create(out);if err!=nil{return err};defer output.Close()
 return png.Encode(output,dst)
}
func startTexture(){
 if textureBusy{return}
 if imagePath==""{alert("Import a straight-on face photo first.");return}
 textureBusy=true
 photo:=imagePath
 set(203,"Scanning face and preparing 820 x 820 alignment preview...")
 go func(){
  exe,e:=os.Executable()
  var out string
  if e==nil{
   dir:=filepath.Dir(exe)
   script:=filepath.Join(dir,"face_scan_preview.py")
   if _,se:=os.Stat(script);se!=nil{e=fmt.Errorf("face_scan_preview.py missing beside CAWForge.exe: %w",se)}else{
    home,he:=os.UserHomeDir();if he!=nil{e=he}else{
     folder:=filepath.Join(home,"Documents","CAWForge","FaceScans")
     if e=os.MkdirAll(folder,0755);e==nil{
      out=filepath.Join(folder,fmt.Sprintf("Face_Scan_PREVIEW_%d.png",time.Now().UnixNano()))
      ctx,cancel:=context.WithTimeout(context.Background(),90*time.Second)
      cmd:=exec.CommandContext(ctx,stableDiffusionPython(dir),script,photo,out)
      cmd.Env=stableDiffusionEnv(dir)
      logs,ce:=cmd.CombinedOutput();cancel()
      if ce!=nil{e=fmt.Errorf("face scan failed: %v: %s",ce,strings.TrimSpace(string(logs)))}
     }
    }
   }
  }
  matchMu.Lock()
  if e!=nil{faceTextureStatus="Face scan error: "+e.Error()}else{faceTextureStatus="820x820 face alignment PREVIEW: "+out+" | Not a game-ready UV texture; hair/ears and template calibration remain."}
  matchMu.Unlock()
  textureBusy=false
  postMessage.Call(hwnd,WM_MATCH_DONE,0,0)
 }()
}

var seamAuto=true
var seamTop=true
var seamTights=true
var seamWaist=true
// Enforce identical left/right borders with a feathered seam transition.
// Game-specific UV seams still require calibrated UV templates.
func tileSeam(src image.Image)*image.RGBA{
 b:=src.Bounds();w,h:=b.Dx(),b.Dy();dst:=image.NewRGBA(image.Rect(0,0,w,h))
 draw.Draw(dst,dst.Bounds(),src,b.Min,draw.Src)
 band:=w/10
 for y:=0;y<h;y++{
  a:=color.RGBAModel.Convert(src.At(b.Min.X,b.Min.Y+y)).(color.RGBA)
  z:=color.RGBAModel.Convert(src.At(b.Min.X+w-1,b.Min.Y+y)).(color.RGBA)
  mid:=color.RGBA{uint8((uint16(a.R)+uint16(z.R))/2),uint8((uint16(a.G)+uint16(z.G))/2),uint8((uint16(a.B)+uint16(z.B))/2),uint8((uint16(a.A)+uint16(z.A))/2)}
  for i:=0;i<band;i++{
   t:=float64(band-i)/float64(band)
   blend:=func(v uint8,target uint8)uint8{return uint8(float64(v)*(1-t)+float64(target)*t+0.5)}
   for _,x:=range []int{i,w-1-i}{
    v:=color.RGBAModel.Convert(src.At(b.Min.X+x,b.Min.Y+y)).(color.RGBA)
    dst.SetRGBA(x,y,color.RGBA{blend(v.R,mid.R),blend(v.G,mid.G),blend(v.B,mid.B),blend(v.A,mid.A)})
   }
  }
  dst.SetRGBA(0,y,mid);dst.SetRGBA(w-1,y,mid)
 }
 return dst
}
func exportSeamPanels(src image.Image,dir,stem string)error{
 for _,p:=range []struct{name string;enabled bool}{{"top",seamTop},{"tights",seamTights},{"waistband",seamWaist}}{
  if !p.enabled{continue}
  f,e:=os.Create(filepath.Join(dir,stem+"_"+p.name+"_seamless.png"));if e!=nil{return e}
  e=png.Encode(f,tileSeam(src));ce:=f.Close();if e!=nil{return e};if ce!=nil{return ce}
 }
 return nil
}
// Render Creator retains source RGB values; only alpha changes.
// This conservative border-connected background mask is intended for plain CAS backdrops.
func removeRenderBackground(path string)(string,error){
 f,e:=os.Open(path);if e!=nil{return "",e};src,_,e:=image.Decode(f);f.Close();if e!=nil{return "",e}
 b:=src.Bounds();w,h:=b.Dx(),b.Dy();if w<16||h<16||int64(w)*int64(h)>32000000{return "",fmt.Errorf("unsupported screenshot size")}
 img:=image.NewNRGBA(image.Rect(0,0,w,h));draw.Draw(img,img.Bounds(),src,b.Min,draw.Src)
 type rgb struct{r,g,b int}
 get:=func(i int)rgb{j:=i*4;return rgb{int(img.Pix[j]),int(img.Pix[j+1]),int(img.Pix[j+2])}}
 dist:=func(a,b rgb)int{dr:=a.r-b.r;dg:=a.g-b.g;db:=a.b-b.b;return dr*dr+dg*dg+db*db}
 corners:=[]rgb{get(0),get(w-1),get((h-1)*w),get(w*h-1)}
 near:=func(p rgb)bool{for _,c:=range corners{if dist(p,c)<24*24{return true}};return false}
 mask:=make([]bool,w*h);queue:=make([]int,0,w*h/4)
 seed:=func(i int){if !mask[i]&&near(get(i)){mask[i]=true;queue=append(queue,i)}}
 for x:=0;x<w;x++{seed(x);seed((h-1)*w+x)}
 for y:=0;y<h;y++{seed(y*w);seed(y*w+w-1)}
 for head:=0;head<len(queue);head++{
  i:=queue[head];y:=i/w
  for _,j:=range []int{i-1,i+1,i-w,i+w}{
   if j<0||j>=w*h||mask[j]||((j==i-1||j==i+1)&&j/w!=y){continue}
   if near(get(j))&&dist(get(j),get(i))<28*28{mask[j]=true;queue=append(queue,j)}
  }
 }
 if len(queue)<w*h/100{return "",fmt.Errorf("background not confidently detected; use a plain contrasting CAS backdrop")}
 if len(queue)>w*h*85/100{return "",fmt.Errorf("background selection covers most of screenshot; refusing destructive cutout")}
 for i,removed:=range mask{if removed{img.Pix[i*4+3]=0}}
 name:=strings.TrimSuffix(filepath.Base(path),filepath.Ext(path))
 out:=filepath.Join(filepath.Dir(path),name+"_caw_transparent.png")
 dest,e:=os.Create(out);if e!=nil{return "",e};e=png.Encode(dest,img);ce:=dest.Close();if e!=nil{return "",e};if ce!=nil{return "",ce}
 return out,nil
}
// A local segmentation helper can produce an alpha mask without regenerating
// the wrestler. Expected command: render-segmenter.exe <input> <output.png>.
// Its output alpha is applied to ORIGINAL screenshot RGB values.
func segmentRenderBackground(path string)(string,error){
 exe,err:=os.Executable();if err!=nil{return "",err}
 helper:=filepath.Join(filepath.Dir(exe),"render-segmenter.exe")
 if _,err:=os.Stat(helper);err!=nil{return "",fmt.Errorf("AI cutout requires render-segmenter.exe beside CAWForge; complex backgrounds cannot be removed by the color-only fallback")}
 sourceFile,err:=os.Open(path);if err!=nil{return "",err}
 source,_,err:=image.Decode(sourceFile);sourceFile.Close();if err!=nil{return "",err}
 b:=source.Bounds();w,h:=b.Dx(),b.Dy()
 if w<16||h<16||int64(w)*int64(h)>32000000{return "",fmt.Errorf("unsupported screenshot size")}
 tmp,err:=os.CreateTemp("","cawforge-mask-*.png");if err!=nil{return "",err}
 maskPath:=tmp.Name();tmp.Close();os.Remove(maskPath);defer os.Remove(maskPath)
 ctx,cancel:=context.WithTimeout(context.Background(),4*time.Minute);defer cancel()
 cmd:=exec.CommandContext(ctx,helper,path,maskPath)
 logs,err:=cmd.CombinedOutput()
 if err!=nil{return "",fmt.Errorf("segmentation failed: %v: %s",err,string(logs))}
 f,err:=os.Open(maskPath);if err!=nil{return "",fmt.Errorf("segmenter did not write mask: %w",err)}
 mask,_,err:=image.Decode(f);f.Close();if err!=nil{return "",err}
 if mask.Bounds().Dx()!=w||mask.Bounds().Dy()!=h{return "",fmt.Errorf("segmentation mask dimensions do not match source screenshot")}
 result:=image.NewNRGBA(image.Rect(0,0,w,h))
 draw.Draw(result,result.Bounds(),source,b.Min,draw.Src)
 kept:=0
 for y:=0;y<h;y++{for x:=0;x<w;x++{
  _,_,_,alpha:=mask.At(mask.Bounds().Min.X+x,mask.Bounds().Min.Y+y).RGBA()
  a:=uint8(alpha>>8)
  result.Pix[y*result.Stride+x*4+3]=a
  if a>127{kept++}
 }}
 if kept<w*h/100||kept>w*h*85/100{return "",fmt.Errorf("segmentation mask failed sanity check; original image remains untouched")}
 name:=strings.TrimSuffix(filepath.Base(path),filepath.Ext(path))
 out:=filepath.Join(filepath.Dir(path),name+"_caw_transparent.png")
 f,err=os.Create(out);if err!=nil{return "",err}
 err=png.Encode(f,result);closeErr:=f.Close();if err!=nil{return "",err};if closeErr!=nil{return "",closeErr}
 return out,nil
}
// Lifelike rendering uses an actual optional image-editing worker. No image
// enhancement is claimed when the worker is absent.
func generateLifelikeRender(source string,realism int)(string,error){
 exe,err:=os.Executable();if err!=nil{return "",err}
 dir:=filepath.Dir(exe)
 sdScript:=filepath.Join(dir,"sd_backend.py")
 if info,e:=os.Stat(sdScript);e==nil&&!info.IsDir(){
  if realism<0||realism>100{return "",fmt.Errorf("realism must be 0-100")}
  home,_:=os.UserHomeDir();if home==""{home=os.TempDir()}
  outdir:=filepath.Join(home,"Documents","CAWForge","Renders")
  if e=os.MkdirAll(outdir,0755);e!=nil{return "",e}
  out:=filepath.Join(outdir,fmt.Sprintf("CAW_SD_Render_%d.png",time.Now().UnixNano()))
  ctx,cancel:=context.WithTimeout(context.Background(),20*time.Minute);defer cancel()
  cmd:=exec.CommandContext(ctx,stableDiffusionPython(dir),sdScript,"--mode","render","--input",source,"--output",out,"--realism",strconv.Itoa(realism))
  cmd.Env=stableDiffusionEnv(dir)
  logs,e:=cmd.CombinedOutput();if e!=nil{return "",fmt.Errorf("Stable Diffusion render failed: %v: %s",e,strings.TrimSpace(string(logs)))}
  f,e:=os.Open(out);if e!=nil{return "",fmt.Errorf("render file missing: %w",e)}
  cfg,format,e:=image.DecodeConfig(f);f.Close()
  if e!=nil||format!="png"||cfg.Width<512||cfg.Height<512{return "",fmt.Errorf("Stable Diffusion render output is invalid")}
  return out,nil
 }
 helper:=filepath.Join(filepath.Dir(exe),"render-enhancer.exe")
 if info,statErr:=os.Stat(helper);statErr!=nil||info.IsDir(){return "",fmt.Errorf("Lifelike mode requires render-enhancer.exe and its compatible image model beside CAWForge; source unchanged")}
 if realism<0||realism>100{return "",fmt.Errorf("realism must be 0-100")}
 name:=strings.TrimSuffix(filepath.Base(source),filepath.Ext(source))
 out:=filepath.Join(filepath.Dir(source),name+"_lifelike_"+strconv.Itoa(realism)+".png")
 if _,err:=os.Stat(out);err==nil{return "",fmt.Errorf("output already exists: %s (original untouched)",out)}
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Minute);defer cancel()
 cmd:=exec.CommandContext(ctx,helper,"--input",source,"--output",out,"--realism",strconv.Itoa(realism),"--preserve-identity","--transparent")
 logs,err:=cmd.CombinedOutput()
 if err!=nil{os.Remove(out);return "",fmt.Errorf("render enhancement failed: %v: %s",err,strings.TrimSpace(string(logs)))}
 f,err:=os.Open(out);if err!=nil{return "",fmt.Errorf("enhancer did not write PNG: %w",err)}
 img,err:=png.Decode(f);f.Close()
 if err!=nil{os.Remove(out);return "",fmt.Errorf("enhancer output is invalid PNG: %w",err)}
 b:=img.Bounds();if b.Dx()<512||b.Dy()<512{os.Remove(out);return "",fmt.Errorf("enhanced image is too small")}
 hasAlpha:=false
 for y:=b.Min.Y;y<b.Max.Y&&!hasAlpha;y+=maxInt(1,b.Dy()/50){for x:=b.Min.X;x<b.Max.X;x+=maxInt(1,b.Dx()/50){_,_,_,a:=img.At(x,y).RGBA();if a<65535{hasAlpha=true;break}}}
 if !hasAlpha{os.Remove(out);return "",fmt.Errorf("enhancer did not provide transparency")}
 return out,nil
}
// Faithful Render always preserves the original RGB pixels and generates transparency.
// Never route the default renderer through Stable Diffusion: it changes identity and attire.
func startRenderCutout(){
 if renderImagePath==""{alert("Import a CAW screenshot first.");return}
 if renderBusy{return}
 renderBusy=true
 realism:=0
 studioPro:=false
 if ctrls[325]!=0{selected,_,_:=send.Call(ctrls[325],CB_GETCURSEL,0,0);studioPro=selected==2}
 if ctrls[327]!=0{v,_,_:=send.Call(ctrls[327],TBM_GETPOS,0,0);realism=int(v)}
 set(322,fmt.Sprintf("Separating wrestler from background; photographic finish %d%%...",realism))
 path:=renderImagePath
 go func(){
  exe,exeErr:=os.Executable()
  var output string
  var err error
  if exeErr!=nil{err=exeErr}else{
   dir:=filepath.Dir(exe)
   script:=filepath.Join(dir,"render_cutout.py")
   if _,e:=os.Stat(script);e!=nil{err=fmt.Errorf("render_cutout.py not found at %q: %v. Put the script beside the running executable (%q), not just in a CAWForge project folder",script,e,exe)}else{
    home,_:=os.UserHomeDir();if home==""{home=os.TempDir()}
    outdir:=filepath.Join(home,"Documents","CAWForge","Renders")
    if e=os.MkdirAll(outdir,0755);e!=nil{err=e}else{
     prefix:="Faithful_Render"
     if studioPro{prefix="Studio_Pro"}
     output=filepath.Join(outdir,fmt.Sprintf("%s_%d.png",prefix,time.Now().UnixNano()))
     ctx,cancel:=context.WithTimeout(context.Background(),5*time.Minute)
     modeFlag:="--studio"
     if studioPro{modeFlag="--studio-pro"}
     cmd:=exec.CommandContext(ctx,stableDiffusionPython(dir),script,path,output,"--realism",strconv.Itoa(realism),modeFlag)
     cmd.Env=stableDiffusionEnv(dir)
     logs,e:=cmd.CombinedOutput();cancel()
     if e!=nil{err=fmt.Errorf("foreground segmentation failed: %v: %s",e,strings.TrimSpace(string(logs)))}
    }
   }
  }
  matchMu.Lock()
  if err!=nil{renderStatus="Render unavailable: "+err.Error()+" (source preserved)"}else{renderStatus="Transparent render saved: "+output+" | Photographic finishing "+strconv.Itoa(realism)+"%. Inspect hair edges."}
  matchMu.Unlock()
  postMessage.Call(hwnd,WM_RENDER_DONE,0,0)
 }()
}
var renderImagePath string
var renderBusy bool
var renderStatus string
const WM_RENDER_DONE=0x8029
// exportSignSheet converts a supplied 3x4 reference sheet into a reusable 12-sign atlas.
// Use prompt: sheet:C:\\path\\to\\sheet.png . No artwork is synthesized or altered.
func exportSignSheet(description string)(string,error){
 source:=strings.TrimSpace(description)
 if !strings.HasPrefix(strings.ToLower(source),"sheet:"){return "",fmt.Errorf("Sign Export splits an EXISTING 3x4 image sheet; it does not generate signs from text yet. Select a 3x4 PNG/JPG and enter sheet:C:\\path\\to\\signs.png. Your description was not treated as a generation prompt.")}
 source=strings.Trim(strings.TrimSpace(source[6:]),"\"")
 if source==""{return "",fmt.Errorf("missing sheet path: enter sheet:C:\\path\\to\\signs.png")}
 f,err:=os.Open(source);if err!=nil{return "",fmt.Errorf("cannot open 3x4 sign sheet %q: %w",source,err)}
 defer f.Close()
 src,_,err:=image.Decode(f);if err!=nil{return "",err}
 bounds:=src.Bounds();w,h:=bounds.Dx(),bounds.Dy()
 if w<300||h<400{return "",fmt.Errorf("sign sheet is too small (%dx%d)",w,h)}
 out:=image.NewNRGBA(image.Rect(0,0,1536,1024))
 // Nearest-neighbor preserves source pixels at the reference dimensions;
 // tiles are split at exact thirds/quarters of the supplied sheet.
 for y:=0;y<1024;y++{sy:=bounds.Min.Y+y*h/1024;for x:=0;x<1536;x++{sx:=bounds.Min.X+x*w/1536;out.Set(x,y,src.At(sx,sy))}}
 home,err:=os.UserHomeDir();if err!=nil{return "",err}
 dir:=filepath.Join(home,"Documents","CAWForge","SignSheets")
 if err=os.MkdirAll(dir,0755);err!=nil{return "",err}
 stem:=strconv.FormatInt(time.Now().UnixNano(),10)
 output:=filepath.Join(dir,"sign_sheet_12_"+stem+".png")
 save:=func(path string,img image.Image)error{f,e:=os.Create(path);if e!=nil{return e};e=png.Encode(f,img);ce:=f.Close();if e!=nil{return e};return ce}
 if err=save(output,out);err!=nil{return "",err}
 for row:=0;row<4;row++{for col:=0;col<3;col++{
  // 512x512 square texture; preserve the rectangular sign aspect ratio by centering.
  tile:=image.NewNRGBA(image.Rect(0,0,512,512))
  draw.Draw(tile,image.Rect(0,128,512,384),out,image.Point{X:col*512,Y:row*256},draw.Src)
  if err=save(filepath.Join(dir,fmt.Sprintf("sign_%02d_%s.png",row*3+col+1,stem)),tile);err!=nil{return "",err}
 }}
 return output,nil
}
func proceduralGraphic(kind,description string) (string,error) {
 desc:=strings.ToLower(description)
 width,height:=1024,1024
 if kind=="Wrestler logo"{width,height=512,512}
 if kind=="Entrance background"{width,height=1280,720}
 if kind=="Wrestling sign"{width,height=1024,512}
 img:=image.NewRGBA(image.Rect(0,0,width,height))
 // Optional transparent base for wearable graphics and logos.
 transparent:=kind=="Wrestler logo"
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
 // No recognized motif means a solid color, never surprise stripes.
 for y:=0;y<height;y++{for x:=0;x<width;x++{
  v:=primary
  // Solid base unless a requested pattern overrides it.
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
 if kind=="Attire pattern"&&seamAuto{if err:=exportSeamPanels(img,dir,strings.TrimSuffix(filepath.Base(out),".png"));err!=nil{return "",err}}
 return out,nil
}
// generateAIAttire uses the OpenAI Images API. No credentials are stored in
// the app; set OPENAI_API_KEY in the user's environment before launching.
// Optional local attire generator contract. The helper must be a real installed
// inference engine; CAWForge does not silently substitute a colored grid.
func localAttireEngine()string{
 exe,err:=os.Executable();if err!=nil{return ""}
 path:=filepath.Join(filepath.Dir(exe),"attire-engine.exe")
 if info,err:=os.Stat(path);err==nil&&!info.IsDir(){return path}
 return ""
}
func generateLocalAIAttire(description string)(string,error){
 engine:=localAttireEngine()
 exe,_:=os.Executable();sdScript:=filepath.Join(filepath.Dir(exe),"sd_backend.py")
 _,sdErr:=os.Stat(sdScript)
 if engine==""&&sdErr!=nil{return "",fmt.Errorf("install sd_backend.py with Python dependencies or attire-engine.exe")}
 home,err:=os.UserHomeDir();if err!=nil{return "",err}
 dir:=filepath.Join(home,"Documents","CAWForge","AIAttire")
 if err=os.MkdirAll(dir,0755);err!=nil{return "",err}
 stamp:=strconv.FormatInt(time.Now().UnixNano(),10)
 for _,piece:=range []string{"top","tights","waistband"}{
  out:=filepath.Join(dir,stamp+"_"+piece+"_concept.png")
  ctx,cancel:=context.WithTimeout(context.Background(),5*time.Minute)
  var cmd *exec.Cmd
  if sdErr==nil{cmd=exec.CommandContext(ctx,stableDiffusionPython(filepath.Dir(exe)),sdScript,"--mode","attire","--piece",piece,"--prompt",description,"--output",out);cmd.Env=stableDiffusionEnv(filepath.Dir(exe))}else{cmd=exec.CommandContext(ctx,engine,"--piece",piece,"--prompt",description,"--output",out)}
  output,runErr:=cmd.CombinedOutput();cancel()
  if runErr!=nil{return "",fmt.Errorf("local attire engine failed for %s: %v: %s",piece,runErr,strings.TrimSpace(string(output)))}
  file,openErr:=os.Open(out);if openErr!=nil{return "",fmt.Errorf("engine did not create %s: %w",out,openErr)}
  imageData,decodeErr:=png.Decode(file);file.Close()
  if decodeErr!=nil{return "",fmt.Errorf("engine output for %s is not a valid PNG: %w",piece,decodeErr)}
  if imageData.Bounds().Dx()<512||imageData.Bounds().Dy()<512{return "",fmt.Errorf("engine output for %s is smaller than 512 pixels",piece)}
 }
 return dir,nil
}
func generateAIAttire(description string)(string,error){
 key:=strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
 if key==""{return "",fmt.Errorf("AI attire requires OPENAI_API_KEY. Offline patterns cannot generate outfits")}
 home,err:=os.UserHomeDir();if err!=nil{return "",err}
 dir:=filepath.Join(home,"Documents","CAWForge","AIAttire")
 if err=os.MkdirAll(dir,0755);err!=nil{return "",err}
 stamp:=strconv.FormatInt(time.Now().UnixNano(),10)
 for _,piece:=range []string{"top","tights","waistband"}{
  prompt:=fmt.Sprintf("Create a high quality, FLAT 2D printable professional wrestling attire TEXTURE DESIGN for the %s. Design brief: %s. Output a single square artwork panel, orthographic graphic only, no human body, no wrestler, no mannequin, no mockup, no text labels, no background scenery. Coordinate colors and motifs across the complete attire. This is concept art, not a game UV template.",piece,description)
  payload,_:=json.Marshal(map[string]interface{}{"model":"gpt-image-1","prompt":prompt,"size":"1024x1024","n":1})
  ctx,cancel:=context.WithTimeout(context.Background(),3*time.Minute)
  req,err:=http.NewRequestWithContext(ctx,"POST","https://api.openai.com/v1/images/generations",bytes.NewReader(payload))
  if err!=nil{cancel();return "",err}
  req.Header.Set("Authorization","Bearer "+key);req.Header.Set("Content-Type","application/json")
  res,err:=http.DefaultClient.Do(req)
  if err!=nil{cancel();return "",err}
  data,readErr:=io.ReadAll(io.LimitReader(res.Body,24<<20));res.Body.Close();cancel()
  if readErr!=nil{return "",readErr}
  if res.StatusCode!=200{return "",fmt.Errorf("image API %s: %s",res.Status,strings.TrimSpace(string(data[:minInt(len(data),600)])))}
  var response struct{Data []struct{B64JSON string `json:"b64_json"`}`json:"data"`}
  if err=json.Unmarshal(data,&response);err!=nil{return "",err}
  if len(response.Data)==0||response.Data[0].B64JSON==""{return "",fmt.Errorf("image API returned no PNG for %s",piece)}
  pngData,err:=base64.StdEncoding.DecodeString(response.Data[0].B64JSON);if err!=nil{return "",err}
  if _,err=png.Decode(bytes.NewReader(pngData));err!=nil{return "",fmt.Errorf("invalid PNG for %s: %w",piece,err)}
  out:=filepath.Join(dir,stamp+"_"+piece+"_concept.png")
  if err=os.WriteFile(out,pngData,0600);err!=nil{return "",err}
 }
 return dir,nil
}
// generatePlacementTestAtlas creates an explicitly provisional combined UV placement sheet.
// The colored panel boxes are test coordinates, NOT verified WWE 2K26 UV islands.
func generatePlacementTestAtlas(description string)(string,error){
 const size=2048
 img:=image.NewNRGBA(image.Rect(0,0,size,size))
 panels:=[]struct{Name string;X,Y,W,H int;R,G,B uint8}{
  {"top_front",80,80,900,620,210,55,70},
  {"top_back",1068,80,900,620,65,125,210},
  {"tights_front",80,790,900,930,50,175,125},
  {"tights_back",1068,790,900,930,175,95,205},
  {"waistband",80,1810,1888,140,225,170,55},
 }
 type region struct{Name string `json:"name"`; X int `json:"x"`;Y int `json:"y"`;W int `json:"width"`;H int `json:"height"`}
 regions:=make([]region,0,len(panels))
 for _,p:=range panels{
  regions=append(regions,region{p.Name,p.X,p.Y,p.W,p.H})
  for y:=p.Y;y<p.Y+p.H;y++{for x:=p.X;x<p.X+p.W;x++{
   col:=color.NRGBA{p.R,p.G,p.B,255}
   if (x-p.X)%100<5||(y-p.Y)%100<5{col=color.NRGBA{255,255,255,255}}
   if x-p.X<9||p.X+p.W-x<=9||y-p.Y<9||p.Y+p.H-y<=9{col=color.NRGBA{255,255,255,255}}
   if (x-p.X-p.W/2)*(x-p.X-p.W/2)+(y-p.Y-p.H/2)*(y-p.Y-p.H/2)<2500{col=color.NRGBA{0,0,0,255}}
   img.SetNRGBA(x,y,col)
  }}
 }
 home,err:=os.UserHomeDir();if err!=nil{return "",err}
 dir:=filepath.Join(home,"Documents","CAWForge","AttirePlacementTests")
 if err=os.MkdirAll(dir,0755);err!=nil{return "",err}
 stem:=strconv.FormatInt(time.Now().UnixNano(),10)
 path:=filepath.Join(dir,"combined_uv_TEST_"+stem+".png")
 f,err:=os.Create(path);if err!=nil{return "",err}
 err=png.Encode(f,img);closeErr:=f.Close();if err!=nil{return "",err};if closeErr!=nil{return "",closeErr}
 manifest:=map[string]interface{}{"status":"PROVISIONAL TEST - NOT WWE 2K26 UV CALIBRATED","width":size,"height":size,"description":description,"regions":regions}
 data,err:=json.MarshalIndent(manifest,"","  ");if err!=nil{return "",err}
 if err=os.WriteFile(filepath.Join(dir,"combined_uv_TEST_"+stem+".json"),data,0644);err!=nil{return "",err}
 return path,nil
}

// The initial seam test deliberately refuses to invent UV adjacency. It validates
// the imported image and writes a lossless copy for in-game baseline comparison.
// Calibrated seam pairs are specified in source-image pixels. No guessed WWE UV coordinates.
type calibratedSeamPoint struct { AX int `json:"ax"`; AY int `json:"ay"`; BX int `json:"bx"`; BY int `json:"by"` }
type calibratedSeamEdge struct { Name string `json:"name"`; Width int `json:"width"`; Points []calibratedSeamPoint `json:"points"` }
type calibratedSeamMap struct { Width int `json:"width"`; Height int `json:"height"`; Edges []calibratedSeamEdge `json:"edges"` }
func stitchCalibratedSeams(src image.Image,configPath string)(image.Image,int,error){
 data,err:=os.ReadFile(configPath);if err!=nil{return nil,0,err}
 var mapping calibratedSeamMap
 if err=json.Unmarshal(data,&mapping);err!=nil{return nil,0,fmt.Errorf("invalid seam mapping JSON: %w",err)}
 b:=src.Bounds();w,h:=b.Dx(),b.Dy()
 if mapping.Width!=w||mapping.Height!=h{return nil,0,fmt.Errorf("mapping expects %dx%d but texture is %dx%d",mapping.Width,mapping.Height,w,h)}
 if len(mapping.Edges)==0{return nil,0,fmt.Errorf("mapping has no calibrated edges")}
 out:=image.NewNRGBA(image.Rect(0,0,w,h));draw.Draw(out,out.Bounds(),src,b.Min,draw.Src)
 original:=image.NewNRGBA(image.Rect(0,0,w,h));draw.Draw(original,original.Bounds(),src,b.Min,draw.Src)
 touched:=make([]bool,w*h)
 edgesDone:=0
 for _,edge:=range mapping.Edges{
  if len(edge.Points)<2{return nil,0,fmt.Errorf("seam %q requires at least two measured point pairs",edge.Name)}
  if edge.Width<1||edge.Width>32{return nil,0,fmt.Errorf("seam %q feather width must be 1..32 pixels",edge.Name)}
  for _,p:=range edge.Points{for _,xy:=range [][2]int{{p.AX,p.AY},{p.BX,p.BY}}{if xy[0]<0||xy[0]>=w||xy[1]<0||xy[1]>=h{return nil,0,fmt.Errorf("seam %q has point outside texture",edge.Name)}}}
  for i:=1;i<len(edge.Points);i++{
   a,c:=edge.Points[i-1],edge.Points[i]
   steps:=int(math.Max(math.Hypot(float64(c.AX-a.AX),float64(c.AY-a.AY)),math.Hypot(float64(c.BX-a.BX),float64(c.BY-a.BY))))*2
   if steps<1{steps=1}
   for step:=0;step<=steps;step++{
    t:=float64(step)/float64(steps)
    ax:=int(math.Round(float64(a.AX)+float64(c.AX-a.AX)*t));ay:=int(math.Round(float64(a.AY)+float64(c.AY-a.AY)*t))
    bx:=int(math.Round(float64(a.BX)+float64(c.BX-a.BX)*t));by:=int(math.Round(float64(a.BY)+float64(c.BY-a.BY)*t))
    ca:=original.NRGBAAt(ax,ay);cb:=original.NRGBAAt(bx,by)
    if ca.A==0||cb.A==0{continue} // preserve transparent artwork
    avg:=color.NRGBA{R:uint8((int(ca.R)+int(cb.R))/2),G:uint8((int(ca.G)+int(cb.G))/2),B:uint8((int(ca.B)+int(cb.B))/2),A:uint8((int(ca.A)+int(cb.A))/2)}
    for _,center:=range [][2]int{{ax,ay},{bx,by}}{
     for dy:=-edge.Width;dy<=edge.Width;dy++{for dx:=-edge.Width;dx<=edge.Width;dx++{
      x,y:=center[0]+dx,center[1]+dy;if x<0||x>=w||y<0||y>=h{continue}
      distance:=math.Hypot(float64(dx),float64(dy));if distance>float64(edge.Width){continue}
      index:=y*w+x;if touched[index]{continue}
      old:=original.NRGBAAt(x,y);if old.A==0{continue}
      factor:=1-distance/float64(edge.Width)
      blend:=func(v,target uint8)uint8{return uint8(math.Round(float64(v)*(1-factor)+float64(target)*factor))}
      out.SetNRGBA(x,y,color.NRGBA{R:blend(old.R,avg.R),G:blend(old.G,avg.G),B:blend(old.B,avg.B),A:old.A})
     }}
    }
   }
  }
  edgesDone++
 }
 return out,edgesDone,nil
}
func startImportedSeamTest(){
 source:=strings.Trim(strings.TrimSpace(txt(370)),"\"")
 if source==""{set(371,"Import an attire PNG first.");return}
 if !seamAuto{set(371,"Enable Auto Seam Connect before exporting a seam baseline.");return}
 set(371,"Validating imported attire texture for baseline export...")
 go func(){
  f,err:=os.Open(source)
  if err!=nil{matchMu.Lock();graphicStatus="Import failed: "+err.Error();matchMu.Unlock();postMessage.Call(hwnd,WM_GRAPHIC_DONE,0,0);return}
  img,_,decodeErr:=image.Decode(f);f.Close()
  if decodeErr!=nil{matchMu.Lock();graphicStatus="Cannot decode imported texture: "+decodeErr.Error();matchMu.Unlock();postMessage.Call(hwnd,WM_GRAPHIC_DONE,0,0);return}
  home,_:=os.UserHomeDir();dir:=filepath.Join(home,"Documents","CAWForge","AttireSeamTests")
  err=os.MkdirAll(dir,0755)
  out:=filepath.Join(dir,strings.TrimSuffix(filepath.Base(source),filepath.Ext(source))+"_SEAM_BASELINE.png")
  if err==nil{var file *os.File;file,err=os.Create(out);if err==nil{err=png.Encode(file,img);closeErr:=file.Close();if err==nil{err=closeErr}}}
  status:="Imported attire saved as unchanged baseline: "+out+" | No calibrated seam map yet."
  config:=strings.TrimSuffix(source,filepath.Ext(source))+"_seams.json"
  if err==nil{if _,statErr:=os.Stat(config);statErr==nil{
   var stitched image.Image;var count int
   stitched,count,err=stitchCalibratedSeams(img,config)
   if err==nil{
    stitchedPath:=filepath.Join(dir,strings.TrimSuffix(filepath.Base(source),filepath.Ext(source))+"_CALIBRATED_SEAMS.png")
    var file *os.File;file,err=os.Create(stitchedPath)
    if err==nil{err=png.Encode(file,stitched);closeErr:=file.Close();if err==nil{err=closeErr}}
    if err==nil{out=stitchedPath;status=fmt.Sprintf("Calibrated seam export: %d mapped edges | %s",count,out)}
   }
  }}
  matchMu.Lock()
  if err!=nil{graphicStatus="Seam export failed (original baseline preserved): "+err.Error()}else{lastGraphicPNG=out;graphicStatus=status}
  matchMu.Unlock();postMessage.Call(hwnd,WM_GRAPHIC_DONE,0,0)
 }()
}

func startGraphic(){
 if graphicBusy{return}
 kindIndex,_,_:=send.Call(ctrls[304],CB_GETCURSEL,0,0)
 kinds:=[]string{"","Attire pattern","Wrestler logo","Entrance background","Wrestling sign"}
 if kindIndex==0||int(kindIndex)>=len(kinds){alert("Choose a graphic category first: attire, logo, entrance background, or fan sign.");return}
 kind:=kinds[kindIndex]
 prompt:=strings.TrimSpace(txt(305))
 if prompt==""{alert("Describe the selected graphic first.");return}
 if kind=="Wrestling sign"{
  graphicBusy=true;set(306,"Exporting 3x4 sign sheet and 12 individual signs...")
  go func(){path,err:=exportSignSheet(prompt);matchMu.Lock();if err!=nil{graphicStatus="Sign export failed: "+err.Error()}else{graphicStatus="Exported 12-sign atlas and individual signs: "+path};matchMu.Unlock();postMessage.Call(hwnd,WM_GRAPHIC_DONE,0,0)}()
  return
 }
 if kind=="Attire pattern"{
  local:=localAttireEngine()!=""
  if exe,e:=os.Executable();e==nil{if _,e=os.Stat(filepath.Join(filepath.Dir(exe),"sd_backend.py"));e==nil{local=true}}
  cloud:=strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))!=""
  if !local&&!cloud{
   set(306,"No attire image engine installed. For EXISTING artwork, use Import Existing Attire PNG and Export Seam Baseline; no API key needed. New AI artwork needs attire-engine.exe or OPENAI_API_KEY.")
   return
  }
  graphicBusy=true
  if local{set(306,"Generating 3 local AI attire concept panels...")}else{set(306,"Generating 3 cloud AI attire concept panels...")}
  go func(){
   var path string;var err error
   if local{path,err=generateLocalAIAttire(prompt)}else{path,err=generateAIAttire(prompt)}
   matchMu.Lock()
   if err!=nil{graphicStatus="Attire artwork generation failed: "+err.Error()}else{graphicStatus="Three attire ARTWORK CONCEPTS saved in: "+path+" | Not yet game-ready UV textures; exact UV mapping requires a real atlas."}
   matchMu.Unlock()
   postMessage.Call(hwnd,WM_GRAPHIC_DONE,0,0)
  }()
  return
 }
 graphicBusy=true
 if kind=="Attire pattern" && strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))!=""{set(306,"Generating three cloud AI attire concept panels...")}else if kind=="Attire pattern"{set(306,"No OpenAI key: generating basic offline attire pattern and seam panels (not AI or game UV).")}else{set(306,"Generating offline graphic...")}
 go func(){
  var path string;var err error
   if kind=="Attire pattern" && strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))!=""{path,err=generateAIAttire(prompt)}else{path,err=proceduralGraphic(kind,prompt)}
  matchMu.Lock()
  if err!=nil{graphicStatus="Generation failed: "+err.Error()}else{lastGraphicPNG=path;graphicStatus="Saved to: "+path+" | Logo export is 512x512 PNG. Offline graphics are basic patterns, not AI-designed custom logos."}
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

// Body Texture Studio: recolor an existing approved WWE 2K26 UV atlas without
// changing island geometry. This is an offline skin-tone matching operation,
// not an AI reconstruction of anatomy or a DDS encoder.
func generateBodyTextureAtlas(atlasPath,facePath string,definition int)(string,error){
 af,e:=os.Open(atlasPath);if e!=nil{return "",e};defer af.Close()
 atlas,_,e:=image.Decode(af);if e!=nil{return "",fmt.Errorf("decode UV atlas: %w",e)}
 b:=atlas.Bounds();w,h:=b.Dx(),b.Dy()
 if w<256||h<256||int64(w)*int64(h)>32000000{return "",fmt.Errorf("unsupported UV atlas size")}
 ff,e:=os.Open(facePath);if e!=nil{return "",e};defer ff.Close()
 face,_,e:=image.Decode(ff);if e!=nil{return "",fmt.Errorf("decode face texture: %w",e)}
 fb:=face.Bounds();if fb.Dx()<32||fb.Dy()<32{return "",fmt.Errorf("face texture too small")}
 // Sample a central cheek band; avoid dark eyes, lips and background.
 var sr,sg,sb,count float64
 for y:=fb.Min.Y+fb.Dy()*43/100;y<fb.Min.Y+fb.Dy()*60/100;y+=2{
  for x:=fb.Min.X+fb.Dx()*25/100;x<fb.Min.X+fb.Dx()*75/100;x+=2{
   rr,gg,bb,aa:=face.At(x,y).RGBA()
   if aa<0xc000{continue}
   r,g,bl:=float64(rr>>8),float64(gg>>8),float64(bb>>8)
   if r<45||g<35||bl<25||r>248||g>245||bl>245{continue}
   sr+=r;sg+=g;sb+=bl;count++
  }
 }
 if count<25{return "",fmt.Errorf("no reliable cheek skin sample found in face texture")}
 target:=[3]float64{sr/count,sg/count,sb/count}
 var ar,ag,ab,n float64
 // Only recolor non-black, skin-colored atlas islands.
 isSkin:=func(r,g,bl float64)bool{return r>45&&g>30&&bl>20&&r>=g*0.88&&g>=bl*0.84&&r>bl*1.13}
 for y:=0;y<h;y+=3{for x:=0;x<w;x+=3{
  rr,gg,bb,aa:=atlas.At(b.Min.X+x,b.Min.Y+y).RGBA()
  if aa<0x8000{continue}
  r,g,bl:=float64(rr>>8),float64(gg>>8),float64(bb>>8)
  if isSkin(r,g,bl){ar+=r;ag+=g;ab+=bl;n++}
 }}
 if n<100{return "",fmt.Errorf("UV atlas has insufficient recognizable skin pixels; no output created")}
 original:=[3]float64{ar/n,ag/n,ab/n}
 dst:=image.NewNRGBA(image.Rect(0,0,w,h))
 clamp:=func(v float64)uint8{if v<0{return 0};if v>255{return 255};return uint8(math.Round(v))}
 for y:=0;y<h;y++{for x:=0;x<w;x++{
  rr,gg,bb,aa:=atlas.At(b.Min.X+x,b.Min.Y+y).RGBA()
  r,g,bl:=float64(rr>>8),float64(gg>>8),float64(bb>>8)
  pixel:=color.NRGBA{uint8(rr>>8),uint8(gg>>8),uint8(bb>>8),uint8(aa>>8)}
  if aa>=0x8000&&isSkin(r,g,bl){
   pixel.R=clamp(r+(target[0]-original[0])*0.85)
   pixel.G=clamp(g+(target[1]-original[1])*0.85)
   pixel.B=clamp(bl+(target[2]-original[2])*0.85)
   // Adjust contrast of EXISTING muscle shading; no synthetic anatomy is invented.
   // Neutral 50 retains source detail. Values 0-100 soften/emphasize it.
   factor:=0.65+float64(definition)*0.007
   lum:=0.2126*float64(pixel.R)+0.7152*float64(pixel.G)+0.0722*float64(pixel.B)
   pixel.R=clamp(lum+(float64(pixel.R)-lum)*factor+(lum-128)*(factor-1)*0.24)
   pixel.G=clamp(lum+(float64(pixel.G)-lum)*factor+(lum-128)*(factor-1)*0.24)
   pixel.B=clamp(lum+(float64(pixel.B)-lum)*factor+(lum-128)*(factor-1)*0.24)
  }
  dst.SetNRGBA(x,y,pixel)
 }}
 home,_:=os.UserHomeDir();if home==""{home=os.TempDir()}
 dir:=filepath.Join(home,"Documents","CAWForge","BodyTextures")
 if e=os.MkdirAll(dir,0755);e!=nil{return "",e}
 out:=filepath.Join(dir,strings.TrimSuffix(filepath.Base(atlasPath),filepath.Ext(atlasPath))+"_skinmatched.png")
 f,e:=os.Create(out);if e!=nil{return "",e}
 e=png.Encode(f,dst);closeErr:=f.Close();if e!=nil{return "",e};if closeErr!=nil{return "",closeErr}
 return out,nil
}
func beginTextureSave(tab int){
 var source string
 switch tab{case 1:source=lastFacePNG;case 2:source=lastGraphicPNG;case 5:source=lastBodyPNG}
 if source==""{alert("Generate a PNG first, then choose Save As.");return}
 if pickerBusy{alert("A file dialog is already open.");return}
 textureSaveSource=source;textureSaveTab=tab
 beginPicker(true)
}
func startBodyTexture(){
 atlas:=strings.Trim(strings.TrimSpace(txt(352)),"\"")
 face:=strings.Trim(strings.TrimSpace(txt(353)),"\"")
 if atlas==""||face==""{set(354,"Enter the approved body UV atlas PNG and face texture PNG paths.");return}
 definition:=50
 if raw:=strings.TrimSpace(txt(361));raw!=""{n,err:=strconv.Atoi(raw);if err!=nil||n<0||n>100{set(354,"Definition must be a number from 0 to 100.");return};definition=n}
 if singletMode{set(354,"Singlet mode: using existing atlas; arm/leg inputs not required. Matching skin and definition...")}else{set(354,"Matching skin tone and adjusting existing texture definition...")}
 go func(){
  out,e:=generateBodyTextureAtlas(atlas,face,definition)
  matchMu.Lock()
  if e!=nil{bodyTextureStatus="Body texture failed: "+e.Error()}else{lastBodyPNG=out;bodyTextureStatus="Skin-matched UV PNG: "+out+" (UV layout preserved; inspect before import; DDS conversion not included)"}
  matchMu.Unlock()
  postMessage.Call(hwnd,WM_BODY_TEXTURE_DONE,0,0)
 }()
}
var bodyTextureStatus string
var singletMode bool
var bodyPickerTarget int
var attireImportPicker bool
var lastFacePNG,lastBodyPNG,lastGraphicPNG string
var textureSaveSource string
var textureSaveTab int
const WM_BODY_TEXTURE_DONE=0x8012
// Track every native control, including labels without numeric IDs.
// A control created for one workspace must never leak into another tab.
var uiBuildTab = -1
var uiTabControls = map[int][]uintptr{}
func switchTab(n int){
 currentTab=n
 for tab,handles:=range uiTabControls{
  visible:=uintptr(0)
  if tab==n{visible=1}
  for _,h:=range handles{u.NewProc("ShowWindow").Call(h,visible)}
 }
 if n==4{if e:=rosterRefresh();e!=nil{set(341,"Roster list error: "+e.Error())}}
}

var values=map[string]int{}
// All morph controls use the same editable canonical layout as the exported sheet.
var groups=loadedMorphSheetSections()
func ctl(class,title string,x,y,w,h,id int,style uint32)uintptr{v,_,_:=create.Call(0,uintptr(unsafe.Pointer(ptr(class))),uintptr(unsafe.Pointer(ptr(title))),uintptr(WS_CHILD|WS_VISIBLE|style),uintptr(x),uintptr(y),uintptr(w),uintptr(h),hwnd,uintptr(id),0,0);if id>0{ctrls[id]=v};if uiBuildTab>=0&&v!=0{uiTabControls[uiBuildTab]=append(uiTabControls[uiBuildTab],v)};return v}
func txt(id int)string{if ctrls[id]==0{return ""};buf:=make([]uint16,2048);gettext.Call(ctrls[id],uintptr(unsafe.Pointer(&buf[0])),uintptr(len(buf)));return syscall.UTF16ToString(buf)}
func set(id int,s string){if ctrls[id]==0{return};settext.Call(ctrls[id],uintptr(unsafe.Pointer(ptr(s))))}
func alert(s string){msgbox.Call(hwnd,uintptr(unsafe.Pointer(ptr(s))),uintptr(unsafe.Pointer(ptr("CAWForge Studio"))),0)}
func fileDialog(save bool,pngSave bool)(string,error){
 // Use the native Windows common dialog directly, without spawning PowerShell.
 // Run on a dedicated locked OS thread and do not attach to the app's UI
 // window: that avoids cross-thread owner/window-message deadlocks.
 file:=make([]uint16,4096)
 var filter string
 var title string
 if save && pngSave {filter="PNG images (*.png)\x00*.png\x00\x00";title="Save Texture As PNG"
 } else if save {
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
 if save {if pngSave{ofn.DefExt=ptr("png")}else{ofn.DefExt=ptr("txt")};ofn.Flags|=0x00000002
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
var pickerSourceTab int
func beginPicker(save bool){
 if pickerBusy{return}
 pickerBusy=true
 pickerSourceTab=currentTab
 // Keep the message loop free of synchronous diagnostics disk I/O.
 set(25,"Opening Windows photo picker...")
 if currentTab==1{set(203,"Opening photo picker...")}
 templateMode:=templatePickerMode
 gridMode:=gridPickerMode
 go func(){
  exe,err:=os.Executable()
  var path string
  if err==nil{
   mode:="--pick-photo";if save{mode="--pick-save";if textureSaveSource!=""{mode="--pick-png"}}
   pickerCtx,cancel:=context.WithTimeout(context.Background(),90*time.Second)
   defer cancel()
   cmd:=exec.CommandContext(pickerCtx,exe,mode)
   cmd.Env=append(os.Environ(),"WWE_PICKER_CHILD=1")
   var stdout,stderr bytes.Buffer
   cmd.Stdout=&stdout;cmd.Stderr=&stderr
   err=cmd.Run()
   if pickerCtx.Err()!=nil{err=fmt.Errorf("Windows file picker timed out after 90 seconds")}
   if err==nil{path=strings.TrimSpace(stdout.String())}
   if err!=nil && stderr.Len()>0{err=fmt.Errorf("%v: %s",err,stderr.String())}
  }
  importTrace("Photo picker process returned")
  pickerMu.Lock();pickerResult=path;pickerErr="";if err!=nil{pickerErr=err.Error()};pickerMu.Unlock()
  var modeFlag uintptr
  if save{modeFlag=1}else if gridMode=="M"{modeFlag=3}else if gridMode=="F"{modeFlag=4}else if templateMode{modeFlag=2}
  if ok,_,_:=postMessage.Call(hwnd,WM_PICKER_DONE,modeFlag,0);ok==0{importTrace("ERROR: failed to post picker completion event")}
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
// Nonblocking import diagnostics: a buffered channel keeps file I/O off the GUI thread.
var importTraceEvents=make(chan string,128)
func init(){
 go func(){
  logPath:=filepath.Join(os.TempDir(),"CAWForge-import-diagnostics.log")
  for event:=range importTraceEvents{
   f,err:=os.OpenFile(logPath,os.O_CREATE|os.O_APPEND|os.O_WRONLY,0600)
   if err==nil{fmt.Fprintln(f,time.Now().Format(time.RFC3339Nano),event);f.Close()}
  }
 }()
}
func importTrace(event string){select{case importTraceEvents<-event:default:}}
func validatePhotoAsync(path string){
 importTrace("photo validation scheduled")
 photoImportMu.Lock();photoImportSerial++;serial:=photoImportSerial;photoImportMu.Unlock()
 set(25,"Checking reference photo in background...")
 if currentTab==1{set(203,"Checking reference photo in background...")}
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
  if failure!=""{importTrace("photo validation failed: "+failure)}else{importTrace("photo validation passed")}
  photoImportMu.Unlock()
  importTrace("posting photo validated message")
  if ok,_,_:=postMessage.Call(hwnd,WM_PHOTO_VALIDATED,0,0);ok==0{importTrace("ERROR: failed to post photo validation event")}
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
// Canonical WWE 2K26 morph-sheet headings; no guessed slider numbers.
var morphSheetSections=[]struct{name string;items []string}{
 {"Head",[]string{"Head Width","Head Height","Head Depth"}},
 {"Ears Top",[]string{"Offset","Height","Alignment"}},
 {"Ears",[]string{"Ears Width","Ears Height","Vertical Stretch","Placement Low/High","Placement Forward/Back"}},
 {"Earlobes",[]string{"Lobe Detachment","Lobe Length","Lobe Alignment"}},
 {"Forehead Upper",[]string{"Forehead Upper Size"}},
 {"Forehead",[]string{"Forehead Size"}},
 {"Eyebrows",[]string{"Eyebrow Length","Eyebrow Depth","Eyebrow Corner Height Left","Eyebrow Corner Height Right","Eyebrow Outer Height Left","Eyebrow Outer Height Right"}},
 {"Eyes",[]string{"Eye Length","Eye Socket Depth","Eye Corner Length","Eye Corner Height","Eye Outer Length","Eye Outer Height"}},
 {"Nose",[]string{"Nose Width","Nose Height","Nose Depth"}},
 {"Nose Bridge",[]string{"Nose Bridge Width","Nose Bridge Height","Nose Bridge Depth"}},
 {"Nose Tip",[]string{"Nose Tip Width","Tip Pointed or Rounded","Nose Tip Length","Tip Height"}},
 {"Nostrils",[]string{"Left Nostril Width","Left Nostril Height","Left Nostril Depth","Right Nostril Width","Right Nostril Height","Right Nostril Depth"}},
 {"Philtrum",[]string{"Philtrum Width","Philtrum Depth"}},
 {"Mouth",[]string{"Mouth Horizontal Length","Mouth Fullness","Mouth Depth","Left Mouth Corner Length","Right Mouth Corner Length","Left Mouth Corner Height","Right Mouth Corner Height"}},
 {"Cheekbones",[]string{"Upper Cheekbone","Middle Cheekbone","Lower Cheekbone"}},
 {"Cheeks",[]string{"Cheek Fullness","Cheek Height"}},
 {"Jawline",[]string{"Middle Jaw Fullness","Lower Jaw Height","Lower Jaw Depth","Rear Jaw Fullness","Lower Jaw Fullness","Lower Jaw Length"}},
 {"Chin",[]string{"Chin Width","Chin Height","Chin Forward/Back","Chin Vertical Placement"}},
 {"Parietal Region",[]string{"Parietal Region Size"}},
}
// Read user-editable section labels from the installed JSON file.
// Missing or invalid files fall back to the built-in canonical layout.
func loadedMorphSheetSections()[]struct{name string;items []string}{
 exe,e:=os.Executable();if e!=nil{return morphSheetSections}
 path:=filepath.Join(filepath.Dir(exe),"assets","templates","morph_sheet_template.json")
 raw,e:=os.ReadFile(path);if e!=nil{return morphSheetSections}
 var config struct{Version int `json:"version"`;Sections []struct{Name string `json:"name"`;Items []string `json:"items"`} `json:"sections"`}
 if e=json.Unmarshal(raw,&config);e!=nil||config.Version!=1||len(config.Sections)==0{return morphSheetSections}
 result:=make([]struct{name string;items []string},0,len(config.Sections))
 for _,section:=range config.Sections{
  if strings.TrimSpace(section.Name)==""||len(section.Items)==0{return morphSheetSections}
  items:=make([]string,0,len(section.Items))
  for _,item:=range section.Items{if strings.TrimSpace(item)==""{return morphSheetSections};items=append(items,item)}
  result=append(result,struct{name string;items []string}{section.Name,items})
 }
 return result
}
func sheet()string{
 var b strings.Builder
 fmt.Fprintf(&b,"## %s\r\n\r\n### Face Template\r\n- Face Template: %s%d\r\n\r\n",txt(20),gender,template)
 for _,g:=range groups{
  fmt.Fprintf(&b,"### %s\r\n",g.name)
  for _,item:=range g.items{
   // Only display values that the existing morph engine actually tracks.
   // Never turn an unsupported WWE 2K26 field into an invented estimate.
   if value,ok:=values[g.name+" / "+item];ok{
    fmt.Fprintf(&b,"- %s: %d\r\n",item,value)
   }else{
    fmt.Fprintf(&b,"- %s: [not calibrated]\r\n",item)
   }
  }
  b.WriteString("\r\n")
 }
 return b.String()
}
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
// One shared slot limits CPU and memory pressure from concurrent image jobs.
var imageWorkGate=make(chan struct{},1)
var previewGate=imageWorkGate
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
 importTrace("photo validated UI handler entered")
 photoImportMu.Lock();path,err:=photoImportPath,photoImportError;photoImportMu.Unlock()
 if err!=""{set(25,"Photo import failed: "+err);if currentTab==1{set(203,"Photo import failed: "+err)};importTrace("photo validated UI handler exited with error");return 0}
 imagePath=path
 if pickerSourceTab==3||currentTab==3{renderImagePath=path;set(322,"CAW screenshot imported: "+filepath.Base(path)+". Starting faithful background removal...");startRenderCutout();return 0}
 set(25,"Photo loaded: "+filepath.Base(path))
 importTrace("photo loaded status set")
 if currentTab==1 {
  // Face Texture import must not trigger the morph analyzer, template
  // search, or preview renderer. Those are separate, potentially costly jobs.
  set(203,"Reference photo ready: "+filepath.Base(path)+". Generate when ready.")
  return 0
 }
 if currentTab==0 {
  // Auto-match after validation on a background worker. Matching completion
  // applies the suggested morph values and redraws visible slider controls.
  // Do not start preview rendering simultaneously with photo decoding.
  set(41,"Photo imported. Auto-matching template and estimating face sliders...")
  matchPhoto()
  importTrace("morph photo auto-match requested")
 }
 return 0
case WM_PREVIEW_READY:applyMorphPreview();return 0
case WM_TIMER:
 if w==9877{importTrace("UI heartbeat: message loop responsive");return 0}
 if w==9876{return 0}
 if w==MORPH_PREVIEW_TIMER{
  u.NewProc("KillTimer").Call(hwnd,MORPH_PREVIEW_TIMER)
  updateSheet()
  return 0
 }
case WM_MATCH_DONE:
 matchMu.Lock();message,best,estimated:=matchMessage,matchID,morphResult;matchMu.Unlock();matchRunning=false
 if best!="" && strings.HasPrefix(best,gender){n,_:=strconv.Atoi(best[1:]);template=n;send.Call(ctrls[3],CB_SETCURSEL,uintptr(n-1),0);set(23,best);for key,value:=range estimated{values[key]=value};drawGroup();updateMorphPreview()};set(41,message);if matchQueued{matchQueued=false;matchPhoto()};return 0
case WM_BODY_TEXTURE_DONE:matchMu.Lock();status:=bodyTextureStatus;path:=lastBodyPNG;matchMu.Unlock();_ = path;set(354,status);return 0
case WM_RENDER_DONE:matchMu.Lock();status:=renderStatus;matchMu.Unlock();renderBusy=false;set(322,status);return 0
case WM_GRAPHIC_DONE:
 matchMu.Lock();status:=graphicStatus;matchMu.Unlock();graphicBusy=false;set(306,status);set(371,status);return 0
case WM_AI_DONE:
 matchMu.Lock();status,path:=aiResult,aiOutput;matchMu.Unlock();if path!=""{lastFacePNG=path};aiBusy=false;set(215,status);if path!=""{showTexturePreview(path)};return 0
case WM_EXPORT_DONE:
 matchMu.Lock();status:=exportStatus;matchMu.Unlock()
 morphExportBusy=false;set(25,status);return 0
case WM_TEXTURE_PREVIEW_READY:applyTexturePreview();return 0
case WM_TEXTURE_DONE:
 matchMu.Lock();status,path:=faceTextureStatus,faceTexturePreviewPath;matchMu.Unlock();if path!=""{lastFacePNG=path};textureBusy=false;set(203,status);if path!=""{showTexturePreview(path)};return 0
case WM_CAL_DONE:
 calMu.Lock();message:=calibrationMessage;calMu.Unlock();calibrationRunning=false;set(41,message);if imagePath!=""{matchPhoto()};return 0
case WM_PICKER_DONE:
 importTrace("picker completed; UI handler entered")
 pickerMu.Lock();path,err:=pickerResult,pickerErr;pickerMu.Unlock();pickerBusy=false
 if rosterMugshotPicker{rosterMugshotPicker=false;if err!=""{set(341,"Mugshot picker failed: "+err)}else if path!=""{if e:=rosterImportMugshot(path);e!=nil{set(341,"Mugshot failed: "+e.Error())}else{set(341,"Mugshot saved for "+txt(331))}};return 0}
 if themePicker{themePicker=false;if err!=""{alert(err)}else if path!=""{themeCustomPath=path;themeIndex=6;send.Call(ctrls[401],CB_SETCURSEL,6,0);loadTheme()};return 0}
 if textureSaveSource!=""{
  source:=textureSaveSource;tab:=textureSaveTab;textureSaveSource=""
  if err!=""{alert("Save As failed: "+err)}else if path!=""{
   if !strings.EqualFold(filepath.Ext(path),".png"){path+=".png"}
   data,e:=os.ReadFile(source)
   if e==nil{e=os.WriteFile(path,data,0644)}
   if e!=nil{alert("Could not save texture: "+e.Error())}else{
    switch tab{case 1:set(203,"Saved as: "+path);case 2:set(306,"Saved as: "+path);case 5:set(354,"Saved as: "+path)}
   }
  }
  return 0
 }
 if attireImportPicker{attireImportPicker=false;if err!=""{set(371,"Import failed: "+err)}else if path!=""{set(370,path);set(371,"Loaded "+filepath.Base(path)+"; run seam baseline to export a separate PNG.")};return 0}
 if bodyPickerTarget!=0{target:=bodyPickerTarget;bodyPickerTarget=0;if err!=""{set(354,"Image selection failed: "+err)}else if path!=""{set(target,path);set(354,"Selected "+filepath.Base(path))};return 0}
 if err!=""{templatePickerMode=false;set(25,"File picker error");alert("File picker failed: "+err)}else if w==2 {templatePickerMode=false;if path!=""{imagePath=path;assignTemplateImage()}else{set(25,"Template selection cancelled.")}}else if w==3 || w==4 {gridPickerMode="";if path!=""{prefix:="M";if w==4{prefix="F"};go func(){result:=importGrid(path,prefix);calMu.Lock();calibrationMessage=result;calMu.Unlock();postMessage.Call(hwnd,WM_CAL_DONE,0,0)}();set(41,"Cropping 20 grid thumbnails...")}}else if w==1 {if path!=""{content:=[]byte(sheet())
  set(25,"Exporting morph sheet...")
  go func(){
   err:=os.WriteFile(path,content,0644)
   matchMu.Lock()
   if err!=nil{exportStatus="Morph sheet export failed: "+err.Error()}else{exportStatus="Morph sheet exported: "+path}
   matchMu.Unlock()
   postMessage.Call(hwnd,WM_EXPORT_DONE,0,0)
  }()}}else{importTrace("picker result accepted");if pickerSourceTab==3&&path!=""{renderImagePath="";set(322,"Validating CAW screenshot: "+filepath.Base(path))};acceptPhoto(path)}
 importTrace("picker completion UI handler exited")
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
 case id==605:installDownloadedUpdate();case id==604:launchAIStudio();case id==600:checkUpdates();case id==602:downloadUpdate();case id==603:openUpdateFolder();case id==200:switchTab(0);case id==300:switchTab(2);case id==320:switchTab(3);case id==350:switchTab(5);case id==355:startBodyTexture();case id==368:attireImportPicker=true;beginPicker(false);case id==369:startImportedSeamTest();case id==365:beginTextureSave(5);case id==372:singletMode=!singletMode;if singletMode{set(372,"Mode: Singlet (click for Full Body)")}else{set(372,"Mode: Full Body (click for Singlet)")};case id==366:beginTextureSave(1);case id==367:beginTextureSave(2);case id==362:set(361,"20");case id==363:set(361,"50");case id==364:set(361,"85");case id==358:bodyPickerTarget=352;beginPicker(false);case id==359:bodyPickerTarget=353;beginPicker(false);case id==330:switchTab(4);case id==339:if e:=rosterSave();e!=nil{set(341,"Save failed: "+e.Error())}else{set(341,"Roster record saved to Documents/CAWForge/roster.json")};case id>=384&&id<=387:if e:=rosterGallerySelect(id-384);e!=nil{set(341,e.Error())};case id==388:rosterPage--;if e:=rosterRefresh();e!=nil{set(341,e.Error())};case id==390:rosterPage++;if e:=rosterRefresh();e!=nil{set(341,e.Error())};case id==391:if txt(331)==""{set(341,"Select a CAW first")}else{rosterEditing=txt(331);set(341,"Editing "+rosterEditing+" — change fields then Save / Update.")};case id==392:rosterMugshotPicker=true;beginPicker(false);case id==344:if e:=rosterViewSelected();e!=nil{set(341,"View failed: "+e.Error())};case id==346:if e:=rosterRefresh();e!=nil{set(341,"Refresh failed: "+e.Error())};case id==340:if e:=rosterLoad();e!=nil{set(341,"Load failed: "+e.Error())}else{set(341,"Roster record loaded")};case id==321:beginPicker(false);case id==323:startRenderCutout();case id==307:startGraphic();case id==308:seamAuto=!seamAuto;set(308,fmt.Sprintf("Auto Seam Connect: %v",seamAuto));case id==309:seamTop=!seamTop;set(309,fmt.Sprintf("Top sides: %v",seamTop));case id==310:seamTights=!seamTights;set(310,fmt.Sprintf("Tights leg sides: %v",seamTights));case id==311:seamWaist=!seamWaist;set(311,fmt.Sprintf("Waistband: %v",seamWaist));case id==202:switchTab(1);case id==201:startTexture();case id==216:runAITexture();case id==220:alert("FACE TEXTURE GUIDE\\n\\n1. Use a high-resolution straight-on photo, eyes open, neutral mouth, even light. Avoid filters, harsh shadows and hair over the face.\\n\\n2. Choose the correct in-game face template and prepare the face morph BEFORE applying a texture.\\n\\n3. The required output is an 820 x 820 flat UV face texture: no hair, ears, jewelry, clothing or background, with skin blended around the edges.\\n\\n4. CAWForge cannot currently extract this accurately. Both old portrait-generation buttons are disabled until face segmentation and calibrated template alignment are implemented.\\n\\n5. Do not import the old Tiffany-style portrait into WWE 2K26; it is not a valid face UV texture.");case id==217:beginPicker(false);case id==13:sel,_,_:=send.Call(ctrls[13],CB_GETCURSEL,0,0);if sel==1{genderMode="F";gender="F"}else if sel==2{genderMode="M";gender="M"}else{genderMode="Auto"};set(23,gender+strconv.Itoa(template));updateSheet();updateMorphPreview();if imagePath!=""{matchPhoto()};set(41,"Template library mode: "+genderMode+". Auto currently uses the selected library; photo classifier not installed.");case id==113:genderMode="Auto";set(41,"Auto: photo-based gender recognition is not installed. Using selected template library; choose Female or Male to lock matching.");case id==1:genderMode="M";gender="M";send.Call(ctrls[13],CB_SETCURSEL,2,0);set(23,"M"+strconv.Itoa(template));updateMorphPreview();set(41,"Male library selected. Matching requires installed reference images.");updateSheet();if imagePath!=""{matchPhoto()};case id==2:genderMode="F";gender="F";send.Call(ctrls[13],CB_SETCURSEL,1,0);set(23,"F"+strconv.Itoa(template));updateMorphPreview();set(41,"Female library selected. Matching requires installed reference images.");updateSheet();if imagePath!=""{matchPhoto()};case id==3:sel,_,_:=send.Call(ctrls[3],CB_GETCURSEL,0,0);if sel>=20{return 0};template=int(sel)+1;set(23,gender+strconv.Itoa(template));updateSheet();updateMorphPreview();case id==4:beginPicker(false);case id==5:exportMorphSheetAsync();case id==7:importTemplateImage();case id==9:beginCalibration();case id==10:matchPhoto();case id==11:gridPickerMode="M";beginPicker(false);case id==12:gridPickerMode="F";beginPicker(false);case id==8:exec.Command("rundll32.exe","url.dll,FileProtocolHandler","https://www.youtube.com/watch?v=WtWG9ZIxtes").Start();case id==6:for k:=range values{values[k]=50};drawGroup();case id>=50&&id<50+len(groups):active=groups[id-50].name;drawGroup()};case WM_HSCROLL:for i:=0;i<8;i++{if l==ctrls[110+i]{for _,g:=range groups{if g.name==active&&i<len(g.items){v,_,_:=send.Call(ctrls[110+i],TBM_GETPOS,0,0);values[g.name+" / "+g.items[i]]=int(v);set(120+i,strconv.Itoa(int(v)));scheduleSheetUpdate()}}}}};r,_,_:=def.Call(h,uintptr(m),w,l);return r}

const currentVersion="v0.9.32"
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
 runtime.LockOSThread()
 defer runtime.UnlockOSThread()
 if len(os.Args)==5 && os.Args[1]=="--apply-update" {if err:=applyUpdate(os.Args[2],os.Args[3],os.Args[4]);err!=nil{os.WriteFile(filepath.Join(os.TempDir(),"CAWForge-update-error.txt"),[]byte(err.Error()),0644)};return}
 // Run the native Windows file picker in a separate process, so a hung
 // shell extension cannot block the main application's message loop.
 if len(os.Args)>1 && (os.Args[1]=="--pick-photo"||os.Args[1]=="--pick-save"||os.Args[1]=="--pick-png"){
  runtime.LockOSThread()
  defer runtime.UnlockOSThread()
  ole:=syscall.NewLazyDLL("ole32.dll")
  init:=ole.NewProc("CoInitializeEx")
  uninit:=ole.NewProc("CoUninitialize")
  hr,_,_:=init.Call(0,2) // COINIT_APARTMENTTHREADED
  if hr==0 || hr==1{defer uninit.Call()}
  path,err:=fileDialog(os.Args[1]!="--pick-photo",os.Args[1]=="--pick-png")
  if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(2)}
  fmt.Print(path)
  return
 }
 exe,_:=os.Executable();templateLibraryDir=filepath.Join(filepath.Dir(exe),"assets","templates");os.MkdirAll(templateLibraryDir,0755);syscall.NewLazyDLL("comctl32.dll").NewProc("InitCommonControls").Call();for _,g:=range groups{for _,s:=range g.items{values[g.name+" / "+s]=50}};inst,_,_:=instance.Call(0);class:=ptr("WWEMorphStudioNative");wc:=WNDCLASSEX{Size:uint32(unsafe.Sizeof(WNDCLASSEX{})),WndProc:syscall.NewCallback(wndproc),Instance:inst,ClassName:class,Background:6};reg.Call(uintptr(unsafe.Pointer(&wc)));hwnd,_,_=create.Call(0,uintptr(unsafe.Pointer(class)),uintptr(unsafe.Pointer(ptr("CAWForge Studio — Desktop Edition v0.9.32"))),WS_OVERLAPPEDWINDOW,100,70,1130,900,0,0,inst,0)
 // Startup safe mode: skip theme restoration until the user requests it.
 // Avoid a full-window STATIC bitmap: it can paint over every interactive
 // control, making the whole application appear blank/white.
 // The chosen theme is retained and shown in a small non-overlapping preview.
 themeBackdrop=0
 ctl("STATIC","CAWFORGE STUDIO",20,15,460,28,0,0);
 ctl("BUTTON","Morph Settings",520,12,130,30,200,0);ctl("BUTTON","Face Texture",655,12,130,30,202,0);ctl("BUTTON","Graphic Generator",790,12,150,30,300,0);ctl("BUTTON","Render Creator",945,12,150,30,320,0);ctl("BUTTON","Roster",945,52,150,30,330,0);ctl("BUTTON","Body Texture",790,52,150,30,350,0);uiBuildTab=0;ctl("STATIC","Character name",20,55,120,22,0,0);ctl("EDIT","New Superstar",145,50,245,28,20,WS_BORDER);ctl("BUTTON","MALE TEMPLATES",400,50,165,28,1,0);ctl("BUTTON","FEMALE TEMPLATES",575,50,175,28,2,0);ctl("COMBOBOX","",760,50,150,360,3,CBS_DROPDOWNLIST|WS_VSCROLL);for i:=1;i<=20;i++{send.Call(ctrls[3],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(fmt.Sprintf("Template %d",i)))))};send.Call(ctrls[3],CB_SETCURSEL,9,0);ctl("STATIC","F10",915,55,25,22,23,0)
 uiBuildTab=0;ctl("BUTTON","Choose reference photo (or drop image)",20,100,260,32,4,0);ctl("STATIC","Template library:",575,105,130,24,0,0);ctl("COMBOBOX","",705,100,195,240,13,CBS_DROPDOWNLIST|WS_VSCROLL);for _,option:=range []string{"Auto (selected library)","Female only","Male only"}{send.Call(ctrls[13],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(option))))};send.Call(ctrls[13],CB_SETCURSEL,0,0);ctl("STATIC","Drop photo onto window or choose file",295,105,270,24,25,0);ctl("BUTTON","Export morph sheet",20,145,210,32,5,0);ctl("BUTTON","Reset sliders",245,145,150,32,6,0);ctl("BUTTON","Import Template Image",415,145,255,32,7,0);ctl("BUTTON","View template showcase",690,145,210,32,8,0);ctl("BUTTON","Refresh Template Library",20,187,170,30,9,0);ctl("BUTTON","Retry Photo Match",205,187,140,30,10,0);ctl("BUTTON","Import MALE Grid",20,688,170,29,11,0);ctl("BUTTON","Import FEMALE Grid",205,688,180,29,12,0);ctl("STATIC","Auto-selects best available match on photo import (requires real template images).",360,191,680,24,0,0);ctl("STATIC","Loading installed template references automatically...",20,218,1040,22,41,0)
 for i,g:=range groups{ctl("BUTTON",g.name,20+(i%7)*155,245+(i/7)*35,149,30,50+i,0)};ctl("STATIC","Category: Head",20,348,400,26,40,0)
 for i:=0;i<8;i++{y:=384+i*37;ctl("STATIC","",20,y,215,25,100+i,0);v:=ctl("msctls_trackbar32","",250,y-4,315,34,110+i,WS_TABSTOP);send.Call(v,TBM_SETRANGE,1,uintptr(100<<16));ctl("STATIC","",575,y,60,24,120+i,0)}
 ctl("STATIC","FACE COMPARISON PREVIEW",665,345,400,26,0,0);
 ctl("STATIC","Imported photo",665,373,185,22,503,0);
 ctl("STATIC","Matched template",865,373,190,22,505,0);
 ctl("STATIC","",665,398,185,150,501,SS_BITMAP|WS_BORDER);
 ctl("STATIC","",865,398,185,150,502,SS_BITMAP|WS_BORDER);
 ctl("STATIC","Import a photo to compare it with the matched template.",665,550,410,34,504,0);
 ctl("EDIT","",665,625,410,51,30,WS_BORDER|ES_MULTILINE|ES_AUTOVSCROLL|WS_VSCROLL|ES_READONLY);drawGroup();
 uiBuildTab=5;ctl("STATIC","BODY TEXTURE STUDIO — Preserve approved WWE 2K26 UV atlas",30,105,950,30,351,0);
 ctl("STATIC","Approved body UV atlas PNG path:",30,165,750,25,356,0);
 ctl("EDIT","",30,197,860,32,352,WS_BORDER);ctl("BUTTON","Browse...",905,197,145,32,358,0);
 ctl("STATIC","Face texture PNG path (skin-tone reference):",30,250,760,25,357,0);
 ctl("EDIT","",30,285,860,32,353,WS_BORDER);ctl("BUTTON","Browse...",905,285,145,32,359,0);
 ctl("STATIC","Existing definition intensity (0-100):",30,345,360,26,360,0);
 ctl("EDIT","50",380,342,90,32,361,WS_BORDER);
 ctl("BUTTON","Natural 20",490,342,160,32,362,0);
 ctl("BUTTON","Athletic 50",660,342,160,32,363,0);
 ctl("BUTTON","Muscular 85",830,342,160,32,364,0);
 ctl("BUTTON","Generate Skin-Matched Body UV PNG",30,395,450,42,355,0);
 ctl("BUTTON","Save Body As...",500,395,220,42,365,0);
 ctl("BUTTON","Mode: Full Body (click for Singlet)",30,545,500,38,372,0);
 ctl("STATIC","Definition adjusts contrast in existing skin shading only; it cannot create new muscles or identify body parts. No DDS conversion.",30,460,1020,85,354,0);
 uiBuildTab=1;ctl("BUTTON","Import Face Photo",30,295,260,34,217,0);
 ctl("BUTTON","Create Face Scan Preview",30,341,260,34,201,0);
 ctl("BUTTON","Save Face As...",30,490,260,34,366,0);
 ctl("STATIC","Texture AI engine:",30,395,160,28,218,0);
 ctl("COMBOBOX","",195,392,205,120,219,CBS_DROPDOWNLIST);
 for _,mode:=range []string{"Local AI (default; requires installed engine)","OpenAI API (optional; paid)"}{send.Call(ctrls[219],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(mode))))};send.Call(ctrls[219],CB_SETCURSEL,0,0);
 ctl("STATIC","Import a clear front-facing photo. Scan Preview creates a centered 820x820 alignment image; it is NOT a finished UV texture.",30,645,980,45,203,0);
 ctl("STATIC","FACE TEXTURE PREVIEW (320 x 320)",425,110,500,28,204,0);
 ctl("STATIC","",425,150,320,320,205,SS_BITMAP|WS_BORDER);
 ctl("STATIC","Face output: 820 x 820 PNG. Template fitting requires calibrated landmarks.",425,480,630,42,206,0);
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
 ctl("EDIT","",30,435,370,45,214,WS_BORDER|ES_MULTILINE);
 ctl("STATIC","AI status: run the bundled installer to configure Stable Diffusion.",30,590,980,40,215,0)
 ctl("BUTTON","Face Texture Guide",425,540,250,34,220,0);;
 ctl("BUTTON","Local AI (unavailable)",30,535,260,34,216,0);
 uiBuildTab=2;ctl("STATIC","GRAPHIC CREATOR - CHOOSE WHAT YOU ARE MAKING",30,110,900,34,301,0);
 ctl("STATIC","1. Graphic category (required):",30,170,270,26,302,0);
 ctl("STATIC","2. Describe the selected graphic (colors, words, logos and layout):",30,240,900,26,303,0);
 ctl("COMBOBOX","",315,165,440,260,304,CBS_DROPDOWNLIST);
 for _,kind:=range []string{"-- Choose graphic category --","Attire design (top, tights, waistband)","Wrestler logo / emblem","Entrance video background","Fan sign / arena placard"}{send.Call(ctrls[304],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(kind))))}
 send.Call(ctrls[304],CB_SETCURSEL,0,0)
 ctl("EDIT","",30,275,850,35,305,WS_BORDER);
 ctl("STATIC","Attire pattern: generates 3 AI concept panels using OPENAI_API_KEY (paid API). Other graphics remain procedural. Concepts are NOT WWE 2K26 UV maps.",30,390,980,70,306,0);
 ctl("BUTTON","Generate Selected Graphic",30,330,330,42,307,0);
 ctl("BUTTON","Save Graphic As...",385,330,300,42,367,0);
 ctl("BUTTON","Import Existing Attire PNG",30,485,320,38,368,0);
 ctl("BUTTON","Export Seam Baseline (UV mapping needed)",370,485,400,38,369,0);
 ctl("EDIT","",30,535,850,30,370,WS_BORDER);
 ctl("STATIC","Seam test needs calibrated WWE UV pairs; uncalibrated images are preserved unchanged.",30,580,1000,65,371,0);
 uiBuildTab=3;ctl("STATIC","RENDER CREATOR - PRESERVE ORIGINAL CAW",30,115,800,35,324,0);
 ctl("BUTTON","Import CAW Screenshot",30,180,290,42,321,0);
 ctl("STATIC","Render mode:",30,235,120,25,326,0);ctl("COMBOBOX","",155,230,360,160,325,CBS_DROPDOWNLIST);for _,mode:=range []string{"Faithful Render — automatic cutout","Faithful Render — original character pixels","Studio Pro — experimental material-aware finish"}{send.Call(ctrls[325],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(mode))))};send.Call(ctrls[325],CB_SETCURSEL,0,0);ctl("BUTTON","Retry Background Removal",30,275,380,42,323,0);ctl("STATIC","Photographic finish (0 = original / 100 = enhanced)",30,326,600,22,0,0);ctl("msctls_trackbar32","",30,350,380,35,327,0);send.Call(ctrls[327],TBM_SETRANGE,1,uintptr(100<<16));send.Call(ctrls[327],TBM_SETPOS,1,0);
 ctl("STATIC","Import to automatically isolate the wrestler. Choose Studio Pro for experimental material-aware lighting. Faithful Render remains available; neither mode regenerates faces or attire.",30,420,1000,50,322,0);
 uiBuildTab=4;ctl("STATIC","ROSTER DATABASE - SAVED LOCALLY",30,115,900,35,335,0);
 ctl("STATIC","Superstar name",30,180,200,25,336,0);ctl("EDIT","",240,175,420,30,331,WS_BORDER);
 ctl("STATIC","Real picbase / identity",30,230,200,25,337,0);ctl("EDIT","",240,225,420,30,332,WS_BORDER);
 ctl("STATIC","Alignment (Face/Heel/Tweener)",30,280,210,25,338,0);ctl("EDIT","Face",240,275,420,30,333,WS_BORDER);
 ctl("STATIC","Group / faction",30,330,200,25,334,0);ctl("EDIT","",240,325,420,30,342,WS_BORDER);
 ctl("BUTTON","Save / Update",30,385,230,40,339,0);ctl("BUTTON","Load by name",280,385,230,40,340,0);
 ctl("STATIC","Select a saved wrestler to view details, or enter a name and click Load.",30,435,960,35,341,0);
 ctl("STATIC","Browse saved roster:",30,625,190,26,0,0);
 ctl("COMBOBOX","",220,615,420,260,343,CBS_DROPDOWNLIST|WS_BORDER);
 ctl("BUTTON","View Selected",660,615,180,35,344,0);
 ctl("BUTTON","Refresh Roster",855,615,180,35,346,0);
 ctl("STATIC","0 wrestlers saved locally",30,655,300,25,345,0);
 ctl("BUTTON","Edit CAW",30,690,170,32,391,0);
 ctl("BUTTON","Add / Change Mugshot",215,690,240,32,392,0);
 for i:=0;i<4;i++{x:=30+i*265;ctl("STATIC","",x,485,115,110,380+i,SS_BITMAP|WS_BORDER);ctl("BUTTON","(empty)",x+120,520,140,42,384+i,0)}
 ctl("BUTTON","Previous",540,690,130,32,388,0);
 ctl("STATIC","Gallery page 1 / 1",685,695,170,25,389,0);
 ctl("BUTTON","Next",865,690,130,32,390,0);
 uiBuildTab=2;ctl("STATIC","AUTO SEAM CONNECT (attire patterns)",30,475,700,28,312,0);
 ctl("BUTTON","Auto Seam Connect: true",30,515,235,34,308,0);
 ctl("BUTTON","Top sides: true",275,515,195,34,309,0);
 ctl("BUTTON","Tights leg sides: true",480,515,225,34,310,0);
 ctl("BUTTON","Waistband: true",715,515,195,34,311,0);
 uiBuildTab=-1;ctl("STATIC","Theme:",635,734,55,24,0,0)
 ctl("COMBOBOX","",690,730,260,190,401,CBS_DROPDOWNLIST)
 for _,label:=range []string{"Original Studio","WWE 2K22 (original if installed)","WWE 2K23 (original if installed)","WWE 2K24 (original if installed)","WWE 2K25 (original if installed)","WWE 2K26 (original if installed)","Custom Image"}{send.Call(ctrls[401],CB_ADDSTRING,0,uintptr(unsafe.Pointer(ptr(label))))}
 send.Call(ctrls[401],CB_SETCURSEL,uintptr(themeIndex),0)
 ctl("BUTTON","Import background",960,730,135,28,402,0)
 ctl("BUTTON","AI Behavior Studio",390,730,175,28,604,0);ctl("BUTTON","Check for Updates",20,775,170,28,600,0)
 ctl("BUTTON","Download Update",200,775,155,28,602,0)
 ctl("BUTTON","Open Download",365,775,145,28,603,0)
 ctl("STATIC","Updates: GitHub Releases | verified ZIP",20,815,500,20,601,0);ctl("BUTTON","Install && Restart",530,810,175,29,605,0)
 // User-supplied original game assets take precedence; fallback is game-inspired.
 importTrace("startup: controls created");dragAccept.Call(hwnd,1);importTrace("startup: drag drop enabled");switchTab(0);importTrace("startup: tab visibility configured");set(41,"Startup safe mode: template scan paused. Click Refresh Template Library when ready.");show.Call(hwnd,SW_SHOW);update.Call(hwnd);importTrace("startup: window shown");u.NewProc("SetTimer").Call(hwnd,9877,5000,0);var msg MSG;for{r,_,_:=getmsg.Call(uintptr(unsafe.Pointer(&msg)),0,0,0);if int32(r)<=0{break};trans.Call(uintptr(unsafe.Pointer(&msg)));dispatch.Call(uintptr(unsafe.Pointer(&msg)))}}
