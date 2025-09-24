# Use Graphiti Memory (MCP) to prevent repeated mistakes

Purpose: Persist and reuse project knowledge using the Graphiti MCP server so we avoid re‑debugging the same issues and can quickly apply proven fixes and patterns across repos.

Graphiti server: configured in  as  (SSE). See Graphiti MCP docs for reference: https://github.com/getzep/graphiti/tree/main/mcp_server

Principles
- Prefer durable memory over ephemeral chat. If a finding will help future tasks, store it.
- Retrieve before reinventing. Search memory at the start of a task and before deep debugging.
- Keep entries concise, actionable, and project‑scoped. Avoid secrets and PII.

When to READ from memory
- New task in an existing repo or stack: fetch prior setup, pitfalls, and decisions.
- Build/test failures: fetch prior root causes and fixes for similar errors.
- Tooling/config work (Docker, Vite, Env, CI): look up working configs and known constraints.

When to WRITE to memory
- Root cause and fix verified (tests pass or run succeeds).
- Non‑obvious configuration required (ports, flags, env, versions, ordering).
- Reusable patterns (commands, code idioms, migration steps, rollout plans).
- Decisions with rationale (why a lib/version/approach was chosen).

How to structure entries
- Title: short, searchable (e.g., "Vite HMR conflict on macOS – fix with port 5174").
- Summary: 1–3 sentences describing the problem and solution.
- Repro + Signals: key error lines, symptoms, environment.
- Resolution Steps: exact steps/commands/config edits.
- Scope: repo/project name, component(s), OS, versions.
- Tags: stack/tools (e.g., vite, docker, neo4j, falkordb, graphiti, macos).
- Links: PRs/commits/files.

Metadata conventions
- group_id: use the repository name (e.g., ).
- Files: absolute workspace paths where applicable.
- Do not store secrets, tokens, or private URLs.

Workflow
1) Search memory before changing code or config.
2) If results are found, follow/adapt the steps and attribute the source.
3) After a verified fix, store a new entry or update an existing one (dedupe first).
4) Prefer structured content; include minimal but sufficient logs and diffs.

Examples (intent, not literal commands)
- "Search Graphiti for: Extended build capabilities with BuildKit

Usage:  docker buildx [OPTIONS] COMMAND

Extended build capabilities with BuildKit

Options:
      --builder string   Override the configured builder instance
  -D, --debug            Enable debug logging

Management Commands:
  history     Commands to work on build records
  imagetools  Commands to work on images in registry

Commands:
  bake        Build from a file
  build       Start a build
  create      Create a new builder instance
  dial-stdio  Proxy current stdio streams to builder instance
  du          Disk usage
  inspect     Inspect current builder instance
  ls          List builder instances
  prune       Remove build cache
  rm          Remove one or more builder instances
  stop        Stop builder instance
  use         Set the current builder instance
  version     Show buildx version information

Run 'docker buildx COMMAND --help' for more information on a command.

Experimental commands and flags are hidden. Set BUILDX_EXPERIMENTAL=1 to show them. in group "
- "Add memory entry: Title 'FalkorDB on custom port 63790', Summary, Steps, Files edited, Tags [falkordb, docker, ports], group "

Quality bar
- Must be reproducible by a teammate from the note alone.
- Include exact commands/flags and key file edits.
- Keep noise out: omit lengthy logs; keep only essential snippets.

Safety
- Strip secrets and personal data.
- Redact hostnames or internal URLs unless approved for sharing.

Fallbacks
- If Graphiti is unreachable, proceed but note to backfill memory once available.
