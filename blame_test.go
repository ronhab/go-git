package git

import (
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	fixtures "github.com/go-git/go-git-fixtures/v4"
	. "gopkg.in/check.v1"
)

type BlameSuite struct {
	BaseSuite
}

var _ = Suite(&BlameSuite{})

func (s *BlameSuite) TestNewLines(c *C) {
	h := plumbing.NewHash("ce9f123d790717599aaeb76bc62510de437761be")
	lines, err := newLines([]string{"foo"}, []*object.Commit{{
		Hash:    h,
		Message: "foo",
	}})

	c.Assert(err, IsNil)
	c.Assert(lines, HasLen, 1)
	c.Assert(lines[0].Text, Equals, "foo")
	c.Assert(lines[0].Hash, Equals, h)
}

func (s *BlameSuite) TestNewLinesWithNewLine(c *C) {
	lines, err := newLines([]string{"foo"}, []*object.Commit{
		{Message: "foo"},
		{Message: "bar"},
	})

	c.Assert(err, IsNil)
	c.Assert(lines, HasLen, 2)
	c.Assert(lines[0].Text, Equals, "foo")
	c.Assert(lines[1].Text, Equals, "\n")
}

type blameTest struct {
	repo   string
	rev    string
	path   string
	blames []string // the commits blamed for each line
}

type extBlameTest struct {
	base blameTest
	names []string
	emails []string
	times []time.Time
	messages []string
}
// run a blame on all the suite's tests
func (s *BlameSuite) TestBlame(c *C) {
	for _, t := range blameTests {
		r := s.NewRepositoryFromPackfile(fixtures.ByURL(t.repo).One())

		exp := s.mockBlame(c, t, r)
		commit, err := r.CommitObject(plumbing.NewHash(t.rev))
		c.Assert(err, IsNil)

		obt, err := Blame(commit, t.path)
		c.Assert(err, IsNil)
		c.Assert(obt, DeepEquals, exp)

		for i, l := range obt.Lines {
			c.Assert(l.Hash.String(), Equals, t.blames[i])
		}
	}

	for _, t := range extBlameTests {
		r := s.NewRepositoryFromPackfile(fixtures.ByURL(t.base.repo).One())

		exp := s.mockBlame(c, t.base, r)
		commit, err := r.CommitObject(plumbing.NewHash(t.base.rev))
		c.Assert(err, IsNil)

		obt, err := Blame(commit, t.base.path)
		c.Assert(err, IsNil)
		c.Assert(obt, DeepEquals, exp)

		for i, l := range obt.Lines {
			c.Assert(l.Hash.String(), Equals, t.base.blames[i])
			c.Assert(l.Author.Name, Equals, t.names[i])
			c.Assert(l.Author.Email, Equals, t.emails[i])
			c.Assert(l.Author.When.String(), Equals, t.times[i].String())
			c.Assert(l.Message, Equals, t.messages[i])
		}
	}
}

func (s *BlameSuite) mockBlame(c *C, t blameTest, r *Repository) (blame *BlameResult) {
	commit, err := r.CommitObject(plumbing.NewHash(t.rev))
	c.Assert(err, IsNil, Commentf("%v: repo=%s, rev=%s", err, t.repo, t.rev))

	f, err := commit.File(t.path)
	c.Assert(err, IsNil)
	lines, err := f.Lines()
	c.Assert(err, IsNil)
	c.Assert(len(t.blames), Equals, len(lines), Commentf(
		"repo=%s, path=%s, rev=%s: the number of lines in the file and the number of expected blames differ (len(blames)=%d, len(lines)=%d)\nblames=%#q\nlines=%#q", t.repo, t.path, t.rev, len(t.blames), len(lines), t.blames, lines))

	blamedLines := make([]*Line, 0, len(t.blames))
	for i := range t.blames {
		commit, err := r.CommitObject(plumbing.NewHash(t.blames[i]))
		c.Assert(err, IsNil)
		l := &Line{
			Author: commit.Author,
			Text:   lines[i],
			Hash:   commit.Hash,
			Message: commit.Message,
		}
		blamedLines = append(blamedLines, l)
	}

	return &BlameResult{
		Path:  t.path,
		Rev:   plumbing.NewHash(t.rev),
		Lines: blamedLines,
	}
}

