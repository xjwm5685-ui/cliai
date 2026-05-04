# Debian Packaging

This directory stores generated Debian package staging trees and `.deb` artifacts for released versions of `cliai`.

Typical workflow:

1. Build the Linux release binary, for example `dist/linux-amd64/cliai`.
2. Run `./scripts/new-deb-package.sh --version <version> --arch amd64`.
3. On a Linux host with `dpkg-deb`, the script also emits a `.deb` package.
4. Without `dpkg-deb`, use `--stage-only` to prepare the Debian staging tree first.

Generated layout typically looks like:

- `packaging/deb/<version>/<arch>/stage/<package>_<version>_<arch>/`
- `packaging/deb/<version>/<arch>/<package>_<version>_<arch>.deb`

The default package name is `cliai`.
