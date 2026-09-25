# Schema 结构

本文记录生成器依赖的 Kodi introspection schema 结构。所有数字均为
实测（可重复：`python3` 读 `schema/vXX/*.json` 按本文维度统计），
schema vendored 在 `schema/vXX/`。

## 文件

每个版本 4 个文件：

| 文件 | 内容 | v12 规模 | v13 规模 |
| ---- | ---- | -------- | -------- |
| `types.json` | 命名类型表 | 187 项 | 188 项 |
| `methods.json` | 方法表 | 170 项 | 177 项 |
| `notifications.json` | 通知表 | 39 项 | 39 项 |
| `version.txt` | `JSONRPC_VERSION x.y.z\n` | 12.4.0 | 13.5.0 |

三个 `.json` 都是**扁平一层 map**：`名字 → 定义`，生成器按文件 token
顺序遍历（不是 Go map 随机顺序），保证输出确定。

名字一律 `Namespace.Name` 形式（v13 共 18 个命名空间：Addons,
Application, AudioLibrary, Database, Favourites, Files, GUI, Input,
JSONRPC, PVR, Player, Playlist, Profiles, Settings, System, Textures,
VideoLibrary, XBMC）。

## 类型节点

`types.json` 的每项是一个 schema 节点，实测形状分布（v13；v12 仅
object 42→41，其余相同）：

| 形状 | v13 数量 | 生成器处理 |
| ---- | -------- | ---------- |
| `extends` + `properties`，无 `type` | 62 | 嵌入基类的 struct |
| 仅 `extends`，无 `type`/`properties` | 26 | 带 `items` → slice；否则基类别名 |
| `{"type":"object","properties"}` | 42 | struct |
| `{"type":"string"}`（可带 `enum`） | 23 | `string`；具名 enum 加 typed consts |
| `{"type":"array","items"}` | 9 | slice |
| `{"type":"integer"}` | 5 | `int` |
| `{"type":"number"}` | 1 | `float64` |
| `{"type":[…]}` union | 20 | 单 non-null 变体且含 null → `*T`；否则 `any` |

字段级 schema（`properties` 的每项）通用键：`required`（bool，缺省
false）、`default`、`description`、`enum`、`minimum/maximum`、
`minLength` 等；类型位置可以是 `$ref`、字符串或 union 数组。

## 方法

`methods.json` 每项键集几乎统一（v13：除 `Settings.GetSkinSettings`
无 `params` 键外，其余 176/177 全键）：

```json
{"description":…, "params":[…], "permission":…, "returns":…,
 "transport":…, "type":"method"}
```

- `params` 是数组，元素为 `{name, …schema}`（schema 即类型节点），
  其中 `$ref` 形式最常见。
- `transport`（v13）：`Response` 173、`Announcing` 2、
  `Files.PrepareDownload`/`Files.Download` 为
  `["Response","FileDownloadRedirect/Direct"]` 列表。
- `permission`（v13 14 种）：ReadData 87、ControlPlayback 26、
  UpdateData 17、Navigate 16、RemoveData 7、ControlGUI 5、ControlPVR 5、
  ControlPower 5、WriteFile 2、WriteSetting 3、ControlNotify/
  ManageAddon/ExecuteAddon/ControlSystem 各 1。
- `returns` 形状（v13）：裸字符串 `string` 80、`boolean` 3、
  `integer`/`object`/`any` 各 1；`$ref` 13 项；内联
  `object+properties` 70、`array+items` 3、其余带
  `additionalProperties/description/required` 修饰的 5 项。
- **例外**：`Settings.GetSkinSettings` 无 `params` 键（生成器按无参处理）。

## 通知

`notifications.json` 每项键集完全统一（v12/v13 各 39 项，无例外）：

```json
{"description":…, "params":[…], "returns":null, "type":"notification"}
```

注意：通知**没有 `transport` 键**，`returns` 恒为 `null`。`params`
通常是 `sender`（string）+ `data`（`$ref` 或内联 object）。

