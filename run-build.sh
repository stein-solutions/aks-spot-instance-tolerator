#!/bin/bash

# Check if the last commit has a tag
if git describe --exact-match --tags $(git rev-parse HEAD) > /dev/null 2>&1; then
    VERSION_TAG=$(git describe --tags $(git rev-parse HEAD))
    IMAGE=$(ko build --platform=all --bare --tags=${TAG} | tail -n 1)
    echo "Image published: $IMAGE"
else
    IMAGE=$(ko build --platform=all --bare | tail -n 1)
    echo "The last commit is not tagged. Not bumping image of chart"
    echo "Image published: $IMAGE"

fi
