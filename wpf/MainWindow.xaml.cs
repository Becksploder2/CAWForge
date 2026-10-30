using Cv2 = OpenCvSharp.Cv2;
using CascadeClassifier = OpenCvSharp.CascadeClassifier;
using ImreadModes = OpenCvSharp.ImreadModes;
using HaarDetectionTypes = OpenCvSharp.HaarDetectionTypes;
using Mat = OpenCvSharp.Mat;
using Microsoft.Win32;
using System.IO;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media.Imaging;
namespace CAWForge;
public partial class MainWindow : Window {
 private string? selectedPhoto;
 private string? attirePhoto;
 private int recognitionVersion=0;
 private readonly Dictionary<string,Slider> morphSliders=new();
 private readonly Dictionary<string,List<StackPanel>> morphCategoryPanels=new();
 public MainWindow() { InitializeComponent(); for(int i=1;i<=21;i++) TemplateBox.Items.Add("Template "+i); TemplateBox.SelectedIndex=9; foreach(var group in new (string Category,string[] Names)[]{("Head",new[]{"Head Width","Head Height","Head Depth"}),("Ears Top",new[]{"Offset","Height","Alignment"}),("Ears",new[]{"Ears Width","Ears Height","Vertical Stretch","Placement Low/High","Placement Forward/Back"}),("Earlobes",new[]{"Lobe Detachment","Lobe Length","Lobe Alignment"}),("Forehead Upper",new[]{"Forehead Upper Size"}),("Forehead",new[]{"Forehead Size"}),("Eyebrows",new[]{"Eyebrow Length","Eyebrow Depth","Eyebrow Corner Height Left","Eyebrow Corner Height Right","Eyebrow Outer Height Left","Eyebrow Outer Height Right"}),("Eyes",new[]{"Eye Length","Eye Socket Depth","Eye Corner Length","Eye Corner Height","Eye Outer Length","Eye Outer Height"}),("Nose",new[]{"Nose Width","Nose Height","Nose Depth"}),("Nose Bridge",new[]{"Nose Bridge Width","Nose Bridge Height","Nose Bridge Depth"}),("Nose Tip",new[]{"Nose Tip Width","Tip Pointed or Rounded","Nose Tip Length","Tip Height"}),("Nostrils",new[]{"Left Nostril Width","Left Nostril Height","Left Nostril Depth","Right Nostril Width","Right Nostril Height","Right Nostril Depth"}),("Philtrum",new[]{"Philtrum Width","Philtrum Depth"}),("Mouth",new[]{"Mouth Horizontal Length","Mouth Fullness","Mouth Depth","Left Mouth Corner Length","Right Mouth Corner Length","Left Mouth Corner Height","Right Mouth Corner Height"}),("Cheekbones",new[]{"Upper Cheekbone","Middle Cheekbone","Lower Cheekbone"}),("Cheeks",new[]{"Cheek Fullness","Cheek Height"}),("Jawline",new[]{"Middle Jaw Fullness","Lower Jaw Height","Lower Jaw Depth","Rear Jaw Fullness","Lower Jaw Fullness","Lower Jaw Length"}),("Chin",new[]{"Chin Width","Chin Height","Chin Forward/Back","Chin Vertical Placement"}),("Parietal Region",new[]{"Parietal Region Size"})}) {morphCategoryPanels[group.Category]=new List<StackPanel>(); foreach(var name in group.Names) {var panel=new StackPanel{Margin=new Thickness(0,0,0,8)};panel.Children.Add(new TextBlock{Text=group.Category+" / "+name,Foreground=System.Windows.Media.Brushes.White});panel.Children.Add(new Slider{Minimum=0,Maximum=100,Value=50,Width=360,HorizontalAlignment=HorizontalAlignment.Left});morphCategoryPanels[group.Category].Add(panel);morphSliders[group.Category+" / "+name]=(Slider)panel.Children[1];} var categoryName=group.Category;var categoryButton=new Button{Content=categoryName,Margin=new Thickness(0,0,8,8),Padding=new Thickness(12,7,12,7),MinWidth=110};categoryButton.Click+=(_,_)=>ShowMorphCategory(categoryName);MorphCategoryButtons.Children.Add(categoryButton);} ShowMorphCategory("Head"); }
 private void ShowMorphCategory(string category){SlidersList.Items.Clear();ActiveMorphCategory.Text=category;foreach(var panel in morphCategoryPanels[category])SlidersList.Items.Add(panel);}

 private async void ImportPhoto_Click(object sender,RoutedEventArgs e)
 {
  var dialog=new OpenFileDialog{Title="Choose reference photo",Filter="Images|*.png;*.jpg;*.jpeg;*.bmp;*.webp",CheckFileExists=true};
  if(dialog.ShowDialog(this)!=true)return;
  var path=dialog.FileName;
  StatusText.Text="Loading image in background…";
  try {
   var result=await Task.Run(()=>{
    var info=new FileInfo(path);
    if(info.Length>75L*1024*1024)throw new InvalidDataException("Image exceeds 75 MB.");
    using var stream=File.OpenRead(path);
    var decoder=BitmapDecoder.Create(stream,BitmapCreateOptions.PreservePixelFormat,BitmapCacheOption.OnLoad);
    var frame=decoder.Frames[0];
    var width=frame.PixelWidth;var height=frame.PixelHeight;
    if(width<1||height<1||(long)width*height>35000000)throw new InvalidDataException("Image dimensions are unsupported.");
    var factor=Math.Min(1.0,480.0/Math.Max(width,height));
    BitmapSource preview=frame;
    if(factor<1)preview=new TransformedBitmap(frame,new System.Windows.Media.ScaleTransform(factor,factor));
    preview.Freeze();
    return (width,height,preview);
   });
   PhotoPreview.Source=result.preview;
   selectedPhoto=path;
   PhotoPathText.Text=Path.GetFileName(path);
   StatusText.Text=$"Photo ready ({result.width} × {result.height}).";
   await MatchTemplateAsync();
  }catch(Exception ex){
   StatusText.Text="Import failed: "+ex.Message;
   MessageBox.Show(this,ex.Message,"Import error",MessageBoxButton.OK,MessageBoxImage.Warning);
  }
 }

