source "buildScript/init/env.sh"
source "buildScript/plugin/wireguard/build.sh"

git reset HEAD --hard
git clean -fdx
