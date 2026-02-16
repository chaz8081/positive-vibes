package term

const (
	ansiReset  = "\x1b[0m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
	ansiCyan   = "\x1b[36m"
	ansiBlue   = "\x1b[34m"
	ansiDim    = "\x1b[2m"
)

func Reset() string {
	return ansiReset
}

func Green() string {
	return ansiGreen
}

func Yellow() string {
	return ansiYellow
}

func Red() string {
	return ansiRed
}

func Cyan() string {
	return ansiCyan
}

func Blue() string {
	return ansiBlue
}

func Dim() string {
	return ansiDim
}
