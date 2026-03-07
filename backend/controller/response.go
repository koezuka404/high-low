package controller

import "github.com/labstack/echo/v4"

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

type ErrorResponse struct {
	Success bool      `json:"success"`
	Error   ErrorBody `json:"error"`
}

func respondSuccess(c echo.Context, status int, data any) error {
	return c.JSON(status, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func respondError(c echo.Context, status int, code string, message string) error {
	return c.JSON(status, ErrorResponse{
		Success: false,
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}
