# Magic Patch Desk v0.14.0-test.14 — Full patch comments and useful links

The Patch Library previously cut group comments after **50 characters**. That concealed important descriptions, setup instructions and links even though the original JMP still contained the full text.

## What changed

- Patch record **group titles now show a wrapped preview** rather than stopping at the 50-character limit. Each group also has a dedicated **Read** action that opens its complete comment in a scrollable panel.
- The **group toggle remains independent**: clicking Read does not enable or disable a patch record, and the existing **+** control still expands individual records.
- Read supports long explanations and common HTML fragments as readable, non-executable text: paragraphs, line breaks, headings, lists and entities. HTML anchor labels now retain their **real website destination**, instead of quietly discarding the `href`.
- Safe **HTTP and HTTPS** links in either a full group comment or `notes=` can be reviewed, copied and opened in the system browser. A separate explicit confirmation shows the destination before opening. Scripts, local file URLs, embedded credentials and unsupported schemes are blocked. No HTML, JavaScript, remote page or script runs within the app.
- The reader provides a **Copy original** action to preserve every character of the underlying comment, including original HTML/URLs. On sufficiently wide windows it also offers **Copy text** with the readable formatting. A bounded display prevents hostile extremely long text from exhausting native UI drawing resources, without modifying/cutting the original JMP or restricting original-content copying.
- The link navigator supports several links in the same comment and displays the full selected URL. UI controls reflow at smaller widths rather than covering each other.
- Existing full `notes=` display remains available and now benefits from the same link features.
- The human-written user `README.md` has not been changed. No record data, grouping, toggle state, patch semantics, originals or save paths are changed.

## Verification and limits

Added Go tests for long group comments, HTML headings/list text, anchor labels/destinations, Markdown/bare URLs, duplicate and unsafe URL rejection, and ensuring parser groups are unmodified. The native Windows UI smoke assertions check that **Read** and the existing toggle are separate hit targets, the complete comment is accessible, the link navigator exists, and closing the reader returns to the independent `notes=` header.

These Windows UI smoke assertions are included in the compiled executable, but they cannot be executed on a Linux CI runner. Visual checks and interaction with real Windows are still necessary. Full HTML/CSS/web-page rendering is deliberately **not** provided: JMP HTML is converted to safe, readable native text and links.

Updater protocol v1 and the canonical `MagicPatchDesk-Windows-x64.exe` / `SHA256SUMS.txt` release assets are preserved. The previous Windows unexpected-exit investigation remains open; this release does not claim to fix it.
