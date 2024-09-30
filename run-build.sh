#!/bin/bash

# Check if the last commit has a tag
if git describe --exact-match --tags $(git rev-parse HEAD) > /dev/null 2>&1; then
    echo "The last commit is tagged."
    VERSION_TAG=$(git describe --tags $(git rev-parse HEAD))
    IMAGE=$(ko publish --platform=all --bare --tags=${TAG} | tail -n 1)
    DIGEST=$(echo "$IMAGE" | awk -F '@' '{print $2}')

    IMAGE_TAG=$VERSION_TAG'@'$DIGEST

    yq eval '.image.tag = "'$IMAGE_TAG'"' -i helm/aks-spot-instance-tolerator/values.yaml

    GIT_COMMIT=$(git rev-parse HEAD)

    CHART_VERSION=$VERSION_TAG'-'$GIT_COMMIT
    yq eval '.version = "'$CHART_VERSION'"' -i helm/aks-spot-instance-tolerator/Chart.yaml


    echo "Commit was tagged, adding image tag to chart values: $TAG"
else
    IMAGE=$(ko publish --platform=all --bare | tail -n 1)
    TAG=@$(echo "$IMAGE" | awk -F '@' '{print $2}')

    echo "The last commit is not tagged. Not bumping image of chart"
fi


echo "Image published: $IMAGE"
echo "Image published: $TAG"



