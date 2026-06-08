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

type DeviceRepository struct {
	db    *sql.DB
	redis *redis.Client
}

func NewDeviceRepository(db *sql.DB, rds *redis.Client) *DeviceRepository {
	return &DeviceRepository{db: db, redis: rds}
}

const listLimit = 20

func (r *DeviceRepository) ListByCcode(ctx context.Context, ccode string, name string, offset int) ([]types.DeviceItem, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var rows *sql.Rows
	var err error

	if name != "" {
		rows, err = r.db.QueryContext(ctx,
			"SELECT id, name, zp_id, status FROM device WHERE ccode = ? AND name LIKE ? LIMIT ? OFFSET ?",
			ccode, "%"+name+"%", listLimit, offset)
	} else {
		rows, err = r.db.QueryContext(ctx,
			"SELECT id, name, zp_id, status FROM device WHERE ccode = ? LIMIT ? OFFSET ?",
			ccode, listLimit, offset)
	}
	if err != nil {
		log.Printf("repository: 查询设备列表失败: %v", err)
		return nil, fmt.Errorf("query device list: %w", err)
	}
	defer rows.Close()

	var devices []types.DeviceItem
	for rows.Next() {
		var d types.DeviceItem
		if err := rows.Scan(&d.ID, &d.Name, &d.ZpID, &d.Status); err != nil {
			log.Printf("repository: 扫描设备行失败: %v", err)
			return nil, fmt.Errorf("scan device row: %w", err)
		}
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		log.Printf("repository: 行迭代错误: %v", err)
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if devices == nil {
		devices = []types.DeviceItem{}
	}

	return devices, nil
}

func (r *DeviceRepository) GetByID(ctx context.Context, id int) (*types.Device, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	d := &types.Device{}
	row := r.db.QueryRowContext(ctx,
		`SELECT id, gwid, ccode, type, name, pwd, model, is_nfv, prod_line, prod_name,
		 version, is_custom, active_time, lst_online_time, lst_offline_time,
		 status, ip, mac, eth0_mac, zp_id, branch_id, parent_id, tpl_id,
		 license, device_belong, create_source, update_source, last_update_time,
		 `+"`desc`"+`, sn, region_id, tags, platforms, addr, order_id,
		 org_id, alias, manager, email, remark, conn_code, conn_code_gen_time,
		 add_type, create_time, vid
		 FROM device WHERE id = ? LIMIT 1`, id)

	err := row.Scan(
		&d.ID, &d.GwID, &d.Ccode, &d.Type, &d.Name, &d.Pwd, &d.Model, &d.IsNfv,
		&d.ProdLine, &d.ProdName, &d.Version, &d.IsCustom, &d.ActiveTime,
		&d.LstOnlineTime, &d.LstOfflineTime, &d.Status, &d.IP, &d.Mac, &d.Eth0Mac,
		&d.ZpID, &d.BranchID, &d.ParentID, &d.TplID, &d.License, &d.DeviceBelong,
		&d.CreateSource, &d.UpdateSource, &d.LastUpdateTime, &d.Desc, &d.SN,
		&d.RegionID, &d.Tags, &d.Platforms, &d.Addr, &d.OrderID, &d.OrgID,
		&d.Alias, &d.Manager, &d.Email, &d.Remark, &d.ConnCode, &d.ConnCodeGenTime,
		&d.AddType, &d.CreateTime, &d.VID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Printf("repository: 查询设备(id=%d)失败: %v", id, err)
		return nil, fmt.Errorf("query device by id: %w", err)
	}

	return d, nil
}

func (r *DeviceRepository) GetDeviceBasicInfo(ctx context.Context, id int) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	key := fmt.Sprintf("device:%d:status:basic", id)
	result, err := r.redis.HGetAll(ctx, key).Result()
	if err != nil {
		log.Printf("repository: Redis HGETALL %s 失败: %v", key, err)
		return nil, fmt.Errorf("redis hgetall %s: %w", key, err)
	}
	return result, nil
}

func (r *DeviceRepository) GetDeviceOnlineStatus(ctx context.Context, id int) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	key := fmt.Sprintf("device_%d", id)
	result, err := r.redis.HGetAll(ctx, key).Result()
	if err != nil {
		log.Printf("repository: Redis HGETALL %s 失败: %v", key, err)
		return nil, fmt.Errorf("redis hgetall %s: %w", key, err)
	}
	return result, nil
}
