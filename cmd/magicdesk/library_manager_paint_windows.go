//go:build windows

package main

import (
    "fmt"
    "magicpatchdesk/core"
    "path/filepath"
    "strings"
)

func drawManager(hdc uintptr,w,h int32){
    drawPageHeader(hdc,"Game library manager","Scan original games, review patch matches and create verified independent copies.",w)
    if h<590 {
        textAt(hdc,"Increase window height to use the library manager.",265,145,w-298,30,app.fonts.Medium,amber)
        return
    }
    x:=int32(265)
    width:=w-294
    manager.mu.Lock()
    mode:=manager.mode
    root:=manager.root
    status:=manager.status
    busy:=manager.scanning||manager.running
    jobs:=append([]core.BatchJob(nil),manager.jobs...)
    selected:=manager.selected
    patchIndex:=manager.patchIndex
    scroll:=manager.scroll
    browse:=append([]core.ImageBrowserEntry(nil),manager.browse...)
    browseSelected:=manager.browseSelected
    browseScroll:=manager.browseScroll
    browsePath:=manager.browsePath
    conflicts:=manager.overlapping
    conflictText:=manager.overlapError
    overlapScroll:=manager.overlapScroll
    logScroll:=manager.logScroll
    confidence:=append([]core.ConfidenceReport(nil),manager.confidence...)
    confidenceScroll:=manager.confidenceScroll
    reportResult:=manager.reportResult
    manager.mu.Unlock()
    panel(hdc,ir(x,122,width,55),surface,13)
    rootLabel:="No library folder selected"
    if root!=""{rootLabel=strLimit(root,92)}
    textAt(hdc,rootLabel,x+18,135,width-209,28,app.fonts.Medium,white)
    if busy{tag(hdc,"WORKING",x+width-160,140,0x594B25)
    }else{button(hdc,"Scan folder",ir(x+width-166,132,150,36),"manager-scan",0,true)}
    tabWidth:=(width-44)/5
    tabs:=[]string{"GAMES","IMAGE FILES","HEX OVERLAPS","VERIFY LOG","FIELD REPORTS"}
    for i,name:=range tabs {
        bx:=x+int32(5+i)*(0) + 4 + int32(i)*(tabWidth+8)
        panelColor:=uint32(surface2)
        if mode==i{panelColor=0x35547E}
        panel(hdc,ir(bx,185,tabWidth,42),panelColor,9)
        textAt(hdc,name,bx+4,190,tabWidth-8,30,app.fonts.Small,white)
        addHit(ir(bx,185,tabWidth,42),"manager-mode",i)
    }
    if status!=""{textAt(hdc,strLimit(status,140),x+6,230,width-12,27,app.fonts.Small,mint)}
    switch mode{
    case 0:drawManagerGames(hdc,x,width,h,jobs,selected,patchIndex,scroll)
    case 1:drawManagerImage(hdc,x,width,h,browse,browseSelected,browseScroll,browsePath)
    case 2:drawManagerOverlap(hdc,x,width,h,conflicts,conflictText,overlapScroll)
    case 3:drawManagerLog(hdc,x,width,h,logScroll)
    case 4:drawManagerConfidence(hdc,x,width,h,jobs,selected,patchIndex,confidence,confidenceScroll,reportResult)
    }
}

