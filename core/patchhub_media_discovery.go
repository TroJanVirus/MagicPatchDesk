package core

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "html"
    "io"
    "net/http"
    "net/url"
    "regexp"
    "sort"
    "strconv"
    "strings"
    "time"
)

// The original PatchHub page currently exposes game covers and game-version
// records, not a complete filename list for patch comparison images. This
// optional index lets the PatchHub server publish exact arbitrary JPG/PNG
// names without costly filename guessing.
const PatchHubMediaURL="https://www.jayxbox.com/All-Patches/patchhub-media-v1.json"
const PatchHubMediaSchema="patchhub-media-v1"
type PatchHubMediaManifest struct{
    Schema string `json:"schema"`
    Images []string `json:"images"`
}
func ParsePatchHubMediaManifest(body []byte)([]HubScreenshot,error){
    if len(body)>8<<20{return nil,errors.New("PatchHub media index too large")}
    var index PatchHubMediaManifest
    if e:=json.Unmarshal(body,&index);e!=nil{return nil,e}
    if index.Schema!=PatchHubMediaSchema{return nil,errors.New("unrecognized PatchHub media index schema")}
    if len(index.Images)>30000{return nil,errors.New("too many PatchHub images")}
    out:=make([]HubScreenshot,0,len(index.Images))
    seen:=map[string]bool{}
    for _,raw:=range index.Images{
        shot,ok:=patchhubGalleryURL(raw)
        if !ok||seen[shot.URL]{continue}
        seen[shot.URL]=true
        out=append(out,shot)
    }
    return out,nil
}

// AppendPatchHubMedia merges the website's actual patch-folder image inventory
// into the cached game index. Neither an exact HTML revision record nor the
// image's region is needed: the directory already establishes image ownership.
// Choose a single game by exact title first, then safe alias/known Title ID.
func AppendPatchHubMedia(hub *HubIndex,urls []HubScreenshot) {
    if hub==nil{return}
    byTitle:=map[string][]int{}
    for i:=range hub.Games {
        k:=CoverMatchKey(hub.Games[i].Title)
        if k!="" {byTitle[k]=append(byTitle[k],i)}
    }
    for _,shot:=range urls {
        matches:=byTitle[CoverMatchKey(shot.Game)]
        if len(matches)==0 {
            for i:=range hub.Games {
                g:=Game{Title:hub.Games[i].Title,Versions:hub.Games[i].Versions}
                if MediaGameMatches(g,shot){matches=append(matches,i)}
            }
        }
        if len(matches)!=1{continue}
        g:=&hub.Games[matches[0]]
        exists:=false
        for _,old:=range g.Screenshots {
            if old.URL==shot.URL{exists=true;break}
        }
        if !exists {g.Screenshots=append(g.Screenshots,shot)}
    }
    for i:=range hub.Games {
        sort.SliceStable(hub.Games[i].Screenshots,func(a,b int)bool{
            x,y:=hub.Games[i].Screenshots[a],hub.Games[i].Screenshots[b]
            if x.Patch!=y.Patch{return x.Patch<y.Patch}
            if x.Set!=y.Set{return x.Set<y.Set}
            if x.Kind!=y.Kind{return x.Kind=="before"}
            return x.URL<y.URL
        })
    }
}
func normalizeHubVersion(value string)string{
    n,e:=strconv.ParseUint(strings.TrimSpace(value),10,32)
    if e!=nil{return ""}
    return fmt.Sprintf("%08d",n)
}
func ReadPatchHubMediaManifest(ctx context.Context,client *http.Client)([]HubScreenshot,error){
    if client==nil{client=&http.Client{Timeout:10*time.Second}}
    req,e:=http.NewRequestWithContext(ctx,http.MethodGet,PatchHubMediaURL,nil)
    if e!=nil{return nil,e}
    req.Header.Set("User-Agent","MagicPatchDesk/1.0 (PatchHub comparison image index)")
    response,e:=client.Do(req)
    if e!=nil{return nil,e}
    defer response.Body.Close()
    if response.StatusCode!=http.StatusOK{
        return nil,fmt.Errorf("PatchHub media index is not available: %s",response.Status)
    }
    if !trustedPatchHubURL(response.Request.URL){return nil,errors.New("untrusted media index redirect")}
    body,e:=io.ReadAll(io.LimitReader(response.Body,(8<<20)+1))
    if e!=nil{return nil,e}
    return ParsePatchHubMediaManifest(body)
}
func patchRegionEquivalent(source,version string)bool{
    a,b:=strings.ToUpper(strings.TrimSpace(source)),strings.ToUpper(strings.TrimSpace(version))
    if a==b{return true}
    if a=="NTSC"&&(b=="USA"||b=="USA-JPN"){return true}
    if a=="NTSC-U"&&b=="USA"{return true}
    if a=="NTSC-J"&&b=="JPN"{return true}
    return false
}

