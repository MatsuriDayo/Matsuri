source "buildScript/init/env.sh"
source "buildScript/plugin/trojan_go/build.sh"

git reset HEAD --hard
git clean -fdx
