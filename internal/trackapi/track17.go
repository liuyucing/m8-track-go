package trackapi

import (
	"context"
	"encoding/json"
	"log"

	"m8-track-go/internal/model"
)

// RegisterResult 注册结果
type RegisterResult struct {
	Accepted          []string // 新注册成功的运单号
	AlreadyRegistered []string // 已注册过的运单号（错误码 -18019901）
	Failed            []string // 真正失败的运单号（格式错误等）
}

// Register 注册运单号到 17track（自动识别承运商）
func (c *Client) Register(ctx context.Context, trackingNumbers []string) (*RegisterResult, error) {
	body := make([]model.Track17RegisterReq, len(trackingNumbers))
	for i, num := range trackingNumbers {
		body[i] = model.Track17RegisterReq{Number: num}
	}
	return c.doRegister(ctx, body)
}

// RegisterWithCarrier 注册运单号到 17track（指定承运商代码）
func (c *Client) RegisterWithCarrier(ctx context.Context, numberCarrierMap map[string]int) (*RegisterResult, error) {
	body := make([]model.Track17RegisterReq, 0, len(numberCarrierMap))
	for num, carrier := range numberCarrierMap {
		body = append(body, model.Track17RegisterReq{Number: num, Carrier: carrier})
	}
	return c.doRegister(ctx, body)
}

func (c *Client) doRegister(ctx context.Context, body []model.Track17RegisterReq) (*RegisterResult, error) {
	data, err := c.post(ctx, "/register", body)
	if err != nil {
		return nil, err
	}

	var regData model.Track17RegisterData
	if err := json.Unmarshal(data, &regData); err != nil {
		return nil, err
	}

	result := &RegisterResult{}

	// 解析 accepted
	for _, item := range regData.Accepted {
		result.Accepted = append(result.Accepted, item.Number)
	}

	// 解析 rejected，区分已注册和真正失败
	for _, raw := range regData.Rejected {
		var rejected model.Track17RejectedItem
		if err := json.Unmarshal(raw, &rejected); err != nil {
			log.Printf("解析 rejected 项失败: %v", err)
			continue
		}
		if rejected.Error.Code == -18019901 {
			// 已注册，视为成功
			result.AlreadyRegistered = append(result.AlreadyRegistered, rejected.Number)
		} else {
			// 格式错误等，真正失败
			log.Printf("运单 %s 注册失败: code=%d, message=%s", rejected.Number, rejected.Error.Code, rejected.Error.Message)
			result.Failed = append(result.Failed, rejected.Number)
		}
	}

	return result, nil
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
