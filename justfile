# this is the default so that when you run `just` you get a nice interface
# for selecting a command
chose-a-command:
	just --choose

pre-pr: lint test

lint:
    golangci-lint run --config golangci.yaml

test:
    go test -v ./...

update-abi-testdata:
    nix run github:blocky/abi-testdata -- --package-name=abi_test > abitestdata_test.go
