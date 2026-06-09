# 工具整改
## 背景
直接调用 bbc-tool 工具太麻烦，需要依赖工具
业务存在多个 k8s 集群，需要部署多套 mcp 服务

## 需求

- 参考 bbc-tool 工具源码，将代码移植到 mcp 服务，bbt-tool 源码路径：D:\github\bbc-tool
- 原有的工具执行保留，通过配置文件方式确定获取设备连接数的方式
- 支持多个 k8s 集群，配置文件中配置k8s 集群的 ip 与 kubeconfig 配置