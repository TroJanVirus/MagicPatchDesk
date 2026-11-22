package core

import (
    "sort"
    "strings"
)

// Gallery-only regional aliases: an explicit website (NTSC+PAL) label can
// contain the GitHub JMP's (NTSC) label without changing JMP applicability.
func parseGalleryRegions(raw string) (map[string]bool, bool) {
    terms := strings.Split(strings.ToUpper(strings.TrimSpace(raw)), "+")
    if len(terms)==0 || len(terms)>3 { return nil,false }
    regions:=make(map[string]bool,len(terms))
    for _,term:=range terms {
        switch strings.TrimSpace(term) {
        case "NTSC","NTSC-U","USA":
            regions["NTSC"]=true
        case "PAL":
            regions["PAL"]=true
        case "JPN","JAP","NTSC-J":
            regions["JPN"]=true
        default:
            return nil,false
        }
    }
    return regions,true
}

func galleryRegionLabel(raw string)(base,author string,regions map[string]bool,ok bool){
    name:=strings.TrimSpace(raw)
    if !strings.HasSuffix(name,"]"){return "", "",nil,false}
    authorStart:=strings.LastIndex(name,"[")
    if authorStart<0{return "", "",nil,false}
    author=CoverMatchKey(name[authorStart+1:len(name)-1])
    name=strings.TrimSpace(name[:authorStart])
    if !strings.HasSuffix(name,")"){return "", "",nil,false}
    regionStart:=strings.LastIndex(name,"(")
    if regionStart<0{return "", "",nil,false}
    regionText:=name[regionStart+1:len(name)-1]
    base=CoverMatchKey(strings.TrimSpace(name[:regionStart]))
    if base==""||author==""{return "", "",nil,false}
    regions,ok=parseGalleryRegions(regionText)
    return base,author,regions,ok
}

func patchFolderRegionalAlias(p CatalogPatch,shot HubScreenshot)bool{
    base,author,patchRegions,ok:=galleryRegionLabel(p.Name)
    if !ok{return false}
    folderBase,folderAuthor,folderRegions,ok:=galleryRegionLabel(shot.Patch)
    if !ok||base!=folderBase||author!=folderAuthor||len(folderRegions)<=len(patchRegions){return false}
    for region:=range patchRegions {
        if !folderRegions[region]{return false}
    }
    // A broader media label alone must not cross into a different game XBE.
    if shot.TitleID!=""&&p.TitleID!=""&&p.TitleID!="00000000"&&
       !strings.EqualFold(shot.TitleID,p.TitleID){return false}
    return true
}

// Same resolution for cached Patch Library galleries and selected-patch lookup:
// exact name > explicitly broader regional label > unique abbreviation.
// Multiple genuine region variants can share an explicitly multi-region
// screenshot; duplicate candidates for the SAME region remain ambiguous.
func galleryFolderOwners(patches []CatalogPatch,shot HubScreenshot,abbreviations bool)[]int{
    var exact,regional,abbreviated []int
    for i,p:=range patches{
        switch {
        case PatchFolderMatches(p.Name,shot.Patch):
            exact=append(exact,i)
        case patchFolderRegionalAlias(p,shot):
            regional=append(regional,i)
        case abbreviations&&patchFolderCandidate(p.Name,shot.Patch):
            abbreviated=append(abbreviated,i)
        }
    }
    if len(exact)>0{return exact}
    if len(regional)>0 {
        seen:=map[string]bool{}
        for _,i:=range regional {
            _,_,regions,_:=galleryRegionLabel(patches[i].Name)
            names:=make([]string,0,len(regions))
            for k:=range regions{names=append(names,k)}
            sort.Strings(names)
            key:=strings.Join(names,"+")
            if seen[key]{return nil}
            seen[key]=true
        }
        return regional
    }
    if len(abbreviated)==1{return abbreviated}
    return nil
}

func sameGalleryPatch(a,b CatalogPatch)bool{
    if a.File!=""&&b.File!=""{return strings.EqualFold(a.File,b.File)}
    return CoverMatchKey(a.Name)==CoverMatchKey(b.Name)&&
        strings.EqualFold(a.TitleID,b.TitleID)&&
        strings.EqualFold(a.Region,b.Region)
}