// SafeCandidates only derives URLs when game, TitleID, region and numeric
// version jointly locate ONE exact PatchHub directory. A patch XBE Title ID
// alone may correspond to many regions/revisions and cannot be used to guess.
func PatchHubPatchDirectory(g Game,p CatalogPatch)(string,error){
    if p.TitleID==""||CoverMatchKey(p.Name)==""||CoverMatchKey(g.Title)==""{
        return "",errors.New("cannot derive a patch-specific media directory")
    }
    choices:=map[string]bool{}
    for _,v:=range g.Versions{
        if !strings.EqualFold(v.TitleID,p.TitleID)||!patchRegionEquivalent(p.Region,v.Region){continue}
        number:=normalizeHubVersion(v.Version)
        if number==""{continue}
        id:=strings.ToUpper(v.Region)+"_"+number+"_"+strings.ToUpper(v.TitleID)
        choices[id]=true
    }
    if len(choices)!=1{return "",errors.New("image directory is ambiguous or unavailable for this game revision")}
    var versionDir string
    for k:=range choices{versionDir=k}
    segments:=[]string{"All-Patches","titles",g.Title,versionDir,p.Name}
    for i:=range segments{
        if segments[i]==""||strings.ContainsAny(segments[i],"/\\\x00")||segments[i]=="."||segments[i]==".."{
            return "",errors.New("invalid PatchHub media path")
        }
        segments[i]=url.PathEscape(segments[i])
    }
    return "https://www.jayxbox.com/"+strings.Join(segments,"/")+"/",nil
}

var listingHref=regexp.MustCompile(`(?is)\bhref\s*=\s*["']([^"'<>]{1,1000})["']`)

// Detect exact arbitrary filenames from directory listings when the site
// enables them. Never trust a result which escapes this one patch directory.
func MediaLinksFromDirectoryHTML(folder string,document []byte)[]HubScreenshot{
    if len(document)>512<<10{return nil}
    parent,e:=url.Parse(folder)
    if e!=nil||!trustedPatchHubURL(parent){return nil}
    out:=[]HubScreenshot{}
    seen:=map[string]bool{}
    for _,match:=range listingHref.FindAllSubmatch(document,128){
        if len(match)<2{continue}
        u,e:=url.Parse(html.UnescapeString(string(match[1])))
        if e!=nil||u.RawQuery!=""||u.Fragment!=""{continue}
        absolute:=parent.ResolveReference(u)
        if !trustedPatchHubURL(absolute)||!strings.HasPrefix(absolute.EscapedPath(),parent.EscapedPath()){continue}
        trailing:=strings.TrimPrefix(absolute.EscapedPath(),parent.EscapedPath())
        if trailing==""||strings.Contains(trailing,"/"){continue}
        shot,ok:=patchhubGalleryURL(absolute.String())
        if ok&&!seen[shot.URL]{seen[shot.URL]=true;out=append(out,shot)}
    }
    return out
}

// Probe only a few established conventions, once when requested by a user.
// Never walk or hammer the entire server: arbitrary filenames require either
// a directory listing or the server's read-only media index.
func CommonPatchHubMediaNames()[]string{
    names:=[]string{"1. Before","2. After","Before","After","1. After","2. Before","3. Before","4. After","3. After","4. Before"}
    out:=make([]string,0,len(names)*2)
    for _,n:=range names{out=append(out,n+".jpg",n+".png")}
    return out
}
// PatchFolderMatches ties media to the exact patch-directory label. Filenames
// of JPG/PNG images are irrelevant and the intermediate website release
// directory is NOT an identity check for gallery art.
func PatchFolderMatches(patchName,folderName string) bool {
    a,b:=CoverMatchKey(patchName),CoverMatchKey(folderName)
    return a!="" && a==b
}

