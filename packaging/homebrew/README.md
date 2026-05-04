# Homebrew Packaging

This directory stores generated Homebrew formula files for released versions of `cliai`.

Typical workflow:

1. Publish the GitHub Release assets for macOS and optionally Linux.
2. Run `./scripts/new-homebrew-formula.sh` with the release URLs and SHA256 values.
3. Commit the generated `packaging/homebrew/<version>/cliai.rb` file.
4. Copy or submit that formula to your dedicated Homebrew tap repository.

The repository does not assume a specific tap name. A common layout is:

- `<owner>/homebrew-tap`

End-user install command is typically:

```bash
brew install <owner>/tap/cliai
```
