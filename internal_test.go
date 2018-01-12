package flock

import (
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"
	"testing"
)

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func Test(t *testing.T) { TestingT(t) }

func (t *TestSuite) TestNew(c *C) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "go-flock-")
	c.Assert(err, IsNil)
	c.Assert(tmpFile, Not(IsNil))
	path := tmpFile.Name()
	defer os.Remove(path)
	tmpFile.Close()

	fh, err := os.Create(path)
	c.Assert(err, IsNil)
	c.Assert(fh, Not(IsNil))

	f := New(fh)
	c.Assert(f, Not(IsNil))
	c.Check(f.fh, Equals, fh)
	c.Check(f.Path(), Equals, path)
	c.Check(f.Locked(), Equals, false)
	c.Check(f.RLocked(), Equals, false)
}
