package service

import (
	"context"
	"log"
	"strings"
	"time"

	"m8-track-go/internal/model"
	"m8-track-go/internal/repository"
	"m8-track-go/internal/trackapi"
)

// TrackSyncService 核心同步服务，移植自 Java TrackSyncServiceImpl
type TrackSyncService struct {
	shipOrderRepo *repository.ShipOrderRepo
	recordRepo    *repository.TrackRecordRepo
	detailRepo    *repository.TrackDetailRepo
	trackClient   *trackapi.Client
	batchSize     int
	destZone      *time.Location // 目的地时区（未签收时发货日的兜底时区）
}

// NewTrackSync 创建同步服务
func NewTrackSync(
	shipOrderRepo *repository.ShipOrderRepo,
	recordRepo *repository.TrackRecordRepo,
	detailRepo *repository.TrackDetailRepo,
	trackClient *trackapi.Client,
	batchSize int,
	destZone *time.Location,
) *TrackSyncService {
	return &TrackSyncService{
		shipOrderRepo: shipOrderRepo,
		recordRepo:    recordRepo,
		detailRepo:    detailRepo,
		trackClient:   trackClient,
		batchSize:     batchSize,
		destZone:      destZone,
	}
}

// RegisterPendingOrders 注册待处理的运单到 17track
func (s *TrackSyncService) RegisterPendingOrders(ctx context.Context) error {
	orders, err := s.shipOrderRepo.SelectPendingOrders(ctx)
	if err != nil {
		return err
	}
	if len(orders) == 0 {
		log.Println("无待注册运单")
		return nil
	}

	// 运单号变更对账：scBNDtl.FID 不变但 MDNo 变了 → 清掉旧 record + 旧号 detail，
	// 重置 TrackDelivered=0，让新号在本轮重新注册并跟踪（若新号也已签收则由 17track 返回 Delivered 自愈）。
	for _, o := range orders {
		if o.OldMDNo == nil {
			continue
		}
		old := strings.TrimSpace(*o.OldMDNo)
		if old == "" || old == o.MDNo {
			continue
		}
		log.Printf("检测到运单号变更 dtlFID=%d 旧号=%s 新号=%s，清理旧数据并重新跟踪", o.DtlFID, old, o.MDNo)
		if err := s.recordRepo.DeleteByDtlFID(ctx, o.DtlFID); err != nil {
			log.Printf("删除旧同步记录失败 dtlFID=%d: %v", o.DtlFID, err)
		}
		if err := s.detailRepo.DeleteByMDNo(ctx, old); err != nil {
			log.Printf("删除旧轨迹详情失败 mdNo=%s: %v", old, err)
		}
		if err := s.shipOrderRepo.ResetDeliveredByFID(ctx, o.DtlFID); err != nil {
			log.Printf("重置签收状态失败 dtlFID=%d: %v", o.DtlFID, err)
		}
	}

	// 收集所有运单号
	allMDNos := make([]string, 0, len(orders))
	mdNoSet := make(map[string]bool)
	for _, o := range orders {
		if !mdNoSet[o.MDNo] {
			allMDNos = append(allMDNos, o.MDNo)
			mdNoSet[o.MDNo] = true
		}
	}

	// 查询已注册的运单
	records, err := s.recordRepo.GetByMDNos(ctx, allMDNos)
	if err != nil {
		return err
	}
	registeredSet := make(map[string]bool)
	for _, r := range records {
		registeredSet[r.MDNo] = true
	}

	// 过滤出未注册的
	var toRegister []string
	for _, mdNo := range allMDNos {
		if !registeredSet[mdNo] {
			toRegister = append(toRegister, mdNo)
		}
	}

	if len(toRegister) == 0 {
		log.Println("所有运单均已注册，无需重复注册")
		return nil
	}

	// 构建 mdNo -> FID / dtlFID / carrier 映射
	mdNoToFID := make(map[string]string)
	mdNoToDtlFID := make(map[string]int64)
	mdNoToCarrier := make(map[string]int)
	for _, o := range orders {
		mdNoToFID[o.MDNo] = o.FID
		mdNoToDtlFID[o.MDNo] = o.DtlFID
		if o.FCKeY != nil {
			mdNoToCarrier[o.MDNo] = *o.FCKeY
		}
	}

	log.Printf("待注册运单数量: %d", len(toRegister))

	batches := trackapi.Partition(toRegister, s.batchSize)
	for _, batch := range batches {
		batchCarrierMap := make(map[string]int, len(batch))
		for _, mdNo := range batch {
			if carrier, ok := mdNoToCarrier[mdNo]; ok {
				batchCarrierMap[mdNo] = carrier
			} else {
				batchCarrierMap[mdNo] = 0
			}
		}

		result, err := s.trackClient.RegisterWithCarrier(ctx, batchCarrierMap)
		if err != nil {
			log.Printf("注册运单失败: %v", err)
			continue
		}

		log.Printf("本批次注册结果: 新注册 %d, 已存在 %d, 失败 %d (共 %d)",
			len(result.Accepted), len(result.AlreadyRegistered), len(result.Failed), len(batch))

		now := time.Now()

		// 新注册成功的运单
		for _, mdNo := range result.Accepted {
			dtlFID := mdNoToDtlFID[mdNo]
			record := &model.TrackSyncRecord{
				FID:         mdNoToFID[mdNo],
				DtlFID:      &dtlFID,
				MDNo:        mdNo,
				IsDelivered: false,
				CreateTime:  now,
				UpdateTime:  now,
			}
			if err := s.recordRepo.Insert(ctx, record); err != nil {
				log.Printf("插入同步记录失败 mdNo=%s: %v", mdNo, err)
			}
		}

		// 已注册过的运单：本地补录记录，后续同步可正常查询轨迹
		for _, mdNo := range result.AlreadyRegistered {
			dtlFID := mdNoToDtlFID[mdNo]
			record := &model.TrackSyncRecord{
				FID:         mdNoToFID[mdNo],
				DtlFID:      &dtlFID,
				MDNo:        mdNo,
				IsDelivered: false,
				CreateTime:  now,
				UpdateTime:  now,
			}
			if err := s.recordRepo.Insert(ctx, record); err != nil {
				log.Printf("补录已注册运单失败 mdNo=%s: %v", mdNo, err)
			} else {
				log.Printf("补录已注册运单: mdNo=%s", mdNo)
			}
		}

		// 真正失败的运单：标记"查询不到"
		for _, mdNo := range result.Failed {
			if err := s.shipOrderRepo.UpdateFCtrackByFID(ctx, mdNoToDtlFID[mdNo], "查询不到"); err != nil {
				log.Printf("更新FCtrack失败 mdNo=%s: %v", mdNo, err)
			}
			log.Printf("运单 %s 注册失败，已写入FCtrack", mdNo)
		}
	}
	return nil
}

