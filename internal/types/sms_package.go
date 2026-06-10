package types

import "time"

type SMSPackage struct {
	ID           int       `json:"id"`
	CorpID       int       `json:"corp_id"`
	OrderID      string    `json:"order_id"`
	PkgType      int       `json:"pkg_type"`
	PkgName      string    `json:"pkg_name"`
	PkgTotalNum  int       `json:"pkg_total_num"`
	PkgAvailNum  int       `json:"pkg_avail_num"`
	PkgPrice     float64   `json:"pkg_price"`
	PackageDesc  string    `json:"package_desc"`
	PurchaseTime time.Time `json:"purchase_time"`
	ExpireTime   time.Time `json:"expire_time"`
}
