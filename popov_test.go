package popov

import (
	"fmt"
	"testing"
)

var (
	content = `
	<div class="markdown-body">
	<h1 id="h1-kubernetes-v1-19-16-"><a name="Kubernetes v1.19.16 二进制高可用部署" class="reference-link"></a><span class="header-link octicon octicon-link"></span>Kubernetes v1.19.16 二进制高可用部署</h1><pre><code>1. 此文适合于有一定Linux基础的同学阅读；
2. 基于 centos 7.x 完成，其他Linux操作系统，请自行调整；
3. 阿里云平台，不能使用keepalive来实现高可用，请使用阿里云上内网LBS或虚拟IP(VIP)来实现；
</code></pre><h2 id="h2--"><a name="一、基础环境" class="reference-link"></a><span class="header-link octicon octicon-link"></span>一、基础环境</h2><h3 id="h3-u670Du52A1u5668u73AFu5883"><a name="服务器环境" class="reference-link"></a><span class="header-link octicon octicon-link"></span>服务器环境</h3><p>5台 CentOS7.x 虚拟机，在MacOS下使用 <code>Parallels Desktop</code> 完成创建</p>
<pre class=" language-sh"><code class=" language-sh">192.168.0.150    # master节点 2C4
192.168.0.151    # master节点 2C4
192.168.0.152    # master节点 2C4
192.168.0.153    # 工作节点 2C4
192.168.0.154    # 工作节点 2C4
</code></pre>
<p><em>注意</em></p>
<pre class=" language-sh"><code class=" language-sh">生产环境时，master节点建议使用4C8的配置
</code></pre>
<h3 id="h3-u8F6Fu4EF6u53CAu7248u672C"><a name="软件及版本" class="reference-link"></a><span class="header-link octicon octicon-link"></span>软件及版本</h3><ul>
<li>kubernetes 1.19.16</li><li>etcd 3.4.18</li><li>calico 3.16.0</li><li>cfssl 1.2.0 (证书工具)</li><li>keepalive (虚拟IP)</li><li>haproxy (高可用)</li><li>coredns 1.7.0 (docker image)</li><li>pause 3.2  (docker image)</li></ul>
<h3 id="h3-u7EA6u5B9Au503C"><a name="约定值" class="reference-link"></a><span class="header-link octicon octicon-link"></span>约定值</h3><pre class=" language-sh"><code class=" language-sh">
# kubernetes服务的ip网段
10.255.0.0/16

# k8s的api-server的服务ip
10.255.0.1

# dns服务的ip地址
10.255.0.2

# pod网段
172.23.0.0/16

# 虚拟IP (VIP)
192.168.0.160


# VIP代理后的IP及端口
192.168.0.160:8443

# node port range
30000-32767
</code></pre>
<h3 id="h3--"><a name="系统设置(所有机器)" class="reference-link"></a><span class="header-link octicon octicon-link"></span>系统设置(所有机器)</h3><p>1、设置hostname，后面会使用hostname进行通信：</p>
<pre class=" language-sh"><code class=" language-sh"># 可分别设置每台机器的hostname
$ hostnamectl set-hostname master1
</code></pre>
<p>配置hosts</p>
<pre class=" language-sh"><code class=" language-sh">$ vi /etc/hosts

192.168.0.150    master1
192.168.0.151    master2
192.168.0.152    master3
192.168.0.153    node1
192.168.0.154    node2
192.168.0.160    vip
</code></pre>
<p>2、安装一些基础软件</p>
<pre class=" language-sh"><code class=" language-sh"># 更新yum
$ yum update -y 

# 安装一些包
$ yum install -y conntrack ipvsadm ipset jq sysstat curl wget iptables libseccomp
</code></pre>
<p>3、系统设置</p>
<pre class=" language-sh"><code class=" language-sh">
# 关闭防火墙
$ systemctl stop firewalld &amp;&amp; systemctl disable firewalld


# 关闭swap-交换分区
$ swapoff -a
$ sed -i '/swap/s/^\(.*\)$/#\1/g' /etc/fstab

# 关闭selinux
$ setenforce 0
$ sed -i "s/SELINUX=enforcing/SELINUX=disabled/g" /etc/selinux/config
</code></pre>
<p>4、修改网络</p>
<pre class=" language-sh"><code class=" language-sh">
$ vi /etc/sysctl.d/k8s.conf 

net.bridge.bridge-nf-call-iptables=1
net.bridge.bridge-nf-call-ip6tables=1
net.ipv4.ip_forward=1
vm.swappiness=0
vm.overcommit_memory=1
vm.panic_on_oom=0
fs.inotify.max_user_watches=89100


$ sysctl -p /etc/sysctl.d/k8s.conf
</code></pre>
<p>5、安装和配置docker ce</p>
<p>安装可自行查询资料</p>
<pre class=" language-sh"><code class=" language-sh"># 启动docker

$ systemctl enable docker &amp;&amp; systemctl start docker


# 查看docker是否运行成功

$ systemctl status docker
</code></pre>
<hr>
<h2 id="h2--"><a name="二、准备二进制文件(所有机器)" class="reference-link"></a><span class="header-link octicon octicon-link"></span>二、准备二进制文件(所有机器)</h2><h3 id="h3-2-1-"><a name="2.1 配置免密登录" class="reference-link"></a><span class="header-link octicon octicon-link"></span>2.1 配置免密登录</h3><p>可以快速从一台机器上复制证书、配置文件、二进制等文件到其他机器</p>
<p>在master1上，操作:</p>
<pre class=" language-sh"><code class=" language-sh">
$ ssh-keygen -t rsa

Generating public/private rsa key pair.
Enter file in which to save the key (/root/.ssh/id_rsa):
Created directory '/root/.ssh'.
Enter passphrase (empty for no passphrase):
Enter same passphrase again:
Your identification has been saved in /root/.ssh/id_rsa.
Your public key has been saved in /root/.ssh/id_rsa.pub.
The key fingerprint is:
SHA256:AHglkRC/dxJ9FtgPqx+F4ULxMucVVnFycT04yj5D//w root@master1
The key's randomart image is:
+---[RSA 2048]----+
|  o+++. .+. o.=o*|
|  ..oo .o.+o + =o|
|   .. o.+.**o . .|
|     . o.B+=o    |
|    . o So+..    |
|     . o. .+ .   |
|         . .o o  |
|          .    o |
|                E|
+----[SHA256]-----+
</code></pre>
<p>查看公钥内容：</p>
<pre class=" language-sh"><code class=" language-sh">$ cat ~/.ssh/id_rsa.pub

ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3d+t/3iv0a2Yh+26afvvUYX6LNad/WRDOMqgkvynUkF5ehQ/rykaGBzglJjbYL11B3lZrKip14CYxaKfdXoK2K2sJ61V7VK+j4GOADStfMdvmoEkR+GQwzZk6ra0hN5LuSpyi1o1g6lqy/KppeHqoZk6hj23Ce7DDsPgmZgn79z2iTjvWA5TyiVtIiRL+BCC8kDTM3ODZS5MXxjYRvwQvlv/Ip8i7Xua0a6hJwspgIlJ7LIouEr+osAwkFeXQW/AJCVawKqUcPVRPXFe6NDRFD1duwl9Ofb+1z/s4R5sOqXkglNqR1v9j5ha/vzE0NaTuSBVIQXFavW9NgFPPIboJ root@master1
</code></pre>
<p>把 <code>id_rsa.pub</code> 中的内容copy所有<code>机器</code>的ssh授权文件中，包括master1</p>
<pre class=" language-sh"><code class=" language-sh"># 如果.ssh目录不存在，先创建: mkdir ~/.ssh
$ echo "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3d+t/3iv0a2Yh+26afvvUYX6LNad/WRDOMqgkvynUkF5ehQ/rykaGBzglJjbYL11B3lZrKip14CYxaKfdXoK2K2sJ61V7VK+j4GOADStfMdvmoEkR+GQwzZk6ra0hN5LuSpyi1o1g6lqy/KppeHqoZk6hj23Ce7DDsPgmZgn79z2iTjvWA5TyiVtIiRL+BCC8kDTM3ODZS5MXxjYRvwQvlv/Ip8i7Xua0a6hJwspgIlJ7LIouEr+osAwkFeXQW/AJCVawKqUcPVRPXFe6NDRFD1duwl9Ofb+1z/s4R5sOqXkglNqR1v9j5ha/vzE0NaTuSBVIQXFavW9NgFPPIboJ root@master1" &gt;&gt; ~/.ssh/authorized_keys
</code></pre>
<p>测试免密登录是否成功，不需要密码，说明设置成功。</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# ssh node1
Last login: Fri Nov  5 04:04:08 2021 from master1
[root@node1 ~]#
</code></pre>
<h3 id="h3-2-2-"><a name="2.2 下载二进制文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>2.2 下载二进制文件</h3><p>只在master1上操作，然后通过从master1批量copy到其他机器</p>
<h4 id="h4--k8s-"><a name="下载和整理k8s文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>下载和整理k8s文件</h4><p>下载并解压</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# cd /usr/local/src
[root@master1 src]# wget https://dl.k8s.io/v1.19.16/kubernetes-server-linux-amd64.tar.gz
[root@master1 src]# tar -zxvf kubernetes-server-linux-amd64.tar.gz
</code></pre>
<p>文件存放在 <code>kubernetes/server/bin</code>下：</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 bin]# ll
total 946884
-rwxr-xr-x 1 root root  46776320 Oct 27 12:34 apiextensions-apiserver
-rwxr-xr-x 1 root root  39063552 Oct 27 12:34 kubeadm
-rwxr-xr-x 1 root root  43872256 Oct 27 12:34 kube-aggregator
-rwxr-xr-x 1 root root 115347456 Oct 27 12:34 kube-apiserver
-rw-r--r-- 1 root root         9 Oct 27 12:33 kube-apiserver.docker_tag
-rw------- 1 root root 120163840 Oct 27 12:33 kube-apiserver.tar
-rwxr-xr-x 1 root root 107319296 Oct 27 12:34 kube-controller-manager
-rw-r--r-- 1 root root         9 Oct 27 12:33 kube-controller-manager.docker_tag
-rw------- 1 root root 112135680 Oct 27 12:33 kube-controller-manager.tar
-rwxr-xr-x 1 root root  42950656 Oct 27 12:34 kubectl
-rwxr-xr-x 1 root root 110113992 Oct 27 12:34 kubelet
-rwxr-xr-x 1 root root  38756352 Oct 27 12:34 kube-proxy
-rw-r--r-- 1 root root         9 Oct 27 12:33 kube-proxy.docker_tag
-rw------- 1 root root 100759040 Oct 27 12:33 kube-proxy.tar
-rwxr-xr-x 1 root root  42938368 Oct 27 12:34 kube-scheduler
-rw-r--r-- 1 root root         9 Oct 27 12:33 kube-scheduler.docker_tag
-rw------- 1 root root  47754752 Oct 27 12:33 kube-scheduler.tar
-rwxr-xr-x 1 root root   1634304 Oct 27 12:34 mounter
</code></pre>
<p>整理文件，把不同节点需要的文件，放在不同的目录：</p>
<pre class=" language-sh"><code class=" language-sh"># 创建两个目录
$ [root@master1 bin]# mkdir -p /usr/local/src/k8s-master
$ [root@master1 bin]# mkdir -p /usr/local/src/k8s-worker

# 分别复制文件到两个目录
$ [root@master1 bin]# for i in kubeadm kube-apiserver kube-controller-manager kubectl kube-scheduler;do cp $i /usr/local/src/k8s-master/; done
$ [root@master1 bin]# for i in kubelet kube-proxy;do cp $i /usr/local/src/k8s-worker/; done
</code></pre>
<h4 id="h4--etcd-"><a name="下载和整理etcd文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>下载和整理etcd文件</h4><p>下载并解压</p>
<pre class=" language-sh"><code class=" language-sh">$ cd /usr/local/src

[root@master1 src]# wget https://github.com/etcd-io/etcd/releases/download/v3.4.18/etcd-v3.4.18-linux-amd64.tar
[root@master1 src]# tar -zxvf etcd-v3.4.18-linux-amd64.tar
</code></pre>
<p>复制etcd相关文件到 <code>k8s-master</code>目录：</p>
<pre class=" language-sh"><code class=" language-sh">
[root@master1 src]# cd etcd-v3.4.18-linux-amd64
[root@master1 etcd-v3.4.18-linux-amd64]# cp etcd* /usr/local/src/k8s-master/
</code></pre>
<p>查看 <code>k8s-master</code>中的文件</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 src]# ls k8s-master/
etcd  etcdctl  kubeadm  kube-apiserver  kube-controller-manager  kubectl  kube-scheduler
</code></pre>
<h3 id="h3-2-3-"><a name="2.3 分发二进制文件到其他机器" class="reference-link"></a><span class="header-link octicon octicon-link"></span>2.3 分发二进制文件到其他机器</h3><p>分别在所有机器上，创建目录 <code>/opt/kubernetes/bin</code></p>
<pre class=" language-sh"><code class=" language-sh">$ mkdir -p /opt/kubernetes/bin
</code></pre>
<p>分发到master节点 </p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3; do scp /usr/local/src/k8s-master/* $i:/opt/kubernetes/bin/; done
</code></pre>
<p>分发到worker节点 </p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in node1 node2; do scp /usr/local/src/k8s-worker/* $i:/opt/kubernetes/bin/; done
</code></pre>
<p>给所有节点设置 <code>PATH</code> 环境变量</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3 node1 node2; do ssh $i "echo 'PATH=/opt/kubernetes/bin:$PATH' &gt;&gt; ~/.bashrc"; done
</code></pre>
<p>分别在每台机器上执行环境变量可用</p>
<pre class=" language-sh"><code class=" language-sh">$ source ~/.bashrc
</code></pre>
<hr>
<h2 id="h2--"><a name="三、集群部署" class="reference-link"></a><span class="header-link octicon octicon-link"></span>三、集群部署</h2><h3 id="h3-3-1-cfssl-"><a name="3.1 安装cfssl证书工具" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.1 安装cfssl证书工具</h3><p>在master1上下载cfssl</p>
<pre class=" language-sh"><code class=" language-sh">root@master1 bin]# wget https://pkg.cfssl.org/R1.2/cfssl_linux-amd64 -O ~/bin/cfssl
[root@master1 bin]# wget https://pkg.cfssl.org/R1.2/cfssljson_linux-amd64 -O ~/bin/cfssljson
</code></pre>
<p>给运行权限</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 bin]# chmod +x cfssl
[root@master1 bin]# chmod +x cfssljson
</code></pre>
<p>设置 <code>~/bin</code> 的环境变量</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 bin]# vi ~/.bashrc

PATH=~/bin:$PATH

# 生效
[root@master1 bin]# source ~/.bashrc
</code></pre>
<h3 id="h3-3-2-kubernetes-"><a name="3.2 生成 kubernetes 所需的根证书" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.2 生成 kubernetes 所需的根证书</h3><pre class=" language-sh"><code class=" language-sh">[root@master1 bin]# vi ca-csr.json

{
"CN": "kubernetes",
"key": {
"algo": "rsa",
"size": 2048
},
"names": [
{
"C": "CN",
"ST": "SICHUAN",
"L": "CHENGDU",
"O": "k8s",
"OU": "system"
}
]
}
</code></pre>
<p>生成证书和私钥</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# cfssl gencert -initca ca-csr.json | cfssljson -bare ca

2021/11/08 02:42:32 [INFO] generating a new CA key and certificate from CSR
2021/11/08 02:42:32 [INFO] generate received request
2021/11/08 02:42:32 [INFO] received CSR
2021/11/08 02:42:32 [INFO] generating key: rsa-2048
2021/11/08 02:42:32 [INFO] encoded CSR
2021/11/08 02:42:32 [INFO] signed certificate with serial number 627140244887982433551543860823384941108151783458
</code></pre>
<p>生成 <code>ca-key.pem</code> 和 <code>ca.pem</code>，一个私钥，一个证书</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# ll -h
total 20K
-rw-------. 1 root root 1.3K Nov  5 03:40 anaconda-ks.cfg
drwxr-xr-x  2 root root   36 Nov  8 02:35 bin
-rw-r--r--  1 root root 1001 Nov  8 02:42 ca.csr
-rw-r--r--  1 root root  208 Nov  8 02:42 ca-csr.json
-rw-------  1 root root 1.7K Nov  8 02:42 ca-key.pem
-rw-r--r--  1 root root 1.4K Nov  8 02:42 ca.pem
</code></pre>
<p>将这两个文件传输到每个 <code>master</code>节点上</p>
<pre class=" language-sh"><code class=" language-sh">#在3个master节点，创建 /etc/kubernetes/pki 目录
[root@master1 ~]# for i in master1 master2 master3; do ssh $i "mkdir -p /etc/kubernetes/pki/"; done

#复制两述两个文件到 三个master节点的 /etc/kubernetes/pki 目录下
[root@master1 ~]# for i in master1 master2 master3; do scp *.pem $i:/etc/kubernetes/pki/; done
</code></pre>
<hr>
<h3 id="h3-3-3-master-etcd-"><a name="3.3 在master节点部署etcd集群" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.3 在master节点部署etcd集群</h3><h4 id="h4-3-3-1-etcd-"><a name="3.3.1 生成etcd所需的私钥和证书" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.3.1 生成etcd所需的私钥和证书</h4><pre class=" language-sh"><code class=" language-sh">[root@master1 ~]#  vi ca-config.json

{
"signing": {
"default": {
"expiry": "87600h"
},
"profiles": {
"kubernetes": {
"usages": [
"signing",
"key encipherment",
"server auth",
"client auth"
],
"expiry": "87600h"
}
}
}
}
</code></pre>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]#  vi etcd-csr.json

{
"CN": "etcd",
"hosts": [
"127.0.0.1",
"192.168.0.150",
"192.168.0.151",
"192.168.0.152"
],
"key": {
"algo": "rsa",
"size": 2048
},
"names": [
{
"C": "CN",
"ST": "SICHUAN",
"L": "CHENGDU",
"O": "k8s",
"OU": "system"
}
]
}
</code></pre>
<p>生成文件 </p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# cfssl gencert -ca=ca.pem \
-ca-key=ca-key.pem \
-config=ca-config.json \
-profile=kubernetes etcd-csr.json | cfssljson -bare etcd
</code></pre>
<p>检查文件：</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# ls etcd*.pem

etcd-key.pem  etcd.pem
</code></pre>
<p>无问题后同步到所有 <code>master</code>节点</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3; do scp etcd*.pem $i:/etc/kubernetes/pki/; done
</code></pre>
<h4 id="h4-3-3-2-etcd-systemd-"><a name="3.3.2 创建etcd的systemd服务文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.3.2 创建etcd的systemd服务文件</h4><pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi etcd.service

[Unit]
Description=Etcd Server
After=network.target
After=network-online.target
Wants=network-online.target
Documentation=https://github.com/coreos

[Service]
Type=notify
WorkingDirectory=/var/lib/etcd/
ExecStart=/opt/kubernetes/bin/etcd \
--data-dir=/var/lib/etcd \
--name=master1 \
--cert-file=/etc/kubernetes/pki/etcd.pem \
--key-file=/etc/kubernetes/pki/etcd-key.pem \
--trusted-ca-file=/etc/kubernetes/pki/ca.pem \
--peer-cert-file=/etc/kubernetes/pki/etcd.pem \
--peer-key-file=/etc/kubernetes/pki/etcd-key.pem \
--peer-trusted-ca-file=/etc/kubernetes/pki/ca.pem \
--peer-client-cert-auth \
--client-cert-auth \
--listen-peer-urls=https://192.168.0.150:2380 \
--initial-advertise-peer-urls=https://192.168.0.150:2380 \
--listen-client-urls=https://192.168.0.150:2379,http://127.0.0.1:2379 \
--advertise-client-urls=https://192.168.0.150:2379 \
--initial-cluster-token=etcd-cluster-0 \
--initial-cluster=master1=https://192.168.0.150:2380,master2=https://192.168.0.151:2380,master3=https://192.168.0.152:2380 \
--initial-cluster-state=new
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
</code></pre>
<p>将<code>etcd.service</code>同步到每个<code>master</code>节点</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3; do scp etcd.service $i:/etc/systemd/system/; done
</code></pre>
<p>修改 master1之外的其他站点IP及名称：</p>
<pre class=" language-sh"><code class=" language-sh">  # 修改成所处节点的hostname
--name=master1 \

# 修改为所处节点的IP(内网)
--listen-peer-urls=https://192.168.0.150:2380 \
--initial-advertise-peer-urls=https://192.168.0.150:2380 \
--listen-client-urls=https://192.168.0.150:2379,http://127.0.0.1:2379 \
--advertise-client-urls=https://192.168.0.150:2379 \
</code></pre>
<p>为每个 <code>master</code>节点上创建 <code>etcd</code>的工作目录 <code>/var/lib/etcd</code></p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3; do ssh $i "mkdir -p /var/lib/etcd"; done
</code></pre>
<h4 id="h4-3-3-3-"><a name="3.3.3 启动服务" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.3.3 启动服务</h4><p>分别在 <code>master1</code> <code>master2</code> <code>master3</code>，启动etcd：</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl daemon-reload &amp;&amp; systemctl enable etcd &amp;&amp; systemctl restart etcd
</code></pre>
<p>查看是否启动后的状态</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl status etcd
</code></pre>
<p>如果启动失败,查看日志</p>
<pre class=" language-sh"><code class=" language-sh">$ journalctl -f -u etcd
</code></pre>
<hr>
<h3 id="h3-3-4-master-kube-apiserver"><a name="3.4 在master节点部署 kube-apiserver" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.4 在master节点部署 kube-apiserver</h3><h4 id="h4-3-4-1-"><a name="3.4.1 生成所需的私钥和证书" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.4.1 生成所需的私钥和证书</h4><p>新建配置文件</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi kubernetes-csr.json


{
"CN": "kubernetes",
"hosts": [
"127.0.0.1",
"192.168.0.150",
"192.168.0.151",
"192.168.0.152",
"192.168.0.160",
"10.255.0.1",
"kubernetes",
"kubernetes.default",
"kubernetes.default.svc",
"kubernetes.default.svc.cluster",
"kubernetes.default.svc.cluster.local"
],
"key": {
"algo": "rsa",
"size": 2048
},
"names": [
{
"C": "CN",
"ST": "SICHUAN",
"L": "CHENGDU",
"O": "k8s",
"OU": "system"
}
]
}
</code></pre>
<p>生成私钥和证书</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# cfssl gencert -ca=ca.pem \
-ca-key=ca-key.pem \
-config=ca-config.json \
-profile=kubernetes kubernetes-csr.json | cfssljson -bare kubernetes
</code></pre>
<p>分发到每个master节点</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]#  for i in master1 master2 master3; do scp kubernetes*.pem $i:/etc/kubernetes/pki/; done
</code></pre>
<h4 id="h4-3-4-2-kube-apiserver-systemd-"><a name="3.4.2 创建kube-apiserver的systemd服务文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.4.2 创建kube-apiserver的systemd服务文件</h4><p>创建文件</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi kube-apiserver.service

[Unit]
Description=Kubernetes API Server
Documentation=https://github.com/GoogleCloudPlatform/kubernetes
After=network.target

[Service]
ExecStart=/opt/kubernetes/bin/kube-apiserver \
--enable-admission-plugins=NamespaceLifecycle,NodeRestriction,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota \
--anonymous-auth=false \
--advertise-address=192.168.0.150 \
--bind-address=0.0.0.0 \
--insecure-port=0 \
--authorization-mode=Node,RBAC \
--runtime-config=api/all=true \
--enable-bootstrap-token-auth \
--service-cluster-ip-range=10.255.0.0/16 \
--service-node-port-range=30000-32767 \
--tls-cert-file=/etc/kubernetes/pki/kubernetes.pem \
--tls-private-key-file=/etc/kubernetes/pki/kubernetes-key.pem \
--client-ca-file=/etc/kubernetes/pki/ca.pem \
--kubelet-client-certificate=/etc/kubernetes/pki/kubernetes.pem \
--kubelet-client-key=/etc/kubernetes/pki/kubernetes-key.pem \
--service-account-key-file=/etc/kubernetes/pki/ca-key.pem \
--etcd-cafile=/etc/kubernetes/pki/ca.pem \
--etcd-certfile=/etc/kubernetes/pki/kubernetes.pem \
--etcd-keyfile=/etc/kubernetes/pki/kubernetes-key.pem \
--etcd-servers=https://192.168.0.150:2379,https://192.168.0.151:2379,https://192.168.0.152:2379 \
--enable-swagger-ui=true \
--allow-privileged=true \
--apiserver-count=3 \
--audit-log-maxage=30 \
--audit-log-maxbackup=3 \
--audit-log-maxsize=100 \
--audit-log-path=/var/log/kube-apiserver-audit.log \
--event-ttl=1h \
--alsologtostderr=true \
--logtostderr=false \
--log-dir=/var/log/kubernetes \
--v=2
Restart=on-failure
RestartSec=5
Type=notify
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
</code></pre>
<p>分发到每个master节点</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3; do scp kube-apiserver.service $i:/etc/systemd/system/; done
</code></pre>
<p>修改除 <code>master1</code>机器之外的 <code>kube-apiserver.service</code>配置</p>
<pre class=" language-sh"><code class=" language-sh">  # 修改为节点所在的内网IP
--advertise-address=192.168.0.150
</code></pre>
<p>在所有<code>master</code>节点，创建 <code>api-server</code>的日志目录</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3; do ssh $i "mkdir -p /var/log/kubernetes"; done
</code></pre>
<h4 id="h4-3-4-3-"><a name="3.4.3 启动服务" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.4.3 启动服务</h4><p>在每个 <code>master</code>节点上启动<code>kube-apiserver</code>服务</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl daemon-reload &amp;&amp; systemctl enable kube-apiserver &amp;&amp; systemctl restart kube-apiserver
</code></pre>
<p>查看状态：</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl status kube-apiserver
</code></pre>
<p>如果启动失败，排查问题</p>
<pre class=" language-sh"><code class=" language-sh">$ journalctl -f -u kube-apiserver
</code></pre>
<hr>
<h3 id="h3-3-5-keepalived"><a name="3.5 安装 keepalived" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.5 安装 keepalived</h3><p>在所有的<code>master</code> 节点安装，用于实现虚拟IP</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# yum install -y keepalived
</code></pre>
<pre class=" language-sh"><code class=" language-sh">
[root@master1 ~]# vi /etc/keepalived/keepalived.conf

global_defs { # 全局配置
notification_email { # 通知邮件，可以多个
301109640@qq.com
}
notification_email_from Alexandre.Cassen@firewall.loc # 通知邮件发件人，可以自行修改
smtp_server 127.0.0.1     # 邮件服务器地址
smtp_connect_timeout 30   # 邮件服务器连接的timeout
router_id LVS_1              # 机器标识，可以不修改，多台机器可相同
}

vrrp_instance VI_1 {  # vroute标识
state MASTER      # 当前节点的状态：主节点       
interface eth0    # 发送vip通告的接口
lvs_sync_daemon_inteface eth0
virtual_router_id 79 # 虚拟路由的ID号是虚拟路由MAC的最后一位地址
advert_int 1         # vip通告的时间间隔   
priority 100          # 此节点的优先级主节点的优先级需要比其他节点高，我配置成：master1 100 master2 80 master3 70   
authentication {      # 认证配置
auth_type PASS # 认证机制默认是明文
auth_pass 1111 # 随机字符当密码，要和虚拟路由器中其它路由器保持一致
}
virtual_ipaddress { # vip
192.168.0.160/20  # 192.168.0.160 的vip
}
}
</code></pre>
<p>启动</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# systemctl enable keepalived &amp;&amp; systemctl restart keepalived
</code></pre>
<p>启动成功后，可以看到类似信息:</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# ip addr

2: eth0: &lt;BROADCAST,MULTICAST,UP,LOWER_UP&gt; mtu 1500 qdisc pfifo_fast state UP group default qlen 1000
link/ether 00:1c:42:3f:7b:c5 brd ff:ff:ff:ff:ff:ff
inet 192.168.0.150/24 brd 192.168.0.255 scope global noprefixroute eth0
valid_lft forever preferred_lft forever
inet 192.168.0.160/20 scope global eth0
valid_lft forever preferred_lft forever
inet6 fe80::d541:71b6:7b10:71cb/64 scope link noprefixroute
valid_lft forever preferred_lft forever
</code></pre>
<p>如果 <code>master1</code>不可用时，VIP可能漂移到 <code>master2</code>或<code>master3上</code></p>
<h3 id="h3-3-6-haproxy"><a name="3.6 安装和配置haproxy" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.6 安装和配置haproxy</h3><p>在所有<code>master</code>节点安装haproxy，用于实现tcp层的<code>kube-apiserver</code>代理</p>
<pre class=" language-sh"><code class=" language-sh">$ yum install -y haproxy
</code></pre>
<p>修改配置</p>
<pre class=" language-sh"><code class=" language-sh">$ vi /etc/haproxy/haproxy.cfg


global
chroot  /var/lib/haproxy
daemon
group haproxy
user haproxy
log 127.0.0.1:514 local0 warning
pidfile /var/lib/haproxy.pid
maxconn 20000
spread-checks 3
nbproc 8

defaults
log     global
mode    tcp
retries 3
option redispatch

listen https-apiserver
bind 0.0.0.0:8443 # 此处为8443
mode tcp
balance roundrobin
timeout server 900s
timeout connect 15s

server master1 192.168.0.150:6443 check port 6443 inter 5000 fall 5
server master2 192.168.0.151:6443 check port 6443 inter 5000 fall 5
server master3 192.168.0.152:6443 check port 6443 inter 5000 fall 5
</code></pre>
<p>启动haproxy</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl enable haproxy &amp;&amp; systemctl restart haproxy
</code></pre>
<p>检测代理后的<code>kube-apiserver</code>地址及端口</p>
<pre class=" language-sh"><code class=" language-sh">$ curl --insecure https://192.168.0.160:8443/


{
"kind": "Status",
"apiVersion": "v1",
"metadata": {

},
"status": "Failure",
"message": "Unauthorized",
"reason": "Unauthorized",
"code": 401
}
</code></pre>
<hr>
<h3 id="h3-3-7-kubectl"><a name="3.7 安装kubectl" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.7 安装kubectl</h3><p>可以在任意节点安装。kubectl是集群的命令行管理工具</p>
<h4 id="h4-3-7-1-"><a name="3.7.1 创建所需的私钥和证书" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.7.1 创建所需的私钥和证书</h4><pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi admin-csr.json

{
"CN": "admin",
"hosts": [],
"key": {
"algo": "rsa",
"size": 2048
},
"names": [
{
"C": "CN",
"ST": "SICHUAN",
"L": "CHENGDU",
"O": "system:masters",
"OU": "system"
}
]
}
</code></pre>
<p>生成私钥和证书</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# cfssl gencert -ca=ca.pem \
-ca-key=ca-key.pem \
-config=ca-config.json \
-profile=kubernetes admin-csr.json | cfssljson -bare admin
</code></pre>
<h4 id="h4-3-7-2-kubeconfig-"><a name="3.7.2 创建kubeconfig配置文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.7.2 创建kubeconfig配置文件</h4><p>设置集群参数</p>
<pre class=" language-sh"><code class=" language-sh">
[root@master1 ~]# kubectl config set-cluster kubernetes \
--certificate-authority=ca.pem \
--embed-certs=true \
--server=https://192.168.0.160:8443 \
--kubeconfig=kube.config
</code></pre>
<p>设置客户端认证参数</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-credentials admin \
--client-certificate=admin.pem \
--client-key=admin-key.pem \
--embed-certs=true \
--kubeconfig=kube.config
</code></pre>
<p>设置下下文参数</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-context kubernetes \
--cluster=kubernetes \
--user=admin \
--kubeconfig=kube.config
</code></pre>
<p>设置默认上下文</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config use-context kubernetes --kubeconfig=kube.config
</code></pre>
<p>复制文件到 <code>.kube</code>目录下，如果没有.kube目录，使用 <code>mkdir -p ~/.kube</code> 创建</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# cp kube.config ~/.kube/config
</code></pre>
<h4 id="h4-3-7-3-code-kubernetes-code-kubelet-api-"><a name="3.7.3 授权 <code>kubernetes</code>访问 kubelet API的权限" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.7.3 授权 <code>kubernetes</code>访问 kubelet API的权限</h4><pre class=" language-sh"><code class=" language-sh">
[root@master1 ~]# kubectl create clusterrolebinding kube-apiserver:kubelet-apis --clusterrole=system:kubelet-api-admin --user kubernetes

clusterrolebinding.rbac.authorization.k8s.io/kube-apiserver:kubelet-apis created
</code></pre>
<h4 id="h4-3-7-4-kubectl-"><a name="3.7.4 测试kubectl可用" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.7.4 测试kubectl可用</h4><p>查看集群信息</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl cluster-info

Kubernetes master is running at https://192.168.0.160:8443

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
</code></pre>
<p>查看所有资源</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl get all --all-namespaces -o wide

NAMESPACE   NAME                 TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE     SELECTOR
default     service/kubernetes   ClusterIP   10.255.0.1   &lt;none&gt;        443/TCP   4h57m   &lt;none&gt;
</code></pre>
<p>查看集群中的所有组件状态</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl get cs
Warning: v1 ComponentStatus is deprecated in v1.19+
NAME                 STATUS      MESSAGE                                                                                       ERROR
scheduler            Unhealthy   Get "http://127.0.0.1:10251/healthz": dial tcp 127.0.0.1:10251: connect: connection refused
controller-manager   Unhealthy   Get "http://127.0.0.1:10252/healthz": dial tcp 127.0.0.1:10252: connect: connection refused
etcd-0               Healthy     {"health":"true"}
etcd-1               Healthy     {"health":"true"}
etcd-2               Healthy     {"health":"true"}
</code></pre>
<hr>
<h3 id="h3-3-8-kube-controller-manager"><a name="3.8 部署 kube-controller-manager" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.8 部署 kube-controller-manager</h3><p>在所有master节点上部署</p>
<h4 id="h4-3-8-1-"><a name="3.8.1 创建私钥和证书" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.8.1 创建私钥和证书</h4><pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi controller-manager-csr.json


{
"CN": "system:kube-controller-manager",
"key": {
"algo": "rsa",
"size": 2048
},
"hosts": [
"127.0.0.1",
"192.168.0.150",
"192.168.0.151",
"192.168.0.152"
],
"names": [
{
"C": "CN",
"ST": "SICHUAN",
"L": "CHENGDU",
"O": "system:kube-controller-manager",
"OU": "system"
}
]
}
</code></pre>
<p>生成私钥和证书</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# cfssl gencert -ca=ca.pem \
-ca-key=ca-key.pem \
-config=ca-config.json \
-profile=kubernetes controller-manager-csr.json | cfssljson -bare controller-manager
</code></pre>
<p>分发到所有master节点</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3; do scp controller-manager*.pem $i:/etc/kubernetes/pki/; done
</code></pre>
<h4 id="h4-3-8-2-controller-manager-kubeconfig"><a name="3.8.2 创建controller-manager的kubeconfig" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.8.2 创建controller-manager的kubeconfig</h4><p>设置集群彩数</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-cluster kubernetes \
--certificate-authority=ca.pem \
--embed-certs=true \
--server=https://192.168.0.160:8443 \
--kubeconfig=controller-manager.kubeconfig
</code></pre>
<p>设置客户端参数</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-credentials system:kube-controller-manager \
--client-certificate=controller-manager.pem \
--client-key=controller-manager-key.pem \
--embed-certs=true \
--kubeconfig=controller-manager.kubeconfig
</code></pre>
<p>设置上下文</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-context system:kube-controller-manager \
--cluster=kubernetes \
--user=system:kube-controller-manager \
--kubeconfig=controller-manager.kubeconfig
</code></pre>
<p>设置默认上下文</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config use-context system:kube-controller-manager --kubeconfig=controller-manager.kubeconfig
</code></pre>
<p>分发 <code>controller-manager.kubeconfig</code> 文件到每个 <code>master</code>节点</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3; do scp controller-manager.kubeconfig $i:/etc/kubernetes/; done
</code></pre>
<h4 id="h4-3-8-3-kube-controller-manager-systemd-"><a name="3.8.3 创建 kube-controller-manager的systemd启动文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.8.3 创建 kube-controller-manager的systemd启动文件</h4><pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi kube-controller-manager.service

[Unit]
Description=Kubernetes Controller Manager
Documentation=https://github.com/GoogleCloudPlatform/kubernetes

[Service]
ExecStart=/opt/kubernetes/bin/kube-controller-manager \
--port=0 \
--secure-port=10252 \
--bind-address=127.0.0.1 \
--kubeconfig=/etc/kubernetes/controller-manager.kubeconfig \
--service-cluster-ip-range=10.255.0.0/16 \
--cluster-name=kubernetes \
--cluster-signing-cert-file=/etc/kubernetes/pki/ca.pem \
--cluster-signing-key-file=/etc/kubernetes/pki/ca-key.pem \
--allocate-node-cidrs=true \
--cluster-cidr=172.23.0.0/16 \
--experimental-cluster-signing-duration=87600h \
--root-ca-file=/etc/kubernetes/pki/ca.pem \
--service-account-private-key-file=/etc/kubernetes/pki/ca-key.pem \
--leader-elect=true \
--feature-gates=RotateKubeletServerCertificate=true \
--controllers=*,bootstrapsigner,tokencleaner \
--horizontal-pod-autoscaler-use-rest-clients=true \
--horizontal-pod-autoscaler-sync-period=10s \
--tls-cert-file=/etc/kubernetes/pki/controller-manager.pem \
--tls-private-key-file=/etc/kubernetes/pki/controller-manager-key.pem \
--use-service-account-credentials=true \
--alsologtostderr=true \
--logtostderr=false \
--log-dir=/var/log/kubernetes \
--v=2
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
</code></pre>
<p>将 <code>kube-controller-manager.service</code>分发到每个master节点</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]#  for i in master1 master2 master3; do scp kube-controller-manager.service $i:/etc/systemd/system/; done
</code></pre>
<p>在每个master上启动 <code>kube-controller-manager</code>服务</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl daemon-reload &amp;&amp; systemctl enable kube-controller-manager &amp;&amp; systemctl restart kube-controller-manager
</code></pre>
<p>查看服务器状态</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl status kube-controller-manager
</code></pre>
<p>如果没有启动成功，查看日志</p>
<pre class=" language-sh"><code class=" language-sh">$ journalctl -f -u kube-controller-manager
</code></pre>
<hr>
<h3 id="h3-3-9-kube-scheduler"><a name="3.9 部署 kube scheduler" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.9 部署 kube scheduler</h3><p>在所有master节点上完成</p>
<h4 id="h4-3-9-1-"><a name="3.9.1 创建私钥和证书" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.9.1 创建私钥和证书</h4><pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi scheduler-csr.json


{
"CN": "system:kube-scheduler",
"hosts": [
"127.0.0.1",
"192.168.0.150",
"192.168.0.151",
"192.168.0.152"
],
"key": {
"algo": "rsa",
"size": 2048
},
"names": [
{
"C": "CN",
"ST": "SICHUAN",
"L": "CHENGDU",
"O": "system:kube-scheduler",
"OU": "system"
}
]
}
</code></pre>
<p>生成私钥和证书</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# cfssl gencert -ca=ca.pem \
-ca-key=ca-key.pem \
-config=ca-config.json \
-profile=kubernetes scheduler-csr.json | cfssljson -bare kube-scheduler
</code></pre>
<p>分发到每个 <code>master</code> 节点</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3;do scp kube-scheduler*.pem $i:/etc/kubernetes/pki;done
</code></pre>
<h4 id="h4-3-9-2-kube-scheduler-kubeconfig"><a name="3.9.2 创建kube scheduler的kubeconfig" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.9.2 创建kube scheduler的kubeconfig</h4><p>设置集群参数</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-cluster kubernetes \
--certificate-authority=ca.pem \
--embed-certs=true \
--server=https://192.168.0.160:8443 \
--kubeconfig=kube-scheduler.kubeconfig
</code></pre>
<p>设置客户端认证参数</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-credentials system:kube-scheduler \
--client-certificate=kube-scheduler.pem \
--client-key=kube-scheduler-key.pem \
--embed-certs=true \
--kubeconfig=kube-scheduler.kubeconfig
</code></pre>
<p>设置上下文参数</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-context system:kube-scheduler \
--cluster=kubernetes \
--user=system:kube-scheduler \
--kubeconfig=kube-scheduler.kubeconfig
</code></pre>
<p>设置默认下下文</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config use-context system:kube-scheduler --kubeconfig=kube-scheduler.kubeconfig
</code></pre>
<p>将 <code>kube-scheduler.kubeconfig</code>文件分发到每个 <code>master</code>节点上</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3; do scp kube-scheduler.kubeconfig $i:/etc/kubernetes/; done
</code></pre>
<h4 id="h4-3-9-3-kube-scheduler-systemd-"><a name="3.9.3 创建 kube-scheduler的systemd启动文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.9.3 创建 kube-scheduler的systemd启动文件</h4><p>创建 <code>kube-scheduler.service</code> 文件:</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi kube-scheduler.service

[Unit]
Description=Kubernetes Scheduler
Documentation=https://github.com/GoogleCloudPlatform/kubernetes

[Service]
ExecStart=/opt/kubernetes/bin/kube-scheduler \
--address=127.0.0.1 \
--kubeconfig=/etc/kubernetes/kube-scheduler.kubeconfig \
--leader-elect=true \
--alsologtostderr=true \
--logtostderr=false \
--log-dir=/var/log/kubernetes \
--v=2
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
</code></pre>
<p>将 <code>kube-scheduler.service</code>文件分发到每个<code>master</code>节点上</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in master1 master2 master3;do scp kube-scheduler.service $i:/etc/systemd/system/;done
</code></pre>
<h4 id="h4-3-9-4-kube-scheduler-"><a name="3.9.4 启动kube-scheduler服务" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.9.4 启动kube-scheduler服务</h4><p>在每个<code>master</code>节点上启动服务:</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl daemon-reload &amp;&amp; systemctl enable kube-scheduler &amp;&amp; systemctl restart kube-scheduler
</code></pre>
<p>查看服务状态：</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl status kube-scheduler
</code></pre>
<p>异常时，查看启动日志：</p>
<pre class=" language-sh"><code class=" language-sh">$ journalctl -f -u kube-scheduler
</code></pre>
<hr>
<h3 id="h3-3-10-kubelet"><a name="3.10 部署kubelet" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.10 部署kubelet</h3><pre class=" language-sh"><code class=" language-sh">
$ docker pull registry.cn-hangzhou.aliyuncs.com/google_containers/pause-amd64:3.2
$ docker tag registry.cn-hangzhou.aliyuncs.com/google_containers/pause-amd64:3.2 k8s.gcr.io/pause-amd64:3.2
$ docker rmi registry.cn-hangzhou.aliyuncs.com/google_containers/pause-amd64:3.2

$ docker pull registry.cn-hangzhou.aliyuncs.com/google_containers/coredns:1.7.0
$ docker tag registry.cn-hangzhou.aliyuncs.com/google_containers/coredns:1.7.0 k8s.gcr.io/coredns:1.7.0
$ docker rmi registry.cn-hangzhou.aliyuncs.com/google_containers/coredns:1.7.0
</code></pre>
<p>在所有worker节点上完成</p>
<h4 id="h4-3-10-1-bootstrap-"><a name="3.10.1 创建bootstrap配置文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.10.1 创建bootstrap配置文件</h4><p>创建token并设置环境变量</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# export BOOTSTRAP_TOKEN=$(kubeadm token create \
--description kubelet-bootstrap-token \
--groups system:bootstrappers:worker \
--kubeconfig kube.config)
</code></pre>
<p>创建 <code>kube-bootstrap.kubeconfig</code></p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-cluster kubernetes \
--certificate-authority=ca.pem \
--embed-certs=true \
--server=https://192.168.0.160:8443 \
--kubeconfig=kubelet-bootstrap.kubeconfig
</code></pre>
<p>设置客户端认证参数</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-credentials kubelet-bootstrap \
--token=${BOOTSTRAP_TOKEN} \
--kubeconfig=kubelet-bootstrap.kubeconfig
</code></pre>
<p>设置上下文参数</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-context default \
--cluster=kubernetes \
--user=kubelet-bootstrap \
--kubeconfig=kubelet-bootstrap.kubeconfig
</code></pre>
<p>设置默认上下文</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config use-context default --kubeconfig=kubelet-bootstrap.kubeconfig
</code></pre>
<p>分发 <code>kubelet-bootstrap.kubeconfig</code>到 <code>worker</code>节点</p>
<pre class=" language-sh"><code class=" language-sh">
# 创建目录
[root@master1 ~]# for i in node1 node2; do ssh $i "mkdir /etc/kubernetes/"; done

# 分发文件
[root@master1 ~]# for i in node1 node2; do scp kubelet-bootstrap.kubeconfig $i:/etc/kubernetes/kubelet-bootstrap.kubeconfig; done
</code></pre>
<p>分发证书和密钥文件到<code>worker</code>节点</p>
<pre class=" language-sh"><code class=" language-sh"># 创建证书目录
[root@master1 ~]# for i in node1 node2; do ssh $i "mkdir -p /etc/kubernetes/pki"; done

# 分发文件
[root@master1 ~]# for i in node1 node2; do scp ca.pem $i:/etc/kubernetes/pki/; done
</code></pre>
<h4 id="h4-3-10-2-kubelet-"><a name="3.10.2 创建kubelet配置文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.10.2 创建kubelet配置文件</h4><pre class=" language-sh"><code class=" language-sh">
[root@master1 ~]# vi kubelet.config.json

{
"kind": "KubeletConfiguration",
"apiVersion": "kubelet.config.k8s.io/v1beta1",
"authentication": {
"x509": {
"clientCAFile": "/etc/kubernetes/pki/ca.pem"
},
"webhook": {
"enabled": true,
"cacheTTL": "2m0s"
},
"anonymous": {
"enabled": false
}
},
"authorization": {
"mode": "Webhook",
"webhook": {
"cacheAuthorizedTTL": "5m0s",
"cacheUnauthorizedTTL": "30s"
}
},
"address": "192.168.0.153",
"port": 10250,
"readOnlyPort": 10255,
"cgroupDriver": "cgroupfs",
"hairpinMode": "promiscuous-bridge",
"serializeImagePulls": false,
"featureGates": {
"RotateKubeletClientCertificate": true,
"RotateKubeletServerCertificate": true
},
"clusterDomain": "cluster.local.",
"clusterDNS": ["10.255.0.2"]
}
</code></pre>
<p>把 <code>kubelet.config.json</code>配置文件分到到每个<code>worker</code>节点上</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in node1 node2; do scp kubelet.config.json $i:/etc/kubernetes/; done
</code></pre>
<p><em>注意</em>：分发完成后，需要修改配置文件中的<code>address</code>字段，为所在节点的内网IP</p>
<h4 id="h4-3-10-3-kubelet-systemd-"><a name="3.10.3 创建kubelet的systemd启动文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.10.3 创建kubelet的systemd启动文件</h4><pre class=" language-sh"><code class=" language-sh">
[root@master1 ~]# vi kubelet.service


[Unit]
Description=Kubernetes Kubelet
Documentation=https://github.com/GoogleCloudPlatform/kubernetes
After=docker.service
Requires=docker.service

[Service]
WorkingDirectory=/var/lib/kubelet
ExecStart=/opt/kubernetes/bin/kubelet \
--bootstrap-kubeconfig=/etc/kubernetes/kubelet-bootstrap.kubeconfig \
--cert-dir=/etc/kubernetes/pki \
--kubeconfig=/etc/kubernetes/kubelet.kubeconfig \
--config=/etc/kubernetes/kubelet.config.json \
--network-plugin=cni \
--pod-infra-container-image=k8s.gcr.io/pause-amd64:3.2 \
--alsologtostderr=true \
--logtostderr=false \
--log-dir=/var/log/kubernetes \
--v=2
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
</code></pre>
<p>把 <code>kubelet.service</code> 分发到每个<code>worker</code>节点上</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in node1 node2; do scp kubelet.service $i:/etc/systemd/system/; done
</code></pre>
<p>创建每个<code>worker</code>节点 <code>kubelet</code>工作目录</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in node1 node2; do ssh $i "mkdir -p /var/lib/kubelet"; done
</code></pre>
<h4 id="h4-3-10-4-kubelet-"><a name="3.10.4 启动kubelet服务" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.10.4 启动kubelet服务</h4><p>bootstrap赋权，创建一个角色绑定</p>
<pre class=" language-sh"><code class=" language-sh">
[root@master1 ~]#  ~]# kubectl create clusterrolebinding kubelet-bootstrap --clusterrole=system:node-bootstrapper --group=system:bootstrappers
</code></pre>
<p>在每个 <code>worker</code>节点启动 kubelet</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl daemon-reload &amp;&amp; systemctl enable kubelet &amp;&amp; systemctl restart kubelet
</code></pre>
<p>查看启动状态</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl status kubelet
</code></pre>
<p>如果没有启动成功，可以查看日志</p>
<pre class=" language-sh"><code class=" language-sh">$ journalctl -f -u kubelet
</code></pre>
<h4 id="h4-3-10-5-"><a name="3.10.5 加入集群" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.10.5 加入集群</h4><p>确保 <code>kubelet</code> 服务启动成功后，查看两个<code>worker</code>节点的请求。</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl get csr

NAME                                                   AGE   SIGNERNAME                                    REQUESTOR                 CONDITION
node-csr-Wg5tb9HaItJp3pirkva2E4uLwW58gRyV68FIHCHqPPg   30s   kubernetes.io/kube-apiserver-client-kubelet   system:bootstrap:wmp9un   Pending
node-csr-glVMjyBuo3vceYH4hCIrbi-YsguLuUSOaa1S_AMkFPo   29s   kubernetes.io/kube-apiserver-client-kubelet   system:bootstrap:wmp9un   Pending
</code></pre>
<p>分别<code>Approve</code>(同意) 这两个请求：</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]#  kubectl certificate approve node-csr-Wg5tb9HaItJp3pirkva2E4uLwW58gRyV68FIHCHqPPg

certificatesigningrequest.certificates.k8s.io/node-csr-Wg5tb9HaItJp3pirkva2E4uLwW58gRyV68FIHCHqPPg approved

[root@master1 ~]#  kubectl certificate approve node-csr-glVMjyBuo3vceYH4hCIrbi-YsguLuUSOaa1S_AMkFPo

certificatesigningrequest.certificates.k8s.io/node-csr-glVMjyBuo3vceYH4hCIrbi-YsguLuUSOaa1S_AMkFPo approved
</code></pre>
<p>此时执行，两个 <code>worker</code>节点已经加入，但是状态为 <code>NotReady</code>，说明还需要完成后续操作 </p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]#  kubectl get node


NAME    STATUS     ROLES    AGE   VERSION
node1   NotReady   &lt;none&gt;   42s   v1.19.16
node2   NotReady   &lt;none&gt;   15s   v1.19.16
</code></pre>
<p><em>说明</em>：因为 kubelet 没有部署在 <code>master</code>节点，所以 <code>kubectl get node</code>时看不到任何 <code>master</code> 节点 </p>
<hr>
<h3 id="h3-3-11-kube-proxy-"><a name="3.11 部署 kube-proxy 服务" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.11 部署 kube-proxy 服务</h3><p>在<code>worker</code>节点完成</p>
<h4 id="h4-3-11-1-"><a name="3.11.1 创建私钥和证书" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.11.1 创建私钥和证书</h4><p>创建 csr 文件</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi kube-proxy-csr.json

{
"CN": "system:kube-proxy",
"key": {
"algo": "rsa",
"size": 2048
},
"names": [
{
"C": "CN",
"ST": "SICHUAN",
"L": "CHENGDU",
"O": "k8s",
"OU": "system"
}
]
}
</code></pre>
<p>生成私钥和证书</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# cfssl gencert -ca=ca.pem \
-ca-key=ca-key.pem \
-config=ca-config.json \
-profile=kubernetes kube-proxy-csr.json | cfssljson -bare kube-proxy
</code></pre>
<p>创建 <code>kube-proxy.kubeconfig</code>文件</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-cluster kubernetes \
--certificate-authority=ca.pem \
--embed-certs=true \
--server=https://192.168.0.160:8443 \
--kubeconfig=kube-proxy.kubeconfig
</code></pre>
<p>设置客户端认证参数</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]#  kubectl config set-credentials kube-proxy \
--client-certificate=kube-proxy.pem \
--client-key=kube-proxy-key.pem \
--embed-certs=true \
--kubeconfig=kube-proxy.kubeconfig
</code></pre>
<p>设置上下文</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config set-context default \
--cluster=kubernetes \
--user=kube-proxy \
--kubeconfig=kube-proxy.kubeconfig
</code></pre>
<p>切换默认上下文</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl config use-context default --kubeconfig=kube-proxy.kubeconfig
</code></pre>
<p>分发 <code>kube-proxy.kubeconfig</code> 文件到每个 <code>worker</code>节点</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in node1 node2;do scp kube-proxy.kubeconfig $i:/etc/kubernetes/;done
</code></pre>
<h4 id="h4-3-11-2-kube-proxy-"><a name="3.11.2 创建和分发kube-proxy配置文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.11.2 创建和分发kube-proxy配置文件</h4><p>创建 <code>kube-proxy.config.yaml</code>文件</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi kube-proxy.config.yaml
</code></pre>
<pre class=" language-yaml"><code class=" language-yaml">
<span class="token key atrule">apiVersion</span><span class="token punctuation">:</span> kubeproxy.config.k8s.io/v1alpha1
<span class="token comment"># 修改为所在节点的ip</span>
<span class="token key atrule">bindAddress</span><span class="token punctuation">:</span> <span class="token punctuation">{</span>worker_ip<span class="token punctuation">}</span>
<span class="token key atrule">clientConnection</span><span class="token punctuation">:</span>
<span class="token key atrule">kubeconfig</span><span class="token punctuation">:</span> /etc/kubernetes/kube<span class="token punctuation">-</span>proxy.kubeconfig
<span class="token key atrule">clusterCIDR</span><span class="token punctuation">:</span> 172.23.0.0/16
<span class="token comment"># 修改为所在节点的ip</span>
<span class="token key atrule">healthzBindAddress</span><span class="token punctuation">:</span> <span class="token punctuation">{</span>worker_ip<span class="token punctuation">}</span><span class="token punctuation">:</span><span class="token number">10256</span>
<span class="token key atrule">kind</span><span class="token punctuation">:</span> KubeProxyConfiguration
<span class="token comment"># 修改为所在节点的ip</span>
<span class="token key atrule">metricsBindAddress</span><span class="token punctuation">:</span> <span class="token punctuation">{</span>worker_ip<span class="token punctuation">}</span><span class="token punctuation">:</span><span class="token number">10249</span>
<span class="token key atrule">mode</span><span class="token punctuation">:</span> <span class="token string">"iptables"</span>
</code></pre>
<p><em>注意:</em> 其中的 <code>{worker_ip}</code> 为每个 <code>worker</code> 节点的内网IP，记得分发后修改</p>
<p>将 <code>kube-proxy.config.yaml</code> 文件分到到每个 <code>worker</code>节点上</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in node1 node2;do scp kube-proxy.config.yaml $i:/etc/kubernetes/;done
</code></pre>
<h4 id="h4-3-11-3-kube-proxy-systemd-"><a name="3.11.3 创建和分发kube-proxy的systemd服务文件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.11.3 创建和分发kube-proxy的systemd服务文件</h4><p><code>kube-proxy.service</code>文件内容</p>
<pre class=" language-sh"><code class=" language-sh">
[root@master1 ~]# vi kube-proxy.service 

[Unit]
Description=Kubernetes Kube-Proxy Server
Documentation=https://github.com/GoogleCloudPlatform/kubernetes
After=network.target

[Service]
WorkingDirectory=/var/lib/kube-proxy
ExecStart=/opt/kubernetes/bin/kube-proxy \
--config=/etc/kubernetes/kube-proxy.config.yaml \
--alsologtostderr=true \
--logtostderr=false \
--log-dir=/var/log/kubernetes \
--v=2
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
</code></pre>
<p>将 <code>kube-proxy.service</code>文件分发到 <code>worker</code>节点上:</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in node1 node2;do scp kube-proxy.service $i:/etc/systemd/system/;done
</code></pre>
<h4 id="h4-3-11-4-kube-proxy-"><a name="3.11.4 启动kube-proxy服务" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.11.4 启动kube-proxy服务</h4><p>创建 <code>kube-proxy</code> 服务需要的工作及日志目录：</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# for i in node1 node2; do ssh $i "mkdir -p /var/lib/kube-proxy &amp;&amp; mkdir -p /var/log/kubernetes"; done
</code></pre>
<p>在每个<code>worker</code>节点启动服务</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl daemon-reload &amp;&amp; systemctl enable kube-proxy &amp;&amp; systemctl restart kube-proxy
</code></pre>
<p>查看状态：</p>
<pre class=" language-sh"><code class=" language-sh">$ systemctl status kube-proxy
</code></pre>
<p>查看日志：</p>
<pre class=" language-sh"><code class=" language-sh">$ journalctl -f -u kube-proxy
</code></pre>
<hr>
<h3 id="h3-3-12-cni-"><a name="3.12 部署CNI网络插件" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.12 部署CNI网络插件</h3><p>本次官方的安装方式，使用部署 <code>calico</code></p>
<p>创建 <code>calico-rbac-kdd.yaml</code>文件：</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi calico.rbac-kdd.yaml

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
name: calico-node
rules:
- apiGroups: [""]
resources:
- namespaces
verbs:
- get
- list
- watch
- apiGroups: [""]
resources:
- pods/status
verbs:
- update
- apiGroups: [""]
resources:
- pods
verbs:
- get
- list
- watch
- patch
- apiGroups: [""]
resources:
- services
verbs:
- get
- apiGroups: [""]
resources:
- endpoints
verbs:
- get
- apiGroups: [""]
resources:
- nodes
verbs:
- get
- list
- update
- watch
- apiGroups: ["extensions"]
resources:
- networkpolicies
verbs:
- get
- list
- watch
- apiGroups: ["networking.k8s.io"]
resources:
- networkpolicies
verbs:
- watch
- list
- apiGroups: ["crd.projectcalico.org"]
resources:
- globalfelixconfigs
- felixconfigurations
- bgppeers
- globalbgpconfigs
- bgpconfigurations
- ippools
- globalnetworkpolicies
- globalnetworksets
- networkpolicies
- clusterinformations
- hostendpoints
verbs:
- create
- get
- list
- update
- watch

---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
name: calico-node
roleRef:
apiGroup: rbac.authorization.k8s.io
kind: ClusterRole
name: calico-node
subjects:
- kind: ServiceAccount
name: calico-node
namespace: kube-system
</code></pre>
<p>使用kubectl安装calico:</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl apply -f calico-rbac-kdd.yaml
[root@master1 ~]# kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml
</code></pre>
<p>等待 <code>worker</code>节点pull好calico的image，状态主为：<code>Running</code>，表示部署成功</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl get pod --all-namespaces

NAMESPACE     NAME                                      READY   STATUS    RESTARTS   AGE
kube-system   calico-kube-controllers-85c867d48-c6qlc   1/1     Running   0          15m
kube-system   calico-node-5z9nj                         1/1     Running   0          15m
kube-system   calico-node-8gfsn                         1/1     Running   0          15m
</code></pre>
<hr>
<h3 id="h3-3-13-dns-coredns"><a name="3.13 部署DNS插件 coredns" class="reference-link"></a><span class="header-link octicon octicon-link"></span>3.13 部署DNS插件 coredns</h3><pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# vi coredns.yaml


apiVersion: v1
kind: ServiceAccount
metadata:
name: coredns
namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
labels:
kubernetes.io/bootstrapping: rbac-defaults
name: system:coredns
rules:
- apiGroups:
- ""
resources:
- endpoints
- services
- pods
- namespaces
verbs:
- list
- watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
annotations:
rbac.authorization.kubernetes.io/autoupdate: "true"
labels:
kubernetes.io/bootstrapping: rbac-defaults
name: system:coredns
roleRef:
apiGroup: rbac.authorization.k8s.io
kind: ClusterRole
name: system:coredns
subjects:
- kind: ServiceAccount
name: coredns
namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
name: coredns
namespace: kube-system
data:
Corefile: |
.:53 {
errors
health {
lameduck 5s
}
ready
kubernetes cluster.local in-addr.arpa ip6.arpa {
fallthrough in-addr.arpa ip6.arpa
}
prometheus :9153
forward . /etc/resolv.conf {
max_concurrent 1000
}
cache 30
loop
reload
loadbalance
}
---
apiVersion: apps/v1
kind: Deployment
metadata:
name: coredns
namespace: kube-system
labels:
k8s-app: kube-dns
kubernetes.io/name: "CoreDNS"
spec:
# replicas: not specified here:
# 1. Default is 1.
# 2. Will be tuned in real time if DNS horizontal auto-scaling is turned on.
strategy:
type: RollingUpdate
rollingUpdate:
maxUnavailable: 1
selector:
matchLabels:
k8s-app: kube-dns
template:
metadata:
labels:
k8s-app: kube-dns
spec:
priorityClassName: system-cluster-critical
serviceAccountName: coredns
tolerations:
- key: "CriticalAddonsOnly"
operator: "Exists"
nodeSelector:
kubernetes.io/os: linux
affinity:
podAntiAffinity:
preferredDuringSchedulingIgnoredDuringExecution:
- weight: 100
 podAffinityTerm:
   labelSelector:
	 matchExpressions:
	   - key: k8s-app
		 operator: In
		 values: ["kube-dns"]
   topologyKey: kubernetes.io/hostname
containers:
- name: coredns
image: coredns/coredns:1.7.0
imagePullPolicy: IfNotPresent
resources:
limits:
memory: 170Mi
requests:
cpu: 100m
memory: 70Mi
args: [ "-conf", "/etc/coredns/Corefile" ]
volumeMounts:
- name: config-volume
mountPath: /etc/coredns
readOnly: true
ports:
- containerPort: 53
name: dns
protocol: UDP
- containerPort: 53
name: dns-tcp
protocol: TCP
- containerPort: 9153
name: metrics
protocol: TCP
securityContext:
allowPrivilegeEscalation: false
capabilities:
add:
- NET_BIND_SERVICE
drop:
- all
readOnlyRootFilesystem: true
livenessProbe:
httpGet:
path: /health
port: 8080
scheme: HTTP
initialDelaySeconds: 60
timeoutSeconds: 5
successThreshold: 1
failureThreshold: 5
readinessProbe:
httpGet:
path: /ready
port: 8181
scheme: HTTP
dnsPolicy: Default
volumes:
- name: config-volume
configMap:
name: coredns
items:
- key: Corefile
  path: Corefile
---
apiVersion: v1
kind: Service
metadata:
name: kube-dns
namespace: kube-system
annotations:
prometheus.io/port: "9153"
prometheus.io/scrape: "true"
labels:
k8s-app: kube-dns
kubernetes.io/cluster-service: "true"
kubernetes.io/name: "CoreDNS"
spec:
selector:
k8s-app: kube-dns
clusterIP: 10.255.0.2
ports:
- name: dns
port: 53
protocol: UDP
- name: dns-tcp
port: 53
protocol: TCP
- name: metrics
port: 9153
protocol: TCP
</code></pre>
<p>安装<code>coredns</code></p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl apply -f coredns.yaml
</code></pre>
<p>查看是否成功：</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl get pod --all-namespaces | grep coredns
kube-system   coredns-7bf4bd64bd-gsfpk                  1/1     Running   0          16m
</code></pre>
<p>查看集群中的节点状态：</p>
<pre class=" language-sh"><code class=" language-sh">[root@master1 ~]# kubectl get node
NAME    STATUS   ROLES    AGE   VERSION
node1   Ready    &lt;none&gt;   11d   v1.19.16
node2   Ready    &lt;none&gt;   11d   v1.19.16
</code></pre>
<p>其中<code>STATUS</code>状态都为 <code>Ready</code>表示安装成功</p>
<hr>
<h2 id="h2-u7ED3u5C3E"><a name='end' class="reference-link"></a><span class="header-link octicon octicon-link"></span>结尾</h2><p>二进制的集群到此部署完毕，关于 <code>dashboard</code> 和 <code>ingress</code> 的部署，请参考 <a href="https://22v.net">22v.net</a> 上的其他文章。</p>
</div>
	`
)

func TestPopov(t *testing.T) {
	data := NewDirNode(content)
	for _, item := range data {
		fmt.Printf("%s%s\n", getSpace(item.Depth), item.Title)
	}
}

func getSpace(tag int) string {
	s := ""
	for i := 1; i < tag; i++ {
		s += "├─ "
	}
	return s
}
