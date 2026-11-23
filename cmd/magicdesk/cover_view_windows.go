//go:build windows

package main

import "magicpatchdesk/core"

// A validated local cover is drawn first. Network candidates use the existing
// bounded asynchronous cache; painting never waits for an HTTP response.
func visibleGameCover(g core.Game) string {
    if g.Cover!="" && core.ValidCoverFile(g.Cover) {return g.Cover}
    return scheduleCoverCandidates(core.CoverCandidates(g))
}

// Maximize the cover within the window while preserving its native aspect.
func drawGameCoverZoom(hdc uintptr,w,h int32) {
    backdrop:=ir(233,0,w-233,h-42)
    panel(hdc,backdrop,0x0D1725,0)
    addHit(backdrop,"cover-close",0)
    x,y:=int32(260),int32(89)
    width,height:=w-x-30,h-y-66
    if width<230||height<230{
        button(hdc,"Close",ir(x+12,y+12,105,35),"cover-close",0,false)
        return
    }
    panel(hdc,ir(x,y,width,height),surface,16)
    if app.selectedGame<0||app.selectedGame>=len(app.cat.Games){
        app.coverOpen=false
        return
    }
    g:=app.cat.Games[app.selectedGame]
    textWrapped(hdc,g.Title,x+21,y+14,width-157,44,app.fonts.Title,white)
    textAt(hdc,"GAME COVER · Original proportions",x+22,y+61,width-160,21,app.fonts.Micro,muted)
    button(hdc,"Close",ir(x+width-109,y+16,92,35),"cover-close",0,false)
    area:=ir(x+22,y+92,width-44,height-114)
    panel(hdc,area,surface2,12)
    art:=visibleGameCover(g)
    if art==""||!drawArtFitted(hdc,art,area.L+10,area.T+9,
        area.R-area.L-20,area.B-area.T-18){
        txt(hdc,"Cover image unavailable",area,app.fonts.Medium,muted,
            DT_CENTER|DT_SINGLELINE|DT_VCENTER)
    }
}
