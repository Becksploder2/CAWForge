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
   var png=await Task.Run(()=>MakeFaceTexture(source));
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
 private static byte[] MakeFaceTexture(string path){
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
  Cv2.Resize(crop,aligned,new OpenCvSharp.Size(820,820));
  // Use genuine cheek skin for a seamless, full-canvas underskin. Never
  // mirror the photo, preserve its original background, or paste an oval.
  // Facial detail is composited only where the image plausibly contains skin.
  var cheekA=aligned.At<OpenCvSharp.Vec3b>(490,255);
  var cheekB=aligned.At<OpenCvSharp.Vec3b>(490,565);
  var browSkin=aligned.At<OpenCvSharp.Vec3b>(245,410);
  var chinSkin=aligned.At<OpenCvSharp.Vec3b>(685,410);
  using var texture=new Mat(new OpenCvSharp.Size(820,820),OpenCvSharp.MatType.CV_8UC3);
  for(int y=0;y<820;y++)for(int x=0;x<820;x++){
   double u=x/819.0,v=y/819.0;
   double[] baseTone=new double[3];
   for(int c=0;c<3;c++){
    double l=c==0?cheekA.Item0:c==1?cheekA.Item1:cheekA.Item2;
    double rr=c==0?cheekB.Item0:c==1?cheekB.Item1:cheekB.Item2;
    double forehead=c==0?browSkin.Item0:c==1?browSkin.Item1:browSkin.Item2;
    double chin=c==0?chinSkin.Item0:c==1?chinSkin.Item1:chinSkin.Item2;
    baseTone[c]=0.72*(l*(1-u)+rr*u)+0.28*(forehead*(1-v)+chin*v);
   }
   // Fade into sampled skin on all four edges. No reflection or circular
   // photo boundary. Constrain top and sides to avoid source hair and ears.
   double h=Math.Clamp((x-215.0)/90.0,0,1)*Math.Clamp((605.0-x)/90.0,0,1);
   double vMask=Math.Clamp((y-255.0)/95.0,0,1)*Math.Clamp((690.0-y)/85.0,0,1);
   double alpha=h*vMask;
   alpha=alpha*alpha*(3-2*alpha);
   var p=aligned.At<OpenCvSharp.Vec3b>(y,x);
   texture.Set(y,x,new OpenCvSharp.Vec3b(
    (byte)Math.Clamp(p.Item0*alpha+baseTone[0]*(1-alpha),0,255),
    (byte)Math.Clamp(p.Item1*alpha+baseTone[1]*(1-alpha),0,255),
    (byte)Math.Clamp(p.Item2*alpha+baseTone[2]*(1-alpha),0,255)));
  }
  Cv2.ImEncode(".png",texture,out var bytes);
  return bytes;
 }

 private void ImportAttire_Click(object sender,RoutedEventArgs e){
  var dlg=new OpenFileDialog{Title="Choose attire pattern",Filter="Images|*.png;*.jpg;*.jpeg;*.bmp;*.webp",CheckFileExists=true};
  if(dlg.ShowDialog(this)!=true)return;
  attirePhoto=dlg.FileName;AttireFileText.Text=Path.GetFileName(attirePhoto);
  StatusText.Text="Attire pattern selected.";
 }
 private async void GenerateSeams_Click(object sender,RoutedEventArgs e){
  if(attirePhoto is null){MessageBox.Show(this,"Choose an attire pattern image first.");return;}
  if(TopSeamCheck.IsChecked!=true && BottomSeamCheck.IsChecked!=true){MessageBox.Show(this,"Select top, bottoms, or both.");return;}
  var folderDialog=new Microsoft.Win32.SaveFileDialog{Title="Choose output PNG name and folder",Filter="PNG image|*.png",FileName="CAWForge-Attire.png"};
  if(folderDialog.ShowDialog(this)!=true)return;
  var path=attirePhoto;var output=folderDialog.FileName;var mode=SeamModeBox.SelectedIndex;var blend=(int)SeamBlendSlider.Value;
  var top=TopSeamCheck.IsChecked==true;var bottom=BottomSeamCheck.IsChecked==true;
  GenerateSeamsButton.IsEnabled=false;StatusText.Text="Generating seamless pattern on background worker…";
  try{
   var png=await Task.Run(()=>MakeSeamPng(path,mode,blend));
   var directory=Path.GetDirectoryName(output)!;var basename=Path.GetFileNameWithoutExtension(output);
   if(top)await File.WriteAllBytesAsync(Path.Combine(directory,basename+"-Top.png"),png);
   if(bottom)await File.WriteAllBytesAsync(Path.Combine(directory,basename+"-Bottoms.png"),png);
   using var ms=new MemoryStream(png);var preview=new BitmapImage();preview.BeginInit();preview.StreamSource=ms;preview.CacheOption=BitmapCacheOption.OnLoad;preview.EndInit();preview.Freeze();
   AttirePreview.Source=preview;StatusText.Text="Seamless pattern PNG(s) saved. Verify placement in WWE 2K26.";
  }catch(Exception ex){StatusText.Text="Seam generation failed";MessageBox.Show(this,ex.Message,"Attire generator",MessageBoxButton.OK,MessageBoxImage.Error);}
  finally{GenerateSeamsButton.IsEnabled=true;}
 }
 private static byte[] MakeSeamPng(string path,int mode,int blend){
  var info=new FileInfo(path);if(info.Length>75L*1024*1024)throw new InvalidDataException("Image exceeds 75 MB.");
  using var stream=File.OpenRead(path);
  var decoder=BitmapDecoder.Create(stream,BitmapCreateOptions.PreservePixelFormat,BitmapCacheOption.OnLoad);
  var source=decoder.Frames[0];
  if((long)source.PixelWidth*source.PixelHeight>35000000)throw new InvalidDataException("Image exceeds 35 megapixels.");
  var scale=Math.Min(1.0,1024.0/Math.Max(source.PixelWidth,source.PixelHeight));
  BitmapSource bitmap=source;
  if(scale<1){var scaled=new TransformedBitmap(source,new System.Windows.Media.ScaleTransform(scale,scale));scaled.Freeze();bitmap=scaled;}
  var converted=new System.Windows.Media.Imaging.FormatConvertedBitmap(bitmap,System.Windows.Media.PixelFormats.Bgra32,null,0);converted.Freeze();
  var w=converted.PixelWidth;var h=converted.PixelHeight;var stride=checked(w*4);
  var pixels=new byte[checked(stride*h)];converted.CopyPixels(pixels,stride,0);
  if(mode==1){
   // A horizontally mirrored extension has matching outer edges.
   var wide=checked(w*2);var result=new byte[checked(wide*h*4)];
   for(int y=0;y<h;y++)for(int x=0;x<wide;x++){
    int sx=x<w?x:wide-1-x;Buffer.BlockCopy(pixels,y*stride+sx*4,result,(y*wide+x)*4,4);
   }
   pixels=result;w=wide;stride=checked(w*4);
  }else{
   var band=Math.Min(Math.Max(blend,mode==2?8:1),Math.Max(1,w/4));
   // Blend opposing edges toward a shared seam color; keep center unchanged.
   for(int y=0;y<h;y++)for(int x=0;x<band;x++){
    int a=y*stride+x*4,b=y*stride+(w-1-x)*4;
    double t=(double)(band-x)/band;
    for(int c=0;c<4;c++){
     var left=pixels[a+c];var right=pixels[b+c];var middle=(left+right)/2.0;
     pixels[a+c]=(byte)Math.Clamp((int)Math.Round(left*(1-t)+middle*t),0,255);
     pixels[b+c]=(byte)Math.Clamp((int)Math.Round(right*(1-t)+middle*t),0,255);
    }
   }
  }
  var image=BitmapSource.Create(w,h,96,96,System.Windows.Media.PixelFormats.Bgra32,null,pixels,stride);image.Freeze();
  var encoder=new PngBitmapEncoder();encoder.Frames.Add(BitmapFrame.Create(image));
  using var output=new MemoryStream();encoder.Save(output);return output.ToArray();
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