 private void ExportMorph_Click(object sender,RoutedEventArgs e) {
  var dlg=new SaveFileDialog{Title="Export morph sheet",Filter="Text files|*.txt",FileName="CAWForge-Morph-Sheet.txt"};
  if(dlg.ShowDialog(this)!=true)return;
  try {
   var lines=new List<string>{"CAWForge WPF Morph Sheet","Character template: "+(GenderBox.SelectedIndex==0?"M":"F")+(TemplateBox.SelectedIndex+1),"Reference: "+(selectedPhoto is null?"None":Path.GetFileName(selectedPhoto)),"","Slider values (0-100):"};
   string? currentCategory=null;
   foreach(var entry in morphSliders){
    int separator=entry.Key.IndexOf(" / ",StringComparison.Ordinal);
    string category=separator>=0?entry.Key[..separator]:"Other";
    string sliderName=separator>=0?entry.Key[(separator+3)..]:entry.Key;
    if(!string.Equals(category,currentCategory,StringComparison.Ordinal)){
     if(currentCategory is not null)lines.Add("");
     lines.Add("## "+category);
     currentCategory=category;
    }
    lines.Add(sliderName+": "+Math.Round(entry.Value.Value));
   }
   File.WriteAllLines(dlg.FileName,lines);StatusText.Text="Morph sheet saved: "+dlg.FileName;
  }catch(Exception ex){MessageBox.Show(this,ex.Message,"Export error",MessageBoxButton.OK,MessageBoxImage.Error);}
 }
 private async void GenerateTexture_Click(object sender,RoutedEventArgs e) {
  if(selectedPhoto is null){MessageBox.Show(this,"Import a reference photo in Morph Settings first.","Face texture",MessageBoxButton.OK,MessageBoxImage.Information);return;}
  string pictures=Environment.GetFolderPath(Environment.SpecialFolder.MyPictures);
  if(string.IsNullOrWhiteSpace(pictures)||!Directory.Exists(pictures))pictures=Environment.GetFolderPath(Environment.SpecialFolder.DesktopDirectory);
  var dlg=new SaveFileDialog{
   Title="Save WWE 2K26 face texture",
   Filter="PNG image (*.png)|*.png",
   DefaultExt=".png",AddExtension=true,OverwritePrompt=true,
   FileName="CAWForge-Face-Texture-820.png",
   InitialDirectory=pictures
  };
  if(dlg.ShowDialog(this)!=true)return;
  var output=Path.GetFullPath(dlg.FileName);
  StatusText.Text="Generating 820 × 820 face texture…";
  try{
   var source=selectedPhoto;
   int upscale=FaceUpscaleBox.SelectedIndex==1?2:FaceUpscaleBox.SelectedIndex==2?4:1;
   var png=await Task.Run(()=>MakeFaceTexture(source,upscale));
   if(png.Length==0)throw new InvalidDataException("The generated PNG is empty.");
   // Write atomically to avoid leaving behind a damaged partial PNG.
   var folder=Path.GetDirectoryName(output);
   if(string.IsNullOrWhiteSpace(folder)||!Directory.Exists(folder))throw new DirectoryNotFoundException("The selected output folder does not exist.");
   var temporary=Path.Combine(folder,"."+Guid.NewGuid().ToString("N")+".tmp");
   try{
    await File.WriteAllBytesAsync(temporary,png);
    File.Move(temporary,output,true);
   }finally{if(File.Exists(temporary))File.Delete(temporary);}
   if(!File.Exists(output)||new FileInfo(output).Length!=png.Length)throw new IOException("The saved PNG could not be verified.");
   using var ms=new MemoryStream(png);
   var preview=new BitmapImage();preview.BeginInit();preview.StreamSource=ms;preview.CacheOption=BitmapCacheOption.OnLoad;preview.EndInit();preview.Freeze();
   TexturePreview.Source=preview;
   StatusText.Text="Face texture saved to: "+output;
   MessageBox.Show(this,"Face texture saved successfully:\\n"+output,"Export complete",MessageBoxButton.OK,MessageBoxImage.Information);
  }catch(Exception ex){
   StatusText.Text="Face texture save failed: "+ex.Message;
   MessageBox.Show(this,"Could not save the face texture.\\n\\n"+ex.Message+"\\n\\nTry a local folder such as Pictures.","Face texture export failed",MessageBoxButton.OK,MessageBoxImage.Error);
  }
 }
 private static byte[] MakeFaceTexture(string path,int upscale=1){
  using var src=Cv2.ImRead(path,ImreadModes.Color);
  if(src.Empty())throw new InvalidDataException("Unable to decode reference photo.");
  if((long)src.Width*src.Height>35000000)throw new InvalidDataException("Image exceeds 35 megapixels.");
  // Detect the face; never use the whole photograph as the texture.
  OpenCvSharp.Rect face;
  var classifierPath=Path.Combine(AppContext.BaseDirectory,"Models","haarcascade_frontalface_default.xml");
  if(!File.Exists(classifierPath))throw new FileNotFoundException("Face detector is missing. Reinstall the complete CAWForge build.",classifierPath);
  using(var detector=new CascadeClassifier(classifierPath)){
   var found=detector.DetectMultiScale(src,1.1,5);
   if(found.Length==0)throw new InvalidDataException("No frontal face detected. Use a clear, straight-on reference photograph.");
   face=found.OrderByDescending(f=>f.Width*f.Height).First();
  }
  // Frame only the detected face. The former wide crop included clothing,
  // water bottles and stadium seating; none belong in a UV face texture.
  int left=Math.Max(0,(int)(face.X-face.Width*0.10));
  int top=Math.Max(0,(int)(face.Y-face.Height*0.16));
  int right=Math.Min(src.Width,(int)(face.Right+face.Width*0.10));
  int bottom=Math.Min(src.Height,(int)(face.Bottom+face.Height*0.10));
  if(right-left<32||bottom-top<32)throw new InvalidDataException("Detected face is too small for texture generation.");
  using var crop=new Mat(src,new OpenCvSharp.Rect(left,top,right-left,bottom-top));
  using var aligned=new Mat();
  if(upscale>1){
   var tool=Path.Combine(AppContext.BaseDirectory,"AI","realesrgan-ncnn-vulkan.exe");
   var modelFolder=Path.Combine(AppContext.BaseDirectory,"AI","models");
   if(!File.Exists(tool)||!Directory.Exists(modelFolder))throw new FileNotFoundException("Local AI upscaler files are missing from this build. Reinstall the full Windows artifact or select Original resolution.");
   string tempDir=Path.Combine(Path.GetTempPath(),"CAWForge-Upscale-"+Guid.NewGuid().ToString("N"));
   Directory.CreateDirectory(tempDir);
   try{
    string input=Path.Combine(tempDir,"face.png"),output=Path.Combine(tempDir,"enhanced.png");
    Cv2.ImWrite(input,crop);
    var start=new System.Diagnostics.ProcessStartInfo(tool){
     Arguments=$"-i \"{input}\" -o \"{output}\" -n realesrgan-x4plus -s {upscale} -m \"{modelFolder}\" -f png",
     UseShellExecute=false,CreateNoWindow=true,RedirectStandardError=true,RedirectStandardOutput=true,
     WorkingDirectory=Path.GetDirectoryName(tool)!
    };
    using var process=System.Diagnostics.Process.Start(start)??throw new IOException("Could not start the local AI upscaler.");
    if(!process.WaitForExit(120000)){process.Kill(true);throw new TimeoutException("AI upscaling exceeded two minutes.");}
    if(process.ExitCode!=0||!File.Exists(output))throw new IOException("AI upscaling failed (exit "+process.ExitCode+"). Check Vulkan driver compatibility or choose Original resolution.");
    using var enhanced=Cv2.ImRead(output,ImreadModes.Color);
    if(enhanced.Empty())throw new IOException("AI upscaler returned an invalid image.");
    Cv2.Resize(enhanced,aligned,new OpenCvSharp.Size(820,820),0,0,OpenCvSharp.InterpolationFlags.Lanczos4);
   }finally{try{Directory.Delete(tempDir,true);}catch{}}
  }else{
   Cv2.Resize(crop,aligned,new OpenCvSharp.Size(820,820),0,0,OpenCvSharp.InterpolationFlags.Lanczos4);
  }
  // Improve fine facial texture before compositing. Enhance local luminance
  // (pores, freckles, lips and eye details) without exaggerating skin color.
  using(var lab=new Mat()){
   Cv2.CvtColor(aligned,lab,OpenCvSharp.ColorConversionCodes.BGR2Lab);
   var channels=Cv2.Split(lab);
   try{
    using(var clahe=Cv2.CreateCLAHE(1.6,new OpenCvSharp.Size(8,8))){
     using var enhanced=new Mat();
     clahe.Apply(channels[0],enhanced);
     enhanced.CopyTo(channels[0]);
    }
    Cv2.Merge(channels,lab);
    Cv2.CvtColor(lab,aligned,OpenCvSharp.ColorConversionCodes.Lab2BGR);
   }finally{foreach(var ch in channels)ch.Dispose();}
  }
  using(var blurred=new Mat()){
   Cv2.GaussianBlur(aligned,blurred,new OpenCvSharp.Size(0,0),0.85);
   Cv2.AddWeighted(aligned,1.28,blurred,-0.28,0,aligned);
  }
  // Use genuine cheek skin for a seamless, full-canvas underskin. Never
  // mirror the photo, preserve its original background, or paste an oval.
  // Facial detail is composited only where the image plausibly contains skin.
  // Sample a broad patch of cheek skin, not individual pixels (which can
  // land on hair, makeup or a specular highlight).
  static OpenCvSharp.Vec3b MedianPatch(Mat m,int cx,int cy,int radius){
   var bs=new List<byte>();var gs=new List<byte>();var rs=new List<byte>();
   for(int y=Math.Max(0,cy-radius);y<Math.Min(m.Height,cy+radius);y+=3)
    for(int x=Math.Max(0,cx-radius);x<Math.Min(m.Width,cx+radius);x+=3){
     var p=m.At<OpenCvSharp.Vec3b>(y,x);
     if(p.Item0<25||p.Item1<25||p.Item2<25)continue;
     bs.Add(p.Item0);gs.Add(p.Item1);rs.Add(p.Item2);
    }
   if(bs.Count==0)throw new InvalidDataException("Cannot sample usable skin tone from the photograph.");
   bs.Sort();gs.Sort();rs.Sort();int k=bs.Count/2;
   return new OpenCvSharp.Vec3b(bs[k],gs[k],rs[k]);
  }
  var cheekA=MedianPatch(aligned,285,505,32);
  var cheekB=MedianPatch(aligned,535,505,32);
  var browSkin=MedianPatch(aligned,410,305,22);
  var chinSkin=MedianPatch(aligned,410,650,20);
  using var texture=new Mat(new OpenCvSharp.Size(820,820),OpenCvSharp.MatType.CV_8UC3);
  // Use a facial-detail mask shaped to exclude the hairline, ears, and neck.
  // Outside that mask, render continuous skin sampled from the reference.
  for(int y=0;y<820;y++)for(int x=0;x<820;x++){
   double u=x/819.0,v=y/819.0;
   double[] tone=new double[3];
   for(int c=0;c<3;c++){
    double l=c==0?cheekA.Item0:c==1?cheekA.Item1:cheekA.Item2;
    double rr=c==0?cheekB.Item0:c==1?cheekB.Item1:cheekB.Item2;
    double forehead=c==0?browSkin.Item0:c==1?browSkin.Item1:browSkin.Item2;
    double chin=c==0?chinSkin.Item0:c==1?chinSkin.Item1:chinSkin.Item2;
    tone[c]=0.80*(l*(1-u)+rr*u)+0.20*(forehead*(1-v)+chin*v);
   }
   // Rectangular, feathered facial-detail region: no oval portrait vignette.
   double edgeX=Math.Min(x-205.0,615.0-x);
   double edgeY=Math.Min(y-275.0,685.0-y);
   double alpha=Math.Clamp(Math.Min(edgeX/75.0,edgeY/75.0),0,1);
   alpha=alpha*alpha*(3-2*alpha);
   // Protect the brow/eye band from the underskin feather mask.
   // Keep original eyebrow hairs and definition instead of washing them out.
   if(y>=275&&y<=430&&x>=190&&x<=630)alpha=1.0;
   var p=aligned.At<OpenCvSharp.Vec3b>(y,x);
   texture.Set(y,x,new OpenCvSharp.Vec3b(
    (byte)Math.Clamp(p.Item0*alpha+tone[0]*(1-alpha),0,255),
    (byte)Math.Clamp(p.Item1*alpha+tone[1]*(1-alpha),0,255),
    (byte)Math.Clamp(p.Item2*alpha+tone[2]*(1-alpha),0,255)));
  }
  Cv2.ImEncode(".png",texture,out var bytes);
  return bytes;
 }

