//go:build windows

package main

import "unsafe"

// Native light-mode remapping. Earlier themes painted hard-coded midnight blue
// panels/buttons over the light canvas and reused pale text, making the UI
// unreadable. The semantic palette handles ordinary surfaces; these adapters
// cover legacy GDI literal colours in the other screens too.
func updateWindowsTitleBarTheme() {
    if app.hwnd==0{return}
    dark:=int32(1)
    if app.theme=="Light" {dark=0}
    sysCall(dwmAttr,app.hwnd,20,uintptr(unsafe.Pointer(&dark)),4)
}

func themeSurfaceColor(c uint32) uint32 {
    if app.theme!="Light" {return c}
    if c==blue||c==mint||c==amber||c==surface||c==surface2||c==bg||c==sidebar {return c}
    // Preserve saturated teal/action colours (including hover blends), while
    // replacing legacy midnight navy backgrounds.
    red,green,blueChannel:=int(c>>16)&255,int(c>>8)&255,int(c)&255
    if green>=75&&blueChannel>=green&&blueChannel-green<=48&&
        green>red*2&&blueChannel>red*3 {return c}
    switch c {
    case 0x101A2A,0x111A29,0x0C111C:
        return 0xEAF0F4
    case 0x1E2A3E,0x1F2D44,0x182335:
        return 0xFFFFFF
    case 0x283F65,0x2E4E78,0x325071:
        return 0xD5E8EF
    case 0x223756,0x24344B,0x283247,0x2D3951,0x354258,0x3A4150,0x1F2F45:
        return 0xE5EDF3
    case 0x3F73E6:
        return 0x0E7185
    case 0x447FF5,0x72A7FF:
        return 0x13758A
    }
    // A limited fallback for legacy deep navy surfaces. Preserve danger,
    // success, warm decorative art and already-light theme surfaces.
    r,g,b:=int(c>>16)&0xFF,int(c>>8)&0xFF,int(c)&0xFF
    if r<105&&g<145&&b>r+22&&b>g+8 {
        if b>175 {return 0xD3E7EE}
        return 0xE7EFF4
    }
    return c
}

func themeTextColor(c uint32) uint32 {
    if app.theme!="Light" {return c}
    // Theme palette foreground and secondary are already dark.
    if c==white||c==muted||c==blue||c==mint||c==amber{return c}
    switch c {
    case 0xFFFFFF,0xEDF3FF,0xF5F8FF,0xABB3BF:
        return 0x203845
    case 0x91A3BD,0x88A6CE,0x637793,0x7C92AD,0x7390B4:
        return 0x526773
    case 0xFF667C,0x5A73FF:
        return 0xB52F43
    }
    r,g,b:=int(c>>16)&0xFF,int(c>>8)&0xFF,int(c)&0xFF
    if r>=135&&g>=145&&b>=140 && b>=r {
        return 0x4D6573
    }
    return c
}

// The theme selector must render actual swatches for ALL four palettes,
// regardless of which colour remapping is currently active.
func rawSwatch(hdc uintptr,r rect,color uint32) {
    b:=sysCall(createSolidBrush,col(color))
    pen:=sysCall(createPen,0,1,col(color))
    oldB:=sysCall(selectObject,hdc,b)
    oldP:=sysCall(selectObject,hdc,pen)
    sysCall(roundRect,hdc,uintptr(r.L),uintptr(r.T),uintptr(r.R),uintptr(r.B),12,12)
    sysCall(selectObject,hdc,oldB)
    sysCall(selectObject,hdc,oldP)
    sysCall(deleteObject,b)
    sysCall(deleteObject,pen)
}
