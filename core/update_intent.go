package core

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "strings"
)

// Portable update intents extend the EXISTING five-argument v1 helper protocol
// without ever relaxing managed-installer path validation. A helper will only
// replace a nonmanaged EXE if the running app previously staged a specific,
// locally signed-off target and PID beside the verified download.
const portableIntentSchema="magicpatchdesk-portable-update-intent-v1"
type PortableUpdateIntent struct{
    Schema string `json:"schema"`
    PID int `json:"pid"`
    Target string `json:"target"`
    Staged string `json:"staged"`
    SHA256 string `json:"sha256"`
    Nonce string `json:"nonce"`
}
func PortableUpdateIntentPath(staged string)string{return staged+".intent.json"}

func SavePortableUpdateIntent(staged,target,digest string,pid int)error{
    if pid<=0||staged==""||target==""||len(digest)!=64{return errors.New("invalid portable update request")}
    stageAbs,e:=filepath.Abs(staged)
    if e!=nil{return e}
    targetAbs,e:=filepath.Abs(target)
    if e!=nil{return e}
    if !strings.EqualFold(filepath.Dir(stageAbs),filepath.Dir(targetAbs)) ||
        !strings.HasPrefix(strings.ToLower(filepath.Base(stageAbs)),".mpd-update-") ||
        !strings.EqualFold(filepath.Ext(stageAbs),".exe")||
        !strings.EqualFold(filepath.Ext(targetAbs),".exe")||
        strings.EqualFold(stageAbs,targetAbs) {
        return errors.New("portable updater must stage a new EXE beside the original application")
    }
    if e=ValidateDownloadedDigest(stageAbs,digest);e!=nil{return e}
    if st,e:=os.Stat(targetAbs);e!=nil||!st.Mode().IsRegular(){return errors.New("portable update source executable does not exist")}
    var bytes [12]byte
    if _,e=rand.Read(bytes[:]);e!=nil{return e}
    intent:=PortableUpdateIntent{Schema:portableIntentSchema,PID:pid,Target:targetAbs,Staged:stageAbs,SHA256:strings.ToLower(digest),Nonce:hex.EncodeToString(bytes[:])}
    payload,e:=json.Marshal(intent)
    if e!=nil{return e}
    if len(payload)>4096{return errors.New("portable update intent exceeds limit")}
    file,e:=os.OpenFile(PortableUpdateIntentPath(stageAbs),os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600)
    if e!=nil{return e}
    _,writeErr:=file.Write(append(payload,'\n'))
    syncErr:=file.Sync()
    closeErr:=file.Close()
    if writeErr!=nil||syncErr!=nil||closeErr!=nil{
        _=os.Remove(file.Name())
        return errors.New("cannot persist portable update authorization")
    }
    return nil
}
func VerifyPortableUpdateIntent(staged,target,digest string,pid int)error{
    if pid<=0{return errors.New("invalid portable update parent")}
    data,e:=os.ReadFile(PortableUpdateIntentPath(staged))
    if e!=nil{return errors.New("portable update authorization missing; start update from the running app")}
    if len(data)>4096{return errors.New("portable update authorization too large")}
    var intent PortableUpdateIntent
    if e=json.Unmarshal(data,&intent);e!=nil{return errors.New("portable update authorization invalid")}
    a,e:=filepath.Abs(staged)
    if e!=nil{return e}
    b,e:=filepath.Abs(target)
    if e!=nil{return e}
    if intent.Schema!=portableIntentSchema||intent.PID!=pid||
       !strings.EqualFold(intent.Staged,a)||!strings.EqualFold(intent.Target,b)||
       intent.SHA256!=strings.ToLower(digest)||len(intent.Nonce)!=24 {
        return errors.New("portable update authorization does not match requested EXE")
    }
    if e=ValidateDownloadedDigest(a,digest);e!=nil{return e}
    return nil
}
func UpdateTargetBackup(target,managed string)(string,error){
    t,e:=filepath.Abs(target)
    if e!=nil{return "",e}
    m,e:=filepath.Abs(managed)
    if e!=nil{return "",e}
    if !strings.EqualFold(filepath.Ext(t),".exe"){return "",errors.New("update target must be an EXE")}
    name:=strings.TrimSuffix(filepath.Base(t),filepath.Ext(t))
    if name==""||strings.HasPrefix(strings.ToLower(name),".mpd-update-") ||
       strings.Contains(strings.ToLower(name),"updater")||strings.HasSuffix(strings.ToLower(name),".previous"){
       return "",errors.New("unsafe target executable filename")
    }
    if strings.EqualFold(t,m) {return filepath.Join(filepath.Dir(t),"MagicPatchDesk.previous.exe"),nil}
    return filepath.Join(filepath.Dir(t),name+".previous.exe"),nil
}
// IsPortableUpdateAuthorized is used in the v1 updater helper, alongside the
// existing strict managed path validation. Read-only local unit tests protect
// both modes without launching a real process or touching an installed app.
func IsPortableUpdateAuthorized(staged,target,digest string,pid int)error{
    return VerifyPortableUpdateIntent(staged,target,digest,pid)
}
