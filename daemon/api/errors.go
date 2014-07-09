package api

import (
	"encoding/json"
	"github.com/ngerakines/codederror"
	"github.com/ngerakines/preview/common"
	"log"
	"net/http"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newCodedErrorResponseJson(codeErr codederror.CodedError) (string, error) {
	resp := errorResponse{
		Code:    codeErr.Error(),
		Message: codeErr.Description(),
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func httpErrorResponse(res http.ResponseWriter, resErr error, code int) {
	var errData string
	var err error
	switch resErr := resErr.(type) {
	case codederror.CodedError:
		errData, err = newCodedErrorResponseJson(resErr)
	default:
		errData, err = newCodedErrorResponseJson(common.ErrorNotImplemented)
	}
	if err != nil {
		log.Println("Error producing error response", err)
		http.Error(res, "Error producing error response", 500)
	}
	http.Error(res, errData, code)
	return
}
