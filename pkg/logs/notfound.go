package logs

// NotFound is logged when the requested resource is not found
type NotFound struct {
	Reason  string `logevent:"reason"`
	Message string `logevent:"message,default=not-found"`
}
