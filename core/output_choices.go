package core

import (
    "errors"
    "fmt"
)

// OutputFormatForSource returns the explicit 'preserve current format' choice.
// This refers to the actual ORIGINAL source type, not a prior export preference.
func OutputFormatForSource(sourceKind string) (uint32,error) {
    switch sourceKind {
    case "xbe":return GameOutputXBE,nil
    case "folder":return GameOutputFolder,nil
    case "xiso":return GameOutputXISO,nil
    case "cci":return GameOutputCCI,nil
    case "fulliso":return GameOutputISO,nil
    default:return 0,errors.New("choose a source game before selecting an output")
    }
}

// AllowedOutputFormats prevents impossible conversions from being presented as
// selectable output options. Full-disc ISO requires a real full-disc source.
// A loose XBE cannot supply the rest of a game image.
func AllowedOutputFormats(sourceKind string)(uint32,error) {
    preserve,e:=OutputFormatForSource(sourceKind)
    if e!=nil{return 0,e}
    switch sourceKind{
    case "xbe":return GameOutputXBE,nil
    case "fulliso":return GameOutputAll,nil
    default:return (GameOutputAll&^GameOutputISO)|preserve,nil
    }
}

// ResolveGameOutputChoice unifies individually selected outputs with an
// optional preserve-the-original-format choice. No implicit default is ever
// applied; the user must explicitly select at least one option per source.
func ResolveGameOutputChoice(sourceKind string,chosen uint32,preserve bool)(uint32,error){
    allowed,e:=AllowedOutputFormats(sourceKind)
    if e!=nil{return 0,e}
    mask:=chosen
    if preserve {
        sourceFormat,e:=OutputFormatForSource(sourceKind)
        if e!=nil{return 0,e}
        mask|=sourceFormat
    }
    if mask==0{return 0,errors.New("choose at least one output format or Preserve current format")}
    if mask&^allowed!=0{
        return 0,fmt.Errorf("selected output requires game assets not present in %s source",sourceKind)
    }
    return mask,nil
}
