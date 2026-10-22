package main

import "testing"

// TestAnimationMapping pins every animation state to its row in the sprite
// sheet and guards the frame counts against the 10-column layout.
func TestAnimationMapping(t *testing.T) {
	want := map[AnimID]int{
		AnimIdle:       0,
		AnimWalk:       1,
		AnimRun:        3,
		AnimJump:       4,
		AnimSlide:      6,
		AnimIdleSword:  7,
		AnimWalkSword:  8,
		AnimRunSword:   9,
		AnimJumpSword:  11,
		AnimSlideSword: 13,
		AnimAttack:     14,
		AnimDeath:      15,
		AnimDeathSword: 16,
	}

	if len(animIndex) != len(want) {
		t.Errorf("animIndex has %d entries, want %d", len(animIndex), len(want))
	}
	for id, idx := range want {
		if got, ok := animIndex[id]; !ok {
			t.Errorf("animation %d missing from animIndex", id)
		} else if got != idx {
			t.Errorf("animation %d index = %d, want %d", id, got, idx)
		}
		if n := animFrames[id]; n < 1 || n > sheetCols {
			t.Errorf("animation %d has %d frames, want between 1 and %d", id, n, sheetCols)
		}
	}
}

// TestSpriteSheetGeometry checks the embedded sheet still matches the grid the
// animation table was written against.
func TestSpriteSheetGeometry(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	b := g.assets.Player.Bounds()
	const sheetRows = 17
	if w, h := b.Dx(), b.Dy(); w != sheetCols*frameW || h != sheetRows*frameH {
		t.Errorf("sheet is %dx%d, want %dx%d", w, h, sheetCols*frameW, sheetRows*frameH)
	}
	if len(animIndex) > sheetRows {
		t.Errorf("animIndex references %d rows, sheet only has %d", len(animIndex), sheetRows)
	}
}

// TestPlayerLandsOnGround exercises gravity and ground collision.
func TestPlayerLandsOnGround(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	g.player.Y = 260
	g.player.VY = 0
	g.player.Grounded = false
	for i := 0; i < 240 && !g.player.Grounded; i++ {
		g.Update()
	}
	if !g.player.Grounded {
		t.Fatal("player never landed")
	}
	if g.player.Y != g.level.GroundY {
		t.Errorf("player rest height = %v, want %v", g.player.Y, g.level.GroundY)
	}
}

// TestDeathAndRestart verifies the game-over path and the reset.
func TestDeathAndRestart(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	g.player.HP = 0
	for i := 0; i < deathAnimDelay+10 && g.state == statePlaying; i++ {
		g.Update()
	}
	if g.state != stateGameOver {
		t.Fatalf("state = %v, want game over", g.state)
	}
	if !g.player.Dead {
		t.Error("player should be in the dead state")
	}
	g.score = 4200
	g.reset()
	if g.state != statePlaying || g.score != 0 || g.player.HP != g.player.MaxHP {
		t.Errorf("restart left state=%v score=%d hp=%d", g.state, g.score, g.player.HP)
	}
	if len(g.enemies) != len(g.level.Spawns) {
		t.Errorf("restart restored %d enemies, want %d", len(g.enemies), len(g.level.Spawns))
	}
}

// TestVictoryNeedsEveryGoblinAndTheBoss verifies the level-clear transition
// now waits on the warden as well as the twenty goblins.
func TestVictoryNeedsEveryGoblinAndTheBoss(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range g.enemies {
		e.Damage(enemyHP, 1, g)
	}
	// Goblins down but the warden still standing: the level is not cleared.
	for i := 0; i < 40; i++ {
		g.Update()
	}
	if g.state != statePlaying {
		t.Fatalf("state = %v, want playing while the boss is alive", g.state)
	}

	g.boss.Damage(bossHP, 1, g)
	for i := 0; i < 80 && g.state == statePlaying; i++ {
		g.Update()
	}
	if g.state != stateVictory {
		t.Fatalf("state = %v, want victory", g.state)
	}
	if want := len(g.level.Spawns)*200 + bossScore; g.score < want {
		t.Errorf("score = %d, want at least %d", g.score, want)
	}
}

// TestAttackBoxFacesForward checks the sword hitbox mirrors with facing.
func TestAttackBoxFacesForward(t *testing.T) {
	p := &Player{X: 100, Y: 200, FacingRight: true}
	b := p.AttackBox()
	if b.X != 112 || b.W != attackRange {
		t.Errorf("right-facing box = %+v, want x=112 w=%v", b, attackRange)
	}
	p.FacingRight = false
	b = p.AttackBox()
	if b.X+b.W != 88 {
		t.Errorf("left-facing box right edge = %v, want 88", b.X+b.W)
	}
}

