package main

import (
	"math/rand"
	"seehuhn.de/go/ncurses"
	"time"
)

const DELAY = 10 * time.Millisecond // 100 fps
const DELAY_VARIANCE_MS = 60
const CELL_DELAY_MS = 100
const MAX_AGE = 35
const SHADES = 20

type Cell struct {
	character   string
	age         int
	next_change time.Time
	delay       time.Duration
	live        bool
}

func (c Cell) Draw(y int, x int, screen ncurses.Window) {
	screen.Move(y, x)
	if c.age == 0 {
		screen.AttrSet(ncurses.ColorPair(1).AsAttr())
	} else {
		color_pair_id := max(c.age-MAX_AGE+SHADES+2, 2)
		screen.AttrSet(ncurses.ColorPair(color_pair_id).AsAttr())
	}
	if c.live {
		screen.Print(c.character)
	} else {
		screen.Print("  ")
	}
}

type Canvas struct {
	height          int
	width           int
	grid            [][]Cell
	character_pool  Characters
	character_width int
	screen          ncurses.Window
}

func NewCanvas(screen ncurses.Window) Canvas {
	ncurses.CursSet(ncurses.CursorOff)
	tip_color := ncurses.Color(8)
	tip_color.Init(886, 1000, 851)
	ncurses.ColorPair(1).Init(tip_color, ncurses.ColorBlack)
	for i := 0; i <= SHADES; i++ {
		trail_color := ncurses.Color(9 + i)
		r, g, b := 90, 690, 175
		r -= i * r / SHADES
		g -= i * g / SHADES
		b -= i * b / SHADES
		trail_color.Init(r, g, b)
		ncurses.ColorPair(2+i).Init(trail_color, ncurses.ColorBlack)
	}
	character_pool, character_width := GetSample()
	height, width := screen.GetMaxYX()
	width /= character_width
	grid := make([][]Cell, height)
	for i := range grid {
		grid[i] = make([]Cell, width)
	}
	return Canvas{
		height,
		width,
		grid,
		character_pool,
		character_width,
		screen,
	}
}

func (c Canvas) NewCell(t time.Time) Cell {
	delay := time.Duration((CELL_DELAY_MS + rand.Intn(DELAY_VARIANCE_MS*2) - DELAY_VARIANCE_MS) * int(time.Millisecond))
	new_cell := Cell{
		c.character_pool.Random(),
		0,
		t.Add(delay),
		delay,
		true,
	}
	col := rand.Intn(c.width)
	c.grid[0][col] = new_cell
	return new_cell
}

func (c Canvas) Tick(t time.Time) {
	if rand.Intn(10) == 0 {
		c.NewCell(t)
	}
	for y, col := range c.grid {
		for x, cell := range col {
			if cell.live && t.After(cell.next_change) {
				c.grid[y][x].next_change = cell.next_change.Add(cell.delay)
				if cell.age == 0 && y < c.height-1 {
					c.grid[y+1][x] = c.grid[y][x]
					c.grid[y+1][x].character = c.character_pool.Random()
					c.grid[y+1][x].Draw(y+1, x*c.character_width, c.screen)
				}
				c.grid[y][x].age += 1
				if c.grid[y][x].age > MAX_AGE {
					c.grid[y][x] = Cell{}
				}
				c.grid[y][x].Draw(y, x*c.character_width, c.screen)
				c.screen.Refresh()
			}
		}
	}
}

func main() {
	scr := ncurses.Init()
	defer ncurses.EndWin()
	canvas := NewCanvas(*scr)
	ticker := time.NewTicker(DELAY)
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		for {
			ch := scr.GetCh()
			if ch == 'q' {
				done <- true
			} else {
				canvas.NewCell(time.Now())
			}
		}
	}()
	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			canvas.Tick(t)
		}
	}
}
