package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/japanoise/termbox-util"
	"github.com/nsf/termbox-go"
	"golang.org/x/text/unicode/runenames"
)

type state struct {
	current    rune
	unicodeMax rune
	screenx    int
	screeny    int
	status     string
	searchPrev string
}

func findNextPrintable(s *state, ru rune) rune {
	ret := ru
	for ret < s.unicodeMax && !unicode.IsGraphic(ret) {
		ret++
	}
	return ret
}

func scroll(s *state) {
	if s.current > s.unicodeMax {
		s.current = s.unicodeMax
		return
	}
	if s.current < 0 {
		s.current = 0
	}
	s.current = findNextPrintable(s, s.current)
}

func setMax(s *state) {
	// Do this dynamically, as it is subject to change when new versions of Unicode come out
	s.unicodeMax = 0x10FFFF
	for !unicode.IsGraphic(s.unicodeMax) {
		s.unicodeMax--
	}
}

func refresh(s *state) {
	defer termbox.Sync()

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	sx, sy := termbox.Size()
	s.screenx = sx
	s.screeny = sy
	ru := s.current
	for i := 0; i < sy-2; i++ {
		ru = findNextPrintable(s, ru)
		termutil.Printstring(
			fmt.Sprintf("  %d 0x%02x 0%0o '%c' %s", ru, ru, ru, ru, runenames.Name(ru)),
			0, i)
		ru++
	}

	for i := 0; i < sx; i++ {
		termutil.PrintRune(i, sy-2, ' ', termbox.AttrReverse)
	}
	termutil.PrintstringColored(termbox.AttrReverse, "^C:Quit ^F:Find ^S:Search M-g:Jump M-<:Start M->:End", 0, sy-2)

	termutil.PrintRune(0, 0, '>', termbox.ColorDefault)
	termbox.SetCursor(0, 0)
	if s.status != "" {
		termutil.Printstring(s.status, 0, sy-1)
	}
}

func doKey(key string, s *state) bool {
	switch key {
	case "C-c":
		return false
	case "M-<":
		s.current = 0
	case "M->":
		s.current = s.unicodeMax
	case "M-g":
		dest := termutil.Prompt("Jump to #",
			func(_, _ int) { refresh(s) })
		destrune, err := strconv.ParseUint(dest, 0, 64)
		if err == nil {
			if rune(destrune) > s.unicodeMax {
				s.current = s.unicodeMax
			} else {
				s.current = rune(destrune)
			}
		} else {
			s.status = err.Error()
		}
	case "C-s":
		query := termutil.Prompt("Search for rune name",
			func(_, _ int) { refresh(s) })
		if query == "" {
			query = s.searchPrev
		}
		queryNorm := strings.ToUpper(query)
		orig := s.current
		s.current++
		for s.current < s.unicodeMax {
			if strings.Contains(runenames.Name(s.current), queryNorm) {
				s.searchPrev = query
				return true
			}
			s.current++
		}
		s.status = "No match found for " + query
		s.current = orig
	case "C-f":
		run := termutil.Prompt("Find rune",
			func(_, _ int) { refresh(s) })
		ru, _ := utf8.DecodeRuneInString(run)
		if ru == utf8.RuneError {
			s.status = "No or erroneous rune provided"
		} else {
			s.current = ru
		}
	case "UP", "C-p":
		s.current--
		for !unicode.IsGraphic(s.current) && s.current > 0 {
			s.current--
		}
	case "DOWN", "C-n":
		s.current++
	case "NEXT", "C-v":
		s.current += rune(s.screeny)
	case "PRIOR", "C-z", "M-v":
		s.current -= rune(s.screeny)
	}
	return true
}

func main() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	termbox.SetInputMode(termbox.InputAlt)
	defer termbox.Close()

	s := state{}
	setMax(&s)
	running := true

	for running {
		scroll(&s)
		refresh(&s)
		ev := termbox.PollEvent()
		if ev.Type == termbox.EventResize {
			termbox.Sync()
		} else if ev.Type == termbox.EventKey {
			running = doKey(termutil.ParseTermboxEvent(ev), &s)
		}
	}
}
