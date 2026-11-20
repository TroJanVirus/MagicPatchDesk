//go:build ignore
package main
import("magicpatchdesk/core";"fmt";"os";"bytes")
func main(){const dir="testdata/interop/";a,_:=os.ReadFile(dir+"source.bin");b,_:=os.ReadFile(dir+"target.bin");js,_:=os.ReadFile(dir+"js.xdelta");decoded,err:=core.DecodeXdelta(a,js);fmt.Println("Go decodes JS-produced xdelta",err,bytes.Equal(decoded,b));enc,_:=os.ReadFile(dir+"js.eips");clear,err:=core.DecryptIPS(enc,"interop-passphrase");orig,_:=os.ReadFile(dir+"go.ips");fmt.Println("Go decrypts JS EIPS",err,bytes.Equal(clear,orig))}
