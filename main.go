package main

import (
	"bufio"
	"fmt"
	"github.com/ahmetalpbalkan/go-cursor"
	"golang.org/x/term"
	"os"
	"strconv"
	"strings"
)

var screenX int = 1
var screenY int = 1
var textX int = 0
var lineno int = 0
var height int = 0
var width int = 0
var quit bool = false
var filename string
var top *line
var topOfScreen *line
var currentLine *line
var clipboard string
var searchTerm string

const (
	ENTER_CODE     = 13
	ESCAPE_CODE    = 27
	BACKSPACE_CODE = 127
)

type line struct {
	text string
	prev *line
	next *line
}

func lineNew() *line {
	return &line{
		text: "",
		prev: nil,
		next: nil,
	}
}

func walkBack() {
	if textX > 0 {
		textX--
	}
	if screenX > 1 {
		screenX--
	}
}

func restore() {
	move(screenX, screenY)
}

func clear() {
	fmt.Print(cursor.ClearEntireScreen())
}

func getchar() byte {
	var b []byte = make([]byte, 1)
	os.Stdin.Read(b) //nolint
	return b[0]
}

func move(x int, y int) {
	fmt.Print(cursor.MoveTo(y, x))
}

func left() {
	if currentLine != nil {
		if textX > 0 {
			textX--
			screenX--
			restore()
		}
	}
}

func right() {
	if currentLine != nil {
		if textX < len(currentLine.text)-1 {
			textX++
			screenX++
			restore()
		}
	}
}

func up() {
	if currentLine == nil {
		return
	}
	if screenY > 1 {
		screenY--
		currentLine = currentLine.prev
		lineno--
	} else if topOfScreen.prev != nil {
		clear()
		topOfScreen = topOfScreen.prev
		currentLine = currentLine.prev
		lineno--
		draw()
	}
	restore()
}

func down() {
	if currentLine == nil || currentLine.next == nil {
		return
	}
	if screenY < height-1 {
		screenY++
		currentLine = currentLine.next
		lineno++
	} else {
		clear()
		topOfScreen = topOfScreen.next
		currentLine = currentLine.next
		lineno++
		draw()
	}
	restore()
}

func startOfLine() {
	screenX = 1
	textX = 0
	restore()
}

func displayLineno() {
	move(60, height)
	fmt.Print("            ")
	move(60, height)
	fmt.Printf("%d - %d", screenX, 1+lineno)
	restore()
}

func flash(msg string) {
	move(1, height)
	fmt.Print(msg)
	restore()
}

func clearBanner() {
	flash("                                                              ")
	restore()
}

func deleteChar(pos int) {
	txt := currentLine.text
	if len(txt) <= 0 {
		return
	}
	if pos > 0 {
		currentLine.text = txt[:pos-1] + txt[pos:]
	} else {
		currentLine.text = txt[1:]
	}
	walkBack()
	clear()
	draw()
}

func deleteLine(line *line) {
	prev := line.prev
	next := line.next
	if prev != nil {
		prev.next = next
	}
	if next != nil {
		next.prev = prev
	}
}

func backspace() {
	if len(currentLine.text) == 0 {
		deleteLine(currentLine)
		up()
		textX = len(currentLine.text) - 1
		setXPos()
		draw()
	} else if textX == 0 && currentLine.prev != top {
		// shift line up
		oldLine := currentLine
		txt := currentLine.text
		prev := currentLine.prev
		newTextX := 0
		if prev != nil {
			newTextX = len(prev.text)
			prev.text += txt
		}
		up()
		deleteLine(oldLine)
		textX = newTextX
		setXPos()
		draw()
	} else if textX != 0 {
		deleteChar(textX)
	}
}

func insert() {
	clear()
	draw()
	flash("-- INSERT --")
	defer clearBanner()

	clipboard = currentLine.text

	for {
		c := getchar()
		switch c {
		case ESCAPE_CODE:
			walkBack()
			return
		case ENTER_CODE:
			prevText := currentLine.text[textX:]
			nextText := currentLine.text[:textX]
			startOfLine()

			newLine := lineNew()

			newLine.next = currentLine
			newLine.prev = currentLine.prev
			newLine.text = nextText
			currentLine.text = prevText
			currentLine.prev.next = newLine
			currentLine.prev = newLine
			currentLine = newLine
			down()
			clear()
			draw()
		case BACKSPACE_CODE:
			backspace()
		default:
			// add character to string at proper position
			pos := textX
			txt := currentLine.text
			if pos == len(currentLine.text) {
				currentLine.text = fmt.Sprintf("%s%c", txt, c)
				textX++
				screenX++
			} else {
				currentLine.text = fmt.Sprintf(
					"%s%c%s",
					txt[:pos],
					c,
					txt[pos:],
				)
			}
			if c == '\t' {
				screenX += 7
			}
			displayLine(currentLine.text, screenY)
			right()
		}
	}
}

