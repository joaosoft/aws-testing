fmt:
	go fmt ./...

vet:
	go vet ./*

gometalinter:
	gometalinter ./*

init:
	docker-compose up -d aws.dynamodb