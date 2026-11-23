//go:build windows

package main

import (
    "fmt"
    "magicpatchdesk/core"
    "path/filepath"
    "strings"
)

func selectedLibraryGallery()([]core.PatchGalleryImage,string){
    app.mu.Lock()
    defer app.mu.Unlock()
    if app.selectedGame<0||app.selectedGame>=len(app.cat.Games){return nil,""}
    game:=app.cat.Games[app.selectedGame]
    if app.selectedPatch<0||app.selectedPatch>=len(game.Patches){return nil,""}
    patch:=game.Patches[app.selectedPatch]
    return galleryForPatch(patch),game.Title+" · "+patch.Name
}

func renderGalleryPhoto(hdc uintptr,shot core.PatchGalleryImage,area rect){
    width,height:=area.R-area.L,area.B-area.T
    panel(hdc,area,0x1C2B40,12)
    source:=shot.Local
    if source==""&&shot.URL!="" {source=scheduleCover(shot.URL)}
    if source!="" && drawArtFitted(hdc,source,area.L+9,area.T+8,width-18,height-16) {return}
    txt(hdc,"Image not cached yet",area,app.fonts.Small,muted,DT_SINGLELINE|DT_CENTER|DT_VCENTER)
}

// A modal viewing surface: original images are cached on demand and shown
// without cropping. No game bytes or patch payloads are transmitted.
func drawPatchGallery(hdc uintptr,w,h int32){
    photos,label:=selectedLibraryGallery()
    if len(photos)==0 {app.galleryOpen=false;return}
    pages:=(len(photos)+1)/2
    if app.galleryPage<0{app.galleryPage=0}
    if app.galleryPage>=pages{app.galleryPage=pages-1}
    // Cover the other Patch Library hit targets. Because this hit is added
    // after the underlying page, modal controls win the reverse hit test.
    backdrop:=ir(233,0,w-233,h-42)
    panel(hdc,backdrop,0x0D1725,0)
    addHit(backdrop,"gallery-close",0)
    x,y:=int32(257),int32(105)
    width:=w-x-25
    height:=h-y-58
    if width<430||height<270 {
        textAt(hdc,"Increase window size to view screenshots",x+18,y+24,width-36,34,app.fonts.Title,white)
        button(hdc,"Close",ir(x+20,y+70,120,36),"gallery-close",0,false)
        return
    }
    panel(hdc,ir(x,y,width,height),surface,17)
    textAt(hdc,"PATCHHUB · BEFORE / AFTER IMAGES",x+23,y+12,width-205,27,app.fonts.Medium,mint)
    textAt(hdc,strLimit(label,105),x+23,y+41,width-60,26,app.fonts.Small,white)
    button(hdc,"Close",ir(x+width-111,y+18,94,36),"gallery-close",0,false)
    pictureY:=y+88
    pictureH:=height-164
    cellW:=(width-58)/2
    for side:=0;side<2;side++{
        idx:=app.galleryPage*2+side
        cellX:=x+20+int32(side)*(cellW+18)
        if idx>=len(photos){
            panel(hdc,ir(cellX,pictureY,cellW,pictureH),0x19253A,12)
            continue
        }
        shot:=photos[idx]
        phase:=strings.ToUpper(shot.Kind)
        if phase!="BEFORE"&&phase!="AFTER" {phase="SCREENSHOT"}
        label:=fmt.Sprintf("%s · %d of %d",phase,idx+1,len(photos))
        textAt(hdc,label,cellX+2,pictureY-15,cellW-4,18,app.fonts.Micro,muted)
        renderGalleryPhoto(hdc,shot,ir(cellX,pictureY+9,cellW,pictureH-9))
        if shot.Local!="" {
            textAt(hdc,strLimit(filepath.Base(shot.Local),62),cellX+6,pictureY+pictureH+2,cellW-12,18,app.fonts.Micro,muted)
        }
    }
    footerY:=y+height-51
    // At the first or last page draw disabled controls with NO hit target.
    previous:=ir(x+19,footerY,150,35)
    if app.galleryPage>0{
        button(hdc,"← Previous",previous,"gallery-prev",0,false)
    }else{
        panel(hdc,previous,0x283245,9)
        txt(hdc,"← Previous",previous,app.fonts.Small,muted,DT_SINGLELINE|DT_CENTER|DT_VCENTER)
        addHit(previous,"gallery-disabled",0)
    }
    textAt(hdc,fmt.Sprintf("Image pair %d / %d",app.galleryPage+1,pages),x+width/2-87,footerY,174,35,app.fonts.Medium,white)
    next:=ir(x+width-166,footerY,147,35)
    if app.galleryPage+1<pages{
        button(hdc,"Next →",next,"gallery-next",0,false)
    }else{
        panel(hdc,next,0x283245,9)
        txt(hdc,"Next →",next,app.fonts.Small,muted,DT_SINGLELINE|DT_CENTER|DT_VCENTER)
        addHit(next,"gallery-disabled",0)
    }
}
