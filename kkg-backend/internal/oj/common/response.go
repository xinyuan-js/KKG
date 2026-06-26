package common

type BaseResponse struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func Success(data interface{}) BaseResponse {
	return BaseResponse{Code: 0, Data: data, Message: "ok"}
}

func Error(code int, message string) BaseResponse {
	return BaseResponse{Code: code, Data: nil, Message: message}
}
