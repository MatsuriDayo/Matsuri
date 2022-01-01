COMMIT=$(cat libcore/core_commit.txt)

cd ..
[ -d v2ray-core ] && exit 0
rm -rf v2ray-core
git clone --no-checkout https://github.com/MatsuriDayo/v2ray-core.git
cd v2ray-core
git checkout $COMMIT
