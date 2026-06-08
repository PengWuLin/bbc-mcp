package types

import "time"

// DeviceItem 设备列表返回用结构
type DeviceItem struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	ZpID   *int   `json:"zp_id"`
	Status *int   `json:"status"`
}

// Device 设备完整信息，映射 device 表全字段
type Device struct {
	ID              int        `json:"id"`
	GwID            string     `json:"gwid"`
	Ccode           int        `json:"ccode"`
	Type            int        `json:"type"`
	Name            string     `json:"name"`
	Pwd             string     `json:"-"`
	Model           *string    `json:"model"`
	IsNfv           *bool      `json:"is_nfv"`
	ProdLine        *int       `json:"prod_line"`
	ProdName        *string    `json:"prod_name"`
	Version         *string    `json:"version"`
	IsCustom        *bool      `json:"is_custom"`
	ActiveTime      *time.Time `json:"active_time"`
	LstOnlineTime   *time.Time `json:"lst_online_time"`
	LstOfflineTime  *time.Time `json:"lst_offline_time"`
	Status          *int       `json:"status"`
	IP              *string    `json:"ip"`
	Mac             *string    `json:"mac"`
	Eth0Mac         *string    `json:"eth0_mac"`
	ZpID            *int       `json:"zp_id"`
	BranchID        *int       `json:"branch_id"`
	ParentID        *int       `json:"parent_id"`
	TplID           *int       `json:"tpl_id"`
	License         *string    `json:"license"`
	DeviceBelong    int        `json:"device_belong"`
	CreateSource    int        `json:"create_source"`
	UpdateSource    int        `json:"update_source"`
	LastUpdateTime  *time.Time `json:"last_update_time"`
	Desc            *string    `json:"desc"`
	SN              *string    `json:"sn"`
	RegionID        int        `json:"region_id"`
	Tags            *string    `json:"tags"`
	Platforms       *string    `json:"platforms"`
	Addr            *string    `json:"addr"`
	OrderID         *string    `json:"order_id"`
	OrgID           string     `json:"org_id"`
	Alias           string     `json:"alias"`
	Manager         *string    `json:"manager"`
	Email           *string    `json:"email"`
	Remark          *string    `json:"remark"`
	ConnCode        *string    `json:"conn_code"`
	ConnCodeGenTime *time.Time `json:"conn_code_gen_time"`
	AddType         *int       `json:"add_type"`
	CreateTime      *time.Time `json:"create_time"`
	VID             string     `json:"vid"`
}

// DeviceResponse device_status 返回的设备信息（Pwd 脱敏）
type DeviceResponse struct {
	ID              int        `json:"id"`
	GwID            string     `json:"gwid"`
	Ccode           int        `json:"ccode"`
	Type            int        `json:"type"`
	Name            string     `json:"name"`
	Pwd             string     `json:"pwd"`
	Model           *string    `json:"model"`
	IsNfv           *bool      `json:"is_nfv"`
	ProdLine        *int       `json:"prod_line"`
	ProdName        *string    `json:"prod_name"`
	Version         *string    `json:"version"`
	IsCustom        *bool      `json:"is_custom"`
	ActiveTime      *time.Time `json:"active_time"`
	LstOnlineTime   *time.Time `json:"lst_online_time"`
	LstOfflineTime  *time.Time `json:"lst_offline_time"`
	Status          *int       `json:"status"`
	IP              *string    `json:"ip"`
	Mac             *string    `json:"mac"`
	Eth0Mac         *string    `json:"eth0_mac"`
	ZpID            *int       `json:"zp_id"`
	BranchID        *int       `json:"branch_id"`
	ParentID        *int       `json:"parent_id"`
	TplID           *int       `json:"tpl_id"`
	License         *string    `json:"license"`
	DeviceBelong    int        `json:"device_belong"`
	CreateSource    int        `json:"create_source"`
	UpdateSource    int        `json:"update_source"`
	LastUpdateTime  *time.Time `json:"last_update_time"`
	Desc            *string    `json:"desc"`
	SN              *string    `json:"sn"`
	RegionID        int        `json:"region_id"`
	Tags            *string    `json:"tags"`
	Platforms       *string    `json:"platforms"`
	Addr            *string    `json:"addr"`
	OrderID         *string    `json:"order_id"`
	OrgID           string     `json:"org_id"`
	Alias           string     `json:"alias"`
	Manager         *string    `json:"manager"`
	Email           *string    `json:"email"`
	Remark          *string    `json:"remark"`
	ConnCode        *string    `json:"conn_code"`
	ConnCodeGenTime *time.Time `json:"conn_code_gen_time"`
	AddType         *int       `json:"add_type"`
	CreateTime      *time.Time `json:"create_time"`
	VID             string     `json:"vid"`
}

// DeviceStatusResult device_status 完整返回结构
type DeviceStatusResult struct {
	Device   *DeviceResponse    `json:"device"`
	Realtime map[string]string  `json:"realtime"`
}

// DeviceListResult device_list 返回结构
type DeviceListResult struct {
	Total   bool         `json:"total"`
	Devices []DeviceItem `json:"devices"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
}

func SanitizeDevice(d *Device) *DeviceResponse {
	return &DeviceResponse{
		ID:              d.ID,
		GwID:            d.GwID,
		Ccode:           d.Ccode,
		Type:            d.Type,
		Name:            d.Name,
		Pwd:             "***",
		Model:           d.Model,
		IsNfv:           d.IsNfv,
		ProdLine:        d.ProdLine,
		ProdName:        d.ProdName,
		Version:         d.Version,
		IsCustom:        d.IsCustom,
		ActiveTime:      d.ActiveTime,
		LstOnlineTime:   d.LstOnlineTime,
		LstOfflineTime:  d.LstOfflineTime,
		Status:          d.Status,
		IP:              d.IP,
		Mac:             d.Mac,
		Eth0Mac:         d.Eth0Mac,
		ZpID:            d.ZpID,
		BranchID:        d.BranchID,
		ParentID:        d.ParentID,
		TplID:           d.TplID,
		License:         d.License,
		DeviceBelong:    d.DeviceBelong,
		CreateSource:    d.CreateSource,
		UpdateSource:    d.UpdateSource,
		LastUpdateTime:  d.LastUpdateTime,
		Desc:            d.Desc,
		SN:              d.SN,
		RegionID:        d.RegionID,
		Tags:            d.Tags,
		Platforms:       d.Platforms,
		Addr:            d.Addr,
		OrderID:         d.OrderID,
		OrgID:           d.OrgID,
		Alias:           d.Alias,
		Manager:         d.Manager,
		Email:           d.Email,
		Remark:          d.Remark,
		ConnCode:        d.ConnCode,
		ConnCodeGenTime: d.ConnCodeGenTime,
		AddType:         d.AddType,
		CreateTime:      d.CreateTime,
		VID:             d.VID,
	}
}
