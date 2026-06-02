# Comparing Runs

`iterforge compare` diffs two experiment records and recommends keep vs promote. It
reads the results log only — it never writes.

```bash
make compare                                  # best run vs latest run
make compare BASELINE=<id> CANDIDATE=<id>     # explicit ids
go run ./cmd/iterforge compare -baseline <id> -candidate <id>
```

Defaults: baseline = best run (per score direction), candidate = latest run.
Score direction and `minimum_delta` come from `policy.yaml`.

## Output

- baseline / candidate ids, scores, gate status, decision;
- deltas (candidate − baseline) for score, accuracy, latency, failed, duration;
- candidate failure reasons, if any;
- a recommendation.

## Recommendations

| Recommendation | When |
|----------------|------|
| `promote candidate` | candidate passed gates and improved beyond `minimum_delta` |
| `keep baseline: candidate regressed` | candidate is worse beyond `minimum_delta` |
| `no change: within minimum_delta, keep baseline` | change is below the threshold |
| `reject candidate: <decision>` | candidate hit a guardrail (frozen change, tamper, hardcoding) |
| `reject candidate: hard gates failed` | candidate failed an eval gate |

## Exit code

`0` only when the recommendation is `promote candidate`; otherwise `3`. This lets
scripts gate promotion on the comparison. (`make compare` surfaces the non-zero
exit as a make error — that is expected when the recommendation is not promote.)
