package mcp

func tools() []toolDef {
	return []toolDef{
		{"new_build", "Create an empty PoE1 character from class (and ascendancy, level). Returns the handle (b1, b2...) every other tool takes.", schemaNewBuild, (*Server).newBuild},
		{"import_build", "Load a Path of Building build from source: an export code, a pobb.in / poe.ninja / pastebin / pob.codes / maxroll URL, raw XML or an absolute path to a file holding XML or a code. Returns its handle, summary and an audit (also in get_build section=warnings) over every skill set and config set, each config line naming its set: gem quality above 20 (23 needs a corrupted gem, above 23 is impossible) and levels above the natural maximum (one more needs a corrupted gem; level and quality from one corruption are called out), legacy gems, Delve-only mods on equipped items, defences custom modifiers supply (resistances and their maximums, damage reduction, block, life, ...) and the Full DPS / main DPS with and without them, legacy customMods text Path of Building ignores, config values above their limits (charges, Rage and other capped counts) and Rage with no source, and items the game could not produce in every item set and every tree's jewel sockets, each naming its sets (illegal_items: more than 3 prefixes or suffixes, more than 6 explicit mods counting {custom} lines, more crafted mods than allowed, explicit values above every mod with that wording, implicits above the base's range; socket colours are never illegal, scan_build parts=[\"sockets\"] ranks recolouring).", schemaImportBuild, (*Server).importBuild},
		{"get_build", "Read a build's structure by section: summary, tree, items, sockets (each item's sockets, largest link and the groups in it), skills (socket groups, their slot, unsocketed groups), config, notes or warnings (PoB's own plus link, level and charge-override checks). Pass handle; without it you get the list of handles. Numbers are in get_stats.", schemaGetBuild, (*Server).getBuild},
		{"get_stats", "Read calculated stats for a handle by PoB output key (Life, FireResist, CombinedDPS, MainDPS...); explain=<key> lists its largest sources. Unknown keys come back with did_you_mean; search_data kind=stat finds keys.", schemaGetStats, (*Server).getStats},
		{"search_data", "Search game data by kind (gem, support, unique, base, node, keystone, mastery, affix, config, stat) and a short query; one or two words match best. kind=affix with base lists what a base can roll (its prefixes and suffixes, then its crafting bench and veiled mods, type 'Crafted ...' or 'Veiled ...'); kind=support with handle marks supports compatible with the main skill; kind=stat with handle also searches every output key the build calculates.", schemaSearchData, (*Server).searchData},
		{"edit_tree", "Edit the passive tree with a batch of ops: allocate (with the cheapest path), deallocate (also removes the nodes that depend on it; do not list those again in the same batch), set_mastery (node + effect id; allocates the mastery too), set_class, set_ascendancy, reset. Returns a stat delta; dry_run previews.", schemaEditTree, (*Server).editTree},
		{"edit_items", "Edit equipment with a batch of ops: craft (base + up to 3 prefix and 3 suffix affixes; search_data kind=affix base=<base> lists them), equip_unique, equip_text, unequip, set_flask_active, socket_jewel. Base and unique names must exist; search_data finds them. Returns a stat delta; dry_run previews and shows the item's mods.", schemaEditItems, (*Server).editItems},
		{"edit_skills", "Edit socket groups and gems with a batch of ops (add_group, set_gems, add_gem, remove_gem, set_main, set_slot, enable/disable/remove_group, set_gem_level, set_part). Give add_group a slot (the item whose sockets hold the gems); a group without one gets no links in game and every result lists it under unsocketed. " + legacyAwakenedNote() + " A gem given no level takes the build's default gem level, held to the highest level the character meets the requirement of. A swapped-in gem keeps the quality of the gem it replaces and a new gem starts at the build's default (often 0): set quality (up to 20, as Gemcutter's Prisms raise any gem's quality) with set_gem_level, because quality is real damage. set_gems keeps each gem's enabled state unless enabled is given; add_gem gives a new active gem the group's Full DPS count. Returns a stat delta; dry_run previews.", schemaEditSkills, (*Server).editSkills},
		{"edit_build", "Edit level, bandit, pantheon, configuration options (set_config var=<option var or label> value=...; search_data kind=config finds them) and notes with a batch of ops. Returns a stat delta; dry_run previews.", schemaEditBuild, (*Server).editBuild},
		{"rank_candidates", "Rank what to take by PoB's power calculation for a stat: kind=nodes (masteries come with their best effect id for set_mastery), remove_nodes (keep=[...] limits it to removals that hold your constraints; \"also\" lists limits, caps and immunities a removal changes), supports (on a full group, rows rank the best swap instead), or uniques for a slot. kind=nodes also lists mastery_switches: better effects for masteries you already have. Each row shows the watched stats after the change, d (its change to DPS, EHP, lowest max hit, life, totem life, unreserved mana) and flags (forced config conditions its value depends on, new warnings, overfilled sockets, totems, lost immunities or ailments); headroom appears when it differs from base_headroom. Re-rank after each pick.", schemaRankCandidates, (*Server).rankCandidates},
		{"compare_builds", "Compare two builds or two revisions (a and b: handle or handle@revision): stat differences and what changed.", schemaCompareBuilds, (*Server).compareBuilds},
		{"revert_build", "Move a handle back (or forward) to another revision. Returns the stat delta.", schemaRevertBuild, (*Server).revertBuild},
		{"export_build", "Export a build as a PoB code (or XML) that pastes into Path of Building. Export once the build is finished.", schemaExportBuild, (*Server).exportBuild},
		{"scan_build", "Scan the whole build in one call and rank, by PoB's calculation of one stat, what to change in it: uniques for every gear slot, a flask (unique or utility) in the weakest or an empty flask slot, the best affix for each open prefix/suffix on rares and magic items (affix tiers the base can roll at its item level and bench crafts; Delve drop-restricted and fossil-only mods only with delve:true), anoints, the notables of socketed cluster jewels (parts=[\"clusters\"]), and with parts=[\"reservation\"] each skill's reservation plus, with mana_needed, the nodes, gem levels and support removals that free unreserved mana, and with parts=[\"sockets\"] recolouring each item's sockets to its gems' colours (a matching socket adds gem quality; socket count and links stay). Rows carry gain, d (change to DPS, EHP, lowest max hit, life, totem life, unreserved mana), flags and resist_gap (resistances a swap leaves under cap; keep_resists drops those rows). Apply a row with apply=<row id>.", schemaScanBuild, (*Server).scanBuild},
		{"optimize", "Search combinations of listed choices for the best objective stat under constraints: items per slot, support sets for a group, nodes to add (candidates, as strings) or remove (remove lists allocated nodes only). Large spaces use local search. Nothing is applied unless apply:true.", schemaOptimize, (*Server).optimize},
		{"price_build", priceBuildDesc, schemaPriceBuild, (*Server).priceBuild},
	}
}

