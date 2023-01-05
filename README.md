# Popov

* 使用golang正则，提取文章目录结构的实现，即 h1~h6 标签的信息。
* 支持go.mod方式

## 安装

```shell
go get -u github.com/zituocn/popov
```

## 运行测试代码

popov_test.go

```sh
go test  -run ^TestPopov$
```


测试文本地址：

> https://22v.net/article/3263/


运行结果：

```shell
Kubernetes v1.19.16 二进制高可用部署
├─ 一、基础环境
├─ ├─ 服务器环境
├─ ├─ 软件及版本
├─ ├─ 约定值
├─ ├─ 系统设置(所有机器)
├─ 二、准备二进制文件(所有机器)
├─ ├─ 2.1 配置免密登录
├─ ├─ 2.2 下载二进制文件
├─ ├─ ├─ 下载和整理k8s文件
├─ ├─ ├─ 下载和整理etcd文件
├─ ├─ 2.3 分发二进制文件到其他机器
├─ 三、集群部署
├─ ├─ 3.1 安装cfssl证书工具
├─ ├─ 3.2 生成 kubernetes 所需的根证书
├─ ├─ 3.3 在master节点部署etcd集群
├─ ├─ ├─ 3.3.1 生成etcd所需的私钥和证书
├─ ├─ ├─ 3.3.2 创建etcd的systemd服务文件
├─ ├─ ├─ 3.3.3 启动服务
├─ ├─ 3.4 在master节点部署 kube-apiserver
├─ ├─ ├─ 3.4.1 生成所需的私钥和证书
├─ ├─ ├─ 3.4.2 创建kube-apiserver的systemd服务文件
├─ ├─ ├─ 3.4.3 启动服务
├─ ├─ 3.5 安装 keepalived
├─ ├─ 3.6 安装和配置haproxy
├─ ├─ 3.7 安装kubectl
├─ ├─ ├─ 3.7.1 创建所需的私钥和证书
├─ ├─ ├─ 3.7.2 创建kubeconfig配置文件
├─ ├─ ├─ 3.7.3 授权 kubernetes访问 kubelet API的权限
├─ ├─ ├─ 3.7.4 测试kubectl可用
├─ ├─ 3.8 部署 kube-controller-manager
├─ ├─ ├─ 3.8.1 创建私钥和证书
├─ ├─ ├─ 3.8.2 创建controller-manager的kubeconfig
├─ ├─ ├─ 3.8.3 创建 kube-controller-manager的systemd启动文件
├─ ├─ 3.9 部署 kube scheduler
├─ ├─ ├─ 3.9.1 创建私钥和证书
├─ ├─ ├─ 3.9.2 创建kube scheduler的kubeconfig
├─ ├─ ├─ 3.9.3 创建 kube-scheduler的systemd启动文件
├─ ├─ ├─ 3.9.4 启动kube-scheduler服务
├─ ├─ 3.10 部署kubelet
├─ ├─ ├─ 3.10.1 创建bootstrap配置文件
├─ ├─ ├─ 3.10.2 创建kubelet配置文件
├─ ├─ ├─ 3.10.3 创建kubelet的systemd启动文件
├─ ├─ ├─ 3.10.4 启动kubelet服务
├─ ├─ ├─ 3.10.5 加入集群
├─ ├─ 3.11 部署 kube-proxy 服务
├─ ├─ ├─ 3.11.1 创建私钥和证书
├─ ├─ ├─ 3.11.2 创建和分发kube-proxy配置文件
├─ ├─ ├─ 3.11.3 创建和分发kube-proxy的systemd服务文件
├─ ├─ ├─ 3.11.4 启动kube-proxy服务
├─ ├─ 3.12 部署CNI网络插件
├─ ├─ 3.13 部署DNS插件 coredns
├─ 结尾
```

## 感谢

代码中的strip.go，原始地址：

> https://github.com/grokify/html-strip-tags-go/blob/master/strip.go
