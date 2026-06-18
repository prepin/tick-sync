# Releases

Releases are handled by GoReleaser from GitHub Actions when a version tag is pushed.

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow creates a GitHub Release for the tag and publishes a multi-architecture Docker image to GitHub Container Registry:

```sh
docker pull ghcr.io/prepin/tick-sync:v0.1.0
docker pull ghcr.io/prepin/tick-sync:latest
```

Images support `linux/amd64` and `linux/arm64`.

Local checks:

```sh
goreleaser check
goreleaser build --snapshot --clean
```
