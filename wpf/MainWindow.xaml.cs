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
 private readonly List<(string Name,string Path)> fusionLibrary=new();
 private static string InstalledFaceTemplates=>Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),"CAWForge","FaceTextureTemplates");
 private void RefreshFusionLibrary(string? preferred=null){
  fusionLibrary.Clear();FusionLibraryBox.Items.Clear();
  var directories=new[]{InstalledFaceTemplates,Path.Combine(AppContext.BaseDirectory,"FaceTextureTemplates"),preferred}
   .Where(p=>!string.IsNullOrWhiteSpace(p)&&Directory.Exists(p)).Distinct(StringComparer.OrdinalIgnoreCase);
  foreach(var folder in directories){
   foreach(var file in Directory.EnumerateFiles(folder!,"*.png",SearchOption.AllDirectories).OrderBy(p=>p)){
    var name=Path.GetFileNameWithoutExtension(file);
    if(!System.Text.RegularExpressions.Regex.IsMatch(name,@"^(?:Female(?: Template)? ?[1-9][0-9]*|Female Skintone [1-9][0-9]*|Male(?: Template)? ?[1-9][0-9]*|[FM][1-9][0-9]*)$",System.Text.RegularExpressions.RegexOptions.IgnoreCase))continue;
    using var check=Cv2.ImRead(file,ImreadModes.Unchanged);
    if(check.Empty()||check.Width!=820||check.Height!=820)continue;
    if(fusionLibrary.Any(t=>t.Name.Equals(name,StringComparison.OrdinalIgnoreCase)))continue;
    fusionLibrary.Add((name,file));
   }
  }
  fusionLibrary.Sort((a,b)=>string.Compare(a.Name,b.Name,StringComparison.OrdinalIgnoreCase));
  foreach(var item in fusionLibrary)FusionLibraryBox.Items.Add(item.Name);
  if(fusionLibrary.Count>0){
   FusionLibraryBox.SelectedIndex=0;
   FusionTemplateText.Text=fusionLibrary.Count+" installed templates — choose one above";
  }else{
   fusionTemplatePath=null;
   FusionTemplateText.Text="No templates installed. Select your CAWForge template ZIP once to install the complete library.";
  }
 }
 private void FusionLibraryBox_SelectionChanged(object sender,System.Windows.Controls.SelectionChangedEventArgs e){
  int index=FusionLibraryBox.SelectedIndex;
  if(index<0||index>=fusionLibrary.Count)return;
  fusionTemplatePath=fusionLibrary[index].Path;
  FusionTemplateText.Text="Selected: "+fusionLibrary[index].Name;
  try{
   var preview=new BitmapImage();
   preview.BeginInit();preview.CacheOption=BitmapCacheOption.OnLoad;
   preview.UriSource=new Uri(fusionTemplatePath,UriKind.Absolute);
   preview.EndInit();preview.Freeze();FusionTemplatePreview.Source=preview;
  }catch{FusionTemplatePreview.Source=null;}
 }
 private void LoadFusionLibrary_Click(object sender,RoutedEventArgs e){
  var dlg=new OpenFileDialog{Title="Install your CAWForge face template pack",Filter="Template pack ZIP or PNG|*.zip;*.png",CheckFileExists=true};
  if(dlg.ShowDialog(this)!=true)return;
  try{
   Directory.CreateDirectory(InstalledFaceTemplates);
   if(Path.GetExtension(dlg.FileName).Equals(".zip",StringComparison.OrdinalIgnoreCase)){
    using var archive=System.IO.Compression.ZipFile.OpenRead(dlg.FileName);
    int count=0;
    foreach(var entry in archive.Entries){
     if(!entry.FullName.EndsWith(".png",StringComparison.OrdinalIgnoreCase))continue;
     var filename=Path.GetFileName(entry.FullName);
     if(!System.Text.RegularExpressions.Regex.IsMatch(Path.GetFileNameWithoutExtension(filename),@"^(?:Female(?: Template)? ?[1-9][0-9]*|Female Skintone [1-9][0-9]*|Male(?: Template)? ?[1-9][0-9]*|[FM][1-9][0-9]*)$",System.Text.RegularExpressions.RegexOptions.IgnoreCase))continue;
     if(entry.Length>12000000)continue;
     var target=Path.Combine(InstalledFaceTemplates,filename);
     using(var input=entry.Open())using(var output=File.Create(target))input.CopyTo(output);
     count++;
    }
    if(count==0)throw new InvalidDataException("No matching female or male template PNGs found in ZIP.");
   }else{
    File.Copy(dlg.FileName,Path.Combine(InstalledFaceTemplates,Path.GetFileName(dlg.FileName)),true);
   }
   RefreshFusionLibrary();
   MessageBox.Show(this,$"{fusionLibrary.Count} face templates installed. They will be available automatically next time.","Template library");
  }catch(Exception ex){MessageBox.Show(this,"Could not install templates: "+ex.Message,"Template library",MessageBoxButton.OK,MessageBoxImage.Error);}
 }
 private string? fusionTemplatePath;
 private void ImportFusionTemplate_Click(object sender,RoutedEventArgs e){
  var dlg=new OpenFileDialog{Title="Select 820 x 820 Sarina master face texture",Filter="PNG images|*.png",CheckFileExists=true};
  if(dlg.ShowDialog(this)!=true)return;
  using var check=Cv2.ImRead(dlg.FileName,ImreadModes.Color);
  if(check.Empty()||check.Width!=820||check.Height!=820){
   MessageBox.Show(this,"The master template must be a valid 820 x 820 PNG.","Face Scan Fusion",MessageBoxButton.OK,MessageBoxImage.Warning);return;
  }
  fusionTemplatePath=dlg.FileName;
  FusionTemplateText.Text="Master: "+Path.GetFileName(dlg.FileName);
 }
 private string? selectedPhoto;
 private string? attirePhoto;
 private int recognitionVersion=0;
 private readonly Dictionary<string,Slider> morphSliders=new();
 private readonly Dictionary<string,List<StackPanel>> morphCategoryPanels=new();
 public MainWindow() { InitializeComponent(); RefreshFusionLibrary(); for(int i=1;i<=21;i++) TemplateBox.Items.Add("Template "+i); TemplateBox.SelectedIndex=9; foreach(var group in new (string Category,string[] Names)[]{("Head",new[]{"Head Width","Head Height","Head Depth"}),("Ears Top",new[]{"Offset","Height","Alignment"}),("Ears",new[]{"Ears Width","Ears Height","Vertical Stretch","Placement Low/High","Placement Forward/Back"}),("Earlobes",new[]{"Lobe Detachment","Lobe Length","Lobe Alignment"}),("Forehead Upper",new[]{"Forehead Upper Size"}),("Forehead",new[]{"Forehead Size"}),("Eyebrows",new[]{"Eyebrow Length","Eyebrow Depth","Eyebrow Corner Height Left","Eyebrow Corner Height Right","Eyebrow Outer Height Left","Eyebrow Outer Height Right"}),("Eyes",new[]{"Eye Length","Eye Socket Depth","Eye Corner Length","Eye Corner Height","Eye Outer Length","Eye Outer Height"}),("Nose",new[]{"Nose Width","Nose Height","Nose Depth"}),("Nose Bridge",new[]{"Nose Bridge Width","Nose Bridge Height","Nose Bridge Depth"}),("Nose Tip",new[]{"Nose Tip Width","Tip Pointed or Rounded","Nose Tip Length","Tip Height"}),("Nostrils",new[]{"Left Nostril Width","Left Nostril Height","Left Nostril Depth","Right Nostril Width","Right Nostril Height","Right Nostril Depth"}),("Philtrum",new[]{"Philtrum Width","Philtrum Depth"}),("Mouth",new[]{"Mouth Horizontal Length","Mouth Fullness","Mouth Depth","Left Mouth Corner Length","Right Mouth Corner Length","Left Mouth Corner Height","Right Mouth Corner Height"}),("Cheekbones",new[]{"Upper Cheekbone","Middle Cheekbone","Lower Cheekbone"}),("Cheeks",new[]{"Cheek Fullness","Cheek Height"}),("Jawline",new[]{"Middle Jaw Fullness","Lower Jaw Height","Lower Jaw Depth","Rear Jaw Fullness","Lower Jaw Fullness","Lower Jaw Length"}),("Chin",new[]{"Chin Width","Chin Height","Chin Forward/Back","Chin Vertical Placement"}),("Parietal Region",new[]{"Parietal Region Size"})}) {morphCategoryPanels[group.Category]=new List<StackPanel>(); foreach(var name in group.Names) {var panel=new StackPanel{Margin=new Thickness(0,0,0,8)};panel.Children.Add(new TextBlock{Text=group.Category+" / "+name,Foreground=System.Windows.Media.Brushes.White});panel.Children.Add(new Slider{Minimum=0,Maximum=100,Value=50,Width=360,HorizontalAlignment=HorizontalAlignment.Left});morphCategoryPanels[group.Category].Add(panel);morphSliders[group.Category+" / "+name]=(Slider)panel.Children[1];} var categoryName=group.Category;var categoryButton=new Button{Content=categoryName,Margin=new Thickness(0,0,8,8),Padding=new Thickness(12,7,12,7),MinWidth=110};categoryButton.Click+=(_,_)=>ShowMorphCategory(categoryName);MorphCategoryButtons.Children.Add(categoryButton);} ShowMorphCategory("Head"); }
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
   var master=fusionTemplatePath;
   var png=await Task.Run(()=>MakeFaceTexture(source,upscale,master));
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
 private static byte[] MakeFaceTexture(string path,int upscale=1,string? masterPath=null){
  using var src=Cv2.ImRead(path,ImreadModes.Color);
  if(src.Empty())throw new InvalidDataException("Unable to decode reference photo.");
  if((long)src.Width*src.Height>35000000)throw new InvalidDataException("Image exceeds 35 megapixels.");
  // Refuse previous generated textures and composite/quadrant source images.
  // They must never be recycled as reference photographs.
  if(Path.GetFileName(path).StartsWith("CAWForge-Face-Texture",StringComparison.OrdinalIgnoreCase))
   throw new InvalidDataException("A generated face texture was selected as the reference photo. Import the original, unedited portrait in Morph Settings.");
  if(!string.IsNullOrEmpty(masterPath)&&Path.GetFullPath(path).Equals(Path.GetFullPath(masterPath),StringComparison.OrdinalIgnoreCase))
   throw new InvalidDataException("The reference photo and face template are the same file. Import a real portrait in Morph Settings.");
  // Large straight mid-image discontinuities indicate a four-panel collage,
  // not a continuous portrait. Stop rather than exporting another broken PNG.
  double VerticalSeam(int x){
   if(x<2||x>=src.Width-2)return 0;
   double sum=0;int n=0;
   for(int y=src.Height/6;y<src.Height*5/6;y+=3){
    var a=src.At<OpenCvSharp.Vec3b>(y,x-1);var b=src.At<OpenCvSharp.Vec3b>(y,x);
    sum+=Math.Abs((int)a.Item0-b.Item0)+Math.Abs((int)a.Item1-b.Item1)+Math.Abs((int)a.Item2-b.Item2);n++;
   }
   return sum/Math.Max(1,n);
  }
  double HorizontalSeam(int y){
   if(y<2||y>=src.Height-2)return 0;
   double sum=0;int n=0;
   for(int x=src.Width/6;x<src.Width*5/6;x+=3){
    var a=src.At<OpenCvSharp.Vec3b>(y-1,x);var b=src.At<OpenCvSharp.Vec3b>(y,x);
    sum+=Math.Abs((int)a.Item0-b.Item0)+Math.Abs((int)a.Item1-b.Item1)+Math.Abs((int)a.Item2-b.Item2);n++;
   }
   return sum/Math.Max(1,n);
  }
  double centerV=VerticalSeam(src.Width/2),centerH=HorizontalSeam(src.Height/2);
  if(centerV>65&&centerH>65)
   throw new InvalidDataException("The reference appears to be a four-panel composite with hard seams. Import the original portrait, not a previously generated texture.");
  // Detect the face; never use the whole photograph as the texture.
  OpenCvSharp.Rect face;
  var classifierPath=Path.Combine(AppContext.BaseDirectory,"Models","haarcascade_frontalface_default.xml");
  if(!File.Exists(classifierPath))throw new FileNotFoundException("Face detector is missing. Reinstall the complete CAWForge build.",classifierPath);
  using(var detector=new CascadeClassifier(classifierPath)){
   var found=detector.DetectMultiScale(src,1.1,5);
   if(found.Length==0)throw new InvalidDataException("No frontal face detected. Use a clear, straight-on reference photograph.");
   face=found.OrderByDescending(f=>f.Width*f.Height).First();
  }
  // Validate that the detected region contains a genuine frontal pair of
  // eyes. A collage of hair/cheeks can fool the generic face detector.
  var eyeCascadePath=Path.Combine(AppContext.BaseDirectory,"Models","haarcascade_eye_tree_eyeglasses.xml");
  if(!File.Exists(eyeCascadePath))throw new FileNotFoundException("Eye landmark classifier missing from this build.",eyeCascadePath);
  using(var eyeDetector=new CascadeClassifier(eyeCascadePath)){
   int upperHeight=Math.Min(src.Height-face.Y,Math.Max(1,(int)(face.Height*0.65)));
   using var upperFace=new Mat(src,new OpenCvSharp.Rect(face.X,face.Y,face.Width,upperHeight));
   var eyes=eyeDetector.DetectMultiScale(upperFace,1.07,3,minSize:new OpenCvSharp.Size(Math.Max(8,face.Width/11),Math.Max(8,face.Height/12)));
   var candidates=eyes.Where(e=>e.Y+e.Height/2.0<upperHeight*0.86).OrderByDescending(e=>e.Width*e.Height).Take(8).ToArray();
   bool pair=false;
   for(int i=0;i<candidates.Length;i++)for(int j=i+1;j<candidates.Length;j++){
    var e=candidates[i];var f=candidates[j];
    double dx=Math.Abs((e.X+e.Width/2.0)-(f.X+f.Width/2.0));
    double dy=Math.Abs((e.Y+e.Height/2.0)-(f.Y+f.Height/2.0));
    if(dx>face.Width*0.19&&dx<face.Width*0.72&&dy<face.Height*0.17){pair=true;break;}
   }
   if(!pair)throw new InvalidDataException("Face Scan Fusion stopped: two aligned eyes could not be verified in the original reference photo. The input may be cropped, a collage, or not frontal. No broken texture was exported.");
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
  // Template is the actual seamless UV base. Never tile rectangular photo
  // patches or use a blurred copy of the original photo as the underskin.
  using var texture=new Mat(new OpenCvSharp.Size(820,820),OpenCvSharp.MatType.CV_8UC3);
  if(string.IsNullOrEmpty(masterPath)){
   // Without a template, return a continuous photographic face rather than
   // fabricate an underskin from arbitrary blocks.
   aligned.CopyTo(texture);
  }else{
   using var master=Cv2.ImRead(masterPath,ImreadModes.Color);
   if(master.Empty()||master.Width!=820||master.Height!=820)
    throw new InvalidDataException("The selected face template must be an 820 x 820 PNG.");
   // Sample skin (not eyes, brows or lips) to match the entire template
   // to the reference. Sampling uses several cheek locations.
   var samples=new (int X,int Y)[]{(250,490),(280,530),(555,490),(535,530),(330,635),(490,635)};
   double[] offsets=new double[3];
   foreach(var point in samples){
    var facePixel=aligned.At<OpenCvSharp.Vec3b>(point.Y,point.X);
    var basePixel=master.At<OpenCvSharp.Vec3b>(point.Y,point.X);
    offsets[0]+=facePixel.Item0-basePixel.Item0;
    offsets[1]+=facePixel.Item1-basePixel.Item1;
    offsets[2]+=facePixel.Item2-basePixel.Item2;
   }
   for(int k=0;k<3;k++)offsets[k]=Math.Clamp(offsets[k]/samples.Length,-85,85);
   // A broad oval replaces template identity details with the user's
   // photographed features; outer UV and ears stay template-shaped.
   // A smoothstep feather has no straight rectangular boundaries.
   for(int y=0;y<820;y++)for(int x=0;x<820;x++){
    var original=aligned.At<OpenCvSharp.Vec3b>(y,x);
    var basis=master.At<OpenCvSharp.Vec3b>(y,x);
    double dx=(x-410.0)/305.0,dy=(y-460.0)/350.0;
    double distance=Math.Sqrt(dx*dx+dy*dy);
    double blend=Math.Clamp((1.0-distance)/0.20,0,1);
    blend=blend*blend*(3-2*blend);
    // Neutral-expression mouth: for smiling reference photos, the
    // template's closed, neutral mouth takes precedence in the lip region.
    // Smooth feathering prevents rectangular mouth-patch boundaries.
    double mouthX=(x-410.0)/155.0,mouthY=(y-610.0)/78.0;
    double mouthRadius=Math.Sqrt(mouthX*mouthX+mouthY*mouthY);
    double mouthNeutral=Math.Clamp((1.0-mouthRadius)/0.24,0.0,1.0);
    mouthNeutral=mouthNeutral*mouthNeutral*(3-2*mouthNeutral);
    blend*=1.0-mouthNeutral;
    // Brow region is fully photographic; never fade eyebrows away.
    if(y>=275&&y<=410&&x>=205&&x<=615&&distance<0.83)blend=1;
    byte Mix(byte detail,byte template,double offset)=>
     (byte)Math.Clamp(detail*blend+Math.Clamp(template+offset,0,255)*(1-blend),0,255);
    texture.Set(y,x,new OpenCvSharp.Vec3b(
     Mix(original.Item0,basis.Item0,offsets[0]),
     Mix(original.Item1,basis.Item1,offsets[1]),
     Mix(original.Item2,basis.Item2,offsets[2])));
   }
  }
  // Final sanity check: reject outputs that lost the face during processing.
  // Do not overwrite a previously usable PNG with a broken export.
  using(var faceCheck=new CascadeClassifier(classifierPath)){
   var verified=faceCheck.DetectMultiScale(texture,1.08,3);
   if(verified.Length==0)throw new InvalidDataException("Generated texture failed facial integrity validation. Export cancelled; use a clearer frontal reference photo.");
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
  using var rgb=new Mat();
  Cv2.Resize(source,rgb,new OpenCvSharp.Size(fitW,fitH),0,0,OpenCvSharp.InterpolationFlags.Lanczos4);
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
  // Select the dominant central connected subject, excluding isolated HUD
  // prompts and floating menu elements. Preserve all limbs within the mask.
  using var binary=new Mat();Cv2.Threshold(soft,binary,128,255,OpenCvSharp.ThresholdTypes.Binary);
  using var labels=new Mat();using var stats=new Mat();using var centroids=new Mat();
  int components=Cv2.ConnectedComponentsWithStats(binary,labels,stats,centroids,OpenCvSharp.PixelConnectivity.Connectivity8,OpenCvSharp.MatType.CV_32S);
  int chosen=-1;double bestScore=0;
  for(int i=1;i<components;i++){
   int x=stats.At<int>(i,(int)OpenCvSharp.ConnectedComponentsTypes.Left);
   int y=stats.At<int>(i,(int)OpenCvSharp.ConnectedComponentsTypes.Top);
   int ww=stats.At<int>(i,(int)OpenCvSharp.ConnectedComponentsTypes.Width);
   int hh=stats.At<int>(i,(int)OpenCvSharp.ConnectedComponentsTypes.Height);
   int area=stats.At<int>(i,(int)OpenCvSharp.ConnectedComponentsTypes.Area);
   double cx=(x+ww/2.0)/w,cy=(y+hh/2.0)/h;
   double centrality=1.0/(1.0+2.0*Math.Abs(cx-0.5)+Math.Abs(cy-0.5));
   double score=area*centrality;
   if(hh<h*0.15||area<200)continue;
   if(score>bestScore){bestScore=score;chosen=i;}
  }
  if(chosen<0)throw new InvalidDataException("No complete character was isolated. Try a screenshot with a plain backdrop.");
  int bx=stats.At<int>(chosen,(int)OpenCvSharp.ConnectedComponentsTypes.Left);
  int by=stats.At<int>(chosen,(int)OpenCvSharp.ConnectedComponentsTypes.Top);
  int bw=stats.At<int>(chosen,(int)OpenCvSharp.ConnectedComponentsTypes.Width);
  int bh=stats.At<int>(chosen,(int)OpenCvSharp.ConnectedComponentsTypes.Height);
  if(bw>=w-2||bh>=h-2)throw new InvalidDataException("Background isolation included the entire screenshot. Use a plain in-game background for a clean render.");
  // Only retain pixels belonging to the selected connected character.
  for(int y=0;y<h;y++)for(int x=0;x<w;x++){
   if(labels.At<int>(y,x)!=chosen){
    var pixel=result.At<OpenCvSharp.Vec4b>(y,x);
    result.Set(y,x,new OpenCvSharp.Vec4b(pixel.Item0,pixel.Item1,pixel.Item2,0));
   }
  }
  // Scale the isolated subject to occupy up to 88% of the render, without
  // clipping any visible arm, hand, boot, hair or accessory.
  int margin=Math.Max(8,(int)(Math.Max(bw,bh)*0.025));
  bx=Math.Max(0,bx-margin);by=Math.Max(0,by-margin);
  bw=Math.Min(w-bx,bw+2*margin);bh=Math.Min(h-by,bh+2*margin);
  double scale=Math.Min(900.0/bw,900.0/bh);
  int outW=Math.Clamp((int)Math.Round(bw*scale),1,900);
  int outH=Math.Clamp((int)Math.Round(bh*scale),1,900);
  using var subject=new Mat(result,new OpenCvSharp.Rect(bx,by,bw,bh));
  using var enlarged=new Mat();Cv2.Resize(subject,enlarged,new OpenCvSharp.Size(outW,outH),0,0,OpenCvSharp.InterpolationFlags.Lanczos4);
  using var canvas=new Mat(new OpenCvSharp.Size(1024,1024),OpenCvSharp.MatType.CV_8UC4,new OpenCvSharp.Scalar(0,0,0,0));
  using(var region=new Mat(canvas,new OpenCvSharp.Rect((1024-outW)/2,(1024-outH)/2,outW,outH)))enlarged.CopyTo(region);
  Cv2.ImEncode(".png",canvas,out var bytes);return bytes;
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