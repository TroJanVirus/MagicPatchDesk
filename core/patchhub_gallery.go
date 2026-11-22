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
    Game string `json:"game,omitempty"`
    Region string `json:"region,omitempty"`
    Version string `json:"version,omitempty"`
    Patch string `json:"patch,omitempty"`
    TitleID string `json:"title_id,omitempty"`
    Kind string `json:"kind"`
    Set int `json:"set,omitempty"`
    URL string `json:"url"`
}

// PatchHub may emit source images as img src/data-src, clickable anchors or
// data-* attributes. Only explicitly named before/after JPEG/PNG/GIF images
// under the same site's /All-Patches/titles tree are accepted.
var hubGalleryAttr=regexp.MustCompile(`(?is)\b(?:src|data-src|href|data-full|data-image|data-before|data-after)\s*=\s*["']([^"'<>]{1,1000})["']`)
var hubGalleryBare=regexp.MustCompile(`(?is)["']((?:https://(?:www\.)?jayxbox\.com)?/All-Patches/titles/[^"'<>\s]{1,1000}\.(?:jpg|jpeg|png|gif))["']`)
// Image files may begin with an ordinal ("1. Before.jpg", "2. After.png"),
// have a trailing index ("before_1.png"), or be named "After.jpg". Neither
// filename order nor the case of its extension is fixed by PatchHub.
var validScreenshotFile=regexp.MustCompile(`(?i)\.(?:jpg|jpeg|png|gif)$`)
var mediaPhase=regexp.MustCompile(`(?i)^(?:(?:[0-9]{1,3}|screenshot|shot|image|comparison)[._\s-]*)*(before|after)(?:[._\s-]*[0-9]{1,3})?package core

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
    Game string `json:"game,omitempty"`
    Region string `json:"region,omitempty"`
    Version string `json:"version,omitempty"`
    Patch string `json:"patch,omitempty"`
    TitleID string `json:"title_id,omitempty"`
    Kind string `json:"kind"`
    Set int `json:"set,omitempty"`
    URL string `json:"url"`
}

// PatchHub may emit source images as img src/data-src, clickable anchors or
// data-* attributes. Only explicitly named before/after JPEG/PNG/GIF images
// under the same site's /All-Patches/titles tree are accepted.
var hubGalleryAttr=regexp.MustCompile(`(?is)\b(?:src|data-src|href|data-full|data-image|data-before|data-after)\s*=\s*["']([^"'<>]{1,1000})["']`)
var hubGalleryBare=regexp.MustCompile(`(?is)["']((?:https://(?:www\.)?jayxbox\.com)?/All-Patches/titles/[^"'<>\s]{1,1000}\.(?:jpg|jpeg|png|gif))["']`)
// Image files may begin with an ordinal ("1. Before.jpg", "2. After.png"),
// have a trailing index ("before_1.png"), or be named "After.jpg". Neither
// filename order nor the case of its extension is fixed by PatchHub.
var validScreenshotFile=regexp.MustCompile(`(?i)\.(?:jpg|jpeg|png|gif)$`)
)
var leadingMediaNumber=regexp.MustCompile(`^\s*([0-9]{1,3})[._\s-]+`)
var trailingMediaNumber=regexp.MustCompile(`(?i)(?:^|[._\s-])(?:before|after)[._\s-]*([0-9]{1,3})(?:$|[._\s-])`)

func galleryPhase(filename string)(string,int){
    // Never match 'Before' in the game's name: only the image basename
    // determines comparison phase, not ancestors in the PatchHub folder.
    filename=filepath.Base(filepath.ToSlash(filename))
    if !validScreenshotFile.MatchString(filename){return "",0}
    base:=strings.TrimSuffix(filename,filepath.Ext(filename))
    match:=mediaPhase.FindStringSubmatch(base)
    if len(match)<2{return "",0}
    phase:=strings.ToLower(match[1])
    index:=1
    if m:=leadingMediaNumber.FindStringSubmatch(base);len(m)>1{
        if n,e:=strconv.Atoi(m[1]);e==nil&&n>0{index=n}
    }else if m:=trailingMediaNumber.FindStringSubmatch(base);len(m)>1{
        if n,e:=strconv.Atoi(m[1]);e==nil&&n>0{index=n}
    }
    return phase,index
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
    if pos<0||pos<1||!strings.EqualFold(segments[pos-1],"All-Patches")||len(segments)!=pos+5{return HubScreenshot{},false}
    for _,component:=range segments {
        if component==""||component=="."||component==".." {return HubScreenshot{},false}
    }
    phase,n:=galleryPhase(segments[len(segments)-1])
    if phase=="" {
        // The site also contains patch-specific screenshots with arbitrary
        // JPG/PNG names. Keep them without falsely labelling Before/After.
        phase="screenshot"
        n=0
    }
    shot:=HubScreenshot{Game:segments[pos+1],Kind:phase,Set:n,URL:u.String()}
    version:=strings.Split(segments[pos+2],"_")
    if len(version)==3&&len(version[2])==8{
        shot.Region=strings.ToUpper(version[0])
        shot.Version=version[1]
        shot.TitleID=strings.ToUpper(version[2])
    }
    if pos+4<len(segments){shot.Patch=segments[pos+3]}
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
        kind,set:=galleryPhase(filename)
        if kind=="" {kind="screenshot";set=0}
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
        if (a.Kind=="screenshot")!=(b.Kind=="screenshot"){return a.Kind!="screenshot"}
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
            if shot.Region!=""&&p.Region!=""&&!patchRegionEquivalent(p.Region,shot.Region){continue}
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
