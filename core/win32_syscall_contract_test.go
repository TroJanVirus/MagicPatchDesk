package core

import (
    "go/ast"
    "go/parser"
    "go/token"
    "path/filepath"
    "testing"
)

// The original custom Win32 helper could lose the pointer-lifetime contract
// of syscall.LazyProc.Call. This checks that every future code change preserves
// the compiler annotation; it runs on Linux and does not require an Xbox image.
func TestWin32SysCallWrapperKeepsGoPointerArgumentsAlive(t *testing.T) {
    path:=filepath.Join("..","cmd","magicdesk","main_windows.go")
    src,err:=parser.ParseFile(token.NewFileSet(),path,nil,parser.ParseComments)
    if err!=nil{t.Fatal(err)}
    seen:=0
    for _,decl:=range src.Decls{
        fn,ok:=decl.(*ast.FuncDecl)
        if !ok||fn.Name.Name!="sysCall"{continue}
        seen++
        annotated:=false
        if fn.Doc!=nil{
            for _,comment:=range fn.Doc.List{
                if comment.Text=="//go:uintptrescapes"{annotated=true}
            }
        }
        if !annotated{
            t.Fatal("sysCall must retain //go:uintptrescapes: Win32 arguments include Go-owned pointers")
        }
    }
    if seen!=1{t.Fatalf("expected exactly one guarded sysCall implementation, got %d",seen)}
}
