package main

import (
	//	"bitbucket.org/kardianos/osext"
	"fmt"
	"log"
	//	"github.com/mattn/go-gtk/gdkpixbuf"
	//	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	height int = 3
	width  int = 3
)

var window *gtk.Window
var button [][]*gtk.Button
var state [][]int
var address map[string]interface{}
var isMyTurn bool
var blank, nought, cross *os.File
var startButton *gtk.Button
var addressList *gtk.ComboBoxText
var opponent string

func main() {
	opponent = ""
	isMyTurn = false
	blank = readResource("blank.png")
	nought = readResource("nought.png")
	cross = readResource("cross.png")
	log.Println(cross.Name())

	gtk.Init(nil)
	window = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetTitle("GTK Icon View")
	window.Connect("destroy", gtk.MainQuit)

	address = make(map[string]interface{})

	alignment := gtk.NewAlignment(1, 1, 1, 1)
	table := gtk.NewTable(3, 3, false)
	button = make([][]*gtk.Button, height, height)
	state = make([][]int, height, height)
	for i := 0; i < height; i++ {
		button[i] = make([]*gtk.Button, width, width)
		state[i] = make([]int, width, width)
		for j := 0; j < width; j++ {
			button[i][j] = gtk.NewButton()
			image := gtk.NewImageFromFile(blank.Name())
			button[i][j].SetImage(image)
			table.Attach(button[i][j], uint(i), uint(i+1), uint(j), uint(j+1), gtk.FILL, gtk.FILL, 0, 0)

			state[i][j] = 0
			copiedI := i
			copiedJ := j
			button[i][j].Connect("clicked", func() {
				if opponent != "" && isMyTurn == true && state[copiedI][copiedJ] == 0 {
					state[copiedI][copiedJ] = 1
					image := gtk.NewImageFromFile(nought.Name())
					button[copiedI][copiedJ].SetImage(image)
					message := "done " + strconv.Itoa(copiedI) + " " + strconv.Itoa(copiedJ) + " "
					sendMessage(opponent, message)
					judge()
					isMyTurn = false
				}
			})

		}
	}
	addressList = gtk.NewComboBoxText()

	//startButton
	startButton = gtk.NewButtonWithLabel("Start")
	startButton.Connect("clicked", func() {
		if addressList.GetActiveText() != "" {
			sendMessage(addressList.GetActiveText(), "start ")
			startButton.SetState(gtk.STATE_INSENSITIVE)
		}

	})
	go listenUsers(addressList)
	go broadcast()

	fixed := gtk.NewFixed()

	alignment.Add(table)
	fixed.Put(alignment, 0, 0)
	fixed.Put(addressList, 0, 330)
	fixed.Put(startButton, 200, 330)

	window.Add(fixed)

	window.SetSizeRequest(400, 400)
	window.ShowAll()

	gtk.Main()
}

func sendMessage(remoteAddr, message string) {
	serverAddr, err := net.ResolveUDPAddr("udp", remoteAddr+":6392")
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Panic("Unicast cannot be sent")
	}
	log.Println("Send: " + message)
	fmt.Fprintf(conn, message)
}

func judge() {
	msg := []string{"", "You've won.", "You've lost."}
	for n := 1; n <= 2; n++ {
		if (state[0][0] == n && state[1][0] == n && state[2][0] == n) ||
			(state[0][1] == n && state[1][1] == n && state[2][1] == n) ||
			(state[0][2] == n && state[1][2] == n && state[2][2] == n) ||
			(state[0][0] == n && state[0][1] == n && state[0][2] == n) ||
			(state[1][0] == n && state[1][1] == n && state[1][2] == n) ||
			(state[2][0] == n && state[2][1] == n && state[2][2] == n) ||
			(state[0][0] == n && state[1][1] == n && state[2][2] == n) ||
			(state[2][0] == n && state[1][1] == n && state[0][2] == n) {
			log.Println("aaa")
			log.Println(msg[n])
			dialog := gtk.NewMessageDialog(window, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_OK, msg[n])
			log.Println("bbb")
			dialog.SetTitle("Game Over")
			dialog.Response(func() {
				dialog.Destroy()
			})
			log.Println("ccc")
			dialog.Run()
			log.Println("ddd")
			for i := 0; i < height; i++ {
				for j := 0; j < width; j++ {
					image := gtk.NewImageFromFile(blank.Name())
					button[i][j].SetImage(image)
					state[i][j] = 0
				}
			}
			opponent = ""
			startButton.SetState(gtk.STATE_NORMAL)
			isMyTurn = false
			break
		}
	}
}

func listenUsers(addressList *gtk.ComboBoxText) {
	serverAddr, err := net.ResolveUDPAddr("udp", ":6392")
	l, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	for {
		// Wait for a connection.
		//log.Println("Broadcast listening")
		data := make([]byte, 4096)
		_, remoteAddr, err := l.ReadFromUDP(data)
		if err != nil {

		}
		//localAddr := l.LocalAddr()
		message := strings.Split(string(data), " ")
		switch message[0] {
		case "broadcast":
			addressAry := strings.Split(remoteAddr.String(), ":")
			if _, ok := address[addressAry[0]]; ok == false {
				address[addressAry[0]] = nil
				addressList.AppendText(addressAry[0])
				log.Println("Broadcast accept: " + remoteAddr.String())
			}
		case "start":
			addressAry := strings.Split(remoteAddr.String(), ":")
			if opponent == "" {
				opponent = addressAry[0]
				sendMessage(opponent, "accept ")
				startButton.SetState(gtk.STATE_INSENSITIVE)
				isMyTurn = true
			} else {
				sendMessage(addressAry[0], "deny ")
			}
		case "accept":
			addressAry := strings.Split(remoteAddr.String(), ":")
			opponent = addressAry[0]
		case "deny":
			dialog := gtk.NewMessageDialog(window, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_OK, "Opponent Busy")
			dialog.SetTitle("Opponent Busy")
			dialog.Response(func() {
				dialog.Destroy()
			})
			dialog.Run()
			startButton.SetState(gtk.STATE_NORMAL)
		case "done":
			fmt.Println(message[0])
			fmt.Println(message[1])
			fmt.Println(message[2])
			i, _ := strconv.Atoi(message[1])
			j, _ := strconv.Atoi(message[2])
			fmt.Println(i)
			fmt.Println(j)
			state[i][j] = 2
			image := gtk.NewImageFromFile(cross.Name())
			button[i][j].SetImage(image)
			isMyTurn = true
			judge()
			//	if err != nil {
			//		log.Fatal(err)
			//	}
		}
	}
}

func broadcast() {
	serverAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:6392")
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Panic("Broadcast cannot be sent")
	}
	log.Println("Broadcasting...")
	for {
		fmt.Fprintf(conn, "broadcast ")
		time.Sleep(time.Second)
	}
}

func readResource(file string) *os.File {
	filename := os.Args[0]
	result, err := os.Open(path.Join(path.Dir(filename), file))
	if err != nil {
		result, err = os.Open(file)
		if err != nil {
			log.Panicln("Resource file " + file + " are not found.")
		}
	}
	return result
}
