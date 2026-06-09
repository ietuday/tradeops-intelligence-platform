# Final Validation Checklist

Use this before tagging the public portfolio release.

- [ ] Repository cleanup done.
- [ ] No real `.env` files committed.
- [ ] `.gitignore` reviewed.
- [ ] `docker compose -f infrastructure/docker/docker-compose.yml config` passes.
- [ ] All Bash scripts validate with `bash -n`.
- [ ] Go tests pass where service code changed.
- [ ] Node/API Gateway tests pass where gateway code changed.
- [ ] Python tests pass where Python service code changed.
- [ ] Grafana dashboard JSON reviewed.
- [ ] `./scripts/validate-helm.sh` passes or skips gracefully when Helm is unavailable.
- [ ] README links reviewed.
- [ ] Release notes complete through `v2.0.0`.
- [ ] Demo scripts reviewed.
- [ ] Screenshot guide reviewed and screenshots captured if desired.
- [ ] Secrets check reviewed with `./scripts/security-check.sh`.

## Git Tag

```bash
git tag -a v2.0.0 -m "v2.0.0 Final Portfolio Release"
git push origin v2.0.0
```
