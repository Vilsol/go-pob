package pob

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestMoveSocketGroupMatchesPoBsDrop: move_socket_group.tsv is the pinned PoB's own ListControl drop and SkillListControl:OnOrderChange run over every drag of lists up to 5 long.
func TestMoveSocketGroupMatchesPoBsDrop(t *testing.T) {
	f, err := os.Open("testdata/move_socket_group.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	atoi := func(s string) int {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("bad number %q: %v", s, err)
		}
		return n
	}

	rows, moved := 0, 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		cols := strings.Split(scanner.Text(), "\t")
		if len(cols) != 8 {
			t.Fatalf("bad row %q", scanner.Text())
		}
		n, from, slot, main, skillNumber := atoi(cols[0]), atoi(cols[1]), atoi(cols[2]), atoi(cols[3]), atoi(cols[4])
		wantOrder, wantMain, wantSkillNumber := cols[5], atoi(cols[6]), atoi(cols[7])

		to := slot
		if slot > from {
			to = slot - 1
		}

		for _, withInput := range []bool{true, false} {
			if !withInput && skillNumber != 1 {
				continue
			}
			b := &PathOfBuilding{Build: Build{MainSocketGroup: main}, Skills: Skills{ActiveSkillSet: 1, SkillSets: []SkillSet{{ID: 1}}}}
			for i := 1; i <= n; i++ {
				b.Skills.SkillSets[0].Skills = append(b.Skills.SkillSets[0].Skills, Skill{Label: strconv.Itoa(i)})
			}
			if withInput {
				number := float64(skillNumber)
				b.Calcs.Inputs = []Input{{Name: "skill_number", Number: &number}}
			}

			b.MoveSocketGroup(from-1, to-1)

			order := make([]string, 0, n)
			for _, g := range b.Skills.SkillSets[0].Skills {
				order = append(order, g.Label)
			}
			gotSkillNumber := 1
			for _, input := range b.Calcs.Inputs {
				if input.Name == "skill_number" {
					gotSkillNumber = int(*input.Number)
				}
			}
			if got := strings.Join(order, ","); got != wantOrder || b.Build.MainSocketGroup != wantMain || gotSkillNumber != wantSkillNumber {
				t.Errorf("n=%d from=%d slot=%d main=%d skill_number=%d (input %v): got %s main %d skill_number %d, PoB %s main %d skill_number %d",
					n, from, slot, main, skillNumber, withInput, got, b.Build.MainSocketGroup, gotSkillNumber, wantOrder, wantMain, wantSkillNumber)
			}
		}
		rows++
		if from != to {
			moved++
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if rows == 0 || moved == 0 {
		t.Fatalf("REFUSING: %d rows, %d of them moves; the fixture is empty or moves nothing", rows, moved)
	}
}

// TestMoveSocketGroupActsOnTheActiveSet: The drag acts on the active set, found by id.
func TestMoveSocketGroupActsOnTheActiveSet(t *testing.T) {
	b := twoSkillSets()
	b.Skills.ActiveSkillSet = 2
	b.MoveSocketGroup(1, 0)
	if b.Skills.SkillSets[1].Skills[0].Label != "swap b" || b.Skills.SkillSets[0].Skills[0].Label != "final" {
		t.Fatalf("got %+v", b.Skills.SkillSets)
	}
}
