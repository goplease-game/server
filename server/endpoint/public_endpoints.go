package endpoint

// PublicAPIEndpoints registers all publicly accessible API routes.
func (r *Router) PublicAPIEndpoints() {
	r.GET("status/", r.handler.ServerStatus)

	// users
	r.Group("users").
		POST("sign-up/", r.handler.UserSignUp).
		POST("sign-in/", r.handler.UserSignIn).
		POST("confirm-email/", r.handler.ConfirmEmail).
		POST("password-reset-request/", r.handler.PasswordResetRequest).
		POST("password-reset/", r.handler.PasswordResetConfirm)
}