// utility function to avoid writing so many repeated commits
func repeat(s string, n int) []string {
	if n < 0 {
		panic("repeat: n < 0")
	}
	r := make([]string, 0, n)
	for i := 0; i < n; i++ {
		r = append(r, s)
	}

	return r
}

// utility function to concat slices
func concat(vargs ...[]string) []string {
	var r []string
	for _, ss := range vargs {
		r = append(r, ss...)
	}

	return r
}

// utility function to avoid writing so many repeated commit dates
func repeatTime(t time.Time, n int) []time.Time {
	if n < 0 {
		panic("repeat: n < 0")
	}
	r := make([]time.Time, 0, n)
	for i := 0; i < n; i++ {
		r = append(r, t)
	}

	return r
}

// utility function to concat Time slices
func concatTime(vargs ...[]time.Time) []time.Time {
	var r []time.Time
	for _, t := range vargs {
		r = append(r, t...)
	}

	return r
}

var blameTests = [...]blameTest{
	// use the blame2humantest.bash script to easily add more tests.
	{"https://github.com/git-fixtures/basic.git", "6ecf0ef2c2dffb796033e5a02219af86ec6584e5", "binary.jpg", concat(
		repeat("35e85108805c84807bc66a02d91535e1e24b38b9", 285),
	)},
	{"https://github.com/git-fixtures/basic.git", "6ecf0ef2c2dffb796033e5a02219af86ec6584e5", "CHANGELOG", concat(
		repeat("b8e471f58bcbca63b07bda20e428190409c2db47", 1),
	)},
	{"https://github.com/git-fixtures/basic.git", "6ecf0ef2c2dffb796033e5a02219af86ec6584e5", "go/example.go", concat(
		repeat("918c48b83bd081e863dbe1b80f8998f058cd8294", 142),
	)},
	{"https://github.com/git-fixtures/basic.git", "6ecf0ef2c2dffb796033e5a02219af86ec6584e5", "json/long.json", concat(
		repeat("af2d6a6954d532f8ffb47615169c8fdf9d383a1a", 6492),
	)},
	{"https://github.com/git-fixtures/basic.git", "6ecf0ef2c2dffb796033e5a02219af86ec6584e5", "json/short.json", concat(
		repeat("af2d6a6954d532f8ffb47615169c8fdf9d383a1a", 22),
	)},
	{"https://github.com/git-fixtures/basic.git", "6ecf0ef2c2dffb796033e5a02219af86ec6584e5", "LICENSE", concat(
		repeat("b029517f6300c2da0f4b651b8642506cd6aaf45d", 22),
	)},
	{"https://github.com/git-fixtures/basic.git", "6ecf0ef2c2dffb796033e5a02219af86ec6584e5", "php/crappy.php", concat(
		repeat("918c48b83bd081e863dbe1b80f8998f058cd8294", 259),
	)},
	{"https://github.com/git-fixtures/basic.git", "6ecf0ef2c2dffb796033e5a02219af86ec6584e5", "vendor/foo.go", concat(
		repeat("6ecf0ef2c2dffb796033e5a02219af86ec6584e5", 7),
	)},
	/*
		// Failed
		{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "InstallSpinnaker.sh", concat(
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 2),
			repeat("a47d0aaeda421f06df248ad65bd58230766bf118", 1),
			repeat("23673af3ad70b50bba7fdafadc2323302f5ba520", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 29),
			repeat("9a06d3f20eabb254d0a1e2ff7735ef007ccd595e", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 4),
			repeat("a47d0aaeda421f06df248ad65bd58230766bf118", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 5),
			repeat("0c5bb1e4392e751f884f3c57de5d4aee72c40031", 2),
			repeat("d4b48a39aba7d3bd3e8abef2274a95b112d1ae73", 3),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 7),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("b7015a5d36990d69a054482556127b9c7404a24a", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 2),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 5),
			repeat("d4b48a39aba7d3bd3e8abef2274a95b112d1ae73", 7),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 3),
			repeat("d4b48a39aba7d3bd3e8abef2274a95b112d1ae73", 6),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 10),
			repeat("b7015a5d36990d69a054482556127b9c7404a24a", 4),
			repeat("0c5bb1e4392e751f884f3c57de5d4aee72c40031", 2),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 2),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 4),
			repeat("23673af3ad70b50bba7fdafadc2323302f5ba520", 4),
			repeat("d4b48a39aba7d3bd3e8abef2274a95b112d1ae73", 4),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 1),
			repeat("d4b48a39aba7d3bd3e8abef2274a95b112d1ae73", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 1),
			repeat("b7015a5d36990d69a054482556127b9c7404a24a", 1),
			repeat("0c5bb1e4392e751f884f3c57de5d4aee72c40031", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 13),
			repeat("b7015a5d36990d69a054482556127b9c7404a24a", 2),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 6),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 1),
			repeat("b7015a5d36990d69a054482556127b9c7404a24a", 2),
			repeat("0c5bb1e4392e751f884f3c57de5d4aee72c40031", 1),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 4),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 3),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 2),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 4),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 3),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 15),
			repeat("b41d7c0e5b20bbe7c8eb6606731a3ff68f4e3941", 1),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 1),
			repeat("b41d7c0e5b20bbe7c8eb6606731a3ff68f4e3941", 8),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 2),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 12),
			repeat("505577dc87d300cf562dc4702a05a5615d90d855", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 5),
			repeat("370d61cdbc1f3c90db6759f1599ccbabd40ad6c1", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 4),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 1),
			repeat("b41d7c0e5b20bbe7c8eb6606731a3ff68f4e3941", 5),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 3),
			repeat("b41d7c0e5b20bbe7c8eb6606731a3ff68f4e3941", 2),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 2),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 9),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 1),
			repeat("b41d7c0e5b20bbe7c8eb6606731a3ff68f4e3941", 3),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 4),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("8eb116de9128c314ac8a6f5310ca500b8c74f5db", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 6),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 1),
			repeat("b7015a5d36990d69a054482556127b9c7404a24a", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 6),
			repeat("d2f6214b625db706384b378a29cc4c22237db97a", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 3),
			repeat("d2f6214b625db706384b378a29cc4c22237db97a", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 4),
			repeat("b7015a5d36990d69a054482556127b9c7404a24a", 1),
			repeat("c9c2a0ec03968ab17e8b16fdec9661eb1dbea173", 1),
			repeat("d2f6214b625db706384b378a29cc4c22237db97a", 2),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("b7015a5d36990d69a054482556127b9c7404a24a", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 12),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 5),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 3),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("01e65d67eed8afcb67a6bdf1c962541f62b299c9", 5),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 3),
			repeat("a47d0aaeda421f06df248ad65bd58230766bf118", 5),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 5),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 1),
			repeat("01e65d67eed8afcb67a6bdf1c962541f62b299c9", 2),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("01e65d67eed8afcb67a6bdf1c962541f62b299c9", 1),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("b2c7142082d52b09ca20228606c31c7479c0833e", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 1),
			repeat("495c7118e7cf757aa04eab410b64bfb5b5149ad2", 1),
			repeat("d4b48a39aba7d3bd3e8abef2274a95b112d1ae73", 1),
			repeat("495c7118e7cf757aa04eab410b64bfb5b5149ad2", 3),
			repeat("d4b48a39aba7d3bd3e8abef2274a95b112d1ae73", 1),
			repeat("495c7118e7cf757aa04eab410b64bfb5b5149ad2", 1),
			repeat("50d0556563599366f29cb286525780004fa5a317", 1),
			repeat("dd2d03c19658ff96d371aef00e75e2e54702da0e", 1),
			repeat("d4b48a39aba7d3bd3e8abef2274a95b112d1ae73", 1),
			repeat("dd2d03c19658ff96d371aef00e75e2e54702da0e", 2),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 2),
			repeat("01e65d67eed8afcb67a6bdf1c962541f62b299c9", 1),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("b5c6053a46993b20d1b91e7b7206bffa54669ad7", 1),
			repeat("9e74d009894d73dd07773ea6b3bdd8323db980f7", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 1),
			repeat("d4b48a39aba7d3bd3e8abef2274a95b112d1ae73", 4),
			repeat("01e65d67eed8afcb67a6bdf1c962541f62b299c9", 1),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 1),
			repeat("b7015a5d36990d69a054482556127b9c7404a24a", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 1),
			repeat("d2f6214b625db706384b378a29cc4c22237db97a", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 3),
			repeat("b41d7c0e5b20bbe7c8eb6606731a3ff68f4e3941", 2),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 2),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 4),
			repeat("d2f6214b625db706384b378a29cc4c22237db97a", 1),
			repeat("b7015a5d36990d69a054482556127b9c7404a24a", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 5),
			repeat("b41d7c0e5b20bbe7c8eb6606731a3ff68f4e3941", 2),
			repeat("d2f6214b625db706384b378a29cc4c22237db97a", 1),
			repeat("ce9f123d790717599aaeb76bc62510de437761be", 5),
			repeat("ba486de7c025457963701114c683dcd4708e1dee", 4),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("01e65d67eed8afcb67a6bdf1c962541f62b299c9", 1),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("01e65d67eed8afcb67a6bdf1c962541f62b299c9", 3),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 1),
			repeat("01e65d67eed8afcb67a6bdf1c962541f62b299c9", 3),
			repeat("6328ee836affafc1b52127147b5ca07300ac78e6", 2),
			repeat("01e65d67eed8afcb67a6bdf1c962541f62b299c9", 3),
			repeat("3de4f77c105f700f50d9549d32b9a05a01b46c4b", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 2),
			repeat("370d61cdbc1f3c90db6759f1599ccbabd40ad6c1", 6),
			repeat("dd7e66c862209e8b912694a582a09c0db3227f0d", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 2),
			repeat("dd7e66c862209e8b912694a582a09c0db3227f0d", 3),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("dd7e66c862209e8b912694a582a09c0db3227f0d", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 3),
		)},
	*/
	{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "pylib/spinnaker/reconfigure_spinnaker.py", concat(
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 22),
		repeat("c89dab0d42f1856d157357e9010f8cc6a12f5b1f", 7),
	)},
	{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "pylib/spinnaker/validate_configuration.py", concat(
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 29),
		repeat("1e3d328a2cabda5d0aaddc5dec65271343e0dc37", 19),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 15),
		repeat("b5d999e2986e190d81767cd3cfeda0260f9f6fb8", 1),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 12),
		repeat("1e14f94bcf82694fdc7e2dcbbfdbbed58db0f4d9", 1),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 4),
		repeat("b5d999e2986e190d81767cd3cfeda0260f9f6fb8", 8),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 1),
		repeat("b5d999e2986e190d81767cd3cfeda0260f9f6fb8", 4),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 46),
		repeat("1e14f94bcf82694fdc7e2dcbbfdbbed58db0f4d9", 1),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 4),
		repeat("1e3d328a2cabda5d0aaddc5dec65271343e0dc37", 42),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 1),
		repeat("1e3d328a2cabda5d0aaddc5dec65271343e0dc37", 1),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 3),
		repeat("1e3d328a2cabda5d0aaddc5dec65271343e0dc37", 1),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 1),
		repeat("1e14f94bcf82694fdc7e2dcbbfdbbed58db0f4d9", 8),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 1),
		repeat("1e14f94bcf82694fdc7e2dcbbfdbbed58db0f4d9", 2),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 3),
		repeat("1e3d328a2cabda5d0aaddc5dec65271343e0dc37", 3),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 12),
		repeat("1e14f94bcf82694fdc7e2dcbbfdbbed58db0f4d9", 10),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 69),
		repeat("b5d999e2986e190d81767cd3cfeda0260f9f6fb8", 7),
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 4),
	)},
	{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "pylib/spinnaker/run.py", concat(
		repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 185),
	)},
	/*
		// Fail by 3
		{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "pylib/spinnaker/configurator.py", concat(
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 53),
			repeat("c89dab0d42f1856d157357e9010f8cc6a12f5b1f", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 4),
			repeat("e805183c72f0426fb073728c01901c2fd2db1da6", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 6),
			repeat("023d4fb17b76e0fe0764971df8b8538b735a1d67", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 36),
			repeat("1e14f94bcf82694fdc7e2dcbbfdbbed58db0f4d9", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 3),
			repeat("1e14f94bcf82694fdc7e2dcbbfdbbed58db0f4d9", 3),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 4),
			repeat("c89dab0d42f1856d157357e9010f8cc6a12f5b1f", 13),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 2),
			repeat("c89dab0d42f1856d157357e9010f8cc6a12f5b1f", 18),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 2),
			repeat("1e14f94bcf82694fdc7e2dcbbfdbbed58db0f4d9", 1),
			repeat("023d4fb17b76e0fe0764971df8b8538b735a1d67", 17),
			repeat("c89dab0d42f1856d157357e9010f8cc6a12f5b1f", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 43),
		)},
	*/
	{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "pylib/spinnaker/__init__.py", []string{}},
	{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "gradle/wrapper/gradle-wrapper.jar", concat(
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 1),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 7),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 2),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 2),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 3),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 1),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 10),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 11),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 29),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 7),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 58),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 1),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 1),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 2),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 2),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 13),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 4),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 3),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 13),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 2),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 9),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 3),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 1),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 17),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 3),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 6),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 6),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 3),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 5),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 4),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 3),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 2),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 1),
		repeat("11d6c1020b1765e236ca65b2709d37b5bfdba0f4", 6),
		repeat("bc02440df2ff95a014a7b3cb11b98c3a2bded777", 55),
	)},
	{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "config/settings.js", concat(
		repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 17),
		repeat("99534ecc895fe17a1d562bb3049d4168a04d0865", 1),
		repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 43),
		repeat("d2838db9f6ef9628645e7d04cd9658a83e8708ea", 1),
		repeat("637ba49300f701cfbd859c1ccf13c4f39a9ba1c8", 1),
		repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 13),
	)},
	/*
		// fail a few lines
		{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "config/default-spinnaker-local.yml", concat(
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 9),
			repeat("5e09821cbd7d710405b61cab0a795c2982a71b9c", 2),
			repeat("99534ecc895fe17a1d562bb3049d4168a04d0865", 1),
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 2),
			repeat("a596972a661d9a7deca8abd18b52ce1a39516e89", 1),
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 5),
			repeat("5e09821cbd7d710405b61cab0a795c2982a71b9c", 2),
			repeat("a596972a661d9a7deca8abd18b52ce1a39516e89", 1),
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 5),
			repeat("5e09821cbd7d710405b61cab0a795c2982a71b9c", 1),
			repeat("8980daf661408a3faa1f22c225702a5c1d11d5c9", 1),
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 25),
			repeat("caf6d62e8285d4681514dd8027356fb019bc97ff", 1),
			repeat("eaf7614cad81e8ab5c813dd4821129d0c04ea449", 1),
			repeat("caf6d62e8285d4681514dd8027356fb019bc97ff", 1),
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 24),
			repeat("974b775a8978b120ff710cac93a21c7387b914c9", 2),
			repeat("3ce7b902a51bac2f10994f7d1f251b616c975e54", 1),
			repeat("5a2a845bc08974a36d599a4a4b7e25be833823b0", 6),
			repeat("41e96c54a478e5d09dd07ed7feb2d8d08d8c7e3c", 14),
			repeat("7c8d9a6081d9cb7a56c479bfe64d70540ea32795", 5),
			repeat("5a2a845bc08974a36d599a4a4b7e25be833823b0", 2),
		)},
	*/
	/*
		// fail one line
		{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "config/spinnaker.yml", concat(
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 32),
			repeat("41e96c54a478e5d09dd07ed7feb2d8d08d8c7e3c", 2),
			repeat("5a2a845bc08974a36d599a4a4b7e25be833823b0", 1),
			repeat("41e96c54a478e5d09dd07ed7feb2d8d08d8c7e3c", 6),
			repeat("5a2a845bc08974a36d599a4a4b7e25be833823b0", 2),
			repeat("41e96c54a478e5d09dd07ed7feb2d8d08d8c7e3c", 2),
			repeat("5a2a845bc08974a36d599a4a4b7e25be833823b0", 2),
			repeat("41e96c54a478e5d09dd07ed7feb2d8d08d8c7e3c", 3),
			repeat("7c8d9a6081d9cb7a56c479bfe64d70540ea32795", 3),
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 50),
			repeat("974b775a8978b120ff710cac93a21c7387b914c9", 2),
			repeat("d4553dac205023fa77652308af1a2d1cf52138fb", 1),
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 9),
			repeat("caf6d62e8285d4681514dd8027356fb019bc97ff", 1),
			repeat("eaf7614cad81e8ab5c813dd4821129d0c04ea449", 1),
			repeat("caf6d62e8285d4681514dd8027356fb019bc97ff", 1),
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 39),
			repeat("079e42e7c979541b6fab7343838f7b9fd4a360cd", 6),
			repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 15),
		)},
	*/
	/*
		{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "dev/install_development.sh", concat(
			repeat("99534ecc895fe17a1d562bb3049d4168a04d0865", 1),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 71),
		)},
	*/
	/*
		// FAIL two lines interchanged
		{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "dev/bootstrap_dev.sh", concat(
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 95),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 10),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 7),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 2),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 1),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 3),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 4),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 12),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 2),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 2),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 2),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 3),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 6),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 1),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 4),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 1),
			repeat("376599177551c3f04ccc94d71bbb4d037dec0c3f", 2),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 17),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 2),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 2),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 2),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 3),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 3),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 5),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 5),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 8),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 4),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 1),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 6),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 1),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 4),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 10),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 2),
			repeat("fc28a378558cdb5bbc08b6dcb96ee77c5b716760", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 1),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 8),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 1),
			repeat("fc28a378558cdb5bbc08b6dcb96ee77c5b716760", 1),
			repeat("d1ff4e13e9e0b500821aa558373878f93487e34b", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 4),
			repeat("24551a5d486969a2972ee05e87f16444890f9555", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 2),
			repeat("24551a5d486969a2972ee05e87f16444890f9555", 1),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 8),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 13),
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 5),
			repeat("24551a5d486969a2972ee05e87f16444890f9555", 1),
			repeat("838aed816872c52ed435e4876a7b64dba0bed500", 8),
		)},
	*/
	/*
		// FAIL move?
		{"https://github.com/spinnaker/spinnaker.git", "f39d86f59a0781f130e8de6b2115329c1fbe9545", "dev/create_google_dev_vm.sh", concat(
			repeat("a24001f6938d425d0e7504bdf5d27fc866a85c3d", 20),
		)},
	*/
}

