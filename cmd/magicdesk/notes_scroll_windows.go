//go:build windows

package main

import (
    "magicpatchdesk/core"
    "syscall"
    "unsafe"
)

const dtCalcRect=0x400
const dtNoPrefix=0x800
type listScrollBar struct{
    Track rect
    Knob rect
    Max int
}
func inRect(r rect,x,y int32)bool{return x>=r.L&&x<r.R&&y>=r.T&&y<r.B}
func boundedScroll(value,max int)int{
    if value<0{return 0}
    if value>max{return max}
    return value
}
// Draw a real, draggable scrollbar with clickable track; no list moves unless
// its own region is scrolled. The thumb scales with the visible item count.
func drawScrollBar(hdc uintptr,track rect,current,total,visible int,prefix string)listScrollBar{
    region:=listScrollBar{Track:track}
    if visible<1{visible=1}
    region.Max=total-visible
    if region.Max<=0{return region}
    height:=track.B-track.T
    if height<26{return region}
    current=boundedScroll(current,region.Max)
    knobHeight:=int32(int64(height)*int64(visible)/int64(total))
    if knobHeight<24{knobHeight=24}
    if knobHeight>height{knobHeight=height}
    travel:=height-knobHeight
    knobY:=track.T
    if region.Max>0{knobY+=int32(int64(travel)*int64(current)/int64(region.Max))}
    region.Knob=ir(track.L,knobY,track.R-track.L,knobHeight)
    panel(hdc,track,0x23334B,6)
    addHit(track,prefix+"-scroll-track",0)
    panel(hdc,region.Knob,0x6C9CFF,6)
    addHit(region.Knob,prefix+"-scroll-thumb",0)
    return region
}
func valueFromScrollTrack(bar listScrollBar,y int32)int{
    if bar.Max<=0{return 0}
    travel:=(bar.Track.B-bar.Track.T)-(bar.Knob.B-bar.Knob.T)
    if travel<=0{return 0}
    offset:=y-bar.Track.T-(bar.Knob.B-bar.Knob.T)/2
    return boundedScroll(int(int64(offset)*int64(bar.Max)/int64(travel)),bar.Max)
}
func valueFromThumbDrag(bar listScrollBar,startValue int,startY,currentY int32)int{
    if bar.Max<=0{return 0}
    travel:=(bar.Track.B-bar.Track.T)-(bar.Knob.B-bar.Knob.T)
    if travel<=0{return 0}
    return boundedScroll(startValue+int(int64(currentY-startY)*int64(bar.Max)/int64(travel)),bar.Max)
}
func setScrollSection(section string,value int){
    switch section{
    case "patch":app.patchScroll=boundedScroll(value,app.patchBar.Max)
    case "records":app.groupScroll=boundedScroll(value,app.recordBar.Max)
    case "notes":app.notesScroll=boundedScroll(value,app.notesBar.Max)
    }
}
func scrollSectionBar(section string)listScrollBar{
    switch section{
    case "patch":return app.patchBar
    case "records":return app.recordBar
    case "notes":return app.notesBar
    default:return listScrollBar{}
    }
}
func changeScrollByWheel(section string,delta int){
    if delta==0{return}
    // Mouse delta is normally ±120 per wheel notch, but high-resolution wheels
    // can return smaller values. Never change two separate lists at once.
    steps:=delta/120
    if steps==0 {if delta>0{steps=1}else{steps=-1}}
    switch section{
    case "patch":setScrollSection(section,app.patchScroll-steps*2)
    case "records":setScrollSection(section,app.groupScroll-steps*2)
    case "notes":setScrollSection(section,app.notesScroll-steps*58)
    }
}
func selectedPatchNoteText()string{
    if app.parsed!=nil {return core.FormatJMPNotes(app.parsed.Headers["notes"])}
    app.mu.Lock()
    defer app.mu.Unlock()
    if app.selectedGame>=0&&app.selectedGame<len(app.cat.Games) {
        g:=app.cat.Games[app.selectedGame]
        if app.selectedPatch>=0&&app.selectedPatch<len(g.Patches){
            return core.FormatJMPNotes(g.Patches[app.selectedPatch].Notes)
        }
    }
    return ""
}
func drawFullPatchNotes(hdc uintptr,w,h int32){
    body:=selectedPatchNoteText()
    // Reverse-hit-test modal prevents interactions with the underlying Patch
    // Library while the notes reader is active.
    background:=ir(233,0,w-233,h-42)
    panel(hdc,background,0x101B2B,0)
    addHit(background,"notes-close",0)
    x:=int32(258)
    y:=int32(107)
    width:=w-x-26
    height:=h-y-68
    if width<240||height<180{
        button(hdc,"Close",ir(x+14,y+12,110,34),"notes-close",0,false)
        return
    }
    panel(hdc,ir(x,y,width,height),surface,15)
    textAt(hdc,"FULL JMP PATCH NOTES",x+20,y+15,width-274,30,app.fonts.Title,white)
    button(hdc,"Copy all notes",ir(x+width-270,y+13,139,35),"notes-copy",0,false)
    button(hdc,"Close",ir(x+width-120,y+13,100,35),"notes-close",0,false)
    textAt(hdc,"HTML line breaks, lists and entities are shown as readable text. Nothing is omitted.",x+21,y+54,width-44,23,app.fonts.Small,muted)
    area:=ir(x+19,y+87,width-38,height-105)
    panel(hdc,area,0x202F47,9)
    textLeft:=area.L+14
    textTop:=area.T+10
    textWidth:=area.R-area.L-47
    visibleHeight:=area.B-area.T-20
    if textWidth<40||visibleHeight<35{return}
    if body==""{body="No notes are present for the selected patch."}
    wide:=syscall.StringToUTF16(body)
    measured:=ir(textLeft,textTop,textWidth,32000)
    sysCall(drawText,hdc,uintptr(unsafe.Pointer(&wide[0])),uintptr(len(wide)-1),
        uintptr(unsafe.Pointer(&measured)),DT_WORDBREAK|dtCalcRect|dtNoPrefix)
    totalHeight:=measured.B-measured.T+12
    if totalHeight<visibleHeight{totalHeight=visibleHeight}
    barMax:=int(totalHeight-visibleHeight)
    app.notesScroll=boundedScroll(app.notesScroll,barMax)
    save:=gdi32.NewProc("SaveDC")
    restore:=gdi32.NewProc("RestoreDC")
    clip:=gdi32.NewProc("IntersectClipRect")
    dc:=sysCall(save,hdc)
    sysCall(clip,hdc,uintptr(textLeft),uintptr(textTop),uintptr(textLeft+textWidth),uintptr(textTop+visibleHeight))
    textRect:=ir(textLeft,textTop-int32(app.notesScroll),textWidth,totalHeight)
    txt(hdc,body,textRect,app.fonts.Regular,white,DT_WORDBREAK|dtNoPrefix)
    if dc!=0{sysCall(restore,hdc,dc)}
    // Scroll in 1px units for long notes rather than discarding text.
    app.notesBar=drawScrollBar(hdc,ir(area.R-19,textTop,11,visibleHeight),
        app.notesScroll,int(totalHeight),int(visibleHeight),"notes")
}
