//go:build windows
// Optional local rembg bridge. Does not bundle Python or segmentation weights.
package main
import("fmt";"os";"os/exec";"context";"time";"strings";"path/filepath")
func main(){
 if len(os.Args)!=3{fmt.Fprintln(os.Stderr,"usage: render-segmenter.exe input.png output.png");os.Exit(2)}
 ctx,cancel:=context.WithTimeout(context.Background(),3*time.Minute);defer cancel()
 exe,_:=os.Executable()
 local:=filepath.Join(filepath.Dir(exe),"rembg.exe")
 command:="rembg"
 if _,err:=os.Stat(local);err==nil{command=local}else if found,err:=exec.LookPath("rembg");err==nil{command=found}else{
  fmt.Fprintln(os.Stderr,"AI background remover not installed. CAWForge currently requires rembg with CLI support. Install it with: py -m pip install \"rembg[cpu,cli]\" (and restart CAWForge), or place rembg.exe beside CAWForge. A self-contained engine is not bundled in this test build.")
  os.Exit(1)
 }
 cmd:=exec.CommandContext(ctx,command,"i","-m","isnet-general-use",os.Args[1],os.Args[2])
 out,err:=cmd.CombinedOutput()
 if err!=nil{fmt.Fprintln(os.Stderr,"rembg segmentation failed:",err,strings.TrimSpace(string(out)));os.Exit(1)}
 if _,err:=os.Stat(os.Args[2]);err!=nil{fmt.Fprintln(os.Stderr,"rembg did not create output:",err);os.Exit(1)}
}
