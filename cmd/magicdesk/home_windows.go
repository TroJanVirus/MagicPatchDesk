//go:build windows

package main

import (
    "bytes"
    _ "embed"
    "os"
    "path/filepath"
    "sync"
)

// This transparent Patch Hub logo was provided by the project owner.
// Bundling its PNG inside the executable keeps the app self-contained.
//go:embed patchhub_home_logo.png
var patchHubHomeLogoPNG []byte

var patchHubLogoOnce sync.Once
var patchHubLogoFile string

// GDI+ needs a real UTF-16 file path for GdipLoadImageFromFile. Materialize
// the embedded, public branding image once in the app's own user-data assets
// directory; never depend on a bundled sidecar or a network request.
func patchHubHomeLogoPath() string {
    patchHubLogoOnce.Do(func(){
        if app.root==""{return}
        dir:=filepath.Join(app.root,"Assets")
        if err:=os.MkdirAll(dir,0700);err!=nil{return}
        dest:=filepath.Join(dir,"patchhub-home-logo-v1.png")
        if old,err:=os.ReadFile(dest);err==nil&&bytes.Equal(old,patchHubHomeLogoPNG){
            patchHubLogoFile=dest
            return
        }
        tmp,err:=os.CreateTemp(dir,".patchhub-logo-*.png")
        if err!=nil{return}
        defer os.Remove(tmp.Name())
        if _,err=tmp.Write(patchHubHomeLogoPNG);err!=nil{tmp.Close();return}
        if err=tmp.Sync();err!=nil{tmp.Close();return}
        if err=tmp.Close();err!=nil{return}
        if err=os.Rename(tmp.Name(),dest);err!=nil{return}
        patchHubLogoFile=dest
    })
    return patchHubLogoFile
}

// The landing screen is deliberately a short, human-facing introduction;
// technical format caveats live in the Help/About areas and documentation.
// All actions use the existing input, library and browser code paths.
func drawHome(hdc uintptr,w,h int32) {
    drawPageHeader(hdc,"Welcome home","Patch Original Xbox games, keep your originals safe, and choose your outputs.",w)
    x:=int32(255)
    contentW:=w-x-25
    heroY:=int32(119)
    compact:=h<750
    heroH:=int32(228)
    if compact{heroH=205}
    panel(hdc,ir(x,heroY,contentW,heroH),surface,18)
    textAt(hdc,"DRAG & DROP TO GET STARTED",x+25,heroY+16,contentW-50,23,app.fonts.Micro,mint)
    headline:="Drop an Xbox game or patch anywhere in this window."
    textWrapped(hdc,headline,x+25,heroY+46,contentW-50,55,app.fonts.Title,white)
    intro:="Supports XBE, ISO/XISO and Cerbios CCI, including numbered disc parts. We read the game ID and look for matching patches."
    textWrapped(hdc,intro,x+25,heroY+103,contentW-50,47,app.fonts.Small,muted)

    // Both buttons remain clickable when the window is resized to 800px.
    buttonY:=heroY+heroH-56
    firstW:=int32(168)
    secondW:=int32(185)
    if contentW<540{firstW=142;secondW=164}
    button(hdc,"Open Xbox game",ir(x+24,buttonY,firstW,39),"home-open-game",0,true)
    button(hdc,"Browse patches",ir(x+35+firstW,buttonY,secondW,39),"home-library",0,false)

    featuresY:=heroY+heroH+15
    rowText:=[]struct{title,detail string}{
        {"Find your game","Reads Title ID and CRC32 from default.xbe, even inside split CCI images."},
        {"Choose a patch","Browse widescreen, 480p, 720p and other community patches."},
        {"Pick your outputs","Export a patched XBE, extracted folder, XISO or CCI as supported."},
    }
    if contentW>=610 {
        gap:=int32(12)
        cardW:=(contentW-2*gap)/3
        cardH:=int32(134)
        for i,item:=range rowText {
            xx:=x+int32(i)*(cardW+gap)
            panel(hdc,ir(xx,featuresY,cardW,cardH),surface,15)
            textWrapped(hdc,item.title,xx+16,featuresY+15,cardW-30,43,app.fonts.Medium,white)
            textWrapped(hdc,item.detail,xx+16,featuresY+56,cardW-30,63,app.fonts.Small,muted)
        }
    }else{
        // At 800px window width a 3-column layout makes captions unreadable.
        // Three short stacked rows are better than truncated feature names.
        rowH:=int32(69)
        for i,item:=range rowText{
            yy:=featuresY+int32(i)*(rowH+7)
            panel(hdc,ir(x,yy,contentW,rowH),surface,13)
            textAt(hdc,item.title,x+14,yy+6,contentW-30,22,app.fonts.Medium,white)
            textWrapped(hdc,item.detail,x+14,yy+29,contentW-30,37,app.fonts.Small,muted)
        }
    }

    featuresBottom:=featuresY+int32(134)
    if contentW<610{featuresBottom=featuresY+3*69+2*7}
    linkY:=h-120
    if linkY<featuresBottom+12{linkY=featuresBottom+12}
    // The app already syncs the complete public JMP catalogue, so the
    // website link is about visiting the original Patch Hub, NOT discovering
    // missing patches. The entire branded card is one accessible click target.
    if linkY+72<h-43 {
        frame:=ir(x,linkY,contentW,72)
        panelColor:=surface
        if app.hoverAction=="home-patchhub"{panelColor=surface2}
        panel(hdc,frame,panelColor,13)
        logoW:=int32(204)
        if contentW<680{logoW=156}
        logoArea:=ir(x+13,linkY+9,logoW,53)
        if path:=patchHubHomeLogoPath();path!=""{
            if !drawArtFitted(hdc,path,logoArea.L,logoArea.T,
                logoArea.R-logoArea.L,logoArea.B-logoArea.T){
                textAt(hdc,"PATCH HUB",logoArea.L+8,logoArea.T+9,logoW-12,36,app.fonts.Title,mint)
            }
        }else{
            textAt(hdc,"PATCH HUB",logoArea.L+8,logoArea.T+9,logoW-12,36,app.fonts.Title,mint)
        }
        textX:=x+logoW+25
        textW:=contentW-logoW-64
        textWrapped(hdc,"Check out the Patch Hub online!",textX,linkY+11,
            textW,31,app.fonts.Medium,white)
        textAt(hdc,"So pretty!  ·  jayxbox.com",textX,linkY+39,
            textW,20,app.fonts.Small,muted)
        textAt(hdc,"↗",x+contentW-39,linkY+19,26,34,app.fonts.Title,mint)
        addHit(frame,"home-patchhub",0)
    }
}
