# Original:
# https://github.com/jonico/pscale-workflow-helper-scripts/blob/4ceba75919d259abc513fcf38285fcca3f3e6abb/.pscale/cli-helper-scripts/authenticate-ps.sh

# if MEWST_PLANETSCALE_SERVICE_TOKEN is not set, use pscale auth login
if [ -z "$MEWST_PLANETSCALE_SERVICE_TOKEN" ]; then
    echo "Going to authenticate PlanetScale CLI, please follow the link displayed in your browser and confirm ..."
    pscale auth login
    # if command failed, exit
    if [ $? -ne 0 ]; then
        echo "pscale auth login failed, please try again"
        exit 1
    fi
fi
