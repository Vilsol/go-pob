package mcp

func tools() []toolDef {
	return []toolDef{
		{"new_build", "Create an empty PoE1 character. Returns a handle (b1, b2...) used by every other tool.", schemaNewBuild, (*Server).newBuild},
		{"import_build", "Load a Path of Building build from an export code, a pobb.in / poe.ninja / pastebin / pob.codes / maxroll URL, raw XML or an absolute .xml path.", schemaImportBuild, (*Server).importBuild},
		{"get_build", "Read a build's structure by section (summary, tree, items, skills, config, notes, warnings). Without handle, lists live handles. Numbers are in get_stats.", schemaGetBuild, (*Server).getBuild},
		{"get_stats", "Read calculated stats (PoB output keys, e.g. Life, FireResist, CombinedDPS, MainDPS). explain=<key> lists its top sources.", schemaGetStats, (*Server).getStats},
		{"search_data", "Search game data: gems, supports, uniques, bases, passive nodes, keystones, masteries, affixes, config options, stat keys.", schemaSearchData, (*Server).searchData},
		{"edit_tree", "Edit the passive tree with a batch of ops (allocate with path, deallocate with dependents, set_mastery, set_class, set_ascendancy, reset). Returns a stat delta; dry_run previews.", schemaEditTree, (*Server).editTree},
		{"edit_items", "Edit equipment with a batch of ops (equip_unique, craft, equip_text, unequip, set_flask_active, socket_jewel). Returns a stat delta; dry_run previews.", schemaEditItems, (*Server).editItems},
		{"edit_skills", "Edit socket groups and gems with a batch of ops (add_group, set_gems, add_gem, remove_gem, set_main, enable/disable/remove_group, set_gem_level). Returns a stat delta; dry_run previews.", schemaEditSkills, (*Server).editSkills},
		{"edit_build", "Edit level, bandit, pantheon, configuration options and notes with a batch of ops. Returns a stat delta; dry_run previews.", schemaEditBuild, (*Server).editBuild},
		{"rank_candidates", "Rank what to take by PoB's power calculation: passive nodes, nodes to remove, support gems, or uniques for a slot.", schemaRankCandidates, (*Server).rankCandidates},
		{"compare_builds", "Compare two builds or two revisions (handle or handle@revision): stat differences and what changed.", schemaCompareBuilds, (*Server).compareBuilds},
		{"revert_build", "Move a handle back (or forward) to another revision. Returns the stat delta.", schemaRevertBuild, (*Server).revertBuild},
		{"export_build", "Export a build as a PoB code (or XML). The code can be pasted into Path of Building.", schemaExportBuild, (*Server).exportBuild},
		{"optimize", "Search combinations of listed choices (items per slot, support sets, node sets) for the best objective under constraints. Nothing is applied unless apply:true.", schemaOptimize, (*Server).optimize},
	}
}

const schemaNewBuild = `{"type":"object","required":["class"],"additionalProperties":false,"properties":{
  "class":{"type":"string","description":"Marauder|Ranger|Witch|Duelist|Templar|Shadow|Scion"},
  "ascendancy":{"type":"string","description":"e.g. Elementalist; omit for none"},
  "level":{"type":"integer","minimum":1,"maximum":100,"default":90},
  "name":{"type":"string"},
  "from_handle":{"type":"string","description":"copy an existing handle instead (fork); other fields then override"}}}`

const schemaImportBuild = `{"type":"object","required":["source"],"additionalProperties":false,"properties":{
  "source":{"type":"string","description":"PoB export code, a pobb.in / poe.ninja / pastebin / poedb / pob.codes / maxroll URL, raw XML, or an absolute .xml file path"},
  "convert_tree":{"type":"boolean","default":true},
  "name":{"type":"string"}}}`

const schemaGetBuild = `{"type":"object","additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "section":{"type":"string","enum":["summary","tree","items","skills","config","notes","warnings"],"default":"summary"},
  "detail":{"type":"boolean","default":false}}}`

const schemaGetStats = `{"type":"object","required":["handle"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "stats":{"type":"array","items":{"type":"string"},"description":"PoB output keys; default = summary set"},
  "groups":{"type":"array","items":{"type":"string","enum":["offence","defence","resist","attributes","charges","minion","recovery"]}},
  "explain":{"type":"string","description":"one stat key: return its top contributing sources"},
  "revision":{"type":"integer","description":"read an older revision"}}}`

