Patch Hub is the canonical DISPLAY title source; it is not necessarily the artwork folder name.

Run a dry-run (Python 3.10+, Git):

  python scripts/patchhub_title_reconcile.py --patches-root PATH_TO_PATCH_REPO --output PATH_OUTSIDE_PATCH_REPO

Reports: title_audit.csv, manual_review.csv, proposed_title_changes.patch, canonical_titles.v1.json, summary.json.

Safe changes require identical ordered title words (punctuation may vary) AND matching nonzero Xbox Title ID. Ambiguous or materially different names are review-only. This never renames JMP files or patch records. To apply safe game-title headers to a LOCAL clone, run with --apply-safe and inspect git diff before committing.

For an exact game-cover path inventory add --titles-root PATH_TO_WEBSITE/All-Patches/titles, which uses actual files rather than guessing image directories. Note that the American Idol cover folder differs from the site's displayed title.

JSON distinction: app-generated patchhub-index.json is a local cache, not a master file; hosted patchhub-media-v1.json lists screenshots, not game names. The new canonical_titles.v1.json is a proposed export for website master-data integration, not yet an app-consumed file.

The connected GitHub installation can READ the JayYardley JMP repository but cannot PUSH to it. Approval, write access or a local Git commit is required to apply reviewed changes there. No webroot or source JMPs are altered automatically.
