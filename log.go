package srcdom

import (
	"log"
	"os"
)

// WarnLog is log destination for warning messages
var WarnLog = log.New(os.Stderr, "srcdom:warn ", log.LstdFlags)

func warnf(s string, args ...interface{}) {
	if WarnLog != nil {
		WarnLog.Printf(s, args...)
	}
}

//// DebugLog is log destination for debug messages
//var DebugLog = log.New(os.Stderr, "srcdom:debug ", 0)
//
//func debugf(s string, args ...interface{}) {
//	if DebugLog != nil {
//		DebugLog.Printf(s, args...)
//	}
//}
