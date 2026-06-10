package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"bbc-mcp/internal/types"
)

type SMSPackageRepository struct {
	db *sql.DB
}

func NewSMSPackageRepository(db *sql.DB) *SMSPackageRepository {
	return &SMSPackageRepository{db: db}
}

func (r *SMSPackageRepository) QueryByCorpID(ctx context.Context, corpID int) ([]types.SMSPackage, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, corp_id, order_id, pkg_type, pkg_name,
		 pkg_total_num, pkg_avail_num, pkg_price, package_desc,
		 purchase_time, expire_time
		 FROM package WHERE corp_id = ?`, corpID)
	if err != nil {
		log.Printf("repository: 查询短信套餐失败(corp_id=%d): %v", corpID, err)
		return nil, fmt.Errorf("query sms package: %w", err)
	}
	defer rows.Close()

	var packages []types.SMSPackage
	for rows.Next() {
		var p types.SMSPackage
		if err := rows.Scan(&p.ID, &p.CorpID, &p.OrderID, &p.PkgType,
			&p.PkgName, &p.PkgTotalNum, &p.PkgAvailNum, &p.PkgPrice,
			&p.PackageDesc, &p.PurchaseTime, &p.ExpireTime); err != nil {
			log.Printf("repository: 扫描套餐行失败: %v", err)
			return nil, fmt.Errorf("scan package row: %w", err)
		}
		packages = append(packages, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if packages == nil {
		packages = []types.SMSPackage{}
	}

	return packages, nil
}
