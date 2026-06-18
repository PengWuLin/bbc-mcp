# 短信云续期
## 背景
 对指定租户短信续期

## 技术方案
### 数据库说明
- 数据库为 csmsdb 数据库
- redis 为独立 redis，需要能够单独配置 ip 密码等配置

### 短信云续费
- 入参1为套餐 id，参数名为 id
- 入参2为租户 id，参数名为 ccode
- 续费步骤1，更新 mysql，sql 语句为 `update package set expire_time='2027-02-05 23:59:00' where id=2670 limit 1; `
  - 默认时间为当前时间加一年
  - id 为入参，套餐 id
- 删除 redis 缓存，redis key 的规则为 csms:corp:51455869:package:availnum，其中 51455869 为租户 ccode



