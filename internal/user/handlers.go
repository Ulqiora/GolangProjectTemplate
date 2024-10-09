package user

import "github.com/gofiber/fiber/v2"

type HttpHandler interface {
	Register() fiber.Handler
}
