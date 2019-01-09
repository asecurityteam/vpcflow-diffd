package logs

// InvalidInput is logged when the input provided is not valid
type InvalidInput struct {
	Reason  string `logevent:"reason"`
	Message string `logevent:"message,default=invalid-input"`
}
