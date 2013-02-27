package identityservice

import (
	"crypto/rand"
	"fmt"
	. "launchpad.net/gocheck"
	"reflect"
)

type UtilSuite struct{}

var _ = Suite(&UtilSuite{})

func (s *UtilSuite) TestRandomHexTokenHasLength(c *C) {
	val := randomHexToken()
	c.Assert(val, HasLen, 32)
}

func (s *UtilSuite) TestRandomHexTokenIsHex(c *C) {
	val := randomHexToken()
	for i, b := range val {
		switch {
		case (b >= 'a' && b <= 'f') || (b >= '0' && b <= '9'):
			continue
		default:
			c.Logf("char %d of %s was not in the right range",
				i, val)
			c.Fail()
		}
	}
}

func (s *UtilSuite) TestDefaultRandomizer(c *C) {
	raw := make([]byte, 6)
	c.Assert(string(raw), Equals, "\x00\x00\x00\x00\x00\x00")
	n, err := getRandom(raw)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 6)
	c.Assert(string(raw), Not(Equals), "\x00\x00\x00\x00\x00\x00")
}

type funcEqualsChecker struct {
	*CheckerInfo
}

var _ Checker = (*funcEqualsChecker)(nil)

func (checker *funcEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	ptr1 := reflect.ValueOf(params[0]).Pointer()
	ptr2 := reflect.ValueOf(params[1]).Pointer()
	return Equals.Check([]interface{}{ptr1, ptr2}, names)
}

// The Go language specification doesn't allow you to compare function pointers
// directly, so we use a little bit of a workaround. This may break in future
// versions of go, but it can be updated when necessary.
// http://stackoverflow.com/questions/9643205/how-do-i-compare-two-functions-for-pointer-equality-in-the-latest-go-weekly
var FuncEquals Checker = &funcEqualsChecker{
	&CheckerInfo{Name: "FuncEquals", Params: []string{"obtained", "expected"}},
}

func (s *UtilSuite) TestSetRandomizer(c *C) {
	orig := getRandom
	// This test will be mutating global state (getRandom), ensure that we
	// restore it sanely even if tests fail
	defer func() { getRandom = orig }()
	// "randomize" everything to the letter 'n'
	nRandomizer := func(b []byte) (n int, err error) {
		for i := range b {
			b[i] = 'n'
		}
		n = len(b)
		return
	}

	c.Assert(getRandom, FuncEquals, rand.Read)
	cleanup := setRandomizer(nRandomizer)
	c.Assert(getRandom, FuncEquals, nRandomizer)
	raw := make([]byte, 6)
	n, err := getRandom(raw)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 6)
	c.Assert(string(raw), Equals, "nnnnnn")
	cleanup()
	c.Assert(getRandom, FuncEquals, rand.Read)
}

func (s *UtilSuite) TestNotEnoughRandomBytes(c *C) {
	// No error, just not enough bytes
	shortRand := func(b []byte) (n int, err error) {
		var i = 0
		for ; i < 2; i++ {
			b[i] = 'x'
		}
		return i, nil
	}
	cleanup := setRandomizer(shortRand)
	defer cleanup()
	c.Assert(randomHexToken, PanicMatches, "Could not read 16 random bytes safely \\(read 2 bytes\\): <unknown>")
}

func (s *UtilSuite) TestRandomBytesError(c *C) {
	// No error, just not enough bytes
	errRand := func(b []byte) (n int, err error) {
		var i = 0
		for ; i < 3; i++ {
			b[i] = 'x'
		}
		return i, fmt.Errorf("Not enough bytes")
	}
	cleanup := setRandomizer(errRand)
	defer cleanup()
	c.Assert(randomHexToken, PanicMatches, "Could not read 16 random bytes safely \\(read 3 bytes\\): Not enough bytes")
}
