# Project skills

Task skills for this repository. Load the `SKILL.md` whose description matches the work.

| Skill | Use it for |
|---|---|
| [add-endpoint](add-endpoint/SKILL.md) | Adding a file under `api/`, a `lib.Handle` adapter, test, spec path, or example request |
| [update-api-docs](update-api-docs/SKILL.md) | Keeping README, guides, and OpenAPI in sync with the code and the `lib/` contract |
| [verify-changes](verify-changes/SKILL.md) | Running the full local gate before finishing |
| [deploy-vercel](deploy-vercel/SKILL.md) | Preview and production deploys, plus deployed 404 and 500 behavior |
| [port-to-host](port-to-host/SKILL.md) | Docker, Fly.io, Render, Cloud Run, Railway, and Lambda |

These describe the current Vercel per-file function mode: every `.go` file in `api/` is its own function, and all routes live under `/api`. JSON endpoints share the request, response, and error contract in `lib/`; update these skills in the same change when that contract or the deployment setup changes.
