# lotus-sign



离线签名 转账 修改owner地址 矿工提现

使用
export KEYS="01234567890123456789012345678901" 默认私钥

自定义私钥
openssl rand -hex 16
5a5c9f7f6da302d2c82f23e34159434d

export KEYS=5a5c9f7f6da302d2c82f23e34159434d

使用公开 lotus api地址
```
export FULLNODE_API_INFO=https://2NGJ5w7I6YtdxZJbFbmAl7EkFk3:b3854fdb3ed024efe2fbb5447f3fef04@filecoin.infura.io
```
钱包地址私钥本地离线存储 ~/.lotus/keystore 目录里面 公共infura api 签名消息上链
系统要求环境是linux系统

## 基础系统环境


### Basic Build Instructions

-----------

System-specific Software Dependencies:

Building Lotus requires some system dependencies, usually provided by your distribution.

Ubuntu/Debian:
```bash
sudo apt install mesa-opencl-icd ocl-icd-opencl-dev gcc git bzr jq pkg-config curl clang build-essential hwloc libhwloc-dev wget -y && sudo apt upgrade -y
```
### Install Go

## Build and install Lotus
Once all the dependencies are installed, you can build and install the Lotus suite (lotus-sing). 
  
 : 1 Clone the repository:

```bash
git clone https://proxy.jeongen.com/https://github.com/wxshope/lotus-sign.git
cd lotus-sign
```

To build filecoin-lotus-sign, you need to install [Go 1.20.02 or higher](https://golang.org/dl/):

```shell
wget -c https://golang.org/dl/go1.20.02.linux-amd64.tar.gz -O - | sudo tar -xz -C /usr/local
```

### Build

1. To build you need to download some Go modules. These are usually hosted on Github, which has a low bandwidth from China. To work around this, use a local proxy before running by setting the following variables.

```shell
export GOPROXY=https://goproxy.cn
```

2. Depending on your CPU model, you will want to export additional environment variables:

a. If you have **an AMD Zen or Intel Ice Lake CPU (or later)**, enable the use of SHA extensions by adding these two environment variables:

```shell
export RUSTFLAGS="-C target-cpu=native -g"
export FFI_BUILD_FROM_SOURCE=1
```

b. Some older Intel and AMD processors without the ADX instruction support may panic with illegal instruction errors. To solve this, add the `CGO_CFLAGS` environment variable:

```shell
export CGO_CFLAGS_ALLOW="-D__BLST_PORTABLE__"
export CGO_CFLAGS="-D__BLST_PORTABLE__"
```


d.An experimental `gpu` option using CUDA can be used in the proofs library. This feature is disabled by default (opencl is the default, when FFI_USE_GPU=1 is set.). To enable building with the `gpu` CUDA dependency, set FFI_USE_CUDA=1 when building from source.
```shell
export FFI_USE_CUDA=1
```

3. Build and install

```shell
make clean all
sudo make install
```

Install `lotus-sign` to `/usr/local/bin`

4. Once the installation is complete, use the following command to ensure that the installation is successful for the correct network.



```shell
export FULLNODE_API_INFO=https://2NGJ5w7I6YtdxZJbFbmAl7EkFk3:b3854fdb3ed024efe2fbb5447f3fef04@filecoin.infura.io
```





#### Export sector metadata tool:
```shell
lotus-sign 
```






## License

Licensed under [Apache 2.0](https://github.com/wxshope/lotus-sign/blob/main/LICENSE)


