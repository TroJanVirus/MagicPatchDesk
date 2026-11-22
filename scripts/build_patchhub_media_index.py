#!/usr/bin/env python3
"""Publish an optional, read-only PatchHub media index.

Run this on the PatchHub web host (or a mirrored public All-Patches directory):
  python3 scripts/build_patchhub_media_index.py /path/to/webroot/All-Patches/titles

Upload/write the generated JSON to:
  https://www.jayxbox.com/All-Patches/patchhub-media-v1.json

This enumerates actual patch-specific filenames rather than inventing names.
No authentication, executable/game bytes, private paths or JMP payloads appear
in the JSON. The MagicPatchDesk client verifies every URL against its
game/region/version/TitleID/patch identity and image type before using it.
"""
import argparse
import json
import os
from pathlib import Path
import re
import tempfile
from urllib.parse import quote

ROOT_URL="https://www.jayxbox.com/All-Patches/titles/"
SCHEMA="patchhub-media-v1"
VERSION_FOLDER=re.compile(r"^[A-Za-z-]{2,12}_[0-9]{8}_[0-9A-Fa-f]{8}$")
SUFFIXES={".jpg",".jpeg",".png",".gif"}

def build_index(root):
    if not root.is_dir() or root.name.lower()!="titles":
        raise ValueError("Input must be the real public All-Patches/titles directory")
    images=[]
    for dirpath,subdirs,files in os.walk(root,followlinks=False):
        subdirs[:]=sorted(d for d in subdirs if not Path(dirpath,d).is_symlink())
        parent=Path(dirpath)
        for file in sorted(files):
            candidate=parent/file
            if candidate.is_symlink() or candidate.suffix.lower() not in SUFFIXES:
                continue
            rel=candidate.relative_to(root)
            parts=rel.parts
            # Exactly one game/version/patch directory and one media filename.
            if len(parts)!=4 or not VERSION_FOLDER.fullmatch(parts[1]):
                continue
            if any(p.startswith(".") or p in (".","..") for p in parts):
                continue
            stat=candidate.stat()
            if stat.st_size<32 or stat.st_size>5<<20:
                continue
            with candidate.open("rb") as stream:
                magic=stream.read(8)
            known=(magic.startswith(b"\x89PNG\r\n\x1a\n") or
                   magic.startswith(b"\xff\xd8\xff") or
                   magic.startswith((b"GIF87a",b"GIF89a")))
            if not known:
                continue
            images.append(ROOT_URL+"/".join(quote(p,safe="") for p in parts))
            if len(images)>30000:
                raise ValueError("Too many patch images; check for misplaced files")
    return {"schema":SCHEMA,"images":images}

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("titles_directory",type=Path)
    parser.add_argument("--output",type=Path,help="Default: adjacent All-Patches/patchhub-media-v1.json")
    args=parser.parse_args()
    root=args.titles_directory.resolve()
    output=args.output or root.parent/"patchhub-media-v1.json"
    index=build_index(root)
    output.parent.mkdir(parents=True,exist_ok=True)
    payload=(json.dumps(index,ensure_ascii=False,separators=(",",":"))+"\n").encode("utf-8")
    # No partially written server JSON is ever visible.
    with tempfile.NamedTemporaryFile(mode="wb",prefix=".patchhub-media-",suffix=".tmp",
                                 dir=output.parent,delete=False) as f:
        temp=Path(f.name)
        f.write(payload)
        f.flush()
        os.fsync(f.fileno())
    os.replace(temp,output)
    print(f"Indexed {len(index['images'])} verified JPG/PNG/GIF filenames -> {output}")

if __name__=="__main__":
    main()
