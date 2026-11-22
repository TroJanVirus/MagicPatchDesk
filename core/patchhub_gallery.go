package core

import (
    "html"
    "net/url"
    "path"
    "path/filepath"
    "regexp"
    "sort"
    "strconv"
    "strings"
    "os"
)

// PatchGalleryImage represents one BEFORE or AFTER image from PatchHub or a
// locally imported PatchHub titles tree. No executable patch data is involved.
type PatchGalleryImage struct {
    Kind string `json:"kind"` // before / after
    Set int `json:"set,omitempty"`
    URL string `json:"url,omitempty"`
    Local string `json:"local,omitempty"`
}
type HubScreenshot struct {
    Patch string `json:"patch,omitempty"`
    TitleID string `json:"title_id,omitempty"`
    Kind string `json:"kind"`
    Set int `json:"set,omitempty"`
    URL string `json:"url"`
}

// PatchHub may emit source images as img src/data-src, clickable anchors or
// data-* attributes. Only explicitly named before/after JPEG/PNG/GIF images
// under the same site's /All-Patches/titles tree are accepted.
var hubGalleryAttr=regexp.MustCompile(`(?is)\b(?:src|data-src|href|data-full|data-image|data-before|data-after)\s*=\s*["']([^"'<>]{1,2048})["']`)
var hubGalleryBare=regexp.MustCompile(`(?is)["']((?:https://(?:www\\.)?jayxbox\\.com)?/All-Patches/titles/[^"'<>\\s]{1,1500}\\.(?:jpg|jpeg|png|gif))["']`)
var galleryNumber=regexp.MustCompile(`(?:^|[-_.\s])(?:before|after)[-_.\s]*([0-9]{1,3})(?:[-_.\s]|$)`)
var validScreenshotFile=regexp.MustCompile(`(?i)\.(?:jpg|jpeg|png|gif)$`)

func galleryPhase(p string)(string,int){
    parts:=strings.Split(strings.ToLower(filepath.ToSlash(p)),"/")
    for i:=len(parts)-1;i>=0;i--{
        part:=parts[i]
        // A word boundary avoids treating a game called "Nightmare Before
        // Christmas" as evidence of an image comparison.
        for _,phase:=range []string{"before","after"}{
            if part==phase||strings.HasPrefix(part,phase+".")||
                strings.HasPrefix(part,phase+"_")||strings.HasPrefix(part,phase+"-")||
                strings.HasPrefix(part,phase+" ") ||
                (len(part)>len(phase) && strings.HasPrefix(part,phase) && part[len(phase)]>='0'&&part[len(phase)]<='9'){
                n:=1
                if match:=galleryNumber.FindStringSubmatch(part);len(match)>1{
                    if parsed,e:=strconv.Atoi(match[1]);e==nil&&parsed>0{n=parsed}
                }
                return phase,n
            }
        }
    }
    return "",0
}
func patchhubGalleryURL(raw string)(HubScreenshot,bool) {
    raw=html.UnescapeString(strings.ReplaceAll(strings.TrimSpace(raw),`\/`,"/"))
    parsed,e:=url.Parse(raw)
    if e!=nil{return HubScreenshot{},false}
    base,_:=url.Parse("https://www.jayxbox.com")
    u:=base.ResolveReference(parsed)
    if !trustedPatchHubURL(u)||u.RawQuery!=""||u.Fragment!=""{return HubScreenshot{},false}
    decoded,e:=url.PathUnescape(u.EscapedPath())
    if e!=nil||len(decoded)>1300||!validScreenshotFile.MatchString(decoded){return HubScreenshot{},false}
    cleaned:=path.Clean(decoded)
    if strings.Contains(decoded,"\\")||strings.Contains(decoded,"\x00")||
       strings.HasPrefix(cleaned,"//") {return HubScreenshot{},false}
    segments:=strings.Split(strings.Trim(cleaned,"/"),"/")
    pos:=-1
    for i,s:=range segments{if strings.EqualFold(s,"titles"){pos=i;break}}
    if pos<0||pos<1||!strings.EqualFold(segments[pos-1],"All-Patches")||pos+3>=len(segments){return HubScreenshot{},false}
    phase,n:=galleryPhase(strings.Join(segments[pos+3:],"/"))
    if phase==""{return HubScreenshot{},false}
    shot:=HubScreenshot{Kind:phase,Set:n,URL:u.String()}
    version:=strings.Split(segments[pos+2],"_")
    if len(version)>=3&&len(version[2])==8{shot.TitleID=strings.ToUpper(version[2])}
    if pos+4<len(segments) {if stage,_:=galleryPhase(segments[pos+3]);stage==""{shot.Patch=segments[pos+3]}}
    return shot,true
}
func parseHubGameScreenshots(fragment string)[]HubScreenshot{
    shots:=[]HubScreenshot{}
    seen:=map[string]bool{}
    candidates:=[]string{}
    for _,match:=range hubGalleryAttr.FindAllStringSubmatch(fragment,256){if len(match)>1{candidates=append(candidates,match[1])}}
    // Some PatchHub pages construct image galleries in a small inline JSON
    // block instead of IMG elements. Still require trusted, known paths.
    for _,match:=range hubGalleryBare.FindAllStringSubmatch(fragment,128){if len(match)>1{candidates=append(candidates,match[1])}}
    for _,raw:=range candidates{
        image,ok:=patchhubGalleryURL(raw)
        if !ok||seen[image.URL]{continue}
        seen[image.URL]=true
        shots=append(shots,image)
        if len(shots)>=64{break}
    }
    return shots
}

