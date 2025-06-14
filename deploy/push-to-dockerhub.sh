#!/bin/bash
set -e

VERSION=$(cat ../VERSION)
DOCKER_REPO="buru2020/one-mcp"

echo "Building and tagging images..."
docker build -t one-mcp:$VERSION ../
docker tag one-mcp:$VERSION one-mcp:latest
docker tag one-mcp:latest $DOCKER_REPO:latest
docker tag one-mcp:$VERSION $DOCKER_REPO:$VERSION

echo "Pushing to Docker Hub..."
docker push $DOCKER_REPO:latest
docker push $DOCKER_REPO:$VERSION

echo "Successfully pushed to Docker Hub!"
echo "Latest: https://hub.docker.com/r/$DOCKER_REPO/tags"
echo "You can now pull the image with:"
echo "  docker pull $DOCKER_REPO:latest"
echo "  docker pull $DOCKER_REPO:$VERSION" 