package auth

import "encoding/json"

type ResponseResult struct {
	ResultCode uint8 `json:"result"`
}

func (re ResponseResult) ToJSON() []byte {
	data, _ := json.Marshal(re)
	return data
}

func NewResponseResult(code uint8) ResponseResult {
	return ResponseResult{
		ResultCode: code,
	}
}
