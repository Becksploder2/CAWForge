using Microsoft.Win32;
using System.IO;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media.Imaging;
namespace CAWForge;
public partial class MainWindow : Window {
 private string? selectedPhoto;
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
}