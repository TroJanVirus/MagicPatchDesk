# Code signing policy — draft for owner review

**Draft only. Not yet the project's operative code-signing policy.**

Magic Patch Desk v0.14.0 is **unsigned**. The maintainer intends to apply to SignPath Foundation for free open-source Authenticode signing. No foundation approval, certificate, signing token or verified signed binary has been obtained.

If accepted, the required provider credit will be:

> Free code signing provided by SignPath.io, certificate by SignPath Foundation

Before publishing this wording as an active provider statement, confirm actual programme approval.

## Code ownership and roles (proposed, owner must confirm)

- **Authors/committers:** trusted contributors granted repository write access by the project owner, currently proposed primary maintainer `TroJanVirus`.
- **Reviewers:** project maintainers who review contributions from people who do not have trusted commit access. A contributor may not approve their own untrusted contribution. Specific assigned individuals/groups must be recorded before signing begins.
- **Release signing approver(s):** the trusted project owner/maintainer(s), with manual review and explicit approval required for every signed release. **SignPath must confirm the allowed approval arrangement for a solo maintainer**. No release signing is automatic merely because a Git tag exists.
- All people trusted to build, review, approve or sign must use multi-factor authentication for their GitHub and SignPath accounts.

## Source-to-binary and release rules (proposed)

1. The signed artefact must be built exclusively from the source and documented build scripts committed to `TroJanVirus/MagicPatchDesk`; no unsigned binary downloaded from another repository may be substituted.
2. Only approved production branches/tags may submit a release-signing request, with SignPath's trusted GitHub build/origin verification enabled. Changes to `.github/workflows`, `scripts/`, `go.mod` and other build inputs require particular review.
3. Verify `go test ./...`, core race tests, `go vet ./...`, official icon generation, self-updater compatibility, Windows x64 executable compilation, native UI smoke and updater healthcheck **before** requesting release signature.
4. A human release approver explicitly approves the exact unsigned artifact's signing request in SignPath; do not allow a PR from an untrusted contributor to release a signed binary.
5. After signing, verify Authenticode and timestamp, run the signed binary's `--mpd-healthcheck-v1`, then create the SHA-256 manifest for the **signed** EXE. Publish `MagicPatchDesk-Windows-x64.exe` and `SHA256SUMS.txt` only after every gate succeeds. No fallback-to-unsigned publication.
6. Signing secrets must be held in GitHub Actions secrets/SignPath and never written to source control or release logs; never commit a private key, certificate password, API token or access credential.
7. Preserve existing updater protocol v1 and rollback behaviour; do not modify user original XBE/ISO/CCI or imported patch data.

## Privacy

A reviewed privacy policy must link to `docs/PRIVACY_POLICY.md` after it is approved. The current application contacts public upstream services for catalogue and release updates at startup by default, and supports opt-in public GitHub diagnostic issues. Do **not** use the inaccurate statement that it never connects unless the user explicitly requests it.

The policy will link to the public code repository, release notes and the third-party services' privacy policies.

**Official criteria:** https://signpath.org/terms

**Do not merge this draft as an operative policy until the licence, branding permissions, roles and privacy questions are resolved.**
