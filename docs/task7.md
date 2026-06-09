# 安全整改

## 背景

目前服务的 token 和 k8s 的 token 都是明文存储在配置文件，存在安全问题

## 方案

参考 design-task4.md 将 token 也加密保存