func swallowErr(t time.Time, err error) time.Time {
	return t
}

var messages = []string {
`Standard configuration files for a Spinnaker deployment.

These files are intended to be installed at the base of a spring config path.
The default-spinnaker-local.yml is intended to be copied into a
spinnaker-local.yml and modified for a custom deployment. The other files can
be modified as <subsystem>-local.yml as well if needed.

This PR is more about the structure and policy than the details though I
would like to get the basic namespace right. I'm primarily using "services"
but maybe this should be "spinnaker".

The spinnaker.yml file contains the system wide policy and values shared among
multiple systems. The individual system files contain the particular
configuration for a given subsystem. The namespace in the spinnaker.yml is
intentionally disjoint from those in the individual system files requiring
the systems to explicitly document their configuration -- both how it is
standard (by referencing spinnaker.yml values) and how it is non-standard
(by not referencing spinnaker.yml values).

There are future CLs that add more scripting and support of this but
fundamentally using this assumes setting the spring.config.location system
property to something like

$INSTALL/config/spinnaker.yml,\
$INSTALL/config/,\
$HOME/.spinnaker/spinnaker-local.yml,\
$HOME/.spinnaker/

Recently the module spring loader was changed to look for 'spinnaker.yml'
so the config location would be $INSTALL/config,$HOME/.spinnaker/
If the subsystems config yaml files in this PR were moved into the subsystems
themselves, then the spring location could remain the default $HOME/.spinnaker/

With this approach, users typically ovewrite a single spinnaker-local.yml file
for most needs. The spring expression language cannot handle "subtrees", only
values. Therefore configuration of repeated nodes requires overriding the
<subsystem>-local.yml in order to add the lists in. Otherwise, the
spinnaker.yml defines "primary" values and the <subsystem_local.yml provide
a list containing the "primary" value so that the spinnaker-local.yml can
still serve as a central configuration.

Deck is a different story.

I'm including a "settings.js" here as a placeholder. This is what I actively
use, but it is out of date from chris' current work. The only "interesting"
thing here is the use of variable declarations that reference the
spinnaker.yml namespace. There is a script (in a future CL) that can
substitute that block with current config values. For purposes of this CL,
the details of settings.js can be changed later without worry. It's the
policy of calling out key configuration variables that may be needed and
resolving them with a script (I'll provide later) that is of interest for
this PR.

CAVEAT:

I've been having trouble getting AWS to work.

I can get it to work using root credentials when I run out of debian packages
(on GCE) and use environment variables (with a launch script that sets them
based on the YAML file) but the same strategy does not work for gradlew runs.
The gradle runs complain that it does not know about the "default profile".
I can run out of gradle if I have an .aws/config [sic] file (e.g. from an
awscli). I can run the debian packages with an .aws/credentials file.
It seems user credentials need more attributes in clouddriver. For example a
role. However roles are not valid with root credentials and a null role is
not valid either, so it seems clouddriver-local.yml may have to exist for
maintaining aws credentials.
`,
`minor config tweaks to get things running. Adds a README.md for the local developer getting started experience
`,
`Add rebakeControlEnabled=true flag to deck settings.
`,
`Sync feature block in settings.js.
`,
}

