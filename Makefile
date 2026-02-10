build:
	go build -o fwdx .

run:
	go run .

test:
	FWDX_SKIP_DOCKER_E2E=1 go test ./...

test-docker:
	cd tests/docker && sh scripts/run-e2e.sh
	go test ./tests/... -run TestE2E_Docker -v

clean:
	rm -f fwdx

watch:
	air