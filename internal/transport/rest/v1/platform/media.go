package platform

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/bookly-kbtu/backend/internal/transport/rest/v1/response"
	platformuc "github.com/bookly-kbtu/backend/internal/usecase/platform"
)

func (h *Handler) registerMedia(api fiber.Router, auth fiber.Handler) {
	api.Get("/media/*", h.media)
	api.Post("/users/me/avatar", auth, h.uploadClientAvatar)
	api.Delete("/users/me/avatar", auth, h.deleteClientAvatar)
	api.Post("/masters/me/avatar", auth, h.uploadMasterAvatar)
	api.Delete("/masters/me/avatar", auth, h.deleteMasterAvatar)
}

// media godoc
// @Summary Uploaded file
// @Description Public, immutable: every upload gets a new key.
// @Tags media
// @Produce image/jpeg,image/png,image/webp
// @Param key path string true "Object key, e.g. avatars/{user}/{id}.jpg"
// @Success 200 {file} binary
// @Failure 404 {object} response.ErrorBody
// @Failure 503 {object} response.ErrorBody
// @Router /media/{key} [get]
func (h *Handler) media(c *fiber.Ctx) error {
	obj, err := h.uc.OpenMedia(c.UserContext(), c.Params("*"))
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, obj.ContentType)
	c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	c.Set("X-Content-Type-Options", "nosniff")
	if obj.ETag != "" {
		c.Set(fiber.HeaderETag, strconv.Quote(obj.ETag))
	}
	// fasthttp closes the body after streaming it.
	return c.SendStream(obj.Body, int(obj.ContentLength))
}

func formImage(c *fiber.Ctx) (platformuc.Upload, func(), error) {
	header, err := c.FormFile("file")
	if err != nil {
		return platformuc.Upload{}, nil, fiber.NewError(fiber.StatusBadRequest, "multipart field \"file\" is required")
	}
	file, err := header.Open()
	if err != nil {
		return platformuc.Upload{}, nil, fiber.NewError(fiber.StatusBadRequest, "cannot read uploaded file")
	}
	return platformuc.Upload{Body: file, Size: header.Size}, func() { _ = file.Close() }, nil
}

// uploadClientAvatar godoc
// @Summary Upload current client avatar
// @Description JPEG, PNG or WebP up to 5 MB. Replaces the previous avatar.
// @Tags users
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image"
// @Success 200 {object} ClientProfileResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 413 {object} response.ErrorBody
// @Failure 503 {object} response.ErrorBody
// @Router /users/me/avatar [post]
func (h *Handler) uploadClientAvatar(c *fiber.Ctx) error {
	file, done, err := formImage(c)
	if err != nil {
		return err
	}
	defer done()
	profile, err := h.uc.SetClientAvatar(c.UserContext(), mustUser(c).ID, file)
	if err != nil {
		return err
	}
	return response.OK(c, clientProfileResponse(*profile))
}

// deleteClientAvatar godoc
// @Summary Remove current client avatar
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} ClientProfileResponse
// @Failure 401 {object} response.ErrorBody
// @Router /users/me/avatar [delete]
func (h *Handler) deleteClientAvatar(c *fiber.Ctx) error {
	profile, err := h.uc.DeleteClientAvatar(c.UserContext(), mustUser(c).ID)
	if err != nil {
		return err
	}
	return response.OK(c, clientProfileResponse(*profile))
}

// uploadMasterAvatar godoc
// @Summary Upload current master avatar
// @Description JPEG, PNG or WebP up to 5 MB. The master profile must exist.
// @Tags master-cabinet
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image"
// @Success 200 {object} MasterProfileResponse
// @Failure 400 {object} response.ErrorBody
// @Failure 401 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Failure 413 {object} response.ErrorBody
// @Failure 503 {object} response.ErrorBody
// @Router /masters/me/avatar [post]
func (h *Handler) uploadMasterAvatar(c *fiber.Ctx) error {
	file, done, err := formImage(c)
	if err != nil {
		return err
	}
	defer done()
	profile, err := h.uc.SetMasterAvatar(c.UserContext(), mustUser(c).ID, file)
	if err != nil {
		return err
	}
	return response.OK(c, masterProfileResponse(*profile))
}

// deleteMasterAvatar godoc
// @Summary Remove current master avatar
// @Tags master-cabinet
// @Security BearerAuth
// @Produce json
// @Success 200 {object} MasterProfileResponse
// @Failure 401 {object} response.ErrorBody
// @Failure 404 {object} response.ErrorBody
// @Router /masters/me/avatar [delete]
func (h *Handler) deleteMasterAvatar(c *fiber.Ctx) error {
	profile, err := h.uc.DeleteMasterAvatar(c.UserContext(), mustUser(c).ID)
	if err != nil {
		return err
	}
	return response.OK(c, masterProfileResponse(*profile))
}