// SyncTrackingInfo 同步已注册运单的轨迹信息
func (s *TrackSyncService) SyncTrackingInfo(ctx context.Context) error {
	orders, err := s.shipOrderRepo.SelectPendingOrders(ctx)
	if err != nil {
		return err
	}
	if len(orders) == 0 {
		log.Println("无需同步的运单")
		return nil
	}

	allMDNos := make([]string, 0, len(orders))
	mdNoSet := make(map[string]bool)
	for _, o := range orders {
		if !mdNoSet[o.MDNo] {
			allMDNos = append(allMDNos, o.MDNo)
			mdNoSet[o.MDNo] = true
		}
	}

	records, err := s.recordRepo.GetByMDNos(ctx, allMDNos)
	if err != nil {
		return err
	}
	recordMap := make(map[string]*model.TrackSyncRecord)
	registeredMDNos := make([]string, 0, len(records))
	for i := range records {
		rec := records[i]
		recordMap[rec.MDNo] = &rec
		registeredMDNos = append(registeredMDNos, rec.MDNo)
	}

	// mdNo -> ERP 明细行（含客户收货信息），签收后回写地点/匹配结果需要
	orderMap := make(map[string]*model.ShipOrder, len(orders))
	for i := range orders {
		o := orders[i]
		if _, ok := orderMap[o.MDNo]; !ok {
			orderMap[o.MDNo] = &o
		}
	}

	if len(registeredMDNos) == 0 {
		log.Println("暂无已注册运单可同步")
		return nil
	}

	log.Printf("同步轨迹运单数量: %d", len(registeredMDNos))
	batches := trackapi.Partition(registeredMDNos, s.batchSize)
	for _, batch := range batches {
		infoList, err := s.trackClient.GetTrackInfo(ctx, batch)
		if err != nil {
			log.Printf("获取轨迹失败: %v", err)
			continue
		}
		for _, info := range infoList {
			rec := recordMap[info.Number]
			if err := s.processTrackInfo(ctx, &info, rec, orderMap[info.Number]); err != nil {
				log.Printf("处理运单 %s 轨迹异常，跳过: %v", info.Number, err)
			}
		}
	}
	return nil
}

