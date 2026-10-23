package player

// The XP curve is quadratic: reaching level n takes 25·n·(n-1) XP in total,
// so each level needs 50 XP more than the previous one (50, 100, 150, ...).
const xpStep = 25

// XPForLevel is the total XP needed to reach level.
func XPForLevel(level int) int64 {
	if level <= 1 {
		return 0
	}
	n := int64(level)
	return xpStep * n * (n - 1)
}

func LevelFor(xp int64) int {
	level := 1
	for XPForLevel(level+1) <= xp {
		level++
	}
	return level
}

func (p Player) Level() int {
	return LevelFor(p.XP)
}

// GainXP adds xp and reports whether the player reached a new level.
func (p *Player) GainXP(xp int64) (leveledUp bool) {
	if xp <= 0 {
		return false
	}
	before := p.Level()
	p.XP += xp
	return p.Level() > before
}
