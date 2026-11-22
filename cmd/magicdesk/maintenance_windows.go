//go:build windows

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"magicpatchdesk/core"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const helperOption = "--mpd-apply-update-v1"
const healthOption = "--mpd-healthcheck-v1"

var maintMsgBox = user32.NewProc("MessageBoxW")

var maintenancePref = struct{ AutoReports, Beta, NoAutoCheck bool }{}
var maintenanceUpdate struct {
	Release     core.Release
	Found, Busy bool
	Status      string
}

func maintenanceDialog(title, message string, flags uintptr) uintptr {
	return sysCall(maintMsgBox, app.hwnd, uintptr(unsafe.Pointer(ptr(message))), uintptr(unsafe.Pointer(ptr(title))), flags)
}
func maintenanceProgress(text string) {
	app.mu.Lock()
	maintenanceUpdate.Status = text
	app.status = text
	app.mu.Unlock()
	redraw()
}
func installedExePath() (string, error) {
	d, e := core.InstalledApplicationFolder()
	if e != nil {
		return "", e
	}
	return filepath.Join(d, "MagicPatchDesk.exe"), nil
}
func copyExecutable(src, dest string) error {
	if strings.EqualFold(filepath.Clean(src), filepath.Clean(dest)) {
		return nil
	}
	in, e := os.Open(src)
	if e != nil {
		return e
	}
	defer in.Close()
	if e = os.MkdirAll(filepath.Dir(dest), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(dest), ".mpd-install-*.exe")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = io.Copy(f, io.LimitReader(in, 80<<20)); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if _, e = os.Stat(dest); e == nil {
		return errors.New("installation target already exists; use Check for Updates for existing installations")
	}
	return os.Rename(tmp, dest)
}
func installToFolder() {
	source, e := os.Executable()
	if e != nil {
		notify(e.Error())
		return
	}
	target, e := installedExePath()
	if e != nil {
		notify(e.Error())
		return
	}
	if core.IsInstalledPath(source) {
		notify("Magic Patch Desk is installed in its own application folder.")
		return
	}
	if _, e = os.Stat(target); e == nil {
		maintenanceDialog("Already installed", "The app is already installed at:\n"+target+"\n\nUse the installed copy to update; this loose EXE has not overwritten it.", 0x40)
		return
	}
	if maintenanceDialog("Install Magic Patch Desk", "Copy this EXE to:\n"+target+"\n\nYour library, settings, diagnostics and recent history will remain in %APPDATA%\\MagicPatchDesk.\n\nInstall and open it?", 0x24|0x100) != 6 {
		return
	}
	if e = copyExecutable(source, target); e != nil {
		notify("Installation failed: " + e.Error())
		return
	}
	// The helper is a copy of this executable with a stable CLI update protocol.
	helper := filepath.Join(filepath.Dir(target), "MagicPatchDeskUpdater.exe")
	if e = copyExecutable(source, helper); e != nil {
		os.Remove(target)
		notify("Installing updater failed: " + e.Error())
		return
	}
	cmd := exec.Command(target)
	cmd.Dir = filepath.Dir(target)
	if e = cmd.Start(); e != nil {
		notify("Could not launch installed app: " + e.Error())
		return
	}
	sysCall(postMsg, app.hwnd, WM_CLOSE, 0, 0)
}
func checkForNewRelease(silent bool) {
	app.mu.Lock()
	if maintenanceUpdate.Busy {
		app.mu.Unlock()
		return
	}
	maintenanceUpdate.Busy = true
	beta := maintenancePref.Beta
	app.mu.Unlock()
    // One check = app release discovery + GitHub JMP sync + PatchHub metadata.
    // These background downloads are independent and safe when offline.
    go func(){reloadAndSync();syncPatchHubMetadata()}()
	if !silent {
		maintenanceProgress("Checking official Magic Patch Desk releases…")
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		release, available, err := core.CheckForUpdate(ctx, nil, "", core.AppVersion, beta)
		app.mu.Lock()
		maintenanceUpdate.Busy = false
		maintenanceUpdate.Release = release
		maintenanceUpdate.Found = available
		if err != nil {
			maintenanceUpdate.Status = "Update check unavailable: " + err.Error()
		} else if available {
			maintenanceUpdate.Status = "Available: " + release.TagName
		} else {
			maintenanceUpdate.Status = "Up to date (" + core.AppVersion + ")"
		}
		if !silent {
			app.status = maintenanceUpdate.Status
		}
		app.mu.Unlock()
		redraw()
        // Confidence metadata updates are independent of EXE updates. Never
        // silently upload a user report while checking the public index.
        refreshConfidenceOnUpdate()
	}()
}
// preparePortableUpdateHelper refreshes the helper from the *running* app.
// The existing helper may be an older v1 implementation that only accepts
// installed targets; reusing it for a portable update would fail again.
func preparePortableUpdateHelper(source string)(string,error){
    helper:=filepath.Join(filepath.Dir(source),"MagicPatchDeskUpdater.exe")
    expected,e:=sourceIntegrity(source)
    if e!=nil{return "",e}
    actual,e:=sourceIntegrity(helper)
    if e!=nil||actual!=expected{
        if _,e=os.Stat(helper);e==nil{
            if e=os.Remove(helper);e!=nil{
                return "",fmt.Errorf("existing updater helper cannot be replaced (another update may be active): %w",e)
            }
        }
        if e=copyExecutable(source,helper);e!=nil{return "",e}
    }
    if e=exec.Command(helper,healthOption).Run();e!=nil{
        return "",fmt.Errorf("portable updater helper health-check failed: %w",e)
    }
    return helper,nil
}

