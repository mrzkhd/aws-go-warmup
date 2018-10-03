build:
	dep ensure -v
	env GOOS=linux go build -ldflags="-s -w" -o bin/req1 req1/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/req2 req2/main.go

.PHONY: clean
clean:
	rm -rf ./bin ./vendor Gopkg.lock

.PHONY: deploy
deploy: clean build
	sls deploy --verbose