// TestBodySeparation ensures a goblin cannot stand inside the warrior.
func TestBodySeparation(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	e := g.enemies[0]
	e.X = g.player.X
	for i := 0; i < 30; i++ {
		e.Update(g.player, g)
		if e.Box().Intersects(g.player.Box()) {
			t.Fatalf("enemy still overlaps player after %d ticks (enemy x=%v player x=%v)",
				i, e.X, g.player.X)
		}
	}
}

// TestRectIntersects covers the collision helper.
func TestRectIntersects(t *testing.T) {
	a := Rect{0, 0, 10, 10}
	if !a.Intersects(Rect{5, 5, 10, 10}) {
		t.Error("overlapping rects reported as disjoint")
	}
	if a.Intersects(Rect{10, 10, 5, 5}) {
		t.Error("touching rects reported as overlapping")
	}
	if !a.Contains(0, 0) || a.Contains(11, 11) {
		t.Error("Contains is wrong")
	}
}

// TestAnimationDoesNotRestartWhileHeld documents that re-selecting the current
// animation must not reset playback (this is what keeps a held attack from
// looping its first frame forever).
func TestAnimationDoesNotRestartWhileHeld(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	a := g.player.Anim
	a.Force(AnimWalk)
	a.Update()
	a.Update()
	frame, timer := a.Frame(), a.timer
	a.Set(AnimWalk)
	if a.Frame() != frame || a.timer != timer {
		t.Errorf("Set(AnimWalk) restarted playback: frame %d->%d timer %d->%d",
			frame, a.Frame(), timer, a.timer)
	}
}

// TestAnimationStateMachine covers the priority order of the player state
// machine: death > attack > slide > airborne > locomotion > idle, and the
// armed/unarmed variants of every locomotion state.
func TestAnimationStateMachine(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	p := g.player

	cases := []struct {
		name  string
		setup func()
		want  AnimID
	}{
		{"idle", func() {
			p.Dead, p.Attacking, p.Sliding, p.Grounded, p.VX, p.Running, p.Sword = false, false, false, true, 0, false, false
		}, AnimIdle},
		{"walk", func() {
			p.Dead, p.Attacking, p.Sliding, p.Grounded, p.VX, p.Running, p.Sword = false, false, false, true, walkSpeed, false, false
		}, AnimWalk},
		{"run", func() {
			p.Dead, p.Attacking, p.Sliding, p.Grounded, p.VX, p.Running, p.Sword = false, false, false, true, runSpeed, true, false
		}, AnimRun},
		{"jump", func() { p.Dead, p.Attacking, p.Sliding, p.Grounded, p.Sword = false, false, false, false, false }, AnimJump},
		{"slide", func() { p.Dead, p.Attacking, p.Sliding, p.Grounded, p.Sword = false, false, true, true, false }, AnimSlide},
		{"idle sword", func() {
			p.Dead, p.Attacking, p.Sliding, p.Grounded, p.VX, p.Running, p.Sword = false, false, false, true, 0, false, true
		}, AnimIdleSword},
		{"walk sword", func() {
			p.Dead, p.Attacking, p.Sliding, p.Grounded, p.VX, p.Running, p.Sword = false, false, false, true, walkSpeed, false, true
		}, AnimWalkSword},
		{"run sword", func() {
			p.Dead, p.Attacking, p.Sliding, p.Grounded, p.VX, p.Running, p.Sword = false, false, false, true, runSpeed, true, true
		}, AnimRunSword},
		{"jump sword", func() { p.Dead, p.Attacking, p.Sliding, p.Grounded, p.Sword = false, false, false, false, true }, AnimJumpSword},
		{"slide sword", func() { p.Dead, p.Attacking, p.Sliding, p.Grounded, p.Sword = false, false, true, true, true }, AnimSlideSword},
		{"attack", func() { p.Dead, p.Attacking, p.Sliding, p.Grounded, p.Sword = false, true, false, true, true }, AnimAttack},
		{"death", func() { p.Dead, p.Sword = true, false }, AnimDeath},
		{"death sword", func() { p.Dead, p.Sword = true, true }, AnimDeathSword},
		{"attack beats slide", func() { p.Dead, p.Attacking, p.Sliding, p.Grounded, p.Sword = false, true, true, true, true }, AnimAttack},
		{"slide beats airborne", func() { p.Dead, p.Attacking, p.Sliding, p.Grounded, p.Sword = false, false, true, false, false }, AnimSlide},
	}

	for _, c := range cases {
		c.setup()
		if got := p.CurrentAnimation(); got != c.want {
			t.Errorf("%s: got animation %d, want %d", c.name, got, c.want)
		}
	}
}

