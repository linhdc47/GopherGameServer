// Package gopher is used to start and change the core settings for the Gopher Game Server. The
// type ServerSettings contains all the parameters for changing the core settings. You can either
// pass a ServerSettings when calling Server.Start() or nil if you want to use the default server
// settings.
package gopher

import (
	"context"
	"fmt"
	"github.com/hewiefreeman/GopherGameServer/actions"
	"github.com/hewiefreeman/GopherGameServer/database"
	"github.com/hewiefreeman/GopherGameServer/rooms"
	"github.com/hewiefreeman/GopherGameServer/users"
	"net/http"
	"strconv"
)

/////////// TO DOs:
///////////	- Save state on shut-down
///////////		- Test
///////////	- Admin tools

// ServerSettings are the core settings for the Gopher Game Server. You must fill one of these out to customize
// the server's functionality to your liking.
type ServerSettings struct {
	ServerName     string // The server's name. Used for the server's ownership of private Rooms. (Required)
	MaxConnections int    // The maximum amount of concurrent connections the server will accept. Setting this to 0 means infinite.

	HostName  string // Server's host name. Use 'https://' for TLS connections. (ex: 'https://example.com') (Required)
	HostAlias string // Server's host alias name. Use 'https://' for TLS connections. (ex: 'https://www.example.com')
	IP        string // Server's IP address. (Required)
	Port      int    // Server's port. (Required)

	TLS         bool   // Enables TLS/SSL connections.
	CertFile    string // SSL/TLS certificate file location (starting from system's root folder). (Required for TLS)
	PrivKeyFile string // SSL/TLS private key file location (starting from system's root folder). (Required for TLS)

	OriginOnly bool // When enabled, the server declines connections made from outside the origin server (Admin logins always check origin). IMPORTANT: Enable this for web apps and LAN servers.

	MultiConnect   bool // Enables multiple connections under the same User. When enabled, will override KickDupOnLogin's functionality.
	KickDupOnLogin bool // When enabled, a logged in User will be disconnected from service when another User logs in with the same name.

	UserRoomControl   bool // Enables Users to create Rooms, invite/uninvite(AKA revoke) other Users to their owned private rooms, and destroy their owned rooms.
	RoomDeleteOnLeave bool // When enabled, Rooms created by a User will be deleted when the owner leaves. WARNING: If disabled, you must remember to at some point delete the rooms created by Users, or they will pile up endlessly!

	EnableSqlFeatures bool   // Enables the built-in SQL User authentication and friending. NOTE: It is HIGHLY recommended to use TLS over an SSL/HTTPS connection when using the SQL features. Otherwise, sensitive User information can be compromised with network "snooping" (AKA "sniffing").
	SqlIP             string // SQL Database IP address. (Required for SQL features)
	SqlPort           int    // SQL Database port. (Required for SQL features)
	SqlProtocol       string // The protocol to use while comminicating with the MySQL database. Most use either 'udp' or 'tcp'. (Required for SQL features)
	SqlUser           string // SQL user name (Required for SQL features)
	SqlPassword       string // SQL user password (Required for SQL features)
	SqlDatabase       string // SQL database name (Required for SQL features)
	EncryptionCost    int    // The amount of encryption iterations the server will run when storing and checking passwords. The higher the number, the longer encryptions take, but are more secure. Default is 4, range is 4-31.
	CustomLoginColumn string // The custom AccountInfoColumn you wish to use for logging in instead of the default name column.
	RememberMe        bool   // Enables the "Remember Me" login feature. You can read more about this in project's "Usage" section.

	EnableRecovery   bool   // Enables the recovery of all Rooms, their settings, and their variables on start-up after terminating the server.
	RecoveryLocation string // The folder location (starting from system's root folder) where you would like to store the recovery data. (Required for recovery)

	EnableAdminTools   bool   // Enables the use of the Admin Tools
	EnableRemoteAdmin  bool   // Enabled administration (only) from outside the origin server. When enabled, will override OriginOnly's functionality, but only for administrator connections.
	AdminToolsLogin    string // The login name for the Admin Tools (Required for Admin Tools)
	AdminToolsPassword string // The password for the Admin Tools (Required for Admin Tools)
}

