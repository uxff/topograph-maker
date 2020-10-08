package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const As = `aeiou`

var Ass = []string{
	"a", "an", "ai", "ao", "au", "ang", "ar", "al",
	"e", "ea", "ee", "ei", "eo", "eu", "en", "eng", "er", "el",
	"i", "ia", "ie", "ii", "io", "iu", "in", "ing", "ir", "il", "iai", "ian", "iang", "iao",
	"o", "oa", "oe", "oi", "oo", "ou", "on", "ong", "or", "ol",
	"u", "ua", "ue", "ui", "uo", "uu", "un", "ung", "ur", "ul", "uai", "uan", "uang", "uao",
	"iui", "iua", "iue", "iuo", "iun", "iuan", "iuang", "iuai", "iuao",
}

var Bss = []string{
	"b", "p", "m", "f", "v", "w",
	"d", "t", "n", "l",
	"ds", "ts", "s", "z",
	"j", "ch", "sh", "r",
	// "ji", "chi", "shi", // j q x
}

func MakeName() string {
	s := make([]string, 0)
	alllen := 2 + rand.Intn(6)
	for i := 0; i < alllen; i++ {
		abRoller := rand.Float32()
		if abRoller > 0.3 {
			// ass
			aRoller := rand.Intn(len(Ass))
			s = append(s, Ass[aRoller])
		} else {
			// bss
			bRoller := rand.Intn(len(Bss))
			s = append(s, Bss[bRoller])
		}
	}
	s1 := strings.Join(s, "")
	// s1 = strings.Trim(s1, " ")
	if len(s1) > 0 && s1[0] > 96 {
		//s1[0] = s1[0] - 32
		s1 = strings.ToUpper(s1[:1]) + s1[1:]
	}
	return s1
}

var Acc = "aeiou"
var Bcc = "qwrtypsdfghjklzxcvbnm"

// 此方法来自VB
func MakeName2() []byte {
	s1 := make([]byte, 0)
	rnum := rand.Intn(5) + 3 // 循环2到7次,产生4到14个随机字母
	for i := 1; i < rnum; i++ {
		rand.Seed(time.Now().UnixNano() + int64(i))
		if rand.Intn(5) > 2 { // 随机置换声母和韵母的位置
			//
			s1 = append(s1, byte(rand.Intn(26)+97), Acc[rand.Intn(len(Acc))])
		} else { // 随机取一个声母和一个韵母
			//
			s1 = append(s1, Acc[rand.Intn(len(Acc))], byte(rand.Intn(26)+97))
		}
	}
	// s1 = bytes.Trim(s1, " ")
	if len(s1) > 0 && s1[0] > 96 {
		s1[0] = s1[0] - 32
	}
	return s1
}

func main() {
	rand.Seed(time.Now().UnixNano())
	s1 := MakeName()
	fmt.Printf("after make, s=%s\n", s1)
	s2 := MakeName2()
	fmt.Printf("after make, s=%s\n", s2)
}
