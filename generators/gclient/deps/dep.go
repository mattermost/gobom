package deps

type DepType int

const (
	GitDepType DepType = iota
	CIPDDepType
)

type Dep interface {
	Type() DepType
}

type GitDep struct {
	URL    string
	Parent string
}

type CIPDDep struct {
	Parent string
}

func (g *GitDep) Type() DepType {
	return GitDepType
}

func (c *CIPDDep) Type() DepType {
	return CIPDDepType
}