var (
	httpServer *http.Server

	settings *ServerSettings

	serverStarted  bool       = false
	serverPaused   bool       = false
	serverStopping bool       = false
	serverEndChan  chan error = make(chan error)

	startCallback         func()
	pauseCallback         func()
	stopCallback          func()
	resumeCallback        func()
	clientConnectCallback func(*http.ResponseWriter, *http.Request) bool

	//SERVER VERSION NUMBER
	version string = "1.0-ALPHA.3"
)

// Start will start the server. Call with a pointer to your `ServerSettings` (or nil for defaults) to start the server. The default
// settings are for local testing ONLY. There are security-related options in `ServerSettings`
// for SSL/TLS, connection origin testing, administrator tools, and more. It's highly recommended to look into
// all `ServerSettings` options to tune the server for your desired functionality and security needs.
//
// This function will block the thread that it is ran on until the server either errors, or is manually shut-down. To run code after the
// server starts/stops/pauses/etc, use the provided server callback setter functions.
func Start(s *ServerSettings) {
	fmt.Println(" _____             _               _____\n|  __ \\           | |             /  ___|\n| |  \\/ ___  _ __ | |__   ___ _ __\\ `--.  ___ _ ____   _____ _ __\n| | __ / _ \\| '_ \\| '_ \\ / _ \\ '__|`--. \\/ _ \\ '__\\ \\ / / _ \\ '__|\n| |_\\ \\ (_) | |_) | | | |  __/ |  /\\__/ /  __/ |   \\ V /  __/ |\n \\____/\\___/| .__/|_| |_|\\___|_|  \\____/ \\___|_|    \\_/ \\___|_|\n            | |\n            |_|                                      v" + version + "\n\n")
	fmt.Println("Starting server...")
	//SET SERVER SETTINGS
	if s != nil {
		settings = s
		if settings.ServerName == "" {
			fmt.Println("ServerName in ServerSettings is required. Shutting down...")

		} else if settings.HostName == "" || settings.IP == "" || settings.Port > 1 {
			fmt.Println("HostName, IP, and Port in ServerSettings are required. Shutting down...")

		} else if settings.TLS == true && (settings.CertFile == "" || settings.PrivKeyFile == "") {
			fmt.Println("CertFile and PrivKeyFile in ServerSettings are required for a TLS connection. Shutting down...")

		} else if settings.EnableSqlFeatures == true && (settings.SqlIP == "" || settings.SqlPort < 1 || settings.SqlProtocol == "" ||
			settings.SqlUser == "" || settings.SqlPassword == "" || settings.SqlDatabase == "") {
			fmt.Println("SqlIP, SqlPort, SqlProtocol, SqlUser, SqlPassword, and SqlDatabase in ServerSettings are required for the SQL features. Shutting down...")

		} else if settings.EnableRecovery == true && settings.RecoveryLocation == "" {
			fmt.Println("RecoveryLocation in ServerSettings is required for server recovery. Shutting down...")

		} else if settings.EnableAdminTools == true && (settings.AdminToolsLogin == "" || settings.AdminToolsPassword == "") {
			fmt.Println("AdminToolsLogin and AdminToolsPassword in ServerSettings are required for Administrator Tools. Shutting down...")

		}
	} else {
		//DEFAULT localhost SETTINGS
		fmt.Println("Using default settings...")
		settings = &ServerSettings{
			ServerName:     "!server!",
			MaxConnections: 0,

			HostName:  "localhost",
			HostAlias: "localhost",
			IP:        "localhost",
			Port:      8080,

			TLS:         false,
			CertFile:    "",
			PrivKeyFile: "",

			OriginOnly: false,

			MultiConnect:   false,
			KickDupOnLogin: false,

			UserRoomControl:   true,
			RoomDeleteOnLeave: true,

			EnableSqlFeatures: false,
			SqlIP:             "localhost",
			SqlPort:           3306,
			SqlProtocol:       "tcp",
			SqlUser:           "user",
			SqlPassword:       "password",
			SqlDatabase:       "database",
			EncryptionCost:    4,
			CustomLoginColumn: "",
			RememberMe:        false,

			EnableRecovery:   false,
			RecoveryLocation: "C:/",

			EnableAdminTools:   true,
			EnableRemoteAdmin:  false,
			AdminToolsLogin:    "admin",
			AdminToolsPassword: "password"}
	}

	//UPDATE SETTINGS IN users PACKAGE, THEN users WILL UPDATE SETTINGS FOR rooms PACKAGE
	users.SettingsSet((*settings).KickDupOnLogin, (*settings).ServerName, (*settings).RoomDeleteOnLeave, (*settings).EnableSqlFeatures,
		(*settings).RememberMe, (*settings).MultiConnect)

	//NOTIFY PACKAGES OF SERVER START
	serverStarted = true
	users.SetServerStarted(true)
	rooms.SetServerStarted(true)
	actions.SetServerStarted(true)
	database.SetServerStarted(true)

	//START UP DATABASE
	if (*settings).EnableSqlFeatures {
		fmt.Println("Initializing database...")
		dbErr := database.Init((*settings).SqlUser, (*settings).SqlPassword, (*settings).SqlDatabase,
			(*settings).SqlProtocol, (*settings).SqlIP, (*settings).SqlPort, (*settings).EncryptionCost,
			(*settings).RememberMe, (*settings).CustomLoginColumn)
		if dbErr != nil {
			fmt.Println("Database error:", dbErr.Error())
		}
		fmt.Println("Database initialized")
	}

	//RECOVER PREVIOUS SERVER STATE

	//START HTTP/SOCKET LISTENER
	if settings.TLS {
		httpServer = makeServer("/wss", settings.TLS)
	} else {
		httpServer = makeServer("/ws", settings.TLS)
	}

	//RUN START CALLBACK
	if startCallback != nil {
		startCallback()
	}

	//START MACRO COMMAND LISTENER
	go macroListener()

	fmt.Println("Startup complete")

	//BLOCK UNTIL SERVER SHUT-DOWN
	doneErr := <-serverEndChan

	if doneErr != http.ErrServerClosed {
		fmt.Println("Fatal server error:", doneErr.Error())
		fmt.Println("Shutting server down...")

		if !serverStopping {
			//PAUSE SERVER
			Pause()

			//SAVE STATE
			fmt.Println("Saving server state...")
		}
	}

	fmt.Println("Server shut-down completed")
}

