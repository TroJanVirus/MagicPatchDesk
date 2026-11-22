package core

import (
    "html"
    "net/url"
    "regexp"
    "strings"
)

// FormatJMPNotes is a small, non-executing HTML-to-text renderer for the
// fragments found in JMP 'notes=' headers. The native UI deliberately does
// not run HTML, scripts, links or WebView content. It preserves line breaks,
// paragraphs, list items, web link destinations and character entities for readable instructions.
var (
    notesLineBreak = regexp.MustCompile(`(?i)<br\s*/?\s*>`)
    notesBlockOpen = regexp.MustCompile(`(?i)<(?:p|div|ul|ol|h[1-6])(?:\s+[^>]*)?>`)
    notesBlockClose = regexp.MustCompile(`(?i)</(?:p|div|ul|ol|h[1-6])\s*>`)
    notesItemOpen = regexp.MustCompile(`(?i)<li(?:\s+[^>]*)?>`)
    notesItemClose = regexp.MustCompile(`(?i)</li\s*>`)
    notesTag = regexp.MustCompile(`<[^>]{1,512}>`)
    notesWhitespace = regexp.MustCompile(`[\t\v\f ]+`)
    notesAnchor = regexp.MustCompile(`(?is)<a\b([^>]{0,2048})>(.*?)</a\s*>`)
    notesHref = regexp.MustCompile(`(?i)\bhref\s*=\s*(?:"([^"]{1,2048})"|'([^']{1,2048})'|([^\s>]{1,2048}))`)
    notesURL = regexp.MustCompile(`(?i)https?://[^\s<>"'\x00-\x1f]+`)
    notesMarkdownLink = regexp.MustCompile(`\[([^\]\n]{1,200})\]\((https?://[^\s)]+)\)`)

)
// JMPNoteLink is a user-supplied external URL from a patch comment or
// notes= header. This type is metadata only; extraction never opens links.
type JMPNoteLink struct {
    Label string
    URL string
}

// SafeJMPLinkURL permits only explicit HTTP(S) website links. File paths,
// scripts, custom URL schemes, credentials and control characters must never
// reach the Windows ShellExecute action.
func SafeJMPLinkURL(raw string) (string,bool) {
    raw=html.UnescapeString(raw)
    if len(raw)==0||len(raw)>2048{return "",false}
    for _,r:=range raw {
        if r<0x20||r==0x7f{return "",false}
    }
    raw=strings.TrimSpace(raw)
    parsed,err:=url.Parse(raw)
    if err!=nil{return "",false}
    if (parsed.Scheme!="https"&&parsed.Scheme!="http")||parsed.Hostname()==""||parsed.User!=nil {
        return "",false
    }
    return raw,true
}

// ExtractJMPNoteLinks finds both visible URLs and HTML/Markdown hyperlinks,
// retaining their full destination even if their displayed label is shorter.
// Duplicate URLs are shown once, with a safe maximum for the UI.
func ExtractJMPNoteLinks(raw string) []JMPNoteLink {
    if len(raw)>1<<18{raw=raw[:1<<18]}
    links:=make([]JMPNoteLink,0,4)
    seen:=map[string]bool{}
    appendLink:=func(label,candidate string){
        if len(links)>=32{return}
        safe,ok:=SafeJMPLinkURL(candidate)
        if !ok||seen[safe]{return}
        seen[safe]=true
        label=strings.TrimSpace(notesTag.ReplaceAllString(html.UnescapeString(label),""))
        if label==""{label=safe}
        labelRunes:=[]rune(label)
        if len(labelRunes)>160{label=string(labelRunes[:160])}
        links=append(links,JMPNoteLink{Label:label,URL:safe})
    }
    for _,m:=range notesAnchor.FindAllStringSubmatch(raw,-1) {
        if href:=notesHref.FindStringSubmatch(m[1]);len(href)>0{
            link:=href[1]
            if link==""{link=href[2]}
            if link==""{link=href[3]}
            appendLink(m[2],link)
        }
    }
    for _,m:=range notesMarkdownLink.FindAllStringSubmatch(raw,-1){appendLink(m[1],m[2])}
    for _,found:=range notesURL.FindAllString(html.UnescapeString(raw),-1){
        found=strings.TrimRight(found,".,;:!?)]}")
        appendLink(found,found)
    }
    return links
}

// Show the link destination as well as the label in the readable text. The
// original JMP bytes are not modified; this only affects presentation.
func formatJMPAnchor(match string) string {
    m:=notesAnchor.FindStringSubmatch(match)
    if len(m)<3{return match}
    label:=strings.TrimSpace(notesTag.ReplaceAllString(html.UnescapeString(m[2]),""))
    href:=notesHref.FindStringSubmatch(m[1])
    if len(href)<4{return label}
    candidate:=href[1]
    if candidate==""{candidate=href[2]}
    if candidate==""{candidate=href[3]}
    dest,ok:=SafeJMPLinkURL(candidate)
    if !ok{return label}
    if label==""||label==dest{return dest}
    return label+" ("+dest+")"
}

func FormatJMPNotes(raw string) string {
    if raw==""{return ""}
    // Bounded to prevent a hostile or malformed header from allocating giant
    // Paint buffers. The app can still save/parse the full raw patch unchanged.
    const limit=1<<18
    if len(raw)>limit{raw=raw[:limit]+"\n[Additional text exceeds viewer safety limit]"}
    s:=notesAnchor.ReplaceAllStringFunc(raw,formatJMPAnchor)
    s=strings.ReplaceAll(html.UnescapeString(s),"\x00","�")
    s=notesLineBreak.ReplaceAllString(s,"\n")
    s=notesItemOpen.ReplaceAllString(s,"\n• ")
    s=notesItemClose.ReplaceAllString(s,"\n")
    s=notesBlockOpen.ReplaceAllString(s,"\n")
    s=notesBlockClose.ReplaceAllString(s,"\n")
    s=notesTag.ReplaceAllString(s,"")
    s=strings.ReplaceAll(strings.ReplaceAll(s,"\r\n","\n"),"\r","\n")
    lines:=strings.Split(s,"\n")
    for i,v:=range lines {lines[i]=strings.TrimSpace(notesWhitespace.ReplaceAllString(v," "))}
    // Don't collapse intentional blank lines from <br><br>; only remove
    // leading/trailing empty fragments introduced by markup wrappers.
    start,end:=0,len(lines)
    for start<end&&lines[start]==""{start++}
    for end>start&&lines[end-1]==""{end--}
    return strings.Join(lines[start:end],"\n")
}
