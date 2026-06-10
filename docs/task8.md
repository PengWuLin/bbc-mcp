# 新增短信云套餐查询工具
## 背景
短信云业务需要通过 mcp 服务进行查询租户短信云套餐信息

## 方案
### 数据库说明
- 数据库为独立数据库，与 scloudb 不是同一个数据库，配置需要能够支持配置多个数据库配置

### 数据表说明

```sql
package | CREATE TABLE `package` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `corp_id` int(11) NOT NULL DEFAULT '0',
  `order_id` varchar(45) NOT NULL DEFAULT '',
  `pkg_type` int(11) NOT NULL DEFAULT '1',
  `pkg_name` varchar(32) NOT NULL DEFAULT '',
  `pkg_total_num` int(11) NOT NULL DEFAULT '0',
  `pkg_avail_num` int(11) NOT NULL DEFAULT '0',
  `pkg_price` double NOT NULL DEFAULT '0',
  `package_desc` varchar(255) NOT NULL DEFAULT '',
  `purchase_time` datetime NOT NULL,
  `expire_time` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2729 DEFAULT CHARSET=utf8
```
#### 例子

```bash
select * from package limit 1 \G
*************************** 1. row ***************************
           id: 1292
      corp_id: 26912728
     order_id: 201905178888
     pkg_type: 1
     pkg_name: 50条短信套餐
pkg_total_num: 50
pkg_avail_num: 0
    pkg_price: 2.4
 package_desc: 套餐允许发送50条短信
purchase_time: 2019-05-17 22:44:57
  expire_time: 2025-01-01 23:59:00
1 row in set (0.00 sec)
```

### 功能
- 支持查询某个租户的短信套餐
  - 接口参数1为租户id ，参数名为 corp_id
  - 返回数据将数据库所有字段都返回
  - expire_time 是套餐过期时间
  - pkg_total_num 是当前短信条数