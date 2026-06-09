package prompt

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

const bbcGuideContent = `## 一、服务概述

你正在使用 BBC MCP 服务，该服务提供对 minibbc 数据中心设备信息的安全查询能力。
通过以下三个 MCP Tool，你可以查询设备连接数、租户设备列表和设备实时连接状态。

## 二、设备数据库字段说明

` + "`" + `device` + "`" + ` 表字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 设备主键 ID |
| gwid | string | 网关设备 ID |
| ccode | int | 公司/租户 ID |
| type | int | 设备类型，见类型码映射表 |
| name | string | 设备名称（用户自定义） |
| pwd | string | 设备密码（总是返回 "***"） |
| model | string | 设备型号 |
| is_nfv | bool | 是否 NFV 设备 |
| prod_line | int | 产品线编号 |
| prod_name | string | 产品名称 |
| version | string | 当前软件版本 |
| is_custom | bool | 是否定制版本（0=否, 1=是） |
| active_time | timestamp | 设备激活时间 |
| lst_online_time | timestamp | 最近一次上线时间 |
| lst_offline_time | timestamp | 最近一次离线时间 |
| status | int | 设备状态，见状态码映射表 |
| ip | string | 外网 IP 地址 |
| mac | string | 外网 MAC 地址 |
| eth0_mac | string | eth0 网卡 MAC 地址 |
| zp_id | int | 接入服务器 ID，200 为奥飞机房，400 为深圳四区，800 为深圳一区 |
| branch_id | int | 分支机构 ID |
| parent_id | int | 父设备 ID（-1 表示无） |
| tpl_id | int | 策略模板 ID |
| license | string | 授权序列号 |
| device_belong | int | 设备归属：0=旧设备, 1=安服(AnFu), 2=XDR, 3=云图(YunTu) |
| create_source | int | 创建来源 |
| update_source | int | 更新来源 |
| last_update_time | timestamp | 最后更新时间 |
| desc | string | 设备描述 |
| sn | string | 硬件序列号 |
| region_id | int | 区域编码 |
| tags | string | 设备标签 |
| platforms | string | 平台标识（逗号分隔，如 sase,xdr） |
| addr | string | 详细地址 |
| order_id | string | 订单编号 |
| org_id | string | 组织 ID |
| alias | string | 设备别名 |
| manager | string | 负责人 |
| email | string | 负责人邮箱 |
| remark | string | 备注 |
| conn_code | string | 连接码 |
| conn_code_gen_time | datetime | 连接码生成时间 |
| add_type | int | 添加方式 |
| create_time | timestamp | 创建时间 |
| vid | string | V-ID |

**设备类型码映射（type 字段）：**

| 编码 | 设备类型 |
|------|----------|
| 1 | ABOS |
| 2 | AC (Access Controller) |
| 3 | AF (Application Firewall) |
| 4 | MIG |
| 5 | WOC |
| 6 | SG |
| 7 | EDR |
| 8 | CM |
| 9 | CG |

**设备状态码映射（status 字段）：**

| 编码 | 状态 | 说明 |
|------|------|------|
| 0 | 未激活 | 设备尚未激活使用 |
| 1 | 已启用 | 聚合状态，包含离线/在线/告警子状态 |
| 2 | 离线 | 设备当前不在线 |
| 3 | 在线 | 设备正常运行中 |
| 4 | 告警 | 设备在线但存在告警 |
| 5 | 停用 | 设备已被停用 |

**设备归属映射（device_belong 字段）：**

| 编码 | 归属 |
|------|------|
| 0 | 旧设备 |
| 1 | 安服 (AnFu) |
| 2 | XDR |
| 3 | 云图 (YunTu) |

## 三、Redis 实时数据字段说明

device_status Tool 返回的 realtime 字段来自 Redis，字段含义如下：

| Redis 字段 | 说明 |
|------------|------|
| product_name | 产品名称（如 AF, AC 等） |
| tenant | 租户 ID（对应 ccode） |
| zp_id | 接入服务器 ID，200 为奥飞机房，400 为深圳四区，800 为深圳一区 |
| cpu | CPU 使用率（百分比） |
| mem | 内存使用率（百分比） |
| disk | 磁盘使用率（百分比） |
| version | 当前软件版本 |
| session_num | 会话数 |
| user | 在线用户数 |
| bandwidth | 带宽占用 |
| last_online_time | 最近上线时间 |
| last_offline_time | 最近离线时间 |
| status | 实时状态（编码同上表） |
| bbc:heartbeat | BBC 心跳标识 |
| device_id | 设备 ID |
| gateway_id | 网关 ID |
| send | 发送流量 |
| recv | 接收流量 |
| vm_num | 虚拟机数量 |
| minibbc | minibbc 标识 |

## 四、Tool 使用说明

### 4.1 gateway_status — 查询设备连接数

功能：查询 minibbc 数据中心网关各 Pod 的 ESTABLISHED 连接数和设备数。

参数：
  - cluster (string, 可选): K8s 集群名称，不填则查询所有已配置集群

返回值（native 模式）：
  JSON 数组，每项包含：
    - cluster: 集群名称
    - pods: Pod 连接信息列表 [{pod_name, status, connections, devices, error}]
    - connections: TCP ESTABLISHED 连接数
    - devices: connections / 3（每设备 3 个连接）

注意事项：
  - 若 Pod 状态非 Running，其 error 字段会有说明
  - 若指定的 cluster 名称不存在，返回错误提示
  - CLI 模式下返回 bbc-tool 命令原始输出

### 4.2 device_list — 查询租户设备列表

功能：根据租户 ID 查询设备列表，支持按名称模糊搜索和分页。

参数：
  - ccode (string, 必填): 租户 ID（公司代码），不能为空
  - name (string, 可选): 设备名称，支持模糊匹配
  - offset (integer, 必填): 分页偏移量，负数自动置为 0

返回值 JSON 对象：
  {
    "total": true,
    "devices": [
      {"id": 260860, "name": "sz1_af_Connect_7000", "zp_id": null, "status": 0}
    ],
    "limit": 20,
    "offset": 0
  }

注意事项：
  - 每页固定返回最多 20 条记录，LIMIT 不可被调用方覆盖
  - name 参数为空时返回该租户所有设备（分页）
  - name 不为空时执行 LIKE %name% 模糊查询
  - 若 ccode 为空字符串，返回参数错误
  - device_list 不返回 type 字段，需使用 device_status 查询设备类型

### 4.3 device_status — 查询设备详情与实时状态

功能：根据设备 ID 查询设备完整信息（MySQL）和实时连接状态（Redis）。

参数：
  - id (integer, 必填): 设备 ID

返回值 JSON 对象：
  {
    "device": {
      "id": 223166,
      "ccode": 56626662,
      "type": 3,
      "name": "sz1_af_Connect_7000",
      "pwd": "***",
      "status": 3,
      ...（共 35 个字段，pwd 已脱敏）
    },
    "realtime": {
      "product_name": "AF",
      "cpu": "98",
      "mem": "41",
      "status": "3",
      ...（共 18 个字段）
    }
  }

注意事项：
  - pwd 字段永远返回 "***"，不会泄露真实密码
  - realtime 字段可能为 null（Redis 不可用时），不视为错误
  - realtime 中的 status 为实时采集值，优先于 MySQL 中的持久化状态
  - 设备不存在时返回错误："设备不存在"
  - 查询超时时间：MySQL 5s，Redis 3s

## 五、注意事项

1. 所有查询都有超时限制（数据库 5s，Redis 3s，K8s exec 30s）
2. 服务有全局 QPS=1 的限流，并发请求会收到"服务繁忙"错误，请串行调用
3. 请求需要 Bearer Token 认证：Authorization: Bearer <token>
4. device_list 和 device_status 配合使用：
   - 先用 device_list 获取设备列表（返回 id, name, zp_id, status）
   - 再用 device_status 获取单个设备的详细信息和实时状态
5. status 字段在 MySQL 和 Redis 中独立存在：
   - MySQL status 为持久化状态，更新有延迟
   - Redis status 为实时采集状态，请优先使用 Redis 的值
6. zp_id 机房位置映射：200=奥飞机房，400=深圳四区，800=深圳一区
`

// NewBBCGuidePrompt returns the prompt definition and handler for the bbc-guide prompt.
func NewBBCGuidePrompt() (mcp.Prompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error)) {
	p := mcp.NewPrompt("bbc-guide",
		mcp.WithPromptDescription("BBC MCP 服务使用指南：包含数据库字段说明、设备/状态码映射、Tool 使用说明"),
	)
	handler := func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return mcp.NewGetPromptResult(
			"BBC MCP 服务使用指南",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleAssistant, mcp.NewTextContent(bbcGuideContent)),
			},
		), nil
	}
	return p, handler
}