// Some website patch-folder titles are abbreviated (e.g. "720p Patch"
// instead of "Halo 2 720p Patch"). A partial match may only be used when it
// identifies a single patch in that game; never attach on ambiguous guesses.
func patchFolderCandidate(patchName,folderName string) bool {
    if PatchFolderMatches(patchName,folderName){return true}
    a,b:=CoverMatchKey(patchName),CoverMatchKey(folderName)
    return len(b)>=5&&strings.Contains(a,b)
}
func matchingPatchFolderCount(patches []CatalogPatch,folderName string)int{
    exact:=0
    for _,p:=range patches {if PatchFolderMatches(p.Name,folderName){exact++}}
    if exact>0{return exact}
    candidate:=0
    for _,p:=range patches {if patchFolderCandidate(p.Name,folderName){candidate++}}
    return candidate
}

// A website game title may include a parenthetical subtitle not present in a
// JMP game header. Use a conservative title alias or an already-known Title ID
// for the game, without requiring region/version equality.
func MediaGameMatches(g Game,shot HubScreenshot)bool{
    a,b:=CoverMatchKey(g.Title),CoverMatchKey(shot.Game)
    if a==""||b==""{return false}
    if a==b{return true}
    if len(a)>=6 && len(b)>=6 && (strings.HasPrefix(a,b)||strings.HasPrefix(b,a)){
        return true
    }
    if shot.TitleID!="" {
        for _,v:=range g.Versions {
            if strings.EqualFold(v.TitleID,shot.TitleID){return true}
        }
        for _,p:=range g.Patches {
            if strings.EqualFold(p.TitleID,shot.TitleID){return true}
        }
    }
    return false
}

// MatchPatchHubMedia finds ALL JPG/PNG images inside the selected website
// patch folder, including repeated folders under different region/revisions.
// The image filename is only used to offer optional Before/After labels.
func MatchPatchHubMedia(game Game,patch CatalogPatch,all []HubScreenshot)[]HubScreenshot{
    if CoverMatchKey(patch.Name)==""{return nil}
    patches:=game.Patches
    allowAbbreviations:=len(patches)>0
    if len(patches)==0{patches=[]CatalogPatch{patch}}
    result:=make([]HubScreenshot,0,8)
    seen:=map[string]bool{}
    for _,shot:=range all{
        if !MediaGameMatches(game,shot)||seen[shot.URL]{continue}
        selected:=false
        for _,i:=range galleryFolderOwners(patches,shot,allowAbbreviations){
            if sameGalleryPatch(patches[i],patch){selected=true;break}
        }
        if !selected{continue}
        seen[shot.URL]=true
        result=append(result,shot)
        if len(result)>=128{break}
    }
    sort.SliceStable(result,func(i,j int)bool{
        if result[i].Set!=result[j].Set{return result[i].Set<result[j].Set}
        if result[i].Kind!=result[j].Kind{return result[i].Kind=="before"}
        return result[i].URL<result[j].URL
    })
    return result
}