var extBlameTests = [...]extBlameTest{
	{
		base: blameTest{
			repo: "https://github.com/spinnaker/spinnaker.git", rev: "f39d86f59a0781f130e8de6b2115329c1fbe9545", path: "config/settings.js", blames: concat(
				repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 17),
				repeat("99534ecc895fe17a1d562bb3049d4168a04d0865", 1),
				repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 43),
				repeat("d2838db9f6ef9628645e7d04cd9658a83e8708ea", 1),
				repeat("637ba49300f701cfbd859c1ccf13c4f39a9ba1c8", 1),
				repeat("ae904e8d60228c21c47368f6a10f1cc9ca3aeebf", 13),
			),
		},
		names: concat(
			repeat("Eric Wiseblatt", 17),
			repeat("Cameron Fieber", 1),
			repeat("Eric Wiseblatt", 43),
			repeat("duftler", 1),
			repeat("duftler", 1),
			repeat("Eric Wiseblatt", 13),
		),
		emails: concat(
			repeat("ewiseblatt@google.com", 17),
			repeat("cfieber@netflix.com", 1),
			repeat("ewiseblatt@google.com", 43),
			repeat("duftler@google.com", 1),
			repeat("duftler@google.com", 1),
			repeat("ewiseblatt@google.com", 13),
		),
		times: concatTime(
			repeatTime(swallowErr(time.Parse("2006-01-02 15:04:05 -0700", "2015-10-27 13:31:49 +0000")), 17),
			repeatTime(swallowErr(time.Parse("2006-01-02 15:04:05 -0700", "2015-10-29 15:28:08 -0700")), 1),
			repeatTime(swallowErr(time.Parse("2006-01-02 15:04:05 -0700", "2015-10-27 13:31:49 +0000")), 43),
			repeatTime(swallowErr(time.Parse("2006-01-02 15:04:05 -0700", "2015-11-14 22:51:32 -0500")), 1),
			repeatTime(swallowErr(time.Parse("2006-01-02 15:04:05 -0700", "2015-11-19 08:04:05 -0500")), 1),
			repeatTime(swallowErr(time.Parse("2006-01-02 15:04:05 -0700", "2015-10-27 13:31:49 +0000")), 13),
		),
		messages: concat(
			repeat(messages[0], 17),
			repeat(messages[1], 1),
			repeat(messages[0], 43),
			repeat(messages[2], 1),
			repeat(messages[3], 1),
			repeat(messages[0], 13),
		),
	},
}
