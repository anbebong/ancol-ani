package main

// Adapter wasm: an optional, swappable source of the widget's text, imported
// by the user as one .wasm file per customer/source instead of needing a
// rebuilt client. The contract with it is deliberately just two things:
//   - the app gives it config (a JSON object, whose shape the adapter itself
//     declares - the app never hardcodes what fields exist)
//   - the adapter gives back text to show on the widget (one line = one
//     rotating message, same convention /text already uses)
// Everything else - what it connects to, how, how often, how it parses a
// response - is the adapter's own business logic, opaque to this app.
//
// wazero runs the .wasm fully sandboxed: no filesystem, no raw sockets - the
// ONLY thing from outside the sandbox it can touch is the one host import
// below (http_fetch), because business logic ("gọi mạng/parse ra sao" - the
// user's own words) is meant to live entirely inside the adapter, not be
// something this app decides on the adapter's behalf.
//
// Required exports (UTF-8 strings in/out of guest linear memory, see
// readGuestString/writeGuestBytes for the ptr/len convention):
//   - alloc(size uint32) -> ptr uint32
//       Lets the host (this app) safely place bytes inside the guest's own
//       memory before calling configSchema/poll with them, without
//       colliding with whatever the guest's own code/data is using.
//   - config_schema() -> packed(ptr, len)
//       A JSON array describing the fields this adapter needs from the
//       user, e.g. [{"key":"api_key","label":"API Key"}, ...]. The
//       Settings UI renders a form from this - the app never assumes what
//       an adapter needs.
//   - configure(ptr uint32, len uint32)
//       The app calls this once with the JSON object of values the user
//       filled in (keyed the same as config_schema), whenever the config is
//       saved/changed. The adapter is expected to keep whatever it needs
//       from this internally for subsequent poll() calls.
//   - poll() -> packed(ptr, len)
//       Called on a timer; returns the text to show right now (empty =
//       "nothing new, leave the widget as-is"). May return a reserved
//       "_interval_sec" instruction via config_schema's defaults instead of
//       a separate function - see pollInterval below.
//   - info() -> packed(ptr, len)
//       A JSON object the adapter describes itself with - "name" (tên
//       module), "description" (chức năng), "author" (thông tin người
//       tạo), "version" are REQUIRED (non-empty), any other key is
//       optional. Every field (required or not) must be <= 255 characters;
//       importWasmAdapter rejects the file otherwise. See
//       docs/WASM-ADAPTER.md for the exact contract.
//
// Host import the adapter may call (its only way to reach the network -
// wasm has no sockets of its own):
//   - env.http_fetch(urlPtr uint32, urlLen uint32) -> packed(respPtr, respLen)
//       A plain GET to the URL the adapter itself built (e.g. baking in an
//       api_key from configure()). This app does the actual HTTP call and
//       writes the raw response body into the adapter's own memory (via
//       the adapter's alloc) - it never looks at the URL or the response,
//       both are the adapter's business. A failed request returns an empty
//       result (ptr=0, len=0), never an error code - the adapter decides
//       what an empty response means for it.
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// adapterHTTPClient is deliberately shared/plain - the adapter only ever
// supplies a URL string, the actual request is ordinary Go net/http, no
// different from any other outbound call this app already makes (GitHub
// releases, the sync server...).
var adapterHTTPClient = &http.Client{Timeout: 10 * time.Second}

// hostHTTPFetch backs the "env.http_fetch" import every adapter module may
// call. `mod` here is the CALLING (guest) module, given to us by wazero's
// GoModuleFunc convention - that's how this reaches the adapter's own
// alloc/memory without needing a reference to it threaded through
// elsewhere. stack[0]/stack[1] are the two i32 params (url ptr/len) on the
// way in; stack[0] is overwritten with the packed i64 result on the way
// out, per wazero's calling convention for WithGoModuleFunction.
func hostHTTPFetch(ctx context.Context, mod api.Module, stack []uint64) {
	urlPtr, urlLen := uint32(stack[0]), uint32(stack[1])
	stack[0] = 0 // ptr=0,len=0 (packed as 0) on any failure path below

	urlBytes, ok := mod.Memory().Read(urlPtr, urlLen)
	if !ok {
		return
	}
	resp, err := adapterHTTPClient.Get(string(urlBytes))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return
	}

	allocFn := mod.ExportedFunction("alloc")
	if allocFn == nil {
		return
	}
	res, err := allocFn.Call(ctx, uint64(len(body)))
	if err != nil {
		return
	}
	respPtr := uint32(res[0])
	if !mod.Memory().Write(respPtr, body) {
		return
	}
	stack[0] = uint64(respPtr)<<32 | uint64(uint32(len(body)))
}

