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
    hUpdate,hReports,hCCI,gap:=int32(260),int32(135),int32(99),int32(12)
    if compact{hUpdate,hReports,hCCI,gap=244,113,90,8}
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
    button(hdc,"Check updates",ir(x+29+bw,by,bw,36),"maint-check",0,false)
    button(hdc,"Update now",ir(x+38+bw*2,by,bw,36),"maint-update",0,maintenanceUpdate.Found)
    fill(hdc,ir(x+23,y+136,width-46,1),0x354258)
    textAt(hdc,"UPDATE SETTINGS",x+24,y+146,width-48,25,app.fonts.Medium,white)
    betaName:="STABLE ONLY"
    if maintenancePref.Beta{betaName="PREVIEWS ON"}
    checkName:="CHECK AT START"
    if maintenancePref.NoAutoCheck{checkName="CHECK MANUALLY"}
    settingsY:=y+176
    button(hdc,betaName,ir(x+22,settingsY,145,34),"maint-beta",0,maintenancePref.Beta)
    button(hdc,checkName,ir(x+174,settingsY,164,34),"maint-autocheck",0,!maintenancePref.NoAutoCheck)
    if width>530{
        button(hdc,"App folder",ir(x+width-202,settingsY,81,34),"maint-program",0,false)
        button(hdc,"Data folder",ir(x+width-114,settingsY,91,34),"maint-data",0,false)
    }
    textAt(hdc,"Previews ON = allow downloading preview builds. CHECK AT START = check for releases at launch.",
        x+24,y+214,width-45,21,app.fonts.Small,muted)

    y+=hUpdate+gap
    panel(hdc,ir(x,y,width,hReports),surface,16)
    textAt(hdc,"AUTOMATED DIAGNOSTIC REPORTS",x+24,y+8,width-48,27,app.fonts.Title,white)
    info:="Not connected · Link a GitHub token to enable optional issue reports."
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
