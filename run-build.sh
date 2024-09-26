#!/bin/bash

export KO_DOCKER_REPO="mariusstein77/aks-spot-instance-tolerater"

# Check if the last commit has a tag
if git describe --exact-match --tags $(git rev-parse HEAD) > /dev/null 2>&1; then
    echo "The last commit is tagged."
    TAG=$(git describe --tags $(git rev-parse HEAD))
    echo "Tag: $TAG"
else
    TAG=$(git rev-parse HEAD)
    echo "The last commit is not tagged."
fi

ko build --platform=all --bare --tags=${TAG}