// Several adapters can be imported at once (one per customer/source), each
// kept in its own subdirectory keyed by a stable id - but only ONE is ever
// running (widgetSettings.ActiveAdapterID + AdapterEnabled). Importing a
// second one never touches the first's files or config.
func adaptersDir() string {
	dir := filepath.Join(appDataDir(), "adapters")
	os.MkdirAll(dir, 0o755)
	return dir
}

func adapterDir(id string) string        { return filepath.Join(adaptersDir(), id) }
func adapterWasmPath(id string) string   { return filepath.Join(adapterDir(id), "adapter.wasm") }
func adapterConfigPath(id string) string { return filepath.Join(adapterDir(id), "config.json") }
func adapterInfoPath(id string) string   { return filepath.Join(adapterDir(id), "info.json") }

var adapterIDRe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// newAdapterID turns an uploaded filename into a short, filesystem-safe,
// unique directory name - human-readable (so a GET /text-adapter/list
// response is self-explanatory) rather than a random UUID.
func newAdapterID(filename string) string {
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	base = adapterIDRe.ReplaceAllString(base, "_")
	if base == "" {
		base = "adapter"
	}
	id := base
	for i := 2; ; i++ {
		if _, err := os.Stat(adapterDir(id)); os.IsNotExist(err) {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, i)
	}
}

// adapterListEntry is what the Settings page's "which source" picker is
// built from - the 4 required info() fields (see validateAdapterInfo)
// flattened out, plus whichever one is currently active.
type adapterListEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Active      bool   `json:"active"`
}

// readAdapterInfo loads one adapter's saved, already-validated info.json
// (written once at import time - see importWasmAdapter) - falls back to
// the raw id as name if the file is somehow missing/unreadable rather
// than failing the whole list.
func readAdapterInfo(id string) map[string]any {
	raw, err := os.ReadFile(adapterInfoPath(id))
	if err != nil {
		return map[string]any{"name": id}
	}
	var info map[string]any
	if json.Unmarshal(raw, &info) != nil {
		return map[string]any{"name": id}
	}
	return info
}

func infoString(info map[string]any, key string) string {
	s, _ := info[key].(string)
	return s
}

// listAdapters enumerates every imported adapter's own directory.
func listAdapters() ([]adapterListEntry, error) {
	entries, err := os.ReadDir(adaptersDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	activeID := getSettings().ActiveAdapterID
	var out []adapterListEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		info := readAdapterInfo(id)
		out = append(out, adapterListEntry{
			ID:          id,
			Name:        infoString(info, "name"),
			Description: infoString(info, "description"),
			Author:      infoString(info, "author"),
			Version:     infoString(info, "version"),
			Active:      id == activeID,
		})
	}
	return out, nil
}

const (
	maxWasmAdapterBytes = 10 << 20 // 10MB - generous for a small compiled module
	defaultPollInterval = 10 * time.Second
	minPollInterval     = 2 * time.Second
	maxPollInterval     = 10 * time.Minute
)

// wasmAdapter wraps one instantiated module and its four required exports.
// Used from a single goroutine (the poll loop) at a time, so no locking
// needed inside it.
type wasmAdapter struct {
	runtime wazero.Runtime
	mod     api.Module

	alloc        api.Function
	configSchema api.Function
	configureFn  api.Function
	pollFn       api.Function
	infoFn       api.Function
}

// adapterInfoMaxFieldLen matches the ABI doc: no field in info() - required
// or optional - may be longer than this many characters.
const adapterInfoMaxFieldLen = 255