// TestRestartButtonFitsScreen checks the end-screen button stays on screen and
// inside the panel drawn behind it.
func TestRestartButtonFitsScreen(t *testing.T) {
	b := restartButtonRect()
	if b.X < 0 || b.Y < 0 || b.X+b.W > screenWidth || b.Y+b.H > screenHeight {
		t.Fatalf("button %+v is off screen (%dx%d)", b, screenWidth, screenHeight)
	}
	panelTop := float64(screenHeight/2 - 140)
	if b.Y < panelTop || b.Y+b.H > panelTop+260 {
		t.Errorf("button %+v escapes the panel (top %v)", b, panelTop)
	}
	if !b.Contains(b.X+1, b.Y+1) || b.Contains(b.X-40, b.Y-40) {
		t.Error("button hit box is wrong")
	}
}

// TestTwentySpawnsStandOnSolidGround pins the goblin count and makes sure none
// of them starts its life inside a pit.
func TestTwentySpawnsStandOnSolidGround(t *testing.T) {
	lvl := NewLevel()
	if len(lvl.Spawns) != 20 {
		t.Fatalf("got %d spawns, want 20", len(lvl.Spawns))
	}
	for _, s := range lvl.Spawns {
		if !groundAhead(lvl, s.X, s.Y) {
			t.Errorf("spawn at x=%v is not above solid ground", s.X)
		}
		if s.Y != lvl.GroundY {
			t.Errorf("spawn at x=%v has y=%v, want ground level %v", s.X, s.Y, lvl.GroundY)
		}
	}
	if !groundAhead(lvl, lvl.BossSpawn.X, lvl.BossSpawn.Y) {
		t.Errorf("boss spawn at x=%v is not above solid ground", lvl.BossSpawn.X)
	}
}

// TestGoblinStopsAtPitEdge makes sure a chasing goblin has the sense not to
// walk off a ledge - the warrior can fall in, the goblin should not.
func TestGoblinStopsAtPitEdge(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	// The first pit spans x=900..980. Park a goblin just short of it with the
	// warrior waiting on the far side so it has every reason to keep coming.
	e := newEnemy(860, g.level.GroundY)
	g.player.X = 1200
	for i := 0; i < 120; i++ {
		e.Update(g.player, g)
		if e.Y > g.level.GroundY+1 {
			t.Fatalf("goblin fell into the pit at tick %d (x=%v y=%v)", i, e.X, e.Y)
		}
	}
	if e.X > 900 {
		t.Errorf("goblin walked past the ledge to x=%v", e.X)
	}
}

// TestGoblinFallsIntoPit proves the pits affect goblins the same way they
// affect the warrior: one standing over open air drops and is finished off.
func TestGoblinFallsIntoPit(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	// x=940 sits inside the first pit.
	e := newEnemy(940, g.level.GroundY)
	g.player.X = 1400
	fell := false
	for i := 0; i < 240; i++ {
		e.Update(g.player, g)
		if e.Y > g.level.GroundY+40 {
			fell = true
		}
	}
	if !fell {
		t.Fatalf("goblin at x=940 never fell (y=%v)", e.Y)
	}
	if !e.Dead {
		t.Error("goblin that dropped out of the level should be dead")
	}
}

// TestBossFrameRectsInsideSheet guards the hand-measured frame boxes in
// boss.go against the real dimensions of assets/villain.png.
func TestBossFrameRectsInsideSheet(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	b := g.assets.Villain.Bounds()

	strips := map[string][]villainRect{
		"idle": bossIdle, "walk": bossWalk, "run": bossRun, "jump": bossJump,
		"fall": bossFall, "sword": bossSword, "hurt": bossHurt, "dead": bossDead,
	}
	// The sword row is the longest one in the sheet; the mapping below was
	// measured off the alpha channel and is easy to break by hand.
	if n := len(bossSword); n != 9 {
		t.Errorf("sword strip has %d frames, want 9", n)
	}
	if n := len(bossWalk); n != 9 {
		t.Errorf("walk strip has %d frames, want 9", n)
	}
	for name, frames := range strips {
		if len(frames) == 0 {
			t.Errorf("%s strip is empty", name)
			continue
		}
		for i, r := range frames {
			if r.X < 0 || r.Y < 0 || r.X+r.W > b.Dx() || r.Y+r.H > b.Dy() {
				t.Errorf("%s frame %d %+v escapes the %dx%d sheet", name, i, r, b.Dx(), b.Dy())
			}
		}
	}

	// Every upright strip must end on the same foot line, otherwise the boss
	// hops up and down as it cycles through the animation.
	for name, frames := range map[string][]villainRect{
		"idle": bossIdle, "walk": bossWalk, "run": bossRun,
	} {
		foot := frames[0].Y + frames[0].H
		for i, r := range frames {
			if r.Y+r.H != foot {
				t.Errorf("%s frame %d foot line = %d, want %d", name, i, r.Y+r.H, foot)
			}
		}
	}
}

