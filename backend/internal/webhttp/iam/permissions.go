package iam

// Catalog is the full permission point list (obj, act) for Admin UI.
var Catalog = [][2]string{
	{"portal", "access"}, {"admin", "access"}, {"app", "access"},
	{"user", "view"}, {"user", "edit"}, {"role", "view"}, {"role", "edit"}, {"audit", "view"},
	{"warehouse", "view"}, {"inbound", "edit"}, {"inbound", "confirm"}, {"outbound", "edit"}, {"outbound", "confirm"}, {"code", "issue"}, {"code", "void"},
	{"material", "view"}, {"material", "edit"}, {"material", "export"}, {"stock", "adjust"},
	{"approval", "view"}, {"approval", "act"},
	{"inventory", "view"}, {"inventory", "create"}, {"inventory", "scan"}, {"inventory", "adjust"},
	{"alarm", "view"}, {"alarm", "ack"}, {"threshold", "edit"},
	{"transport", "view"}, {"transport", "edit"}, {"unit", "edit"}, {"checkin", "exec"}, {"track", "view"},
	{"geofence", "view"}, {"geofence", "edit"},
	{"monitor", "view"}, {"device", "edit"},
	{"mqtt", "schema:edit"}, {"openapi", "config"},
	{"inspection", "view"}, {"inspection", "create"}, {"inspection", "exec"},
}

func CatalogStrings() []string {
	out := make([]string, 0, len(Catalog))
	for _, p := range Catalog {
		out = append(out, p[0]+":"+p[1])
	}
	return out
}
