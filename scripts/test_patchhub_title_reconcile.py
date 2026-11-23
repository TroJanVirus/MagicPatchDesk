import tempfile
import unittest
from pathlib import Path
from patchhub_title_reconcile import words, site_games, replace_header, audit

def fake_page():
    pages=[f'<div class="game-card"><div class="game-title">Game {i}</div>Title ID: {i+1:08X} | Region: USA | Version: 1</div>' for i in range(50)]
    pages.append('<div class="game-card"><div class="game-title">BeatDown - Fists of Vengeance</div>Title ID: 43430015 | Region: USA | Version: 2</div>')
    return ("<html>"+"".join(pages)+"</html>").encode()

def fake_jmp(title, id_="43430015"):
    return (f"#Jay's Magic Patcher\r\nsystem=Xbox\r\ngame-title={title}\r\nversion={id_} (CC-021)\r\nauthor=Silverrock\r\n#720p\r\n0x220:FF\r\n").encode()

class AuditSafetyTests(unittest.TestCase):
    def test_token_boundaries(self):
        self.assertEqual(words("BeatDown: Fists of Vengeance"),words("BeatDown - Fists of Vengeance"))
        self.assertNotEqual(words("AB C"),words("A BC"))
    def test_game_cards(self):
        self.assertEqual(len(site_games(fake_page())),51)
    def test_bytes_only_header(self):
        original=fake_jmp("BeatDown: Fists of Vengeance")
        changed=replace_header(original,"BeatDown: Fists of Vengeance","BeatDown - Fists of Vengeance")
        self.assertEqual(changed.split(b"version=")[1],original.split(b"version=")[1])
        self.assertEqual(changed.count(b"\r\n"),original.count(b"\r\n"))
        self.assertIn(b"game-title=BeatDown - Fists of Vengeance",changed)
    def test_default_dry_run_then_safe_edit(self):
        with tempfile.TemporaryDirectory() as d:
            root=Path(d);clone=root/"patches";clone.mkdir()
            folder=clone/"Silverrock";folder.mkdir()
            patch=folder/"BeatDown - Fists of Vengeance {720p} (NTSC) [Silverrock].JMP"
            original=fake_jmp("BeatDown: Fists of Vengeance")
            patch.write_bytes(original)
            page=root/"hub.html";page.write_bytes(fake_page())
            summary=audit(clone,page,root/"report")
            self.assertEqual(summary["counts"],{"safe":1})
            self.assertEqual(patch.read_bytes(),original)
            changed=audit(clone,page,root/"report",apply_safe=True)
            self.assertEqual(changed["locally_changed"],1)
            self.assertEqual(patch.read_bytes(),replace_header(original,"BeatDown: Fists of Vengeance","BeatDown - Fists of Vengeance"))
    def test_bad_id_must_not_rewrite(self):
        with tempfile.TemporaryDirectory() as d:
            root=Path(d);clone=root/"patches";clone.mkdir()
            p=clone/"notmatch.JMP";original=fake_jmp("BeatDown: Fists of Vengeance","DEADBEEF");p.write_bytes(original)
            page=root/"site.html";page.write_bytes(fake_page())
            result=audit(clone,page,root/"report",apply_safe=True)
            self.assertEqual(result["counts"],{"unmatched":1})
            self.assertEqual(p.read_bytes(),original)

if __name__=="__main__": unittest.main()
