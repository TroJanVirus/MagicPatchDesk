package core

import (
    "bytes"
    "strings"
    "testing"
)

// All formats offered in Convert Patch must reconstruct the exact reference
// modified XBE before the UI commits its output file.
func TestVerifyPatchAgainstModifiedAllConverterFormats(t *testing.T) {
    original:=bytes.Repeat([]byte{0x39},4096)
    modified:=append([]byte(nil),original...)
    copy(modified[0x300:0x308],[]byte("PATCHED!"))
    modified=append(modified,[]byte("trailing bytes")...)
    extensions:=[]string{".jmp",".ips",".xdelta",".eips",".bps"}
    for format,ext:=range extensions {
        t.Run(ext,func(t *testing.T){
            patch,e:=BuildPatchWithOptions(original,modified,format,"secret-password-1234","Fixture",true)
            if e!=nil {t.Fatal(e)}
            if e=VerifyPatchAgainstModified(original,modified,patch,"ConvertedPatch"+ext,"secret-password-1234");e!=nil{
                t.Fatalf("output patch cannot reproduce reference XBE: %v",e)
            }
            wrong:=append([]byte(nil),modified...)
            wrong[0x300]^=0x01
            if e=VerifyPatchAgainstModified(original,wrong,patch,"ConvertedPatch"+ext,"secret-password-1234");e==nil{
                t.Fatal("accepted a converted patch whose bytes differ from the reference XBE")
            }
        })
    }
}

func TestVerifyPatchAgainstModifiedRejectsInvalidReferenceAndType(t *testing.T){
    if e:=VerifyPatchAgainstModified(nil,[]byte{1},[]byte("bad"),"a.ips","");e==nil{t.Fatal("missing original accepted")}
    if e:=VerifyPatchAgainstModified([]byte{1},nil,[]byte("bad"),"a.ips","");e==nil{t.Fatal("missing reference accepted")}
    if e:=VerifyPatchAgainstModified([]byte{1},[]byte{2},nil,"a.ips","");e==nil{t.Fatal("empty patch accepted")}
    if e:=VerifyPatchAgainstModified([]byte{1},[]byte{2},[]byte("hello"),"a.exe","");e==nil||!strings.Contains(e.Error(),"unsupported"){
        t.Fatalf("wrong format accepted: %v",e)
    }
}