const schemaSearchData = `{"type":"object","required":["kind"],"additionalProperties":false,"properties":{
  "kind":{"type":"string","enum":["gem","support","unique","base","node","keystone","mastery","affix","config","stat"]},
  "query":{"type":"string","description":"words matched against name and text; empty lists everything (paged)"},
  "handle":{"type":"string","description":"node: adds path_cost from this build; support: adds compatible-with-main-skill flag; unique/base: slot fit; affix: required with base"},
  "slot":{"type":"string","description":"unique/base/affix: restrict to items for this slot"},
  "base":{"type":"string","description":"affix: the item base the affixes are for"},
  "tags":{"type":"array","items":{"type":"string"},"description":"gem tags, e.g. [\"fire\",\"spell\"]"},
  "max_level_req":{"type":"integer"},
  "limit":{"type":"integer","default":10,"maximum":50},
  "cursor":{"type":"string"}}}`

const schemaEditTree = `{"type":"object","required":["handle","ops"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "dry_run":{"type":"boolean","default":false},
  "watch":{"type":"array","items":{"type":"string"}},
  "ops":{"type":"array","minItems":1,"maxItems":20,"items":{"type":"object","required":["op"],"additionalProperties":false,"properties":{
    "op":{"type":"string","enum":["allocate","deallocate","set_mastery","set_class","set_ascendancy","reset"]},
    "node":{"description":"node id (integer) or name"},
    "effect":{"type":"string","description":"set_mastery: effect text or id"},
    "path":{"type":"boolean","default":true,"description":"allocate: also allocate the cheapest path"},
    "with_dependents":{"type":"boolean","default":true,"description":"deallocate: also remove nodes that would be disconnected"},
    "class":{"type":"string"},"ascendancy":{"type":"string"}}}}}}`

const schemaEditItems = `{"type":"object","required":["handle","ops"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},"dry_run":{"type":"boolean","default":false},
  "watch":{"type":"array","items":{"type":"string"}},
  "ops":{"type":"array","minItems":1,"maxItems":10,"items":{"type":"object","required":["op"],"additionalProperties":false,"properties":{
    "op":{"type":"string","enum":["equip_unique","craft","equip_text","unequip","set_flask_active","socket_jewel"]},
    "slot":{"type":"string","description":"Helmet, Body Armour, Gloves, Boots, Amulet, Ring 1, Ring 2, Belt, Weapon 1, Weapon 2, Flask 1-5"},
    "name":{"type":"string","description":"equip_unique: unique name; variant chosen as PoB's default"},
    "base":{"type":"string","description":"craft: base type, e.g. Astral Plate"},
    "affixes":{"type":"array","maxItems":6,"items":{"type":"string"},"description":"craft: affix labels, e.g. \"+# to maximum Life\"; best tier the base's item level allows, max roll"},
    "sockets":{"type":"string","description":"craft: e.g. R-R-G-G-B-B; default = PoB's crafted default"},
    "text":{"type":"string","description":"equip_text: full PoB item text"},
    "active":{"type":"boolean"},
    "node":{"description":"socket_jewel: jewel socket node id or name; slot not used"}}}}}}`

const schemaEditSkills = `{"type":"object","required":["handle","ops"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},"dry_run":{"type":"boolean","default":false},
  "watch":{"type":"array","items":{"type":"string"}},
  "ops":{"type":"array","minItems":1,"maxItems":10,"items":{"type":"object","required":["op"],"additionalProperties":false,"properties":{
    "op":{"type":"string","enum":["add_group","set_gems","add_gem","remove_gem","set_main","enable_group","disable_group","remove_group","set_gem_level"]},
    "group":{"type":"integer","description":"1-based socket group index as shown by get_build section=skills"},
    "slot":{"type":"string"},
    "gems":{"type":"array","items":{"type":"string"},"description":"gem names, active skill first"},
    "gem":{"type":"string"},
    "level":{"type":"integer"},"quality":{"type":"integer"},
    "full_dps":{"type":"boolean","description":"add_group/set_gems: include in Full DPS"}}}}}}`

