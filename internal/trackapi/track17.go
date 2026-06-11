package trackapi

import (
	"context"
	"encoding/json"
	"log"

	"m8-track-go/internal/model"
)

// Register 注册运单号到 17track（自动识别承运商）
func (c *Client) Register(ctx context.Context, trackingNumbers []string) ([]string, error) {
	body := make([]model.Track17RegisterReq, len(trackingNumbers))
	for i, num := range trackingNumbers {
		body[i] = model.Track17RegisterReq{Number: num}
	}
	return c.doRegister(ctx, body)
}

// RegisterWithCarrier 注册运单号到 17track（指定承运商代码）
func (c *Client) RegisterWithCarrier(ctx context.Context, numberCarrierMap map[string]int) ([]string, error) {
	body := make([]model.Track17RegisterReq, 0, len(numberCarrierMap))
	for num, carrier := range numberCarrierMap {
		body = append(body, model.Track17RegisterReq{Number: num, Carrier: carrier})
	}
	return c.doRegister(ctx, body)
}

func (c *Client) doRegister(ctx context.Context, body []model.Track17RegisterReq) ([]string, error) {
	data, err := c.post(ctx, "/register", body)
	if err != nil {
		return nil, err
	}

	var regData model.Track17RegisterData
	if err := json.Unmarshal(data, &regData); err != nil {
		return nil, err
	}

	accepted := make([]string, 0, len(regData.Accepted))
	for _, item := range regData.Accepted {
		accepted = append(accepted, item.Number)
	}

	if len(regData.Rejected) > 0 {
	var rejectedStrs []string
		for _, r := range regData.Rejected {
			rejectedStrs = append(rejectedStrs, string(r))
		}
		log.Printf("17track 注册被拒绝的运单: %v", rejectedStrs)
	}

	return accepted, nil
}

// GetTrackInfo 查询运单轨迹信息
func (c *Client) GetTrackInfo(ctx context.Context, trackingNumbers []string) ([]model.Track17TrackInfo, error) {
	body := make([]model.Track17RegisterReq, len(trackingNumbers))
	for i, num := range trackingNumbers {
		body[i] = model.Track17RegisterReq{Number: num}
	}

	data, err := c.post(ctx, "/gettrackinfo", body)
	if err != nil {
		return nil, err
	}

	var dataStruct struct {
		Accepted json.RawMessage `json:"accepted"`
	}
	if err := json.Unmarshal(data, &dataStruct); err != nil {
		return nil, err
	}

	var result []model.Track17TrackInfo
	if err := json.Unmarshal(dataStruct.Accepted, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Partition 将切片分批，替代 Java ListUtils.partition()
func Partition[T any](slice []T, size int) [][]T {
	var batches [][]T
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		batches = append(batches, slice[i:end])
	}
	return batches
}
