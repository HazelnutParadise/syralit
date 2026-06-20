package syralit

// Dev control messages shared between the supervisor and a hot reload child.
//
// On the supervisor<->child WebSocket, regular Syralit frames (widget_change,
// ui_patch) flow as usual; these extra messages carry session state across a
// child restart so reloads preserve state like Streamlit (SPEC §10.3).
//
//	supervisor -> child : devMsgDump    (please serialize your session state)
//	child -> supervisor : devMsgState   (here is the state, JSON)
//	supervisor -> child : devMsgRestore (seed this state into the fresh child)
//
// And on the supervisor<->browser socket, the supervisor injects dev status so
// the front end can show a reloading indicator and a compile-error overlay:
//
//	supervisor -> browser : devMsgStatus     {state: "building"|"ready"}
//	supervisor -> browser : devMsgBuildError {error: "..."}
//	supervisor -> browser : devMsgAssetReload {asset: "css"|"js"}
const (
	devMsgDump        = "__dev_dump"
	devMsgState       = "__dev_state"
	devMsgRestore     = "__dev_restore"
	devMsgStatus      = "__dev_status"
	devMsgBuildError  = "__dev_build_error"
	devMsgAssetReload = "__dev_asset_reload"
)

// sessionState is the JSON-serializable snapshot of a session carried across a
// hot reload. Only JSON-compatible values survive; complex objects (e.g.
// *insyra.DataTable) should live in sy.Cache, not session state. See state.go
// for how typed State[T] values are coerced back after a JSON round trip.
type sessionState struct {
	Widgets     map[string]any `json:"widgets"`
	Store       map[string]any `json:"store"`
	CurrentPage string         `json:"current_page,omitempty"`
}
