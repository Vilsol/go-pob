package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func flaskSourcedMinionMods(env *Environment) (int, []string) {
	var sources []string
	for _, name := range []string{"Armour", "Evasion"} {
		for _, r := range env.Minion.ModDB.Tabulate(mod.TypeBase, nil, name) {
			if src := string(r.Mod.GetSource()); strings.Contains(src, "Flask") {
				sources = append(sources, name+"<-"+src)
			}
		}
	}
	return len(sources), sources
}

// TestNonUniqueFlasksReachTheLinkedMinion is the end-to-end for the ExtraLinkEffect -> minion flask chain.
func TestNonUniqueFlasksReachTheLinkedMinion(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const subject = "1007-luminary-animate-guardian-of-smiting-4.xml"
	env := corpusEnvForLoadedData(t, subject)
	if env.Minion == nil || env.Minion.ModDB == nil {
		t.Fatalf("REFUSING: %s builds no minion actor", subject)
	}
	if len(env.ModDB.List(nil, "ExtraLinkEffect")) == 0 {
		t.Fatalf("PREMISE BROKEN: %s carries no ExtraLinkEffect — the belt row did not parse", subject)
	}
	if !env.Minion.ModDB.Conditions["AffectedByLink"] {
		t.Fatalf("PREMISE BROKEN: %s minion is not AffectedByLink, so the Link arm never ran", subject)
	}
	n, srcs := flaskSourcedMinionMods(env)
	if n == 0 {
		t.Errorf("%s: no flask-sourced Armour/Evasion BASE reached the minion (PoB's dump has the Granite "+
			"Flask's 1500 on both)", subject)
	}
	if !env.Minion.ModDB.Conditions["UsingFlask"] {
		t.Errorf("%s: minion condition UsingFlask not set (:1793-1795)", subject)
	}
	if !env.Minion.ModDB.Flag(nil, "ParentNonUniqueFlasksAppliedToYou") {
		t.Errorf("%s: the ExtraLinkEffect flag did not reach the minion through the link buff", subject)
	}
	t.Logf("subject flask-sourced minion rows: %v", srcs)

	const control = "0662-luminary-animate-guardian-of-smiting.xml"
	cenv := corpusEnvForLoadedData(t, control)
	if cenv.Minion == nil || cenv.Minion.ModDB == nil {
		t.Fatalf("REFUSING: control %s builds no minion actor, so it controls nothing", control)
	}
	if len(cenv.ModDB.List(nil, "ExtraLinkEffect")) == 0 {
		t.Fatalf("REFUSING: control %s carries no ExtraLinkEffect, so it controls nothing", control)
	}
	if len(cenv.Flasks) == 0 {
		t.Fatalf("REFUSING: control %s has no active flask, so it controls nothing", control)
	}
	if cn, csrcs := flaskSourcedMinionMods(cenv); cn != 0 || cenv.Minion.ModDB.Conditions["UsingFlask"] {
		t.Errorf("control %s (belt, minion, NOT linked) got flasks on its minion: %v, UsingFlask=%v — "+
			"the minion arm is opening without the Link", control, csrcs, cenv.Minion.ModDB.Conditions["UsingFlask"])
	}
}
