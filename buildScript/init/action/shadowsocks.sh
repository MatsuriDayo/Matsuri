#!/usr/bin/env bash

curl --proto '=https' --tlsv1.2 -sSf https://sh.#rustup.rs | sh -s -- --default-toolchain none -y
echo "source \$HOME/.cargo/env" >>$HOME/.bashrc

bash "buildScript/zipVersion/downloadZip.sh" library/shadowsocks/src/main/rust buildScript/zipVersion/shadowsocks_name buildScript/zipVersion/shadowsocks_status

cd library/shadowsocks/src/main/rust/shadowsocks-rust
rustup target install armv7-linux-androideabi aarch64-linux-android i686-linux-android x86_64-linux-android

# rustup default $(cat rust-toolchain)