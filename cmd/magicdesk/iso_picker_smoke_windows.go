//go:build windows

package main

import (
    "magicpatchdesk/core"
)

// This is intentionally filesystem-free. It exercises the Win32 output
// gate on the exact sequence: patch chosen -> ISO picked -> asynchronous
// XDVDFS classification -> show allowed outputs only after validation.
func smokeISOPickerValidation(){
    savedSource,savedKind,savedValidated:=gameOutputSource,gameOutputKind,gameOutputValidated
    savedMask,savedPreserve:=gameOutputMask,gameOutputPreserve
    savedInput,savedInputKind,savedInfo:=app.input,app.inputKind,app.inputInfo
    savedPicker:=app.outputPickerOpen
    defer func(){
        gameOutputSource,gameOutputKind,gameOutputValidated=savedSource,savedKind,savedValidated
        gameOutputMask,gameOutputPreserve=savedMask,savedPreserve
        app.input,app.inputKind,app.inputInfo=savedInput,savedInputKind,savedInfo
        app.outputPickerOpen=savedPicker
    }()

    path:="C:\\Fixture\\Phantom Crash.iso"
    app.input=path
    app.inputKind=""
    app.inputInfo=core.XBEInfo{}
    selectGameSource(path)
    if gameOutputValidated||currentGameOutputKind()!=""||availableGameOutputs()!=0{
        panic("ISO source advertised game output formats before XDVDFS validation")
    }
    setGameOutputValidation(path,"fulliso",true)
    if !gameOutputValidated||currentGameOutputKind()!="fulliso"||
        availableGameOutputs()&core.GameOutputISO==0{
        panic("Validated full-disc ISO lost its original full-disc output option")
    }
    next:="C:\\Fixture\\Phantom Crash second.iso"
    selectGameSource(next)
    setGameOutputValidation(path,"fulliso",true)
    if gameOutputValidated||availableGameOutputs()!=0{
        panic("Stale ISO result re-enabled output choices for a different file")
    }
    setGameOutputValidation(next,"xiso",true)
    if !gameOutputValidated||currentGameOutputKind()!="xiso"||
        availableGameOutputs()&core.GameOutputISO!=0{
        panic("Game-partition XISO incorrectly allowed full-disc ISO preservation")
    }
    selectGameSource(next)
    setGameOutputValidation(next,"xiso",false)
    if gameOutputValidated||availableGameOutputs()!=0{
        panic("Invalid ISO was accepted after a failed inspection")
    }
}