// LocalPatchScreenshots only searches a named PatchHub patch directory, not a
// shared author folder or arbitrary same-title screenshot next to a JMP.
func LocalPatchScreenshots(jmp string)[]PatchGalleryImage {
    norm:=filepath.ToSlash(jmp)
    if !strings.Contains(strings.ToLower(norm),"/titles/"){return nil}
    dir:=filepath.Dir(jmp)
    results:=[]PatchGalleryImage{}
    visited:=0
    _=filepath.WalkDir(dir,func(filename string,e os.DirEntry,err error)error{
        if err!=nil{return nil}
        visited++
        if visited>160{return filepath.SkipAll}
        if e.Type()&os.ModeSymlink!=0{return nil}
        if e.IsDir(){
            if filename!=dir&&strings.Count(strings.TrimPrefix(filepath.ToSlash(filename),filepath.ToSlash(dir)),"/")>2{return filepath.SkipDir}
            return nil
        }
        if !validScreenshotFile.MatchString(filename){return nil}
        kind,set:=galleryPhase(strings.TrimPrefix(filepath.ToSlash(filename),filepath.ToSlash(dir)+"/"))
        if kind==""{return nil}
        results=append(results,PatchGalleryImage{Kind:kind,Set:set,Local:filename})
        if len(results)>=64{return filepath.SkipAll}
        return nil
    })
    SortGallery(results)
    return results
}
func SortGallery(images []PatchGalleryImage){
    sort.SliceStable(images,func(i,j int)bool{
        a,b:=images[i],images[j]
        if a.Set!=b.Set{return a.Set<b.Set}
        if a.Kind!=b.Kind{return a.Kind=="before"}
        return a.Local+a.URL<b.Local+b.URL
    })
}

// AttachHubGallery attaches remote screenshots only if exactly one local
// patch matches the screenshot's game/version/patch name. No fuzzy guesses:
// mismatched versions or multiple plausible JMPs leave the image unattached.
func AttachHubGallery(patches []CatalogPatch,shots []HubScreenshot) {
    for _,shot:=range shots{
        matches:=[]int{}
        shotKey:=CoverMatchKey(shot.Patch)
        for i,p:=range patches{
            if shot.TitleID!="" && !strings.EqualFold(shot.TitleID,p.TitleID){continue}
            if shotKey!=""{
                patchKey:=CoverMatchKey(p.Name)
                // PatchHub folder labels may be shorter than the filename.
                // Prefix/substring matches are only accepted when UNIQUE
                // within this game and the same original Title ID.
                if shotKey!=patchKey && (len(shotKey)<5||!strings.Contains(patchKey,shotKey)){continue}
            }
            matches=append(matches,i)
        }
        if len(matches)!=1{continue}
        p:=&patches[matches[0]]
        duplicate:=false
        for _,existing:=range p.Gallery{if existing.URL==shot.URL {duplicate=true;break}}
        if !duplicate{p.Gallery=append(p.Gallery,PatchGalleryImage{Kind:shot.Kind,Set:shot.Set,URL:shot.URL})}
    }
    for i:=range patches{SortGallery(patches[i].Gallery)}
}
