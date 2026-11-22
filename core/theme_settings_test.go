package core

import (
    "os"
    "path/filepath"
    "testing"
)

func TestThemeSettingsPersistAndMigrate(t *testing.T) {
    defaults:=DefaultPreferences()
    if got:=defaults.Validate().Theme;got!="Midnight"{t.Fatalf("default theme = %q",got)}
    for _,theme:=range []string{"Midnight","Forest","Plum","Light"}{
        root:=t.TempDir()
        p:=DefaultPreferences()
        p.Theme=theme
        if err:=SavePreferences(root,p);err!=nil{t.Fatal(err)}
        got,err:=LoadPreferences(root)
        if err!=nil||got.Theme!=theme{t.Fatalf("theme %s failed round-trip: %+v %v",theme,got,err)}
    }
    root:=t.TempDir()
    if err:=os.WriteFile(filepath.Join(root,"settings.json"),[]byte(`{"workers":4,"catalog_filter":0}`),0600);err!=nil{t.Fatal(err)}
    old,err:=LoadPreferences(root)
    if err!=nil||old.Theme!="Midnight"{t.Fatalf("old settings migration: %+v %v",old,err)}
    bad:=Preferences{Workers:4,Theme:"Unrecognized"}
    if got:=bad.Validate().Theme;got!="Midnight"{t.Fatalf("invalid theme not defaulted: %q",got)}
}
