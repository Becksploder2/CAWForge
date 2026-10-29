using Microsoft.Win32;
using System.IO;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media.Imaging;
namespace CAWForge;
public partial class MainWindow : Window {
 private string? selectedPhoto;
 private string? attirePhoto;
 private readonly Dictionary<string,Slider> morphSliders=new();
 public MainWindow() { InitializeComponent(); for(int i=1;i<=20;i++) TemplateBox.Items.Add("Template "+i); TemplateBox.SelectedIndex=9; foreach(var name in new[]{"Head Width","Head Height","Head Depth","Eye Width","Eye Height","Nose Width","Nose Height","Mouth Width","Jaw Width","Chin Height"}) {var panel=new StackPanel{Margin=new Thickness(0,0,0,8)};panel.Children.Add(new TextBlock{Text=name,Foreground=System.Windows.Media.Brushes.White});panel.Children.Add(new Slider{Minimum=0,Maximum=100,Value=50,Width=360,HorizontalAlignment=HorizontalAlignment.Left});SlidersList.Items.Add(panel);morphSliders[name]=(Slider)panel.Children[1];} }
 private async void ImportPhoto_Click(object sender,RoutedEventArgs e) {var dialog=new OpenFileDialog{Title="Choose reference photo",Filter="Images|*.png;*.jpg;*.jpeg;*.bmp;*.webp",CheckFileExists=true};if(dialog.ShowDialog(this)!=true)return;var path=dialog.FileName;StatusText.Text="Validating image…";try {var bitmap=await Task.Run(()=>{var info=new FileInfo(path);if(info.Length>75L*1024*1024)throw new InvalidDataException("Image exceeds 75 MB.");using var stream=File.OpenRead(path);var decoder=BitmapDecoder.Create(stream,BitmapCreateOptions.DelayCreation,BitmapCacheOption.None);var frame=decoder.Frames[0];if((long)frame.PixelWidth*frame.PixelHeight>35000000)throw new InvalidDataException("Image exceeds 35 megapixels.");return (frame.PixelWidth,frame.PixelHeight);});var preview=new BitmapImage();preview.BeginInit();preview.UriSource=new Uri(path);preview.DecodePixelWidth=480;preview.CacheOption=BitmapCacheOption.OnLoad;preview.EndInit();preview.Freeze();PhotoPreview.Source=preview;selectedPhoto=path;PhotoPathText.Text=Path.GetFileName(path);StatusText.Text=$"Photo ready ({bitmap.Item1} × {bitmap.Item2}).";}catch(Exception ex){StatusText.Text="Import failed: "+ex.Message;MessageBox.Show(this,ex.Message,"Import error",MessageBoxButton.OK,MessageBoxImage.Warning);} }

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
}