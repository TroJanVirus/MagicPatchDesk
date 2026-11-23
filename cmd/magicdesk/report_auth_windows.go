//go:build windows

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"magicpatchdesk/core"
	"os"
	"path/filepath"
	"strings"
	"sync"
    "sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const gitHubCredentialTarget = "MagicPatchDesk/GitHubIssues-v1"

var reportSyncMutex sync.Mutex
var githubConnected atomic.Bool

// Only retain the connection STATE in memory, never a copy of the PAT.
func initGitHubConnectionState() {
    _,err:=loadGitHubCredential()
    githubConnected.Store(err==nil)
}
func isGitHubConnected()bool {return githubConnected.Load()}

var credapi = syscall.NewLazyDLL("advapi32.dll")
var credentialWrite = credapi.NewProc("CredWriteW")
var credentialRead = credapi.NewProc("CredReadW")
var credentialFree = credapi.NewProc("CredFree")
var credentialDelete = credapi.NewProc("CredDeleteW")
var getClipboardData = user32.NewProc("GetClipboardData")

type nativeFILETIME struct{ Low, High uint32 }
type nativeCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        nativeFILETIME
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

func saveGitHubCredential(token string) error {
	if !strings.HasPrefix(token, "ghp_") && !strings.HasPrefix(token, "github_pat_") {
		return errors.New("clipboard doesn't contain a recognised GitHub Personal Access Token")
	}
	if len(token) > 512 || len(token) < 24 || strings.ContainsAny(token, " \r\n\t") {
		return errors.New("invalid token length or whitespace")
	}
	secret := []byte(token)
	defer func() {
		for i := range secret {
			secret[i] = 0
		}
	}()
	target := syscall.StringToUTF16Ptr(gitHubCredentialTarget)
	name := syscall.StringToUTF16Ptr("MagicPatchDesk")
	c := nativeCredential{Type: 1, TargetName: target, UserName: name, CredentialBlobSize: uint32(len(secret)), CredentialBlob: &secret[0], Persist: 2}
	status, _, e := credentialWrite.Call(uintptr(unsafe.Pointer(&c)), 0)
	if status == 0 {
		return fmt.Errorf("Windows Credential Manager refused token: %v", e)
	}
	return nil
}
func loadGitHubCredential() (string, error) {
	var credential *nativeCredential
	status, _, e := credentialRead.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(gitHubCredentialTarget))), 1, 0, uintptr(unsafe.Pointer(&credential)))
	if status == 0 {
		return "", fmt.Errorf("GitHub token not connected: %v", e)
	}
	defer credentialFree.Call(uintptr(unsafe.Pointer(credential)))
	if credential == nil || credential.CredentialBlobSize == 0 || credential.CredentialBlobSize > 1024 {
		return "", errors.New("invalid Windows stored credential")
	}
	return string(unsafe.Slice(credential.CredentialBlob, int(credential.CredentialBlobSize))), nil
}
func deleteGitHubCredential() error {
	status, _, e := credentialDelete.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(gitHubCredentialTarget))), 1, 0)
	if status == 0 {
		return fmt.Errorf("couldn't delete GitHub credential: %v", e)
	}
	return nil
}
func readTokenClipboard() (string, error) {
	if sysCall(openClipboard, app.hwnd) == 0 {
		return "", errors.New("couldn't access clipboard")
	}
	defer sysCall(closeClipboard)
	handle := sysCall(getClipboardData, 13) // CF_UNICODETEXT
	if handle == 0 {
		return "", errors.New("clipboard doesn't have text")
	}
	ptrHandle := sysCall(globalLock, handle)
	if ptrHandle == 0 {
		return "", errors.New("couldn't read clipboard")
	}
	size := sysCall(k32.NewProc("GlobalSize"), handle)
	if size > 2048 {
		size = 2048
	}
	raw := unsafe.Slice((*uint16)(unsafe.Pointer(ptrHandle)), int(size/2))
	n := 0
	for n < len(raw) && raw[n] != 0 {
		n++
	}
	text := syscall.UTF16ToString(raw[:n])
	sysCall(globalUnlock, handle)
	sysCall(emptyClipboard) // remove pasted secret from clipboard after reading
	return strings.TrimSpace(text), nil
}
func connectGitHubToken() {
	if maintenanceDialog("Connect GitHub diagnostics", "This allows your copy of Magic Patch Desk to create diagnostic issues and comments on TroJanVirus/MagicPatchDesk automatically.\n\n1. Create a GitHub Personal Access Token with permission to write issues in that repo.\n2. Copy the token to the Windows clipboard.\n3. Choose Yes here.\n\nThe token is stored only in Windows Credential Manager, never in diagnostic JSON or the app folder. Enable Auto-submit separately.\n\nRead clipboard token now?", 0x24|0x100) != 6 {
		return
	}
	token, e := readTokenClipboard()
	if e != nil {
		notify(e.Error())
		return
	}
	if e = saveGitHubCredential(token); e != nil {
		notify(e.Error())
		return
	}
    githubConnected.Store(true)
	notify("GitHub issue reporting token saved in Windows Credential Manager (Auto-submit still opt-in).")
}
func disconnectGitHubToken() {
	if maintenanceDialog("Disconnect GitHub", "Remove the saved GitHub Issues credential from Windows Credential Manager? No diagnostic JSON reports will be removed.", 0x24|0x100) != 6 {
		return
	}
	if e := deleteGitHubCredential(); e != nil {
		notify(e.Error())
		return
	}
	maintenancePref.AutoReports = false
    githubConnected.Store(false)
	saveMaintenanceSettings()
	notify("GitHub credential disconnected; automatic reporting disabled")
}
func syncStoredReports() {
	reportSyncMutex.Lock()
	defer reportSyncMutex.Unlock()
	token, e := loadGitHubCredential()
	if e != nil {
		notify("Report sync unavailable: connect GitHub in Maintenance")
		return
	}
	app.mu.Lock()
	root := app.root
	app.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	n, e := core.SubmitQueuedReports(ctx, root, token, 5)
	if e != nil {
		notify("Diagnostics retained locally; GitHub sync unavailable: " + e.Error())
		return
	}
	if n > 0 {
		notify(fmt.Sprintf("Submitted %d privacy-filtered diagnostic report(s) to GitHub issues", n))
	}
}
func maintenanceStorePath(root string) string { return filepath.Join(root, "maintenance.json") }
func loadMaintenanceSettings(root string) {
	prefs := struct{ AutoReports, Beta, NoAutoCheck bool }{}
	prefs.Beta = false // Stable channel is the default now that v0.14.0 is released.
	b, e := os.ReadFile(maintenanceStorePath(root))
	if e == nil { // Keep missing fields backward-compatible.
		var loaded struct{ AutoReports, Beta, NoAutoCheck bool }
		if jsonErr := unmarshalMaintenance(b, &loaded); jsonErr == nil {
			prefs = loaded
		}
	}
	maintenancePref = prefs
}
func saveMaintenanceSettings() {
	app.mu.Lock()
	root := app.root
	app.mu.Unlock()
	if root == "" {
		return
	}
	if e := os.MkdirAll(root, 0700); e != nil {
		notify(e.Error())
		return
	}
	b := marshalMaintenance(maintenancePref)
	f, e := os.CreateTemp(root, ".maintenance-*.tmp")
	if e != nil {
		notify(e.Error())
		return
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		notify(e.Error())
		return
	}
	if e = f.Sync(); e != nil {
		f.Close()
		notify(e.Error())
		return
	}
	if e = f.Close(); e != nil {
		notify(e.Error())
		return
	}
	if e = os.Rename(f.Name(), maintenanceStorePath(root)); e != nil {
		notify(e.Error())
	}
}
func marshalMaintenance(m struct{ AutoReports, Beta, NoAutoCheck bool }) []byte {
	return []byte(fmt.Sprintf("{\"auto_reports\":%t,\"beta\":%t,\"no_auto_check\":%t}\n", m.AutoReports, m.Beta, m.NoAutoCheck))
}
func unmarshalMaintenance(b []byte, out *struct{ AutoReports, Beta, NoAutoCheck bool }) error {
	if len(b) > 4096 {
		return errors.New("maintenance settings too large")
	}
	var raw struct {
		AutoReports bool `json:"auto_reports"`
		Beta        bool `json:"beta"`
		NoAutoCheck bool `json:"no_auto_check"`
	}
	if e := json.Unmarshal(b, &raw); e != nil {
		return e
	}
	out.AutoReports, out.Beta, out.NoAutoCheck = raw.AutoReports, raw.Beta, raw.NoAutoCheck
	return nil
}
