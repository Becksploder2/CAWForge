//go:build windows
// render-segmenter is a small bridge to an installed rembg CLI.
// Install rembg separately; the model is not bundled with CAWForge.
package main
import("fmt";"os";"os/exec";"context";"time";"strings")
func main(){
 if len(os.Args)!=3{fmt.Fprintln(os.Stderr,"usage: render-segmenter.exe input.png output.png");os.Exit(2)}
 ctx,cancel:=context.WithTimeout(context.Background(),3*time.Minute);defer cancel()
 args:=[]string{"i","-m","isnet-general-use",os.Args[1],os.Args[2]}
 cmd:=exec.CommandContext(ctx,"rembg",args...)
 out,err:=cmd.CombinedOutput()
 if err!=nil{fmt.Fprintln(os.Stderr,"rembg CLI failed (install rembg with CLI support):",err,strings.TrimSpace(string(out)));os.Exit(1)}
 if _,err:=os.Stat(os.Args[2]);err!=nil{fmt.Fprintln(os.Stderr,"rembg did not create output:",err);os.Exit(1)}
}
