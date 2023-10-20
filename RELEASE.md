# Release

This project uses [Go Releaser](https://goreleaser.com/) CLI for releasing.
This document is mostly for my reference, so I don't forget how to do it next time.

1. Update Changelog
2. Update `main.go` version
3. Update dependencies `go get -t -u=patch ./... && go mod tidy`
4. Commit `git add . && git commit -am "release: vX.X.X" && git push`
5. Tag `git tag -s vX.X.X`
6. Push Tag `git push origin vX.X.X`
7. Create Release `GITHUB_TOKEN=$(pass github/token) make release`
