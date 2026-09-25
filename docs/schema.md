# Schema structure

This documents the Kodi introspection schema structure the generator depends
on. All counts are measured (reproducible: `python3` reading
`schema/vXX/*.json` along the dimensions used here); schemas are vendored
under `schema/vXX/`.

## Files

4 files per version:

| File | Contents | v12 size | v13 size |
| ---- | -------- | -------- | -------- |
| `types.json` | named type table | 187 entries | 188 entries |
| `methods.json` | method table | 170 entries | 177 entries |
| `notifications.json` | notification table | 39 entries | 39 entries |
| `version.txt` | `JSONRPC_VERSION x.y.z\n` | 12.4.0 | 13.5.0 |

All three `.json` files are **flat one-level maps**: `name → definition`.
The generator walks them in file token order (not Go map random order), so
output is deterministic.

Names are always `Namespace.Name` (v13 has 18 namespaces: Addons,
Application, AudioLibrary, Database, Favourites, Files, GUI, Input,
JSONRPC, PVR, Player, Playlist, Profiles, Settings, System, Textures,
VideoLibrary, XBMC).

## Type nodes

Each entry in `types.json` is a schema node. Measured shape distribution
(v13; v12 differs only in object 42→41, everything else identical):

| Shape | v13 count | Generator handling |
| ----- | --------- | ------------------ |
| `extends` + `properties`, no `type` | 62 | struct embedding the base |
| bare `extends`, no `type`/`properties` | 26 | with `items` → slice; otherwise base-type alias |
| `{"type":"object","properties"}` | 42 | struct |
| `{"type":"string"}` (optionally with `enum`) | 23 | `string`; named enums also get typed consts |
| `{"type":"array","items"}` | 9 | slice |
| `{"type":"integer"}` | 5 | `int` |
| `{"type":"number"}` | 1 | `float64` |
| `{"type":[…]}` union | 20 | single non-null variant plus null → `*T`; otherwise `any` |

Field-level schemas (each entry of `properties`) share common keys:
`required` (bool, defaults to false), `default`, `description`, `enum`,
`minimum/maximum`, `minLength`, etc.; the type position may be a `$ref`, a
string, or a union array.

## Methods

Entries in `methods.json` use an almost uniform key set (v13: all but
`Settings.GetSkinSettings`, which has no `params` key, carry the full set —
176/177):

```json
{"description":…, "params":[…], "permission":…, "returns":…,
 "transport":…, "type":"method"}
```

- `params` is an array whose elements are `{name, …schema}` (the schema is a
  type node), with the `$ref` form being the most common.
- `transport` (v13): `Response` 173, `Announcing` 2,
  `Files.PrepareDownload`/`Files.Download` use the
  `["Response","FileDownloadRedirect/Direct"]` list.
- `permission` (v13, 14 kinds): ReadData 87, ControlPlayback 26,
  UpdateData 17, Navigate 16, RemoveData 7, ControlGUI 5, ControlPVR 5,
  ControlPower 5, WriteFile 2, WriteSetting 3, ControlNotify/
  ManageAddon/ExecuteAddon/ControlSystem 1 each.
- `returns` shapes (v13): bare string `string` 80, `boolean` 3,
  `integer`/`object`/`any` 1 each; `$ref` 13 entries; inline
  `object+properties` 70, `array+items` 3, plus 5 entries decorated with
  `additionalProperties/description/required`.
- **Exception**: `Settings.GetSkinSettings` has no `params` key (the
  generator treats it as parameterless).

## Notifications

Entries in `notifications.json` use a fully uniform key set (39 entries each
in v12/v13, no exceptions):

```json
{"description":…, "params":[…], "returns":null, "type":"notification"}
```

Note: notifications have **no `transport` key**, and `returns` is always
`null`. `params` is usually `sender` (string) + `data` (`$ref` or an inline
object).

## Quirks the generator depends on

These are counter-intuitive but measured behaviors the generator must handle
— check them first when changing the schema source:

1. **Two reference spellings**: `{"$ref":"Foo"}`, plus `"type":"Video.Ratings"`
   which writes the type name directly into `type`. There is also
   `"type":"Integer"` with a capital initial, handled as `integer`.
2. **14 `$ref`s do not exist in `types.json`** (identical in v12/v13):
   `Addon.Types`, `GUI.Window`, `Input.Action`,
   `List.Filter.Fields.{Albums,Artists,Episodes,Movies,MusicVideos,Songs,TVShows,Textures}`,
   `List.Filter.Operators`, `Notifications.Library.{Audio,Video}.Type`.
   The generator emits `string` stubs for them and prints the list on every
   run.
3. **Single-variant nullable union**: `Optional.*` is `["null", T]` with only
   one non-null variant → a named non-pointer type (e.g.
   `type OptionalBoolean bool`), taking a pointer at the use site. Do not
   confuse this with "multi-variant union → `any`".
4. **Inline (anonymous) enums collapse to bare `string`**: only named types
   get typed consts (e.g. `PlayerGetActivePlayersResultItem.Playertype string`).
5. **Inline objects are always named**: anonymous objects in params/results/data
   get stable names from the `method+field` path (e.g.
   `PlayerOpenParamsOptions`) instead of anonymous structs.
6. **Field pointer rules**: optional scalar/string-enum/struct → `*T` +
   `omitempty`; slices/maps/`any` stay value types + `omitempty`;
   `required` fields have no `omitempty`.

## v12 → v13 delta (purely additive, no removals)

- Types +1: `Player.Tempo`
- Methods +7: `GUI.ActivateScreenSaver`, `Player.GetAudioDelay/SetAudioDelay/SetTempo`,
  `Settings.GetSkinSettingValue/GetSkinSettings/SetSkinSettingValue`
- Notifications +0

## Kodi release ↔ API version mapping

Measured `version.txt` per release tag in the upstream repo (first→last):

| Kodi | Codename | API | schema version range |
| ---- | -------- | --- | -------------------- |
| 14 | Helix | v6 | 6.21.2 (frozen across 14.0→14.2) |
| 15 | Isengard | v6 | 6.25.2 (frozen across 15.0→15.2) |
| 16 | Jarvis | v6 | 6.32.4 (16.0) → 6.32.5 (16.1, patch only) |
| 17 | Krypton | v8 | 8.0.0 (frozen across 17.0→17.6) |
| 18 | Leia | v10 | 10.1.1 (18.0) → 10.3.0 (18.9) |
| 19 | Matrix | v12 | 12.2.1 (19.0) → 12.4.0 (19.5) |
| 20 | Nexus | v13 | 13.0.0 (frozen across 20.0→20.5) |
| 21 | Omega | v13 | 13.5.0 (frozen across 21.0→21.2) |
| 22 | Piers (master) | v13 | 13.202.0 |
| 11/12/13 | Eden/Frodo/Gotham | unconfirmed | old releases have no `schema/` directory layout |

Notes:

- API major and Kodi release are **not 1:1**: v6 spans three generations
  (Helix/Isengard/Jarvis), v13 spans three (Nexus/Omega/Piers); minor
  versions increase within one major.
- **v7/v9/v11 were never formally released**: they appear only in
  alpha/beta builds (v9.x during the Leia preview, v11.x during the Matrix
  preview, up to 11.20.0 in `19.0b2`, already 12.0.0 by RC1). Target formal
  releases when covering old devices (e.g. v10 for Leia).
- Vendored in this repo: v12 takes the Matrix branch tip (12.4.0);
  v13 takes the Omega branch (13.5.0, stable).