// Discovery checks the real site index FIRST: constructing a directory from a
// GitHub patch filename is only a fallback and can be wrong or ambiguous.
// Caller receives a diagnostic error rather than a silent "no images" result.
func DiscoverPatchHubMedia(ctx context.Context,client *http.Client,game Game,patch CatalogPatch)([]HubScreenshot,error){
    // Keep a bounded and trusted HTTP client for HEAD and GET probes.
    if client==nil {
        client=&http.Client{Timeout:5*time.Second,CheckRedirect:func(req *http.Request,via []*http.Request)error{
            if len(via)>3||!trustedPatchHubURL(req.URL){return errors.New("unsafe PatchHub redirect")}
            return nil
        }}
    }
    indexed,indexErr:=ReadPatchHubMediaManifest(ctx,client)
    if indexErr==nil {
        matched:=MatchPatchHubMedia(game,patch,indexed)
        if len(matched)>0{return matched,nil}
        // A complete, authoritative folder inventory obsoletes URL guessing.
        return nil,errors.New("media index contains no images for this exact game/patch folder")
    }
    folder,folderErr:=PatchHubPatchDirectory(game,patch)
    if folderErr!=nil {
        if indexErr!=nil {return nil,fmt.Errorf("media index unavailable (%v); filename-derived directory unavailable (%v)",indexErr,folderErr)}
        return nil,fmt.Errorf("media index has no matching patch images; directory unavailable: %w",folderErr)
    }
    results:=make([]HubScreenshot,0,8)
    seen:=map[string]bool{}
    add:=func(s HubScreenshot) {
        if !seen[s.URL]&&len(results)<32 {seen[s.URL]=true;results=append(results,s)}
    }
    listing,err:=http.NewRequestWithContext(ctx,http.MethodGet,folder,nil)
    if err==nil {
        response,e:=client.Do(listing)
        if e==nil {
            if response.StatusCode==200&&strings.Contains(strings.ToLower(response.Header.Get("Content-Type")),"text/html")&&response.ContentLength<512<<10 {
                content,_:=io.ReadAll(io.LimitReader(response.Body,(512<<10)+1))
                for _,s:=range MediaLinksFromDirectoryHTML(folder,content){add(s)}
            }
            response.Body.Close()
        }
    }
    if len(results)==0 {
        for _,name:=range CommonPatchHubMediaNames() {
            if ctx.Err()!=nil {break}
            raw:=folder+url.PathEscape(name)
            req,e:=http.NewRequestWithContext(ctx,http.MethodHead,raw,nil)
            if e!=nil {continue}
            response,e:=client.Do(req)
            if e!=nil {continue}
            ok:=response.StatusCode==200&&response.ContentLength<=5<<20&&trustedPatchHubURL(response.Request.URL)
            response.Body.Close()
            if ok {if shot,valid:=patchhubGalleryURL(raw);valid{add(shot)}}
        }
    }
    // Real servers can reject HEAD requests. Probe only a handful of common
    // names with GET. Arbitrary names require a site index or directory list.
    if len(results)==0 {
        for _,name:=range []string{"1. Before.jpg","1. Before.png","2. After.jpg","2. After.png"} {
            if ctx.Err()!=nil{break}
            raw:=folder+url.PathEscape(name)
            req,e:=http.NewRequestWithContext(ctx,http.MethodGet,raw,nil)
            if e!=nil {continue}
            req.Header.Set("Range","bytes=0-15")
            response,e:=client.Do(req)
            if e!=nil {continue}
            trusted:=trustedPatchHubURL(response.Request.URL)
            statusOK:=response.StatusCode==http.StatusOK||response.StatusCode==http.StatusPartialContent
            header:=make([]byte,8)
            n,_:=io.ReadFull(io.LimitReader(response.Body,8),header)
            response.Body.Close()
            jpg:=n>=3&&header[0]==0xFF&&header[1]==0xD8&&header[2]==0xFF
            png:=n>=8&&string(header)==string([]byte{0x89,'P','N','G',0x0D,0x0A,0x1A,0x0A})
            if trusted&&statusOK&&(jpg||png) {
                if shot,valid:=patchhubGalleryURL(raw);valid {add(shot)}
            }
        }
    }
    if len(results)==0 {
        if indexErr!=nil {
            return nil,fmt.Errorf("no images at exact directory; PatchHub index unavailable (%v). Publish patchhub-media-v1.json to discover arbitrary filenames",indexErr)
        }
        return nil,errors.New("no images matched this exact patch; media index and bounded directory probes returned nothing")
    }
    sort.SliceStable(results,func(i,j int)bool{
        if results[i].Set!=results[j].Set{return results[i].Set<results[j].Set}
        if results[i].Kind!=results[j].Kind{return results[i].Kind=="before"}
        return results[i].URL<results[j].URL
    })
    return results,nil
}
