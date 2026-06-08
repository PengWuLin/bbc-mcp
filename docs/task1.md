# 实现 bbc mcp 服务

##  要求
- 参考 example 目录下实例实现
## 
mcp 功能
- 查询当前 minibbc 数据中心 设备连接数
  - 通过执行 bbc-tool 工具，命令为 bbc-tool status gateway
  - 工具的路径通过配置文件配置
- 查询某个租户设备列表
  - 支持分页查询，每次最多返回20个设备数据 避免查询导致系统异常
  - 接口入参1为租户 id，参数名为 ccode
  - 接口入参2为设备名称，参数名为 name, 该参数为可选
  - 接口入参3为查询偏移量，参数名为 offset
  - 该接口返回参数为 name id
  - 数据库参数语句为 select id,name,zp_id,status from device where ccode = '租户 id' limit 20 offset 0
  - 如果参数带有 name，则数据库语句变为 select id,name,zp_id,status from device where ccode = '租户 id' and name like %设备名% limit 20 offset 0
  - 数据库的账号密码写死在代码
- 查下某个设备连接状态
  - 接口参数1为设备 id ，参数名为 id
  - 数据库的查询语句为 select * from device where id = '设备 id' limit 1;
  - 解析所有信息并返回，数据库表结构参考后面的 scloudb 数据库表结构 章节
  - 读取 redis 信息返回，key 参考 redis 信息章节




# scloudb 数据库表结构

