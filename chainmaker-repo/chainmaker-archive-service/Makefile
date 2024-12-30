ut:
	./ut_cover.sh

lint:
	golangci-lint run ./...


DOCKERFILE:=Dockerfile
CUR_TIME:= $(shell date +"%Y%m%d%H%M")
IMAGE_VERSION:=v$(CUR_TIME)
IMAGE_NAME:=ubuntu/archivecenter/app-server:$(IMAGE_VERSION)

build:
	docker buildx build -t "$(IMAGE_NAME)" --no-cache -f "$(DOCKERFILE)" .
	@echo "Image Name: $(IMAGE_NAME)"
	@echo "done"