// adapterInfoRequiredKeys are validated present, non-empty and within
// adapterInfoMaxFieldLen on every import - see validateAdapterInfo.
var adapterInfoRequiredKeys = []string{"name", "description", "author", "version"}

func packed(v uint64) (ptr, length uint32) {
	return uint32(v >> 32), uint32(v)
}

func (a *wasmAdapter) readString(ctx context.Context, v uint64) (string, error) {
	ptr, length := packed(v)
	if length == 0 {
		return "", nil
	}
	buf, ok := a.mod.Memory().Read(ptr, length)
	if !ok {
		return "", fmt.Errorf("module lỗi khi đọc dữ liệu")
	}
	out := make([]byte, length)
	copy(out, buf)
	return string(out), nil
}

// writeString allocates room inside the guest for data via its own alloc
// export, then copies data in - the only safe way to hand the guest bytes
// without knowing its internal memory layout.
func (a *wasmAdapter) writeString(ctx context.Context, data string) (ptr, length uint32, err error) {
	if data == "" {
		return 0, 0, nil
	}
	res, err := a.alloc.Call(ctx, uint64(len(data)))
	if err != nil {
		return 0, 0, err
	}
	ptr = uint32(res[0])
	if !a.mod.Memory().Write(ptr, []byte(data)) {
		return 0, 0, fmt.Errorf("module lỗi khi ghi dữ liệu")
	}
	return ptr, uint32(len(data)), nil
}

// loadWasmAdapter instantiates wasmBytes and validates every required
// export exists, failing fast with a clear message instead of panicking
// partway through the first real call.
func loadWasmAdapter(ctx context.Context, wasmBytes []byte) (*wasmAdapter, error) {
	rt := wazero.NewRuntime(ctx)

	_, err := rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(hostHTTPFetch), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("http_fetch").
		Instantiate(ctx)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("không khởi tạo được module: %w", err)
	}

	mod, err := rt.Instantiate(ctx, wasmBytes)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("không nạp được module: %w", err)
	}

	a := &wasmAdapter{
		runtime:      rt,
		mod:          mod,
		alloc:        mod.ExportedFunction("alloc"),
		configSchema: mod.ExportedFunction("config_schema"),
		configureFn:  mod.ExportedFunction("configure"),
		pollFn:       mod.ExportedFunction("poll"),
		infoFn:       mod.ExportedFunction("info"),
	}
	var missing []string
	if a.alloc == nil {
		missing = append(missing, "alloc")
	}
	if a.configSchema == nil {
		missing = append(missing, "config_schema")
	}
	if a.configureFn == nil {
		missing = append(missing, "configure")
	}
	if a.pollFn == nil {
		missing = append(missing, "poll")
	}
	if a.infoFn == nil {
		missing = append(missing, "info")
	}
	if len(missing) > 0 {
		mod.Close(ctx)
		rt.Close(ctx)
		return nil, fmt.Errorf("module thiếu chức năng bắt buộc: %v", missing)
	}
	return a, nil
}

func (a *wasmAdapter) Close(ctx context.Context) {
	a.runtime.Close(ctx)
}

func (a *wasmAdapter) ConfigSchema(ctx context.Context) (string, error) {
	res, err := a.configSchema.Call(ctx)
	if err != nil {
		return "", err
	}
	return a.readString(ctx, res[0])
}

// Name returns "" (never an error) if the adapter has no "name" export -
// the caller falls back to the uploaded filename in that case.
// Info returns the raw JSON string from the adapter's info() export, not
// yet parsed/validated - see validateAdapterInfo for that.
func (a *wasmAdapter) Info(ctx context.Context) (string, error) {
	res, err := a.infoFn.Call(ctx)
	if err != nil {
		return "", err
	}
	return a.readString(ctx, res[0])
}

