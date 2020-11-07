package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/wwt/guac"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	servlet := guac.NewServer(DemoDoConnect)
	authServlet := guac.NewAuthManager(servlet)
	wsServer := guac.NewWebsocketServer(DemoDoConnect)
	authWsServer := guac.NewAuthManager(wsServer)

	sessions := guac.NewMemorySessionStore()
	wsServer.OnConnect = sessions.Add
	wsServer.OnDisconnect = sessions.Delete

	setting := guac.GetSetting()

	authSetting := guac.NewAuthManager(setting)

	mux := http.NewServeMux()
	mux.Handle("/tunnel", authServlet)
	mux.Handle("/tunnel/", authServlet)
	mux.Handle("/websocket-tunnel", authWsServer)
	authSessionHandle := guac.NewAuthManagerWithFunc(sessions.HandleSession)
	mux.HandleFunc("/sessions/", authSessionHandle.ServeHTTP)

	fs := http.FileServer(http.Dir(setting.Server.Static.Path))

	authStaticHandler := guac.NewAuthManager(http.StripPrefix("/", fs))

	mux.Handle("/", authStaticHandler)

	mux.Handle("/config", authSetting)

	logrus.Println("Serving on http://127.0.0.1:4567")

	s := &http.Server{
		Addr:           "0.0.0.0:4567",
		Handler:        mux,
		ReadTimeout:    guac.SocketTimeout,
		WriteTimeout:   guac.SocketTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	err := s.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}
func createNewSetting(config *guac.Config, query url.Values) {
	config.Protocol = query.Get("scheme")
	config.Parameters = map[string]string{}
	for k, v := range query {
		config.Parameters[k] = v[0]
	}
	guac.AddConn(config.Parameters)
}

func readNewSetting(config *guac.Config, query url.Values) {
	config.Protocol = query.Get("scheme")
	config.Parameters = map[string]string{}
	for k, v := range query {
		config.Parameters[k] = v[0]
	}
	guac.AddConn(config.Parameters)
}

// DemoDoConnect creates the tunnel to the remote machine (via guacd)
func DemoDoConnect(request *http.Request) (guac.Tunnel, error) {
	config := guac.NewGuacamoleConfiguration()
	var query url.Values
	if request.URL.RawQuery == "connect" {
		// http tunnel uses the body to pass parameters
		data, err := ioutil.ReadAll(request.Body)
		if err != nil {
			logrus.Error("Failed to read body ", err)
			return nil, err
		}
		_ = request.Body.Close()
		queryString := string(data)
		query, err = url.ParseQuery(queryString)
		if err != nil {
			logrus.Error("Failed to parse body query ", err)
			return nil, err
		}
		logrus.Debugln("body:", queryString, query)
	} else {
		query = request.URL.Query()
	}
	var guacdAddress = guac.GetGuacd()
	if query.Get("id") != "" {
		var connSeting = guac.GetConn(query.Get("id"))
		if connSeting != nil {
			config.Protocol = connSeting["scheme"]
			config.Parameters = guac.GetConn(query.Get("id"))
			if connSeting["guacd"] != "" {
				guacdAddress = connSeting["guacd"]
			}
		} else {
			createNewSetting(config, query)
		}
	} else {
		createNewSetting(config, query)
	}
	if guac.GetSetting().Guacd.Override && query.Get("guacd") != "" {
		guacdAddress = query.Get("guacd")
	}
	var err error
	if query.Get("width") != "" {
		config.OptimalScreenHeight, err = strconv.Atoi(query.Get("width"))
		if err != nil || config.OptimalScreenHeight == 0 {
			logrus.Error("Invalid height")
			config.OptimalScreenHeight = 600
		}
	}
	if query.Get("height") != "" {
		config.OptimalScreenWidth, err = strconv.Atoi(query.Get("height"))
		if err != nil || config.OptimalScreenWidth == 0 {
			logrus.Error("Invalid width")
			config.OptimalScreenWidth = 800
		}
	}
	config.AudioMimetypes = []string{"audio/L16", "rate=44100", "channels=2"}

	logrus.Debug("Connecting to guacd")

	addr, err := net.ResolveTCPAddr("tcp", guacdAddress)

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		logrus.Errorln("error while connecting to guacd", err)
		return nil, err
	}

	stream := guac.NewStream(conn, guac.SocketTimeout)

	logrus.Debugf("Connected to guacd %s", guacdAddress)
	if request.URL.Query().Get("uuid") != "" {
		config.ConnectionID = request.URL.Query().Get("uuid")
	}
	logrus.Debugf("Starting handshake with %#v", config)
	err = stream.Handshake(config)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Socket configured")
	return guac.NewSimpleTunnel(stream), nil
}
