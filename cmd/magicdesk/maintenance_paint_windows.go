//go:build windows

package main

import (
	"fmt"
	"magicpatchdesk/core"
	"os"
)

func drawMaintenance(hdc uintptr,w,h int32){
    drawPageHeader(hdc,"Updates & maintenance","Check releases, configure updates, report problems and convert Cerbios CCI.",w)
    x:=int32(266)
    width:=w-298
    compact:=h<800
    narrow:=width<665
    hUpdate,hReports,hCCI,gap:=int32(260),int32(135),int32(99),int32(12)
    if compact{hUpdate,hReports,hCCI,gap=244,113,90,8}
    // At smaller widths, place the folder buttons on their own row. Do not
    // squeeze every action into 80px labels or let rows overlap one another.
    if narrow{hUpdate+=52}
    y:=int32(130)
    panel(hdc,ir(x,y,width,hUpdate),surface,16)
    textAt(hdc,"CHECK FOR UPDATES",x+24,y+8,width-48,29,app.fonts.Title,white)
    status:=maintenanceUpdate.Status
    if status==""{status="Check for new releases to see available updates"}
    textAt(hdc,"Installed version: "+core.AppVersion+" · "+strLimit(status,80),
        x+24,y+41,width-48,24,app.fonts.Small,muted)
    exe,_:=os.Executable()
    where:="Running portable copy · updates apply to this copy"
    if core.IsInstalledPath(exe){where="Running installed copy · updates apply to this installation"}
    if !compact{textAt(hdc,where,x+24,y+65,width-48,22,app.fonts.Small,muted)}
    by:=y+85
    bw:=(width-75)/3
    button(hdc,"Install app",ir(x+20,by,bw,36),"maint-install",0,false)
    button(hdc,"Check + sync",ir(x+29+bw,by,bw,36),"maint-check",0,false)
    button(hdc,"Update now",ir(x+38+bw*2,by,bw,36),"maint-update",0,maintenanceUpdate.Found)
    fill(hdc,ir(x+23,y+136,width-46,1),0x354258)
    textAt(hdc,"UPDATE SETTINGS",x+24,y+146,width-48,25,app.fonts.Medium,white)
    betaName:="Stable only"
    if maintenancePref.Beta{betaName="Previews on"}
    checkName:="Check + sync at start"
    if maintenancePref.NoAutoCheck{checkName="Manual check + sync"}
    settingsY:=y+176
    if !narrow{
        // Each folder button has room for its complete name, even at 1024px
        // window width. Action labels use natural casing for readability.
        button(hdc,betaName,ir(x+22,settingsY,149,35),"maint-beta",0,maintenancePref.Beta)
        button(hdc,checkName,ir(x+180,settingsY,221,35),"maint-autocheck",0,!maintenancePref.NoAutoCheck)
        button(hdc,"App folder",ir(x+width-244,settingsY,110,35),"maint-program",0,false)
        button(hdc,"Data folder",ir(x+width-126,settingsY,108,35),"maint-data",0,false)
        textAt(hdc,"At startup: check app updates, sync GitHub patches and refresh PatchHub artwork.",
            x+24,y+216,width-45,22,app.fonts.Small,muted)
    }else{
        leftWidth:=int32(142)
        rightWidth:=width-22-22-leftWidth-10
        button(hdc,betaName,ir(x+22,settingsY,leftWidth,35),"maint-beta",0,maintenancePref.Beta)
        button(hdc,checkName,ir(x+32+leftWidth,settingsY,rightWidth,35),"maint-autocheck",0,!maintenancePref.NoAutoCheck)
        button(hdc,"App folder",ir(x+22,settingsY+45,119,35),"maint-program",0,false)
        button(hdc,"Data folder",ir(x+149,settingsY+45,119,35),"maint-data",0,false)
        textWrapped(hdc,"Startup checking also refreshes GitHub patches and PatchHub images.",
            x+24,y+260,width-48,34,app.fonts.Small,muted)
    }

    y+=hUpdate+gap
    panel(hdc,ir(x,y,width,hReports),surface,16)
    textAt(hdc,"AUTOMATED DIAGNOSTIC REPORTS",x+24,y+8,width-48,27,app.fonts.Title,white)
    info:="Not connected · Connect GitHub to share optional reports."
    connected:=isGitHubConnected()
    if connected {
        info="Connected · reports stay private until you submit or enable Auto-report."
        if maintenancePref.AutoReports{
            info="Connected · Auto-report ON: send privacy-filtered reports, never raw crash logs."
        }
    }
    textAt(hdc,info,x+24,y+38,width-47,23,app.fonts.Small,muted)
    if !connected{
        button(hdc,"Connect",ir(x+22,y+hReports-47,156,35),"maint-connect",0,false)
    }else{
        buttonY:=y+hReports-45
        bw=(width-60)/3
        reportName:="Auto-report OFF"
        if maintenancePref.AutoReports{reportName="Auto-report ON"}
        for i,item:=range []struct{name,action string;on bool}{
            {"Disconnect","maint-disconnect",false},
            {reportName,"maint-reports",maintenancePref.AutoReports},
            {"Sync now","maint-sync",false},
        }{
            button(hdc,item.name,ir(x+20+int32(i)*(bw+10),buttonY,bw,35),item.action,0,item.on)
        }
    }

    y+=hReports+gap
    panel(hdc,ir(x,y,width,hCCI),surface,16)
    textAt(hdc,"CERBIOS CCI TOOLS",x+24,y+8,width-47,25,app.fonts.Title,white)
    if !compact{
        textAt(hdc,"Decode CCIM/LZ4 to XISO or compress ISO to CCI (source preserved).",x+24,y+35,width-48,22,app.fonts.Small,muted)
    }
    by=y+hCCI-42
    bw=(width-62)/2
    button(hdc,"CCI → ISO",ir(x+21,by,bw,33),"maint-cci-decompress",0,false)
    button(hdc,"ISO → CCI",ir(x+32+bw,by,bw,33),"maint-cci-compress",0,false)
    if h>850 {
        textAt(hdc,fmt.Sprintf("Update protocol %d · No administrator rights needed · App data stays in %%APPDATA%%",core.UpdateProtocol),
            x+22,y+hCCI+7,width-50,19,app.fonts.Small,muted)
    }
}
