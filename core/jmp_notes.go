package core

import (
    "html"
    "regexp"
    "strings"
)

// FormatJMPNotes is a small, non-executing HTML-to-text renderer for the
// fragments found in JMP 'notes=' headers. The native UI deliberately does
// not run HTML, scripts, links or WebView content. It preserves line breaks,
// paragraphs, list items and character entities for readable instructions.
var (
    notesLineBreak = regexp.MustCompile(`(?i)<br\s*/?\s*>`)
    notesBlockOpen = regexp.MustCompile(`(?i)<(?:p|div|ul|ol|h[1-6])(?:\s+[^>]*)?>`)
    notesBlockClose = regexp.MustCompile(`(?i)</(?:p|div|ul|ol|h[1-6])\s*>`)
    notesItemOpen = regexp.MustCompile(`(?i)<li(?:\s+[^>]*)?>`)
    notesItemClose = regexp.MustCompile(`(?i)</li\s*>`)
    notesTag = regexp.MustCompile(`<[^>]{1,512}>`)
    notesWhitespace = regexp.MustCompile(`[\t\v\f ]+`)
)
func FormatJMPNotes(raw string) string {
    if raw==""{return ""}
    // Bounded to prevent a hostile or malformed header from allocating giant
    // Paint buffers. The app can still save/parse the full raw patch unchanged.
    const limit=1<<18
    if len(raw)>limit{raw=raw[:limit]+"\n[Additional text exceeds viewer safety limit]"}
    s:=strings.ReplaceAll(html.UnescapeString(raw),"\x00","�")
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
