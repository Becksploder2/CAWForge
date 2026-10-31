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
 private string MorphTemplateId => (GenderBox.SelectedIndex==0?"M":"F")+(TemplateBox.SelectedIndex+1);
 private string UltraRealisticTexturePrompt => """
 # WWE 2K26 – Ultra-Realistic Face Texture Master Prompt
 Using the provided photograph of a real person, generate an ultra-high-quality, photorealistic WWE 2K26 face texture at 820 × 820 pixels.

 Facial Accuracy:
 Preserve the person's facial likeness as accurately as possible. Match their natural skin tone, complexion, undertones, and pigmentation to the reference photograph. Preserve eye shape, eyebrows, nose, lips, facial proportions, and natural makeup. Maintain sharp details, realistic pores, freckles, wrinkles, and natural skin texture.

 Facial Positioning:
 Completely remove all hair and ears. Position the face perfectly straight-on and centered. Use a neutral expression, closed mouth, no visible teeth, and both eyes facing directly forward.

 Lighting and Skin Blending:
 Use neutral, even studio lighting. Eliminate harsh shadows, directional lighting, and excessive highlights. Completely remove the shadow underneath the face, jawline, and chin. Remove dark outlines, shading bands, and visible borders. Match the underlying skin color to the natural facial skin tone as closely as possible. Blend forehead, temples, cheeks, jawline, chin, and neck seamlessly. No abrupt color, brightness, saturation, or texture transitions. No pasted-on face appearance. Preserve natural definition without artificial shadows.

 Image Quality:
 Ultra-high-resolution photorealistic appearance, maximum sharpness and fine facial detail. No artificial smoothing, excessive shine, exaggerated features, visible hairline, ears, jewelry, accessories, or distracting background. Optimize for WWE 2K26 Create-a-Superstar face-photo mapping.

 Final Priority:
 Maximum facial likeness, accurate skin tone, seamless face-to-skin blending, and complete elimination of shadows beneath the face. Face and underlying skin must match naturally and continuously.

 Make this face texture fit WWE 2K26 face template {TEMPLATE_ID} selected in CAWForge Morph Settings. Automatically align eyes, nose, mouth, jaw, and facial proportions to that exact template.
 """.Replace("{TEMPLATE_ID}",MorphTemplateId);
 private void CopyTexturePrompt_Click(object sender,RoutedEventArgs e){
  Clipboard.SetText(UltraRealisticTexturePrompt);
  MessageBox.Show(this,"Master prompt copied with Morph Settings template "+MorphTemplateId+". This is a prompt specification; the current OpenCV generator does not execute generative image prompts.","Face Texture Prompt");
 }
 private void RefreshMorphTextureLink(){
  if(MorphTextureTemplateText!=null)MorphTextureTemplateText.Text="Morph Settings face template: "+MorphTemplateId+" (automatically synchronized with master prompt)";
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
 public MainWindow() { InitializeComponent(); RefreshFusionLibrary(); for(int i=1;i<=21;i++) TemplateBox.Items.Add("Template "+i); TemplateBox.SelectedIndex=9; GenderBox.SelectionChanged+=(_,_)=>RefreshMorphTextureLink(); TemplateBox.SelectionChanged+=(_,_)=>RefreshMorphTextureLink(); RefreshMorphTextureLink(); foreach(var group in new (string Category,string[] Names)[]{("Head",new[]{"Head Width","Head Height","Head Depth"}),("Ears Top",new[]{"Offset","Height","Alignment"}),("Ears",new[]{"Ears Width","Ears Height","Vertical Stretch","Placement Low/High","Placement Forward/Back"}),("Earlobes",new[]{"Lobe Detachment","Lobe Length","Lobe Alignment"}),("Forehead Upper",new[]{"Forehead Upper Size"}),("Forehead",new[]{"Forehead Size"}),("Eyebrows",new[]{"Eyebrow Length","Eyebrow Depth","Eyebrow Corner Height Left","Eyebrow Corner Height Right","Eyebrow Outer Height Left","Eyebrow Outer Height Right"}),("Eyes",new[]{"Eye Length","Eye Socket Depth","Eye Corner Length","Eye Corner Height","Eye Outer Length","Eye Outer Height"}),("Nose",new[]{"Nose Width","Nose Height","Nose Depth"}),("Nose Bridge",new[]{"Nose Bridge Width","Nose Bridge Height","Nose Bridge Depth"}),("Nose Tip",new[]{"Nose Tip Width","Tip Pointed or Rounded","Nose Tip Length","Tip Height"}),("Nostrils",new[]{"Left Nostril Width","Left Nostril Height","Left Nostril Depth","Right Nostril Width","Right Nostril Height","Right Nostril Depth"}),("Philtrum",new[]{"Philtrum Width","Philtrum Depth"}),("Mouth",new[]{"Mouth Horizontal Length","Mouth Fullness","Mouth Depth","Left Mouth Corner Length","Right Mouth Corner Length","Left Mouth Corner Height","Right Mouth Corner Height"}),("Cheekbones",new[]{"Upper Cheekbone","Middle Cheekbone","Lower Cheekbone"}),("Cheeks",new[]{"Cheek Fullness","Cheek Height"}),("Jawline",new[]{"Middle Jaw Fullness","Lower Jaw Height","Lower Jaw Depth","Rear Jaw Fullness","Lower Jaw Fullness","Lower Jaw Length"}),("Chin",new[]{"Chin Width","Chin Height","Chin Forward/Back","Chin Vertical Placement"}),("Parietal Region",new[]{"Parietal Region Size"})}) {morphCategoryPanels[group.Category]=new List<StackPanel>(); foreach(var name in group.Names) {var panel=new StackPanel{Margin=new Thickness(0,0,0,8)};panel.Children.Add(new TextBlock{Text=group.Category+" / "+name,Foreground=System.Windows.Media.Brushes.White});panel.Children.Add(new Slider{Minimum=0,Maximum=100,Value=50,Width=360,HorizontalAlignment=HorizontalAlignment.Left});morphCategoryPanels[group.Category].Add(panel);morphSliders[group.Category+" / "+name]=(Slider)panel.Children[1];} var categoryName=group.Category;var categoryButton=new Button{Content=categoryName,Margin=new Thickness(0,0,8,8),Padding=new Thickness(12,7,12,7),MinWidth=110};categoryButton.Click+=(_,_)=>ShowMorphCategory(categoryName);MorphCategoryButtons.Children.Add(categoryButton);} ShowMorphCategory("Head"); }
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
   MessageBox.Show(this,"Face texture saved successfully: "+output,"Export complete",MessageBoxButton.OK,MessageBoxImage.Information);
  }catch(Exception ex){
   StatusText.Text="Face texture save failed: "+ex.Message;
   MessageBox.Show(this,"Could not save the face texture. "+ex.Message+" Try a local folder such as Pictures.","Face texture export failed",MessageBoxButton.OK,MessageBoxImage.Error);
  }
 }
    MessageBox.Show(this,"Four connected panel PNGs saved to: "+folder+". Front, Back, Left and Right. Check garment UV orientation in-game.","Attire seam export",MessageBoxButton.OK,MessageBoxImage.Information);
}