// SyncAll 先注册再同步
func (s *TrackSyncService) SyncAll(ctx context.Context) error {
	if err := s.RegisterPendingOrders(ctx); err != nil {
		log.Printf("注册运单失败: %v", err)
	}
	return s.SyncTrackingInfo(ctx)
}

// processTrackInfo 处理单条轨迹信息：
//  1. 提取最新状态/事件；
//  2. 持久化全部轨迹事件（按 本地墙上时钟分钟+stage+描述 去重，避免重复同步重复入库）；
//  3. 推导发货/签收日期（按目的地时区），回写 ERP scBNDtl 与本地 record；
//  4. 签收则标记 TrackDelivered=1，并把签收地点及与客户收货信息的匹配结果
//     回写 scBNDtl.location / scBNDtl.locationmatch（仅签收后同步）；
//  5. 更新 track_sync_record。
func (s *TrackSyncService) processTrackInfo(ctx context.Context, info *model.Track17TrackInfo, record *model.TrackSyncRecord, order *model.ShipOrder) error {
	if record == nil {
		return nil
	}

	mdNo := info.Number
	var newStatus *string
	var newEvent *string
	var newEventTime *time.Time

	if info.TrackInfo != nil {
		if info.TrackInfo.LatestStatus != nil {
			newStatus = &info.TrackInfo.LatestStatus.Status
		}
		if info.TrackInfo.LatestEvent != nil {
			newEvent = &info.TrackInfo.LatestEvent.Description
			if info.TrackInfo.LatestEvent.TimeISO != "" {
				if t, err := time.Parse(time.RFC3339, info.TrackInfo.LatestEvent.TimeISO); err == nil {
					newEventTime = &t
				}
			}
		}
	}

	// 全量轨迹事件持久化（取代只记最近一条）；最新事件取收集后按时间升序的最后一条
	collected := CollectEvents(info)
	// 本地已有明细（按事件时间倒序）：供去重比对，签收且本轮无事件时作地点兜底
	existing, err := s.detailRepo.ListByMDNo(ctx, mdNo, 1000)
	if err != nil {
		log.Printf("查询已有轨迹详情失败 mdNo=%s: %v", mdNo, err)
		existing = nil
	}
	if len(collected) > 0 {
		existingKeys := make(map[string]struct{}, len(existing))
		for _, d := range existing {
			ev := NormalizedEvent{Desc: derefStr(d.EventDesc), Stage: derefStr(d.Stage)}
			if d.EventTime != nil {
				ev.Time = *d.EventTime
			}
			existingKeys[eventDedupKey(ev)] = struct{}{}
		}
		inserted := 0
		now := time.Now()
		for _, ev := range collected {
			if _, ok := existingKeys[eventDedupKey(ev)]; ok {
				continue
			}
			// track_status 取每条事件自身的阶段（ev.Stage），不能用整票货的 LatestStatus，
			// 否则已签收运单的所有事件都会被写成 Delivered。
			detail := &model.TrackSyncDetail{
				MDNo:        mdNo,
				TrackStatus: strPtrOrNil(ev.Stage),
				EventDesc:   strPtrOrNil(ev.Desc),
				EventTime:   &ev.Time,
				Stage:       strPtrOrNil(ev.Stage),
				Location:    strPtrOrNil(ev.Location),
				CreateTime:  now,
			}
			if err := s.detailRepo.Insert(ctx, detail); err != nil {
				log.Printf("插入轨迹详情失败 mdNo=%s: %v", mdNo, err)
			} else {
				inserted++
			}
		}
		if inserted > 0 {
			log.Printf("运单 %s 新增 %d 条轨迹事件", mdNo, inserted)
		}
		last := collected[len(collected)-1]
		newEvent = &last.Desc
		t := last.Time
		newEventTime = &t
	}

	// FCtrack 回写（最新事件描述）
	if newEvent != nil && *newEvent != "" {
		if record.DtlFID == nil {
			log.Printf("跳过 FCtrack 回写：记录缺少 dtl_fid mdNo=%s（请回填 track_sync_record.dtl_fid）", mdNo)
		} else if err := s.shipOrderRepo.UpdateFCtrackByFID(ctx, *record.DtlFID, *newEvent); err != nil {
			log.Printf("更新FCtrack失败 mdNo=%s: %v", mdNo, err)
		}
	}

	// 发货/签收日期（按目的地时区）
	dates := ResolveDates(info, s.destZone)
	if dates.ShipDate != nil {
		record.ShipDate = dates.ShipDate
	}
	if dates.DeliveredDate != nil {
		record.DeliveredDate = dates.DeliveredDate
	}
	if (dates.ShipDate != nil || dates.DeliveredDate != nil) && record.DtlFID != nil {
		if err := s.shipOrderRepo.UpdateDatesByFID(ctx, *record.DtlFID, dates.ShipDate, dates.DeliveredDate); err != nil {
			log.Printf("回写日期失败 mdNo=%s: %v", mdNo, err)
		} else if dates.DestZoneName != "" {
			log.Printf("运单 %s 写回时间: 发货=%v 签收=%v (目的地时区 %s)", mdNo, fmtDateTime(dates.ShipDate), fmtDateTime(dates.DeliveredDate), dates.DestZoneName)
		}
	}

	isDelivered := newStatus != nil && *newStatus == "Delivered"
	record.TrackStatus = newStatus
	record.LastEvent = newEvent
	record.LastEventTime = newEventTime
	now := time.Now()
	record.LastSyncTime = &now
	record.UpdateTime = now
	if isDelivered {
		record.IsDelivered = true
		if record.DtlFID == nil {
			log.Printf("跳过签收标记：记录缺少 dtl_fid mdNo=%s（请回填 track_sync_record.dtl_fid）", mdNo)
		} else {
			if err := s.shipOrderRepo.MarkDeliveredByFID(ctx, *record.DtlFID); err != nil {
				log.Printf("标记签收失败 mdNo=%s: %v", mdNo, err)
			}
			// 签收地点回写：location 取轨迹签收（最后）事件地点，locationmatch 按客户收货信息比对
			s.syncDeliveryLocation(ctx, mdNo, *record.DtlFID, order, collected, existing)
		}
		log.Printf("运单 %s 已签收，标记不再扫描", mdNo)
	}

	return s.recordRepo.Update(ctx, record)
}