func updateNow(){
    app.mu.Lock()
    available:=maintenanceUpdate.Found
    release:=maintenanceUpdate.Release
    if maintenanceUpdate.Busy{app.mu.Unlock();return}
    app.mu.Unlock()
    if !available{
        notify("Check for updates first, or enable Preview releases to see test builds.")
        return
    }
    source,e:=os.Executable()
    if e!=nil{notify("Cannot locate running application: "+e.Error());return}
    mode:="this portable EXE"
    if core.IsInstalledPath(source){mode="the installed application"}
    message:="Download and update "+mode+" to "+release.TagName+"?\n\n"+
        "The exact executable is checked against the release SHA256 manifest and health-checked before replacement. "+
        "The previous version is retained in the same folder.\n\n"+
        "A separate installed or portable copy will NOT be replaced. Your games, patches, settings and reports are untouched."
    if maintenanceDialog("Update Magic Patch Desk",message,0x24|0x100)!=6{return}
    app.mu.Lock()
    maintenanceUpdate.Busy=true
    app.mu.Unlock()
    maintenanceProgress("Downloading and verifying "+release.TagName+"…")
    go func(){
        defer catchFatalPanic()
        target:=source // Always update the copy the user actually launched.
        dir:=filepath.Dir(target)
        ctx,cancel:=context.WithTimeout(context.Background(),4*time.Minute)
        defer cancel()
        stage,digest,e:=core.FetchVerifiedUpdate(ctx,release,dir)
        if e!=nil{maintenanceUpdateDone(e);return}
        cleanup:=func(){_ =os.Remove(stage);_=os.Remove(core.PortableUpdateIntentPath(stage))}
        // Both managed and portable updates validate the newly downloaded PE
        // while the original application is still running.
        if e=exec.Command(stage,healthOption).Run();e!=nil{
            cleanup()
            maintenanceUpdateDone(fmt.Errorf("new release failed launch self-check: %w",e))
            return
        }
        helper:=filepath.Join(dir,"MagicPatchDeskUpdater.exe")
        if core.IsInstalledPath(target){
            if _,x:=os.Stat(helper);errors.Is(x,os.ErrNotExist){
                if e=copyExecutable(source,helper);e!=nil{
                    cleanup();maintenanceUpdateDone(e);return
                }
            }
        }else{
            helper,e=preparePortableUpdateHelper(source)
            if e!=nil{cleanup();maintenanceUpdateDone(e);return}
            if e=core.SavePortableUpdateIntent(stage,target,digest,os.Getpid());e!=nil{
                cleanup();maintenanceUpdateDone(e);return
            }
        }
        cmd:=exec.Command(helper,helperOption,strconv.Itoa(os.Getpid()),target,stage,digest)
        cmd.Dir=dir
        if e=cmd.Start();e!=nil{cleanup();maintenanceUpdateDone(e);return}
        sysCall(postMsg,app.hwnd,WM_CLOSE,0,0)
    }()
}
func maintenanceUpdateDone(e error) {
	app.mu.Lock()
	maintenanceUpdate.Busy = false
	if e != nil {
		maintenanceUpdate.Status = e.Error()
		app.status = "Update failed: " + e.Error()
	}
	app.mu.Unlock()
	redraw()
}