func executeSearch(term string) {
	if currentLine == nil {
		return
	}
	i := lineno + 1
	for line := currentLine.next; line != nil; line = line.next {
		if strings.Contains(line.text, term) {
			goToNumber(i + 1)
			return
		}
		i++
	}
	flash(fmt.Sprintf("could not find '%s'", term))
}

func executeReverseSearch(term string) {
	if currentLine == nil {
		return
	}

	i := lineno
	for line := currentLine.prev; line != nil; line = line.prev {
		if i <= 0 {
			flash(fmt.Sprintf("could not find '%s'", term))
			break
		}
		if strings.Contains(line.text, term) {
			goToNumber(i)
			return
		}
		i--
	}
	flash(fmt.Sprintf("could not find '%s'", term))
}

func search() {
	oldScreenX := screenX
	oldScreenY := screenY

	defer func() {
		screenX = oldScreenX
	}()
	defer func() {
		screenY = oldScreenY
	}()

	screenY = height
	screenX = 2

	clearBanner()
	term := "/"
	for {
		flash(term)
		c := getchar()
		switch c {
		case ENTER_CODE:
			searchTerm = term[1:]
			executeSearch(searchTerm)
			return
		case ESCAPE_CODE:
			clearBanner()
			return
		case BACKSPACE_CODE:
			term = term[:len(term)-1]
			screenX--
			clearBanner()
			if screenX == 1 {
				return
			}
		default:
			term += string(c)
			screenX++
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func goToNumber(gotoNum int) {
	linesAway := gotoNum - lineno - 1
	if linesAway > 0 {
		for i := 0; i < linesAway; i++ {
			down()
		}
	} else if linesAway < 0 {
		for i := 0; i < abs(linesAway); i++ {
			up()
		}
	}
}

func execute(cmd string) {
	// is it a number?
	if gotoNum, err := strconv.Atoi(cmd); err == nil {
		goToNumber(gotoNum)
		return
	}

	// execute each letter command
	for _, c := range cmd {
		switch c {
		case 'w':
			writeFile()
		case 'q':
			quit = true
		default:
			clearBanner()
			flash(fmt.Sprintf(": unknown command: '%c'", c))
			return
		}
	}
}

func command() {
	oldScreenX := screenX
	oldScreenY := screenY

	defer func() {
		screenX = oldScreenX
	}()
	defer func() {
		screenY = oldScreenY
	}()

	screenY = height
	screenX = 2
	clearBanner()
	cmd := ":"
	for {
		flash(cmd)
		c := getchar()
		switch c {
		case ENTER_CODE:
			execute(cmd[1:])
			return
		case ESCAPE_CODE:
			clearBanner()
			return
		case BACKSPACE_CODE:
			cmd = cmd[:len(cmd)-1]
			screenX--
			clearBanner()
			if screenX == 1 {
				return
			}
		default:
			cmd += string(c)
			screenX++
		}
		draw()
	}
}

func writeFile() {
	file, err := os.Create(filename)
	if err != nil {
		flash(fmt.Sprintf("failed to write: '%s': %v", filename, err))
		return
	}
	defer file.Close()

	for line := top.next; line != nil; line = line.next {
		file.Write([]byte(line.text)) //nolint
		file.Write([]byte("\n"))      //nolint
	}
	flash(fmt.Sprintf("wrote file: \"%s\"", filename))
}

func displayLine(line string, y int) {
	move(1, y)
	fmt.Print("                                                           ")
	move(1, y)
	for i, c := range line {
		if i == width {
			break
		}
		if c == '\t' {
			fmt.Print("        ")
		} else {
			fmt.Printf("%c", c)
		}
	}
	restore()
}

func draw() {
	i := 0
	for line := topOfScreen; line != nil; line = line.next {
		if i >= height {
			break
		}
		displayLine(line.text, i)
		i++
	}

	for ; i < height; i++ {
		displayLine("~", i)
	}
}

func min(a int, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func setXPos() {
	screenX = 1
	if currentLine == nil {
		return
	}
	for i := 0; i < min(textX, len(currentLine.text)-1); i++ {
		c := currentLine.text[i]
		if c == '\t' {
			screenX += 8
		} else {
			screenX++
		}
	}
}

func goToTop() {
	clear()
	currentLine = top.next
	topOfScreen = top
	screenX = 1
	screenY = 1
	textX = 0
	lineno = 0
	move(screenX, screenY)
	draw()
	restore()
}

func gHandle() {
	c := getchar()
	switch c {
	case 'g':
		goToTop()
	default:
		flash(fmt.Sprintf("unknown command 'g%c'", c))
	}
}

func GHandle() {
	for line := currentLine; line.next != nil; line = line.next {
		down()
	}
}

func dHandle() {
	for {
		c := getchar()
		switch c {
		case 'd':
			clipboard = currentLine.text
			oldLine := currentLine
			up()
			deleteLine(oldLine)
			clear()
			draw()
			return
		default:
			flash(fmt.Sprintf("unknown command: 'd%c'", c))
			return
		}
	}
}

func yHandle() {
	for {
		c := getchar()
		switch c {
		case 'y':
			clipboard = currentLine.text
			return
		default:
			flash(fmt.Sprintf("unknown command: 'y%c'", c))
			return
		}
	}
}

func scan() {
	draw()
	for {
		if quit {
			return
		}
		displayLineno()

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
		case 'u':
			currentLine.text = clipboard
			startOfLine()
		case 'i':
			insert()
		case 'g':
			gHandle()
		case 'G':
			GHandle()
		case '$':
			fallthrough
		case 'E':
			if currentLine == nil {
				break
			}
			textX = len(currentLine.text) - 1
		case 'A':
			if len(currentLine.text) == 0 {
				insert()
			} else {
				textX = len(currentLine.text)
				setXPos()
				screenX++
				insert()
			}
		case 'o':
			newline := lineNew()
			newline.prev = currentLine
			newline.next = currentLine.next
			currentLine.next = newline
			down()
			startOfLine()
			clear()
			draw()
			insert()
		case 'r':
			clipboard = currentLine.text
			char := getchar()
			txt := currentLine.text
			currentLine.text = fmt.Sprintf(
				"%s%c%s",
				txt[:textX],
				char,
				txt[textX+1:],
			)
		case 'w':
			if len(currentLine.text) == 0 {
				down()
				break
			}
			ch := currentLine.text[textX]
			if ch == ' ' || ch == '\t' {
				for _, char := range currentLine.text[textX:] {
					right()
					if char != ' ' && char != '\t' {
						break
					}
					if textX >= len(currentLine.text)-1 || len(currentLine.text) == 0 {
						down()
						startOfLine()
					}
				}
				left()
			} else {
				for _, char := range currentLine.text[textX:] {
					right()
					if char == ' ' || char == '\t' {
						break
					}
					if textX >= len(currentLine.text)-1 || len(currentLine.text) == 0 {
						down()
						startOfLine()
					}
				}
			}
		case 'p':
			oldNext := currentLine.next
			newline := lineNew()
			newline.text = clipboard
			newline.prev = currentLine
			newline.next = oldNext
			oldNext.prev = newline
			currentLine.next = newline
		case 'y':
			yHandle()
		case 'd':
			dHandle()
		case 'D':
			clipboard = currentLine.text
			currentLine.text = currentLine.text[:textX]
		case 'x':
			clipboard = currentLine.text
			deleteChar(textX + 1)
			right()
		case 'n':
			executeSearch(searchTerm)
		case 'N':
			executeReverseSearch(searchTerm)
		case '/':
			search()
		case ':':
			command()
		case '0':
			startOfLine()
		case ESCAPE_CODE:
			break // do nothing
		default:
			flash(fmt.Sprintf("unknown command: '%c'", c))
		}
		setXPos()
	}
}

func eventLoop() error {
	var err error

	width, height, err = term.GetSize(0)
	if err != nil {
		return err
	}

	oldIn, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldIn) //nolint

	oldOut, err := term.MakeRaw(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdout.Fd()), oldOut) //nolint

	clear()
	move(screenX, screenY)
	scan()
	return nil
}

func initialSetup() {
	top = lineNew()
	currentLine = lineNew()
	top.next = currentLine
	currentLine.prev = top
	topOfScreen = top
}

func readFile(filename string) {
	readFile, err := os.Open(filename)
	if err != nil {
		return // file does not exist, so we'll create a new one
	}
	defer readFile.Close()

	top = lineNew()
	lines := top

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		line := lineNew()
		line.text = fileScanner.Text()
		lines.next = line
		line.prev = lines
		lines = lines.next
	}
	currentLine = top.next
	topOfScreen = top
}

func main() {
	initialSetup()
	defer clear()
	if len(os.Args) > 1 {
		filename = os.Args[1]
		readFile(filename)
	}
	err := eventLoop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
}
