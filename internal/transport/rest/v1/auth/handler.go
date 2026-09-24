package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/pkg/ctxuser"
	"github.com/bookly-kbtu/backend/internal/transport/rest/v1/response"
	authuc "github.com/bookly-kbtu/backend/internal/usecase/auth"
)

type Handler struct {
	uc *authuc.Service
}

func New(uc *authuc.Service) *Handler {
	return &Handler{uc: uc}
}

type RequestOTPRequest struct {
	Phone   string `json:"phone" example:"+77011234567"`
	Channel string `json:"channel" enums:"sms,whatsapp,telegram" example:"sms"` // default sms
}

type RequestOTPResponse struct {
	ExpiresAt         time.Time `json:"expires_at"`
	ResendAvailableAt time.Time `json:"resend_available_at"`
}

type RegisterRequest struct {
	Phone string `json:"phone" example:"+77011234567"`
	Name  string `json:"name" example:"Дани"`
	Code  string `json:"code" example:"1234"`
}

type LoginRequest struct {
	Phone string `json:"phone" example:"+77011234567"`
	Code  string `json:"code" example:"1234"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type TokenResponse struct {
	TokenType             string    `json:"token_type" example:"Bearer"`
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

type UserResponse struct {
	ID        uuid.UUID     `json:"id"`
	Phone     string        `json:"phone"`
	FirstName string        `json:"first_name"`
	Status    string        `json:"status" example:"active"`
	Roles     []domain.Role `json:"roles" swaggertype:"array,string" example:"client"`
}

type AuthResponse struct {
	TokenResponse
	User UserResponse `json:"user"`
}

// Register mounts auth routes. requireAuth is applied per route: a Fiber group
// with middleware would also cover the public routes under the same prefix.
func (h *Handler) Register(auth fiber.Router, requireAuth fiber.Handler) {
	auth.Post("/otp/request", h.requestOTP)
	auth.Post("/register", h.register)
	auth.Post("/login", h.login)
	auth.Post("/refresh", h.refresh)
	auth.Post("/logout", h.logout)

	auth.Post("/logout-all", requireAuth, h.logoutAll)
	auth.Get("/me", requireAuth, h.me)
}

// requestOTP godoc
//
//	@Summary		Request OTP code
//	@Description	Sends a 4-digit code to the phone. Same code works for login and register. MVP: code is OTP_STATIC_CODE, nothing is sent.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RequestOTPRequest	true	"Phone and channel"
//	@Success		200		{object}	RequestOTPResponse
//	@Failure		400		{object}	response.ErrorBody	"Invalid phone or channel"
//	@Failure		429		{object}	response.ErrorBody	"Resend cooldown or hourly limit"
//	@Router			/auth/otp/request [post]
func (h *Handler) requestOTP(c *fiber.Ctx) error {
	var req RequestOTPRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}

	out, err := h.uc.RequestOTP(c.UserContext(), authuc.RequestOTPInput{
		Phone:   req.Phone,
		Channel: req.Channel,
		IP:      c.IP(),
	})
	if err != nil {
		return err
	}

	return response.OK(c, RequestOTPResponse{
		ExpiresAt:         out.ExpiresAt,
		ResendAvailableAt: out.ResendAvailableAt,
	})
}

// register godoc
//
//	@Summary		Register by phone
//	@Description	Creates user with client role and profile, returns tokens.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RegisterRequest	true	"Phone, name, OTP code"
//	@Success		201		{object}	AuthResponse
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody	"Invalid or expired code"
//	@Failure		409		{object}	response.ErrorBody	"Phone already registered"
//	@Router			/auth/register [post]
func (h *Handler) register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}

	out, err := h.uc.Register(c.UserContext(), authuc.RegisterInput{
		Phone: req.Phone,
		Name:  req.Name,
		Code:  req.Code,
		Meta:  clientMeta(c),
	})
	if err != nil {
		return err
	}

	return response.Created(c, authResponse(out))
}

// login godoc
//
//	@Summary	Login by phone
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		LoginRequest	true	"Phone and OTP code"
//	@Success	200		{object}	AuthResponse
//	@Failure	401		{object}	response.ErrorBody	"Invalid or expired code"
//	@Failure	403		{object}	response.ErrorBody	"Account blocked"
//	@Failure	404		{object}	response.ErrorBody	"Phone not registered"
//	@Router		/auth/login [post]
func (h *Handler) login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}

	out, err := h.uc.Login(c.UserContext(), authuc.LoginInput{
		Phone: req.Phone,
		Code:  req.Code,
		Meta:  clientMeta(c),
	})
	if err != nil {
		return err
	}

	return response.OK(c, authResponse(out))
}

// refresh godoc
//
//	@Summary		Rotate tokens
//	@Description	Old refresh token is revoked. Reusing a revoked token revokes all sessions of the user.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RefreshRequest	true	"Refresh token"
//	@Success		200		{object}	TokenResponse
//	@Failure		401		{object}	response.ErrorBody
//	@Router			/auth/refresh [post]
func (h *Handler) refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}

	tokens, err := h.uc.Refresh(c.UserContext(), authuc.RefreshInput{
		RefreshToken: req.RefreshToken,
		Meta:         clientMeta(c),
	})
	if err != nil {
		return err
	}

	return response.OK(c, tokenResponse(*tokens))
}

// logout godoc
//
//	@Summary	Logout current session
//	@Tags		auth
//	@Accept		json
//	@Param		body	body	RefreshRequest	true	"Refresh token"
//	@Success	204
//	@Router		/auth/logout [post]
func (h *Handler) logout(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}

	if err := h.uc.Logout(c.UserContext(), req.RefreshToken); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// logoutAll godoc
//
//	@Summary	Logout all devices
//	@Tags		auth
//	@Security	BearerAuth
//	@Success	204
//	@Failure	401	{object}	response.ErrorBody
//	@Router		/auth/logout-all [post]
func (h *Handler) logoutAll(c *fiber.Ctx) error {
	user, _ := ctxuser.From(c.UserContext())

	if err := h.uc.LogoutAll(c.UserContext(), user.ID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// me godoc
//
//	@Summary	Current user
//	@Tags		auth
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	UserResponse
//	@Failure	401	{object}	response.ErrorBody
//	@Router		/auth/me [get]
func (h *Handler) me(c *fiber.Ctx) error {
	user, _ := ctxuser.From(c.UserContext())

	me, err := h.uc.Me(c.UserContext(), user.ID)
	if err != nil {
		return err
	}
	return response.OK(c, userResponse(me))
}

func parseBody(c *fiber.Ctx, dest any) error {
	if err := c.BodyParser(dest); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	return nil
}

func clientMeta(c *fiber.Ctx) authuc.ClientMeta {
	return authuc.ClientMeta{IP: c.IP(), UserAgent: c.Get(fiber.HeaderUserAgent)}
}

func tokenResponse(t authuc.TokenPair) TokenResponse {
	return TokenResponse{
		TokenType:             "Bearer",
		AccessToken:           t.AccessToken,
		AccessTokenExpiresAt:  t.AccessTokenExpiresAt,
		RefreshToken:          t.RefreshToken,
		RefreshTokenExpiresAt: t.RefreshTokenExpiresAt,
	}
}

func userResponse(me *authuc.MeOutput) UserResponse {
	roles := me.Roles
	if roles == nil {
		roles = []domain.Role{}
	}
	return UserResponse{
		ID:        me.ID,
		Phone:     me.Phone,
		FirstName: me.FirstName,
		Status:    string(me.Status),
		Roles:     roles,
	}
}

func authResponse(out *authuc.AuthOutput) AuthResponse {
	return AuthResponse{TokenResponse: tokenResponse(out.Tokens), User: userResponse(out.User)}
}