```sql

 CREATE TABLE `device` (
    ->   `id` int(11) NOT NULL AUTO_INCREMENT COMMENT '主键ID(设备表)',
    ->   `gwid` varchar(100) DEFAULT NULL COMMENT '设备ID/网关ID(如果是HCI设备则是设备ID)',
    ->   `ccode` int(11) NOT NULL COMMENT '公司代码(企业ID)',
    ->   `type` smallint(6) NOT NULL COMMENT '设备类型(公司定义的名称)如下：1.ABOS 2.AC 3.AF 4.MIG 5.WOC 6.SG 7.EDR 8.CM(云镜) 9.CG（云眼）',
  `email` varchar(100) DEFAULT '' COMMENT '邮箱',
  `remark` varchar(1024) DEFAULT '' COMMENT '备注',
  `conn_code` varchar(1024) DEFAULT '' COMMENT '接入',
  `conn_code_gen_time` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '接入码生成时    -> 间',
  `add_type` tinyint(4) DEFAULT NULL COMMENT '设备添加方式',
  `name` varchar(120) NOT NULL COMMENT '设备名称(用户自定义，用于设备接入用户名)',
  KEY `idx_gwid` (`gwid`),
  KEY `ccode_and_orgid` (`ccode`,`org_id`),
  KEY `sn_index` (`sn`),
    ->   `pwd` varchar(1024) NOT NULL COMMENT '设备接入密码(跟所属分支保持一致)',
    ->   `model` varchar(100) DEFAULT NULL COMMENT '设备型号',
    ->   `is_nfv` tinyint(1) DEFAULT NULL COMMENT '是否为NFV设备',
    ->   `prod_line` smallint(6) DEFAULT NULL COMMENT '产品线',
    ->   `prod_name` varchar(100) DEFAULT NULL COMMENT '产品名称',
    ->   `version` varchar(1024) DEFAULT NULL COMMENT '当前版本',
    ->   `is_custom` tinyint(1) DEFAULT NULL COMMENT '是否是定制版本，0非定制版本、1定制版本',
    ->   `active_time` timestamp NULL DEFAULT NULL COMMENT '激活时间',
    ->   `lst_online_time` timestamp NULL DEFAULT NULL COMMENT '最近一次上线时间',
    ->   `lst_offline_time` timestamp NULL DEFAULT NULL COMMENT '最后一次离线时间',
    ->   `status` smallint(6) DEFAULT NULL COMMENT '设备当前状态，0未激活、1启用(2离线、3在线、4告警)、5停用',
    ->   `ip` varchar(100) DEFAULT NULL COMMENT '设备外网IP(支持IPv6)',
    ->   `mac` varchar(100) DEFAULT NULL COMMENT '设备接入外网mac',
    ->   `eth0_mac` varchar(100) DEFAULT NULL COMMENT '设备eth0的mac地址',
    ->   `zp_id` int(11) DEFAULT NULL COMMENT '接入服务器ID',
    ->   `branch_id` int(11) DEFAULT NULL COMMENT '分支ID',
    ->   `parent_id` int(11) DEFAULT '-1' COMMENT '父设备ID(如：vAC在aBos上运行)',
    ->   `tpl_id` int(11) DEFAULT NULL COMMENT '关联策略模版ID',
    ->   `license` text COMMENT '设备授权序列号',
    ->   `device_belong` int(11) NOT NULL DEFAULT '0' COMMENT '设备归属（0：老设备 1：安服 2：XDR 3：云图）',
    ->   `create_source` int(11) NOT NULL DEFAULT '0' COMMENT '创建来源（0：老设备 1：安服 2：XDR 3：云图）',
    ->   `update_source` int(11) NOT NULL DEFAULT '0' COMMENT '更新来源（0：老设备 1：安服 2：XDR 3：云图）',
    ->   `last_update_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    ->   `desc` text COMMENT '描述',
    ->   `sn` varchar(256) DEFAULT NULL COMMENT '硬件序列号',
    ->   `region_id` int(11) NOT NULL COMMENT '区域编码',
    ->   `tags` varchar(256) DEFAULT NULL COMMENT '设备标签',
    ->   `platforms` varchar(128) DEFAULT NULL COMMENT 'sase。服务之间 ，号分割，如sase,xdr',
    ->   `addr` text COMMENT '详细地址',
    ->   `order_id` varchar(256) DEFAULT NULL COMMENT '订单号',
    ->   `org_id` varchar(64) NOT NULL COMMENT '组织id',
    ->   `alias` varchar(120) NOT NULL COMMENT '设备别名，只是名称的作用',
    ->   `manager` varchar(100) DEFAULT '' COMMENT '负责人',
    ->   `email` varchar(100) DEFAULT '' COMMENT '邮箱',
    ->   `remark` varchar(1024) DEFAULT '' COMMENT '备注',
    ->   `conn_code` varchar(1024) DEFAULT '' COMMENT '接入',
    ->   `conn_code_gen_time` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '接入码生成时间',
    ->   `add_type` tinyint(4) DEFAULT NULL COMMENT '设备添加方式',
    ->   `create_time` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
    ->   `vid` varchar(32) NOT NULL DEFAULT '',
    ->   PRIMARY KEY (`id`),
    ->   UNIQUE KEY `unique_idx_ccode_type_name` (`ccode`,`type`,`name`),
    ->   KEY `idx_gwid` (`gwid`),
    ->   KEY `ccode_and_orgid` (`ccode`,`org_id`),
    ->   KEY `sn_index` (`sn`),
    ->   KEY `gwid` (`gwid`)
    -> ) ENGINE=InnoDB AUTO_INCREMENT=340574 DEFAULT CHARSET=utf8
    -> ;


```

```sql
select * from device where ccode ="56626662" and status != 1  limit 1\G
*************************** 1. row ***************************
                id: 260860
              gwid: 
             ccode: 56626662
              type: 3
              name: sz1_af_Connect_7000
               pwd: pzUW9ZOpS07x_PfF
             model: NULL
            is_nfv: 1
         prod_line: NULL
         prod_name: AF
           version: NULL
         is_custom: NULL
       active_time: NULL
   lst_online_time: NULL
  lst_offline_time: NULL
            status: 0
                ip: NULL
               mac: NULL
          eth0_mac: NULL
             zp_id: NULL
         branch_id: -1
         parent_id: -1
            tpl_id: -1
           license: NULL
     device_belong: 3
     create_source: 3
     update_source: 3
  last_update_time: 2026-06-05 14:52:19
              desc: NULL
                sn: 
         region_id: 0
              tags: NULL
         platforms: xcentral,sase
              addr: NULL
          order_id: 
            org_id: 0
             alias: sz1_af_Connect_7000
           manager: 
             email: 
            remark: 
         conn_code: 
