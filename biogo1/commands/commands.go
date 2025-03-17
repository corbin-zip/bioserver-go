package commands

import "fmt"

const (
	SERVER     byte = 0x18 // a packet from the server
	CLIENT     byte = 0x81 // a packet from the client
	GAMESERVER byte = 0x28 // a packet from the gameserver
	GAMECLIENT byte = 0x82 // a packet from the gameclient

	QUERY     byte = 0x01 // ask a question
	TELL      byte = 0x02 // answer a question
	BROADCAST byte = 0x10 // send a packet to all clients
)

const (
	LOGIN        int = 0x6101
	UNKN61A0     int = 0x61A0 // TIMEOUTS ?
	CHECKVERSION int = 0x6103 // check the clients version
	CHECKRND     int = 0x600E // check random numbers
	UNKN61A1     int = 0x61A1
	IDHNPAIRS    int = 0x6131 // send available ID/HN pairs

	HNSELECT int = 0x6132 // which pair shall be used
	UNKN6104 int = 0x6104
	MOTHEDAY int = 0x614C // message of the day, kind of html/xml used

	CHARSELECT    int = 0x6190 // selected char and its statistics
	UNKN6881      int = 0x6881
	UNKN6882      int = 0x6882
	RANKINGS      int = 0x6145 // playerranking one can see in the area lobby
	UNKN6141      int = 0x6141
	AREACOUNT     int = 0x6203 // how many areas has this server
	AREAPLAYERCNT int = 0x6205 // number of players in the area
	AREASTATUS    int = 0x6206 // area available (0) or locked (3)
	AREANAME      int = 0x6204 // name of the area
	AREADESCRIPT  int = 0x620A // description of the area
	HEARTBEAT     int = 0x6202 // send every 30 secs to the clients

	AREASELECT    int = 0x6207 // choose area
	EXITAREA      int = 0x6209 // leave the roomlist (back to arealist)
	ROOMSCOUNT    int = 0x6301 // rooms in area
	ROOMPLAYERCNT int = 0x6303
	ROOMSTATUS    int = 0x6304 // status of a room
	ROOMNAME      int = 0x6302 // Name of a room
	UNKN6308      int = 0x6308

	ENTERROOM     int = 0x6305
	SLOTCOUNT     int = 0x6401 // How many gameslots are in the room ?
	SLOTPLRSTATUS int = 0x6403 // how many players are in slot / available in slot ?
	SLOTSTATUS    int = 0x6404 // is slot available, used or full ?
	SLOTTITLE     int = 0x6402 // title of the gameslot
	SLOTATTRIB2   int = 0x640B
	SLOTPWDPROT   int = 0x6405 // flag for password protection
	SLOTSCENTYPE  int = 0x650A // scenario and type (DVD/HDD) for this slot

	RULESCOUNT   int = 0x6603 // how many rules are there for slot ?
	RULEATTCOUNT int = 0x6607 // how many attributes has a rule ?
	UNKN6601     int = 0x6601
	UNKN6602     int = 0x6602
	RULEDESCRIPT int = 0x6604 // name of the rule
	RULEVALUE    int = 0x6606 // get value of rule
	RULEATTRIB   int = 0x6605 // additional attribute 2 of rule
	ATTRDESCRIPT int = 0x6608 // name of the choice
	ATTRATTRIB   int = 0x660E // attribute of the choice (always 0?)
	PLAYERSTATS  int = 0x640A // statistics of players in room
	EXITSLOTLIST int = 0x6408 // leave the slotlist (back to roomlist)
	CREATESLOT   int = 0x6407 // create a new slot
	SCENESELECT  int = 0x6509 // select scenario for slot
	SLOTNAME     int = 0x6609 // set name of the slot
	SETRULE      int = 0x660B // set rule for gameslot
	UNKN660C     int = 0x660C
	SLOTTIMER    int = 0x6409 // wait time for a gameslot
	UNKN6412     int = 0x6412
	UNKN6504     int = 0x6504
	CANCELSLOT   int = 0x6501 // cancel game in slot
	LEAVESLOT    int = 0x6502 // leave slot
	PLAYERSTATBC int = 0x6503 // broadcasting statistics of a joining player
	CANCELSLOTBC int = 0x6505 // broadcast when host cancels slot
	PLAYEROK     int = 0x6506 // broadcast when player is "unlocked"
	STARTGAME    int = 0x6508 // broadcast by host when game will be started

	CHATIN  int = 0x6701 // chat message from a client
	CHATOUT int = 0x6702 // chat message from server

	GETREADY     int = 0x6910 // broadcasted by server, clients request game details then
	PLAYERCOUNT  int = 0x6911 // total number of players for the game session
	PLAYERNUMBER int = 0x6912 // number of player
	PLAYERSTAT   int = 0x6913 // statistic of a player in slot
	PLAYERSCORE  int = 0x6917 // scoring from the ranklist for a player
	GAMESESSION  int = 0x6915 // the session number for this game
	GAMEDIFF     int = 0x6914 // difficulty of the game
	GSINFO       int = 0x6916 // gameserver info (192.168.2.1:8590)
	UNKN6002     int = 0x6002

	ENTERAGL     int = 0x6210 // entering the aftergame
	AGLSTATS     int = 0x6213 // stats after the game
	AGLPLAYERCNT int = 0x6212 // number of players in aftergame lobby
	LEAVEAGL     int = 0x6211 // leave after game lobby ?
	JOINGAME     int = 0x6406 // joining a slot not as host
	AGLLEAVE     int = 0x6214 // broadcasting that a player left
	AGLJOIN      int = 0x6215 // broadcast stats of client to aftergame lobby
	GETINFO      int = 0x6801 // request the information

	EVENTDAT   int = 0x670D
	EVENTDATBC int = 0x670E

	BUDDYLIST    int = 0x6707 // check status of a buddy
	CHECKBUDDY   int = 0x6703 // online status checker
	PRIVATEMSG   int = 0x6704 // private messaging
	PRIVATEMSGBC int = 0x6705 // broadcast of the private msg

	UNKN6181 int = 0x6181 // unknown, deep in the database functions ...
	// something along registration ?

	PATCHSTART     int = 0x6121 // indicates the start of a ptch file
	PATCHDATA      int = 0x6122 // block of 0x100 bytes patch
	PATCHFOOTER    int = 0x6123 // indicate the end of a patch file
	PATCHLINECHECK int = 0x6124 // after 8 blocks of patchdata this check packet occurs
	PATCHFINISH    int = 0x6125 // done ...

	SHUTDOWN int = 0x6003 // shutdown a client and put notice on screen

	CONNCHECK    int = 0x6001 // send every 60 secs to client
	LOGOUT       int = 0x6006
	SLOTPASSWD   int = 0x660A // set password for slot
	POSTGAMEINFO int = 0x6138 // statistics for the played game, used for rankings
	GSLOGIN      int = 0x1031 // first login packet for gameserver
)

