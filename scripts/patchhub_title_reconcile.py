#!/usr/bin/env python3
"""Read-only by default: audit JMP game-title headers against Patch Hub."""
import argparse,csv,difflib,hashlib,html,json,re
from collections import defaultdict,Counter
from pathlib import Path
from urllib.request import Request,urlopen
from urllib.parse import quote

SITE="https://www.jayxbox.com/Retail-Game-Modification/PatchHub.php"
CARD=re.compile(r'<div\s+class=["\x27]game-card["\x27][^>]*>',re.I)
TITLE=re.compile(r'<div\s+class=["\x27]game-title["\x27]>\s*([^<]+?)\s*</div>',re.I|re.S)
IDS=re.compile(r'Title ID:\s*([0-9A-F]{8})\s*\|\s*Region:\s*([^|<]+)\|\s*Version:\s*([^<]+)',re.I)
HEADER=re.compile(rb'(?im)^(game-title=)([^\r\n]*)(\r?\n|$)')
VERSION=re.compile(rb'(?im)^version=\s*(?:0x)?([0-9a-f]{8})\b')
HEX=re.compile(r'^[0-9A-F]{8}$')

def words(title):
    return " ".join("".join(c.lower() if c.isalnum() else " " for c in title).split())

def read_site(src):
    if src.startswith("http"):
        if not src.startswith("https://www.jayxbox.com/"): raise ValueError("Untrusted site URL")
        with urlopen(Request(src,headers={"User-Agent":"MagicPatchDesk-title-audit/1.0"}),timeout=35) as r:
            raw=r.read(24*1024*1024+1)
    else: raw=Path(src).read_bytes()
    if len(raw)>24*1024*1024: raise ValueError("Page is too large")
    return raw

def site_games(raw,minimum=50):
    games=[]; seen=set()
    for section in CARD.split(raw.decode("utf-8","replace"))[1:]:
        section=section[:65536]
        m=TITLE.search(section)
        if not m: continue
        title=html.unescape(m.group(1)).strip()
        if not title or title in seen: continue
        seen.add(title)
        versions=[{"title_id":i.upper(),"region":html.unescape(r).strip(),"version":html.unescape(v).strip()} for i,r,v in IDS.findall(section)]
        games.append({"title":title,"versions":versions,"title_ids":sorted({v["title_id"] for v in versions if v["title_id"]!="00000000"})})
    if len(games)<minimum: raise ValueError("Only %d site games: incomplete page" % len(games))
    return games

def parse_jmp(path,root):
    b=path.read_bytes()
    m=HEADER.search(b[:32768]); v=VERSION.search(b[:32768])
    result={"path":path.relative_to(root).as_posix(),"sha256":hashlib.sha256(b).hexdigest(),"title_id":v.group(1).decode().upper() if v else "","old_title":"","canonical_title":"","status":"","reason":"","filename_title":path.stem.split(" {",1)[0],"filename_review":"","data":b}
    if not m: result.update(status="review",reason="Missing game-title header")
    else:
        try: result["old_title"]=m.group(2).decode("utf-8-sig").strip()
        except UnicodeDecodeError: result.update(status="review",reason="Non-UTF8 header")
    return result

def classify(row,by_id):
    if row["status"]:return row
    id_=row["title_id"]; old=row["old_title"]
    if not old or not HEX.fullmatch(id_) or id_=="00000000":
        row.update(status="review",reason="No valid Title ID or title"); return row
    options=by_id.get(id_,[])
    if not options:row.update(status="unmatched",reason="Title ID not on Patch Hub");return row
    exact=[x for x in options if x["title"].casefold()==old.casefold()]
    normalized=[x for x in options if words(x["title"])==words(old)]
    if len(exact)==1: row.update(status="aligned",canonical_title=exact[0]["title"],reason="Exact title and Title ID")
    elif len(normalized)==1:row.update(status="safe",canonical_title=normalized[0]["title"],reason="Same words and Title ID")
    elif len(options)==1:row.update(status="review",canonical_title=options[0]["title"],reason="Same Title ID but different words")
    else:row.update(status="review",reason="Multiple Patch Hub games share this Title ID")
    if row["canonical_title"] and words(row["filename_title"])!=words(row["canonical_title"]):
        row["filename_review"]="Filename differs; review before rename"
    return row

def replace_header(b,old,new):
    matches=list(HEADER.finditer(b[:32768]))
    if len(matches)!=1 or matches[0].group(2).decode("utf-8-sig").strip()!=old:raise ValueError("Header changed or ambiguous")
    m=matches[0];return b[:m.start(2)]+new.encode("utf-8")+b[m.end(2):]