// validateAdapterInfo parses raw JSON from info(), checks every required
// key (adapterInfoRequiredKeys) is present/non-empty, and that EVERY
// string field - required or not - is at most adapterInfoMaxFieldLen
// characters. Returns the parsed map so callers get it back without
// re-parsing.
func validateAdapterInfo(raw string) (map[string]any, error) {
	var info map[string]any
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		return nil, fmt.Errorf("thông tin module không hợp lệ: %w", err)
	}
	for _, key := range adapterInfoRequiredKeys {
		v, ok := info[key].(string)
		if !ok || strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("module thiếu thông tin bắt buộc %q", key)
		}
	}
	for key, v := range info {
		s, ok := v.(string)
		if !ok {
			continue
		}
		if len([]rune(s)) > adapterInfoMaxFieldLen {
			return nil, fmt.Errorf("thông tin module ở trường %q dài quá %d ký tự", key, adapterInfoMaxFieldLen)
		}
	}
	return info, nil
}

func (a *wasmAdapter) Configure(ctx context.Context, configJSON string) error {
	ptr, length, err := a.writeString(ctx, configJSON)
	if err != nil {
		return err
	}
	_, err = a.configureFn.Call(ctx, uint64(ptr), uint64(length))
	return err
}

func (a *wasmAdapter) Poll(ctx context.Context) (string, error) {
	res, err := a.pollFn.Call(ctx)
	if err != nil {
		return "", err
	}
	return a.readString(ctx, res[0])
}

// --- lifecycle: one adapter + poll loop running at a time, swappable from
// the HTTP handlers below without restarting the app. ---

var wasmLoopCancel atomic.Value // func()
var wasmLoopMu sync.Mutex

// stopWasmAdapter cancels whatever poll loop is currently running (if any)
// and waits for nothing - the loop closes its own adapter/runtime on exit.
func stopWasmAdapter() {
	wasmLoopMu.Lock()
	defer wasmLoopMu.Unlock()
	if cancel, ok := wasmLoopCancel.Load().(func()); ok && cancel != nil {
		cancel()
	}
	wasmLoopCancel.Store((func())(nil))
	latestAdapterText.Store("") // leaving adapter mode: stop showing its last output
}

// currentText is what GET /text should actually return: the adapter's
// latest output while it's switched on and has produced something, or
// the manually typed/saved text otherwise. Manual text (getText) is never
// overwritten by the adapter, so this is always safe to fall back to.
func currentText() string {
	if getSettings().AdapterEnabled {
		if v, _ := latestAdapterText.Load().(string); v != "" {
			return v
		}
	}
	return getText()
}

// lastAdapterError records why the active adapter most recently fell back
// to manual mode on its own (see runWasmAdapterLoop) - surfaced by GET
// /text-adapter so the Settings page can tell the user why, instead of
// AdapterEnabled just silently flipping back to false with no explanation.
var lastAdapterError atomic.Value // string

// disableAdapterOnError is what an adapter failing mid-poll falls back
// to: switch AdapterEnabled off (so currentText()/widget go back to the
// manual text immediately), remember why, and push that manual text live
// - all without needing the user to notice and flip the switch themselves.
// disableAdapterOnError's reason is shown as-is in the Settings UI - never
// pass a raw Go/wazero error straight through (its message is a wasm
// stack trace, not something a user should ever see); log that for
// debugging and store a plain-language reason instead.
func disableAdapterOnError(appCtx context.Context, rawErr error) {
	debugLogEmit("adapter error (fell back to manual): " + rawErr.Error())
	lastAdapterError.Store("Module gặp lỗi khi lấy dữ liệu")
	saveSettings(func(s *widgetSettings) { s.AdapterEnabled = false })
	stopWasmAdapter()
	emitServerText(appCtx, getText())
}

func lastAdapterErrorString() string {
	v, _ := lastAdapterError.Load().(string)
	return v
}

// pollInterval reads an optional "_interval_sec" number out of the saved
// config JSON (a reserved key an adapter's config_schema may document),
// falling back to defaultPollInterval - no separate wasm export needed just
// for this.
func pollInterval(configJSON string) time.Duration {
	var raw map[string]any
	if json.Unmarshal([]byte(configJSON), &raw) != nil {
		return defaultPollInterval
	}
	n, ok := raw["_interval_sec"].(float64)
	if !ok {
		return defaultPollInterval
	}
	d := time.Duration(n) * time.Second
	if d < minPollInterval {
		return minPollInterval
	}
	if d > maxPollInterval {
		return maxPollInterval
	}
	return d
}

