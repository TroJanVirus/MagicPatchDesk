//go:build windows

package main

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
    // Keep the trusted website visible and clickable without implying that
    // the app itself is the Patch Hub website or an official upstream client.
    if linkY+58<h-45 {
        panel(hdc,ir(x,linkY,contentW,58),surface,13)
        textAt(hdc,"Discover more patches on Patch Hub",x+16,linkY+6,contentW-210,23,app.fonts.Medium,white)
        textAt(hdc,"jayxbox.com",x+16,linkY+30,contentW-210,19,app.fonts.Small,muted)
        button(hdc,"Visit Patch Hub",ir(x+contentW-177,linkY+11,160,36),"home-patchhub",0,false)
    }
}
