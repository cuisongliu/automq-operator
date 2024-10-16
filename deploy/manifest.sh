#!/bin/bash
IMAGE_NAME=${1:-"ghcr.io/cuisongliu/automq-operator:latest"}
sudo sealos push "${IMAGE_NAME}"-amd64
sudo sealos push "${IMAGE_NAME}"-arm64
sudo sealos images
sudo sealos manifest create "${IMAGE_NAME}"
sudo sealos manifest add "$IMAGE_NAME" docker://"$IMAGE_NAME-amd64"
sudo sealos manifest add "$IMAGE_NAME" docker://"$IMAGE_NAME-arm64"
sudo sealos manifest push --all "$IMAGE_NAME" docker://"$IMAGE_NAME" && echo "$IMAGE_NAME push success"
sudo sealos images
