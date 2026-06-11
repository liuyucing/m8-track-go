package service

import (
	"context"
	"log"
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
}

// NewTrackSync 创建同步服务
func NewTrackSync(
	shipOrderRepo *repository.ShipOrderRepo,
	recordRepo *repository.TrackRecordRepo,
	detailRepo *repository.TrackDetailRepo,
	trackClient *trackapi.Client,
	batchSize int,
) *TrackSyncService {
	return &TrackSyncService{
		shipOrderRepo: shipOrderRepo,
		recordRepo:    recordRepo,
		detailRepo:    detailRepo,
		trackClient:   trackClient,
		batchSize:     batchSize,
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

	// 构建 mdNo -> FID 和 mdNo -> carrier 映射
	mdNoToFID := make(map[string]string)
	mdNoToCarrier := make(map[string]int)
	for _, o := range orders {
		mdNoToFID[o.MDNo] = o.FID
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
			record := &model.TrackSyncRecord{
				FID:         mdNoToFID[mdNo],
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
			record := &model.TrackSyncRecord{
				FID:         mdNoToFID[mdNo],
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
			if err := s.shipOrderRepo.UpdateFCtrack(ctx, mdNo, "查询不到"); err != nil {
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
			if err := s.processTrackInfo(ctx, &info, rec); err != nil {
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

// processTrackInfo 处理单条轨迹信息
func (s *TrackSyncService) processTrackInfo(ctx context.Context, info *model.Track17TrackInfo, record *model.TrackSyncRecord) error {
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

	if newEvent != nil && *newEvent != "" {
		if err := s.shipOrderRepo.UpdateFCtrack(ctx, mdNo, *newEvent); err != nil {
			log.Printf("更新FCtrack失败 mdNo=%s: %v", mdNo, err)
		}
	}

	eventChanged := !strEqual(record.LastEvent, newEvent)
	if eventChanged {
		detail := &model.TrackSyncDetail{
			MDNo:        mdNo,
			TrackStatus: newStatus,
			EventDesc:   newEvent,
			EventTime:   newEventTime,
			CreateTime:  time.Now(),
		}
		if err := s.detailRepo.Insert(ctx, detail); err != nil {
			log.Printf("插入轨迹详情失败 mdNo=%s: %v", mdNo, err)
		}
		log.Printf("运单 %s 状态变更: [%v] -> [%v]", mdNo, record.LastEvent, newEvent)
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
		if err := s.shipOrderRepo.MarkDelivered(ctx, mdNo); err != nil {
			log.Printf("标记签收失败 mdNo=%s: %v", mdNo, err)
		}
		log.Printf("运单 %s 已签收，标记不再扫描", mdNo)
	}

	return s.recordRepo.Update(ctx, record)
}

func strEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
