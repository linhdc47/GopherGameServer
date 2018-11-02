// This package is used to start and change the core settings for the Gopher Game Server. The
// type ServerSettings contains all the parameters for changing the core settings. You can either
// pass a ServerSettings when calling Server.Start() or nil if you want to use the default server
// settings.
package gopher

import (
	"github.com/hewiefreeman/GopherGameServer/rooms"
	"github.com/hewiefreeman/GopherGameServer/users"
	"github.com/hewiefreeman/GopherGameServer/actions"
	"math/rand"
	"time"
	"net/http"
	"strconv"
)

/////////// TO DOs:
///////////    - Voice chat
///////////    	- Mutable Users
///////////    - Change Users' roomNames when deleting a Room
///////////    - SQL Authentication:
///////////    	- Initialization that checks if database is set-up and configured correctly, and if not configures it correctly.
///////////    	- CRUD+ helpers
///////////    	- SQL Authentication
///////////         - "Remember Me" login key pairs
///////////         - Database helpers for developers
///////////    - Multi-connect
///////////    - SQLite Database:
///////////    	- CRUD helpers
///////////    	- Save state on shut-down
///////////    - Admin tools

// Core server settings for the Gopher Game Server
type ServerSettings struct {
	ServerName string // The server's name. Used for the server's ownership of private Rooms.

	HostName string // Server's host name. Use 'https://' for TLS connections. (ex: 'https://example.com')
	HostAlias string // Server's host alias name. Use 'https://' for TLS connections. (ex: 'https://www.example.com')
	IP string // Server's IP address.
	Port int // Server's port.

	TLS bool // Enables TLS/SSL connections.
	CertFile string // SSL/TLS certificate file location (starting from system's root folder).
	PrivKeyFile string // SSL/TLS private key file location (starting from system's root folder).

	OriginOnly bool // When enabled, the server declines connections made from outside the origin server. IMPORTANT: Enable this for web apps and LAN servers.

	MultiConnect bool // Enabled multiple connections under the same User. When enabled, will override KickDupOnLogin's functionality. (TO DO - THIRD TO LAST)
	KickDupOnLogin bool // When enabled, a logged in User will be disconnected from service when another User logs in with the same name.

	UserRoomControl bool // Enables Users to create Rooms, invite/uninvite(AKA revoke) other Users to their owned private rooms, and destroy their owned rooms.
	RoomDeleteOnLeave bool // When enabled, Rooms created by a User will be deleted when the owner leaves.

	EnableSqlAuth bool // Enables the built-in SQL User authentication. (TO DO)
	SqlIP string // SQL Database IP address. (TO DO)
	SqlPort int // SQL Database port. (TO DO)

	EnableRecovery bool // Enables the recovery of all Rooms, their settings, and their variables on start-up after terminating the server. (TO DO - SECOND TO LAST)
	RecoveryLocation string // The folder location (starting from system's root folder) where you would like to store the recovery data. (TO DO - SECOND TO LAST)

	EnableAdminTools bool // Enables the use of the Admin Tools (TO DO - LAST)
	EnableRemoteAdmin bool // Enabled administration (only) from outside the origin server. When enabled, will override OriginOnly's functionality, but only for administrator connections. (TO DO - LAST)
	AdminToolsLogin string // The login name for the Admin Tools (TO DO - LAST)
	AdminToolsPassword string // The password for the Admin Tools (TO DO - LAST)
}

var (
	settings *ServerSettings
)

// Call with a pointer to your ServerSettings (or nil for defaults) to start the server. The default
// settings are for local testing ONLY. There are security-related options in ServerSettings
// for SSL/TLS, connection origin testing, Admin Tools, and more. It's highly recommended to look into
// all ServerSettings options to tune the server for your desired functionality and security needs.
func Start(s *ServerSettings){
	//SET SERVER SETTINGS
	if(s != nil){
		settings = s;
	}else{
		//DEFAULT localhost SETTINGS
		settings = &ServerSettings{
					ServerName: "!server!",

					HostName: "localhost",
					HostAlias: "localhost",
					IP: "localhost",
					Port: 8080,

					TLS: false,
					CertFile: "",
					PrivKeyFile: "",

					OriginOnly: false,

					MultiConnect: false,
					KickDupOnLogin: false,

					UserRoomControl: true,
					RoomDeleteOnLeave: true,

					EnableSqlAuth: false,
					SqlIP: "localhost",
					SqlPort: 3306,

					EnableRecovery: false,
					RecoveryLocation: "C:/",

					EnableAdminTools: true,
					EnableRemoteAdmin: false,
					AdminToolsLogin: "admin",
					AdminToolsPassword: "password" }
	}

	//SEED THE rand LIBRARY
	rand.Seed(time.Now().UTC().UnixNano());

	//UPDATE SETTINGS IN PACKAGES
	users.SettingsSet((*settings).KickDupOnLogin, (*settings).ServerName, (*settings).RoomDeleteOnLeave);

	//NOTIFY PACKAGES OF SERVER START
	users.SetServerStarted(true);
	rooms.SetServerStarted(true);
	actions.SetServerStarted(true);

	//START HTTP/SOCKET LISTENER
	if(settings.TLS){
		http.HandleFunc("/wss", socketInitializer);
		panic(http.ListenAndServeTLS(settings.IP+":"+strconv.Itoa(settings.Port), settings.CertFile, settings.PrivKeyFile, nil));
	}else{
		http.HandleFunc("/ws", socketInitializer);
		panic(http.ListenAndServe(settings.IP+":"+strconv.Itoa(settings.Port), nil));
	}
}
