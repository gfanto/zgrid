package api

type errorResponse struct {
	Error string `json:"error"`
}

func newErrResp(msg string) errorResponse {
	return errorResponse{Error: msg}
}
