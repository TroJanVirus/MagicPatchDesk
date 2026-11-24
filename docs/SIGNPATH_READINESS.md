# Preparing Magic Patch Desk for free SignPath Foundation signing

**Status: preparation only, 2026-11-24.** The project is **not enrolled, approved or signed** by SignPath Foundation. No code-signing certificate or SignPath credential is present in this repository. This document is a readiness checklist, not a claim of compliance.

## Why we're applying

Magic Patch Desk is a non-commercially operated, publicly developed native Go/Win32 tool for applying independently authored modifications to user-supplied Original Xbox game files. The application keeps the user's original games unchanged and distributes a single Windows x64 EXE via GitHub Releases. SignPath Foundation offers free signing for approved **open-source** projects, but final approval is discretionary and the certificate would identify **SignPath Foundation** as publisher, not an individual project maintainer.

Official conditions: https://signpath.org/terms
Application: https://signpath.org/apply

## Verified against the current repository

- Public repository: https://github.com/TroJanVirus/MagicPatchDesk
- Latest released unsigned build at review: `v0.14.0`, released as stable; canonical `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt`.
- Source: Go 1.23 standard library, Win32/GDI/GDI+ integration, independent parser/converter implementations. Existing `CREDITS.md` describes most external technical sources as *reference only*. This is a **documented claim, not proof that all contributions are clear of licensing issues**.
- Repeatable verification exists: `go test ./...`, core race test, vet, `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ...`, native Windows executable smoke and updater healthcheck. Existing workflow `.github/workflows/one-shot-verify.yml` is manual and does **not** currently submit signing requests.
- **No root `LICENSE`/COPYING file exists.** Public GitHub visibility is not a substitute for an OSI-approved licence.
- Owner-supplied JMP/Patch Hub logos are embedded in the executable. Rights/permission and proposed licence for each **distributed** asset must be explicitly reviewed. Downloaded game cover artwork/JMPs are not distributed inside the EXE; they remain third-party upstream content.
- The app contacts GitHub (updates, patch catalogue) and Patch Hub (metadata/artwork) automatically at first launch unless users subsequently turn off startup checking under Maintenance. IP address and ordinary HTTP request metadata can be visible to those services. Optional redacted GitHub Issue reports require a connected token and explicit Auto-report opt-in; local crash logs stay local.
- The README has building, installation and privacy notes, but needs a correctly linked **Code signing policy**, a full **privacy policy** and explicit **uninstall instructions** before a Foundation application. Do not publish inaccurate privacy claims.

## Owner decisions needed (do not guess)

- [ ] Select an **OSI-approved licence** for the source, taking contributors' and third-party rights into account. MIT, Apache-2.0 and GPL-3.0 are examples. All allow commercial use. A non-commercial-only licence will **not** meet SignPath's OSS requirement.
- [ ] Confirm ownership/redistribution permissions for the two bundled logos and any other non-code release assets. Decide whether the selected OSS licence applies to the branding or whether additional permission/compatible licensing is necessary.
- [ ] Confirm the publisher/maintainer role assignments and whether a solo maintainer can satisfy SignPath's required signing-approval rules. Initial proposed maintainer: the GitHub repository owner, `TroJanVirus`; do not assume any other accounts' roles.
- [ ] Confirm the privacy language, particularly startup Internet access, first-run disclosure/opt-out, the independently operated Patch Hub service's logging/privacy, and the program's uninstall instructions.
- [ ] Enable GitHub MFA for every signer/committer/approver and SignPath MFA when enrolled.

## Engineering and submission steps

1. **Licence and rights:** once the owner chooses, add an unmodified OSI-approved `LICENSE`, clear SPDX identifiers and third-party asset notices. Get all necessary permission before offering SignPath a signed release.
2. **Privacy/consent:** confirm and publish `docs/PRIVACY_POLICY.md`. SignPath conditions say that transferring user data to systems not specified by the user requires a privacy policy displayed during installation plus options to disable the transfers. First-run disclosure or first-run explicit opt-in/opt-out should be discussed before making changes to the current auto-sync behaviour.
3. **Signing policy:** approve `docs/CODE_SIGNING_POLICY.md` from the prepared draft, specify maintainers/reviewers/signing approvers, restrict signed source branches and clarify that every release needs manual approval. Include the provider acknowledgement **only as a live claim after approval**.
4. **Release-page links:** add a small `Code signing policy` link to README and to GitHub Release notes/templates. Avoid altering the owner's other README wording. Add a clear uninstall procedure.
5. **Trusted build and artifact:** configure a SignPath project for the exact repository; link `GitHub.com` as trusted build system; enable origin verification; define Win64 PE artifact configuration with fixed product/version metadata restrictions; configure a release signing policy, branch restriction and manually approved requests. Install the SignPath GitHub App if required for audit log evaluation.
6. **Manual CI:** use a **manual-only** `workflow_dispatch` release flow to conserve Actions minutes. Build from source, run tests, upload **the unsigned EXE** as a GitHub Actions artifact, submit the **artifact ID** through `signpath/github-action-submit-signing-request@v3`, wait for approved signed binary, verify Authenticode on Windows, re-run EXE healthcheck on the **signed** binary, then generate `SHA256SUMS.txt` from **signed** bytes. Only then publish a GitHub Release with the existing canonical filename and checksum. Never commit signing secrets or a certificate private key.
7. **Apply:** submit the public project with honest functionality, verified release links, code signing/privacy policies and proof of rights. SignPath independently determines reputation and eligibility; no automatic acceptance or SmartScreen guarantee.
8. **Field QA:** check File Properties → Digital Signatures (issuer should be SignPath Foundation if approved); test the updater with a signed build and known previous executable, preserve `--mpd-apply-update-v1`/`--mpd-healthcheck-v1` and rollback.

Official CI instructions: https://docs.signpath.io/trusted-build-systems/github
Origin verification: https://docs.signpath.io/origin-verification/
Signing project/policy configuration: https://docs.signpath.io/projects

## Do not do these while approval is pending

- Do not add a licence without the owner's explicit choice and rights review.
- Do not claim an unsigned binary is signed or a SignPath application has been approved.
- Do not publish a signing job that succeeds by silently falling back to an unsigned EXE.
- Do not change automatic network behaviour or the privacy notice without checking the owner requirements.
- Do not edit upstream GitHub JMPs, Patch Hub files, game images, the validated updater protocol, or the stable `v0.14.0` release asset.
- Do not schedule automatic recurring GitHub Actions builds or spend CI minutes to test documentation-only work.
