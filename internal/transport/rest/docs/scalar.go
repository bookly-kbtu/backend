package docs

import (
	"errors"
	"io/fs"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
)

// Pinned Scalar version: bump deliberately, the CDN script runs in the browser.
const scalarScript = "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.71.0"

const scalarPage = `<!doctype html>
<html>
<head>
  <title>Bookly API</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
  <div id="app"></div>
  <script src="` + scalarScript + `"></script>
  <script>
    Scalar.createApiReference('#app', {
      url: '/docs/openapi.json',
      persistAuth: true,
    })
  </script>
</body>
</html>`

type Config struct {
	// SpecPath is the swagger.json produced by `make swagger`. It is read on every
	// request, so regenerating the spec needs no restart.
	SpecPath string
	// User and Password enable basic auth. Empty User means no auth (dev only).
	User     string
	Password string
}

// Register serves the Scalar UI at /docs and the spec at /docs/openapi.json.
func Register(app fiber.Router, cfg Config) {
	handlers := []fiber.Handler{}
	if cfg.User != "" {
		handlers = append(handlers, basicauth.New(basicauth.Config{
			Users: map[string]string{cfg.User: cfg.Password},
			Realm: "Bookly API docs",
		}))
	}

	app.Get("/docs", append(handlers, func(c *fiber.Ctx) error {
		c.Type("html", "utf-8")
		return c.SendString(scalarPage)
	})...)

	app.Get("/docs/openapi.json", append(handlers, func(c *fiber.Ctx) error {
		spec, err := os.ReadFile(cfg.SpecPath)
		if errors.Is(err, fs.ErrNotExist) {
			return fiber.NewError(fiber.StatusNotFound, "OpenAPI spec not generated: run `make swagger`")
		}
		if err != nil {
			return err
		}

		c.Type("json", "utf-8")
		return c.Send(spec)
	})...)
}
