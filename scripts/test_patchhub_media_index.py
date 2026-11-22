#!/usr/bin/env python3
"""Offline smoke for the PatchHub media index producer; no website writes."""
import importlib.util
from pathlib import Path
import tempfile

spec=importlib.util.spec_from_file_location("media_index",Path(__file__).with_name("build_patchhub_media_index.py"))
module=importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

def main():
    with tempfile.TemporaryDirectory(prefix="mpd-media-index-") as d:
        root=Path(d)/"All-Patches"/"titles"
        folder=root/"Phantom Crash"/"USA_00000005_504C0001"/"Phantom Crash {Widescreen} (NTSC) [Jay]"
        folder.mkdir(parents=True)
        for fn in ("1. Before.jpg","2. After.png","Screenshot with no standard name.PNG"):
            path=folder/fn
            magic=b"\xff\xd8\xff" if fn.lower().endswith(".jpg") else b"\x89PNG\r\n\x1a\n"
            path.write_bytes(magic+b"x"*64)
        (folder/"ignore.exe").write_bytes(b"MZ"+b"x"*64)
        (root/"unrelated.jpg").write_bytes(b"\xff\xd8\xff"+b"x"*64)
        data=module.build_index(root)
        assert data["schema"]=="patchhub-media-v1",data
        assert len(data["images"])==3,data["images"]
        assert all("/Phantom%20Crash/" in x and "504C0001/" in x for x in data["images"]),data
        assert any("1.%20Before.jpg" in x for x in data["images"]),data
        assert any("2.%20After.png" in x for x in data["images"]),data
        assert any("Screenshot%20with%20no%20standard%20name.PNG" in x for x in data["images"]),data
    print("PatchHub media index generator: expected 3 verified public images, PASS")

if __name__=="__main__":
    main()
