package exposition

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/Vilsol/go-pob/data/tree2"
)

var (
	treeLinkVersionRE      = regexp.MustCompile(`tree/([0-9]+)\.([0-9]+)\.([0-9]+)/`)
	treeLinkRuthlessRE     = regexp.MustCompile(`tree/ruthless`)
	treeLinkPoeplannerRE   = regexp.MustCompile(`poeplanner\.com/`)
	treeLinkPoeskilltreeRE = regexp.MustCompile(`poeskilltree\.com/`)
	treeLinkOldVersionRE   = regexp.MustCompile(`/\?v=([0-9]+)\.([0-9]+)\.([0-9]+)-?r?u?t?h?l?e?s?s?#`)
	treeLinkOldRuthlessRE  = regexp.MustCompile(`-ruthless#`)
	treeLinkOldPrefixRE    = regexp.MustCompile(`/\?v=.+#`)
)

// ExportTreeLink is the Export Tree popup's link for the active spec; treeVersions[v].url is nil in PoB-PoE2, so the
// link has no prefix.
//
//pob2:src/Classes/TreeTab.lua:822-823
func (c *Calculator) ExportTreeLink() (string, error) {
	spec, err := c.spec()
	if err != nil {
		return "", err
	}
	return spec.EncodeURL(""), nil
}

// treeLinkVersion is validateTreeVersion; isRuthless reports a "tree/ruthless" link.
//
//pob2:src/Classes/TreeTab.lua:739-753
func treeLinkVersion(isRuthless bool, m []string) string {
	suffix := ""
	if isRuthless {
		suffix = "_ruthless"
	}
	if m != nil {
		major, _ := strconv.Atoi(m[1])
		minor, _ := strconv.Atoi(m[2])
		num, _ := strconv.ParseFloat(fmt.Sprintf("%d.%02d", major, minor), 64)
		// treeVersions[defaultTreeVersion].num and treeVersions[latestTreeVersion].num
		if num >= 0.1 && num <= 0.5 {
			return fmt.Sprintf("%d_%d", major, minor) + suffix
		}
	}
	return TreeVersion + suffix
}

// ImportTreeLink is the Import Tree popup's Import: a link becomes a new spec at the end of the list, made active. The
// result is the popup's message, empty when the link imported. A poeurl.com link arrives already resolved.
// Go must differ: a link naming a tree version other than the loaded one is refused, where PoB-PoE2 raises loading it.
//
//pob2:src/Classes/TreeTab.lua:695-814
func (c *Calculator) ImportTreeLink(title, treeLink string) (string, error) {
	if treeLink == "" {
		return "", nil
	}
	version := ""
	switch {
	case treeLinkPoeplannerRE.MatchString(treeLink):
		v, err := tree2.DecodePoePlannerTreeVersion(treeLinkOldPrefixRE.ReplaceAllString(treeLink, "/"))
		if err != nil {
			return err.Error(), nil //nolint:nilerr // the popup's message
		}
		version = v
	case treeLinkPoeskilltreeRE.MatchString(treeLink):
		version = treeLinkVersion(treeLinkOldRuthlessRE.MatchString(treeLink), treeLinkOldVersionRE.FindStringSubmatch(treeLink))
		treeLink = treeLinkOldPrefixRE.ReplaceAllString(treeLink, "/")
	default:
		version = treeLinkVersion(treeLinkRuthlessRE.MatchString(treeLink), treeLinkVersionRE.FindStringSubmatch(treeLink))
	}
	if version != TreeVersion {
		return fmt.Sprintf("Invalid tree link (tree version %s is not available)", version), nil
	}

	t, err := loadTree()
	if err != nil {
		return "", err
	}
	newSpec := tree2.NewPassiveSpec(t)
	if err := newSpec.DecodeURL(treeLink); err != nil {
		return err.Error(), nil //nolint:nilerr // the popup's message
	}
	c.build.Tree.Specs = append(c.build.Tree.Specs, newSpecElement(newSpec, title, nil))
	c.SetActiveTreeSpec(len(c.build.Tree.Specs))
	return "", nil
}
