# Autoresearch: Refine SFT vocabs and schema — max coverage, min vocabs

## Config
- **Benchmark**: `bash benchmark.sh`
- **Target metric**: `vocab_count` (lower is better), `coverage_pct` must stay at 100
- **Scope**: `README.md` + `examples/*.sft.yaml`
- **Branch**: `autoresearch/refine-vocabs-schema-max-coverage-min-vocabs`
- **Started**: 2026-03-16T00:00:00Z

## Rules
1. One change per experiment
2. Run benchmark after every change
3. Keep if vocab_count decreases AND coverage_pct stays at 100, discard otherwise
4. Log every run to autoresearch.jsonl
5. Commit kept changes with `Result:` trailer

## Current vocab (19 keywords)
action, app, apps, contains, description, events, flows, from, name, on, regions, screens, sequence, state_carries, states, tags, to, transitions, values
