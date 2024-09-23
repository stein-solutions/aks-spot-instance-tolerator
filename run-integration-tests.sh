#!/bin/bash

export KO_DOCKER_REPO="mariusstein77/aks-spot-instance-tolerater-2bd6b0028c6413dcf3a1c311ebc3369c"

output=$(ko build --platform=all --bare)
image_hash=$(echo "$output" | awk -F '@' '{print $2}')

export AKS_SPOT_INSTANCE_TOLERATER_ITEST_IMAGE="mariusstein77/aks-spot-instance-tolerater-2bd6b0028c6413dcf3a1c311ebc3369c@"$image_hash
echo $AKS_SPOT_INSTANCE_TOLERATER_ITEST_IMAGE
go test -v ./test/integration/... -count=1 -tags=integration