package core

import (
    "strings"
    "testing"
)

func TestJMPGroupAndNotesPreserveImportantHTMLLinkDestinations(t *testing.T){
    raw:="Widescreen Support <b>(Everything is less stretched and centred)</b><br>Read <a href=\"https://example.org/guide?game=pc&amp;mode=wide\">full instructions</a>. <ul><li>Back up files first</li><li>Use an original XBE</li></ul>"
    display:=FormatJMPNotes(raw)
    for _,want:=range []string{
        "Widescreen Support (Everything is less stretched and centred)",
        "https://example.org/guide?game=pc&mode=wide",
        "full instructions", "• Back up files first","• Use an original XBE",
    }{
        if !strings.Contains(display,want){t.Fatalf("lost vital markup text %q in %q",want,display)}
    }
    if strings.Contains(display,"<a ")||strings.Contains(display,"<li>"){
        t.Fatalf("raw HTML leaked instead of readable text: %q",display)
    }
    found:=ExtractJMPNoteLinks(raw)
    if len(found)!=1||found[0].Label!="full instructions"||found[0].URL!="https://example.org/guide?game=pc&mode=wide"{
        t.Fatalf("link text/destination not retained: %+v",found)
    }
}

func TestJMPNoteLinksSupportBareAndMarkdownURLsWithoutExecution(t *testing.T){
    text:=`Visit https://example.com/patch-notes.html. Or [other guide](https://example.org/readme). <a href='http://example.net/notes'>Plain HTTP</a>`
    found:=ExtractJMPNoteLinks(text)
    want:=map[string]bool{
        "https://example.com/patch-notes.html":false,
        "https://example.org/readme":false,
        "http://example.net/notes":false,
    }
    for _,v:=range found {
        if _,ok:=want[v.URL];!ok{t.Fatalf("unexpected URL: %+v",v)}
        want[v.URL]=true
    }
    if len(found)!=len(want){t.Fatalf("unexpected link count: %+v",found)}
    for url,ok:=range want{if !ok{t.Fatalf("link %q missing: %+v",url,found)}}
}

func TestJMPSafeExternalLinkRejectsScriptAndLocalFiles(t *testing.T){
    bad:=[]string{
        "javascript:alert(1)", "file:///C:/Windows/system.ini","data:text/html,<script>alert(1)</script>",
        "https://user:password@example.com/a","https://","http://\nexample.com/",
        "https://example.com/\r\n",
    }
    for _,link:=range bad{
        if v,ok:=SafeJMPLinkURL(link);ok{t.Fatalf("unsafe destination accepted: %q => %q",link,v)}
    }
    raw:=`<a href="javascript:alert(1)">Danger</a> <a href="file:///C:/Windows/file">File</a> <a href="https://safe.example/help">Help</a>`
    found:=ExtractJMPNoteLinks(raw)
    if len(found)!=1||found[0].URL!="https://safe.example/help"{
        t.Fatalf("unsafe HTML links were exposed: %+v",found)
    }
    text:=FormatJMPNotes(raw)
    if !strings.Contains(text,"Danger")||!strings.Contains(text,"File")||strings.Contains(text,"javascript:"){
        t.Fatalf("malicious link markup lost descriptive text or left executable code: %q",text)
    }
}

func TestJMPLongRecordCommentStaysWholeInGroupDetails(t *testing.T){
    original:=strings.Repeat("Important display instructions, ",13)+"https://example.com/complete-details"
    src:="#Jay's Magic Patcher\nsystem=Xbox\ngame-title=Test\nregion=NTSC\nversion=1\nauthor=Community\nnotes=See <a href=\"https://example.org/notes\">notes</a>\n#"+original+"\n0x300:0102\n"
    patch,err:=ParseJMP("long.jmp",[]byte(src))
    if err!=nil{t.Fatal(err)}
    if len(patch.Groups)!=1||patch.Groups[0].Name!=original{t.Fatal("group comment altered by patch parser")}
    formatted:=FormatJMPNotes(patch.Groups[0].Name)
    if !strings.Contains(formatted,"https://example.com/complete-details")||len(formatted)<=50{
        t.Fatal("long group comment cannot be fully read")
    }
    if len(ExtractJMPNoteLinks(patch.Groups[0].Name))!=1{t.Fatal("group detail link unavailable")}
    if len(ExtractJMPNoteLinks(patch.Headers["notes"]))!=1{t.Fatal("header notes link unavailable")}
}