// TestBossAttackBoxFacesPlayer mirrors the player's own hitbox test.
func TestBossAttackBoxFacesPlayer(t *testing.T) {
	b := newBoss(500, 470)
	b.FacingRight = true
	if box := b.AttackBox(); box.X < b.X {
		t.Errorf("right-facing box %+v starts left of the boss", box)
	}
	b.FacingRight = false
	if box := b.AttackBox(); box.X+box.W > b.X {
		t.Errorf("left-facing box %+v ends right of the boss", box)
	}
}

// TestBossChasesTheWarrior makes sure the warden actually closes distance the
// way a goblin does, instead of only swapping to a walk pose while standing
// its ground.
func TestBossChasesTheWarrior(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	b := g.boss
	b.Active = true
	g.player.X = b.X - 500
	g.player.Y = b.Y

	start := b.X
	for i := 0; i < 40; i++ {
		b.Update(g.player, g)
	}
	moved := start - b.X
	if moved <= 0 {
		t.Fatalf("boss moved away from the warrior: x %v -> %v", start, b.X)
	}
	if moved < 40 {
		t.Errorf("boss only covered %v px in 40 ticks, want at least 40", moved)
	}
	wantRight := g.player.X > b.X
	if b.FacingRight != wantRight {
		t.Errorf("boss facing right=%v while the warrior is at %v (boss at %v)",
			b.FacingRight, g.player.X, b.X)
	}
}

// TestBossFlipConvention pins which way villain.png natively faces. The warden
// sheet looks RIGHT (the run frames lean right with the cape trailing left), so
// it has to be mirrored when the boss faces left - the opposite of the warrior
// sheet. Flipping this the wrong way makes the boss moonwalk toward the player
// with its back turned.
func TestBossFlipConvention(t *testing.T) {
	if bossFlip(true) {
		t.Error("a right-facing boss must be drawn as-is, not mirrored")
	}
	if !bossFlip(false) {
		t.Error("a left-facing boss must be mirrored")
	}
}

// TestBossSleepsUntilApproached checks the warden does not wander its arena
// before the warrior arrives.
func TestBossSleepsUntilApproached(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	g.player.X = 120
	for i := 0; i < 60; i++ {
		g.boss.Update(g.player, g)
	}
	if g.boss.Active {
		t.Fatal("boss woke up while the warrior was across the level")
	}
	if g.boss.X != g.level.BossSpawn.X {
		t.Errorf("boss drifted to x=%v while idle, want %v", g.boss.X, g.level.BossSpawn.X)
	}

	g.player.X = g.level.BossSpawn.X - 300
	g.boss.Update(g.player, g)
	if !g.boss.Active {
		t.Error("boss did not wake up when the warrior entered the arena")
	}
}

// TestBossFacesTheWarrior checks the warden turns to face whatever it is
// chasing.
func TestBossFacesTheWarrior(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	b := g.boss
	b.Active = true
	g.player.X = b.X + 200
	for i := 0; i < 4; i++ {
		b.Update(g.player, g)
	}
	if !b.FacingRight {
		t.Error("boss should face right when the warrior is to its right")
	}
	g.player.X = b.X - 200
	for i := 0; i < 4; i++ {
		b.Update(g.player, g)
	}
	if b.FacingRight {
		t.Error("boss should face left when the warrior is to its left")
	}
}

// TestBossCyclesEveryState drives the warden long enough to chase, leap, swing
// and get hurt, so a bad frame index or a state it can never leave shows up
// here instead of mid-fight.
func TestBossCyclesEveryState(t *testing.T) {
	g, err := NewGame()
	if err != nil {
		t.Fatal(err)
	}
	b := g.boss
	b.Active = true
	seen := map[string]bool{}

	for i := 0; i < 1200; i++ {
		// Keep the warrior inside the arena so the warden has to chase, leap
		// and swing rather than stand still.
		g.player.X = bossArenaL + 150 + float64((i/40)%3)*160
		g.player.Y = b.Y
		if i%200 == 7 {
			b.Damage(40, 1, g)
		}
		b.Update(g.player, g)
		seen[b.State] = true
		if n := len(b.frames()); n == 0 {
			t.Fatalf("tick %d: state %q has no frames", i, b.State)
		}
		if b.State != "dead" {
			if idx := b.animFrame(len(b.frames())); idx < 0 || idx >= len(b.frames()) {
				t.Fatalf("tick %d: state %q picked frame %d of %d", i, b.State, idx, len(b.frames()))
			}
		}
	}

	for _, want := range []string{"idle", "walk", "attack", "hurt"} {
		if !seen[want] {
			t.Errorf("boss never entered the %q state (saw %v)", want, seen)
		}
	}
	if b.Dead || b.HP <= 0 {
		t.Errorf("six 40-damage hits should leave the boss standing, hp=%d", b.HP)
	}
}
