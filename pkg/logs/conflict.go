package logs

// Conflict is logged when the request conflicts with an existing resource
type Conflict struct {
	Reason  string `logevent:"reason"`
	Message string `logevent:"message,default=conflict"`
}
