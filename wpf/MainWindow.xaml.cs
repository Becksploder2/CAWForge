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
 public MainWindow() { InitializeComponent(); for(int i=1;i<=21;i++) TemplateBox.Items.Add("Template "+i); TemplateBox.SelectedIndex=9; foreach(var group in new (string Category,string[] Names)[]{("Head",new[]{"Head Width","Head Height","Head Depth"}),("Ears Top",new[]{"Offset","Height","Alignment"}),("Ears",new[]{"Ears Width","Ears Height","Vertical Stretch","Placement Low/High","Placement Forward/Back"}),("Earlobes",new[]{"Lobe Detachment","Lobe Length","Lobe Alignment"}),("Forehead",new[]{"Forehead Upper Size","Forehead Size"}),("Eyebrows",new[]{"Eyebrow Length","Eyebrow Depth","Eyebrow Height","Eyebrow Angle","Eyebrow Spacing"}),("Eyes",new[]{"Eye Width","Eye Height","Eye Depth","Eye Angle","Eye Spacing","Eye Position"}),("Nose",new[]{"Nose Width","Nose Height","Nose Length","Nose Bridge Width","Nose Tip Width","Nostril Width","Nostril Height"}),("Cheeks",new[]{"Cheek Width","Cheek Height","Cheek Depth"}),("Mouth",new[]{"Mouth Width","Mouth Height","Upper Lip Thickness","Lower Lip Thickness","Mouth Depth"}),("Jaw",new[]{"Jaw Width","Jaw Height","Jaw Depth","Jaw Angle"}),("Chin",new[]{"Chin Width","Chin Height","Chin Depth","Chin Shape"})}) foreach(var name in group.Names) {var panel=new StackPanel{Margin=new Thickness(0,0,0,8)};panel.Children.Add(new TextBlock{Text=group.Category+" / "+name,Foreground=System.Windows.Media.Brushes.White});panel.Children.Add(new Slider{Minimum=0,Maximum=100,Value=50,Width=360,HorizontalAlignment=HorizontalAlignment.Left});SlidersList.Items.Add(panel);morphSliders[group.Category+" / "+name]=(Slider)panel.Children[1];} }
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
   foreach(var entry in morphSliders)lines.Add(entry.Key+": "+Math.Round(entry.Value.Value));
   File.WriteAllLines(dlg.FileName,lines);StatusText.Text="Morph sheet saved: "+dlg.FileName;
  }catch(Exception ex){MessageBox.Show(this,ex.Message,"Export error",MessageBoxButton.OK,MessageBoxImage.Error);}
 }
 private async void GenerateTexture_Click(object sender,RoutedEventArgs e) {
  if(selectedPhoto is null){MessageBox.Show(this,"Import a reference photo in Morph Settings first.");return;}
  var dlg=new SaveFileDialog{Title="Save reference crop",Filter="PNG image|*.png",FileName="CAWForge-Face-Reference-820.png"};
  if(dlg.ShowDialog(this)!=true)return;
  StatusText.Text="Generating 820 × 820 reference crop…";
  try{
   var source=selectedPhoto;var output=dlg.FileName;
   await Task.Run(()=>{
    using var stream=File.OpenRead(source);
    var decoder=BitmapDecoder.Create(stream,BitmapCreateOptions.PreservePixelFormat,BitmapCacheOption.OnLoad);
    var frame=decoder.Frames[0];
    var side=Math.Min(frame.PixelWidth,frame.PixelHeight);
    var crop=new CroppedBitmap(frame,new System.Windows.Int32Rect((frame.PixelWidth-side)/2,(frame.PixelHeight-side)/2,side,side));
    crop.Freeze();
    var target=new TransformedBitmap(crop,new System.Windows.Media.ScaleTransform(820.0/side,820.0/side));target.Freeze();
    var encoder=new PngBitmapEncoder();encoder.Frames.Add(BitmapFrame.Create(target));
    using var dest=File.Create(output);encoder.Save(dest);
   });
   var preview=new BitmapImage();preview.BeginInit();preview.UriSource=new Uri(output);preview.CacheOption=BitmapCacheOption.OnLoad;preview.EndInit();preview.Freeze();TexturePreview.Source=preview;
   StatusText.Text="820 × 820 reference crop saved: "+output;
  }catch(Exception ex){StatusText.Text="Texture crop failed";MessageBox.Show(this,ex.Message,"Texture error",MessageBoxButton.OK,MessageBoxImage.Error);}
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