 private string? renderPhoto;
 private byte[]? preparedRender;
 private static BitmapImage PreviewPng(byte[] png){
  using var stream=new MemoryStream(png);
  var image=new BitmapImage();image.BeginInit();image.CacheOption=BitmapCacheOption.OnLoad;image.StreamSource=stream;image.EndInit();image.Freeze();return image;
 }
 private async void ImportRender_Click(object sender,RoutedEventArgs e){
  var dlg=new OpenFileDialog{Title="Choose in-game CAW screenshot",Filter="Image files|*.png;*.jpg;*.jpeg;*.bmp",CheckFileExists=true};
  if(dlg.ShowDialog(this)!=true)return;
  renderPhoto=dlg.FileName;preparedRender=null;
  RenderFileText.Text=Path.GetFileName(renderPhoto)+" — removing background…";
  StatusText.Text="Removing background from imported CAW screenshot…";
  try{
   string path=renderPhoto;int threshold=(int)RenderThreshold.Value;
   preparedRender=await Task.Run(()=>CreateStudioRender(path,threshold));
   RenderPreview.Source=PreviewPng(preparedRender);
   RenderFileText.Text=Path.GetFileName(path)+" — transparent preview ready";
   StatusText.Text="Background removed. Inspect the preview before exporting.";
  }catch(Exception ex){
   RenderFileText.Text="Background removal failed: "+ex.Message;
   StatusText.Text="Render import failed.";
   MessageBox.Show(this,ex.Message,"Render Creator",MessageBoxButton.OK,MessageBoxImage.Error);
  }
 }
 private async void GenerateRender_Click(object sender,RoutedEventArgs e){
  if(renderPhoto==null){MessageBox.Show(this,"Import an in-game CAW screenshot first.");return;}
  var dlg=new SaveFileDialog{Title="Export CAW studio render",Filter="PNG image|*.png",FileName="CAWForge-Render.png",AddExtension=true};
  if(dlg.ShowDialog(this)!=true)return;
  try{
   StatusText.Text="Creating transparent studio render…";
   var path=renderPhoto;int threshold=(int)RenderThreshold.Value;
   byte[] png=preparedRender??await Task.Run(()=>CreateStudioRender(path,threshold));
   await File.WriteAllBytesAsync(dlg.FileName,png);
   RenderPreview.Source=PreviewPng(png);
   StatusText.Text="Render saved: "+dlg.FileName;
  }catch(Exception ex){StatusText.Text="Render failed: "+ex.Message;MessageBox.Show(this,ex.Message,"Render Creator",MessageBoxButton.OK,MessageBoxImage.Error);}
 }
 private static byte[] CreateStudioRender(string path,int threshold){
  using var source=Cv2.ImRead(path,ImreadModes.Color);
  if(source.Empty())throw new InvalidDataException("Unable to open screenshot.");
  if((long)source.Width*source.Height>35000000)throw new InvalidDataException("Screenshot is too large.");
  // NEVER center-crop: that previously cut off extended arms, hands and gear.
  // Letterbox the entire screenshot, retaining its aspect ratio and all pixels.
  int fitW=Math.Max(1,(int)Math.Round(source.Width*Math.Min(1024.0/source.Width,1024.0/source.Height)));
  int fitH=Math.Max(1,(int)Math.Round(source.Height*Math.Min(1024.0/source.Width,1024.0/source.Height)));
  using var rgb=new Mat(new OpenCvSharp.Size(1024,1024),OpenCvSharp.MatType.CV_8UC3,new OpenCvSharp.Scalar(0,0,0));
  using(var scaled=new Mat()){
   Cv2.Resize(source,scaled,new OpenCvSharp.Size(fitW,fitH),0,0,OpenCvSharp.InterpolationFlags.Lanczos4);
   using var target=new Mat(rgb,new OpenCvSharp.Rect((1024-fitW)/2,(1024-fitH)/2,fitW,fitH));
   scaled.CopyTo(target);
  }
  // Connected flood-fill from image borders identifies a roughly uniform
  // game backdrop. A complex arena may need a dedicated segmentation model.
  int w=rgb.Width,h=rgb.Height;
  var background=new bool[w*h];var queued=new bool[w*h];var queue=new Queue<int>();
  var corners=new[]{rgb.At<OpenCvSharp.Vec3b>(0,0),rgb.At<OpenCvSharp.Vec3b>(0,w-1),rgb.At<OpenCvSharp.Vec3b>(h-1,0),rgb.At<OpenCvSharp.Vec3b>(h-1,w-1)};
  bool Similar(OpenCvSharp.Vec3b p){
   foreach(var c in corners){
    int d=Math.Abs(p.Item0-c.Item0)+Math.Abs(p.Item1-c.Item1)+Math.Abs(p.Item2-c.Item2);
    if(d<=threshold*3)return true;
   }
   return false;
  }
  void Add(int x,int y){
   if(x<0||x>=w||y<0||y>=h)return;
   int index=y*w+x;if(queued[index])return;
   queued[index]=true;
   if(Similar(rgb.At<OpenCvSharp.Vec3b>(y,x)))queue.Enqueue(index);
  }
  for(int x=0;x<w;x++){Add(x,0);Add(x,h-1);}
  for(int y=0;y<h;y++){Add(0,y);Add(w-1,y);}
  while(queue.Count>0){
   int i=queue.Dequeue();if(background[i])continue;background[i]=true;
   int x=i%w,y=i/w;
   Add(x-1,y);Add(x+1,y);Add(x,y-1);Add(x,y+1);
  }
  using var alpha=new Mat(new OpenCvSharp.Size(w,h),OpenCvSharp.MatType.CV_8UC1);
  for(int y=0;y<h;y++)for(int x=0;x<w;x++)alpha.Set(y,x,(byte)(background[y*w+x]?0:255));
  using var soft=new Mat();Cv2.GaussianBlur(alpha,soft,new OpenCvSharp.Size(3,3),0.7);
  using var result=new Mat(new OpenCvSharp.Size(w,h),OpenCvSharp.MatType.CV_8UC4);
  for(int y=0;y<h;y++)for(int x=0;x<w;x++){
   var p=rgb.At<OpenCvSharp.Vec3b>(y,x);
   // Gentle studio contrast and rim-like highlights, no outfit recoloring.
   double light=1.035+0.06*(1.0-Math.Abs(x-w/2.0)/(w/2.0));
   byte C(byte v)=>(byte)Math.Clamp((v-128)*1.055*light+128,0,255);
   result.Set(y,x,new OpenCvSharp.Vec4b(C(p.Item0),C(p.Item1),C(p.Item2),soft.At<byte>(y,x)));
  }
  Cv2.ImEncode(".png",result,out var bytes);return bytes;
 }
 private string? frontAttire,backAttire,leftAttire,rightAttire;
 private string? PickPanel(string label){var d=new OpenFileDialog{Title="Choose "+label+" attire texture",Filter="Images|*.png;*.jpg;*.jpeg;*.bmp",CheckFileExists=true};return d.ShowDialog(this)==true?d.FileName:null;}
 private void ImportFront_Click(object sender,RoutedEventArgs e){var p=PickPanel("front");if(p!=null){frontAttire=p;FrontFileText.Text=Path.GetFileName(p);}}
 private void ImportBack_Click(object sender,RoutedEventArgs e){var p=PickPanel("back");if(p!=null){backAttire=p;BackFileText.Text=Path.GetFileName(p);}}
 private void ImportLeft_Click(object sender,RoutedEventArgs e){var p=PickPanel("left side");if(p!=null){leftAttire=p;LeftFileText.Text=Path.GetFileName(p);}}
 private void ImportRight_Click(object sender,RoutedEventArgs e){var p=PickPanel("right side");if(p!=null){rightAttire=p;RightFileText.Text=Path.GetFileName(p);}}
 private async void GenerateSeams_Click(object sender,RoutedEventArgs e){
  if(new[]{frontAttire,backAttire,leftAttire,rightAttire}.Any(string.IsNullOrEmpty)){MessageBox.Show(this,"Import all four panels: front, back, left side and right side.");return;}
  var d=new Microsoft.Win32.SaveFileDialog{Title="Choose output filename prefix",Filter="PNG image|*.png",FileName="CAWForge-Connected-Attire.png"};
  if(d.ShowDialog(this)!=true)return;
  var paths=new[]{frontAttire!,backAttire!,leftAttire!,rightAttire!};int blend=(int)SeamBlendSlider.Value;
  GenerateSeamsButton.IsEnabled=false;StatusText.Text="Connecting side panels to front and back…";
  try{
   var result=await Task.Run(()=>ConnectAttirePanels(paths,blend));
   var folder=Path.GetDirectoryName(d.FileName)!;var prefix=Path.GetFileNameWithoutExtension(d.FileName);
   var names=new[]{"Front","Back","Left","Right"};
   for(int i=0;i<4;i++)await File.WriteAllBytesAsync(Path.Combine(folder,prefix+"-"+names[i]+".png"),result[i]);
   using var ms=new MemoryStream(result[4]);var preview=new BitmapImage();preview.BeginInit();preview.StreamSource=ms;preview.CacheOption=BitmapCacheOption.OnLoad;preview.EndInit();preview.Freeze();AttirePreview.Source=preview;
   StatusText.Text="Saved four connected PNGs in "+folder;
   MessageBox.Show(this,"Four connected panel PNGs saved to:\n"+folder+"\n\nFront, Back, Left and Right. Check garment UV orientation in-game.","Attire seam export",MessageBoxButton.OK,MessageBoxImage.Information);
  }catch(Exception ex){StatusText.Text="Seam connection failed: "+ex.Message;MessageBox.Show(this,ex.Message,"Attire seam connector",MessageBoxButton.OK,MessageBoxImage.Error);}
  finally{GenerateSeamsButton.IsEnabled=true;}
 }
 private static byte[][] ConnectAttirePanels(string[] paths,int blend){
  // PNG alpha defines the garment; transparent canvas pixels are never seams.
  var originals=paths.Select(p=>Cv2.ImRead(p,OpenCvSharp.ImreadModes.Unchanged)).ToArray();
  try{
   if(originals.Any(m=>m.Empty()))throw new InvalidDataException("Could not read one of the four textures.");
   // Accept opaque black-backed texture previews as well as transparent PNGs.
   foreach(var m in originals)if(m.Channels()!=4)throw new InvalidDataException("Use four-channel PNGs for garment-safe export. Import transparent PNGs, not screenshots.");
   int w=originals[0].Width,h=originals[0].Height;
   if(w<32||h<32||w>4096||h>4096)throw new InvalidDataException("Front dimensions must be 32–4096 pixels.");
   // Preserve each source image's original size and alpha for WWE texture import.
   // Never stretch the side panels to match the front.
   var f=originals[0];var bk=originals[1];var ls=originals[2];var rs=originals[3];
   // Left panel joins the left silhouette of front and the right silhouette of back.
   // Right panel joins the right silhouette of front and the left silhouette of back.
   // Each edge is found per scanline from the alpha mask, not the canvas border.
   AlignSidePattern(bk,ls,false,true,blend);
   JoinSilhouette(bk,ls,false,true,blend);
   AlignSidePattern(ls,f,false,true,blend);
   JoinSilhouette(ls,f,false,true,blend);
   AlignSidePattern(f,rs,false,true,blend);
   JoinSilhouette(f,rs,false,true,blend);
   AlignSidePattern(rs,bk,false,true,blend);
   JoinSilhouette(rs,bk,false,true,blend);
   var result=new byte[5][];
   for(int i=0;i<4;i++)Cv2.ImEncode(".png",originals[i],out result[i]);
   // A contact sheet is a preview, not a claim of 3D UV alignment.
   int tileW=260,tileH=260;
   using var sheet=new Mat(new OpenCvSharp.Size(tileW*4,tileH),OpenCvSharp.MatType.CV_8UC4,new OpenCvSharp.Scalar(35,35,35,255));
   var ordered=new[]{bk,ls,f,rs};
   for(int i=0;i<4;i++){
    double scale=Math.Min((double)(tileW-12)/ordered[i].Width,(double)(tileH-12)/ordered[i].Height);
    using var scaled=new Mat();Cv2.Resize(ordered[i],scaled,new OpenCvSharp.Size(Math.Max(1,(int)(ordered[i].Width*scale)),Math.Max(1,(int)(ordered[i].Height*scale))));
    using var dest=new Mat(sheet,new OpenCvSharp.Rect(i*tileW+(tileW-scaled.Width)/2,(tileH-scaled.Height)/2,scaled.Width,scaled.Height));
    for(int y=0;y<scaled.Height;y++)for(int x=0;x<scaled.Width;x++){
     var p=scaled.At<OpenCvSharp.Vec4b>(y,x);double alpha=p.Item3/255.0;
     var bg=dest.At<OpenCvSharp.Vec4b>(y,x);
     dest.Set(y,x,new OpenCvSharp.Vec4b((byte)(p.Item0*alpha+bg.Item0*(1-alpha)),(byte)(p.Item1*alpha+bg.Item1*(1-alpha)),(byte)(p.Item2*alpha+bg.Item2*(1-alpha)),255));
    }
   }
   Cv2.ImEncode(".png",sheet,out result[4]);return result;
  }finally{foreach(var m in originals)m.Dispose();}
 }
 private static void AlignSidePattern(Mat a,Mat b,bool aLeft,bool bLeft,int blend){
  // Determine a vertical offset from edge color features. Restrict the
  // adjustment to a side panel's near-seam band, preserving its UV silhouette.
  // This is heuristic pattern alignment, not a calibrated WWE UV map.
  int maxShift=Math.Min(28,Math.Min(a.Height,b.Height)/10);
  double best=double.MaxValue;int shift=0;
  for(int dy=-maxShift;dy<=maxShift;dy+=2){
   double error=0;int count=0;
   for(int row=0;row<Math.Min(a.Height,b.Height);row+=4){
    int other=row+dy;if(other<0||other>=b.Height)continue;
    int x=FindSilhouetteEdge(a,row,aLeft),z=FindSilhouetteEdge(b,other,bLeft);
    if(x<0||z<0)continue;
    var p=a.At<OpenCvSharp.Vec4b>(row,x);var q=b.At<OpenCvSharp.Vec4b>(other,z);
    if(p.Item3<128||q.Item3<128)continue;
    error+=Math.Abs((int)p.Item0-q.Item0)+Math.Abs((int)p.Item1-q.Item1)+Math.Abs((int)p.Item2-q.Item2);count++;
   }
   if(count<12)continue;
   double score=error/count+Math.Abs(dy)*0.7;
   if(score<best){best=score;shift=dy;}
  }
  if(shift==0||!double.IsFinite(best))return;
  // Shift RGB details vertically only inside the seam strip. Keep alpha and
  // garment silhouette unchanged; feather the adjustment into the interior.
  using var original=b.Clone();
  int width=Math.Min(Math.Max(4,blend*3),b.Width/3);
  for(int y=0;y<b.Height;y++){
   int edge=FindSilhouetteEdge(b,y,bLeft);if(edge<0)continue;
   int srcY=Math.Clamp(y+shift,0,b.Height-1);
   int sourceEdge=FindSilhouetteEdge(original,srcY,bLeft);if(sourceEdge<0)continue;
   for(int d=0;d<width;d++){
    int x=edge+(bLeft?d:-d),sx=sourceEdge+(bLeft?d:-d);
    if(x<0||x>=b.Width||sx<0||sx>=b.Width)break;
    var current=b.At<OpenCvSharp.Vec4b>(y,x);var source=original.At<OpenCvSharp.Vec4b>(srcY,sx);
    if(current.Item3<128||source.Item3<128)continue;
    double weight=1.0-(double)d/width;weight=weight*weight*(3-2*weight);
    b.Set(y,x,new OpenCvSharp.Vec4b(
     (byte)(current.Item0*(1-weight)+source.Item0*weight),
     (byte)(current.Item1*(1-weight)+source.Item1*weight),
     (byte)(current.Item2*(1-weight)+source.Item2*weight),current.Item3));
   }
  }
 }
 private static void JoinSilhouette(Mat a,Mat b,bool aLeft,bool bLeft,int blend){
  // Pair rows by proportional vertical position; do not alter transparency,
  // silhouettes, logos, or the center of the garment.
  int rows=Math.Min(a.Height,b.Height),band=Math.Max(2,blend);
  for(int row=0;row<rows;row++){
   int ay=(int)((row+0.5)*a.Height/rows),by=(int)((row+0.5)*b.Height/rows);
   ay=Math.Min(a.Height-1,ay);by=Math.Min(b.Height-1,by);
   int ax=FindSilhouetteEdge(a,ay,aLeft),bx=FindSilhouetteEdge(b,by,bLeft);
   if(ax<0||bx<0)continue;
   var av=a.At<OpenCvSharp.Vec4b>(ay,ax);var bv=b.At<OpenCvSharp.Vec4b>(by,bx);
   // Match colors on the two actual garment boundaries. Blend gradually
   // inward, never into transparent pixels or across the whole image.
   double[] seam={((int)av.Item0+bv.Item0)/2.0,((int)av.Item1+bv.Item1)/2.0,((int)av.Item2+bv.Item2)/2.0};
   for(int d=0;d<band;d++){
    int x1=ax+(aLeft?d:-d),x2=bx+(bLeft?d:-d);
    if(x1<0||x1>=a.Width||x2<0||x2>=b.Width)break;
    var p=a.At<OpenCvSharp.Vec4b>(ay,x1);var q=b.At<OpenCvSharp.Vec4b>(by,x2);
    if(p.Item3<128||q.Item3<128)break;
    double t=1.0-(double)d/band;t=t*t*(3-2*t);
    a.Set(ay,x1,new OpenCvSharp.Vec4b((byte)(p.Item0*(1-t)+seam[0]*t),(byte)(p.Item1*(1-t)+seam[1]*t),(byte)(p.Item2*(1-t)+seam[2]*t),p.Item3));
    b.Set(by,x2,new OpenCvSharp.Vec4b((byte)(q.Item0*(1-t)+seam[0]*t),(byte)(q.Item1*(1-t)+seam[1]*t),(byte)(q.Item2*(1-t)+seam[2]*t),q.Item3));
   }
  }
 }
 private static int FindSilhouetteEdge(Mat m,int y,bool left){
  if(left){for(int x=0;x<m.Width;x++)if(m.At<OpenCvSharp.Vec4b>(y,x).Item3>=128)return x;}
  else{for(int x=m.Width-1;x>=0;x--)if(m.At<OpenCvSharp.Vec4b>(y,x).Item3>=128)return x;}
  return -1;
 }
 private async void MatchTemplate_Click(object sender,RoutedEventArgs e){await MatchTemplateAsync();}
 private async Task MatchTemplateAsync(){
  if(selectedPhoto is null){MatchStatusText.Text="Import a reference photo first.";return;}
  var gender=GenderBox.SelectedIndex==0?"M":"F";
  var folder=Path.Combine(AppContext.BaseDirectory,"Templates");
  var photo=selectedPhoto;
  var request=++recognitionVersion;
  MatchTemplateButton.IsEnabled=false;MatchStatusText.Text="Comparing installed template references in background…";
  try{
   var matches=await Task.Run(()=>{
    var files=Enumerable.Range(1,gender=="F"?21:20)
     .SelectMany(i=>new[]{".png",".jpg",".jpeg",".bmp"}.Select(ext=>(id:i,path:Path.Combine(folder,gender+i+ext))))
     .Where(x=>File.Exists(x.path)).ToList();
    if(files.Count==0)return new List<(int id,double distance,double[] descriptor)>();
    var reference=ReadVisualDescriptor(photo);
    return files.Select(x=>{try{var descriptor=ReadVisualDescriptor(x.path);return (x.id,distance:VisualDistance(reference,descriptor),descriptor);}catch{return (x.id,distance:double.PositiveInfinity,descriptor:Array.Empty<double>());}})
     .Where(x=>double.IsFinite(x.distance)).OrderBy(x=>x.distance).Take(3).ToList();
   });
   if(request!=recognitionVersion || photo!=selectedPhoto)return;
   if(matches.Count==0){MatchStatusText.Text="No usable "+gender+" reference images found in "+folder+". Add M1–M20 or F1–F21 PNG/JPG files.";return;}
   var best=matches[0];TemplateBox.SelectedIndex=best.id-1;
   var faceResult=await Task.Run(()=>DetectFace(photo));
   if(request!=recognitionVersion || photo!=selectedPhoto)return;
   if(faceResult is null){MatchStatusText.Text="No face detected. Template selection was based on image similarity; automatic slider estimates were not applied.";return;}
   var photoDescriptor=await Task.Run(()=>ReadVisualDescriptor(photo));
   if(request!=recognitionVersion || photo!=selectedPhoto)return;
   ApplyEstimatedMorph(photoDescriptor,best.descriptor);
   MatchStatusText.Text="Face detected. Closest visual references: "+string.Join(", ",matches.Select(x=>gender+x.id))+". Selected "+gender+best.id+". Experimental image similarity, NOT facial landmark recognition.";
   StatusText.Text="Template suggestion selected. Verify against the in-game face.";
  }catch(Exception ex){MatchStatusText.Text="Template comparison failed: "+ex.Message;}
  finally{if(request==recognitionVersion)MatchTemplateButton.IsEnabled=true;}
 }
 private static double[] ReadVisualDescriptor(string path){
  using var stream=File.OpenRead(path);
  var decoder=BitmapDecoder.Create(stream,BitmapCreateOptions.PreservePixelFormat,BitmapCacheOption.OnLoad);
  var frame=decoder.Frames[0];
  if((long)frame.PixelWidth*frame.PixelHeight>35000000)throw new InvalidDataException("Reference exceeds 35 megapixels.");
  int side=Math.Min(frame.PixelWidth,frame.PixelHeight);
  var crop=new CroppedBitmap(frame,new System.Windows.Int32Rect((frame.PixelWidth-side)/2,(frame.PixelHeight-side)/2,side,side));crop.Freeze();
  var scaled=new TransformedBitmap(crop,new System.Windows.Media.ScaleTransform(16.0/side,16.0/side));scaled.Freeze();
  var gray=new FormatConvertedBitmap(scaled,System.Windows.Media.PixelFormats.Gray8,null,0);gray.Freeze();
  int w=gray.PixelWidth,h=gray.PixelHeight;
  var pixels=new byte[w*h];gray.CopyPixels(pixels,w,0);
  var result=new double[pixels.Length];double mean=pixels.Average(x=>(double)x);
  double sd=Math.Sqrt(pixels.Average(x=>Math.Pow(x-mean,2)));if(sd<1)sd=1;
  for(int i=0;i<pixels.Length;i++)result[i]=(pixels[i]-mean)/sd;
  return result;
 }
 private static double VisualDistance(double[] a,double[] b){
  if(a.Length!=b.Length)return double.PositiveInfinity;
  double sum=0;for(int i=0;i<a.Length;i++){var d=a[i]-b[i];sum+=d*d;}return sum;
 }

