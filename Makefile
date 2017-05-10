build:
	CGO_ENABLED=0 go build

docker: build
	docker build -t digibib/splitteverk:$(shell git rev-parse HEAD) .

push:
	docker push digibib/splitteverk:$(shell git rev-parse HEAD)