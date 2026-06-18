package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"bbc-mcp/internal/types"
)

type SMSPackageRepository struct {
	db    *sql.DB
	redis *redis.Client
}

func NewSMSPackageRepository(db *sql.DB, rds *redis.Client) *SMSPackageRepository {
	return &SMSPackageRepository{db: db, redis: rds}
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

func (r *SMSPackageRepository) RenewExpireTime(ctx context.Context, id int, ccode string) error {
	ctxDB, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	expireTime := time.Now().AddDate(1, 0, 0).Format("2006-01-02") + " 23:59:00"
	result, err := r.db.ExecContext(ctxDB,
		"UPDATE package SET expire_time=? WHERE id=? LIMIT 1",
		expireTime, id)
	if err != nil {
		log.Printf("repository: 续期套餐失败(id=%d): %v", id, err)
		return fmt.Errorf("renew sms package: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("套餐不存在: id=%d", id)
	}

	if r.redis != nil {
		ctxRedis, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		key := fmt.Sprintf("csms:corp:%s:package:availnum", ccode)
		if err := r.redis.Del(ctxRedis, key).Err(); err != nil {
			log.Printf("repository: 删除 Redis 缓存失败(key=%s): %v", key, err)
		}
	}

	return nil
}