const schemaEditBuild = `{"type":"object","required":["handle","ops"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},"dry_run":{"type":"boolean","default":false},
  "watch":{"type":"array","items":{"type":"string"}},
  "ops":{"type":"array","minItems":1,"maxItems":20,"items":{"type":"object","required":["op"],"additionalProperties":false,"properties":{
    "op":{"type":"string","enum":["set_level","set_bandit","set_pantheon","set_config","unset_config","set_notes","append_notes"]},
    "level":{"type":"integer"},
    "bandit":{"type":"string","enum":["None","Alira","Kraityn","Oak"]},
    "major":{"type":"string"},"minor":{"type":"string"},
    "var":{"type":"string","description":"config option var or its label, e.g. enemyIsBoss / \"Is the enemy a Boss?\""},
    "value":{"description":"bool, number or list value"},
    "text":{"type":"string"}}}}}}`

const schemaRankCandidates = `{"type":"object","required":["handle","kind"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "kind":{"type":"string","enum":["nodes","remove_nodes","supports","uniques"]},
  "stat":{"type":"string","default":"MainDPS","description":"any power stat (search_data kind=stat shows them) or \"OffenceDefence\""},
  "max_points":{"type":"integer","default":5,"description":"nodes: longest path considered"},
  "node_types":{"type":"array","items":{"type":"string","enum":["notable","keystone","mastery","small","ascendancy"]}},
  "group":{"type":"integer","description":"supports: socket group, default main"},
  "slot":{"type":"string","description":"uniques: slot to fill"},
  "max_level_req":{"type":"integer"},
  "limit":{"type":"integer","default":8,"maximum":25}}}`

const schemaCompareBuilds = `{"type":"object","required":["a","b"],"additionalProperties":false,"properties":{
  "a":{"type":"string","description":"handle, or handle@revision"},
  "b":{"type":"string"},
  "stats":{"type":"array","items":{"type":"string"}},
  "structure":{"type":"boolean","default":true}}}`

const schemaRevertBuild = `{"type":"object","required":["handle"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "to_revision":{"type":"integer","description":"default = the revision before the current one"}}}`

const schemaExportBuild = `{"type":"object","required":["handle"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "format":{"type":"string","enum":["code","xml"],"default":"code"},
  "path":{"type":"string","description":"optional absolute path to also write the XML"}}}`

const schemaOptimize = `{"type":"object","required":["handle","slots","objective"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "slots":{"type":"array","minItems":1,"maxItems":6,"items":{"type":"object","required":["kind"],"additionalProperties":false,"properties":{
    "kind":{"type":"string","enum":["item","supports","nodes"]},
    "slot":{"type":"string","description":"item: equipment slot, e.g. \"Ring 1\""},
    "uniques":{"type":"array","items":{"type":"string"},"description":"item: unique names (DB text)"},
    "texts":{"type":"array","items":{"type":"string"},"description":"item: full PoB item texts"},
    "keep_current":{"type":"boolean","default":true,"description":"item/supports: the current choice is also a candidate"},
    "group":{"type":"integer","description":"supports: socket group, default main"},
    "candidates":{"type":"array","items":{"type":"string"},"description":"supports: support gem names; nodes: node names or ids"},
    "replace":{"type":"array","items":{"type":"string"},"description":"supports: current supports that may be swapped out (default: all current supports)"},
    "k":{"type":"integer","description":"supports: number of replaceable sockets to fill (default = len(replace)); nodes: up to k nodes"},
    "path":{"type":"boolean","default":true,"description":"nodes: count path points to each chosen node"}}}},
  "objective":{"type":"object","required":["stat"],"additionalProperties":false,"properties":{
    "stat":{"type":"string","description":"any stat key, e.g. MainDPS, TotalEHP, Life"},
    "goal":{"type":"string","enum":["maximize","minimize"],"default":"maximize"}}},
  "constraints":{"type":"array","maxItems":10,"items":{"type":"object","required":["stat","op","value"],"additionalProperties":false,"properties":{
    "stat":{"type":"string","description":"stat key, or \"points\" (passive points added), \"level_needed\""},
    "op":{"type":"string","enum":[">=","<=","=="]},
    "value":{"type":"number"},
    "relative":{"type":"boolean","default":false,"description":"value is a fraction of the current value, e.g. Life >= 0.95"}}}},
  "budget":{"type":"integer","default":300,"maximum":2000,"description":"max full calculations"},
  "max_ms":{"type":"integer","default":20000,"maximum":60000},
  "seed":{"type":"integer","default":1},
  "top":{"type":"integer","default":3,"maximum":5},
  "apply":{"type":"boolean","default":false,"description":"commit the best feasible combination as a new revision"}}}`
