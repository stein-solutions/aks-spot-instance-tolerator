#!/bin/bash

export KO_DOCKER_REPO="mariusstein77/aks-spot-instance-tolerater"

output=$(ko build --platform=all --bare)
image_hash=$(echo "$output" | awk -F '@' '{print $2}')

export AKS_SPOT_INSTANCE_TOLERATER_ITEST_IMAGE="mariusstein77/aks-spot-instance-tolerater@"$image_hash
echo $AKS_SPOT_INSTANCE_TOLERATER_ITEST_IMAGE
go test -v ./test/integration/... -count=1 -tags=integration