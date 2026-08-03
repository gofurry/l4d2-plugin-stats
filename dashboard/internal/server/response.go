package server

import "github.com/gofiber/fiber/v3"

type envelope struct {
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}

type errorBody struct {
	Error     apiError `json:"error"`
	RequestID string   `json:"request_id"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func sendData(c fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(envelope{Data: data, RequestID: c.RequestID()})
}

func sendError(c fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(errorBody{
		Error: apiError{Code: code, Message: message}, RequestID: c.RequestID(),
	})
}
