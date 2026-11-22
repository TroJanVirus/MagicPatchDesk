package core

import (
    "bytes"
    "fmt"
    "path/filepath"
    "strings"
)

// VerifyPatchAgainstModified independently applies a source or converted patch
// in memory to the original XBE and compares its *entire* output with the
// owner's expected modified XBE. Nothing is written to a game file.
// This also catches incompatible output semantics when creating a patch from
// the two XBEs, rather than validating only the input patch before conversion.
func VerifyPatchAgainstModified(original, modified, patchBytes []byte, patchName, password string) error {
    if original==nil||modified==nil {
        return fmt.Errorf("original and modified XBEs are both required for byte-for-byte verification")
    }
    if len(patchBytes)==0 {
        return fmt.Errorf("generated patch is empty")
    }
    var got []byte
    var err error
    switch strings.ToLower(filepath.Ext(patchName)) {
    case ".jmp":
        var p *Patch
        p,err=ParseJMP(filepath.Base(patchName),patchBytes)
        if err==nil {got,err=p.Apply(original,false)}
    case ".ips", ".eips":
        raw:=patchBytes
        if strings.EqualFold(filepath.Ext(patchName),".eips") {
            raw,err=DecryptIPS(patchBytes,password)
        }
        if err==nil {
            var p IPSPatch
            p,err=DecodeIPS(raw)
            if err==nil {got,err=p.Apply(original)}
        }
    case ".bps":
        got,err=DecodeBPS(original,patchBytes)
    case ".xdelta", ".vcdiff":
        got,err=DecodeXdelta(original,patchBytes)
    default:
        return fmt.Errorf("unsupported patch format for verification: %q",filepath.Ext(patchName))
    }
    if err!=nil {
        return fmt.Errorf("patch cannot reproduce the expected modified XBE: %w",err)
    }
    if !bytes.Equal(got,modified){
        return fmt.Errorf("patch does not reproduce the modified XBE byte-for-byte (generated %d bytes, expected %d bytes)",len(got),len(modified))
    }
    return nil
}
