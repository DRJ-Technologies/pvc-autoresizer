# Agent Instructions

<!-- DRJ:CONTEXT BEGIN v=1f54f6e — generated from DRJ-Technologies/drj-context; do not edit here -->
> **Parent context:** [DRJ-Technologies/drj-context](https://github.com/DRJ-Technologies/drj-context)
> is canonical for DRJ platform patterns, preferences, and laws.
> Read the authenticated repository through `gh api` or its registered linked worktree.
> (private repo — a plain web fetch will 404; source edits require an owned linked worktree).
>
> **Everything below the END marker is repo-local and supersedes this block on conflict.**

## DRJ org context (synced — edit in drj-context, never here)

**Environment** — identify the session manager first; Herdr commands/paths/sweeps require verified Herdr scope. AgentDeck keeps its own setup. Keep agent/session/terminal IDs out of shared repos; discover contacts in the owning environment, never route from a copied UUID. → `operating.md#environment-and-session-manager`
**Secrets / configuration** — never print/log/commit credentials. All new work uses AWS Secrets Manager for credentials, SSM for centrally managed nonsecret settings and the existing ESO/ConfigMap GitOps pattern for Kubernetes; other consumers use the same AWS sources. No new Doppler dependencies. → `laws.md#secrets`, `laws.md#aws-configuration-and-eso`, `patterns.md#aws-configuration-and-eso`
**Infrastructure** — all new Terraform/OpenTofu work uses OTF, not Spacelift: native PR plans, relevant main commits create saved plans, explicit UI/robot CLI approval; no auto-apply. Keep the documented independent management bootstrap exception and Kubernetes GitOps path. → `laws.md#infrastructure-delivery-through-otf`, `operating.md#native-otf-gitops`
**Co-founders** — Dan and Rob have equal DRJ permissions under their own identities; no Dan-only build, publish, deploy or admin gate. Preserve common safeguards and separate credentials/sessions. → `laws.md#co-founder-access-parity`
**Upstream first** — native interfaces/formats, one maintained implementation and inputs defined once; derive metadata at its consumer. No copied query/build hashes in schemas/templates/manifests, hashes-of-copies inventories or tests only synchronizing them; remove existing duplication when changing affected tooling. Custom schemas/gates/receipts need a named failure existing tools do not prevent. Scoped read-only diagnostics use reusable commands/private output without per-attempt release gates or one-shot retry bans; preserve access/privacy/bounds, review/CI, custody, holds and irreversible safeguards. → `operating.md#process-only-if-absolutely-needed`
**Process** — only if absolutely needed: rigor (main-proven inputs, receipts, repeated review rounds) for irreversible actions only; docs/plans/coordination stay light; one review round, merges at the reviewed head; no maintenance windows unless the owner asks; act autonomously, never route approvals to the owner; leads drive the accepted plan to full delivery with no idle lanes. → `operating.md#process-only-if-absolutely-needed`, `#act-autonomously-never-route-approvals-to-the-owner`, `#leads-drive-the-plan-to-full-delivery`
**Teams** — autonomous: own your members, reviewers and infrastructure; app teams may change and roll out platform systems for their usage through approved paths with their own identities. Coordinate actual shared conflicts lead-to-lead; no global queue approval or default maintenance ceremony. Preserve review/CI/main-only provenance/holds; never replay or interrupt other owners' operations. Never share workers; bias toward N independent coders and one reviewer per PR with project context and an author brief. Only leads communicate across teams; members inspect read-only. → `operating.md#team-ownership-and-cross-team-boundaries`
**Unfamiliar tools** — read the real interface and prove it on one row first; plausible output can be wrong. → `operating.md`
**Model allocation** — consider Astra 6/xhigh, Sol 6.1/xhigh, Opus 5.5 and Fable 5.1 using task evidence and fresh capacity. → `operating.md#model-and-account-aware-planning`
**GitHub** — use `gh` and verify DRJ access before pushing; on SSH-agent failure use the
`gh` credential wrapper. → `patterns.md#git-and-github`
**Stacked PRs** — prefer native GitHub stacks for dependent changes; preserve ownership, checks and merge scope. → `operating.md#prefer-github-stacked-pull-requests`
**Branches** — `agent/<slug>-YYYYMMDD`; other prefixes for human-initiated work.
The date suffix is required; use conventional commit subjects. → `patterns.md#branches-and-commits`
**Shell** — use `cp -f`, `mv -f`, `rm -f` to avoid interactive aliases. → `patterns.md#non-interactive-shell-commands`
**SSM** — no `->`, `()`, heredocs, or streaming commands like `journalctl -f`;
append `|| true` so a correct command doesn't report `Status: Failed`.
→ `patterns.md#ssm-command-escaping`

**Docs** — short instruction routers; one canonical runbook per distinct recurring
operation. Link, don't copy; edit in place, retire obsolete duplicates and fix links.
Keep current status compact; PR/CI records hold routine detail. Historical logs are not current posture; runbooks win over drifting agent files.
→ `patterns.md#documentation-structure`

**Agent files** — `AGENTS.md` is canonical and real; `CLAUDE.md` is a git
symlink to it. After creating one, verify `git ls-files -s CLAUDE.md` prints
mode `120000` — a `core.symlinks=false` checkout silently commits a text file
instead and looks fine locally. → `patterns.md#agent-files`

**Roadmaps/docs** — update only for material shifts that merged code, PRs and receipts
don't already document; batch them (a few per day at most), never one docs PR per merge
or milestone. When authorized, publish the browser view and verify the served revision
in Chrome/Chromium. → `operating.md#keep-roadmaps-and-browser-views-current`

**Storybook** — in a repo with a catalog: component change ⇒ story change ⇒ story tests
green ⇒ story ids cited in the PR. → `operating.md#keep-the-component-catalog-current`

**Worktrees/tasks** — agents own linked worktrees; keep each team together in its project workspace.
Leads accept tested-head task receipts before worker cleanup; preserve source/history.
→ `operating.md#lead-owned-task-receipts-and-passive-events`, `#agent-completion-and-workspace-closure`
<!-- DRJ:CONTEXT END -->