// syncDeliveryLocation 签收后把承运商地点回写 scBNDtl.location，并按客户收货信息
// （CustShipInfo）是否包含该地点写 locationmatch（"匹配"/"不匹配"）。
// 仅签收后调用。本轮无事件时退回本地明细中最近一个非空地点（明细按事件时间倒序）；仍无则跳过。
func (s *TrackSyncService) syncDeliveryLocation(ctx context.Context, mdNo string, dtlFID int64, order *model.ShipOrder, events []NormalizedEvent, details []model.TrackSyncDetail) {
	loc := DeliveryLocation(events)
	if loc == "" {
		for _, d := range details {
			if v := strings.TrimSpace(derefStr(d.Location)); v != "" {
				loc = v
				break
			}
		}
	}
	if loc == "" {
		log.Printf("运单 %s 已签收但无可用地点，跳过 location 回写", mdNo)
		return
	}
	cust := ""
	if order != nil && order.CustShipInfo != nil {
		cust = *order.CustShipInfo
	}
	match := MatchLocation(cust, loc)
	if err := s.shipOrderRepo.UpdateLocationByFID(ctx, dtlFID, loc, match); err != nil {
		log.Printf("回写签收地点失败 mdNo=%s: %v", mdNo, err)
	}
}

// strPtrOrNil 空字符串返回 nil，否则返回其指针。
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// fmtDateTime 格式化为 "YYYY-MM-DD HH:MM:SS"（按时间自身时区），用于日志。
func fmtDateTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