const schemaNewBuild = `{"type":"object","required":["class"],"additionalProperties":false,"properties":{
  "class":{"type":"string","description":"Marauder|Ranger|Witch|Duelist|Templar|Shadow|Scion"},
  "ascendancy":{"type":"string","description":"e.g. Elementalist; omit for none"},
  "level":{"type":"integer","minimum":1,"maximum":100,"default":90},
  "name":{"type":"string"},
  "from_handle":{"type":"string","description":"copy an existing handle instead (fork); other fields then override"}}}`

const schemaImportBuild = `{"type":"object","required":["source"],"additionalProperties":false,"properties":{
  "source":{"type":"string","description":"PoB export code, a pobb.in / poe.ninja / pastebin / poedb / pob.codes / maxroll URL, raw XML, or an absolute path to a file holding the XML or the code (.xml, .txt)"},
  "convert_tree":{"type":"boolean","default":true},
  "name":{"type":"string"}}}`

const schemaGetBuild = `{"type":"object","additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "section":{"type":"string","enum":["summary","tree","items","sockets","skills","config","notes","warnings"],"default":"summary"},
  "detail":{"type":"boolean","default":false},
  "cursor":{"type":"string","description":"skills, items: the next_cursor of a truncated page"}}}`

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
    "effect":{"type":"string","description":"set_mastery: effect id (from rank_candidates or search_data kind=mastery) or its text"},
    "path":{"type":"boolean","default":true,"description":"allocate: also allocate the cheapest path"},
    "with_dependents":{"type":"boolean","default":true,"description":"deallocate: also remove nodes that would be disconnected"},
    "class":{"type":"string"},"ascendancy":{"type":"string"}}}}}}`

const schemaEditItems = `{"type":"object","required":["handle","ops"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},"dry_run":{"type":"boolean","default":false},
  "watch":{"type":"array","items":{"type":"string"}},
  "ops":{"type":"array","minItems":1,"maxItems":10,"items":{"type":"object","required":["op"],"additionalProperties":false,"properties":{
    "op":{"type":"string","enum":["equip_unique","craft","equip_text","unequip","set_flask_active","socket_jewel"]},
    "slot":{"type":"string","description":"Helmet, Body Armour, Gloves, Boots, Amulet, Ring 1, Ring 2, Belt, Weapon 1, Weapon 2, Flask 1-5"},
    "name":{"type":"string","description":"equip_unique: unique name; variant chosen as PoB's default. \"Foulborn <unique>\" equips its Foulborn form; one with several mutations takes \"Foulborn <unique>: <part of a mutated line>\" (several joined with ;)"},
    "base":{"type":"string","description":"craft: base type, e.g. Astral Plate"},
    "affixes":{"type":"array","maxItems":6,"items":{"type":"string"},"description":"craft: affix labels, e.g. \"+# to maximum Life\", or intent words resolved per base (life, mana, es, armour, evasion, fire_res, cold_res, lightning_res, chaos_res, all_res, str, dex, int, attributes, attack_speed, cast_speed, move_speed); best tier the base's item level allows, max roll"},
    "strict":{"type":"boolean","default":false,"description":"craft: fail the batch on an affix the base cannot take; by default it is skipped and listed in skipped"},
    "sockets":{"type":"string","description":"craft: e.g. R-R-G-G-B-B; default = PoB's crafted default"},
    "text":{"type":"string","description":"equip_text: full PoB item text"},
    "active":{"type":"boolean"},
    "node":{"description":"socket_jewel: jewel socket node id or name; slot not used"}}}}}}`

const schemaEditSkills = `{"type":"object","required":["handle","ops"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},"dry_run":{"type":"boolean","default":false},
  "watch":{"type":"array","items":{"type":"string"}},
  "ops":{"type":"array","minItems":1,"maxItems":10,"items":{"type":"object","required":["op"],"additionalProperties":false,"properties":{
    "op":{"type":"string","enum":["add_group","set_gems","add_gem","remove_gem","set_main","set_slot","enable_group","disable_group","remove_group","set_gem_level","set_part"]},
    "group":{"type":"integer","description":"1-based socket group index as shown by get_build section=skills; every op in a batch names the groups as they were before it, and groups add_group creates are numbered after them. set_gem_level may leave it out when one gem in the build has that name"},
    "slot":{"type":"string","description":"add_group/set_gems/set_slot: the item whose sockets hold the group, e.g. Body Armour; without one the group sits in no item and gets no links in game"},
    "gems":{"type":"array","items":{"type":"string"},"description":"gem names, active skill first"},
    "gem":{"type":"string"},
    "level":{"type":"integer","description":"new gems: default the build's default gem level; set_gems keeps the level of gems already in the group"},
    "quality":{"type":"integer","description":"new gems: default the build's default gem quality (often 0); set_gems keeps the quality of the gem a new one replaces; set_gem_level sets it (0-20) on an existing gem"},
    "full_dps":{"type":"boolean","description":"add_group/set_gems: include in Full DPS"},
    "part":{"type":["string","integer"],"description":"set_part: the skill part (name or 1-based index) of gem, default the group's main active gem; get_build section=skills lists each multi-part gem's part and parts"},
    "enabled":{"type":"boolean","description":"set_gems: true enables every gem it sets, false disables them; by default each gem keeps its slot's enabled state, as Path of Building does"}}}}}}`

const schemaEditBuild = `{"type":"object","required":["handle","ops"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},"dry_run":{"type":"boolean","default":false},
  "watch":{"type":"array","items":{"type":"string"}},
  "ops":{"type":"array","minItems":1,"maxItems":20,"items":{"type":"object","required":["op"],"additionalProperties":false,"properties":{
    "op":{"type":"string","enum":["set_level","set_bandit","set_pantheon","set_config","unset_config","set_notes","append_notes"]},
    "level":{"type":"integer"},
    "bandit":{"type":"string","enum":["None","Alira","Kraityn","Oak"]},
    "major":{"type":"string"},"minor":{"type":"string"},
    "var":{"type":"string","description":"config option var or its label, e.g. enemyIsBoss / \"Is the enemy a Boss?\"; customMods takes PoB custom modifier lines, one per line"},
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
  "limit":{"type":"integer","default":8,"maximum":25},
  "watch":{"type":"array","items":{"type":"string"},"description":"extra stat keys each row reports after the change (always: the ranked stat, Life, TotalEHP and resistance headroom)"},
  "keep":{"type":"array","maxItems":10,"description":"remove_nodes: only removals that keep these, most points freed first","items":{"type":"object","required":["stat","op","value"],"additionalProperties":false,"properties":{
    "stat":{"type":"string"},
    "op":{"type":"string","enum":[">=","<="]},
    "value":{"type":"number"},
    "relative":{"type":"boolean","default":false,"description":"value is a fraction of the current value, e.g. Life >= 0.95; TotalEHP >= 0.9 or TotemLife >= 0.9 keeps a result from trading away defences or totem life"}}}},
  "worst_case":{"type":"boolean","default":false,"description":"also score each row with its favourable combat conditions questioned (forced ones off, unset ones a modifier reads negated on) and Full DPS counts held to the totem/mine limit: worst_gain / worst_loss and change columns worst_d, go-pob estimates; rank by that. A worst_gain is capped at the gain's share of the build where the change itself provides a questioned condition"}}}`

const schemaCompareBuilds = `{"type":"object","required":["a","b"],"additionalProperties":false,"properties":{
  "a":{"type":"string","description":"handle, or handle@revision"},
  "b":{"type":"string"},
  "stats":{"type":"array","items":{"type":"string"}},
  "structure":{"type":"boolean","default":true},
  "steps":{"type":"boolean","default":false,"description":"a and b are two revisions of one handle: one row per revision between them, with what changed and the stats that moved"}}}`

const schemaRevertBuild = `{"type":"object","required":["handle"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "to_revision":{"type":"integer","description":"default = the revision before the current one"}}}`

const schemaExportBuild = `{"type":"object","required":["handle"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "format":{"type":"string","enum":["code","xml","both"],"default":"code"},
  "path":{"type":"string","description":"optional absolute file path: the chosen format is written there and only paths and sizes are returned; with both, the code goes beside the XML as <name>.txt"}}}`

const schemaOptimize = `{"type":"object","required":["handle","slots","objective"],"additionalProperties":false,"properties":{
  "handle":{"type":"string"},
  "slots":{"type":"array","minItems":1,"maxItems":6,"items":{"type":"object","required":["kind"],"additionalProperties":false,"properties":{
    "kind":{"type":"string","enum":["item","supports","nodes","gem"]},
    "slot":{"type":"string","description":"item: equipment slot, e.g. \"Ring 1\""},
    "uniques":{"type":"array","items":{"type":"string"},"description":"item: unique names (DB text)"},
    "texts":{"type":"array","items":{"type":"string"},"description":"item: full PoB item texts"},
    "keep_current":{"type":"boolean","default":true,"description":"item/supports: the current choice is also a candidate"},
    "group":{"type":"integer","description":"supports/gem: socket group, default main"},
    "gem":{"type":"string","description":"gem: the gem in group whose level and quality vary"},
    "levels":{"type":"array","items":{"type":"integer"},"description":"gem: levels to try (default current and maximum)"},
    "qualities":{"type":"array","items":{"type":"integer"},"description":"gem: qualities to try (default current and 20, 23 if corrupted)"},
    "corrupted":{"type":"boolean","default":false,"description":"gem: allow one corruption: one level over the natural maximum or quality up to 23, never both"},
    "candidates":{"type":"array","items":{"type":"string"},"description":"supports: support gem names; nodes: node names or ids, as strings (\"12345\")"},
    "replace":{"type":"array","items":{"type":"string"},"description":"supports: current supports that may be swapped out (default: all current supports)"},
    "remove":{"type":"array","items":{"type":"string"},"description":"nodes: currently allocated nodes that may be removed (with the nodes that depend on them), freeing points"},
    "k":{"type":"integer","description":"supports: number of replaceable sockets to fill (default = len(replace)); nodes: add up to k of candidates"},
    "path":{"type":"boolean","default":true,"description":"nodes: count path points to each chosen node, and offer each removal also with the dead-end path (up to 5 points) that leads only to it"}}}},
  "objective":{"type":"object","required":["stat"],"additionalProperties":false,"properties":{
    "stat":{"type":"string","description":"any stat key, e.g. MainDPS, TotalEHP, Life, or BuildCost (the build's price in chaos; minimize it with constraints on what to keep)"},
    "goal":{"type":"string","enum":["maximize","minimize"],"default":"maximize"}}},
  "constraints":{"type":"array","maxItems":10,"items":{"type":"object","required":["stat","op","value"],"additionalProperties":false,"properties":{
    "stat":{"type":"string","description":"stat key, or \"points_added\" (passive points added, net of removals; a mastery costs 1), \"points_total\" (points used after the change), \"level_needed\", \"BuildCost\" (the build's price in chaos after the change, at poe.ninja prices)"},
    "op":{"type":"string","enum":[">=","<=","=="]},
    "value":{"type":"number"},
    "relative":{"type":"boolean","default":false,"description":"value is a fraction of the current value, e.g. Life >= 0.95; TotalEHP >= 0.9 or TotemLife >= 0.9 keeps a result from trading away defences or totem life"}}}},
  "budget":{"type":"integer","default":2000,"maximum":2000,"description":"max full calculations"},
  "max_ms":{"type":"integer","default":20000,"maximum":60000},
  "seed":{"type":"integer","default":1},
  "top":{"type":"integer","default":3,"maximum":5},
  "apply":{"type":"boolean","default":false,"description":"commit the best feasible combination as a new revision"},
  "pareto":{"type":"boolean","default":false,"description":"also list the feasible results no other evaluated one beats on both MainDPS and TotalEHP"},` + schemaBudgetProps + `}}`
