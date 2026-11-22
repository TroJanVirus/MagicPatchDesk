#!/usr/bin/env python3
"""Build the official JMP Windows icon resource, without third-party tools.

Decode the user-supplied transparent JMP logo ICO from the human-reviewable
Base64 asset and emit a standard AMD64 COFF .rsrc .syso. Go automatically
links .syso objects from cmd/magicdesk when building the executable.
The ICO is multi-resolution; no separate runtime artwork file is required.
"""
import argparse
import base64
import hashlib
import pathlib
import struct

def parse_ico(blob):
    if len(blob) < 6:
        raise ValueError("ICO too small")
    reserved, kind, count = struct.unpack_from("<HHH",blob,0)
    if reserved != 0 or kind != 1 or not (1 <= count <= 20):
        raise ValueError("Invalid ICO header")
    if len(blob) < 6 + 16 * count:
        raise ValueError("Truncated ICO directory")
    images=[]
    for i in range(count):
        w,h,colors,_reserved,planes,bpp,length,offset=struct.unpack_from("<BBBBHHII",blob,6+i*16)
        if length < 32 or offset+length>len(blob):
            raise ValueError("Invalid image extent")
        payload=blob[offset:offset+length]
        if not payload.startswith(b"\x89PNG\r\n\x1a\n"):
            raise ValueError("Only PNG-compressed icon frames supported")
        images.append((w,h,colors,planes,bpp,payload))
    return images

def resource_object(images):
    # Win32 resource tree: RT_ICON=3, RT_GROUP_ICON=14, language=0x0409.
    resource=bytearray()
    def reserve(n):
        p=len(resource);resource.extend(b"\0"*n);return p
    def directory(count):
        p=reserve(16+8*count)
        struct.pack_into("<IIHHHH",resource,p,0,0,0,0,0,count)
        return p
    def child(parent,index,rid,offset,subdir):
        struct.pack_into("<II",resource,parent+16+index*8,rid,offset|(0x80000000 if subdir else 0))
    root=directory(2)
    icons=directory(len(images))
    groups=directory(1)
    child(root,0,3,icons,True)
    child(root,1,14,groups,True)
    entries=[]
    for index,image in enumerate(images):
        lang=directory(1)
        child(icons,index,index+1,lang,True)
        descriptor=reserve(16)
        child(lang,0,0x409,descriptor,False)
        entries.append((descriptor,image[5]))
    lang=directory(1)
    child(groups,0,1,lang,True)
    descriptor=reserve(16)
    child(lang,0,0x409,descriptor,False)
    group=bytearray(struct.pack("<HHH",0,1,len(images)))
    for i,(w,h,col,planes,bpp,payload) in enumerate(images):
        group.extend(struct.pack("<BBBBHHIH",w,h,col,0,planes,bpp,len(payload),i+1))
    entries.append((descriptor,bytes(group)))
    relocations=[]
    for descriptor,payload in entries:
        while len(resource)%4:resource.append(0)
        offset=reserve(len(payload))
        resource[offset:offset+len(payload)]=payload
        struct.pack_into("<IIII",resource,descriptor,offset,len(payload),0,0)
        relocations.append((descriptor,0,0x0003)) # IMAGE_REL_AMD64_ADDR32NB
    while len(resource)%4:resource.append(0)
    resource_start=60
    reloc_start=resource_start+len(resource)
    symbols_start=reloc_start+10*len(relocations)
    hdr=struct.pack("<HHIIIHH",0x8664,1,0,symbols_start,1,0,0)
    section=struct.pack("<8sIIIIIIHHI",b".rsrc\0\0\0",0,0,len(resource),resource_start,reloc_start,0,len(relocations),0,0x40000040)
    reloc=b"".join(struct.pack("<IIH",*r) for r in relocations)
    symbol=struct.pack("<8sIhHBB",b".rsrc\0\0\0",0,1,0,3,0)
    return hdr+section+resource+reloc+symbol+struct.pack("<I",4)

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("--asset",default="assets/jmp-logo.ico.b64")
    parser.add_argument("--output",default="cmd/magicdesk/jmp_windows_amd64.syso")
    args=parser.parse_args()
    root=pathlib.Path(__file__).resolve().parent.parent
    asset_path=pathlib.Path(args.asset)
    output_path=pathlib.Path(args.output)
    if not asset_path.is_absolute():asset_path=root/asset_path
    if not output_path.is_absolute():output_path=root/output_path
    asset=asset_path.read_text(encoding="ascii")
    icon=base64.b64decode("".join(asset.split()),validate=True)
    if hashlib.sha256(icon).hexdigest()!="f0d72a8f212f656df607234262f4501a5ee66ee482d1a5f1632f24f5db8c3b63":
        raise ValueError("Official JMP logo asset hash mismatch")
    frames=parse_ico(icon)
    data=resource_object(frames)
    output_path.write_bytes(data)
    print(f"Embedded official JMP logo: {len(frames)} sizes, {len(data)}-byte PE resource")

if __name__=="__main__":
    main()
