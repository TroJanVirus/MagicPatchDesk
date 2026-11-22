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

// This is the user's provided actual public filename, not a guessed URL.
// Fetching the image remains best-effort and is verified as a real image.
const ProvidedPhantomCrashImage="https://www.jayxbox.com/All-Patches/titles/Phantom%20Crash/USA_00000005_504C0001/Phantom%20Crash%20%7BWidescreen%7D%20%28NTSC%29%20%5BJay%5D/1.%20Before.jpg"
const ProvidedAMFBowlingImage="https://www.jayxbox.com/All-Patches/titles/AMF%20Xtreme%20Bowling/USA_00000001_42530014/AMF%20Xtreme%20Bowling%20%7BWidescreen%7D%20%28NTSC%29%20%5BEsppiral%5D/1.%20Before.jpg"

// Exact user-provided paths are seeds, not inferred server files. Their
// original game, patch name, revision, Title ID and region must all match.
func knownPatchHubImages(game Game,patch CatalogPatch) []HubScreenshot {
    urls:=[]string{ProvidedPhantomCrashImage,ProvidedAMFBowlingImage}
    result:=[]HubScreenshot{}
    for _,raw:=range urls {
        shot,ok:=patchhubGalleryURL(raw)
        if !ok||CoverMatchKey(shot.Game)!=CoverMatchKey(game.Title)||
            !strings.EqualFold(shot.Patch,patch.Name)||
            !strings.EqualFold(shot.TitleID,patch.TitleID)||
            !patchRegionEquivalent(patch.Region,shot.Region) {continue}
        result=append(result,shot)
    }
    return result
}


func AppendPatchHubMedia(hub *HubIndex,urls []HubScreenshot){
    if hub==nil{return}
    byTitle:=map[string][]int{}
    for i:=range hub.Games{
        key:=CoverMatchKey(hub.Games[i].Title)
        if key!=""{byTitle[key]=append(byTitle[key],i)}
    }
    for _,shot:=range urls{
        gameMatch:=byTitle[CoverMatchKey(shot.Game)]
        if len(gameMatch)!=1{continue}
        g:=&hub.Games[gameMatch[0]]
        matchVersion:=false
        for _,v:=range g.Versions{
            if strings.EqualFold(v.TitleID,shot.TitleID)&&strings.EqualFold(v.Region,shot.Region)&&normalizeHubVersion(v.Version)==shot.Version{
                matchVersion=true
                break
            }
        }
        if !matchVersion{continue}
        found:=false
        for _,old:=range g.Screenshots{if old.URL==shot.URL{found=true;break}}
        if !found {g.Screenshots=append(g.Screenshots,shot)}
    }
    for i:=range hub.Games{
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
func DiscoverPatchHubMedia(ctx context.Context,client *http.Client,game Game,patch CatalogPatch)([]HubScreenshot,error){
    known:=knownPatchHubImages(game,patch)
    folder,e:=PatchHubPatchDirectory(game,patch)
    if e!=nil {
        if len(known)>0{return known,nil}
        return nil,e
    }
    if client==nil{
        client=&http.Client{Timeout:5*time.Second,CheckRedirect:func(req *http.Request,via []*http.Request)error{
            if len(via)>3||!trustedPatchHubURL(req.URL){return errors.New("unsafe PatchHub redirect")}
            return nil
        }}
    }
    results:=[]HubScreenshot{}
    seen:=map[string]bool{}
    add:=func(s HubScreenshot){
        if !seen[s.URL]&&len(results)<32{seen[s.URL]=true;results=append(results,s)}
    }
    // Listing optional. A 403/404 is expected on servers with autoindex off.
    listing,err:=http.NewRequestWithContext(ctx,http.MethodGet,folder,nil)
    if err==nil{
        response,e:=client.Do(listing)
        if e==nil{
            if response.StatusCode==200&&strings.Contains(strings.ToLower(response.Header.Get("Content-Type")),"text/html")&&response.ContentLength<512<<10{
                content,_:=io.ReadAll(io.LimitReader(response.Body,(512<<10)+1))
                for _,s:=range MediaLinksFromDirectoryHTML(folder,content){add(s)}
            }
            response.Body.Close()
        }
    }
    // When there is no directory listing, make a strictly bounded set of HEAD
    // requests for known conventions. 404s are expected and not fatal.
    if len(results)==0{
        for _,name:=range CommonPatchHubMediaNames(){
            if ctx.Err()!=nil{break}
            raw:=folder+url.PathEscape(name)
            req,e:=http.NewRequestWithContext(ctx,http.MethodHead,raw,nil)
            if e!=nil{continue}
            response,e:=client.Do(req)
            if e!=nil{continue}
            ok:=response.StatusCode==200&&response.ContentLength<=5<<20&&
                trustedPatchHubURL(response.Request.URL)
            response.Body.Close()
            if ok{
                s,valid:=patchhubGalleryURL(raw)
                if valid{add(s)}
            }
        }
    }
    // Some image hosts refuse HEAD even though GET works. Only if the
    // ordinary checks found nothing, perform four bounded byte-range GETs
    // for the two most common before/after names. Never infer success from
    // status alone: require recognizable JPEG/PNG magic.
    if len(results)==0 {
        for _,name:=range []string{"1. Before.jpg","1. Before.png","2. After.jpg","2. After.png"} {
            if ctx.Err()!=nil {break}
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
                if s,valid:=patchhubGalleryURL(raw);valid {add(s)}
            }
        }
    }
    if len(results)==0 {for _,shot:=range known {add(shot)}}
    sort.SliceStable(results,func(i,j int)bool{
        if results[i].Set!=results[j].Set{return results[i].Set<results[j].Set}
        if results[i].Kind!=results[j].Kind{return results[i].Kind=="before"}
        return results[i].URL<results[j].URL
    })
    return results,nil
}