func makeServer(handleDir string, tls bool) *http.Server {
	server := &http.Server{Addr: settings.IP + ":" + strconv.Itoa(settings.Port)}
	http.HandleFunc(handleDir, socketInitializer)
	if tls {
		go func() {
			err := server.ListenAndServeTLS(settings.CertFile, settings.PrivKeyFile)
			serverEndChan <- err
		}()
	} else {
		go func() {
			err := server.ListenAndServe()
			serverEndChan <- err
		}()
	}

	//
	return server
}

// Pause will log all Users off and prevent anyone from logging in. All rooms and their variables created by the server will remain in memory.
// Same goes for rooms created by Users unless `RoomDeleteOnLeave` in `ServerSettings` is set to true.
func Pause() {
	if !serverPaused {
		serverPaused = true

		fmt.Println("Pausing server...")

		users.Pause()
		rooms.Pause()
		actions.Pause()
		database.Pause()

		//RUN CALLBACK
		if pauseCallback != nil {
			pauseCallback()
		}

		fmt.Println("Server paused")

		serverStarted = false
	}

}

// Resume will allow Users to login again after pausing the server.
func Resume() {
	if serverPaused {
		serverStarted = true

		fmt.Println("Resuming server...")
		users.Resume()
		rooms.Resume()
		actions.Resume()
		database.Resume()

		//RUN CALLBACK
		if resumeCallback != nil {
			resumeCallback()
		}

		fmt.Println("Server resumed")

		serverPaused = false
	}
}

// Stop will log all Users off, save the state of the server if EnableRecovery in ServerSettings is set to true, then shut the server down.
func ShutDown() error {
	fmt.Println("Stopping server...")

	//PAUSE SERVER
	Pause()

	//
	serverStopping = true

	//SAVE STATE
	fmt.Println("Saving server state...")

	//SHUT DOWN SERVER
	fmt.Println("Shutting server down...")
	shutdownErr := httpServer.Shutdown(context.Background())

	//RUN CALLBACK
	if stopCallback != nil {
		stopCallback()
	}

	//
	if shutdownErr != http.ErrServerClosed {
		return shutdownErr
	}

	//
	return nil
}