// ------------------------------

var constNames = map[int]string{
	int(SERVER):     "SERVER",
	int(CLIENT):     "CLIENT",
	int(GAMESERVER): "GAMESERVER",
	int(GAMECLIENT): "GAMECLIENT",
	int(QUERY):      "QUERY",
	int(TELL):       "TELL",
	int(BROADCAST):  "BROADCAST",

	LOGIN:          "LOGIN",
	UNKN61A0:       "UNKN61A0",
	CHECKVERSION:   "CHECKVERSION",
	CHECKRND:       "CHECKRND",
	UNKN61A1:       "UNKN61A1",
	IDHNPAIRS:      "IDHNPAIRS",
	HNSELECT:       "HNSELECT",
	UNKN6104:       "UNKN6104",
	MOTHEDAY:       "MOTHEDAY",
	CHARSELECT:     "CHARSELECT",
	UNKN6881:       "UNKN6881",
	UNKN6882:       "UNKN6882",
	RANKINGS:       "RANKINGS",
	UNKN6141:       "UNKN6141",
	AREACOUNT:      "AREACOUNT",
	AREAPLAYERCNT:  "AREAPLAYERCNT",
	AREASTATUS:     "AREASTATUS",
	AREANAME:       "AREANAME",
	AREADESCRIPT:   "AREADESCRIPT",
	HEARTBEAT:      "HEARTBEAT",
	AREASELECT:     "AREASELECT",
	EXITAREA:       "EXITAREA",
	ROOMSCOUNT:     "ROOMSCOUNT",
	ROOMPLAYERCNT:  "ROOMPLAYERCNT",
	ROOMSTATUS:     "ROOMSTATUS",
	ROOMNAME:       "ROOMNAME",
	UNKN6308:       "UNKN6308",
	ENTERROOM:      "ENTERROOM",
	SLOTCOUNT:      "SLOTCOUNT",
	SLOTPLRSTATUS:  "SLOTPLRSTATUS",
	SLOTSTATUS:     "SLOTSTATUS",
	SLOTTITLE:      "SLOTTITLE",
	SLOTATTRIB2:    "SLOTATTRIB2",
	SLOTPWDPROT:    "SLOTPWDPROT",
	SLOTSCENTYPE:   "SLOTSCENTYPE",
	RULESCOUNT:     "RULESCOUNT",
	RULEATTCOUNT:   "RULEATTCOUNT",
	UNKN6601:       "UNKN6601",
	UNKN6602:       "UNKN6602",
	RULEDESCRIPT:   "RULEDESCRIPT",
	RULEVALUE:      "RULEVALUE",
	RULEATTRIB:     "RULEATTRIB",
	ATTRDESCRIPT:   "ATTRDESCRIPT",
	ATTRATTRIB:     "ATTRATTRIB",
	PLAYERSTATS:    "PLAYERSTATS",
	EXITSLOTLIST:   "EXITSLOTLIST",
	CREATESLOT:     "CREATESLOT",
	SCENESELECT:    "SCENESELECT",
	SLOTNAME:       "SLOTNAME",
	SETRULE:        "SETRULE",
	UNKN660C:       "UNKN660C",
	SLOTTIMER:      "SLOTTIMER",
	UNKN6412:       "UNKN6412",
	UNKN6504:       "UNKN6504",
	CANCELSLOT:     "CANCELSLOT",
	LEAVESLOT:      "LEAVESLOT",
	PLAYERSTATBC:   "PLAYERSTATBC",
	CANCELSLOTBC:   "CANCELSLOTBC",
	PLAYEROK:       "PLAYEROK",
	STARTGAME:      "STARTGAME",
	CHATIN:         "CHATIN",
	CHATOUT:        "CHATOUT",
	GETREADY:       "GETREADY",
	PLAYERCOUNT:    "PLAYERCOUNT",
	PLAYERNUMBER:   "PLAYERNUMBER",
	PLAYERSTAT:     "PLAYERSTAT",
	PLAYERSCORE:    "PLAYERSCORE",
	GAMESESSION:    "GAMESESSION",
	GAMEDIFF:       "GAMEDIFF",
	GSINFO:         "GSINFO",
	UNKN6002:       "UNKN6002",
	CONNCHECK:      "CONNCHECK",
	LOGOUT:         "LOGOUT",
	SLOTPASSWD:     "SLOTPASSWD",
	POSTGAMEINFO:   "POSTGAMEINFO",
	GSLOGIN:        "GSLOGIN",
	ENTERAGL:       "ENTERAGL",
	AGLSTATS:       "AGLSTATS",
	AGLPLAYERCNT:   "AGLPLAYERCNT",
	LEAVEAGL:       "LEAVEAGL",
	JOINGAME:       "JOINGAME",
	AGLLEAVE:       "AGLLEAVE",
	AGLJOIN:        "AGLJOIN",
	GETINFO:        "GETINFO",
	EVENTDAT:       "EVENTDAT",
	EVENTDATBC:     "EVENTDATBC",
	BUDDYLIST:      "BUDDYLIST",
	CHECKBUDDY:     "CHECKBUDDY",
	PRIVATEMSG:     "PRIVATEMSG",
	PRIVATEMSGBC:   "PRIVATEMSGBC",
	UNKN6181:       "UNKN6181",
	PATCHSTART:     "PATCHSTART",
	PATCHDATA:      "PATCHDATA",
	PATCHFOOTER:    "PATCHFOOTER",
	PATCHLINECHECK: "PATCHLINECHECK",
	PATCHFINISH:    "PATCHFINISH",
	SHUTDOWN:       "SHUTDOWN",
}

func GetConstName(v any) string {
	var key int
	switch t := v.(type) {
	case int:
		key = t
	case byte:
		key = int(t)
	default:
		return fmt.Sprintf("invalid type %T", v)
	}
	if name, ok := constNames[key]; ok {
		return name
	}
	return "UNKNOWN"
}