def cover_index(root,games):
    if not root.is_dir() or root.name.lower()!="titles":raise ValueError("Need physical All-Patches/titles directory")
    by_id=defaultdict(list); physical=defaultdict(list)
    for g in games:
        for id_ in g["title_ids"]:by_id[id_].append(g)
    for d in root.iterdir():
        if not d.is_dir() or d.is_symlink():continue
        for p in d.iterdir():
            if p.is_symlink() or not p.is_file() or p.suffix.lower() not in (".jpg",".png",".jpeg"):continue
            if HEX.fullmatch(p.stem.upper()) and p.stem!="00000000":physical[p.stem.upper()].append(p)
    covers={};issues={}
    for g in games:
        choices=set()
        for id_ in g["title_ids"]:
            if len(by_id[id_])!=1:issues[g["title"]]="Shared Xbox Title ID";continue
            matched=physical[id_]; preferred=[p for p in matched if p.parent.name==g["title"]]
            choices.update(preferred or matched)
        if len(choices)==1:
            p=next(iter(choices)).relative_to(root)
            covers[g["title"]]={"cover_path":p.as_posix(),"cover_url":"https://www.jayxbox.com/All-Patches/titles/"+"/".join(quote(s,safe="") for s in p.parts)}
        elif len(choices)>1:issues[g["title"]]=str(len(choices))+" possible covers"
    return covers,issues

def audit(root,source,out,apply_safe=False,titles_root=None,minimum=50):
    root=Path(root).resolve();out=Path(out).resolve()
    if not root.is_dir() or out==root or root in out.parents:raise ValueError("Output must be OUTSIDE local JMP clone")
    games=site_games(read_site(source),minimum=minimum)
    by_id=defaultdict(list)
    for g in games:
        for id_ in g["title_ids"]:by_id[id_].append(g)
    rows=[classify(parse_jmp(p,root),by_id) for p in sorted(root.rglob("*")) if p.is_file() and not p.is_symlink() and p.suffix.lower()==".jmp"]
    if not rows:raise ValueError("No JMP files found")
    patches=[];changed=0
    for row in rows:
        if row["status"]!="safe":continue
        old=row["data"]
        try:
            new=replace_header(old,row["old_title"],row["canonical_title"])
            before,after=old.decode("utf-8"),new.decode("utf-8")
        except (ValueError,UnicodeDecodeError):
            row.update(status="review",reason="Non-UTF8 or ambiguous patch file");continue
        patches.append("".join(difflib.unified_diff(before.splitlines(keepends=True),after.splitlines(keepends=True),fromfile="a/"+row["path"],tofile="b/"+row["path"],n=2)))
        if apply_safe and old!=new:
            path=root/row["path"]
            if hashlib.sha256(path.read_bytes()).hexdigest()!=row["sha256"]:raise ValueError("File changed during audit: "+row["path"])
            path.write_bytes(new);changed+=1
    covers,issues=cover_index(Path(titles_root).resolve(),games) if titles_root else ({},{})
    out.mkdir(parents=True,exist_ok=True)
    fields=["status","path","title_id","old_title","canonical_title","reason","filename_title","filename_review","sha256"]
    for name,subset in [("title_audit.csv",rows),("manual_review.csv",[r for r in rows if r["status"] in ("review","unmatched") or r["filename_review"]])]:
        with (out/name).open("w",encoding="utf-8-sig",newline="") as f:
            w=csv.DictWriter(f,fieldnames=fields);w.writeheader();w.writerows({k:r.get(k,"") for k in fields} for r in subset)
    (out/"proposed_title_changes.patch").write_text("".join(patches),encoding="utf-8")
    (out/"canonical_titles.v1.json").write_text(json.dumps({"schema":"patchhub-canonical-titles-v1","source":SITE,"games":[{**g,**covers.get(g["title"],{})} for g in games],"cover_review":issues},indent=2,ensure_ascii=False)+"\n",encoding="utf-8")
    summary={"site_games":len(games),"jmps":len(rows),"counts":dict(Counter(r["status"] for r in rows)),"proposed_changes":len(patches),"locally_changed":changed,"remote_changes":0}
    (out/"summary.json").write_text(json.dumps(summary,indent=2)+"\n",encoding="utf-8")
    print(json.dumps(summary,indent=2));return summary

if __name__=="__main__":
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument("--patches-root",required=True);p.add_argument("--patchhub-html",default=SITE)
    p.add_argument("--output",required=True);p.add_argument("--titles-root");p.add_argument("--apply-safe",action="store_true")
    a=p.parse_args()
    try:audit(a.patches_root,a.patchhub_html,a.output,a.apply_safe,a.titles_root)
    except (OSError,ValueError) as e:p.exit(2,"Refused: "+str(e)+"\n")
