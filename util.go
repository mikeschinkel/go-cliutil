package cliutil

//goland:noinspection GoUnusedParameter
func noop(...any) {

}
func must(err error) {
	if err != nil {
		logger.Error(err.Error())
	}
}
