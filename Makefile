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

# Generate Go from api/tunnel/v1/tunnel.proto. Requires: protoc, protoc-gen-go, protoc-gen-go-grpc.
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/tunnel/v1/tunnel.proto

watch:
	air