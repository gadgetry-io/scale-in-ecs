VERSION=latest

zip: build
	zip scale-in-ecs.zip scale-in-ecs

build:
	go build