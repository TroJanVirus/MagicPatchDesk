//go:build windows

package main

import (
    "context"
    "fmt"
    "magicpatchdesk/core"
    "sync"
    "time"
)

type patchMediaState struct {
    images []core.PatchGalleryImage
    lastAttempt time.Time
    fetching bool
}
var patchMediaMu sync.Mutex
var patchMediaCache=map[string]patchMediaState{}

func galleryForPatch(p core.CatalogPatch)[]core.PatchGalleryImage{
    base:=append([]core.PatchGalleryImage(nil),p.Gallery...)
    patchMediaMu.Lock()
    discovery:=patchMediaCache[p.File]
    patchMediaMu.Unlock()
    seen:=map[string]bool{}
    combined:=make([]core.PatchGalleryImage,0,len(base)+len(discovery.images))
    for _,item:=range append(base,discovery.images...){
        id:=item.URL
        if id==""{id=item.Local}
        if id==""||seen[id]{continue}
        seen[id]=true
        combined=append(combined,item)
    }
    core.SortGallery(combined)
    return combined
}
func mediaDiscoveryRunning(p core.CatalogPatch)bool{
    patchMediaMu.Lock()
    defer patchMediaMu.Unlock()
    return patchMediaCache[p.File].fetching
}

// A request is scoped to ONE explicitly selected patch. The Go library never
// crawls every game on the server or guesses image paths for a different XBE
// version; discovery is cached in memory to avoid hitting PatchHub repeatedly.
func discoverSelectedPatchMedia(force bool){
    app.mu.Lock()
    if app.selectedGame<0||app.selectedGame>=len(app.cat.Games){
        app.mu.Unlock()
        return
    }
    game:=app.cat.Games[app.selectedGame]
    if app.selectedPatch<0||app.selectedPatch>=len(game.Patches){
        app.mu.Unlock()
        return
    }
    patch:=game.Patches[app.selectedPatch]
    root:=app.root
    app.mu.Unlock()
    patchMediaMu.Lock()
    state:=patchMediaCache[patch.File]
    if state.fetching||(!force&&!state.lastAttempt.IsZero()&&time.Since(state.lastAttempt)<12*time.Hour){
        patchMediaMu.Unlock()
        return
    }
    state.fetching=true
    state.lastAttempt=time.Now()
    patchMediaCache[patch.File]=state
    patchMediaMu.Unlock()
    redraw()
    go func(){
        defer catchFatalPanic()
        ctx,cancel:=context.WithTimeout(context.Background(),25*time.Second)
        defer cancel()
        images,e:=core.DiscoverPatchHubMedia(ctx,nil,game,patch)
        found:=make([]core.PatchGalleryImage,0,len(images))
        if e==nil{
            // Only present successfully fetched, validated images. A URL
            // probe, or a user-supplied path, is not proof of image content.
            for _,shot:=range images{
                if ctx.Err()!=nil{break}
                if _,downloadErr:=core.DownloadCover(ctx,root,shot.URL);downloadErr==nil{
                    found=append(found,core.PatchGalleryImage{Kind:shot.Kind,Set:shot.Set,URL:shot.URL})
                }
            }
        }
        patchMediaMu.Lock()
        state:=patchMediaCache[patch.File]
        state.fetching=false
        state.images=found
        patchMediaCache[patch.File]=state
        patchMediaMu.Unlock()
        if e==nil&&len(found)>0{
            manager.log.Add("pass","PatchHub images",fmt.Sprintf("%d public JPG/PNG media links found for %s",len(found),patch.Name))
        }
        if app.hwnd!=0{
            sysCall(postMsg,app.hwnd,WM_APP+1,0,0)
        }
    }()
}
