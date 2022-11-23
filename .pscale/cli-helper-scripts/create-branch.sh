#!/bin/bash

# https://planetscale.com/blog/using-the-planetscale-cli-with-github-actions-workflows

. use-pscale-docker-image.sh
. wait-for-branch-readiness.sh

. authenticate-ps.sh

BRANCH_NAME="$1"

. ps-create-helper-functions.sh
create-db-branch "$MEWST_DATABASE_NAME" "$MEWST_BRANCH_NAME" "$MEWST_PLANETSCALE_ORG_NAME"
