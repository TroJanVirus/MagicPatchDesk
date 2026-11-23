package core

import (
    "os"
    "path/filepath"
    "testing"
)

func TestDiagnosticsTabHiddenByDefaultAndSettingsMigration(t *testing.T) {
    if DefaultPreferences().ShowDiagnostics {
        t.Fatal("new installations must hide the optional developer Diagnostics workspace")
    }
    root:=t.TempDir()
    if err:=os.WriteFile(filepath.Join(root,"settings.json"),
        []byte(`{"workers":4,"catalog_filter":0,"theme":"Midnight"}`),0600);err!=nil{
        t.Fatal(err)
    }
    older,err:=LoadPreferences(root)
    if err!=nil||older.ShowDiagnostics{
        t.Fatalf("old user preferences unexpectedly enabled Diagnostics: %+v %v",older,err)
    }
    older.ShowDiagnostics=true
    if err:=SavePreferences(root,older);err!=nil{t.Fatal(err)}
    shown,err:=LoadPreferences(root)
    if err!=nil||!shown.ShowDiagnostics{t.Fatalf("visible Diagnostics flag not retained: %+v %v",shown,err)}
    shown.ShowDiagnostics=false
    if err:=SavePreferences(root,shown);err!=nil{t.Fatal(err)}
    hidden,err:=LoadPreferences(root)
    if err!=nil||hidden.ShowDiagnostics{t.Fatalf("hidden Diagnostics flag not retained: %+v %v",hidden,err)}
}

func TestStableReleaseOrderingAfterTest25(t *testing.T){
    stable,err:=ParseReleaseVersion("v0.14.0")
    if err!=nil{t.Fatal(err)}
    preview,err:=ParseReleaseVersion("v0.14.0-test.25")
    if err!=nil{t.Fatal(err)}
    if stable.Compare(preview)<=0{
        t.Fatal("stable release must update over all v0.14.0-test prereleases")
    }
}
