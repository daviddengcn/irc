/*
	irc is a command-line based IRC client application.
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/daviddengcn/go-irc"
	"github.com/hailiang/gosocks"
	"log"
	"net"
	"os"
	"strings"
)

var (
	proxy = flag.String("proxy", os.Getenv("IRC_PROXY"),
		`Proxy server. If not specified, use environmental variable IRC_PROXY.`)
	nick     = flag.String("nick", "Guest", `Nick name`)
	username = flag.String("user", "User", `User name`)
	password = flag.String("pass", "", `Password if any`)
	msgOnly  = flag.Bool("msgonly", false,
		`Only show messages, no join/quit notifications`)
	server  = "irc.freenode.net:6667"
	channel = "#go-nuts"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`  irc [<flags>] [<server:port> [<channel>]]
    If <server:port> is not specified or as !, it is set to "irc.freenode.net:6667".
    If <channel> is not set, it is set to "#go-nuts".`)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if flag.NArg() >= 1 {
		if flag.Arg(0) != "!" {
			server = flag.Arg(0)
		}
	}
	if flag.NArg() >= 2 {
		channel = flag.Arg(1)
	}

	var dial func(string, string) (net.Conn, error)
	if *proxy == "" {
		fmt.Printf("Connecting %s %s ...\n", server, channel)
		dial = net.Dial
	} else {
		fmt.Printf("Connecting %s %s throught proxy %s ...\n", server, channel, *proxy)
		dial = socks.DialSocksProxy(socks.SOCKS5, *proxy)
	}

	client := irc.NewClient(*nick, *username)
	client.Password = *password
	if !*msgOnly {
		client.DefaultHandler = func(e *irc.Event) {
			fmt.Printf("%s %+v\n", e.Code, *e)
		}

		client.SetHandler(irc.JOIN, func(e *irc.Event) {
			fmt.Printf("> %s has joined\n", e.Nick)
		})
		client.SetHandler(irc.QUIT, func(e *irc.Event) {
			fmt.Printf("> %s has quit\n", e.Nick)
		})

		client.SetHandler(irc.PART, func(e *irc.Event) {
			fmt.Printf("> %s leaves channel %s\n", e.Nick, e.Arguments[0])
		})
	}

	socket, err := dial("tcp", server)
	if err != nil {
		log.Fatal(err)
	}
	client.Start(socket)

	client.SetHandler(irc.RPL_TOPIC, func(e *irc.Event) {
		fmt.Printf("> Topic: %s\n", e.Message)
	})
	client.SetHandler(irc.PRIVMSG, func(e *irc.Event) {
		fmt.Printf("> %s: %s\n", e.Nick, e.Message)
	})

	client.SetHandler(irc.RPL_LUSERCHANNELS, func(e *irc.Event) {
		fmt.Printf("> %s %s\n", e.Arguments[1], e.Message)
	})

	for _, c := range []string{
		irc.MODE, irc.NOTICE, irc.ERROR,
		irc.RPL_YOURHOST, irc.RPL_LUSERCLIENT, irc.RPL_LUSEROP,
		irc.RPL_LUSERUNKNOWN, irc.RPL_LUSERME, irc.RPL_LOCALUSERS,
		irc.RPL_GLOBALUSERS, irc.RPL_MOTDSTART, irc.RPL_STATSCONN,
	} {
		client.SetHandler(c, func(e *irc.Event) {
			fmt.Printf("> %s %s\n", e.Code, e.Message)
		})
	}
	client.Join(channel)

	ignored := func(*irc.Event) {}
	for _, c := range []string{
		irc.RPL_NAMREPLY, irc.RPL_ENDOFNAMES, irc.RPL_MOTD, irc.RPL_ENDOFMOTD,
		irc.RPL_CREATED, irc.RPL_MYINFO, irc.RPL_ISUPPORT,
	} {
		client.SetHandler(c, ignored)
	}

	go func() {
		rd := bufio.NewReader(os.Stdin)
		for {
			line, _ := rd.ReadString('\n')
			line = strings.TrimRight(line, "\r\n")
			if line == "/quit" {
				client.Quit()
				break
			}
			client.Privmsg(channel, line)
		}
	}()

	if err := client.Serve(); err != nil {
		log.Fatal(err)
	}
}
