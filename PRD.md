# PRD: Per-Environment API Key Env Var Selection

## Background
CCE manages multiple environments for Claude Code. Historically it always exported the API key as `ANTHROPIC_API_KEY`. New provider configurations require choosing between `ANTHROPIC_API_KEY` and `ANTHROPIC_AUTH_TOKEN` per environment.

## Goals
- Allow each environment to choose the API key environment variable name.
- Default to `ANTHROPIC_API_KEY` for backward compatibility.
- Persist the choice in `~/.claude-code-env/config.json`.
- At launch, set only the selected key variable name (not both).
- Display the chosen variable name in `cce list`.

## Non-Goals
- No simultaneous export of both variables.
- No global switch; this is per environment.
- No new CLI flag to override per-run (can be added later if needed).

## Requirements
- Valid values for the key variable name: `ANTHROPIC_API_KEY` (default), `ANTHROPIC_AUTH_TOKEN`.
- If field missing or empty, treat as `ANTHROPIC_API_KEY`.
- Keep existing behavior for `ANTHROPIC_BASE_URL` and optional `ANTHROPIC_MODEL`.
- Continue filtering inherited `ANTHROPIC_*` environment variables to avoid conflicts.
- API key validation must be provider-agnostic (length and safety only).

## UX
- `cce add`: add a prompt to choose the variable name (1=API_KEY default, 2=AUTH_TOKEN).
- `cce list`: display a new line `Key Var: <value>`.

## Data Model
- `Environment` struct adds `APIKeyEnv string \`json:"api_key_env,omitempty"\``.
- Validation `validateAPIKeyEnv` accepts empty, `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`.

## Behavior
- In launcher `prepareEnvironment`, compute `keyVar := env.APIKeyEnv` (default to `ANTHROPIC_API_KEY`) and set `keyVar=APIKey`.
- Do not set the other key variable.

## Compatibility
- Existing configs without `api_key_env` continue to work (default to `ANTHROPIC_API_KEY`).
- API key validation is relaxed (no Anthropic substring requirement) for third-party providers.

## Testing
- Unit tests for `validateAPIKeyEnv` and launcher behavior when `ANTHROPIC_AUTH_TOKEN` is chosen.
- All pre-existing tests remain valid.

## Documentation
- README and README_zh updated: add config examples, list output with `Key Var`, and add-step prompt.