func drawManagerGames(hdc uintptr,x,width,h int32,jobs []core.BatchJob,selected,patchIndex,scroll int){
    left:=width*57/100
    right:=width-left-22
    if right<235{left=width-253;right=235}
    panel(hdc,ir(x,265,left,h-355),surface,13)
    panel(hdc,ir(x+left+12,265,right,h-355),surface,13)
    textAt(hdc,fmt.Sprintf("GAMES (%d)  ·  READY / REVIEW / SKIP",len(jobs)),x+13,273,left-20,24,app.fonts.Micro,muted)
    textAt(hdc,"SELECTED GAME",x+left+25,273,right-28,23,app.fonts.Micro,muted)
    maxVisible:=int((h-360-300)/52) // corrected below: available list height
    maxVisible=int((h-355-34)/52)
    if maxVisible<1{maxVisible=1}
    if scroll>len(jobs)-1{scroll=len(jobs)-1}
    if scroll<0{scroll=0}
    for i:=scroll;i<len(jobs)&&i<scroll+maxVisible;i++{
        job:=jobs[i]
        rowY:=int32(302+(i-scroll)*52)
        active:=selected==i
        color:=uint32(surface2)
        if active{color=0x304B6B}
        panel(hdc,ir(x+11,rowY,left-22,48),color,8)
        label:=job.Game.Title
        if label==""{label=filepath.Base(job.Game.Path)}
        textAt(hdc,strLimit(label,47),x+22,rowY+4,left-150,23,app.fonts.Medium,white)
        textAt(hdc,strLimit(job.Game.Kind+" · "+job.Game.Reason,69),x+22,rowY+27,left-42,18,app.fonts.Micro,muted)
        tagColor:=uint32(0x334661)
        switch job.Decision{case core.BatchAuto:tagColor=0x245D50;case core.BatchSkip:tagColor=0x524050}
        tag(hdc,strings.ToUpper(string(job.Decision)),x+left-110,rowY+9,tagColor)
        addHit(ir(x+11,rowY,left-22,48),"manager-select",i)
    }
    if len(jobs)==0{
        textAt(hdc,"Scan a folder to discover games and matching patches.",x+20,330,left-40,60,app.fonts.Regular,muted)
    }
    side:=x+left+23
    if selected<0||selected>=len(jobs){
        textAt(hdc,"Select a game to choose its patch, output format and processing mode.",side,328,right-26,88,app.fonts.Regular,muted)
    }else{
        job:=jobs[selected]
        title:=job.Game.Title
        if title==""{title=filepath.Base(job.Game.Path)}
        textAt(hdc,strLimit(title,37),side,300,right-30,25,app.fonts.Medium,white)
        textAt(hdc,fmt.Sprintf("ID %s  ·  CRC %s",job.Game.TitleID,job.Game.CRC32),side,327,right-33,22,app.fonts.Micro,muted)
        patch:="No matching patch"
        reason:=job.Game.Reason
        if len(job.Game.Patches)>0{
            if patchIndex>=len(job.Game.Patches){patchIndex=0}
            patch=job.Game.Patches[patchIndex].Name+" ["+job.Game.Patches[patchIndex].Status+"]"
            reason=job.Game.Patches[patchIndex].Reason
        }
        textAt(hdc,strLimit(patch,43),side,354,right-32,22,app.fonts.Small,white)
        textAt(hdc,strLimit(reason,57),side,379,right-30,35,app.fonts.Micro,amber)
        if len(job.Game.Patches)>1{
            button(hdc,"< PATCH",ir(side,415,(right-36)/2,30),"manager-patch-prev",0,false)
            button(hdc,"PATCH >",ir(side+8+(right-36)/2,415,(right-36)/2,30),"manager-patch-next",0,false)
        }
        controlY:=int32(452)
        button(hdc,"Auto",ir(side,controlY,(right-37)/3,33),"manager-auto",0,job.Decision==core.BatchAuto)
        button(hdc,"Review",ir(side+6+(right-37)/3,controlY,(right-37)/3,33),"manager-review",0,job.Decision==core.BatchReview)
        button(hdc,"Skip",ir(side+12+2*((right-37)/3),controlY,(right-37)/3,33),"manager-skip",0,job.Decision==core.BatchSkip)
        format:=outputNames(job.OutputMask)
        button(hdc,"Output: "+strLimit(format,17),ir(side,494,right-33,33),"manager-format",0,false)
        button(hdc,"Open for manual patching",ir(side,534,right-33,34),"manager-open",0,false)
    }
    button(hdc,"Mark exact matches Auto",ir(x+8,h-80,left-14,32),"manager-all-safe",0,false)
    button(hdc,"RUN APPROVED AUTO",ir(x+left+13,h-80,right-4,32),"manager-run",0,true)
}
func drawManagerImage(hdc uintptr,x,width,h int32,entries []core.ImageBrowserEntry,selected,scroll int,imagePath string){
    button(hdc,"Choose XISO / ISO",ir(x+10,266,183,36),"manager-image-choose",0,false)
    button(hdc,"Inspect selected game",ir(x+203,266,190,36),"manager-image-current",0,false)
    button(hdc,"Extract selected file",ir(x+width-204,266,191,36),"manager-image-extract",0,true)
    if imagePath!=""{textAt(hdc,strLimit(imagePath,115),x+12,309,width-26,25,app.fonts.Small,muted)}
    panel(hdc,ir(x,342,width,h-432),surface,13)
    visible:=int((h-437)/34)
    if visible<1{visible=1}
    for i:=scroll;i<len(entries)&&i<scroll+visible;i++{
        entry:=entries[i]
        ry:=int32(349+(i-scroll)*34)
        back:=uint32(surface2)
        if selected==i{back=0x304C71}
        panel(hdc,ir(x+10,ry,width-20,31),back,6)
        marker:="FILE"
        if entry.IsDirectory{marker="DIR "}
        textAt(hdc,marker,x+21,ry+2,48,25,app.fonts.Micro,muted)
        textAt(hdc,strLimit(entry.Path,95),x+74,ry+2,width-225,25,app.fonts.Small,white)
        if !entry.IsDirectory{textAt(hdc,fmt.Sprintf("%d bytes",entry.Size),x+width-148,ry+2,132,25,app.fonts.Micro,muted)}
        addHit(ir(x+10,ry,width-20,31),"manager-image-select",i)
    }
    if imagePath==""{textAt(hdc,"Browse an ISO/XISO without extracting or patching it. Choose a file to make one verified copy.",x+21,373,width-50,64,app.fonts.Regular,muted)}
}
func drawManagerOverlap(hdc uintptr,x,width,h int32,r core.OverlapReport,status string,scroll int){
    textAt(hdc,"Compare eligible JMPs against the SAME original XBE; differing writes are warnings, not automatically merged.",x+10,266,width-25,33,app.fonts.Small,muted)
    textAt(hdc,strLimit(status,140),x+10,307,width-22,26,app.fonts.Regular,amber)
    panel(hdc,ir(x,344,width,h-434),surface,13)
    headers:=[]struct{label string;pos int32}{{"OFFSET",15},{"ORIGINAL",115},{"PATCH A",310},{"PATCH B",510}}
    for _,head:=range headers{textAt(hdc,head.label,x+head.pos,348,180,20,app.fonts.Micro,muted)}
    visible:=int((h-482)/43)
    if visible<1{visible=1}
    for i:=scroll;i<len(r.Overlaps)&&i<scroll+visible;i++{
        c:=r.Overlaps[i]
        ry:=int32(376+(i-scroll)*43)
        color:=uint32(0x244936)
        if !c.Same{color=0x633B40}
        panel(hdc,ir(x+9,ry,width-18,39),color,7)
        textAt(hdc,fmt.Sprintf("0x%08X",c.Offset),x+18,ry+4,98,27,app.fonts.Small,white)
        textAt(hdc,fmt.Sprintf("%02X",c.Original),x+129,ry+4,67,27,app.fonts.Medium,muted)
        textAt(hdc,fmt.Sprintf("%02X",c.A),x+325,ry+4,67,27,app.fonts.Medium,white)
        textAt(hdc,fmt.Sprintf("%02X",c.B),x+522,ry+4,67,27,app.fonts.Medium,white)
        textAt(hdc,strLimit(filepath.Base(c.PatchA)+" ↔ "+filepath.Base(c.PatchB),40),x+width-300,ry+7,278,24,app.fonts.Micro,white)
    }
    if r.Truncated{textAt(hdc,"More overlaps exist than can be displayed. Do not auto-combine these patches.",x+18,h-108,width-38,24,app.fonts.Small,amber)}
}
func drawManagerLog(hdc uintptr,x,width,h int32,scroll int){
    entries:=manager.log.Snapshot()
    textAt(hdc,fmt.Sprintf("LOCAL OPERATION HISTORY · %d events (no automatic upload)",len(entries)),x+12,270,width-300,24,app.fonts.Micro,muted)
    button(hdc,"Older",ir(x+width-244,261,75,34),"manager-log-prev",0,false)
    button(hdc,"Newer",ir(x+width-160,261,75,34),"manager-log-next",0,false)
    button(hdc,"Clear",ir(x+width-76,261,66,34),"manager-log-clear",0,false)
    panel(hdc,ir(x,311,width,h-401),surface,13)
    count:=int((h-405)/31)
    if count<1{count=1}
    start:=len(entries)-count-scroll
    if start<0{start=0}
    end:=start+count
    if end>len(entries){end=len(entries)}
    for i:=start;i<end;i++{
        entry:=entries[i]
        ry:=int32(319+(i-start)*31)
        color:=uint32(muted)
        switch entry.Level{case "pass":color=mint;case "warning":color=amber;case "error":color=0xFF9393}
        if i%2==0{panel(hdc,ir(x+7,ry,width-14,30),surface2,6)}
        textAt(hdc,entry.AtUTC.Format("15:04:05Z"),x+14,ry+2,84,25,app.fonts.Micro,muted)
        textAt(hdc,strings.ToUpper(entry.Level),x+104,ry+2,74,25,app.fonts.Micro,color)
        textAt(hdc,strLimit(entry.Step+": "+entry.Detail,175),x+186,ry+2,width-199,25,app.fonts.Small,white)
    }
    if len(entries)==0{textAt(hdc,"Verification and conversion events appear here as you scan or patch games.",x+16,345,width-35,30,app.fonts.Regular,muted)}
}
func drawManagerConfidence(hdc uintptr,x,width,h int32,jobs []core.BatchJob,selected,choice int,reports []core.ConfidenceReport,scroll int,result string){
    textAt(hdc,"No percentage score: individual exact-patch confirmations, separated by test environment.",x+11,262,width-20,26,app.fonts.Small,muted)
    selectedLabel:="Select a scanned game and patch in GAMES first"
    if selected>=0&&selected<len(jobs){
        g:=jobs[selected].Game
        selectedLabel=g.Title+" · "+g.TitleID+" · CRC "+g.CRC32
        if choice>=0&&choice<len(g.Patches){selectedLabel+=" · "+g.Patches[choice].Name}
    }
    textAt(hdc,strLimit(selectedLabel,125),x+11,292,width-19,29,app.fonts.Medium,white)
    button(hdc,"Refresh reports",ir(x+12,328,160,33),"manager-confidence-refresh",0,false)
    button(hdc,"Result: "+result,ir(x+185,328,164,33),"manager-confidence-result",0,false)
    button(hdc,"Confirm in xemu",ir(x+362,328,170,33),"manager-confidence-xemu",0,false)
    button(hdc,"Confirm real Xbox",ir(x+544,328,188,33),"manager-confidence-hardware",0,true)
    panel(hdc,ir(x,370,width,h-460),surface,13)
    visible:=int((h-464)/58)
    if visible<1{visible=1}
    for i:=scroll;i<len(reports)&&i<scroll+visible;i++{
        r:=reports[i]
        ry:=int32(380+(i-scroll)*58)
        color:=uint32(surface2)
        if r.Result=="working"{color=0x23433E}
        if r.Result=="failed"{color=0x603E40}
        panel(hdc,ir(x+11,ry,width-22,53),color,8)
        line:=strings.ToUpper(r.Environment)+" · "+strings.ToUpper(r.Result)+" · "+r.ReportedUTC
        if r.SourceIssue>0{line+=fmt.Sprintf(" · GitHub issue #%d",r.SourceIssue)}
        textAt(hdc,strLimit(line,130),x+22,ry+4,width-45,22,app.fonts.Medium,white)
        detail:=r.Hardware
        if r.Note!=""{detail+=" · "+r.Note}
        textAt(hdc,strLimit(detail,155),x+22,ry+26,width-45,20,app.fonts.Small,muted)
    }
    if len(reports)==0{textAt(hdc,"No reviewed compatibility reports for this exact patch SHA256 / original XBE combination.",x+20,401,width-40,50,app.fonts.Regular,muted)}
    textAt(hdc,"Report buttons ask for consent before creating a public review issue; no auto-upload.",x+11,h-110,width-20,24,app.fonts.Micro,amber)
}
