#!/bin/bash

export KO_DOCKER_REPO="mariusstein77/aks-spot-instance-tolerater"

# Check if the last commit has a tag
if git describe --exact-match --tags $(git rev-parse HEAD) > /dev/null 2>&1; then
    echo "The last commit is tagged."
    TAG=$(git describe --tags $(git rev-parse HEAD))
    IMAGE=$(ko publish --platform=all --bare --tags=${TAG} | tail -n 1)

    yq eval '.image.tag = "'$TAG'"' -i helm/aks-spot-instance-tolerator/values.yaml 

    echo "Commit was tagged, adding image tag to chart values: $TAG"
else
    IMAGE=$(ko publish --platform=all --bare | tail -n 1)
    TAG=@$(echo "$IMAGE" | awk -F '@' '{print $2}')

    echo "The last commit is not tagged. Not bumping image of chart"
fi


echo "Image published: $IMAGE"
echo "Image published: $TAG"



