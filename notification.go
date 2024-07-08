package main

import (
	"strings"
	"time"

	"github.com/openconfig/gnmi/proto/gnmi"
)

type UpdateCallback func(string, *time.Time, string)
type DeleteCallback func(string, *time.Time)

func collectPath(path *gnmi.Path) string {
	var b strings.Builder

	// TODO: This might scramble order, but we'll handle that if that
	// ever becomes a problem.
	for _, elem := range path.Elem {
		b.WriteString("/")
		b.WriteString(elem.Name)
		if elem.Key != nil {
			b.WriteString("[")
			first := true
			for k, v := range elem.Key {
				if !first {
					b.WriteString(",")
				}
				first = false
				b.WriteString(k)
				b.WriteString("=")
				b.WriteString(v)
			}
			b.WriteString("]")
		}
	}
	return b.String()
}

func WalkNotification(notif *gnmi.Notification, updateCb UpdateCallback, deleteCb DeleteCallback) {
	prefix := ""
	ts := time.UnixMicro(notif.Timestamp / 1000).UTC()
	if notif.Prefix != nil {
		prefix = collectPath(notif.Prefix)
	}
	if updateCb != nil {
		for _, upd := range notif.Update {
			fqn := prefix + collectPath(upd.Path)
			val := string(upd.Val.GetJsonIetfVal())
			updateCb(fqn, &ts, val)
		}
	}
	if deleteCb != nil {
		for _, dele := range notif.Delete {
			fqn := prefix + collectPath(dele)
			deleteCb(fqn, &ts)
		}
	}
}