conn_code_gen_time: 2026-04-28 13:39:24
          add_type: 0
       create_time: 2026-04-28 13:39:24
               vid: 
1 row in set (0.03 sec)
```

# redis 信息

## 基础信息
格式为 device:设备id:status:basic，如设备 223166 key 为 device:223166:status:basic
```bash
hgetall device:223166:status:basic
 1) "product_name"
 2) "AF"
 3) "tenant"
 4) "56626662"
 5) "bbc:heartbeat"
 6) "31693"
 7) "zp_id"
 8) "8001"
 9) "bandwidth"
10) "0"
11) "send"
12) "1866"
13) "mem"
14) "41"
15) "device_id"
16) "223166"
17) "last_online_time"
18) "2026-06-05-17:56:07"
19) "minibbc"
20) "true:2026-06-05-17:56:07"
21) "disk"
22) "64"
23) "session_num"
24) "20"
25) "recv"
26) "1128"
27) "vm_num"
28) "0"
29) "gateway_id"
30) "2WNPEKJG"
31) "cpu"
32) "98"
33) "user"
34) "10"
35) "version"
36) "8.0.7"
37) "last_offline_time"
38) "2026-05-14-16:41:11"
```

## 状态信息
格式为 device_设备id，如设备 223166 key 为 device_223166

```bash
 hgetall device_223166
1) "status"
2) "3"
```

## 测试数据库
```sql
INSERT INTO `device` (
    `id`, `gwid`, `ccode`, `type`, `name`, `pwd`, `model`, `is_nfv`,
    `prod_line`, `prod_name`, `version`, `is_custom`, `active_time`,
    `lst_online_time`, `lst_offline_time`, `status`, `ip`, `mac`,
    `eth0_mac`, `zp_id`, `branch_id`, `parent_id`, `tpl_id`, `license`,
    `device_belong`, `create_source`, `update_source`, `last_update_time`,
    `desc`, `sn`, `region_id`, `tags`, `platforms`, `addr`, `order_id`,
    `org_id`, `alias`, `manager`, `email`, `remark`, `conn_code`,
    `conn_code_gen_time`, `add_type`, `create_time`, `vid`
) VALUES (
    223166,
    '2WNPEKJG',
    '56626662',
    3,
    'pwl.af.simulator.china',
    'r2P+6j9vmRiY6Cr+',
    NULL,
    1,
    NULL,
    'AF',
    NULL,
    NULL,
    '2026-04-09 14:10:54',
    '2026-06-06 22:17:08',
    '2026-06-06 20:39:19',
    1,
    NULL,
    NULL,
    NULL,
    8001,
    217995,
    -1,
    -1,
    NULL,
    3,
    3,
    3,
    '2026-06-06 22:17:06',
    NULL,
    '',
    1190308,
    NULL,
    'xcentral,sase',
    '',
    '',
    0,
    'pwl.af.simulator.china',
    '',
    '',
    '',
    '',
    '2026-04-08 10:18:53',
    0,
    '2026-04-08 10:18:53',
    ''
);
```


```sql
HSET device_223166 status 3
```

```sql
HSET device:223166:status:basic \
  product_name "AF" \
  tenant "56626662" \
  "bbc:heartbeat" "31693" \
  zp_id "8001" \
  bandwidth "0" \
  send "1866" \
  mem "41" \
  device_id "223166" \
  last_online_time "2026-06-05-17:56:07" \
  minibbc "true:2026-06-05-17:56:07" \
  disk "64" \
  session_num "20" \
  recv "1128" \
  vm_num "0" \
  gateway_id "2WNPEKJG" \
  cpu "98" \
  user "10" \
  version "8.0.7" \
  last_offline_time "2026-05-14-16:41:11"
```