// startWasmAdapter (re)starts the poll loop from whichever adapter is
// currently ActiveAdapterID + its own saved config. Called at app startup
// and again every time the user selects a different adapter, re-enables
// one, or edits its config - each call fully replaces the previous loop.
// Returns nil (no-op) if nothing is selected or AdapterEnabled is false.
func startWasmAdapter(appCtx context.Context) error {
	stopWasmAdapter()

	settings := getSettings()
	if !settings.AdapterEnabled || settings.ActiveAdapterID == "" {
		return nil
	}
	id := settings.ActiveAdapterID

	wasmBytes, err := os.ReadFile(adapterWasmPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	configJSON := "{}"
	if raw, err := os.ReadFile(adapterConfigPath(id)); err == nil {
		configJSON = string(raw)
	}

	adapter, err := loadWasmAdapter(appCtx, wasmBytes)
	if err != nil {
		return err
	}
	if err := adapter.Configure(appCtx, configJSON); err != nil {
		adapter.Close(appCtx)
		return fmt.Errorf("module từ chối cấu hình: %w", err)
	}

	ctx, cancelFn := context.WithCancel(appCtx)
	// context.CancelFunc is a distinct named type from plain func() -
	// atomic.Value panics if the concrete type stored ever changes, so this
	// must be converted to match stopWasmAdapter's func() nil above.
	var cancel func() = cancelFn
	wasmLoopMu.Lock()
	wasmLoopCancel.Store(cancel)
	wasmLoopMu.Unlock()
	lastAdapterError.Store("")

	interval := pollInterval(configJSON)
	go runWasmAdapterLoop(ctx, appCtx, adapter, interval)
	return nil
}

// latestAdapterText holds the most recent non-empty poll() result, kept
// ONLY in memory - unlike manually typed text (saveSettings' Text field,
// persisted to settings.data), an adapter's output is transient: it's
// whatever the adapter says right now, not something to remember across
// restarts or fall back to once the adapter is switched off again. GET
// /text reads this instead of settings.Text while AdapterEnabled is true
// (see the /text handler in app.go).
var latestAdapterText atomic.Value // string

// runWasmAdapterLoop feeds every non-empty poll() result straight to the
// widget - normalizeText + latestAdapterText + EventsEmit, deliberately
// skipping saveSettings: the manually typed text underneath is left
// completely untouched, so switching back to "Thủ công" shows exactly
// what was there before, not whatever the adapter last said. A real error
// from poll() (the wasm call itself failing/trapping - NOT an empty
// string, which just means "nothing new this tick") falls back to manual
// mode on its own instead of leaving the widget stuck on a dead source.
func runWasmAdapterLoop(ctx, appCtx context.Context, adapter *wasmAdapter, interval time.Duration) {
	defer adapter.Close(appCtx)

	// true once this tick already triggered the fallback - stops the
	// select loop below from trying again after stopWasmAdapter cancelled
	// our own ctx.
	failed := false
	tick := func() {
		text, err := adapter.Poll(ctx)
		if err != nil {
			disableAdapterOnError(appCtx, err)
			failed = true
			return
		}
		if text == "" {
			return
		}
		normalized := normalizeText(text)
		if normalized == "" {
			return
		}
		latestAdapterText.Store(normalized)
		emitServerText(appCtx, normalized)
	}

	tick()
	if failed {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick()
			if failed {
				return
			}
		}
	}
}

