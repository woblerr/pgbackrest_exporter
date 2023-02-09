package backrest

var stanzas []stanza

//	[{
//	  "archive": [{}],
//	  "cipher": "string",
//	  "backup": [{}],
//	  "db": [{}],
//	  "name": "string",
//	  "repo": [{}],
//	  "status": {}
//	}]
type stanza struct {
	Archive []archive `json:"archive"`
	Backup  []backup  `json:"backup"`
	Cipher  string    `json:"cipher"`
	DB      []db      `json:"db"`
	Name    string    `json:"name"`
	Repo    []repo    `json:"repo"`
	Status  status    `json:"status"`
}

//	"archive": [{
//	  "database": {},
//	  "id": "string",
//	  "max": "string",
//	  "min": "string"
//	}]
type archive struct {
	Database  databaseID `json:"database"`
	PGVersion string     `json:"id"`
	WALMax    string     `json:"max"`
	WALMin    string     `json:"min"`
}

//	"database": {
//	  "id": number,
//	  "repo-key": number
//	}
type databaseID struct {
	ID      int `json:"id"`
	RepoKey int `json:"repo-key"`
}

//	"backup": [{
//	  "archive": {
//	      "start": "string",
//	      "stop": "string"
//	  },
//	  "backrest": {
//	      "format": number,
//	      "version": "string"
//	  },
//	  "database": {},
//	  "error": bool,
//	  "info": {},
//	  "label": "string",
//	  "lsn": {
//	      "start": "string,
//	      "stop": "string"
//	  },
//	  "prior": "string",
//	  "reference": "string",
//	  "timestamp": {
//	      "start": number,
//	      "stop": number
//	  },
//	  "type": "string"
//	}]
type backup struct {
	Archive struct {
		StartWAL string `json:"start"`
		StopWAL  string `json:"stop"`
	} `json:"archive"`
	BackrestInfo struct {
		Format  int    `json:"format"`
		Version string `json:"version"`
	} `json:"backrest"`
	Database    databaseID     `json:"database"`
	DatabaseRef *[]databaseRef `json:"database-ref"`
	Error       *bool          `json:"error"`
	Info        backupInfo     `json:"info"`
	Label       string         `json:"label"`
	Link        *[]struct {
		Destination string `json:"destination"`
		Name        string `json:"name"`
	} `json:"link"`
	Lsn struct {
		StartLSN string `json:"start"`
		StopLSN  string `json:"stop"`
	} `json:"lsn"`
	Prior      string   `json:"prior"`
	Reference  []string `json:"reference"`
	Tablespace *[]struct {
		Destination string `json:"destination"`
		Name        string `json:"name"`
		OID         int    `json:"oid"`
	} `json:"tablespace"`
	Timestamp struct {
		Start int64 `json:"start"`
		Stop  int64 `json:"stop"`
	} `json:"timestamp"`
	Type string `json:"type"`
}

//	"database-ref": [{
//	  "name": "string",
//	  "oid": number
//	}]
type databaseRef struct {
	Name string `json:"name"`
	OID  int    `json:"oid"`
}

//	"info": {
//	  "delta": number,
//	  "repository": {
//	      "delta": number,
//	      "delta-map": number,
//	      "size": number,
//	      "size-map": number
//	  },
//	  "size": number
//	}
type backupInfo struct {
	Delta      int64 `json:"delta"`
	Repository struct {
		Delta    int64  `json:"delta"`
		DeltaMap *int64 `json:"delta-map"`
		Size     int64  `json:"size"`
		SizeMap  *int64 `json:"size-map"`
	} `json:"repository"`
	Size int64 `json:"size"`
}

//	"db": [{
//	  "id": number,
//	  "repo-key": number,
//	  "system-id": number,
//	  "version": "string"
//	}]
type db struct {
	ID       int    `json:"id"`
	RepoKey  int    `json:"repo-key"`
	SystemID int64  `json:"system-id"`
	Version  string `json:"version"`
}

//	"repo": [{
//	  "cipher": "string",
//	  "key": number,
//	  "status": {
//	      "code": number,
//	      "message": "string"
//	  }
//	}]
type repo struct {
	Cipher string `json:"cipher"`
	Key    int    `json:"key"`
	Status struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
}

//	"status": {
//	  "code": number,
//	  "lock": {
//	      "backup": {
//	          "held": bool
//	      }
//	  },
//	  "message": "string"
//	}
type status struct {
	Code int `json:"code"`
	Lock struct {
		Backup struct {
			Held bool `json:"held"`
		} `json:"backup"`
	} `json:"lock"`
	Message string `json:"message"`
}
