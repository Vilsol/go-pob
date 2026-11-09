package calculator2

import (
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

type CompiledList[T any] struct {
	Regex *lazyRegexp
	Value T
}

var conquerorList = map[string]mod.ConquerorType{
	//pob2:block:src/Modules/ModParser.lua:52-59
	"vorana":  {ID: "1", Type: "kalguur"},
	"medved":  {ID: "2", Type: "kalguur"},
	"olroth":  {ID: "3", Type: "kalguur"},
	"amanamu": {ID: "1", Type: "abyss"},
	"kulemak": {ID: "2", Type: "abyss"},
	"kurgal":  {ID: "3", Type: "abyss"},
	"tecrod":  {ID: "4", Type: "abyss"},
	"ulaman":  {ID: "5", Type: "abyss"},
}

// List of modifier forms
var addToClusterRe = regexp.MustCompile(`^Added Small Passive Skills also grant: (.+)$`)

var formListCompiled map[string]CompiledList[string]
var formList = map[string]string{
	//pob2:block:src/Modules/ModParser.lua:63-157
	`^(\d+)% increased`:                                      "INC",
	`^(\d+)% faster`:                                         "INC",
	`^(\d+)% reduced`:                                        "RED",
	`^(\d+)% slower`:                                         "RED",
	`^(\d+)% more`:                                           "MORE",
	`^you have (\d+)% more`:                                  "MORE",
	`^(\d+)% less`:                                           "LESS",
	`^you have (\d+)% less`:                                  "LESS",
	`^([\+\-][\d\.]+)%?`:                                     "BASE",
	`^([\+\-][\d\.]+)%? to`:                                  "BASE",
	`^([\+\-]?[\d\.]+)%? of`:                                 "BASE",
	`^([\+\-][\d\.]+)%? base`:                                "BASE",
	`^([\+\-]?[\d\.]+)%? additional`:                         "BASE",
	`(\d+) additional hits?`:                                 "BASE",
	`^throw up to (\d+)`:                                     "BASE",
	`^you gain ([\d\.]+)`:                                    "GAIN",
	`^gains? ([\d\.]+)% of their`:                            "GAIN",
	`^gains? ([\d\.]+)% ?o?f?`:                               "GAIN",
	`^gains? ([\d\.]+)`:                                      "GAIN",
	`^gain \+(\d+)% to`:                                      "GAIN",
	`^you lose ([\d\.]+)`:                                    "LOSE",
	`^loses? ([\d\.]+)% of`:                                  "LOSE",
	`^lose ([\d\.]+)`:                                        "LOSE",
	`^lose \+(\d+)% to`:                                      "LOSE",
	`^grants ([\d\.]+)`:                                      "GRANTS",
	`^grants ([\d\.]+) additional`:                           "GRANTS_GLOBAL",
	`^removes? ([\d\.]+) ?o?f? ?y?o?u?r?`:                    "REMOVES",
	`^(\d+)`:                                                 "BASE",
	`^a? ?([\+\-]?\d+)% chance`:                              "CHANCE",
	`^([\+\-]?\d+)% chance to gain `:                         "FLAG",
	`^([\+\-]?\d+)% additional chance`:                       "CHANCE",
	`costs? ([\+\-]?\d+)`:                                    "TOTALCOST",
	`skills cost ([\+\-]?\d+)`:                               "BASECOST",
	`penetrates? (\d+)%`:                                     "PEN",
	`penetrates (\d+)% of`:                                   "PEN",
	`penetrates (\d+)% of enemy`:                             "PEN",
	`^([\d\.]+) (.+) regeneration per second`:                "REGENFLAT",
	`^([\d\.]+)% (.+) regeneration per second`:               "REGENPERCENT",
	`^([\d\.]+)% of (.+) regeneration per second`:            "REGENPERCENT",
	`^([\d\.]+)% of (.+) regenerated per second`:             "REGENPERCENT",
	`^regenerate ([\d\.]+) (.*?) per second`:                 "REGENFLAT",
	`^regenerate ([\d\.]+)% (.*?) per second`:                "REGENPERCENT",
	`^regenerate ([\d\.]+)% of (.*?) per second`:             "REGENPERCENT",
	`^regenerate ([\d\.]+)% of your (.*?) per second`:        "REGENPERCENT",
	`^you regenerate ([\d\.]+)% of (.*?) per second`:         "REGENPERCENT",
	`^([\d\.]+) (.+) lost per second`:                        "DEGENFLAT",
	`^([\d\.]+)% (.+) lost per second`:                       "DEGENPERCENT",
	`^([\d\.]+)% of (.+) lost per second`:                    "DEGENPERCENT",
	`^lose ([\d\.]+) (.*?) per second`:                       "DEGENFLAT",
	`^lose ([\d\.]+)% (.*?) per second`:                      "DEGENPERCENT",
	`^lose ([\d\.]+)% of (.*?) per second`:                   "DEGENPERCENT",
	`^lose ([\d\.]+)% of your (.*?) per second`:              "DEGENPERCENT",
	`^you lose ([\d\.]+)% of (.*?) per second`:               "DEGENPERCENT",
	`^([\d\.]+) (\w+) damage taken per second`:               "DEGEN",
	`^([\d\.]+) (\w+) damage per second`:                     "DEGEN",
	`(\d+) to (\d+) added (\w+) damage`:                      "DMG",
	`(\d+)\-(\d+) added (\w+) damage`:                        "DMG",
	`(\d+) to (\d+) additional (\w+) damage`:                 "DMG",
	`(\d+)\-(\d+) additional (\w+) damage`:                   "DMG",
	`^(\d+) to (\d+) (\w+) damage`:                           "DMG",
	`adds (\d+) to (\d+) (\w+) damage`:                       "DMG",
	`adds (\d+)\-(\d+) (\w+) damage`:                         "DMG",
	`adds (\d+) to (\d+) (\w+) damage to attacks`:            "DMGATTACKS",
	`adds (\d+)\-(\d+) (\w+) damage to attacks`:              "DMGATTACKS",
	`adds (\d+) to (\d+) (\w+) attack damage`:                "DMGATTACKS",
	`adds (\d+)\-(\d+) (\w+) attack damage`:                  "DMGATTACKS",
	`(\d+) to (\d+) added attack (\w+) damage`:               "DMGATTACKS",
	`(\d+) to (\d+) additional attack (\w+) damage`:          "DMGATTACKS",
	`adds (\d+) to (\d+) (\w+) damage to spells`:             "DMGSPELLS",
	`adds (\d+)\-(\d+) (\w+) damage to spells`:               "DMGSPELLS",
	`adds (\d+) to (\d+) (\w+) spell damage`:                 "DMGSPELLS",
	`adds (\d+)\-(\d+) (\w+) spell damage`:                   "DMGSPELLS",
	`(\d+) to (\d+) added spell (\w+) damage`:                "DMGSPELLS",
	`(\d+) to (\d+) (\w+) thorns damage`:                     "DMGTHORNS",
	`gain (\w+) thorns damage`:                               "DMGTHORNSBASE",
	`adds (\d+) to (\d+) (\w+) damage to attacks and spells`: "DMGBOTH",
	`adds (\d+)\-(\d+) (\w+) damage to attacks and spells`:   "DMGBOTH",
	`adds (\d+) to (\d+) (\w+) damage to spells and attacks`: "DMGBOTH",
	`adds (\d+)\-(\d+) (\w+) damage to spells and attacks`:   "DMGBOTH",
	`adds (\d+) to (\d+) (\w+) damage to hits`:               "DMGBOTH",
	`adds (\d+)\-(\d+) (\w+) damage to hits`:                 "DMGBOTH",
	"^you have ":                                             "FLAG",
	"^have ":                                                 "FLAG",
	"^you are ":                                              "FLAG",
	"^are ":                                                  "FLAG",
	"^gain ":                                                 "FLAG",
	"^you gain ":                                             "FLAG",
	`is (\-?\d+)%? `:                                         "OVERRIDE",
	"is doubled":                                             "DOUBLED",
	"doubles?":                                               "DOUBLED",
	"causes? double":                                         "DOUBLED",
	"^immunity to ":                                          "IMMUNE",
	"^immune to being ":                                      "IMMUNE",
	"^immune to ":                                            "IMMUNE",
	"^cannot be ":                                            "IMMUNE",
}

type modNameListType struct {
	names          []string
	tag            mod.Tag
	tagList        []mod.Tag
	flags          mod.MFlag
	addToMinion    bool
	keywordFlags   mod.KeywordFlag
	addToSkill     mod.Tag
	addToMinionTag mod.Tag

	playerTag         mod.Tag
	playerTagList     []mod.Tag
	applyToEnemy      bool
	fn                func(caps []string) modNameListType
	modSuffix         string
	addToAura         bool
	onlyAddToBanners  bool
	newAuraOnlyAllies bool
	actorEnemy        bool
	newAura           bool

	flagModType  string
	flagModValue float64
}

// Map of modifier names
var modNameListCompiled map[string]CompiledList[modNameListType]
var modNameList = map[string]modNameListType{
	//pob2:block:src/Modules/ModParser.lua:163-984
	"strength":                                  {names: []string{"Str"}},
	"dexterity":                                 {names: []string{"Dex"}},
	"intelligence":                              {names: []string{"Int"}},
	"strength and dexterity":                    {names: []string{"Str", "Dex", "StrDex"}},
	"strength and intelligence":                 {names: []string{"Str", "Int", "StrInt"}},
	"dexterity and intelligence":                {names: []string{"Dex", "Int", "DexInt"}},
	"attributes":                                {names: []string{"Str", "Dex", "Int", "All"}},
	"all attributes":                            {names: []string{"Str", "Dex", "Int", "All"}},
	"devotion":                                  {names: []string{"Devotion"}},
	"tribute":                                   {names: []string{"Tribute"}},
	"maximum darkness":                          {names: []string{"Darkness"}},
	"spirit":                                    {names: []string{"Spirit"}},
	"maximum spirit":                            {names: []string{"Spirit"}},
	"life":                                      {names: []string{"Life"}},
	"maximum life":                              {names: []string{"Life"}},
	"life regeneration":                         {names: []string{"LifeRegen"}},
	"life regeneration rate":                    {names: []string{"LifeRegen"}},
	"life and mana regeneration rate":           {names: []string{"LifeRegen", "ManaRegen"}},
	"mana":                                      {names: []string{"Mana"}},
	"maximum mana":                              {names: []string{"Mana"}},
	"mana regeneration":                         {names: []string{"ManaRegen"}},
	"mana regeneration rate":                    {names: []string{"ManaRegen"}},
	"runic ward regeneration rate":              {names: []string{"WardRegen"}},
	"mana cost":                                 {names: []string{"ManaCost"}},
	"mana cost of":                              {names: []string{"ManaCost"}},
	"mana cost of skills":                       {names: []string{"ManaCost"}},
	"mana cost of attacks":                      {names: []string{"ManaCost"}, tag: mod.SkillType("Attack")},
	"total cost":                                {names: []string{"Cost"}},
	"total mana cost":                           {names: []string{"ManaCost"}},
	"total mana cost of skills":                 {names: []string{"ManaCost"}},
	"life cost":                                 {names: []string{"LifeCost"}},
	"life cost of skills":                       {names: []string{"LifeCost"}},
	"rage cost of skills":                       {names: []string{"RageCost"}},
	"cost of":                                   {names: []string{"Cost"}},
	"cost of skills":                            {names: []string{"Cost"}},
	"cost of attacks":                           {names: []string{"Cost"}, tag: mod.SkillType("Attack")},
	"cost of retaliation skills":                {names: []string{"Cost"}, tag: mod.SkillType("Retaliation")},
	"cost efficiency":                           {names: []string{"CostEfficiency"}},
	"cost efficiency of skills":                 {names: []string{"CostEfficiency"}},
	"cost efficiency of attacks":                {names: []string{"CostEfficiency"}, tag: mod.SkillType("Attack")},
	"mana cost efficiency":                      {names: []string{"ManaCostEfficiency"}},
	"mana cost efficiency of skills":            {names: []string{"ManaCostEfficiency"}},
	"mana cost efficiency of attacks":           {names: []string{"ManaCostEfficiency"}, tag: mod.SkillType("Attack")},
	"life cost efficiency":                      {names: []string{"LifeCostEfficiency"}},
	"life cost efficiency of skills":            {names: []string{"LifeCostEfficiency"}},
	"spirit cost efficiency":                    {names: []string{"SpiritCostEfficiency"}},
	"spirit cost efficiency of skills":          {names: []string{"SpiritCostEfficiency"}},
	"runic ward cost efficiency":                {names: []string{"WardCostEfficiency"}},
	"energy shield cost efficiency":             {names: []string{"ESCostEfficiency"}},
	"energy shield cost efficiency of skills":   {names: []string{"ESCostEfficiency"}},
	"es cost efficiency":                        {names: []string{"ESCostEfficiency"}},
	"es cost efficiency of skills":              {names: []string{"ESCostEfficiency"}},
	"soul cost efficiency":                      {names: []string{"SoulCostEfficiency"}},
	"soul cost efficiency of skills":            {names: []string{"SoulCostEfficiency"}},
	"rage cost efficiency":                      {names: []string{"RageCostEfficiency"}},
	"rage cost efficiency of skills":            {names: []string{"RageCostEfficiency"}},
	"skill mana costs converted to life costs":  {names: []string{"HybridManaAndLifeCost_Life"}},
	"spell mana cost converted to life cost":    {names: []string{"HybridManaAndLifeCost_Life"}, tag: mod.SkillType("Spell")},
	"mana reserved":                             {names: []string{"ManaReserved"}},
	"mana reservation":                          {names: []string{"ManaReserved"}},
	"mana reservation of skills":                {names: []string{"ManaReserved"}, tag: mod.SkillType("Aura")},
	"mana reservation efficiency of skills":     {names: []string{"ManaReservationEfficiency"}},
	"life reservation efficiency of skills":     {names: []string{"LifeReservationEfficiency"}},
	"spirit reservation efficiency of skills":   {names: []string{"SpiritReservationEfficiency"}},
	"spirit reservation efficiency":             {names: []string{"SpiritReservationEfficiency"}},
	"reservation of skills":                     {names: []string{"Reserved"}},
	"mana reservation if cast as an aura":       {names: []string{"ManaReserved"}, tag: mod.SkillType("Aura")},
	"reservation if cast as an aura":            {names: []string{"Reserved"}, tag: mod.SkillType("Aura")},
	"reservation":                               {names: []string{"Reserved"}},
	"reservation efficiency":                    {names: []string{"ReservationEfficiency"}},
	"reservation efficiency of skills":          {names: []string{"ReservationEfficiency"}},
	"mana reservation efficiency":               {names: []string{"ManaReservationEfficiency"}},
	"life reservation efficiency":               {names: []string{"LifeReservationEfficiency"}},
	"maximum energy shield":                     {names: []string{"EnergyShield"}},
	"energy shield recharge rate":               {names: []string{"EnergyShieldRecharge"}},
	"start of energy shield recharge":           {names: []string{"EnergyShieldRechargeFaster"}},
	"restoration of ward":                       {names: []string{"WardRechargeFaster"}},
	"armour":                                    {names: []string{"Armour"}},
	"evasion":                                   {names: []string{"Evasion"}},
	"evasion rating":                            {names: []string{"Evasion"}},
	"energy shield":                             {names: []string{"EnergyShield"}},
	"ward":                                      {names: []string{"Ward"}},
	"runic ward":                                {names: []string{"Ward"}},
	"maximum runic ward":                        {names: []string{"Ward"}},
	"armour and evasion":                        {names: []string{"ArmourAndEvasion"}},
	"armour and evasion rating":                 {names: []string{"ArmourAndEvasion"}},
	"evasion rating and armour":                 {names: []string{"ArmourAndEvasion"}},
	"armour and energy shield":                  {names: []string{"ArmourAndEnergyShield"}},
	"evasion rating and energy shield":          {names: []string{"EvasionAndEnergyShield"}},
	"global evasion rating and energy shield":   {names: []string{"Evasion", "EnergyShield"}},
	"evasion and energy shield":                 {names: []string{"EvasionAndEnergyShield"}},
	"armour, evasion and energy shield":         {names: []string{"Armour", "Evasion", "EnergyShield"}},
	"defences":                                  {names: []string{"Defences"}},
	"to evade":                                  {names: []string{"EvadeChance"}},
	"chance to evade":                           {names: []string{"EvadeChance"}},
	"to evade attacks":                          {names: []string{"EvadeChance"}},
	"to evade attack hits":                      {names: []string{"EvadeChance"}},
	"chance to evade attacks":                   {names: []string{"EvadeChance"}},
	"chance to evade attack hits":               {names: []string{"EvadeChance"}},
	"chance to evade projectile attacks":        {names: []string{"ProjectileEvadeChance"}},
	"chance to evade spells":                    {names: []string{"SpellEvadeChance"}},
	"chance to evade spell hits":                {names: []string{"SpellEvadeChance"}},
	"chance to evade melee attacks":             {names: []string{"MeleeEvadeChance"}},
	"evasion rating against melee attacks":      {names: []string{"MeleeEvasion"}},
	"evasion rating against projectile attacks": {names: []string{"ProjectileEvasion"}},
	"physical damage reduction":                 {names: []string{"PhysicalDamageReduction"}},
	"physical damage reduction from hits":       {names: []string{"PhysicalDamageReductionWhenHit"}},
	"elemental damage reduction":                {names: []string{"ElementalDamageReduction"}},
	"fire resistance":                           {names: []string{"FireResist"}},
	"maximum fire resistance":                   {names: []string{"FireResistMax"}},
	"maximum fire resistances":                  {names: []string{"FireResistMax"}},
	"cold resistance":                           {names: []string{"ColdResist"}},
	"maximum cold resistance":                   {names: []string{"ColdResistMax"}},
	"maximum cold resistances":                  {names: []string{"ColdResistMax"}},
	"lightning resistance":                      {names: []string{"LightningResist"}},
	"maximum lightning resistance":              {names: []string{"LightningResistMax"}},
	"maximum lightning resistances":             {names: []string{"LightningResistMax"}},
	"chaos resistance":                          {names: []string{"ChaosResist"}},
	"maximum chaos resistance":                  {names: []string{"ChaosResistMax"}},
	"fire and cold resistances":                 {names: []string{"FireResist", "ColdResist"}},
	"fire and lightning resistances":            {names: []string{"FireResist", "LightningResist"}},
	"cold and lightning resistances":            {names: []string{"ColdResist", "LightningResist"}},
	"elemental resistance":                      {names: []string{"ElementalResist"}},
	"elemental resistances":                     {names: []string{"ElementalResist"}},
	"all elemental resistances":                 {names: []string{"ElementalResist"}},
	"all resistances":                           {names: []string{"ElementalResist", "ChaosResist"}},
	"all maximum elemental resistances":         {names: []string{"ElementalResistMax"}},
	"all maximum resistances":                   {names: []string{"ElementalResistMax", "ChaosResistMax"}},
	"all elemental resistances and maximum elemental resistances": {names: []string{"ElementalResist", "ElementalResistMax"}},
	"fire and chaos resistances":                                  {names: []string{"FireResist", "ChaosResist"}},
	"cold and chaos resistances":                                  {names: []string{"ColdResist", "ChaosResist"}},
	"lightning and chaos resistances":                             {names: []string{"LightningResist", "ChaosResist"}},
	"damage taken":                                                {names: []string{"DamageTaken"}},
	"damage taken when hit":                                       {names: []string{"DamageTakenWhenHit"}},
	"damage taken from hits":                                      {names: []string{"DamageTakenWhenHit"}},
	"damage over time taken":                                      {names: []string{"DamageTakenOverTime"}},
	"damage taken from damage over time":                          {names: []string{"DamageTakenOverTime"}},
	"attack damage taken":                                         {names: []string{"AttackDamageTaken"}},
	"spell damage taken":                                          {names: []string{"SpellDamageTaken"}},
	"physical damage taken":                                       {names: []string{"PhysicalDamageTaken"}},
	"physical damage from hits taken":                             {names: []string{"PhysicalDamageFromHitsTaken"}},
	"physical damage taken when hit":                              {names: []string{"PhysicalDamageTakenWhenHit"}},
	"physical damage taken from hits":                             {names: []string{"PhysicalDamageTakenWhenHit"}},
	"physical damage taken from attacks":                          {names: []string{"PhysicalDamageTakenFromAttacks"}},
	"physical damage taken from attack hits":                      {names: []string{"PhysicalDamageTakenFromAttacks"}},
	"physical damage taken from projectile attacks":               {names: []string{"PhysicalDamageTakenFromProjectileAttacks"}},
	"physical damage taken from projectile attack hits":           {names: []string{"PhysicalDamageTakenFromProjectileAttacks"}},
	"fire, cold and lightning damage taken from spell hits":       {names: []string{"FireDamageTakenFromSpells", "ColdDamageTakenFromSpells", "LightningDamageTakenFromSpells"}},
	"physical damage taken over time":                             {names: []string{"PhysicalDamageTakenOverTime"}},
	"physical damage over time taken":                             {names: []string{"PhysicalDamageTakenOverTime"}},
	"physical damage over time damage taken":                      {names: []string{"PhysicalDamageTakenOverTime"}},
	"reflected physical damage taken":                             {names: []string{"PhysicalReflectedDamageTaken"}},
	"lightning damage taken":                                      {names: []string{"LightningDamageTaken"}},
	"lightning damage from hits taken":                            {names: []string{"LightningDamageFromHitsTaken"}},
	"lightning damage taken when hit":                             {names: []string{"LightningDamageTakenWhenHit"}},
	"lightning damage taken from attacks":                         {names: []string{"LightningDamageTakenFromAttacks"}},
	"lightning damage taken from attack hits":                     {names: []string{"LightningDamageTakenFromAttacks"}},
	"lightning damage taken over time":                            {names: []string{"LightningDamageTakenOverTime"}},
	"cold damage taken":                                           {names: []string{"ColdDamageTaken"}},
	"cold damage from hits taken":                                 {names: []string{"ColdDamageFromHitsTaken"}},
	"cold damage taken when hit":                                  {names: []string{"ColdDamageTakenWhenHit"}},
	"cold damage taken from hits":                                 {names: []string{"ColdDamageTakenWhenHit"}},
	"cold damage taken from attacks":                              {names: []string{"ColdDamageTakenFromAttacks"}},
	"cold damage taken from attack hits":                          {names: []string{"ColdDamageTakenFromAttacks"}},
	"cold damage taken over time":                                 {names: []string{"ColdDamageTakenOverTime"}},
	"fire damage taken":                                           {names: []string{"FireDamageTaken"}},
	"fire damage from hits taken":                                 {names: []string{"FireDamageFromHitsTaken"}},
	"fire damage taken when hit":                                  {names: []string{"FireDamageTakenWhenHit"}},
	"fire damage taken from hits":                                 {names: []string{"FireDamageTakenWhenHit"}},
	"fire damage taken from attacks":                              {names: []string{"FireDamageTakenFromAttacks"}},
	"fire damage taken from attack hits":                          {names: []string{"FireDamageTakenFromAttacks"}},
	"fire damage taken over time":                                 {names: []string{"FireDamageTakenOverTime"}},
	"chaos damage taken":                                          {names: []string{"ChaosDamageTaken"}},
	"chaos damage from hits taken":                                {names: []string{"ChaosDamageFromHitsTaken"}},
	"chaos damage taken when hit":                                 {names: []string{"ChaosDamageTakenWhenHit"}},
	"chaos damage taken from hits":                                {names: []string{"ChaosDamageTakenWhenHit"}},
	"chaos damage taken from attacks":                             {names: []string{"ChaosDamageTakenFromAttacks"}},
	"chaos damage taken from attack hits":                         {names: []string{"ChaosDamageTakenFromAttacks"}},
	"chaos damage taken over time":                                {names: []string{"ChaosDamageTakenOverTime"}},
	"chaos damage over time taken":                                {names: []string{"ChaosDamageTakenOverTime"}},
	"elemental damage taken":                                      {names: []string{"ElementalDamageTaken"}},
	"elemental damage from hits taken":                            {names: []string{"ElementalDamageFromHitsTaken"}},
	"elemental damage taken when hit":                             {names: []string{"ElementalDamageTakenWhenHit"}},
	"elemental damage taken from hits":                            {names: []string{"ElementalDamageTakenWhenHit"}},
	"elemental damage taken over time":                            {names: []string{"ElementalDamageTakenOverTime"}},
	"cold and lightning damage taken":                             {names: []string{"ColdDamageTaken", "LightningDamageTaken"}},
	"fire and lightning damage taken":                             {names: []string{"FireDamageTaken", "LightningDamageTaken"}},
	"fire and cold damage taken":                                  {names: []string{"FireDamageTaken", "ColdDamageTaken"}},
	"physical and chaos damage taken":                             {names: []string{"PhysicalDamageTaken", "ChaosDamageTaken"}},
	"reflected elemental damage taken":                            {names: []string{"ElementalReflectedDamageTaken"}},
	"armour also applies to elemental damage":                     {names: []string{"ArmourAppliesToFireDamageTaken", "ArmourAppliesToColdDamageTaken", "ArmourAppliesToLightningDamageTaken"}},
	"to dodge attacks":                                            {names: []string{"AttackDodgeChance"}},
	"to dodge attack hits":                                        {names: []string{"AttackDodgeChance"}},
	"to dodge spells":                                             {names: []string{"SpellDodgeChance"}},
	"to dodge spell hits":                                         {names: []string{"SpellDodgeChance"}},
	"to dodge spell damage":                                       {names: []string{"SpellDodgeChance"}},
	"to dodge attacks and spells":                                 {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to dodge attacks and spell damage":                           {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to dodge attack and spell hits":                              {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to dodge attack or spell hits":                               {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"deflection rating":                                           {names: []string{"DeflectionRating"}},
	"amount of damage prevented by deflection":                    {names: []string{"DeflectEffect"}},
	"to suppress spell damage":                                    {names: []string{"SpellSuppressionChance"}},
	"amount of suppressed spell damage prevented":                 {names: []string{"SpellSuppressionEffect"}},
	"to amount of suppressed spell damage prevented":              {names: []string{"SpellSuppressionEffect"}},
	"to block":                                                          {names: []string{"BlockChance"}},
	"to block attacks":                                                  {names: []string{"BlockChance"}},
	"to block attack damage":                                            {names: []string{"BlockChance"}},
	"block chance":                                                      {names: []string{"BlockChance"}},
	"chance to block":                                                   {names: []string{"BlockChance"}},
	"block chance with staves":                                          {names: []string{"BlockChance"}, tag: mod.Condition("UsingStaff")},
	"to block with staves":                                              {names: []string{"BlockChance"}, tag: mod.Condition("UsingStaff")},
	"block chance against projectiles":                                  {names: []string{"ProjectileBlockChance"}},
	"to block projectile attack damage":                                 {names: []string{"ProjectileBlockChance"}},
	"to block projectile spell damage":                                  {names: []string{"ProjectileSpellBlockChance"}},
	"spell block chance":                                                {names: []string{"SpellBlockChance"}},
	"to block spells":                                                   {names: []string{"SpellBlockChance"}},
	"to block spell damage":                                             {names: []string{"SpellBlockChance"}},
	"chance to block attacks and spells":                                {names: []string{"BlockChance", "SpellBlockChance"}},
	"chance to block attack and spell damage":                           {names: []string{"BlockChance", "SpellBlockChance"}},
	"to block attack and spell damage":                                  {names: []string{"BlockChance", "SpellBlockChance"}},
	"maximum block chance":                                              {names: []string{"BlockChanceMax"}},
	"maximum chance to block attack damage":                             {names: []string{"BlockChanceMax"}},
	"maximum chance to block spell damage":                              {names: []string{"SpellBlockChanceMax"}},
	"life gained when you block":                                        {names: []string{"LifeOnBlock"}},
	"mana gained when you block":                                        {names: []string{"ManaOnBlock"}},
	"energy shield when you block":                                      {names: []string{"EnergyShieldOnBlock"}},
	"maximum chance to dodge spell hits":                                {names: []string{"SpellDodgeChanceMax"}},
	"to avoid physical damage from hits":                                {names: []string{"AvoidPhysicalDamageChance"}},
	"to avoid chaos damage from hits":                                   {names: []string{"AvoidChaosDamageChance"}},
	"to avoid fire damage when hit":                                     {names: []string{"AvoidFireDamageChance"}},
	"to avoid fire damage from hits":                                    {names: []string{"AvoidFireDamageChance"}},
	"to avoid cold damage when hit":                                     {names: []string{"AvoidColdDamageChance"}},
	"to avoid cold damage from hits":                                    {names: []string{"AvoidColdDamageChance"}},
	"to avoid lightning damage when hit":                                {names: []string{"AvoidLightningDamageChance"}},
	"to avoid lightning damage from hits":                               {names: []string{"AvoidLightningDamageChance"}},
	"to avoid elemental damage when hit":                                {names: []string{"AvoidFireDamageChance", "AvoidColdDamageChance", "AvoidLightningDamageChance"}},
	"to avoid elemental damage from hits":                               {names: []string{"AvoidFireDamageChance", "AvoidColdDamageChance", "AvoidLightningDamageChance"}},
	"to avoid projectiles":                                              {names: []string{"AvoidProjectilesChance"}},
	"to avoid being stunned":                                            {names: []string{"AvoidStun"}},
	"to avoid interruption from stuns while casting":                    {names: []string{"AvoidInterruptStun"}},
	"to ignore stuns while casting":                                     {names: []string{"AvoidInterruptStun"}},
	"to avoid being shocked":                                            {names: []string{"AvoidShock"}},
	"to avoid being frozen":                                             {names: []string{"AvoidFreeze"}},
	"to avoid being chilled":                                            {names: []string{"AvoidChill"}},
	"to avoid being ignited":                                            {names: []string{"AvoidIgnite"}},
	"to avoid non-damaging ailments on you":                             {names: []string{"AvoidShock", "AvoidFreeze", "AvoidChill", "AvoidSap", "AvoidBrittle", "AvoidScorch"}},
	"to avoid blind":                                                    {names: []string{"AvoidBlind"}},
	"to avoid elemental ailments":                                       {names: []string{"AvoidElementalAilments"}},
	"to avoid elemental status ailments":                                {names: []string{"AvoidElementalAilments"}},
	"to avoid ailments":                                                 {names: []string{"AvoidAilments"}},
	"to avoid status ailments":                                          {names: []string{"AvoidAilments"}},
	"to avoid bleeding":                                                 {names: []string{"AvoidBleed"}},
	"to avoid being poisoned":                                           {names: []string{"AvoidPoison"}},
	"to avoid being impaled":                                            {names: []string{"AvoidImpale"}},
	"damage is taken from mana before life":                             {names: []string{"DamageTakenFromManaBeforeLife"}},
	"elemental damage is taken from mana before life":                   {names: []string{"LightningDamageTakenFromManaBeforeLife", "ColdDamageTakenFromManaBeforeLife", "FireDamageTakenFromManaBeforeLife"}},
	"lightning damage is taken from mana before life":                   {names: []string{"LightningDamageTakenFromManaBeforeLife"}},
	"damage taken from mana before life":                                {names: []string{"DamageTakenFromManaBeforeLife"}},
	"effect of curses on you":                                           {names: []string{"CurseEffectOnSelf"}},
	"effect of curses on them":                                          {names: []string{"CurseEffectOnSelf"}},
	"effect of exposure on you":                                         {names: []string{"ExposureEffectOnSelf"}},
	"effect of withered on you":                                         {names: []string{"WitherEffectOnSelf"}},
	"magnitude of abyssal wasting you inflict":                          {names: []string{"AbyssalWastingEffect"}},
	"life recovery rate":                                                {names: []string{"LifeRecoveryRate"}},
	"mana recovery rate":                                                {names: []string{"ManaRecoveryRate"}},
	"energy shield recovery rate":                                       {names: []string{"EnergyShieldRecoveryRate"}},
	"energy shield regeneration rate":                                   {names: []string{"EnergyShieldRegen"}},
	"recovery rate of life, mana and energy shield":                     {names: []string{"LifeRecoveryRate", "ManaRecoveryRate", "EnergyShieldRecoveryRate"}},
	"recovery rate of life and energy shield":                           {names: []string{"LifeRecoveryRate", "EnergyShieldRecoveryRate"}},
	"maximum life, mana and global energy shield":                       {names: []string{"Life", "Mana", "EnergyShield"}, tag: mod.Global()},
	"non-chaos damage taken bypasses energy shield":                     {names: []string{"PhysicalEnergyShieldBypass", "LightningEnergyShieldBypass", "ColdEnergyShieldBypass", "FireEnergyShieldBypass"}},
	"damage taken recouped as life":                                     {names: []string{"LifeRecoup"}},
	"physical damage taken recouped as life":                            {names: []string{"PhysicalLifeRecoup"}},
	"lightning damage taken recouped as life":                           {names: []string{"LightningLifeRecoup"}},
	"cold damage taken recouped as life":                                {names: []string{"ColdLifeRecoup"}},
	"fire damage taken recouped as life":                                {names: []string{"FireLifeRecoup"}},
	"chaos damage taken recouped as life":                               {names: []string{"ChaosLifeRecoup"}},
	"elemental damage taken recouped as life":                           {names: []string{"LightningLifeRecoup", "ColdLifeRecoup", "FireLifeRecoup"}},
	"damage taken recouped as energy shield":                            {names: []string{"EnergyShieldRecoup"}},
	"elemental damage taken recouped as energy shield":                  {names: []string{"LightningEnergyShieldRecoup", "ColdEnergyShieldRecoup", "FireEnergyShieldRecoup"}},
	"damage taken recouped as mana":                                     {names: []string{"ManaRecoup"}},
	"damage taken recouped as life, mana and energy shield":             {names: []string{"LifeRecoup", "EnergyShieldRecoup", "ManaRecoup"}},
	"ailment threshold":                                                 {names: []string{"AilmentThreshold"}},
	"elemental ailment threshold":                                       {names: []string{"AilmentThreshold"}},
	"stun recovery":                                                     {names: []string{"StunRecovery"}},
	"stun and block recovery":                                           {names: []string{"StunRecovery"}},
	"block and stun recovery":                                           {names: []string{"StunRecovery"}},
	"stun duration on you":                                              {names: []string{"StunDuration"}},
	"stun threshold":                                                    {names: []string{"StunThreshold"}},
	"ailment and stun threshold":                                        {names: []string{"StunThreshold", "AilmentThreshold"}},
	"block recovery":                                                    {names: []string{"BlockRecovery"}},
	"enemy stun threshold":                                              {names: []string{"EnemyStunThreshold"}},
	"stun duration":                                                     {names: []string{"EnemyStunDuration"}},
	"to double stun duration":                                           {names: []string{"DoubleEnemyStunDurationChance"}},
	"to knock enemies back on hit":                                      {names: []string{"EnemyKnockbackChance"}},
	"knockback distance":                                                {names: []string{"EnemyKnockbackDistance"}},
	"aura effect":                                                       {names: []string{"AuraEffect"}},
	"effect of auras from your aura skills":                             {names: []string{"AuraEffect"}, tag: mod.SkillType("Aura")},
	"effect of non-curse auras you cast":                                {names: []string{"AuraEffect"}, tagList: []mod.Tag{mod.SkillType("Aura"), mod.SkillType("AppliesCurse").Neg(true)}},
	"effect of non-curse auras from your skills":                        {names: []string{"AuraEffect"}, tagList: []mod.Tag{mod.SkillType("Aura"), mod.SkillType("AppliesCurse").Neg(true)}},
	"effect of non-curse auras from your skills on your minions":        {names: []string{"AuraEffectOnSelf"}, tagList: []mod.Tag{mod.SkillType("Aura"), mod.SkillType("AppliesCurse").Neg(true)}, addToMinion: true},
	"effect of non-curse auras":                                         {names: []string{"AuraEffect"}, tag: mod.SkillType("AppliesCurse").Neg(true)},
	"effect of your curses":                                             {names: []string{"CurseEffect"}},
	"effect of auras on you":                                            {names: []string{"AuraEffectOnSelf"}},
	"effect of auras on your minions":                                   {names: []string{"AuraEffectOnSelf"}, addToMinion: true},
	"effect of auras from mines":                                        {names: []string{"AuraEffect"}, keywordFlags: KeywordFlagMine},
	"effect of consecrated ground you create":                           {names: []string{"ConsecratedGroundEffect"}},
	"curse effect":                                                      {names: []string{"CurseEffect"}},
	"effect of curses applied by bane":                                  {names: []string{"CurseEffect"}, tag: mod.Condition("AppliedByBane")},
	"effect of your marks":                                              {names: []string{"MarkEffect"}, tag: mod.SkillType("Mark")},
	"effect of arcane surge on you":                                     {names: []string{"ArcaneSurgeEffect"}},
	"curse duration":                                                    {names: []string{"Duration"}, keywordFlags: KeywordFlagCurse},
	"hex duration":                                                      {names: []string{"Duration"}, tag: mod.SkillType("Hex")},
	"radius of auras":                                                   {names: []string{"AreaOfEffect"}, keywordFlags: KeywordFlagAura},
	"radius of curses":                                                  {names: []string{"AreaOfEffect"}, keywordFlags: KeywordFlagCurse},
	"area of effect of curses":                                          {names: []string{"AreaOfEffect"}, keywordFlags: KeywordFlagCurse},
	"buff effect":                                                       {names: []string{"BuffEffect"}},
	"effect of buffs on you":                                            {names: []string{"BuffEffectOnSelf"}},
	"effect of buffs granted by your golems":                            {names: []string{"BuffEffect"}, tag: mod.SkillType("Golem")},
	"effect of buffs granted by socketed golem skills":                  {names: []string{"BuffEffect"}, addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	"effect of the buff granted by your stone golems":                   {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Stone Golem").Transfigured(true)},
	"effect of the buff granted by your lightning golems":               {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Lightning Golem").Transfigured(true)},
	"effect of the buff granted by your ice golems":                     {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Ice Golem").Transfigured(true)},
	"effect of the buff granted by your flame golems":                   {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Flame Golem").Transfigured(true)},
	"effect of the buff granted by your chaos golems":                   {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Chaos Golem").Transfigured(true)},
	"effect of the buff granted by your carrion golems":                 {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Carrion Golem").Transfigured(true)},
	"effect of offering spells":                                         {names: []string{"BuffEffect"}, tag: mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering")},
	"effect of offerings":                                               {names: []string{"BuffEffect"}, tag: mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering")},
	"effect of heralds on you":                                          {names: []string{"BuffEffect"}, tag: mod.SkillType("Herald")},
	"effect of herald buffs on you":                                     {names: []string{"BuffEffect"}, tag: mod.SkillType("Herald")},
	"effect of buffs granted by your active ancestor totems":            {names: []string{"BuffEffect"}, tag: mod.SkillName("Ancestral Warchief", "Ancestral Protector", "Earthbreaker")},
	"effect of buffs your ancestor totems grant ":                       {names: []string{"BuffEffect"}, tag: mod.SkillName("Ancestral Warchief", "Ancestral Protector", "Earthbreaker")},
	"effect of shrine buffs on you":                                     {names: []string{"ShrineBuffEffect"}},
	"effect of withered":                                                {names: []string{"WitherEffect"}},
	"withered magnitude":                                                {names: []string{"WitherEffect"}},
	"effect of fully broken armour":                                     {names: []string{"FullyBrokenArmourEffect"}},
	"warcry effect":                                                     {names: []string{"BuffEffect"}, keywordFlags: KeywordFlagWarcry},
	"aspect of the avian buff effect":                                   {names: []string{"BuffEffect"}, tag: mod.SkillName("Aspect of the Avian")},
	"maximum rage":                                                      {names: []string{"MaximumRage"}},
	"minimum rage":                                                      {names: []string{"MinimumRage"}},
	"rage effect":                                                       {names: []string{"RageEffect"}},
	"maximum fortification":                                             {names: []string{"MaximumFortification"}},
	"fortification":                                                     {names: []string{"MinimumFortification"}},
	"maximum valour":                                                    {names: []string{"MaximumValour"}},
	"parried debuff magnitude":                                          {names: []string{"ParryDebuffMagnitude"}},
	"parried debuff duration":                                           {names: []string{"ParryDebuffDuration"}},
	"parry range":                                                       {names: []string{"ParryRangeNonProj", "ParryRangeProj"}},
	"maximum power charge":                                              {names: []string{"PowerChargesMax"}},
	"maximum power charges":                                             {names: []string{"PowerChargesMax"}},
	"minimum power charge":                                              {names: []string{"PowerChargesMin"}},
	"minimum power charges":                                             {names: []string{"PowerChargesMin"}},
	"power charge duration":                                             {names: []string{"PowerChargesDuration"}},
	"maximum frenzy charge":                                             {names: []string{"FrenzyChargesMax"}},
	"maximum frenzy charges":                                            {names: []string{"FrenzyChargesMax"}},
	"minimum frenzy charge":                                             {names: []string{"FrenzyChargesMin"}},
	"minimum frenzy charges":                                            {names: []string{"FrenzyChargesMin"}},
	"frenzy charge duration":                                            {names: []string{"FrenzyChargesDuration"}},
	"maximum endurance charge":                                          {names: []string{"EnduranceChargesMax"}},
	"maximum endurance charges":                                         {names: []string{"EnduranceChargesMax"}},
	"minimum endurance charge":                                          {names: []string{"EnduranceChargesMin"}},
	"minimum endurance charges":                                         {names: []string{"EnduranceChargesMin"}},
	"minimum endurance, frenzy and power charges":                       {names: []string{"PowerChargesMin", "FrenzyChargesMin", "EnduranceChargesMin"}},
	"endurance charge duration":                                         {names: []string{"EnduranceChargesDuration"}},
	"maximum frenzy charges and maximum power charges":                  {names: []string{"FrenzyChargesMax", "PowerChargesMax"}},
	"maximum power charges and maximum endurance charges":               {names: []string{"PowerChargesMax", "EnduranceChargesMax"}},
	"maximum endurance, frenzy and power charges":                       {names: []string{"EnduranceChargesMax", "PowerChargesMax", "FrenzyChargesMax"}},
	"endurance, frenzy and power charge duration":                       {names: []string{"PowerChargesDuration", "FrenzyChargesDuration", "EnduranceChargesDuration"}},
	"maximum siphoning charge":                                          {names: []string{"SiphoningChargesMax"}},
	"maximum siphoning charges":                                         {names: []string{"SiphoningChargesMax"}},
	"maximum challenger charges":                                        {names: []string{"ChallengerChargesMax"}},
	"maximum blitz charges":                                             {names: []string{"BlitzChargesMax"}},
	"maximum number of crab barriers":                                   {names: []string{"CrabBarriersMax"}},
	"maximum blood charges":                                             {names: []string{"BloodChargesMax"}},
	"maximum spirit charges":                                            {names: []string{"SpiritChargesMax"}},
	"charge duration":                                                   {names: []string{"ChargeDuration"}},
	"life gained on kill":                                               {names: []string{"LifeOnKill"}},
	"life per enemy killed":                                             {names: []string{"LifeOnKill"}},
	"life on kill":                                                      {names: []string{"LifeOnKill"}},
	"life per enemy hit":                                                {names: []string{"LifeOnHit"}, flags: ModFlagHit},
	"life gained for each enemy hit":                                    {names: []string{"LifeOnHit"}, flags: ModFlagHit},
	"life for each enemy hit":                                           {names: []string{"LifeOnHit"}, flags: ModFlagHit},
	"mana gained on kill":                                               {names: []string{"ManaOnKill"}},
	"mana per enemy killed":                                             {names: []string{"ManaOnKill"}},
	"mana on kill":                                                      {names: []string{"ManaOnKill"}},
	"mana per enemy hit":                                                {names: []string{"ManaOnHit"}, flags: ModFlagHit},
	"mana gained for each enemy hit":                                    {names: []string{"ManaOnHit"}, flags: ModFlagHit},
	"mana for each enemy hit":                                           {names: []string{"ManaOnHit"}, flags: ModFlagHit},
	"energy shield gained on kill":                                      {names: []string{"EnergyShieldOnKill"}},
	"energy shield per enemy killed":                                    {names: []string{"EnergyShieldOnKill"}},
	"energy shield on kill":                                             {names: []string{"EnergyShieldOnKill"}},
	"energy shield per enemy hit":                                       {names: []string{"EnergyShieldOnHit"}, flags: ModFlagHit},
	"energy shield gained for each enemy hit":                           {names: []string{"EnergyShieldOnHit"}, flags: ModFlagHit},
	"energy shield for each enemy hit":                                  {names: []string{"EnergyShieldOnHit"}, flags: ModFlagHit},
	"life and mana gained for each enemy hit":                           {names: []string{"LifeOnHit", "ManaOnHit"}, flags: ModFlagHit},
	"life and mana for each enemy hit":                                  {names: []string{"LifeOnHit", "ManaOnHit"}, flags: ModFlagHit},
	"damage as life":                                                    {names: []string{"DamageLifeLeech"}},
	"life leeched per second":                                           {names: []string{"LifeLeechRate"}},
	"mana leeched per second":                                           {names: []string{"ManaLeechRate"}},
	"total recovery per second from life leech":                         {names: []string{"LifeLeechRate"}},
	"recovery per second from life leech":                               {names: []string{"LifeLeechRate"}},
	"total recovery per second from energy shield leech":                {names: []string{"EnergyShieldLeechRate"}},
	"recovery per second from energy shield leech":                      {names: []string{"EnergyShieldLeechRate"}},
	"total recovery per second from mana leech":                         {names: []string{"ManaLeechRate"}},
	"recovery per second from mana leech":                               {names: []string{"ManaLeechRate"}},
	"total recovery per second from life, mana, or energy shield leech": {names: []string{"LifeLeechRate", "ManaLeechRate", "EnergyShieldLeechRate"}},
	"maximum recovery per life leech":                                   {names: []string{"MaxLifeLeechInstance"}},
	"maximum recovery per energy shield leech":                          {names: []string{"MaxEnergyShieldLeechInstance"}},
	"maximum recovery per mana leech":                                   {names: []string{"MaxManaLeechInstance"}},
	"maximum total recovery per second from life leech":                 {names: []string{"MaxLifeLeechRate"}},
	"maximum total life recovery per second from leech":                 {names: []string{"MaxLifeLeechRate"}},
	"maximum total recovery per second from energy shield leech":        {names: []string{"MaxEnergyShieldLeechRate"}},
	"maximum total energy shield recovery per second from leech":        {names: []string{"MaxEnergyShieldLeechRate"}},
	"maximum total recovery per second from mana leech":                 {names: []string{"MaxManaLeechRate"}},
	"maximum total mana recovery per second from leech":                 {names: []string{"MaxManaLeechRate"}},
	"maximum total life, mana and energy shield recovery per second from leech": {names: []string{"MaxLifeLeechRate", "MaxManaLeechRate", "MaxEnergyShieldLeechRate"}},
	"life and mana leech is instant":                                            {names: []string{"InstantManaLeech", "InstantLifeLeech"}},
	"life leech is instant":                                                     {names: []string{"InstantLifeLeech"}},
	"mana leech is instant":                                                     {names: []string{"InstantManaLeech"}},
	"amount of mana leeched":                                                    {names: []string{"MaxManaLeechRate"}},
	"amount of life leeched":                                                    {names: []string{"MaxLifeLeechRate"}},
	"energy shield leech is instant":                                            {names: []string{"InstantEnergyShieldLeech"}},
	"leech is instant":                                                          {names: []string{"InstantEnergyShieldLeech", "InstantManaLeech", "InstantLifeLeech"}},
	"to impale enemies on hit":                                                  {names: []string{"ImpaleChance"}},
	"to impale on spell hit":                                                    {names: []string{"ImpaleChance"}, flags: ModFlagSpell},
	"impale effect":                                                             {names: []string{"ImpaleEffect"}},
	"effect of impales you inflict":                                             {names: []string{"ImpaleEffect"}},
	"effects of impale inflicted":                                               {names: []string{"ImpaleEffect"}},
	"effect of impales inflicted":                                               {names: []string{"ImpaleEffect"}},
	"magnitude of impales inflicted":                                            {names: []string{"ImpaleEffect"}},
	"projectile":                                                                {names: []string{"ProjectileCount"}},
	"projectiles":                                                               {names: []string{"ProjectileCount"}},
	"arrow":                                                                     {names: []string{"ProjectileCount"}, keywordFlags: KeywordFlagArrow},
	"arrows":                                                                    {names: []string{"ProjectileCount"}, keywordFlags: KeywordFlagArrow},
	"projectile speed":                                                          {names: []string{"ProjectileSpeed"}},
	"arrow speed":                                                               {names: []string{"ProjectileSpeed"}, keywordFlags: KeywordFlagArrow},
	"bolt speed":                                                                {names: []string{"ProjectileSpeed"}, flags: ModFlagCrossbow},
	"totem placement speed":                                                     {names: []string{"TotemPlacementSpeed"}},
	"totem life":                                                                {names: []string{"TotemLife"}},
	"totem duration":                                                            {names: []string{"TotemDuration"}},
	"maximum number of summoned totems":                                         {names: []string{"ActiveTotemLimit"}},
	`maximum number of summoned totems\.`:                                       {names: []string{"ActiveTotemLimit"}},
	"maximum number of summoned ballista totems":                                {names: []string{"ActiveBallistaLimit"}, tag: mod.SkillType("TotemsAreBallistae")},
	"trap throwing speed":                                                       {names: []string{"TrapThrowingSpeed"}},
	"trap and mine throwing speed":                                              {names: []string{"TrapThrowingSpeed", "MineLayingSpeed"}},
	"trap trigger area of effect":                                               {names: []string{"TrapTriggerAreaOfEffect"}},
	"trap duration":                                                             {names: []string{"TrapDuration"}},
	"cooldown recovery speed for throwing traps":                                {names: []string{"CooldownRecovery"}, keywordFlags: KeywordFlagTrap},
	"cooldown recovery rate for throwing traps":                                 {names: []string{"CooldownRecovery"}, keywordFlags: KeywordFlagTrap},
	"additional trap":                                                           {names: []string{"TrapThrowCount"}},
	"additional traps":                                                          {names: []string{"TrapThrowCount"}},
	"mine laying speed":                                                         {names: []string{"MineLayingSpeed"}},
	"mine throwing speed":                                                       {names: []string{"MineLayingSpeed"}},
	"mine detonation area of effect":                                            {names: []string{"MineDetonationAreaOfEffect"}},
	"mine duration":                                                             {names: []string{"MineDuration"}},
	"additional mine":                                                           {names: []string{"MineThrowCount"}},
	"additional mines":                                                          {names: []string{"MineThrowCount"}},
	"activation frequency":                                                      {names: []string{"BrandActivationFrequency"}},
	"brand activation frequency":                                                {names: []string{"BrandActivationFrequency"}},
	"brand attachment range":                                                    {names: []string{"BrandAttachmentRange"}},
	"maximum number of skeletons":                                               {names: []string{"ActiveSkeletonLimit"}},
	"maximum number of zombies":                                                 {names: []string{"ActiveZombieLimit"}},
	"maximum number of raised zombies":                                          {names: []string{"ActiveZombieLimit"}},
	"number of zombies allowed":                                                 {names: []string{"ActiveZombieLimit"}},
	"maximum number of spectres":                                                {names: []string{"ActiveSpectreLimit"}},
	"maximum number of golems":                                                  {names: []string{"ActiveGolemLimit"}},
	"maximum number of summoned golems":                                         {names: []string{"ActiveGolemLimit"}},
	"maximum number of summoned raging spirits":                                 {names: []string{"ActiveRagingSpiritLimit"}},
	"maximum number of raging spirits":                                          {names: []string{"ActiveRagingSpiritLimit"}},
	"maximum number of summoned phantasms":                                      {names: []string{"ActivePhantasmLimit"}},
	"maximum number of summoned holy relics":                                    {names: []string{"ActiveHolyRelicLimit"}},
	"number of summoned arbalists":                                              {names: []string{"ActiveArbalistLimit"}},
	"minion duration":                                                           {names: []string{"Duration"}, tag: mod.SkillType("CreatesMinion")},
	"skeleton duration":                                                         {names: []string{"Duration"}, tag: mod.SkillName("Summon Skeletons").Transfigured(true)},
	"sentinel of dominance duration":                                            {names: []string{"Duration"}, tag: mod.SkillName("Dominating Blow").Transfigured(true)},
	"radius":                                                                    {names: []string{"AreaOfEffect"}},
	"radius of area skills":                                                     {names: []string{"AreaOfEffect"}},
	"area of effect radius":                                                     {names: []string{"AreaOfEffect"}},
	"area of effect":                                                            {names: []string{"AreaOfEffect"}},
	"area of effect of skills":                                                  {names: []string{"AreaOfEffect"}},
	"area of effect of area skills":                                             {names: []string{"AreaOfEffect"}},
	"aspect of the spider area of effect":                                       {names: []string{"AreaOfEffect"}, tag: mod.SkillName("Aspect of the Spider")},
	"firestorm explosion area of effect":                                        {names: []string{"AreaOfEffectSecondary"}, tag: mod.SkillName("Firestorm").Transfigured(true)},
	"presence area of effect":                                                   {names: []string{"PresenceArea"}},
	"presence radius":                                                           {names: []string{"PresenceRadius"}},
	"surrounded area of effect":                                                 {names: []string{"SurroundedArea"}},
	"duration":                                                                  {names: []string{"Duration"}},
	"skill effect duration":                                                     {names: []string{"Duration"}},
	"fuse duration":                                                             {names: []string{"Duration"}},
	"detonation time":                                                           {names: []string{"DetonationTime"}},
	"grenade detonation time":                                                   {names: []string{"DetonationTime"}, tag: mod.SkillType("Grenade")},
	"chaos skill effect duration":                                               {names: []string{"Duration"}, keywordFlags: KeywordFlagChaos},
	"soul gain prevention duration":                                             {names: []string{"SoulGainPreventionDuration"}},
	"aspect of the spider debuff duration":                                      {names: []string{"Duration"}, tag: mod.SkillName("Aspect of the Spider")},
	"fire trap burning ground duration":                                         {names: []string{"Duration"}, tag: mod.SkillName("Fire Trap")},
	"sentinel of absolution duration":                                           {names: []string{"SecondaryDuration"}, tag: mod.SkillName("Absolution").Transfigured(true)},
	"cooldown recovery":                                                         {names: []string{"CooldownRecovery"}},
	"cooldown recovery speed":                                                   {names: []string{"CooldownRecovery"}},
	"cooldown recovery rate":                                                    {names: []string{"CooldownRecovery"}},
	"cooldown use":                                                              {names: []string{"AdditionalCooldownUses"}},
	"cooldown uses":                                                             {names: []string{"AdditionalCooldownUses"}},
	"weapon range":                                                              {names: []string{"WeaponRange"}},
	"metres to weapon range":                                                    {names: []string{"WeaponRangeMetre"}},
	"metre to weapon range":                                                     {names: []string{"WeaponRangeMetre"}},
	"melee range":                                                               {names: []string{"MeleeWeaponRange"}},
	"melee weapon range":                                                        {names: []string{"MeleeWeaponRange"}},
	"melee weapon and unarmed range":                                            {names: []string{"MeleeWeaponRange", "UnarmedRange"}},
	"melee weapon and unarmed attack range":                                     {names: []string{"MeleeWeaponRange", "UnarmedRange"}},
	"melee strike range":                                                        {names: []string{"MeleeWeaponRange", "UnarmedRange"}},
	"melee strike range with this weapon":                                       {names: []string{"MeleeWeaponRange"}},
	"metres to melee strike range":                                              {names: []string{"MeleeWeaponRangeMetre", "UnarmedRangeMetre"}},
	"metre to melee strike range":                                               {names: []string{"MeleeWeaponRangeMetre", "UnarmedRangeMetre"}},
	"to deal double damage":                                                     {names: []string{"DoubleDamageChance"}},
	"to deal triple damage":                                                     {names: []string{"TripleDamageChance"}},
	"curse activation":                                                          {names: []string{"CurseActivation"}},
	"ice crystal life":                                                          {names: []string{"IceCrystalLife"}},
	"onslaught effect":                                                          {names: []string{"OnslaughtEffect"}},
	"effect of onslaught on you":                                                {names: []string{"OnslaughtEffect"}},
	"adrenaline duration":                                                       {names: []string{"AdrenalineDuration"}},
	"effect of tailwind on you":                                                 {names: []string{"TailwindEffectOnSelf"}},
	"elusive effect":                                                            {names: []string{"ElusiveEffect"}},
	"effect of elusive on you":                                                  {names: []string{"ElusiveEffect"}},
	"effect of infusion":                                                        {names: []string{"InfusionEffect"}},
	"effect of incision":                                                        {names: []string{"IncisionEffect"}},
	"fire exposure effect":                                                      {names: []string{"FireExposureEffect"}},
	"cold exposure effect":                                                      {names: []string{"ColdExposureEffect"}},
	"lightning exposure effect":                                                 {names: []string{"LightningExposureEffect"}},
	"exposure effect":                                                           {names: []string{"FireExposureEffect", "ColdExposureEffect", "LightningExposureEffect"}},
	"damage":                                                                    {names: []string{"Damage"}},
	"physical damage":                                                           {names: []string{"PhysicalDamage"}},
	"lightning damage":                                                          {names: []string{"LightningDamage"}},
	"cold damage":                                                               {names: []string{"ColdDamage"}},
	"fire damage":                                                               {names: []string{"FireDamage"}},
	"chaos damage":                                                              {names: []string{"ChaosDamage"}},
	"non-chaos damage":                                                          {names: []string{"NonChaosDamage"}},
	"elemental damage":                                                          {names: []string{"ElementalDamage"}},
	"attack damage":                                                             {names: []string{"Damage"}, flags: ModFlagAttack},
	"attack physical damage":                                                    {names: []string{"PhysicalDamage"}, flags: ModFlagAttack},
	"attack lightning damage":                                                   {names: []string{"LightningDamage"}, flags: ModFlagAttack},
	"attack cold damage":                                                        {names: []string{"ColdDamage"}, flags: ModFlagAttack},
	"attack fire damage":                                                        {names: []string{"FireDamage"}, flags: ModFlagAttack},
	"attack chaos damage":                                                       {names: []string{"ChaosDamage"}, flags: ModFlagAttack},
	"physical attack damage":                                                    {names: []string{"PhysicalDamage"}, flags: ModFlagAttack},
	"minimum physical attack damage":                                            {names: []string{"MinPhysicalDamage"}, tag: mod.SkillType("Attack")},
	"maximum physical attack damage":                                            {names: []string{"MaxPhysicalDamage"}, tag: mod.SkillType("Attack")},
	"physical weapon damage":                                                    {names: []string{"PhysicalDamage"}, flags: ModFlagWeapon},
	"physical damage with weapons":                                              {names: []string{"PhysicalDamage"}, flags: ModFlagWeapon},
	"melee damage":                                                              {names: []string{"Damage"}, flags: ModFlagMelee},
	"physical melee damage":                                                     {names: []string{"PhysicalDamage"}, flags: ModFlagMelee},
	"melee physical damage":                                                     {names: []string{"PhysicalDamage"}, flags: ModFlagMelee},
	"maximum lightning damage":                                                  {names: []string{"MaxLightningDamage"}},
	"projectile damage":                                                         {names: []string{"Damage"}, flags: ModFlagProjectile},
	"projectile attack damage":                                                  {names: []string{"Damage"}, flags: ModFlagProjectile | ModFlagAttack},
	"attack area damage":                                                        {names: []string{"Damage"}, flags: ModFlagArea | ModFlagAttack},
	"spell area damage":                                                         {names: []string{"Damage"}, flags: ModFlagArea | ModFlagSpell},
	"bow damage":                                                                {names: []string{"Damage"}, flags: ModFlagBow | ModFlagHit},
	"damage with arrow hits":                                                    {names: []string{"Damage"}, flags: ModFlagHit, keywordFlags: KeywordFlagArrow},
	"wand damage":                                                               {names: []string{"Damage"}, flags: ModFlagWand | ModFlagHit},
	"wand physical damage":                                                      {names: []string{"PhysicalDamage"}, flags: ModFlagWand | ModFlagHit},
	"claw physical damage":                                                      {names: []string{"PhysicalDamage"}, flags: ModFlagClaw | ModFlagHit},
	"sword physical damage":                                                     {names: []string{"PhysicalDamage"}, flags: ModFlagSword | ModFlagHit},
	"damage over time":                                                          {names: []string{"Damage"}, flags: ModFlagDot},
	"physical damage over time":                                                 {names: []string{"PhysicalDamage"}, keywordFlags: KeywordFlagPhysicalDot},
	"cold damage over time":                                                     {names: []string{"ColdDamage"}, keywordFlags: KeywordFlagColdDot},
	"chaos damage over time":                                                    {names: []string{"ChaosDamage"}, keywordFlags: KeywordFlagChaosDot},
	"burning damage":                                                            {names: []string{"FireDamage"}, keywordFlags: KeywordFlagFireDot},
	"damage with ignite":                                                        {names: []string{"Damage"}, keywordFlags: KeywordFlagIgnite},
	"damage with ignites":                                                       {names: []string{"Damage"}, keywordFlags: KeywordFlagIgnite},
	"damage with ignites inflicted":                                             {names: []string{"Damage"}, keywordFlags: KeywordFlagIgnite},
	"damage with bleed":                                                         {names: []string{"Damage"}, keywordFlags: KeywordFlagBleed},
	"damage with bleeding":                                                      {names: []string{"Damage"}, keywordFlags: KeywordFlagBleed},
	"damage with poison":                                                        {names: []string{"Damage"}, keywordFlags: KeywordFlagPoison},
	"thorns damage":                                                             {names: []string{"Damage"}, flags: ModFlagThorns},
	"incinerate damage for each stage":                                          {names: []string{"Damage"}, tagList: []mod.Tag{mod.Multiplier("IncinerateStage"), mod.SkillName("Incinerate")}},
	"physical damage over time multiplier":                                      {names: []string{"PhysicalDotMultiplier"}},
	"fire damage over time multiplier":                                          {names: []string{"FireDotMultiplier"}},
	"cold damage over time multiplier":                                          {names: []string{"ColdDotMultiplier"}},
	"chaos damage over time multiplier":                                         {names: []string{"ChaosDotMultiplier"}},
	"damage over time multiplier":                                               {names: []string{"DotMultiplier"}},
	"critical hit chance":                                                       {names: []string{"CritChance"}},
	"attack critical hit chance":                                                {names: []string{"CritChance"}, flags: ModFlagAttack},
	"thorns critical hit chance":                                                {names: []string{"CritChance"}, flags: ModFlagThorns},
	"critical damage bonus":                                                     {names: []string{"CritMultiplier"}},
	"attack critical damage bonus":                                              {names: []string{"CritMultiplier"}, flags: ModFlagAttack},
	"critical spell damage bonus":                                               {names: []string{"CritMultiplier"}, flags: ModFlagSpell},
	"thorns critical damage bonus":                                              {names: []string{"CritMultiplier"}, flags: ModFlagThorns},
	"accuracy":                                                                  {names: []string{"Accuracy"}},
	"accuracy rating":                                                           {names: []string{"Accuracy"}},
	"minion accuracy rating":                                                    {names: []string{"Accuracy"}, addToMinion: true},
	"attack speed":                                                              {names: []string{"Speed"}, flags: ModFlagAttack},
	"cast speed":                                                                {names: []string{"Speed"}, flags: ModFlagCast},
	"warcry speed":                                                              {names: []string{"WarcrySpeed"}, keywordFlags: KeywordFlagWarcry},
	"attack and cast speed":                                                     {names: []string{"Speed"}},
	"use speed":                                                                 {names: []string{"Speed"}},
	"skill speed":                                                               {names: []string{"Speed", "WarcrySpeed", "TotemPlacementSpeed"}},
	"reload speed":                                                              {names: []string{"ReloadSpeed"}, flags: ModFlagAttack},
	"dps":                                                                       {names: []string{"DPS"}},
	"to shock":                                                                  {names: []string{"EnemyShockChance"}},
	"shock chance":                                                              {names: []string{"EnemyShockChance"}},
	"chance to shock":                                                           {names: []string{"EnemyShockChance"}},
	"to freeze":                                                                 {names: []string{"EnemyFreezeChance"}},
	"freeze chance":                                                             {names: []string{"EnemyFreezeChance"}},
	"chance to freeze":                                                          {names: []string{"EnemyFreezeChance"}},
	"to ignite":                                                                 {names: []string{"EnemyIgniteChance"}},
	"ignite chance":                                                             {names: []string{"EnemyIgniteChance"}},
	"chance to ignite":                                                          {names: []string{"EnemyIgniteChance"}},
	"to freeze, shock and ignite":                                               {names: []string{"EnemyFreezeChance", "EnemyShockChance", "EnemyIgniteChance"}},
	"flammability magnitude":                                                    {names: []string{"EnemyIgniteChance"}},
	"magnitude of shock":                                                        {names: []string{"EnemyShockMagnitude"}},
	"magnitude of shock you inflict":                                            {names: []string{"EnemyShockMagnitude"}},
	"magnitude of chill":                                                        {names: []string{"EnemyChillMagnitude"}},
	"magnitude of chill you inflict":                                            {names: []string{"EnemyChillMagnitude"}},
	"magnitude of jagged ground you create":                                     {names: []string{"EnemyJaggedGroundMagnitude"}},
	"magnitude of bleeding":                                                     {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagBleed},
	"magnitude of bleeding you inflict":                                         {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagBleed},
	"magnitude of ignite":                                                       {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagIgnite},
	`magnitude of ignite \[you inflict\]`:                                       {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagIgnite},
	"ignite magnitude":                                                          {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagIgnite},
	"bleed magnitude":                                                           {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagBleed},
	"magnitude of poison":                                                       {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagPoison},
	"magnitude of poisons":                                                      {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagPoison},
	"magnitude of poison you inflict":                                           {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagPoison},
	"magnitude of poisons you inflict":                                          {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagPoison},
	"effect of poison you inflict":                                              {names: []string{"AilmentEffect"}, keywordFlags: KeywordFlagPoison},
	"effect of poisons you inflict":                                             {names: []string{"AilmentEffect"}, keywordFlags: KeywordFlagPoison},
	"magnitude of ailments":                                                     {names: []string{"AilmentMagnitude"}},
	"magnitude of ailments you inflict":                                         {names: []string{"AilmentMagnitude"}},
	"magnitude of damaging ailments you inflict":                                {names: []string{"AilmentMagnitude"}, keywordFlags: KeywordFlagPoison | KeywordFlagBleed | KeywordFlagIgnite},
	"effect of lightning ailments":                                              {names: []string{"EnemyShockMagnitude"}},
	"effect of chill and shock on you":                                          {names: []string{"SelfChillEffect", "SelfShockEffect"}},
	"chill effect":                                                              {names: []string{"EnemyChillMagnitude"}},
	"effect of cold ailments":                                                   {names: []string{"EnemyChillMagnitude"}},
	"effect of chill on you":                                                    {names: []string{"SelfChillEffect"}},
	"effect of freeze on you":                                                   {names: []string{"SelfFreezeEffect"}},
	"effect of shock on you":                                                    {names: []string{"SelfShockEffect"}},
	"effect of non-damaging ailments on you":                                    {names: []string{"SelfChillEffect", "SelfFreezeEffect", "SelfShockEffect"}},
	"effect of non-damaging ailments":                                           {names: []string{"EnemyShockMagnitude", "EnemyChillMagnitude", "EnemyFreezeBuildup"}},
	"effect of non-damaging ailments you inflict":                               {names: []string{"EnemyShockMagnitude", "EnemyChillMagnitude", "EnemyFreezeBuildup"}},
	"magnitude of non-damaging ailments you inflict":                            {names: []string{"EnemyShockMagnitude", "EnemyChillMagnitude", "EnemyFreezeBuildup"}},
	"shock duration":                                                            {names: []string{"EnemyShockDuration"}},
	"duration of shocks you inflict":                                            {names: []string{"EnemyShockDuration"}},
	"shock duration on you":                                                     {names: []string{"SelfShockDuration"}},
	"duration of lightning ailments":                                            {names: []string{"EnemyShockDuration", "EnemySapDuration"}},
	"freeze duration":                                                           {names: []string{"EnemyFreezeDuration"}},
	"duration of freezes you inflict":                                           {names: []string{"EnemyFreezeDuration"}},
	"freeze duration on you":                                                    {names: []string{"SelfFreezeDuration"}},
	"freeze threshold":                                                          {names: []string{"FreezeThreshold"}},
	"chill duration":                                                            {names: []string{"EnemyChillDuration"}},
	"duration of chills you inflict":                                            {names: []string{"EnemyChillDuration"}},
	"chill duration on you":                                                     {names: []string{"SelfChillDuration"}},
	"duration of cold ailments":                                                 {names: []string{"EnemyFreezeDuration", "EnemyChillDuration", "EnemyBrittleDuration"}},
	"chill and freeze duration":                                                 {names: []string{"EnemyFreezeDuration", "EnemyChillDuration"}},
	"ignite duration":                                                           {names: []string{"EnemyIgniteDuration"}},
	"duration of ignites you inflict":                                           {names: []string{"EnemyIgniteDuration"}},
	"ignite duration on you":                                                    {names: []string{"SelfIgniteDuration"}},
	"duration of ignite on you":                                                 {names: []string{"SelfIgniteDuration"}},
	"effect of ignite on you":                                                   {names: []string{"SelfIgniteEffect"}},
	"magnitude of ignite on you":                                                {names: []string{"SelfIgniteEffect"}},
	"duration of elemental ailments":                                            {names: []string{"EnemyElementalAilmentDuration"}},
	"duration of elemental ailments on you":                                     {names: []string{"SelfElementalAilmentDuration"}},
	"duration of elemental status ailments":                                     {names: []string{"EnemyElementalAilmentDuration"}},
	"duration of ailments":                                                      {names: []string{"EnemyAilmentDuration"}},
	"duration of ailments on you":                                               {names: []string{"SelfAilmentDuration"}},
	"elemental ailment duration on you":                                         {names: []string{"SelfElementalAilmentDuration"}},
	"duration of ailments you inflict":                                          {names: []string{"EnemyAilmentDuration"}},
	"duration of ailments inflicted":                                            {names: []string{"EnemyAilmentDuration"}},
	"duration of ailments inflicted on you":                                     {names: []string{"SelfAilmentDuration"}},
	"duration of damaging ailments":                                             {names: []string{"EnemyIgniteDuration", "EnemyBleedDuration", "EnemyPoisonDuration"}},
	"duration of damaging ailments on you":                                      {names: []string{"SelfIgniteDuration", "SelfBleedDuration", "SelfPoisonDuration"}},
	"duration of ignite, shock and chill":                                       {names: []string{"EnemyShockDuration", "EnemyChillDuration", "EnemyIgniteDuration"}},
	"chance to inflict ailments":                                                {names: []string{"AilmentChance"}},
	"chance to inflict elemental ailments":                                      {names: []string{"EnemyIgniteChance", "EnemyShockChance"}},
	"to poison":                                                                 {names: []string{"PoisonChance"}},
	"to cause poison":                                                           {names: []string{"PoisonChance"}},
	"to poison on hit":                                                          {names: []string{"PoisonChance"}},
	"poison duration":                                                           {names: []string{"EnemyPoisonDuration"}},
	"poison duration on you":                                                    {names: []string{"SelfPoisonDuration"}},
	"duration of poisons on you":                                                {names: []string{"SelfPoisonDuration"}},
	"duration of poisons you inflict":                                           {names: []string{"EnemyPoisonDuration"}},
	"to cause bleeding":                                                         {names: []string{"BleedChance"}},
	"to cause bleeding on hit":                                                  {names: []string{"BleedChance"}},
	"to inflict bleeding":                                                       {names: []string{"BleedChance"}},
	"to inflict bleeding on hit":                                                {names: []string{"BleedChance"}},
	"to bleed":                                                                  {names: []string{"BleedChance"}},
	"bleed duration":                                                            {names: []string{"EnemyBleedDuration"}},
	"bleeding duration":                                                         {names: []string{"EnemyBleedDuration"}},
	"bleed duration on you":                                                     {names: []string{"SelfBleedDuration"}},
	"bleeding duration on you":                                                  {names: []string{"SelfBleedDuration"}},
	"duration of bleeding on you":                                               {names: []string{"SelfBleedDuration"}},
	"magnitude of bleeding on you":                                              {names: []string{"SelfBleedEffect"}},
	"to blind enemies on hit":                                                   {names: []string{"BlindChance"}},
	"curse magnitudes":                                                          {names: []string{"CurseEffect"}},
	"aura magnitudes":                                                           {names: []string{"AuraEffect"}},
	"to daze":                                                                   {names: []string{"DazeChance"}},
	"to inflict daze":                                                           {names: []string{"DazeChance"}},
	"electrocute buildup":                                                       {names: []string{"EnemyElectrocuteBuildup"}},
	"stun buildup":                                                              {names: []string{"EnemyHeavyStunBuildup"}},
	"projectile stun buildup":                                                   {names: []string{"EnemyHeavyStunBuildup"}, flags: ModFlagProjectile},
	"freeze buildup":                                                            {names: []string{"EnemyFreezeBuildup"}},
	"pin buildup":                                                               {names: []string{"EnemyPinBuildup"}},
	"immobilisation buildup":                                                    {names: []string{"EnemyImmobilisationBuildup"}},
	"movement speed":                                                            {names: []string{"MovementSpeed"}},
	"attack, cast and movement speed":                                           {names: []string{"Speed", "MovementSpeed"}},
	"action speed":                                                              {names: []string{"ActionSpeed"}},
	"light radius":                                                              {names: []string{"LightRadius"}},
	"rarity of items found":                                                     {names: []string{"LootRarity"}},
	"rarity of items dropped":                                                   {names: []string{"LootRarity"}},
	"quantity of items found":                                                   {names: []string{"LootQuantity"}},
	"quantity of items dropped":                                                 {names: []string{"LootQuantity"}},
	"item quantity":                                                             {names: []string{"LootQuantity"}},
	"strength requirement":                                                      {names: []string{"StrRequirement"}},
	"dexterity requirement":                                                     {names: []string{"DexRequirement"}},
	"intelligence requirement":                                                  {names: []string{"IntRequirement"}},
	"strength and intelligence requirement":                                     {names: []string{"StrRequirement", "IntRequirement"}},
	"attribute requirements":                                                    {names: []string{"StrRequirement", "DexRequirement", "IntRequirement"}},
	"effect of socketed jewels":                                                 {names: []string{"SocketedJewelEffect"}},
	"effect of the socketed jewel":                                              {names: []string{"SocketedJewelEffect"}},
	"effect of socketed abyss jewels":                                           {names: []string{"SocketedJewelEffect"}},
	"effect of socketed soul cores":                                             {names: []string{"SocketedSoulCoreEffect"}},
	"effect of socketed runes":                                                  {names: []string{"SocketedRuneEffect"}},
	"effect of socketed augment items":                                          {names: []string{"SocketedAugmentItemEffect"}},
	"to inflict fire exposure on hit":                                           {names: []string{"FireExposureChance"}},
	"to apply fire exposure on hit":                                             {names: []string{"FireExposureChance"}},
	"to inflict cold exposure on hit":                                           {names: []string{"ColdExposureChance"}},
	"to apply cold exposure on hit":                                             {names: []string{"ColdExposureChance"}},
	"to inflict lightning exposure on hit":                                      {names: []string{"LightningExposureChance"}},
	"to inflict lightning exposure":                                             {names: []string{"LightningExposureChance"}},
	"to ignore enemy physical damage reduction":                                 {names: []string{"ChanceToIgnoreEnemyPhysicalDamageReduction"}},
	"to ignore enemy armour":                                                    {names: []string{"ChanceToIgnoreEnemyArmour"}},
	"weapon swap speed":                                                         {names: []string{"WeaponSwapSpeed"}},
	"to chain an additional time from terrain":                                  {names: []string{"TerrainChainChance"}},
	"to apply lightning exposure on hit":                                        {names: []string{"LightningExposureChance"}},
	"to pierce an enemy":                                                        {names: []string{"PierceChance"}},
	"to chain an additional time":                                               {names: []string{"ChainChance"}},
	"penalty to accuracy rating at range":                                       {names: []string{"AccuracyPenalty"}},
	"when you reload a crossbow to be immediate":                                {names: []string{"InstantReloadChance"}},
	"to not expend ammunition":                                                  {names: []string{"ChanceToNotConsumeAmmo"}},
	"second to base energy shield recharge delay":                               {names: []string{"EnergyShieldRechargeFaster"}},
	"culling strike threshold":                                                  {names: []string{"CullPercent"}},
	"effect":                                                                    {names: []string{"LocalEffect"}},
	"effect of flasks":                                                          {names: []string{"FlaskEffect"}},
	"effect of charms":                                                          {names: []string{"CharmEffect"}},
	"magnitudes":                                                                {names: []string{"Magnitude"}},
	"amount recovered":                                                          {names: []string{"FlaskRecovery"}},
	"life recovered":                                                            {names: []string{"FlaskRecovery"}},
	"life recovery from flasks used":                                            {names: []string{"FlaskLifeRecovery"}},
	"recovery":                                                                  {names: []string{"FlaskLifeRecoveryLowLife"}},
	"mana recovered":                                                            {names: []string{"FlaskRecovery"}},
	"life recovery from flasks":                                                 {names: []string{"FlaskLifeRecovery"}},
	"mana recovery from flasks":                                                 {names: []string{"FlaskManaRecovery"}},
	"life and mana recovery from flasks":                                        {names: []string{"FlaskLifeRecovery", "FlaskManaRecovery"}},
	"recovery from flasks":                                                      {names: []string{"FlaskLifeRecovery", "FlaskManaRecovery"}},
	"flask effect duration":                                                     {names: []string{"FlaskDuration"}},
	"charm effect duration":                                                     {names: []string{"CharmDuration"}},
	"recovery speed":                                                            {names: []string{"FlaskRecoveryRate"}},
	"recovery rate":                                                             {names: []string{"FlaskRecoveryRate"}},
	"flask recovery rate":                                                       {names: []string{"FlaskRecoveryRate"}},
	"flask recovery speed":                                                      {names: []string{"FlaskRecoveryRate"}},
	"flask life recovery rate":                                                  {names: []string{"FlaskLifeRecoveryRate"}},
	"flask mana recovery rate":                                                  {names: []string{"FlaskManaRecoveryRate"}},
	"extra charges":                                                             {names: []string{"FlaskCharges"}},
	"charges":                                                                   {names: []string{"FlaskCharges"}},
	"maximum charges":                                                           {names: []string{"FlaskCharges"}},
	"charges gained":                                                            {names: []string{"FlaskChargesGained"}},
	"charges per use":                                                           {names: []string{"FlaskChargesUsed"}},
	"charges used":                                                              {names: []string{"FlaskChargesUsed"}},
	"flask charges used":                                                        {names: []string{"FlaskChargesUsed"}},
	"flask charges used from life flasks":                                       {names: []string{"LifeFlaskChargesUsed"}},
	"flask charges used from mana flasks":                                       {names: []string{"ManaFlaskChargesUsed"}},
	"charm charges used":                                                        {names: []string{"CharmChargesUsed"}},
	"flask charges gained":                                                      {names: []string{"FlaskChargesGained"}},
	"life flask charges gained":                                                 {names: []string{"LifeFlaskChargesGained"}},
	"mana flask charges gained":                                                 {names: []string{"ManaFlaskChargesGained"}},
	"charm charges gained":                                                      {names: []string{"CharmChargesGained"}},
	"flask and charm charges gained":                                            {names: []string{"CharmChargesGained", "FlaskChargesGained"}},
	"charge recovery":                                                           {names: []string{"FlaskChargeRecovery"}},
	"charges per second":                                                        {names: []string{"FlaskChargesGenerated"}},
	"for flasks you use to not consume charges":                                 {names: []string{"FlaskChanceNotConsumeCharges"}},
	"impales you inflict last":                                                  {names: []string{"ImpaleStacksMax"}},
	"adrenaline":                                                                {names: []string{"Condition:Adrenaline"}},
	"elusive":                                                                   {names: []string{"Condition:CanBeElusive"}},
	"onslaught":                                                                 {names: []string{"Condition:Onslaught"}},
	"rampage":                                                                   {names: []string{"Condition:Rampage"}},
	"soul eater":                                                                {names: []string{"Condition:CanHaveSoulEater"}},
	"tailwind":                                                                  {names: []string{"Condition:CanHaveTailwind"}},
	"phasing":                                                                   {names: []string{"Condition:Phasing"}},
	"arcane surge":                                                              {names: []string{"Condition:ArcaneSurge"}},
	"unholy might":                                                              {names: []string{"Condition:UnholyMight"}},
	"chaotic might":                                                             {names: []string{"Condition:ChaoticMight"}},
	"lesser brutal shrine buff":                                                 {names: []string{"Condition:LesserBrutalShrine"}},
	"lesser massive shrine buff":                                                {names: []string{"Condition:LesserMassiveShrine"}},
	"diamond shrine buff":                                                       {names: []string{"Condition:DiamondShrine"}},
	"massive shrine buff":                                                       {names: []string{"Condition:MassiveShrine"}},
	"skill slots":                                                               {names: []string{"SkillSlots"}},
	"skill slot":                                                                {names: []string{"SkillSlots"}},
}

// List of modifier flags
var modFlagListCompiled map[string]CompiledList[modNameListType]
var modFlagList = map[string]modNameListType{
	//pob2:block:src/Modules/ModParser.lua:990-1199
	"with axes":                        {flags: ModFlagAxe | ModFlagHit},
	"to axe attacks":                   {flags: ModFlagAxe | ModFlagHit},
	"with axe attacks":                 {flags: ModFlagAxe | ModFlagHit},
	"with axes or swords":              {flags: ModFlagHit, tag: mod.ModFlagOr(ModFlagAxe | ModFlagSword)},
	"with bows":                        {flags: ModFlagBow | ModFlagHit},
	"to bow attacks":                   {flags: ModFlagBow | ModFlagHit},
	"with bow attacks":                 {flags: ModFlagBow | ModFlagHit},
	"with crossbows":                   {flags: ModFlagCrossbow | ModFlagHit},
	"with claws":                       {flags: ModFlagClaw | ModFlagHit},
	"with claws or daggers":            {flags: ModFlagHit, tag: mod.ModFlagOr(ModFlagClaw | ModFlagDagger)},
	"to claw attacks":                  {flags: ModFlagClaw | ModFlagHit},
	"with claw attacks":                {flags: ModFlagClaw | ModFlagHit},
	"claw attacks":                     {flags: ModFlagClaw | ModFlagHit},
	"dealt with claws":                 {flags: ModFlagClaw | ModFlagHit},
	"with daggers":                     {flags: ModFlagDagger | ModFlagHit},
	"to dagger attacks":                {flags: ModFlagDagger | ModFlagHit},
	"with dagger attacks":              {flags: ModFlagDagger | ModFlagHit},
	"with maces":                       {flags: ModFlagMace | ModFlagHit},
	"to mace attacks":                  {flags: ModFlagMace | ModFlagHit},
	"with mace attacks":                {flags: ModFlagMace | ModFlagHit},
	"with staves":                      {flags: ModFlagStaff | ModFlagHit},
	"to staff attacks":                 {flags: ModFlagStaff | ModFlagHit},
	"with staff attacks":               {flags: ModFlagStaff | ModFlagHit},
	"with quarterstaves":               {flags: ModFlagStaff | ModFlagHit},
	"to quarterstaff attacks":          {flags: ModFlagStaff | ModFlagHit},
	"with quarterstaff attacks":        {flags: ModFlagStaff | ModFlagHit},
	"with swords":                      {flags: ModFlagSword | ModFlagHit},
	"to sword attacks":                 {flags: ModFlagSword | ModFlagHit},
	"with sword attacks":               {flags: ModFlagSword | ModFlagHit},
	"with flails":                      {flags: ModFlagFlail | ModFlagHit},
	"to flail attacks":                 {flags: ModFlagFlail | ModFlagHit},
	"with flail attacks":               {flags: ModFlagFlail | ModFlagHit},
	"with spears":                      {flags: ModFlagSpear | ModFlagHit},
	"to spear attacks":                 {flags: ModFlagSpear | ModFlagHit},
	"with spear attacks":               {flags: ModFlagSpear | ModFlagHit},
	"with wands":                       {flags: ModFlagWand | ModFlagHit},
	"to wand attacks":                  {flags: ModFlagWand | ModFlagHit},
	"with wand attacks":                {flags: ModFlagWand | ModFlagHit},
	"unarmed":                          {flags: ModFlagUnarmed | ModFlagHit},
	"unarmed melee":                    {flags: ModFlagUnarmed | ModFlagMelee | ModFlagHit},
	"with unarmed attacks":             {flags: ModFlagUnarmed | ModFlagHit},
	"with unarmed melee attacks":       {flags: ModFlagUnarmed | ModFlagMelee},
	"to unarmed attacks":               {flags: ModFlagUnarmed | ModFlagHit},
	"to unarmed melee hits":            {flags: ModFlagUnarmed | ModFlagMelee | ModFlagHit},
	"with melee weapons":               {flags: ModFlagWeaponMelee},
	"with melee weapon attacks":        {flags: ModFlagWeaponMelee},
	"from melee weapons":               {flags: ModFlagWeaponMelee},
	"from melee weapon attacks":        {flags: ModFlagWeaponMelee | ModFlagHit},
	"melee weapon":                     {flags: ModFlagWeaponMelee},
	"with one-handed attacks":          {flags: ModFlagWeapon1H | ModFlagHit},
	"with one handed weapons":          {flags: ModFlagWeapon1H | ModFlagHit},
	"with one handed melee weapons":    {flags: ModFlagWeapon1H | ModFlagWeaponMelee | ModFlagHit},
	"with two handed weapons":          {flags: ModFlagWeapon2H | ModFlagHit},
	"with two handed melee weapons":    {flags: ModFlagWeapon2H | ModFlagWeaponMelee | ModFlagHit},
	"with ranged weapons":              {flags: ModFlagWeaponRanged | ModFlagHit},
	"spell":                            {flags: ModFlagSpell},
	"fire spell":                       {flags: ModFlagSpell, keywordFlags: KeywordFlagFire},
	"cold spell":                       {flags: ModFlagSpell, keywordFlags: KeywordFlagCold},
	"lightning spell":                  {flags: ModFlagSpell, keywordFlags: KeywordFlagLightning},
	"chaos spell":                      {flags: ModFlagSpell, keywordFlags: KeywordFlagChaos},
	"for spells":                       {flags: ModFlagSpell},
	"for spell skills":                 {flags: ModFlagSpell},
	"for spell damage":                 {flags: ModFlagSpell},
	"with spell damage":                {flags: ModFlagSpell},
	"with spells":                      {keywordFlags: KeywordFlagSpell},
	"with spell skills":                {keywordFlags: KeywordFlagSpell},
	"with triggered spells":            {keywordFlags: KeywordFlagSpell, tag: mod.SkillType("Triggered")},
	"by spells":                        {keywordFlags: KeywordFlagSpell},
	"by your spells":                   {keywordFlags: KeywordFlagSpell},
	"with attacks":                     {keywordFlags: KeywordFlagAttack},
	"by attacks":                       {keywordFlags: KeywordFlagAttack},
	"by your attacks":                  {keywordFlags: KeywordFlagAttack},
	"with attack skills":               {keywordFlags: KeywordFlagAttack},
	"for attacks":                      {flags: ModFlagAttack},
	"for attack damage":                {flags: ModFlagAttack},
	"weapon":                           {flags: ModFlagWeapon},
	"with weapons":                     {flags: ModFlagWeapon},
	"melee":                            {flags: ModFlagMelee},
	"crossbow":                         {flags: ModFlagCrossbow},
	"for crossbow attacks":             {flags: ModFlagCrossbow, tag: mod.SkillType("CrossbowSkill")},
	"with melee attacks":               {flags: ModFlagMelee},
	"with melee damage":                {flags: ModFlagMelee},
	"with melee critical hits":         {flags: ModFlagMelee, tag: mod.Condition("CriticalStrike")},
	"with melee skills":                {tag: mod.SkillType("Melee")},
	"with bow skills":                  {keywordFlags: KeywordFlagBow},
	"on melee hit":                     {flags: ModFlagMelee | ModFlagHit},
	"on hit":                           {flags: ModFlagHit},
	"with hits":                        {keywordFlags: KeywordFlagHit},
	"with hits against nearby enemies": {keywordFlags: KeywordFlagHit},
	"with hits and ailments":           {keywordFlags: KeywordFlagHit | KeywordFlagAilment},
	"with ailments":                    {flags: ModFlagAilment},
	"with ailments from attack skills": {flags: ModFlagAilment, keywordFlags: KeywordFlagAttack},
	"with poison":                      {keywordFlags: KeywordFlagPoison},
	"with bleeding":                    {keywordFlags: KeywordFlagBleed},
	"for ailments":                     {flags: ModFlagAilment},
	"for poison":                       {keywordFlags: KeywordFlagPoison | KeywordFlagMatchAll},
	"for bleeding":                     {keywordFlags: KeywordFlagBleed},
	"for ignite":                       {keywordFlags: KeywordFlagIgnite},
	"against damage over time":         {flags: ModFlagDot},
	"area":                             {flags: ModFlagArea},
	"with projectiles":                 {flags: ModFlagProjectile},
	"mine":                             {keywordFlags: KeywordFlagMine},
	"with mines":                       {keywordFlags: KeywordFlagMine},
	"trap":                             {keywordFlags: KeywordFlagTrap},
	"with traps":                       {keywordFlags: KeywordFlagTrap},
	"for traps":                        {keywordFlags: KeywordFlagTrap},
	"that place mines or throw traps":  {keywordFlags: KeywordFlagMine | KeywordFlagTrap},
	"that throw mines":                 {keywordFlags: KeywordFlagMine},
	"that throw traps":                 {keywordFlags: KeywordFlagTrap},
	"grenade":                          {tag: mod.SkillType("Grenade")},
	"for grenade skills":               {tag: mod.SkillType("Grenade")},
	"hazard":                           {tag: mod.SkillType("Hazard")},
	"brand":                            {tag: mod.SkillType("Brand")},
	"totem":                            {keywordFlags: KeywordFlagTotem},
	"ballista":                         {keywordFlags: KeywordFlagTotem, tag: mod.Condition("BallistaSkill")},
	"with totem skills":                {keywordFlags: KeywordFlagTotem},
	"for skills used by totems":        {keywordFlags: KeywordFlagTotem},
	"totem skills that cast an aura":   {tag: mod.SkillType("Aura"), keywordFlags: KeywordFlagTotem},
	"aura skills that summon totems":   {tag: mod.SkillType("Aura"), keywordFlags: KeywordFlagTotem},
	"of aura skills":                   {tag: mod.SkillType("Aura")},
	"of buff skills":                   {tag: mod.SkillType("Buff")},
	"curse skills":                     {keywordFlags: KeywordFlagCurse},
	"of curse skills":                  {keywordFlags: KeywordFlagCurse},
	"with curse skills":                {keywordFlags: KeywordFlagCurse},
	"of curse aura skills":             {tag: mod.SkillType("Aura"), keywordFlags: KeywordFlagCurse},
	"of curse auras":                   {keywordFlags: KeywordFlagCurse | KeywordFlagAura | KeywordFlagMatchAll},
	"of hex skills":                    {tag: mod.SkillType("Hex")},
	"with hex skills":                  {tag: mod.SkillType("Hex")},
	"of herald skills":                 {tag: mod.SkillType("Herald")},
	"with herald skills":               {tag: mod.SkillType("Herald")},
	"with hits from herald skills":     {tag: mod.SkillType("Herald"), keywordFlags: KeywordFlagHit},
	"minion skills":                    {tag: mod.SkillType("Minion")},
	"of minion skills":                 {tag: mod.SkillType("Minion")},
	"of command skills":                {tag: mod.Condition("CommandableSkill")},
	"of companion skills":              {tag: mod.SkillType("CreatesCompanion")},
	"of non-companion skills":          {tag: mod.SkillType("CreatesCompanion").Neg(true)},
	"link skills":                      {tag: mod.SkillType("Link")},
	"of link skills":                   {tag: mod.SkillType("Link")},
	"for curses":                       {keywordFlags: KeywordFlagCurse},
	"for hexes":                        {tag: mod.SkillType("Hex")},
	"with warcries":                    {keywordFlags: KeywordFlagWarcry},
	"warcry":                           {keywordFlags: KeywordFlagWarcry},
	"vaal":                             {keywordFlags: KeywordFlagVaal},
	"vaal skill":                       {keywordFlags: KeywordFlagVaal},
	"with vaal skills":                 {keywordFlags: KeywordFlagVaal},
	"with non-vaal skills":             {tag: mod.SkillType("Vaal").Neg(true)},
	"with movement skills":             {keywordFlags: KeywordFlagMovement},
	"of movement skills":               {keywordFlags: KeywordFlagMovement},
	"of movement skills used":          {keywordFlags: KeywordFlagMovement},
	"of travel skills":                 {tag: mod.SkillType("Travel")},
	"of banner skills":                 {tag: mod.SkillType("Banner")},
	"with lightning skills":            {keywordFlags: KeywordFlagLightning},
	"for lightning skills":             {keywordFlags: KeywordFlagLightning},
	"with cold skills":                 {keywordFlags: KeywordFlagCold},
	"with fire skills":                 {keywordFlags: KeywordFlagFire},
	"with elemental skills":            {keywordFlags: KeywordFlagLightning | KeywordFlagCold | KeywordFlagFire},
	"with chaos skills":                {keywordFlags: KeywordFlagChaos},
	"with physical skills":             {keywordFlags: KeywordFlagPhysical},
	"with channelling skills":          {tag: mod.SkillType("Channel")},
	"with non-channelling skills":      {tag: mod.SkillType("Channel").Neg(true)},
	"channelling":                      {tag: mod.SkillType("Channel")},
	"channelling skills":               {tag: mod.SkillType("Channel")},
	"non-channelling":                  {tag: mod.SkillType("Channel").Neg(true)},
	"non-channelling skills":           {tag: mod.SkillType("Channel").Neg(true)},
	"with brand skills":                {tag: mod.SkillType("Brand")},
	"for stance skills":                {tag: mod.SkillType("Stance")},
	"of stance skills":                 {tag: mod.SkillType("Stance")},
	"mark skills":                      {tag: mod.SkillType("Mark")},
	"of marks":                         {tag: mod.SkillType("Mark")},
	"of mark skills":                   {tag: mod.SkillType("Mark")},
	"of your mark skills":              {tag: mod.SkillType("Mark")},
	"with command skills":              {tag: mod.Condition("CommandableSkill")},
	"for command skills":               {tag: mod.Condition("CommandableSkill")},
	"while shapeshifted":               {tag: mod.Condition("Shapeshifted")},
	"with plant skills":                {tag: mod.SkillType("Plant")},
	"with storm skills":                {tag: mod.SkillType("Storm")},
	"minion":                           {addToMinion: true},
	"zombie":                           {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie").Transfigured(true)},
	"raised zombie":                    {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie").Transfigured(true)},
	"skeleton":                         {addToMinion: true, addToMinionTag: mod.SkillName("Summon Skeletons").Transfigured(true)},
	"spectre":                          {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre").Transfigured(true)},
	"raised spectre":                   {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre").Transfigured(true)},
	"golem":                            {addToMinion: true, addToMinionTag: mod.SkillType("Golem")},
	"chaos golem":                      {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem").Transfigured(true)},
	"flame golem":                      {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem").Transfigured(true)},
	"increased flame golem":            {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem").Transfigured(true)},
	"ice golem":                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem").Transfigured(true)},
	"lightning golem":                  {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem").Transfigured(true)},
	"stone golem":                      {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem").Transfigured(true)},
	"animated guardian":                {addToMinion: true, addToMinionTag: mod.SkillName("Animate Guardian").Transfigured(true)},
	"with physical damage":             {tag: mod.Condition("PhysicalHasDamage")},
	"with lightning damage":            {tag: mod.Condition("LightningHasDamage")},
	"with cold damage":                 {tag: mod.Condition("ColdHasDamage")},
	"with fire damage":                 {tag: mod.Condition("FireHasDamage")},
	"with chaos damage":                {tag: mod.Condition("ChaosHasDamage")},
	"global":                           {tag: mod.Global()},
	"from equipped shield":             {tagList: []mod.Tag{mod.SlotName("Weapon 2"), mod.Condition("UsingShield")}},
	"from equipped focus":              {tagList: []mod.Tag{mod.SlotName("Weapon 2"), mod.Condition("UsingFocus")}},
	"from equipped helmet":             {tag: mod.SlotName("Helmet")},
	"from equipped gloves and boots":   {tag: mod.SlotName("Gloves", "Boots")},
	"from equipped boots and gloves":   {tag: mod.SlotName("Gloves", "Boots")},
	"from equipped helmet and gloves":  {tag: mod.SlotName("Helmet", "Gloves")},
	"from equipped helmet and boots":   {tag: mod.SlotName("Helmet", "Boots")},
	"from your equipped body armour":   {tag: mod.SlotName("Body Armour")},
	"from equipped body armour":        {tag: mod.SlotName("Body Armour")},
	"from body armour":                 {tag: mod.SlotName("Body Armour")},
	"from your body armour":            {tag: mod.SlotName("Body Armour")},
}

// List of modifier flags/tags that appear at the start of a line
var preFlagListCompiled map[string]CompiledList[modNameListType]
var preFlagList = map[string]modNameListType{
	//pob2:block:src/Modules/ModParser.lua:1205-1446
	"^axe attacks [hd][ae][va][el] ":                           {flags: ModFlagAxe},
	"^axe or sword attacks [hd][ae][va][el] ":                  {tag: mod.ModFlagOr(ModFlagAxe | ModFlagSword)},
	"^bow attacks [hd][ae][va][el] ":                           {flags: ModFlagBow},
	"^claw attacks [hd][ae][va][el] ":                          {flags: ModFlagClaw},
	"^claw or dagger attacks [hd][ae][va][el] ":                {tag: mod.ModFlagOr(ModFlagClaw | ModFlagDagger)},
	"^dagger attacks [hd][ae][va][el] ":                        {flags: ModFlagDagger},
	"^staff attacks [hd][ae][va][el] ":                         {flags: ModFlagStaff},
	"^sword attacks [hd][ae][va][el] ":                         {flags: ModFlagSword},
	"^wand attacks [hd][ae][va][el] ":                          {flags: ModFlagWand},
	"^unarmed attacks [hd][ae][va][el] ":                       {flags: ModFlagUnarmed},
	`^attacks with one[ \-]handed weapons [hd][ae][va][el] `:   {flags: ModFlagWeapon1H},
	"^attacks with two handed weapons [hd][ae][va][el] ":       {flags: ModFlagWeapon2H},
	"^attacks with melee weapons [hd][ae][va][el] ":            {flags: ModFlagWeaponMelee},
	"^melee weapon hits [hd][ae][va][el] ":                     {flags: ModFlagWeaponMelee | ModFlagHit},
	"^attacks with one handed melee weapons [hd][ae][va][el] ": {flags: ModFlagWeapon1H | ModFlagWeaponMelee},
	"^attacks with two handed melee weapons [hd][ae][va][el] ": {flags: ModFlagWeapon2H | ModFlagWeaponMelee},
	"^attacks with ranged weapons [hd][ae][va][el] ":           {flags: ModFlagWeaponRanged},
	"^attack damage ":                                          {flags: ModFlagAttack},
	"^thorns damage has ":                                      {flags: ModFlagThorns},
	"^hits deal ":                                              {keywordFlags: KeywordFlagHit},
	"^melee weapon damage":                                     {flags: ModFlagWeaponMelee},
	"^deal ":                                                   {},
	"^causes ":                                                 {},
	"^critical hits deal ":                                     {tag: mod.Condition("CriticalStrike")},
	"^poisons you inflict with critical hits have ":            {keywordFlags: KeywordFlagPoison | KeywordFlagMatchAll, tag: mod.Condition("CriticalStrike")},
	"^minions ":                         {addToMinion: true},
	"^minions [cthd][ae][ukva][sel]e? ": {addToMinion: true},
	"^while a unique enemy is in your presence, minions [hd][ae][va][el] ":        {addToMinion: true, playerTag: mod.ActorCondition("enemy", "RareOrUnique")},
	"^while a pinnacle atlas boss is in your presence, minions [hd][ae][va][el] ": {addToMinion: true, playerTag: mod.ActorCondition("enemy", "PinnacleBoss")},
	"^minions leech ":                                                {addToMinion: true},
	"^minions' attacks deal ":                                        {addToMinion: true, flags: ModFlagAttack},
	"^undead minions [thd][ae][kva][el] ":                            {addToMinion: true, addToMinionTag: mod.SkillType("CreatesUndeadMinion")},
	"^golems [hd][ae][va][el] ":                                      {addToMinion: true, addToMinionTag: mod.SkillType("Golem")},
	"^summoned golems [hd][ae][va][el] ":                             {addToMinion: true, addToMinionTag: mod.SkillType("Golem")},
	"^golem skills have ":                                            {tag: mod.SkillType("Golem")},
	`^non\-minion skills have `:                                      {tag: mod.SkillType("Minion").Neg(true)},
	"^zombies [hd][ae][va][el] ":                                     {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie").Transfigured(true)},
	"^raised zombies [hd][ae][va][el] ":                              {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie").Transfigured(true)},
	"^skeletons [hd][ae][va][el] ":                                   {addToMinion: true, addToMinionTag: mod.SkillName("Summon Skeletons").Transfigured(true)},
	"^raging spirits [hd][ae][va][el] ":                              {addToMinion: true, addToMinionTag: mod.SkillName("Summon Raging Spirit").Transfigured(true)},
	"^summoned raging spirits [hd][ae][va][el] ":                     {addToMinion: true, addToMinionTag: mod.SkillName("Summon Raging Spirit").Transfigured(true)},
	"^spectres [hd][ae][va][el] ":                                    {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre").Transfigured(true)},
	"^chaos golems [hd][ae][va][el] ":                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem").Transfigured(true)},
	"^summoned chaos golems [hd][ae][va][el] ":                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem").Transfigured(true)},
	"^flame golems [hd][ae][va][el] ":                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem").Transfigured(true)},
	"^summoned flame golems [hd][ae][va][el] ":                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem").Transfigured(true)},
	"^ice golems [hd][ae][va][el] ":                                  {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem").Transfigured(true)},
	"^summoned ice golems [hd][ae][va][el] ":                         {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem").Transfigured(true)},
	"^lightning golems [hd][ae][va][el] ":                            {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem").Transfigured(true)},
	"^summoned lightning golems [hd][ae][va][el] ":                   {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem").Transfigured(true)},
	"^stone golems [hd][ae][va][el] ":                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem").Transfigured(true)},
	"^summoned stone golems [hd][ae][va][el] ":                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem").Transfigured(true)},
	"^summoned carrion golems [hd][ae][va][el] ":                     {addToMinion: true, addToMinionTag: mod.SkillName("Summon Carrion Golem").Transfigured(true)},
	"^summoned skitterbots [hd][ae][va][el] ":                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Carrion Golem").Transfigured(true)},
	"^blink arrow and blink arrow clones [hd][ae][va][el] ":          {addToMinion: true, addToMinionTag: mod.SkillName("Blink Arrow").Transfigured(true)},
	"^mirror arrow and mirror arrow clones [hd][ae][va][el] ":        {addToMinion: true, addToMinionTag: mod.SkillName("Mirror Arrow").Transfigured(true)},
	"^animated weapons [hd][ae][va][el] ":                            {addToMinion: true, addToMinionTag: mod.SkillName("Animate Weapon").Transfigured(true)},
	"^animated guardians? deals? ":                                   {addToMinion: true, addToMinionTag: mod.SkillName("Animate Guardian").Transfigured(true)},
	"^summoned holy relics [hd][ae][va][el] ":                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Holy Relic")},
	"^summoned reaper [dh][ea][as]l?s? ":                             {addToMinion: true, addToMinionTag: mod.SkillName("Summon Reaper").Transfigured(true)},
	"^summoned arbalists [hgdf][aei][vair][eln] ":                    {addToMinion: true, addToMinionTag: mod.SkillName("Summon Arbalists")},
	"^summoned arbalists' attacks have ":                             {addToMinion: true, addToMinionTag: mod.SkillName("Summon Arbalists")},
	"^herald skills [hd][ae][va][el] ":                               {tag: mod.SkillType("Herald")},
	"^shapeshift skills [hd][ae][va][el] ":                           {tag: mod.SkillType("Shapeshift")},
	"^agony crawler deals ":                                          {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Agony")},
	"^summoned agony crawler fires ":                                 {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Agony")},
	"^sentinels of purity deal ":                                     {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity")},
	"^summoned sentinels of absolution have ":                        {addToMinion: true, addToMinionTag: mod.SkillName("Absolution").Transfigured(true)},
	"^summoned sentinels have ":                                      {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity", "Dominating Blow", "Absolution").Transfigured(true)},
	"^raised zombies' slam attack has ":                              {addToMinion: true, tag: mod.SkillId("ZombieSlam")},
	"^raised spectres, raised zombies, and summoned skeletons have ": {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre", "Raise Zombie", "Summon Skeletons").Transfigured(true)},
	"^companions [hd][ae][va][el] ":                                  {addToMinion: true, addToMinionTag: mod.SkillType("CreatesCompanion")},
	"^companions ":                                                   {addToMinion: true, addToMinionTag: mod.SkillType("CreatesCompanion")},
	"^attacks used by totems have ":                                  {flags: ModFlagAttack, keywordFlags: KeywordFlagTotem},
	"^spells cast by totems [hd][ae][va][el] ":                       {flags: ModFlagSpell, keywordFlags: KeywordFlagTotem},
	"^skills used by totems [hd][ae][va][el] ":                       {keywordFlags: KeywordFlagTotem},
	"^trap and mine damage ":                                         {keywordFlags: KeywordFlagTrap | KeywordFlagMine},
	"^skills used by traps [hgd][ae][via][enl] ":                     {keywordFlags: KeywordFlagTrap},
	`^skills which throw traps ([hgd][ae][via][enl])\? `:             {keywordFlags: KeywordFlagTrap},
	"^skills used by mines [hgd][ae][via][enl] ":                     {keywordFlags: KeywordFlagMine},
	`^skills which throw mines ([hgd][ae][via][enl])\? `:             {keywordFlags: KeywordFlagMine},
	"^attacks with this weapon ":                                     {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType("Attack")}},
	"^attacks with this weapon [hd][ae][va][el] ":                    {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType("Attack")}},
	"^hits with this weapon [hd][ae][va][el] ":                       {flags: ModFlagHit, tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType("Attack")}},
	"^attacks [ghd][ae][iva][eln] ":                                  {flags: ModFlagAttack},
	"^attacks using your weapons [ghd][ae][iva][eln] ":               {flags: ModFlagAttack, tag: mod.SkillType("NonWeaponAttack").Neg(true)},
	"^attacks used by ballistas [hd][ae][va][el] ":                   {flags: ModFlagAttack, keywordFlags: KeywordFlagTotem, tag: mod.Condition("BallistaSkill")},
	"^attack skills [hd][ae][va][el] ":                               {keywordFlags: KeywordFlagAttack},
	"^spells [hgdf][aei][ivar][nel] a? ?":                            {flags: ModFlagSpell},
	"^spells which cost life gain ":                                  {keywordFlags: KeywordFlagSpell, tag: mod.StatThreshold("", 1).Stats("LifeCost", "LifePerSecondCost")},
	"^spell skills [hd][ae][va][el] ":                                {tag: mod.SkillType("Spell")},
	"^spell hits [ghd][ae][iva][eln] ":                               {flags: ModFlagHit, keywordFlags: KeywordFlagSpell},
	"^offering skills [hd][ae][va][el] ":                             {tag: mod.SkillType("Offering")},
	"^projectile attack skills [hd][ae][va][el] ":                    {tag: mod.SkillType("RangedAttack")},
	"^projectiles from attacks [hd][ae][va][el] ":                    {tag: mod.SkillType("RangedAttack")},
	"^arrows [hd][ae][va][el] ":                                      {tag: mod.KeywordFlagAnd(KeywordFlagArrow)},
	"^bow skills [hdf][aei][var][el] ":                               {keywordFlags: KeywordFlagBow},
	"^projectiles [hdf][aei][var][el] ":                              {flags: ModFlagProjectile},
	"^grenade skills [hdf][aei][var][el] ":                           {tag: mod.SkillType("Grenade")},
	"^detonator skills [hd][ae][va][el] ":                            {tag: mod.SkillType("Detonator")},
	"^melee attacks have ":                                           {flags: ModFlagMelee},
	"^movement attack skills have ":                                  {flags: ModFlagAttack, keywordFlags: KeywordFlagMovement},
	"^travel skills have ":                                           {tag: mod.SkillType("Travel")},
	"^link skills have ":                                             {tag: mod.SkillType("Link")},
	"^lightning skills [hd][ae][va][el] a? ?":                        {keywordFlags: KeywordFlagLightning},
	"^lightning spells [hd][ae][va][el] a? ?":                        {keywordFlags: KeywordFlagLightning, flags: ModFlagSpell},
	"^cold skills [hd][ae][va][el] a? ?":                             {keywordFlags: KeywordFlagCold},
	"^cold spells [hd][ae][va][el] a? ?":                             {keywordFlags: KeywordFlagCold, flags: ModFlagSpell},
	"^fire skills [hd][ae][va][el] a? ?":                             {keywordFlags: KeywordFlagFire},
	"^fire spells [hd][ae][va][el] a? ?":                             {keywordFlags: KeywordFlagFire, flags: ModFlagSpell},
	"^chaos skills [hd][ae][va][el] a? ?":                            {keywordFlags: KeywordFlagChaos},
	"^vaal skills [hd][ae][va][el] ":                                 {keywordFlags: KeywordFlagVaal},
	"^brand skills [hd][ae][va][el] ":                                {keywordFlags: KeywordFlagBrand},
	"^channelling skills [hd][ae][va][el] ":                          {tag: mod.SkillType("Channel")},
	"^curse skills [hd][ae][va][el] ":                                {keywordFlags: KeywordFlagCurse},
	"^hex skills [hd][ae][va][el] ":                                  {tag: mod.SkillType("Hex")},
	"^mark skills [hd][ae][va][el] ":                                 {tag: mod.SkillType("Mark")},
	"^melee skills [hd][ae][va][el] ":                                {tag: mod.SkillType("Melee")},
	"^melee attack skills [hd][ae][va][el] ":                         {tagList: []mod.Tag{mod.SkillType("Melee"), mod.SkillType("Attack")}},
	"^guard skills [hd][ae][va][el] ":                                {tag: mod.SkillType("Guard")},
	"^banner skills [hd][ae][va][el] ":                               {tag: mod.SkillType("Banner")},
	"^nova spells [hd][ae][va][el] ":                                 {tag: mod.SkillType("Nova")},
	"^area skills [hd][ae][va][el] ":                                 {tag: mod.SkillType("Area")},
	"^aura skills [hd][ae][va][el] ":                                 {tag: mod.SkillType("Aura")},
	"^prismatic skills [hd][ae][va][el] ":                            {tag: mod.SkillType("RandomElement")},
	"^retaliation skills [hd][ae][va][el] ":                          {tag: mod.SkillType("Retaliation")},
	"^meta skills [hd][ae][va][el] ":                                 {tag: mod.SkillType("Meta")},
	"^invocated skills [hd][ae][va][el] ":                            {tag: mod.Condition("InvocationSkill")},
	"^invocated spells [hd][ae][va][el] ":                            {keywordFlags: KeywordFlagSpell, tag: mod.Condition("InvocationSkill")},
	"^slam skills [hd][ae][va][el] ":                                 {tag: mod.SkillType("Slam")},
	"^warcry skills have ":                                           {tag: mod.SkillType("Warcry")},
	`^non\-curse aura skills have `:                                  {tagList: []mod.Tag{mod.SkillType("Aura"), mod.SkillType("AppliesCurse").Neg(true)}},
	`^non\-channelling skills have `:                                 {tag: mod.SkillType("Channel").Neg(true)},
	`^non\-channelling spells `:                                      {keywordFlags: KeywordFlagSpell, tag: mod.SkillType("Channel").Neg(true)},
	`^non\-channelling spells [hd][ae][va][el] `:                     {flags: ModFlagSpell, tag: mod.SkillType("Channel").Neg(true)},
	`^non\-vaal skills deal `:                                        {tag: mod.SkillType("Vaal").Neg(true)},
	"^bolts fired by crossbow attacks [hd][ae][va][el] ":             {flags: ModFlagCrossbow, tag: mod.SkillType("CrossbowSkill")},
	"^skills [hgdf][aei][vari][eln] ":                                {},
	"^triggered spells [hd][ae][va][el] ":                            {keywordFlags: KeywordFlagSpell, tag: mod.SkillType("Triggered")},
	"^totems have ":                                                  {keywordFlags: KeywordFlagTotem},
	"^persistent buffs have ":                                        {tagList: []mod.Tag{mod.SkillType("Persistent"), mod.SkillType("Buff")}},
	"^left ring slot: ":                                              {tag: mod.SlotNumber(int(1))},
	"^right ring slot: ":                                             {tag: mod.SlotNumber(int(2))},
	"^socketed gems [hgd][ae][via][enl] ":                            {addToSkill: mod.SocketedIn("{SlotName}")},
	"^socketed skills [hgd][ae][via][enl] ":                          {addToSkill: mod.SocketedIn("{SlotName}")},
	"^socketed travel skills [hgd][ae][via][enl] ":                   {addToSkill: mod.SocketedIn("{SlotName}").Keyword("travel")},
	"^socketed warcry skills [hgd][ae][via][enl] ":                   {addToSkill: mod.SocketedIn("{SlotName}").Keyword("warcry")},
	"^socketed attacks [hgd][ae][via][enl] ":                         {addToSkill: mod.SocketedIn("{SlotName}").Keyword("attack")},
	"^socketed spells [hgd][ae][via][enl] ":                          {addToSkill: mod.SocketedIn("{SlotName}").Keyword("spell")},
	"^socketed curse gems [hgd][ae][via][enl] ":                      {addToSkill: mod.SocketedIn("{SlotName}").Keyword("curse")},
	"^socketed melee gems [hgd][ae][via][enl] ":                      {addToSkill: mod.SocketedIn("{SlotName}").Keyword("melee")},
	"^socketed golem gems [hgd][ae][via][enl] ":                      {addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	"^socketed golem skills [hgd][ae][via][enl] ":                    {addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	"^socketed golem skills have minions ":                           {addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	"^socketed vaal skills [hgd][ae][via][enl] ":                     {addToSkill: mod.SocketedIn("{SlotName}").Keyword("vaal")},
	"^socketed projectile spells [hgdf][aei][viar][enl] ":            {addToSkill: mod.SocketedIn("{SlotName}"), tagList: []mod.Tag{mod.SkillType("Projectile"), mod.SkillType("Spell")}},
	"^enemies withered by you [th]a[vk]e ":                           {tag: mod.MultiplierThreshold("WitheredStack").Threshold(1), applyToEnemy: true},
	`^enemies (\w+) by you take `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Condition(firstToUpper(caps[0])), applyToEnemy: true, modSuffix: "Taken"}
		},
	},
	`^enemies (\w+) by `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Condition(firstToUpper(caps[0])), applyToEnemy: true}
		},
	},
	`^enemies (\w+) by you have `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Condition(firstToUpper(caps[0])), applyToEnemy: true}
		},
	},
	`^enemies you (\w+) have `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Condition(firstToUpper(caps[0])), applyToEnemy: true}
		},
	},
	"^while a pinnacle atlas boss is in your presence, enemies you've hit recently have ": {
		fn: func(caps []string) modNameListType {
			return modNameListType{playerTagList: []mod.Tag{mod.Condition("HitRecently"), mod.ActorCondition("enemy", "PinnacleBoss")}, applyToEnemy: true}
		},
	},
	"^while a unique enemy is in your presence, enemies you've hit recently have ": {
		fn: func(caps []string) modNameListType {
			return modNameListType{playerTagList: []mod.Tag{mod.Condition("HitRecently"), mod.ActorCondition("enemy", "RareOrUnique")}, applyToEnemy: true}
		},
	},
	"^enemies you've hit recently have ": {
		fn: func(caps []string) modNameListType {
			return modNameListType{playerTag: mod.Condition("HitRecently"), applyToEnemy: true}
		},
	},
	`^hits against enemies (\w+) by you have `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ActorCondition("enemy", firstToUpper(caps[0]))}
		},
	},
	"^enemies shocked or frozen by you take ":                         {tag: mod.Condition("Shocked", "Frozen"), applyToEnemy: true, modSuffix: "Taken"},
	"^enemies affected by your spider's webs [thd][ae][avk][el] ":     {tag: mod.MultiplierThreshold("Spider's WebStack").Threshold(1), applyToEnemy: true},
	"^enemies you curse take ":                                        {tag: mod.Condition("Cursed"), applyToEnemy: true, modSuffix: "Taken"},
	"^enemies you curse ":                                             {tag: mod.Condition("Cursed"), applyToEnemy: true},
	"^targets cursed by you [hgd][ae][via][enl] ":                     {tag: mod.Condition("Cursed"), applyToEnemy: true},
	"^enemies you electrocute have ":                                  {tag: mod.Condition("Electrocuted"), applyToEnemy: true},
	"^nearby enemies take ":                                           {modSuffix: "Taken", applyToEnemy: true},
	"^nearby enemies have ":                                           {applyToEnemy: true},
	"^nearby enemies deal ":                                           {applyToEnemy: true},
	"^nearby enemies'? ":                                              {applyToEnemy: true},
	"^nearby enemy monsters' ":                                        {applyToEnemy: true},
	"against you":                                                     {applyToEnemy: true, actorEnemy: true},
	"^hits against you ":                                              {applyToEnemy: true, flags: ModFlagHit},
	"^hits against you [hd][ae][va][el] ":                             {applyToEnemy: true, flags: ModFlagHit},
	"^enemies near your totems deal ":                                 {applyToEnemy: true},
	"^enemies affected by your hazards recently [hgd][ae][via][enl] ": {applyToEnemy: true, tag: mod.ActorCondition("", "AffectedByHazardRecently")},
	"^your flasks grant ":                                             {},
	"^when hit, ":                                                     {},
	"^you and allies [hgd][ae][via][enl] ":                            {},
	"^auras from your skills grant ":                                  {addToAura: true},
	"^auras grant ":                                                   {addToAura: true},
	"^banners also grant ":                                            {addToAura: true, onlyAddToBanners: true},
	"^you and nearby allies ":                                         {newAura: true},
	"^you and nearby allies [hgd][ae][via][enl] ":                     {newAura: true},
	"^nearby allies [hgd][ae][via][enl] ":                             {newAura: true, newAuraOnlyAllies: true},
	"allies in your presence ":                                        {newAura: true, newAuraOnlyAllies: true},
	"allies in your presence [hgd][ae][via][enl] ":                    {newAura: true, newAuraOnlyAllies: true},
	"^you and allies in your presence [hgd][ae][via][enl] ":           {newAura: true},
	"^every rage also grants ":                                        {tag: mod.Multiplier("RageEffect")},
	"^each rage also grants ":                                         {tag: mod.Multiplier("RageEffect")},
	`^every (\d+) rage also grants `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("RageEffect").Div(utils.Float(caps[0]))}
		},
	},
	"^you and allies affected by auras from your skills [hgd][ae][via][enl] ": {tag: mod.Condition("AffectedByAura")},
	"^take ":      {modSuffix: "Taken"},
	"^marauder: ": {tag: mod.Condition("ConnectedToMarauderStart")},
	"^duelist: ":  {tag: mod.Condition("ConnectedToDuelistStart")},
	"^ranger: ":   {tag: mod.Condition("ConnectedToRangerStart")},
	"^shadow: ":   {tag: mod.Condition("ConnectedToShadowStart")},
	"^witch: ":    {tag: mod.Condition("ConnectedToWitchStart")},
	"^templar: ":  {tag: mod.Condition("ConnectedToTemplarStart")},
	"^scion: ":    {tag: mod.Condition("ConnectedToScionStart")},
	"^skills supported by spellslinger have ":                            {tag: mod.Condition("SupportedBySpellslinger")},
	"^skills that have dealt a critical hit in the past 8 seconds deal ": {tag: mod.Condition("CritInPast8Sec")},
	"^blink arrow and mirror arrow have ":                                {tag: mod.SkillName("Blink Arrow", "Mirror Arrow").Transfigured(true)},
	"attacks with energy blades ":                                        {flags: ModFlagAttack, tag: mod.Condition("AffectedByEnergyBlade")},
	"^for each nearby corpse, ":                                          {tag: mod.Multiplier("NearbyCorpse")},
	"^enemies in your link beams have ":                                  {tag: mod.Condition("BetweenYouAndLinkedTarget"), applyToEnemy: true},
	"^consecrated ground you create also grants ":                        {tag: mod.Condition("OnConsecratedGround")},
	"^while a unique enemy is in your presence, ":                        {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	"^while a pinnacle atlas boss is in your presence, ":                 {tag: mod.ActorCondition("enemy", "PinnacleBoss")},
	"^enemies in your presence ":                                         {applyToEnemy: true, tag: mod.ActorCondition("enemy", "EnemyInPresence")},
	"^enemies in your presence [hgd][ae][via][enl] ":                     {applyToEnemy: true, tag: mod.ActorCondition("enemy", "EnemyInPresence")},
	"^body armour grants ":                                               {tag: mod.ItemCondition("Body Armour").Rarity("NORMAL")},
}

// List of modifier tags
var modTagListCompiled map[string]CompiledList[modNameListType]
var modTagList = map[string]modNameListType{
	//pob2:block:src/Modules/ModParser.lua:1451-1706
	"on enemies":                       {},
	"while active":                     {},
	`for (\d+) seconds`:                {},
	"when you hit a unique enemy":      {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	" on critical hit":                 {tag: mod.Condition("CriticalStrike")},
	"from critical hits":               {tag: mod.Condition("CriticalStrike")},
	"with critical hits":               {tag: mod.Condition("CriticalStrike")},
	"while affected by auras you cast": {tag: mod.Condition("AffectedByAura")},
	"for you and nearby allies":        {newAura: true},
	"per power charge":                 {tag: mod.Multiplier("PowerCharge")},
	`if you've consumed an? (\D+) charge recently`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("Removable" + firstToUpper(caps[0]) + "Charge").Limit(1)}
		},
	},
	"per frenzy charge":     {tag: mod.Multiplier("FrenzyCharge")},
	"per endurance charge":  {tag: mod.Multiplier("EnduranceCharge")},
	"per siphoning charge":  {tag: mod.Multiplier("SiphoningCharge")},
	"per spirit charge":     {tag: mod.Multiplier("SpiritCharge")},
	"per challenger charge": {tag: mod.Multiplier("ChallengerCharge")},
	"per intensity":         {tag: mod.Multiplier("Intensity")},
	"per brand":             {tag: mod.Multiplier("ActiveBrand")},
	`per brand, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ActiveBrand").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	"per blitz charge":                       {tag: mod.Multiplier("BlitzCharge")},
	"per ghost shroud":                       {tag: mod.Multiplier("GhostShroud")},
	"per crab barrier":                       {tag: mod.Multiplier("CrabBarrier")},
	"per rage":                               {tag: mod.Multiplier("Rage")},
	"per rage while you are not losing rage": {tag: mod.Multiplier("Rage")},
	`per combo consumed, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tagList: []mod.Tag{mod.Multiplier("ComboStacks").Limit(utils.Float(caps[0])).LimitTotal(true), mod.SkillType("ComboStacking")}}
		},
	},
	`per (\d+) rage`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("Rage").Div(utils.Float(caps[0]))}
		},
	},
	"per mana burn":        {tag: mod.Multiplier("ManaBurnStacks")},
	"per mana burn on you": {tag: mod.Multiplier("ManaBurnStacks")},
	`per mana burn, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ManaBurnStacks").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	"per level":        {tag: mod.Multiplier("Level")},
	"per player level": {tag: mod.Multiplier("Level")},
	`per (\d+) player levels`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("Level").Div(utils.Float(caps[0]))}
		},
	},
	`per (\d+) levels`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("Level").Div(utils.Float(caps[0]))}
		},
	},
	"per defiance": {tag: mod.Multiplier("Defiance")},
	`per (\d+)% (\w+) effect on enemy`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier(firstToUpper(caps[1]) + "Effect").Div(utils.Float(caps[0])).Actor("enemy")}
		},
	},
	"per socketed rune or soul core":          {tag: mod.Multiplier("RunesSocketedIn{SlotName}")},
	"per socket filled":                       {tag: mod.Multiplier("RunesSocketedIn{SlotName}")},
	"per idol socketed in your equipment":     {tag: mod.Multiplier("IdolsInEquipment").Actor("player")},
	"per idol in your equipment":              {tag: mod.Multiplier("IdolsInEquipment").Actor("player")},
	`per non\-idol augment in your equipment`: {tag: mod.Multiplier("NonIdolAugmentsInEquipment").Actor("player")},
	`per (\d+) (\w+) support gems socketed`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier(firstToUpper(caps[1]) + "SupportGems").Div(utils.Float(caps[0]))}
		},
	},
	`per socketed (\w+) support gem`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier(firstToUpper(caps[0]) + "SupportGems")}
		},
	},
	"for each equipped normal item":                     {tag: mod.Multiplier("NormalItem")},
	"for each normal item equipped":                     {tag: mod.Multiplier("NormalItem")},
	"for each normal item you have equipped":            {tag: mod.Multiplier("NormalItem")},
	"for each equipped magic item":                      {tag: mod.Multiplier("MagicItem")},
	"for each magic item equipped":                      {tag: mod.Multiplier("MagicItem")},
	"for each magic item you have equipped":             {tag: mod.Multiplier("MagicItem")},
	"for each equipped rare item":                       {tag: mod.Multiplier("RareItem")},
	"for each rare item equipped":                       {tag: mod.Multiplier("RareItem")},
	"for each rare item you have equipped":              {tag: mod.Multiplier("RareItem")},
	"for each equipped unique item":                     {tag: mod.Multiplier("UniqueItem")},
	"for each unique item equipped":                     {tag: mod.Multiplier("UniqueItem")},
	"for each unique item you have equipped":            {tag: mod.Multiplier("UniqueItem")},
	"for each corrupted item equipped":                  {tag: mod.Multiplier("CorruptedItem")},
	"for each equipped corrupted item":                  {tag: mod.Multiplier("CorruptedItem")},
	"for each uncorrupted item equipped":                {tag: mod.Multiplier("NonCorruptedItem")},
	"per equipped claw":                                 {tag: mod.Multiplier("ClawItem")},
	"per equipped dagger":                               {tag: mod.Multiplier("DaggerItem")},
	"per equipped axe":                                  {tag: mod.Multiplier("AxeItem")},
	"per equipped ring":                                 {tag: mod.Multiplier("RingItem")},
	"per equipped flask":                                {tag: mod.Multiplier("FlaskItem")},
	"per equipped sword":                                {tag: mod.Multiplier("SwordItem")},
	"per equipped jewel":                                {tag: mod.Multiplier("JewelItem")},
	"per equipped mace":                                 {tag: mod.Multiplier("MaceItem")},
	"per equipped sceptre":                              {tag: mod.Multiplier("SceptreItem")},
	"per equipped wand":                                 {tag: mod.Multiplier("WandItem")},
	"per claw":                                          {tag: mod.Multiplier("ClawItem")},
	"per dagger":                                        {tag: mod.Multiplier("DaggerItem")},
	"per axe":                                           {tag: mod.Multiplier("AxeItem")},
	"per ring":                                          {tag: mod.Multiplier("RingItem")},
	"per flask":                                         {tag: mod.Multiplier("FlaskItem")},
	"per sword":                                         {tag: mod.Multiplier("SwordItem")},
	"per jewel":                                         {tag: mod.Multiplier("JewelItem")},
	"per mace":                                          {tag: mod.Multiplier("MaceItem")},
	"per sceptre":                                       {tag: mod.Multiplier("SceptreItem")},
	"per wand":                                          {tag: mod.Multiplier("WandItem")},
	"per abyssa?l? jewel affecting you":                 {tag: mod.Multiplier("AbyssJewel")},
	"for each herald b?u?f?f?s?k?i?l?l? ?affecting you": {tag: mod.Multiplier("Herald")},
	"for each of your aura or herald skills affecting you": {tag: mod.Multiplier("Herald", "AuraAffectingSelf")},
	"for each type of abyssa?l? jewel affecting you":       {tag: mod.Multiplier("AbyssJewelType")},
	`per (.+) eye jewel affecting you, up to a maximum of \+?(\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier(firstToUpper(caps[0]) + "EyeJewel").Limit(utils.Float(caps[1])).LimitTotal(true)}
		},
	},
	"per sextant affecting the area": {tag: mod.Multiplier("Sextant")},
	"per buff on you":                {tag: mod.Multiplier("BuffOnSelf")},
	"per hit suppressed recently":    {tag: mod.Multiplier("HitsSuppressedRecently")},
	"per curse on enemy":             {tag: mod.Multiplier("CurseOnEnemy")},
	"per curse on target":            {tag: mod.Multiplier("CurseOnEnemy")},
	"for each curse on enemy":        {tag: mod.Multiplier("CurseOnEnemy")},
	"for each curse on the enemy":    {tag: mod.Multiplier("CurseOnEnemy")},
	"per curse on you":               {tag: mod.Multiplier("CurseOnSelf")},
	"per poison on you":              {tag: mod.Multiplier("PoisonStacks")},
	"for each poison on you":         {tag: mod.Multiplier("PoisonStacks")},
	`for each poison on you up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonStacks").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`per poison on you, up to (\d+) per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonStacks").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	"for each poison you have inflicted recently": {tag: mod.Multiplier("PoisonAppliedRecently")},
	"per withered debuff on enemy":                {tag: mod.Multiplier("WitheredStack").Limit(15).Actor("enemy")},
	`for each poison you have inflicted recently, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonAppliedRecently").GlobalLimit(utils.Float(caps[0])).GlobalLimitKey("NoxiousStrike")}
		},
	},
	`for each time you have shocked a non\-shocked enemy recently, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ShockedNonShockedEnemyRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	"for each shocked enemy you've killed recently": {tag: mod.Multiplier("ShockedEnemyKilledRecently")},
	`per enemy killed recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("EnemyKilledRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`per (\d+) rampage kills`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("Rampage").Div(utils.Float(caps[0])).Limit((1000 / utils.Float(caps[0]))).LimitTotal(true)}
		},
	},
	`per minion, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("SummonedMinion").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	"per minion": {tag: mod.Multiplier("SummonedMinion")},
	"for each different type of persistent minion in your presence": {tag: mod.Multiplier("PersistentMinionTypes")},
	`for each minion in your presence, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("MinionPresenceCount").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each type of ammunition fired in the past (\d+) seconds`: {tag: mod.Multiplier("DifferentAmmoFired").LimitVar("AmmoTypes")},
	`for every different grenade fired in the past (\d+) seconds`: {tag: mod.Multiplier("DifferentGrenadeFired").LimitVar("GrenadeTypes")},
	`for each enemy you or your minions have killed recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("EnemyKilledRecently", "EnemyKilledByMinionsRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each enemy you or your minions have killed recently, up to (\d+)% per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("EnemyKilledRecently", "EnemyKilledByMinionsRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each (\d+) total mana y?o?u? ?h?a?v?e? ?spent recently`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ManaSpentRecently").Div(utils.Float(caps[0]))}
		},
	},
	`for each (\d+) total mana you have spent recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ManaSpentRecently").Div(utils.Float(caps[0])).Limit(utils.Float(caps[1])).LimitTotal(true)}
		},
	},
	`per (\d+) mana spent recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ManaSpentRecently").Div(utils.Float(caps[0])).Limit(utils.Float(caps[1])).LimitTotal(true)}
		},
	},
	`per (\d+) life spent in the past 8 seconds`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("LifeSpentPast8s").Div(utils.Float(caps[0]))}
		},
	},
	`per (\d+) life spent in the past 4 seconds`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("LifeSpentRecently").Div(utils.Float(caps[0]))}
		},
	},
	"for each time you've blocked in the past 10 seconds": {tag: mod.Multiplier("BlockedPast10Sec")},
	"per enemy killed by you or your totems recently":     {tag: mod.Multiplier("EnemyKilledRecently", "EnemyKilledByTotemsRecently")},
	`per nearby enemy, up to \+?(\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("NearbyEnemies").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	"per enemy in close range":                              {tagList: []mod.Tag{mod.Condition("AtCloseRange"), mod.Multiplier("NearbyEnemies")}},
	"to you and allies":                                     {},
	"for each impale on enemy":                              {tag: mod.Multiplier("ImpaleStacks").Actor("enemy")},
	"per impale on enemy":                                   {tag: mod.Multiplier("ImpaleStacks").Actor("enemy")},
	"per animated weapon":                                   {tag: mod.Multiplier("AnimatedWeapon").Actor("parent")},
	"per grasping vine":                                     {tag: mod.Multiplier("GraspingVinesCount")},
	"per fragile regrowth":                                  {tag: mod.Multiplier("FragileRegrowthCount")},
	"per allocated mastery passive skill":                   {tag: mod.Multiplier("AllocatedMastery")},
	"per allocated notable passive skill":                   {tag: mod.Multiplier("AllocatedNotable")},
	"for each connected notable passive skill allocated":    {tag: mod.Multiplier("AllocatedConnectedNotable")},
	"for each different type of mastery you have allocated": {tag: mod.Multiplier("AllocatedMasteryType")},
	"per s?o?c?k?e?t?e?d? ?grand spectrum":                  {tag: mod.Multiplier("GrandSpectrum")},
	`per second you've been stationary, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("StationarySeconds").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	"per elemental ailment you've inflicted recently": {tag: mod.Multiplier("AppliedAilmentsRecently")},
	"per tailwind": {tag: mod.Multiplier("Tailwind")},
	`per (\d+)% of maximum mana they reserve`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ManaReservedPercent")}
		},
	},
	"per strength":     {tag: mod.PerStat(1, "Str")},
	"per dexterity":    {tag: mod.PerStat(1, "Dex")},
	"per intelligence": {tag: mod.PerStat(1, "Int")},
	`per (\d+) strength`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Str")}
		},
	},
	`per (\d+) dexterity`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Dex")}
		},
	},
	`per (\d+) intelligence`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Int")}
		},
	},
	`for every (\d+) intelligence`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Int")}
		},
	},
	`per (\d+) total attributes`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Str", "Dex", "Int")}
		},
	},
	`per (\d+) of your lowest attribute`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LowestAttribute")}
		},
	},
	`per (\d+) reserved life`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LifeReserved")}
		},
	},
	`per (\d+) unreserved maximum mana`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ManaUnreserved")}
		},
	},
	`per (\d+) unreserved maximum mana, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ManaUnreserved").Limit(utils.Float(caps[1])).LimitTotal(true)}
		},
	},
	`per (\d+) armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Armour")}
		},
	},
	`per (\d+) evasion rating`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Evasion")}
		},
	},
	`per (\d+) evasion rating, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Evasion").Limit(utils.Float(caps[1])).LimitTotal(true)}
		},
	},
	`per (\d+) maximum energy shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShield")}
		},
	},
	`per (\d+) maximum life`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Life")}
		},
	},
	`per (\d+) life`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Life")}
		},
	},
	`per (\d+) current life`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LifeUnreserved")}
		},
	},
	`per (\d+) of maximum life or maximum mana, whichever is lower`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LowestOfMaximumLifeAndMaximumMana")}
		},
	},
	`per (\d+) player maximum life`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Life").Actor("parent")}
		},
	},
	`per (\d+) life cost`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LifeCost")}
		},
	},
	`per (\d+) runic ward cost`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "WardCost")}
		},
	},
	`per (\d+) maximum runic ward`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Ward")}
		},
	},
	`per (\d+) maximum mana`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Mana")}
		},
	},
	`per (\d+) maximum mana, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Mana").Limit(utils.Float(caps[1])).LimitTotal(true)}
		},
	},
	`per (\d+) maximum mana, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Mana").Limit(utils.Float(caps[1])).LimitTotal(true)}
		},
	},
	`equal to (\d+)% of maximum life`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PercentStat("Life", utils.Float(caps[0]))}
		},
	},
	`equal to (\d+)% of (y?o?u?r? ?) maximum mana`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PercentStat("Mana", utils.Float(caps[0]))}
		},
	},
	`equal to (\d+)% of (i?t?e?m? ?)armour on equipped body armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PercentStat("ArmourOnBody Armour", utils.Float(caps[0]))}
		},
	},
	`per (\d+) spirit`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Spirit")}
		},
	},
	`per (\d+) spirit, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Spirit").Limit(utils.Float(caps[1])).LimitTotal(true)}
		},
	},
	`per (\d+) unreserved darkness`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "UnreservedDarkness")}
		},
	},
	`per (\d+) accuracy rating`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Accuracy")}
		},
	},
	`per (\d+) accuracy rating, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Accuracy").Limit(utils.Float(caps[1])).LimitTotal(true)}
		},
	},
	`per (\d+)% block chance`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "BlockChance")}
		},
	},
	`per (\d+)% chance to block`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "BlockChance")}
		},
	},
	`per (\d+)% chance to block on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ShieldBlockChance")}
		},
	},
	`per (\d+)% chance to block attack damage`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "BlockChance")}
		},
	},
	`per (\d+)% chance to block spell damage`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "SpellBlockChance")}
		},
	},
	`per (\d+) of the lowest of armour and evasion rating`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LowestOfArmourAndEvasion")}
		},
	},
	`per (\d+) maximum energy shield on equipped helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnHelmet")}
		},
	},
	`per (\d+) (i?t?e?m? ?)energy shield on equipped helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnHelmet")}
		},
	},
	`per (\d+) maximum energy shield on helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnHelmet")}
		},
	},
	`per (\d+) item evasion rating on equipped helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnHelmet")}
		},
	},
	`per (\d+) (i?t?e?m? ?)evasion rating on body armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBody Armour")}
		},
	},
	`per (\d+) (i?t?e?m? ?)evasion r?a?t?i?n?g? ?on equipped body armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBody Armour")}
		},
	},
	`for every (\d+) (i?t?e?m? ?)evasion rating on equipped body armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBody Armour")}
		},
	},
	`per (\d+) (i?t?e?m? ?)evasion rating on equipped armour items`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnAllArmourItems")}
		},
	},
	`per (\d+) t?o?t?a?l? ?(i?t?e?m? ?)armour on equipped armour items`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnAllArmourItems")}
		},
	},
	`per (\d+) (i?t?e?m? ?)energy shield on equipped armour items`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnAllArmourItems")}
		},
	},
	`for every (\d+) (i?t?e?m? ?)energy shield on equipped body armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnBody Armour")}
		},
	},
	`per (\d+) (i?t?e?m? ?)armour on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tagList: []mod.Tag{mod.PerStat(utils.Float(caps[0]), "ArmourOnWeapon 2"), mod.Condition("UsingShield")}}
		},
	},
	`per (\d+) (i?t?e?m? ?)armour [ao][nr]d? evasion r?a?t?i?n?g? ?on shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tagList: []mod.Tag{mod.PerStat(utils.Float(caps[0]), "ArmourOnWeapon 2", "EvasionOnWeapon 2"), mod.Condition("UsingShield")}}
		},
	},
	`per (\d+) (i?t?e?m? ?)armour [ao][nr]d? evasion r?a?t?i?n?g? ?on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tagList: []mod.Tag{mod.PerStat(utils.Float(caps[0]), "ArmourOnWeapon 2", "EvasionOnWeapon 2"), mod.Condition("UsingShield")}}
		},
	},
	`per (\d+) (i?t?e?m? ?)evasion r?a?t?i?n?g? ?on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tagList: []mod.Tag{mod.PerStat(utils.Float(caps[0]), "EvasionOnWeapon 2"), mod.Condition("UsingShield")}}
		},
	},
	`per (\d+) maximum energy shield on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tagList: []mod.Tag{mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnWeapon 2"), mod.Condition("UsingShield")}}
		},
	},
	`per (\d+) maximum energy shield on shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tagList: []mod.Tag{mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnWeapon 2"), mod.Condition("UsingShield")}}
		},
	},
	`per (\d+) (i?t?e?m? ?)energy shield on equipped focus`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tagList: []mod.Tag{mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnWeapon 2"), mod.Condition("UsingFocus")}}
		},
	},
	`per (\d+) (i?t?e?m? ?)energy shield on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tagList: []mod.Tag{mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnWeapon 2"), mod.Condition("UsingShield")}}
		},
	},
	`per (\d+) (i?t?e?m? ?)energy shield on equipped boots`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnBoots")}
		},
	},
	`per (\d+) (i?t?e?m? ?)evasion on equipped boots`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBoots")}
		},
	},
	`per (\d+) (i?t?e?m? ?)evasion on boots`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBoots")}
		},
	},
	`per (\d+) (i?t?e?m? ?)armour on equipped gloves`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnGloves")}
		},
	},
	`per (\d+) (i?t?e?m? ?)armour on equipped helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnHelmet")}
		},
	},
	`per (\d+) (i?t?e?m? ?)armour on gloves`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnHelmet")}
		},
	},
	"for each of your empty charm slots": {tag: mod.PerStat(1, "EmptyCharms")},
	`per (\d+)% (\w+) resistance`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), firstToUpper(caps[1])+"Resist")}
		},
	},
	`per (\d+)% (\w+) resistance above 75%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), firstToUpper(caps[1])+"ResistOver75")}
		},
	},
	`per (\d+) devotion`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Devotion").Actor("parent")}
		},
	},
	`per (\d+) tribute`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Tribute").Actor("parent")}
		},
	},
	`per (\d+)% missing fire resistance, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "MissingFireResist").GlobalLimit(utils.Float(caps[1])).GlobalLimitKey("ReplicaNebulisFire")}
		},
	},
	`per (\d+)% missing cold resistance, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "MissingColdResist").GlobalLimit(utils.Float(caps[1])).GlobalLimitKey("ReplicaNebulisCold")}
		},
	},
	"per endurance, frenzy or power charge": {tag: mod.PerStat(1, "TotalCharges")},
	"per fortification":                     {tag: mod.PerStat(1, "FortificationStacks")},
	"per totem":                             {tag: mod.PerStat(1, "TotemsSummoned")},
	"per summoned totem":                    {tag: mod.PerStat(1, "TotemsSummoned")},
	"per summoned totem in your presence":   {tag: mod.PerStat(1, "TotemsSummoned")},
	"for each summoned totem":               {tag: mod.PerStat(1, "TotemsSummoned")},
	"for each time they have chained":       {tag: mod.PerStat(1, "Chain")},
	"for each time it has chained":          {tag: mod.PerStat(1, "Chain")},
	"for each summoned golem":               {tag: mod.PerStat(1, "ActiveGolemLimit")},
	"for each golem you have summoned":      {tag: mod.PerStat(1, "ActiveGolemLimit")},
	"per summoned golem":                    {tag: mod.PerStat(1, "ActiveGolemLimit")},
	"per summoned sentinel of purity":       {tag: mod.PerStat(1, "ActiveSentinelOfPurityLimit")},
	"per summoned skeleton":                 {tag: mod.PerStat(1, "ActiveSkeletonLimit")},
	"per skeleton you own":                  {tag: mod.PerStat(1, "ActiveSkeletonLimit").Actor("parent")},
	"per summoned raging spirit":            {tag: mod.PerStat(1, "ActiveRagingSpiritLimit")},
	"for each raised zombie":                {tag: mod.PerStat(1, "ActiveZombieLimit")},
	"per zombie you own":                    {tag: mod.PerStat(1, "ActiveZombieLimit").Actor("parent")},
	"per raised zombie":                     {tag: mod.PerStat(1, "ActiveZombieLimit")},
	"per raised spectre":                    {tag: mod.PerStat(1, "ActiveSpectreLimit")},
	"per spectre you own":                   {tag: mod.PerStat(1, "ActiveSpectreLimit").Actor("parent")},
	"for each remaining chain":              {tag: mod.PerStat(1, "ChainRemaining")},
	"for each enemy pierced":                {tag: mod.PerStat(1, "PiercedCount")},
	"for each time they've pierced":         {tag: mod.PerStat(1, "PiercedCount")},
	`with (\d+) or more strength`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Str", utils.Float(caps[0]))}
		},
	},
	`with at least (\d+) strength`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Str", utils.Float(caps[0]))}
		},
	},
	`w?h?i[lf]e? you have at least (\d+) strength`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Str", utils.Float(caps[0]))}
		},
	},
	`w?h?i[lf]e? you have at least (\d+) dexterity`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Dex", utils.Float(caps[0]))}
		},
	},
	`w?h?i[lf]e? you have at least (\d+) intelligence`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Int", utils.Float(caps[0]))}
		},
	},
	`w?h?i[lf]e? strength is below (\d+)`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Str", (utils.Float(caps[0]) - 1)).Upper(true)}
		},
	},
	`w?h?i[lf]e? dexterity is below (\d+)`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Dex", (utils.Float(caps[0]) - 1)).Upper(true)}
		},
	},
	`w?h?i[lf]e? intelligence is below (\d+)`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Int", (utils.Float(caps[0]) - 1)).Upper(true)}
		},
	},
	`at least (\d+) intelligence`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Int", utils.Float(caps[0]))}
		},
	},
	"if dexterity is higher than intelligence": {tag: mod.Condition("DexHigherThanInt")},
	"if strength is higher than intelligence":  {tag: mod.Condition("StrHigherThanInt")},
	`w?h?i[lf]e? you have at least (\d+) maximum energy shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("EnergyShield", utils.Float(caps[0]))}
		},
	},
	"against targets they pierce": {tag: mod.StatThreshold("PierceCount", 1)},
	"against pierced targets":     {tag: mod.StatThreshold("PierceCount", 1)},
	"to targets they pierce":      {tag: mod.StatThreshold("PierceCount", 1)},
	`w?h?i[lf]e? you have at least (\d+) devotion`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Devotion", utils.Float(caps[0]))}
		},
	},
	`w?h?i[lf]e? you have at least (\d+) tribute`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Tribute", utils.Float(caps[0]))}
		},
	},
	`while you have at least (\d+) rage`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("Rage").Threshold(utils.Float(caps[0]))}
		},
	},
	"while affected by a unique abyss jewel":                   {tag: mod.MultiplierThreshold("UniqueAbyssJewels").Threshold(1)},
	"while affected by a rare abyss jewel":                     {tag: mod.MultiplierThreshold("RareAbyssJewels").Threshold(1)},
	"while affected by a magic abyss jewel":                    {tag: mod.MultiplierThreshold("MagicAbyssJewels").Threshold(1)},
	"while affected by a normal abyss jewel":                   {tag: mod.MultiplierThreshold("NormalAbyssJewels").Threshold(1)},
	"while an enemy with an open weakness is in your presence": {tag: mod.Condition("OpenWeaknessEnemyPresence")},
	"against enemies with an open weakness":                    {tag: mod.Condition("EnemyHasOpenWeakness")},
	//pob2:block:src/Modules/ModParser.lua:1707-1708
	"with skills that cost life": {tag: mod.StatThreshold("", 1).Stats("LifeCost", "LifePerSecondCost")},
	"with spells that cost life": {keywordFlags: KeywordFlagSpell, tag: mod.StatThreshold("", 1).Stats("LifeCost", "LifePerSecondCost")},
	//pob2:block:src/Modules/ModParser.lua:1710-2174
	`if you have at least (\d+) (\w+) support gems socketed`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold(firstToUpper(caps[1]) + "SupportGems").Threshold(utils.Float(caps[0]))}
		},
	},
	`if you have at least (\d+) red, green and blue support gems socketed`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tagList: []mod.Tag{mod.MultiplierThreshold("RedSupportGems").Threshold(utils.Float(caps[0])), mod.MultiplierThreshold("GreenSupportGems").Threshold(utils.Float(caps[0])), mod.MultiplierThreshold("BlueSupportGems").Threshold(utils.Float(caps[0]))}}
		},
	},
	"when in main hand":          {tag: mod.SlotNumber(int(1))},
	"when in off hand":           {tag: mod.SlotNumber(int(2))},
	"in main hand":               {tag: mod.InSlot(int(1))},
	"in off hand":                {tag: mod.InSlot(int(2))},
	"w?i?t?h? main hand":         {tagList: []mod.Tag{mod.Condition("MainHandAttack"), mod.SkillType("Attack")}},
	"w?i?t?h? off hand":          {tagList: []mod.Tag{mod.Condition("OffHandAttack"), mod.SkillType("Attack")}},
	"with this weapon":           {flags: ModFlagWeapon, tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType("NonWeaponAttack").Neg(true)}},
	"inflicted with this weapon": {flags: ModFlagWeapon, tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType("NonWeaponAttack").Neg(true)}},
	"from hits with this weapon": {flags: ModFlagWeapon, tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType("NonWeaponAttack").Neg(true)}},
	`if you have a (\w+) (\w+) in (\w+) slot`: {
		fn: func(caps []string) modNameListType {
			name := utils.Capital(caps[0])
			name += "ItemIn"
			name += utils.Capital(caps[1])
			name += " "
			if caps[2] == "left" {
				name += "1"
			} else {
				name += "2"
			}
			return modNameListType{tag: mod.Condition(name)}
		},
	},
	"of skills supported by spellslinger": {tag: mod.Condition("SupportedBySpellslinger")},
	`while holding a (\w+)`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Condition("Using" + firstToUpper(caps[0]))}
		},
	},
	`while holding a (\w+) or (\w+)`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Condition("Using"+firstToUpper(caps[0]), "Using"+firstToUpper(caps[1]))}
		},
	},
	"while off hand is empty":                   {tag: mod.Condition("OffHandIsEmpty")},
	"with shields":                              {tag: mod.Condition("UsingShield")},
	"while dual wielding":                       {tag: mod.Condition("DualWielding")},
	"while dual wielding claws":                 {tag: mod.Condition("DualWieldingClaws")},
	"while dual wielding or holding a shield":   {tag: mod.Condition("DualWielding", "UsingShield")},
	"while wielding an axe":                     {tag: mod.Condition("UsingAxe")},
	"while wielding an axe or sword":            {tag: mod.Condition("UsingAxe", "UsingSword")},
	"while wielding a bow":                      {tag: mod.Condition("UsingBow")},
	"while wielding a claw":                     {tag: mod.Condition("UsingClaw")},
	"while wielding a dagger":                   {tag: mod.Condition("UsingDagger")},
	"while wielding a claw or dagger":           {tag: mod.Condition("UsingClaw", "UsingDagger")},
	"while wielding a mace":                     {tag: mod.Condition("UsingMace")},
	"while wielding a staff":                    {tag: mod.Condition("UsingStaff")},
	"while wielding a quarterstaff":             {tag: mod.Condition("UsingStaff")},
	"while wielding a sword":                    {tag: mod.Condition("UsingSword")},
	"while wielding a melee weapon":             {tag: mod.Condition("UsingMeleeWeapon")},
	"while wielding a one handed weapon":        {tag: mod.Condition("UsingOneHandedWeapon")},
	"while wielding a two handed weapon":        {tag: mod.Condition("UsingTwoHandedWeapon")},
	"while wielding a two handed melee weapon":  {tagList: []mod.Tag{mod.Condition("UsingTwoHandedWeapon"), mod.Condition("UsingMeleeWeapon")}},
	"while wielding a wand":                     {tag: mod.Condition("UsingWand")},
	"while wielding two different weapon types": {tag: mod.Condition("WieldingDifferentWeaponTypes")},
	"while unarmed":                             {tag: mod.Condition("Unarmed")},
	"while you are unencumbered":                {tag: mod.Condition("Unencumbered")},
	"while surrounded":                          {tag: mod.Condition("Surrounded")},
	"equipped bow":                              {tag: mod.Condition("UsingBow")},
	`if equipped ([\w\s]+) has an ([\w\s]+) modifier`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ItemCondition(caps[0]).Search(caps[1])}
		},
	},
	`if there are no ([\w\s]+) modifiers on equipped ([\w\s]+)`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ItemCondition(caps[1]).Search(caps[0]).Negate(true)}
		},
	},
	`if there are no (\w+) modifiers on other equipped items`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ItemCondition("{SlotName}").Search(caps[0]).Negate(true).WithAllSlots().WithExcludeSelf()}
		},
	},
	"if corrupted":                        {tag: mod.ItemCondition("{SlotName}").Corrupted(true)},
	"with a normal item equipped":         {tag: mod.MultiplierThreshold("NormalItem").Threshold(1)},
	"with a magic item equipped":          {tag: mod.MultiplierThreshold("MagicItem").Threshold(1)},
	"with a rare item equipped":           {tag: mod.MultiplierThreshold("RareItem").Threshold(1)},
	"with a unique item equipped":         {tag: mod.MultiplierThreshold("UniqueItem").Threshold(1)},
	"if you wear no corrupted items":      {tag: mod.MultiplierThreshold("CorruptedItem").Threshold(0).Upper(true)},
	"if no worn items are corrupted":      {tag: mod.MultiplierThreshold("CorruptedItem").Threshold(0).Upper(true)},
	"if no equipped items are corrupted":  {tag: mod.MultiplierThreshold("CorruptedItem").Threshold(0).Upper(true)},
	"if all worn items are corrupted":     {tag: mod.MultiplierThreshold("NonCorruptedItem").Threshold(0).Upper(true)},
	"if all equipped items are corrupted": {tag: mod.MultiplierThreshold("NonCorruptedItem").Threshold(0).Upper(true)},
	`if equipped shield has at least (\d+)% chance to block`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("ShieldBlockChance", utils.Float(caps[0]))}
		},
	},
	`if you have (\d+) primordial items socketed or equipped`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("PrimordialItem").Threshold(utils.Float(caps[0]))}
		},
	},
	"if equipped helmet, body armour, gloves, and boots all have armour":         {tagList: []mod.Tag{mod.StatThreshold("ArmourOnHelmet", 1), mod.StatThreshold("ArmourOnBody Armour", 1), mod.StatThreshold("ArmourOnGloves", 1), mod.StatThreshold("ArmourOnBoots", 1)}},
	"if equipped helmet, body armour, gloves, and boots all have evasion rating": {tagList: []mod.Tag{mod.StatThreshold("EvasionOnHelmet", 1), mod.StatThreshold("EvasionOnBody Armour", 1), mod.StatThreshold("EvasionOnGloves", 1), mod.StatThreshold("EvasionOnBoots", 1)}},
	"if you have shapeshifted to an animal form recently":                        {tag: mod.Condition("ShapeshiftToAnimal")},
	"if you have shapeshifted to human form recently":                            {tag: mod.Condition("ShapeshiftToHuman")},
	"if you have consumed an elemental infusion recently":                        {tag: mod.Condition("InfusionConsumedRecently")},
	`if used while y?o?u?\s?a?r?e?\s?on low life`:                                {tag: mod.Condition("LowLife")},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?on low life`:                                 {tag: mod.Condition("LowLife")},
	"on reaching low life":                                                       {tag: mod.Condition("LowLife")},
	"while you are missing runic ward":                                           {tag: mod.Condition("MissingRunicWard")},
	"while missing runic ward":                                                   {tag: mod.Condition("MissingRunicWard")},
	"while you have no runic ward":                                               {tag: mod.Condition("NoRunicWard")},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?on low runic ward`:                           {tag: mod.Condition("LowRunicWard")},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?not on low life`:                             {tag: mod.Condition("LowLife").Neg(true)},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?on low mana`:                                 {tag: mod.Condition("LowMana")},
	`if y?o?u?\s?a?r?e?\s?on low mana`:                                           {tag: mod.Condition("LowMana")},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?not on low mana`:                             {tag: mod.Condition("LowMana").Neg(true)},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?on full life`:                                {tag: mod.Condition("FullLife")},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?on full mana`:                                {tag: mod.Condition("FullMana")},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?not on full life`:                            {tag: mod.Condition("FullLife").Neg(true)},
	"wh[ie][ln]e? no life is reserved":                                           {tag: mod.StatThreshold("LifeReserved", 0).Upper(true)},
	"wh[ie][ln]e? no mana is reserved":                                           {tag: mod.StatThreshold("ManaReserved", 0).Upper(true)},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?on low energy shield`:                        {tag: mod.Condition("LowEnergyShield")},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?on full energy shield`:                       {tag: mod.Condition("FullEnergyShield")},
	`wh[ie][ln]e? y?o?u?\s?a?r?e?\s?not on full energy shield`:                   {tag: mod.Condition("FullEnergyShield").Neg(true)},
	"wh[ie][ln]e? you have energy shield":                                        {tag: mod.Condition("HaveEnergyShield")},
	"wh[ie][ln]e? you have no energy shield":                                     {tag: mod.Condition("HaveEnergyShield").Neg(true)},
	"if you have energy shield":                                                  {tag: mod.Condition("HaveEnergyShield")},
	"if you have no energy shield":                                               {tag: mod.Condition("HaveEnergyShield").Neg(true)},
	"while stationary":                                                           {tag: mod.Condition("Stationary")},
	"while you are stationary":                                                   {tag: mod.ActorCondition("player", "Stationary")},
	"while moving":                                                               {tag: mod.Condition("Moving")},
	"while sprinting":                                                            {tag: mod.Condition("Sprinting")},
	"while channelling":                                                          {tag: mod.Condition("Channelling")},
	"while channelling snipe":                                                    {tag: mod.Condition("Channelling")},
	`after channelling for (\d+) seconds?`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("ChannellingTime").Threshold(utils.Float(caps[0]))}
		},
	},
	`if you've been channelling for at least (\d+) seconds?`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("ChannellingTime").Threshold(utils.Float(caps[0]))}
		},
	},
	"if you've inflicted exposure recently": {tag: mod.Condition("AppliedExposureRecently")},
	"while you have no power charges":       {tag: mod.StatThreshold("PowerCharges", 0).Upper(true)},
	"while you have no frenzy charges":      {tag: mod.StatThreshold("FrenzyCharges", 0).Upper(true)},
	"while you have no endurance charges":   {tag: mod.StatThreshold("EnduranceCharges", 0).Upper(true)},
	"while you have a power charge":         {tag: mod.StatThreshold("PowerCharges", 1)},
	"while you have a frenzy charge":        {tag: mod.StatThreshold("FrenzyCharges", 1)},
	"while you have an endurance charge":    {tag: mod.StatThreshold("EnduranceCharges", 1)},
	"while at maximum power charges":        {tag: mod.StatThresholdStat("PowerCharges", "PowerChargesMax")},
	"while at maximum frenzy charges":       {tag: mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax")},
	"while on full frenzy charges":          {tag: mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax")},
	"while at maximum endurance charges":    {tag: mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax")},
	"while at maximum rage":                 {tag: mod.Condition("HaveMaximumRage")},
	"while at maximum fortification":        {tag: mod.Condition("HaveMaximumFortification")},
	`while you have at least (\d+) crab barriers`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("CrabBarriers", utils.Float(caps[0]))}
		},
	},
	`while you have at least (\d+) fortification`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("FortificationStacks", utils.Float(caps[0]))}
		},
	},
	`while you have at least (\d+) total endurance, frenzy and power charges`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("TotalCharges").Threshold(utils.Float(caps[0]))}
		},
	},
	"while you have a totem":                                      {tag: mod.Condition("HaveTotem")},
	"while you have at least one nearby ally":                     {tag: mod.MultiplierThreshold("NearbyAlly").Threshold(1)},
	"while you have an ally in your presence":                     {tag: mod.MultiplierThreshold("NearbyAlly").Threshold(1)},
	"while your companion is in your presence":                    {tag: mod.Condition("CompanionInPresence")},
	"while you have a linked target":                              {tag: mod.MultiplierThreshold("LinkedTargets").Threshold(1)},
	"while you have fortify":                                      {tag: mod.Condition("Fortified")},
	"while you have phasing":                                      {tag: mod.Condition("Phasing")},
	"if you[' ]h?a?ve suppressed spell damage recently":           {tag: mod.Condition("SuppressedRecently")},
	"while you have elusive":                                      {tag: mod.Condition("Elusive")},
	"while physical aegis is depleted":                            {tag: mod.Condition("PhysicalAegisDepleted")},
	"during onslaught":                                            {tag: mod.Condition("Onslaught")},
	"while you have onslaught":                                    {tag: mod.Condition("Onslaught")},
	"while phasing":                                               {tag: mod.Condition("Phasing")},
	"while dodge rolling":                                         {tag: mod.Condition("InDodgeRoll")},
	"while you have tailwind":                                     {tag: mod.MultiplierThreshold("Tailwind").Threshold(1)},
	"while elusive":                                               {tag: mod.Condition("Elusive")},
	"gain elusive":                                                {tag: mod.Condition("CanBeElusive", "Elusive")},
	"while you have arcane surge":                                 {tag: mod.Condition("AffectedByArcaneSurge")},
	"while you have cat's stealth":                                {tag: mod.Condition("AffectedByCat'sStealth")},
	"while you have cat's agility":                                {tag: mod.Condition("AffectedByCat'sAgility")},
	"while you have avian's might":                                {tag: mod.Condition("AffectedByAvian'sMight")},
	"while you have avian's flight":                               {tag: mod.Condition("AffectedByAvian'sFlight")},
	"while affected by aspect of the cat":                         {tag: mod.Condition("AffectedByCat'sStealth", "AffectedByCat'sAgility")},
	`while affected by a non\-vaal guard skill`:                   {tag: mod.Condition("AffectedByNonVaalGuardSkill")},
	`if a non\-vaal guard buff was lost recently`:                 {tag: mod.Condition("LostNonVaalBuffRecently")},
	"while affected by a guard skill buff":                        {tag: mod.Condition("AffectedByGuardSkill")},
	"while affected by a herald":                                  {tag: mod.Condition("AffectedByHerald")},
	"while fortified":                                             {tag: mod.Condition("Fortified")},
	"while in blood stance":                                       {tag: mod.Condition("BloodStance")},
	"while in sand stance":                                        {tag: mod.Condition("SandStance")},
	"while you have a bestial minion":                             {tag: mod.Condition("HaveBestialMinion")},
	"while you have infusion":                                     {tag: mod.Condition("InfusionActive")},
	"while focus?sed":                                             {tag: mod.Condition("Focused")},
	"while leeching":                                              {tag: mod.Condition("Leeching")},
	"while leeching life":                                         {tag: mod.Condition("LeechingLife")},
	"while leeching energy shield":                                {tag: mod.Condition("LeechingEnergyShield")},
	"while leeching mana":                                         {tag: mod.Condition("LeechingMana")},
	"while you have an active charm":                              {tag: mod.Condition("UsingCharm")},
	"while using a flask":                                         {tag: mod.Condition("UsingFlask")},
	"during effect":                                               {tag: mod.Condition("UsingFlask")},
	"during flask effect":                                         {tag: mod.Condition("UsingFlask")},
	"during any flask effect":                                     {tag: mod.Condition("UsingFlask")},
	"while under no flask effects":                                {tag: mod.Condition("UsingFlask").Neg(true)},
	"during effect of any mana flask":                             {tag: mod.Condition("UsingManaFlask")},
	"during any mana flask effect":                                {tag: mod.Condition("UsingManaFlask")},
	"during effect of any life flask":                             {tag: mod.Condition("UsingLifeFlask")},
	"during any life flask effect":                                {tag: mod.Condition("UsingLifeFlask")},
	"if you've used a life flask in the past 10 seconds":          {tag: mod.Condition("UsingLifeFlask")},
	"if you've used a mana flask in the past 10 seconds":          {tag: mod.Condition("UsingManaFlask")},
	"if you've used a mana flask recently":                        {tag: mod.Condition("UsingManaFlask")},
	"while you have no life flask uses left":                      {tag: mod.Condition("NoLifeFlaskUsesLeft")},
	"during effect of any life or mana flask":                     {tag: mod.Condition("UsingManaFlask", "UsingLifeFlask")},
	"while on consecrated ground":                                 {tag: mod.Condition("OnConsecratedGround")},
	"while on caustic ground":                                     {tag: mod.Condition("OnCausticGround")},
	"when you create consecrated ground":                          {},
	"on burning ground":                                           {tag: mod.Condition("OnBurningGround")},
	"while on burning ground":                                     {tag: mod.Condition("OnBurningGround")},
	"on chilled ground":                                           {tag: mod.Condition("OnChilledGround")},
	"on shocked ground":                                           {tag: mod.Condition("OnShockedGround")},
	"while in a caustic cloud":                                    {tag: mod.Condition("OnCausticCloud")},
	"while blinded":                                               {tagList: []mod.Tag{mod.Condition("Blinded"), mod.Condition("CannotBeBlinded").Neg(true)}},
	"while burning":                                               {tag: mod.Condition("Burning")},
	"while ignited":                                               {tag: mod.Condition("Ignited")},
	"while you are ignited":                                       {tag: mod.Condition("Ignited")},
	"while on ignited ground":                                     {tag: mod.Condition("OnIgnitedGround")},
	"while chilled":                                               {tag: mod.Condition("Chilled")},
	"while you are chilled":                                       {tag: mod.Condition("Chilled")},
	"while on chilled ground":                                     {tag: mod.Condition("OnChilledGround")},
	"while frozen":                                                {tag: mod.Condition("Frozen")},
	"while shocked":                                               {tag: mod.Condition("Shocked")},
	"while you are shocked":                                       {tag: mod.Condition("Shocked")},
	"while on shocked ground":                                     {tag: mod.Condition("OnShockedGround")},
	"while you are bleeding":                                      {tag: mod.Condition("Bleeding")},
	"while not ignited, frozen or shocked":                        {tag: mod.Condition("Ignited", "Frozen", "Shocked").Neg(true)},
	"while bleeding":                                              {tag: mod.Condition("Bleeding")},
	"while poisoned":                                              {tag: mod.Condition("Poisoned")},
	"while you are poisoned":                                      {tag: mod.Condition("Poisoned")},
	"while cursed":                                                {tag: mod.Condition("Cursed")},
	"while not cursed":                                            {tag: mod.Condition("Cursed").Neg(true)},
	"while affected by any damaging ailment":                      {tag: mod.Condition("Poisoned", "Ignited", "Bleeding")},
	"while affected by an ailment":                                {tag: mod.Condition("Bleeding", "Poisoned", "Ignited", "Chilled", "Frozen", "Shocked", "Electrocuted")},
	"while there is only one nearby enemy":                        {tagList: []mod.Tag{mod.Multiplier("NearbyEnemies").Limit(1), mod.Condition("OnlyOneNearbyEnemy")}},
	"while t?h?e?r?e? ?i?s? ?a rare or unique enemy i?s? ?nearby": {tag: mod.ActorCondition("enemy", "NearbyRareOrUniqueEnemy", "RareOrUnique")},
	"while a rare or unique enemy is in your presence":            {tag: mod.ActorCondition("enemy", "NearbyRareOrUniqueEnemy", "RareOrUnique")},
	"against enemies that have exited your presence recently":     {tag: mod.ActorCondition("enemy", "ExitedPresenceRecently")},
	"against enemies that have entered your presence recently":    {tag: mod.ActorCondition("enemy", "EnteredPresenceRecently")},
	"if you[' ]h?a?ve been stunned recently":                      {tag: mod.Condition("StunnedRecently")},
	"if you[' ]h?a?ve been heavy stunned recently":                {tag: mod.Condition("StunnedRecently")},
	"if you haven't been stunned recently":                        {tag: mod.Condition("StunnedRecently").Neg(true)},
	"if you[' ]h?a?ve hit recently":                               {tag: mod.Condition("HitRecently")},
	"if you[' ]h?a?ve hit an enemy recently":                      {tag: mod.Condition("HitRecently")},
	"if you[' ]h?a?ve hit with your main hand weapon recently":    {tag: mod.Condition("HitRecentlyWithWeapon")},
	"if you[' ]h?a?ve hit with your off hand weapon recently":     {tagList: []mod.Tag{mod.Condition("HitRecentlyWithWeapon"), mod.Condition("DualWielding")}},
	"if you[' ]h?a?ve hit a cursed enemy recently":                {tagList: []mod.Tag{mod.Condition("HitRecently"), mod.ActorCondition("enemy", "Cursed")}},
	"if you[' ]h?a?ve dealt a projectile attack hit in the past eight seconds": {tag: mod.Condition("HitProjectileRecently")},
	"if you[' ]h?a?ve dealt a melee hit in the past eight seconds":             {tag: mod.Condition("HitMeleeRecently")},
	"if you[' ]h?a?ve hit an enemy with a melee attack recently":               {tag: mod.Condition("HitMeleeRecently")},
	"if you[' ]h?a?ve triggered a skill recently":                              {tag: mod.Condition("TriggeredSkillRecently")},
	"when you or your totems hit an enemy with a spell":                        {tag: mod.Condition("HitSpellRecently", "TotemsHitSpellRecently")},
	"on hit with spells":                                                            {tag: mod.Condition("HitSpellRecently")},
	"if you[' ]h?a?ve crit recently":                                                {tag: mod.Condition("CritRecently")},
	"if you[' ]h?a?ve dealt a critical hit recently":                                {tag: mod.Condition("CritRecently")},
	"when you deal a critical hit":                                                  {tag: mod.Condition("CritRecently")},
	"if you[' ]h?a?ve dealt a critical hit with this weapon recently":               {tag: mod.Condition("CritRecently")},
	"if you[' ]h?a?ve crit in the past 8 seconds":                                   {tag: mod.Condition("CritInPast8Sec")},
	"if you[' ]h?a?ve dealt a crit in the past 8 seconds":                           {tag: mod.Condition("CritInPast8Sec")},
	"if you[' ]h?a?ve dealt a critical hit in the past 8 seconds":                   {tag: mod.Condition("CritInPast8Sec")},
	"if you haven't crit recently":                                                  {tag: mod.Condition("CritRecently").Neg(true)},
	"if you haven't dealt a critical hit recently":                                  {tag: mod.Condition("CritRecently").Neg(true)},
	`if you[' ]h?a?ve dealt a non\-critical hit recently`:                           {tag: mod.Condition("NonCritRecently")},
	"if your skills have dealt a critical hit recently":                             {tag: mod.Condition("SkillCritRecently")},
	"if you dealt a critical hit with a herald skill recently":                      {tag: mod.Condition("CritWithHeraldSkillRecently")},
	"if you[' ]h?a?ve dealt a critical hit with a two handed melee weapon recently": {flags: ModFlagWeapon2H | ModFlagWeaponMelee, tag: mod.Condition("CritRecently")},
	"if you[' ]h?a?ve killed recently":                                              {tag: mod.Condition("KilledRecently")},
	"on killing taunted enemies":                                                    {tag: mod.Condition("KilledTauntedEnemyRecently")},
	"on kill":                                                                       {tag: mod.Condition("KilledRecently")},
	"on melee kill":                                                                 {flags: ModFlagWeaponMelee, tag: mod.Condition("KilledRecently")},
	"when you kill an enemy":                                                        {tag: mod.Condition("KilledRecently")},
	"if you[' ]h?a?ve killed an enemy recently":                                     {tag: mod.Condition("KilledRecently")},
	`if you[' ]h?a?ve killed at least (\d) enemies recently`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("EnemyKilledRecently").Threshold(utils.Float(caps[0]))}
		},
	},
	"if you haven't killed recently":                                              {tag: mod.Condition("KilledRecently").Neg(true)},
	"if you or your totems have killed recently":                                  {tag: mod.Condition("KilledRecently", "TotemsKilledRecently")},
	"if you[' ]h?a?ve thrown a trap or mine recently":                             {tag: mod.Condition("TrapOrMineThrownRecently")},
	"on throwing a trap":                                                          {tag: mod.Condition("TrapOrMineThrownRecently")},
	"if you[' ]h?a?ve killed a maimed enemy recently":                             {tagList: []mod.Tag{mod.Condition("KilledRecently"), mod.ActorCondition("enemy", "Maimed")}},
	"if you[' ]h?a?ve killed a cursed enemy recently":                             {tagList: []mod.Tag{mod.Condition("KilledRecently"), mod.ActorCondition("enemy", "Cursed")}},
	"if you[' ]h?a?ve killed a bleeding enemy recently":                           {tagList: []mod.Tag{mod.Condition("KilledRecently"), mod.ActorCondition("enemy", "Bleeding")}},
	"if you[' ]h?a?ve killed an enemy affected by your damage over time recently": {tag: mod.Condition("KilledAffectedByDotRecently")},
	"if you[' ]h?a?ve frozen an enemy recently":                                   {tag: mod.Condition("FrozenEnemyRecently")},
	"if you[' ]h?a?ve chilled an enemy recently":                                  {tag: mod.Condition("ChilledEnemyRecently")},
	"if you[' ]h?a?ve ignited an enemy recently":                                  {tag: mod.Condition("IgnitedEnemyRecently")},
	"if you[' ]h?a?ve shocked an enemy recently":                                  {tag: mod.Condition("ShockedEnemyRecently")},
	"if you[' ]h?a?ve stunned an enemy recently":                                  {tag: mod.Condition("StunnedEnemyRecently")},
	"if you[' ]h?a?ve pinned an enemy recently":                                   {tag: mod.Condition("PinnedEnemyRecently")},
	"if you[' ]h?a?ve stunned an enemy with a two handed melee weapon recently":   {tagList: []mod.Tag{mod.Condition("UsingTwoHandedWeapon"), mod.Condition("UsingMeleeWeapon"), mod.Condition("StunnedEnemyRecently")}},
	"if you[' ]h?a?ve been hit recently":                                          {tag: mod.Condition("BeenHitRecently")},
	"if you[' ]h?a?ve been hit by an attack recently":                             {tag: mod.Condition("BeenHitByAttackRecently")},
	"if you were hit recently":                                                    {tag: mod.Condition("BeenHitRecently")},
	"if you were damaged by a hit recently":                                       {tag: mod.Condition("BeenHitRecently")},
	"if you[' ]h?a?ve taken a critical hit recently":                              {tag: mod.Condition("BeenCritRecently")},
	"if you[' ]h?a?ve taken a savage hit recently":                                {tag: mod.Condition("BeenSavageHitRecently")},
	"if you have ?n[o']t been hit recently":                                       {tag: mod.Condition("BeenHitRecently").Neg(true)},
	"if you have ?n[o']t been hit by an attack recently":                          {tag: mod.Condition("BeenHitByAttackRecently").Neg(true)},
	"if you[' ]h?a?ve taken no damage from hits recently":                         {tag: mod.Condition("BeenHitRecently").Neg(true)},
	"if you[' ]h?a?ve taken fire damage from a hit recently":                      {tag: mod.Condition("HitByFireDamageRecently")},
	"if you[' ]h?a?ve taken fire damage from an enemy hit recently":               {tag: mod.Condition("TakenFireDamageFromEnemyHitRecently")},
	"if you[' ]h?a?ve taken spell damage recently":                                {tag: mod.Condition("HitBySpellDamageRecently")},
	"if you haven't taken damage recently":                                        {tag: mod.Condition("BeenHitRecently").Neg(true)},
	"if you[' ]h?a?ve blocked recently":                                           {tag: mod.Condition("BlockedRecently")},
	"if you[' ]h?a?ve successfully parried recently":                              {tag: mod.Condition("ParriedRecently")},
	"if you[' ]h?a?ve blocked with active block recently":                         {tag: mod.Condition("ActiveBlockedRecently")},
	"if you haven't blocked recently":                                             {tag: mod.Condition("BlockedRecently").Neg(true)},
	"if you[' ]h?a?ve blocked an attack recently":                                 {tag: mod.Condition("BlockedAttackRecently")},
	"if you[' ]h?a?ve blocked attack damage recently":                             {tag: mod.Condition("BlockedAttackRecently")},
	"if you[' ]h?a?ve blocked a spell recently":                                   {tag: mod.Condition("BlockedSpellRecently")},
	"if you[' ]h?a?ve blocked spell damage recently":                              {tag: mod.Condition("BlockedSpellRecently")},
	"if you[' ]h?a?ve blocked damage from a unique enemy in the past 10 seconds":  {tag: mod.Condition("BlockedHitFromUniqueEnemyInPast10Sec")},
	"if you[' ]h?a?ve attacked recently":                                          {tag: mod.Condition("AttackedRecently")},
	"if you have ?n[o']t attacked recently":                                       {tag: mod.Condition("AttackedRecently").Neg(true)},
	"if you[' ]h?a?ve cast a spell recently":                                      {tag: mod.Condition("CastSpellRecently")},
	"if you[' ]h?a?ve been stunned while casting recently":                        {tag: mod.Condition("StunnedWhileCastingRecently")},
	"if you[' ]h?a?ve consumed a corpse recently":                                 {tag: mod.Condition("ConsumedCorpseRecently")},
	"if you[' ]h?a?ve cursed an enemy recently":                                   {tag: mod.Condition("CursedEnemyRecently")},
	"if you[' ]h?a?ve cast a mark spell recently":                                 {tag: mod.Condition("CastMarkRecently")},
	"if you[' ]h?a?ve used a mark recently":                                       {tag: mod.Condition("CastMarkRecently")},
	"if you have ?n[o']t consumed a corpse recently":                              {tag: mod.Condition("ConsumedCorpseRecently").Neg(true)},
	"for each corpse consumed recently":                                           {tag: mod.Multiplier("CorpseConsumedRecently")},
	"if you[' ]h?a?ve taunted an enemy recently":                                  {tag: mod.Condition("TauntedEnemyRecently")},
	"if you[' ]h?a?ve used a skill recently":                                      {tag: mod.Condition("UsedSkillRecently")},
	"if you[' ]h?a?ve used a travel skill recently":                               {tag: mod.Condition("UsedTravelSkillRecently")},
	"if you[' ]h?a?ve reloaded recently":                                          {tag: mod.Condition("ReloadedRecently")},
	"on skill use":                                                                {},
	`for each skill you've used recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("SkillUsedRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each different non\-instant spell you[' ]h?a?ve cast recently`: {tag: mod.Multiplier("NonInstantSpellCastRecently")},
	"if you[' ]h?a?ve used a warcry recently":                           {tag: mod.Condition("UsedWarcryRecently")},
	"if you[' ]h?a?ve warcried recently":                                {tag: mod.Condition("UsedWarcryRecently")},
	"for each time you[' ]h?a?ve warcried recently":                     {tag: mod.Multiplier("WarcryUsedRecently")},
	"for each time you[' ]h?a?ve been stunned recently":                 {tag: mod.Multiplier("StunnedRecently")},
	"when you warcry": {tag: mod.Condition("UsedWarcryRecently")},
	"if you[' ]h?a?ve warcried in the past 8 seconds": {tag: mod.Condition("UsedWarcryInPast8Seconds")},
	`for each second you've been affected by a warcry buff, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("AffectedByWarcryBuffDuration").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each of your mines detonated recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("MineDetonatedRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`[fp][oe]r ?e?a?c?h? mine detonated recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("MineDetonatedRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`[fp][oe]r ?e?a?c?h? mine detonated recently, up to (\d+)% per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("MineDetonatedRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	"[fp][oe]r ?e?a?c?h? mine detonated recently": {tag: mod.Multiplier("MineDetonatedRecently")},
	`for each of your traps triggered recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("TrapTriggeredRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each trap triggered recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("TrapTriggeredRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each trap triggered recently, up to (\d+)% per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("TrapTriggeredRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	"if you[' ]h?a?ve used a fire skill recently":                    {tag: mod.Condition("UsedFireSkillRecently")},
	"if you[' ]h?a?ve used a cold skill recently":                    {tag: mod.Condition("UsedColdSkillRecently")},
	"if you[' ]h?a?ve used a fire skill in the past 10 seconds":      {tag: mod.Condition("UsedFireSkillInPast10Sec")},
	"if you[' ]h?a?ve used a cold skill in the past 10 seconds":      {tag: mod.Condition("UsedColdSkillInPast10Sec")},
	"if you[' ]h?a?ve used a lightning skill in the past 10 seconds": {tag: mod.Condition("UsedLightningSkillInPast10Sec")},
	"if you[' ]h?a?ve summoned a totem recently":                     {tag: mod.Condition("SummonedTotemRecently")},
	"when you summon a totem":                                        {tag: mod.Condition("SummonedTotemRecently")},
	"if you summoned a golem in the past 8 seconds":                  {tag: mod.Condition("SummonedGolemInPast8Sec")},
	"if you haven't summoned a totem in the past 2 seconds":          {tag: mod.Condition("NoSummonedTotemsInPastTwoSeconds")},
	"if you[' ]h?a?ve used a minion skill recently":                  {tag: mod.Condition("UsedMinionSkillRecently")},
	"if you[' ]h?a?ve used a movement skill recently":                {tag: mod.Condition("UsedMovementSkillRecently")},
	"if you haven't cast dash recently":                              {tag: mod.Condition("CastDashRecently").Neg(true)},
	"if you[' ]h?a?ve cast dash recently":                            {tag: mod.Condition("CastDashRecently")},
	"if you[' ]h?a?ve used a vaal skill recently":                    {tag: mod.Condition("UsedVaalSkillRecently")},
	"if you[' ]h?a?ve used a socketed vaal skill recently":           {tag: mod.Condition("UsedVaalSkillRecently")},
	"when you use a vaal skill":                                      {tag: mod.Condition("UsedVaalSkillRecently")},
	"if you haven't used a brand skill recently":                     {tag: mod.Condition("UsedBrandRecently").Neg(true)},
	"if you[' ]h?a?ve used a brand skill recently":                   {tag: mod.Condition("UsedBrandRecently")},
	"if you[' ]h?a?ve used a retaliation skill recently":             {tag: mod.Condition("UsedRetaliationRecently")},
	`if you[' ]h?a?ve spent (\d+) total mana recently`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("ManaSpentRecently").Threshold(utils.Float(caps[0]))}
		},
	},
	"if you[' ]h?a?ve spent life recently": {tag: mod.MultiplierThreshold("LifeSpentRecently").Threshold(1)},
	`for \d+ seconds after spending a total of (\d+) mana`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("ManaSpentRecently").Threshold(utils.Float(caps[0]))}
		},
	},
	"if you've impaled an enemy recently":                         {tag: mod.Condition("ImpaledRecently")},
	"if you've changed stance recently":                           {tag: mod.Condition("ChangedStanceRecently")},
	"if you've gained a power charge recently":                    {tag: mod.Condition("GainedPowerChargeRecently")},
	"if you haven't gained a power charge recently":               {tag: mod.Condition("GainedPowerChargeRecently").Neg(true)},
	"if you haven't gained a frenzy charge recently":              {tag: mod.Condition("GainedFrenzyChargeRecently").Neg(true)},
	"if you've stopped taking damage over time recently":          {tag: mod.Condition("StoppedTakingDamageOverTimeRecently")},
	"during soul gain prevention":                                 {tag: mod.Condition("SoulGainPrevention")},
	"if you detonated mines recently":                             {tag: mod.Condition("DetonatedMinesRecently")},
	"if you detonated a mine recently":                            {tag: mod.Condition("DetonatedMinesRecently")},
	"if you[' ]h?a?ve detonated a mine recently":                  {tag: mod.Condition("DetonatedMinesRecently")},
	"when your mine is detonated targeting an enemy":              {tag: mod.Condition("DetonatedMinesRecently")},
	"when your trap is triggered by an enemy":                     {tag: mod.Condition("TriggeredTrapsRecently")},
	"if energy shield recharge has started recently":              {tag: mod.Condition("EnergyShieldRechargeRecently")},
	"if energy shield recharge has started in the past 2 seconds": {tag: mod.Condition("EnergyShieldRechargePastTwoSec")},
	"when cast on frostbolt":                                      {tag: mod.Condition("CastOnFrostbolt")},
	"branded enemy's":                                             {tag: mod.MultiplierThreshold("BrandsAttachedToEnemy").Threshold(1)},
	"to enemies they're attached to":                              {tag: mod.MultiplierThreshold("BrandsAttachedToEnemy").Threshold(1)},
	`for each hit you've taken recently up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("BeenHitRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each time you've been hit by an enemy recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("BeenHitRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each nearby enemy, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("NearbyEnemies").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	"while you have iron reflexes":                 {tag: mod.Condition("HaveIronReflexes")},
	"while you do not have iron reflexes":          {tag: mod.Condition("HaveIronReflexes").Neg(true)},
	"while you have elemental overload":            {tag: mod.Condition("HaveElementalOverload")},
	"while you do not have elemental overload":     {tag: mod.Condition("HaveElementalOverload").Neg(true)},
	"while you have resolute technique":            {tag: mod.Condition("HaveResoluteTechnique")},
	"while you do not have resolute technique":     {tag: mod.Condition("HaveResoluteTechnique").Neg(true)},
	"while you have avatar of fire":                {tag: mod.Condition("HaveAvatarOfFire")},
	"while you do not have avatar of fire":         {tag: mod.Condition("HaveAvatarOfFire").Neg(true)},
	"if you have a summoned golem":                 {tag: mod.Condition("HavePhysicalGolem", "HaveLightningGolem", "HaveColdGolem", "HaveFireGolem", "HaveChaosGolem", "HaveCarrionGolem")},
	"while you have a summoned golem":              {tag: mod.Condition("HavePhysicalGolem", "HaveLightningGolem", "HaveColdGolem", "HaveFireGolem", "HaveChaosGolem", "HaveCarrionGolem")},
	"if a minion has died recently":                {tag: mod.Condition("MinionsDiedRecently")},
	"if one of your minions has died recently":     {tag: mod.Condition("MinionsDiedRecently")},
	"if a minion has been killed recently":         {tag: mod.Condition("MinionsDiedRecently")},
	"while you have sacrificial zeal":              {tag: mod.Condition("SacrificialZeal")},
	"while sane":                                   {tag: mod.Condition("Insane").Neg(true)},
	"while insane":                                 {tag: mod.Condition("Insane")},
	"while you have defiance":                      {tag: mod.MultiplierThreshold("Defiance").Threshold(1)},
	"while affected by glorious madness":           {tag: mod.Condition("AffectedByGloriousMadness")},
	"if you have reserved life and mana":           {tagList: []mod.Tag{mod.StatThreshold("LifeReserved", 1), mod.StatThreshold("ManaReserved", 1)}},
	"if you've shattered an enemy recently":        {tag: mod.Condition("ShatteredEnemyRecently")},
	"if you've dodge rolled recently":              {tag: mod.Condition("DodgeRolledRecently")},
	"at close range":                               {tag: mod.Condition("AtCloseRange")},
	"against rare [ao][nr]d? unique enemies":       {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	"by s?l?a?i?n? rare [ao][nr]d? unique enemies": {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	"against unique enemies":                       {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	"against enemies on full life":                 {tag: mod.ActorCondition("enemy", "FullLife")},
	"against enemies that are on full life":        {tag: mod.ActorCondition("enemy", "FullLife")},
	"that are on full life":                        {tag: mod.ActorCondition("enemy", "FullLife")},
	"against enemies on low life":                  {tag: mod.ActorCondition("enemy", "LowLife")},
	"against enemies that are on low life":         {tag: mod.ActorCondition("enemy", "LowLife")},
	"to enemies which have energy shield":          {tag: mod.ActorCondition("enemy", "HaveEnergyShield"), keywordFlags: KeywordFlagHit | KeywordFlagAilment},
	"against cursed enemies":                       {tag: mod.ActorCondition("enemy", "Cursed")},
	"against dazed enemies":                        {tag: mod.ActorCondition("enemy", "Dazed")},
	"against stunned enemies":                      {tag: mod.ActorCondition("enemy", "Stunned")},
	"against heavy stunned enemies":                {tag: mod.ActorCondition("enemy", "HeavyStunned")},
	"against enemies with fully broken armour":     {tag: mod.ActorCondition("enemy", "ArmourFullyBroken")},
	"against enemies with exposure":                {tag: mod.ActorCondition("enemy", "HasExposure")},
	"against pinned enemies":                       {tag: mod.ActorCondition("enemy", "Pinned")},
	"against immobilised enemies":                  {tag: mod.ActorCondition("enemy", "Immobilised")},
	"on cursed enemies":                            {tag: mod.ActorCondition("enemy", "Cursed")},
	"of cursed enemies'":                           {tag: mod.ActorCondition("enemy", "Cursed")},
	"when hitting cursed enemies":                  {tag: mod.ActorCondition("enemy", "Cursed"), keywordFlags: KeywordFlagHit},
	"from cursed enemies":                          {tag: mod.ActorCondition("enemy", "Cursed")},
	"against marked enemy":                         {tag: mod.ActorCondition("enemy", "Marked")},
	"against marked enemies":                       {tag: mod.ActorCondition("enemy", "Marked")},
	"when hitting marked enemy":                    {tag: mod.ActorCondition("enemy", "Marked"), keywordFlags: KeywordFlagHit},
	"from marked enemy":                            {tag: mod.ActorCondition("enemy", "Marked")},
	"to y?o?u?r? ?marked targets?":                 {tag: mod.ActorCondition("enemy", "Marked")},
	"against taunted enemies":                      {tag: mod.ActorCondition("enemy", "Taunted")},
	"against bleeding enemies":                     {tag: mod.ActorCondition("enemy", "Bleeding")},
	"you inflict on bleeding enemies":              {tag: mod.ActorCondition("enemy", "Bleeding")},
	"to bleeding enemies":                          {tag: mod.ActorCondition("enemy", "Bleeding")},
	"from bleeding enemies":                        {tag: mod.ActorCondition("enemy", "Bleeding")},
	"against poisoned enemies":                     {tag: mod.ActorCondition("enemy", "Poisoned")},
	"you inflict on poisoned enemies":              {tag: mod.ActorCondition("enemy", "Poisoned")},
	"to poisoned enemies":                          {tag: mod.ActorCondition("enemy", "Poisoned")},
	`against enemies affected by (\d+) or more poisons`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("PoisonStacks").Threshold(utils.Float(caps[0])).Actor("enemy")}
		},
	},
	`against enemies affected by at least (\d+) poisons`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("PoisonStacks").Threshold(utils.Float(caps[0])).Actor("enemy")}
		},
	},
	"against enemies affected by incision":                       {tag: mod.MultiplierThreshold("IncisionStack").Threshold(1).Actor("enemy")},
	"against hindered enemies":                                   {tag: mod.ActorCondition("enemy", "Hindered")},
	"against maimed enemies":                                     {tag: mod.ActorCondition("enemy", "Maimed")},
	"you inflict on maimed enemies":                              {tag: mod.ActorCondition("enemy", "Maimed")},
	"against blinded enemies":                                    {tag: mod.ActorCondition("enemy", "Blinded")},
	"from blinded enemies":                                       {tag: mod.ActorCondition("enemy", "Blinded")},
	"against burning enemies":                                    {tag: mod.ActorCondition("enemy", "Burning")},
	"against ignited enemies":                                    {tag: mod.ActorCondition("enemy", "Ignited")},
	"to ignited enemies":                                         {tag: mod.ActorCondition("enemy", "Ignited")},
	"against electrocuted enemies":                               {tag: mod.ActorCondition("enemy", "Electrocuted")},
	"against shocked enemies":                                    {tag: mod.ActorCondition("enemy", "Shocked")},
	"you inflict on shocked enemies":                             {tag: mod.ActorCondition("enemy", "Shocked")},
	"to shocked enemies":                                         {tag: mod.ActorCondition("enemy", "Shocked")},
	"inflicted on shocked enemies":                               {tag: mod.ActorCondition("enemy", "Shocked")},
	"enemies which are shocked":                                  {tag: mod.ActorCondition("enemy", "Shocked")},
	"against frozen enemies":                                     {tag: mod.ActorCondition("enemy", "Frozen")},
	"to frozen enemies":                                          {tag: mod.ActorCondition("enemy", "Frozen")},
	"against chilled enemies":                                    {tag: mod.ActorCondition("enemy", "Chilled")},
	"you inflict on chilled enemies":                             {tag: mod.ActorCondition("enemy", "Chilled")},
	"to chilled enemies":                                         {tag: mod.ActorCondition("enemy", "Chilled")},
	"inflicted on chilled enemies":                               {tag: mod.ActorCondition("enemy", "Chilled")},
	"enemies which are chilled":                                  {tag: mod.ActorCondition("enemy", "Chilled")},
	"against chilled or frozen enemies":                          {tag: mod.ActorCondition("enemy", "Chilled", "Frozen")},
	"against frozen, shocked or ignited enemies":                 {tag: mod.ActorCondition("enemy", "Frozen", "Shocked", "Ignited")},
	"against enemies affected by elemental ailments":             {tag: mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped")},
	"against enemies affected by ailments":                       {tag: mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped", "Poisoned", "Bleeding")},
	"against enemies that are affected by elemental ailments":    {tag: mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped")},
	"against enemies that are affected by no elemental ailments": {tagList: []mod.Tag{mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped").Neg(true), mod.Condition("Effective")}},
	`against enemies affected by (\d+) spider's webs`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("Spider's WebStack").Threshold(utils.Float(caps[0])).Actor("enemy")}
		},
	},
	"against enemies that are not poisoned": {tag: mod.MultiplierThreshold("PoisonStacks").Threshold(1).Upper(true).Actor("enemy")},
	"on targets that are not poisoned":      {tag: mod.MultiplierThreshold("PoisonStacks").Threshold(1).Upper(true).Actor("enemy")},
	"against enemies on consecrated ground": {tag: mod.ActorCondition("enemy", "OnConsecratedGround")},
	`if (\d+)% of curse duration expired`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("CurseExpired").Threshold(utils.Float(caps[0])).Actor("enemy")}
		},
	},
	`against enemies with (\w+) exposure`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ActorCondition("enemy", "Has"+firstToUpper(caps[0])+"Exposure")}
		},
	},
	"by s?l?a?i?n? ?frozen enemies":  {tag: mod.ActorCondition("enemy", "Frozen")},
	"by s?l?a?i?n? ?shocked enemies": {tag: mod.ActorCondition("enemy", "Shocked")},
	`against enemies further than (\d+)m`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("enemyDistance").Threshold((utils.Float(caps[0]) * 10))}
		},
	},
	`against enemies within (\d+) ?me?t?r?e?s?`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("enemyDistance").Threshold((utils.Float(caps[0]) * 10)).Upper(true)}
		},
	},
	"against slowed enemies":                       {tag: mod.ActorCondition("enemy", "Slowed")},
	"per freeze, shock [ao][nr]d? ignite on enemy": {tag: mod.Multiplier("FreezeShockIgniteOnEnemy")},
	`per (\d+)% chill magnitude on enemy`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ChillEffect").Div(utils.Float(caps[0])).Actor("enemy")}
		},
	},
	"per poison affecting enemy": {tag: mod.Multiplier("PoisonStack").Actor("enemy")},
	`per poison affecting enemy, up to \+([\d\.]+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonStack").Limit(utils.Float(caps[0])).LimitTotal(true).Actor("enemy")}
		},
	},
	"for each spider's web on the enemy": {tag: mod.Multiplier("Spider's WebStack").Actor("enemy")},
}

type gemLookups struct {
	skillIDForName map[string]string
	gemForBaseName map[string]*poe2.Gem
}

var gemLookup = sync.OnceValue(buildGemLookup)

var gemKeywordWordPattern = regexp.MustCompile(`\S+`)

var gemPropertyWordRe = regexp.MustCompile(`[A-Za-z0-9]+`)

// poe2.Gem carries no grantedEffect.fromItem, so every gem keys on its own name and then on its baseTypeName.
//
//pob2:block:src/Modules/ModParser.lua:2186-2197
func buildGemLookup() *gemLookups {
	lookups := &gemLookups{
		skillIDForName: map[string]string{},
		gemForBaseName: make(map[string]*poe2.Gem),
	}
	ids := make([]int, len(poe2.Gems))
	for i := range ids {
		ids[i] = i
	}
	sort.SliceStable(ids, func(a, b int) bool { return poe2.Gems[ids[a]].ID < poe2.Gems[ids[b]].ID })
	for _, i := range ids {
		gem := &poe2.Gems[i]
		lookups.skillIDForName[strings.ToLower(gem.Name)] = gem.GrantedEffectID
		lookups.gemForBaseName[strings.ToLower(gem.Name)] = gem
	}
	// stands in for the fromItem baseTypeName key, without overriding a gem's own name
	for _, i := range ids {
		gem := &poe2.Gems[i]
		if name := strings.ToLower(gem.BaseTypeName); name != "" {
			if _, taken := lookups.skillIDForName[name]; !taken {
				lookups.skillIDForName[name] = gem.GrantedEffectID
			}
		}
	}
	return lookups
}
func extraSkillIDForName(name string) string {
	return gemLookup().skillIDForName[strings.ToLower(name)]
}

// flaskCurseSkillID is gemIdLookup[skill:gsub(" skill","")] or "Unknown".
func flaskCurseSkillID(skill string) string {
	if id := extraSkillIDForName(strings.ReplaceAll(skill, " skill", "")); id != "" {
		return id
	}
	return "Unknown"
}
func extraSupportSlotName(slot string) string {
	switch strings.ToLower(strings.TrimSpace(slot)) {
	case "main hand", "main hand weapon":
		return "Weapon 1"
	case "off hand", "off hand weapon":
		return "Weapon 2"
	case "":
		return "{SlotName}"
	}
	words := strings.Fields(strings.ToLower(slot))
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

type extraSupportEntry struct {
	skillID string
	asSkill bool
}

//pob2:block:src/Modules/ModParser.lua:2224-2259
func extraSupportIDsForName(name string) []extraSupportEntry {
	lookups := gemLookup()

	name = strings.ToLower(name)

	skillID, ok := lookups.skillIDForName[name]
	if !ok {
		skillID, ok = lookups.skillIDForName[strings.TrimPrefix(name, "increased ")]
	}
	if !ok {
		skillID, ok = lookups.skillIDForName[strings.TrimSuffix(name, " support")]
	}
	if !ok {
		return nil
	}

	skillName := ""
	for i := range poe2.Gems {
		if poe2.Gems[i].GrantedEffectID == skillID {
			skillName = poe2.Gems[i].GrantedEffectName
			break
		}
	}
	supportGem, ok := lookups.gemForBaseName[strings.ToLower(skillName+" Support")]
	if !ok {
		return []extraSupportEntry{{skillID: skillID}}
	}
	return []extraSupportEntry{{skillID: supportGem.GrantedEffectID}}
}

func extraSupportMods(entries []extraSupportEntry, level int, slotName string, appliesToGrantedSkills bool) []mod.Mod {
	mods := make([]mod.Mod, 0, len(entries))
	for _, entry := range entries {
		if entry.asSkill {
			mods = append(mods, MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: entry.skillID, Level: level}))
			continue
		}
		mods = append(mods, MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: entry.skillID, Level: level, AppliesToGrantedSkills: appliesToGrantedSkills}).Tag(mod.SocketedIn(slotName)))
	}
	return mods
}

//pob2:src/Modules/ModParser.lua:2198-2205
func grantedExtraSkill(name string, level int, noSupports bool) []mod.Mod {
	name = strings.ReplaceAll(strings.ToLower(name), " skill", "")
	id := extraSkillIDForName(name)
	if id == "" {
		return nil
	}
	return []mod.Mod{
		MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: id, SkillName: name, Level: level, NoSupports: noSupports}),
	}
}

//pob2:src/Modules/ModParser.lua:2206-2223
func triggerExtraSkill(name string, level int, noSupports bool, sourceSkill string) []mod.Mod {
	name = strings.ReplaceAll(strings.ToLower(name), " skill", "")
	sourceSkill = strings.ReplaceAll(strings.ToLower(sourceSkill), " skill", "")
	id := extraSkillIDForName(name)
	if id == "" {
		return []mod.Mod{}
	}
	skill := mod.ExtraSkill{SkillID: id, SkillName: name, Level: level, NoSupports: noSupports, Triggered: true}
	if sourceSkill != "" {
		skill.Source = sourceSkill
	}
	return []mod.Mod{
		MOD("ExtraSkill", "LIST", skill),
	}
}

var damageTypeList = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

// Makes the "deal no" modifiers for every damage type except the one being kept.
func dealNoNonDamageType(dmgType string, forMinion bool) []mod.Mod {
	dmgType = firstToUpper(dmgType)
	if !slices.Contains(damageTypeList, dmgType) {
		return nil
	}
	mods := []mod.Mod{}
	for _, damageType := range damageTypeList {
		if damageType != dmgType {
			dealNo := FLAG("DealNo" + damageType)
			if forMinion {
				mods = append(mods, MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: dealNo}))
			} else {
				mods = append(mods, dealNo)
			}
		}
	}
	return mods
}
func withTriggerChance(mods []mod.Mod, chance float64) []mod.Mod {
	if len(mods) == 0 {
		return mods
	}
	if skill, ok := mods[0].Value().List().(mod.ExtraSkill); ok {
		skill.TriggerChance = chance
		mods[0] = MOD("ExtraSkill", "LIST", skill)
	}
	return mods
}
func triggerExtraSkillOnCrit(name string, level int, noSupports bool) []mod.Mod {
	name, _ = strings.CutSuffix(strings.ToLower(name), " skill")
	mods := triggerExtraSkill(name, level, noSupports, "")
	return append(mods,
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{
			Mod: MOD("SkillData", "LIST", mod.SkillData{Key: "triggerOnCrit", Value: 1}),
		}).Tag(mod.SkillId(extraSkillIDForName(name))))
}

type ExplodeMod struct {
	Type           string
	Value          float64
	Amount         float64
	KeyOfScaledMod string
}

func (e ExplodeMod) Clone(pointer bool) mod.Cloneable {
	obj := e
	if pointer {
		return &obj
	}
	return obj
}

//pob2:src/Modules/ModParser.lua:2261-2272
func explodeFunc(chance float64, amount string, typ string, tags ...mod.Tag) ([]mod.Mod, string) {
	amountNumber, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		switch amount {
		case "tenth":
			amountNumber = 10
		case "quarter":
			amountNumber = 25
		default:
			return nil, ""
		}
	}
	return []mod.Mod{
		MOD("ExplodeMod", "LIST", ExplodeMod{Type: firstToUpper(typ), Value: chance, Amount: amountNumber, KeyOfScaledMod: "value"}).Tag(tags...),
		FLAG("CanExplode"),
	}, ""
}

type SpecialFuncType func(num float64, captures []string) ([]mod.Mod, string)

// List of special modifiers
var specialModListCompiled map[string]CompiledList[interface{}]
var specialModList = map[string]interface{}{
	//pob2:block:src/Modules/ModParser.lua:2280-2315
	`enemies you kill have a (\d+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2])
	},
	`enemies you kill have ?a? ?(\d+)% chance to explode, dealing (\d+)% of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2])
	},
	`enemies you or your totems kill have (\d+)% chance to explode, dealing (\d+)% of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2])
	},
	`enemies you kill with empowered attacks have a (\d+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2])
	},
	`enemies you kill while using pride have (\d+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2], mod.Condition("AffectedByPride"))
	},
	`enemies you kill during effect have a (\d+)% chance to explode, dealing a (.+) of their maximum life as damage of a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], "randomElement", mod.Condition("UsingFlask"))
	},
	`lose (\d+)% life per second while you have no runic ward during effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeDegenPercent", "BASE", num).Tag(mod.Condition("NoRunicWard"), mod.Condition("UsingFlask"))}, ""
	},
	`enemies you kill while affected by glorious madness have a (\d+)% chance to explode, dealing a (.+) of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2], mod.Condition("AffectedByGloriousMadness"))
	},
	`enemies killed with wand hits have a (\d+)% chance to explode, dealing a (.+) of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2], mod.Condition("UsingWand"))
	},
	`cursed enemies you or your minions kill have a (\d+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2], mod.ActorCondition("enemy", "Cursed"))
	},
	`cursed enemies killed by you, or by allies in your presence, have a (\d+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2], mod.ActorCondition("enemy", "Cursed"))
	},
	`enemies you kill explode, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1])
	},
	//pob2:block:src/Modules/ModParser.lua:2316-2318
	`causes enemies to explode on critical kill, for (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) { // Crumbling Maul implicit
		return explodeFunc(100, captures[0], captures[1])
	},
	//pob2:block:src/Modules/ModParser.lua:2319-2341
	`enemies on fungal ground you kill explode, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.ActorCondition("enemy", "OnFungalGround"))
	},
	`enemies killed with attack or spell hits explode, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1])
	},
	`shocked enemies you kill explode, dealing (\d+)% of their life as (.+) damage which cannot shock`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.ActorCondition("enemy", "Shocked"))
	},
	`bleeding enemies you kill explode, dealing (\d+)% of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.ActorCondition("enemy", "Bleeding"))
	},
	`burning enemies you kill have a (\d+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2], mod.ActorCondition("enemy", "Burning"))
	},
	"non*?aura curses you inflict are not removed from dying enemies": []mod.Mod{},
	"enemies near corpses affected by your curses are blinded":        []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Blinded")}).Tag(mod.MultiplierThreshold("NearbyCorpse").Threshold(1), mod.ActorCondition("enemy", "Cursed"))},
	`enemies killed near corpses affected by your curses explode, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.MultiplierThreshold("NearbyCorpse").Threshold(1), mod.ActorCondition("enemy", "Cursed"))
	},
	`enemies taunted by your warcries explode on death, dealing (\d+)% of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.ActorCondition("enemy", "Taunted"), mod.Condition("UsedWarcryRecently"))
	},
	//pob2:block:src/Modules/ModParser.lua:2345-2734
	`totems explode on death, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1])
	},
	`nearby corpses explode when you warcry, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1])
	},
	`(\d+)% chance for enemies you kill to explode, dealing (\d+)% of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2])
	},
	"dodge roll cannot avoid damage": []mod.Mod{},
	`take (\d+)% less damage from hits while dodge rolling`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageTakenWhenHit", "MORE", -num).Tag(mod.Condition("InDodgeRoll"))}, ""
	},
	`(\d+)% more skill speed while off hand is empty and you have a one\-handed martial weapon equipped in your main hand`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "MORE", num).Tag(mod.Condition("UsingOneHandedWeapon"), mod.Condition("OffHandIsEmpty")), MOD("WarcrySpeed", "MORE", num).Tag(mod.Condition("UsingOneHandedWeapon"), mod.Condition("OffHandIsEmpty")), MOD("TotemPlacementSpeed", "MORE", num).Tag(mod.Condition("UsingOneHandedWeapon"), mod.Condition("OffHandIsEmpty"))}, ""
	},
	`(\d+) rage regenerated for every (\d+) mana regeneration per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RageRegen", "BASE", num).Tag(mod.PerStat(utils.Float(captures[1]), "ManaRegen")), FLAG("Condition:CanGainRage")}, ""
	},
	`(\w+) recovery from regeneration is not applied`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("UnaffectedBy" + firstToUpper(captures[0]) + "Regen")}, ""
	},
	`(\d+)% less damage taken for every (\d+)% life recovery per second from leech`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageTaken", "MORE", -num).Tag(mod.PerStat(utils.Float(captures[1]), "MaxLifeLeechRatePercent"))}, ""
	},
	`(\w+) recovery from non\-instant leech is not applied`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("UnaffectedByNonInstant" + firstToUpper(captures[0]) + "Leech")}, ""
	},
	`(\d+)% additional physical damage reduction for every (\d+)% life recovery per second from leech`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageReduction", "BASE", num).Tag(mod.PerStat(utils.Float(captures[1]), "MaxLifeLeechRatePercent"), mod.Condition("Leeching"))}, ""
	},
	"modifiers to chance to suppress spell damage instead apply to chance to dodge spell hits at 50% of their value": []mod.Mod{FLAG("ConvertSpellSuppressionToSpellDodge"), MOD("SpellSuppressionChance", "OVERRIDE", 0).Source(mod.Source("Acrobatics"))},
	`maximum chance to dodge spell hits is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellDodgeChanceMax", "OVERRIDE", num).Source(mod.Source("Acrobatics"))}, ""
	},
	"inherent life granted by strength is halved":            []mod.Mod{FLAG("HalvesLifeFromStrength")},
	"dexterity provides no bonus to evasion rating":          []mod.Mod{FLAG("NoDexBonusToEvasion")},
	"dexterity provides no inherent bonus to evasion rating": []mod.Mod{FLAG("NoDexBonusToEvasion")},
	"your hits can't be evaded":                              []mod.Mod{FLAG("CannotBeEvaded")},
	"minion hits can't be evaded":                            []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("CannotBeEvaded")})},
	"never deal critical hits":                               []mod.Mod{FLAG("NeverCrit"), FLAG("Condition:NeverCrit")},
	"minions never deal critical hits":                       []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("NeverCrit")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Condition:NeverCrit")})},
	"never deal critical hits with spells":                   []mod.Mod{FLAG("NeverCrit").Flag(ModFlagSpell), FLAG("Condition:NeverCrit").Flag(ModFlagSpell)},
	"never deal critical hits with attacks":                  []mod.Mod{FLAG("NeverCrit").Flag(ModFlagAttack), FLAG("Condition:NeverCrit").Flag(ModFlagAttack)},
	"cannot deal critical hits":                              []mod.Mod{FLAG("NeverCrit"), FLAG("Condition:NeverCrit")},
	"cannot deal critical hits with spells":                  []mod.Mod{FLAG("NeverCrit").Flag(ModFlagSpell), FLAG("Condition:NeverCrit").Flag(ModFlagSpell)},
	"cannot deal critical hits with attacks":                 []mod.Mod{FLAG("NeverCrit").Flag(ModFlagAttack), FLAG("Condition:NeverCrit").Flag(ModFlagAttack)},
	"no critical damage bonus":                               []mod.Mod{FLAG("NoCritMultiplier")},
	"ailments never count as being from critical hits":       []mod.Mod{FLAG("AilmentsAreNeverFromCrit")},
	"converts all evasion rating to armour":                  []mod.Mod{FLAG("IronReflexes"), MOD("EvasionConvertToArmour", "BASE", 100)},
	`30% chance to dodge attack hits\. 50% less armour, 30% less energy shield, 30% less chance to block spell and attack damage`: []mod.Mod{MOD("AttackDodgeChance", "BASE", 30), MOD("Armour", "MORE", -50), MOD("EnergyShield", "MORE", -30), MOD("BlockChance", "MORE", -30), MOD("SpellBlockChance", "MORE", -30)},
	`(\d+)% increased blind effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("BlindEffect", "INC", num)})}, ""
	},
	`(\d+)% increased effect of blind from melee weapons`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("BlindEffect", "INC", num)})}, ""
	},
	`\+(\d+)% chance to block spell damage for each (\d+)% overcapped chance to block attack damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellBlockChance", "BASE", num).Tag(mod.PerStat(utils.Float(captures[1]), "BlockChanceOverCap"))}, ""
	},
	"maximum life becomes 1, immune to chaos damage": []mod.Mod{FLAG("ChaosInoculation"), MOD("ChaosDamageTaken", "MORE", -100)},
	"immune to chaos damage":                         []mod.Mod{FLAG("ChaosInoculation"), MOD("ChaosDamageTaken", "MORE", -100)},
	"immune to chaos damage and bleeding":            []mod.Mod{FLAG("ChaosInoculation"), MOD("ChaosDamageTaken", "MORE", -100), FLAG("BleedImmune")},
	`leech life (\d+)% slower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeLeechRate", "INC", -num)}, ""
	},
	`leech life (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeLeechRate", "INC", num)}, ""
	},
	"life regeneration is applied to energy shield instead":              []mod.Mod{FLAG("ZealotsOath")},
	"mana recovery from regeneration is also applied to runic ward":      []mod.Mod{FLAG("ManaRegenerationRecoversWard")},
	"excess life recovery from regeneration is applied to energy shield": []mod.Mod{FLAG("ZealotsOath").Tag(mod.Condition("FullLife"))},
	"life regeneration has no effect":                                    []mod.Mod{FLAG("NoLifeRegen")},
	"life recharges instead of energy shield":                            []mod.Mod{FLAG("EnergyShieldRechargeAppliesToLife")},
	`deal no non\-fire damage`:                                           []mod.Mod{FLAG("DealNoPhysical"), FLAG("DealNoLightning"), FLAG("DealNoCold"), FLAG("DealNoChaos")},
	`blade vortex and blade blast deal no non\-physical damage`:          []mod.Mod{FLAG("DealNoLightning").Tag(mod.SkillName("Blade Vortex", "Blade Blast").Transfigured(true)), FLAG("DealNoCold").Tag(mod.SkillName("Blade Vortex", "Blade Blast").Transfigured(true)), FLAG("DealNoFire").Tag(mod.SkillName("Blade Vortex", "Blade Blast").Transfigured(true)), FLAG("DealNoChaos").Tag(mod.SkillName("Blade Vortex", "Blade Blast").Transfigured(true))},
	`(\d+)% of physical, cold and lightning damage converted to fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageConvertToFire", "BASE", num), MOD("LightningDamageConvertToFire", "BASE", num), MOD("ColdDamageConvertToFire", "BASE", num)}, ""
	},
	"all elemental damage converted to chaos damage":           []mod.Mod{MOD("ColdDamageConvertToChaos", "BASE", 100), MOD("FireDamageConvertToChaos", "BASE", 100), MOD("LightningDamageConvertToChaos", "BASE", 100)},
	"all damage is taken from mana before life":                []mod.Mod{MOD("DamageTakenFromManaBeforeLife", "BASE", 100)},
	`removes all mana\. spend life instead of mana for skills`: []mod.Mod{MOD("Mana", "OVERRIDE", 0), FLAG("CostLifeInsteadOfMana")},
	"removes all mana": []mod.Mod{MOD("Mana", "OVERRIDE", 0)},
	"you have no mana": []mod.Mod{MOD("Mana", "OVERRIDE", 0)},
	"d?o?u?b?l?e?s? ?mana costs ?a?r?e? ?d?o?u?b?l?e?d?": []mod.Mod{MOD("ManaCost", "MORE", 100)},
	"removes all energy shield":                          []mod.Mod{MOD("EnergyShield", "OVERRIDE", 0)},
	"converts all energy shield to mana":                 []mod.Mod{MOD("EnergyShieldConvertToMana", "BASE", 100)},
	`convert (\d+)% of maximum energy shield to maximum mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldConvertToMana", "BASE", num)}, ""
	},
	"convert all armour to evasion rating":                  []mod.Mod{MOD("ArmourConvertToEvasion", "BASE", 100)},
	"skills cost life instead of mana":                      []mod.Mod{FLAG("CostLifeInsteadOfMana")},
	"skills reserve life instead of mana":                   []mod.Mod{FLAG("BloodMagicReserved")},
	`non\-aura skills cost no mana or life while focus?sed`: []mod.Mod{MOD("ManaCost", "MORE", -100).Tag(mod.Condition("Focused"), mod.SkillType("Aura").Neg(true)), MOD("LifeCost", "MORE", -100).Tag(mod.Condition("Focused"), mod.SkillType("Aura").Neg(true))},
	"spend life instead of mana for effects of skills":      []mod.Mod{},
	`skills cost \+(\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RageCostBase", "BASE", num)}, ""
	},
	`skills have \+(\d+) to rage cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RageCostBase", "BASE", num), FLAG("Condition:CanGainRage")}, ""
	},
	`non\-aura vaal skills require (\d+)% reduced souls per use during effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SoulCost", "INC", -num).Tag(mod.Condition("UsingFlask"), mod.SkillType("Aura").Neg(true), mod.SkillType("Vaal"))}, ""
	},
	`non\-aura vaal skills require (\d+)% reduced souls per use`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SoulCost", "INC", -num).Tag(mod.SkillType("Aura").Neg(true), mod.SkillType("Vaal"))}, ""
	},
	`vaal skills used during effect have (\d+)% reduced soul gain prevention duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SoulGainPreventionDuration", "INC", -num).Tag(mod.Condition("UsingFlask"), mod.SkillType("Vaal"))}, ""
	},
	`vaal volcanic fissure and vaal molten strike have (\d+)% reduced soul gain prevention duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SoulGainPreventionDuration", "INC", -num).Tag(mod.SkillName("Volcanic Fissure", "Molten Strike").Transfigured(true), mod.SkillType("Vaal"))}, ""
	},
	"vaal attack skills cost rage instead of requiring souls to use":           []mod.Mod{FLAG("CostRageInsteadOfSouls").Flag(ModFlagAttack).Tag(mod.SkillType("Vaal"))},
	"vaal attack skills you use yourself cost rage instead of requiring souls": []mod.Mod{FLAG("CostRageInsteadOfSouls").Flag(ModFlagAttack).Tag(mod.SkillType("Vaal"))},
	"you cannot gain rage during soul gain prevention":                         []mod.Mod{MOD("RageRegen", "MORE", -100).Tag(mod.Condition("SoulGainPrevention"))},
	`hits that deal elemental damage remove exposure to those elements and inflict exposure to other elements exposure inflicted this way applies (\-\d+)% to resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("ElementalEquilibrium"), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", num).Tag(mod.Condition("HitByColdDamage", "HitByLightningDamage"))}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", num).Tag(mod.Condition("HitByFireDamage", "HitByLightningDamage"))}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", num).Tag(mod.Condition("HitByFireDamage", "HitByColdDamage"))})}, ""
	},
	`enemies you hit with elemental damage temporarily get (\+\d+)% resistance to those elements and (\-\d+)% resistance to other elements`: func(plus float64, captures []string) ([]mod.Mod, string) {
		minus := utils.Float(captures[1])
		return []mod.Mod{
			mod.NewFlag("ElementalEquilibrium", true),
			mod.NewFlag("ElementalEquilibriumLegacy", true),
			mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("FireResist", mod.TypeBase, plus).Tag(mod.Condition("HitByFireDamage"))}),
			mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("FireResist", mod.TypeBase, minus).Tag(mod.Condition("HitByFireDamage").Neg(true)).Tag(mod.Condition("HitByColdDamage", "HitByLightningDamage"))}),
			mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("ColdResist", mod.TypeBase, plus).Tag(mod.Condition("HitByColdDamage"))}),
			mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("ColdResist", mod.TypeBase, minus).Tag(mod.Condition("HitByColdDamage").Neg(true)).Tag(mod.Condition("HitByFireDamage", "HitByLightningDamage"))}),
			mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("LightningResist", mod.TypeBase, plus).Tag(mod.Condition("HitByLightningDamage"))}),
			mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("LightningResist", mod.TypeBase, minus).Tag(mod.Condition("HitByLightningDamage").Neg(true)).Tag(mod.Condition("HitByFireDamage", "HitByColdDamage"))}),
		}, ""
	},
	"projectile attack hits deal up to 30% more damage to targets at the start of their movement, dealing less damage to targets as the projectile travels farther": []mod.Mod{FLAG("PointBlank")},
	"life leech is converted to energy shield leech":                                                                    []mod.Mod{FLAG("GhostReaver")},
	"minions explode when reduced to low life, dealing 33% of their maximum life as fire damage to surrounding enemies": []mod.Mod{MOD("ExtraMinionSkill", "LIST", mod.ExtraMinionSkill{SkillID: "MinionInstability"})},
	"minions explode when reduced to low life, dealing 33% of their life as fire damage to surrounding enemies":         []mod.Mod{MOD("ExtraMinionSkill", "LIST", mod.ExtraMinionSkill{SkillID: "MinionInstability"})},
	"all bonuses from ?a?n? equipped amulet apply to your minions instead of you":                                       []mod.Mod{},
	"spend energy shield before mana for skill m?a?n?a? ?costs":                                                         []mod.Mod{},
	"you have perfect agony if you've dealt a critical hit recently":                                                    []mod.Mod{MOD("Keystone", "LIST", "Perfect Agony").Tag(mod.Condition("CritRecently"))},
	"energy shield protects mana instead of life":                                                                       []mod.Mod{FLAG("EnergyShieldProtectsMana")},
	`modifiers to critical damage bonus also apply to damage over time multiplier for ailments from critical hits at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritMultiplierAppliesToDegen", "BASE", num)}, ""
	},
	"damage over time multiplier for ailments is equal to critical damage bonus":                  []mod.Mod{FLAG("DotMultiplierIsCritMultiplier")},
	"your bleeding does not deal extra damage while the enemy is moving":                          []mod.Mod{FLAG("Condition:NoExtraBleedDamageToMovingEnemy")},
	"your bleeding does not deal extra damage while the enemy is moving and cannot be aggravated": []mod.Mod{FLAG("Condition:NoExtraBleedDamageToMovingEnemy"), FLAG("Condition:CannotAggravate")},
	"you and enemies in your presence count as moving while affected by elemental ailments":       []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Moving").Tag(mod.Condition("Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"))}), FLAG("Condition:Moving").Tag(mod.Condition("Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"))},
	`you can inflict bleeding on an enemy up to (\d+) times?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BleedStacksMax", "OVERRIDE", num), FLAG("Condition:HaveCrimsonDance")}, ""
	},
	`base (\w+) duration is (\d+) seconds?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(strings.ReplaceAll(captures[0], "ing", ""))+"DurationBase", "OVERRIDE", utils.Float(captures[1]))}, ""
	},
	"bleeding you inflict is aggravated":                                                                        []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:BleedAggravated")})},
	"bleeding you inflict on pinned enemies is aggravated":                                                      []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:BleedAggravated").Tag(mod.Condition("Pinned"))})},
	"your minions spread caustic ground on death, dealing 20% of their maximum life as chaos damage per second": []mod.Mod{MOD("ExtraMinionSkill", "LIST", mod.ExtraMinionSkill{SkillID: "SiegebreakerCausticGround"})},
	"your minions spread burning ground on death, dealing 20% of their maximum life as fire damage per second":  []mod.Mod{MOD("ExtraMinionSkill", "LIST", mod.ExtraMinionSkill{SkillID: "ReplicaSiegebreakerBurningGround"})},
	"you can have an additional brand attached to an enemy":                                                     []mod.Mod{MOD("BrandsAttachedLimit", "BASE", 1)},
	`gain (\d+) grasping vines each second while stationary`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:GraspingVinesCount", "BASE", num).Tag(mod.Multiplier("StationarySeconds").Limit(10).LimitTotal(true), mod.Condition("Stationary"))}, ""
	},
	`all damage inflicts poison against enemies affected by at least (\d+) grasping vines`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)), FLAG("FireCanPoison").Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)), FLAG("ColdCanPoison").Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)), FLAG("LightningCanPoison").Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num))}, ""
	},
	"attack projectiles always inflict bleeding and maim, and knock back enemies": []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(ModFlagAttack | ModFlagProjectile), MOD("EnemyKnockbackChance", "BASE", 100).Flag(ModFlagAttack | ModFlagProjectile)},
	"projectiles cannot pierce, fork or chain":                                    []mod.Mod{FLAG("CannotPierce").Flag(ModFlagProjectile), FLAG("CannotChain").Flag(ModFlagProjectile), FLAG("CannotFork").Flag(ModFlagProjectile)},
	"critical hits inflict scorch, brittle and sapped":                            []mod.Mod{FLAG("CritAlwaysAltAilments")},
	`you take (\d+)% of damage from blocked hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BlockEffect", "BASE", num)}, ""
	},
	"ignore attribute requirements":                []mod.Mod{FLAG("IgnoreAttributeRequirements")},
	"gain no inherent bonuses from attributes":     []mod.Mod{FLAG("NoAttributeBonuses")},
	"gain no inherent bonuses from strength":       []mod.Mod{FLAG("NoStrengthAttributeBonuses")},
	"gain no inherent bonuses from dexterity":      []mod.Mod{FLAG("NoDexterityAttributeBonuses")},
	"gain no inherent bonuses from intelligence":   []mod.Mod{FLAG("NoIntelligenceAttributeBonuses")},
	"all damage taken bypasses energy shield":      []mod.Mod{MOD("PhysicalEnergyShieldBypass", "OVERRIDE", 100), MOD("LightningEnergyShieldBypass", "OVERRIDE", 100), MOD("ColdEnergyShieldBypass", "OVERRIDE", 100), MOD("FireEnergyShieldBypass", "OVERRIDE", 100), MOD("ChaosEnergyShieldBypass", "OVERRIDE", 100)},
	"physical damage taken bypasses energy shield": []mod.Mod{MOD("PhysicalEnergyShieldBypass", "BASE", 100)},
	`(\d+)% of damage taken bypasses energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalEnergyShieldBypass", "BASE", num), MOD("LightningEnergyShieldBypass", "BASE", num), MOD("ColdEnergyShieldBypass", "BASE", num), MOD("FireEnergyShieldBypass", "BASE", num), MOD("ChaosEnergyShieldBypass", "BASE", num)}, ""
	},
	"auras from your skills do not affect allies": []mod.Mod{FLAG("SelfAuraSkillsCannotAffectAllies")},
	`auras from your skills have (\d+)% more effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillAuraEffectOnSelf", "MORE", num)}, ""
	},
	`auras from your skills have (\d+)% increased effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillAuraEffectOnSelf", "INC", num)}, ""
	},
	"increases and reductions to mana regeneration rate instead apply to rage regeneration rate": []mod.Mod{FLAG("ManaRegenToRageRegen")},
	"increases and reductions to maximum energy shield instead apply to ward":                    []mod.Mod{FLAG("EnergyShieldToWard")},
	"all damage taken bypasses runic ward":                                                       []mod.Mod{MOD("WardBypass", "BASE", 100)},
	`maximum energy shield is (\d+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShield", "OVERRIDE", num)}, ""
	},
	"cannot have energy shield": []mod.Mod{FLAG("CannotHaveES")},
	`regenerate ([\d\.]+) life per second per maximum energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegen", "BASE", num).Tag(mod.PerStat(1, "MaximumEnergyShield"))}, ""
	},
	`while not on full life, sacrifice ([\d\.]+)% of ?m?a?x?i?m?u?m? mana per second to recover that much life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaDegenPercent", "BASE", num).Tag(mod.Condition("FullLife").Neg(true)), MOD("LifeRecovery", "BASE", 1).Tag(mod.PercentStat("Mana", num), mod.Condition("FullLife").Neg(true))}, ""
	},
	`(\d+)% increased maximum energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShield", "INC", num).Tag(mod.Global())}, ""
	},
	"you are blind": []mod.Mod{FLAG("Condition:Blinded").Tag(mod.Condition("CannotBeBlinded").Neg(true))},
	"armour applies to fire, cold and lightning damage taken from hits instead of physical damage": []mod.Mod{MOD("ArmourAppliesToFireDamageTaken", "BASE", 100), MOD("ArmourAppliesToColdDamageTaken", "BASE", 100), MOD("ArmourAppliesToLightningDamageTaken", "BASE", 100), FLAG("ArmourDoesNotApplyToPhysicalDamageTaken")},
	`(\d+)% of armour applies to fire, cold and lightning damage taken from hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourAppliesToFireDamageTaken", "BASE", num), MOD("ArmourAppliesToColdDamageTaken", "BASE", num), MOD("ArmourAppliesToLightningDamageTaken", "BASE", num)}, ""
	},
	"armour applies to elemental damage": []mod.Mod{MOD("ArmourAppliesToFireDamageTaken", "BASE", 100), MOD("ArmourAppliesToColdDamageTaken", "BASE", 100), MOD("ArmourAppliesToLightningDamageTaken", "BASE", 100)},
	`\+?(\d+)% of armour (a?l?s?o? ?)applies to elemental damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourAppliesToFireDamageTaken", "BASE", num), MOD("ArmourAppliesToColdDamageTaken", "BASE", num), MOD("ArmourAppliesToLightningDamageTaken", "BASE", num)}, ""
	},
	`armour also applies to (\w+) damage taken from hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourAppliesTo"+firstToUpper(captures[0])+"DamageTaken", "BASE", 100)}, ""
	},
	`armour also applies to (\w+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourAppliesTo"+firstToUpper(captures[0])+"DamageTaken", "BASE", 100)}, ""
	},
	`\+?(\d+)% of armour also applies to (\w+) damage taken from hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourAppliesTo"+firstToUpper(captures[1])+"DamageTaken", "BASE", num)}, ""
	},
	`\+?(\d+)% of armour also applies to (\w+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourAppliesTo"+firstToUpper(captures[1])+"DamageTaken", "BASE", num)}, ""
	},
	`\+?(\d+)% of armour also applies to (\w+) damage while on full energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourAppliesTo"+firstToUpper(captures[1])+"DamageTaken", "BASE", num).Tag(mod.Condition("FullEnergyShield"))}, ""
	},
	`maximum damage reduction for any damage type is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageReductionMax", "MAX", num)}, ""
	},
	`maximum (\w+) damage reduction is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"DamageReductionMax", "MAX", utils.Float(captures[1]))}, ""
	},
	"gain additional elemental damage reduction equal to half your chaos resistance": []mod.Mod{MOD("ElementalDamageReduction", "BASE", 1).Tag(mod.PerStat(2, "ChaosResist"))},
	"life recovery from flasks also applies to energy shield":                        []mod.Mod{FLAG("LifeFlaskAppliesToEnergyShield")},
	"life recovery from flasks applies to energy shield instead":                     []mod.Mod{FLAG("LifeFlaskAppliesToEnergyShield"), FLAG("LifeFlaskDoesNotApply")},
	`(\d+)% life recovery from flasks also applies to runic ward`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeFlaskRecoveryAppliesToWard", "BASE", num)}, ""
	},
	`non\-instant mana recovery from flasks is also recovered as life`:    []mod.Mod{FLAG("ManaFlaskAppliesToLife")},
	"life leech effects recover energy shield instead while on full life": []mod.Mod{FLAG("ImmortalAmbition").Tag(mod.Condition("FullLife"), mod.Condition("LeechingLife"))},
	"shepherd of souls": []mod.Mod{MOD("Damage", "MORE", -30).Tag(mod.SkillType("Vaal").Neg(true))},
	`(\d+)% more attack damage if accuracy rating is higher than maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).Source(mod.Source("Damage")).Flag(ModFlagAttack).Tag(mod.Condition("MainHandAccRatingHigherThanMaxLife"), mod.Condition("MainHandAttack")), MOD("Damage", "MORE", num).Source(mod.Source("Damage")).Flag(ModFlagAttack).Tag(mod.Condition("OffHandAccRatingHigherThanMaxLife"), mod.Condition("OffHandAttack"))}, ""
	},
	"your hexes have infinite duration":  []mod.Mod{MOD("Duration", "BASE", math.Inf(1)).Tag(mod.SkillType("AppliesCurse"))},
	"your curses have infinite duration": []mod.Mod{MOD("Duration", "BASE", math.Inf(1)).Tag(mod.SkillType("AppliesCurse"))},
	`(\d+)% chance to defend with double your armour for each time you've been hit by an enemy recently, up to (\d+)%`: func(numChance float64, captures []string) ([]mod.Mod, string) {
		capPct := utils.Float(captures[1])
		return []mod.Mod{
			mod.NewFloat("ArmourDefense", mod.TypeMAX, 100).Tag(mod.Condition("ArmourMax")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(numChance/100, 1.0)*100).Tag(mod.Condition("ArmourAvg")).Tag(mod.Multiplier("BeenHitRecently").Limit(capPct / numChance)),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(math.Floor(numChance/100), 1.0)*100).Tag(mod.Condition("ArmourMax").Neg(true)).Tag(mod.Condition("ArmourAvg").Neg(true)).Tag(mod.Multiplier("BeenHitRecently").Limit(capPct / numChance)),
		}, ""
	},
	`you can wield two\-handed axes, maces and swords in one hand`:  []mod.Mod{FLAG("GiantsBlood")},
	"you can equip a focus while wielding a staff":                  []mod.Mod{FLAG("InstrumentsOfPower")},
	`you can equip a non\-unique sceptre while wielding a talisman`: []mod.Mod{FLAG("LordOfTheWilds")},
	`gain armour equal to (\d+)% of total strength requirements of equipped boots, gloves and helmet`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Armour", "BASE", 1).Tag(mod.PercentStat("StrRequirementsOnBoots", num)), MOD("Armour", "BASE", 1).Tag(mod.PercentStat("StrRequirementsOnGloves", num)), MOD("Armour", "BASE", 1).Tag(mod.PercentStat("StrRequirementsOnHelmet", num))}, ""
	},
	`(\d+)% of strength requirements from boots, gloves and helmets also added to armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Armour", "BASE", 1).Tag(mod.PercentStat("StrRequirementsOnBoots", num)), MOD("Armour", "BASE", 1).Tag(mod.PercentStat("StrRequirementsOnGloves", num)), MOD("Armour", "BASE", 1).Tag(mod.PercentStat("StrRequirementsOnHelmet", num))}, ""
	},
	"iron grip": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", 1).Flag(ModFlagSpell).Tag(mod.PerStat(2, "Str")), FLAG("NoStrBonusToLife")}, ""
	},
	"iron will": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", 1).Flag(ModFlagProjectile | ModFlagAttack).Tag(mod.PerStat(2, "Str")), FLAG("NoStrBonusToLife")}, ""
	},
	`(\w+) spells convert (\d+)% of (\w+) damage to (\w+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[2])+"DamageConvertTo"+firstToUpper(captures[3]), "BASE", utils.Float(captures[1])).Tag(mod.SkillType("Spell"), mod.SkillType(firstToUpper(captures[0])))}, ""
	},
	`(\w+) skills convert (\d+)% of (\w+) damage to (\w+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[2])+"DamageConvertTo"+firstToUpper(captures[3]), "BASE", utils.Float(captures[1])).Tag(mod.SkillType(firstToUpper(captures[0])))}, ""
	},
	`(\w+) damage from (\w+) spells contributes to flammability and ignite magnitudes`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG(firstToUpper(captures[0])+"CanIgnite").Tag(mod.SkillType("Spell"), mod.SkillType(firstToUpper(captures[1])))}, ""
	},
	`ignite inflicted with (\w+) spells deals chaos damage instead of fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("IgniteToChaos").Tag(mod.SkillType("Spell"), mod.SkillType(firstToUpper(captures[0]))), MOD("SkillData", "LIST", LuaTable{"key": "IgniteToChaos", "value": true}).Tag(mod.SkillType("Spell"), mod.SkillType(firstToUpper(captures[0])))}, ""
	},
	"mana leech recovers based on elemental damage types instead of physical damage": []mod.Mod{FLAG("ManaLeechBasedOnElementalDamage"), FLAG("Condition:NoManaLeechFromPhysicalDamage")},
	`(\d+)% chance to defend with double armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("ArmourDefense", mod.TypeMAX, 100).Tag(mod.Condition("ArmourMax")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(num/100, 1.0)*100).Tag(mod.Condition("ArmourAvg")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(math.Floor(num/100), 1.0)*100).Tag(mod.Condition("ArmourMax").Neg(true)).Tag(mod.Condition("ArmourAvg").Neg(true)),
		}, ""
	},
	`hits have (\d+)% chance to treat enemy monster elemental resistance values as inverted`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HitsInvertEleResChance", "CHANCE", (num / 100))}, ""
	},
	"off hand accuracy is equal to main hand accuracy while wielding a sword": []mod.Mod{FLAG("Condition:OffHandAccuracyIsMainHandAccuracy").Tag(mod.Condition("UsingSword"))},
	`(\d+)% increased accuracy rating at close range`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", "INC", num).Tag(mod.Condition("AtCloseRange"))}, ""
	},
	`(\d+)% more accuracy rating at close range`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", "MORE", num).Tag(mod.Condition("AtCloseRange"))}, ""
	},
	`(\d+)% increased accuracy rating against unique enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", "INC", num).Tag(mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	`(\d+)% more accuracy rating against unique enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", "MORE", num).Tag(mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	`(\d+)% chance to defend with (\d+)% of armour`: func(numChance float64, captures []string) ([]mod.Mod, string) {
		numArmourMultiplier := utils.Float(captures[1])
		return []mod.Mod{
			mod.NewFloat("ArmourDefense", mod.TypeMAX, utils.Float(captures[1])-100).Tag(mod.Condition("ArmourMax")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(numChance/100, 1.0)*(numArmourMultiplier-100)).Tag(mod.Condition("ArmourAvg")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(math.Floor(numChance/100), 1.0)*(numArmourMultiplier-100)).Tag(mod.Condition("ArmourMax").Neg(true)).Tag(mod.Condition("ArmourAvg").Neg(true)),
		}, ""
	},
	`defend with (\d+)% of armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourDefense", "MAX", (num - 100))}, ""
	},
	`defend with (\d+)% of armour during effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourDefense", "MAX", (num - 100))}, ""
	},
	`defend with (\d+)% of armour while not on low energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourDefense", "MAX", (num - 100)).Source(mod.Source("Armour and Energy Shield Mastery")).Tag(mod.Condition("LowEnergyShield").Neg(true))}, ""
	},
	`defend with (\d+)% of armour while you have energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourDefense", "MAX", (num - 100)).Tag(mod.Condition("HaveEnergyShield"))}, ""
	},
	`(\d+)% increased armour and energy shield from equipped body armour if equipped helmet, gloves and boots all have armour and energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Body ArmourESAndArmour", "INC", num).Tag(mod.StatThreshold("ArmourOnGloves", 1), mod.StatThreshold("EnergyShieldOnGloves", 1), mod.StatThreshold("ArmourOnHelmet", 1), mod.StatThreshold("EnergyShieldOnHelmet", 1), mod.StatThreshold("ArmourOnBoots", 1), mod.StatThreshold("EnergyShieldOnBoots", 1))}, ""
	},
	`brands have (\d+)% increased area of effect if (\d+)% of attached duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("BrandLastHalf"), mod.SkillType("Brand"))}, ""
	},
	"corrupted blood cannot be inflicted on you": []mod.Mod{FLAG("CorruptedBloodImmune")},
	"cannot be light stunned":                    []mod.Mod{FLAG("StunImmune")},
	"you cannot be hindered":                     []mod.Mod{FLAG("HinderImmune")},
	"you cannot be maimed":                       []mod.Mod{FLAG("MaimImmune")},
	"you cannot be impaled":                      []mod.Mod{FLAG("ImpaleImmune")},
	"cannot dodge roll or sprint":                []mod.Mod{FLAG("Condition:CannotDodgeRoll"), FLAG("Condition:CannotSprint")},
	"you cannot sprint":                          []mod.Mod{FLAG("Condition:CannotSprint")},
	`exerted attacks deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExertIncrease", "INC", num).Flag(ModFlagAttack)}, ""
	},
	`exerted attacks have (\d+)% chance to deal double damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExertDoubleDamageChance", "BASE", num).Flag(ModFlagAttack)}, ""
	},
	`empowered attacks deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EmpoweredIncrease", "INC", num).Flag(ModFlagAttack).Tag(mod.Condition("Empowered"))}, ""
	},
	`ancestrally boosted attacks deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AncestralBoostDamage", "INC", num).Flag(ModFlagAttack)}, ""
	},
	`(\d+)% increased area of effect of ancestrally boosted attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AncestralBoostAreaOfEffect", "INC", num).Flag(ModFlagAttack)}, ""
	},
	"every second slam skill you use yourself is ancestrally boosted": []mod.Mod{FLAG("AncestralEmpowerment").Tag(mod.SkillType("Slam")), FLAG("Condition:AncestrallyBoosted").Tag(mod.SkillType("Slam"))},
	"life leech is instant":                         []mod.Mod{MOD("InstantLifeLeech", "BASE", 100)},
	"mana leech is instant":                         []mod.Mod{MOD("InstantManaLeech", "BASE", 100)},
	"mana leech effects also recover energy shield": []mod.Mod{FLAG("ManaLeechRecoversEnergyShield")},
	`leeche?s? ([\d\.]+)% of (\w+) damage as mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[1])+"DamageManaLeech", "BASE", num)}, ""
	},
	`leeche?s? ([\d\.]+)% of (\w+) damage as life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[1])+"DamageLifeLeech", "BASE", num)}, ""
	},
	`leeche?s? ([\d\.]+)% of (\w+) damage as energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[1])+"DamageEnergyShieldLeech", "BASE", num)}, ""
	},
	`leeche?s? ([\d\.]+)% of (\w+) attack damage as mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[1])+"DamageManaLeech", "BASE", num).Flag(ModFlagAttack)}, ""
	},
	`leeche?s? ([\d\.]+)% of (\w+) attack damage as life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[1])+"DamageLifeLeech", "BASE", num).Flag(ModFlagAttack)}, ""
	},
	`leeche?s? ([\d\.]+)% of (\w+) attack damage as energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[1])+"DamageEnergyShieldLeech", "BASE", num).Flag(ModFlagAttack)}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:2759-2905
	`leech recovers based on y?o?u?r? ?(\w+) damage as well as physical damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("LifeLeechBasedOn" + firstToUpper(captures[0]) + "Damage"), FLAG("ManaLeechBasedOn" + firstToUpper(captures[0]) + "Damage"), FLAG("EnergyShieldLeechBasedOn" + firstToUpper(captures[0]) + "Damage")}, ""
	},
	`leech recovers based on y?o?u?r? ?(\w+) damage instead of physical damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("LifeLeechBasedOn" + firstToUpper(captures[0]) + "Damage"), FLAG("Condition:NoLifeLeechFromPhysicalDamage"), FLAG("ManaLeechBasedOn" + firstToUpper(captures[0]) + "Damage"), FLAG("Condition:NoManaLeechFromPhysicalDamage"), FLAG("EnergyShieldLeechBasedOn" + firstToUpper(captures[0]) + "Damage"), FLAG("Condition:NoEnergyShieldLeechFromPhysicalDamage")}, ""
	},
	"chance to hit with attacks can exceed 100%": []mod.Mod{FLAG("Condition:HitChanceCanExceed100").Tag(mod.SkillType("Attack"))},
	`gain additional critical hit chance equal to (\d+)% of excess chance to hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChance", "BASE", (0.01*num)).Tag(mod.Multiplier("ExcessHitChance"), mod.SkillType("Attack"))}, ""
	},
	`attacks using your weapons have added (\w+) damage equal to (\d+)% of the accuracy rating on the weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"Min", "BASE", 1).Flag(ModFlagAttack).Tag(mod.PercentStat("AccuracyOnWeapon 1", utils.Float(captures[1])), mod.SkillType("NonWeaponAttack").Neg(true), mod.Condition("MainHandAttack")), MOD(firstToUpper(captures[0])+"Max", "BASE", 1).Flag(ModFlagAttack).Tag(mod.PercentStat("AccuracyOnWeapon 1", utils.Float(captures[1])), mod.SkillType("NonWeaponAttack").Neg(true), mod.Condition("MainHandAttack")), MOD(firstToUpper(captures[0])+"Min", "BASE", 1).Flag(ModFlagAttack).Tag(mod.PercentStat("AccuracyOnWeapon 2", utils.Float(captures[1])), mod.SkillType("NonWeaponAttack").Neg(true), mod.Condition("OffHandAttack")), MOD(firstToUpper(captures[0])+"Max", "BASE", 1).Flag(ModFlagAttack).Tag(mod.PercentStat("AccuracyOnWeapon 2", utils.Float(captures[1])), mod.SkillType("NonWeaponAttack").Neg(true), mod.Condition("OffHandAttack"))}, ""
	},
	"evasion rating from equipped helmet, gloves and boots is doubled": []mod.Mod{MOD("Evasion", "MORE", 100).Tag(mod.SlotName("Helmet", "Boots", "Gloves"))},
	"evasion rating from equipped body armour is halved":               []mod.Mod{MOD("Evasion", "MORE", -50).Tag(mod.SlotName("Body Armour"))},
	`grants (\d+) passive skill points?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraPoints", "BASE", num)}, ""
	},
	`can allocate passives from the (\w+)'s starting point`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AlternateClassStart", "LIST", firstToUpper(captures[0]))}, ""
	},
	`can allocate passive skills from the (\w+)'s starting point`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AlternateClassStart", "LIST", firstToUpper(captures[0]))}, ""
	},
	`projectiles gain damage as they travel farther, dealing up to (\d+)% increased damage with hits to targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Flag(ModFlagHit | ModFlagProjectile).Tag(mod.DistanceRamp([][]float64{{35, 0}, {70, 1}}))}, ""
	},
	`(\d+)% chance to gain elusive on kill`:                           []mod.Mod{FLAG("Condition:CanBeElusive")},
	"immun[ei]t?y? to elemental ailments while on consecrated ground": []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("OnConsecratedGround"))},
	`poison you inflict with critical hits deals (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagPoison).Tag(mod.Condition("CriticalStrike"))}, ""
	},
	`(\d+)% chance to gain elusive on critical hit`: []mod.Mod{FLAG("Condition:CanBeElusive")},
	`(\d+)% more damage while there is at most one rare or unique enemy nearby`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).Tag(mod.Condition("AtMostOneNearbyRareOrUniqueEnemy"))}, ""
	},
	`(\d+)% reduced damage taken while there are at least two rare or unique enemies nearby`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageTaken", "INC", -num).Tag(mod.MultiplierThreshold("NearbyRareOrUniqueEnemies").Threshold(2))}, ""
	},
	"you take no extra damage from critical hits while elusive":                  []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.Condition("Elusive"))},
	`gain \d+ rage when you kill an enemy`:                                       []mod.Mod{FLAG("Condition:CanGainRage")},
	`gain \d+ rage when you use a warcry`:                                        []mod.Mod{FLAG("Condition:CanGainRage")},
	`you and nearby party members gain \d+ rage when you warcry`:                 []mod.Mod{FLAG("Condition:CanGainRage")},
	`gain \d+ rage on hit with attacks, no more than once every [\d\.]+ seconds`: []mod.Mod{FLAG("Condition:CanGainRage")},
	`gain \d+ rage on attack hit`:                                                []mod.Mod{FLAG("Condition:CanGainRage")},
	`gain \d+ rage on hit with axes or swords`:                                   []mod.Mod{FLAG("Condition:CanGainRage")},
	`gain \d+ rage on melee hit`:                                                 []mod.Mod{FLAG("Condition:CanGainRage")},
	`gain \d+ rage on melee weapon hit`:                                          []mod.Mod{FLAG("Condition:CanGainRage")},
	`gain \d+ rage on ([\D]+)`:                                                   []mod.Mod{FLAG("Condition:CanGainRage")},
	`grants \d+ rage on ([\D]+)`:                                                 []mod.Mod{FLAG("Condition:CanGainRage")},
	`gain \d+ rage on hit with axes`:                                             []mod.Mod{FLAG("Condition:CanGainRage")},
	`gain \d+ rage when hit by an enemy`:                                         []mod.Mod{FLAG("Condition:CanGainRage")},
	`gain \d+ rage on hit with retaliation skills`:                               []mod.Mod{FLAG("Condition:CanGainRage")},
	`while a unique enemy is in your presence, gain \d+ rage on hit with attacks, no more than once every [\d\.]+ seconds`:        []mod.Mod{FLAG("Condition:CanGainRage").Tag(mod.ActorCondition("enemy", "RareOrUnique"))},
	`while a pinnacle atlas boss is in your presence, gain \d+ rage on hit with attacks, no more than once every [\d\.]+ seconds`: []mod.Mod{FLAG("Condition:CanGainRage").Tag(mod.ActorCondition("enemy", "PinnacleBoss"))},
	"no rage effect": []mod.Mod{MOD("RageEffect", "OVERRIDE", 0)},
	"inherent effects from having rage are tripled": []mod.Mod{MOD("RageEffect", "MORE", 200)},
	"inherent effects from having rage are doubled": []mod.Mod{MOD("RageEffect", "MORE", 100)},
	`cannot be stunned while you have at least (\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("StunImmune").Tag(mod.MultiplierThreshold("Rage").Threshold(num))}, ""
	},
	`lose ([\d\.]+)% of life per second per rage while you are not losing rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeDegen", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.Multiplier("RageEffect"))}, ""
	},
	`if you've warcried recently, you and nearby allies have (\d+)% increased attack speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("Speed", "INC", num).Flag(ModFlagAttack)}).Tag(mod.Condition("UsedWarcryRecently"))}, ""
	},
	`gain (\d+)% increased armour per (\d+) power for 8 seconds when you warcry, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Armour", "INC", num).Tag(mod.Multiplier("WarcryPower").Div(utils.Float(captures[1])).GlobalLimit(utils.Float(captures[2])).GlobalLimitKey("WarningCall"), mod.Condition("UsedWarcryInPast8Seconds"))}, ""
	},
	`warcries grant (\d+) rage per (\d+) power if you have less than (\d+) rage`: []mod.Mod{FLAG("Condition:CanGainRage")},
	`warcries grant (\d+) rage per (\d+) enemy power, up to (\d+)`:               []mod.Mod{FLAG("Condition:CanGainRage")},
	`exerted attacks deal (\d+)% more attack damage if a warcry sacrificed rage recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExertAttackIncrease", "MORE", num).Flag(ModFlagAttack)}, ""
	},
	`deal (\d+)% less damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", -num)}, ""
	},
	"warcries exert twice as many attacks":                               []mod.Mod{MOD("ExtraExertedAttacks", "MORE", 100)},
	"skills gain a base life cost equal to base mana cost":               []mod.Mod{MOD("BaseManaCostAsLifeCost", "BASE", 100)},
	"gain energy shield from equipped body armour as extra maximum life": []mod.Mod{MOD("Life", "BASE", 1).Tag(mod.PerStat(1, "EnergyShieldOnBody Armour"))},
	`gain additional maximum life equal to (\d+)% of the (i?t?e?m? ?)energy shield on equipped body armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Life", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnBody Armour", num))}, ""
	},
	"bleeding you inflict on cursed targets is aggravated": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:BleedAggravated").Tag(mod.Condition("Cursed"))})},
	"critical hits ignore monster armour":                  []mod.Mod{FLAG("IgnoreEnemyArmour").Tag(mod.Condition("CriticalStrike"))},
	`(\d+)% of life loss from hits is prevented, then that much life is lost over 4 seconds instead`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeLossPrevented", "BASE", num)}, ""
	},
	"cannot be stunned while you have fortify":                 []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("Fortified"))},
	"cannot be stunned while fortified":                        []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("Fortified"))},
	"you cannot be stunned while at maximum endurance charges": []mod.Mod{FLAG("StunImmune").Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax"))},
	"fortify": []mod.Mod{FLAG("Condition:Fortified")},
	`you have (\d+) fortification`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinimumFortification", "BASE", num)}, ""
	},
	"nearby allies count as having fortification equal to yours":            []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("YourFortifyEqualToParent", "FLAG", true).Tag(mod.GlobalEffect("Global").Unscalable(true)), OnlyAllies: true})},
	"enemies taunted by you cannot evade attacks":                           []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("CannotEvade").Tag(mod.Condition("Taunted"))})},
	"targets that are blinded, maimed, and bleeding cannot evade your hits": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("CannotEvade").Tag(mod.ActorCondition("", "Blinded"), mod.ActorCondition("", "Maimed"), mod.ActorCondition("", "Bleeding"))})},
	`if you've impaled an enemy recently, you and nearby allies have \+(\d+) to armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("Armour", "BASE", num)}).Tag(mod.Condition("ImpaledRecently"))}, ""
	},
	"your hits permanently intimidate enemies that are on full life": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated").Tag(mod.Condition("ChampionIntimidate"))})},
	`you and allies affected by your placed banners regenerate ([\d\.]+)% of life per second for each stage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("AffectedByPlacedBanner"), mod.Multiplier("BannerValour"))})}, ""
	},
	`you and allies near your banner regenerate ([\d\.]+)% of life per second for each valour consumed for that banner`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("AffectedByPlacedBanner"), mod.Multiplier("BannerValour"))})}, ""
	},
	`enemies near your totems take (\d+)% increased physical and fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("PhysicalDamageTaken", "INC", num)}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireDamageTaken", "INC", num)})}, ""
	},
	`every (\d+) seconds, gain (\d+)% of physical damage as extra fire damage for (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsFire", "BASE", utils.Float(captures[1])).Tag(mod.Condition("NgamahuFlamesAdvance"))}, ""
	},
	`(\d+)% more damage for each endurance charge lost recently, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).Tag(mod.Multiplier("EnduranceChargesLostRecently").Limit(utils.Float(captures[1])).LimitTotal(true))}, ""
	},
	`(\d+)% more damage if you've lost an endurance charge in the past 8 seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).Tag(mod.Condition("LostEnduranceChargeInPast8Sec"))}, ""
	},
	`trigger level (\d+) (.+) when you attack with a non\-vaal slam or strike skill near an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	//pob2:block:src/Modules/ModParser.lua:2922-3387
	`enemies in your presence are slowed by (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ActionSpeed", "INC", (num * -1))}).Tag(mod.Condition("EnemyInPresence"))}, ""
	},
	"projectiles pierce all nearby targets":                          []mod.Mod{FLAG("PierceAllTargets")},
	"you have no accuracy penalty at distance":                       []mod.Mod{FLAG("NoAccuracyDistancePenalty")},
	"enemies have an accuracy penalty against you based on distance": []mod.Mod{FLAG("EnemyAccuracyDistancePenalty")},
	"has no accuracy penalty from range":                             []mod.Mod{FLAG("NoAccuracyDistancePenalty").Tag(mod.Condition("{Hand}Attack"))},
	`gain \+(\d+) life when you hit a bleeding enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnHit", "BASE", num).Flag(ModFlagHit).Tag(mod.ActorCondition("enemy", "Bleeding"))}, ""
	},
	`(\d+)% increased blink arrow and mirror arrow cooldown recovery speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "INC", num).Tag(mod.SkillName("Blink Arrow", "Mirror Arrow").Transfigured(true))}, ""
	},
	"critical hits which inflict bleeding also inflict rupture": []mod.Mod{FLAG("Condition:CanInflictRupture").Tag(mod.Condition("NeverCrit").Neg(true))},
	`projectiles deal (\d+)% more damage for each remaining chain`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).Flag(ModFlagProjectile).Tag(mod.PerStat(1, "ChainRemaining"))}, ""
	},
	`projectiles deal (\d+)% increased damage with hits and ailments for each remaining chain`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(KeywordFlagHit|KeywordFlagAilment).Tag(mod.PerStat(1, "ChainRemaining"), mod.SkillType("Projectile"))}, ""
	},
	`projectiles deal (\d+)% increased damage for each remaining chain`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Flag(ModFlagProjectile).Tag(mod.PerStat(1, "ChainRemaining"))}, ""
	},
	"far shot": []mod.Mod{FLAG("FarShot")},
	`projectiles gain damage as they travel farther, dealing up to (\d+)% more damage with hits and ailments`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagHit | KeywordFlagAilment).Tag(mod.DistanceRamp([][]float64{{35, 0}, {70, 1}}))}, ""
	},
	`(\d+)% increased mirage archer duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MirageArcherDuration", "INC", num)}, ""
	},
	`([\-\+]\d+) to maximum number of summoned mirage archers`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MirageArcherMaxCount", "BASE", num)}, ""
	},
	`([\-\+]\d+) to maximum number of sacred wisps`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SacredWispsMaxCount", "BASE", num)}, ""
	},
	`projectiles deal (\d+)% more hit damage to targets in the first ([\d\.]+) metres of their movement, scaling [du][op]w?n? with distance travelled to reach (\d+)% after ([\d\.]+) metres`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", 100).Flag(ModFlagHit | ModFlagProjectile).Tag(mod.DistanceRamp([][]float64{{(10 * utils.Float(captures[1])), (num / 100)}, {(10 * utils.Float(captures[3])), (utils.Float(captures[2]) / 100)}}))}, ""
	},
	`benefits from consuming (\w+) charges for your skills have (\d+)% chance to be doubled`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:Consumed"+firstToUpper(captures[0])+"ChargeEffect", "BASE", utils.Float(captures[1]))}, ""
	},
	`companions gain added attack damage equal to (\d+)% of your main hand weapon's damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("GainMainHandDmgFromParent")}).Tag(mod.SkillType("CreatesCompanion")), MOD("Multiplier:MainHandDamageToAllies", "BASE", num)}, ""
	},
	`(\d+)% of your current energy shield is added to your armour for determining your physical damage reduction from armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldAppliesToPhysicalDamageTaken", "BASE", num)}, ""
	},
	`gain (\d+)% increased area of effect for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("PendulumOfDestructionAreaOfEffect"))}, ""
	},
	`gain (\d+)% increased elemental damage for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalDamage", "INC", num).Tag(mod.Condition("PendulumOfDestructionElementalDamage"))}, ""
	},
	`for each element you've been hit by damage of recently, (\d+)% increased damage of that element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireDamage", "INC", num).Tag(mod.Condition("HitByFireDamageRecently")), MOD("ColdDamage", "INC", num).Tag(mod.Condition("HitByColdDamageRecently")), MOD("LightningDamage", "INC", num).Tag(mod.Condition("HitByLightningDamageRecently"))}, ""
	},
	`for each element you've been hit by damage of recently, (\d+)% reduced damage taken of that element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireDamageTaken", "INC", -num).Tag(mod.Condition("HitByFireDamageRecently")), MOD("ColdDamageTaken", "INC", -num).Tag(mod.Condition("HitByColdDamageRecently")), MOD("LightningDamageTaken", "INC", -num).Tag(mod.Condition("HitByLightningDamageRecently"))}, ""
	},
	`gain convergence when you hit a unique enemy, no more than once every \d+ seconds`: []mod.Mod{FLAG("Condition:CanGainConvergence")},
	`(\d+)% increased area of effect while you don't have convergence`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("Convergence").Neg(true))}, ""
	},
	`exposure you inflict applies an extra (\-?\d+)% to the affected resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraExposure", "BASE", -num)}, ""
	},
	`exposure you inflict lowers the affected resistance by an additional (\-?\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraExposure", "BASE", num)}, ""
	},
	`exposure you inflict lowers resistances by an additional (\-?\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraExposure", "BASE", num)}, ""
	},
	"cannot take reflected elemental damage":                                              []mod.Mod{MOD("ElementalReflectedDamageTaken", "MORE", -100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	`every \d+ seconds:`:                                                                  []mod.Mod{},
	`gain chilling conflux for \d seconds`:                                                []mod.Mod{FLAG("PhysicalCanChill").Tag(mod.Condition("ChillingConflux")), FLAG("LightningCanChill").Tag(mod.Condition("ChillingConflux")), FLAG("FireCanChill").Tag(mod.Condition("ChillingConflux")), FLAG("ChaosCanChill").Tag(mod.Condition("ChillingConflux"))},
	`gain shocking conflux for \d seconds`:                                                []mod.Mod{MOD("EnemyShockChance", "BASE", 100).Tag(mod.Condition("ShockingConflux")), FLAG("PhysicalCanShock").Tag(mod.Condition("ShockingConflux")), FLAG("ColdCanShock").Tag(mod.Condition("ShockingConflux")), FLAG("FireCanShock").Tag(mod.Condition("ShockingConflux")), FLAG("ChaosCanShock").Tag(mod.Condition("ShockingConflux"))},
	`gain igniting conflux for \d seconds`:                                                []mod.Mod{MOD("EnemyIgniteChance", "BASE", 100).Tag(mod.Condition("IgnitingConflux")), FLAG("PhysicalCanIgnite").Tag(mod.Condition("IgnitingConflux")), FLAG("LightningCanIgnite").Tag(mod.Condition("IgnitingConflux")), FLAG("ColdCanIgnite").Tag(mod.Condition("IgnitingConflux")), FLAG("ChaosCanIgnite").Tag(mod.Condition("IgnitingConflux"))},
	`gain chilling, shocking and igniting conflux for \d seconds`:                         []mod.Mod{},
	"you have igniting, chilling and shocking conflux while affected by glorious madness": []mod.Mod{FLAG("PhysicalCanChill").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("LightningCanChill").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("FireCanChill").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("ChaosCanChill").Tag(mod.Condition("AffectedByGloriousMadness")), MOD("EnemyIgniteChance", "BASE", 100).Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("PhysicalCanIgnite").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("LightningCanIgnite").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("ColdCanIgnite").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("ChaosCanIgnite").Tag(mod.Condition("AffectedByGloriousMadness")), MOD("EnemyShockChance", "BASE", 100).Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("PhysicalCanShock").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("ColdCanShock").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("FireCanShock").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("ChaosCanShock").Tag(mod.Condition("AffectedByGloriousMadness"))},
	"immun[ei]t?y? to elemental ailments while affected by glorious madness":              []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("AffectedByGloriousMadness"))},
	"immun[ei]t?y? to elemental ailments while focus?sed":                                 []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("Focused"))},
	"summoned golems are immune to elemental damage":                                      []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Elemancer")}).Tag(mod.SkillType("Golem")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ElementalDamageTaken", "MORE", -100)}).Tag(mod.SkillType("Golem"))},
	`(\d+)% increased golem damage per summoned golem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num)}).Tag(mod.SkillType("Golem"), mod.PerStat(1, "ActiveGolemLimit"))}, ""
	},
	`shocks from your hits always increase damage taken by at least (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ShockMinimum", "BASE", num)}, ""
	},
	`chills from your hits always reduce action speed by at least (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChillBase", "BASE", num)}, ""
	},
	`(\d+)% more damage with ignites you inflict with hits for which the highest damage type is fire`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagIgnite).Tag(mod.Condition("FireIsHighestDamageType")), FLAG("ChecksHighestDamage")}, ""
	},
	`(\d+)% more effect of cold ailments you inflict with hits for which the highest damage type is cold`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyChillMagnitude", "MORE", num).Tag(mod.Condition("ColdIsHighestDamageType")), FLAG("ChecksHighestDamage")}, ""
	},
	`(\d+)% more effect of lightning ailments you inflict with hits if the highest damage type is lightning`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyShockMagnitude", "MORE", num).Tag(mod.Condition("LightningIsHighestDamageType")), FLAG("ChecksHighestDamage")}, ""
	},
	`your chills can reduce action speed by up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChillMax", "OVERRIDE", num)}, ""
	},
	"your hits always ignite":         []mod.Mod{MOD("EnemyIgniteChance", "BASE", 100)},
	"hits always ignite":              []mod.Mod{MOD("EnemyIgniteChance", "BASE", 100)},
	"your hits always shock":          []mod.Mod{MOD("EnemyShockChance", "BASE", 100)},
	"hits always shock":               []mod.Mod{MOD("EnemyShockChance", "BASE", 100)},
	"always freeze, shock and ignite": []mod.Mod{MOD("EnemyFreezeChance", "BASE", 100), MOD("EnemyShockChance", "BASE", 100), MOD("EnemyIgniteChance", "BASE", 100)},
	`all damage ?w?i?t?h? ?h?i?t?s? can (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Can" + firstToUpper(captures[0]))}, ""
	},
	"all damage from hits contributes to shock chance":                                                                     []mod.Mod{FLAG("PhysicalCanShock"), FLAG("FireCanShock"), FLAG("ColdCanShock"), FLAG("ChaosCanShock")},
	"fire damage from hits contributes to shock chance instead of flammability and ignite magnitudes":                      []mod.Mod{FLAG("FireCanShock"), FLAG("FireCannotIgnite")},
	"cold damage from hits contributes to flammability and ignite magnitudes instead of chill magnitude or freeze buildup": []mod.Mod{FLAG("ColdCanIgnite"), FLAG("ColdCannotChill"), FLAG("ColdCannotFreeze")},
	"lightning damage from hits contributes to freeze buildup instead of shock chance":                                     []mod.Mod{FLAG("LightningCanFreeze"), FLAG("LightningCannotShock")},
	`all damage from hits contributes to (\w+) magnitude`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Can" + firstToUpper(captures[0]))}, ""
	},
	`all damage from hits with this weapon contributes to (\w+) magnitude`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Can"+firstToUpper(captures[0])).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`all damage from hits against (\w+) targets contributes to (\w+) magnitude`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Can" + firstToUpper(captures[1])).Tag(mod.ActorCondition("enemy", firstToUpper(captures[0])))}, ""
	},
	"other aegis skills are disabled": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("DisableSkill").Tag(mod.SkillType("Aegis")),
			FLAG("EnableSkill").Tag(mod.SkillName()),
		}, ""
	},
	`primal aegis can take (\d+) elemental damage per allocated notable passive skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ElementalAegisValue", "MAX", num).Tag(mod.Multiplier("AllocatedNotable").Base(0)).Tag(mod.GlobalEffect("Buff").Unscalable(true)),
		}, ""
	},
	"chance to block spell damage is equal to chance to block attack damage":                                 []mod.Mod{FLAG("SpellBlockChanceIsBlockChance")},
	"maximum chance to block spell damage is equal to maximum chance to block attack damage":                 []mod.Mod{FLAG("SpellBlockChanceMaxIsBlockChanceMax")},
	`attack damage is lucky if you[' ]h?a?ve blocked in the past (\d+) seconds`:                              []mod.Mod{MOD("LuckyHitsChance", "BASE", 100).Flag(ModFlagAttack).Tag(mod.Condition("BlockedRecently"))},
	`attack damage while dual wielding is lucky if you[' ]h?a?ve blocked in the past (\d+) seconds`:          []mod.Mod{MOD("LuckyHitsChance", "BASE", 100).Flag(ModFlagAttack).Tag(mod.Condition("BlockedRecently"), mod.Condition("DualWielding"))},
	`hits ignore enemy monster physical damage reduction if you[' ]h?a?ve blocked in the past (\d+) seconds`: []mod.Mod{FLAG("IgnoreEnemyPhysicalDamageReduction").Tag(mod.Condition("BlockedRecently"))},
	`(\d+)% more attack and movement speed per challenger charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "MORE", num).Flag(ModFlagAttack).Tag(mod.Multiplier("ChallengerCharge")), MOD("MovementSpeed", "MORE", num).Tag(mod.Multiplier("ChallengerCharge"))}, ""
	},
	`gain (\d+)% chance to block from equipped shield instead of the shield's value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReplaceShieldBlock", "OVERRIDE", num).Tag(mod.Condition("UsingShield"))}, ""
	},
	`gain (\d+)% base chance to block from equipped shield instead of the shield's value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReplaceShieldBlock", "OVERRIDE", num).Tag(mod.Condition("UsingShield"))}, ""
	},
	`deal (\d+)% more damage with hits and ailments to rare and unique enemies for each second they've ever been in your presence, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagHit|KeywordFlagAilment).Tag(mod.Multiplier("EnemyPresenceSeconds").Limit(utils.Float(captures[1])).Actor("enemy"), mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	`deal (\d+)% more damage with hits and ailments to rare and unique enemies for every (\d+) seconds they've ever been in your presence, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagHit|KeywordFlagAilment).Tag(mod.Multiplier("EnemyPresenceSeconds").Div(utils.Float(captures[1])).Limit(utils.Float(captures[2])).Actor("enemy"), mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	`(\d+)% more damage with hits and ailments against enemies that are on low life while you are wielding an axe`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagHit|KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "LowLife"), mod.Condition("UsingAxe"))}, ""
	},
	`retaliation skills have (\d+)% increased speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "INC", num).Tag(mod.SkillType("Retaliation")), MOD("WarcrySpeed", "INC", num).KeywordFlag(KeywordFlagWarcry).Tag(mod.SkillType("Retaliation"))}, ""
	},
	`grants armour equal to (\d+)% of your reserved life to you and nearby allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GrantReservedLifeAsAura", "LIST", LuaTable{"mod": MOD("Armour", "BASE", (num / 100))})}, ""
	},
	`grants armour equal to (\d+)% of your reserved mana to you and nearby allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GrantReservedManaAsAura", "LIST", LuaTable{"mod": MOD("Armour", "BASE", (num / 100))})}, ""
	},
	`grants maximum energy shield equal to (\d+)% of your reserved mana to you and nearby allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GrantReservedManaAsAura", "LIST", LuaTable{"mod": MOD("EnergyShield", "BASE", (num / 100))})}, ""
	},
	"warcries cost no mana": []mod.Mod{MOD("ManaCost", "MORE", -100).KeywordFlag(KeywordFlagWarcry)},
	`\+(\d+)% chance to block attack damage for \d seconds? every \d seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BlockChance", "BASE", num).Tag(mod.Condition("BastionOfHopeActive"))}, ""
	},
	`if you've blocked in the past \d+ seconds, you and nearby allies cannot be stunned`: []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("StunImmune")}).Tag(mod.Condition("BlockedRecently"), mod.GlobalEffect("Global").Unscalable(true))},
	`if you've attacked recently, you and nearby allies have \+(\d+)% chance to block attack damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("BlockChance", "BASE", num)}).Tag(mod.Condition("AttackedRecently"))}, ""
	},
	`if you've cast a spell recently, you and nearby allies have \+(\d+)% chance to block spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("SpellBlockChance", "BASE", num)}).Tag(mod.Condition("CastSpellRecently"))}, ""
	},
	`while there is at least one nearby ally, you and nearby allies deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("Damage", "MORE", num)}).Tag(mod.MultiplierThreshold("NearbyAlly").Threshold(1))}, ""
	},
	"while there are at least five nearby allies, you and nearby allies have onslaught": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("Onslaught")}).Tag(mod.MultiplierThreshold("NearbyAlly").Threshold(5))},
	`linked targets and allies in your link beams have \+(\d+)% to all maximum elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("ElementalResistMax", "BASE", num).Tag(mod.Condition("AffectedByLink").Neg(true)), OnlyAllies: true}).Tag(mod.MultiplierThreshold("LinkedTargets").Threshold(1)), MOD("ExtraLinkEffect", "LIST", LuaTable{"mod": MOD("ElementalResistMax", "BASE", num).Tag(mod.GlobalEffect("Global").Unscalable(true))})}, ""
	},
	"enemies in your link beams cannot apply elemental ailments": []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.ActorCondition("enemy", "BetweenYouAndLinkedTarget"))},
	`(\d+)% of damage from hits is taken from your sentinel of radiance's life before you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("takenFromRadianceSentinelBeforeYou", "BASE", num)}, ""
	},
	`you and your totems regenerate ([\d\.]+)% of life per second for each summoned totem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeRegenPercent", "BASE", num).Tag(mod.PerStat(0, "TotemsSummoned")),
			MOD("LifeRegenPercent", "BASE", num).KeywordFlag(KeywordFlagTotem),
		}, ""
	},
	`enemies take (\d+)% increased damage for each of your brands attached to them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Multiplier("BrandsAttached"))})}, ""
	},
	`non-damaging ailments have (\d+)% reduced effect on you while you have arcane surge`: func(num float64, captures []string) ([]mod.Mod, string) {
		ailments := dataNonDamagingAilmentTypeList
		mods := make([]mod.Mod, len(ailments))
		for i, ailment := range ailments {
			mods[i] = MOD("Self"+ailment+"Effect", "INC", -num).Tag(mod.Condition("AffectedByArcaneSurge"))
		}
		return mods, ""
	},
	"immun[ei]t?y? to elemental ailments while you have arcane surge": []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("AffectedByArcaneSurge"))},
	`brands have (\d+)% more activation frequency if (\d+)% of attached duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BrandActivationFrequency", "MORE", num).Tag(mod.Condition("BrandLastQuarter"))}, ""
	},
	`arcane surge grants (\d+)% more spell damage to you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArcaneSurgeDamage", "MAX", num)}, ""
	},
	"critical hits ignore enemy monster elemental resistances": []mod.Mod{FLAG("IgnoreElementalResistances").Tag(mod.Condition("CriticalStrike"))},
	`non\-critical hits penetrate (\d+)% of enemy elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalPenetration", "BASE", num).Tag(mod.Condition("CriticalStrike").Neg(true))}, ""
	},
	`consecrated ground you create applies (\d+)% increased damage taken to enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTakenConsecratedGround", "INC", num).Tag(mod.Condition("OnConsecratedGround"))})}, ""
	},
	"you have consecrated ground around you while stationary":                                    []mod.Mod{FLAG("Condition:OnConsecratedGround").Tag(mod.Condition("Stationary"))},
	"consecrated ground you create grants immun[ei]t?y? to elemental ailments to you and allies": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("ElementalAilmentImmune").Tag(mod.Condition("OnConsecratedGround"))})},
	"gain fanaticism for 4 seconds on reaching maximum fanatic charges":                          []mod.Mod{FLAG("Condition:CanGainFanaticism")},
	`(\d+)% increased critical hit chance per point of strength or intelligence, whichever is lower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChance", "INC", num).Tag(mod.PerStat(1, "Str"), mod.Condition("IntHigherThanStr")), MOD("CritChance", "INC", num).Tag(mod.PerStat(1, "Int"), mod.Condition("IntHigherThanStr").Neg(true))}, ""
	},
	"consecrated ground you create causes life regeneration to also recover energy shield for you and allies": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("LifeRegenerationRecoversEnergyShield").Tag(mod.Condition("OnConsecratedGround"))})},
	`(\d+)% more attack damage for each non\-instant spell you've cast in the past 8 seconds, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).Flag(ModFlagAttack).Tag(mod.Multiplier("CastLast8Seconds").Limit(utils.Float(captures[1])).LimitTotal(true))}, ""
	},
	"armour received from body armour is doubled":                                      []mod.Mod{FLAG("Unbreakable")},
	"armour from equipped body armour is doubled":                                      []mod.Mod{FLAG("Unbreakable")},
	"action speed cannot be modified to below base value":                              []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"movement speed cannot be modified to below base value":                            []mod.Mod{FLAG("MovementSpeedCannotBeBelowBase")},
	"you cannot be slowed to below base speed":                                         []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"cannot be slowed to below base speed":                                             []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"gain accuracy rating equal to your strength":                                      []mod.Mod{MOD("Accuracy", "BASE", 1).Tag(mod.PerStat(1, "Str"))},
	"gain accuracy rating equal to twice your strength":                                []mod.Mod{MOD("Accuracy", "BASE", 2).Tag(mod.PerStat(1, "Str"))},
	"your life cannot change while you have energy shield":                             []mod.Mod{FLAG("EternalLife")},
	"while you are not on low mana, you and allies in your presence have unholy might": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("Condition:UnholyMight")}).Tag(mod.Condition("LowMana").Neg(true))},
	`non\-channelling spells cost an additional (\d+)% of maximum energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ESCostBase", "BASE", 1).KeywordFlag(KeywordFlagSpell).Tag(mod.PercentStat("EnergyShield", num).Floor(), mod.SkillType("Channel").Neg(true))}, ""
	},
	`non\-channelling spells consume a power charge if able to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagSpell).Tag(mod.SkillType("Channel").Neg(true), mod.MultiplierThreshold("RemovablePowerCharge").Threshold(1))}, ""
	},
	`spells consume a power charge if able to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagSpell).Tag(mod.MultiplierThreshold("RemovablePowerCharge").Threshold(1))}, ""
	},
	"no inherent mana regeneration": []mod.Mod{FLAG("Condition:NoInherentManaRegen")},
	`\+1 ring slot`:                 []mod.Mod{FLAG("AdditionalRingSlot")},
	`regenerate (\D+) equal to (\d+)% of maximum (\D+) per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(combineToUpper(captures[0])+"Regen", "BASE", 1).Tag(mod.PercentStat(combineToUpper(captures[2]), utils.Float(captures[1])))}, ""
	},
	`can socket a non\-unique basic jewel into the phylactery`: []mod.Mod{},
	`spells for which this sacrifice was fully made deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("EldritchEmpowerment"), MOD("Damage", "MORE", num).Flag(ModFlagSpell).Tag(mod.Condition("EldritchEmpowermentSacrifice"))}, ""
	},
	"skeletal minions you would create instead grant you umbral souls for each minion you would have created": []mod.Mod{FLAG("UmbralWell")},
	"your offerings affect you instead of your minions":                                                       []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", LuaTable{"key": "buffNotPlayer", "value": false})}).Tag(mod.SkillName("Bone Offering", "Pain Offering", "Soul Offering")), MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", LuaTable{"key": "buffMinions", "value": false})}).Tag(mod.SkillName("Bone Offering", "Pain Offering", "Soul Offering"))},
	`offerings created by culling enemies have (\d+)% increased effect per power of culled enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("UnwillingOffering"), MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("BuffEffect", "INC", num).Tag(mod.Multiplier("UnwillingOfferingPower").Limit(20))}).Tag(mod.SkillName("Bone Offering", "Pain Offering", "Soul Offering"))}, ""
	},
	`\+(\d) weapon set passive skill points`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("WeaponSetPassivePoints", "BASE", num)}, ""
	},
	`(\d+) passive skill points become weapon set skill points`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PassivePointsToWeaponSetPoints", "BASE", num)}, ""
	},
	"your offering skills also affect you": []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", LuaTable{"key": "buffNotPlayer", "value": false})}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"))},
	`your offerings have (\d+)% reduced effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("BuffEffectOnPlayer", "INC", -num)}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"))}, ""
	},
	`your offerings have (\d+)% increased effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("BuffEffectOnPlayer", "INC", num)}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering"))}, ""
	},
	`if you've consumed a corpse recently, you and your minions have (\d+)% increased area of effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("ConsumedCorpseRecently")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("AreaOfEffect", "INC", num)}).Tag(mod.Condition("ConsumedCorpseRecently"))}, ""
	},
	`with at least one nearby corpse, you and nearby allies deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("Damage", "MORE", num)}).Tag(mod.MultiplierThreshold("NearbyCorpse").Threshold(1))}, ""
	},
	`with at least one nearby corpse, nearby enemies deal (\d+)% reduced damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("Damage", "INC", -num)}).Tag(mod.MultiplierThreshold("NearbyCorpse").Threshold(1))}, ""
	},
	`for each nearby corpse, you and nearby allies regenerate ([\d\.]+)% of energy shield per second, up to ([\d\.]+)% per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("EnergyShieldRegenPercent", "BASE", num)}).Tag(mod.Multiplier("NearbyCorpse").Limit(utils.Float(captures[1])).LimitTotal(true))}, ""
	},
	`for each nearby corpse, you and nearby allies regenerate (\d+) mana per second, up to (\d+) per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("ManaRegen", "BASE", num)}).Tag(mod.Multiplier("NearbyCorpse").Limit(utils.Float(captures[1])).LimitTotal(true))}, ""
	},
	`(\d+)% increased attack and cast speed for each corpse consumed recently, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "INC", num).Tag(mod.Multiplier("CorpseConsumedRecently").Limit((utils.Float(captures[1]) / num)))}, ""
	},
	"enemies near corpses you spawned recently are chilled and shocked": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Chilled")}).Tag(mod.Condition("SpawnedCorpseRecently")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Shocked")}).Tag(mod.Condition("SpawnedCorpseRecently")), MOD("ChillBase", "BASE", dataNonDamagingAilmentDefault["Chill"]).Tag(mod.Condition("SpawnedCorpseRecently")), MOD("ShockBase", "BASE", dataNonDamagingAilmentDefault["Shock"]).Tag(mod.Condition("SpawnedCorpseRecently"))},
	`regenerate (\d+)% of energy shield over 2 seconds when you consume a corpse`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldRegenPercent", "BASE", (num / 2)).Tag(mod.Condition("ConsumedCorpseInPast2Sec"))}, ""
	},
	`regenerate (\d+)% of mana over 2 seconds when you consume a corpse`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaRegen", "BASE", 1).Tag(mod.PercentStat("Mana", (num/2)), mod.Condition("ConsumedCorpseInPast2Sec"))}, ""
	},
	`corpses you spawn have (\d+)% increased maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CorpseLife", "INC", num)}, ""
	},
	`corpses you spawn have (\d+)% reduced maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CorpseLife", "INC", -num)}, ""
	},
	`minions gain added physical damage equal to (\d+)% of maximum energy shield on your equipped helmet`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("PhysicalMin", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnHelmet", num).Actor("parent"))}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("PhysicalMax", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnHelmet", num).Actor("parent"))})}, ""
	},
	`when you kill an enemy, for each curse on that enemy, gain (\d+)% of non\-chaos damage as extra chaos damage for 4 seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("NonChaosDamageGainAsChaos", "BASE", num).Tag(mod.Condition("KilledRecently"), mod.Multiplier("CurseOnEnemy"))}, ""
	},
	"cannot be stunned while you have energy shield":                              []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("HaveEnergyShield"))},
	`every second, inflict withered on nearby enemies for (\d+) seconds`:          []mod.Mod{FLAG("Condition:CanWither")},
	"unwithered enemies are withered for 8 seconds when they enter your presence": []mod.Mod{FLAG("Condition:CanWither")},
	`nearby hindered enemies deal (\d+)% reduced damage over time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageOverTime", "INC", -num)}).Tag(mod.ActorCondition("enemy", "Hindered"))}, ""
	},
	`nearby chilled enemies deal (\d+)% reduced damage with hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("Damage", "INC", -num)}).Tag(mod.ActorCondition("enemy", "Chilled"))}, ""
	},
	"always poison on hit while using a flask": []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	`poisons you inflict during any flask effect have (\d+)% chance to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", ((utils.Float(captures[1]) * num) / 100)).KeywordFlag(KeywordFlagPoison).Tag(mod.Condition("UsingFlask"))}, ""
	},
	"immun[ei]t?y? to elemental ailments during any flask effect":                                                                           []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("UsingFlask"))},
	`grant bonuses to non\-channelling skills you use by consuming (\d+) charges from a flask of each of the following types, if possible:`: []mod.Mod{},
	`if diamond flask charges are consumed, (\d+)% increased critical hit chance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChance", "INC", num).Tag(mod.SkillType("Triggered").Neg(true), mod.SkillType("Channel").Neg(true), mod.Condition("UsingDiamondFlask"))}, ""
	},
	`if bismuth flask charges are consumed, penetrate (\d+)% elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalPenetration", "BASE", num).Tag(mod.SkillType("Triggered").Neg(true), mod.SkillType("Channel").Neg(true), mod.Condition("UsingBismuthFlask"))}, ""
	},
	`if amethyst flask charges are consumed, (\d+)% of physical damage as extra chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsChaos", "BASE", num).Tag(mod.SkillType("Triggered").Neg(true), mod.SkillType("Channel").Neg(true), mod.Condition("UsingAmethystFlask"))}, ""
	},
	"double the number of your poisons that targets can be affected by at the same time": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("PoisonCanStack"), MOD("PoisonStacks", "MORE", 100)}, ""
	},
	"your speed is unaffected by slows": []mod.Mod{FLAG("UnaffectedBySlows")},
	`(\d+)% of evasion rating also grants elemental damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EvasionAppliesToFireDamageTaken", "BASE", num), MOD("EvasionAppliesToColdDamageTaken", "BASE", num), MOD("EvasionAppliesToLightningDamageTaken", "BASE", num)}, ""
	},
	"cannot be heavy stunned while sprinting": []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("Sprinting"))},
	`nearby enemies have (\d+)% less accuracy rating while you have phasing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("Accuracy", "MORE", -num)}).Tag(mod.Condition("Phasing"))}, ""
	},
	"immun[ei]t?y? to elemental ailments while phasing": []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("Phasing"))},
	`nearby enemies have fire, cold and lightning exposure while you have phasing, applying \-(\d+)% to those resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", num)}).Tag(mod.Condition("Phasing")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", num)}).Tag(mod.Condition("Phasing")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", num)}).Tag(mod.Condition("Phasing"))}, ""
	},
	"nearby enemies have fire, cold and lightning exposure while you have phasing": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", 20)}).Tag(mod.Condition("Phasing")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", 20)}).Tag(mod.Condition("Phasing")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", 20)}).Tag(mod.Condition("Phasing"))},
	`hits have (\d+)% chance to deal (\d+)% more area damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", ((num * utils.Float(captures[1])) / 100)).Flag(ModFlagArea | ModFlagHit)}, ""
	},
	"immun[ei]t?y? to ignite and shock": []mod.Mod{FLAG("IgniteImmune"), FLAG("ShockImmune")},
	`you gain (\d+)% increased damage for each trap`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Tag(mod.PerStat(1, "ActiveTrapLimit"))}, ""
	},
	`you gain (\d+)% increased area of effect for each mine`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.PerStat(1, "ActiveMineLimit"))}, ""
	},
	`triggers level (\d+) summon triggerbots when allocated`:                         []mod.Mod{FLAG("HaveTriggerBots")},
	"cannot be stunned while leeching":                                               []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("Leeching"))},
	"you are immune to bleeding while leeching":                                      []mod.Mod{FLAG("BleedImmune").Tag(mod.Condition("Leeching"))},
	"life leech effects are not removed at full life":                                []mod.Mod{FLAG("CanLeechLifeOnFullLife")},
	"life leech effects are not removed when unreserved life is filled":              []mod.Mod{FLAG("CanLeechLifeOnFullLife")},
	"energy shield leech effects from attacks are not removed at full energy shield": []mod.Mod{FLAG("CanLeechEnergyShieldOnFullEnergyShield")},
	"cannot take reflected physical damage":                                          []mod.Mod{MOD("PhysicalReflectedDamageTaken", "MORE", -100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	`gain (\d+)% increased movement speed for 20 seconds when you kill an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeed", "INC", num).Tag(mod.Condition("KilledRecently"))}, ""
	},
	`gain (\d+)% increased attack speed for 20 seconds when you kill a rare or unique enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "INC", num).Flag(ModFlagAttack).Tag(mod.Condition("KilledUniqueEnemy"))}, ""
	},
	`kill enemies that have (\d+)% or lower life when hit by your skills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CullPercent", "MAX", num)}, ""
	},
	"you are unaffected by bleeding while leeching": []mod.Mod{MOD("SelfBleedEffect", "MORE", -100).Tag(mod.Condition("Leeching"))},
	`allies in your presence gain added attack damage equal to (\d+)% of your main hand weapon's damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("GainMainHandDmgFromParent"), OnlyAllies: true}), MOD("Multiplier:MainHandDamageToAllies", "BASE", num)}, ""
	},
	"projectile damage builds pin":                                             []mod.Mod{FLAG("CanPin").Flag(ModFlagProjectile)},
	"totems you place grant embankment auras":                                  []mod.Mod{FLAG("Condition:StrategicEmbankments")},
	"grants skill: sorcery ward":                                               []mod.Mod{FLAG("Condition:SorceryWard"), MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: "SorceryWardPlayer", Level: int(1)})},
	"sorcery ward's barrier can also take physical and chaos damage from hits": []mod.Mod{FLAG("Condition:CeremonialAblution")},
	`body armour grants armour also applies to (\w+) damage taken from hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourAppliesTo"+firstToUpper(captures[0])+"DamageTaken", "BASE", 100).Tag(mod.ItemCondition("Body Armour").Rarity("NORMAL"))}, ""
	},
	`body armour grants \+(\d+)% of armour also applies to (\w+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourAppliesTo"+firstToUpper(captures[1])+"DamageTaken", "BASE", num).Tag(mod.ItemCondition("Body Armour").Rarity("NORMAL"))}, ""
	},
	`body armour grants hits against you have (\d+)% reduced critical damage bonus`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.ItemCondition("Body Armour").Rarity("NORMAL"))}, ""
	},
	"body armour grants unaffected by ignite":                      []mod.Mod{MOD("SelfIgniteEffect", "MORE", -100).Tag(mod.ItemCondition("Body Armour").Rarity("NORMAL"))},
	"body armour grants unaffected by damaging ailments":           []mod.Mod{MOD("SelfBleedEffect", "MORE", -100).Tag(mod.ItemCondition("Body Armour").Rarity("NORMAL")), MOD("SelfIgniteEffect", "MORE", -100).Tag(mod.ItemCondition("Body Armour").Rarity("NORMAL")), MOD("SelfPoisonEffect", "MORE", -100).Tag(mod.ItemCondition("Body Armour").Rarity("NORMAL"))},
	"current energy shield also grants elemental damage reduction": []mod.Mod{MOD("EnergyShieldAppliesToColdDamageTaken", "BASE", 1).Tag(mod.Multiplier("CurrentEnergyShield").Limit(150), mod.Condition("UseCurrentEnergyShield")), MOD("EnergyShieldAppliesToFireDamageTaken", "BASE", 1).Tag(mod.Multiplier("CurrentEnergyShield").Limit(150), mod.Condition("UseCurrentEnergyShield")), MOD("EnergyShieldAppliesToLightningDamageTaken", "BASE", 1).Tag(mod.Multiplier("CurrentEnergyShield").Limit(150), mod.Condition("UseCurrentEnergyShield"))},
	`defend against hits as though you had (\d+)% more armour per (\d+)% current energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourDefense", "MAX", num).Tag(mod.Multiplier("CurrentEnergyShield").Div(utils.Float(captures[1])).Limit(150), mod.Condition("UseCurrentEnergyShield"))}, ""
	},
	`(\d+)% increased effect of small passive skills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SmallPassiveSkillEffect", "INC", num)}, ""
	},
	`carry a chest which adds (\d+) inventory slots`:                               []mod.Mod{},
	"ignore warcry cooldowns":                                                      []mod.Mod{MOD("CooldownRecovery", "OVERRIDE", 0).Tag(mod.SkillType("Warcry"))},
	`break armour equal to (\d+)% of hit damage dealt`:                             []mod.Mod{FLAG("Condition:CanArmourBreak").Tag(mod.GlobalEffect("Buff").Name("ArmourBreak"))},
	"you can break enemy armour to below 0":                                        []mod.Mod{FLAG("Condition:CanArmourBreakBelowZero").Tag(mod.GlobalEffect("Buff").Name("ImplodingImpacts"))},
	"fully broken armour you inflict increases all damage taken from hits instead": []mod.Mod{FLAG("FullyBrokenArmourIncreasesAllDamageFromHits").Tag(mod.GlobalEffect("Buff").Name("ImplodingImpacts"))},
	"trigger ancestral spirits when you summon a totem":                            []mod.Mod{},
	`(\d+)% chance to gain (\d+)% of non\-chaos damage with hits as extra chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("NonChaosDamageGainAsChaos", "BASE", ((num / 100) * utils.Float(captures[1])))}, ""
	},
	"movement skills cost no mana": []mod.Mod{MOD("ManaCost", "MORE", -100).KeywordFlag(KeywordFlagMovement)},
	"cannot be stunned while you have ghost shrouds": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("StunImmune").Tag(mod.MultiplierThreshold("GhostShroud").Threshold(1))}, ""
	},
	`your action speed is at least (\d+)% of base value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinimumActionSpeed", "MAX", num)}, ""
	},
	`nearby enemy monsters' action speed is at most (\d+)% of base value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("MaximumActionSpeedReduction", "MAX", (100 - num))})}, ""
	},
	`prevent \+(\d+)% of suppressed spell damage while on full energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellSuppressionEffect", "BASE", num).Tag(mod.Condition("FullEnergyShield"))}, ""
	},
	"energy shield leech effects are not removed when energy shield is filled": []mod.Mod{FLAG("CanLeechEnergyShieldOnFullEnergyShield")},
	`take (\d+)% less damage from hits for (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageTakenWhenHit", "MORE", -num).Tag(mod.Condition("HeartstopperHIT")), MOD("DamageTakenWhenHit", "MORE", ((-num * utils.Float(captures[1])) / 10)).Tag(mod.Condition("HeartstopperAVERAGE"))}, ""
	},
	`take (\d+)% less damage over time for (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageTakenOverTime", "MORE", -num).Tag(mod.Condition("HeartstopperDOT")), MOD("DamageTakenOverTime", "MORE", ((-num * utils.Float(captures[1])) / 10)).Tag(mod.Condition("HeartstopperAVERAGE"))}, ""
	},
	"hits that would ignite instead scorch":              []mod.Mod{FLAG("IgniteCanScorch"), FLAG("CannotIgnite")},
	"you can inflict an additional scorch on each enemy": []mod.Mod{FLAG("ScorchCanStack"), MOD("ScorchStacksMax", "BASE", 1)},
	`maximum effect of shock is (\d+)% increased damage taken`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ShockMax", "OVERRIDE", num)}, ""
	},
	`you can apply up to (\d+) shocks to each enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("ShockCanStack"), MOD("ShockStacksMax", "OVERRIDE", num)}, ""
	},
	"hits that fail to freeze due to insufficient freeze duration inflict hoarfrost": []mod.Mod{FLAG("HitsCanInflictHoarfrost")},
	"your hits always inflict freeze, shock and ignite while unbound":                []mod.Mod{MOD("EnemyFreezeChance", "BASE", 100).Tag(mod.Condition("Unbound")), MOD("EnemyShockChance", "BASE", 100).Tag(mod.Condition("Unbound")), MOD("EnemyIgniteChance", "BASE", 100).Tag(mod.Condition("Unbound"))},
	`(\d+)% more elemental damage while unbound`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalDamage", "MORE", num).Tag(mod.Condition("Unbound"))}, ""
	},
	`spells you cast yourself gain added physical damage equal to (\d+)% of life cost, if life cost is not higher than the maximum you could spend`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalMin", "BASE", 1).Tag(mod.PercentStat("LifeCost", num).Floor(), mod.StatThresholdStat("LifeUnreserved", "LifeCost")), MOD("PhysicalMax", "BASE", 1).Tag(mod.PercentStat("LifeCost", num).Floor(), mod.StatThresholdStat("LifeUnreserved", "LifeCost"))}, ""
	},
	"gain maximum life instead of maximum energy shield from equipped armour items": []mod.Mod{FLAG("ConvertArmourESToLife")},
	"attribute requirements of gems can be satisi?fied by your highest attribute":   []mod.Mod{FLAG("GemAttributeRequirementsSatisfiedByHighestAttribute")},
	"gem quality grants socketed skills an additional effect":                       []mod.Mod{FLAG("GemlingQuality")},
	"you can use two copies of the same support gem in different skills":            []mod.Mod{MOD("MaxSupportGemCopies", "OVERRIDE", 2)},
	"you can use each type of support gem an additional time in different skills":   []mod.Mod{MOD("MaxSupportGemCopies", "OVERRIDE", 2)},
	//pob2:block:src/Modules/ModParser.lua:3388-3396
	`skills have (\d+)% increased critical hit chance per connected blue support gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillCritChanceIncreasedPerBlueSupport", "FLAG", num)}, ""
	},
	`skills deal (\d+)% increased damage per connected red support gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillDamageIncreasedPerRedSupport", "FLAG", num)}, ""
	},
	`skills have (\d+)% increased skill speed per connected green support gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillSpeedIncreasedPerGreenSupport", "FLAG", num)}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:3397-3647
	"for each colour of socketed support gem that is most numerous, gain:": []mod.Mod{},
	"red: hits against you have no critical damage bonus":                  []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.Condition("MostNumerousRedSocketedSupports"))},
	`blue: skills have (\d+)% less cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaCost", "MORE", -num).Tag(mod.Condition("MostNumerousBlueSocketedSupports"))}, ""
	},
	`green: (\d+)% less movement speed penalty from using skills while moving`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeedPenalty", "MORE", -num).Tag(mod.Condition("MostNumerousGreenSocketedSupports"))}, ""
	},
	"targets can be affected by two of your shocks at the same time": []mod.Mod{FLAG("ShockCanStack"), MOD("ShockStacksMax", "OVERRIDE", 2)},
	"targets can be affected by two of your chills at the same time": []mod.Mod{FLAG("ChillCanStack"), MOD("ChillStacksMax", "OVERRIDE", 2)},
	`your chills can slow targets by up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChillMax", "OVERRIDE", num)}, ""
	},
	`\+(\d+) to limit for elemental skills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AdditionalCooldownUses", "BASE", num).Tag(mod.SkillType("Cold", "Fire", "Lightning"))}, ""
	},
	`base maximum darkness is (\d+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("PlayerHasDarkness"), MOD("Darkness", "BASE", num)}, ""
	},
	"removes all spirit": []mod.Mod{MOD("Spirit", "OVERRIDE", 0)},
	`(\d+)% chance to gain (\d+)% of damage with hits as extra (\w+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageGainAs"+firstToUpper(captures[2]), "BASE", (utils.Float(captures[1]) * (num / 100))).Flag(ModFlagHit)}, ""
	},
	"effect and duration of flames of chayula on you is doubled":                        []mod.Mod{FLAG("BreachFlameEffectDoubled")},
	"all flames of chayula that you manifest are red":                                   []mod.Mod{FLAG("BreachFlameOnlyRed")},
	"all flames of chayula that you manifest are blue":                                  []mod.Mod{FLAG("BreachFlameOnlyBlue")},
	"all flames of chayula that you manifest are purple":                                []mod.Mod{FLAG("BreachFlameOnlyPurple")},
	"mana leech recovers based on other damage types damage as well as physical damage": []mod.Mod{FLAG("ManaLeechBasedOnElementalDamage"), FLAG("ManaLeechBasedOnChaosDamage")},
	"grants unravelling": []mod.Mod{FLAG("Unravelling")},
	`critical hits ignore non\-negative enemy monster elemental resistances`:                    []mod.Mod{FLAG("IgnoreNonNegativeEleRes").Tag(mod.Condition("CriticalStrike"))},
	`(\d+)% chance on shocking enemies to created shocked ground`:                               []mod.Mod{MOD("ShockBase", "BASE", dataNonDamagingAilmentDefault["Shock"]).Tag(mod.ActorCondition("enemy", "OnShockedGround"))},
	"on freezing enemies create chilled ground":                                                 []mod.Mod{MOD("ChillBase", "BASE", dataNonDamagingAilmentDefault["Chill"]).Tag(mod.ActorCondition("enemy", "OnChilledGround"))},
	"physical damage reduction from armour is based on your combined armour and evasion rating": []mod.Mod{MOD("EvasionAppliesToPhysicalDamageTaken", "BASE", 100)},
	`skills have (\d+)% chance to not consume a cooldown when used`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownChanceNotConsume", "BASE", (num / 100)).Tag(mod.SkillType("Cooldown"))}, ""
	},
	`every (\d+) seconds, gain (\d+)% more cast speed for (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "MORE", utils.Float(captures[1])).Flag(ModFlagCast).Tag(mod.GlobalEffect("Buff").Name("Quicksand Hourglass"), mod.Condition("QuicksandHourglass")), FLAG("Condition:CanGainQuicksandHourglass")}, ""
	},
	"demonflame has no maximum":                                               []mod.Mod{MOD("Multiplier:DemonFlameMaximum", "BASE", 999)},
	"enemies in your presence have exposure":                                  []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", 20).Tag(mod.ActorCondition("enemy", "EnemyInPresence"))}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", 20).Tag(mod.ActorCondition("enemy", "EnemyInPresence"))}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", 20).Tag(mod.ActorCondition("enemy", "EnemyInPresence"))})},
	"inevitable critical hits":                                                []mod.Mod{FLAG("InevitableCriticalHits")},
	"walk the paths not taken":                                                []mod.Mod{},
	"gain the benefits of bonded modifiers on runes and idols":                []mod.Mod{FLAG("CanUseBonded")},
	"idols socketed in this item gain the benefits of their bonded modifiers": []mod.Mod{FLAG("SocketedIdolsUseBondedModifiers")},
	"has no sockets":                                                          []mod.Mod{FLAG("NoSockets")},
	`has ([\+\-]\d+) to evasion rating per player level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EvasionPerLevel", "BASE", num)}, ""
	},
	`has ([\+\-]\d+) to maximum energy shield per player level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldPerLevel", "BASE", num)}, ""
	},
	`has ([\+\-]\d+) to maximum runic ward per player level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("WardPerLevel", "BASE", num)}, ""
	},
	"reflects your other ring":    []mod.Mod{},
	"reflects opposite ring":      []mod.Mod{},
	`maximum quality is (\d+)%`:   []mod.Mod{},
	`\+(\d+)% to maximum quality`: []mod.Mod{},
	`can have (\d+) additional instilled modifiers?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{}, ""
	},
	"can have an additional instilled modifier": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{}, ""
	},
	"only soul cores can be socketed in this item":                                    []mod.Mod{FLAG("SocketedSoulCoresOnly")},
	"only runes can be socketed in this item":                                         []mod.Mod{FLAG("SocketedRunesOnly")},
	"this item gains bonuses from socketed items as though it was a? ?(.+)":           []mod.Mod{},
	"this item gains bonuses from socketed soul cores as though it was also a? ?(.+)": []mod.Mod{},
	`has (\d+) sockets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketCount", "BASE", num)}, ""
	},
	"no physical damage":             []mod.Mod{MOD("WeaponData", "LIST", mod.WeaponData{Key: "PhysicalMin"}), MOD("WeaponData", "LIST", mod.WeaponData{Key: "PhysicalMax"}), MOD("WeaponData", "LIST", mod.WeaponData{Key: "PhysicalDPS"})},
	"cannot load or fire ammunition": []mod.Mod{MOD("WeaponData", "LIST", LuaTable{"key": "cannotUseGemTag", "value": "ammunition"})},
	`has (\d+)% increased elemental damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LocalElementalDamage", "INC", num)}, ""
	},
	"all attacks with this weapon are critical hits": []mod.Mod{MOD("WeaponData", "LIST", mod.WeaponData{Key: "CritChance", Value: 100})},
	`this weapon's critical hit chance is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("WeaponData", "LIST", mod.WeaponData{Key: "CritChance", Value: num})}, ""
	},
	"counts as dual wielding":                     []mod.Mod{MOD("WeaponData", "LIST", mod.WeaponData{Key: "countsAsDualWielding", Value: 1})},
	"counts as all one handed melee weapon types": []mod.Mod{MOD("WeaponData", "LIST", mod.WeaponData{Key: "countsAsAll1H", Value: 1})},
	"no block chance":                             []mod.Mod{MOD("ArmourData", "LIST", mod.ArmourData{Key: "BlockChance", Value: 0})},
	"no chance to block":                          []mod.Mod{MOD("ArmourData", "LIST", mod.ArmourData{Key: "BlockChance", Value: 0})},
	"has no energy shield":                        []mod.Mod{MOD("ArmourData", "LIST", mod.ArmourData{Key: "EnergyShield", Value: 0})},
	"hits can't be evaded":                        []mod.Mod{FLAG("CannotBeEvaded").Tag(mod.Condition("{Hand}Attack"))},
	"always hits":                                 []mod.Mod{FLAG("CannotBeEvaded").Tag(mod.Condition("{Hand}Attack"))},
	"causes bleeding on hit":                      []mod.Mod{MOD("BleedChance", "BASE", 100).Tag(mod.Condition("{Hand}Attack"))},
	"dazes on hit":                                []mod.Mod{MOD("DazeChance", "BASE", 100).Tag(mod.Condition("{Hand}Attack"))},
	"poisonous hit":                               []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.Condition("{Hand}Attack"))},
	"attacks with this weapon deal double damage": []mod.Mod{MOD("DoubleDamageChance", "BASE", 100).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))},
	`hits with this weapon gain (\d+)% of physical damage as extra cold or lightning damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsColdOrLightning", "BASE", (num/2)).Flag(ModFlagHit).Tag(mod.Condition("DualWielding"), mod.SkillType("Attack")), MOD("PhysicalDamageGainAsColdOrLightning", "BASE", num).Flag(ModFlagHit).Tag(mod.Condition("DualWielding").Neg(true), mod.SkillType("Attack"))}, ""
	},
	`hits with this weapon shock enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ShockAsThoughDealing", "MORE", num).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`hits with this weapon freeze enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FreezeAsThoughDealing", "MORE", num).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`ignites inflicted with this weapon deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagIgnite).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	"hits with this weapon always ignite, freeze, and shock":         []mod.Mod{MOD("EnemyIgniteChance", "BASE", 100).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack")), MOD("EnemyFreezeChance", "BASE", 100).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack")), MOD("EnemyShockChance", "BASE", 100).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))},
	"attacks with this weapon deal double damage to chilled enemies": []mod.Mod{MOD("DoubleDamageChance", "BASE", 100).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"), mod.ActorCondition("enemy", "Chilled"))},
	"life leech from hits with this weapon applies instantly":        []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Tag(mod.Condition("{Hand}Attack"))},
	"life leech from hits with this weapon is instant":               []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Tag(mod.Condition("{Hand}Attack"))},
	"mana leech from hits with this weapon is instant":               []mod.Mod{MOD("InstantManaLeech", "BASE", 100).Tag(mod.Condition("{Hand}Attack"))},
	"life leech from melee damage is instant":                        []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Flag(ModFlagMelee)},
	"gain life from leech instantly from hits with this weapon":      []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))},
	`(\d+)% of leech from hits with this weapon is instant per enemy power`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("InstantLifeLeech", "BASE", num).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"), mod.Multiplier("EnemyPower"))}, ""
	},
	"instant recovery": []mod.Mod{MOD("FlaskInstantRecovery", "BASE", 100)},
	"life flasks used while on low life apply recovery instantly": []mod.Mod{MOD("LifeFlaskInstantRecovery", "BASE", 100).Tag(mod.Condition("LowLife"))},
	"mana flasks used while on low mana apply recovery instantly": []mod.Mod{MOD("ManaFlaskInstantRecovery", "BASE", 100).Tag(mod.Condition("LowMana"))},
	`(\d+)% of recovery applied instantly`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskInstantRecovery", "BASE", num)}, ""
	},
	`(\d+)% of flask recovery applied instantly`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskInstantRecovery", "BASE", num)}, ""
	},
	"has no attribute requirements": []mod.Mod{FLAG("NoAttributeRequirements")},
	`([\+\-]\d+)% to quality of all skills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("grants_active_skill"), Key: "quality", Value: num, KeyOfScaledMod: "value"})}, ""
	},
	`([\+\-]\d+)%? to (\w+) of all ?([\w\-' ]*) skills? ?w?i?t?h? ?a?n? ?(\w+) ?r?e?q?u?i?r?e?m?e?n?t?`: func(num float64, captures []string) ([]mod.Mod, string) {
		property, typ, gemReq := captures[1], strings.ToLower(captures[2]), strings.ToLower(captures[3])
		if typ == "" {
			typ = "all"
		}
		var gemReqType string
		switch gemReq {
		case "intelligence":
			gemReqType = "reqInt"
		case "dexterity":
			gemReqType = "reqDex"
		case "strength":
			gemReqType = "reqStr"
		}
		var gemRequirements map[string]float64
		if gemReqType != "" {
			gemRequirements = map[string]float64{gemReqType: 1}
		}
		ids := gemLookup().skillIDForName
		if ids[typ] != "" || ids["load "+typ] != "" || ids[typ+" minion"] != "" {
			return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr(typ), Key: "level", Value: num, KeyOfScaledMod: "value", GemRequirements: gemRequirements})}, ""
		}
		wordList := gemPropertyWordRe.FindAllString(typ, -1)
		if len(wordList) == 1 {
			return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr(wordList[0]), Key: property, Value: num, KeyOfScaledMod: "value", GemRequirements: gemRequirements})}, ""
		}
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: wordList, Key: property, Value: num, KeyOfScaledMod: "value", GemRequirements: gemRequirements})}, ""
	},
	`grants level (\d+) snipe skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: "Snipe", Level: int(num)}), MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "ChannelledSnipeSupport", Level: int(num)}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`gems socketed in red sockets have [\+\-](\d+) to level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("all"), Key: "level", Value: num, KeyOfScaledMod: "value"}).Tag(mod.SocketedIn("{SlotName}").SocketColor("R"))}, ""
	},
	`gems socketed in green sockets have [\+\-](\d+)% to quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("all"), Key: "quality", Value: num, KeyOfScaledMod: "value"}).Tag(mod.SocketedIn("{SlotName}").SocketColor("G"))}, ""
	},
	`\+(\d+)% to fire resistance when socketed with a red gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketProperty", "LIST", LuaTable{"value": MOD("FireResist", "BASE", num)}).Tag(mod.SocketedIn("{SlotName}").Keyword("strength").Sockets(int(1)))}, ""
	},
	`\+(\d+)% to cold resistance when socketed with a green gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketProperty", "LIST", LuaTable{"value": MOD("ColdResist", "BASE", num)}).Tag(mod.SocketedIn("{SlotName}").Keyword("dexterity").Sockets(int(1)))}, ""
	},
	`\+(\d+)% to lightning resistance when socketed with a blue gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketProperty", "LIST", LuaTable{"value": MOD("LightningResist", "BASE", num)}).Tag(mod.SocketedIn("{SlotName}").Keyword("intelligence").Sockets(int(1)))}, ""
	},
	`attack skills gain (\d+)% of physical damage as extra fire damage per socketed red gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketProperty", "LIST", LuaTable{"value": MOD("PhysicalDamageGainAsFire", "BASE", num).Flag(ModFlagAttack)}).Tag(mod.SocketedIn("{SlotName}").Keyword("strength").Sockets(int(1), int(2), int(3), int(4), int(5), int(6)))}, ""
	},
	"you have vaal pact while all socketed gems are red": []mod.Mod{MOD("GroupProperty", "LIST", LuaTable{"value": MOD("Keystone", "LIST", "Vaal Pact")}).Tag(mod.SocketedIn("{SlotName}").SocketColor("R").SocketsAll())},
	"everlasting sacrifice":                              []mod.Mod{FLAG("Condition:EverlastingSacrifice")},
	`take (\d+) (.+) damage when you ignite an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EyeOfInnocenceSelfDamage", "LIST", LuaTable{"baseDamage": float64(num), "damageType": captures[1]})}, ""
	},
	`when you attack, take (\d+)% of life as (.+) damage for each warcry exerting the attack`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EchoesOfCreationSelfDamage", "LIST", LuaTable{"dmgMult": float64(num), "damageType": captures[1]})}, ""
	},
	`(\d+) (.+) damage taken on minion death`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HeartboundLoopSelfDamage", "LIST", LuaTable{"baseDamage": float64(num), "damageType": captures[1]})}, ""
	},
	`take (\d+) (.+) damage when herald of thunder hits an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("StormSecretSelfDamage", "LIST", LuaTable{"baseDamage": float64(num), "damageType": captures[1]})}, ""
	},
	`your skills deal you (\d+)% of mana cost as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ScoldsBridleSelfDamage", "LIST", LuaTable{"dmgMult": float64(num), "damageType": captures[1]})}, ""
	},
	"grants skill: spear throw": []mod.Mod{},
	`grants skill: (\D+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return grantedExtraSkill(captures[0], int(1), false), ""
	},
	`grants skill: level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return grantedExtraSkill(captures[1], int(num), false), ""
	},
	`[ct][ar][si][tg]g?e?r?s? level (\d+) (.+) when equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`[ct][ar][si][tg]g?e?r?s? level (\d+) (.+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`use level (\d+) (.+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`[ct][ar][si][tg]g?e?r?s? level (\d+) (.+) when you attack`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`[ct][ar][si][tg]g?e?r?s? level (\d+) (.+) when you deal a critical hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`[ct][ar][si][tg]g?e?r?s? level (\d+) (.+) when hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`[ct][ar][si][tg]g?e?r?s? level (\d+) (.+) when you kill an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`[ct][ar][si][tg]g?e?r?s? level (\d+) (.+) when you use a skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`(.+) can trigger level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[2], utils.Int(captures[1]), false, captures[0]), ""
	},
	`trigger level (\d+) (.+) when you use a skill while you have a spirit charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you hit an enemy while cursed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you hit a bleeding enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you hit a rare or unique enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you hit a rare or unique enemy and have no mark`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you hit a frozen enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you kill a frozen enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you consume a corpse`: func(num float64, captures []string) ([]mod.Mod, string) {
		if strings.EqualFold(captures[1], "summon phantasm skill") {
			return triggerExtraSkill("triggered summon phantasm skill", int(num), false, ""), ""
		}
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you attack with a bow`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when animated guardian kills an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you lose cat's stealth`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when your trap is triggered`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) on hit with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) on melee hit while cursed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) on melee hit with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) every [\d\.]+ seconds while phasing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you gain avian's might or avian's flight`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) on melee hit if you have at least (\d+) strength`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`triggers level (\d+) (.+) when equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`triggers level (\d+) (.+) when allocated`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`(\d+)% chance to attack with level (\d+) (.+) on melee hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkill(captures[2], int(utils.Float(captures[1])), false, ""), num), ""
	},
	`(\d+)% chance to trigger level (\d+) (.+) when animated weapon kills an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkill(captures[2], int(utils.Float(captures[1])), false, ""), num), ""
	},
	`(\d+)% chance to trigger level (\d+) (.+) on melee hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkill(captures[2], int(utils.Float(captures[1])), false, ""), num), ""
	},
	`(\d+)% chance to trigger level (\d+) (.+) [ow][nh]e?n? ?y?o?u? kill ?a?n? ?e?n?e?m?y?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkill(captures[2], int(utils.Float(captures[1])), false, ""), num), ""
	},
	`(\d+)% chance to trigger level (\d+) (.+) when you use a socketed skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkill(captures[2], int(utils.Float(captures[1])), false, ""), num), ""
	},
	`(\d+)% chance to trigger level (\d+) (.+) when you gain avian's might or avian's flight`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkill(captures[2], int(utils.Float(captures[1])), false, ""), num), ""
	},
	`(\d+)% chance to trigger level (\d+) (.+) when you or a nearby ally kill an enemy, or hit a rare or unique enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkill(captures[2], int(utils.Float(captures[1])), false, ""), num), ""
	},
	`(\d+)% chance to [ct][ar][si][tg]g?e?r? level (\d+) (.+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkill(captures[2], int(utils.Float(captures[1])), false, ""), num), ""
	},
	`attack with level (\d+) (.+) when you kill a bleeding enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`triggers? level (\d+) (.+) when you kill a bleeding enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`curse enemies with (\D+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	`curse enemies with (\D+) on \w+, with (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: extraSkillIDForName(captures[0]), SkillName: captures[0], Level: 1, NoSupports: true, Triggered: true}),
			MOD("CurseEffect", "INC", utils.Float(captures[1])).Tag(mod.SkillName(utils.CapitalEach(captures[0]))),
		}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:3652-3662
	`\d+% chance to curse n?o?n?-?c?u?r?s?e?d? ?enemies with (\D+) on \w+, with (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: extraSkillIDForName(captures[0]), SkillName: captures[0], Level: 1, NoSupports: true, Triggered: true}),
			MOD("CurseEffect", "INC", utils.Float(captures[1])).Tag(mod.SkillName(utils.CapitalEach(captures[0]))),
		}, ""
	},
	`\d+% chance to curse n?o?n?-?c?u?r?s?e?d? ?enemies with (\D+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: extraSkillIDForName(captures[0]), SkillName: captures[0], Level: 1, NoSupports: true, Triggered: true}),
		}, ""
	},
	`curse enemies with level (\d+) (\D+) on \w+, which can apply to hexproof enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return append(triggerExtraSkill(captures[1], int(num), true, ""),
			MOD("SkillData", "LIST", mod.SkillData{Key: "ignoreHexproof", Value: 1}).Tag(mod.SkillId(extraSkillIDForName(strings.ReplaceAll(strings.ToLower(captures[1]), " skill", ""))))), ""
	},
	`curse enemies with level (\d+) (.+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), true, ""), ""
	},
	`socketed [\w+]* ?gems a?r?e? ?supported by level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		entries := extraSupportIDsForName(captures[1])
		if len(entries) == 0 {
			return nil, ""
		}
		return extraSupportMods(entries, int(num), "{SlotName}", false), ""
	},
	`skills from equipped (.+) are supported by level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		entries := extraSupportIDsForName(captures[2])
		if len(entries) == 0 {
			return nil, ""
		}
		return extraSupportMods(entries, int(utils.Float(captures[1])), extraSupportSlotName(captures[0]), false), ""
	},
	//pob2:block:src/Modules/ModParser.lua:3670-3679
	"socketed hex curse skills are triggered by doedre's effigy when summoned": []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportCursePillarTriggerCurses", Level: int(20)}).Tag(mod.SocketedIn("{SlotName}"))},
	`socketed projectile spells have \+([\d\.]+) seconds to cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "BASE", num).Tag(mod.SocketedIn("{SlotName}"), mod.SkillType("Projectile"), mod.SkillType("Spell"))}, ""
	},
	`trigger level (\d+) (.+) every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+), (.+) or (.+) every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: extraSkillIDForName(captures[1]), Level: int(num), Triggered: true}),
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: extraSkillIDForName(captures[2]), Level: int(num), Triggered: true}),
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: extraSkillIDForName(captures[3]), Level: int(num), Triggered: true}),
		}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:3680-3685
	`inflict corrupted blood for ([\d\.]+) seconds on block, dealing ([\d\.]+)% of your maximum life as physical damage per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: "BloodbarrierPlayer", Level: 1, NoSupports: true, SkillName: "Bloodbarrier"}),
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "unique_blood_barrier_applies_x_stacks_of_corrupted_blood_on_block", Value: float64(1)}).Tag(mod.SkillId("BloodbarrierPlayer")),
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "base_skill_effect_duration", Value: math.Floor(float64(utils.Float(captures[0])*1000) + 0.001)}).Tag(mod.SkillId("BloodbarrierPlayer")),
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "base_physical_damage_%_of_maximum_life_to_deal_per_minute", Value: utils.Float(captures[1])}).Tag(mod.SkillId("BloodbarrierPlayer")),
		}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:3686-3760
	"offering skills triggered this way also affect you": []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", LuaTable{"key": "buffNotPlayer", "value": false})}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering"), mod.SocketedIn("{SlotName}"))},
	`trigger level (\d+) (.+) after spending a total of (\d+) mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`consumes a void charge to trigger level (\d+) (.+) when you fire arrows`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`consumes a void charge to trigger level (\d+) (.+) when you fire arrows with a non\-triggered skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`your hits treat cold resistance as (\d+)% higher than actual value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ColdPenetration", "BASE", -num).KeywordFlag(KeywordFlagHit)}, ""
	},
	`your hits can penetrate elemental resistances down to a minimum of \-(\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalPenetrationMinimum", "BASE", -num).KeywordFlag(KeywordFlagHit)}, ""
	},
	"increases and reductions to minion damage also affects? you": []mod.Mod{FLAG("MinionDamageAppliesToPlayer"), MOD("ImprovedMinionDamageAppliesToPlayer", "MAX", 100)},
	`increases and reductions to minion damage also affects? you at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("MinionDamageAppliesToPlayer"), MOD("ImprovedMinionDamageAppliesToPlayer", "MAX", num)}, ""
	},
	`increases and reductions to minion damage also affect dominating blow and absolution at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("MinionDamageAppliesToPlayer").Tag(mod.SkillName("Dominating Blow", "Absolution").Transfigured(true)), MOD("ImprovedMinionDamageAppliesToPlayer", "MAX", num).Tag(mod.SkillName("Dominating Blow", "Absolution").Transfigured(true))}, ""
	},
	"increases and reductions to companion damage also apply to you":                          []mod.Mod{FLAG("CompanionDamageAppliesToPlayer")},
	"increases and reductions to mana regeneration rate also apply to rage regeneration rate": []mod.Mod{FLAG("ManaRegenAppliesToRageRegen")},
	"increases and reductions to minion attack speed also affects? you":                       []mod.Mod{FLAG("MinionAttackSpeedAppliesToPlayer"), MOD("ImprovedMinionAttackSpeedAppliesToPlayer", "MAX", 100)},
	`increases and reductions to cast speed apply to attack speed at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("CastSpeedAppliesToAttacks"), MOD("ImprovedCastSpeedAppliesToAttacks", "MAX", num)}, ""
	},
	"increases and reductions to cast speed apply to attack speed": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("CastSpeedAppliesToAttacks"), MOD("ImprovedCastSpeedAppliesToAttacks", "MAX", 100)}, ""
	},
	"increases and reductions to spell damage also apply to attacks": []mod.Mod{FLAG("SpellDamageAppliesToAttacks"), MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", 100)},
	`increases and reductions to spell damage also apply to attacks at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("SpellDamageAppliesToAttacks"), MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", num)}, ""
	},
	"increases and reductions to spell damage also apply to attacks while wielding a wand": []mod.Mod{FLAG("SpellDamageAppliesToAttacks").Tag(mod.Condition("UsingWand")), MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", 100).Tag(mod.Condition("UsingWand"))},
	`increases and reductions to maximum mana also apply to shock effect at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("ManaAppliesToShockEffect"), MOD("ImprovedManaAppliesToShockEffect", "MAX", num)}, ""
	},
	"modifiers to claw damage also apply to unarmed":                                                    []mod.Mod{FLAG("ClawDamageAppliesToUnarmed")},
	"modifiers to claw damage also apply to unarmed attack damage":                                      []mod.Mod{FLAG("ClawDamageAppliesToUnarmed")},
	"modifiers to claw damage also apply to unarmed attack damage with melee skills":                    []mod.Mod{FLAG("ClawDamageAppliesToUnarmed")},
	"modifiers to claw attack speed also apply to unarmed":                                              []mod.Mod{FLAG("ClawAttackSpeedAppliesToUnarmed")},
	"modifiers to claw attack speed also apply to unarmed attack speed":                                 []mod.Mod{FLAG("ClawAttackSpeedAppliesToUnarmed")},
	"modifiers to claw attack speed also apply to unarmed attack speed with melee skills":               []mod.Mod{FLAG("ClawAttackSpeedAppliesToUnarmed")},
	"modifiers to claw critical hit chance also apply to unarmed":                                       []mod.Mod{FLAG("ClawCritChanceAppliesToUnarmed")},
	"modifiers to claw critical hit chance also apply to unarmed attack critical hit chance":            []mod.Mod{FLAG("ClawCritChanceAppliesToUnarmed")},
	"modifiers to claw critical hit chance also apply to unarmed critical hit chance with melee skills": []mod.Mod{FLAG("ClawCritChanceAppliesToUnarmed")},
	"increases and reductions to light radius also apply to accuracy":                                   []mod.Mod{FLAG("LightRadiusAppliesToAccuracy")},
	"increases and reductions to light radius also apply to area of effect at 50% of their value":       []mod.Mod{FLAG("LightRadiusAppliesToAreaOfEffect")},
	"increases and reductions to light radius also apply to damage":                                     []mod.Mod{FLAG("LightRadiusAppliesToDamage")},
	"increases and reductions to cast speed also apply to trap throwing speed":                          []mod.Mod{FLAG("CastSpeedAppliesToTrapThrowingSpeed")},
	`increases and reductions to armour also apply to energy shield recharge rate at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("ArmourAppliesToEnergyShieldRecharge"), MOD("ImprovedArmourAppliesToEnergyShieldRecharge", "MAX", num)}, ""
	},
	"increases and reductions to projectile speed also apply to damage with bows":                  []mod.Mod{FLAG("ProjectileSpeedAppliesToBowDamage")},
	"increases and reductions to mana regeneration rate also apply to energy shield recharge rate": []mod.Mod{FLAG("ManaRegenAppliesToEnergyShieldRecharge")},
	`modifiers to maximum (\w+) resistance also [ga][rp][ap][nl][ty] ?t?o? maximum (\w+) and (\w+) resistances?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"MaxResConvertTo"+firstToUpper(captures[1]), "BASE", 100), MOD(firstToUpper(captures[0])+"MaxResConvertTo"+firstToUpper(captures[2]), "BASE", 100)}, ""
	},
	`modifiers to (\w+) resistance also [ga][rp][ap][nl][ty] ?t?o? (\w+) and (\w+) resistances? at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"ResConvertTo"+firstToUpper(captures[1]), "BASE", utils.Float(captures[3])), MOD(firstToUpper(captures[0])+"ResConvertTo"+firstToUpper(captures[2]), "BASE", utils.Float(captures[3]))}, ""
	},
	`gain (\d+)% of bow physical damage as extra damage of each element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsLightning", "BASE", num).Flag(ModFlagBow), MOD("PhysicalDamageGainAsCold", "BASE", num).Flag(ModFlagBow), MOD("PhysicalDamageGainAsFire", "BASE", num).Flag(ModFlagBow)}, ""
	},
	`gain (\d+)% of weapon physical damage as extra damage of each element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsLightning", "BASE", num).Flag(ModFlagWeapon), MOD("PhysicalDamageGainAsCold", "BASE", num).Flag(ModFlagWeapon), MOD("PhysicalDamageGainAsFire", "BASE", num).Flag(ModFlagWeapon)}, ""
	},
	// PoB passes nil before the tags, so they land at [2] and later and ModStore's ipairs never reads them.
	`attacks with this weapon gain (\d+)% of physical damage as extra damage of each element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsLightning", "BASE", num), MOD("PhysicalDamageGainAsCold", "BASE", num), MOD("PhysicalDamageGainAsFire", "BASE", num)}, ""
	},
	`gain (\d+)% of physical damage as extra damage of each element per spirit charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsLightning", "BASE", num).Tag(mod.Multiplier("SpiritCharge")), MOD("PhysicalDamageGainAsCold", "BASE", num).Tag(mod.Multiplier("SpiritCharge")), MOD("PhysicalDamageGainAsFire", "BASE", num).Tag(mod.Multiplier("SpiritCharge"))}, ""
	},
	`gain (\d+)% of damage as extra damage of all elements`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageGainAsLightning", "BASE", num), MOD("DamageGainAsCold", "BASE", num), MOD("DamageGainAsFire", "BASE", num)}, ""
	},
	`attacks with this weapon have added (\w+) damage equal to (\d+)% to (\d+)% of maximum (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"Min", "BASE", 1).Tag(mod.PercentStat("Mana", utils.Float(captures[1])), mod.Condition("{Hand}Attack"), mod.SkillType("Attack")), MOD(firstToUpper(captures[0])+"Max", "BASE", 1).Tag(mod.PercentStat("Mana", utils.Float(captures[2])), mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:3764-3774
	`gain (\d+)% of weapon physical damage as extra damage of an? r?a?n?d?o?m? ?element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", num).Flag(ModFlagWeapon)}, ""
	},
	`gain (\d+)% of physical damage as extra damage of a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", num)}, ""
	},
	`(\d+)% chance for hits to deal (\d+)% of physical damage as extra damage of a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", ((num * utils.Float(captures[1])) / 100))}, ""
	},
	`gain (\d+)% of physical damage as a random element if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", num).Tag(mod.Condition(selfCastCondition(captures[1])))}, ""
	},
	`gain (\d+)% of physical damage as extra damage of a random element while you are ignited`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", num).Tag(mod.Condition("Ignited"))}, ""
	},
	`(\d+)% of physical damage from hits with this weapon is converted to a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageConvertToRandom", "BASE", num)}, ""
	},
	`(\d+)% of physical damage converted to a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageConvertToRandom", "BASE", num)}, ""
	},
	`nearby enemies convert (\d+)% of their (\w+) damage to (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(firstToUpper(captures[1])+"DamageConvertTo"+firstToUpper(captures[2]), "BASE", num)})}, ""
	},
	`enemies ignited by you have (\d+)% of (\w+) damage they deal converted to (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(firstToUpper(captures[1])+"DamageConvertTo"+firstToUpper(captures[2]), "BASE", num).Tag(mod.Condition("Ignited"))})}, ""
	},
	`enemies shocked by you have (\d+)% of (\w+) damage they deal converted to (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(firstToUpper(captures[1])+"DamageConvertTo"+firstToUpper(captures[2]), "BASE", num).Tag(mod.Condition("Shocked"))})}, ""
	},
	"shield crush and spectral shield throw do not gain added physical damage based on armour or evasion on shield": []mod.Mod{FLAG("Condition:ShieldThrowCrushNoArmourEvasion").Tag(mod.SkillName("Spectral Shield Throw", "Shield Crush").Transfigured(true))},
	//pob2:block:src/Modules/ModParser.lua:3791-3793
	`(\d+)% of consecrated path and purifying flame fire damage converted to chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillFireDamageConvertToChaos", "BASE", num).Tag(mod.SkillName("Consecrated Path", "Purifying Flame").Transfigured(true))}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:3797-3903
	"exsanguinate debuffs deal fire damage per second instead of physical damage per second": []mod.Mod{FLAG("Condition:ExsanguinateDebuffIsFireDamage").Tag(mod.SkillName("Exsanguinate").Transfigured(true))},
	"reap debuffs deal fire damage per second instead of physical damage per second":         []mod.Mod{FLAG("Condition:ReapDebuffIsFireDamage").Tag(mod.SkillName("Reap"))},
	`convert (\d+)% of requirements to (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("AttributeRequirementsConverted"), MOD("AttributeRequirementsConvertedTo"+firstToUpper(captures[1]), "BASE", utils.Float(captures[0]))}, ""
	},
	"your critical hit chance is lucky":                                   []mod.Mod{FLAG("CritChanceLucky")},
	"your critical hit chance is lucky while on low life":                 []mod.Mod{FLAG("CritChanceLucky").Tag(mod.Condition("LowLife"))},
	"your critical hit chance is lucky while focus?sed":                   []mod.Mod{FLAG("CritChanceLucky").Tag(mod.Condition("Focused"))},
	"your critical hits do not deal extra damage":                         []mod.Mod{FLAG("NoCritMultiplier")},
	"critical hits do not deal extra damage":                              []mod.Mod{FLAG("NoCritMultiplier")},
	"you have no critical damage bonus":                                   []mod.Mod{FLAG("NoCritMultiplier")},
	"hits with this weapon have no critical damage bonus":                 []mod.Mod{FLAG("NoCritMultiplier").Tag(mod.Condition("{Hand}Attack"))},
	"minion critical hits do not deal extra damage":                       []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("NoCritMultiplier")})},
	`lightning damage with non\-critical hits is lucky`:                   []mod.Mod{FLAG("LightningNoCritLucky")},
	"your damage with critical hits is lucky":                             []mod.Mod{FLAG("CritLucky")},
	"forks critical hits":                                                 []mod.Mod{FLAG("BifurcateCrit")},
	"bifurcates critical hits":                                            []mod.Mod{FLAG("BifurcateCrit")},
	"enemy critical hit chance against you is unlucky":                    []mod.Mod{FLAG("EnemyUnluckyCrit")},
	"critical hits deal no damage":                                        []mod.Mod{MOD("Damage", "MORE", -100).Tag(mod.Condition("CriticalStrike"))},
	"critical hit chance is increased by uncapped lightning resistance":   []mod.Mod{FLAG("CritChanceIncreasedByUncappedLightningRes")},
	"critical hit chance is increased by lightning resistance":            []mod.Mod{FLAG("CritChanceIncreasedByLightningRes")},
	"critical hit chance is increased by overcapped lightning resistance": []mod.Mod{FLAG("CritChanceIncreasedByOvercappedLightningRes")},
	`barrage and frenzy have (\d+)% increased critical hit chance per endurance charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChance", "INC", num).Tag(mod.Multiplier("EnduranceCharge"), mod.SkillName("Barrage", "Frenzy").Transfigured(true))}, ""
	},
	`\+(\d+)% to critical damage bonus per power charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritMultiplier", "INC", num).Tag(mod.Multiplier("PowerCharge"))}, ""
	},
	`non\-critical hits deal (\d+)% damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", (-100 + num)).Flag(ModFlagHit).Tag(mod.Condition("CriticalStrike").Neg(true))}, ""
	},
	`non\-critical hits deal no damage`: []mod.Mod{MOD("Damage", "MORE", -100).Flag(ModFlagHit).Tag(mod.Condition("CriticalStrike").Neg(true))},
	`non\-critical hits deal (\d+)% less damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", -num).Flag(ModFlagHit).Tag(mod.Condition("CriticalStrike").Neg(true))}, ""
	},
	"spell skills always deal critical hits on final repeat":        []mod.Mod{FLAG("SpellSkillsAlwaysDealCriticalStrikesOnFinalRepeat").Flag(ModFlagSpell)},
	"spell skills cannot deal critical hits except on final repeat": []mod.Mod{FLAG("SpellSkillsCannotDealCriticalStrikesExceptOnFinalRepeat").Flag(ModFlagSpell), FLAG("").Tag(mod.Condition("alwaysFinalRepeat"))},
	`critical hits penetrate (\d+)% of enemy elemental resistances while affected by zealotry`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalPenetration", "BASE", num).Tag(mod.Condition("CriticalStrike"), mod.Condition("AffectedByZealotry"))}, ""
	},
	"attack critical hits ignore enemy monster elemental resistances": []mod.Mod{FLAG("IgnoreElementalResistances").Tag(mod.Condition("CriticalStrike"), mod.SkillType("Attack"))},
	"critical hits ignore enemy monster lightning resistance":         []mod.Mod{FLAG("IgnoreLightningResistance").Tag(mod.Condition("CriticalStrike"))},
	"treats enemy monster elemental resistance values as inverted":    []mod.Mod{MOD("HitsInvertEleResChance", "CHANCE", 100).Tag(mod.Condition("{Hand}Attack"))},
	`([\+\-]\d+)% to critical damage bonus if you've shattered an enemy recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritMultiplier", "BASE", num).Tag(mod.Condition("ShatteredEnemyRecently"))}, ""
	},
	`(\d+)% chance to gain a flask charge when you deal a critical hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskChargeOnCritChance", "BASE", num)}, ""
	},
	"gain a flask charge when you deal a critical hit":                                 []mod.Mod{MOD("FlaskChargeOnCritChance", "BASE", 100)},
	"gain a flask charge when you deal a critical hit while affected by precision":     []mod.Mod{MOD("FlaskChargeOnCritChance", "BASE", 100).Tag(mod.Condition("AffectedByPrecision"))},
	"gain a flask charge when you deal a critical hit while at maximum frenzy charges": []mod.Mod{MOD("FlaskChargeOnCritChance", "BASE", 100).Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	"enemies poisoned by you cannot deal critical hits":                                []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("NeverCrit").Tag(mod.Condition("Poisoned"))}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:NeverCrit").Tag(mod.Condition("Poisoned"))})},
	"pinned enemies cannot deal critical hits":                                         []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("NeverCrit").Tag(mod.Condition("Pinned"))}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:NeverCrit").Tag(mod.Condition("Pinned"))})},
	"physical spell critical hits build pin":                                           []mod.Mod{FLAG("CanPin").Flag(ModFlagSpell).KeywordFlag(KeywordFlagPhysical).Tag(mod.Condition("CriticalStrike"))},
	"enemies you mark cannot deal critical hits":                                       []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("NeverCrit").Tag(mod.Condition("Marked"))}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:NeverCrit").Tag(mod.Condition("Marked"))})},
	"hits against you cannot be critical hits if you've been stunned recently":         []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("NeverCrit")}).Tag(mod.Condition("StunnedRecently")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:NeverCrit")}).Tag(mod.Condition("StunnedRecently"))},
	"nearby enemies cannot deal critical hits":                                         []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("NeverCrit")}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:NeverCrit")})},
	"hits against you are always critical hits":                                        []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("AlwaysCrit")}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:AlwaysCrit")})},
	"your hits are always critical hits":                                               []mod.Mod{MOD("CritChance", "OVERRIDE", 100)},
	"always deals critical hits against heavy stunned enemies":                         []mod.Mod{MOD("CritChance", "OVERRIDE", 100).Tag(mod.ActorCondition("enemy", "HeavyStunned"), mod.Condition("{Hand}Attack"))},
	`hits have (\d+)% increased critical hit chance against you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyCritChance", "INC", num)}, ""
	},
	`hits have (\d+)% reduced critical hit chance against you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyCritChance", "INC", -num)}, ""
	},
	`stuns from critical hits have (\d+)% increased duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyStunDurationOnCrit", "INC", num)}, ""
	},
	`break armour on critical hit with spells equal to (\d+)% of physical damage dealt`: []mod.Mod{FLAG("Condition:CanArmourBreak").Tag(mod.GlobalEffect("Buff").Name("ArmourBreak"), mod.Condition("NeverCrit").Neg(true))},
	`enemies take (\d+)% increased damage for each type of ailment you have inflicted on them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Frozen")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Chilled")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Ignited")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Shocked")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Scorched")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Brittle")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Sapped")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Bleeding")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Poisoned")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Electrocuted"))}, ""
	},
	`non\-channelling spells have (\d+)% increased magnitude of ailments per (\d+) maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AilmentMagnitude", "INC", num).KeywordFlag(KeywordFlagSpell).Tag(mod.SkillType("Channel").Neg(true), mod.PerStat(utils.Float(captures[1]), "Life"))}, ""
	},
	`non\-channelling spells have (\d+)% reduced magnitude of ailments per (\d+) maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AilmentMagnitude", "INC", -num).KeywordFlag(KeywordFlagSpell).Tag(mod.SkillType("Channel").Neg(true), mod.PerStat(utils.Float(captures[1]), "Life"))}, ""
	},
	`enemies take (\d+)% increased damage for each elemental ailment type among your ailments on them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Frozen")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Chilled")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Ignited")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Shocked")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Scorched")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Brittle")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Sapped")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Electrocuted"))}, ""
	},
	`(\d+)% increased damage for each type of elemental ailment on enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Tag(mod.ActorCondition("enemy", "Electrocuted")), MOD("Damage", "INC", num).Tag(mod.ActorCondition("enemy", "Frozen")), MOD("Damage", "INC", num).Tag(mod.ActorCondition("enemy", "Chilled")), MOD("Damage", "INC", num).Tag(mod.ActorCondition("enemy", "Ignited")), MOD("Damage", "INC", num).Tag(mod.ActorCondition("enemy", "Shocked"))}, ""
	},
	`(\d+)% increased elemental damage with hits and ailments for each type of elemental ailment on enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalDamage", "INC", num).KeywordFlag(KeywordFlagHit | KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Frozen")), MOD("ElementalDamage", "INC", num).KeywordFlag(KeywordFlagHit | KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Chilled")), MOD("ElementalDamage", "INC", num).KeywordFlag(KeywordFlagHit | KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Ignited")), MOD("ElementalDamage", "INC", num).KeywordFlag(KeywordFlagHit | KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Shocked")), MOD("ElementalDamage", "INC", num).KeywordFlag(KeywordFlagHit | KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Scorched")), MOD("ElementalDamage", "INC", num).KeywordFlag(KeywordFlagHit | KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Brittle")), MOD("ElementalDamage", "INC", num).KeywordFlag(KeywordFlagHit | KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Sapped")), MOD("ElementalDamage", "INC", num).KeywordFlag(KeywordFlagHit | KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "Electrocuted"))}, ""
	},
	`your shocks can increase damage taken by up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ShockMax", "OVERRIDE", num)}, ""
	},
	`\+(\d+)% to maximum effect of shock`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ShockMax", "BASE", num)}, ""
	},
	"your elemental damage can shock":                   []mod.Mod{FLAG("ColdCanShock"), FLAG("FireCanShock")},
	"your fire damage can shock":                        []mod.Mod{FLAG("FireCanShock")},
	"all y?o?u?r? ?damage can freeze":                   []mod.Mod{FLAG("PhysicalCanFreeze"), FLAG("LightningCanFreeze"), FLAG("FireCanFreeze"), FLAG("ChaosCanFreeze")},
	"all damage with maces and sceptres inflicts chill": []mod.Mod{FLAG("PhysicalCanChill").Tag(mod.Condition("UsingMace")), FLAG("LightningCanChill").Tag(mod.Condition("UsingMace")), FLAG("FireCanChill").Tag(mod.Condition("UsingMace")), FLAG("ChaosCanChill").Tag(mod.Condition("UsingMace"))},
	//pob2:block:src/Modules/ModParser.lua:3913-4322
	`(\w+) damage is pinning`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG(firstToUpper(captures[0]) + "CanPin")}, ""
	},
	"your cold damage can ignite":                                                 []mod.Mod{FLAG("ColdCanIgnite")},
	"your lightning damage can ignite":                                            []mod.Mod{FLAG("LightningCanIgnite")},
	"all damage from lightning strike and frost blades hits can ignite":           []mod.Mod{FLAG("PhysicalCanIgnite").Tag(mod.SkillName("Lightning Strike", "Frost Blades").Transfigured(true)), FLAG("ColdCanIgnite").Tag(mod.SkillName("Lightning Strike", "Frost Blades").Transfigured(true)), FLAG("LightningCanIgnite").Tag(mod.SkillName("Lightning Strike", "Frost Blades").Transfigured(true)), FLAG("ChaosCanIgnite").Tag(mod.SkillName("Lightning Strike", "Frost Blades").Transfigured(true))},
	"all damage from lightning arrow and ice shot hits can ignite":                []mod.Mod{FLAG("PhysicalCanIgnite").Tag(mod.SkillName("Lightning Arrow", "Ice Shot").Transfigured(true)), FLAG("ColdCanIgnite").Tag(mod.SkillName("Lightning Arrow", "Ice Shot").Transfigured(true)), FLAG("LightningCanIgnite").Tag(mod.SkillName("Lightning Arrow", "Ice Shot").Transfigured(true)), FLAG("ChaosCanIgnite").Tag(mod.SkillName("Lightning Arrow", "Ice Shot").Transfigured(true))},
	"all damage from shock nova and storm call hits can ignite":                   []mod.Mod{FLAG("PhysicalCanIgnite").Tag(mod.SkillName("Shock Nova", "Storm Call").Transfigured(true)), FLAG("ColdCanIgnite").Tag(mod.SkillName("Shock Nova", "Storm Call").Transfigured(true)), FLAG("LightningCanIgnite").Tag(mod.SkillName("Shock Nova", "Storm Call").Transfigured(true)), FLAG("ChaosCanIgnite").Tag(mod.SkillName("Shock Nova", "Storm Call").Transfigured(true))},
	"your fire damage can shock but not ignite":                                   []mod.Mod{FLAG("FireCanShock"), FLAG("FireCannotIgnite")},
	"your cold damage can ignite but not freeze or chill":                         []mod.Mod{FLAG("ColdCanIgnite"), FLAG("ColdCannotFreeze"), FLAG("ColdCannotChill")},
	"your cold damage cannot freeze":                                              []mod.Mod{FLAG("ColdCannotFreeze")},
	"your cold damage cannot chill":                                               []mod.Mod{FLAG("ColdCannotChill")},
	"your lightning damage can freeze but not shock":                              []mod.Mod{FLAG("LightningCanFreeze"), FLAG("LightningCannotShock")},
	"your chaos damage can shock":                                                 []mod.Mod{FLAG("ChaosCanShock")},
	"chaos damage from hits also contributes to shock chance":                     []mod.Mod{FLAG("ChaosCanShock")},
	"your chaos damage can chill":                                                 []mod.Mod{FLAG("ChaosCanChill")},
	"your chaos damage can ignite":                                                []mod.Mod{FLAG("ChaosCanIgnite")},
	"chaos damage can ignite, chill and shock":                                    []mod.Mod{FLAG("ChaosCanIgnite"), FLAG("ChaosCanChill"), FLAG("ChaosCanShock")},
	"your physical damage can chill":                                              []mod.Mod{FLAG("PhysicalCanChill")},
	"physical damage from hits contributes to chill magnitude and freeze buildup": []mod.Mod{FLAG("PhysicalCanChill"), FLAG("PhysicalCanFreeze")},
	"your physical damage can shock":                                              []mod.Mod{FLAG("PhysicalCanShock")},
	"your physical damage can freeze":                                             []mod.Mod{FLAG("PhysicalCanFreeze")},
	"your lightning damage can freeze":                                            []mod.Mod{FLAG("LightningCanFreeze")},
	"you always ignite while burning":                                             []mod.Mod{MOD("EnemyIgniteChance", "BASE", 100).Tag(mod.Condition("Burning"))},
	"critical hits do not a?l?w?a?y?s?i?n?h?e?r?e?n?t?l?y? freeze":                []mod.Mod{FLAG("CritsDontAlwaysFreeze")},
	"cannot inflict elemental ailments":                                           []mod.Mod{FLAG("CannotIgnite"), FLAG("CannotChill"), FLAG("CannotFreeze"), FLAG("CannotShock"), FLAG("CannotScorch"), FLAG("CannotBrittle"), FLAG("CannotSap")},
	`non\-critical hits cannot inflict ailments`:                                  []mod.Mod{FLAG("AilmentsOnlyFromCrit")},
	"flameblast and incinerate cannot inflict elemental ailments":                 []mod.Mod{FLAG("CannotIgnite").Tag(mod.SkillName("Flameblast", "Incinerate").Transfigured(true)), FLAG("CannotChill").Tag(mod.SkillName("Flameblast", "Incinerate").Transfigured(true)), FLAG("CannotFreeze").Tag(mod.SkillName("Flameblast", "Incinerate").Transfigured(true)), FLAG("CannotShock").Tag(mod.SkillName("Flameblast", "Incinerate").Transfigured(true)), FLAG("CannotScorch").Tag(mod.SkillName("Flameblast", "Incinerate").Transfigured(true)), FLAG("CannotBrittle").Tag(mod.SkillName("Flameblast", "Incinerate").Transfigured(true)), FLAG("CannotSap").Tag(mod.SkillName("Flameblast", "Incinerate").Transfigured(true))},
	`you can inflict up to (\d+) ignites on an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("IgniteCanStack"), MOD("IgniteStacks", "OVERRIDE", num)}, ""
	},
	"you can inflict an additional ignite on [ea][an]c?h? enemy": []mod.Mod{FLAG("IgniteCanStack"), MOD("IgniteStacks", "BASE", 1)},
	`targets can be affected by \+(\d+) of your poisons at the same time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("PoisonCanStack"), MOD("PoisonStacks", "BASE", num)}, ""
	},
	`enemies chilled by you take (\d+)% increased burning damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireDamageTakenOverTime", "INC", num)}).Tag(mod.ActorCondition("enemy", "Chilled"))}, ""
	},
	`damaging ailments deal damage (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("IgniteFaster", "INC", num), MOD("BleedFaster", "INC", num), MOD("PoisonFaster", "INC", num)}, ""
	},
	`damaging ailments you inflict deal damage (\d+)% faster while affected by malevolence`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("IgniteFaster", "INC", num).Tag(mod.Condition("AffectedByMalevolence")), MOD("BleedFaster", "INC", num).Tag(mod.Condition("AffectedByMalevolence")), MOD("PoisonFaster", "INC", num).Tag(mod.Condition("AffectedByMalevolence"))}, ""
	},
	`ignited enemies burn (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("IgniteFaster", "INC", num)}, ""
	},
	`ignited enemies burn (\d+)% slower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("IgniteSlower", "INC", num)}, ""
	},
	`enemies ignited by an attack burn (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("IgniteFaster", "INC", num).Flag(ModFlagAttack)}, ""
	},
	`ignites you inflict with attacks deal damage (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("IgniteFaster", "INC", num).Flag(ModFlagAttack)}, ""
	},
	`ignites you inflict deal damage (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("IgniteFaster", "INC", num)}, ""
	},
	`(\d+)% chance for ignites inflicted with lightning strike or frost blades to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", ((utils.Float(captures[1]) * num) / 100)).KeywordFlag(KeywordFlagIgnite).Tag(mod.SkillName("Lightning Strike", "Frost Blades").Transfigured(true))}, ""
	},
	`(\d+)% chance for ignites inflicted with lightning arrow or ice shot to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", ((utils.Float(captures[1]) * num) / 100)).KeywordFlag(KeywordFlagIgnite).Tag(mod.SkillName("Lightning Arrow", "Ice Shot").Transfigured(true))}, ""
	},
	`(\d+)% chance for ignites inflicted with shock nova or storm call to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", ((utils.Float(captures[1]) * num) / 100)).KeywordFlag(KeywordFlagIgnite).Tag(mod.SkillName("Shock Nova", "Storm Call").Transfigured(true))}, ""
	},
	`enemies ignited by you during f?l?a?s?k? ?effect take (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Ignited"))}, ""
	},
	"enemies ignited by you take chaos damage instead of fire damage from ignite": []mod.Mod{FLAG("IgniteToChaos")},
	"ignite you inflict deals chaos damage instead of fire damage":                []mod.Mod{FLAG("IgniteToChaos")},
	"enemies chilled by your hits are shocked":                                    []mod.Mod{MOD("ShockBase", "BASE", dataNonDamagingAilmentDefault["Shock"]).Tag(mod.ActorCondition("enemy", "ChilledByYourHits")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Shocked").Tag(mod.Condition("ChilledByYourHits"))})},
	"cannot inflict ignite":                 []mod.Mod{FLAG("CannotIgnite")},
	"cannot inflict freeze or chill":        []mod.Mod{FLAG("CannotFreeze"), FLAG("CannotChill")},
	"cannot inflict shock":                  []mod.Mod{FLAG("CannotShock")},
	"cannot ignite, chill, freeze or shock": []mod.Mod{FLAG("CannotIgnite"), FLAG("CannotChill"), FLAG("CannotFreeze"), FLAG("CannotShock")},
	`shock enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ShockAsThoughDealing", "MORE", num)}, ""
	},
	`chill enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChillAsThoughDealing", "MORE", num)}, ""
	},
	`inflict non\-damaging ailments as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ShockAsThoughDealing", "MORE", num), MOD("ChillAsThoughDealing", "MORE", num), MOD("FreezeAsThoughDealing", "MORE", num), MOD("ScorchAsThoughDealing", "MORE", num), MOD("BrittleAsThoughDealing", "MORE", num), MOD("SapAsThoughDealing", "MORE", num)}, ""
	},
	`non\-damaging elemental ailments you inflict have (\d+)% more effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyShockMagnitude", "MORE", num), MOD("EnemyChillMagnitude", "MORE", num), MOD("EnemyFreezeBuildup", "MORE", num)}, ""
	},
	`modifiers to stun buildup apply to freeze buildup instead for (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("FreezeBuildupInsteadOfStunBuildup").Tag(mod.SkillName(firstToUpper(captures[0])).Transfigured(true)), FLAG("CannotStun").Tag(mod.SkillName(firstToUpper(captures[0])).Transfigured(true)), FLAG("CannotHeavyStun").Tag(mod.SkillName(firstToUpper(captures[0])).Transfigured(true))}, ""
	},
	`immun[ei]t?y? to elemental ailments while on consecrated ground if you have at least (\d+) devotion`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("OnConsecratedGround"), mod.StatThreshold("Devotion", num))}, ""
	},
	`freeze enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FreezeAsThoughDealing", "MORE", num)}, ""
	},
	`freeze chilled enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FreezeAsThoughDealing", "MORE", num).Tag(mod.ActorCondition("enemy", "Chilled"))}, ""
	},
	`manabond and stormbind freeze enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FreezeAsThoughDealing", "MORE", num).Tag(mod.SkillName("Manabond", "Stormbind").Transfigured(true))}, ""
	},
	`(\d+)% chance to inflict brittle on enemies when you block their damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyBrittleChance", "BASE", num)}, ""
	},
	`(\d+)% chance to inflict sap on enemies when you block their damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemySapChance", "BASE", num)}, ""
	},
	`(\d+)% chance to inflict scorch on enemies when you block their damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyScorchChance", "BASE", num)}, ""
	},
	"scorch enemies in close range when you block":                                                              []mod.Mod{MOD("EnemyScorchChance", "BASE", 100)},
	`(\d+)% chance to shock attackers for (\d+) seconds on block`:                                               []mod.Mod{MOD("ShockBase", "BASE", dataNonDamagingAilmentDefault["Shock"])},
	`shock attackers for (\d+) seconds on block`:                                                                []mod.Mod{MOD("ShockBase", "BASE", dataNonDamagingAilmentDefault["Shock"]).Tag(mod.Condition("BlockedRecently")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Shocked")}).Tag(mod.Condition("BlockedRecently"))},
	`shock nearby enemies for (\d+) seconds when you focus`:                                                     []mod.Mod{MOD("ShockBase", "BASE", dataNonDamagingAilmentDefault["Shock"]).Tag(mod.Condition("Focused")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Shocked")}).Tag(mod.Condition("Focused"))},
	`shock yourself for (\d+) seconds when you focus`:                                                           []mod.Mod{MOD("ShockBase", "BASE", dataNonDamagingAilmentDefault["Shock"]).Tag(mod.Condition("Focused")), FLAG("Condition:Shocked").Tag(mod.Condition("Focused"))},
	`drops shocked ground while moving, lasting (\d+) seconds`:                                                  []mod.Mod{MOD("ShockBase", "BASE", dataNonDamagingAilmentDefault["Shock"]).Tag(mod.ActorCondition("enemy", "OnShockedGround"))},
	`while a unique enemy is in your presence, drops shocked ground while moving, lasting (\d+) seconds`:        []mod.Mod{MOD("ShockBase", "BASE", dataNonDamagingAilmentDefault["Shock"]).Tag(mod.ActorCondition("enemy", "OnShockedGround"), mod.ActorCondition("enemy", "RareOrUnique"))},
	`while a pinnacle atlas boss is in your presence, drops shocked ground while moving, lasting (\d+) seconds`: []mod.Mod{MOD("ShockBase", "BASE", dataNonDamagingAilmentDefault["Shock"]).Tag(mod.ActorCondition("enemy", "OnShockedGround"), mod.ActorCondition("enemy", "PinnacleBoss"))},
	`\+(\d+)% chance to ignite, freeze, shock, and poison cursed enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyIgniteChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed")), MOD("EnemyFreezeChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed")), MOD("EnemyShockChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed")), MOD("PoisonChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	"you have scorching conflux, brittle conflux and sapping conflux while your two highest attributes are equal": []mod.Mod{MOD("EnemyScorchChance", "BASE", 100).Tag(mod.Condition("TwoHighestAttributesEqual")), MOD("EnemyBrittleChance", "BASE", 100).Tag(mod.Condition("TwoHighestAttributesEqual")), MOD("EnemySapChance", "BASE", 100).Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("PhysicalCanScorch").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("LightningCanScorch").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("ColdCanScorch").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("ChaosCanScorch").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("PhysicalCanBrittle").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("LightningCanBrittle").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("FireCanBrittle").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("ChaosCanBrittle").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("PhysicalCanSap").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("ColdCanSap").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("FireCanSap").Tag(mod.Condition("TwoHighestAttributesEqual")), FLAG("ChaosCanSap").Tag(mod.Condition("TwoHighestAttributesEqual"))},
	"all damage from cold snap and creeping frost can sap":                                                        []mod.Mod{FLAG("PhysicalCanSap").Tag(mod.SkillName("Cold Snap", "Creeping Frost").Transfigured(true)), FLAG("ColdCanSap").Tag(mod.SkillName("Cold Snap", "Creeping Frost").Transfigured(true)), FLAG("FireCanSap").Tag(mod.SkillName("Cold Snap", "Creeping Frost").Transfigured(true)), FLAG("ChaosCanSap").Tag(mod.SkillName("Cold Snap", "Creeping Frost").Transfigured(true))},
	"always inflict scorch, brittle and sapped with elemental hit and wild strike hits":                           []mod.Mod{MOD("EnemyScorchChance", "BASE", 100).Tag(mod.SkillName("Elemental Hit", "Wild Strike").Transfigured(true)), MOD("EnemyBrittleChance", "BASE", 100).Tag(mod.SkillName("Elemental Hit", "Wild Strike").Transfigured(true)), MOD("EnemySapChance", "BASE", 100).Tag(mod.SkillName("Elemental Hit", "Wild Strike").Transfigured(true))},
	`critical hits do not inherently apply non\-damaging ailments`:                                                []mod.Mod{FLAG("CritsDontAlwaysChill"), FLAG("CritsDontAlwaysFreeze"), FLAG("CritsDontAlwaysShock")},
	"critical hits do not inherently ignite":                                                                      []mod.Mod{FLAG("CritsDontAlwaysIgnite")},
	"always scorch while affected by anger":                                                                       []mod.Mod{MOD("EnemyScorchChance", "BASE", 100).Tag(mod.Condition("AffectedByAnger"))},
	"always inflict brittle while affected by hatred":                                                             []mod.Mod{MOD("EnemyBrittleChance", "BASE", 100).Tag(mod.Condition("AffectedByHatred"))},
	"always sap while affected by wrath":                                                                          []mod.Mod{MOD("EnemySapChance", "BASE", 100).Tag(mod.Condition("AffectedByWrath"))},
	`(\d+)% chance to sap enemies in chilling areas`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemySapChance", "BASE", num).Tag(mod.ActorCondition("enemy", "InChillingArea"))}, ""
	},
	`(\d+)% chance for cold snap and creeping frost to sap enemies in chilling areas`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemySapChance", "BASE", num).Tag(mod.SkillName("Cold Snap", "Creeping Frost").Transfigured(true), mod.ActorCondition("enemy", "InChillingArea"))}, ""
	},
	`drops burning ground while moving, dealing (\d+) fire damage per second for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DropsBurningGround", "BASE", num)}, ""
	},
	`take (\d+) fire damage per second while flame\-touched`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireDegen", "BASE", num).Tag(mod.Condition("AffectedByApproachingFlames"))}, ""
	},
	`gain adrenaline when you become flame\-touched`:                             []mod.Mod{FLAG("Condition:Adrenaline").Tag(mod.Condition("AffectedByApproachingFlames"))},
	`lose adrenaline when you cease to be flame\-touched`:                        []mod.Mod{},
	"modifiers to ignite duration on you apply to all elemental ailments":        []mod.Mod{FLAG("IgniteDurationAppliesToElementalAilments")},
	"chance to avoid being shocked applies to all elemental ailments":            []mod.Mod{FLAG("ShockAvoidAppliesToElementalAilments")},
	"modifiers to chance to avoid being shocked apply to all elemental ailments": []mod.Mod{FLAG("ShockAvoidAppliesToElementalAilments")},
	`enemies permanently take (\d+)% increased damage for each second they've ever been frozen by you, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Condition("FrozenByYou"), mod.Multiplier("FrozenByYouSeconds").Limit((utils.Float(captures[1]) / num)))})}, ""
	},
	`enemies permanently take (\d+)% increased damage for each second they've ever been chilled by you, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Condition("ChilledByYou"), mod.Multiplier("ChilledByYouSeconds").Limit((utils.Float(captures[1]) / num)))})}, ""
	},
	`modifiers to chance to suppress spell damage also apply to chance to avoid elemental ailments at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellSuppressionAppliesToAilmentAvoidancePercent", "BASE", num), FLAG("SpellSuppressionAppliesToAilmentAvoidance")}, ""
	},
	"enemies chilled by your hits have damage taken increased by chill effect":                                               []mod.Mod{FLAG("ChillEffectIncDamageTaken")},
	"enemies chilled by your hits increase damage taken by chill magnitude":                                                  []mod.Mod{FLAG("ChillEffectIncDamageTaken")},
	"elemental damage from hits contributes to flammability, ignite, and chill magnitudes, freeze buildup, and shock chance": []mod.Mod{FLAG("FireCanChill"), FLAG("FireCanFreeze"), FLAG("FireCanShock"), FLAG("ColdCanIgnite"), FLAG("ColdCanShock"), FLAG("LightningCanIgnite"), FLAG("LightningCanChill"), FLAG("LightningCanFreeze")},
	"melee attacks cause bleeding":                       []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(ModFlagMelee)},
	"attacks cause bleeding when hitting cursed enemies": []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(ModFlagAttack).Tag(mod.ActorCondition("enemy", "Cursed"))},
	"melee critical hits cause bleeding":                 []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(ModFlagMelee).Tag(mod.Condition("CriticalStrike"))},
	"causes bleeding on melee critical hit":              []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(ModFlagMelee).Tag(mod.Condition("CriticalStrike"))},
	`melee critical hits have (\d+)% chance to cause bleeding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BleedChance", "BASE", num).Flag(ModFlagMelee).Tag(mod.Condition("CriticalStrike"))}, ""
	},
	"attacks always inflict bleeding while you have cat's stealth":      []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(ModFlagAttack).Tag(mod.Condition("AffectedByCat'sStealth"))},
	"you have crimson dance while you have cat's stealth":               []mod.Mod{MOD("Keystone", "LIST", "Crimson Dance").Tag(mod.Condition("AffectedByCat'sStealth"))},
	"you have crimson dance if you have dealt a critical hit recently":  []mod.Mod{MOD("Keystone", "LIST", "Crimson Dance").Tag(mod.Condition("CritRecently"))},
	"bleeding you inflict deals fire damage instead of physical damage": []mod.Mod{FLAG("BleedToFire"), MOD("SkillData", "LIST", LuaTable{"key": "BleedToFire", "value": true})},
	`bleeding you inflict deals damage (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BleedFaster", "INC", num)}, ""
	},
	`bleeding you inflict on non\-bleeding enemies deals (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagBleed).Tag(mod.Condition("SingleBleed"))}, ""
	},
	`(\d+)% chance for bleeding inflicted with this weapon to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", ((utils.Float(captures[1])*num)/100)).KeywordFlag(KeywordFlagBleed).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`(\d+)% chance for bleeding inflicted with cobra lash or venom gyre to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", ((utils.Float(captures[1]) * num) / 100)).KeywordFlag(KeywordFlagBleed).Tag(mod.SkillName("Cobra Lash", "Venom Gyre"))}, ""
	},
	`bleeding you inflict deals damage (\d+)% faster per frenzy charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BleedFaster", "INC", num).Tag(mod.Multiplier("FrenzyCharge"))}, ""
	},
	`rain of arrows and toxic rain deal (\d+)% more damage with bleeding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagBleed).Tag(mod.SkillName("Rain of Arrows", "Toxic Rain").Transfigured(true))}, ""
	},
	"elemental damage also contributes to bleeding magnitude": []mod.Mod{FLAG("FireCanBleed"), FLAG("ColdCanBleed"), FLAG("LightningCanBleed")},
	`(\w+) damage also contributes to bleeding magnitude`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG(firstToUpper(captures[0]) + "CanBleed")}, ""
	},
	`(\d+)% increased effect of impales inflicted by hits that also inflict bleeding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ImpaleEffectOnBleed", "INC", num).KeywordFlag(KeywordFlagHit)}, ""
	},
	`(\d+)% chance for blade vortex and blade blast to impale enemies on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ImpaleChance", "BASE", num).Tag(mod.SkillName("Blade Vortex", "Blade Blast").Transfigured(true))}, ""
	},
	"critical hits with spells inflict impale": []mod.Mod{MOD("ImpaleChance", "BASE", 100).Flag(ModFlagSpell).Tag(mod.Condition("CriticalStrike"))},
	`(\d+)% chance on hitting an enemy for all impales on that enemy to last for an additional hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ImpaleAdditionalDurationChance", "BASE", num)}, ""
	},
	`(\d+)% increased damage with bleeding inflicted on poisoned enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(KeywordFlagBleed).Tag(mod.ActorCondition("enemy", "Poisoned"))}, ""
	},
	"y?o?u?r? ?fire damage can poison":                                                                  []mod.Mod{FLAG("FireCanPoison")},
	"y?o?u?r? ?cold damage can poison":                                                                  []mod.Mod{FLAG("ColdCanPoison")},
	"y?o?u?r? ?lightning damage can poison":                                                             []mod.Mod{FLAG("LightningCanPoison")},
	"all damage from hits with this weapon can poison":                                                  []mod.Mod{FLAG("FireCanPoison").Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack")), FLAG("ColdCanPoison").Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack")), FLAG("LightningCanPoison").Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))},
	"all damage inflicts poison while affected by glorious madness":                                     []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("FireCanPoison").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("ColdCanPoison").Tag(mod.Condition("AffectedByGloriousMadness")), FLAG("LightningCanPoison").Tag(mod.Condition("AffectedByGloriousMadness"))},
	"all damage from blast rain and artillery ballista hits can poison":                                 []mod.Mod{FLAG("FireCanPoison").Tag(mod.SkillName("Blast Rain", "Artillery Ballista")), FLAG("ColdCanPoison").Tag(mod.SkillName("Blast Rain", "Artillery Ballista")), FLAG("LightningCanPoison").Tag(mod.SkillName("Blast Rain", "Artillery Ballista"))},
	"all damage from hits with freezing pulse and eye of winter can poison":                             []mod.Mod{FLAG("FireCanPoison").Tag(mod.SkillName("Freezing Pulse", "Eye of Winter").Transfigured(true)), FLAG("ColdCanPoison").Tag(mod.SkillName("Freezing Pulse", "Eye of Winter").Transfigured(true)), FLAG("LightningCanPoison").Tag(mod.SkillName("Freezing Pulse", "Eye of Winter").Transfigured(true))},
	"all damage from you and allies in your presence contributes to ignite chance and magnitude":        []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("ColdCanIgnite")}), MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("LightningCanIgnite")}), MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("ChaosCanIgnite")}), MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("PhysicalCanIgnite")})},
	"all damage from you and allies in your presence contributes to flammability and ignite magnitudes": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("ColdCanIgnite")}), MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("LightningCanIgnite")}), MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("ChaosCanIgnite")}), MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("PhysicalCanIgnite")})},
	"your chaos damage poisons enemies":                                                                 []mod.Mod{MOD("ChaosPoisonChance", "BASE", 100)},
	`your chaos damage has (\d+)% chance to poison enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosPoisonChance", "BASE", num)}, ""
	},
	"melee attacks poison on hit": []mod.Mod{MOD("PoisonChance", "BASE", 100).Flag(ModFlagMelee)},
	`melee critical hits have (\d+)% chance to poison the enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PoisonChance", "BASE", num).Flag(ModFlagMelee).Tag(mod.Condition("CriticalStrike"))}, ""
	},
	`critical hits with daggers have a (\d+)% chance to poison the enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PoisonChance", "BASE", num).Flag(ModFlagDagger).Tag(mod.Condition("CriticalStrike"))}, ""
	},
	"critical hits with daggers poison the enemy":                    []mod.Mod{MOD("PoisonChance", "BASE", 100).Flag(ModFlagDagger).Tag(mod.Condition("CriticalStrike"))},
	"poison cursed enemies on hit":                                   []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.ActorCondition("enemy", "Cursed"))},
	"always poison on hit against cursed enemies":                    []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.ActorCondition("enemy", "Cursed"))},
	"wh[ie][ln]e? at maximum frenzy charges, attacks poison enemies": []mod.Mod{MOD("PoisonChance", "BASE", 100).Flag(ModFlagAttack).Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	`traps and mines have a (\d+)% chance to poison on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PoisonChance", "BASE", num).KeywordFlag(KeywordFlagTrap | KeywordFlagMine)}, ""
	},
	`poisons you inflict deal damage (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PoisonFaster", "INC", num)}, ""
	},
	`(\d+)% chance for poisons inflicted with this weapon to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", ((utils.Float(captures[1])*num)/100)).KeywordFlag(KeywordFlagPoison).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`(\d+)% chance for poisons inflicted with blast rain or artillery balls?i?s?ta to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", ((utils.Float(captures[1]) * num) / 100)).KeywordFlag(KeywordFlagPoison).Tag(mod.SkillName("Blast Rain", "Artillery Ballista"))}, ""
	},
	`(\d+)% chance for poisons inflicted with freezing pulse and eye of winter to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", ((utils.Float(captures[1]) * num) / 100)).KeywordFlag(KeywordFlagPoison).Tag(mod.SkillName("Freezing Pulse", "Eye of Winter").Transfigured(true))}, ""
	},
	`poisons you inflict on non\-poisoned enemies deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(KeywordFlagPoison).Tag(mod.Condition("SinglePoison"))}, ""
	},
	`poisons inflicted by sunder or ground slam on non\-poisoned enemies deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(KeywordFlagPoison).Tag(mod.Condition("SinglePoison"), mod.SkillName("Sunder", "Ground Slam").Transfigured(true))}, ""
	},
	`poisons on you expire (\d+)% slower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SelfPoisonDebuffExpirationRate", "BASE", -num)}, ""
	},
	"any number of poisons from this weapon can affect a target at the same time": []mod.Mod{FLAG("PoisonCanStack"), MOD("PoisonStacks", "OVERRIDE", math.Inf(1)).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("NonWeaponAttack").Neg(true))},
	"critical hits poison the enemy":                                              []mod.Mod{MOD("PoisonChance", "OVERRIDE", 100).Tag(mod.Condition("CriticalStrike"))},
	"always poison on hit with this weapon":                                       []mod.Mod{MOD("PoisonChance", "OVERRIDE", 100).Flag(ModFlagWeapon).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("NonWeaponAttack").Neg(true))},
	`gain deflection rating equal to (\d+)% of evasion rating`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EvasionGainAsDeflection", "BASE", num)}, ""
	},
	`gain deflection rating equal to (\d+)% of armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourGainAsDeflection", "BASE", num)}, ""
	},
	`prevent \+(\d+)% of damage from deflected hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DeflectEffect", "BASE", num)}, ""
	},
	"chance to deflect is lucky":                           []mod.Mod{FLAG("DeflectIsLucky")},
	"chance to deflect is lucky while on low life":         []mod.Mod{FLAG("DeflectIsLucky").Tag(mod.Condition("LowLife"))},
	"y?o?u?r? ?chance to suppress spell damage is lucky":   []mod.Mod{FLAG("SpellSuppressionChanceIsLucky")},
	"y?o?u?r? ?chance to suppress spell damage is unlucky": []mod.Mod{FLAG("SpellSuppressionChanceIsUnlucky")},
	`prevent \+(\d+)% of suppressed spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellSuppressionEffect", "BASE", num)}, ""
	},
	`prevent \+(\d+)% of suppressed spell damage per hit suppressed recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellSuppressionEffect", "BASE", num).Tag(mod.Multiplier("HitsSuppressedRecently"))}, ""
	},
	"inflict fire, cold and lightning exposure on enemies when you suppress their spell damage": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", 20)}).Tag(mod.Condition("Effective"), mod.Condition("SuppressedRecently")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", 20)}).Tag(mod.Condition("Effective"), mod.Condition("SuppressedRecently")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", 20)}).Tag(mod.Condition("Effective"), mod.Condition("SuppressedRecently"))},
	"critical hit chance is increased by chance to suppress spell damage":                       []mod.Mod{FLAG("CritChanceIncreasedBySpellSuppressChance")},
	`you take (\d+)% reduced extra damage from suppressed critical hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceSuppressedCritExtraDamage", "BASE", num)}, ""
	},
	`\+(\d+)% chance to suppress spell damage if your e?q?u?i?p?p?e?d? ?boots, helmet and gloves have evasion`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellSuppressionChance", "BASE", num).Tag(mod.StatThreshold("EvasionOnBoots", 1), mod.StatThreshold("EvasionOnHelmet", 1), mod.StatThreshold("EvasionOnGloves", 1))}, ""
	},
	"evasion rating is doubled against projectile attacks": []mod.Mod{MOD("ProjectileEvasion", "MORE", 100)},
	"evasion rating is doubled against melee attacks":      []mod.Mod{MOD("MeleeEvasion", "MORE", 100)},
	`\+(\d+)% chance to suppress spell damage for each dagger you're wielding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellSuppressionChance", "BASE", num).Tag(mod.Condition("UsingDagger")), MOD("SpellSuppressionChance", "BASE", num).Tag(mod.Condition("DualWieldingDaggers"))}, ""
	},
	"phasing":                              []mod.Mod{FLAG("Condition:Phasing")},
	"onslaught":                            []mod.Mod{FLAG("Condition:Onslaught")},
	"rampage":                              []mod.Mod{FLAG("Condition:Rampage")},
	"soul eater":                           []mod.Mod{FLAG("Condition:CanHaveSoulEater")},
	"unholy might":                         []mod.Mod{FLAG("Condition:UnholyMight")},
	"chaotic might":                        []mod.Mod{FLAG("Condition:ChaoticMight")},
	"elusive":                              []mod.Mod{FLAG("Condition:CanBeElusive")},
	"adrenaline":                           []mod.Mod{FLAG("Condition:Adrenaline")},
	"arcane surge":                         []mod.Mod{FLAG("Condition:ArcaneSurge")},
	"your aura buffs do not affect allies": []mod.Mod{FLAG("SelfAurasCannotAffectAllies")},
	`your curses have (\d+)% increased effect if (\d+)% of curse duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CurseEffect", "INC", num).Tag(mod.MultiplierThreshold("CurseExpired").Threshold(utils.Float(captures[1])).Actor("enemy"), mod.SkillType("AppliesCurse"))}, ""
	},
	`your curses have (\d+)% increased magnitudes if (\d+)% of curse duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Magnitude", "INC", num).Tag(mod.MultiplierThreshold("CurseExpired").Threshold(utils.Float(captures[1])).Actor("enemy"), mod.SkillType("AppliesCurse"))}, ""
	},
	`non\-aura hexes expire upon reaching (\d+)% of base effect non\-aura hexes gain (\d+)% increased effect per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CurseEffect", "INC", utils.Float(captures[1])).Tag(mod.Multiplier("CurseDurationExpired").Limit(num).LimitTotal(true).Actor("enemy"), mod.SkillType("Aura").Neg(true), mod.SkillType("Hex"))}, ""
	},
	`enemies cursed by you have malediction if (\d+)% of curse duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMalediction").Tag(mod.MultiplierThreshold("CurseExpired").Threshold(num), mod.ActorCondition("", "Cursed"))})}, ""
	},
	"auras from your skills can only affect you": []mod.Mod{FLAG("SelfAurasOnlyAffectYou")},
	`aura buffs from skills have (\d+)% increased effect on you for each herald affecting you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillAuraEffectOnSelf", "INC", num).Tag(mod.Multiplier("Herald"))}, ""
	},
	`aura buffs from skills have (\d+)% increased effect on you for each herald affecting you, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillAuraEffectOnSelf", "INC", num).Tag(mod.Multiplier("Herald").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("PurposefulHarbinger"))}, ""
	},
	`auras from your skills have (\d+)% increased effect on you for each herald affecting you, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillAuraEffectOnSelf", "INC", num).Tag(mod.Multiplier("Herald").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("PurposefulHarbinger"))}, ""
	},
	`(\d+)% increased area of effect per power charge, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Multiplier("PowerCharge").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("VastPower"))}, ""
	},
	`(\d+)% increased chaos damage per (\d+) maximum mana, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosDamage", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "Mana").GlobalLimit(utils.Float(captures[2])).GlobalLimitKey("DarkIdeation"))}, ""
	},
	`minions have \+(\d+)% to damage over time multiplier per ghastly eye jewel affecting you, up to a maximum of \+(\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("DotMultiplier", "BASE", num).Tag(mod.Multiplier("GhastlyEyeJewel").Actor("parent").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("AmanamuGaze"))})}, ""
	},
	`(\d+)% increased effect of arcane surge on you per hypnotic eye jewel affecting you, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArcaneSurgeEffect", "INC", num).Tag(mod.Multiplier("HypnoticEyeJewel").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("KurgalGaze"))}, ""
	},
	`(\d+)% increased main hand critical hit chance per murderous eye jewel affecting you, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChance", "INC", num).Tag(mod.Multiplier("MurderousEyeJewel").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("TecrodGazeMainHand"), mod.Condition("MainHandAttack"))}, ""
	},
	`\+(\d+)% to off hand critical damage bonus per murderous eye jewel affecting you, up to a maximum of \+(\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritMultiplier", "BASE", num).Tag(mod.Multiplier("MurderousEyeJewel").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("TecrodGazeOffHand"), mod.Condition("OffHandAttack"))}, ""
	},
	"nearby allies' damage with hits is lucky":                       []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("LuckyHitsChance", "BASE", 100), OnlyAllies: true})},
	"damage with hits is lucky against heavy stunned enemies":        []mod.Mod{MOD("LuckyHitsChance", "BASE", 100).Tag(mod.ActorCondition("enemy", "HeavyStunned"))},
	"damage with hits is lucky against enemies that are on low life": []mod.Mod{MOD("LuckyHitsChance", "BASE", 100).Tag(mod.ActorCondition("enemy", "LowLife"))},
	"your damage with hits is lucky":                                 []mod.Mod{MOD("LuckyHitsChance", "BASE", 100)},
	`(physical\|fire\|cold\|lightning\|chaos) damage with hits is lucky`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"LuckyHitsChance", "BASE", 100)}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:4336-4514
	`(\d+)% chance for (\w*) ?damage with hits to be lucky`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[1])+"LuckyHitsChance", "BASE", num)}, ""
	},
	"elemental damage with hits is lucky while you are shocked": []mod.Mod{FLAG("ElementalLuckHits").Tag(mod.Condition("Shocked"))},
	`break (\d+)% of armour on heavy stunning an enemy`:         []mod.Mod{FLAG("Condition:CanArmourBreak").Tag(mod.GlobalEffect("Buff").Name("ArmourBreak"))},
	`break (\d+)% increased armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourBreakPerHit", "INC", num)}, ""
	},
	`break (\d+)% increased armour on enemies affected by ailments`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourBreakPerHit", "INC", num).Tag(mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Electrocuted", "Ignited", "Poisoned", "Bleeding"))}, ""
	},
	`hits break (\d+)% increased armour on targets with ailments`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArmourBreakPerHit", "INC", num).Tag(mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Electrocuted", "Ignited", "Poisoned", "Bleeding"))}, ""
	},
	"allies' aura buffs do not affect you": []mod.Mod{FLAG("AlliesAurasCannotAffectSelf")},
	`(\d+)% increased effect of non\-curse auras from your skills on enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DebuffEffect", "INC", num).Tag(mod.SkillType("Aura"), mod.SkillType("AppliesCurse").Neg(true)), MOD("AuraEffect", "INC", num).Tag(mod.SkillName("Death Aura"))}, ""
	},
	"enemies can have 1 additional curse":                              []mod.Mod{MOD("EnemyCurseLimit", "BASE", 1)},
	"you can apply an additional curse":                                []mod.Mod{MOD("EnemyCurseLimit", "BASE", 1)},
	"you can apply an additional curse while affected by malevolence":  []mod.Mod{MOD("EnemyCurseLimit", "BASE", 1).Tag(mod.Condition("AffectedByMalevolence"))},
	"you can apply an additional curse while at maximum power charges": []mod.Mod{MOD("EnemyCurseLimit", "BASE", 1).Tag(mod.StatThresholdStat("PowerCharges", "PowerChargesMax"))},
	"you can apply one fewer curse":                                    []mod.Mod{MOD("EnemyCurseLimit", "BASE", -1)},
	"curses you inflict ignore curse limit":                            []mod.Mod{MOD("EnemyCurseLimit", "BASE", 99)},
	"magnitudes of curses you inflict are zero":                        []mod.Mod{MOD("CurseEffect", "MORE", -100)},
	"you can apply an additional mark":                                 []mod.Mod{MOD("EnemyMarkLimit", "BASE", 1)},
	"double activation delay of curses":                                []mod.Mod{MOD("CurseDelay", "MORE", 100)},
	"curses have no activation delay":                                  []mod.Mod{MOD("CurseDelay", "MORE", -100)},
	`curses on enemies in your chilling areas have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CurseEffect", "INC", num).Tag(mod.ActorCondition("enemy", "InChillingArea"))}, ""
	},
	"hexes you inflict have their effect increased by twice their doom instead": []mod.Mod{MOD("DoomEffect", "MORE", 100)},
	`nearby enemies have an additional (\d+)% chance to receive a critical hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("SelfExtraCritChance", "BASE", num)})}, ""
	},
	`nearby enemies have (\-\d+)% to all resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ElementalResist", "BASE", num)}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ChaosResist", "BASE", num)})}, ""
	},
	`enemies ignited or chilled by you have (\-\d+)% to elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ElementalResist", "BASE", num)}).Tag(mod.ActorCondition("enemy", "Ignited", "Chilled"))}, ""
	},
	`enemies in your presence have at least (\d+)% of life reserved`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LifeReservationPercent", "BASE", num).Tag(mod.ActorCondition("enemy", "EnemyInPresence"))})}, ""
	},
	`targets cursed by you have at least (\d+)% of life reserved`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LifeReservationPercent", "BASE", num).Tag(mod.Condition("Cursed"))})}, ""
	},
	"enemies in your presence count as being on low life": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:LowLife")}).Tag(mod.Condition("EnemyInPresence"))},
	"enemies in your presence are hindered":               []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Hindered")}).Tag(mod.Condition("EnemyInPresence"))},
	"enemies in your presence have fire exposure":         []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", 20).Tag(mod.ActorCondition("enemy", "EnemyInPresence"))})},
	`your hits inflict decay, dealing (\d+) chaos damage per second for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "decay", Value: num, Merge: "MAX"})}, ""
	},
	`inflict decay on enemies you curse with hex or mark skills, dealing (\d+) chaos damage per second for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "decay", Value: num, Merge: "MAX"}).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`inflict decay on enemies you curse with hex skills, dealing (\d+) chaos damage per second for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "decay", Value: num, Merge: "MAX"}).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`temporal chains has (\d+)% reduced effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CurseEffectOnSelf", "INC", -num).Tag(mod.SkillName("Temporal Chains"))}, ""
	},
	"unaffected by temporal chains": []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.SkillName("Temporal Chains"), mod.GlobalEffect("Global").Unscalable(true))},
	`([\+\-][\d\.]+) seconds to cat's stealth duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PrimaryDuration", "BASE", num).Tag(mod.SkillName("Aspect of the Cat"))}, ""
	},
	`([\+\-][\d\.]+) seconds to cat's agility duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SecondaryDuration", "BASE", num).Tag(mod.SkillName("Aspect of the Cat"))}, ""
	},
	`([\+\-][\d\.]+) seconds to avian's might duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PrimaryDuration", "BASE", num).Tag(mod.SkillName("Aspect of the Avian"))}, ""
	},
	`([\+\-][\d\.]+) seconds to avian's flight duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SecondaryDuration", "BASE", num).Tag(mod.SkillName("Aspect of the Avian"))}, ""
	},
	"aspect of the spider can inflict spider's web on enemies an additional time":       []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("Multiplier:SpiderWebApplyStackMax", "BASE", 1)}).Tag(mod.SkillName("Aspect of the Spider"))},
	"aspect of the avian also grants avian's might and avian's flight to nearby allies": []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: FLAG("BuffAppliesToAllies")}).Tag(mod.SkillName("Aspect of the Avian"))},
	`enemies you mark take (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "Marked"))}, ""
	},
	`enemies you mark have (\d+)% reduced accuracy rating`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("Accuracy", "INC", -num)}).Tag(mod.ActorCondition("enemy", "Marked"))}, ""
	},
	`you are cursed with level (\d+) (\D+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraCurse", "LIST", mod.ExtraCurse{SkillID: extraSkillIDForName(captures[1]), Level: int(num), ApplyToPlayer: true})}, ""
	},
	`you are cursed with (\D+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraCurse", "LIST", mod.ExtraCurse{SkillID: extraSkillIDForName(captures[0]), Level: 1, ApplyToPlayer: true})}, ""
	},
	`you are cursed with (\D+), with (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraCurse", "LIST", mod.ExtraCurse{SkillID: extraSkillIDForName(captures[0]), Level: 1, ApplyToPlayer: true}),
			MOD("CurseEffectOnSelf", "INC", utils.Float(captures[1])).Tag(mod.SkillName(utils.CapitalEach(captures[0]))),
		}, ""
	},
	"you count as on low life while you are cursed with vulnerability":  []mod.Mod{FLAG("Condition:LowLife").Tag(mod.Condition("AffectedByVulnerability"))},
	"you count as on full life while you are cursed with vulnerability": []mod.Mod{FLAG("Condition:FullLife").Tag(mod.Condition("AffectedByVulnerability"))},
	`if you consumed a corpse recently, you and nearby allies regenerate (\d+)% of life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("LifeRegenPercent", "BASE", num)}).Tag(mod.Condition("ConsumedCorpseRecently"))}, ""
	},
	`if you have blocked recently, you and nearby allies regenerate (\d+)% of life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("LifeRegenPercent", "BASE", num)}).Tag(mod.Condition("BlockedRecently"))}, ""
	},
	"you are at maximum chance to block attack damage if you have not blocked recently": []mod.Mod{FLAG("MaxBlockIfNotBlockedRecently").Tag(mod.Condition("BlockedRecently").Neg(true))},
	"you are at maximum chance to block spell damage if you have not blocked recently":  []mod.Mod{FLAG("MaxSpellBlockIfNotBlockedRecently").Tag(mod.Condition("BlockedRecently").Neg(true))},
	`\+(\d+)% chance to block attack damage if you have not blocked recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BlockChance", "BASE", num).Tag(mod.Condition("BlockedRecently").Neg(true))}, ""
	},
	"chance to block damage is lucky":                                            []mod.Mod{FLAG("BlockChanceIsLucky")},
	"y?o?u?r? ?chance to block is lucky":                                         []mod.Mod{FLAG("BlockChanceIsLucky"), FLAG("ProjectileBlockChanceIsLucky"), FLAG("SpellBlockChanceIsLucky"), FLAG("SpellProjectileBlockChanceIsLucky")},
	"y?o?u?r? ?chance to block is unlucky":                                       []mod.Mod{FLAG("BlockChanceIsUnlucky"), FLAG("ProjectileBlockChanceIsUnlucky"), FLAG("SpellBlockChanceIsUnlucky"), FLAG("SpellProjectileBlockChanceIsUnlucky")},
	"chance to block attack or spell damage is lucky if you've blocked recently": []mod.Mod{FLAG("BlockChanceIsLucky").Tag(mod.Condition("BlockedRecently")), FLAG("ProjectileBlockChanceIsLucky").Tag(mod.Condition("BlockedRecently")), FLAG("SpellBlockChanceIsLucky").Tag(mod.Condition("BlockedRecently")), FLAG("SpellProjectileBlockChanceIsLucky").Tag(mod.Condition("BlockedRecently"))},
	`([\d\.]+)% of evasion rating is regenerated as life per second while focus?sed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegen", "BASE", 1).Tag(mod.PercentStat("Evasion", num), mod.Condition("Focused"))}, ""
	},
	`nearby allies have (\d+)% increased defences per (\d+) strength you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("Defences", "INC", num), OnlyAllies: true}).Tag(mod.PerStat(utils.Float(captures[1]), "Str"))}, ""
	},
	`nearby allies have \+(\d+)% to critical damage bonus per (\d+) dexterity you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("CritMultiplier", "BASE", num), OnlyAllies: true}).Tag(mod.PerStat(utils.Float(captures[1]), "Dex"))}, ""
	},
	`nearby allies have (\d+)% increased cast speed per (\d+) intelligence you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("Speed", "INC", num).Flag(ModFlagCast), OnlyAllies: true}).Tag(mod.PerStat(utils.Float(captures[1]), "Int"))}, ""
	},
	"quicksilver flasks you use also apply to nearby allies":                 []mod.Mod{FLAG("QuickSilverAppliesToAllies")},
	`you gain divinity for \d+ seconds on reaching maximum divine charges`:   []mod.Mod{MOD("ElementalDamage", "MORE", 50).Tag(mod.Condition("Divinity")), MOD("ElementalDamageTaken", "MORE", -20).Tag(mod.Condition("Divinity"))},
	"your nearby party members maximum endurance charges is equal to yours":  []mod.Mod{FLAG("PartyMemberMaximumEnduranceChargesEqualToYours")},
	"your maximum endurance charges is equal to your maximum frenzy charges": []mod.Mod{FLAG("MaximumEnduranceChargesIsMaximumFrenzyCharges")},
	"your maximum frenzy charges is equal to your maximum power charges":     []mod.Mod{FLAG("MaximumFrenzyChargesIsMaximumPowerCharges")},
	"your curse limit is equal to your maximum power charges":                []mod.Mod{FLAG("CurseLimitIsMaximumPowerCharges")},
	`consecrated ground you create while affected by zealotry causes enemies to take (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTakenConsecratedGround", "INC", num)}).Tag(mod.ActorCondition("enemy", "OnConsecratedGround"), mod.Condition("AffectedByZealotry"))}, ""
	},
	`if you've warcried recently, you and nearby allies have (\d+)% increased attack, cast and movement speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("Speed", "INC", num)}).Tag(mod.Condition("UsedWarcryRecently")), MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("MovementSpeed", "INC", num)}).Tag(mod.Condition("UsedWarcryRecently"))}, ""
	},
	`(\d+)% increased movement speed while on full life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeed", "INC", num).Tag(mod.Condition("FullLife"))}, ""
	},
	"when you warcry, you and nearby allies gain onslaught for 4 seconds": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("Onslaught")}).Tag(mod.Condition("UsedWarcryRecently"))},
	`warcries grant arcane surge to you and allies, with (\d+)% increased effect per (\d+) power, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("Condition:ArcaneSurge")}).Tag(mod.Condition("UsedWarcryRecently")), MOD("ArcaneSurgeEffect", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "WarcryPower").GlobalLimit(utils.Float(captures[2])).GlobalLimitKey("Brinerot Flag"), mod.Condition("UsedWarcryRecently"))}, ""
	},
	`gain arcane surge after spending a total of (\d+) mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("Condition:ArcaneSurge")}).Tag(mod.MultiplierThreshold("ManaSpentRecently").Threshold(num))}, ""
	},
	`gain arcane surge after spending a total of (\d+) life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("Condition:ArcaneSurge")}).Tag(mod.MultiplierThreshold("LifeSpentRecently").Threshold(num))}, ""
	},
	"arcane surge grants more life regeneration rate instead of mana regeneration rate": []mod.Mod{FLAG("ArcaneSurgeLifeRegen")},
	`(\d+)% chance to gain onslaught on killing hits with this weapon`:                  []mod.Mod{FLAG("Onslaught").Tag(mod.Condition("KilledRecently"))},
	`gain onslaught for (\d+) seconds on hit while at maximum frenzy charges`:           []mod.Mod{FLAG("Onslaught").Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"), mod.Condition("HitRecently"))},
	`enemies in your chilling areas take (\d+)% increased lightning damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningDamageTaken", "INC", num)}).Tag(mod.ActorCondition("enemy", "InChillingArea"))}, ""
	},
	`warcries count as having (\d+) additional nearby enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:WarcryNearbyEnemies", "BASE", num)}, ""
	},
	`enemies taunted by your warcries take (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Condition("Taunted"))}).Tag(mod.Condition("UsedWarcryRecently"))}, ""
	},
	`warcries have minimum of (\d+) power`:                               []mod.Mod{FLAG("CryWolfMinimumPower")},
	"warcries have infinite power":                                       []mod.Mod{FLAG("WarcryInfinitePower")},
	"your warcries do not grant buffs or charges to you":                 []mod.Mod{FLAG("CannotGainWarcryBuffs")},
	`(\d+)% chance to inflict corrosion on hit with attacks`:             []mod.Mod{FLAG("Condition:CanCorrode")},
	`(\d+)% chance to inflict withered for (\d+) seconds on hit`:         []mod.Mod{FLAG("Condition:CanWither")},
	`melee weapon hits inflict (\d+) withered debuffs for (\d+) seconds`: []mod.Mod{FLAG("Condition:CanWither")},
	`inflict withered for (\d+) seconds on hit if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Condition:CanWither").Tag(mod.Condition(selfCastCondition(captures[1])))}, ""
	},
	`(\d+)% chance for attack hits to apply t?e?n? ?incision`:                                                        []mod.Mod{FLAG("Condition:CanInflictIncision")},
	`(\d+)% chance to inflict withered for (\d+) seconds on hit with this weapon`:                                    []mod.Mod{FLAG("Condition:CanWither")},
	`(\d+)% chance to inflict withered for two seconds on hit if there are (\d+) or fewer withered debuffs on enemy`: []mod.Mod{FLAG("Condition:CanWither")},
	`inflict withered for (\d+) seconds on hit with this weapon`:                                                     []mod.Mod{FLAG("Condition:CanWither")},
	`minions have (\d+)% chance to inflict withered on hit`:                                                          []mod.Mod{FLAG("Condition:CanWither")},
	`enemies take (\d+)% increased elemental damage from your hits for each withered you have inflicted on them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ElementalDamageTaken", "INC", num).Tag(mod.Multiplier("WitheredStack").Limit(15))})}, ""
	},
	`enemies you apply incision to take (\d+)% increased physical damage per incision`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("PhysicalDamageTaken", "INC", num).Tag(mod.Multiplier("IncisionStack"))})}, ""
	},
	"your hits cannot penetrate or ignore elemental resistances":                             []mod.Mod{FLAG("CannotElePenIgnore")},
	"nearby enemies have malediction":                                                        []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMalediction")})},
	"gain shaper's presence for 10 seconds when you kill a rare or unique enemy":             []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("HasShapersPresence")}).Tag(mod.Condition("KilledUniqueEnemy"))},
	"gain maddening presence for 10 seconds when you kill a rare or unique enemy":            []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMaddeningPresence")}).Tag(mod.Condition("KilledUniqueEnemy"))},
	"elemental damage you deal with hits is resisted by lowest elemental resistance instead": []mod.Mod{FLAG("ElementalDamageUsesLowestResistance")},
	"enemies in your presence resist elemental damage based on their lowest resistance":      []mod.Mod{FLAG("ElementalDamageUsesLowestResistance").Tag(mod.Condition("EnemyInPresence"))},
	"enemies in your presence have no elemental resistances":                                 []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireResist", "OVERRIDE", 0).Tag(mod.ActorCondition("enemy", "EnemyInPresence"))}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdResist", "OVERRIDE", 0).Tag(mod.ActorCondition("enemy", "EnemyInPresence"))}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningResist", "OVERRIDE", 0).Tag(mod.ActorCondition("enemy", "EnemyInPresence"))})},
	`you take (\d+) chaos damage per second for 3 seconds on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosDegen", "BASE", num).Tag(mod.Condition("KilledLast3Seconds"))}, ""
	},
	`regenerate (\d+) life per second for each (\d+)% uncapped fire resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegen", "BASE", num).Tag(mod.PerStat((1 / utils.Float(captures[1])), "FireResistTotal"))}, ""
	},
	`regenerate (\d+) life over 1 second for each spell you cast`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegen", "BASE", num).Tag(mod.Condition("CastLast1Seconds"))}, ""
	},
	`and nearby allies regenerate (\d+) life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("LifeRegen", "BASE", num)}).Tag(mod.Condition("KilledPoisonedLast2Seconds"))}, ""
	},
	`(\d+)% increased life regeneration rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegen", "INC", num)}, ""
	},
	`every (\d+) seconds, regenerate life equal to (\d+)% of armour and evasion rating over (\d+) second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegen", "BASE", 1).Tag(mod.Condition("LifeRegenBurstFull"), mod.PercentStat("Armour", utils.Float(captures[1]))), MOD("LifeRegen", "BASE", ((1/num)*utils.Float(captures[2]))).Tag(mod.Condition("LifeRegenBurstAvg"), mod.PercentStat("Armour", utils.Float(captures[1]))), MOD("LifeRegen", "BASE", 1).Tag(mod.Condition("LifeRegenBurstFull"), mod.PercentStat("Evasion", utils.Float(captures[1]))), MOD("LifeRegen", "BASE", ((1/num)*utils.Float(captures[2]))).Tag(mod.Condition("LifeRegenBurstAvg"), mod.PercentStat("Evasion", utils.Float(captures[1])))}, ""
	},
	`every (\d+) seconds, regenerate energy shield equal to (\d+)% of evasion rating over (\d+) second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldRegen", "BASE", 1).Tag(mod.Condition("LifeRegenBurstFull"), mod.PercentStat("Evasion", utils.Float(captures[1]))), MOD("EnergyShieldRegen", "BASE", ((1/num)*utils.Float(captures[2]))).Tag(mod.Condition("LifeRegenBurstAvg"), mod.PercentStat("Evasion", utils.Float(captures[1])))}, ""
	},
	`regenerate (\d+)% of life per second for each different ailment affecting you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Bleeding")), MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Ignited")), MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Scorched")), MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Chilled")), MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Frozen")), MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Brittle")), MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Shocked")), MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Sapped")), MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Poisoned"))}, ""
	},
	`fire skills have a (\d+)% chance to apply fire exposure on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireExposureChance", "BASE", num)}, ""
	},
	`cold skills have a (\d+)% chance to apply cold exposure on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ColdExposureChance", "BASE", num)}, ""
	},
	`lightning skills have a (\d+)% chance to apply lightning exposure on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningExposureChance", "BASE", num)}, ""
	},
	`(\d+)% chance to inflict cold exposure on hit with cold damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ColdExposureChance", "BASE", num)}, ""
	},
	"inflict abyssal wasting on hit": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("AbyssalWasted").Flag(ModFlagHit).Tag(mod.Condition("Effective"))})},
	//pob2:block:src/Modules/ModParser.lua:4515-4527
	`targets affected by abyssal wasting you inflict are (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		conditions := map[string]string{
			"debilitated": "Debilitated",
			"hindered":    "Hindered",
			"blinded":     "Blinded",
		}
		condType, ok := conditions[captures[0]]
		if !ok {
			return nil, ""
		}
		return []mod.Mod{MOD("AbyssalWastingImpliesCondition", "LIST", LuaTable{"condition": condType, "applyToEnemy": true})}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:4528-4536
	`abyssal wasting also applies \-(\d+)% to (\w+) resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		resType, ok := dmgTypes[captures[1]]
		if !ok {
			return nil, ""
		}
		modName := resType + "Resist"
		return []mod.Mod{MOD("AbyssalWastingAlsoGrants", "LIST", LuaTable{"mod": MOD(modName, "BASE", -num), "applyToEnemy": true})}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:4537-4541
	"abyssal wasting you inflict also prevents targets from dealing critical hits":            []mod.Mod{MOD("AbyssalWastingImpliesCondition", "LIST", LuaTable{"condition": "Condition:NeverCrit", "applyToEnemy": true}), MOD("AbyssalWastingImpliesCondition", "LIST", LuaTable{"condition": "NeverCrit", "applyToEnemy": true})},
	`(\d+)% chance to inflict withered with hits against targets affected by abyssal wasting`: []mod.Mod{MOD("AbyssalWastingImpliesCondition", "LIST", LuaTable{"condition": "Condition:CanWither"})},
	//pob2:block:src/Modules/ModParser.lua:4542-4545
	`(\d+)% of ([lm][ia][fn][ea]) leeched from targets affected by abyssal wasting is instant`: func(num float64, captures []string) ([]mod.Mod, string) {
		modName := "Instant" + firstToUpper(captures[1]) + "Leech"
		return []mod.Mod{MOD("AbyssalWastingAlsoGrants", "LIST", LuaTable{"mod": MOD(modName, "BASE", num), "unscalable": true})}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:4546-4595
	`(\d+)% increased accuracy rating against enemies affected by abyssal wasting`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AbyssalWastingAlsoGrants", "LIST", LuaTable{"mod": MOD("Accuracy", "INC", num), "unscalable": true})}, ""
	},
	`(\d+)% increased chance to inflict ailments against enemies affected by abyssal wasting`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AbyssalWastingAlsoGrants", "LIST", LuaTable{"mod": MOD("AilmentChance", "INC", num), "unscalable": true})}, ""
	},
	`(\d+)% increased immobilisation buildup against targets affected by abyssal wasting`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AbyssalWastingAlsoGrants", "LIST", LuaTable{"mod": MOD("EnemyImmobilisationBuildup", "INC", num), "unscalable": true})}, ""
	},
	"abyssal wasting you inflict also prevents targets from inflicting elemental ailments": []mod.Mod{MOD("AbyssalWastingAlsoGrants", "LIST", LuaTable{"mod": MOD("AvoidElementalAilments", "BASE", 100), "unscalable": true})},
	"hits against you have no critical damage bonus while on consecrated ground":           []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.Condition("OnConsecratedGround"))},
	"socketed skills apply fire, cold and lightning exposure on hit":                       []mod.Mod{MOD("FireExposureChance", "BASE", 100).Tag(mod.Condition("Effective")), MOD("ColdExposureChance", "BASE", 100).Tag(mod.Condition("Effective")), MOD("LightningExposureChance", "BASE", 100).Tag(mod.Condition("Effective"))},
	"inflict fire, cold, and lightning exposure on hit":                                    []mod.Mod{MOD("FireExposureChance", "BASE", 100).Tag(mod.Condition("Effective")), MOD("ColdExposureChance", "BASE", 100).Tag(mod.Condition("Effective")), MOD("LightningExposureChance", "BASE", 100).Tag(mod.Condition("Effective"))},
	"inflict fire exposure on hit":                                                         []mod.Mod{MOD("FireExposureChance", "BASE", 100).Tag(mod.Condition("Effective"))},
	"inflict cold exposure on hit":                                                         []mod.Mod{MOD("ColdExposureChance", "BASE", 100).Tag(mod.Condition("Effective"))},
	"inflict lightning exposure on hit":                                                    []mod.Mod{MOD("LightningExposureChance", "BASE", 100).Tag(mod.Condition("Effective"))},
	"nearby enemies have fire exposure":                                                    []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", 20)}).Tag(mod.Condition("Effective"))},
	"nearby enemies have cold exposure":                                                    []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", 20)}).Tag(mod.Condition("Effective"))},
	"nearby enemies have lightning exposure":                                               []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", 20)}).Tag(mod.Condition("Effective"))},
	"nearby enemies have fire exposure while you are affected by herald of ash":            []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", 20)}).Tag(mod.Condition("Effective"), mod.Condition("AffectedByHeraldofAsh"))},
	"nearby enemies have cold exposure while you are affected by herald of ice":            []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", 20)}).Tag(mod.Condition("Effective"), mod.Condition("AffectedByHeraldofIce"))},
	"nearby enemies have lightning exposure while you are affected by herald of thunder":   []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", 20)}).Tag(mod.Condition("Effective"), mod.Condition("AffectedByHeraldofThunder"))},
	//pob2:block:src/Modules/ModParser.lua:4599-5314
	"inflict fire, cold and lightning exposure on nearby enemies when used": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", 20)}).Tag(mod.Condition("Effective"), mod.Condition("UsingFlask")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", 20)}).Tag(mod.Condition("Effective"), mod.Condition("UsingFlask")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", 20)}).Tag(mod.Condition("Effective"), mod.Condition("UsingFlask"))},
	`inflict elemental exposure on hit, lowering total elemental resistances by (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", num)}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", num)}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", num)}), FLAG("Condition:CanApplyFireExposure"), FLAG("Condition:CanApplyColdExposure"), FLAG("Condition:CanApplyLightningExposure")}, ""
	},
	"enemies near your linked targets have fire, cold and lightning exposure": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", 20).Tag(mod.Condition("NearLinkedTarget"))}).Tag(mod.Condition("Effective")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", 20).Tag(mod.Condition("NearLinkedTarget"))}).Tag(mod.Condition("Effective")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", 20).Tag(mod.Condition("NearLinkedTarget"))}).Tag(mod.Condition("Effective"))},
	`inflict (\w+) exposure on hit, applying \-(\d+)% to (\w+) resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"ExposureChance", "BASE", 100).Tag(mod.Condition("Effective")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(firstToUpper(captures[2])+"Exposure", "BASE", utils.Float(captures[1]))}).Tag(mod.Condition("Effective"))}, ""
	},
	`while a unique enemy is in your presence, inflict (\w+) exposure on hit, applying \-(\d+)% to (\w+) resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"ExposureChance", "BASE", 100).Tag(mod.ActorCondition("enemy", "RareOrUnique"), mod.Condition("Effective")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(firstToUpper(captures[2])+"Exposure", "BASE", utils.Float(captures[1])).Tag(mod.Condition("RareOrUnique"))}).Tag(mod.Condition("Effective"))}, ""
	},
	`while a pinnacle atlas boss is in your presence, inflict (\w+) exposure on hit, applying \-(\d+)% to (\w+) resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"ExposureChance", "BASE", 100).Tag(mod.ActorCondition("enemy", "PinnacleBoss"), mod.Condition("Effective")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(firstToUpper(captures[2])+"Exposure", "BASE", utils.Float(captures[1])).Tag(mod.Condition("PinnacleBoss"))}).Tag(mod.Condition("Effective"))}, ""
	},
	`fire exposure you inflict applies an extra (\-?\d+)% to fire resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraFireExposure", "BASE", -num)}, ""
	},
	`fire exposure you inflict lowers total fire resistance by an extra (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraFireExposure", "BASE", num)}, ""
	},
	`cold exposure you inflict applies an extra (\-?\d+)% to cold resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraColdExposure", "BASE", -num)}, ""
	},
	`cold exposure you inflict lowers total cold resistance by an extra (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraColdExposure", "BASE", num)}, ""
	},
	`lightning exposure you inflict applies an extra (\-?\d+)% to lightning resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraLightningExposure", "BASE", -num)}, ""
	},
	`lightning exposure you inflict lowers total lightning resistance by an extra (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraLightningExposure", "BASE", num)}, ""
	},
	`exposure you inflict applies at least (\-\d+)% to the affected resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExposureMin", "OVERRIDE", -num)}, ""
	},
	"enemies in your presence gain critical weakness every second":                          []mod.Mod{FLAG("ApplyCriticalWeakness")},
	`every second, inflicts critical weakness on enemies in your presence for \d+ seconds?`: []mod.Mod{FLAG("ApplyCriticalWeakness")},
	`applies critical weakness to enemies every ([\d\.]+) seconds?`:                         []mod.Mod{FLAG("ApplyCriticalWeakness")},
	"inflicts critical weakness on hit":                                                     []mod.Mod{FLAG("ApplyCriticalWeakness")},
	`apply (\d+) critical weakness to enemies when consuming a mark on them`:                []mod.Mod{FLAG("ApplyCriticalWeakness")},
	"modifiers to minimum endurance charges instead apply to minimum brutal charges":        []mod.Mod{FLAG("MinimumEnduranceChargesEqualsMinimumBrutalCharges")},
	"modifiers to minimum frenzy charges instead apply to minimum affliction charges":       []mod.Mod{FLAG("MinimumFrenzyChargesEqualsMinimumAfflictionCharges")},
	"modifiers to minimum power charges instead apply to minimum absorption charges":        []mod.Mod{FLAG("MinimumPowerChargesEqualsMinimumAbsorptionCharges")},
	"maximum brutal charges is equal to maximum endurance charges":                          []mod.Mod{FLAG("MaximumEnduranceChargesEqualsMaximumBrutalCharges")},
	"maximum affliction charges is equal to maximum frenzy charges":                         []mod.Mod{FLAG("MaximumFrenzyChargesEqualsMaximumAfflictionCharges")},
	"maximum absorption charges is equal to maximum power charges":                          []mod.Mod{FLAG("MaximumPowerChargesEqualsMaximumAbsorptionCharges")},
	"gain brutal charges instead of endurance charges":                                      []mod.Mod{FLAG("EnduranceChargesConvertToBrutalCharges")},
	"gain affliction charges instead of frenzy charges":                                     []mod.Mod{FLAG("FrenzyChargesConvertToAfflictionCharges")},
	"gain absorption charges instead of power charges":                                      []mod.Mod{FLAG("PowerChargesConvertToAbsorptionCharges")},
	`regenerate (\d+)% life over one second when hit while sane`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Insane").Neg(true), mod.Condition("BeenHitRecently"))}, ""
	},
	`you [ac][ro][eu][ n][ct][o ][na]si?d?e?r?e?d? on low (\w+) while at (\d+)% of maximum (\w+) or below ?i?n?s?t?e?a?d?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Low"+firstToUpper(captures[0])+"Percentage", "BASE", (utils.Float(captures[1]) / 100))}, ""
	},
	`you [ac][ro][eu][ n][ct][o ][na]si?d?e?r?e?d? on full (\w+) while at (\d+)% of maximum (\w+) or above ?i?n?s?t?e?a?d?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Full"+firstToUpper(captures[0])+"Percentage", "BASE", (utils.Float(captures[1]) / 100))}, ""
	},
	`(\d+)% more maximum life if you have at least (\d+) life masteries allocated`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Life", "MORE", num).Tag(mod.MultiplierThreshold("AllocatedLifeMastery").Threshold(utils.Float(captures[1])))}, ""
	},
	"left ring slot: cover enemies in ash for 5 seconds when you ignite them":                      []mod.Mod{MOD("CoveredInAshEffect", "BASE", 20).Tag(mod.SlotNumber(int(1)), mod.ActorCondition("enemy", "Ignited"))},
	"right ring slot: cover enemies in frost for 5 seconds when you freeze them":                   []mod.Mod{MOD("CoveredInFrostEffect", "BASE", 20).Tag(mod.SlotNumber(int(2)), mod.ActorCondition("enemy", "Frozen"))},
	"nearby enemies are covered in ash":                                                            []mod.Mod{MOD("CoveredInAshEffect", "BASE", 20)},
	`enemies near targets you shatter have (\d+)% chance to be covered in frost for (\d+) seconds`: []mod.Mod{MOD("CoveredInFrostEffect", "BASE", 20).Tag(mod.Condition("ShatteredEnemyRecently"))},
	`([\w\s]+) has (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BuffEffect", "INC", utils.Float(captures[1])).Tag(mod.SkillId(extraSkillIDForName(captures[0])))}, ""
	},
	`debuffs on you expire (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SelfDebuffExpirationRate", "BASE", num)}, ""
	},
	`buffs on you expire (\d+)% slower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Duration", "INC", num).Tag(mod.SkillType("Buff"))}, ""
	},
	`warcries debilitate enemies for (\d+) seconds?`:                             []mod.Mod{MOD("DebilitateChance", "BASE", 100)},
	"warcries debilitate enemies":                                                []mod.Mod{MOD("DebilitateChance", "BASE", 100)},
	`debilitate enemies for (\d+) seconds? when you suppress their spell damage`: []mod.Mod{MOD("DebilitateChance", "BASE", 100)},
	`debilitate nearby enemies for (\d+) seconds? when f?l?a?s?k? ?effect ends`:  []mod.Mod{MOD("DebilitateChance", "BASE", 100)},
	`counterattacks have a (\d+)% chance to debilitate on hit for (\d+) seconds?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DebilitateChance", "BASE", num)}, ""
	},
	`retaliation skills debilitate enemies for (\d+) seconds on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DebilitateChance", "BASE", 100)}, ""
	},
	"eat a soul when you hit a unique enemy, no more than once every second":                                                               []mod.Mod{FLAG("Condition:CanHaveSoulEater")},
	`eat a soul when you hit a rare or unique enemy, no more than once every [\d\.]+ seconds`:                                              []mod.Mod{FLAG("Condition:CanHaveSoulEater")},
	`(\d+)% chance to gain soul eater for (\d+) seconds on killing blow against rare and unique enemies with double strike or dual strike`: []mod.Mod{FLAG("Condition:CanHaveSoulEater")},
	`maximum (\d+) eaten souls`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SoulEaterMax", "OVERRIDE", num)}, ""
	},
	`([\+\-]\d+) to maximum number of eaten souls`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SoulEaterMax", "BASE", num)}, ""
	},
	`(\d+)% increased attack and cast speed if you've killed recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "INC", num).Flag(ModFlagCast).Tag(mod.Condition("KilledRecently")), MOD("Speed", "INC", num).Flag(ModFlagAttack).Tag(mod.Condition("KilledRecently"))}, ""
	},
	"gain adrenaline for 1 second when you change stance":                                     []mod.Mod{FLAG("Condition:Adrenaline").Tag(mod.Condition("StanceChangeLastSecond"))},
	`with a searching eye jewel socketed, maim enemies for (\d) seconds on hit with attacks`:  []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Maimed").Flag(ModFlagAttack)}).Tag(mod.Condition("HaveSearchingEyeJewelIn{SlotName}"))},
	`with a searching eye jewel socketed, blind enemies for (\d) seconds on hit with attacks`: []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Blinded").Flag(ModFlagAttack)}).Tag(mod.Condition("HaveSearchingEyeJewelIn{SlotName}"))},
	`enemies maimed by you take (\d+)% increased damage over time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTakenOverTime", "INC", num)}).Tag(mod.ActorCondition("enemy", "Maimed"))}, ""
	},
	`(\d+)% increased defences while you have at least four linked targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Defences", "INC", num).Tag(mod.MultiplierThreshold("LinkedTargets").Threshold(4))}, ""
	},
	"your movement speed is equal to the highest movement speed among linked players": []mod.Mod{FLAG("MovementSpeedEqualHighestLinkedPlayers").Tag(mod.MultiplierThreshold("LinkedTargets").Threshold(1))},
	`(\d+)% increased movement speed while you have at least two linked targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeed", "INC", num).Tag(mod.MultiplierThreshold("LinkedTargets").Threshold(2))}, ""
	},
	`link skills have (\d+)% increased buff effect if you have linked to a target recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BuffEffect", "INC", num).Tag(mod.SkillType("Link"), mod.Condition("LinkedRecently"))}, ""
	},
	"link skills can target damageable minions":                         []mod.Mod{FLAG("Condition:CanLinkToMinions")},
	"curses are inflicted on you instead of linked targets":             []mod.Mod{MOD("ExtraLinkEffect", "LIST", LuaTable{"mod": FLAG("CurseImmune")})},
	"elemental ailments are inflicted on you instead of linked targets": []mod.Mod{MOD("ExtraLinkEffect", "LIST", LuaTable{"mod": FLAG("ElementalAilmentImmune")})},
	`non\-unique utility flasks you use apply to linked targets`:        []mod.Mod{MOD("ExtraLinkEffect", "LIST", LuaTable{"mod": MOD("ParentNonUniqueFlasksAppliedToYou", "FLAG", true).Tag(mod.GlobalEffect("Global").Unscalable(true))})},
	`gain unholy might on block for (\d) seconds`:                       []mod.Mod{FLAG("Condition:UnholyMight").Tag(mod.Condition("BlockedRecently"))},
	`require (\d+) fewer enemies to be surrounded`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SurroundedMinimum", "BASE", -num)}, ""
	},
	`traps and mines deal (\d+)\-(\d+) additional physical damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalMin", "BASE", utils.Float(captures[0])).KeywordFlag(KeywordFlagTrap | KeywordFlagMine), MOD("PhysicalMax", "BASE", utils.Float(captures[1])).KeywordFlag(KeywordFlagTrap | KeywordFlagMine)}, ""
	},
	`traps and mines deal (\d+) to (\d+) additional physical damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalMin", "BASE", utils.Float(captures[0])).KeywordFlag(KeywordFlagTrap | KeywordFlagMine), MOD("PhysicalMax", "BASE", utils.Float(captures[1])).KeywordFlag(KeywordFlagTrap | KeywordFlagMine)}, ""
	},
	`each mine applies (\d+)% increased damage taken to enemies near it, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Multiplier("ActiveMineCount").Limit((utils.Float(captures[1]) / num)))})}, ""
	},
	`each mine applies (\d+)% reduced damage dealt to enemies near it, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("Damage", "INC", -num).Tag(mod.Multiplier("ActiveMineCount").Limit((utils.Float(captures[1]) / num)))})}, ""
	},
	`stormblast, icicle and pyroclast mine have (\d+)% increased aura effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AuraEffect", "INC", num).KeywordFlag(KeywordFlagMine).Tag(mod.SkillName("Stormblast Mine", "Icicle Mine", "Pyroclast Mine").Transfigured(true))}, ""
	},
	"stormblast, icicle and pyroclast mine deal no damage": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("DealNoLightning").Tag(mod.SkillName("Stormblast Mine", "Icicle Mine", "Pyroclast Mine").Transfigured(true)), FLAG("DealNoCold").Tag(mod.SkillName("Stormblast Mine", "Icicle Mine", "Pyroclast Mine").Transfigured(true)), FLAG("DealNoFire").Tag(mod.SkillName("Stormblast Mine", "Icicle Mine", "Pyroclast Mine").Transfigured(true)), FLAG("DealNoChaos").Tag(mod.SkillName("Stormblast Mine", "Icicle Mine", "Pyroclast Mine").Transfigured(true)), FLAG("DealNoPhysical").Tag(mod.SkillName("Stormblast Mine", "Icicle Mine", "Pyroclast Mine").Transfigured(true))}, ""
	},
	`can have up to (\d+) additional traps? placed at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveTrapLimit", "BASE", num)}, ""
	},
	`can have (\d+) fewer traps placed at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveTrapLimit", "BASE", -num)}, ""
	},
	`can have up to (\d+) additional remote mines? placed at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveMineLimit", "BASE", num)}, ""
	},
	"throw an additional trap": []mod.Mod{MOD("TrapThrowCount", "BASE", 1)},
	`(\d+)% chance to throw up to (\d+) additional traps?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TrapThrowCount", "BASE", ((utils.Float(captures[1]) * num) / 100))}, ""
	},
	"throw an additional mine": []mod.Mod{MOD("MineThrowCount", "BASE", 1)},
	`(\d+)% chance to throw up to (\d+) additional mines?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MineThrowCount", "BASE", ((utils.Float(captures[1]) * num) / 100))}, ""
	},
	`(\d+)% chance to throw up to (\d+) additional traps? or mines?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TrapThrowCount", "BASE", ((utils.Float(captures[1]) * num) / 100)), MOD("MineThrowCount", "BASE", ((utils.Float(captures[1]) * num) / 100))}, ""
	},
	`(\d+)% increased damage for each hazard triggered recently, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Tag(mod.Multiplier("HazardsTriggeredRecently").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("DmgPerHazardRecently"))}, ""
	},
	`hazards have (\d+)% chance to rearm after they are triggered`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HazardRearmChance", "BASE", num).Tag(mod.SkillType("Hazard"))}, ""
	},
	`can have up to (\d+) additional totems? summoned at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveTotemLimit", "BASE", num)}, ""
	},
	`attack skills can have (\d+) additional totems? summoned at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveTotemLimit", "BASE", num).KeywordFlag(KeywordFlagAttack)}, ""
	},
	`can [hs][au][vm][em]o?n? 1 additional siege ballista totem per (\d+) dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveBallistaLimit", "BASE", 1).Tag(mod.SkillName("Siege Ballista").Transfigured(true), mod.PerStat(num, "Dex"))}, ""
	},
	`totems fire (\d+) additional projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).KeywordFlag(KeywordFlagTotem)}, ""
	},
	`([\d\.]+)% of damage dealt by y?o?u?r? ?totems is leeched to you as life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageLifeLeechToPlayer", "BASE", num).KeywordFlag(KeywordFlagTotem)}, ""
	},
	`([\d\.]+)% of damage dealt by y?o?u?r? ?mines is leeched to you as life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageLifeLeechToPlayer", "BASE", num).KeywordFlag(KeywordFlagMine)}, ""
	},
	"you can cast an additional brand": []mod.Mod{MOD("ActiveBrandLimit", "BASE", 1)},
	`you can cast (\d+) additional brands`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveBrandLimit", "BASE", num)}, ""
	},
	`(\d+)% increased damage while you are wielding a bow and have a totem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Tag(mod.Condition("HaveTotem"), mod.Condition("UsingBow"))}, ""
	},
	`bow attacks consume (\d+)% of your maximum life flask charges if possible to deal added physical damage equal to (\d+)% of flask's life recovery amount`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalMin", "BASE", 1).Flag(ModFlagAttack | ModFlagBow).Tag(mod.PercentStat("LifeFlaskRecovery", utils.Float(captures[1]))), MOD("PhysicalMax", "BASE", 1).Flag(ModFlagAttack | ModFlagBow).Tag(mod.PercentStat("LifeFlaskRecovery", utils.Float(captures[1])))}, ""
	},
	`each totem applies (\d+)% increased damage taken to enemies near it`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Multiplier("TotemsSummoned"))})}, ""
	},
	`each totem applies (\d+)% increased damage taken to enemies in their presence`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Multiplier("TotemsSummoned"))})}, ""
	},
	`totems gain \+(\d+)% to (\w+) resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Totem"+firstToUpper(captures[1])+"Resist", "BASE", num)}, ""
	},
	`totems gain \+(\d+)% to all elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TotemElementalResist", "BASE", num)}, ""
	},
	`totems gain \+(\d+)% to all maximum elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TotemElementalResistMax", "BASE", num)}, ""
	},
	`totems reserve (\d+) spirit each`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("AncestralBond"), MOD("ExtraSpirit", "BASE", num).Tag(mod.SkillType("SummonsTotem"))}, ""
	},
	`minions revive (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionRevivalSpeed", "INC", num)}, ""
	},
	"your strength is added to your minions":         []mod.Mod{FLAG("StrengthAddedToMinions")},
	"your dexterity is added to your minions":        []mod.Mod{FLAG("DexterityAddedToMinions")},
	"companions gain your dexterity":                 []mod.Mod{FLAG("DexterityAddedToCompanions")},
	"companions gain your strength":                  []mod.Mod{FLAG("StrengthAddedToCompanions")},
	"half of your strength is added to your minions": []mod.Mod{FLAG("HalfStrengthAddedToMinions")},
	"your minions are gigantic":                      []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Gigantic")})},
	`minions created recently have (\d+)% increased attack and cast speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Speed", "INC", num)}).Tag(mod.Condition("MinionsCreatedRecently"))}, ""
	},
	`minions created recently have (\d+)% increased movement speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("MovementSpeed", "INC", num)}).Tag(mod.Condition("MinionsCreatedRecently"))}, ""
	},
	"minions poison enemies on hit": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("PoisonChance", "BASE", 100)})},
	`minions have (\d+)% chance to poison enemies on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("PoisonChance", "BASE", num)})}, ""
	},
	`(\d+)% increased minion damage if you have hit recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num)}).Tag(mod.Condition("HitRecently"))}, ""
	},
	`(\d+)% increased minion damage if you've used a minion skill recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num)}).Tag(mod.Condition("UsedMinionSkillRecently"))}, ""
	},
	`minions deal (\d+)% increased damage if you've used a minion skill recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num)}).Tag(mod.Condition("UsedMinionSkillRecently"))}, ""
	},
	`minions have (\d+)% increased attack and cast speed if you or your minions have killed recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Speed", "INC", num)}).Tag(mod.Condition("KilledRecently", "MinionsKilledRecently"))}, ""
	},
	`minions gain (\d+)% of their maximum life as extra maximum energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnergyShield", "BASE", 1).Tag(mod.PercentStat("Life", num))})}, ""
	},
	`(\d+)% increased minion attack speed per (\d+) dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Speed", "INC", num).Flag(ModFlagAttack)}).Tag(mod.PerStat(utils.Float(captures[1]), "Dex"))}, ""
	},
	`(\d+)% increased minion movement speed per (\d+) dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("MovementSpeed", "INC", num)}).Tag(mod.PerStat(utils.Float(captures[1]), "Dex"))}, ""
	},
	`minions deal (\d+)% increased damage per (\d+) dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num)}).Tag(mod.PerStat(utils.Float(captures[1]), "Dex"))}, ""
	},
	"minions' resistances are equal to yours": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("FireResist", "OVERRIDE", 1).Tag(mod.PerStat(1, "FireResist").Actor("parent"))}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ColdResist", "OVERRIDE", 1).Tag(mod.PerStat(1, "ColdResist").Actor("parent"))}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("LightningResist", "OVERRIDE", 1).Tag(mod.PerStat(1, "LightningResist").Actor("parent"))}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChaosResist", "OVERRIDE", 1).Tag(mod.PerStat(1, "ChaosResist").Actor("parent"))})}, ""
	},
	`minions have (\d+)% chance to deal double damage while they are on full life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("DoubleDamageChance", "BASE", num).Tag(mod.Condition("FullLife"))})}, ""
	},
	`(\d+)% increased golem damage for each type of golem you have summoned`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HavePhysicalGolem"))}).Tag(mod.SkillType("Golem")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HaveLightningGolem"))}).Tag(mod.SkillType("Golem")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HaveColdGolem"))}).Tag(mod.SkillType("Golem")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HaveFireGolem"))}).Tag(mod.SkillType("Golem")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HaveChaosGolem"))}).Tag(mod.SkillType("Golem")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HaveCarrionGolem"))}).Tag(mod.SkillType("Golem"))}, ""
	},
	`can summon up to (\d) additional golems? at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveGolemLimit", "BASE", num)}, ""
	},
	`\+(\d) to maximum number of sentinels of purity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveSentinelOfPurityLimit", "BASE", num)}, ""
	},
	`if you have 3 primordial jewels, can summon up to (\d) additional golems? at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveGolemLimit", "BASE", num).Tag(mod.MultiplierThreshold("PrimordialItem").Threshold(3))}, ""
	},
	`golems regenerate (\d)% of their maximum life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("LifeRegenPercent", "BASE", num)}).Tag(mod.SkillType("Golem"))}, ""
	},
	`summoned golems regenerate (\d)% of their life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("LifeRegenPercent", "BASE", num)}).Tag(mod.SkillType("Golem"))}, ""
	},
	"summoned carrion golems impale on hit if you have the same number of them as summoned chaos golems": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ImpaleChance", "BASE", 100).Tag(mod.ActorCondition("parent", "CarrionEqualChaosGolem"), mod.ActorCondition("parent", "HaveChaosGolem"))}).Tag(mod.SkillName("Summon Carrion Golem").Transfigured(true))},
	"summoned chaos golems impale on hit if you have the same number of them as summoned stone golems":   []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ImpaleChance", "BASE", 100).Tag(mod.ActorCondition("parent", "ChaosEqualStoneGolem"), mod.ActorCondition("parent", "HavePhysicalGolem"))}).Tag(mod.SkillName("Summon Chaos Golem").Transfigured(true))},
	"summoned stone golems impale on hit if you have the same number of them as summoned carrion golems": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ImpaleChance", "BASE", 100).Tag(mod.ActorCondition("parent", "StoneEqualCarrionGolem"), mod.ActorCondition("parent", "HaveCarrionGolem"))}).Tag(mod.SkillName("Summon Stone Golem").Transfigured(true))},
	"maximum life of summoned elemental golems is doubled":                                               []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Life", "MORE", 100)}).Tag(mod.SkillType("Golem"), mod.SkillType("Lightning", "Cold", "Fire"))},
	`golems summoned in the past 8 seconds deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "SummonedGolemInPast8Sec"))}).Tag(mod.SkillType("Golem"))}, ""
	},
	"raised zombies and spectres gain adrenaline for 8 seconds when raised": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Condition:Adrenaline")}).Tag(mod.Condition("SummonedSpectreInPast8Sec"), mod.SkillName("Raise Spectre").Transfigured(true)), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Condition:Adrenaline")}).Tag(mod.Condition("SummonedZombieInPast8Sec"), mod.SkillName("Raise Zombie").Transfigured(true))},
	"gain onslaught for 10 seconds when you cast socketed golem skill": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Condition:Onslaught").Tag(mod.Condition("SummonedGolemInPast10Sec"))}, ""
	},
	"s?u?m?m?o?n?e?d? ?raging spirits' hits always ignite": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnemyIgniteChance", "BASE", 100)}).Tag(mod.SkillName("Summon Raging Spirit").Transfigured(true))},
	"raised zombies have avatar of fire":                   []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Keystone", "LIST", "Avatar of Fire")}).Tag(mod.SkillName("Raise Zombie").Transfigured(true))},
	`raised zombies take ([\d\.]+)% of their maximum life per second as fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("FireDegen", "BASE", 1).Tag(mod.PercentStat("Life", num))}).Tag(mod.SkillName("Raise Zombie").Transfigured(true))}, ""
	},
	`maximum number of summoned raging spirits is (\d+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveRagingSpiritLimit", "OVERRIDE", num)}, ""
	},
	`maximum number of summoned phantasms is (\d+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActivePhantasmLimit", "OVERRIDE", num)}, ""
	},
	"summoned raging spirits have diamond shrine and massive shrine buffs": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Condition:DiamondShrine")}).Tag(mod.SkillName("Summon Raging Spirit").Transfigured(true)), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Condition:MassiveShrine")}).Tag(mod.SkillName("Summon Raging Spirit").Transfigured(true))},
	"summoned phantasms have diamond shrine and massive shrine buffs":      []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Condition:DiamondShrine")}).Tag(mod.SkillName("Summon Phantasm")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Condition:MassiveShrine")}).Tag(mod.SkillName("Summon Phantasm"))},
	`minions deal no non\-physical damage`:                                 []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoLightning")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoCold")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoFire")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoChaos")})},
	`minions deal no non\-lightning damage`:                                []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoPhysical")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoLCold")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoFire")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoChaos")})},
	`minions deal no non\-cold damage`:                                     []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoPhysical")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoLightning")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoFire")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoChaos")})},
	`minions deal no non\-fire damage`:                                     []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoPhysical")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoLightning")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoCold")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoChaos")})},
	`minions deal no non\-chaos damage`:                                    []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoPhysical")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoLightning")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoCold")}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("DealNoFire")})},
	`minions convert (\d+)% of (.+) damage to (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD(firstToUpper(captures[1])+"DamageConvertTo"+firstToUpper(captures[2]), "BASE", num)})}, ""
	},
	"summoned skeletons have avatar of fire": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Keystone", "LIST", "Avatar of Fire")}).Tag(mod.SkillName("Summon Skeletons").Transfigured(true))},
	`summoned skeletons take ([\d\.]+)% of their maximum life per second as fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("FireDegen", "BASE", 1).Tag(mod.PercentStat("Life", num))}).Tag(mod.SkillName("Summon Skeletons").Transfigured(true))}, ""
	},
	`summoned skeletons have (\d+)% chance to wither enemies for (\d+) seconds on hit`: []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: FLAG("Condition:CanWither")}).Tag(mod.SkillName("Summon Skeletons").Transfigured(true))},
	`summoned skeletons have (\d+)% of physical damage converted to chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("PhysicalDamageConvertToChaos", "BASE", num)}).Tag(mod.SkillName("Summon Skeletons").Transfigured(true))}, ""
	},
	`summoned skeletons gain added chaos damage equal to (\d+)% of maximum energy shield on your equipped shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnWeapon 2", num).Actor("parent"))}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnWeapon 2", num).Actor("parent"))})}, ""
	},
	`skeletons gain added chaos damage equal to (\d+)% of maximum energy shield on your equipped shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnWeapon 2", num).Actor("parent"))}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnWeapon 2", num).Actor("parent"))})}, ""
	},
	`minions convert (\d+)% of (.+) damage to (.+) damage per (.+) socket`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD(firstToUpper(captures[1])+"DamageConvertTo"+firstToUpper(captures[2]), "BASE", num)}).Tag(mod.Multiplier(firstToUpper(captures[3]) + "SocketIn{SlotName}"))}, ""
	},
	`minions have a (\d+)% chance to impale on hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ImpaleChance", "BASE", num)})}, ""
	},
	`minions from herald skills deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "MORE", num)}).Tag(mod.SkillType("Herald"))}, ""
	},
	`minions have (\d+)% increased movement speed for each herald affecting you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("MovementSpeed", "INC", num).Tag(mod.Multiplier("Herald").Actor("parent"))})}, ""
	},
	`minions deal (\d+)% increased damage while you are affected by a herald`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "AffectedByHerald"))})}, ""
	},
	`minions have (\d+)% increased attack and cast speed while you are affected by a herald`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Speed", "INC", num).Tag(mod.ActorCondition("parent", "AffectedByHerald"))})}, ""
	},
	"minions have unholy might": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Condition:UnholyMight")})},
	"summoned skeleton warriors a?n?d? ?s?o?l?d?i?e?r?s? ?deal triple damage with this weapon if you've hit with this weapon recently": []mod.Mod{MOD("Dummy", "DUMMY", 1).Tag(mod.Condition("HitRecentlyWithWeapon")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("TripleDamageChance", "BASE", 100).Tag(mod.ActorCondition("parent", "HitRecentlyWithWeapon"))}).Tag(mod.SkillName("Summon Skeletons"))},
	"summoned skeleton warriors a?n?d? ?s?o?l?d?i?e?r?s? ?wield a? ?c?o?p?y? ?o?f? ?this weapon while in your main hand":               []mod.Mod{},
	"each summoned phantasm grants you phantasmal might":                                                                               []mod.Mod{FLAG("Condition:PhantasmalMight")},
	`minions have (\d+)% increased critical hit chance per maximum power charge you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("CritChance", "INC", num).Tag(mod.Multiplier("PowerChargeMax").Actor("parent"))})}, ""
	},
	"minions' base attack critical hit chance is equal to the critical hit chance of your main hand weapon": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("AttackCritIsEqualToParentMainHand").Flag(ModFlagAttack)})},
	"minions can hear the whispers for 5 seconds after they deal a critical hit":                            []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", 50).Tag(mod.Condition("NeverCrit").Neg(true))}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Speed", "INC", 50).Flag(ModFlagAttack).Tag(mod.Condition("NeverCrit").Neg(true))}), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChaosDegen", "BASE", 1).Tag(mod.PercentStat("Life", 20), mod.Condition("NeverCrit").Neg(true))})},
	"chaos damage t?a?k?e?n? ?does not bypass minions' energy shield":                                       []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("ChaosNotBypassEnergyShield")})},
	"while minions have energy shield, their hits ignore monster elemental resistances":                     []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("IgnoreElementalResistances").Tag(mod.StatThreshold("EnergyShield", 1))})},
	`summoned arbalists' projectiles pierce (\d+) additional targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("PierceCount", "BASE", num)}).Tag(mod.SkillName("Summon Arbalists"))}, ""
	},
	"summoned arbalists' projectiles fork": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("ForkOnce")}).Tag(mod.SkillName("Summon Arbalists")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ForkCountMax", "BASE", 1)}).Tag(mod.SkillName("Summon Arbalists"))},
	`summoned arbalists' projectiles chain \+(\d+) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChainCountMax", "BASE", num)}).Tag(mod.SkillName("Summon Arbalists"))}, ""
	},
	`summoned arbalists have (\d+)% chance to inflict (.+) exposure on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[1])+"ExposureChance", "BASE", num).Tag(mod.SkillName("Summon Arbalists"))}, ""
	},
	`summoned arbalists convert (\d+)% of (.+) damage to (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD(firstToUpper(captures[1])+"DamageConvertTo"+firstToUpper(captures[2]), "BASE", num)}).Tag(mod.SkillName("Summon Arbalists"))}, ""
	},
	`summoned arbalists have (\d+)% chance to freeze, shock, and ignite`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnemyFreezeChance", "BASE", num)}).Tag(mod.SkillName("Summon Arbalists")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnemyShockChance", "BASE", num)}).Tag(mod.SkillName("Summon Arbalists")), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnemyIgniteChance", "BASE", num)}).Tag(mod.SkillName("Summon Arbalists"))}, ""
	},
	"skeleton warriors are permanent minions and follow you":  []mod.Mod{FLAG("RaisedSkeletonPermanentDuration").Tag(mod.SkillName("Summon Skeletons"))},
	"summoned skeleton warriors are permanent and follow you": []mod.Mod{FLAG("RaisedSkeletonPermanentDuration").Tag(mod.SkillName("Summon Skeletons"))},
	`minions recoup (\d+)% of damage taken as life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("LifeRecoup", "BASE", num)})}, ""
	},
	`temporary minion skills have \+(\d+) to limit of minions summoned`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveMinionLimit", "BASE", num).Tag(mod.BaseFlag("duration").Neg(true))}, ""
	},
	`every five rage also grants you (\d+)% increased movement speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeed", "INC", num).Tag(mod.PerStat(5, "Rage"))}, ""
	},
	`every rage also grants you (\d+)% increased minion damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.Multiplier("RageEffect").Actor("player"))})}, ""
	},
	`every rage also grants you (\d+)% increased minion attack speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Speed", "INC", num).Flag(ModFlagAttack).Tag(mod.Multiplier("RageEffect").Actor("player"))})}, ""
	},
	`skills chain \+(\d) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChainCountMax", "BASE", num)}, ""
	},
	`arrows chain \+(\d) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChainCountMax", "BASE", num).KeywordFlag(KeywordFlagArrow)}, ""
	},
	"skills chain an additional time while at maximum frenzy charges": []mod.Mod{MOD("ChainCountMax", "BASE", 1).Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	"attacks chain an additional time when in main hand":              []mod.Mod{MOD("ChainCountMax", "BASE", 1).Flag(ModFlagAttack).Tag(mod.SlotNumber(int(1)))},
	"attacks chain an additional time":                                []mod.Mod{MOD("ChainCountMax", "BASE", 1).Flag(ModFlagAttack)},
	`attacks chain (\d+) additional times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChainCountMax", "BASE", num).Flag(ModFlagAttack)}, ""
	},
	`projectiles chain \+(\d) times while you have phasing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChainCountMax", "BASE", num).Flag(ModFlagProjectile).Tag(mod.Condition("Phasing"))}, ""
	},
	`projectiles split towards \+(\d) targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SplitCount", "BASE", num)}, ""
	},
	"adds an additional arrow": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(ModFlagAttack)},
	`(\d+) additional arrows`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).KeywordFlag(KeywordFlagArrow)}, ""
	},
	"bow attacks fire an additional arrow": []mod.Mod{MOD("ProjectileCount", "BASE", 1).KeywordFlag(KeywordFlagArrow)},
	`\+(\d+)% surpassing chance to fire an additional arrow`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SurpassingProjectileChance", "BASE", num).KeywordFlag(KeywordFlagArrow)}, ""
	},
	`\+(\d+)% surpassing chance to fire an additional projectile`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SurpassingProjectileChance", "BASE", num)}, ""
	},
	`bow attacks fire (\d+) additional arrows`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).KeywordFlag(KeywordFlagArrow)}, ""
	},
	`bow attacks fire (\d+) additional arrows if you haven't cast dash recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).KeywordFlag(KeywordFlagArrow).Tag(mod.Condition("CastDashRecently").Neg(true))}, ""
	},
	"wand attacks fire an additional projectile": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(ModFlagWand)},
	"loads? an additional bolt":                  []mod.Mod{MOD("CrossbowBoltCount", "BASE", 1).Flag(ModFlagCrossbow).Tag(mod.SkillType("CrossbowSkill"))},
	`loads? (\d+) additional bolts?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CrossbowBoltCount", "BASE", num).Flag(ModFlagCrossbow).Tag(mod.SkillType("CrossbowSkill"))}, ""
	},
	"grenade skills fire an additional projectile": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(ModFlagProjectile).Tag(mod.SkillType("Grenade"))},
	`grenades have (\d+)% chance to activate a second time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GrenadeActivateTwice", "BASE", num).Tag(mod.SkillType("Grenade"))}, ""
	},
	"skills fire an additional projectile":                   []mod.Mod{MOD("ProjectileCount", "BASE", 1)},
	"spells [hf][ai][vr]e an additional projectile":          []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(ModFlagSpell)},
	"attacks fire an additional projectile":                  []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(ModFlagAttack)},
	"attacks have an additional projectile when in off hand": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(ModFlagAttack).Tag(mod.SlotNumber(int(2)))},
	// PoB passes nil before the tags, so they land at [2] and later and ModStore's ipairs never reads them.
	`caustic arrow and scourge arrow fire (\d+)% more projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "MORE", num)}, ""
	},
	`essence drain and soulrend fire (\d+) additional projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num)}, ""
	},
	`(\d+)% reduced essence drain and soulrend projectile speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileSpeed", "INC", -num)}, ""
	},
	"tornado shot fires an additional secondary projectile": []mod.Mod{MOD("tornadoShotSecondaryProjectiles", "BASE", 1)},
	"projectiles pierce an additional target":               []mod.Mod{MOD("PierceCount", "BASE", 1)},
	"projectiles pierce enemies with fully broken armour":   []mod.Mod{MOD("PierceCount", "BASE", 1).Tag(mod.Condition("ArmourFullyBroken"))},
	`projectiles pierce (\d+) targets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PierceCount", "BASE", num)}, ""
	},
	`projectiles pierce (\d+) additional targets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PierceCount", "BASE", num)}, ""
	},
	`projectiles pierce (\d+) additional targets while you have phasing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PierceCount", "BASE", num).Tag(mod.Condition("Phasing"))}, ""
	},
	"projectiles pierce all targets while you have phasing": []mod.Mod{FLAG("PierceAllTargets").Tag(mod.Condition("Phasing"))},
	"projectiles pierce all burning enemies":                []mod.Mod{FLAG("PierceAllTargets").Tag(mod.ActorCondition("enemy", "Burning"))},
	"arrows pierce an additional target":                    []mod.Mod{MOD("PierceCount", "BASE", 1).KeywordFlag(KeywordFlagArrow)},
	`arrows pierce (\d+) additional targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PierceCount", "BASE", num).KeywordFlag(KeywordFlagArrow)}, ""
	},
	"arrows pierce one target": []mod.Mod{MOD("PierceCount", "BASE", 1).KeywordFlag(KeywordFlagArrow)},
	`arrows pierce (\d+) targets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PierceCount", "BASE", num).KeywordFlag(KeywordFlagArrow)}, ""
	},
	"always pierce with arrows":         []mod.Mod{FLAG("PierceAllTargets").KeywordFlag(KeywordFlagArrow)},
	"arrows always pierce":              []mod.Mod{FLAG("PierceAllTargets").KeywordFlag(KeywordFlagArrow)},
	"arrows pierce all targets":         []mod.Mod{FLAG("PierceAllTargets").KeywordFlag(KeywordFlagArrow)},
	"arrows that pierce cause bleeding": []mod.Mod{MOD("BleedChance", "BASE", 100).KeywordFlag(KeywordFlagArrow).Tag(mod.StatThreshold("PierceCount", 1))},
	`arrows that pierce have (\d+)% chance to cause bleeding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BleedChance", "BASE", num).KeywordFlag(KeywordFlagArrow).Tag(mod.StatThreshold("PierceCount", 1))}, ""
	},
	`arrows that pierce deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(KeywordFlagArrow).Tag(mod.StatThreshold("PierceCount", 1))}, ""
	},
	`projectiles gain (\d+)% of non\-chaos damage as extra chaos damage per chain`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("NonChaosDamageGainAsChaos", "BASE", num).Flag(ModFlagProjectile).Tag(mod.PerStat(1, "Chain"))}, ""
	},
	`projectiles that have chained gain (\d+)% of non\-chaos damage as extra chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("NonChaosDamageGainAsChaos", "BASE", num).Flag(ModFlagProjectile).Tag(mod.StatThreshold("Chain", 1))}, ""
	},
	"left ring slot: projectiles from spells cannot chain":                                    []mod.Mod{FLAG("CannotChain").Flag(ModFlagSpell | ModFlagProjectile).Tag(mod.SlotNumber(int(1)))},
	"left ring slot: projectiles from spells fork":                                            []mod.Mod{FLAG("ForkOnce").Flag(ModFlagSpell | ModFlagProjectile).Tag(mod.SlotNumber(int(1))), MOD("ForkCountMax", "BASE", 1).Flag(ModFlagSpell | ModFlagProjectile).Tag(mod.SlotNumber(int(1)))},
	"left ring slot: your chilling skitterbot's aura applies socketed h?e?x? ?curse instead":  []mod.Mod{FLAG("SkitterbotsCannotChill").Tag(mod.SlotNumber(int(1)))},
	`right ring slot: projectiles from spells chain \+1 times`:                                []mod.Mod{MOD("ChainCountMax", "BASE", 1).Flag(ModFlagSpell | ModFlagProjectile).Tag(mod.SlotNumber(int(2)))},
	"right ring slot: projectiles from spells cannot fork":                                    []mod.Mod{FLAG("CannotFork").Flag(ModFlagSpell | ModFlagProjectile).Tag(mod.SlotNumber(int(2)))},
	"right ring slot: your shocking skitterbot's aura applies socketed h?e?x? ?curse instead": []mod.Mod{FLAG("SkitterbotsCannotShock").Tag(mod.SlotNumber(int(2)))},
	"projectiles from spells cannot pierce":                                                   []mod.Mod{FLAG("CannotPierce").Flag(ModFlagSpell)},
	"projectiles fork":                                                                        []mod.Mod{FLAG("ForkOnce").Flag(ModFlagProjectile), MOD("ForkCountMax", "BASE", 1).Flag(ModFlagProjectile)},
	"arrows fork":                                                                             []mod.Mod{FLAG("ForkOnce").KeywordFlag(KeywordFlagArrow), MOD("ForkCountMax", "BASE", 1).Flag(ModFlagProjectile)},
	"projectiles from attacks fork":                                                           []mod.Mod{FLAG("ForkOnce").Flag(ModFlagProjectile).Tag(mod.SkillType("RangedAttack")), MOD("ForkCountMax", "BASE", 1).Flag(ModFlagProjectile).Tag(mod.SkillType("RangedAttack"))},
	"projectiles from attacks fork an additional time":                                        []mod.Mod{FLAG("ForkTwice").Flag(ModFlagProjectile).Tag(mod.SkillType("RangedAttack")), MOD("ForkCountMax", "BASE", 1).Flag(ModFlagProjectile).Tag(mod.SkillType("RangedAttack"))},
	`projectiles from attacks can fork (\d+) additional times?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("ForkTwice").Flag(ModFlagProjectile).Tag(mod.SkillType("RangedAttack")), MOD("ForkCountMax", "BASE", num).Flag(ModFlagProjectile).Tag(mod.SkillType("RangedAttack"))}, ""
	},
	`(\d+)% increased critical hit chance with arrows that fork`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChance", "INC", num).KeywordFlag(KeywordFlagArrow).Tag(mod.StatThreshold("ForkRemaining", 1), mod.StatThreshold("PierceCount", 0).Upper(true))}, ""
	},
	`arrows that pierce have \+(\d+)% to critical damage bonus`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritMultiplier", "BASE", num).KeywordFlag(KeywordFlagArrow).Tag(mod.StatThreshold("PierceCount", 1))}, ""
	},
	"arrows pierce all targets after forking":                                                             []mod.Mod{FLAG("PierceAllTargets").KeywordFlag(KeywordFlagArrow).Tag(mod.StatThreshold("ForkedCount", 1))},
	"modifiers to number of projectiles instead apply to the number of targets projectiles split towards": []mod.Mod{FLAG("NoAdditionalProjectiles"), FLAG("AdditionalProjectilesAddSplitsInstead")},
	"modifiers to number of projectiles do not apply to fireball and rolling magma":                       []mod.Mod{FLAG("NoAdditionalProjectiles").Tag(mod.SkillName("Fireball", "Rolling Magma"))},
	"attack skills fire an additional projectile while wielding a claw or dagger":                         []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(ModFlagAttack).Tag(mod.ModFlagOr(ModFlagClaw | ModFlagDagger))},
	`skills fire (\d+) additional projectiles for 4 seconds after you consume a total of 12 steel shards`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).Tag(mod.Condition("Consumed12SteelShardsRecently"))}, ""
	},
	`non\-projectile chaining lightning skills chain \+(\d+) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChainCountMax", "BASE", num).Tag(mod.SkillType("Projectile").Neg(true), mod.SkillType("Chains"), mod.SkillType("Lightning"))}, ""
	},
	`arrows gain damage as they travel farther, dealing up to (\d+)% increased damage with hits to targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Flag(ModFlagHit).KeywordFlag(KeywordFlagArrow).Tag(mod.DistanceRamp([][]float64{{35, 0}, {70, 1}}))}, ""
	},
	`arrows gain critical hit chance as they travel farther, up to (\d+)% increased critical hit chance after 7 metres`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChance", "INC", num).KeywordFlag(KeywordFlagArrow).Tag(mod.DistanceRamp([][]float64{{35, 0}, {70, 1}}))}, ""
	},
	`projectiles deal (\d+)% increased damage with hits to targets at the start of their movement, reducing to (\d+)% as they travel farther`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Flag(ModFlagHit | ModFlagProjectile).Tag(mod.DistanceRamp([][]float64{{35, 1}, {70, 0}}))}, ""
	},
	`projectiles deal (\d+)% increased damage with hits and ailments for each time they have chained`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(KeywordFlagHit|KeywordFlagAilment).Tag(mod.PerStat(1, "Chain"), mod.SkillType("Projectile"))}, ""
	},
	`projectiles deal (\d+)% increased damage with hits and ailments for each enemy pierced`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(KeywordFlagHit|KeywordFlagAilment).Tag(mod.PerStat(1, "PiercedCount"), mod.SkillType("Projectile"))}, ""
	},
	`(\d+)% increased bonuses gained from equipped quiver`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EffectOfBonusesFromQuiver", "INC", num)}, ""
	},
	`(\d+)% reduced bonuses gained from equipped focus`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EffectOfBonusesFromFocus", "INC", -num)}, ""
	},
	`(\d+)% increased bonuses gained from equipped rings`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EffectOfBonusesFromRing 1", "INC", num), MOD("EffectOfBonusesFromRing 2", "INC", num), MOD("EffectOfBonusesFromRing 3", "INC", num)}, ""
	},
	`(\d+)% increased bonuses gained from equipped rings and amulets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EffectOfBonusesFromRing 1", "INC", num), MOD("EffectOfBonusesFromRing 2", "INC", num), MOD("EffectOfBonusesFromRing 3", "INC", num), MOD("EffectOfBonusesFromAmulet", "INC", num)}, ""
	},
	`(\d+)% increased bonuses gained from ([lr][ei][fg][th]t?) equipped ring`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EffectOfBonusesFromRing"+func() string {
			if captures[1] == "left" {
				return " 1"
			}
			return " 2"
		}(), "INC", num)}, ""
	},
	`(\d+)% chance for spell skills to fire 2 additional projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TwoAdditionalProjectilesChance", "BASE", num).Flag(ModFlagSpell)}, ""
	},
	`non\-vaal strike skills target (\d+) additional nearby enem[yi]e?s?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AdditionalStrikeTarget", "BASE", num).Tag(mod.SkillType("MeleeSingleTarget"), mod.SkillType("Vaal").Neg(true))}, ""
	},
	"cannot leech life":                                                                       []mod.Mod{FLAG("CannotLeechLife")},
	"cannot leech mana":                                                                       []mod.Mod{FLAG("CannotLeechMana")},
	"cannot leech when on low life":                                                           []mod.Mod{FLAG("CannotLeechLife").Tag(mod.Condition("LowLife")), FLAG("CannotLeechMana").Tag(mod.Condition("LowLife"))},
	"cannot leech life from critical hits":                                                    []mod.Mod{FLAG("CannotLeechLife").Tag(mod.Condition("CriticalStrike"))},
	"leech applies instantly on critical hit":                                                 []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike")), MOD("InstantManaLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike")), MOD("InstantEnergyShieldLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike"))},
	"leech from critical hits is instant":                                                     []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike")), MOD("InstantManaLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike")), MOD("InstantEnergyShieldLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike"))},
	"gain life and mana from leech instantly on critical hit":                                 []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike")), MOD("InstantManaLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike")), MOD("InstantEnergyShieldLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike"))},
	"leech applies instantly during f?l?a?s?k? ?effect":                                       []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Tag(mod.Condition("UsingFlask")), MOD("InstantManaLeech", "BASE", 100).Tag(mod.Condition("UsingFlask")), MOD("InstantEnergyShieldLeech", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"gain life and mana from leech instantly during f?l?a?s?k? ?effect":                       []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Tag(mod.Condition("UsingFlask")), MOD("InstantManaLeech", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"life and mana leech are instant during f?l?a?s?k? ?effect":                               []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Tag(mod.Condition("UsingFlask")), MOD("InstantManaLeech", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"life and mana leech from critical hits are instant":                                      []mod.Mod{MOD("InstantLifeLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike")), MOD("InstantManaLeech", "BASE", 100).Tag(mod.Condition("CriticalStrike"))},
	"with 5 corrupted items equipped: life leech recovers based on your chaos damage instead": []mod.Mod{FLAG("LifeLeechBasedOnChaosDamage").Tag(mod.MultiplierThreshold("CorruptedItem").Threshold(5)), FLAG("Condition:NoLifeLeechFromPhysicalDamage").Tag(mod.MultiplierThreshold("CorruptedItem").Threshold(5))},
	"you have vaal pact if you've dealt a critical hit recently":                              []mod.Mod{MOD("Keystone", "LIST", "Vaal Pact").Tag(mod.Condition("CritRecently"))},
	"you have vaal pact while at maximum endurance charges":                                   []mod.Mod{MOD("Keystone", "LIST", "Vaal Pact").Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax"))},
	"you have vaal pact while focus?sed":                                                      []mod.Mod{MOD("Keystone", "LIST", "Vaal Pact").Tag(mod.Condition("Focused"))},
	`gain (\d+) energy shield for each enemy you hit which is affected by a spider's web`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnHit", "BASE", num).Flag(ModFlagHit).Tag(mod.MultiplierThreshold("Spider's WebStack").Threshold(1).Actor("enemy"))}, ""
	},
	`(\d+)% chance to gain (\d+) life on hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnHit", "BASE", ((utils.Float(captures[1]) * num) / 100)).Flag(ModFlagAttack | ModFlagHit).Tag(mod.Condition("AverageResourceGain")), MOD("LifeOnHit", "BASE", utils.Float(captures[1])).Flag(ModFlagAttack | ModFlagHit).Tag(mod.Condition("MaxResourceGain"))}, ""
	},
	`(\d+) life gained for each cursed enemy hit by your attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnHit", "BASE", num).Flag(ModFlagAttack | ModFlagHit).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`gain (\d+) life per cursed enemy hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnHit", "BASE", num).Flag(ModFlagAttack | ModFlagHit).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`(\d+) mana gained for each cursed enemy hit by your attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnHit", "BASE", num).Flag(ModFlagAttack | ModFlagHit).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`gain (\d+) mana per cursed enemy hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnHit", "BASE", num).Flag(ModFlagAttack | ModFlagHit).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`gain (\d+) life per blinded enemy hit with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnHit", "BASE", num).Flag(ModFlagHit).Tag(mod.ActorCondition("enemy", "Blinded"), mod.Condition("{Hand}Attack"))}, ""
	},
	`recover (\d+)% of ?m?a?x?i?m?u?m? life on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num))}, ""
	},
	`recover (\d+)% of life on kill for each different type of mastery you have allocated`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.Multiplier("AllocatedMasteryType"))}, ""
	},
	`recover (\d+)% of life on killing a poisoned enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.MultiplierThreshold("PoisonStack").Threshold(1).Actor("enemy"))}, ""
	},
	`recover (\d+)% of life on killing a chilled enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.ActorCondition("enemy", "Chilled"))}, ""
	},
	`recover (\d+)% of life when you kill a cursed enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`recover (\d+)% of life per withered debuff on each enemy you kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.Multiplier("WitheredStack").Limit(15).Actor("enemy"))}, ""
	},
	`minions recover (\d+)% of life on killing a poisoned enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.MultiplierThreshold("PoisonStack").Threshold(1).Actor("enemy"))})}, ""
	},
	`recover (\d+)% of mana when you kill a cursed enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnKill", "BASE", 1).Tag(mod.PercentStat("Mana", num), mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`recover (\d+)% of energy shield when you kill a cursed enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnKill", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num), mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`recover (\d+)% of life on kill if you've spent life recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.MultiplierThreshold("LifeSpentRecently").Threshold(1))}, ""
	},
	`(\d+)% chance to recover all life when you kill an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.Condition("AverageResourceGain")), MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", 100), mod.Condition("MaxResourceGain"))}, ""
	},
	`lose (\d+)% of ?m?a?x?i?m?u?m? life on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", -1).Tag(mod.PercentStat("Life", num))}, ""
	},
	`\+(\d+) life gained on killing ignited enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", num).Tag(mod.ActorCondition("enemy", "Ignited"))}, ""
	},
	`gain (\d+) life per ignited enemy killed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", num).Tag(mod.ActorCondition("enemy", "Ignited"))}, ""
	},
	`recover (\d+)% of ?m?a?x?i?m?u?m? mana on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnKill", "BASE", 1).Tag(mod.PercentStat("Mana", num))}, ""
	},
	`recover (\d+)% of mana on kill for each different type of mastery you have allocated`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnKill", "BASE", 1).Tag(mod.PercentStat("Mana", num), mod.Multiplier("AllocatedMasteryType"))}, ""
	},
	`lose (\d+)% of mana on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnKill", "BASE", -1).Tag(mod.PercentStat("Mana", num))}, ""
	},
	`\+(\d+) mana gained on killing a frozen enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnKill", "BASE", num).Tag(mod.ActorCondition("enemy", "Frozen"))}, ""
	},
	`recover (\d+)% of energy shield on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnKill", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num))}, ""
	},
	`recover (\d+)% of energy shield on kill for each different type of mastery you have allocated`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnKill", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num), mod.Multiplier("AllocatedMasteryType"))}, ""
	},
	`lose (\d+)% of energy shield on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnKill", "BASE", -1).Tag(mod.PercentStat("EnergyShield", num))}, ""
	},
	`\+(\d+) energy shield gained on killing a shocked enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnKill", "BASE", num).Tag(mod.ActorCondition("enemy", "Shocked"))}, ""
	},
	`\+(\d+) energy shield gained on kill per level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnKill", "BASE", num).Tag(mod.Multiplier("Level"))}, ""
	},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield": []mod.Mod{FLAG("ChaosNotBypassEnergyShield")},
	`(\d+)% of chaos damage t?a?k?e?n? ?does not bypass energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosEnergyShieldBypass", "BASE", -num)}, ""
	},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life":             []mod.Mod{FLAG("ChaosNotBypassEnergyShield").Tag(mod.Condition("LowLife").Neg(true))},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life or low mana": []mod.Mod{FLAG("ChaosNotBypassEnergyShield").Tag(mod.Condition("LowLife", "LowMana").Neg(true))},
	"chaos damage is taken from mana before life":                                              []mod.Mod{MOD("ChaosDamageTakenFromManaBeforeLife", "BASE", 100)},
	`minions take (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("DamageTaken", "INC", num)})}, ""
	},
	`minions take (\d+)% reduced damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("DamageTaken", "INC", -num)})}, ""
	},
	`you and your minions take (\d+)% reduced reflected (\w+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[1])+"ReflectedDamageTaken", "INC", -num).Tag(mod.GlobalEffect("Global").Unscalable(true)), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD(firstToUpper(captures[1])+"ReflectedDamageTaken", "INC", -num).Tag(mod.GlobalEffect("Global").Unscalable(true))})}, ""
	},
	`you and your minions take (\d+)% reduced reflected damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReflectedDamageTaken", "INC", -num).Tag(mod.GlobalEffect("Global").Unscalable(true)), MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ReflectedDamageTaken", "INC", -num).Tag(mod.GlobalEffect("Global").Unscalable(true))})}, ""
	},
	"damage cannot be reflected": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReflectedDamageTaken", "MORE", -100).Tag(mod.GlobalEffect("Global").Unscalable(true))}, ""
	},
	"you have mind over matter while at maximum power charges": []mod.Mod{MOD("Keystone", "LIST", "Mind Over Matter").Tag(mod.StatThresholdStat("PowerCharges", "PowerChargesMax"))},
	"cannot evade enemy attacks":                               []mod.Mod{FLAG("CannotEvade")},
	"attacks cannot hit you":                                   []mod.Mod{FLAG("AlwaysEvade")},
	"attacks against you always hit":                           []mod.Mod{FLAG("CannotEvade")},
	`maximum chance to evade is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EvadeChanceMax", "MAX", num)}, ""
	},
	"chance to evade is unlucky": []mod.Mod{FLAG("UnluckyEvade")},
	"modifiers to maximum block chance instead apply to maximum resistances": []mod.Mod{FLAG("MaxBlockChanceModsApplyMaxResist")},
	"you cannot block": []mod.Mod{FLAG("CannotBlockAttacks")},
	"cannot block":     []mod.Mod{FLAG("CannotBlockAttacks"), FLAG("CannotBlockSpells")},
	"cannot block while you have no energy shield":                        []mod.Mod{FLAG("CannotBlockAttacks").Tag(mod.Condition("HaveEnergyShield").Neg(true)), FLAG("CannotBlockSpells").Tag(mod.Condition("HaveEnergyShield").Neg(true))},
	"cannot block attacks":                                                []mod.Mod{FLAG("CannotBlockAttacks")},
	"cannot block attack damage":                                          []mod.Mod{FLAG("CannotBlockAttacks")},
	"cannot block spells":                                                 []mod.Mod{FLAG("CannotBlockSpells")},
	"cannot block spell damage":                                           []mod.Mod{FLAG("CannotBlockSpells")},
	"monsters cannot block your attacks":                                  []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("CannotBlockAttacks")})},
	"damage t?a?k?e?n? from blocked hits cannot bypass energy shield":     []mod.Mod{FLAG("BlockedDamageDoesntBypassES").Tag(mod.Condition("EVBypass").Neg(true))},
	"damage t?a?k?e?n? from unblocked hits always bypasses energy shield": []mod.Mod{FLAG("UnblockedDamageDoesBypassES").Tag(mod.Condition("EVBypass").Neg(true))},
	`recover (\d+) life when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnBlock", "BASE", num)}, ""
	},
	`recover (\d+) runic ward when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("WardOnBlock", "BASE", num)}, ""
	},
	`recover (\d+) energy shield when you block spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnSpellBlock", "BASE", num)}, ""
	},
	`recover (\d+) energy shield when you suppress spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnSuppress", "BASE", num)}, ""
	},
	`recover (\d+) life when you suppress spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnSuppress", "BASE", num)}, ""
	},
	`recover (\d+)% of  ?m?a?x?i?m?u?m? life when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnBlock", "BASE", 1).Tag(mod.PercentStat("Life", num))}, ""
	},
	`recover (\d+)% of  ?m?a?x?i?m?u?m? life when you block attack damage while wielding a staff`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnBlock", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.Condition("UsingStaff"))}, ""
	},
	`recover (\d+)% of your maximum mana when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnBlock", "BASE", 1).Tag(mod.PercentStat("Mana", num))}, ""
	},
	`recover (\d+)% of energy shield when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnBlock", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num))}, ""
	},
	`recover (\d+)% of energy shield when you block spell damage while wielding a staff`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnSpellBlock", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num), mod.Condition("UsingStaff"))}, ""
	},
	`replenishes energy shield by (\d+)% of armour when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnBlock", "BASE", 1).Tag(mod.PercentStat("Armour", num))}, ""
	},
	`recover energy shield equal to (\d+)% of armour when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnBlock", "BASE", 1).Tag(mod.PercentStat("Armour", num))}, ""
	},
	`(\d+)% of damage taken while affected by clarity recouped as mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaRecoup", "BASE", num).Tag(mod.Condition("AffectedByClarity"))}, ""
	},
	"recoup effects instead occur over 4 seconds":      []mod.Mod{FLAG("4SecondRecoup")},
	"life recoup effects instead occur over 4 seconds": []mod.Mod{FLAG("4SecondLifeRecoup")},
	`damage taken recouped as (\w+) is also recouped as energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Add" + firstToUpper(captures[0]) + "RecoupToEnergyShieldRecoup")}, ""
	},
	`([\d\).]+)% of physical damage prevented from hits in the past (\d+) seconds is regenerated as life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageMitigatedLifePseudoRecoup", "BASE", (num * utils.Float(captures[1]))), MOD("PhysicalDamageMitigatedLifePseudoRecoupDuration", "BASE", utils.Float(captures[1]))}, ""
	},
	`([\d\.]+)% of physical damage prevented from hits recently is regenerated as energy shield per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageMitigatedEnergyShieldPseudoRecoup", "BASE", (num * 4))}, ""
	},
	"cannot leech or regenerate mana":                                     []mod.Mod{FLAG("NoManaRegen"), FLAG("CannotLeechMana")},
	"right ring slot: you cannot regenerate mana":                         []mod.Mod{FLAG("NoManaRegen").Tag(mod.SlotNumber(int(2)))},
	"cannot regenerate mana if you haven't dealt a critical hit recently": []mod.Mod{FLAG("NoManaRegen").Tag(mod.Condition("CritRecently").Neg(true))},
	"y?o?u? ?cannot recharge energy shield":                               []mod.Mod{FLAG("NoEnergyShieldRecharge")},
	"energy shield does not recharge":                                     []mod.Mod{FLAG("NoEnergyShieldRecharge")},
	"you cannot regenerate energy shield":                                 []mod.Mod{FLAG("NoEnergyShieldRegen")},
	"cannot recharge or regenerate energy shield":                         []mod.Mod{FLAG("NoEnergyShieldRecharge"), FLAG("NoEnergyShieldRegen")},
	`enemies you (\w+) cannot recharge energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("NoEnergyShieldRecharge")}).Tag(mod.Condition(firstToUpper(captures[0])))}, ""
	},
	"left ring slot: you cannot recharge or regenerate energy shield": []mod.Mod{FLAG("NoEnergyShieldRecharge").Tag(mod.SlotNumber(int(1))), FLAG("NoEnergyShieldRegen").Tag(mod.SlotNumber(int(1)))},
	"cannot gain energy shield":                                       []mod.Mod{FLAG("CannotGainEnergyShield")},
	"cannot gain life":                                                []mod.Mod{FLAG("CannotGainLife")},
	"cannot gain mana":                                                []mod.Mod{FLAG("CannotGainMana")},
	"cannot recover life other than from leech":                       []mod.Mod{FLAG("CannotRecoverLifeOutsideLeech"), FLAG("NoLifeRegen")},
	"cannot gain energy shield during f?l?a?s?k? ?effect":             []mod.Mod{FLAG("CannotGainEnergyShield").Tag(mod.Condition("UsingFlask"))},
	"cannot gain life during f?l?a?s?k? ?effect":                      []mod.Mod{FLAG("CannotGainLife").Tag(mod.Condition("UsingFlask"))},
	"cannot gain mana during f?l?a?s?k? ?effect":                      []mod.Mod{FLAG("CannotGainMana").Tag(mod.Condition("UsingFlask"))},
	"cannot gain spirit from equipment":                               []mod.Mod{FLAG("CannotGainSpiritFromEquipment")},
	"life that would be lost by taking damage is instead reserved":    []mod.Mod{FLAG("DamageInsteadReservesLife")},
	"you have no armour or energy shield":                             []mod.Mod{MOD("Armour", "OVERRIDE", 0), MOD("EnergyShield", "OVERRIDE", 0)},
	"you have no armour or maximum energy shield":                     []mod.Mod{MOD("Armour", "OVERRIDE", 0), MOD("EnergyShield", "OVERRIDE", 0)},
	"defences are zero":                                               []mod.Mod{MOD("Armour", "OVERRIDE", 0), MOD("EnergyShield", "OVERRIDE", 0), MOD("Evasion", "OVERRIDE", 0), MOD("Ward", "OVERRIDE", 0)},
	"you have no intelligence":                                        []mod.Mod{MOD("Int", "OVERRIDE", 0)},
	"elemental resistances are zero":                                  []mod.Mod{MOD("FireResist", "OVERRIDE", 0), MOD("ColdResist", "OVERRIDE", 0), MOD("LightningResist", "OVERRIDE", 0)},
	"you have no spirit":                                              []mod.Mod{MOD("Spirit", "OVERRIDE", 0)},
	"you have no elemental resistances":                               []mod.Mod{MOD("FireResist", "OVERRIDE", 0), MOD("ColdResist", "OVERRIDE", 0), MOD("LightningResist", "OVERRIDE", 0)},
	"chaos resistance is zero":                                        []mod.Mod{MOD("ChaosResist", "OVERRIDE", 0)},
	`nearby enemies' chaos resistance is (\d+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ChaosResist", "OVERRIDE", num)})}, ""
	},
	`your maximum resistances are (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireResistMax", "OVERRIDE", num), MOD("ColdResistMax", "OVERRIDE", num), MOD("LightningResistMax", "OVERRIDE", num), MOD("ChaosResistMax", "OVERRIDE", num)}, ""
	},
	`fire resistance is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireResist", "OVERRIDE", num)}, ""
	},
	`cold resistance is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ColdResist", "OVERRIDE", num)}, ""
	},
	`lightning resistance is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningResist", "OVERRIDE", num)}, ""
	},
	"elemental resistances are capped by your highest maximum elemental resistance instead": []mod.Mod{FLAG("ElementalResistMaxIsHighestResistMax")},
	`nearby enemies have (\d+)% increased fire and cold resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireResist", "INC", num)}), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdResist", "INC", num)})}, ""
	},
	"nearby enemies are blinded while physical aegis is not depleted":    []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Blinded")}).Tag(mod.Condition("PhysicalAegisDepleted").Neg(true))},
	"armour is increased by uncapped fire resistance":                    []mod.Mod{FLAG("ArmourIncreasedByUncappedFireRes")},
	"armour is increased by overcapped fire resistance":                  []mod.Mod{FLAG("ArmourIncreasedByOvercappedFireRes")},
	"minion life is increased by t?h?e?i?r? ?overcapped fire resistance": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Life", "INC", 1).Tag(mod.PerStat(1, "FireResistOverCap"))})},
	`(\d+)% to maximum fire resistance for each (\d+)% uncapped fire resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireResistMax", "BASE", num).Tag(mod.PerStat(utils.Float(captures[1]), "FireResistTotal"))}, ""
	},
	"evasion rating is increased by uncapped cold resistance":               []mod.Mod{FLAG("EvasionRatingIncreasedByUncappedColdRes")},
	"evasion rating is increased by uncapped lightning resistance":          []mod.Mod{FLAG("EvasionRatingIncreasedByUncappedLightningRes")},
	"evasion rating is increased by overcapped cold resistance":             []mod.Mod{FLAG("EvasionRatingIncreasedByOvercappedColdRes")},
	"energy shield is increased by overcapped cold resistance":              []mod.Mod{FLAG("EnergyShieldIncreasedByOvercappedColdRes")},
	"energy shield is increased by uncapped cold resistance":                []mod.Mod{FLAG("EnergyShieldIncreasedByUncappedColdRes")},
	"evasion rating is increased by overcapped lightning resistance":        []mod.Mod{FLAG("EvasionRatingIncreasedByOvercappedLightningRes")},
	"elemental ailment threshold is increased by uncapped chaos resistance": []mod.Mod{FLAG("AilmentThresholdIncreasedByUncappedChaosRes")},
	`reflects (\d+) physical damage to melee attackers`:                     []mod.Mod{},
	"ignore all movement penalties from armour":                             []mod.Mod{FLAG("Condition:IgnoreMovementPenalties")},
	"gain armour equal to your reserved mana":                               []mod.Mod{MOD("Armour", "BASE", 1).Tag(mod.PerStat(1, "ManaReserved"))},
	`gain ward instead of (\d+)% of armour and evasion rating from equipped body armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("ConvertBodyArmourArmourEvasionToWard"), MOD("BodyArmourArmourEvasionToWardPercent", "BASE", num)}, ""
	},
	`(\d+)% increased armour per (\d+) reserved mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Armour", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "ManaReserved"))}, ""
	},
	"cannot be stunned":                                  []mod.Mod{FLAG("StunImmune")},
	"cannot be stunned while bleeding":                   []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("Bleeding"))},
	"cannot be stunned when on low life":                 []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("LowLife"))},
	"cannot be stunned if you haven't been hit recently": []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("BeenHitRecently").Neg(true))},
	`cannot be stunned if you have at least (\d+) crab barriers`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("StunImmune").Tag(mod.StatThreshold("CrabBarriers", num))}, ""
	},
	"cannot be blinded":                             []mod.Mod{FLAG("Condition:CannotBeBlinded"), FLAG("BlindImmune")},
	"cannot be blinded while affected by precision": []mod.Mod{FLAG("Condition:CannotBeBlinded").Tag(mod.Condition("AffectedByPrecision")), FLAG("BlindImmune").Tag(mod.Condition("AffectedByPrecision"))},
	"cannot be blinded while on full life":          []mod.Mod{FLAG("Condition:CannotBeBlinded").Tag(mod.Condition("FullLife"))},
	"cannot be knocked back":                        []mod.Mod{FLAG("KnockbackImmune")},
	"cannot be shocked":                             []mod.Mod{FLAG("ShockImmune")},
	"immun[ei]t?y? to shock":                        []mod.Mod{FLAG("ShockImmune")},
	"cannot be frozen":                              []mod.Mod{FLAG("FreezeImmune")},
	"immun[ei]t?y? to freeze":                       []mod.Mod{FLAG("FreezeImmune")},
	"cannot be chilled":                             []mod.Mod{FLAG("ChillImmune")},
	"immun[ei]t?y? to chill":                        []mod.Mod{FLAG("ChillImmune")},
	"cannot be ignited":                             []mod.Mod{FLAG("IgniteImmune")},
	"immun[ei]t?y? to ignite":                       []mod.Mod{FLAG("IgniteImmune")},
	"you cannot be chilled or frozen":               []mod.Mod{FLAG("ChillImmune"), FLAG("FreezeImmune")},
	"critical hits against you do not inherently inflict elemental ailments":      []mod.Mod{FLAG("CritsOnYouDontAlwaysApplyElementalAilments")},
	"cannot be ignited while at maximum endurance charges":                        []mod.Mod{FLAG("IgniteImmune").Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax"), mod.GlobalEffect("Global").Unscalable(true))},
	`grants immunity to ignite for (\d+) seconds if used while ignited`:           []mod.Mod{FLAG("IgniteImmune").Tag(mod.Condition("UsingFlask"))},
	`grants immunity to bleeding for (\d+) seconds if used while bleeding`:        []mod.Mod{FLAG("BleedImmune").Tag(mod.Condition("UsingFlask"))},
	`grants immunity to poison for (\d+) seconds if used while poisoned`:          []mod.Mod{FLAG("PoisonImmune").Tag(mod.Condition("UsingFlask"))},
	`grants immunity to freeze for (\d+) seconds if used while frozen`:            []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("UsingFlask"))},
	`grants immunity to chill for (\d+) seconds if used while chilled`:            []mod.Mod{FLAG("ChillImmune").Tag(mod.Condition("UsingFlask"))},
	`grants immunity to shock for (\d+) seconds if used while shocked`:            []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("UsingFlask"))},
	"cannot be chilled while at maximum frenzy charges":                           []mod.Mod{FLAG("ChillImmune").Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	"cannot be shocked while at maximum power charges":                            []mod.Mod{FLAG("ShockImmune").Tag(mod.StatThresholdStat("PowerCharges", "PowerChargesMax"))},
	"you cannot be shocked while at maximum endurance charges":                    []mod.Mod{FLAG("ShockImmune").Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax"))},
	"you cannot be shocked while chilled":                                         []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("Chilled"))},
	"you cannot be shocked while frozen":                                          []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("Frozen"))},
	"cannot be shocked while chilled":                                             []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("Chilled"))},
	"cannot be shocked if intelligence is higher than strength":                   []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("IntHigherThanStr"))},
	"cannot be frozen if dexterity is higher than intelligence":                   []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("DexHigherThanInt"))},
	"cannot be frozen if energy shield recharge has started recently":             []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("EnergyShieldRechargeRecently"))},
	"cannot be ignited if strength is higher than dexterity":                      []mod.Mod{FLAG("IgniteImmune").Tag(mod.Condition("StrHigherThanDex"))},
	"cannot be chilled while burning":                                             []mod.Mod{FLAG("ChillImmune").Tag(mod.Condition("Burning"))},
	"cannot be chilled while you have onslaught":                                  []mod.Mod{FLAG("ChillImmune").Tag(mod.Condition("Onslaught"))},
	"cannot be chilled during onslaught":                                          []mod.Mod{FLAG("ChillImmune").Tag(mod.Condition("Onslaught"))},
	"cannot be frozen or chilled if you've used a fire skill recently":            []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("UsedFireSkillRecently")), FLAG("ChillImmune").Tag(mod.Condition("UsedFireSkillRecently"))},
	"cannot be inflicted with bleeding":                                           []mod.Mod{FLAG("BleedImmune")},
	"bleeding cannot be inflicted on you":                                         []mod.Mod{FLAG("BleedImmune")},
	"you are immune to bleeding":                                                  []mod.Mod{FLAG("BleedImmune")},
	"immune to bleeding if equipped helmet has higher armour than evasion rating": []mod.Mod{FLAG("BleedImmune").Tag(mod.Condition("HelmetArmourHigherThanEvasion"))},
	"immune to poison if equipped helmet has higher evasion rating than armour":   []mod.Mod{FLAG("PoisonImmune").Tag(mod.Condition("HelmetEvasionHigherThanArmour"))},
	"immun[ei]t?y? to bleeding and corrupted blood during f?l?a?s?k? ?effect":     []mod.Mod{FLAG("BleedImmune").Tag(mod.Condition("UsingFlask"))},
	"immun[ei]t?y? to poison":                                                     []mod.Mod{FLAG("PoisonImmune")},
	"cannot be poisoned":                                                          []mod.Mod{FLAG("PoisonImmune")},
	"cannot be poisoned while bleeding":                                           []mod.Mod{FLAG("PoisonImmune").Tag(mod.Condition("Bleeding"))},
	"immun[ei]t?y? to poison during f?l?a?s?k? ?effect":                           []mod.Mod{FLAG("PoisonImmune").Tag(mod.Condition("UsingFlask"))},
	"immun[ei]t?y? to shock during f?l?a?s?k? ?effect":                            []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("UsingFlask"))},
	"immun[ei]t?y? to freeze and chill during f?l?a?s?k? ?effect":                 []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("UsingFlask")), FLAG("ChillImmune").Tag(mod.Condition("UsingFlask"))},
	"immun[ei]t?y? to freeze and chill while ignited":                             []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("Ignited")), FLAG("ChillImmune").Tag(mod.Condition("Ignited"))},
	"immun[ei]t?y? to ignite during f?l?a?s?k? ?effect":                           []mod.Mod{FLAG("IgniteImmune").Tag(mod.Condition("UsingFlask"))},
	"immun[ei]t?y? to bleeding during f?l?a?s?k? ?effect":                         []mod.Mod{FLAG("BleedImmune").Tag(mod.Condition("UsingFlask"))},
	"immun[ei]t?y? to curses during f?l?a?s?k? ?effect":                           []mod.Mod{FLAG("CurseImmune").Tag(mod.Condition("UsingFlask"))},
	`you are unaffected by (.+) if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Self"+firstToUpper(captures[0])+"Effect", "MORE", -100).
			Tag(mod.Condition(selfCastCondition(captures[1]))).
			Tag(mod.GlobalEffect("Global").Unscalable(true))}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:5318-5349
	`when you kill an enemy affected by a non\-aura hex, become immune to curses for remaining hex duration`: []mod.Mod{FLAG("Condition:CanBeCurseImmune")},
	`when you kill an enemy cursed with a non\-aura hex, become immune to curses for remaining hex duration`: []mod.Mod{FLAG("Condition:CanBeCurseImmune")},
	"immun[ei]t?y? to freeze, chill, curses and stuns during f?l?a?s?k? ?effect":                             []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("UsingFlask")), FLAG("ChillImmune").Tag(mod.Condition("UsingFlask")), FLAG("CurseImmune").Tag(mod.Condition("UsingFlask")), FLAG("StunImmune").Tag(mod.Condition("UsingFlask"))},
	`immun[ei]t?y? to (\w+) if a majority of your socketed support gems are (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG(firstToUpper(captures[0]) + "Immune").Tag(mod.Condition("Majority" + firstToUpper(captures[1]) + "SocketedSupports"))}, ""
	},
	`gain (\d+)% of maximum energy shield as additional (\w+) threshold`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[1])+"Threshold", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num))}, ""
	},
	`gain additional (\w+) threshold equal to (\d+)% of maximum energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"Threshold", "BASE", 1).Tag(mod.PercentStat("EnergyShield", utils.Float(captures[1])))}, ""
	},
	`(\d+)% increased maximum life, mana and energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Life", "INC", num), MOD("Mana", "INC", num), MOD("EnergyShield", "INC", num)}, ""
	},
	"gain stun threshold equal to the lowest of evasion and armour on your helmet":   []mod.Mod{MOD("StunThreshold", "BASE", 1).Tag(mod.PerStat(1, "LowestOfArmourAndEvasionOnHelmet"))},
	"gain ailment threshold equal to the lowest of evasion and armour on your boots": []mod.Mod{MOD("AilmentThreshold", "BASE", 1).Tag(mod.PerStat(1, "LowestOfArmourAndEvasionOnBoots"))},
	//pob2:block:src/Modules/ModParser.lua:5353-5367
	"your stun threshold is doubled": []mod.Mod{MOD("StunThreshold", "MORE", 100)},
	`(\d+)% of base armour from equipment also added to stun threshold`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("StunThreshold", "BASE", 1).Tag(mod.PercentStat("ArmourOnHelmet", num)), MOD("StunThreshold", "BASE", 1).Tag(mod.PercentStat("ArmourOnGloves", num)), MOD("StunThreshold", "BASE", 1).Tag(mod.PercentStat("ArmourOnBoots", num)), MOD("StunThreshold", "BASE", 1).Tag(mod.PercentStat("ArmourOnBody Armour", num))}, ""
	},
	`gain additional stun threshold equal to (\d+)% of (i?t?e?m? ?)armour on equipped armour items`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("StunThreshold", "BASE", 1).Tag(mod.PercentStat("ArmourOnHelmet", num)), MOD("StunThreshold", "BASE", 1).Tag(mod.PercentStat("ArmourOnGloves", num)), MOD("StunThreshold", "BASE", 1).Tag(mod.PercentStat("ArmourOnBoots", num)), MOD("StunThreshold", "BASE", 1).Tag(mod.PercentStat("ArmourOnBody Armour", num))}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:5373-5590
	"enemies inflict elemental ailments on you instead of nearby allies": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("ElementalAilmentImmune"), OnlyAllies: true})},
	"unaffected by curses":                            []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"unaffected by curses while affected by zealotry": []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.Condition("AffectedByZealotry"), mod.GlobalEffect("Global").Unscalable(true))},
	`immun[ei]t?y? to curses while you have at least (\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("CurseImmune").Tag(mod.MultiplierThreshold("Rage").Threshold(num))}, ""
	},
	"you cannot be cursed with silence":                []mod.Mod{FLAG("SilenceImmune")},
	"unaffected by ignite":                             []mod.Mod{MOD("SelfIgniteEffect", "MORE", -100)},
	"unaffected by chill":                              []mod.Mod{MOD("SelfChillEffect", "MORE", -100)},
	"unaffected by chill while leeching mana":          []mod.Mod{MOD("SelfChillEffect", "MORE", -100).Tag(mod.Condition("LeechingMana"))},
	"unaffected by chill while channelling":            []mod.Mod{MOD("SelfChillEffect", "MORE", -100).Tag(mod.Condition("Channelling"))},
	"unaffected by freeze":                             []mod.Mod{MOD("SelfFreezeEffect", "MORE", -100)},
	"unaffected by shock":                              []mod.Mod{MOD("SelfShockEffect", "MORE", -100)},
	"unaffected by shock while leeching energy shield": []mod.Mod{MOD("SelfShockEffect", "MORE", -100).Tag(mod.Condition("LeechingEnergyShield"))},
	"unaffected by shock while channelling":            []mod.Mod{MOD("SelfShockEffect", "MORE", -100).Tag(mod.Condition("Channelling"))},
	"unaffected by scorch":                             []mod.Mod{MOD("SelfScorchEffect", "MORE", -100)},
	"unaffected by brittle":                            []mod.Mod{MOD("SelfBrittleEffect", "MORE", -100)},
	"unaffected by sap":                                []mod.Mod{MOD("SelfSapEffect", "MORE", -100)},
	"unaffected by damaging ailments":                  []mod.Mod{MOD("SelfBleedEffect", "MORE", -100), MOD("SelfIgniteEffect", "MORE", -100), MOD("SelfPoisonEffect", "MORE", -100)},
	"the effect of chill on you is reversed":           []mod.Mod{FLAG("SelfChillEffectIsReversed")},
	`your movement speed is (\d+)% of its base value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeed", "OVERRIDE", (num / 100))}, ""
	},
	`action speed cannot be modified to below (\d+)% base value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinimumActionSpeed", "MAX", num)}, ""
	},
	`action speed cannot be slowed below base value if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.Condition(selfCastCondition(captures[0])))}, ""
	},
	`action speed cannot be modified to below base value if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.Condition(selfCastCondition(captures[0])))}, ""
	},
	"nearby allies' action speed cannot be modified to below base value": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.GlobalEffect("Global").Unscalable(true)), OnlyAllies: true})},
	"lightning resistance does not affect lightning damage taken":        []mod.Mod{FLAG("SelfIgnoreLightningResistance")},
	`(\d+)% increased maximum life and reduced fire resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Life", "INC", num), MOD("FireResist", "INC", -num)}, ""
	},
	`(\d+)% increased maximum mana and reduced cold resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Mana", "INC", num), MOD("ColdResist", "INC", -num)}, ""
	},
	`(\d+)% increased global maximum energy shield and reduced lightning resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShield", "INC", num).Tag(mod.Global()), MOD("LightningResist", "INC", -num)}, ""
	},
	"phasing while on low life":                                []mod.Mod{FLAG("Condition:Phasing").Tag(mod.Condition("LowLife"))},
	"cannot be ignited while on low life":                      []mod.Mod{FLAG("IgniteImmune").Tag(mod.Condition("LowLife"))},
	"stun threshold is based on energy shield instead of life": []mod.Mod{FLAG("StunThresholdBasedOnEnergyShieldInsteadOfLife"), MOD("StunThresholdEnergyShieldPercent", "BASE", 100)},
	`stun threshold is based on (\d+)% of your energy shield instead of life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("StunThresholdBasedOnEnergyShieldInsteadOfLife"), MOD("StunThresholdEnergyShieldPercent", "BASE", num)}, ""
	},
	`stun threshold is based on (\d+)% of your mana instead of life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("StunThresholdBasedOnManaInsteadOfLife"), MOD("StunThresholdManaPercent", "BASE", num)}, ""
	},
	`(\d+)% of your energy shield is added to your stun threshold`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("AddESToStunThreshold"), MOD("ESToStunThresholdPercent", "BASE", num)}, ""
	},
	"heavy stuns enemies that are on full life": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:HeavyStunned").Tag(mod.Condition("FullLife"))})},
	`(\d+)% increased armour per second you've been stationary, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Armour", "INC", num).Tag(mod.Multiplier("StationarySeconds").Limit((utils.Float(captures[1]) / num)), mod.Condition("Stationary"))}, ""
	},
	`(\d+)% of damage from hits is taken from your spectres' life before you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TakenFromSpectresBeforeYou", "BASE", num)}, ""
	},
	`(\d+)% of damage from hits is taken from your nearest totem's life before you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TakenFromTotemsBeforeYou", "BASE", num).Tag(mod.Condition("HaveTotem"))}, ""
	},
	`(\d+)% of damage from hits is taken from your damageable companion's life before you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TakenFromCompanionBeforeYou", "BASE", num)}, ""
	},
	`(\d+)% of damage from deflected hits is taken from damageable companion's life before you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TakenFromCompanionBeforeYouFromDeflected", "BASE", num)}, ""
	},
	`(\w+) resistance cannot be penetrated`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("EnemyCannotPen" + firstToUpper(captures[0]) + "Resistance")}, ""
	},
	`your base energy shield recharge delay is (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldRechargeBase", "OVERRIDE", num)}, ""
	},
	"cannot knock enemies back":  []mod.Mod{FLAG("CannotKnockback")},
	"knocks enemies back on hit": []mod.Mod{FLAG("Knockback")},
	"knocks back enemies if you get a critical hit with a q?u?a?r?t?e?r?staff":                     []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(ModFlagStaff).Tag(mod.Condition("CriticalStrike"))},
	"knocks back enemies if you get a critical hit with a bow":                                     []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(ModFlagBow).Tag(mod.Condition("CriticalStrike"))},
	"bow knockback at close range":                                                                 []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(ModFlagBow).Tag(mod.Condition("AtCloseRange"))},
	"adds knockback during f?l?a?s?k? ?effect":                                                     []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"adds knockback to melee attacks during f?l?a?s?k? ?effect":                                    []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(ModFlagMelee).Tag(mod.Condition("UsingFlask"))},
	"knockback direction is reversed":                                                              []mod.Mod{MOD("EnemyKnockbackDistance", "MORE", -200)},
	"culling strike":                                                                               []mod.Mod{MOD("CanCull", "FLAG", true).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"culling strike with melee weapons":                                                            []mod.Mod{MOD("CanCull", "FLAG", true).Flag(ModFlagWeaponMelee).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"melee weapon attacks have culling strike":                                                     []mod.Mod{MOD("CanCull", "FLAG", true).Flag(ModFlagAttack | ModFlagWeaponMelee).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"culling strike during f?l?a?s?k? ?effect":                                                     []mod.Mod{MOD("CanCull", "FLAG", true).Tag(mod.Condition("UsingFlask"), mod.GlobalEffect("Global").Unscalable(true))},
	"hits with this weapon have culling strike against bleeding enemies":                           []mod.Mod{MOD("CanCull", "FLAG", true).Tag(mod.ActorCondition("enemy", "Bleeding"))},
	"you have culling strike against cursed enemies":                                               []mod.Mod{FLAG("CanCull").Tag(mod.ActorCondition("enemy", "Cursed"))},
	"critical hits have culling strike":                                                            []mod.Mod{FLAG("CritCanCull")},
	"your critical hits have culling strike":                                                       []mod.Mod{FLAG("CritCanCull")},
	"your spells have culling strike":                                                              []mod.Mod{MOD("CanCull", "FLAG", true).Flag(ModFlagSpell)},
	"bow attacks have culling strike":                                                              []mod.Mod{MOD("CanCull", "FLAG", true).Flag(ModFlagAttack | ModFlagBow)},
	"culling strike against burning enemies":                                                       []mod.Mod{MOD("CanCull", "FLAG", true).Tag(mod.ActorCondition("enemy", "Burning"))},
	"culling strike against frozen enemies":                                                        []mod.Mod{MOD("CanCull", "FLAG", true).Tag(mod.ActorCondition("enemy", "Frozen"))},
	"culling strike against enemies you mark":                                                      []mod.Mod{MOD("CanCull", "FLAG", true).Tag(mod.ActorCondition("enemy", "Marked"))},
	"nearby allies have culling strike":                                                            []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("CanCull"), OnlyAllies: true})},
	"hits that stun enemies have culling strike":                                                   []mod.Mod{MOD("CanCull", "FLAG", true).Tag(mod.Condition("AlwaysStunning"))},
	"hits that heavy stun enemies have culling strike":                                             []mod.Mod{MOD("CanCull", "FLAG", true).Tag(mod.Condition("AlwaysHeavyStunning"))},
	"permanently intimidate enemies on block":                                                      []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).Tag(mod.Condition("BlockedRecently"))},
	`with a murderous eye jewel socketed, intimidate enemies for (\d) seconds on hit with attacks`: []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).Tag(mod.Condition("HaveMurderousEyeJewelIn{SlotName}"))},
	"enemies taunted by your warcries are intimidated":                                             []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated").Tag(mod.Condition("Taunted"))}).Tag(mod.Condition("UsedWarcryRecently"))},
	`intimidate enemies for (\d+) seconds on block while holding a shield`:                         []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).Tag(mod.Condition("BlockedRecently"), mod.Condition("UsingShield"))},
	`intimidate enemies on hit if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).
			Flag(ModFlagHit).
			Tag(mod.Condition(selfCastCondition(captures[0])))}, ""
	},
	`intimidate enemies for (\d+) seconds on hit with attacks while at maximum endurance charges`: []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax"), mod.Condition("HitRecently"))},
	"nearby enemies are intimidated while you have rage":                                          []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).Tag(mod.MultiplierThreshold("RageStack").Threshold(1))},
	"flasks do not apply to you":                                                                  []mod.Mod{FLAG("FlasksDoNotApplyToPlayer")},
	"flasks apply to your zombies and spectres":                                                   []mod.Mod{FLAG("FlasksApplyToMinion").Tag(mod.SkillName("Raise Zombie", "Raise Spectre").Transfigured(true))},
	"flasks apply to your raised zombies and spectres":                                            []mod.Mod{FLAG("FlasksApplyToMinion").Tag(mod.SkillName("Raise Zombie", "Raise Spectre").Transfigured(true))},
	"flasks you use apply to your raised zombies and spectres":                                    []mod.Mod{FLAG("FlasksApplyToMinion").Tag(mod.SkillName("Raise Zombie", "Raise Spectre").Transfigured(true))},
	"your minions use your flasks when summoned":                                                  []mod.Mod{FLAG("FlasksApplyToMinion")},
	`recover an additional (\d+)% of flask's life recovery amount over 10 seconds if used while not on full life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskAdditionalLifeRecovery", "BASE", num)}, ""
	},
	"creates a smoke cloud on use":                     []mod.Mod{},
	"creates chilled ground on use":                    []mod.Mod{},
	"creates consecrated ground on use":                []mod.Mod{},
	"removes bleeding on use":                          []mod.Mod{},
	"removes burning on use":                           []mod.Mod{},
	"removes all burning when used":                    []mod.Mod{},
	"removes curses on use":                            []mod.Mod{},
	"removes freeze and chill on use":                  []mod.Mod{},
	"removes poison on use":                            []mod.Mod{},
	"removes shock on use":                             []mod.Mod{},
	"g?a?i?n? ?unholy might during f?l?a?s?k? ?effect": []mod.Mod{FLAG("Condition:UnholyMight").Tag(mod.Condition("UsingFlask"))},
	"zealot's oath during f?l?a?s?k? ?effect":          []mod.Mod{FLAG("ZealotsOath").Tag(mod.Condition("UsingFlask"))},
	`grants level (\d+) (.+) curse aura during f?l?a?s?k? ?effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraCurse", "LIST", mod.ExtraCurse{SkillID: flaskCurseSkillID(captures[1]), Level: int(num)}).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`shocks nearby enemies during f?l?a?s?k? ?effect, causing (\d+)% increased damage taken`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ShockOverride", "BASE", num).Tag(mod.Condition("UsingFlask"))}, ""
	},
	`during f?l?a?s?k? ?effect, (\d+)% reduced damage taken of each element for which your uncapped elemental resistance is lowest`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningDamageTaken", "INC", -num).Tag(mod.StatThresholdStat("LightningResistTotal", "ColdResistTotal").Upper(true), mod.StatThresholdStat("LightningResistTotal", "FireResistTotal").Upper(true)), MOD("ColdDamageTaken", "INC", -num).Tag(mod.StatThresholdStat("ColdResistTotal", "LightningResistTotal").Upper(true), mod.StatThresholdStat("ColdResistTotal", "FireResistTotal").Upper(true)), MOD("FireDamageTaken", "INC", -num).Tag(mod.StatThresholdStat("FireResistTotal", "LightningResistTotal").Upper(true), mod.StatThresholdStat("FireResistTotal", "ColdResistTotal").Upper(true))}, ""
	},
	`during f?l?a?s?k? ?effect, damage penetrates (\d+)% o?f? ?resistance of each element for which your uncapped elemental resistance is highest`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningPenetration", "BASE", num).Tag(mod.StatThresholdStat("LightningResistTotal", "ColdResistTotal"), mod.StatThresholdStat("LightningResistTotal", "FireResistTotal")), MOD("ColdPenetration", "BASE", num).Tag(mod.StatThresholdStat("ColdResistTotal", "LightningResistTotal"), mod.StatThresholdStat("ColdResistTotal", "FireResistTotal")), MOD("FirePenetration", "BASE", num).Tag(mod.StatThresholdStat("FireResistTotal", "LightningResistTotal"), mod.StatThresholdStat("FireResistTotal", "ColdResistTotal"))}, ""
	},
	`recover (\d+)% of life when you kill an enemy during f?l?a?s?k? ?effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.Condition("UsingFlask"))}, ""
	},
	`recover (\d+)% of mana when you kill an enemy during f?l?a?s?k? ?effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnKill", "BASE", 1).Tag(mod.PercentStat("Mana", num), mod.Condition("UsingFlask"))}, ""
	},
	`recover (\d+)% of energy shield when you kill an enemy during f?l?a?s?k? ?effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnKill", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num), mod.Condition("UsingFlask"))}, ""
	},
	`(\d+)% of maximum life taken as chaos damage per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosDegen", "BASE", 1).Tag(mod.PercentStat("Life", num))}, ""
	},
	"your critical hits do not deal extra damage during f?l?a?s?k? ?effect": []mod.Mod{FLAG("NoCritMultiplier").Tag(mod.Condition("UsingFlask"))},
	"grants perfect agony during f?l?a?s?k? ?effect":                        []mod.Mod{MOD("Keystone", "LIST", "Perfect Agony").Tag(mod.Condition("UsingFlask"))},
	"grants eldritch battery during f?l?a?s?k? ?effect":                     []mod.Mod{MOD("Keystone", "LIST", "Eldritch Battery").Tag(mod.Condition("UsingFlask"))},
	"eldritch battery during f?l?a?s?k? ?effect":                            []mod.Mod{MOD("Keystone", "LIST", "Eldritch Battery").Tag(mod.Condition("UsingFlask"))},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield during effect":  []mod.Mod{FLAG("ChaosNotBypassEnergyShield")},
	`when hit during effect, (\d+)% of life loss from damage taken occurs over 4 seconds instead`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeLossPrevented", "BASE", num).Tag(mod.Condition("UsingFlask"))}, ""
	},
	`when taking damage from hits, (\d+)% of life loss is prevented, then (\d+)% of life loss prevented this way is lost over 4 seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeLossPrevented", "BASE", num), MOD("LifeLossLost", "BASE", utils.Float(captures[1]))}, ""
	},
	"y?o?u?r? ?skills [ch][oa][sv][te] no mana c?o?s?t? ?during f?l?a?s?k? ?effect":     []mod.Mod{MOD("ManaCost", "MORE", -100).Tag(mod.Condition("UsingFlask"))},
	"life recovery from flasks also applies to energy shield during f?l?a?s?k? ?effect": []mod.Mod{FLAG("LifeFlaskAppliesToEnergyShield").Tag(mod.Condition("UsingFlask"))},
	`consecrated ground created during effect applies (\d+)% increased damage taken to enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTakenConsecratedGround", "INC", num).Tag(mod.Condition("OnConsecratedGround"))}).Tag(mod.Condition("UsingFlask"))}, ""
	},
	"gain alchemist's genius when you use a flask":                  []mod.Mod{FLAG("Condition:CanHaveAlchemistGenius")},
	`(\d+)% chance to gain alchemist's genius when you use a flask`: []mod.Mod{FLAG("Condition:CanHaveAlchemistGenius")},
	`(\d+)% less flask charges gained from kills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskChargesGained", "MORE", -num).Source(mod.Source("from Kills"))}, ""
	},
	`flasks gain (\d+) charges? every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskChargesGenerated", "BASE", (num / utils.Float(captures[1])))}, ""
	},
	`flasks gain a charge every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskChargesGenerated", "BASE", (1 / utils.Float(captures[0])))}, ""
	},
	`while a unique enemy is in your presence, flasks gain a charge every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskChargesGenerated", "BASE", (1 / utils.Float(captures[0]))).Tag(mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	`while a pinnacle atlas boss is in your presence, flasks gain a charge every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskChargesGenerated", "BASE", (1 / utils.Float(captures[0]))).Tag(mod.ActorCondition("enemy", "PinnacleBoss"))}, ""
	},
	`utility flasks gain (\d+) charges? every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("UtilityFlaskChargesGenerated", "BASE", (num / utils.Float(captures[1])))}, ""
	},
	`life flasks gain (\d+) charges? every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeFlaskChargesGenerated", "BASE", (num / utils.Float(captures[1])))}, ""
	},
	`mana flasks gain (\d+) charges? every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaFlaskChargesGenerated", "BASE", (num / utils.Float(captures[1])))}, ""
	},
	`flasks gain (\d+) charges? per second if you've hit a unique enemy recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskChargesGenerated", "BASE", num).Tag(mod.Condition("HitRecently"), mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	"effect is not removed when unreserved mana is filled":              []mod.Mod{FLAG("ManaFlaskEffectNotRemoved")},
	"life flask effects are not removed when unreserved life is filled": []mod.Mod{FLAG("LifeFlaskEffectNotRemoved")},
	`flasks gain (\d*\.?\d+) charges? per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskChargesGenerated", "BASE", num)}, ""
	},
	`life flasks gain (\d*\.?\d+) charges? per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeFlaskChargesGenerated", "BASE", num)}, ""
	},
	`mana flasks gain (\d*\.?\d+) charges? per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaFlaskChargesGenerated", "BASE", num)}, ""
	},
	`charms gain (\d*\.?\d+) charges? per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CharmChargesGenerated", "BASE", num)}, ""
	},
	`h?a?s? ?\+?(\d+) charm slots?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CharmLimit", "BASE", num)}, ""
	},
	"charms use no charges": []mod.Mod{FLAG("CharmsUseNoCharges")},
	//pob2:block:src/Modules/ModParser.lua:5594-5737
	`charms applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CharmEffect", "INC", num).Tag(mod.ActorCondition("player"))}, ""
	},
	`non\-keystone passive skills in medium radius of allocated keystone passive skills can be allocated without being connected to your tree`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AllocateFromNodeRadius", "LIST", LuaTable{"from": "Keystone", "radiusIndex": float64(2), "to": []any{"Notable", "Normal"}})}, ""
	},
	`passives in radius of ([\w\s']+) can be allocated without being connected to your tree`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "fromNothingKeystone", Value: captures[0]}), MOD("FromNothingKeystones", "LIST", LuaTable{"key": captures[0], "value": true})}, ""
	},
	"passives in radius can be allocated without being connected to your tree": []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "intuitiveLeapLike", Value: true})},
	"affects passives in small ring":                                           []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(6)})},
	"affects passives in medium ring":                                          []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(8)})},
	"affects passives in large ring":                                           []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(10)})},
	"affects passives in very large ring":                                      []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(11)})},
	"affects passives in massive ring":                                         []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(12)})},
	"only affects passives in very small ring":                                 []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(5)})},
	"only affects passives in small ring":                                      []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(6)})},
	`only affects passives in medium\-small ring`:                              []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(7)})},
	"only affects passives in medium ring":                                     []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(8)})},
	`only affects passives in medium\-large ring`:                              []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(9)})},
	"only affects passives in large ring":                                      []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(10)})},
	"only affects passives in very large ring":                                 []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(11)})},
	"only affects passives in massive ring":                                    []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: float64(12)})},
	"upgrades radius to medium":                                                []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "timeLostJewelRadiusOverride", Value: float64(2)})},
	"upgrades radius to large":                                                 []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "timeLostJewelRadiusOverride", Value: float64(3)})},
	"upgrades radius to very large":                                            []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "timeLostJewelRadiusOverride", Value: float64(4)})},
	"primordial":                                                               []mod.Mod{MOD("Multiplier:PrimordialItem", "BASE", 1)},
	`spectres have a base duration of (\d+) seconds`:                           []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "duration", Value: 6}).Tag(mod.SkillName("Raise Spectre").Transfigured(true))},
	`flasks applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffect", "INC", num).Tag(mod.ActorCondition("player"))}, ""
	},
	`flasks applied to you have (\d+)% increased effect per level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffect", "INC", num).Tag(mod.ActorCondition("player"), mod.Multiplier("Level"))}, ""
	},
	`while a unique enemy is in your presence, flasks applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffect", "INC", num).Tag(mod.ActorCondition("enemy", "RareOrUnique"), mod.ActorCondition("player"))}, ""
	},
	`while a pinnacle atlas boss is in your presence, flasks applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffect", "INC", num).Tag(mod.ActorCondition("enemy", "PinnacleBoss"), mod.ActorCondition("player"))}, ""
	},
	`magic utility flasks applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MagicUtilityFlaskEffect", "INC", num).Tag(mod.ActorCondition("player"))}, ""
	},
	`flasks applied to you have (\d+)% reduced effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffect", "INC", -num).Tag(mod.ActorCondition("player"))}, ""
	},
	`remnants have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RemnantEffect", "INC", num)}, ""
	},
	`remnants you create have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RemnantEffect", "INC", num)}, ""
	},
	`(\d+)% increased charm cooldown recovery rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CharmCooldownRecovery", "INC", num)}, ""
	},
	`adds (\d+) passive skills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelNodeCount", Value: float64(num)})}, ""
	},
	"1 added passive skill is a jewel socket": []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelSocketCount", Value: float64(1)})},
	`(\d+) added passive skills are jewel sockets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelSocketCount", Value: float64(num)})}, ""
	},
	`adds (\d+) jewel socket passive skills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelSocketCountOverride", Value: float64(num)})}, ""
	},
	`adds (\d+) small passive skills? which grants? nothing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelNothingnessCount", Value: float64(num)})}, ""
	},
	`you can only socket (\w+) jewels in this item`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("JewelSocketRestriction"), FLAG("CanSocketJewelBase" + firstToUpper(captures[0]))}, ""
	},
	"added small passive skills grant nothing": []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelSmallsAreNothingness", Value: true})},
	`added small passive skills have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelIncEffect", Value: float64(num)})}, ""
	},
	`this jewel's socket has (\d+)% increased effect per allocated passive skill between it and your class' starting location`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "jewelIncEffectFromClassStart", Value: float64(num)})}, ""
	},
	`(\d+)% increased effect of jewel socket passive skills containing corrupted (m?r?ag?r?i?e?c?) jewels, if not from cluster jewels`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "corrupted" + firstToUpper(captures[1]) + "JewelIncEffect", Value: float64(num)})}, ""
	},
	`(\d+)% increased effect of jewel socket passive skills containing corrupted (m?r?ag?r?i?e?c?) jewels`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "corrupted" + firstToUpper(captures[1]) + "JewelIncEffect", Value: float64(num)})}, ""
	},
	`legacy of (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LegacyOf"+firstToUpper(captures[0]), "BASE", 1), FLAG("MagebloodEquipped")}, ""
	},
	`all mage's legacies have (\d+)% increased effect per duplicate mage's legacy you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MagesLegacyEffect", "INC", num)}, ""
	},
	"fully broken armour effects also apply to fire damage taken from hits":                    []mod.Mod{FLAG("ArmourBreakFireDamageTaken")},
	"fully broken armour you inflict also increases fire damage taken from hits":               []mod.Mod{FLAG("ArmourBreakFireDamageTaken")},
	"fully broken armour you inflict also increases cold and lightning damage taken from hits": []mod.Mod{FLAG("ArmourBreakColdDamageTaken"), FLAG("ArmourBreakLightningDamageTaken")},
	"can't use c?h?e?s?t? ?b?o?d?y? ?armour":                                                   []mod.Mod{MOD("CanNotUseBody", "Flag", 1).Tag(mod.DisablesItem("Body Armour"))},
	"can't use helmet":                                                                         []mod.Mod{MOD("CanNotUseHelmet", "Flag", 1).Tag(mod.DisablesItem("Helmet"))},
	"can't use other rings":                                                                    []mod.Mod{MOD("CanNotUseRightRing", "Flag", 1).Tag(mod.DisablesItem("Ring 2"), mod.SlotNumber(int(1))), MOD("CanNotUseLeftRing", "Flag", 1).Tag(mod.DisablesItem("Ring 1"), mod.SlotNumber(int(2)))},
	"uses both hand slots":                                                                     []mod.Mod{MOD("CanNotUseRightWeapon", "Flag", 1).Tag(mod.DisablesItem("Weapon 2"), mod.SlotNumber(int(1))), MOD("CanNotUseLeftWeapon", "Flag", 1).Tag(mod.DisablesItem("Weapon 1"), mod.SlotNumber(int(2)))},
	"can't use flask in fifth slot":                                                            []mod.Mod{MOD("CanNotUseFifthFlask", "Flag", 1).Tag(mod.DisablesItem("Flask 5"))},
	"cannot use charms":                                                                        []mod.Mod{MOD("CharmLimit", "OVERRIDE", 0)},
	`boneshatter has (\d+)% chance to grant \+1 trauma`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraTrauma", "BASE", num).Tag(mod.SkillName("Boneshatter").Transfigured(true))}, ""
	},
	"your minimum frenzy, endurance and power charges are equal to your maximum while you are stationary": []mod.Mod{FLAG("MinimumFrenzyChargesIsMaximumFrenzyCharges").Tag(mod.Condition("Stationary")), FLAG("MinimumEnduranceChargesIsMaximumEnduranceCharges").Tag(mod.Condition("Stationary")), FLAG("MinimumPowerChargesIsMaximumPowerCharges").Tag(mod.Condition("Stationary"))},
	"minimum power charges equal to maximum while stationary":                                             []mod.Mod{FLAG("MinimumPowerChargesIsMaximumPowerCharges").Tag(mod.Condition("Stationary"))},
	"minimum frenzy charges equal to maximum while stationary":                                            []mod.Mod{FLAG("MinimumFrenzyChargesIsMaximumFrenzyCharges").Tag(mod.Condition("Stationary"))},
	"minimum endurance charges equal to maximum while stationary":                                         []mod.Mod{FLAG("MinimumEnduranceChargesIsMaximumEnduranceCharges").Tag(mod.Condition("Stationary"))},
	"count as having maximum number of power charges":                                                     []mod.Mod{FLAG("HaveMaximumPowerCharges")},
	"count as having maximum number of frenzy charges":                                                    []mod.Mod{FLAG("HaveMaximumFrenzyCharges")},
	"count as having maximum number of endurance charges":                                                 []mod.Mod{FLAG("HaveMaximumEnduranceCharges")},
	"quarterstaff skills that consume power charges count as consuming an additional power charge":        []mod.Mod{MOD("Multiplier:ExtraConsumablePowerCharges", "BASE", 1).Flag(ModFlagAttack).Tag(mod.Condition("UsingStaff"))},
	`leftmost (\d+) magic utility flasks constantly apply their flask effects to you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveMagicUtilityFlasks", "BASE", num)}, ""
	},
	`marauder: melee skills have (\d+)% increased area of effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("ConnectedToMarauderStart"), mod.SkillType("Melee"))}, ""
	},
	"intelligence provides no bonus to energy shield":          []mod.Mod{FLAG("NoIntBonusToES")},
	"intelligence provides no inherent bonus to energy shield": []mod.Mod{FLAG("NoIntBonusToES")},
	"gain accuracy rating equal to your intelligence":          []mod.Mod{MOD("Accuracy", "BASE", 1).Tag(mod.PerStat(1, "Int"))},
	"intelligence is added to accuracy rating with wands":      []mod.Mod{MOD("Accuracy", "BASE", 1).Flag(ModFlagWand).Tag(mod.PerStat(1, "Int"))},
	`dexterity's accuracy bonus instead grants \+(\d+) to accuracy rating per dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DexAccBonusOverride", "OVERRIDE", num)}, ""
	},
	`(\d+)% increased accuracy rating against enemies you mark`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", "INC", num).Tag(mod.ActorCondition("enemy", "Marked"))}, ""
	},
	`(\d+)% more accuracy rating against enemies you mark`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", "MORE", num).Tag(mod.ActorCondition("enemy", "Marked"))}, ""
	},
	`\+(\d+) to accuracy against bleeding enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", "BASE", num).Tag(mod.ActorCondition("enemy", "Bleeding"))}, ""
	},
	"y?o?u? ?cannot recover energy shield to above armour": []mod.Mod{FLAG("ArmourESRecoveryCap")},
	"cannot recover energy shield to above evasion rating": []mod.Mod{FLAG("EvasionESRecoveryCap")},
	`warcries empower (\d+) additional attacks?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraEmpoweredAttacks", "BASE", num)}, ""
	},
	"warcries empower an additional attack": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraEmpoweredAttacks", "BASE", 1)}, ""
	},
	`warcries have (\d+)% chance to empower (\d+) additional attacks?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraEmpoweredAttacks", "BASE", ((num * utils.Float(captures[1])) / 100))}, ""
	},
	`skills deal (\d+)% more damage for each warcry exerting them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EchoesExertAverageIncrease", "MORE", num)}, ""
	},
	"iron reflexes while stationary":                          []mod.Mod{MOD("Keystone", "LIST", "Iron Reflexes").Tag(mod.Condition("Stationary"))},
	"you have iron reflexes while at maximum frenzy charges":  []mod.Mod{MOD("Keystone", "LIST", "Iron Reflexes").Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	"you have zealot's oath if you haven't been hit recently": []mod.Mod{MOD("Keystone", "LIST", "Zealot's Oath").Tag(mod.Condition("BeenHitRecently").Neg(true))},
	"deal no physical damage":                                 []mod.Mod{FLAG("DealNoPhysical")},
	"deal no cold damage":                                     []mod.Mod{FLAG("DealNoCold")},
	"deal no fire damage":                                     []mod.Mod{FLAG("DealNoFire")},
	"deal no lightning damage":                                []mod.Mod{FLAG("DealNoLightning")},
	"deal no elemental damage":                                []mod.Mod{FLAG("DealNoLightning"), FLAG("DealNoCold"), FLAG("DealNoFire")},
	"deal no chaos damage":                                    []mod.Mod{FLAG("DealNoChaos")},
	"deal no damage":                                          []mod.Mod{FLAG("DealNoLightning"), FLAG("DealNoCold"), FLAG("DealNoFire"), FLAG("DealNoChaos"), FLAG("DealNoPhysical")},
	"you can't deal damage with skills yourself":              []mod.Mod{FLAG("DealNoLightning").Tag(mod.SkillType("SummonsTotem", "RemoteMined", "Trapped").Neg(true), mod.Condition("usedByMirage").Neg(true)), FLAG("DealNoCold").Tag(mod.SkillType("SummonsTotem", "RemoteMined", "Trapped").Neg(true), mod.Condition("usedByMirage").Neg(true)), FLAG("DealNoFire").Tag(mod.SkillType("SummonsTotem", "RemoteMined", "Trapped").Neg(true), mod.Condition("usedByMirage").Neg(true)), FLAG("DealNoChaos").Tag(mod.SkillType("SummonsTotem", "RemoteMined", "Trapped").Neg(true), mod.Condition("usedByMirage").Neg(true)), FLAG("DealNoPhysical").Tag(mod.SkillType("SummonsTotem", "RemoteMined", "Trapped").Neg(true), mod.Condition("usedByMirage").Neg(true))},
	`deal no non\-elemental damage`:                           []mod.Mod{FLAG("DealNoPhysical"), FLAG("DealNoChaos")},
	`deal no non\-lightning damage`:                           []mod.Mod{FLAG("DealNoPhysical"), FLAG("DealNoCold"), FLAG("DealNoFire"), FLAG("DealNoChaos")},
	`deal no non\-physical damage`:                            []mod.Mod{FLAG("DealNoLightning"), FLAG("DealNoCold"), FLAG("DealNoFire"), FLAG("DealNoChaos")},
	`cannot deal non\-chaos damage`:                           []mod.Mod{FLAG("DealNoPhysical"), FLAG("DealNoCold"), FLAG("DealNoFire"), FLAG("DealNoLightning")},
	"deal no physical or elemental damage":                    []mod.Mod{FLAG("DealNoPhysical"), FLAG("DealNoCold"), FLAG("DealNoFire"), FLAG("DealNoLightning")},
	"deal no damage when not on low life":                     []mod.Mod{FLAG("DealNoLightning").Tag(mod.Condition("LowLife").Neg(true)), FLAG("DealNoCold").Tag(mod.Condition("LowLife").Neg(true)), FLAG("DealNoFire").Tag(mod.Condition("LowLife").Neg(true)), FLAG("DealNoChaos").Tag(mod.Condition("LowLife").Neg(true)), FLAG("DealNoPhysical").Tag(mod.Condition("LowLife").Neg(true))},
	`trigger a socketed elemental spell on block, with a ([\d\.]+) second cooldown`:                              []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerElementalSpellOnBlock", Level: int(1)}).Tag(mod.SocketedIn("{SlotName}"))},
	`(\d+)% chance to cast a? ?socketed lightning spells? on hit`:                                                []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: int(1)}).Tag(mod.SocketedIn("{SlotName}"))},
	"cast a socketed lightning spell on hit":                                                                     []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: int(1)}).Tag(mod.SocketedIn("{SlotName}"))},
	"trigger a socketed lightning spell on hit":                                                                  []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: int(1)}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed lightning spell on hit, with a ([\d\.]+) second cooldown`:                                []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: int(1)}).Tag(mod.SocketedIn("{SlotName}"))},
	"[ct][ar][si][tg]g?e?r? a socketed cold s[pk][ei]ll on melee critical hit":                                   []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike", Level: int(1)}).Tag(mod.SocketedIn("{SlotName}"))},
	`[ct][ar][si][tg]g?e?r? a socketed cold s[pk][ei]ll on melee critical hit, with a ([\d\.]+) second cooldown`: []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike", Level: int(1)}).Tag(mod.SocketedIn("{SlotName}"))},
	"your curses can apply to hexproof enemies":                                                                  []mod.Mod{FLAG("CursesIgnoreHexproof")},
	"your hexes can affect hexproof enemies":                                                                     []mod.Mod{FLAG("CursesIgnoreHexproof")},
	//pob2:block:src/Modules/ModParser.lua:5741-5741
	`hexes from socketed skills can apply (\d) additional curses`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketedCursesHexLimitValue", "BASE", num), FLAG("SocketedCursesAdditionalLimit").Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:5745-6037
	"hexes from socketed skills ignore curse limit": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketedCursesHexLimitValue", "BASE", 5), FLAG("SocketedCursesAdditionalLimit").Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`reserves (\d+)% of life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraLifeReserved", "BASE", num)}, ""
	},
	`(\d+)% of cold damage taken as lightning`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ColdDamageTakenAsLightning", "BASE", num)}, ""
	},
	`(\d+)% of fire damage taken as lightning`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireDamageTakenAsLightning", "BASE", num)}, ""
	},
	`(\d+)% of fire and lightning damage from hits taken as cold damage during effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireDamageFromHitsTakenAsCold", "BASE", num).Tag(mod.Condition("UsingFlask")), MOD("LightningDamageFromHitsTakenAsCold", "BASE", num).Tag(mod.Condition("UsingFlask"))}, ""
	},
	`(\d+)% of physical damage from hits taken as damage of a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageFromHitsTakenAsFire", "BASE", (num / 3)), MOD("PhysicalDamageFromHitsTakenAsCold", "BASE", (num / 3)), MOD("PhysicalDamageFromHitsTakenAsLightning", "BASE", (num / 3))}, ""
	},
	`equipment and skill gems have (\d+)% reduced attribute requirements`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GlobalAttributeRequirements", "INC", -num)}, ""
	},
	`equipment and skill gems have (\d+)% increased attribute requirements`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GlobalAttributeRequirements", "INC", num)}, ""
	},
	`skill gems have (\d+)% more attribute requirements`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GlobalGemAttributeRequirements", "MORE", num)}, ""
	},
	"skill gems have no attribute requirements": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GlobalGemAttributeRequirements", "MORE", -100)}, ""
	},
	"triple attribute requirements of martial weapons": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GlobalWeaponAttributeRequirements", "MORE", 200)}, ""
	},
	"strength can satisfy other attribute requirements of melee weapons and melee skills": []mod.Mod{FLAG("StrengthSatisfiesMeleeWeaponsAndSkills")},
	`mana reservation of herald skills is always (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationPercentForced", Value: num}).Tag(mod.SkillType("Herald"))}, ""
	},
	`([\w\s]+) reserves no mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Blessing").Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Blessing").Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Blessing").Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Blessing").Neg(true)),
		}, ""
	},
	`([\w\s]+) has no reservation`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Blessing").Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Blessing").Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Blessing").Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Blessing").Neg(true)),
		}, ""
	},
	`([\w\s]+) has no reservation if cast as an aura`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Aura")).Tag(mod.SkillType("Blessing").Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Aura")).Tag(mod.SkillType("Blessing").Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Aura")).Tag(mod.SkillType("Blessing").Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType("Aura")).Tag(mod.SkillType("Blessing").Neg(true)),
		}, ""
	},
	"banner skills reserve no mana":     []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "manaReservationPercent", Value: 0}).Tag(mod.SkillType("Banner"), mod.SkillType("Blessing").Neg(true)), MOD("SkillData", "LIST", mod.SkillData{Key: "lifeReservationPercent", Value: 0}).Tag(mod.SkillType("Banner"), mod.SkillType("Blessing").Neg(true))},
	"banner skills have no reservation": []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "manaReservationPercent", Value: 0}).Tag(mod.SkillType("Banner"), mod.SkillType("Blessing").Neg(true)), MOD("SkillData", "LIST", mod.SkillData{Key: "lifeReservationPercent", Value: 0}).Tag(mod.SkillType("Banner"), mod.SkillType("Blessing").Neg(true))},
	`minions have (\d+)% reduced reservation`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Reserved", "INC", -num).Tag(mod.SkillType("Minion"))}, ""
	},
	`companions have (\d+)% reduced reservation`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Reserved", "INC", -num).Tag(mod.SkillType("CreatesCompanion"))}, ""
	},
	`undead minions have (\d+)% reduced reservation`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Reserved", "INC", -num).Tag(mod.SkillType("CreatesUndeadMinion"))}, ""
	},
	`(\d+)% increased reservation efficiency of skills which create undead minions`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReservationEfficiency", "INC", num).Tag(mod.SkillType("CreatesUndeadMinion"))}, ""
	},
	`placed banners also grant (\d+)% increased attack damage to you and allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAuraEffect", "LIST", mod.ExtraAuraEffect{Mod: MOD("Damage", "INC", num).Flag(ModFlagAttack)}).Tag(mod.Condition("BannerPlanted"), mod.SkillType("Banner"))}, ""
	},
	`banners also cause enemies to take (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAuraDebuffEffect", "LIST", LuaTable{"mod": MOD("DamageTaken", "INC", num).Tag(mod.GlobalEffect("AuraDebuff").Unscalable(true))}).Tag(mod.Condition("BannerPlanted"), mod.SkillType("Banner"))}, ""
	},
	`dread banner grants an additional \+(\d+) to maximum fortification when placing the banner`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("MaximumFortification", "BASE", num).Tag(mod.GlobalEffect("Buff"))}).Tag(mod.Condition("BannerPlanted"), mod.SkillName("Dread Banner"))}, ""
	},
	"all attacks count as empowered attacks": []mod.Mod{FLAG("Condition:Empowered").Flag(ModFlagAttack), FLAG("MaxEmpoweredUptimeRatio").Flag(ModFlagAttack)},
	"cannot use warcries":                    []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType("Warcry"))},
	"cannot use shield skills":               []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType("RequiresShield"))},
	"cannot use projectile attacks":          []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType("Attack"), mod.SkillType("Projectile"))},
	"your aura skills are disabled":          []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType("Aura"))},
	"your blessing skills are disabled":      []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType("Blessing"))},
	"your spells are disabled":               []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType("Spell"))},
	"your travel skills are disabled":        []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType("Travel"))},
	`aura skills other than ([\w\s]+) are disabled`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType("Aura"), mod.SkillType("RemoteMined").Neg(true)), FLAG("EnableSkill").Tag(mod.SkillName(captures[0]))}, ""
	},
	`travel skills other than ([\w\s]+) are disabled`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("DisableSkill").Tag(mod.SkillType("Travel")),
			FLAG("EnableSkill").Tag(mod.SkillId(extraSkillIDForName(captures[0]))),
		}, ""
	},
	`while in her embrace, take ([\d\.]+)% of your total maximum life and energy shield as fire damage per second per level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireDegen", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.Multiplier("Level"), mod.Condition("HerEmbrace")), MOD("FireDegen", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num), mod.Multiplier("Level"), mod.Condition("HerEmbrace"))}, ""
	},
	`gain her embrace for \d+ seconds when you ignite an enemy`: []mod.Mod{FLAG("Condition:CanGainHerEmbrace")},
	`when you cast a spell, sacrifice all mana to gain added maximum lightning damage equal to (\d+)% of sacrificed mana for 4 seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Condition:HaveManaStorm"), MOD("LightningMax", "BASE", 1).Tag(mod.PercentStat("ManaUnreserved", num), mod.Condition("SacrificeManaForLightning"))}, ""
	},
	`attacks with this weapon have added maximum lightning damage equal to (\d+)% of your energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningMax", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num), mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`attacks with this weapon have added maximum lightning damage equal to (\d+)% of your maximum energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningMax", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num), mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`adds (\d+)% of your maximum energy shield as cold damage to attacks with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ColdMin", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num), mod.Condition("{Hand}Attack"), mod.SkillType("Attack")), MOD("ColdMax", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num), mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`attacks with this weapon have added maximum lightning damage equal to (\d+)% of player'?s? maximum energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningMax", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num).Actor("parent"), mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`adds (\d+)% of your maximum mana as fire damage to attacks with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireMin", "BASE", 1).Tag(mod.PercentStat("Mana", num), mod.Condition("{Hand}Attack"), mod.SkillType("Attack")), MOD("FireMax", "BASE", 1).Tag(mod.PercentStat("Mana", num), mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`gain added chaos damage equal to (\d+)% of ward`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("Ward", num)), MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("Ward", num))}, ""
	},
	`spells deal added chaos damage equal to (\d+)% of your maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.SkillType("Spell")), MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.SkillType("Spell"))}, ""
	},
	`attacks have added physical damage equal to (\d+)% of maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalMin", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.SkillType("Attack")), MOD("PhysicalMax", "BASE", 1).Tag(mod.PercentStat("Life", num), mod.SkillType("Attack"))}, ""
	},
	"every 16 seconds you gain iron reflexes for 8 seconds":                                      []mod.Mod{FLAG("Condition:HaveArborix")},
	"every 16 seconds you gain elemental overload for 8 seconds":                                 []mod.Mod{FLAG("Condition:HaveAugyre")},
	"every 8 seconds, gain avatar of fire for 4 seconds":                                         []mod.Mod{FLAG("Condition:HaveVulconus")},
	"gain 0% to 40% increased movement speed at random when hit, until hit again":                []mod.Mod{FLAG("Condition:HaveGamblesprint")},
	`trigger socketed curse spell when you cast a curse spell, with a ([\d\.]+) second cooldown`: []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueCastCurseOnCurse", Level: int(1)}).Tag(mod.SocketedIn("{SlotName}"))},
	`attribute requirements can be satisfied by (\d+)% of omniscience`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("OmniAttributeRequirements", "INC", num), FLAG("OmniscienceRequirements")}, ""
	},
	"inherent bonuses from intelligence, strength and dexterity are doubled": []mod.Mod{FLAG("DoubledInherentAttributeBonuses")},
	"inherent bonuses gained from attributes are doubled":                    []mod.Mod{FLAG("DoubledInherentAttributeBonuses")},
	"you have far shot while you do not have iron reflexes":                  []mod.Mod{FLAG("FarShot").Tag(mod.Condition("HaveIronReflexes").Neg(true))},
	"you have resolute technique while you do not have elemental overload":   []mod.Mod{MOD("Keystone", "LIST", "Resolute Technique").Tag(mod.Condition("HaveElementalOverload").Neg(true))},
	"hits ignore enemy monster fire resistance while you are ignited":        []mod.Mod{FLAG("IgnoreFireResistance").Tag(mod.Condition("Ignited"))},
	"hits ignore enemy fire resistance":                                      []mod.Mod{FLAG("IgnoreFireResistance")},
	"hits ignore enemy cold resistance":                                      []mod.Mod{FLAG("IgnoreColdResistance")},
	"hits ignore enemy lightning resistance":                                 []mod.Mod{FLAG("IgnoreLightningResistance")},
	"your hits can't be evaded by blinded enemies":                           []mod.Mod{FLAG("CannotBeEvaded").Tag(mod.ActorCondition("enemy", "Blinded"))},
	"your hits cannot be evaded by pinned enemies":                           []mod.Mod{FLAG("CannotBeEvaded").Tag(mod.ActorCondition("enemy", "Pinned"))},
	"your hits cannot be evaded by heavy stunned enemies":                    []mod.Mod{FLAG("CannotBeEvaded").Tag(mod.ActorCondition("enemy", "HeavyStunned"))},
	"cannot immobilise enemies":                                              []mod.Mod{FLAG("CannotElectrocute"), FLAG("CannotFreeze"), FLAG("CannotHeavyStun"), FLAG("CannotPin")},
	`immobilise enemies at (\d+)% buildup instead of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("PoiseThreshold", "MORE", (num - 100))})}, ""
	},
	"the effect of blind on you is reversed":                        []mod.Mod{FLAG("BlindEffectReversed")},
	"blind does not affect your chance to hit":                      []mod.Mod{FLAG("IgnoreBlindHitChance")},
	"enemies blinded by you while you are blinded have malediction": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMalediction").Tag(mod.Condition("Blinded"))}).Tag(mod.Condition("Blinded"), mod.Condition("CannotBeBlinded").Neg(true))},
	"enemies blinded by you have malediction":                       []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMalediction").Tag(mod.Condition("Blinded"))})},
	"skills which throw traps have blood magic":                     []mod.Mod{FLAG("CostLifeInsteadOfMana").Tag(mod.SkillType("Trapped"))},
	"skills which throw traps cost life instead of mana":            []mod.Mod{FLAG("CostLifeInsteadOfMana").Tag(mod.SkillType("Trapped"))},
	"strength provides no bonus to maximum life":                    []mod.Mod{FLAG("NoStrBonusToLife")},
	"strength provides no inherent bonus to maximum life":           []mod.Mod{FLAG("NoStrBonusToLife")},
	"gain no inherent bonus from strength":                          []mod.Mod{FLAG("NoStrBonusToLife")},
	"intelligence provides no bonus to maximum mana":                []mod.Mod{FLAG("NoIntBonusToMana")},
	"intelligence provides no inherent bonus to maximum mana":       []mod.Mod{FLAG("NoIntBonusToMana")},
	"gain no inherent bonus from intelligence":                      []mod.Mod{FLAG("NoIntBonusToMana")},
	`strength's inherent bonus is (\d+)% increased energy shield per (\d+) strength instead`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShield", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "Str")), FLAG("NoStrBonusToLife")}, ""
	},
	`dexterity's inherent bonus is (\d+)% increased armour per (\d+) dexterity instead`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Armour", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "Dex")), FLAG("NoDexBonusToAccuracy")}, ""
	},
	`intelligence's inherent bonus is (\d+)% increased evasion rating per (\d+) intelligence instead`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Evasion", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "Int")), FLAG("NoIntBonusToMana")}, ""
	},
	`with a ghastly eye jewel socketed, minions have \+(\d+) to accuracy rating`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Accuracy", "BASE", num)}).Tag(mod.Condition("HaveGhastlyEyeJewelIn{SlotName}"))}, ""
	},
	"with a hypnotic eye jewel socketed, gain arcane surge on hit with spells": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Condition:ArcaneSurge").Tag(mod.Condition("HitSpellRecently"), mod.Condition("HaveHypnoticEyeJewelIn{SlotName}"))}, ""
	},
	`gain \d+ rage on critical hit with attacks, no more than once every [\d\.]+ seconds`: []mod.Mod{FLAG("Condition:CanGainRage")},
	`warcry skills' cooldown time is (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "OVERRIDE", num).KeywordFlag(KeywordFlagWarcry)}, ""
	},
	`warcry skills have (\+\d+) seconds to cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "BASE", num).KeywordFlag(KeywordFlagWarcry)}, ""
	},
	`(\d+)% increased total power counted by warcries`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("WarcryPower", "INC", num)}, ""
	},
	`warcries have a minimum of (\d+) power`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinimumWarcryPower", "BASE", num)}, ""
	},
	`stance skills have (\+\d+) seconds to cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "BASE", num).Tag(mod.SkillType("Stance"))}, ""
	},
	`skills have \-([\d\.]+) seconds to cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecoveryFromTemporalis", "BASE", -num)}, ""
	},
	"using warcries is instant": []mod.Mod{FLAG("InstantWarcry")},
	`attacks with axes or swords grant (\d+) rage on hit, no more than once every second`: []mod.Mod{FLAG("Condition:CanGainRage").Tag(mod.Condition("UsingAxe", "UsingSword"))},
	`regenerate (\d+) rage per second for every (\d+) life recovery per second from regeneration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RageRegen", "BASE", num).Tag(mod.PercentStat("LifeRegen", ((num / utils.Float(captures[1])) * 100))), FLAG("Condition:CanGainRage")}, ""
	},
	`regenerate (\d+) rage per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RageRegen", "BASE", num), FLAG("Condition:CanGainRage")}, ""
	},
	`your maximum energy shield is equal to (\d+)% of your strength`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShield", "OVERRIDE", 1).Tag(mod.PercentStat("Str", num))}, ""
	},
	"maximum energy shield cannot be converted":                                                                  []mod.Mod{FLAG("EnergyShieldCannotBeConverted")},
	"when you lose temporal chains you gain maximum rage":                                                        []mod.Mod{FLAG("Condition:CanGainRage")},
	`with a murderous eye jewel socketed, melee attacks grant (\d+) rage on hit, no more than once every second`: []mod.Mod{FLAG("Condition:CanGainRage").Tag(mod.Condition("HaveMurderousEyeJewelIn{SlotName}"))},
	`gain \d+ rage after spending a total of \d+ mana`:                                                           []mod.Mod{FLAG("Condition:CanGainRage")},
	"rage grants cast speed instead of attack speed":                                                             []mod.Mod{FLAG("Condition:RageCastSpeed")},
	"rage grants spell damage instead of attack damage":                                                          []mod.Mod{FLAG("Condition:RageSpellDamage")},
	`inherent loss of rage is (\d+)% slower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("InherentRageLoss", "INC", -num)}, ""
	},
	`inherent loss of rage is (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("InherentRageLoss", "INC", num)}, ""
	},
	"no inherent loss of rage": []mod.Mod{FLAG("InherentRageLossIsPrevented")},
	`inherent rage loss starts (\d+) seconds? later`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("InherentRageLossDelay", "BASE", num)}, ""
	},
	`your critical damage bonus is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritMultiplier", "OVERRIDE", num)}, ""
	},
	`base critical hit chance for attacks with weapons is ([\d\.]+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("WeaponBaseCritChance", "OVERRIDE", num)}, ""
	},
	"base critical hit chance of spells is the critical hit chance of y?o?u?r? ?main hand weapon": []mod.Mod{FLAG("BaseCritFromMainHand").Flag(ModFlagSpell)},
	`base critical hit chance for spells is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChanceBase", "OVERRIDE", num).Tag(mod.SkillType("Spell"))}, ""
	},
	"base spell critical hit chance of spells is equal to that of main hand weapon": []mod.Mod{FLAG("BaseCritFromMainHand").Flag(ModFlagSpell)},
	`critical hit chance is (\d+)% for hits with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChance", "OVERRIDE", num).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`hits with this weapon have \+(\d+)% to critical damage bonus per enemy power`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritMultiplier", "BASE", num).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"), mod.Multiplier("EnemyPower"))}, ""
	},
	`maximum critical hit chance is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChanceCap", "OVERRIDE", num)}, ""
	},
	"allocates (.+) if you have the matching modifiers? on forbidden (.+)": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GrantedAscendancyNode", "LIST", LuaTable{"side": captures[1], "name": captures[0]})}, ""
	},
	`allocates (\d+) sinister jewel sockets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GrantedPassive", "LIST", LuaTable{"type": "SinisterJewelSockets", "count": float64(num)})}, ""
	},
	"allocates (.+)": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GrantedPassive", "LIST", captures[0])}, ""
	},
	"battlemage":              []mod.Mod{FLAG("Battlemage"), MOD("MainHandWeaponDamageAppliesToSpells", "MAX", 100)},
	"transfiguration of body": []mod.Mod{FLAG("TransfigurationOfBody")},
	"transfiguration of mind": []mod.Mod{FLAG("TransfigurationOfMind")},
	"transfiguration of soul": []mod.Mod{FLAG("TransfigurationOfSoul")},
	`enemies have \-(\d+)% to total physical damage reduction against your hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyPhysicalDamageReduction", "BASE", -num)}, ""
	},
	`enemies you impale have \-(\d+)% to total physical damage reduction against impale hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyImpalePhysicalDamageReduction", "BASE", -num)}, ""
	},
	`hits with this weapon overwhelm (\d+)% physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyPhysicalDamageReduction", "BASE", -num).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`overwhelm (\d+)% physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyPhysicalDamageReduction", "BASE", -num)}, ""
	},
	`hits have (\d+)% chance to ignore enemy physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", num)}, ""
	},
	`hits with this weapon have (\d+)% chance to ignore enemy physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", num).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	"hits with this weapon ignore enemy physical damage reduction": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", 100).Flag(ModFlagHit).Tag(mod.Condition("{Hand}Attack"), mod.SkillType("Attack"))}, ""
	},
	`hits against you overwhelm (\d+)% of physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyPhysicalOverwhelm", "BASE", num)}, ""
	},
	`impale damage dealt to enemies impaled by you overwhelms (\d+)% physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyImpalePhysicalDamageReduction", "BASE", -num)}, ""
	},
	"impale damage dealt to enemies impaled by you ignores enemy physical damage reduction": []mod.Mod{FLAG("IgnoreEnemyImpalePhysicalDamageReduction")},
	`nearby enemies are crushed while you have ?a?t? least (\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Crushed")}).Tag(mod.MultiplierThreshold("RageStack").Threshold(num))}, ""
	},
	"you are crushed":                                     []mod.Mod{FLAG("Condition:Crushed")},
	"nearby enemies are crushed":                          []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Crushed")})},
	"crush enemies on hit with maces and sceptres":        []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Crushed")}).Tag(mod.Condition("UsingMace"))},
	"crushes enemies on hit":                              []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Crushed")})},
	"you have fungal ground around you while stationary":  []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("ChaosResist", "BASE", 25)}).Tag(mod.Condition("OnFungalGround", "Stationary")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ChaosResist", "BASE", -10)}).Tag(mod.ActorCondition("enemy", "OnFungalGround", "Stationary")), MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ElementalResist", "BASE", -10)}).Tag(mod.ActorCondition("enemy", "OnFungalGround", "Stationary"))},
	"create fungal ground instead of consecrated ground":  []mod.Mod{FLAG("Condition:CreateFungalGround")},
	"create profane ground instead of consecrated ground": []mod.Mod{FLAG("Condition:CreateProfaneGround")},
	`(\d+)% chance to create profane ground on critical hit if intelligence is your highest attribute`: []mod.Mod{FLAG("Condition:CreateProfaneGround").Tag(mod.Condition("IntHighestAttribute"))},
	"consecrated path and purifying flame create profane ground instead of consecrated ground":         []mod.Mod{FLAG("Condition:CreateProfaneGround")},
	"you have consecrated ground around you while stationary if strength is your highest attribute":    []mod.Mod{FLAG("Condition:OnConsecratedGround").Tag(mod.Condition("StrHighestAttribute"), mod.Condition("Stationary"))},
	`enemies have no critical damage bonus for \d+ seconds after you blind them`:                       []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.ActorCondition("enemy", "Blinded"))},
	"you count as dual wielding while you are unencumbered":                                            []mod.Mod{FLAG("Condition:DualWielding").Tag(mod.Condition("Unencumbered"))},
	`can attack as though using a quarterstaff while both of your hand slots are empty unarmed attacks that would use your quarterstaff's damage gain: physical damage based on their skill level (\d+)% more attack speed per (\d+) item evasion rating on equipped armour items \+(\d+\.?\d*)% to critical hit chance per (\d+) item energy shield on equipped armour items`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "MORE", num).Flag(ModFlagAttack).Tag(mod.Condition("HollowPalm"), mod.PerStat(utils.Float(captures[1]), "EvasionOnAllArmourItems")), MOD("CritChance", "BASE", utils.Float(captures[2])).Flag(ModFlagAttack).Tag(mod.Condition("HollowPalm"), mod.PerStat(utils.Float(captures[3]), "EnergyShieldOnAllArmourItems"))}, ""
	},
	`can attack as though using a quarterstaff while both of your hand slots are empty unarmed attacks that would use an equipped quarterstaff's damage have: base unarmed physical damage replaced with damage based on their skill level (\d+)% more attack speed per (\d+) item evasion on equipped armour items \+(\d+\.?\d*)% to critical hit chance per (\d+) item energy shield on equipped armour items`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "MORE", num).Flag(ModFlagAttack).Tag(mod.Condition("HollowPalm"), mod.PerStat(utils.Float(captures[1]), "EvasionOnAllArmourItems")), MOD("CritChance", "BASE", utils.Float(captures[2])).Flag(ModFlagAttack).Tag(mod.Condition("HollowPalm"), mod.PerStat(utils.Float(captures[3]), "EnergyShieldOnAllArmourItems"))}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:6041-6049
	`has (\d+) to (\d+) (\w+) damage, \+(\d+) to \+(\d+) per boss's face broken`: func(num float64, captures []string) ([]mod.Mod, string) {
		dmg := firstToUpper(captures[2])
		return []mod.Mod{
			MOD("Facebreaker"+dmg+"Min", "BASE", num),
			MOD("Facebreaker"+dmg+"Max", "BASE", utils.Float(captures[1])),
			MOD("Facebreaker"+dmg+"Min", "BASE", utils.Float(captures[3])).Tag(mod.Multiplier("BossFaceBroken")),
			MOD("Facebreaker"+dmg+"Max", "BASE", utils.Float(captures[4])).Tag(mod.Multiplier("BossFaceBroken")),
		}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:6050-6157
	"can attack as though using a one handed mace while both of your hand slots are empty":     []mod.Mod{FLAG("CanAttackAsOneHandMaceUnarmed")},
	"unarmed attacks that would use an equipped one hand mace's damage use this item's damage": []mod.Mod{FLAG("UseFacebreakerItemDamage")},
	`storm and plant spells: deal (\d+)% more damage cost (\d+)% less have (\d+)% less duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(KeywordFlagSpell).Tag(mod.SkillType("Storm", "Plant")), MOD("Cost", "MORE", -utils.Float(captures[1])).KeywordFlag(KeywordFlagSpell).Tag(mod.SkillType("Storm", "Plant")), MOD("Duration", "MORE", -utils.Float(captures[2])).KeywordFlag(KeywordFlagSpell).Tag(mod.SkillType("Storm", "Plant"))}, ""
	},
	"dual wielding does not inherently grant chance to block attack damage":              []mod.Mod{FLAG("Condition:NoInherentBlock")},
	"inherent attack speed bonus from dual wielding is doubled while wielding two claws": []mod.Mod{FLAG("Condition:DoubledInherentDualWieldingSpeed").Tag(mod.Condition("DualWieldingClaws"))},
	"inherent bonuses from dual wielding are doubled":                                    []mod.Mod{FLAG("Condition:DoubledInherentDualWieldingSpeed"), FLAG("Condition:DoubledInherentDualWieldingBlock")},
	`(\d+)% reduced enemy chance to block sword attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("reduceEnemyBlock", "BASE", num).Flag(ModFlagSword)}, ""
	},
	"you do not inherently take less damage for having fortification": []mod.Mod{FLAG("Condition:NoFortificationMitigation")},
	`skills supported by intensify have \+(\d) to maximum intensity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:IntensityLimit", "BASE", num)}, ""
	},
	`spells which can gain intensity have \+(\d) to maximum intensity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:IntensityLimit", "BASE", num)}, ""
	},
	`final repeat of spells has (\d+)% increased area of effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RepeatFinalAreaOfEffect", "INC", num).Flag(ModFlagSpell).Tag(mod.Condition("CastOnFrostbolt").Neg(true), mod.Condition("averageRepeat", "alwaysFinalRepeat"))}, ""
	},
	`hexes you inflict have ([\+\-]\d+) to maximum doom`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MaxDoom", "BASE", num)}, ""
	},
	`while stationary, gain (\d+)% increased area of effect every second, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Multiplier("StationarySeconds").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("ExpansiveMight"), mod.Condition("Stationary"))}, ""
	},
	`fireball and rolling magma have (\d+)% more area of effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "MORE", num).Tag(mod.SkillName("Fireball", "Rolling Magma"))}, ""
	},
	`attack skills have added lightning damage equal to (\d+)% of maximum mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningMin", "BASE", 1).Flag(ModFlagAttack).Tag(mod.PercentStat("Mana", num)), MOD("LightningMax", "BASE", 1).Flag(ModFlagAttack).Tag(mod.PercentStat("Mana", num))}, ""
	},
	`arc and crackling lance gains added cold damage equal to (\d+)% of mana cost, if mana cost is not higher than the maximum you could spend`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ColdMin", "BASE", 1).Tag(mod.PercentStat("ManaCost", num).Floor(), mod.StatThresholdStat("ManaCostPayablePool", "ManaCost"), mod.SkillName("Arc", "Crackling Lance").Transfigured(true)), MOD("ColdMax", "BASE", 1).Tag(mod.PercentStat("ManaCost", num).Floor(), mod.StatThresholdStat("ManaCostPayablePool", "ManaCost"), mod.SkillName("Arc", "Crackling Lance").Transfigured(true))}, ""
	},
	`forbidden rite and dark pact gains added chaos damage equal to (\d+)% of mana cost, if mana cost is not higher than the maximum you could spend`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("ManaCost", num).Floor(), mod.StatThresholdStat("ManaCostPayablePool", "ManaCost"), mod.SkillName("Forbidden Rite", "Dark Bargain").Transfigured(true)), MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("ManaCost", num).Floor(), mod.StatThresholdStat("ManaCostPayablePool", "ManaCost"), mod.SkillName("Forbidden Rite", "Dark Bargain").Transfigured(true))}, ""
	},
	`skills gain added chaos damage equal to (\d+)% of mana cost, if mana cost is not higher than the maximum you could spend`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("ManaCost", num).Floor(), mod.StatThresholdStat("ManaCostPayablePool", "ManaCost")), MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("ManaCost", num).Floor(), mod.StatThresholdStat("ManaCostPayablePool", "ManaCost"))}, ""
	},
	`herald of thunder's storms hit enemies with (\d+)% increased frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HeraldStormFrequency", "INC", num)}, ""
	},
	`storms hit enemies with (\d+)% increased frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HeraldStormFrequency", "INC", num)}, ""
	},
	`your critical hits have a (\d+)% chance to deal double damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DoubleDamageChanceOnCrit", "BASE", num)}, ""
	},
	"elemental skills deal triple damage":      []mod.Mod{MOD("TripleDamageChance", "BASE", 100).Tag(mod.SkillType("Cold", "Fire", "Lightning"))},
	"deal triple damage with elemental skills": []mod.Mod{MOD("TripleDamageChance", "BASE", 100).Tag(mod.SkillType("Cold", "Fire", "Lightning"))},
	`skills supported by unleash have \+(\d) to maximum number of seals`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SealCount", "BASE", num)}, ""
	},
	`skills supported by unleash have (\d+)% increased seal gain frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SealGainFrequency", "INC", num)}, ""
	},
	`(\d+)% increased critical hit chance with spells which remove the maximum number of seals`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MaxSealCrit", "INC", num)}, ""
	},
	"gain elusive on critical hit":                                     []mod.Mod{FLAG("Condition:CanBeElusive")},
	`(\d+)% chance to gain elusive when you block while dual wielding`: []mod.Mod{FLAG("Condition:CanBeElusive").Tag(mod.Condition("DualWielding"))},
	`elusive is removed from you at (\d+)% effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElusiveEffectMinThreshold", "OVERRIDE", num)}, ""
	},
	`enemies in your presence have (\w+) resistance equal to yours`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Enemy" + firstToUpper(captures[0]) + "ResistEqualToYours").Tag(mod.Condition("EnemyInPresence"))}, ""
	},
	`for each nearby corpse, regenerate ([\d\.]+)% life per second, up to ([\d\.]+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegenPercent", "BASE", num).Tag(mod.Multiplier("NearbyCorpse").Limit(utils.Float(captures[1])).LimitTotal(true))}, ""
	},
	`gain sacrificial zeal when you use a skill, dealing you \d+% of the skill's mana cost as physical damage per second`: []mod.Mod{FLAG("Condition:SacrificialZeal")},
	"attacks have blood magic":          []mod.Mod{FLAG("CostLifeInsteadOfMana").Flag(ModFlagAttack)},
	"attacks cost life instead of mana": []mod.Mod{FLAG("CostLifeInsteadOfMana").Flag(ModFlagAttack)},
	`skills gain a base life cost equal to (\d+)% of base mana cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BaseManaCostAsLifeCost", "BASE", num)}, ""
	},
	`skills gain (\d+)% of mana cost as extra life cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BaseManaCostAsLifeCost", "BASE", num)}, ""
	},
	`skills gain a base energy shield cost equal to (\d+)% of base mana cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaCostAsEnergyShieldCost", "BASE", num)}, ""
	},
	`skills cost life instead of (\d+)% of mana cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HybridManaAndLifeCost_Life", "BASE", num)}, ""
	},
	"skill mana costs converted to life costs": []mod.Mod{MOD("HybridManaAndLifeCost_Life", "BASE", 100)},
	`attack skills cost life instead of (\d+)% of mana cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HybridManaAndLifeCost_Life", "BASE", num).Flag(ModFlagAttack)}, ""
	},
	`skills from corrupted gems have (\d+)% of mana costs converted to life costs`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HybridManaAndLifeCost_Life", "BASE", num).Tag(mod.Condition("GemCorrupted"))}, ""
	},
	`non\-channelling spells cost an additional (\d+)% of your maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeCostBase", "BASE", 1).KeywordFlag(KeywordFlagSpell).Tag(mod.PercentStat("Life", num).Floor(), mod.SkillType("Channel").Neg(true))}, ""
	},
	`attacks cost an additional (\d+)% of your maximum mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaCostBase", "BASE", 1).KeywordFlag(KeywordFlagAttack).Tag(mod.PercentStat("Mana", num).Floor())}, ""
	},
	`(\d+)% increased cost of arc and crackling lance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Cost", "INC", num).Tag(mod.SkillName("Arc", "Crackling Lance").Transfigured(true))}, ""
	},
	// PoB passes nil before the tags, so they land at [2] and later and ModStore's ipairs never reads them.
	`hits overwhelm (\d+)% of physical damage reduction while you have sacrificial zeal`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyPhysicalDamageReduction", "BASE", -num)}, ""
	},
	`(\d+)% chance for hits to ignore enemy physical damage reduction while you have sacrificial zeal`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", num)}, ""
	},
	`minions attacks overwhelm (\d+)% physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnemyPhysicalDamageReduction", "BASE", -num).Tag(mod.SkillType("Attack"))})}, ""
	},
	`minions hits have (\d+)% chance to ignore enemy physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", num)})}, ""
	},
	`focus has (\d+)% increased cooldown recovery rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FocusCooldownRecovery", "INC", num).Tag(mod.Condition("Focused"))}, ""
	},
	`focus has (\d+)% reduced cooldown recovery rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FocusCooldownRecovery", "INC", -num).Tag(mod.Condition("Focused"))}, ""
	},
	`(\d+)% more frozen legion and general's cry cooldown recovery rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "MORE", num).Tag(mod.SkillName("Frozen Legion", "General's Cry").Transfigured(true))}, ""
	},
	`flamethrower, seismic and lightning spire trap have (\d+)% increased cooldown recovery rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "INC", num).Tag(mod.SkillName("Flamethrower Trap", "Seismic Trap", "Lightning Spire Trap").Transfigured(true))}, ""
	},
	`flamethrower, seismic and lightning spire trap have \-(\d+) cooldown uses?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AdditionalCooldownUses", "BASE", -num).Tag(mod.SkillName("Flamethrower Trap", "Seismic Trap", "Lightning Spire Trap").Transfigured(true))}, ""
	},
	`skills have \+(\d+) to limit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AdditionalCooldownUses", "BASE", num)}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:6158-6168
	`(\w+) skills have \+(\d+) to limit`: func(num float64, captures []string) ([]mod.Mod, string) {
		skillTypeMap := map[string]string{
			"storm":   "Storm",
			"fissure": "CreatesFissure",
			"orb":     "Orb",
		}
		skillType, ok := skillTypeMap[captures[0]]
		if !ok {
			return nil, ""
		}
		return []mod.Mod{MOD("AdditionalCooldownUses", "BASE", utils.Float(captures[1])).Tag(mod.SkillType(skillType))}, ""
	},
	//pob2:block:src/Modules/ModParser.lua:6171-6324
	`\+([\d\.]+) seconds to flameblast and incinerate cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", mod.SkillData{Key: "cooldown", Value: 0})}).Tag(mod.SkillName("Incinerate", "Flameblast").Transfigured(true)), MOD("CooldownRecovery", "BASE", num).Tag(mod.SkillName("Incinerate", "Flameblast").Transfigured(true))}, ""
	},
	`(\d+)% chance to deal double damage with attacks if attack time is longer than 1 second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DoubleDamageChance", "BASE", num).Tag(mod.Condition("OneSecondAttackTime")),
		}, ""
	},
	`elusive also grants \+(\d+)% to critical damage bonus for skills supported by nightblade`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("NightbladeElusiveCritMultiplier", "BASE", num)}, ""
	},
	`skills supported by nightblade have (\d+)% increased effect of elusive`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("NightbladeSupportedElusiveEffect", "INC", num)}, ""
	},
	"nearby enemies are scorched": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Scorched")}), MOD("ScorchBase", "BASE", 10)},
	`hits have (\d+)% chance to ignore enemy monster physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PartialIgnoreEnemyPhysicalDamageReduction", "BASE", num)}, ""
	},
	`hits ignore non\-negative elemental resistances of frozen enemies`: []mod.Mod{FLAG("IgnoreNonNegativeEleRes").Tag(mod.ActorCondition("enemy", "Frozen"))},
	`final repeat of attack skills deals (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RepeatFinalDamage", "MORE", num).Flag(ModFlagAttack)}, ""
	},
	`non\-travel attack skills repeat an additional time`: []mod.Mod{MOD("RepeatCount", "BASE", 1).Flag(ModFlagAttack).Tag(mod.Condition("averageRepeat", "alwaysFinalRepeat"))},
	`viper strike and pestilent strike deal (\d+)% increased attack damage per frenzy charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Flag(ModFlagAttack).Tag(mod.Multiplier("FrenzyCharge"), mod.SkillName("Viper Strike", "Pestilent Strike").Transfigured(true))}, ""
	},
	`shield charge and chain hook have (\d+)% increased attack speed per (\d+) rampage kills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "INC", num).Flag(ModFlagAttack).Tag(mod.Multiplier("Rampage").Div(utils.Float(captures[1])).Limit((1000 / utils.Float(captures[1]))).LimitTotal(true), mod.SkillName("Shield Charge", "Chain Hook").Transfigured(true))}, ""
	},
	`tectonic slam and infernal blow deal (\d+)% increased attack damage per (\d+) armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Flag(ModFlagAttack).Tag(mod.PerStat(utils.Float(captures[1]), "Armour"), mod.SkillName("Tectonic Slam", "Infernal Blow").Transfigured(true))}, ""
	},
	`frozen sweep deals (\d+)% less damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", -num).Tag(mod.SkillName("Frozen Sweep").Transfigured(true))}, ""
	},
	`ice trap and lightning trap damage penetrates (\d+)% of enemy elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningPenetration", "BASE", num).Tag(mod.SkillName("Ice Trap", "Lightning Trap").Transfigured(true)), MOD("ColdPenetration", "BASE", num).Tag(mod.SkillName("Ice Trap", "Lightning Trap").Transfigured(true)), MOD("FirePenetration", "BASE", num).Tag(mod.SkillName("Ice Trap", "Lightning Trap").Transfigured(true))}, ""
	},
	`volatile dead and cremation penetrate (\d+)% fire resistance per (\d+) dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FirePenetration", "BASE", num).Tag(mod.PerStat(utils.Float(captures[1]), "Dex"), mod.SkillName("Volatile Dead", "Cremation").Transfigured(true))}, ""
	},
	`regenerate (\d+) mana per second while any enemy is in your righteous fire or scorching ray`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaRegen", "BASE", num).Tag(mod.Condition("InRFOrScorchingRay"))}, ""
	},
	`\+(\d+)% to wave of conviction damage over time multiplier per ([\d\.]+) seconds of duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("WaveOfConvictionDurationDotMulti", "INC", num)}, ""
	},
	`when an enemy hit deals elemental damage to you, their resistance to those elements becomes zero for (\d+) seconds`: []mod.Mod{FLAG("Condition:HaveTrickstersSmile")},
	`skills have a (\d+)% longer perfect timing window`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PerfectTiming", "INC", num)}, ""
	},
	`deal (\d+)% increased damage with hits to rare or unique enemies for each second they've ever been in your presence, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(KeywordFlagHit).Tag(mod.Multiplier("EnemyPresenceSeconds").Limit(utils.Float(captures[1])).LimitTotal(true).Actor("enemy"), mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	`attacks have added maximum lightning damage equal to (\d+)% of maximum mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningMax", "BASE", 1).Tag(mod.PercentStat("Mana", num), mod.SkillType("Attack"))}, ""
	},
	`(\d+)% reduced movement speed penalty from using skills while moving`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeedPenalty", "INC", -num)}, ""
	},
	`(\d+)% reduced movement speed penalty while actively blocking`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeedPenalty", "INC", -num).Tag(mod.SkillType("ActiveBlock"))}, ""
	},
	`(\d+)% less movement speed penalty from using skills while moving`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeedPenalty", "MORE", -num)}, ""
	},
	"no movement speed penalty while shield is raised": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeedPenalty", "MORE", -100).Tag(mod.SkillType("ActiveBlock"))}, ""
	},
	`(\d+)% increased quantity of items dropped by slain normal enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LootQuantityNormalEnemies", "INC", num)}, ""
	},
	`(\d+)% increased rarity of items dropped by slain magic enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LootRarityMagicEnemies", "INC", num)}, ""
	},
	`(\d+)% increased decoy totem life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TotemLife", "INC", num).Tag(mod.SkillName("Decoy Totem"))}, ""
	},
	`shock nova ring deals (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Tag(mod.SkillName("Shock Nova").Transfigured(true), mod.SkillPart(int(1)))}, ""
	},
	`enemies affected by bear trap take (\d+)% increased damage from trap or mine hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("TrapMineDamageTaken", "INC", num).Tag(mod.GlobalEffect("Debuff"))})}).Tag(mod.SkillName("Bear Trap").Transfigured(true))}, ""
	},
	`blade vortex has \+(\d+)% to critical damage bonus for each blade`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritMultiplier", "BASE", num).Tag(mod.Multiplier("BladeVortexBlade"), mod.SkillName("Blade Vortex").Transfigured(true))}, ""
	},
	`burning arrow has (\d+)% increased debuff effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DebuffEffect", "INC", num).Tag(mod.SkillName("Burning Arrow"))}, ""
	},
	`double strike has a (\d+)% chance to deal double damage to bleeding enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DoubleDamageChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Bleeding"), mod.SkillName("Double Strike").Transfigured(true))}, ""
	},
	`frost bomb has (\d+)% increased debuff duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SecondaryDuration", "INC", num).Tag(mod.SkillName("Frost Bomb"))}, ""
	},
	`incinerate has \+(\d+) to maximum stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:IncinerateMaxStages", "BASE", num).Tag(mod.SkillName("Incinerate"))}, ""
	},
	`perforate creates \+(\d+) spikes?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:PerforateMaxSpikes", "BASE", num)}, ""
	},
	`scourge arrow has (\d+)% chance to poison per stage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PoisonChance", "BASE", num).Tag(mod.SkillName("Scourge Arrow").Transfigured(true), mod.Multiplier("ScourgeArrowStage"))}, ""
	},
	`winter orb has \+(\d+) maximum stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:WinterOrbMaxStages", "BASE", num)}, ""
	},
	`summoned holy relics have (\d+)% increased buff effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BuffEffect", "INC", num).Tag(mod.SkillName("Summon Holy Relic"))}, ""
	},
	`\+(\d) to maximum virulence`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:VirulenceStacksMax", "BASE", num)}, ""
	},
	`winter orb has (\d+)% increased area of effect per stage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.SkillName("Winter Orb"), mod.Multiplier("WinterOrbStage"))}, ""
	},
	`wintertide brand has \+(\d+) to maximum stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:WintertideBrandMaxStages", "BASE", num).Tag(mod.SkillName("Wintertide Brand"))}, ""
	},
	`wave of conviction's exposure applies (\-\d+)% elemental resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "purge_expose_resist_%_matching_highest_element_damage", Value: float64(num)}).Tag(mod.SkillName("Wave of Conviction"))}, ""
	},
	`wave of conviction's exposure applies an extra (\-\d+)% to elemental resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "purge_expose_resist_%_matching_highest_element_damage", Value: float64(num)}).Tag(mod.SkillName("Wave of Conviction"))}, ""
	},
	`arcane cloak spends an additional (\d+)% of current mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "arcane_cloak_consume_%_of_mana", Value: float64(num)}).Tag(mod.SkillName("Arcane Cloak"))}, ""
	},
	`arcane cloak grants life regeneration equal to (\d+)% of mana spent per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("LifeRegen", "BASE", num/100).Tag(mod.Multiplier("ArcaneCloakConsumedMana").Base(0), mod.GlobalEffect("Buff"))}).Tag(mod.SkillName("Arcane Cloak")),
		}, ""
	},
	`caustic arrow has (\d+)% chance to inflict withered on hit for (\d+) seconds base duration`: []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: FLAG("Condition:CanWither")}).Tag(mod.SkillName("Caustic Arrow").Transfigured(true))},
	`venom gyre has a (\d+)% chance to inflict withered for (\d+) seconds on hit`:                []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: FLAG("Condition:CanWither")}).Tag(mod.SkillName("Venom Gyre"))},
	`cobra lash chains (\d+) additional times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("ChainCountMax", "BASE", num)}).Tag(mod.SkillName("Cobra Lash"))}, ""
	},
	`general's cry has ([\+\-]\d) to maximum number of mirage warriors`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GeneralsCryDoubleMaxCount", "BASE", num)}, ""
	},
	`([\+\-]\d) to maximum blade flurry stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:BladeFlurryMaxStages", "BASE", num), MOD("Multiplier:BladeFlurryofIncisionMaxStages", "BASE", num)}, ""
	},
	`steelskin buff can take (\d+)% increased amount of damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "steelskin_damage_limit_+%", Value: float64(num)}).Tag(mod.SkillName("Steelskin"))}, ""
	},
	`hydrosphere has (\d+)% increased pulse frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HydroSphereFrequency", "INC", num)}, ""
	},
	`void sphere has (\d+)% increased pulse frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("VoidSphereFrequency", "INC", num)}, ""
	},
	`shield crush central wave has (\d+)% more area of effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "MORE", num).Tag(mod.SkillName("Shield Crush").Transfigured(true), mod.SkillPart(int(2)))}, ""
	},
	`storm rain has (\d+)% increased beam frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("StormRainBeamFrequency", "INC", num)}, ""
	},
	`voltaxic burst deals (\d+)% increased damage per ([\d\.]+) seconds of duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("VoltaxicDurationIncDamage", "INC", num)}, ""
	},
	`earthquake deals (\d+)% increased damage per ([\d\.]+) seconds duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EarthquakeDurationIncDamage", "INC", num)}, ""
	},
	`consecrated ground from holy flame totem applies (\d+)% increased damage taken to enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTakenConsecratedGround", "INC", num).Tag(mod.Condition("OnConsecratedGround"))})}, ""
	},
	`consecrated ground from purifying flame applies (\d+)% increased damage taken to enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "consecrated_ground_enemy_damage_taken_+%", Value: float64(num)}).Tag(mod.SkillName("Purifying Flame").Transfigured(true))}, ""
	},
	`enemies drenched by hydrosphere have cold and lightning exposure, applying (\-\d+)% to resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "water_sphere_cold_lightning_exposure_%", Value: float64(num)}).Tag(mod.SkillName("Hydrosphere"))}, ""
	},
	`frost shield has \+(\d+) to maximum life per stage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "frost_globe_health_per_stage", Value: float64(num)}).Tag(mod.SkillName("Frost Shield"))}, ""
	},
	`flame wall grants (\d+) to (\d+) added fire damage to projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "flame_wall_minimum_added_fire_damage", Value: captures[0]}).Tag(mod.SkillName("Flame Wall")), MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "flame_wall_maximum_added_fire_damage", Value: captures[1]}).Tag(mod.SkillName("Flame Wall"))}, ""
	},
	`plague bearer buff grants \+(\d+)% to poison damage over time multiplier while infecting`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "corrosive_shroud_poison_dot_multiplier_+_while_aura_active", Value: float64(num)}).Tag(mod.SkillName("Plague Bearer"))}, ""
	},
	`(\d+)% increased lightning trap lightning ailment effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "shock_effect_+%", Value: float64(num)}).Tag(mod.SkillName("Lightning Trap").Transfigured(true))}, ""
	},
	`wild strike's beam chains an additional (\d+) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("ChainCountMax", "BASE", num)}).Tag(mod.SkillName("Wild Strike").Transfigured(true), mod.SkillPart(int(4)))}, ""
	},
	`energy blades have (\d+)% increased attack speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyBladeAttackSpeed", "INC", num)}, ""
	},
	`ensnaring arrow has (\d+)% increased debuff effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DebuffEffect", "INC", num).Tag(mod.SkillName("Ensnaring Arrow"))}, ""
	},
	`unearth spawns corpses with ([\+\-]\d) level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CorpseLevel", "BASE", num).Tag(mod.SkillName("Unearth"))}, ""
	},
	"seismic trap releases an additional wave":        []mod.Mod{MOD("MaximumWaves", "BASE", 1).Tag(mod.SkillName("Seismic Trap").Transfigured(true))},
	"lightning spire trap strikes an additional area": []mod.Mod{MOD("MaximumWaves", "BASE", 1).Tag(mod.SkillName("Lightning Spire Trap").Transfigured(true))},
	`explosive trap causes (\d+) additional smaller explosions`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SmallExplosions", "BASE", num).Tag(mod.SkillName("Explosive Trap", "Explosive Trap of Swells"))}, ""
	},
	`frozen sweep deals (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Tag(mod.SkillName("Frozen Sweep").Transfigured(true))}, ""
	},
	`(\d+)% increased attack speed with snipe`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "INC", num).Flag(ModFlagAttack).Tag(mod.SkillName("Snipe"))}, ""
	},
	`\+(\d+) to maximum snipe stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:SnipeStagesMax", "BASE", num).Tag(mod.GlobalEffect("Buff").Unscalable(true))}, ""
	},
	`chain hook has \+([\d\.]+) metres? to radius per (\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "BASE", (num*10)).Tag(mod.PerStat(utils.Float(captures[1]), "Rage"), mod.SkillName("Chain Hook").Transfigured(true))}, ""
	},
	`\+([\d\.]+) metres? to discharge radius`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "BASE", (num * 10)).Tag(mod.SkillName("Discharge").Transfigured(true))}, ""
	},
	"increases and reductions to life regeneration rate also apply to mana regeneration rate": []mod.Mod{FLAG("LifeRegenAppliesToManaRegen")},
	"increases and reductions to movement speed also apply to energy shield recharge rate":    []mod.Mod{FLAG("MovementSpeedAppliesToEnergyShieldRecharge")},
	"quality does not increase physical damage":                                               []mod.Mod{MOD("AlternateQualityWeapon", "BASE", 1)},
	`(\d+)% increased critical hit chance per 4% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AlternateQualityLocalCritChancePer4Quality", "INC", num)}, ""
	},
	`grants (\d+)% increased accuracy per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Accuracy", "INC", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`(\d+)% increased attack speed per 8% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AlternateQualityLocalAttackSpeedPer8Quality", "INC", num)}, ""
	},
	`\+(\d+) weapon range per 10% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AlternateQualityLocalWeaponRangePer10Quality", "BASE", num)}, ""
	},
	`\+([\d\.]+) metres? to weapon range per 10% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AlternateQualityLocalWeaponRangePer10Quality", "BASE", (num * 10))}, ""
	},
	`grants (\d+)% increased elemental damage per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalDamage", "INC", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`grants (\d+)% increased area of effect per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	"quality does not increase defences": []mod.Mod{MOD("AlternateQualityArmour", "BASE", 1)},
	`grants \+(\d+) to maximum life per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Life", "BASE", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`grants \+(\d+) to maximum mana per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Mana", "BASE", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`grants \+(\d+) to strength per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Str", "BASE", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`grants \+(\d+) to dexterity per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Dex", "BASE", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`grants \+(\d+) to intelligence per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Int", "BASE", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`grants \+(\d+)% to fire resistance per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireResist", "BASE", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`grants \+(\d+)% to cold resistance per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ColdResist", "BASE", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`grants \+(\d+)% to lightning resistance per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LightningResist", "BASE", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`\+(\d+)% to quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Quality", "BASE", num)}, ""
	},
	`infernal blow debuff deals an additional (\d+)% of damage per charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DebuffEffect", "BASE", num).Tag(mod.SkillName("Infernal Blow").Transfigured(true))}, ""
	},
	`remembrancing (\d+) songworthy deeds by the line of (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "conqueredBy", Value: mod.LegionJewel{ID: int(num), Conqueror: conquerorList[strings.ToLower(captures[1])]}})}, ""
	},
	`glorifying the defilement of (\d+) souls in tribute to (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "conqueredBy", Value: mod.LegionJewel{ID: int(num), Conqueror: conquerorList[strings.ToLower(captures[1])]}})}, ""
	},
	`passives in radius are conquered by the (\D+)`: []mod.Mod{},
	"historic":   []mod.Mod{},
	"extra gore": []mod.Mod{},
	"prefixes:":  []mod.Mod{},
	"suffixes:":  []mod.Mod{},
	"while your passive skill tree connects to a class' starting location, you gain:":       []mod.Mod{},
	`socketed lightning spells [hd][ae][va][el] (\d+)% increased spell damage if triggered`: []mod.Mod{},
	"manifeste?d? dancing dervishe?s? disables both weapon slots":                           []mod.Mod{},
	"manifeste?d? dancing dervishe?s? dies? when rampage ends":                              []mod.Mod{},
	"survival": []mod.Mod{},
	"you can have two different banners at the same time": []mod.Mod{},
	`[\+\-](\d+) prefix modifiers? allowed`:               []mod.Mod{},
	`[\+\-](\d+) suffix modifiers? allowed`:               []mod.Mod{},
	"can have a second enchantment modifier":              []mod.Mod{},
	`can have (\d+) additional enchantment modifiers`:     []mod.Mod{},
	"this item can be anointed by cassia":                 []mod.Mod{},
	"implicit modifiers cannot be changed":                []mod.Mod{},
	"all sockets are white":                               []mod.Mod{},
	`[\+\-](\d+) to any attribute`:                        []mod.Mod{},
	`every (\d+) seconds, regenerate (\d+)% of life over one second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegenPercent", "BASE", utils.Float(captures[1])).Tag(mod.Condition("LifeRegenBurstFull")), MOD("LifeRegenPercent", "BASE", (utils.Float(captures[1]) / num)).Tag(mod.Condition("LifeRegenBurstAvg"))}, ""
	},
	"take no extra damage from critical hits": []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	//pob2:block:src/Modules/ModParser.lua:6328-6359
	"take no extra damage from critical hits if you have a magic ring in left slot": []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.GlobalEffect("Global").Unscalable(true), mod.Condition("MagicItemInRing 1"))},
	`you take (\d+)% reduced extra damage from critical hits while affected by determination`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.Condition("AffectedByDetermination"))}, ""
	},
	`you take (\d+)% reduced extra damage from critical hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", num)}, ""
	},
	`you take (\d+)% increased extra damage from critical hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", -num)}, ""
	},
	`you take (\d+)% reduced extra damage from critical hits while you have no power charges`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.StatThreshold("PowerCharges", 0).Upper(true))}, ""
	},
	`you take (\d+)% reduced extra damage from critical hits by poisoned enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.ActorCondition("enemy", "Poisoned"))}, ""
	},
	`you take (\d+)% reduced extra damage from critical hits by cursed enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`nearby allies have (\d+)% chance to block attack damage per (\d+) strength you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("BlockChance", "BASE", num), OnlyAllies: true}).Tag(mod.PerStat(utils.Float(captures[1]), "Str"))}, ""
	},
	"you can socket an additional copy of each lineage support gem, in different skills": []mod.Mod{MOD("MaxLineageCount", "BASE", 1)},
	`you can socket (\d+) additional copies of each lineage support gem, in different skills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MaxLineageCount", "BASE", num)}, ""
	},
	`convert ([\d\.]+)% of maximum life to twice as much armour per ([\d\.]+)% chaos resistance above 0%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeConvertToArmour", "BASE", num).Tag(mod.PerStat(utils.Float(captures[1]), "ChaosResist")), MOD("LifeGainAsArmour", "BASE", num).Tag(mod.PerStat(utils.Float(captures[1]), "ChaosResist"))}, ""
	},
	`\d+% [ir][ne][cd][ru][ec][ae][sd]e?d? ?[\w\s]* modifier magnitudes`: []mod.Mod{},
	`\d+% [ir][ne][cd][ru][ec][ae][sd]e?d? effect of [sp][ur][fe]fixes`:  []mod.Mod{},
	`[\w\s]* modifier magnitudes are doubled`:                            []mod.Mod{},
	"can be modified while corrupted":                                    []mod.Mod{},
	"can tattoo runes onto your body, gaining":                           []mod.Mod{FLAG("SocketRunesOnCharacter")},
	`additional rune\-only sockets:`:                                     []mod.Mod{},
	`(\d+) helmet sockets?`:                                              []mod.Mod{},
	`(\d+) body armour sockets?`:                                         []mod.Mod{},
	`(\d+) gloves sockets?`:                                              []mod.Mod{},
	`(\d+) boots sockets?`:                                               []mod.Mod{},
	`raven\-touched`:                                                     []mod.Mod{},
}

// for _, name in ipairs(data.keystones) do
//
//	flag("Condition:Have"..firstToUpper(name):gsub(" %l", string.upper):gsub(" ", "")) }
//
// end
//
//pob2:block:src/Modules/Data.lua:371-407
var dataKeystones = []string{
	"Acrobatics", "Ancestral Bond", "Avatar of Fire", "Blackflame Covenant", "Blood Magic", "Bulwark",
	"Chaos Inoculation", "Conduit", "Crimson Assault", "Dance with Death", "Eldritch Battery", "Elemental Equilibrium",
	"Eternal Youth", "Giant's Blood", "Glancing Blows", "Heartstopper", "Hollow Palm Technique", "Iron Reflexes",
	"Lord of the Wilds", "Mind Over Matter", "Necromantic Talisman", "Oasis", "Pain Attunement", "Primal Hunger",
	"Resolute Technique", "Resonance", "Ritual Cadence", "Scarred Faith", "Trusted Kinship", "Unwavering Stance",
	"Vaal Pact", "Walker of the Wilds", "Whispers of Doom", "Wildsurge Incantation", "Zealot's Oath",
}

//pob2:block:src/Modules/ModParser.lua:6361-6363
func init() {
	for _, name := range dataKeystones {
		specialModList[strings.ToLower(name)] = []mod.Mod{
			MOD("Keystone", "LIST", name),
			mod.NewFlag("Condition:Have"+keystoneConditionSuffix(name), true),
		}
	}
}

func keystoneConditionSuffix(name string) string {
	r := []rune(firstToUpper(name))
	for i := 0; i+1 < len(r); i++ {
		if r[i] == ' ' {
			r[i+1] = unicode.ToUpper(r[i+1])
		}
	}
	return strings.ReplaceAll(string(r), " ", "")
}

// Modifiers that are recognised but unsupported
var unsupportedModList = map[string]bool{
	//pob2:block:src/Modules/ModParser.lua:6396-6396
	"mirrored": true,
}

// Special lookups used for various modifier forms
var suffixTypesCompiled map[string]CompiledList[string]
var suffixTypes = map[string]string{
	//pob2:block:src/Modules/ModParser.lua:6401-6441
	"as extra physical damage":            "GainAsPhysical",
	"added as physical damage":            "GainAsPhysical",
	"gained as extra physical damage":     "GainAsPhysical",
	"as extra lightning damage":           "GainAsLightning",
	"added as lightning damage":           "GainAsLightning",
	"gained as extra lightning damage":    "GainAsLightning",
	"as extra cold damage":                "GainAsCold",
	"added as cold damage":                "GainAsCold",
	"gained as extra cold damage":         "GainAsCold",
	"as extra fire damage":                "GainAsFire",
	"added as fire damage":                "GainAsFire",
	"gained as extra fire damage":         "GainAsFire",
	"as extra chaos damage":               "GainAsChaos",
	"added as chaos damage":               "GainAsChaos",
	"gained as extra chaos damage":        "GainAsChaos",
	"as extra damage of a random element": "GainAsRandom",
	"converted to lightning":              "ConvertToLightning",
	"converted to lightning damage":       "ConvertToLightning",
	"converted to cold damage":            "ConvertToCold",
	"converted to fire damage":            "ConvertToFire",
	"converted to fire":                   "ConvertToFire",
	"converted to chaos damage":           "ConvertToChaos",
	"added as energy shield":              "GainAsEnergyShield",
	"as extra maximum energy shield":      "GainAsEnergyShield",
	"as extra maximum runic ward":         "GainAsWard",
	"converted to energy shield":          "ConvertToEnergyShield",
	"as armour":                           "GainAsArmour",
	"as extra armour":                     "GainAsArmour",
	"as physical damage":                  "AsPhysical",
	"as lightning damage":                 "AsLightning",
	"as cold damage":                      "AsCold",
	"as fire damage":                      "AsFire",
	"as fire":                             "AsFire",
	"as chaos damage":                     "AsChaos",
	"leeched as life and mana":            "Leech",
	"leeched as life":                     "LifeLeech",
	"is leeched as life":                  "LifeLeech",
	"leeched as mana":                     "ManaLeech",
	"is leeched as mana":                  "ManaLeech",
	"leeched as energy shield":            "EnergyShieldLeech",
	"is leeched as energy shield":         "EnergyShieldLeech",
}

var dmgTypesCompiled map[string]CompiledList[string]
var dmgTypes = map[string]string{
	//pob2:block:src/Modules/ModParser.lua:6444-6448
	"physical":  "Physical",
	"lightning": "Lightning",
	"cold":      "Cold",
	"fire":      "Fire",
	"chaos":     "Chaos",
}

var penTypesCompiled map[string]CompiledList[modNameListType]
var penTypes = map[string]modNameListType{
	//pob2:block:src/Modules/ModParser.lua:6451-6456
	"lightning resistance":  {names: []string{"LightningPenetration"}},
	"cold resistance":       {names: []string{"ColdPenetration"}},
	"fire resistance":       {names: []string{"FirePenetration"}},
	"elemental resistance":  {names: []string{"ElementalPenetration"}},
	"elemental resistances": {names: []string{"ElementalPenetration"}},
	"chaos resistance":      {names: []string{"ChaosPenetration"}},
}

var regenTypesCompiled map[string]CompiledList[modNameListType]

var regenTypes = resourceTypesWithSuffix("Regen")

var degenTypesCompiled map[string]CompiledList[modNameListType]
var degenTypes = resourceTypesWithSuffix("Degen")

var costTypesCompiled map[string]CompiledList[modNameListType]
var costTypes = resourceTypesWithSuffix("Cost")

var baseCostTypesCompiled map[string]CompiledList[modNameListType]
var baseCostTypes = resourceTypesWithSuffix("CostNoMult")

var resourceTypes = withMaximumResourceTypes(resourceTypesBase)

func withMaximumResourceTypes(base map[string][]string) map[string][]string {
	out := make(map[string][]string, len(base)*2)
	for line, names := range base {
		out[line] = names
		out["maximum "+line] = names
	}
	return out
}

var resourceTypesBase = map[string][]string{
	//pob2:block:src/Modules/ModParser.lua:6459-6475
	"life":                         {"Life"},
	"mana":                         {"Mana"},
	"energy shield":                {"EnergyShield"},
	"runic ward":                   {"Ward"},
	"life and mana":                {"Life", "Mana"},
	"life and energy shield":       {"Life", "EnergyShield"},
	"life, mana and energy shield": {"Life", "Mana", "EnergyShield"},
	"life, energy shield and mana": {"Life", "Mana", "EnergyShield"},
	"mana and life":                {"Life", "Mana"},
	"mana and energy shield":       {"Mana", "EnergyShield"},
	"mana, life and energy shield": {"Life", "Mana", "EnergyShield"},
	"mana, energy shield and life": {"Life", "Mana", "EnergyShield"},
	"energy shield and life":       {"Life", "EnergyShield"},
	"energy shield and mana":       {"Mana", "EnergyShield"},
	"energy shield, life and mana": {"Life", "Mana", "EnergyShield"},
	"energy shield, mana and life": {"Life", "Mana", "EnergyShield"},
	"rage":                         {"Rage"},
}

//pob2:src/Modules/ModParser.lua:6486-6499
func resourceTypesWithSuffix(suffix string) map[string]modNameListType {
	out := make(map[string]modNameListType, len(resourceTypes))
	for line, names := range resourceTypes {
		suffixed := make([]string, len(names))
		for i, n := range names {
			suffixed[i] = n + suffix
		}
		out[line] = modNameListType{names: suffixed}
	}

	return out
}

var flagTypesCompiled map[string]CompiledList[modNameListType]
var flagTypes = map[string]modNameListType{
	//pob2:block:src/Modules/ModParser.lua:6505-6530
	"phasing":                       {names: []string{"Condition:Phasing"}},
	"onslaught":                     {names: []string{"Condition:Onslaught"}},
	"rampage":                       {names: []string{"Condition:Rampage"}},
	"soul eater":                    {names: []string{"Condition:CanHaveSoulEater"}},
	"adrenaline":                    {names: []string{"Condition:Adrenaline"}},
	"elusive":                       {names: []string{"Condition:CanBeElusive"}},
	"arcane surge":                  {names: []string{"Condition:ArcaneSurge"}},
	"fortify":                       {names: []string{"Condition:Fortified"}},
	"fortified":                     {names: []string{"Condition:Fortified"}},
	"unholy might":                  {names: []string{"Condition:UnholyMight"}},
	"chaotic might":                 {names: []string{"Condition:ChaoticMight"}},
	"lesser brutal shrine buff":     {names: []string{"Condition:LesserBrutalShrine"}},
	"lesser massive shrine buff":    {names: []string{"Condition:LesserMassiveShrine"}},
	"tailwind":                      {names: []string{"Condition:CanHaveTailwind"}},
	"intimidated":                   {names: []string{"Condition:Intimidated"}},
	"crushed":                       {names: []string{"Condition:Crushed"}},
	"chilled":                       {names: []string{"Condition:Chilled"}},
	"blinded":                       {names: []string{"Condition:Blinded"}},
	"no life regeneration":          {names: []string{"NoLifeRegen"}},
	"no inherent mana regeneration": {names: []string{"Condition:NoInherentManaRegen"}},
	"hexproof":                      {names: []string{"CurseEffectOnSelf"}, flagModType: "MORE", flagModValue: -100},
	"maimed":                        {names: []string{"Condition:Maimed"}},
	`hindered,? with (\d+)% reduced movement speed`: {names: []string{"Condition:Hindered"}},
	"hindered":    {names: []string{"Condition:Hindered"}},
	"unnerved":    {names: []string{"Condition:Unnerved"}},
	"malediction": {names: []string{"HasMalediction"}},
}

// Build active skill name lookup
var skillNameListCompiled map[string]CompiledList[modNameListType]
var preSkillNameListCompiled map[string]CompiledList[modNameListType]

// poe2.Gem carries no skillTypes or statSets, so the Buff "grants" rows (6577-6586) are not built.
//
//pob2:block:src/Modules/ModParser.lua:6557-6621
func initializeSkillNameList() {
	skillNameListCompiled = make(map[string]CompiledList[modNameListType])
	skillNameListCompiled[" corpse cremation "] = CompiledList[modNameListType]{
		Regex: lazyRegex(` corpse cremation `),
		Value: modNameListType{
			tag: mod.SkillName("Cremation").Transfigured(true),
		},
	}

	preSkillNameListCompiled = make(map[string]CompiledList[modNameListType])
	deprecatedSkillNames := map[string]bool{"Flammability": true}
	ids := make([]int, len(poe2.Gems))
	for i := range ids {
		ids[i] = i
	}
	sort.SliceStable(ids, func(a, b int) bool { return poe2.Gems[ids[a]].ID < poe2.Gems[ids[b]].ID })
	for _, gi := range ids {
		gemData := &poe2.Gems[gi]
		skillName := gemData.BaseTypeName
		if skillName == "" {
			skillName = gemData.GrantedEffectName
		}
		if gemData.Support || skillName == "" || deprecatedSkillNames[skillName] {
			continue
		}
		if ov, ok := poe2.SkillOverlayByID(gemData.GrantedEffectID); ok && (ov.Hidden || ov.FromItem) {
			continue
		}
		tags := make(map[string]bool, len(gemData.Tags))
		for _, t := range gemData.Tags {
			tags[t] = true
		}
		skillNameLower := strings.ToLower(skillName)
		val := modNameListType{
			tag: mod.SkillName(skillName).Transfigured(true),
		}
		add := func(dst map[string]CompiledList[modNameListType], pattern string, v modNameListType) {
			dst[pattern] = CompiledList[modNameListType]{Regex: lazyRegex(pattern), Value: v}
		}
		add(skillNameListCompiled, " "+skillNameLower+" ", val)
		add(preSkillNameListCompiled, "^"+skillNameLower+" ", val)
		add(preSkillNameListCompiled, "^"+skillNameLower+" has ?a? ", val)
		add(preSkillNameListCompiled, "^"+skillNameLower+" deals ", val)
		add(preSkillNameListCompiled, "^"+skillNameLower+" damage ", val)
		if tags["totem"] {
			add(preSkillNameListCompiled, "^"+skillNameLower+" totem deals ", val)
			add(preSkillNameListCompiled, "^"+skillNameLower+" totem grants ", modNameListType{
				addToSkill: mod.SkillName(skillName).Transfigured(true),
				tag:        mod.GlobalEffect("Buff"),
			})
		}
		if tags["aura"] || tags["herald"] {
			affectedBy := modNameListType{
				tag: mod.Condition("AffectedBy" + strings.ReplaceAll(skillName, " ", "")),
			}
			add(skillNameListCompiled, "while affected by "+skillNameLower, affectedBy)
			add(skillNameListCompiled, "while using "+skillNameLower, affectedBy)
		}
		if tags["curse"] {
			add(skillNameListCompiled, "if you've cast "+skillNameLower+" in the past (\\d+) seconds", modNameListType{
				tag: mod.Condition(selfCastCondition(skillName)),
			})
		}
		addSkillSpecial(skillNameLower, tags, skillName)
	}

	preSkillNameIndex = newAnchoredIndex(preSkillNameListCompiled)
	skillNameIndex = newAnyOffsetIndex(skillNameListCompiled)
	specialModIndex = newAnchoredIndex(specialModListCompiled)
}
func addSkillSpecial(skillNameLower string, tags map[string]bool, skillName string) {
	add := func(pattern string, value interface{}) {
		specialModListCompiled[pattern] = CompiledList[interface{}]{
			Regex: lazyRegex(pattern),
			Value: value,
		}
	}
	extra := func(inner mod.Mod) []mod.Mod {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: inner}).Tag(mod.SkillName(skillName).Transfigured(true))}
	}
	extraFunc := func(name string, modType mod.Type) func(float64, []string) ([]mod.Mod, string) {
		return func(num float64, captures []string) ([]mod.Mod, string) {
			return extra(MOD(name, modType, num)), ""
		}
	}

	if tags["mine"] {
		add("^"+skillNameLower+` has (\d+)% increased throwing speed`, extraFunc("MineLayingSpeed", "INC"))
	}
	if tags["trap"] {
		add(`(\d+)% increased `+skillNameLower+" throwing speed", extraFunc("TrapThrowingSpeed", "INC"))
	}
	if tags["chaining"] {
		add("^"+skillNameLower+" chains an additional time", extra(MOD("ChainCountMax", "BASE", 1)))
		add("^"+skillNameLower+` chains an additional (\d+) times`, extraFunc("ChainCountMax", "BASE"))
		add("^"+skillNameLower+` chains (\d+) additional times`, extraFunc("ChainCountMax", "BASE"))
	}
	if tags["bow"] {
		add("^"+skillNameLower+" fires an additional arrow",
			extra(MOD("ProjectileCount", "BASE", 1).KeywordFlag(KeywordFlagArrow)))
		add("^"+skillNameLower+` fires (\d+) additional arrows?`, func(num float64, captures []string) ([]mod.Mod, string) {
			return extra(MOD("ProjectileCount", "BASE", num).KeywordFlag(KeywordFlagArrow)), ""
		})
	}
	if tags["projectile"] {
		add("^"+skillNameLower+" pierces an additional target",
			[]mod.Mod{MOD("PierceCount", "BASE", 1).Tag(mod.SkillName(skillName).Transfigured(true))})
		add("^"+skillNameLower+` pierces (\d+) additional targets?`,
			func(num float64, captures []string) ([]mod.Mod, string) {
				return []mod.Mod{MOD("PierceCount", "BASE", num).Tag(mod.SkillName(skillName).Transfigured(true))}, ""
			})
	}
	if tags["bow"] || tags["projectile"] {
		add("^"+skillNameLower+" fires an additional projectile", extra(MOD("ProjectileCount", "BASE", 1)))
		add("^"+skillNameLower+` fires (\d+) additional projectiles`, extraFunc("ProjectileCount", "BASE"))
		add("^"+skillNameLower+` fires (\d+) additional shard projectiles`, extraFunc("ProjectileCount", "BASE"))
	}
}

//pob2:src/Modules/ModParser.lua:29-49
func getSimpleConv(srcList []string, dst string, modType mod.Type, remove bool, factor ...float64) radiusJewelFunc {
	return getSimpleConvSrc(srcList, dst, modType, remove, "", factor...)
}
func getSimpleConvSrc(srcList []string, dst string, modType mod.Type, remove bool, srcType mod.Type, factor ...float64) radiusJewelFunc {
	return func(node *radiusJewelNode, out *moddb.ModList, _ *radiusJewelData) {
		if node == nil {
			return
		}
		for _, src := range srcList {
			if node.isTattoo && simpleConvAttributes[src] {
				continue
			}
			for _, m := range node.modList.Mods() {
				if m.Name() != src {
					continue
				}
				if !((srcType != "" && m.Type() == srcType) || m.Type() == modType) {
					continue
				}
				value := m.Value()
				if value == nil || value.Type() != mod.ModValueMultiTypeFloat {
					continue
				}
				if remove {
					mergeNewMod(out, src, modType, -value.Float(), m)
				}
				if len(factor) > 0 {
					mergeNewMod(out, dst, modType, math.Floor(value.Float()*factor[0]), m)
				} else {
					mergeNewMod(out, dst, modType, value.Float(), m)
				}
			}
		}
	}
}

//pob2:src/Modules/ModParser.lua:7252-7260
func getPerStat(dst string, modType mod.Type, flags mod.MFlag, stat string, factor float64) radiusJewelFunc {
	return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
		if node != nil {
			d.stats[stat] += out.Sum(mod.TypeBase, nil, stat)
			return
		}

		value, seen := d.stats[stat]
		if seen && value == 0 {
			return
		}

		m := mod.NewFloat(dst, modType, math.Floor(value*factor)).Source(d.modSource)
		if flags != 0 {
			m.Flag(flags)
		}
		out.AddMod(m)
	}
}

const (
	bowTransformMask1 = ModFlagAxe | ModFlagClaw | ModFlagDagger | ModFlagMace |
		ModFlagStaff | ModFlagSword | ModFlagMelee
	bowTransformMask2 = ModFlagWeapon1H | ModFlagWeaponMelee
	bowTransformMask3 = ModFlagWeapon2H | ModFlagWeaponMelee
)

var bowTransformUsingConditions = map[string]bool{
	"UsingAxe":         true,
	"UsingClaw":        true,
	"UsingDagger":      true,
	"UsingMace":        true,
	"UsingStaff":       true,
	"UsingSword":       true,
	"UsingMeleeWeapon": true,
}

func bowTransform(node *radiusJewelNode, out *moddb.ModList, _ *radiusJewelData) {
	if node == nil {
		return
	}

	for _, m := range node.modList.Mods() {
		value := m.Value()
		if value == nil || value.Type() != mod.ModValueMultiTypeFloat {
			continue
		}

		flags := m.Flags()
		if flags&bowTransformMask1 != 0 ||
			flags&bowTransformMask2 == bowTransformMask2 ||
			flags&bowTransformMask3 == bowTransformMask3 {
			bowTransformMerge(out, m, -value.Float(), flags, m.Tags())
			bowTransformMerge(out, m, value.Float(),
				flags&^(bowTransformMask1|bowTransformMask2|bowTransformMask3)|ModFlagBow, m.Tags())
			continue
		}

		if rewritten, ok := rewriteUsingConditionToBow(m.Tags()); ok {
			bowTransformMerge(out, m, -value.Float(), flags, m.Tags())
			bowTransformMerge(out, m, value.Float(), flags, rewritten)
		}
	}
}
func bowTransformMerge(out *moddb.ModList, proto mod.Mod, value float64, flags mod.MFlag, tags []mod.Tag) {
	m := mod.NewFloat(proto.Name(), proto.Type(), value).
		Source(proto.GetSource()).
		Flag(flags).
		KeywordFlag(proto.KeywordFlags())
	if len(tags) > 0 {
		m.Tag(tags...)
	}
	mergeMod(out, m)
}
func rewriteUsingConditionToBow(tags []mod.Tag) ([]mod.Tag, bool) {
	for i, tag := range tags {
		cond, ok := tag.(*mod.ConditionTag)
		if !ok || len(cond.VarList) != 1 || !bowTransformUsingConditions[cond.VarList[0]] {
			continue
		}

		out := make([]mod.Tag, len(tags))
		copy(out, tags)
		out[i] = mod.Condition("UsingBow").Neg(cond.Negative)

		return out, true
	}

	return nil, false
}

var jewelOtherFuncs = map[string]radiusJewelFunc{
	//pob2:block:src/Modules/ModParser.lua:7087-7093
	"Strength from Passives in Radius is Transformed to Dexterity":                         getSimpleConv([]string{"Str"}, "Dex", mod.TypeBase, true),
	"Dexterity from Passives in Radius is Transformed to Strength":                         getSimpleConv([]string{"Dex"}, "Str", mod.TypeBase, true),
	"Strength from Passives in Radius is Transformed to Intelligence":                      getSimpleConv([]string{"Str"}, "Int", mod.TypeBase, true),
	"Intelligence from Passives in Radius is Transformed to Strength":                      getSimpleConv([]string{"Int"}, "Str", mod.TypeBase, true),
	"Dexterity from Passives in Radius is Transformed to Intelligence":                     getSimpleConv([]string{"Dex"}, "Int", mod.TypeBase, true),
	"Intelligence from Passives in Radius is Transformed to Dexterity":                     getSimpleConv([]string{"Int"}, "Dex", mod.TypeBase, true),
	"Increases and Reductions to Life in Radius are Transformed to apply to Energy Shield": getSimpleConv([]string{"Life"}, "EnergyShield", mod.TypeIncrease, true),
	//pob2:block:src/Modules/ModParser.lua:7100-7102
	"Increases and Reductions to Physical Damage in Radius are Transformed to apply to Cold Damage": getSimpleConv([]string{"PhysicalDamage"}, "ColdDamage", mod.TypeIncrease, true),
	"Increases and Reductions to Cold Damage in Radius are Transformed to apply to Physical Damage": getSimpleConv([]string{"ColdDamage"}, "PhysicalDamage", mod.TypeIncrease, true),
	"Increases and Reductions to other Damage Types in Radius are Transformed to apply to Fire Damage": getSimpleConv(
		[]string{"PhysicalDamage", "ColdDamage", "LightningDamage", "ChaosDamage", "ElementalDamage"}, "FireDamage", mod.TypeIncrease, true),
	//pob2:block:src/Modules/ModParser.lua:7124-7152
	"Melee and Melee Weapon Type modifiers in Radius are Transformed to Bow Modifiers": bowTransform,
	//pob2:block:src/Modules/ModParser.lua:7214-7223
	"Notable Passive Skills in Radius grant nothing": func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
		if node != nil && node.nodeType == "Notable" {
			out.AddMod(mod.NewFlag("PassiveSkillHasNoEffect", true).Source(d.modSource))
		}
	},
	"Allocated Small Passive Skills in Radius grant nothing": func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
		if node != nil && node.nodeType == "Normal" {
			out.AddMod(mod.NewFlag("AllocatedPassiveSkillHasNoEffect", true).Source(d.modSource))
		}
	},
}

var jewelSelfFuncs = map[string]radiusJewelFunc{
	//pob2:block:src/Modules/ModParser.lua:7262-7282
	"Adds 1 to maximum Life per 3 Intelligence in Radius":                                                    getPerStat("Life", mod.TypeBase, 0, "Int", 1.0/3),
	"Adds 1 to Maximum Life per 3 Intelligence Allocated in Radius":                                          getPerStat("Life", mod.TypeBase, 0, "Int", 1.0/3),
	"1% increased Evasion Rating per 3 Dexterity Allocated in Radius":                                        getPerStat("Evasion", mod.TypeIncrease, 0, "Dex", 1.0/3),
	"1% increased Claw Physical Damage per 3 Dexterity Allocated in Radius":                                  getPerStat("PhysicalDamage", mod.TypeIncrease, ModFlagClaw, "Dex", 1.0/3),
	"1% increased Melee Physical Damage while Unarmed per 3 Dexterity Allocated in Radius":                   getPerStat("PhysicalDamage", mod.TypeIncrease, ModFlagUnarmed, "Dex", 1.0/3),
	"3% increased Totem Life per 10 Strength in Radius":                                                      getPerStat("TotemLife", mod.TypeIncrease, 0, "Str", 3.0/10),
	"3% increased Totem Life per 10 Strength Allocated in Radius":                                            getPerStat("TotemLife", mod.TypeIncrease, 0, "Str", 3.0/10),
	"Adds 1 maximum Lightning Damage to Attacks per 1 Dexterity Allocated in Radius":                         getPerStat("LightningMax", mod.TypeBase, ModFlagAttack, "Dex", 1),
	"5% increased Chaos damage per 10 Intelligence from Allocated Passives in Radius":                        getPerStat("ChaosDamage", mod.TypeIncrease, 0, "Int", 5.0/10),
	"-1 Strength per 1 Strength on Allocated Passives in Radius":                                             getPerStat("Str", mod.TypeBase, 0, "Str", -1),
	"1% additional Physical Damage Reduction per 10 Strength on Allocated Passives in Radius":                getPerStat("PhysicalDamageReduction", mod.TypeBase, 0, "Str", 1.0/10),
	"2% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius":                        getPerStat("LifeRecoveryRate", mod.TypeIncrease, 0, "Str", 2.0/10),
	"3% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius":                        getPerStat("LifeRecoveryRate", mod.TypeIncrease, 0, "Str", 3.0/10),
	"-1 Intelligence per 1 Intelligence on Allocated Passives in Radius":                                     getPerStat("Int", mod.TypeBase, 0, "Int", -1),
	"0.4% of Energy Shield Regenerated per Second for every 10 Intelligence on Allocated Passives in Radius": getPerStat("EnergyShieldRegenPercent", mod.TypeBase, 0, "Int", 0.4/10),
	"regenerate 0.4% of energy shield per second for every 10 Intelligence on Allocated Passives in Radius":  getPerStat("EnergyShieldRegenPercent", mod.TypeBase, 0, "Int", 0.4/10),
	"2% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius":                    getPerStat("ManaRecoveryRate", mod.TypeIncrease, 0, "Int", 2.0/10),
	"3% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius":                    getPerStat("ManaRecoveryRate", mod.TypeIncrease, 0, "Int", 3.0/10),
	"-1 Dexterity per 1 Dexterity on Allocated Passives in Radius":                                           getPerStat("Dex", mod.TypeBase, 0, "Dex", -1),
	"2% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius":                           getPerStat("MovementSpeed", mod.TypeIncrease, 0, "Dex", 2.0/10),
	"3% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius":                           getPerStat("MovementSpeed", mod.TypeIncrease, 0, "Dex", 3.0/10),
}

var jewelSelfUnallocFuncs = map[string]radiusJewelFunc{
	//pob2:block:src/Modules/ModParser.lua:7285-7310
	"+5% to Critical Strike Multiplier per 10 Strength on Unallocated Passives in Radius":      getPerStat("CritMultiplier", mod.TypeBase, 0, "Str", 5.0/10),
	"+7% to Critical Strike Multiplier per 10 Strength on Unallocated Passives in Radius":      getPerStat("CritMultiplier", mod.TypeBase, 0, "Str", 7.0/10),
	"2% reduced Life Recovery Rate per 10 Strength on Unallocated Passives in Radius":          getPerStat("LifeRecoveryRate", mod.TypeIncrease, 0, "Str", -2.0/10),
	"+15 to maximum Mana per 10 Dexterity on Unallocated Passives in Radius":                   getPerStat("Mana", mod.TypeBase, 0, "Dex", 15.0/10),
	"+100 to Accuracy Rating per 10 Intelligence on Unallocated Passives in Radius":            getPerStat("Accuracy", mod.TypeBase, 0, "Int", 100.0/10),
	"+125 to Accuracy Rating per 10 Intelligence on Unallocated Passives in Radius":            getPerStat("Accuracy", mod.TypeBase, 0, "Int", 125.0/10),
	"2% reduced Mana Recovery Rate per 10 Intelligence on Unallocated Passives in Radius":      getPerStat("ManaRecoveryRate", mod.TypeIncrease, 0, "Int", -2.0/10),
	"+3% to Damage over Time Multiplier per 10 Intelligence on Unallocated Passives in Radius": getPerStat("DotMultiplier", mod.TypeBase, 0, "Int", 3.0/10),
	"2% reduced Movement Speed per 10 Dexterity on Unallocated Passives in Radius":             getPerStat("MovementSpeed", mod.TypeIncrease, 0, "Dex", -2.0/10),
	"+125 to Accuracy Rating per 10 Dexterity on Unallocated Passives in Radius":               getPerStat("Accuracy", mod.TypeBase, 0, "Dex", 125.0/10),
	"Grants all bonuses of Unallocated Small Passive Skills in Radius":                         grantUnallocatedBonuses("Normal"),
}

func grantUnallocatedBonuses(nodeType string) radiusJewelFunc {
	return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
		if node != nil {
			if node.nodeType == nodeType {
				if d.modList == nil {
					d.modList = moddb.NewModList()
				}
				for _, m := range out.Mods() {
					if strings.HasPrefix(m.Name(), "Condition:ConnectedTo") {
						continue
					}
					d.modList.AddMod(m)
				}
			}
			return
		}
		if d.modList != nil {
			out.AddList(d.modList)
		}
	}
}

func radiusGrantAppliesTo(node *radiusJewelNode) bool {
	return node != nil &&
		node.nodeType != "Keystone" &&
		node.nodeType != "Socket" &&
		node.nodeType != "ClassStart"
}

//pob2:src/Modules/ModParser.lua:13-15
func firstToUpper(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func enemyDamageConvertRow(condition string) func(num float64, captures []string) ([]mod.Mod, string) {
	return func(num float64, captures []string) ([]mod.Mod, string) {
		nested := MOD(firstToUpper(captures[1])+"DamageConvertTo"+firstToUpper(captures[2]), "BASE", num)
		if condition != "" {
			nested = nested.Tag(mod.Condition(condition))
		}
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: nested})}, ""
	}
}

func radiusGrantSimple(name string, modType mod.Type) func(captures []string) radiusJewelFunc {
	return func(captures []string) radiusJewelFunc {
		num := utils.Float(captures[0])
		return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
			if !radiusGrantAppliesTo(node) {
				return
			}
			out.AddMod(mod.NewFloat(name, modType, num).Source(d.modSource))
		}
	}
}

var simpleConvAttributes = map[string]bool{"Str": true, "Dex": true, "Int": true}

func simpleConvPercent(srcList []string, dst string, modType mod.Type, remove bool, srcType mod.Type) func([]string) radiusJewelFunc {
	return func(captures []string) radiusJewelFunc {
		return getSimpleConvSrc(srcList, dst, modType, remove, srcType, utils.Float(captures[0])/100)
	}
}

type jewelFuncPattern struct {
	re    *regexp.Regexp
	typ   string
	build func(captures []string) radiusJewelFunc
}

var jewelFuncPatternsRaw = []struct {
	pattern string
	typ     string
	build   func(captures []string) radiusJewelFunc
}{
	//pob2:block:src/Modules/ModParser.lua:7094-7099
	{`increases and reductions to energy shield in radius are transformed to apply to armour at (\d+)% of their value`, "Other", simpleConvFactor([]string{"EnergyShield"}, "Armour", mod.TypeIncrease, true)},
	{`increases and reductions to life in radius are transformed to apply to mana at (\d+)% of their value`, "Other", simpleConvFactor([]string{"Life"}, "Mana", mod.TypeIncrease, true)},
	//pob2:block:src/Modules/ModParser.lua:7103-7123
	{`passives granting lightning resistance or all elemental resistances in radius also grant chance to block spells at (\d+)% of its value`, "Other", simpleConvFactor([]string{"LightningResist", "ElementalResist"}, "SpellBlockChance", mod.TypeBase, false)},
	{`passives granting lightning resistance or all elemental resistances in radius also grant chance to block spell damage at (\d+)% of its value`, "Other", simpleConvFactor([]string{"LightningResist", "ElementalResist"}, "SpellBlockChance", mod.TypeBase, false)},
	{`passives granting cold resistance or all elemental resistances in radius also grant chance to dodge attacks at (\d+)% of its value`, "Other", simpleConvFactor([]string{"ColdResist", "ElementalResist"}, "AttackDodgeChance", mod.TypeBase, false)},
	{`passives granting cold resistance or all elemental resistances in radius also grant chance to dodge attack hits at (\d+)% of its value`, "Other", simpleConvFactor([]string{"ColdResist", "ElementalResist"}, "AttackDodgeChance", mod.TypeBase, false)},
	{`passives granting cold resistance or all elemental resistances in radius also grant chance to suppress spell damage at (\d+)% of its value`, "Other", simpleConvFactor([]string{"ColdResist", "ElementalResist"}, "SpellSuppressionChance", mod.TypeBase, false)},
	{`passives granting fire resistance or all elemental resistances in radius also grant chance to block attack damage at (\d+)% of its value`, "Other", simpleConvFactor([]string{"FireResist", "ElementalResist"}, "BlockChance", mod.TypeBase, false)},
	{`passives granting fire resistance or all elemental resistances in radius also grant chance to block at (\d+)% of its value`, "Other", simpleConvFactor([]string{"FireResist", "ElementalResist"}, "BlockChance", mod.TypeBase, false)},
	//pob2:block:src/Modules/ModParser.lua:7154-7169
	{`(\d+)% increased effect of small passive skills in radius$`, "Other", func(captures []string) radiusJewelFunc {
		num := tonumber(captures[0])
		return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
			if node != nil && node.nodeType == "Normal" && !node.isAttribute {
				out.AddMod(mod.NewFloat("JewelSmallPassiveSkillEffect", mod.TypeIncrease, num).Source(d.modSource).Tag(mod.GlobalEffect("Global").Unscalable(true)))
			}
		}
	}},
	{`(\d+)% increased effect of notable passive skills in radius$`, "Other", func(captures []string) radiusJewelFunc {
		num := tonumber(captures[0])
		return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
			if node != nil && node.nodeType == "Notable" {
				out.AddMod(mod.NewFloat("JewelNotablePassiveSkillEffect", mod.TypeIncrease, num).Source(d.modSource).Tag(mod.GlobalEffect("Global").Unscalable(true)))
			}
		}
	}},
	//pob2:block:src/Modules/ModParser.lua:7170-7187
	{`^(\w+) passive skills in radius also grant (.*)$`, "Other", func(captures []string) radiusJewelFunc {
		passiveType := firstToUpper(captures[0])
		line := captures[1]
		return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
			if node != nil && (node.nodeType == passiveType || (node.nodeType == "Normal" && !node.isAttribute && passiveType == "Small")) {
				radiusAlsoGrant(line, out, d)
			}
		}
	}},
	//pob2:block:src/Modules/ModParser.lua:7188-7205
	{`conquered (\w+) passive skills also grant (.*)$`, "Other", func(captures []string) radiusJewelFunc {
		passiveType := firstToUpper(captures[0])
		line := captures[1]
		return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
			if node != nil && (node.nodeType == passiveType || (node.nodeType == "Normal" && !node.isAttribute && passiveType == "Small") || (node.nodeType == "Normal" && node.isAttribute && passiveType == "Attribute")) {
				radiusAlsoGrant(line, out, d)
			}
		}
	}},
	//pob2:block:src/Modules/ModParser.lua:7207-7213
	{`(\d+)% increased effect of non-keystone passive skills in radius`, "Other", func(captures []string) radiusJewelFunc {
		num := tonumber(captures[0])
		return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
			if node != nil && node.nodeType != "Keystone" && node.nodeType != "ClassStart" {
				out.AddMod(mod.NewFloat("PassiveSkillEffect", mod.TypeIncrease, num).Source(d.modSource))
			}
		}
	}},
	//pob2:block:src/Modules/ModParser.lua:7224-7248
	{`notable passive skills in radius are transformed to instead grant: (\d+)% increased mana cost of skills and (\d+)% increased spell damage`, "Other", func(captures []string) radiusJewelFunc {
		num1, num2 := tonumber(captures[0]), tonumber(captures[1])
		return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
			if node != nil && node.nodeType == "Notable" {
				out.AddMod(mod.NewFlag("PassiveSkillHasOtherEffect", true).Source(d.modSource))
				out.AddMod(mod.NewList("NodeModifier", nodeModifier{Mod: mod.NewFloat("ManaCost", mod.TypeIncrease, num1).Source(d.modSource)}).Source(d.modSource))
				out.AddMod(mod.NewList("NodeModifier", nodeModifier{Mod: mod.NewFloat("Damage", mod.TypeIncrease, num2).Source(d.modSource).Flag(ModFlagSpell)}).Source(d.modSource))
			}
		}
	}},
	{`notable passive skills in radius are transformed to instead grant: minions take (\d+)% increased damage`, "Other", func(captures []string) radiusJewelFunc {
		num := tonumber(captures[0])
		return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
			if node != nil && node.nodeType == "Notable" {
				out.AddMod(mod.NewFlag("PassiveSkillHasOtherEffect", true).Source(d.modSource))
				out.AddMod(mod.NewList("NodeModifier", nodeModifier{Mod: mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num).Source(d.modSource)})}).Source(d.modSource))
			}
		}
	}},
	{`notable passive skills in radius are transformed to instead grant: minions have (\d+)% reduced movement speed`, "Other", func(captures []string) radiusJewelFunc {
		num := tonumber(captures[0])
		return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
			if node != nil && node.nodeType == "Notable" {
				out.AddMod(mod.NewFlag("PassiveSkillHasOtherEffect", true).Source(d.modSource))
				out.AddMod(mod.NewList("NodeModifier", nodeModifier{Mod: mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("MovementSpeed", mod.TypeIncrease, -num).Source(d.modSource)})}).Source(d.modSource))
			}
		}
	}},
	//pob2:block:src/Modules/ModParser.lua:7372-7374
	{`with at least 40 dexterity in radius, dual strike has (\d+)% increased attack speed while wielding a claw`, "Threshold", func(captures []string) radiusJewelFunc {
		return getThreshold([]string{"Dex"}, MOD("Speed", mod.TypeIncrease, tonumber(captures[0])).Tag(mod.SkillName("Dual Strike").Transfigured(true), mod.Condition("UsingClaw")))
	}},
	{`with at least 40 dexterity in radius, dual strike has \+(\d+)% to critical strike multiplier while wielding a dagger`, "Threshold", func(captures []string) radiusJewelFunc {
		return getThreshold([]string{"Dex"}, MOD("CritMultiplier", mod.TypeBase, tonumber(captures[0])).Tag(mod.SkillName("Dual Strike").Transfigured(true), mod.Condition("UsingDagger")))
	}},
	{`with at least 40 dexterity in radius, dual strike has (\d+)% increased accuracy rating while wielding a sword`, "Threshold", func(captures []string) radiusJewelFunc {
		return getThreshold([]string{"Dex"}, MOD("Accuracy", mod.TypeIncrease, tonumber(captures[0])).Tag(mod.SkillName("Dual Strike").Transfigured(true), mod.Condition("UsingSword")))
	}},
	//pob2:block:src/Modules/ModParser.lua:7396
	{`with at least 40 intelligence in radius, fireball has \+(\d+)% chance to inflict scorch`, "Threshold", func(captures []string) radiusJewelFunc {
		return getThreshold([]string{"Int"}, MOD("EnemyScorchChance", mod.TypeBase, tonumber(captures[0])).Tag(mod.SkillName("Fireball")))
	}},
	//pob2:block:src/Modules/ModParser.lua:7400-7402
	{`with at least 40 intelligence in radius, (\d+)% of damage taken recouped as mana if you've warcried recently`, "Threshold", func(captures []string) radiusJewelFunc {
		return getThreshold([]string{"Int"}, MOD("ManaRecoup", mod.TypeBase, tonumber(captures[0])).Tag(mod.Condition("UsedWarcryRecently")))
	}},
	{`with at least 40 intelligence in radius, fireball projectiles gain radius as they travel farther, up to \+(\d+) radius`, "Threshold", func(captures []string) radiusJewelFunc {
		return getThreshold([]string{"Int"}, MOD("AreaOfEffect", mod.TypeBase, tonumber(captures[0])).Tag(mod.DistanceRamp([][]float64{{0, 0}, {50, 1}})))
	}},
	{`with at least 40 intelligence in radius, projectiles gain radius as they travel farther, up to a maximum of \+([\d.]+) metres? to radius`, "Threshold", func(captures []string) radiusJewelFunc {
		return getThreshold([]string{"Int"}, MOD("AreaOfEffect", mod.TypeBase, tonumber(captures[0])*10).Tag(mod.DistanceRamp([][]float64{{0, 0}, {50, 1}})))
	}},
}

func simpleConvFactor(srcList []string, dst string, modType mod.Type, remove bool) func([]string) radiusJewelFunc {
	return func(captures []string) radiusJewelFunc {
		return getSimpleConv(srcList, dst, modType, remove, tonumber(captures[0])/100)
	}
}

// radiusAlsoGrant is the "also grant" body: the granted line is parsed when the func runs, as PoB does.
//
//pob2:block:src/Modules/ModParser.lua:7173-7184
func radiusAlsoGrant(line string, out *moddb.ModList, d *radiusJewelData) {
	modList, extra := parseMod(line, 0)
	if extra != "" || len(modList) == 0 {
		return
	}
	for _, m := range modList {
		m = m.Clone().Source(d.modSource)
		setNestedModSource(m, d.modSource)
		out.AddMod(m)
	}
}

var (
	jewelFuncList     map[string]jewelFuncEntry
	jewelFuncPatterns []jewelFuncPattern
)

func init() {
	jewelFuncList = make(map[string]jewelFuncEntry,
		len(jewelOtherFuncs)+len(jewelSelfFuncs)+len(jewelSelfUnallocFuncs)+len(jewelThresholdFuncs))
	for line, fn := range jewelOtherFuncs {
		jewelFuncList[strings.ToLower(line)] = jewelFuncEntry{Type: "Other", Func: guardConquered(fn)}
	}
	for line, fn := range jewelSelfFuncs {
		jewelFuncList[strings.ToLower(line)] = jewelFuncEntry{Type: "Self", Func: fn}
	}
	for line, fn := range jewelSelfUnallocFuncs {
		jewelFuncList[strings.ToLower(line)] = jewelFuncEntry{Type: "SelfUnalloc", Func: fn}
	}
	for line, fn := range jewelThresholdFuncs {
		jewelFuncList[strings.ToLower(line)] = jewelFuncEntry{Type: "Threshold", Func: fn}
	}
	for line, fns := range jewelThresholdFuncLists {
		jewelFuncList[strings.ToLower(line)] = jewelFuncEntry{Type: "Threshold", Func: multiThreshold(fns)}
	}

	jewelFuncPatterns = make([]jewelFuncPattern, 0, len(jewelFuncPatternsRaw))
	for _, raw := range jewelFuncPatternsRaw {
		jewelFuncPatterns = append(jewelFuncPatterns, jewelFuncPattern{
			re:    regexp.MustCompile(raw.pattern),
			typ:   raw.typ,
			build: raw.build,
		})
	}
	sort.Slice(jewelFuncPatterns, func(i, j int) bool {
		return jewelFuncPatterns[i].re.String() < jewelFuncPatterns[j].re.String()
	})
}
func guardConquered(fn radiusJewelFunc) radiusJewelFunc {
	return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
		if node != nil && node.conquered {
			return
		}
		fn(node, out, d)
	}
}
func parseJewelFunc(lineLower string) []mod.Mod {
	for _, pattern := range jewelFuncPatterns {
		match := pattern.re.FindStringSubmatch(lineLower)
		if len(match) < 2 {
			continue
		}
		fn := pattern.build(match[1:])
		if pattern.typ == "Other" {
			fn = guardConquered(fn)
		}
		return []mod.Mod{mod.NewList("JewelFunc", jewelFuncEntry{Type: pattern.typ, Func: fn})}
	}

	if entry, ok := jewelFuncList[lineLower]; ok {
		return []mod.Mod{mod.NewList("JewelFunc", entry)}
	}

	return nil
}

//pob2:src/Modules/ModParser.lua:7314-7338
func getThreshold(attribs []string, proto mod.Mod) radiusJewelFunc {
	return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
		if node != nil {
			for _, attrib := range attribs {
				nodeVal := out.Sum(mod.TypeBase, nil, attrib)
				d.stats[attrib] += nodeVal
				d.total += nodeVal
				d.totalOK = true
			}
			return
		}

		if d.total < jewelThresholdMinimum {
			return
		}

		m := proto.Clone().Source(d.modSource)
		setNestedModSource(m, d.modSource)
		out.AddMod(m)
	}
}
func multiThreshold(fns []radiusJewelFunc) radiusJewelFunc {
	return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
		if d.funcData == nil {
			d.funcData = make(map[int]*radiusJewelData, len(fns))
		}
		for i, fn := range fns {
			sub, ok := d.funcData[i]
			if !ok {
				sub = newRadiusJewelData(d.modSource)
				d.funcData[i] = sub
			}
			sub.modSource = d.modSource
			fn(node, out, sub)
		}
	}
}

var jewelThresholdFuncLists = map[string][]radiusJewelFunc{}

const jewelThresholdMinimum = 40

func setNestedModSource(m mod.Mod, source mod.Source) {
	value := m.Value()
	if value == nil || value.Type() != mod.ModValueMultiTypeList {
		return
	}
	switch payload := value.List().(type) {
	case mod.ExtraSkillMod:
		payload.Mod.Source(source)
	case mod.MinionModifier:
		payload.Mod.Source(source)
	case mod.EnemyModifier:
		payload.Mod.Source(source)
	}
}

var jewelThresholdFuncs = map[string]radiusJewelFunc{
	//pob2:block:src/Modules/ModParser.lua:7340-7371
	"With at least 40 Dexterity in Radius, Frost Blades Melee Damage Penetrates 15% Cold Resistance":                      getThreshold([]string{"Dex"}, MOD("ColdPenetration", mod.TypeBase, 15).Flag(ModFlagMelee).Tag(mod.SkillName("Frost Blades").Transfigured(true))),
	"With at least 40 Dexterity in Radius, Melee Damage dealt by Frost Blades Penetrates 15% Cold Resistance":             getThreshold([]string{"Dex"}, MOD("ColdPenetration", mod.TypeBase, 15).Flag(ModFlagMelee).Tag(mod.SkillName("Frost Blades").Transfigured(true))),
	"With at least 40 Dexterity in Radius, Frost Blades has 25% increased Projectile Speed":                               getThreshold([]string{"Dex"}, MOD("ProjectileSpeed", mod.TypeIncrease, 25).Tag(mod.SkillName("Frost Blades").Transfigured(true))),
	"With at least 40 Dexterity in Radius, Ice Shot has 25% increased Area of Effect":                                     getThreshold([]string{"Dex"}, MOD("AreaOfEffect", mod.TypeIncrease, 25).Tag(mod.SkillName("Ice Shot"))),
	"Ice Shot Pierces 5 additional Targets with 40 Dexterity in Radius":                                                   getThreshold([]string{"Dex"}, MOD("PierceCount", mod.TypeBase, 5).Tag(mod.SkillName("Ice Shot"))),
	"With at least 40 Dexterity in Radius, Ice Shot Pierces 3 additional Targets":                                         getThreshold([]string{"Dex"}, MOD("PierceCount", mod.TypeBase, 3).Tag(mod.SkillName("Ice Shot"))),
	"With at least 40 Dexterity in Radius, Ice Shot Pierces 5 additional Targets":                                         getThreshold([]string{"Dex"}, MOD("PierceCount", mod.TypeBase, 5).Tag(mod.SkillName("Ice Shot"))),
	"With at least 40 Intelligence in Radius, Frostbolt fires 2 additional Projectiles":                                   getThreshold([]string{"Int"}, MOD("ProjectileCount", mod.TypeBase, 2).Tag(mod.SkillName("Frostbolt"))),
	"With at least 40 Intelligence in Radius, Rolling Magma fires an additional Projectile":                               getThreshold([]string{"Int"}, MOD("ProjectileCount", mod.TypeBase, 1).Tag(mod.SkillName("Rolling Magma"))),
	"With at least 40 Intelligence in Radius, Rolling Magma has 10% increased Area of Effect per Chain":                   getThreshold([]string{"Int"}, MOD("AreaOfEffect", mod.TypeIncrease, 10).Tag(mod.SkillName("Rolling Magma"), mod.PerStat(0, "Chain"))),
	"With at least 40 Intelligence in Radius, Rolling Magma deals 40% more damage per chain":                              getThreshold([]string{"Int"}, MOD("Damage", mod.TypeMore, 40).Tag(mod.SkillName("Rolling Magma"), mod.PerStat(0, "Chain"))),
	"With at least 40 Intelligence in Radius, Rolling Magma deals 50% less damage":                                        getThreshold([]string{"Int"}, MOD("Damage", mod.TypeMore, -50).Tag(mod.SkillName("Rolling Magma"))),
	"With at least 40 Dexterity in Radius, Shrapnel Shot has 25% increased Area of Effect":                                getThreshold([]string{"Dex"}, MOD("AreaOfEffect", mod.TypeIncrease, 25).Tag(mod.SkillName("Shrapnel Shot"))),
	"With at least 40 Dexterity in Radius, Shrapnel Shot's cone has a 50% chance to deal Double Damage":                   getThreshold([]string{"Dex"}, MOD("DoubleDamageChance", mod.TypeBase, 50).Tag(mod.SkillName("Shrapnel Shot"), mod.SkillPart(2))),
	"With at least 40 Dexterity in Radius, Galvanic Arrow deals 50% increased Area Damage":                                getThreshold([]string{"Dex"}, MOD("Damage", mod.TypeIncrease, 50).Tag(mod.SkillName("Galvanic Arrow").Transfigured(true), mod.SkillPart(2))),
	"With at least 40 Dexterity in Radius, Galvanic Arrow has 25% increased Area of Effect":                               getThreshold([]string{"Dex"}, MOD("AreaOfEffect", mod.TypeIncrease, 25).Tag(mod.SkillName("Galvanic Arrow").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Freezing Pulse fires 2 additional Projectiles":                              getThreshold([]string{"Int"}, MOD("ProjectileCount", mod.TypeBase, 2).Tag(mod.SkillName("Freezing Pulse"))),
	"With at least 40 Intelligence in Radius, 25% increased Freezing Pulse Damage if you've Shattered an Enemy Recently":  getThreshold([]string{"Int"}, MOD("Damage", mod.TypeIncrease, 25).Tag(mod.SkillName("Freezing Pulse"), mod.Condition("ShatteredEnemyRecently"))),
	"With at least 40 Dexterity in Radius, Ethereal Knives fires 10 additional Projectiles":                               getThreshold([]string{"Dex"}, MOD("ProjectileCount", mod.TypeBase, 10).Tag(mod.SkillName("Ethereal Knives").Transfigured(true))),
	"With at least 40 Dexterity in Radius, Ethereal Knives fires 5 additional Projectiles":                                getThreshold([]string{"Dex"}, MOD("ProjectileCount", mod.TypeBase, 5).Tag(mod.SkillName("Ethereal Knives").Transfigured(true))),
	"With at least 40 Strength in Radius, Molten Strike fires 2 additional Projectiles":                                   getThreshold([]string{"Str"}, MOD("ProjectileCount", mod.TypeBase, 2).Tag(mod.SkillName("Molten Strike").Transfigured(true))),
	"With at least 40 Strength in Radius, Molten Strike has 25% increased Area of Effect":                                 getThreshold([]string{"Str"}, MOD("AreaOfEffect", mod.TypeIncrease, 25).Tag(mod.SkillName("Molten Strike").Transfigured(true))),
	"With at least 40 Strength in Radius, Molten Strike Projectiles Chain +1 time":                                        getThreshold([]string{"Str"}, MOD("ChainCountMax", mod.TypeBase, 1).Tag(mod.SkillName("Molten Strike").Transfigured(true))),
	"With at least 40 Strength in Radius, Molten Strike fires 50% less Projectiles":                                       getThreshold([]string{"Str"}, MOD("ProjectileCount", mod.TypeMore, -50).Tag(mod.SkillName("Molten Strike").Transfigured(true))),
	"With at least 40 Strength in Radius, 25% of Glacial Hammer Physical Damage converted to Cold Damage":                 getThreshold([]string{"Str"}, MOD("SkillPhysicalDamageConvertToCold", mod.TypeBase, 25).Tag(mod.SkillName("Glacial Hammer").Transfigured(true))),
	"With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage":                            getThreshold([]string{"Str"}, MOD("DoubleDamageChance", mod.TypeBase, 20).Tag(mod.SkillName("Heavy Strike"))),
	"With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage.":                           getThreshold([]string{"Str"}, MOD("DoubleDamageChance", mod.TypeBase, 20).Tag(mod.SkillName("Heavy Strike"))),
	"With at least 40 Strength in Radius, Cleave has +1 to Radius per Nearby Enemy, up to +10":                            getThreshold([]string{"Str"}, MOD("AreaOfEffect", mod.TypeBase, 1).Tag(mod.Multiplier("NearbyEnemies").Limit(10), mod.SkillName("Cleave").Transfigured(true))),
	"With at least 40 Strength in Radius, Cleave has +0.1 metres to Radius per Nearby Enemy, up to a maximum of +1 metre": getThreshold([]string{"Str"}, MOD("AreaOfEffect", mod.TypeBase, 1).Tag(mod.Multiplier("NearbyEnemies").Limit(10), mod.SkillName("Cleave").Transfigured(true))),
	"With at least 40 Strength in Radius, Cleave grants Fortify on Hit":                                                   getThreshold([]string{"Str"}, MOD("ExtraSkillMod", mod.TypeList, mod.ExtraSkillMod{Mod: FLAG("Condition:Fortified")}).Tag(mod.SkillName("Cleave").Transfigured(true))),
	"With at least 40 Strength in Radius, Hits with Cleave Fortify":                                                       getThreshold([]string{"Str"}, MOD("ExtraSkillMod", mod.TypeList, mod.ExtraSkillMod{Mod: FLAG("Condition:Fortified")}).Tag(mod.SkillName("Cleave").Transfigured(true))),
	"With at least 40 Dexterity in Radius, Dual Strike has a 20% chance to deal Double Damage with the Main-Hand Weapon":  getThreshold([]string{"Dex"}, MOD("DoubleDamageChance", mod.TypeBase, 20).Tag(mod.SkillName("Dual Strike").Transfigured(true), mod.Condition("MainHandAttack"))),
	//pob2:block:src/Modules/ModParser.lua:7375-7399
	"With at least 40 Dexterity in Radius, Dual Strike Hits Intimidate Enemies for 4 seconds while wielding an Axe":                  getThreshold([]string{"Dex"}, MOD("EnemyModifier", mod.TypeList, mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).Tag(mod.Condition("UsingAxe"))),
	"With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack has 100% increased Cooldown Recovery Speed":                getThreshold([]string{"Int"}, MOD("MinionModifier", mod.TypeList, mod.MinionModifier{Mod: MOD("CooldownRecovery", mod.TypeIncrease, 100).Tag(mod.SkillId("ZombieSlam"))})),
	"With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack deals 30% increased Damage":                                getThreshold([]string{"Int"}, MOD("MinionModifier", mod.TypeList, mod.MinionModifier{Mod: MOD("Damage", mod.TypeIncrease, 30).Tag(mod.SkillId("ZombieSlam"))})),
	"With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Attack Damage for each Poison on the Enemy":               getThreshold([]string{"Dex"}, MOD("Damage", mod.TypeIncrease, 2).Flag(ModFlagAttack).Tag(mod.SkillName("Viper Strike").Transfigured(true), mod.Multiplier("PoisonStack").Actor("enemy"))),
	"With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Damage with Hits and Poison for each Poison on the Enemy": getThreshold([]string{"Dex"}, MOD("Damage", mod.TypeIncrease, 2).KeywordFlag(KeywordFlagHit|KeywordFlagPoison).Tag(mod.SkillName("Viper Strike").Transfigured(true), mod.Multiplier("PoisonStack").Actor("enemy"))),
	"With at least 40 Intelligence in Radius, Spark fires 2 additional Projectiles":                                                  getThreshold([]string{"Int"}, MOD("ProjectileCount", mod.TypeBase, 2).Tag(mod.SkillName("Spark").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Blight has 50% increased Hinder Duration":                                              getThreshold([]string{"Int"}, MOD("SecondaryDuration", mod.TypeIncrease, 50).Tag(mod.SkillName("Blight").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Enemies Hindered by Blight take 25% increased Chaos Damage":                            getThreshold([]string{"Int"}, MOD("ExtraSkillMod", mod.TypeList, mod.ExtraSkillMod{Mod: MOD("ChaosDamageTaken", mod.TypeIncrease, 25).Tag(mod.GlobalEffect("Debuff").Name("Hinder"))}).Tag(mod.SkillName("Blight").Transfigured(true), mod.ActorCondition("enemy", "Hindered"))),
	"With 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage":                                getThreshold([]string{"Int"}, MOD("SkillPhysicalDamageConvertToCold", mod.TypeBase, 20).Tag(mod.SkillName("Glacial Cascade").Transfigured(true))),
	"With at least 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage":                       getThreshold([]string{"Int"}, MOD("SkillPhysicalDamageConvertToCold", mod.TypeBase, 20).Tag(mod.SkillName("Glacial Cascade").Transfigured(true))),
	"With 40 total Intelligence and Dexterity in Radius, Elemental Hit and Wild Strike deal 50% less Fire Damage":                    getThreshold([]string{"Int", "Dex"}, MOD("FireDamage", mod.TypeMore, -50).Tag(mod.SkillName("Elemental Hit", "Wild Strike").Transfigured(true))),
	"With 40 total Strength and Intelligence in Radius, Elemental Hit and Wild Strike deal 50% less Cold Damage":                     getThreshold([]string{"Str", "Int"}, MOD("ColdDamage", mod.TypeMore, -50).Tag(mod.SkillName("Elemental Hit", "Wild Strike").Transfigured(true))),
	"With 40 total Dexterity and Strength in Radius, Elemental Hit and Wild Strike deal 50% less Lightning Damage":                   getThreshold([]string{"Dex", "Str"}, MOD("LightningDamage", mod.TypeMore, -50).Tag(mod.SkillName("Elemental Hit", "Wild Strike").Transfigured(true))),
	"With 40 total Intelligence and Dexterity in Radius, Prismatic Skills deal 50% less Fire Damage":                                 getThreshold([]string{"Int", "Dex"}, MOD("FireDamage", mod.TypeMore, -50).Tag(mod.SkillType("RandomElement"))),
	"With 40 total Strength and Intelligence in Radius, Prismatic Skills deal 50% less Cold Damage":                                  getThreshold([]string{"Str", "Int"}, MOD("ColdDamage", mod.TypeMore, -50).Tag(mod.SkillType("RandomElement"))),
	"With 40 total Dexterity and Strength in Radius, Prismatic Skills deal 50% less Lightning Damage":                                getThreshold([]string{"Dex", "Str"}, MOD("LightningDamage", mod.TypeMore, -50).Tag(mod.SkillType("RandomElement"))),
	"With 40 total Dexterity and Strength in Radius, Spectral Shield Throw Chains +4 times":                                          getThreshold([]string{"Dex", "Str"}, MOD("ChainCountMax", mod.TypeBase, 4).Tag(mod.SkillName("Spectral Shield Throw").Transfigured(true))),
	"With 40 total Dexterity and Strength in Radius, Spectral Shield Throw fires 75% less Shard Projectiles":                         getThreshold([]string{"Dex", "Str"}, MOD("ProjectileCount", mod.TypeMore, -75).Tag(mod.SkillName("Spectral Shield Throw").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Blight inflicts Withered for 2 seconds":                                                getThreshold([]string{"Int"}, MOD("ExtraSkillMod", mod.TypeList, mod.ExtraSkillMod{Mod: FLAG("Condition:CanWither")}).Tag(mod.SkillName("Blight").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Blight has 30% reduced Cast Speed":                                                     getThreshold([]string{"Int"}, MOD("Speed", mod.TypeIncrease, -30).Tag(mod.SkillName("Blight").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Fireball cannot ignite":                                                                getThreshold([]string{"Int"}, MOD("ExtraSkillMod", mod.TypeList, mod.ExtraSkillMod{Mod: FLAG("CannotIgnite")}).Tag(mod.SkillName("Fireball"))),
	"With at least 40 Intelligence in Radius, Discharge has 60% less Area of Effect":                                                 getThreshold([]string{"Int"}, MOD("AreaOfEffect", mod.TypeMore, -60).Tag(mod.SkillName("Discharge").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Discharge Cooldown is 250 ms":                                                          getThreshold([]string{"Int"}, MOD("CooldownRecovery", mod.TypeOverride, 0.25).Tag(mod.SkillName("Discharge").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Discharge deals 60% less Damage":                                                       getThreshold([]string{"Int"}, MOD("Damage", mod.TypeMore, -60).Tag(mod.SkillName("Discharge").Transfigured(true))),
}

// Scan a line for the earliest and longest match from the pattern list
// If a match is found, returns the corresponding value from the pattern list, plus the remainder of the line and a table of captures
//
//pob2:src/Modules/ModParser.lua:6623-6645
func scan[T any](line string, patternList map[string]CompiledList[T], plain bool) (*T, string, []string) {
	state := newScanState[T](line)

	for pattern, patternVal := range patternList {
		state.consider(pattern, patternVal)
	}

	return state.result()
}

type scanState[T any] struct {
	line      string
	lineLower string

	bestIndex    int
	bestEndIndex int
	bestPattern  string
	bestValue    T
	bestGroups   []int
	bestStart    int
	bestEnd      int
}

func newScanState[T any](line string) scanState[T] {
	return scanState[T]{line: line, lineLower: toLowerASCII(line), bestIndex: -1}
}
func (s *scanState[T]) consider(pattern string, patternVal CompiledList[T]) {
	indices := patternVal.Regex.FindAllStringSubmatchIndex(s.lineLower, -1)
	if len(indices) == 0 {
		return
	}

	index := indices[0][0]
	endIndex := indices[0][1]

	better := s.bestIndex == -1 || index < s.bestIndex
	if !better && index == s.bestIndex {
		switch {
		case endIndex != s.bestEndIndex:
			better = endIndex > s.bestEndIndex
		case len(pattern) != len(s.bestPattern):
			better = len(pattern) > len(s.bestPattern)
		default:
			better = pattern < s.bestPattern
		}
	}

	if better {
		s.bestIndex = index
		s.bestEndIndex = endIndex
		s.bestPattern = pattern
		s.bestValue = patternVal.Value
		s.bestGroups = indices[0]
		s.bestStart = index
		s.bestEnd = endIndex
	}
}
func (s *scanState[T]) result() (*T, string, []string) {
	if s.bestIndex < 0 {
		return nil, s.line, nil
	}

	captures := make([]string, (len(s.bestGroups)-2)/2)
	for i := range captures {
		captures[i] = s.lineLower[s.bestGroups[2+i*2]:s.bestGroups[2+i*2+1]]
	}

	return utils.Ptr(s.bestValue), s.line[:s.bestStart] + s.line[s.bestEnd:], captures
}

func toLowerASCII(s string) string {
	for i := 0; i < len(s); i++ {
		if c := s[i]; (c >= 'A' && c <= 'Z') || c >= utf8.RuneSelf {
			return strings.ToLower(s)
		}
	}
	return s
}

//pob2:src/Modules/ModParser.lua:6650-7068
func parseMod(line string, order int) ([]mod.Mod, string) {
	lineLower := strings.ToLower(line)

	if jewelFunc := parseJewelFunc(lineLower); jewelFunc != nil {
		return jewelFunc, ""
	}

	if _, ok := unsupportedModList[lineLower]; ok {
		return []mod.Mod{}, line
	}

	specialMod, specialLine, captures := scanAnchored(line, specialModIndex)
	if specialMod != nil && len(specialLine) == 0 {
		if specialFunc, ok := (*specialMod).(func(num float64, captures []string) ([]mod.Mod, string)); ok {
			if len(captures) == 0 {
				return specialFunc(0, captures)
			}
			return specialFunc(tonumber(captures[0]), captures)
		}
		return mod.CloneSlice((*specialMod).([]mod.Mod)), ""
	}

	// Check for add-to-cluster-jewel special
	if m := addToClusterRe.FindStringSubmatch(line); m != nil {
		return []mod.Mod{mod.NewList("AddToClusterJewelNode", m[1])}, ""
	}

	line = line + " "

	// Check for a flag/tag specification at the start of the line
	var preFlag *modNameListType
	var preFlagCap []string
	preFlag, line, preFlagCap = scanAnchored(line, preFlagIndex)
	if preFlag != nil && preFlag.fn != nil {
		temp := preFlag.fn(preFlagCap)
		preFlag = &temp
	}

	// Check for skill name at the start of the line
	var skillTag *modNameListType
	skillTag, line, _ = scanAnchored(line, preSkillNameIndex)

	// Scan for modifier form
	var modForm *string
	var formCap []string
	modForm, line, formCap = scanAny(line, formListIndex)
	if modForm == nil {
		return nil, line
	}

	var modTag *modNameListType
	var modTag2 *modNameListType
	var tagCap []string

	// Check for tags (per-charge, conditionals)
	modTag, line, tagCap = scanAny(line, modTagIndex)
	if modTag != nil && modTag.fn != nil {
		modTag = utils.Ptr(modTag.fn(tagCap))
	}

	if modTag != nil {
		modTag2, line, tagCap = scanAny(line, modTagIndex)
		if modTag2 != nil && modTag2.fn != nil {
			modTag2 = utils.Ptr(modTag2.fn(tagCap))
		}
	}

	// Scan for modifier name and skill name
	var modName *modNameListType
	if order == 2 && skillTag == nil {
		skillTag, line, _ = scanAny(line, skillNameIndex)
	}

	var flagName *modNameListType
	if *modForm == "PEN" {
		modName, line, _ = scan(line, penTypesCompiled, true)
		if modName == nil {
			return []mod.Mod{}, line
		}
		_, line, _ = scanAny(line, modNameIndex)
	} else if *modForm == "BASECOST" || *modForm == "TOTALCOST" {
		types := costTypesCompiled
		if *modForm == "BASECOST" {
			types = baseCostTypesCompiled
		}
		modName, line, _ = scan(line, types, true)
		if modName == nil {
			return []mod.Mod{}, line
		}
		_, line, _ = scanAny(line, modNameIndex)
	} else if *modForm == "FLAG" {
		flagName, line, _ = scan(line, flagTypesCompiled, false)
		if flagName == nil {
			return nil, line
		}
		modName, line, _ = scanAny(line, modNameIndex)
	} else {
		modName, line, _ = scanAny(line, modNameIndex)
	}

	if order == 1 && skillTag == nil {
		skillTag, line, _ = scanAny(line, skillNameIndex)
	}

	// Scan for flags
	var modFlag *modNameListType
	modFlag, line, _ = scanAny(line, modFlagIndex)

	// Find modifier value and type according to form
	modValue := []float64{0.0}
	if len(formCap) > 0 {
		modValue[0] = tonumber(formCap[0])
	}

	modType := "BASE"
	var modTypes []string
	var perModTags [][]mod.Tag
	var modSuffix *string
	var formTags []mod.Tag

	switch *modForm {
	case "INC":
		modType = "INC"
	case "RED":
		modValue[0] = -modValue[0]
		modType = "INC"
	case "MORE":
		modType = "MORE"
	case "LESS":
		modValue[0] = -modValue[0]
		modType = "MORE"
	case "BASE":
		modSuffix, line, _ = scanAny(line, suffixTypesIndex)
	case "GAIN":
		modSuffix, line, _ = scanAny(line, suffixTypesIndex)
	case "LOSE":
		modValue[0] = -modValue[0]
		modSuffix, line, _ = scanAny(line, suffixTypesIndex)
	case "GRANTS":
		formTags = []mod.Tag{mod.Condition("{Hand}Attack")}
		modSuffix, line, _ = scanAny(line, suffixTypesIndex)
	case "GRANTS_GLOBAL":
		modSuffix, line, _ = scanAny(line, suffixTypesIndex)
	case "REMOVES":
		modValue[0] = -modValue[0]
		formTags = []mod.Tag{mod.Condition("{Hand}Attack")}
		modSuffix, line, _ = scanAny(line, suffixTypesIndex)
	case "CHANCE":
	case "REGENPERCENT":
		modName = utils.Ptr(regenTypes[strings.ToLower(formCap[1])])
		modSuffix = utils.Ptr("Percent")
	case "REGENFLAT":
		modName = utils.Ptr(regenTypes[strings.ToLower(formCap[1])])
	case "DEGENPERCENT":
		if t, ok := degenTypes[strings.ToLower(formCap[1])]; ok {
			modName = utils.Ptr(t)
		}
		modSuffix = utils.Ptr("Percent")
	case "DEGENFLAT":
		if t, ok := degenTypes[strings.ToLower(formCap[1])]; ok {
			modName = utils.Ptr(t)
		}
	case "DEGEN":
		damageType := dmgTypes[strings.ToLower(formCap[1])]
		if damageType == "" {
			return []mod.Mod{}, line
		}
		modName = &modNameListType{names: []string{damageType + "Degen"}}
		modSuffix = utils.Ptr("")
	case "DMG":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return []mod.Mod{}, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
	case "DMGATTACKS":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return []mod.Mod{}, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: KeywordFlagAttack}
		}
	case "DMGSPELLS":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return []mod.Mod{}, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: KeywordFlagSpell}
		}
	case "DMGTHORNS":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return []mod.Mod{}, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{flags: ModFlagThorns}
		}
	case "DMGTHORNSBASE":
		damageType := dmgTypes[strings.ToLower(formCap[0])]
		if damageType == "" {
			return []mod.Mod{}, line
		}
		modValue = []float64{1, 1}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}, flags: ModFlagThorns}
		if modFlag == nil {
			modFlag = &modNameListType{flags: ModFlagThorns}
		}
	case "DMGBOTH":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return []mod.Mod{}, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: KeywordFlagAttack | KeywordFlagSpell}
		}
	case "IMMUNE":
		var ok bool
		modName, modTypes, modValue, ok = immuneForm(line)
		if !ok {
			return []mod.Mod{}, line
		}
		line = ""
	case "OVERRIDE":
		modType = "OVERRIDE"
	case "DOUBLED":
		if modName == nil {
			break
		}
		base := modName.names[0]
		names := make([]string, len(modName.names), len(modName.names)+1)
		copy(names, modName.names)
		if len(names) >= 2 {
			names[1] = "Multiplier:" + base + "Doubled"
		} else {
			names = append(names, "Multiplier:"+base+"Doubled")
		}
		clone := *modName
		clone.names = names
		modName = &clone

		modTypes = []string{"MORE", "OVERRIDE"}
		modValue = []float64{100, 1}
		perModTags = [][]mod.Tag{
			{mod.Multiplier(base + "Doubled").GlobalLimit(100).GlobalLimitKey(base + "DoubledLimit")},
			nil,
		}
	case "FLAG":
		modName = flagName
		if flagName.flagModType != "" {
			modType = flagName.flagModType
			modValue = []float64{flagName.flagModValue}
		} else {
			modType = "FLAG"
			modValue = []float64{1}
		}
	}

	if modName == nil {
		return []mod.Mod{}, line
	}

	// Combine flags and tags
	flags := mod.MFlag(0)
	keywordFlags := mod.KeywordFlag(0)
	tagList := make([]mod.Tag, 0)
	tagList = append(tagList, formTags...)
	misc := modNameListType{}
	for _, datum := range []*modNameListType{modName, preFlag, modFlag, modTag, modTag2, skillTag} {
		if datum == nil {
			continue
		}

		flags = flags | datum.flags
		keywordFlags = keywordFlags | datum.keywordFlags
		if datum.tag != nil {
			tagList = append(tagList, datum.tag)
		} else if datum.tagList != nil {
			tagList = append(tagList, datum.tagList...)
		}

		if datum.names != nil {
			misc.names = datum.names
		}

		if datum.tagList != nil {
			misc.tagList = datum.tagList
		}

		if datum.flags != 0 {
			misc.flags = datum.flags
		}

		if datum.keywordFlags != 0 {
			misc.keywordFlags = datum.keywordFlags
		}

		if datum.addToSkill != nil {
			misc.addToSkill = datum.addToSkill
		}

		if datum.addToMinionTag != nil {
			misc.addToMinionTag = datum.addToMinionTag
		}

		if datum.playerTagList != nil {
			misc.playerTagList = datum.playerTagList
		}
		if datum.playerTag != nil {
			misc.playerTag = datum.playerTag
		}

		if datum.fn != nil {
			misc.fn = datum.fn
		}

		if datum.modSuffix != "" {
			misc.modSuffix = datum.modSuffix
		}

		misc.addToMinion = misc.addToMinion || datum.addToMinion
		misc.applyToEnemy = misc.applyToEnemy || datum.applyToEnemy
		misc.addToAura = misc.addToAura || datum.addToAura
		misc.newAuraOnlyAllies = misc.newAuraOnlyAllies || datum.newAuraOnlyAllies
		misc.newAura = misc.newAura || datum.newAura
		misc.onlyAddToBanners = misc.onlyAddToBanners || datum.onlyAddToBanners
		misc.actorEnemy = misc.actorEnemy || datum.actorEnemy
	}

	// Generate modifier list
	nameList := modName
	modList := make([]mod.Mod, len(nameList.names))
	for i, name := range nameList.names {
		fullName := name
		if modSuffix != nil {
			fullName += *modSuffix
		} else if misc.modSuffix != "" {
			fullName += misc.modSuffix
		}

		realValue := modValue[0]
		if len(modValue) > i {
			realValue = modValue[i]
		}

		realType := modType
		if len(modTypes) > i {
			realType = modTypes[i]
		}

		switch realType {
		case "BASE":
			modList[i] = mod.NewFloat(fullName, mod.TypeBase, realValue)
		case "INC":
			modList[i] = mod.NewFloat(fullName, mod.TypeIncrease, realValue)
		case "MORE":
			modList[i] = mod.NewFloat(fullName, mod.TypeMore, realValue)
		case "OVERRIDE":
			modList[i] = mod.NewFloat(fullName, mod.TypeOverride, realValue)
		case "FLAG":
			modList[i] = mod.NewFlag(fullName, realValue > 0)
		}

		modList[i].Flag(flags)
		modList[i].KeywordFlag(keywordFlags)
		modList[i].Tag(tagList...)
		if len(perModTags) > i && perModTags[i] != nil {
			modList[i].Tag(perModTags[i]...)
		}
	}

	if len(modList) > 0 {
		// Special handling for various modifier types
		if misc.addToAura {
			for i, effectMod := range modList {
				wrapped := mod.NewList("ExtraAuraEffect", mod.ExtraAuraEffect{Mod: effectMod})
				if misc.onlyAddToBanners {
					wrapped = wrapped.Tag(mod.SkillType("Banner"))
				}
				modList[i] = wrapped
			}
		} else if misc.newAura {
			// Modifiers that add extra auras
			for i, effectMod := range modList {
				tagList := effectMod.Tags()
				effectMod.ClearTags()
				modList[i] = mod.NewList("ExtraAura", mod.ExtraAura{Mod: effectMod, OnlyAllies: misc.newAuraOnlyAllies}).Tag(tagList...)
			}
		} else if misc.addToMinion {
			for i, effectMod := range modList {
				modList[i] = mod.NewList("MinionModifier", mod.MinionModifier{Mod: effectMod})
				if misc.playerTag != nil {
					modList[i] = modList[i].Tag(misc.playerTag)
				}
				if misc.addToMinionTag != nil {
					modList[i] = modList[i].Tag(misc.addToMinionTag)
				}
				modList[i] = modList[i].Tag(misc.playerTagList...)
			}
		} else if misc.addToSkill != nil {
			// Skill enchants or socketed gem modifiers that add additional effects
			for i, effectMod := range modList {
				modList[i] = mod.NewList("ExtraSkillMod", mod.ExtraSkillMod{Mod: effectMod}).Tag(misc.addToSkill)
			}
		} else if misc.applyToEnemy {
			for i, effectMod := range modList {
				if tags := effectMod.Tags(); len(tags) > 0 && misc.actorEnemy {
					effectMod = effectMod.Clone()
					tags = effectMod.Tags()
					tags[0] = withTagActor(tags[0], "enemy")
				}
				modList[i] = mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: effectMod})
				if misc.playerTag != nil {
					modList[i] = modList[i].Tag(misc.playerTag)
				}
				modList[i] = modList[i].Tag(misc.playerTagList...)
			}
		}
	}

	if strings.TrimSpace(line) != "" {
		return modList, line
	}

	return modList, ""
}

var (
	modCacheMu sync.RWMutex
	modCache   = make(map[string]modCacheEntry)
)

type modCacheEntry struct {
	modList []mod.Mod
	extra   string
}

// ParseMod is modLib.parseMod: the mods a line produces and its unparsed remainder ("" when fully parsed).
//
//pob2:src/Modules/ModParser.lua:7435-7454
func ParseMod(line string) ([]mod.Mod, string) {
	initializeParser()

	modCacheMu.RLock()
	cached, ok := modCache[line]
	modCacheMu.RUnlock()

	if !ok {
		modList, extra := parseMod(line, 1)
		if modList != nil && extra != "" {
			modList, extra = parseMod(line, 2)
		}
		cached = modCacheEntry{modList: modList, extra: extra}

		modCacheMu.Lock()
		if existing, raced := modCache[line]; raced {
			cached = existing
		} else {
			modCache[line] = cached
		}
		modCacheMu.Unlock()
	}

	if cached.modList == nil {
		return nil, cached.extra
	}
	return mod.CloneSlice(cached.modList), cached.extra
}

//pob2:block:src/Modules/ModParser.lua:6388-6392
func initializeRegex() {
	formListCompiled = make(map[string]CompiledList[string])
	for k, v := range formList {
		formListCompiled[k] = CompiledList[string]{
			Regex: lazyRegex(k),
			Value: v,
		}
	}

	modNameListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range modNameList {
		modNameListCompiled[k] = CompiledList[modNameListType]{
			Regex: lazyRegex(k),
			Value: v,
		}
	}

	modFlagListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range modFlagList {
		modFlagListCompiled[k] = CompiledList[modNameListType]{
			Regex: lazyRegex(k),
			Value: v,
		}
	}

	preFlagListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range preFlagList {
		preFlagListCompiled[k] = CompiledList[modNameListType]{
			Regex: lazyRegex(k),
			Value: v,
		}
	}

	modTagListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range modTagList {
		modTagListCompiled[k] = CompiledList[modNameListType]{
			Regex: lazyRegex(k),
			Value: v,
		}
	}

	suffixTypesCompiled = make(map[string]CompiledList[string])
	for k, v := range suffixTypes {
		suffixTypesCompiled[k] = CompiledList[string]{
			Regex: lazyRegex(k),
			Value: v,
		}
	}

	dmgTypesCompiled = make(map[string]CompiledList[string])
	for k, v := range dmgTypes {
		dmgTypesCompiled[k] = CompiledList[string]{
			Regex: lazyRegex(k),
			Value: v,
		}
	}

	penTypesCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range penTypes {
		penTypesCompiled[k] = CompiledList[modNameListType]{
			Regex: lazyRegex(k),
			Value: v,
		}
	}

	regenTypesCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range regenTypes {
		regenTypesCompiled[k] = CompiledList[modNameListType]{
			Regex: lazyRegex(k),
			Value: v,
		}
	}

	for _, pair := range []struct {
		src map[string]modNameListType
		dst *map[string]CompiledList[modNameListType]
	}{{degenTypes, &degenTypesCompiled}, {costTypes, &costTypesCompiled}, {baseCostTypes, &baseCostTypesCompiled}} {
		compiled := make(map[string]CompiledList[modNameListType], len(pair.src))
		for k, v := range pair.src {
			compiled[k] = CompiledList[modNameListType]{
				Regex: lazyRegex(k),
				Value: v,
			}
		}
		*pair.dst = compiled
	}

	flagTypesCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range flagTypes {
		flagTypesCompiled[k] = CompiledList[modNameListType]{
			Regex: lazyRegex(k),
			Value: v,
		}
	}

	specialModListCompiled = make(map[string]CompiledList[interface{}])
	for k, v := range specialModList {
		specialModListCompiled[k] = CompiledList[interface{}]{
			Regex: lazyRegex("^" + k + "$"),
			Value: v,
		}
	}

	specialModIndex = newAnchoredIndex(specialModListCompiled)
	preFlagIndex = newAnchoredIndex(preFlagListCompiled)

	formListIndex = newAnyOffsetIndex(formListCompiled)
	modNameIndex = newAnyOffsetIndex(modNameListCompiled)
	modTagIndex = newAnyOffsetIndex(modTagListCompiled)
	modFlagIndex = newAnyOffsetIndex(modFlagListCompiled)
	suffixTypesIndex = newAnyOffsetIndex(suffixTypesCompiled)
}

var parserInit sync.Once

func initializeParser() {
	parserInit.Do(func() {
		initializeRegex()
		initializeSkillNameList()
	})
}

func titleFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
func titleCaseAfterNonWord(s string) string {
	b := []byte(" " + s)
	for i := 1; i < len(b); i++ {
		prev := b[i-1]
		isWord := (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z') || (prev >= '0' && prev <= '9')
		if !isWord && b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 'a' - 'A'
			i++
		}
	}
	return string(b[1:])
}

func capitaliseResistName(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func gemKeywordList(skill string) []string {
	wordList := gemKeywordWordPattern.FindAllString(strings.ToLower(skill), -1)
	for i, tag := range wordList {
		if tag == "skill" {
			wordList[i] = "active_skill"
		}
	}
	return wordList
}

func selfCastCondition(curse string) string {
	var b strings.Builder
	b.WriteString("SelfCast")
	for _, word := range strings.Fields(curse) {
		b.WriteString(firstToUpper(word))
	}
	return b.String()
}

func selfCastExposure(exposureType, curse string) []mod.Mod {
	return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(firstToUpper(exposureType)+"Exposure", "BASE", -10)}).
		Tag(mod.Condition(selfCastCondition(curse))).
		Tag(mod.Condition("Effective"))}
}

func minionConvertKey(source, target string) string {
	return utils.Capital(strings.ToLower(source)) + "DamageConvertTo" + utils.Capital(strings.ToLower(target))
}