// importWasmAdapter validates wasmBytes (instantiate + check exports, then
// immediately close - it never runs configure/poll here) and adds it as a
// NEW entry under its own id - existing adapters (files, config, which one
// is active) are never touched by importing another one. It does not
// select or enable the new adapter on its own; the user picks and switches
// to it separately (see selectAdapter).
func importWasmAdapter(appCtx context.Context, filename string, wasmBytes []byte) (id, schemaJSON string, err error) {
	adapter, err := loadWasmAdapter(appCtx, wasmBytes)
	if err != nil {
		return "", "", err
	}
	schema, schemaErr := adapter.ConfigSchema(appCtx)
	rawInfo, infoErr := adapter.Info(appCtx)
	adapter.Close(appCtx)
	if schemaErr != nil {
		return "", "", fmt.Errorf("không đọc được cấu hình module: %w", schemaErr)
	}
	if infoErr != nil {
		return "", "", fmt.Errorf("không đọc được thông tin module: %w", infoErr)
	}
	info, err := validateAdapterInfo(rawInfo)
	if err != nil {
		return "", "", err
	}

	id = newAdapterID(filename)
	if err := os.MkdirAll(adapterDir(id), 0o755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(adapterWasmPath(id), wasmBytes, 0o644); err != nil {
		return "", "", err
	}
	os.WriteFile(adapterConfigPath(id), []byte("{}"), 0o644)
	infoJSON, _ := json.Marshal(info)
	os.WriteFile(adapterInfoPath(id), infoJSON, 0o644)

	return id, schema, nil
}

// selectAdapter makes id the one actually running: switches
// ActiveAdapterID to it, enables adapter mode, and (re)starts the poll
// loop - replacing whatever adapter/mode was active before.
func selectAdapter(appCtx context.Context, id string) error {
	if _, err := os.Stat(adapterWasmPath(id)); err != nil {
		return fmt.Errorf("không tìm thấy module %q", id)
	}
	if err := saveSettings(func(s *widgetSettings) {
		s.ActiveAdapterID = id
		s.AdapterEnabled = true
	}); err != nil {
		return err
	}
	return startWasmAdapter(appCtx)
}

// saveWasmAdapterConfig persists the user-filled config for one adapter and,
// if it's the currently active+enabled one, hot-restarts the poll loop with
// it - no app restart needed. Saving a non-active adapter's config just
// updates its file for whenever it's selected later.
func saveWasmAdapterConfig(appCtx context.Context, id, configJSON string) error {
	if err := os.WriteFile(adapterConfigPath(id), []byte(configJSON), 0o644); err != nil {
		return err
	}
	settings := getSettings()
	if settings.AdapterEnabled && settings.ActiveAdapterID == id {
		return startWasmAdapter(appCtx)
	}
	return nil
}

// removeAdapterByID deletes one adapter's whole directory. If it was the
// active one, adapter mode is switched off first (falls back to manual,
// same as an error would) rather than leaving ActiveAdapterID pointing at
// a now-missing directory.
func removeAdapterByID(id string) error {
	if getSettings().ActiveAdapterID == id {
		saveSettings(func(s *widgetSettings) { s.AdapterEnabled = false })
		stopWasmAdapter()
	}
	return os.RemoveAll(adapterDir(id))
}

// currentWasmAdapterSchema re-reads config_schema from one adapter's own
// .wasm, for when the settings page reloads and needs to redraw its config
// form without the user re-importing the file - the schema itself is never
// persisted separately, just re-asked from the module each time it's
// needed.
func currentWasmAdapterSchema(appCtx context.Context, id string) (string, error) {
	wasmBytes, err := os.ReadFile(adapterWasmPath(id))
	if err != nil {
		return "", err
	}
	adapter, err := loadWasmAdapter(appCtx, wasmBytes)
	if err != nil {
		return "", err
	}
	defer adapter.Close(appCtx)
	return adapter.ConfigSchema(appCtx)
}

// emitServerText notifies the widget's live view, same as the manual
// /text POST handler does. A package-level var (not a direct call) so
// tests can swap in a no-op: Wails' real runtime.EventsEmit calls
// log.Fatal (killing the whole process, not just failing the call) when
// given a context that isn't the one from the app's lifecycle hooks -
// something a test harness, or any future caller that's not 100% sure its
// ctx came from a.startup, cannot safely risk.
var emitServerText = func(ctx context.Context, text string) {
	debugLogEmit(text)
	runtime.EventsEmit(ctx, "server-text", text)
}

// debugLogEmit: TEMPORARY - appends every emitServerText call to a plain
// file so it can be inspected directly instead of relying on someone
// watching the widget at the exact right second. Remove once the adapter
// feature is confirmed working end-to-end.
func debugLogEmit(text string) {
	f, err := os.OpenFile(filepath.Join(appDataDir(), "debug-emit.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s emit server-text: %q\n", time.Now().Format(time.RFC3339), text)
}
