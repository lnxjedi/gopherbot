# Proposal 1 Slice 0 Baseline

Date: 2026-02-19
Branch: `codex/simplify-help-metadata`

## Commands Run

```bash
go test ./bot -run 'Test(ScoreHelpCommandMatch|RankHelpMatches|FirstHelpLineUsageAndSummary|StripHelpAddressPrefix|HiddenSlashBotExample|CommandAllowsHidden)'
go test ./bot -run 'TestValidateYAMLPlugin(RejectsLegacyHelpKey|RejectsLegacyCommandMatchersKey|AcceptsCommandsKey)'
```

## Results

Both commands passed:

- `ok github.com/lnxjedi/gopherbot/v2/bot 0.005s`
- `ok github.com/lnxjedi/gopherbot/v2/bot 0.005s`

