package main

import (
	"fmt"
	"github.com/ahmetalpbalkan/go-cursor"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/term"
	"os"
)

var screenX int = 0
var screenY int = 0
var width int = 0
var height int = 0
var quit bool = false

func restore() {
	move(screenX, screenY)
}

func clear() {
	fmt.Print(cursor.ClearEntireScreen())
}

func getchar() byte {
	var b []byte = make([]byte, 1)
	os.Stdin.Read(b)
	return b[0]
}

func move(x int, y int) {
	fmt.Print(cursor.MoveTo(y, x))
}

func right() {
	screenX++
	restore()
}

func up() {
	screenY--
	restore()
}

func down() {
	if screenY < height {
		screenY++
	}
	restore()
}

func left() {
	if screenX > 0 {
		screenX--
	}
	restore()
}

func flash(msg string) {
	move(0, height)
	fmt.Print(msg)
	restore()
}

func clearBanner() {
	flash("                                                               ")
}

func insert() {
	flash("-- INSERT --")
	defer clearBanner()
	for {
		c := getchar()
		switch c {
		case 27:
			return
		default:
			fmt.Printf("%c", c)
		}
	}
}

func execute(cmd string) {
    for _, c := range cmd[1:] {
		switch c {
		case 'w':
			writeFile()
		case 'q':
			quit = true
        default:
            flash(fmt.Sprintf("unknown command: '%c'", c))
            return
		}
	}
}

func command() {
	cmd := ":"
	for {
		flash(cmd)
		c := getchar()
		switch c {
		case 13:
			execute(cmd)
			return
		case 27:
			clearBanner()
			return
		default:
			cmd += string(c)
		}
	}
}

func writeFile() {
	flash("wrote file")
}

func scan() {
	for {
		if quit {
			return
		}
		c := getchar()
		switch c {
		case 'l':
			right()
		case 'h':
			left()
		case 'j':
			down()
		case 'k':
			up()
		case 'i':
			insert()
		case ':':
			command()
		case 'q':
			return
		}
	}
}

func eventLoop() error {
	var err error

	width, height, err = terminal.GetSize(0)
	if err != nil {
		return err
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	clear()
	move(screenX, screenY)
	scan()
	return nil
}

func main() {
	err := eventLoop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}
