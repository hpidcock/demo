#!/bin/sh
set -ex

for f in *.yaml; do
    cat "${f}" | sed "s/{{GIT_COMMIT}}/$(git rev-parse HEAD)/g" | kubectl apply -f -
done
