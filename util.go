package cliutil

//goland:noinspection GoUnusedParameter
func noop(...any) {

}
func must(err error) {
	if err != nil {
		logger.Error(err.Error())
	}
}

func valueOrDefault[T any](ptr *T, def T) T {
	if ptr != nil {
		return *ptr
	}
	return def
}

func ptr[T any](v T) *T {
	return &v
}
