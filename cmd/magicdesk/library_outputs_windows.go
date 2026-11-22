//go:build windows

package main

import (
    "fmt"
    "magicpatchdesk/core"
)

var outputOptions = []struct{
    bit uint32
    name string
    detail string
}{
    {core.GameOutputXBE,"Patched XBE","Standalone default.xbe"},
    {core.GameOutputFolder,"Extracted folder","Complete game files"},
    {core.GameOutputXISO,"Game XISO","Rebuilt game-partition image"},
    {core.GameOutputCCI,"Cerbios CCI","Compressed game image"},
    {core.GameOutputISO,"Full-disc ISO","Original disc structure"},
}

func outputSummary() string {
    if chosenGameSource()==""{return "Choose a source game"}
    mask,err:=chosenOutputMask()
    if err!=nil||mask==0{return "No output format chosen"}
    if mask&^availableGameOutputs()!=0{return "Incompatible output selected · revise formats"}
    title:=outputNames(mask)
    if gameOutputPreserve{
        title="Preserve source + "+outputNames(gameOutputMask)
        if gameOutputMask==0{title="Preserve current format ("+outputNames(mask)+")"}
    }
    if len(title)>110{title=title[:110]}
    return title
}

// An output chooser overlays the Patch Library instead of creating another
// sidebar tab. Every option is independent, supports multiple selections, and
// Preserve current format is an explicit additional choice.
func drawLibraryOutputChooser(hdc uintptr,w,h int32){
    underlay:=ir(232,0,w-232,h-42)
    panel(hdc,underlay,0x101B2B,0)
    addHit(underlay,"output-close",0)
    x:=int32(258)
    y:=int32(100)
    width:=w-x-28
    height:=h-y-68
    if width<315||height<310{
        button(hdc,"Close",ir(x+20,y+15,110,35),"output-close",0,false)
        return
    }
    panel(hdc,ir(x,y,width,height),surface,16)
    textAt(hdc,"OUTPUT FORMATS",x+23,y+14,width-164,34,app.fonts.Title,white)
    button(hdc,"Close",ir(x+width-115,y+15,91,34),"output-close",0,false)
    source:=chosenGameSource()
    kind:=currentGameOutputKind()
    textAt(hdc,fmt.Sprintf("Source: %s · %s",kind,source),x+24,y+55,width-49,23,app.fonts.Small,muted)
    choice,err:=core.OutputFormatForSource(kind)
    if err!=nil {textAt(hdc,"Select a game first.",x+24,y+85,width-52,30,app.fonts.Medium,0xFF8D80);return}
    compact:=height<440
    preserveY:=int32(89)
    if compact{preserveY=84}
    preserveRect:=ir(x+22,y+preserveY,width-44,37)
    panel(hdc,preserveRect,map[bool]uint32{true:0x345782,false:surface2}[gameOutputPreserve],10)
    mark(hdc,ir(preserveRect.L+11,preserveRect.T+10,17,17),gameOutputPreserve)
    textAt(hdc,"Preserve current format ("+outputNames(choice)+")",preserveRect.L+40,preserveRect.T+4,width-84,30,app.fonts.Medium,white)
    addHit(preserveRect,"game-preserve",0)
    labelY:=int32(135)
    gridStart,gridStep,gridRowH:=int32(162),int32(49),int32(43)
    if compact{labelY=129;gridStart=151;gridStep=41;gridRowH=35}
    textAt(hdc,"Choose one or more additional formats",x+24,y+labelY,width-49,20,app.fonts.Small,muted)
    cell:=(width-60)/2
    allowed:=availableGameOutputs()
    for i,opt:=range outputOptions{
        col:=int32(i%2)
        row:=int32(i/2)
        xx:=x+22+col*(cell+14)
        yy:=y+gridStart+row*gridStep
        selected:=gameOutputMask&opt.bit!=0
        enabled:=allowed&opt.bit!=0
        bg:=uint32(surface2)
        if selected {bg=0x345782}
        if !enabled{bg=0x262D3B}
        region:=ir(xx,yy,cell,gridRowH)
        panel(hdc,region,bg,10)
        if enabled||selected{
            mark(hdc,ir(xx+10,yy+12,17,17),selected)
            // Even if a different patch made a previous choice unavailable,
            // allow the user to deselect it instead of trapping the form.
            addHit(region,"game-format",i)
        }
        textColor:=uint32(white)
        if !enabled{textColor=0x697990}
        textAt(hdc,opt.name,xx+36,yy+1,cell-41,22,app.fonts.Medium,textColor)
        description:=opt.detail
        if !enabled{
            description="Unavailable for this source/patch"
            if selected{description="Incompatible · click to deselect"}
        }
        if !compact{textAt(hdc,description,xx+36,yy+22,cell-42,16,app.fonts.Micro,muted)}
    }
    summaryY:=y+314
    if compact {summaryY=y+height-45}
    mask,resolveErr:=chosenOutputMask()
    summaryW:=width-49
    if compact{summaryW=width-183}
    if resolveErr!=nil||mask&^allowed!=0 {
        textAt(hdc,"Choose a supported format.",x+24,summaryY,summaryW,23,app.fonts.Small,0xF7B97B)
    }else{
        textAt(hdc,"Selected: "+outputNames(mask),x+24,summaryY,summaryW,23,app.fonts.Small,mint)
    }
    if !compact{
        textAt(hdc,"The original game is never overwritten. One output folder can contain multiple formats.",x+24,summaryY+26,width-49,22,app.fonts.Small,muted)
    }
    button(hdc,"Done",ir(x+width-136,y+height-44,109,33),"output-close",0,true)
}