// updateHelper is run by a separate EXE after the main process exits. Its CLI
// protocol is stable and must be maintained in all future Windows builds.
func updateHelper(args []string) error {
	if len(args) != 5 || args[0] != helperOption {
		return errors.New("invalid updater arguments")
	}
	parent, e := strconv.Atoi(args[1])
	if e != nil || parent <= 0 {
		return errors.New("invalid updater parent process")
	}
	target, staged, digest := args[2], args[3], args[4]
    expected,e:=installedExePath()
    if e!=nil{return e}
    if !strings.EqualFold(filepath.Dir(staged),filepath.Dir(target))||
       !strings.HasPrefix(strings.ToLower(filepath.Base(staged)),".mpd-update-")||
       !strings.EqualFold(filepath.Ext(staged),".exe"){
        return errors.New("untrusted staged update location")
    }
    managed:=strings.EqualFold(filepath.Clean(target),filepath.Clean(expected))
    if !managed{
        // The installed-target restrictions remain untouched for existing
        // v1 helpers. For portable updates require a one-time explicit intent
        // created by this app before it exited. The helper must be co-located.
        runningHelper,e:=os.Executable()
        if e!=nil{return e}
        wantHelper:=filepath.Join(filepath.Dir(target),"MagicPatchDeskUpdater.exe")
        if !strings.EqualFold(filepath.Clean(runningHelper),filepath.Clean(wantHelper)){
            return errors.New("portable updater helper is not beside the application")
        }
        if e=core.IsPortableUpdateAuthorized(staged,target,digest,parent);e!=nil{return e}
        defer os.Remove(core.PortableUpdateIntentPath(staged))
    }
    backup,e:=core.UpdateTargetBackup(target,expected)
    if e!=nil{return e}
    if e=core.ValidateDownloadedDigest(staged,digest);e!=nil{return e}
	kernel := syscall.NewLazyDLL("kernel32.dll")
	openProcess := kernel.NewProc("OpenProcess")
	wait := kernel.NewProc("WaitForSingleObject")
	close := kernel.NewProc("CloseHandle")
	handle, _, _ := openProcess.Call(0x00100000, 0, uintptr(parent))
	if handle != 0 {
		result, _, callErr := wait.Call(handle, 30000)
		close.Call(handle)
		if result != 0 { // WAIT_OBJECT_0: old executable has exited.
			return fmt.Errorf("old application has not exited safely (wait result %d: %v)", result, callErr)
		}
	} else {
		// The parent may have exited before OpenProcess was called.
		time.Sleep(1300 * time.Millisecond)
	}
    if e=os.Remove(backup);e!=nil&&!errors.Is(e,os.ErrNotExist){
        return fmt.Errorf("previous rollback executable cannot be replaced: %w",e)
    }
	if e = os.Rename(target, backup); e != nil {
		return fmt.Errorf("backup old executable: %w", e)
	}
	if e = os.Rename(staged, target); e != nil {
		os.Rename(backup, target)
		return fmt.Errorf("replace executable: %w", e)
	}
	if e = core.ValidateDownloadedDigest(target, digest); e != nil {
		os.Remove(target)
		os.Rename(backup, target)
		return fmt.Errorf("installed file verification: %w", e)
	}
	cmd := exec.Command(target)
	cmd.Dir = filepath.Dir(target)
	if e = cmd.Start(); e != nil {
		os.Remove(target)
		os.Rename(backup, target)
		return fmt.Errorf("launch updated application: %w", e)
	}
	return nil
}
func helperMode() bool {
	if len(os.Args) < 2 {
		return false
	}
	if os.Args[1] == healthOption {
		return true
	}
	if os.Args[1] != helperOption {
		return false
	}
	if e := updateHelper(os.Args[1:]); e != nil {
		msg := "Magic Patch Desk update failed.\n" + e.Error() + "\n\nThe previous version was preserved or restored where possible."
		sysCall(maintMsgBox, 0, uintptr(unsafe.Pointer(ptr(msg))), uintptr(unsafe.Pointer(ptr("Update failed"))), 0x10)
	}
	return true
}
func maintenanceAction(action string) {
	switch action {
	case "maint-install":
		installToFolder()
	case "maint-check":
		checkForNewRelease(false)
	case "maint-update":
		updateNow()
	case "maint-beta":
		maintenancePref.Beta = !maintenancePref.Beta
		saveMaintenanceSettings()
		maintenanceUpdate.Found = false
		checkForNewRelease(false)
	case "maint-autocheck":
		maintenancePref.NoAutoCheck = !maintenancePref.NoAutoCheck
		saveMaintenanceSettings()
	case "maint-reports":
		if !maintenancePref.AutoReports {
			if _, e := loadGitHubCredential(); e != nil {
				notify("Connect a GitHub token before enabling automatic reporting")
				return
			}
		}
		maintenancePref.AutoReports = !maintenancePref.AutoReports
		saveMaintenanceSettings()
		if maintenancePref.AutoReports {
			go syncStoredReports()
		}
	case "maint-connect":
		connectGitHubToken()
	case "maint-disconnect":
		disconnectGitHubToken()
	case "maint-sync":
		go syncStoredReports()
	case "maint-data":
		openMaintenanceFolder(app.root)
	case "maint-program":
		path, _ := core.InstalledApplicationFolder()
		openMaintenanceFolder(path)
	case "maint-cci-decompress":
		runCCIConverter(false)
	case "maint-cci-compress":
		runCCIConverter(true)
	}
}
func openMaintenanceFolder(path string) {
	if path == "" {
		return
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		notify(err.Error())
		return
	}
	sysCall(shellExecute, app.hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(path))), 0, 0, SW_SHOW)
}
func runCCIConverter(compress bool) {
	if app.busy {
		return
	}
	inFilter, title, outFilter, outputTitle, ext := "CCI image\x00*.cci\x00", "Choose Cerbios CCI image", "XISO\x00*.iso\x00", "Save decoded XISO", ".iso"
	if compress {
		inFilter, title, outFilter, outputTitle, ext = "Xbox ISO\x00*.iso\x00", "Choose uncompressed XISO", "Cerbios CCI\x00*.cci\x00", "Save compressed CCI", ".cci"
	}
	input := pickFile(false, inFilter, ""+title, "")
	if input == "" {
		return
	}
	dest := pickFile(true, outFilter, outputTitle, strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))+ext)
	if dest == "" {
		return
	}
	if !strings.EqualFold(filepath.Ext(dest), ext) {
		notify("Choose the correct output extension " + ext)
		return
	}
	if strings.EqualFold(filepath.Clean(input), filepath.Clean(dest)) {
		notify("CCI output must differ from the source file")
		return
	}
	if _, e := os.Stat(dest); e == nil {
		notify("Output already exists; choose a new file")
		return
	}
	app.mu.Lock()
	app.busy = true
	app.progress = 0
	app.status = "Processing CCI image…"
	app.mu.Unlock()
	redraw()
	go func() {
		progress := func(done, total int64) {
			app.mu.Lock()
			if total > 0 {
				app.progress = float64(done) / float64(total)
			}
			app.mu.Unlock()
			redraw()
		}
		var e error
		if compress {
			if _, verifyErr := core.OpenXISO(input); verifyErr != nil {
				e = fmt.Errorf("input is not a supported Xbox XDVDFS image: %w", verifyErr)
			} else {
				e = core.CompressCCI(input, dest, progress)
			}
		} else {
			e = core.ExtractCCI(input, dest, progress)
		}
		app.mu.Lock()
		app.busy = false
		if e != nil {
			app.status = "CCI operation failed: " + e.Error()
		} else {
			if compress {
				if _, check := os.Stat(dest); os.IsNotExist(check) {
					dest = strings.TrimSuffix(dest, ".cci") + ".1.cci"
				}
			}
			app.status = "CCI output saved: " + dest
			app.progress = 1
		}
		app.mu.Unlock()
		if e != nil {
			recordFailure("cci.convert", e, input, "", dest)
		} else {
			rememberRecent("output", dest)
		}
		redraw()
	}()
}
func sourceIntegrity(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	if e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