 private void ApplyEstimatedMorph(double[] photo,double[] template){
  if(photo.Length<32 || photo.Length!=template.Length)return;
  foreach(var entry in morphSliders){
   uint hash=2166136261;
   foreach(char ch in entry.Key)hash=(hash^ch)*16777619;
   int start=(int)(hash%(uint)(photo.Length-18));
   double delta=0;
   for(int j=0;j<18;j++)delta+=(photo[start+j]-template[start+j])/18.0;
   entry.Value.Value=Math.Clamp(Math.Round(50+delta*13),0,100);
  }
 }

 private static OpenCvSharp.Rect? DetectFace(string path){
  var cascadePath=Path.Combine(AppContext.BaseDirectory,"Models","haarcascade_frontalface_default.xml");
  if(!File.Exists(cascadePath))return null;
  using var cascade=new CascadeClassifier(cascadePath);
  if(cascade.Empty())return null;
  using var source=Cv2.ImRead(path,ImreadModes.Grayscale);
  if(source.Empty())return null;
  using var resized=new Mat();
  double factor=Math.Min(1.0,640.0/Math.Max(source.Width,source.Height));
  if(factor<1)Cv2.Resize(source,resized,new OpenCvSharp.Size(),factor,factor);
  else source.CopyTo(resized);
  Cv2.EqualizeHist(resized,resized);
  var faces=cascade.DetectMultiScale(resized,1.1,5,HaarDetectionTypes.ScaleImage,new OpenCvSharp.Size(36,36));
  if(faces.Length==0)return null;
  return faces.OrderByDescending(x=>x.Width*x.Height).First();
 }
}