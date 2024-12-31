
#DOCKER_FILE:=Dockerfile
DOCKER_FILE:=Dockerfile-ubuntu
CUR_TIME:= $(shell date +"%Y%m%d%H%M")
IMAGE_VERSION:=v$(CUR_TIME)
#IMAGE_NAME:=centos/chainmaker-ca/app-server:$(IMAGE_VERSION)
IMAGE_NAME:=ubuntu/chainmaker-ca/app-server:$(IMAGE_VERSION)


build:
	@echo "Dockerfile : $(DOCKER_FILE)"
	@echo "Current Time: $(CUR_TIME)"
	@echo "Image Version: $(IMAGE_VERSION)"
	@echo "Image Name: $(IMAGE_NAME)"
	@echo "Building Image..."
	docker buildx build -t "$(IMAGE_NAME)"  --progress auto --no-cache -f "$(DOCKER_FILE)" .
	@echo "Image Name: $(IMAGE_NAME)"
	@echo "done"
