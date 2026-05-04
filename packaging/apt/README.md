# APT Repository Packaging

This directory stores generated apt repository trees for released Debian packages of `cliai`.

Typical workflow:

1. Build `.deb` files with `./scripts/new-deb-package.sh`.
2. Run `./scripts/new-apt-repo.sh --repo-root packaging/apt/<version> --deb <path/to/package.deb> ...`.
3. Upload the generated `pool/` and `dists/` directories to your apt repository host.
4. Run `./scripts/sign-apt-repo.sh --repo-root packaging/apt/<version> --require-signature` to generate `Release.gpg` and `InRelease`.
5. Run `./scripts/export-apt-public-key.sh --output cliai-archive-keyring.asc` and publish that file alongside the repository.

The generated repository layout includes:

- `pool/main/c/cliai/*.deb`
- `dists/stable/main/binary-amd64/Packages`
- `dists/stable/main/binary-amd64/Packages.gz`
- `dists/stable/Release`
- `dists/stable/Release.gpg`
- `dists/stable/InRelease`
- `cliai-archive-keyring.asc`

The signing step requires a GPG secret key, provided either as a file or through environment variables.
