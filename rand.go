package gb

// Simple, fast rand generator
var seed = uint32(48202)

func Int() uint32 {
	seed = (seed * 134775813) + 1
	return seed
}

func Intn(n int) uint32 {
	seed = (seed * 134775813) + 1
	return seed % uint32(n)
}

var str_set = []byte("qwertyuiopasdfghjklzxcvbnm1234567890QWERTYUIOPASDFGHJKLZXCVBNM")
var str_set_len = len(str_set)

func genRandString(n int) string {
	return string(genRandChars(n))
}

func genRandChars(n int) []byte {
	buf := make([]byte, n)
	fillRandChars(buf, 0, n)
	return buf
}

func fillRandChars(buf []byte, start, length int) {
	for length > 0 {
		buf[start] = str_set[Intn(str_set_len)]
		start++
		length--
	}
}

