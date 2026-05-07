# Sync, Preserve, Implement First-Class Image Builds

## Summary
Preserve the ARM `feature/image-build-support` commit on the laptop, write this implementation plan to `commit.md`, then implement first-class root-level image management on top of that commit.

## Execution Steps
- Sync ARM commit to laptop without copying dirty ARM worktree changes:
  ```bash
  git -C /Users/pcoiner/dev/containerlab fetch \
    ssh://opc@129.213.18.176/home/opc/containerlab \
    feature/image-build-support:refs/heads/feature/image-build-support
  ```
- Create implementation branch from the synced commit:
  ```bash
  git -C /Users/pcoiner/dev/containerlab checkout -b codex/first-class-image-management feature/image-build-support
  ```
- Save this plan in `/Users/pcoiner/dev/containerlab/commit.md` so the intent survives context loss.

## Implementation Changes
- Add root-level `images` to the lab config beside `name`, `mgmt`, `settings`, and `topology`.
  - Each image target has logical key, `image`, and optional `build`.
  - `build` reuses the ARM branch fields: `mode`, `rebuild`, `context`, `dockerfile`, `network`, `builder`, `commit`, plus `node` for topology-mode binding.
- Keep nodes as runtime objects with one explicit resolved `image`.
  - Node-level `build` is allowed only as shorthand.
  - A node-level `build` requires the node to declare/inherit `image`.
  - Internally, node-level build becomes an anonymous managed image target for that node image.
- Build resolution:
  - Build managed images before normal node image pulls.
  - Match nodes to root `images` by output image tag.
  - `pre-deploy` builds run directly through Docker build.
  - `topology` builds run using the selected node’s topology position.
  - `build.node` explicitly selects the node; node shorthand uses itself; root topology target without `node` may infer only if exactly one node consumes the image.
- Validation:
  - Duplicate output image tags must have identical definitions or fail.
  - `topology` mode requires `builder.image` and `builder.cmd`.
  - `build.node` must reference an existing node.
  - Ambiguous topology target without `node` fails clearly.

## Test Plan
- Unit tests:
  - Parse root `images`.
  - Parse node-level `build` shorthand with `rebuild`.
  - Validate duplicate/conflicting image targets.
  - Validate topology-mode node binding and ambiguity errors.
  - Preserve existing ARM image-build tests.
- Runtime smoke on ARM:
  - Shared `cm-base` pre-deploy image builds once.
  - `cm-card-01` root image target builds with `node: card-01`.
  - `card-02` node-level shorthand builds its explicit image.
  - Final nodes start from their declared images.
- Validation commands:
  ```bash
  go test ./types ./runtime ./runtime/docker ./core
  go build -o /tmp/clab-first-class-images .
  ```
  Then sync to ARM and run the topology smoke against Docker there.

## Assumptions
- Work starts from synced commit `65f6228eb Add image build lifecycle support`.
- Dirty ARM systemd/lifecycle/netavark changes are not part of this sync or implementation branch.
- `images` is a first-class root section, not nested under `topology`.
- Podman remains explicitly unsupported for build/commit in this pass.