## 生成器依赖的 quirks

以下是不符合直觉、但生成器必须处理的实测行为，改 schema 源时重点看：

1. **引用有两种写法**：`{"$ref":"Foo"}`，以及 `"type":"Video.Ratings"`
   这种把类型名直接写在 `type` 里的。另有 `"type":"Integer"` 首字母大写，
   按 `integer` 处理。
2. **14 个 `$ref` 在 `types.json` 里不存在**（v12/v13 相同）：
   `Addon.Types`、`GUI.Window`、`Input.Action`、
   `List.Filter.Fields.{Albums,Artists,Episodes,Movies,MusicVideos,Songs,TVShows,Textures}`、
   `List.Filter.Operators`、`Notifications.Library.{Audio,Video}.Type`。
   生成器输出 `string` 桩 + 每次打印清单。
3. **单变体 nullable union**：`Optional.*` 是 `["null", T]` 且仅一个
   non-null 变体 → 具名非指针类型（如 `type OptionalBoolean bool`），
   用处再取指针。不要和"多变体 union → `any`"搞混。
4. **内联（匿名）枚举坍缩为裸 `string`**：只有具名类型才有 typed
   consts（如 `PlayerGetActivePlayersResultItem.Playertype string`）。
5. **内联对象一律具名化**：params/results/data 里的匿名 object 按
   `方法+字段` 路径生成稳定名（如 `PlayerOpenParamsOptions`），不直接
   用匿名 struct。
6. **字段指针规则**：optional 的标量/string-enum/结构体 → `*T` +
   `omitempty`；slice/map/`any` 保持值类型 + `omitempty`；
   `required` 无 `omitempty`。

## v12 → v13 增量（纯加法，无删除）

- 类型 +1：`Player.Tempo`
- 方法 +7：`GUI.ActivateScreenSaver`、`Player.GetAudioDelay/SetAudioDelay/SetTempo`、
  `Settings.GetSkinSettingValue/GetSkinSettings/SetSkinSettingValue`
- 通知 +0

## Kodi 大版本 ↔ API 版本对照

官方仓库各 release tag 的 `version.txt` 实测（首版→末版）：

| Kodi | 代号 | API | schema 版本区间 |
| ---- | ---- | --- | --------------- |
| 14 | Helix | v6 | 6.21.2（14.0→14.2 全程冻结） |
| 15 | Isengard | v6 | 6.25.2（15.0→15.2 全程冻结） |
| 16 | Jarvis | v6 | 6.32.4（16.0）→ 6.32.5（16.1，仅 patch） |
| 17 | Krypton | v8 | 8.0.0（17.0→17.6 全程冻结） |
| 18 | Leia | v10 | 10.1.1（18.0）→ 10.3.0（18.9） |
| 19 | Matrix | v12 | 12.2.1（19.0）→ 12.4.0（19.5） |
| 20 | Nexus | v13 | 13.0.0（20.0→20.5 全程冻结） |
| 21 | Omega | v13 | 13.5.0（21.0→21.2 全程冻结） |
| 22 | Piers（master） | v13 | 13.202.0 |
| 11/12/13 | Eden/Frodo/Gotham | 未确认 | 老版本尚无 `schema/` 目录结构 |

说明：

- API major 与 Kodi 大版本**不是 1:1**：v6 横跨 Helix/Isengard/Jarvis
  三代，v13 横跨 Nexus/Omega/Piers 三代；同 major 内 minor 递增。
- **v7/v9/v11 从未正式发布**：只出现在 alpha/beta（Leia 预览期 v9.x，
  Matrix 预览期 v11.x，最高 `19.0b2` 的 11.20.0，到 RC1 已是 12.0.0）。
  需要覆盖老设备时以正式版为准（如 Leia 取 v10）。
- 本仓库 vendored：v12 取 Matrix 分支 tip（12.4.0）；
  v13 取 Omega 分支（13.5.0，稳定版）。
