package endpoint

// ProtectedAPIEndpoints registers API routes that require authentication.
func (r *Router) ProtectedAPIEndpoints() {
	r.POST("/users/email-confirmation-code/", r.handler.SendEmailConfirmationCode)

	r.Use(r.mw.EmailMustBeConfirmed)

	// game
	r.POST("/game/create-match", r.handler.RequestEmailChange)

	// users
	r.PUT("/users/password/", r.handler.ChangePassword)
	r.POST("/users/email/", r.handler.RequestEmailChange)
	r.PUT("/users/email/", r.handler.ConfirmEmailChange)
	r.PUT("/users/username/", r.handler.ChangeUsername)
	r.DELETE("/users/", r.handler.DeleteUser)
}
