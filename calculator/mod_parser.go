package calculator

import (
	"math"
	"regexp"
	"strings"

	utils2 "github.com/Vilsol/go-pob-data/utils"
	"golang.org/x/exp/slices"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

type CompiledList[T any] struct {
	Regex *regexp.Regexp
	Value T
}

var conquerorList = map[string]mod.ConquerorType{
	"xibaqua":  {ID: "1", Type: "vaal"},
	"zerphi":   {ID: "2", Type: "vaal"},
	"doryani":  {ID: "3", Type: "vaal"},
	"ahuana":   {ID: "2_v2", Type: "vaal"},
	"deshret":  {ID: "1", Type: "maraketh"},
	"asenath":  {ID: "2", Type: "maraketh"},
	"nasima":   {ID: "3", Type: "maraketh"},
	"balbala":  {ID: "1_v2", Type: "maraketh"},
	"cadiro":   {ID: "1", Type: "eternal"},
	"victario": {ID: "2", Type: "eternal"},
	"chitus":   {ID: "3", Type: "eternal"},
	"caspiro":  {ID: "3_v2", Type: "eternal"},
	"kaom":     {ID: "1", Type: "karui"},
	"rakiata":  {ID: "2", Type: "karui"},
	"kiloava":  {ID: "3", Type: "karui"},
	"akoya":    {ID: "3_v2", Type: "karui"},
	"venarius": {ID: "1", Type: "templar"},
	"dominus":  {ID: "2", Type: "templar"},
	"avarius":  {ID: "3", Type: "templar"},
	"maxarius": {ID: "1_v2", Type: "templar"},
}

// List of modifier forms
var formListCompiled map[string]CompiledList[string]
var formList = map[string]string{
	`^(\d+)% increased`:                                        "INC",
	`^(\d+)% faster`:                                           "INC",
	`^(\d+)% reduced`:                                          "RED",
	`^(\d+)% slower`:                                           "RED",
	`^(\d+)% more`:                                             "MORE",
	`^(\d+)% less`:                                             "LESS",
	`^([\+\-][\d\.]+)%?`:                                       "BASE",
	`^([\+\-][\d\.]+)%? to`:                                    "BASE",
	`^([\+\-]?[\d\.]+)%? of`:                                   "BASE",
	`^([\+\-][\d\.]+)%? base`:                                  "BASE",
	`^([\+\-]?[\d\.]+)%? additional`:                           "BASE",
	`(\d+) additional hits?`:                                   "BASE",
	`^you gain ([\d\.]+)`:                                      "BASE",
	`^gains? ([\d\.]+)% of`:                                    "BASE",
	`^([\+\-]?\d+)% chance`:                                    "CHANCE",
	`^([\+\-]?\d+)% additional chance`:                         "CHANCE",
	`penetrates? (\d+)%`:                                       "PEN",
	`penetrates (\d+)% of`:                                     "PEN",
	`penetrates (\d+)% of enemy`:                               "PEN",
	`^([\d\.]+) (\S+?) regenerated per second`:                 "REGENFLAT",
	`^([\d\.]+)% (?:of )?(\S+?) regenerated per second`:        "REGENPERCENT",
	`^regenerate ([\d\.]+) (\S+?) per second`:                  "REGENFLAT",
	`^regenerate ([\d\.]+)% (?:of |of your )(\S+?) per second`: "REGENPERCENT",
	`^you regenerate ([\d\.]+)% of (\S+?) per second`:          "REGENPERCENT",
	`^([\d\.]+) (\w+) damage taken per second`:                 "DEGEN",
	`^([\d\.]+) (\w+) damage per second`:                       "DEGEN",
	`(\d+) to (\d+) added (\w+) damage`:                        "DMG",
	`(\d+)\-(\d+) added (\w+) damage`:                          "DMG",
	`(\d+) to (\d+) additional (\w+) damage`:                   "DMG",
	`(\d+)\-(\d+) additional (\w+) damage`:                     "DMG",
	`^(\d+) to (\d+) (\w+) damage`:                             "DMG",
	`adds (\d+) to (\d+) (\w+) damage`:                         "DMG",
	`adds (\d+)\-(\d+) (\w+) damage`:                           "DMG",
	`adds (\d+) to (\d+) (\w+) damage to attacks`:              "DMGATTACKS",
	`adds (\d+)\-(\d+) (\w+) damage to attacks`:                "DMGATTACKS",
	`adds (\d+) to (\d+) (\w+) attack damage`:                  "DMGATTACKS",
	`adds (\d+)\-(\d+) (\w+) attack damage`:                    "DMGATTACKS",
	`(\d+) to (\d+) added attack (\w+) damage`:                 "DMGATTACKS",
	`adds (\d+) to (\d+) (\w+) damage to spells`:               "DMGSPELLS",
	`adds (\d+)\-(\d+) (\w+) damage to spells`:                 "DMGSPELLS",
	`adds (\d+) to (\d+) (\w+) spell damage`:                   "DMGSPELLS",
	`adds (\d+)\-(\d+) (\w+) spell damage`:                     "DMGSPELLS",
	`(\d+) to (\d+) added spell (\w+) damage`:                  "DMGSPELLS",
	`adds (\d+) to (\d+) (\w+) damage to attacks and spells`:   "DMGBOTH",
	`adds (\d+)\-(\d+) (\w+) damage to attacks and spells`:     "DMGBOTH",
	`adds (\d+) to (\d+) (\w+) damage to spells and attacks`:   "DMGBOTH", // o_O
	`adds (\d+)\-(\d+) (\w+) damage to spells and attacks`:     "DMGBOTH", // o_O
	`adds (\d+) to (\d+) (\w+) damage to hits`:                 "DMGBOTH",
	`adds (\d+)\-(\d+) (\w+) damage to hits`:                   "DMGBOTH",
	`^you have `:                                               "FLAG",
	`^you are `:                                                "FLAG",
	`^are `:                                                    "FLAG",
}

type modNameListType struct {
	names             []string
	tag               mod.Tag
	tagList           []mod.Tag
	flags             mod.MFlag
	addToMinion       bool
	keywordFlags      mod.KeywordFlag
	addToSkill        mod.Tag
	addToMinionTag    mod.Tag
	applyToEnemy      bool
	fn                func(caps []string) modNameListType
	modSuffix         string
	addToAura         bool
	newAuraOnlyAllies bool
	newAura           bool
	affectedByAura    bool
}

// Map of modifier names
var modNameListCompiled map[string]CompiledList[modNameListType]
var modNameList = map[string]modNameListType{
	// Attributes
	"strength":                   {names: []string{"Str"}},
	"dexterity":                  {names: []string{"Dex"}},
	"intelligence":               {names: []string{"Int"}},
	"omniscience":                {names: []string{"Omni"}},
	"strength and dexterity":     {names: []string{"Str", "Dex", "StrDex"}},
	"strength and intelligence":  {names: []string{"Str", "Int", "StrInt"}},
	"dexterity and intelligence": {names: []string{"Dex", "Int", "DexInt"}},
	"attributes":                 {names: []string{"Str", "Dex", "Int", "All"}},
	"all attributes":             {names: []string{"Str", "Dex", "Int", "All"}},
	"devotion":                   {names: []string{"Devotion"}},

	// Life/mana
	"life":                                  {names: []string{"Life"}},
	"maximum life":                          {names: []string{"Life"}},
	"life regeneration rate":                {names: []string{"LifeRegen"}},
	"mana":                                  {names: []string{"Mana"}},
	"maximum mana":                          {names: []string{"Mana"}},
	"mana regeneration":                     {names: []string{"ManaRegen"}},
	"mana regeneration rate":                {names: []string{"ManaRegen"}},
	"mana cost":                             {names: []string{"ManaCost"}},
	"mana cost of":                          {names: []string{"ManaCost"}},
	"mana cost of skills":                   {names: []string{"ManaCost"}},
	"mana cost of attacks":                  {names: []string{"ManaCost"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"total cost":                            {names: []string{"Cost"}},
	"total mana cost":                       {names: []string{"ManaCost"}},
	"total mana cost of skills":             {names: []string{"ManaCost"}},
	"life cost of skills":                   {names: []string{"LifeCost"}},
	"rage cost of skills":                   {names: []string{"RageCost"}},
	"cost of":                               {names: []string{"Cost"}},
	"cost of skills":                        {names: []string{"Cost"}},
	"mana reserved":                         {names: []string{"ManaReserved"}},
	"mana reservation":                      {names: []string{"ManaReserved"}},
	"mana reservation of skills":            {names: []string{"ManaReserved"}, tag: mod.SkillType(string(data.SkillTypeAura))},
	"mana reservation efficiency of skills": {names: []string{"ManaReservationEfficiency"}},
	"life reservation efficiency of skills": {names: []string{"LifeReservationEfficiency"}},
	"reservation of skills":                 {names: []string{"Reserved"}},
	"mana reservation if cast as an aura":   {names: []string{"ManaReserved"}, tag: mod.SkillType(string(data.SkillTypeAura))},
	"reservation if cast as an aura":        {names: []string{"Reserved"}, tag: mod.SkillType(string(data.SkillTypeAura))},
	"reservation":                           {names: []string{"Reserved"}},
	"reservation efficiency":                {names: []string{"ReservationEfficiency"}},
	"reservation efficiency of skills":      {names: []string{"ReservationEfficiency"}},
	"mana reservation efficiency":           {names: []string{"ManaReservationEfficiency"}},
	"life reservation efficiency":           {names: []string{"LifeReservationEfficiency"}},

	// Primary defences
	"maximum energy shield":              {names: []string{"EnergyShield"}},
	"energy shield recharge rate":        {names: []string{"EnergyShieldRecharge"}},
	"start of energy shield recharge":    {names: []string{"EnergyShieldRechargeFaster"}},
	"restoration of ward":                {names: []string{"WardRechargeFaster"}},
	"armour":                             {names: []string{"Armour"}},
	"evasion":                            {names: []string{"Evasion"}},
	"evasion rating":                     {names: []string{"Evasion"}},
	"energy shield":                      {names: []string{"EnergyShield"}},
	"ward":                               {names: []string{"Ward"}},
	"armour and evasion":                 {names: []string{"ArmourAndEvasion"}},
	"armour and evasion rating":          {names: []string{"ArmourAndEvasion"}},
	"evasion rating and armour":          {names: []string{"ArmourAndEvasion"}},
	"armour and energy shield":           {names: []string{"ArmourAndEnergyShield"}},
	"evasion rating and energy shield":   {names: []string{"EvasionAndEnergyShield"}},
	"evasion and energy shield":          {names: []string{"EvasionAndEnergyShield"}},
	"armour, evasion and energy shield":  {names: []string{"Defences"}},
	"defences":                           {names: []string{"Defences"}},
	"to evade":                           {names: []string{"EvadeChance"}},
	"chance to evade":                    {names: []string{"EvadeChance"}},
	"to evade attacks":                   {names: []string{"EvadeChance"}},
	"to evade attack hits":               {names: []string{"EvadeChance"}},
	"chance to evade attacks":            {names: []string{"EvadeChance"}},
	"chance to evade attack hits":        {names: []string{"EvadeChance"}},
	"chance to evade projectile attacks": {names: []string{"ProjectileEvadeChance"}},
	"chance to evade melee attacks":      {names: []string{"MeleeEvadeChance"}},

	// Resistances
	"physical damage reduction":                                   {names: []string{"PhysicalDamageReduction"}},
	"physical damage reduction from hits":                         {names: []string{"PhysicalDamageReductionWhenHit"}},
	"fire resistance":                                             {names: []string{"FireResist"}},
	"maximum fire resistance":                                     {names: []string{"FireResistMax"}},
	"cold resistance":                                             {names: []string{"ColdResist"}},
	"maximum cold resistance":                                     {names: []string{"ColdResistMax"}},
	"lightning resistance":                                        {names: []string{"LightningResist"}},
	"maximum lightning resistance":                                {names: []string{"LightningResistMax"}},
	"chaos resistance":                                            {names: []string{"ChaosResist"}},
	"maximum chaos resistance":                                    {names: []string{"ChaosResistMax"}},
	"fire and cold resistances":                                   {names: []string{"FireResist", "ColdResist"}},
	"fire and lightning resistances":                              {names: []string{"FireResist", "LightningResist"}},
	"cold and lightning resistances":                              {names: []string{"ColdResist", "LightningResist"}},
	"elemental resistance":                                        {names: []string{"ElementalResist"}},
	"elemental resistances":                                       {names: []string{"ElementalResist"}},
	"all elemental resistances":                                   {names: []string{"ElementalResist"}},
	"all resistances":                                             {names: []string{"ElementalResist", "ChaosResist"}},
	"all maximum elemental resistances":                           {names: []string{"ElementalResistMax"}},
	"all maximum resistances":                                     {names: []string{"ElementalResistMax", "ChaosResistMax"}},
	"all elemental resistances and maximum elemental resistances": {names: []string{"ElementalResist", "ElementalResistMax"}},
	"fire and chaos resistances":                                  {names: []string{"FireResist", "ChaosResist"}},
	"cold and chaos resistances":                                  {names: []string{"ColdResist", "ChaosResist"}},
	"lightning and chaos resistances":                             {names: []string{"LightningResist", "ChaosResist"}},

	// Damage taken
	"damage taken":                           {names: []string{"DamageTaken"}},
	"damage taken when hit":                  {names: []string{"DamageTakenWhenHit"}},
	"damage over time taken":                 {names: []string{"DamageTakenOverTime"}},
	"damage taken from damage over time":     {names: []string{"DamageTakenOverTime"}},
	"attack damage taken":                    {names: []string{"AttackDamageTaken"}},
	"spell damage taken":                     {names: []string{"SpellDamageTaken"}},
	"physical damage taken":                  {names: []string{"PhysicalDamageTaken"}},
	"physical damage from hits taken":        {names: []string{"PhysicalDamageTaken"}},
	"physical damage taken when hit":         {names: []string{"PhysicalDamageTakenWhenHit"}},
	"physical damage taken from hits":        {names: []string{"PhysicalDamageTakenWhenHit"}},
	"physical damage taken from attacks":     {names: []string{"PhysicalDamageTakenFromAttacks"}},
	"physical damage taken over time":        {names: []string{"PhysicalDamageTakenOverTime"}},
	"physical damage over time damage taken": {names: []string{"PhysicalDamageTakenOverTime"}},
	"reflected physical damage taken":        {names: []string{"PhysicalReflectedDamageTaken"}},
	"lightning damage taken":                 {names: []string{"LightningDamageTaken"}},
	"lightning damage from hits taken":       {names: []string{"LightningDamageTaken"}},
	"lightning damage taken when hit":        {names: []string{"LightningDamageTakenWhenHit"}},
	"lightning damage taken from attacks":    {names: []string{"LightningDamageTakenFromAttacks"}},
	"lightning damage taken over time":       {names: []string{"LightningDamageTakenOverTime"}},
	"cold damage taken":                      {names: []string{"ColdDamageTaken"}},
	"cold damage from hits taken":            {names: []string{"ColdDamageTaken"}},
	"cold damage taken when hit":             {names: []string{"ColdDamageTakenWhenHit"}},
	"cold damage taken from attacks":         {names: []string{"ColdDamageTakenFromAttacks"}},
	"cold damage taken over time":            {names: []string{"ColdDamageTakenOverTime"}},
	"fire damage taken":                      {names: []string{"FireDamageTaken"}},
	"fire damage from hits taken":            {names: []string{"FireDamageTaken"}},
	"fire damage taken when hit":             {names: []string{"FireDamageTakenWhenHit"}},
	"fire damage taken from attacks":         {names: []string{"FireDamageTakenFromAttacks"}},
	"fire damage taken over time":            {names: []string{"FireDamageTakenOverTime"}},
	"chaos damage taken":                     {names: []string{"ChaosDamageTaken"}},
	"chaos damage from hits taken":           {names: []string{"ChaosDamageTaken"}},
	"chaos damage taken when hit":            {names: []string{"ChaosDamageTakenWhenHit"}},
	"chaos damage taken from attacks":        {names: []string{"ChaosDamageTakenFromAttacks"}},
	"chaos damage taken over time":           {names: []string{"ChaosDamageTakenOverTime"}},
	"chaos damage over time taken":           {names: []string{"ChaosDamageTakenOverTime"}},
	"elemental damage taken":                 {names: []string{"ElementalDamageTaken"}},
	"elemental damage from hits taken":       {names: []string{"ElementalDamageTaken"}},
	"elemental damage taken when hit":        {names: []string{"ElementalDamageTakenWhenHit"}},
	"elemental damage taken from hits":       {names: []string{"ElementalDamageTakenWhenHit"}},
	"elemental damage taken over time":       {names: []string{"ElementalDamageTakenOverTime"}},
	"cold and lightning damage taken":        {names: []string{"ColdDamageTaken", "LightningDamageTaken"}},
	"fire and lightning damage taken":        {names: []string{"FireDamageTaken", "LightningDamageTaken"}},
	"fire and cold damage taken":             {names: []string{"FireDamageTaken", "ColdDamageTaken"}},
	"physical and chaos damage taken":        {names: []string{"PhysicalDamageTaken", "ChaosDamageTaken"}},
	"reflected elemental damage taken":       {names: []string{"ElementalReflectedDamageTaken"}},

	// Other defences
	"to dodge attacks":                            {names: []string{"AttackDodgeChance"}},
	"to dodge attack hits":                        {names: []string{"AttackDodgeChance"}},
	"to dodge spells":                             {names: []string{"SpellDodgeChance"}},
	"to dodge spell hits":                         {names: []string{"SpellDodgeChance"}},
	"to dodge spell damage":                       {names: []string{"SpellDodgeChance"}},
	"to dodge attacks and spells":                 {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to dodge attacks and spell damage":           {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to dodge attack and spell hits":              {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to dodge attack or spell hits":               {names: []string{"AttackDodgeChance", "SpellDodgeChance"}},
	"to suppress spell damage":                    {names: []string{"SpellSuppressionChance"}},
	"amount of suppressed spell damage prevented": {names: []string{"SpellSuppressionEffect"}},
	"to block":                                        {names: []string{"BlockChance"}},
	"to block attacks":                                {names: []string{"BlockChance"}},
	"to block attack damage":                          {names: []string{"BlockChance"}},
	"block chance":                                    {names: []string{"BlockChance"}},
	"block chance with staves":                        {names: []string{"BlockChance"}, tag: mod.Condition("UsingStaff")},
	"to block with staves":                            {names: []string{"BlockChance"}, tag: mod.Condition("UsingStaff")},
	"block chance against projectiles":                {names: []string{"ProjectileBlockChance"}},
	"to block projectile attack damage":               {names: []string{"ProjectileBlockChance"}},
	"spell block chance":                              {names: []string{"SpellBlockChance"}},
	"to block spells":                                 {names: []string{"SpellBlockChance"}},
	"to block spell damage":                           {names: []string{"SpellBlockChance"}},
	"chance to block attacks and spells":              {names: []string{"BlockChance", "SpellBlockChance"}},
	"to block attack and spell damage":                {names: []string{"BlockChance", "SpellBlockChance"}},
	"maximum block chance":                            {names: []string{"BlockChanceMax"}},
	"maximum chance to block attack damage":           {names: []string{"BlockChanceMax"}},
	"maximum chance to block spell damage":            {names: []string{"SpellBlockChanceMax"}},
	"life gained when you block":                      {names: []string{"LifeOnBlock"}},
	"mana gained when you block":                      {names: []string{"ManaOnBlock"}},
	"maximum chance to dodge spell hits":              {names: []string{"SpellDodgeChanceMax"}},
	"to avoid physical damage from hits":              {names: []string{"AvoidPhysicalDamageChance"}},
	"to avoid fire damage when hit":                   {names: []string{"AvoidFireDamageChance"}},
	"to avoid fire damage from hits":                  {names: []string{"AvoidFireDamageChance"}},
	"to avoid cold damage when hit":                   {names: []string{"AvoidColdDamageChance"}},
	"to avoid cold damage from hits":                  {names: []string{"AvoidColdDamageChance"}},
	"to avoid lightning damage when hit":              {names: []string{"AvoidLightningDamageChance"}},
	"to avoid lightning damage from hits":             {names: []string{"AvoidLightningDamageChance"}},
	"to avoid elemental damage when hit":              {names: []string{"AvoidFireDamageChance", "AvoidColdDamageChance", "AvoidLightningDamageChance"}},
	"to avoid elemental damage from hits":             {names: []string{"AvoidFireDamageChance", "AvoidColdDamageChance", "AvoidLightningDamageChance"}},
	"to avoid projectiles":                            {names: []string{"AvoidProjectilesChance"}},
	"to avoid being stunned":                          {names: []string{"AvoidStun"}},
	"to avoid interruption from stuns while casting":  {names: []string{"AvoidInteruptStun"}},
	"to avoid being shocked":                          {names: []string{"AvoidShock"}},
	"to avoid being frozen":                           {names: []string{"AvoidFreeze"}},
	"to avoid being chilled":                          {names: []string{"AvoidChill"}},
	"to avoid being ignited":                          {names: []string{"AvoidIgnite"}},
	"to avoid elemental ailments":                     {names: []string{"AvoidShock", "AvoidFreeze", "AvoidChill", "AvoidIgnite", "AvoidSap", "AvoidBrittle", "AvoidScorch"}},
	"to avoid elemental status ailments":              {names: []string{"AvoidShock", "AvoidFreeze", "AvoidChill", "AvoidIgnite", "AvoidSap", "AvoidBrittle", "AvoidScorch"}},
	"to avoid bleeding":                               {names: []string{"AvoidBleed"}},
	"to avoid being poisoned":                         {names: []string{"AvoidPoison"}},
	"damage is taken from mana before life":           {names: []string{"DamageTakenFromManaBeforeLife"}},
	"lightning damage is taken from mana before life": {names: []string{"LightningDamageTakenFromManaBeforeLife"}},
	"damage taken from mana before life":              {names: []string{"DamageTakenFromManaBeforeLife"}},
	"effect of curses on you":                         {names: []string{"CurseEffectOnSelf"}},
	"effect of curses on them":                        {names: []string{"CurseEffectOnSelf"}},
	"life recovery rate":                              {names: []string{"LifeRecoveryRate"}},
	"mana recovery rate":                              {names: []string{"ManaRecoveryRate"}},
	"energy shield recovery rate":                     {names: []string{"EnergyShieldRecoveryRate"}},
	"energy shield regeneration rate":                 {names: []string{"EnergyShieldRegen"}},
	"recovery rate of life, mana and energy shield":   {names: []string{"LifeRecoveryRate", "ManaRecoveryRate", "EnergyShieldRecoveryRate"}},
	"recovery rate of life and energy shield":         {names: []string{"LifeRecoveryRate", "EnergyShieldRecoveryRate"}},
	"maximum life, mana and global energy shield":     {names: []string{"Life", "Mana", "EnergyShield"}, tag: mod.Global()},
	"non-chaos damage taken bypasses energy shield":   {names: []string{"PhysicalEnergyShieldBypass", "LightningEnergyShieldBypass", "ColdEnergyShieldBypass", "FireEnergyShieldBypass"}},

	// Stun/knockback modifiers
	"stun recovery":                {names: []string{"StunRecovery"}},
	"stun and block recovery":      {names: []string{"StunRecovery"}},
	"block and stun recovery":      {names: []string{"StunRecovery"}},
	"stun threshold":               {names: []string{"StunThreshold"}},
	"block recovery":               {names: []string{"BlockRecovery"}},
	"enemy stun threshold":         {names: []string{"EnemyStunThreshold"}},
	"stun duration on enemies":     {names: []string{"EnemyStunDuration"}},
	"stun duration":                {names: []string{"EnemyStunDuration"}},
	"to knock enemies back on hit": {names: []string{"EnemyKnockbackChance"}},
	"knockback distance":           {names: []string{"EnemyKnockbackDistance"}},

	// Auras/curses/buffs
	"aura effect":                                                {names: []string{"AuraEffect"}},
	"effect of non-curse auras you cast":                         {names: []string{"AuraEffect"}, tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeAura)), mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)}},
	"effect of non-curse auras from your skills":                 {names: []string{"AuraEffect"}, tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeAura)), mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)}},
	"effect of non-curse auras from your skills on your minions": {names: []string{"AuraEffectOnSelf"}, tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeAura)), mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)}, addToMinion: true},
	"effect of non-curse auras":                                  {names: []string{"AuraEffect"}, tag: mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)},
	"effect of your curses":                                      {names: []string{"CurseEffect"}},
	"effect of auras on you":                                     {names: []string{"AuraEffectOnSelf"}},
	"effect of auras on your minions":                            {names: []string{"AuraEffectOnSelf"}, addToMinion: true},
	"effect of auras from mines":                                 {names: []string{"AuraEffect"}, keywordFlags: mod.KeywordFlagMine},
	"effect of consecrated ground you create":                    {names: []string{"ConsecratedGroundEffect"}},
	"curse effect":                                               {names: []string{"CurseEffect"}},
	"effect of curses applied by bane":                           {names: []string{"CurseEffect"}, tag: mod.Condition("AppliedByBane")},
	"effect of your marks":                                       {names: []string{"CurseEffect"}, tag: mod.SkillType(string(data.SkillTypeMark))},
	"effect of arcane surge on you":                              {names: []string{"ArcaneSurgeEffect"}},
	"curse duration":                                             {names: []string{"Duration"}, keywordFlags: mod.KeywordFlagCurse},
	"hex duration":                                               {names: []string{"Duration"}, tag: mod.SkillType(string(data.SkillTypeHex))},
	"radius of auras":                                            {names: []string{"AreaOfEffect"}, keywordFlags: mod.KeywordFlagAura},
	"radius of curses":                                           {names: []string{"AreaOfEffect"}, keywordFlags: mod.KeywordFlagCurse},
	"buff effect":                                                {names: []string{"BuffEffect"}},
	"effect of buffs on you":                                     {names: []string{"BuffEffectOnSelf"}},
	"effect of buffs granted by your golems":                     {names: []string{"BuffEffect"}, tag: mod.SkillType(string(data.SkillTypeGolem))},
	"effect of buffs granted by socketed golem skills":           {names: []string{"BuffEffect"}, addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	"effect of the buff granted by your stone golems":            {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Stone Golem")},
	"effect of the buff granted by your lightning golems":        {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Lightning Golem")},
	"effect of the buff granted by your ice golems":              {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Ice Golem")},
	"effect of the buff granted by your flame golems":            {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Flame Golem")},
	"effect of the buff granted by your chaos golems":            {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Chaos Golem")},
	"effect of the buff granted by your carrion golems":          {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Carrion Golem")},
	"effect of offering spells":                                  {names: []string{"BuffEffect"}, tag: mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering")},
	"effect of offerings":                                        {names: []string{"BuffEffect"}, tag: mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering")},
	"effect of heralds on you":                                   {names: []string{"BuffEffect"}, tag: mod.SkillType(string(data.SkillTypeHerald))},
	"effect of herald buffs on you":                              {names: []string{"BuffEffect"}, tag: mod.SkillType(string(data.SkillTypeHerald))},
	"effect of buffs granted by your active ancestor totems":     {names: []string{"BuffEffect"}, tag: mod.SkillName("Ancestral Warchief", "Ancestral Protector", "Earthbreaker")},
	"effect of buffs your ancestor totems grant ":                {names: []string{"BuffEffect"}, tag: mod.SkillName("Ancestral Warchief", "Ancestral Protector", "Earthbreaker")},
	"effect of withered":                                         {names: []string{"WitherEffect"}},
	"warcry effect":                                              {names: []string{"BuffEffect"}, keywordFlags: mod.KeywordFlagWarcry},
	"aspect of the avian buff effect":                            {names: []string{"BuffEffect"}, tag: mod.SkillName("Aspect of the Avian")},
	"maximum rage":                                               {names: []string{"MaximumRage"}},
	"maximum fortification":                                      {names: []string{"MaximumFortification"}},
	"fortification":                                              {names: []string{"FortificationStacks"}},

	// Charges
	"maximum power charge":                                {names: []string{"PowerChargesMax"}},
	"maximum power charges":                               {names: []string{"PowerChargesMax"}},
	"minimum power charge":                                {names: []string{"PowerChargesMin"}},
	"minimum power charges":                               {names: []string{"PowerChargesMin"}},
	"power charge duration":                               {names: []string{"PowerChargesDuration"}},
	"maximum frenzy charge":                               {names: []string{"FrenzyChargesMax"}},
	"maximum frenzy charges":                              {names: []string{"FrenzyChargesMax"}},
	"minimum frenzy charge":                               {names: []string{"FrenzyChargesMin"}},
	"minimum frenzy charges":                              {names: []string{"FrenzyChargesMin"}},
	"frenzy charge duration":                              {names: []string{"FrenzyChargesDuration"}},
	"maximum endurance charge":                            {names: []string{"EnduranceChargesMax"}},
	"maximum endurance charges":                           {names: []string{"EnduranceChargesMax"}},
	"minimum endurance charge":                            {names: []string{"EnduranceChargesMin"}},
	"minimum endurance charges":                           {names: []string{"EnduranceChargesMin"}},
	"minimum endurance, frenzy and power charges":         {names: []string{"PowerChargesMin", "FrenzyChargesMin", "EnduranceChargesMin"}},
	"endurance charge duration":                           {names: []string{"EnduranceChargesDuration"}},
	"maximum frenzy charges and maximum power charges":    {names: []string{"FrenzyChargesMax", "PowerChargesMax"}},
	"maximum power charges and maximum endurance charges": {names: []string{"PowerChargesMax", "EnduranceChargesMax"}},
	"endurance, frenzy and power charge duration":         {names: []string{"PowerChargesDuration", "FrenzyChargesDuration", "EnduranceChargesDuration"}},
	"maximum siphoning charge":                            {names: []string{"SiphoningChargesMax"}},
	"maximum siphoning charges":                           {names: []string{"SiphoningChargesMax"}},
	"maximum challenger charges":                          {names: []string{"ChallengerChargesMax"}},
	"maximum blitz charges":                               {names: []string{"BlitzChargesMax"}},
	"maximum number of crab barriers":                     {names: []string{"CrabBarriersMax"}},
	"maximum blood charges":                               {names: []string{"BloodChargesMax"}},
	"charge duration":                                     {names: []string{"ChargeDuration"}},

	// On hit/kill/leech effects
	"life gained on kill":                                               {names: []string{"LifeOnKill"}},
	"mana gained on kill":                                               {names: []string{"ManaOnKill"}},
	"life gained for each enemy hit":                                    {names: []string{"LifeOnHit"}},
	"life gained for each enemy hit by attacks":                         {names: []string{"LifeOnHit"}, flags: mod.MFlagAttack},
	"life gained for each enemy hit by your attacks":                    {names: []string{"LifeOnHit"}, flags: mod.MFlagAttack},
	"life gained for each enemy hit by spells":                          {names: []string{"LifeOnHit"}, flags: mod.MFlagSpell},
	"life gained for each enemy hit by your spells":                     {names: []string{"LifeOnHit"}, flags: mod.MFlagSpell},
	"mana gained for each enemy hit by attacks":                         {names: []string{"ManaOnHit"}, flags: mod.MFlagAttack},
	"mana gained for each enemy hit by your attacks":                    {names: []string{"ManaOnHit"}, flags: mod.MFlagAttack},
	"energy shield gained for each enemy hit":                           {names: []string{"EnergyShieldOnHit"}},
	"energy shield gained for each enemy hit by attacks":                {names: []string{"EnergyShieldOnHit"}, flags: mod.MFlagAttack},
	"energy shield gained for each enemy hit by your attacks":           {names: []string{"EnergyShieldOnHit"}, flags: mod.MFlagAttack},
	"life and mana gained for each enemy hit":                           {names: []string{"LifeOnHit", "ManaOnHit"}, flags: mod.MFlagAttack},
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
	"to impale enemies on hit":                                          {names: []string{"ImpaleChance"}},
	"to impale on spell hit":                                            {names: []string{"ImpaleChance"}, flags: mod.MFlagSpell},
	"impale effect":                                                     {names: []string{"ImpaleEffect"}},
	"effect of impales you inflict":                                     {names: []string{"ImpaleEffect"}},

	// Projectile modifiers
	"projectile":       {names: []string{"ProjectileCount"}},
	"projectiles":      {names: []string{"ProjectileCount"}},
	"projectile speed": {names: []string{"ProjectileSpeed"}},
	"arrow speed":      {names: []string{"ProjectileSpeed"}, flags: mod.MFlagBow},

	// Totem/trap/mine/brand modifiers
	"totem placement speed":                      {names: []string{"TotemPlacementSpeed"}},
	"totem life":                                 {names: []string{"TotemLife"}},
	"totem duration":                             {names: []string{"TotemDuration"}},
	"maximum number of summoned totems":          {names: []string{"ActiveTotemLimit"}},
	"maximum number of summoned totems.":         {names: []string{"ActiveTotemLimit"}}, // Mark plz
	"maximum number of summoned ballista totems": {names: []string{"ActiveBallistaLimit"}, tag: mod.SkillType(string(data.SkillTypeRangedAttack))},
	"trap throwing speed":                        {names: []string{"TrapThrowingSpeed"}},
	"trap and mine throwing speed":               {names: []string{"TrapThrowingSpeed", "MineLayingSpeed"}},
	"trap trigger area of effect":                {names: []string{"TrapTriggerAreaOfEffect"}},
	"trap duration":                              {names: []string{"TrapDuration"}},
	"cooldown recovery speed for throwing traps": {names: []string{"CooldownRecovery"}, keywordFlags: mod.KeywordFlagTrap},
	"cooldown recovery rate for throwing traps":  {names: []string{"CooldownRecovery"}, keywordFlags: mod.KeywordFlagTrap},
	"mine laying speed":                          {names: []string{"MineLayingSpeed"}},
	"mine throwing speed":                        {names: []string{"MineLayingSpeed"}},
	"mine detonation area of effect":             {names: []string{"MineDetonationAreaOfEffect"}},
	"mine duration":                              {names: []string{"MineDuration"}},
	"activation frequency":                       {names: []string{"BrandActivationFrequency"}},
	"brand activation frequency":                 {names: []string{"BrandActivationFrequency"}},
	"brand attachment range":                     {names: []string{"BrandAttachmentRange"}},

	// Minion modifiers
	"maximum number of skeletons":               {names: []string{"ActiveSkeletonLimit"}},
	"maximum number of zombies":                 {names: []string{"ActiveZombieLimit"}},
	"maximum number of raised zombies":          {names: []string{"ActiveZombieLimit"}},
	"number of zombies allowed":                 {names: []string{"ActiveZombieLimit"}},
	"maximum number of spectres":                {names: []string{"ActiveSpectreLimit"}},
	"maximum number of golems":                  {names: []string{"ActiveGolemLimit"}},
	"maximum number of summoned golems":         {names: []string{"ActiveGolemLimit"}},
	"maximum number of summoned raging spirits": {names: []string{"ActiveRagingSpiritLimit"}},
	"maximum number of raging spirits":          {names: []string{"ActiveRagingSpiritLimit"}},
	"maximum number of summoned phantasms":      {names: []string{"ActivePhantasmLimit"}},
	"maximum number of summoned holy relics":    {names: []string{"ActiveHolyRelicLimit"}},
	"minion duration":                           {names: []string{"Duration"}, tag: mod.SkillType(string(data.SkillTypeCreatesMinion))},
	"skeleton duration":                         {names: []string{"Duration"}, tag: mod.SkillName("Summon Skeleton")},
	"sentinel of dominance duration":            {names: []string{"Duration"}, tag: mod.SkillName("Dominating Blow")},

	// Other skill modifiers
	"radius":                                {names: []string{"AreaOfEffect"}},
	"radius of area skills":                 {names: []string{"AreaOfEffect"}},
	"area of effect radius":                 {names: []string{"AreaOfEffect"}},
	"area of effect":                        {names: []string{"AreaOfEffect"}},
	"area of effect of skills":              {names: []string{"AreaOfEffect"}},
	"area of effect of area skills":         {names: []string{"AreaOfEffect"}},
	"aspect of the spider area of effect":   {names: []string{"AreaOfEffect"}, tag: mod.SkillName("Aspect of the Spider")},
	"firestorm explosion area of effect":    {names: []string{"AreaOfEffectSecondary"}, tag: mod.SkillName("Firestorm")},
	"duration":                              {names: []string{"Duration"}},
	"skill effect duration":                 {names: []string{"Duration"}},
	"chaos skill effect duration":           {names: []string{"Duration"}, keywordFlags: mod.KeywordFlagChaos},
	"aspect of the spider debuff duration":  {names: []string{"Duration"}, tag: mod.SkillName("Aspect of the Spider")},
	"fire trap burning ground duration":     {names: []string{"Duration"}, tag: mod.SkillName("Fire Trap")},
	"cooldown recovery":                     {names: []string{"CooldownRecovery"}},
	"cooldown recovery speed":               {names: []string{"CooldownRecovery"}},
	"cooldown recovery rate":                {names: []string{"CooldownRecovery"}},
	"weapon range":                          {names: []string{"WeaponRange"}},
	"melee range":                           {names: []string{"MeleeWeaponRange"}},
	"melee weapon range":                    {names: []string{"MeleeWeaponRange"}},
	"melee weapon and unarmed range":        {names: []string{"MeleeWeaponRange", "UnarmedRange"}},
	"melee weapon and unarmed attack range": {names: []string{"MeleeWeaponRange", "UnarmedRange"}},
	"melee strike range":                    {names: []string{"MeleeWeaponRange", "UnarmedRange"}},
	"to deal double damage":                 {names: []string{"DoubleDamageChance"}},

	// Buffs
	"onslaught effect":          {names: []string{"OnslaughtEffect"}},
	"adrenaline duration":       {names: []string{"AdrenalineDuration"}},
	"effect of tailwind on you": {names: []string{"TailwindEffectOnSelf"}},
	"elusive effect":            {names: []string{"ElusiveEffect"}},
	"effect of elusive on you":  {names: []string{"ElusiveEffect"}},
	"effect of infusion":        {names: []string{"InfusionEffect"}},

	// Basic damage types
	"damage":           {names: []string{"Damage"}},
	"physical damage":  {names: []string{"PhysicalDamage"}},
	"lightning damage": {names: []string{"LightningDamage"}},
	"cold damage":      {names: []string{"ColdDamage"}},
	"fire damage":      {names: []string{"FireDamage"}},
	"chaos damage":     {names: []string{"ChaosDamage"}},
	"non-chaos damage": {names: []string{"NonChaosDamage"}},
	"elemental damage": {names: []string{"ElementalDamage"}},

	// Other damage forms
	"attack damage":                        {names: []string{"Damage"}, flags: mod.MFlagAttack},
	"attack physical damage":               {names: []string{"PhysicalDamage"}, flags: mod.MFlagAttack},
	"physical attack damage":               {names: []string{"PhysicalDamage"}, flags: mod.MFlagAttack},
	"minimum physical attack damage":       {names: []string{"MinPhysicalDamage"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"maximum physical attack damage":       {names: []string{"MaxPhysicalDamage"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"physical weapon damage":               {names: []string{"PhysicalDamage"}, flags: mod.MFlagWeapon},
	"physical damage with weapons":         {names: []string{"PhysicalDamage"}, flags: mod.MFlagWeapon},
	"melee damage":                         {names: []string{"Damage"}, flags: mod.MFlagMelee},
	"physical melee damage":                {names: []string{"PhysicalDamage"}, flags: mod.MFlagMelee},
	"melee physical damage":                {names: []string{"PhysicalDamage"}, flags: mod.MFlagMelee},
	"projectile damage":                    {names: []string{"Damage"}, flags: mod.MFlagProjectile},
	"projectile attack damage":             {names: []string{"Damage"}, flags: mod.MFlagProjectile | mod.MFlagAttack},
	"bow damage":                           {names: []string{"Damage"}, flags: mod.MFlagBow | mod.MFlagHit},
	"damage with arrow hits":               {names: []string{"Damage"}, flags: mod.MFlagBow | mod.MFlagHit},
	"wand damage":                          {names: []string{"Damage"}, flags: mod.MFlagWand | mod.MFlagHit},
	"wand physical damage":                 {names: []string{"PhysicalDamage"}, flags: mod.MFlagWand | mod.MFlagHit},
	"claw physical damage":                 {names: []string{"PhysicalDamage"}, flags: mod.MFlagClaw | mod.MFlagHit},
	"sword physical damage":                {names: []string{"PhysicalDamage"}, flags: mod.MFlagSword | mod.MFlagHit},
	"damage over time":                     {names: []string{"Damage"}, flags: mod.MFlagDot},
	"physical damage over time":            {names: []string{"PhysicalDamage"}, keywordFlags: mod.KeywordFlagPhysicalDot},
	"cold damage over time":                {names: []string{"ColdDamage"}, keywordFlags: mod.KeywordFlagColdDot},
	"chaos damage over time":               {names: []string{"ChaosDamage"}, keywordFlags: mod.KeywordFlagChaosDot},
	"burning damage":                       {names: []string{"FireDamage"}, keywordFlags: mod.KeywordFlagFireDot},
	"damage with ignite":                   {names: []string{"Damage"}, keywordFlags: mod.KeywordFlagIgnite},
	"damage with ignites":                  {names: []string{"Damage"}, keywordFlags: mod.KeywordFlagIgnite},
	"incinerate damage for each stage":     {names: []string{"Damage"}, tagList: []mod.Tag{mod.Multiplier("IncinerateStage").Base(0), mod.SkillName("Incinerate")}},
	"physical damage over time multiplier": {names: []string{"PhysicalDotMultiplier"}},
	"fire damage over time multiplier":     {names: []string{"FireDotMultiplier"}},
	"cold damage over time multiplier":     {names: []string{"ColdDotMultiplier"}},
	"chaos damage over time multiplier":    {names: []string{"ChaosDotMultiplier"}},
	"damage over time multiplier":          {names: []string{"DotMultiplier"}},

	// Crit/accuracy/speed modifiers
	"critical strike chance":            {names: []string{"CritChance"}},
	"attack critical strike chance":     {names: []string{"CritChance"}, flags: mod.MFlagAttack},
	"critical strike multiplier":        {names: []string{"CritMultiplier"}},
	"attack critical strike multiplier": {names: []string{"CritMultiplier"}, flags: mod.MFlagAttack},
	"accuracy":                          {names: []string{"Accuracy"}},
	"accuracy rating":                   {names: []string{"Accuracy"}},
	"minion accuracy rating":            {names: []string{"Accuracy"}, addToMinion: true},
	"attack speed":                      {names: []string{"Speed"}, flags: mod.MFlagAttack},
	"cast speed":                        {names: []string{"Speed"}, flags: mod.MFlagCast},
	"warcry speed":                      {names: []string{"WarcrySpeed"}, keywordFlags: mod.KeywordFlagWarcry},
	"attack and cast speed":             {names: []string{"Speed"}},

	// Elemental ailments
	"to shock":                                    {names: []string{"EnemyShockChance"}},
	"shock chance":                                {names: []string{"EnemyShockChance"}},
	"to freeze":                                   {names: []string{"EnemyFreezeChance"}},
	"freeze chance":                               {names: []string{"EnemyFreezeChance"}},
	"to ignite":                                   {names: []string{"EnemyIgniteChance"}},
	"ignite chance":                               {names: []string{"EnemyIgniteChance"}},
	"to freeze, shock and ignite":                 {names: []string{"EnemyFreezeChance", "EnemyShockChance", "EnemyIgniteChance"}},
	"to scorch enemies":                           {names: []string{"EnemyScorchChance"}},
	"to inflict brittle":                          {names: []string{"EnemyBrittleChance"}},
	"to sap enemies":                              {names: []string{"EnemySapChance"}},
	"effect of scorch":                            {names: []string{"EnemyScorchEffect"}},
	"effect of sap":                               {names: []string{"EnemySapEffect"}},
	"effect of brittle":                           {names: []string{"EnemyBrittleEffect"}},
	"effect of shock":                             {names: []string{"EnemyShockEffect"}},
	"effect of shock on you":                      {names: []string{"SelfShockEffect"}},
	"effect of shock you inflict":                 {names: []string{"EnemyShockEffect"}},
	"effect of lightning ailments":                {names: []string{"EnemyShockEffect", "EnemySapEffect"}},
	"effect of chill":                             {names: []string{"EnemyChillEffect"}},
	"effect of chill and shock on you":            {names: []string{"SelfChillEffect", "SelfShockEffect"}},
	"chill effect":                                {names: []string{"EnemyChillEffect"}},
	"effect of chill you inflict":                 {names: []string{"EnemyChillEffect"}},
	"effect of cold ailments":                     {names: []string{"EnemyChillEffect", "EnemyBrittleEffect"}},
	"effect of chill on you":                      {names: []string{"SelfChillEffect"}},
	"effect of non-damaging ailments":             {names: []string{"EnemyShockEffect", "EnemyChillEffect", "EnemyFreezeEffect", "EnemyScorchEffect", "EnemyBrittleEffect", "EnemySapEffect"}},
	"effect of non-damaging ailments you inflict": {names: []string{"EnemyShockEffect", "EnemyChillEffect", "EnemyFreezeEffect", "EnemyScorchEffect", "EnemyBrittleEffect", "EnemySapEffect"}},
	"shock duration":                              {names: []string{"EnemyShockDuration"}},
	"shock duration on you":                       {names: []string{"SelfShockDuration"}},
	"duration of lightning ailments":              {names: []string{"EnemyShockDuration", "EnemySapDuration"}},
	"freeze duration":                             {names: []string{"EnemyFreezeDuration"}},
	"freeze duration on you":                      {names: []string{"SelfFreezeDuration"}},
	"chill duration":                              {names: []string{"EnemyChillDuration"}},
	"chill duration on you":                       {names: []string{"SelfChillDuration"}},
	"duration of cold ailments":                   {names: []string{"EnemyFreezeDuration", "EnemyChillDuration", "EnemyBrittleDuration"}},
	"ignite duration":                             {names: []string{"EnemyIgniteDuration"}},
	"ignite duration on you":                      {names: []string{"SelfIgniteDuration"}},
	"duration of ignite on you":                   {names: []string{"SelfIgniteDuration"}},
	"duration of elemental ailments":              {names: []string{"EnemyShockDuration", "EnemyFreezeDuration", "EnemyChillDuration", "EnemyIgniteDuration", "EnemyScorchDuration", "EnemyBrittleDuration", "EnemySapDuration"}},
	"duration of elemental ailments on you":       {names: []string{"SelfShockDuration", "SelfFreezeDuration", "SelfChillDuration", "SelfIgniteDuration", "SelfScorchDuration", "SelfBrittleDuration", "SelfSapDuration"}},
	"duration of elemental status ailments":       {names: []string{"EnemyShockDuration", "EnemyFreezeDuration", "EnemyChillDuration", "EnemyIgniteDuration", "EnemyScorchDuration", "EnemyBrittleDuration", "EnemySapDuration"}},
	"duration of ailments":                        {names: []string{"EnemyShockDuration", "EnemyFreezeDuration", "EnemyChillDuration", "EnemyIgniteDuration", "EnemyPoisonDuration", "EnemyBleedDuration", "EnemyScorchDuration", "EnemyBrittleDuration", "EnemySapDuration"}},
	"duration of ailments on you":                 {names: []string{"SelfShockDuration", "SelfFreezeDuration", "SelfChillDuration", "SelfIgniteDuration", "SelfPoisonDuration", "SelfBleedDuration", "SelfScorchDuration", "SelfBrittleDuration", "SelfSapDuration"}},
	"elemental ailment duration on you":           {names: []string{"SelfShockDuration", "SelfFreezeDuration", "SelfChillDuration", "SelfIgniteDuration", "SelfScorchDuration", "SelfBrittleDuration", "SelfSapDuration"}},
	"duration of ailments you inflict":            {names: []string{"EnemyShockDuration", "EnemyFreezeDuration", "EnemyChillDuration", "EnemyIgniteDuration", "EnemyPoisonDuration", "EnemyBleedDuration", "EnemyScorchDuration", "EnemyBrittleDuration", "EnemySapDuration"}},
	"duration of ailments inflicted":              {names: []string{"EnemyShockDuration", "EnemyFreezeDuration", "EnemyChillDuration", "EnemyIgniteDuration", "EnemyPoisonDuration", "EnemyBleedDuration", "EnemyScorchDuration", "EnemyBrittleDuration", "EnemySapDuration"}},

	// Other ailments
	"to poison":                       {names: []string{"PoisonChance"}},
	"to cause poison":                 {names: []string{"PoisonChance"}},
	"to poison on hit":                {names: []string{"PoisonChance"}},
	"poison duration":                 {names: []string{"EnemyPoisonDuration"}},
	"duration of poisons you inflict": {names: []string{"EnemyPoisonDuration"}},
	"to cause bleeding":               {names: []string{"BleedChance"}},
	"to cause bleeding on hit":        {names: []string{"BleedChance"}},
	"to inflict bleeding":             {names: []string{"BleedChance"}},
	"to inflict bleeding on hit":      {names: []string{"BleedChance"}},
	"bleed duration":                  {names: []string{"EnemyBleedDuration"}},
	"bleeding duration":               {names: []string{"EnemyBleedDuration"}},

	// Misc modifiers
	"movement speed":                        {names: []string{"MovementSpeed"}},
	"attack, cast and movement speed":       {names: []string{"Speed", "MovementSpeed"}},
	"action speed":                          {names: []string{"ActionSpeed"}},
	"light radius":                          {names: []string{"LightRadius"}},
	"rarity of items found":                 {names: []string{"LootRarity"}},
	"quantity of items found":               {names: []string{"LootQuantity"}},
	"item quantity":                         {names: []string{"LootQuantity"}},
	"strength requirement":                  {names: []string{"StrRequirement"}},
	"dexterity requirement":                 {names: []string{"DexRequirement"}},
	"intelligence requirement":              {names: []string{"IntRequirement"}},
	"omni requirement":                      {names: []string{"OmniRequirement"}},
	"strength and intelligence requirement": {names: []string{"StrRequirement", "IntRequirement"}},
	"attribute requirements":                {names: []string{"StrRequirement", "DexRequirement", "IntRequirement"}},
	"effect of socketed jewels":             {names: []string{"SocketedJewelEffect"}},
	"effect of socketed abyss jewels":       {names: []string{"SocketedJewelEffect"}},
	"to inflict fire exposure on hit":       {names: []string{"FireExposureChance"}},
	"to apply fire exposure on hit":         {names: []string{"FireExposureChance"}},
	"to inflict cold exposure on hit":       {names: []string{"ColdExposureChance"}},
	"to apply cold exposure on hit":         {names: []string{"ColdExposureChance"}},
	"to inflict lightning exposure on hit":  {names: []string{"LightningExposureChance"}},
	"to apply lightning exposure on hit":    {names: []string{"LightningExposureChance"}},

	// Flask modifiers
	"effect":                             {names: []string{"FlaskEffect"}},
	"effect of flasks":                   {names: []string{"FlaskEffect"}},
	"effect of flasks on you":            {names: []string{"FlaskEffect"}},
	"amount recovered":                   {names: []string{"FlaskRecovery"}},
	"life recovered":                     {names: []string{"FlaskRecovery"}},
	"life recovery from flasks used":     {names: []string{"FlaskLifeRecovery"}},
	"mana recovered":                     {names: []string{"FlaskRecovery"}},
	"life recovery from flasks":          {names: []string{"FlaskLifeRecovery"}},
	"mana recovery from flasks":          {names: []string{"FlaskManaRecovery"}},
	"life and mana recovery from flasks": {names: []string{"FlaskLifeRecovery", "FlaskManaRecovery"}},
	"flask effect duration":              {names: []string{"FlaskDuration"}},
	"recovery speed":                     {names: []string{"FlaskRecoveryRate"}},
	"recovery rate":                      {names: []string{"FlaskRecoveryRate"}},
	"flask recovery rate":                {names: []string{"FlaskRecoveryRate"}},
	"flask recovery speed":               {names: []string{"FlaskRecoveryRate"}},
	"flask life recovery rate":           {names: []string{"FlaskLifeRecoveryRate"}},
	"flask mana recovery rate":           {names: []string{"FlaskManaRecoveryRate"}},
	"extra charges":                      {names: []string{"FlaskCharges"}},
	"maximum charges":                    {names: []string{"FlaskCharges"}},
	"charges used":                       {names: []string{"FlaskChargesUsed"}},
	"charges per use":                    {names: []string{"FlaskChargesUsed"}},
	"flask charges used":                 {names: []string{"FlaskChargesUsed"}},
	"flask charges gained":               {names: []string{"FlaskChargesGained"}},
	"charge recovery":                    {names: []string{"FlaskChargeRecovery"}},
	"impales you inflict last":           {names: []string{"ImpaleStacksMax"}},
}

// List of modifier flags
var modFlagListCompiled map[string]CompiledList[modNameListType]
var modFlagList = map[string]modNameListType{
	// Weapon types
	"with axes":                      {flags: mod.MFlagAxe | mod.MFlagHit},
	"to axe attacks":                 {flags: mod.MFlagAxe | mod.MFlagHit},
	"with axe attacks":               {flags: mod.MFlagAxe | mod.MFlagHit},
	"with axes or swords":            {flags: mod.MFlagHit, tag: mod.ModFlagOr(mod.MFlagAxe | mod.MFlagSword)},
	"with bows":                      {flags: mod.MFlagBow | mod.MFlagHit},
	"to bow attacks":                 {flags: mod.MFlagBow | mod.MFlagHit},
	"with bow attacks":               {flags: mod.MFlagBow | mod.MFlagHit},
	"with claws":                     {flags: mod.MFlagClaw | mod.MFlagHit},
	"with claws or daggers":          {flags: mod.MFlagHit, tag: mod.ModFlagOr(mod.MFlagClaw | mod.MFlagDagger)},
	"to claw attacks":                {flags: mod.MFlagClaw | mod.MFlagHit},
	"with claw attacks":              {flags: mod.MFlagClaw | mod.MFlagHit},
	"dealt with claws":               {flags: mod.MFlagClaw | mod.MFlagHit},
	"with daggers":                   {flags: mod.MFlagDagger | mod.MFlagHit},
	"to dagger attacks":              {flags: mod.MFlagDagger | mod.MFlagHit},
	"with dagger attacks":            {flags: mod.MFlagDagger | mod.MFlagHit},
	"with maces":                     {flags: mod.MFlagMace | mod.MFlagHit},
	"to mace attacks":                {flags: mod.MFlagMace | mod.MFlagHit},
	"with mace attacks":              {flags: mod.MFlagMace | mod.MFlagHit},
	"with maces and sceptres":        {flags: mod.MFlagMace | mod.MFlagHit},
	"with maces or sceptres":         {flags: mod.MFlagMace | mod.MFlagHit},
	"with maces, sceptres or staves": {flags: mod.MFlagHit, tag: mod.ModFlagOr(mod.MFlagMace | mod.MFlagStaff)},
	"to mace and sceptre attacks":    {flags: mod.MFlagMace | mod.MFlagHit},
	"to mace or sceptre attacks":     {flags: mod.MFlagMace | mod.MFlagHit},
	"with mace or sceptre attacks":   {flags: mod.MFlagMace | mod.MFlagHit},
	"with staves":                    {flags: mod.MFlagStaff | mod.MFlagHit},
	"to staff attacks":               {flags: mod.MFlagStaff | mod.MFlagHit},
	"with staff attacks":             {flags: mod.MFlagStaff | mod.MFlagHit},
	"with swords":                    {flags: mod.MFlagSword | mod.MFlagHit},
	"to sword attacks":               {flags: mod.MFlagSword | mod.MFlagHit},
	"with sword attacks":             {flags: mod.MFlagSword | mod.MFlagHit},
	"with wands":                     {flags: mod.MFlagWand | mod.MFlagHit},
	"to wand attacks":                {flags: mod.MFlagWand | mod.MFlagHit},
	"with wand attacks":              {flags: mod.MFlagWand | mod.MFlagHit},
	"unarmed":                        {flags: mod.MFlagUnarmed | mod.MFlagHit},
	"unarmed melee":                  {flags: mod.MFlagUnarmed | mod.MFlagMelee | mod.MFlagHit},
	"with unarmed attacks":           {flags: mod.MFlagUnarmed | mod.MFlagHit},
	"with unarmed melee attacks":     {flags: mod.MFlagUnarmed | mod.MFlagMelee},
	"to unarmed attacks":             {flags: mod.MFlagUnarmed | mod.MFlagHit},
	"to unarmed melee hits":          {flags: mod.MFlagUnarmed | mod.MFlagMelee | mod.MFlagHit},
	"with one handed weapons":        {flags: mod.MFlagWeapon1H | mod.MFlagHit},
	"with one handed melee weapons":  {flags: mod.MFlagWeapon1H | mod.MFlagWeaponMelee | mod.MFlagHit},
	"with two handed weapons":        {flags: mod.MFlagWeapon2H | mod.MFlagHit},
	"with two handed melee weapons":  {flags: mod.MFlagWeapon2H | mod.MFlagWeaponMelee | mod.MFlagHit},
	"with ranged weapons":            {flags: mod.MFlagWeaponRanged | mod.MFlagHit},

	// Skill types
	"spell":                            {flags: mod.MFlagSpell},
	"with spells":                      {flags: mod.MFlagSpell},
	"with spell damage":                {flags: mod.MFlagSpell},
	"for spells":                       {flags: mod.MFlagSpell},
	"for spell damage":                 {flags: mod.MFlagSpell},
	"with attacks":                     {keywordFlags: mod.KeywordFlagAttack},
	"with attack skills":               {keywordFlags: mod.KeywordFlagAttack},
	"for attacks":                      {flags: mod.MFlagAttack},
	"for attack damage":                {flags: mod.MFlagAttack},
	"weapon":                           {flags: mod.MFlagWeapon},
	"with weapons":                     {flags: mod.MFlagWeapon},
	"melee":                            {flags: mod.MFlagMelee},
	"with melee attacks":               {flags: mod.MFlagMelee},
	"with melee critical strikes":      {flags: mod.MFlagMelee, tag: mod.Condition("CriticalStrike")},
	"with melee skills":                {flags: mod.MFlagMelee},
	"with bow skills":                  {keywordFlags: mod.KeywordFlagBow},
	"on melee hit":                     {flags: mod.MFlagMelee},
	"with hits":                        {keywordFlags: mod.KeywordFlagHit},
	"with hits against nearby enemies": {keywordFlags: mod.KeywordFlagHit},
	"with hits and ailments":           {keywordFlags: mod.KeywordFlagHit | mod.KeywordFlagAilment},
	"with ailments":                    {flags: mod.MFlagAilment},
	"with ailments from attack skills": {flags: mod.MFlagAilment, keywordFlags: mod.KeywordFlagAttack},
	"with poison":                      {keywordFlags: mod.KeywordFlagPoison},
	"with bleeding":                    {keywordFlags: mod.KeywordFlagBleed},
	"for ailments":                     {flags: mod.MFlagAilment},
	"for poison":                       {keywordFlags: mod.KeywordFlagPoison | mod.KeywordFlagMatchAll},
	"for bleeding":                     {keywordFlags: mod.KeywordFlagBleed},
	"for ignite":                       {keywordFlags: mod.KeywordFlagIgnite},
	"area":                             {flags: mod.MFlagArea},
	"mine":                             {keywordFlags: mod.KeywordFlagMine},
	"with mines":                       {keywordFlags: mod.KeywordFlagMine},
	"trap":                             {keywordFlags: mod.KeywordFlagTrap},
	"with traps":                       {keywordFlags: mod.KeywordFlagTrap},
	"for traps":                        {keywordFlags: mod.KeywordFlagTrap},
	"that place mines or throw traps":  {keywordFlags: mod.KeywordFlagMine | mod.KeywordFlagTrap},
	"that throw mines":                 {keywordFlags: mod.KeywordFlagMine},
	"that throw traps":                 {keywordFlags: mod.KeywordFlagTrap},
	"brand":                            {tag: mod.SkillType(string(data.SkillTypeBrand))},
	"totem":                            {keywordFlags: mod.KeywordFlagTotem},
	"with totem skills":                {keywordFlags: mod.KeywordFlagTotem},
	"for skills used by totems":        {keywordFlags: mod.KeywordFlagTotem},
	"totem skills that cast an aura":   {tag: mod.SkillType(string(data.SkillTypeAura)), keywordFlags: mod.KeywordFlagTotem},
	"aura skills that summon totems":   {tag: mod.SkillType(string(data.SkillTypeAura)), keywordFlags: mod.KeywordFlagTotem},
	"of aura skills":                   {tag: mod.SkillType(string(data.SkillTypeAura))},
	"of curse skills":                  {keywordFlags: mod.KeywordFlagCurse},
	"with curse skills":                {keywordFlags: mod.KeywordFlagCurse},
	"of curse aura skills":             {tag: mod.SkillType(string(data.SkillTypeAura)), keywordFlags: mod.KeywordFlagCurse},
	"of curse auras":                   {keywordFlags: mod.KeywordFlagCurse | mod.KeywordFlagAura | mod.KeywordFlagMatchAll},
	"of hex skills":                    {tag: mod.SkillType(string(data.SkillTypeHex))},
	"with hex skills":                  {tag: mod.SkillType(string(data.SkillTypeHex))},
	"of herald skills":                 {tag: mod.SkillType(string(data.SkillTypeHerald))},
	"with herald skills":               {tag: mod.SkillType(string(data.SkillTypeHerald))},
	"with hits from herald skills":     {tag: mod.SkillType(string(data.SkillTypeHerald)), keywordFlags: mod.KeywordFlagHit},
	"minion skills":                    {tag: mod.SkillType(string(data.SkillTypeMinion))},
	"of minion skills":                 {tag: mod.SkillType(string(data.SkillTypeMinion))},
	"for curses":                       {keywordFlags: mod.KeywordFlagCurse},
	"for hexes":                        {tag: mod.SkillType(string(data.SkillTypeHex))},
	"warcry":                           {keywordFlags: mod.KeywordFlagWarcry},
	"vaal":                             {keywordFlags: mod.KeywordFlagVaal},
	"vaal skill":                       {keywordFlags: mod.KeywordFlagVaal},
	"with vaal skills":                 {keywordFlags: mod.KeywordFlagVaal},
	"with non-vaal skills":             {tag: mod.SkillType(string(data.SkillTypeVaal)).Neg(true)},
	"with movement skills":             {keywordFlags: mod.KeywordFlagMovement},
	"of movement skills":               {keywordFlags: mod.KeywordFlagMovement},
	"of movement skills used":          {keywordFlags: mod.KeywordFlagMovement},
	"of travel skills":                 {tag: mod.SkillType(string(data.SkillTypeTravel))},
	"of banner skills":                 {tag: mod.SkillType(string(data.SkillTypeBanner))},
	"with lightning skills":            {keywordFlags: mod.KeywordFlagLightning},
	"with cold skills":                 {keywordFlags: mod.KeywordFlagCold},
	"with fire skills":                 {keywordFlags: mod.KeywordFlagFire},
	"with elemental skills":            {keywordFlags: mod.KeywordFlagLightning | mod.KeywordFlagCold | mod.KeywordFlagFire},
	"with chaos skills":                {keywordFlags: mod.KeywordFlagChaos},
	"with physical skills":             {tag: mod.SkillType(string(data.SkillTypePhysical))}, // TODO Verify: https://canary.discord.com/channels/676181894152454150/677968792164368401/1016528721588523159
	"with channelling skills":          {tag: mod.SkillType(string(data.SkillTypeChannel))},
	"channelling skills":               {tag: mod.SkillType(string(data.SkillTypeChannel))},
	"with brand skills":                {tag: mod.SkillType(string(data.SkillTypeBrand))},
	"for stance skills":                {tag: mod.SkillType(string(data.SkillTypeStance))},
	"of stance skills":                 {tag: mod.SkillType(string(data.SkillTypeStance))},
	"with skills that cost life":       {tag: mod.StatThreshold("LifeCost", 1)},
	"minion":                           {addToMinion: true},
	"zombie":                           {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie")},
	"raised zombie":                    {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie")},
	"skeleton":                         {addToMinion: true, addToMinionTag: mod.SkillName("Summon Skeleton")},
	"spectre":                          {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre")},
	"raised spectre":                   {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre")},
	"golem":                            {addToMinion: true, addToMinionTag: mod.SkillType(string(data.SkillTypeGolem))},
	"chaos golem":                      {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem")},
	"flame golem":                      {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem")},
	"increased flame golem":            {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem")},
	"ice golem":                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem")},
	"lightning golem":                  {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem")},
	"stone golem":                      {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem")},
	"animated guardian":                {addToMinion: true, addToMinionTag: mod.SkillName("Animate Guardian")},

	// Other
	"global":                          {tag: mod.Global()},
	"from equipped shield":            {tag: mod.SlotName("Weapon 2")},
	"from equipped gloves and boots":  {tag: mod.SlotName("Gloves", "Boots")},
	"from equipped helmet and gloves": {tag: mod.SlotName("Helmet", "Gloves")},
	"from equipped helmet and boots":  {tag: mod.SlotName("Helmet", "Boots")},
	"from body armour":                {tag: mod.SlotName("Body Armour")},
	"from your body armour":           {tag: mod.SlotName("Body Armour")},
}

// List of modifier flags/tags that appear at the start of a line
var preFlagListCompiled map[string]CompiledList[modNameListType]
var preFlagList = map[string]modNameListType{
	// Weapon types
	`^axe attacks [hd][ae][va][el] `:                           {flags: mod.MFlagAxe},
	`^axe or sword attacks [hd][ae][va][el] `:                  {tag: mod.ModFlagOr(mod.MFlagAxe | mod.MFlagSword)},
	`^bow attacks [hd][ae][va][el] `:                           {flags: mod.MFlagBow},
	`^claw attacks [hd][ae][va][el] `:                          {flags: mod.MFlagClaw},
	`^claw or dagger attacks [hd][ae][va][el] `:                {tag: mod.ModFlagOr(mod.MFlagClaw | mod.MFlagDagger)},
	`^dagger attacks [hd][ae][va][el] `:                        {flags: mod.MFlagDagger},
	`^mace or sceptre attacks [hd][ae][va][el] `:               {flags: mod.MFlagMace},
	`^mace, sceptre or staff attacks [hd][ae][va][el] `:        {tag: mod.ModFlagOr(mod.MFlagMace | mod.MFlagStaff)},
	`^staff attacks [hd][ae][va][el] `:                         {flags: mod.MFlagStaff},
	`^sword attacks [hd][ae][va][el] `:                         {flags: mod.MFlagSword},
	`^wand attacks [hd][ae][va][el] `:                          {flags: mod.MFlagWand},
	`^unarmed attacks [hd][ae][va][el] `:                       {flags: mod.MFlagUnarmed},
	`^attacks with one handed weapons [hd][ae][va][el] `:       {flags: mod.MFlagWeapon1H},
	`^attacks with two handed weapons [hd][ae][va][el] `:       {flags: mod.MFlagWeapon2H},
	`^attacks with melee weapons [hd][ae][va][el] `:            {flags: mod.MFlagWeaponMelee},
	`^attacks with one handed melee weapons [hd][ae][va][el] `: {flags: mod.MFlagWeapon1H | mod.MFlagWeaponMelee},
	`^attacks with two handed melee weapons [hd][ae][va][el] `: {flags: mod.MFlagWeapon2H | mod.MFlagWeaponMelee},
	`^attacks with ranged weapons [hd][ae][va][el] `:           {flags: mod.MFlagWeaponRanged},

	// Damage types
	`^attack damage `:         {flags: mod.MFlagAttack},
	`^hits deal `:             {keywordFlags: mod.KeywordFlagHit},
	`^critical strikes deal `: {tag: mod.Condition("CriticalStrike")},
	`^poisons you inflict with critical strikes have `: {keywordFlags: mod.KeywordFlagPoison | mod.KeywordFlagMatchAll, tag: mod.Condition("CriticalStrike")},

	// Add to minion
	`^minions `:                                                      {addToMinion: true},
	`^minions [hd][ae][va][el] `:                                     {addToMinion: true},
	`^minions leech `:                                                {addToMinion: true},
	`^minions' attacks deal `:                                        {addToMinion: true, flags: mod.MFlagAttack},
	`^golems [hd][ae][va][el] `:                                      {addToMinion: true, addToMinionTag: mod.SkillType(string(data.SkillTypeGolem))},
	`^summoned golems [hd][ae][va][el] `:                             {addToMinion: true, addToMinionTag: mod.SkillType(string(data.SkillTypeGolem))},
	`^golem skills have `:                                            {tag: mod.SkillType(string(data.SkillTypeGolem))},
	`^zombies [hd][ae][va][el] `:                                     {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie")},
	`^raised zombies [hd][ae][va][el] `:                              {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie")},
	`^skeletons [hd][ae][va][el] `:                                   {addToMinion: true, addToMinionTag: mod.SkillName("Summon Skeleton")},
	`^raging spirits [hd][ae][va][el] `:                              {addToMinion: true, addToMinionTag: mod.SkillName("Summon Raging Spirit")},
	`^summoned raging spirits [hd][ae][va][el] `:                     {addToMinion: true, addToMinionTag: mod.SkillName("Summon Raging Spirit")},
	`^spectres [hd][ae][va][el] `:                                    {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre")},
	`^chaos golems [hd][ae][va][el] `:                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem")},
	`^summoned chaos golems [hd][ae][va][el] `:                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem")},
	`^flame golems [hd][ae][va][el] `:                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem")},
	`^summoned flame golems [hd][ae][va][el] `:                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem")},
	`^ice golems [hd][ae][va][el] `:                                  {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem")},
	`^summoned ice golems [hd][ae][va][el] `:                         {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem")},
	`^lightning golems [hd][ae][va][el] `:                            {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem")},
	`^summoned lightning golems [hd][ae][va][el] `:                   {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem")},
	`^stone golems [hd][ae][va][el] `:                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem")},
	`^summoned stone golems [hd][ae][va][el] `:                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem")},
	`^summoned carrion golems [hd][ae][va][el] `:                     {addToMinion: true, addToMinionTag: mod.SkillName("Summon Carrion Golem")},
	`^summoned skitterbots [hd][ae][va][el] `:                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Carrion Golem")},
	`^blink arrow and blink arrow clones [hd][ae][va][el] `:          {addToMinion: true, addToMinionTag: mod.SkillName("Blink Arrow")},
	`^mirror arrow and mirror arrow clones [hd][ae][va][el] `:        {addToMinion: true, addToMinionTag: mod.SkillName("Mirror Arrow")},
	`^animated weapons [hd][ae][va][el] `:                            {addToMinion: true, addToMinionTag: mod.SkillName("Animate Weapon")},
	`^animated guardians? deals? `:                                   {addToMinion: true, addToMinionTag: mod.SkillName("Animate Guardian")},
	`^summoned holy relics [hd][ae][va][el] `:                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Holy Relic")},
	`^summoned reaper [dh][ea][as]l?s? `:                             {addToMinion: true, addToMinionTag: mod.SkillName("Summon Reaper")},
	`^herald skills [hd][ae][va][el] `:                               {tag: mod.SkillType(string(data.SkillTypeHerald))},
	`^agony crawler deals `:                                          {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Agony")},
	`^summoned agony crawler fires `:                                 {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Agony")},
	`^sentinels of purity deal `:                                     {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity")},
	`^summoned sentinels of absolution have `:                        {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity")},
	`^summoned sentinels have `:                                      {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity", "Dominating Blow", "Absolution")},
	`^raised zombies' slam attack has `:                              {addToMinion: true, tag: mod.SkillId("ZombieSlam")},
	`^raised spectres, raised zombies, and summoned skeletons have `: {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre", "Raise Zombie", "Summon Skeleton")},

	// Totem/trap/mine
	`^attacks used by totems have `:            {flags: mod.MFlagAttack, keywordFlags: mod.KeywordFlagTotem},
	`^spells cast by totems [hd][ae][va][el] `: {flags: mod.MFlagSpell, keywordFlags: mod.KeywordFlagTotem},
	`^trap and mine damage `:                   {keywordFlags: mod.KeywordFlagTrap | mod.KeywordFlagMine},
	`^skills used by traps [hd][ae][va][el] `:  {keywordFlags: mod.KeywordFlagTrap},
	`^skills used by mines [hd][ae][va][el] `:  {keywordFlags: mod.KeywordFlagMine},

	// Local damage
	`^attacks with this weapon `:                  {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`^attacks with this weapon [hd][ae][va][el] `: {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`^hits with this weapon [hd][ae][va][el] `:    {flags: mod.MFlagHit, tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},

	// Skill types
	`^attacks [hd][ae][va][el] `:                  {flags: mod.MFlagAttack},
	`^attack skills [hd][ae][va][el] `:            {keywordFlags: mod.KeywordFlagAttack},
	`^spells [hd][ae][va][el] a? ?`:               {flags: mod.MFlagSpell},
	`^spell skills [hd][ae][va][el] `:             {keywordFlags: mod.KeywordFlagSpell},
	`^projectile attack skills [hd][ae][va][el] `: {tag: mod.SkillType(string(data.SkillTypeRangedAttack))},
	`^projectiles from attacks [hd][ae][va][el] `: {tag: mod.SkillType(string(data.SkillTypeRangedAttack))},
	`^arrows [hd][ae][va][el] `:                   {keywordFlags: mod.KeywordFlagBow},
	`^bow skills [hdf][aei][var][el] `:            {keywordFlags: mod.KeywordFlagBow},
	`^projectiles [hdf][aei][var][el] `:           {flags: mod.MFlagProjectile},
	`^melee attacks have `:                        {flags: mod.MFlagMelee},
	`^movement attack skills have `:               {flags: mod.MFlagAttack, keywordFlags: mod.KeywordFlagMovement},
	`^travel skills have `:                        {tag: mod.SkillType(string(data.SkillTypeTravel))},
	`^lightning skills [hd][ae][va][el] a? ?`:     {keywordFlags: mod.KeywordFlagLightning},
	`^lightning spells [hd][ae][va][el] a? ?`:     {keywordFlags: mod.KeywordFlagLightning, flags: mod.MFlagSpell},
	`^cold skills [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagCold},
	`^cold spells [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagCold, flags: mod.MFlagSpell},
	`^fire skills [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagFire},
	`^fire spells [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagFire, flags: mod.MFlagSpell},
	`^chaos skills [hd][ae][va][el] a? ?`:         {keywordFlags: mod.KeywordFlagChaos},
	`^vaal skills [hd][ae][va][el] `:              {keywordFlags: mod.KeywordFlagVaal},
	`^brand skills [hd][ae][va][el] `:             {keywordFlags: mod.KeywordFlagBrand},
	`^channelling skills [hd][ae][va][el] `:       {tag: mod.SkillType(string(data.SkillTypeChannel))},
	`^curse skills [hd][ae][va][el] `:             {keywordFlags: mod.KeywordFlagCurse},
	`^hex skills [hd][ae][va][el] `:               {tag: mod.SkillType(string(data.SkillTypeHex))},
	`^mark skills [hd][ae][va][el] `:              {tag: mod.SkillType(string(data.SkillTypeMark))},
	`^melee skills [hd][ae][va][el] `:             {tag: mod.SkillType(string(data.SkillTypeMelee))},
	`^guard skills [hd][ae][va][el] `:             {tag: mod.SkillType(string(data.SkillTypeGuard))},
	`^nova spells [hd][ae][va][el] `:              {tag: mod.SkillType(string(data.SkillTypeNova))},
	`^area skills [hd][ae][va][el] `:              {tag: mod.SkillType(string(data.SkillTypeArea))},
	`^aura skills [hd][ae][va][el] `:              {tag: mod.SkillType(string(data.SkillTypeAura))},
	`^prismatic skills [hd][ae][va][el] `:         {tag: mod.SkillType(string(data.SkillTypeRandomElement))},
	`^warcry skills have `:                        {tag: mod.SkillType(string(data.SkillTypeWarcry))},
	`^non\-curse aura skills have `:               {tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeAura)), mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)}},
	`^non\-channelling skills have `:              {tag: mod.SkillType(string(data.SkillTypeChannel)).Neg(true)},
	`^non\-vaal skills deal `:                     {tag: mod.SkillType(string(data.SkillTypeVaal)).Neg(true)},
	`^skills [hdfg][aei][vari][eln] `:             {},

	// Slot specific
	`^left ring slot: `:                                   {tag: mod.SlotNumber(1)},
	`^right ring slot: `:                                  {tag: mod.SlotNumber(2)},
	`^socketed gems [hgd][ae][via][enl] `:                 {addToSkill: mod.SocketedIn("{SlotName}")},
	`^socketed skills [hgd][ae][via][enl] `:               {addToSkill: mod.SocketedIn("{SlotName}")},
	`^socketed attacks [hgd][ae][via][enl] `:              {addToSkill: mod.SocketedIn("{SlotName}").Keyword("attack")},
	`^socketed spells [hgd][ae][via][enl] `:               {addToSkill: mod.SocketedIn("{SlotName}").Keyword("spell")},
	`^socketed curse gems [hgd][ae][via][enl] `:           {addToSkill: mod.SocketedIn("{SlotName}").Keyword("curse")},
	`^socketed melee gems [hgd][ae][via][enl] `:           {addToSkill: mod.SocketedIn("{SlotName}").Keyword("melee")},
	`^socketed golem gems [hgd][ae][via][enl] `:           {addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	`^socketed golem skills [hgd][ae][via][enl] `:         {addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	`^socketed golem skills have minions `:                {addToSkill: mod.SocketedIn("{SlotName}").Keyword("golem")},
	`^socketed vaal skills [hgd][ae][via][enl] `:          {addToSkill: mod.SocketedIn("{SlotName}").Keyword("vaal")},
	`^socketed projectile spells [hgdf][aei][viar][enl] `: {addToSkill: mod.SocketedIn("{SlotName}"), tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeProjectile)), mod.SkillType(string(data.SkillTypeSpell))}},

	// Enemy modifiers
	`^enemies withered by you [th]a[vk]e `: {tag: mod.Multiplier("MultiplierThreshold").Base(1), applyToEnemy: true},
	`^enemies (\w+) by you take `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{
				tag:          mod.Condition(utils.Capital(caps[0])),
				applyToEnemy: true,
				modSuffix:    "Taken",
			}
		},
	},
	`^enemies (\w+) by you have `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Condition(utils.Capital(caps[0])), applyToEnemy: true}
		},
	},
	`^hits against enemies (\w+) by you have `: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ActorCondition("enemy", utils.Capital(caps[0]))}
		},
	},
	`^enemies shocked or frozen by you take `:                     {tag: mod.Condition("Shocked", "Frozen"), applyToEnemy: true, modSuffix: "Taken"},
	`^enemies affected by your spider's webs [thd][ae][avk][el] `: {tag: mod.MultiplierThreshold("Spider's WebStack").Threshold(1), applyToEnemy: true},
	`^enemies you curse take `:                                    {tag: mod.Condition("Cursed"), applyToEnemy: true, modSuffix: "Taken"},
	`^enemies you curse ?h?a?v?e? `:                               {tag: mod.Condition("Cursed"), applyToEnemy: true},
	`^nearby enemies take `:                                       {modSuffix: "Taken", applyToEnemy: true},
	`^nearby enemies have `:                                       {applyToEnemy: true},
	`^nearby enemies deal `:                                       {applyToEnemy: true},
	`^nearby enemies `:                                            {applyToEnemy: true},
	`^enemies near your totems deal `:                             {applyToEnemy: true},

	// Other
	`^your flasks grant `:                         {},
	`^when hit, `:                                 {},
	`^you and allies [hgd][ae][via][enl] `:        {},
	`^auras from your skills grant `:              {addToAura: true},
	`^you and nearby allies `:                     {newAura: true},
	`^you and nearby allies [hgd][ae][via][enl] `: {newAura: true},
	`^nearby allies [hgd][ae][via][enl] `:         {newAura: true, newAuraOnlyAllies: true},
	`^you and allies affected by auras from your skills [hgd][ae][via][enl] `: {affectedByAura: true},
	`^take `:      {modSuffix: "Taken"},
	`^marauder: `: {tag: mod.Condition("ConnectedToMarauderStart")},
	`^duelist: `:  {tag: mod.Condition("ConnectedToDuelistStart")},
	`^ranger: `:   {tag: mod.Condition("ConnectedToRangerStart")},
	`^shadow: `:   {tag: mod.Condition("ConnectedToShadowStart")},
	`^witch: `:    {tag: mod.Condition("ConnectedToWitchStart")},
	`^templar: `:  {tag: mod.Condition("ConnectedToTemplarStart")},
	`^scion: `:    {tag: mod.Condition("ConnectedToScionStart")},
	`^skills supported by spellslinger have `:                               {tag: mod.Condition("SupportedBySpellslinger")},
	`^skills that have dealt a critical strike in the past 8 seconds deal `: {tag: mod.Condition("CritInPast8Sec")},
	`^blink arrow and mirror arrow have `:                                   {tag: mod.SkillName("Blink Arrow", "Mirror Arrow")},
	`attacks with energy blades `:                                           {flags: mod.MFlagAttack, tag: mod.Condition("EnergyBladeActive")},

	// While in the presence of...
	`^while a unique enemy is in your presence, `:        {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	`^while a pinnacle atlas boss is in your presence, `: {tag: mod.ActorCondition("enemy", "PinnacleBoss")},
}

// List of modifier tags
var modTagListCompiled map[string]CompiledList[modNameListType]
var modTagList = map[string]modNameListType{
	`on enemies`:                       {},
	`while active`:                     {},
	` on critical strike`:              {tag: mod.Condition("CriticalStrike")},
	`from critical strikes`:            {tag: mod.Condition("CriticalStrike")},
	`with critical strikes`:            {tag: mod.Condition("CriticalStrike")},
	`while affected by auras you cast`: {affectedByAura: true},
	`for you and nearby allies`:        {newAura: true},

	// Multipliers
	`per power charge`:      {tag: mod.Multiplier("PowerCharge").Base(0)},
	`per frenzy charge`:     {tag: mod.Multiplier("FrenzyCharge").Base(0)},
	`per endurance charge`:  {tag: mod.Multiplier("EnduranceCharge").Base(0)},
	`per siphoning charge`:  {tag: mod.Multiplier("SiphoningCharge").Base(0)},
	`per challenger charge`: {tag: mod.Multiplier("ChallengerCharge").Base(0)},
	`per gale force`:        {tag: mod.Multiplier("GaleForce").Base(0)},
	`per intensity`:         {tag: mod.Multiplier("Intensity").Base(0)},
	`per brand`:             {tag: mod.Multiplier("ActiveBrand").Base(0)},
	`per brand, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ActiveBrand").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`per blitz charge`: {tag: mod.Multiplier("BlitzCharge").Base(0)},
	`per ghost shroud`: {tag: mod.Multiplier("GhostShroud").Base(0)},
	`per crab barrier`: {tag: mod.Multiplier("CrabBarrier").Base(0)},
	`per rage`:         {tag: mod.Multiplier("Rage").Base(0)},
	`per (\d+) rage`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("Rage").Base(0).Div(utils.Float(caps[0]))}
		},
	},
	`per level`: {tag: mod.Multiplier("Level").Base(0)},
	`per (\d+) player levels`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("Level").Base(0).Div(utils.Float(caps[0]))}
		},
	},
	`per defiance`:                                         {tag: mod.Multiplier("Defiance").Base(0)},
	`for each equipped normal item`:                        {tag: mod.Multiplier("NormalItem").Base(0)},
	`for each normal item equipped`:                        {tag: mod.Multiplier("NormalItem").Base(0)},
	`for each normal item you have equipped`:               {tag: mod.Multiplier("NormalItem").Base(0)},
	`for each equipped magic item`:                         {tag: mod.Multiplier("MagicItem").Base(0)},
	`for each magic item equipped`:                         {tag: mod.Multiplier("MagicItem").Base(0)},
	`for each magic item you have equipped`:                {tag: mod.Multiplier("MagicItem").Base(0)},
	`for each equipped rare item`:                          {tag: mod.Multiplier("RareItem").Base(0)},
	`for each rare item equipped`:                          {tag: mod.Multiplier("RareItem").Base(0)},
	`for each rare item you have equipped`:                 {tag: mod.Multiplier("RareItem").Base(0)},
	`for each equipped unique item`:                        {tag: mod.Multiplier("UniqueItem").Base(0)},
	`for each unique item equipped`:                        {tag: mod.Multiplier("UniqueItem").Base(0)},
	`for each unique item you have equipped`:               {tag: mod.Multiplier("UniqueItem").Base(0)},
	`per elder item equipped`:                              {tag: mod.Multiplier("ElderItem").Base(0)},
	`per shaper item equipped`:                             {tag: mod.Multiplier("ShaperItem").Base(0)},
	`per elder or shaper item equipped`:                    {tag: mod.Multiplier("ShaperOrElderItem").Base(0)},
	`for each corrupted item equipped`:                     {tag: mod.Multiplier("CorruptedItem").Base(0)},
	`for each equipped corrupted item`:                     {tag: mod.Multiplier("CorruptedItem").Base(0)},
	`for each uncorrupted item equipped`:                   {tag: mod.Multiplier("NonCorruptedItem").Base(0)},
	`per abyssa?l? jewel affecting you`:                    {tag: mod.Multiplier("AbyssJewel").Base(0)},
	`for each herald s?k?i?l?l? ?affecting you`:            {tag: mod.Multiplier("Herald").Base(0)},
	`for each of your aura or herald skills affecting you`: {tag: mod.Multiplier("Herald", "AuraAffectingSelf")},
	`for each type of abyssa?l? jewel affecting you`:       {tag: mod.Multiplier("AbyssJewelType").Base(0)},
	`per (.+) eye jewel affecting you, up to a maximum of \+?(\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{
				tag: mod.Multiplier(utils.Capital(caps[0]) + "EyeJewel").Limit(utils.Float(caps[1])).LimitTotal(true),
			}
		},
	},
	`per sextant affecting the area`: {tag: mod.Multiplier("Sextant").Base(0)},
	`per buff on you`:                {tag: mod.Multiplier("BuffOnSelf").Base(0)},
	`per curse on enemy`:             {tag: mod.Multiplier("CurseOnEnemy").Base(0)},
	`for each curse on enemy`:        {tag: mod.Multiplier("CurseOnEnemy").Base(0)},
	`per curse on you`:               {tag: mod.Multiplier("CurseOnSelf").Base(0)},
	`per poison on you`:              {tag: mod.Multiplier("PoisonStack").Base(0)},
	`for each poison on you`:         {tag: mod.Multiplier("PoisonStack").Base(0)},
	`for each poison on you up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonStack").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`per poison on you, up to (\d+) per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonStack").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each poison you have inflicted recently`: {tag: mod.Multiplier("PoisonAppliedRecently").Base(0)},
	`for each poison you have inflicted recently, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonAppliedRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each shocked enemy you've killed recently`: {tag: mod.Multiplier("ShockedEnemyKilledRecently").Base(0)},
	`per enemy killed recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("EnemyKilledRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`per (\d+) rampage kills`: {
		fn: func(caps []string) modNameListType {
			num := utils.Float(caps[0])
			return modNameListType{tag: mod.Multiplier("Rampage").Div(num).Limit(1000 / num).LimitTotal(true)}
		},
	},
	`per minion, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("SummonedMinion").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
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
	`for each time you've blocked in the past 10 seconds`: {tag: mod.Multiplier("BlockedPast10Sec")},
	`per enemy killed by you or your totems recently`:     {tag: mod.Multiplier("EnemyKilledRecently", "EnemyKilledByTotemsRecently")},
	`per nearby enemy, up to \+?(\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("NearbyEnemies").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`to you and allies`:                    {},
	`per red socket`:                       {tag: mod.Multiplier("RedSocketIn{SlotName}").Base(0)},
	`per green socket on main hand weapon`: {tag: mod.Multiplier("GreenSocketInWeapon 1").Base(0)},
	`per green socket on`:                  {tag: mod.Multiplier("GreenSocketInWeapon 1").Base(0)},
	`per red socket on main hand weapon`:   {tag: mod.Multiplier("RedSocketInWeapon 1").Base(0)},
	`per green socket`:                     {tag: mod.Multiplier("GreenSocketIn{SlotName}").Base(0)},
	`per blue socket`:                      {tag: mod.Multiplier("BlueSocketIn{SlotName}").Base(0)},
	`per white socket`:                     {tag: mod.Multiplier("WhiteSocketIn{SlotName}").Base(0)},
	`for each impale on enemy`:             {tag: mod.Multiplier("ImpaleStacks").Actor("enemy")},
	`per animated weapon`:                  {tag: mod.Multiplier("AnimatedWeapon").Actor("parent")},
	`per grasping vine`:                    {tag: mod.Multiplier("GraspingVinesCount").Base(0)},
	`per fragile regrowth`:                 {tag: mod.Multiplier("FragileRegrowthCount").Base(0)},
	`per allocated mastery passive skill`:  {tag: mod.Multiplier("AllocatedMastery").Base(0)},
	`per allocated notable passive skill`:  {tag: mod.Multiplier("AllocatedNotable").Base(0)},

	// Per stat
	`per (\d+)% of maximum mana they reserve`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ManaReservedPercent")}
		},
	},
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
	`per (\d+) omniscience`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Omni")}
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
	`per (\d+) unreserved maximum mana, up to (\d+)\%`: {
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
	`per (\d+) accuracy rating`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Accuracy")}
		},
	},
	`per (\d+)% block chance`: {
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
	`per (\d+) maximum energy shield on helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnHelmet")}
		},
	},
	`per (\d+) evasion rating on body armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBody Armour")}
		},
	},
	`per (\d+) armour on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnWeapon 2")}
		},
	},
	`per (\d+) armour or evasion rating on shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnWeapon 2", "EvasionOnWeapon 2")}
		},
	},
	`per (\d+) evasion rating on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnWeapon 2")}
		},
	},
	`per (\d+) maximum energy shield on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnWeapon 2")}
		},
	},
	`per (\d+) maximum energy shield on shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnWeapon 2")}
		},
	},
	`per (\d+) evasion on boots`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBoots")}
		},
	},
	`per (\d+) armour on gloves`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnGloves")}
		},
	},
	`per (\d+)% chaos resistance`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ChaosResist")}
		},
	},
	`per (\d+)% cold resistance above 75%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ColdResistOver75")}
		},
	},
	`per (\d+)% lightning resistance above 75%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LightningResistOver75")}
		},
	},
	`per (\d+) devotion`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Devotion")}
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
	`per endurance, frenzy or power charge`: {tag: mod.PerStat(0, "TotalCharges")},
	`per fortification`:                     {tag: mod.PerStat(0, "FortificationStacks")},
	`per totem`:                             {tag: mod.PerStat(0, "TotemsSummoned")},
	`per summoned totem`:                    {tag: mod.PerStat(0, "TotemsSummoned")},
	`for each summoned totem`:               {tag: mod.PerStat(0, "TotemsSummoned")},
	`for each time they have chained`:       {tag: mod.PerStat(0, "Chain")},
	`for each time it has chained`:          {tag: mod.PerStat(0, "Chain")},
	`for each summoned golem`:               {tag: mod.PerStat(0, "ActiveGolemLimit")},
	`for each golem you have summoned`:      {tag: mod.PerStat(0, "ActiveGolemLimit")},
	`per summoned golem`:                    {tag: mod.PerStat(0, "ActiveGolemLimit")},
	`per summoned sentinel of purity`:       {tag: mod.PerStat(0, "ActiveSentinelOfPurityLimit")},
	`per summoned skeleton`:                 {tag: mod.PerStat(0, "ActiveSkeletonLimit")},
	`per skeleton you own`:                  {tag: mod.PerStat(0, "ActiveSkeletonLimit").Actor("parent")},
	`per summoned raging spirit`:            {tag: mod.PerStat(0, "ActiveRagingSpiritLimit")},
	`for each raised zombie`:                {tag: mod.PerStat(0, "ActiveZombieLimit")},
	`per zombie you own`:                    {tag: mod.PerStat(0, "ActiveZombieLimit").Actor("parent")},
	`per raised spectre`:                    {tag: mod.PerStat(0, "ActiveSpectreLimit")},
	`per spectre you own`:                   {tag: mod.PerStat(0, "ActiveSpectreLimit").Actor("parent")},
	`for each remaining chain`:              {tag: mod.PerStat(0, "ChainRemaining")},
	`for each enemy pierced`:                {tag: mod.PerStat(0, "PiercedCount")},

	// Stat conditions
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
	`at least (\d+) intelligence`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Int", utils.Float(caps[0]))}
		},
	},
	`if dexterity is higher than intelligence`: {tag: mod.Condition("DexHigherThanInt")},
	`if strength is higher than intelligence`:  {tag: mod.Condition("StrHigherThanInt")},
	`w?h?i[lf]e? you have at least (\d+) maximum energy shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("EnergyShield", utils.Float(caps[0]))}
		},
	},
	`against targets they pierce`: {tag: mod.StatThreshold("PierceCount", 1)},
	`against pierced targets`:     {tag: mod.StatThreshold("PierceCount", 1)},
	`to targets they pierce`:      {tag: mod.StatThreshold("PierceCount", 1)},
	`w?h?i[lf]e? you have at least (\d+) devotion`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.StatThreshold("Devotion", utils.Float(caps[0]))}
		},
	},
	`while you have at least (\d+) rage`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("Rage").Threshold(utils.Float(caps[0]))}
		},
	},

	// Slot conditions
	`when in main hand`:  {tag: mod.SlotNumber(1)},
	`when in off hand`:   {tag: mod.SlotNumber(2)},
	`in main hand`:       {tag: mod.InSlot(1)},
	`in off hand`:        {tag: mod.InSlot(2)},
	`w?i?t?h? main hand`: {tagList: []mod.Tag{mod.Condition("MainHandAttack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`w?i?t?h? off hand`:  {tagList: []mod.Tag{mod.Condition("OffHandAttack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`[fi]?[rn]?[of]?[ml]?[ i]?[hc]?[it]?[te]?[sd]? ? with this weapon`: {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`if your other ring is a shaper item`:                              {tag: mod.Condition("ShaperItemInRing {OtherSlotNum}")},
	`if your other ring is an elder item`:                              {tag: mod.Condition("ElderItemInRing {OtherSlotNum}")},
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
	`of skills supported by spellslinger`: {tag: mod.Condition("SupportedBySpellslinger")},

	// Equipment conditions
	`while holding a (\w+)`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Condition("Using" + utils.Capital(caps[0]))}
		},
	},
	`while holding a (\w+) or (\w+)`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Condition("Using"+utils.Capital(caps[0]), "Using"+utils.Capital(caps[1]))}
		},
	},
	`while your off hand is empty`:              {tag: mod.Condition("OffHandIsEmpty")},
	`with shields`:                              {tag: mod.Condition("UsingShield")},
	`while dual wielding`:                       {tag: mod.Condition("DualWielding")},
	`while dual wielding claws`:                 {tag: mod.Condition("DualWieldingClaws")},
	`while dual wielding or holding a shield`:   {tag: mod.Condition("DualWielding", "UsingShield")},
	`while wielding an axe`:                     {tag: mod.Condition("UsingAxe")},
	`while wielding an axe or sword`:            {tag: mod.Condition("UsingAxe", "UsingSword")},
	`while wielding a bow`:                      {tag: mod.Condition("UsingBow")},
	`while wielding a claw`:                     {tag: mod.Condition("UsingClaw")},
	`while wielding a dagger`:                   {tag: mod.Condition("UsingDagger")},
	`while wielding a claw or dagger`:           {tag: mod.Condition("UsingClaw", "UsingDagger")},
	`while wielding a mace`:                     {tag: mod.Condition("UsingMace")},
	`while wielding a mace or sceptre`:          {tag: mod.Condition("UsingMace")},
	`while wielding a mace, sceptre or staff`:   {tag: mod.Condition("UsingMace", "UsingStaff")},
	`while wielding a staff`:                    {tag: mod.Condition("UsingStaff")},
	`while wielding a sword`:                    {tag: mod.Condition("UsingSword")},
	`while wielding a melee weapon`:             {tag: mod.Condition("UsingMeleeWeapon")},
	`while wielding a one handed weapon`:        {tag: mod.Condition("UsingOneHandedWeapon")},
	`while wielding a two handed weapon`:        {tag: mod.Condition("UsingTwoHandedWeapon")},
	`while wielding a two handed melee weapon`:  {tagList: []mod.Tag{mod.Condition("UsingTwoHandedWeapon"), mod.Condition("UsingMeleeWeapon")}},
	`while wielding a wand`:                     {tag: mod.Condition("UsingWand")},
	`while wielding two different weapon types`: {tag: mod.Condition("WieldingDifferentWeaponTypes")},
	`while unarmed`:                             {tag: mod.Condition("Unarmed")},
	`while you are unencumbered`:                {tag: mod.Condition("Unencumbered")},
	`equipped bow`:                              {tag: mod.Condition("UsingBow")},
	`with a normal item equipped`:               {tag: mod.MultiplierThreshold("NormalItem").Threshold(1)},
	`with a magic item equipped`:                {tag: mod.MultiplierThreshold("MagicItem").Threshold(1)},
	`with a rare item equipped`:                 {tag: mod.MultiplierThreshold("RareItem").Threshold(1)},
	`with a unique item equipped`:               {tag: mod.MultiplierThreshold("UniqueItem").Threshold(1)},
	`if you wear no corrupted items`:            {tag: mod.MultiplierThreshold("CorruptedItem").Threshold(0).Upper(true)},
	`if no worn items are corrupted`:            {tag: mod.MultiplierThreshold("CorruptedItem").Threshold(0).Upper(true)},
	`if no equipped items are corrupted`:        {tag: mod.MultiplierThreshold("CorruptedItem").Threshold(0).Upper(true)},
	`if all worn items are corrupted`:           {tag: mod.MultiplierThreshold("NonCorruptedItem").Threshold(0).Upper(true)},
	`if all equipped items are corrupted`:       {tag: mod.MultiplierThreshold("NonCorruptedItem").Threshold(0).Upper(true)},
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

	// Player status conditions
	`wh[ie][ln]e? on low life`:                         {tag: mod.Condition("LowLife")},
	`wh[ie][ln]e? not on low life`:                     {tag: mod.Condition("LowLife").Neg(true)},
	`wh[ie][ln]e? on low mana`:                         {tag: mod.Condition("LowMana")},
	`wh[ie][ln]e? not on low mana`:                     {tag: mod.Condition("LowMana").Neg(true)},
	`wh[ie][ln]e? on full life`:                        {tag: mod.Condition("FullLife")},
	`wh[ie][ln]e? not on full life`:                    {tag: mod.Condition("FullLife").Neg(true)},
	`wh[ie][ln]e? no life is reserved`:                 {tag: mod.StatThreshold("LifeReserved", 0).Upper(true)},
	`wh[ie][ln]e? no mana is reserved`:                 {tag: mod.StatThreshold("ManaReserved", 0).Upper(true)},
	`wh[ie][ln]e? on full energy shield`:               {tag: mod.Condition("FullEnergyShield")},
	`wh[ie][ln]e? not on full energy shield`:           {tag: mod.Condition("FullEnergyShield").Neg(true)},
	`wh[ie][ln]e? you have energy shield`:              {tag: mod.Condition("HaveEnergyShield")},
	`wh[ie][ln]e? you have no energy shield`:           {tag: mod.Condition("HaveEnergyShield").Neg(true)},
	`if you have energy shield`:                        {tag: mod.Condition("HaveEnergyShield")},
	`while stationary`:                                 {tag: mod.Condition("Stationary")},
	`while moving`:                                     {tag: mod.Condition("Moving")},
	`while channelling`:                                {tag: mod.Condition("Channelling")},
	`if you've been channelling for at least 1 second`: {tag: mod.Condition("Channelling")},
	`if you've inflicted exposure recently`:            {tag: mod.Condition("AppliedExposureRecently")},
	`while you have no power charges`:                  {tag: mod.StatThreshold("PowerCharges", 0).Upper(true)},
	`while you have no frenzy charges`:                 {tag: mod.StatThreshold("FrenzyCharges", 0).Upper(true)},
	`while you have no endurance charges`:              {tag: mod.StatThreshold("EnduranceCharges", 0).Upper(true)},
	`while you have a power charge`:                    {tag: mod.StatThreshold("PowerCharges", 1)},
	`while you have a frenzy charge`:                   {tag: mod.StatThreshold("FrenzyCharges", 1)},
	`while you have an endurance charge`:               {tag: mod.StatThreshold("EnduranceCharges", 1)},
	`while at maximum power charges`:                   {tag: mod.StatThreshold("PowerCharges", 0).ThresholdStat("PowerChargesMax")},
	`while at maximum frenzy charges`:                  {tag: mod.StatThreshold("FrenzyCharges", 0).ThresholdStat("FrenzyChargesMax")},
	`while on full frenzy charges`:                     {tag: mod.StatThreshold("FrenzyCharges", 0).ThresholdStat("FrenzyChargesMax")},
	`while at maximum endurance charges`:               {tag: mod.StatThreshold("EnduranceCharges", 0).ThresholdStat("EnduranceChargesMax")},
	`while at maximum fortification`:                   {tag: mod.Condition("HaveMaximumFortification")},
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
	`while you have a totem`:                      {tag: mod.Condition("HaveTotem")},
	`while you have at least one nearby ally`:     {tag: mod.MultiplierThreshold("NearbyAlly").Threshold(1)},
	`while you have fortify`:                      {tag: mod.Condition("Fortified")},
	`while you have phasing`:                      {tag: mod.Condition("Phasing")},
	`while you have elusive`:                      {tag: mod.Condition("Elusive")},
	`while physical aegis is depleted`:            {tag: mod.Condition("PhysicalAegisDepleted")},
	`during onslaught`:                            {tag: mod.Condition("Onslaught")},
	`while you have onslaught`:                    {tag: mod.Condition("Onslaught")},
	`while phasing`:                               {tag: mod.Condition("Phasing")},
	`while you have tailwind`:                     {tag: mod.Condition("Tailwind")},
	`while elusive`:                               {tag: mod.Condition("Elusive")},
	`gain elusive`:                                {tag: mod.Condition("CanBeElusive", "Elusive")},
	`while you have arcane surge`:                 {tag: mod.Condition("AffectedByArcaneSurge")},
	`while you have cat's stealth`:                {tag: mod.Condition("AffectedByCat'sStealth")},
	`while you have cat's agility`:                {tag: mod.Condition("AffectedByCat'sAgility")},
	`while you have avian's might`:                {tag: mod.Condition("AffectedByAvian'sMight")},
	`while you have avian's flight`:               {tag: mod.Condition("AffectedByAvian'sFlight")},
	`while affected by aspect of the cat`:         {tag: mod.Condition("AffectedByCat'sStealth", "AffectedByCat'sAgility")},
	`while affected by a non\-vaal guard skill`:   {tag: mod.Condition("AffectedByNonVaalGuardSkill")},
	`if a non\-vaal guard buff was lost recently`: {tag: mod.Condition("LostNonVaalBuffRecently")},
	`while affected by a guard skill buff`:        {tag: mod.Condition("AffectedByGuardSkill")},
	`while affected by a herald`:                  {tag: mod.Condition("AffectedByHerald")},
	`while fortified`:                             {tag: mod.Condition("Fortified")},
	`while in blood stance`:                       {tag: mod.Condition("BloodStance")},
	`while in sand stance`:                        {tag: mod.Condition("SandStance")},
	`while you have a bestial minion`:             {tag: mod.Condition("HaveBestialMinion")},
	`while you have infusion`:                     {tag: mod.Condition("InfusionActive")},
	`while focus?sed`:                             {tag: mod.Condition("Focused")},
	`while leeching`:                              {tag: mod.Condition("Leeching")},
	`while leeching energy shield`:                {tag: mod.Condition("LeechingEnergyShield")},
	`while using a flask`:                         {tag: mod.Condition("UsingFlask")},
	`during effect`:                               {tag: mod.Condition("UsingFlask")},
	`during flask effect`:                         {tag: mod.Condition("UsingFlask")},
	`during any flask effect`:                     {tag: mod.Condition("UsingFlask")},
	`while under no flask effects`:                {tag: mod.Condition("UsingFlask").Neg(true)},
	`during effect of any mana flask`:             {tag: mod.Condition("UsingManaFlask")},
	`during effect of any life flask`:             {tag: mod.Condition("UsingLifeFlask")},
	`during effect of any life or mana flask`:     {tag: mod.Condition("UsingManaFlask", "UsingLifeFlask")},
	`while on consecrated ground`:                 {tag: mod.Condition("OnConsecratedGround")},
	`on burning ground`:                           {tag: mod.Condition("OnBurningGround")},
	`while on burning ground`:                     {tag: mod.Condition("OnBurningGround")},
	`on chilled ground`:                           {tag: mod.Condition("OnChilledGround")},
	`on shocked ground`:                           {tag: mod.Condition("OnShockedGround")},
	`while in a caustic cloud`:                    {tag: mod.Condition("OnCausticCloud")},
	`while blinded`:                               {tag: mod.Condition("Blinded")},
	`while burning`:                               {tag: mod.Condition("Burning")},
	`while ignited`:                               {tag: mod.Condition("Ignited")},
	`while you are ignited`:                       {tag: mod.Condition("Ignited")},
	`while chilled`:                               {tag: mod.Condition("Chilled")},
	`while you are chilled`:                       {tag: mod.Condition("Chilled")},
	`while frozen`:                                {tag: mod.Condition("Frozen")},
	`while shocked`:                               {tag: mod.Condition("Shocked")},
	`while you are shocked`:                       {tag: mod.Condition("Shocked")},
	`while you are bleeding`:                      {tag: mod.Condition("Bleeding")},
	`while not ignited, frozen or shocked`:        {tag: mod.Condition("Ignited", "Frozen", "Shocked").Neg(true)},
	`while bleeding`:                              {tag: mod.Condition("Bleeding")},
	`while poisoned`:                              {tag: mod.Condition("Poisoned")},
	`while you are poisoned`:                      {tag: mod.Condition("Poisoned")},
	`while cursed`:                                {tag: mod.Condition("Cursed")},
	`while not cursed`:                            {tag: mod.Condition("Cursed").Neg(true)},
	`against damage over time`:                    {tag: mod.Condition("AgainstDamageOverTime")},
	`while there is only one nearby enemy`:        {tagList: []mod.Tag{mod.Multiplier("NearbyEnemies").Limit(1), mod.Condition("OnlyOneNearbyEnemy")}},
	`while t?h?e?r?e? ?i?s? ?a rare or unique enemy i?s? ?nearby`:                      {tag: mod.ActorCondition("enemy", "NearbyRareOrUniqueEnemy", "RareOrUnique")},
	`if you[' ]h?a?ve hit recently`:                                                    {tag: mod.Condition("HitRecently")},
	`if you[' ]h?a?ve hit an enemy recently`:                                           {tag: mod.Condition("HitRecently")},
	`if you[' ]h?a?ve hit with your main hand weapon recently`:                         {tag: mod.Condition("HitRecentlyWithWeapon")},
	`if you[' ]h?a?ve hit with your off hand weapon recently`:                          {tagList: []mod.Tag{mod.Condition("HitRecentlyWithWeapon"), mod.Condition("DualWielding")}},
	`if you[' ]h?a?ve hit a cursed enemy recently`:                                     {tagList: []mod.Tag{mod.Condition("HitRecently"), mod.ActorCondition("enemy", "Cursed")}},
	`if you[' ]h?a?ve crit recently`:                                                   {tag: mod.Condition("CritRecently")},
	`if you[' ]h?a?ve dealt a critical strike recently`:                                {tag: mod.Condition("CritRecently")},
	`if you[' ]h?a?ve dealt a critical strike with this weapon recently`:               {tag: mod.Condition("CritRecently")}, // Replica Kongor's
	`if you[' ]h?a?ve crit in the past 8 seconds`:                                      {tag: mod.Condition("CritInPast8Sec")},
	`if you[' ]h?a?ve dealt a crit in the past 8 seconds`:                              {tag: mod.Condition("CritInPast8Sec")},
	`if you[' ]h?a?ve dealt a critical strike in the past 8 seconds`:                   {tag: mod.Condition("CritInPast8Sec")},
	`if you haven't crit recently`:                                                     {tag: mod.Condition("CritRecently").Neg(true)},
	`if you haven't dealt a critical strike recently`:                                  {tag: mod.Condition("CritRecently").Neg(true)},
	`if you[' ]h?a?ve dealt a non\-critical strike recently`:                           {tag: mod.Condition("NonCritRecently")},
	`if your skills have dealt a critical strike recently`:                             {tag: mod.Condition("SkillCritRecently")},
	`if you dealt a critical strike with a herald skill recently`:                      {tag: mod.Condition("CritWithHeraldSkillRecently")},
	`if you[' ]h?a?ve dealt a critical strike with a two handed melee weapon recently`: {flags: mod.MFlagWeapon2H | mod.MFlagWeaponMelee, tag: mod.Condition("CritRecently")},
	`if you[' ]h?a?ve killed recently`:                                                 {tag: mod.Condition("KilledRecently")},
	`if you[' ]h?a?ve killed an enemy recently`:                                        {tag: mod.Condition("KilledRecently")},
	`if you[' ]h?a?ve killed at least (\d) enemies recently`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("EnemyKilledRecently").Threshold(utils.Float(caps[0]))}
		},
	},
	`if you haven't killed recently`:                                              {tag: mod.Condition("KilledRecently").Neg(true)},
	`if you or your totems have killed recently`:                                  {tag: mod.Condition("KilledRecently", "TotemsKilledRecently")},
	`if you[' ]h?a?ve thrown a trap or mine recently`:                             {tag: mod.Condition("TrapOrMineThrownRecently")},
	`if you[' ]h?a?ve killed a maimed enemy recently`:                             {tagList: []mod.Tag{mod.Condition("KilledRecently"), mod.ActorCondition("enemy", "Maimed")}},
	`if you[' ]h?a?ve killed a cursed enemy recently`:                             {tagList: []mod.Tag{mod.Condition("KilledRecently"), mod.ActorCondition("enemy", "Cursed")}},
	`if you[' ]h?a?ve killed a bleeding enemy recently`:                           {tagList: []mod.Tag{mod.Condition("KilledRecently"), mod.ActorCondition("enemy", "Bleeding")}},
	`if you[' ]h?a?ve killed an enemy affected by your damage over time recently`: {tag: mod.Condition("KilledAffectedByDotRecently")},
	`if you[' ]h?a?ve frozen an enemy recently`:                                   {tag: mod.Condition("FrozenEnemyRecently")},
	`if you[' ]h?a?ve chilled an enemy recently`:                                  {tag: mod.Condition("ChilledEnemyRecently")},
	`if you[' ]h?a?ve ignited an enemy recently`:                                  {tag: mod.Condition("IgnitedEnemyRecently")},
	`if you[' ]h?a?ve shocked an enemy recently`:                                  {tag: mod.Condition("ShockedEnemyRecently")},
	`if you[' ]h?a?ve stunned an enemy recently`:                                  {tag: mod.Condition("StunnedEnemyRecently")},
	`if you[' ]h?a?ve stunned an enemy with a two handed melee weapon recently`:   {flags: mod.MFlagWeapon2H | mod.MFlagWeaponMelee, tag: mod.Condition("StunnedEnemyRecently")},
	`if you[' ]h?a?ve been hit recently`:                                          {tag: mod.Condition("BeenHitRecently")},
	`if you[' ]h?a?ve been hit by an attack recently`:                             {tag: mod.Condition("BeenHitByAttackRecently")},
	`if you were hit recently`:                                                    {tag: mod.Condition("BeenHitRecently")},
	`if you were damaged by a hit recently`:                                       {tag: mod.Condition("BeenHitRecently")},
	`if you[' ]h?a?ve taken a critical strike recently`:                           {tag: mod.Condition("BeenCritRecently")},
	`if you[' ]h?a?ve taken a savage hit recently`:                                {tag: mod.Condition("BeenSavageHitRecently")},
	`if you have ?n[o']t been hit recently`:                                       {tag: mod.Condition("BeenHitRecently").Neg(true)},
	`if you have ?n[o']t been hit by an attack recently`:                          {tag: mod.Condition("BeenHitByAttackRecently").Neg(true)},
	`if you[' ]h?a?ve taken no damage from hits recently`:                         {tag: mod.Condition("BeenHitRecently").Neg(true)},
	`if you[' ]h?a?ve taken fire damage from a hit recently`:                      {tag: mod.Condition("HitByFireDamageRecently")},
	`if you[' ]h?a?ve taken fire damage from an enemy hit recently`:               {tag: mod.Condition("TakenFireDamageFromEnemyHitRecently")},
	`if you[' ]h?a?ve taken spell damage recently`:                                {tag: mod.Condition("HitBySpellDamageRecently")},
	`if you haven't taken damage recently`:                                        {tag: mod.Condition("BeenHitRecently").Neg(true)},
	`if you[' ]h?a?ve blocked recently`:                                           {tag: mod.Condition("BlockedRecently")},
	`if you haven't blocked recently`:                                             {tag: mod.Condition("BlockedRecently").Neg(true)},
	`if you[' ]h?a?ve blocked an attack recently`:                                 {tag: mod.Condition("BlockedAttackRecently")},
	`if you[' ]h?a?ve blocked attack damage recently`:                             {tag: mod.Condition("BlockedAttackRecently")},
	`if you[' ]h?a?ve blocked a spell recently`:                                   {tag: mod.Condition("BlockedSpellRecently")},
	`if you[' ]h?a?ve blocked spell damage recently`:                              {tag: mod.Condition("BlockedSpellRecently")},
	`if you[' ]h?a?ve blocked damage from a unique enemy in the past 10 seconds`:  {tag: mod.Condition("BlockedHitFromUniqueEnemyInPast10Sec")},
	`if you[' ]h?a?ve attacked recently`:                                          {tag: mod.Condition("AttackedRecently")},
	`if you[' ]h?a?ve cast a spell recently`:                                      {tag: mod.Condition("CastSpellRecently")},
	`if you[' ]h?a?ve consumed a corpse recently`:                                 {tag: mod.Condition("ConsumedCorpseRecently")},
	`if you[' ]h?a?ve cursed an enemy recently`:                                   {tag: mod.Condition("CursedEnemyRecently")},
	`if you[' ]h?a?ve cast a mark spell recently`:                                 {tag: mod.Condition("CastMarkRecently")},
	`if you have ?n[o']t consumed a corpse recently`:                              {tag: mod.Condition("ConsumedCorpseRecently").Neg(true)},
	`for each corpse consumed recently`:                                           {tag: mod.Multiplier("CorpseConsumedRecently").Base(0)},
	`if you[' ]h?a?ve taunted an enemy recently`:                                  {tag: mod.Condition("TauntedEnemyRecently")},
	`if you[' ]h?a?ve used a skill recently`:                                      {tag: mod.Condition("UsedSkillRecently")},
	`if you[' ]h?a?ve used a travel skill recently`:                               {tag: mod.Condition("UsedTravelSkillRecently")},
	`for each skill you've used recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("SkillUsedRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`if you[' ]h?a?ve used a warcry recently`:         {tag: mod.Condition("UsedWarcryRecently")},
	`if you[' ]h?a?ve warcried recently`:              {tag: mod.Condition("UsedWarcryRecently")},
	`for each time you[' ]h?a?ve warcried recently`:   {tag: mod.Multiplier("WarcryUsedRecently").Base(0)},
	`if you[' ]h?a?ve warcried in the past 8 seconds`: {tag: mod.Condition("UsedWarcryInPast8Seconds")},
	`for each of your mines detonated recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("MineDetonatedRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each mine detonated recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("MineDetonatedRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each mine detonated recently, up to (\d+)% per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("MineDetonatedRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each of your traps triggered recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("TrapTriggeredRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each trap triggered recently, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("TrapTriggeredRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each trap triggered recently, up to (\d+)% per second`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("TrapTriggeredRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`if you[' ]h?a?ve used a fire skill recently`:                    {tag: mod.Condition("UsedFireSkillRecently")},
	`if you[' ]h?a?ve used a cold skill recently`:                    {tag: mod.Condition("UsedColdSkillRecently")},
	`if you[' ]h?a?ve used a fire skill in the past 10 seconds`:      {tag: mod.Condition("UsedFireSkillInPast10Sec")},
	`if you[' ]h?a?ve used a cold skill in the past 10 seconds`:      {tag: mod.Condition("UsedColdSkillInPast10Sec")},
	`if you[' ]h?a?ve used a lightning skill in the past 10 seconds`: {tag: mod.Condition("UsedLightningSkillInPast10Sec")},
	`if you[' ]h?a?ve summoned a totem recently`:                     {tag: mod.Condition("SummonedTotemRecently")},
	`if you summoned a golem in the past 8 seconds`:                  {tag: mod.Condition("SummonedGolemInPast8Sec")},
	`if you haven't summoned a totem in the past 2 seconds`:          {tag: mod.Condition("NoSummonedTotemsInPastTwoSeconds")},
	`if you[' ]h?a?ve used a minion skill recently`:                  {tag: mod.Condition("UsedMinionSkillRecently")},
	`if you[' ]h?a?ve used a movement skill recently`:                {tag: mod.Condition("UsedMovementSkillRecently")},
	`if you haven't cast dash recently`:                              {tag: mod.Condition("CastDashRecently").Neg(true)},
	`if you[' ]h?a?ve cast dash recently`:                            {tag: mod.Condition("CastDashRecently")},
	`if you[' ]h?a?ve used a vaal skill recently`:                    {tag: mod.Condition("UsedVaalSkillRecently")},
	`if you haven't used a brand skill recently`:                     {tag: mod.Condition("UsedBrandRecently").Neg(true)},
	`if you[' ]h?a?ve used a brand skill recently`:                   {tag: mod.Condition("UsedBrandRecently")},
	`if you[' ]h?a?ve spent (\d+) total mana recently`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("ManaSpentRecently").Threshold(utils.Float(caps[0]))}
		},
	},
	`if you[' ]h?a?ve spent life recently`: {tag: mod.MultiplierThreshold("LifeSpentRecently").Threshold(1)},
	`for 4 seconds after spending a total of (\d+) mana`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("ManaSpentRecently").Threshold(utils.Float(caps[0]))}
		},
	},
	`if you've impaled an enemy recently`:                {tag: mod.Condition("ImpaledRecently")},
	`if you've changed stance recently`:                  {tag: mod.Condition("ChangedStanceRecently")},
	`if you've gained a power charge recently`:           {tag: mod.Condition("GainedPowerChargeRecently")},
	`if you haven't gained a power charge recently`:      {tag: mod.Condition("GainedPowerChargeRecently").Neg(true)},
	`if you haven't gained a frenzy charge recently`:     {tag: mod.Condition("GainedFrenzyChargeRecently").Neg(true)},
	`if you've stopped taking damage over time recently`: {tag: mod.Condition("StoppedTakingDamageOverTimeRecently")},
	`during soul gain prevention`:                        {tag: mod.Condition("SoulGainPrevention")},
	`if you detonated mines recently`:                    {tag: mod.Condition("DetonatedMinesRecently")},
	`if you detonated a mine recently`:                   {tag: mod.Condition("DetonatedMinesRecently")},
	`if you[' ]h?a?ve detonated a mine recently`:         {tag: mod.Condition("DetonatedMinesRecently")},
	`if energy shield recharge has started recently`:     {tag: mod.Condition("EnergyShieldRechargeRecently")},
	`when cast on frostbolt`:                             {tag: mod.Condition("CastOnFrostbolt")},
	`branded enemy's`:                                    {tag: mod.MultiplierThreshold("BrandsAttachedToEnemy").Threshold(1)},
	`to enemies they're attached to`:                     {tag: mod.MultiplierThreshold("BrandsAttachedToEnemy").Threshold(1)},
	`for each hit you've taken recently up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("BeenHitRecently").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each nearby enemy, up to (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("NearbyEnemies").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`while you have iron reflexes`:             {tag: mod.Condition("HaveIronReflexes")},
	`while you do not have iron reflexes`:      {tag: mod.Condition("HaveIronReflexes").Neg(true)},
	`while you have elemental overload`:        {tag: mod.Condition("HaveElementalOverload")},
	`while you do not have elemental overload`: {tag: mod.Condition("HaveElementalOverload").Neg(true)},
	`while you have resolute technique`:        {tag: mod.Condition("HaveResoluteTechnique")},
	`while you do not have resolute technique`: {tag: mod.Condition("HaveResoluteTechnique").Neg(true)},
	`while you have avatar of fire`:            {tag: mod.Condition("HaveAvatarOfFire")},
	`while you do not have avatar of fire`:     {tag: mod.Condition("HaveAvatarOfFire").Neg(true)},
	`if you have a summoned golem`:             {tag: mod.Condition("HavePhysicalGolem", "HaveLightningGolem", "HaveColdGolem", "HaveFireGolem", "HaveChaosGolem", "HaveCarrionGolem")},
	`while you have a summoned golem`:          {tag: mod.Condition("HavePhysicalGolem", "HaveLightningGolem", "HaveColdGolem", "HaveFireGolem", "HaveChaosGolem", "HaveCarrionGolem")},
	`if a minion has died recently`:            {tag: mod.Condition("MinionsDiedRecently")},
	`if a minion has been killed recently`:     {tag: mod.Condition("MinionsDiedRecently")},
	`while you have sacrificial zeal`:          {tag: mod.Condition("SacrificialZeal")},
	`while sane`:                               {tag: mod.Condition("Insane").Neg(true)},
	`while insane`:                             {tag: mod.Condition("Insane")},
	`while you have defiance`:                  {tag: mod.MultiplierThreshold("Defiance").Threshold(1)},
	`while affected by glorious madness`:       {tag: mod.Condition("AffectedByGloriousMadness")},

	// Enemy status conditions
	`at close range`:                        {tag: mod.Condition("AtCloseRange")},
	`against rare and unique enemies`:       {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	`against unique enemies`:                {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	`against enemies on full life`:          {tag: mod.ActorCondition("enemy", "FullLife")},
	`against enemies that are on full life`: {tag: mod.ActorCondition("enemy", "FullLife")},
	`against enemies on low life`:           {tag: mod.ActorCondition("enemy", "LowLife")},
	`against enemies that are on low life`:  {tag: mod.ActorCondition("enemy", "LowLife")},
	`against cursed enemies`:                {tag: mod.ActorCondition("enemy", "Cursed")},
	`of cursed enemies'`:                    {tag: mod.ActorCondition("enemy", "Cursed")},
	`when hitting cursed enemies`:           {tag: mod.ActorCondition("enemy", "Cursed"), keywordFlags: mod.KeywordFlagHit},
	`from cursed enemies`:                   {tag: mod.ActorCondition("enemy", "Cursed")},
	`against marked enemy`:                  {tag: mod.ActorCondition("enemy", "Marked")},
	`when hitting marked enemy`:             {tag: mod.ActorCondition("enemy", "Marked"), keywordFlags: mod.KeywordFlagHit},
	`from marked enemy`:                     {tag: mod.ActorCondition("enemy", "Marked")},
	`against taunted enemies`:               {tag: mod.ActorCondition("enemy", "Taunted")},
	`against bleeding enemies`:              {tag: mod.ActorCondition("enemy", "Bleeding")},
	`you inflict on bleeding enemies`:       {tag: mod.ActorCondition("enemy", "Bleeding")},
	`to bleeding enemies`:                   {tag: mod.ActorCondition("enemy", "Bleeding")},
	`from bleeding enemies`:                 {tag: mod.ActorCondition("enemy", "Bleeding")},
	`against poisoned enemies`:              {tag: mod.ActorCondition("enemy", "Poisoned")},
	`you inflict on poisoned enemies`:       {tag: mod.ActorCondition("enemy", "Poisoned")},
	`to poisoned enemies`:                   {tag: mod.ActorCondition("enemy", "Poisoned")},
	`against enemies affected by (\d+) or more poisons`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("PoisonStack").Threshold(utils.Float(caps[0])).Actor("enemy")}
		},
	},
	`against enemies affected by at least (\d+) poisons`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("PoisonStack").Threshold(utils.Float(caps[0])).Actor("enemy")}
		},
	},
	`against hindered enemies`:                                   {tag: mod.ActorCondition("enemy", "Hindered")},
	`against maimed enemies`:                                     {tag: mod.ActorCondition("enemy", "Maimed")},
	`you inflict on maimed enemies`:                              {tag: mod.ActorCondition("enemy", "Maimed")},
	`against blinded enemies`:                                    {tag: mod.ActorCondition("enemy", "Blinded")},
	`from blinded enemies`:                                       {tag: mod.ActorCondition("enemy", "Blinded")},
	`against burning enemies`:                                    {tag: mod.ActorCondition("enemy", "Burning")},
	`against ignited enemies`:                                    {tag: mod.ActorCondition("enemy", "Ignited")},
	`to ignited enemies`:                                         {tag: mod.ActorCondition("enemy", "Ignited")},
	`against shocked enemies`:                                    {tag: mod.ActorCondition("enemy", "Shocked")},
	`to shocked enemies`:                                         {tag: mod.ActorCondition("enemy", "Shocked")},
	`against frozen enemies`:                                     {tag: mod.ActorCondition("enemy", "Frozen")},
	`to frozen enemies`:                                          {tag: mod.ActorCondition("enemy", "Frozen")},
	`against chilled enemies`:                                    {tag: mod.ActorCondition("enemy", "Chilled")},
	`to chilled enemies`:                                         {tag: mod.ActorCondition("enemy", "Chilled")},
	`inflicted on chilled enemies`:                               {tag: mod.ActorCondition("enemy", "Chilled")},
	`enemies which are chilled`:                                  {tag: mod.ActorCondition("enemy", "Chilled")},
	`against chilled or frozen enemies`:                          {tag: mod.ActorCondition("enemy", "Chilled", "Frozen")},
	`against frozen, shocked or ignited enemies`:                 {tag: mod.ActorCondition("enemy", "Frozen", "Shocked", "Ignited")},
	`against enemies affected by elemental ailments`:             {tag: mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped")},
	`against enemies affected by ailments`:                       {tag: mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped", "Poisoned", "Bleeding")},
	`against enemies that are affected by elemental ailments`:    {tag: mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped")},
	`against enemies that are affected by no elemental ailments`: {tagList: []mod.Tag{mod.ActorCondition("enemy", "Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped").Neg(true), mod.Condition("Effective")}},
	`against enemies affected by (\d+) spider's webs`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("Spider's WebStack").Actor("enemy").Threshold(utils.Float(caps[0]))}
		},
	},
	`against enemies on consecrated ground`: {tag: mod.ActorCondition("enemy", "OnConsecratedGround")},

	// Enemy multipliers
	`per freeze, shock [ao][nr]d? ignite on enemy`: {tag: mod.Multiplier("FreezeShockIgniteOnEnemy").Base(0)},
	`per poison affecting enemy`:                   {tag: mod.Multiplier("PoisonStack").Actor("enemy")},
	`per poison affecting enemy, up to \+([\d\.]+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("PoisonStack").Actor("enemy").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`for each spider's web on the enemy`: {tag: mod.Multiplier("Spider's WebStack").Actor("enemy")},
}

func grantedExtraSkill(name string, level int, noSupports bool) []mod.Mod {
	name, _ = strings.CutSuffix(name, " skill")
	return []mod.Mod{
		MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillName: name, Level: level, NoSupports: noSupports, Triggered: true}),
	}
}

func triggerExtraSkill(name string, level int, noSupports bool, sourceSkill string) []mod.Mod {
	name, _ = strings.CutSuffix(name, " skill")
	sourceSkill, _ = strings.CutSuffix(sourceSkill, " skill")
	skill := mod.ExtraSkill{SkillName: name, Level: level, NoSupports: noSupports, Triggered: true}
	if sourceSkill != "" {
		skill.Source = sourceSkill
	}
	return []mod.Mod{
		MOD("ExtraSkill", "LIST", skill),
	}
}

// Keep as example
type SpecialFuncType func(num float64, captures []string) ([]mod.Mod, string)

// List of special modifiers
var specialModListCompiled map[string]CompiledList[interface{}]
var specialModList = map[string]interface{}{
	// Keystones
	`(\d+)% less damage taken for every (\d+)% life recovery per second from leech`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeMore, -num).Tag(mod.PerStat(utils.Float(captures[1]), "MaxLifeLeechRatePercent"))}, ""
	},
	`modifiers to chance to suppress spell damage instead apply to chance to dodge spell hits at 50% of their value`: []mod.Mod{
		mod.NewFlag("ConvertSpellSuppressionToSpellDodge", true),
		mod.NewFloat("SpellSuppressionChance", mod.TypeOverride, 0).Source("Acrobatics"),
	},
	`maximum chance to dodge spell hits is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SpellDodgeChanceMax", mod.TypeOverride, num).Source("Acrobatics")}, ""
	},
	`dexterity provides no inherent bonus to evasion rating`:      []mod.Mod{mod.NewFlag("NoDexBonusToEvasion", true)},
	`strength's damage bonus applies to all spell damage as well`: []mod.Mod{mod.NewFlag("IronWill", true)},
	`your hits can't be evaded`:                                   []mod.Mod{mod.NewFlag("CannotBeEvaded", true)},
	`never deal critical strikes`:                                 []mod.Mod{mod.NewFlag("NeverCrit", true), mod.NewFlag("Condition:NeverCrit", true)},
	`no critical strike multiplier`:                               []mod.Mod{mod.NewFlag("NoCritMultiplier", true)},
	`ailments never count as being from critical strikes`:         []mod.Mod{mod.NewFlag("AilmentsAreNeverFromCrit", true)},
	`the increase to physical damage from strength applies to projectile attacks as well as melee attacks`: []mod.Mod{mod.NewFlag("IronGrip", true)},
	`strength's damage bonus applies to projectile attack damage as well as melee damage`:                  []mod.Mod{mod.NewFlag("IronGrip", true)},
	`converts all evasion rating to armour\. dexterity provides no bonus to evasion rating`:                []mod.Mod{mod.NewFlag("NoDexBonusToEvasion", true), mod.NewFlag("IronReflexes", true)},
	`30% chance to dodge attack hits\. 50% less armour, 30% less energy shield, 30% less chance to block spell and attack damage`: []mod.Mod{
		mod.NewFloat("AttackDodgeChance", mod.TypeBase, 30),
		mod.NewFloat("Armour", mod.TypeMore, -50),
		mod.NewFloat("EnergyShield", mod.TypeMore, -30),
		mod.NewFloat("BlockChance", mod.TypeMore, -30),
		mod.NewFloat("SpellBlockChance", mod.TypeMore, -30),
	},
	`(\d+)% increased blind effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewList("EnemyModifier", mod.NewFloat("BlindEffect", mod.TypeIncrease, num))}, ""
	},
	`\+(\d+)% chance to block spell damage for each (\d+)% overcapped chance to block attack damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SpellBlockChance", mod.TypeBase, num).Tag(mod.PerStat(utils.Float(captures[1]), "BlockChanceOverCap"))}, ""
	},
	`maximum life becomes 1, immune to chaos damage`: []mod.Mod{
		mod.NewFlag("ChaosInoculation", true),
		mod.NewFloat("ChaosDamageTaken", mod.TypeMore, -100),
	},
	`life regeneration is applied to energy shield instead`:                 []mod.Mod{mod.NewFlag("ZealotsOath", true)},
	`life leeched per second is doubled`:                                    []mod.Mod{mod.NewFloat("LifeLeechRate", mod.TypeMore, 100)},
	`total recovery per second from life leech is doubled`:                  []mod.Mod{mod.NewFloat("LifeLeechRate", mod.TypeMore, 100)},
	`maximum total recovery per second from life leech is doubled`:          []mod.Mod{mod.NewFloat("MaxLifeLeechRate", mod.TypeMore, 100)},
	`maximum total life recovery per second from leech is doubled`:          []mod.Mod{mod.NewFloat("MaxLifeLeechRate", mod.TypeMore, 100)},
	`maximum total recovery per second from energy shield leech is doubled`: []mod.Mod{mod.NewFloat("MaxEnergyShieldLeechRate", mod.TypeMore, 100)},
	`maximum total energy shield recovery per second from leech is doubled`: []mod.Mod{mod.NewFloat("MaxEnergyShieldLeechRate", mod.TypeMore, 100)},
	`life regeneration has no effect`:                                       []mod.Mod{mod.NewFlag("NoLifeRegen", true)},
	`energy shield recharge instead applies to life`:                        []mod.Mod{mod.NewFlag("EnergyShieldRechargeAppliesToLife", true)},
	`deal no non\-fire damage`:                                              []mod.Mod{mod.NewFlag("DealNoPhysical", true), mod.NewFlag("DealNoLightning", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoChaos", true)},
	`(\d+)% of physical, cold and lightning damage converted to fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("PhysicalDamageConvertToFire", mod.TypeBase, num),
			mod.NewFloat("LightningDamageConvertToFire", mod.TypeBase, num),
			mod.NewFloat("ColdDamageConvertToFire", mod.TypeBase, num),
		}, ""
	},
	`removes all mana\. spend life instead of mana for skills`: []mod.Mod{
		mod.NewFloat("Mana", mod.TypeMore, -100),
		mod.NewFlag("BloodMagic", true),
	},
	`removes all mana`:                                 []mod.Mod{mod.NewFloat("Mana", mod.TypeMore, -100)},
	`removes all energy shield`:                        []mod.Mod{mod.NewFloat("EnergyShield", mod.TypeMore, -100)},
	`skills cost life instead of mana`:                 []mod.Mod{mod.NewFlag("CostLifeInsteadOfMana", true)},
	`skills reserve life instead of mana`:              []mod.Mod{mod.NewFlag("BloodMagicReserved", true)},
	`spend life instead of mana for effects of skills`: []mod.Mod{},
	`skills cost \+(\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("RageCostBase", mod.TypeBase, num)}, ""
	},
	`hits that deal elemental damage remove exposure to those elements and inflict exposure to other elements exposure inflicted this way applies (\-\d+)% to resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFlag("ElementalEquilibrium", true),
			mod.NewList("EnemyModifier", mod.NewFloat("FireExposure", mod.TypeBase, num).Tag(mod.Condition("HitByColdDamage", "HitByLightningDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("ColdExposure", mod.TypeBase, num).Tag(mod.Condition("HitByFireDamage", "HitByLightningDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("LightningExposure", mod.TypeBase, num).Tag(mod.Condition("HitByFireDamage", "HitByColdDamage"))),
		}, ""
	},
	`enemies you hit with elemental damage temporarily get (\+\d+)% resistance to those elements and (\-\d+)% resistance to other elements`: func(plus float64, captures []string) ([]mod.Mod, interface{}) {
		minus := utils.Float(captures[1])
		return []mod.Mod{
			mod.NewFlag("ElementalEquilibrium", true),
			mod.NewFlag("ElementalEquilibriumLegacy", true),
			mod.NewList("EnemyModifier", mod.NewFloat("FireResist", mod.TypeBase, plus).Tag(mod.Condition("HitByFireDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("FireResist", mod.TypeBase, minus).Tag(mod.Condition("HitByFireDamage").Neg(true)).Tag(mod.Condition("HitByColdDamage", "HitByLightningDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("ColdResist", mod.TypeBase, plus).Tag(mod.Condition("HitByColdDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("ColdResist", mod.TypeBase, minus).Tag(mod.Condition("HitByColdDamage").Neg(true)).Tag(mod.Condition("HitByFireDamage", "HitByLightningDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("LightningResist", mod.TypeBase, plus).Tag(mod.Condition("HitByLightningDamage"))),
			mod.NewList("EnemyModifier", mod.NewFloat("LightningResist", mod.TypeBase, minus).Tag(mod.Condition("HitByLightningDamage").Neg(true)).Tag(mod.Condition("HitByFireDamage", "HitByColdDamage"))),
		}, ""
	},
	`projectile attack hits deal up to 30% more damage to targets at the start of their movement, dealing less damage to targets as the projectile travels farther`: []mod.Mod{mod.NewFlag("PointBlank", true)},
	`leech energy shield instead of life`: []mod.Mod{mod.NewFlag("GhostReaver", true)},
	`minions explode when reduced to low life, dealing 33% of their maximum life as fire damage to surrounding enemies`: []mod.Mod{mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "MinionInstability"})},
	`minions explode when reduced to low life, dealing 33% of their life as fire damage to surrounding enemies`:         []mod.Mod{mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "MinionInstability"})},
	`all bonuses from an equipped shield apply to your minions instead of you`:                                          []mod.Mod{}, // The node itself is detected by the code that handles it
	`spend energy shield before mana for skill m?a?n?a? ?costs`:                                                         []mod.Mod{},
	`you have perfect agony if you've dealt a critical strike recently`:                                                 []mod.Mod{mod.NewList("Keystone", "Perfect Agony").Tag(mod.Condition("CritRecently"))},
	`energy shield protects mana instead of life`:                                                                       []mod.Mod{mod.NewFlag("EnergyShieldProtectsMana", true)},
	`modifiers to critical strike multiplier also apply to damage over time multiplier for ailments from critical strikes at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("CritMultiplierAppliesToDegen", mod.TypeBase, num)}, ""
	},
	`your bleeding does not deal extra damage while the enemy is moving`: []mod.Mod{mod.NewFlag("Condition:NoExtraBleedDamageToMovingEnemy", true)},
	`you can inflict bleeding on an enemy up to (\d+) times?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("BleedStacksMax", mod.TypeOverride, num),
			mod.NewFlag("Condition:HaveCrimsonDance", true),
		}, ""
	},
	`your minions spread caustic ground on death, dealing 20% of their maximum life as chaos damage per second`: []mod.Mod{mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "SiegebreakerCausticGround"})},
	`your minions spread burning ground on death, dealing 20% of their maximum life as fire damage per second`:  []mod.Mod{mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "ReplicaSiegebreakerBurningGround"})},
	`you can have an additional brand attached to an enemy`:                                                     []mod.Mod{mod.NewFloat("BrandsAttachedLimit", mod.TypeBase, 1)},
	`gain (\d+) grasping vines each second while stationary`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("Multiplier:GraspingVinesCount", mod.TypeBase, num).Tag(mod.Multiplier("StationarySeconds").Base(0).Limit(10).LimitTotal(true)).Tag(mod.Condition("Stationary")),
		}, ""
	},
	`all damage inflicts poison against enemies affected by at least (\d+) grasping vines`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("PoisonChance", mod.TypeBase, 100).Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)),
			mod.NewFlag("FireCanPoison", true).Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)),
			mod.NewFlag("ColdCanPoison", true).Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)),
			mod.NewFlag("LightningCanPoison", true).Tag(mod.MultiplierThreshold("GraspingVinesAffectingEnemy").Threshold(num)),
		}, ""
	},
	`attack projectiles always inflict bleeding and maim, and knock back enemies`: []mod.Mod{
		mod.NewFloat("BleedChance", mod.TypeBase, 100).Flag(mod.MFlagAttack).Flag(mod.MFlagProjectile),
		mod.NewFloat("EnemyKnockbackChance", mod.TypeBase, 100).Flag(mod.MFlagAttack).Flag(mod.MFlagProjectile),
	},
	`projectiles cannot pierce, fork or chain`: []mod.Mod{
		mod.NewFlag("CannotPierce", true).Flag(mod.MFlagProjectile),
		mod.NewFlag("CannotChain", true).Flag(mod.MFlagProjectile),
		mod.NewFlag("CannotFork", true).Flag(mod.MFlagProjectile),
	},
	`critical strikes inflict scorch, brittle and sapped`: []mod.Mod{mod.NewFlag("CritAlwaysAltAilments", true)},
	`chance to block attack damage is doubled`:            []mod.Mod{mod.NewFloat("BlockChance", mod.TypeMore, 100)},
	`chance to block spell damage is doubled`:             []mod.Mod{mod.NewFloat("SpellBlockChance", mod.TypeMore, 100)},
	`you take (\d+)% of damage from blocked hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("BlockEffect", mod.TypeBase, num)}, ""
	},
	`ignore attribute requirements`:              []mod.Mod{mod.NewFlag("IgnoreAttributeRequirements", true)},
	`gain no inherent bonuses from attributes`:   []mod.Mod{mod.NewFlag("NoAttributeBonuses", true)},
	`gain no inherent bonuses from strength`:     []mod.Mod{mod.NewFlag("NoStrengthAttributeBonuses", true)},
	`gain no inherent bonuses from dexterity`:    []mod.Mod{mod.NewFlag("NoDexterityAttributeBonuses", true)},
	`gain no inherent bonuses from intelligence`: []mod.Mod{mod.NewFlag("NoIntelligenceAttributeBonuses", true)},
	`all damage taken bypasses energy shield`: []mod.Mod{
		mod.NewFloat("PhysicalEnergyShieldBypass", mod.TypeBase, 100),
		mod.NewFloat("LightningEnergyShieldBypass", mod.TypeBase, 100),
		mod.NewFloat("ColdEnergyShieldBypass", mod.TypeBase, 100),
		mod.NewFloat("FireEnergyShieldBypass", mod.TypeBase, 100),
	},
	`auras from your skills do not affect allies`: []mod.Mod{mod.NewFlag("SelfAuraSkillsCannotAffectAllies", true)},
	`auras from your skills have (\d+)% more effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeMore, num)}, ""
	},
	`auras from your skills have (\d+)% increased effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SkillAuraEffectOnSelf", mod.TypeIncrease, num)}, ""
	},
	`increases and reductions to mana regeneration rate instead apply to rage regeneration rate`: []mod.Mod{mod.NewFlag("ManaRegenToRageRegen", true)},
	`increases and reductions to maximum energy shield instead apply to ward`:                    []mod.Mod{mod.NewFlag("EnergyShieldToWard", true)},
	`(\d+)% of damage taken bypasses ward`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("WardBypass", mod.TypeBase, num)}, ""
	},
	`maximum energy shield is (\d+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("EnergyShield", mod.TypeOverride, num)}, ""
	},
	`while not on full life, sacrifice ([\d\.]+)% of mana per second to recover that much life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("ManaDegen", mod.TypeBase, 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.Condition("FullLife").Neg(true)),
			mod.NewFloat("LifeRecovery", mod.TypeBase, 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.Condition("FullLife").Neg(true)),
		}, ""
	},
	`you are blind`: []mod.Mod{mod.NewFlag("Condition:Blinded", true)},
	`armour applies to fire, cold and lightning damage taken from hits instead of physical damage`: []mod.Mod{
		mod.NewFlag("ArmourAppliesToFireDamageTaken", true),
		mod.NewFlag("ArmourAppliesToColdDamageTaken", true),
		mod.NewFlag("ArmourAppliesToLightningDamageTaken", true),
		mod.NewFlag("ArmourDoesNotApplyToPhysicalDamageTaken", true),
	},
	`maximum damage reduction for any damage type is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("DamageReductionMax", mod.TypeOverride, num)}, ""
	},
	`(\d+)% of maximum mana is converted to twice that much armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("ManaConvertToArmour", mod.TypeBase, num),
		}, ""
	},
	`life recovery from flasks also applies to energy shield`: []mod.Mod{mod.NewFlag("LifeFlaskAppliesToEnergyShield", true)},
	`life leech effects recover energy shield instead while on full life`: []mod.Mod{
		mod.NewFlag("ImmortalAmbition", true).Tag(mod.Condition("FullLife")).Tag(mod.Condition("LeechingLife")),
	},
	`shepherd of souls`: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, -30).Tag(mod.SkillType(string(data.SkillTypeVaal)).Neg(true))},
	`adds (\d+) to (\d+) attack physical damage to melee skills per (\d+) dexterity while you are unencumbered`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			// Hollow Palm 3 suffixes
			mod.NewFloat("PhysicalMin", mod.TypeBase, utils.Float(captures[0])).Flag(mod.MFlagMelee).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.PerStat(utils.Float(captures[2]), "Dex")).Tag(mod.Condition("Unencumbered")),
			mod.NewFloat("PhysicalMax", mod.TypeBase, utils.Float(captures[1])).Flag(mod.MFlagMelee).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.PerStat(utils.Float(captures[2]), "Dex")).Tag(mod.Condition("Unencumbered")),
		}, ""
	},
	`(\d+)% more attack damage if accuracy rating is higher than maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("Damage", mod.TypeMore, num).Source("Damage").Flag(mod.MFlagAttack).Tag(mod.Condition("MainHandAccRatingHigherThanMaxLife")).Tag(mod.Condition("MainHandAttack")),
			mod.NewFloat("Damage", mod.TypeMore, num).Source("Damage").Flag(mod.MFlagAttack).Tag(mod.Condition("OffHandAccRatingHigherThanMaxLife")).Tag(mod.Condition("OffHandAttack")),
		}, ""
	},
	// Legacy support
	`(\d+)% chance to defend with double armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("ArmourDefense", mod.TypeMAX, 100).Tag(mod.Condition("ArmourMax")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(num/100, 1.0)*100).Tag(mod.Condition("ArmourAvg")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(math.Floor(num/100), 1.0)*100).Tag(mod.Condition("ArmourMax").Neg(true)).Tag(mod.Condition("ArmourAvg").Neg(true)),
		}, ""
	},
	// Masteries
	"off hand accuracy is equal to main hand accuracy while wielding a sword": []mod.Mod{mod.NewFlag("Condition:OffHandAccuracyIsMainHandAccuracy", true).Tag(mod.Condition("UsingSword"))},
	`(\d+)% chance to defend with (\d+)% of armour`: func(numChance float64, captures []string) ([]mod.Mod, string) {
		numArmourMultiplier := utils.Float(captures[1])
		return []mod.Mod{
			mod.NewFloat("ArmourDefense", mod.TypeMAX, utils.Float(captures[1])-100).Tag(mod.Condition("ArmourMax")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(numChance/100, 1.0)*(numArmourMultiplier-100)).Tag(mod.Condition("ArmourAvg")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(math.Floor(numChance/100), 1.0)*(numArmourMultiplier-100)).Tag(mod.Condition("ArmourMax").Neg(true)).Tag(mod.Condition("ArmourAvg").Neg(true)),
		}, ""
	},
	`defend with (\d+)% of armour while not on low energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("ArmourDefense", mod.TypeMAX, num-100).Tag(mod.Condition("LowEnergyShield").Neg(true)),
		}, ""
	},
	// Exerted Attacks
	`exerted attacks deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExertIncrease", mod.TypeIncrease, num).Flag(mod.MFlagAttack),
		}, ""
	},
	// Ascendant
	`grants (\d+) passive skill points?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("ExtraPoints", mod.TypeBase, num),
		}, ""
	},
	`can allocate passives from the \w+'s starting point`: []mod.Mod{},
	`projectiles gain damage as they travel farther, dealing up to (\d+)% increased damage with hits to targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", mod.TypeIncrease, num).Flag(mod.MFlagAttack).Flag(mod.MFlagProjectile).Tag(mod.DistanceRamp([][]int{{35, 0}, {70, 1}})),
		}, ""
	},
	`(\d+)% chance to gain elusive on kill`: []mod.Mod{
		mod.NewFlag("Condition:CanBeElusive", true),
	},
	"immune to elemental ailments while on consecrated ground": func(num float64, captures []string) ([]mod.Mod, string) {
		mods := make([]mod.Mod, len(data.Ailment("").Values()))
		for i, ailment := range data.Ailment("").Values() {
			mods[i] = MOD("Avoid"+string(ailment), mod.TypeBase, 100).Tag(mod.Condition("OnConsecratedGround"))
		}
		return mods, ""
	},
	// Assassin
	`poison you inflict with critical strikes deals (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num).KeywordFlag(mod.KeywordFlagPoison).Tag(mod.Condition("CriticalStrike")),
		}, ""
	},
	`(\d+)% chance to gain elusive on critical strike`: []mod.Mod{
		mod.NewFlag("Condition:CanBeElusive", true),
	},
	`(\d+)% more damage while there is at most one rare or unique enemy nearby`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num).Tag(mod.Condition("AtMostOneNearbyRareOrUniqueEnemy")),
		}, ""
	},
	`(\d+)% reduced damage taken while there are at least two rare or unique enemies nearby`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DamageTaken", "INC", -num).Tag(mod.MultiplierThreshold("NearbyRareOrUniqueEnemies").Threshold(2)),
		}, ""
	},
	"you take no extra damage from critical strikes while elusive": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.Condition("Elusive")),
		}, ""
	},
	// Berserker
	`gain \d+ rage when you kill an enemy`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`gain \d+ rage when you use a warcry`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`you and nearby party members gain \d+ rage when you warcry`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`gain \d+ rage on hit with attacks, no more than once every [\d\.]+ seconds`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	"inherent effects from having rage are tripled": []mod.Mod{mod.NewFloat("Multiplier:RageEffect", mod.TypeBase, 2)},
	`cannot be stunned while you have at least (\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AvoidStun", "BASE", 100).Tag(mod.MultiplierThreshold("Rage").Threshold(num)),
		}, ""
	},
	`lose ([\d\.]+)% of life per second per rage while you are not losing rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeDegen", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.Multiplier("Rage")),
		}, ""
	},
	`if you've warcried recently, you and nearby allies have (\d+)% increased attack speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("Speed", "INC", num).Flag(mod.MFlagAttack)}).Tag(mod.Condition("UsedWarcryRecently")),
		}, ""
	},
	`gain (\d+)% increased armour per (\d+) power for 8 seconds when you warcry, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Armour", "INC", num).Tag(mod.Multiplier("WarcryPower").Div(utils.Float(captures[1])).GlobalLimit(utils.Float(captures[2])).GlobalLimitKey("WarningCall")).Tag(mod.Condition("UsedWarcryInPast8Seconds")),
		}, ""
	},
	`warcries grant (\d+) rage per (\d+) power if you have less than (\d+) rage`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`exerted attacks deal (\d+)% more attack damage if a warcry sacrificed rage recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExertAttackIncrease", "MORE", num).Flag(mod.MFlagAttack),
		}, ""
	},
	// Champion
	"cannot be stunned while you have fortify": []mod.Mod{MOD("AvoidStun", "BASE", 100).Tag(mod.Condition("Fortified"))},
	"cannot be stunned while fortified":        []mod.Mod{MOD("AvoidStun", "BASE", 100).Tag(mod.Condition("Fortified"))},
	"fortify":                                  []mod.Mod{mod.NewFlag("Condition:Fortified", true)},
	`you have (\d+) fortification`: []mod.Mod{
		mod.NewFlag("Condition:Fortified", true),
	},
	"enemies taunted by you cannot evade attacks": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("CannotEvade").Tag(mod.Condition("Taunted"))})},
	`if you've impaled an enemy recently, you and nearby allies have \+(\d+) to armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("Armour", mod.TypeBase, num)}).Tag(mod.Condition("ImpaledRecently")),
		}, ""
	},
	"your hits permanently intimidate enemies that are on full life": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Intimidated", true)})},
	`you and allies affected by your placed banners regenerate ([\d\.]+)% of life per second for each stage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("AffectedByPlacedBanner")).Tag(mod.Multiplier("BannerStage").Base(0)),
		}, ""
	},
	// Chieftain
	`enemies near your totems take (\d+)% increased physical and fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("PhysicalDamageTaken", mod.TypeIncrease, num)}),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("FireDamageTaken", mod.TypeIncrease, num)}),
		}, ""
	},
	`every \d+ seconds, gain (\d+)% of physical damage as extra fire damage for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalDamageGainAsFire", "BASE", num).Tag(mod.Condition("NgamahuFlamesAdvance")),
		}, ""
	},
	`(\d+)% more damage for each endurance charge lost recently, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num).Tag(mod.Multiplier("EnduranceChargesLostRecently").Limit(utils.Float(captures[1])).LimitTotal(true)),
		}, ""
	},
	`(\d+)% more damage if you've lost an endurance charge in the past 8 seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num).Tag(mod.Condition("LostEnduranceChargeInPast8Sec")),
		}, ""
	},
	`trigger level (\d+) (.+) when you attack with a non-vaal slam skill near an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	"projectiles pierce all nearby targets": []mod.Mod{mod.NewFlag("PierceAllTargets", true)},
	// Deadeye
	`gain \+(\d+) life when you hit a bleeding enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnHit", "BASE", num).Tag(mod.ActorCondition("enemy", "Bleeding"))}, ""
	},
	"accuracy rating is doubled": []mod.Mod{mod.NewFloat("Accuracy", mod.TypeMore, 100)},
	`(\d+)% increased blink arrow and mirror arrow cooldown recovery speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CooldownRecovery", "INC", num).Tag(mod.SkillName("Blink Arrow", "Mirror Arrow")),
		}, ""
	},
	"critical strikes which inflict bleeding also inflict rupture": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("Condition:CanInflictRupture").Tag(mod.Condition("NeverCrit").Neg(true)),
		}, ""
	},
	`gain \d+ gale force when you use a skill`: []mod.Mod{
		mod.NewFlag("Condition:CanGainGaleForce", true),
	},
	"if you've used a skill recently, you and nearby allies have tailwind": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFlag("Condition:Tailwind", true)}).Tag(mod.Condition("UsedSkillRecently"))},
	"you and nearby allies have tailwind":                                  []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFlag("Condition:Tailwind", true)})},
	`projectiles deal (\d+)% more damage for each remaining chain`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).Flag(mod.MFlagProjectile).Tag(mod.PerStat(0, "ChainRemaining"))}, ""
	},
	`projectiles deal (\d+)% increased damage for each remaining chain`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Flag(mod.MFlagProjectile).Tag(mod.PerStat(0, "ChainRemaining"))}, ""
	},
	"far shot": []mod.Mod{mod.NewFlag("FarShot", true)},
	`(\d+)% increased mirage archer duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MirageArcherDuration", mod.TypeIncrease, num)}, ""
	},
	`([\-\+]\d+) to maximum number of summoned mirage archers`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MirageArcherMaxCount", mod.TypeBase, num)}, ""
	},
	// Elementalist
	`gain (\d+)% increased area of effect for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("PendulumOfDestructionAreaOfEffect"))}, ""
	},
	`gain (\d+)% increased elemental damage for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalDamage", "INC", num).Tag(mod.Condition("PendulumOfDestructionElementalDamage"))}, ""
	},
	`for each element you've been hit by damage of recently, (\d+)% increased damage of that element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FireDamage", "INC", num).Tag(mod.Condition("HitByFireDamageRecently")),
			MOD("ColdDamage", "INC", num).Tag(mod.Condition("HitByColdDamageRecently")),
			MOD("LightningDamage", "INC", num).Tag(mod.Condition("HitByLightningDamageRecently")),
		}, ""
	},
	`for each element you've been hit by damage of recently, (\d+)% reduced damage taken of that element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FireDamageTaken", "INC", -num).Tag(mod.Condition("HitByFireDamageRecently")),
			MOD("ColdDamageTaken", "INC", -num).Tag(mod.Condition("HitByColdDamageRecently")),
			MOD("LightningDamageTaken", "INC", -num).Tag(mod.Condition("HitByLightningDamageRecently")),
		}, ""
	},
	`gain convergence when you hit a unique enemy, no more than once every \d+ seconds`: []mod.Mod{
		mod.NewFlag("Condition:CanGainConvergence", true),
	},
	`(\d+)% increased area of effect while you don't have convergence`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("Convergence").Neg(true))}, ""
	},
	`exposure you inflict applies an extra (-?\d+)% to the affected resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ExtraExposure", mod.TypeBase, num)}, ""
	},
	"cannot take reflected elemental damage": []mod.Mod{MOD("ElementalReflectedDamageTaken", "MORE", -100)},
	`every \d+ seconds:`:                     []mod.Mod{},
	`gain chilling conflux for \d seconds`: []mod.Mod{
		FLAG("PhysicalCanChill").Tag(mod.Condition("ChillingConflux")),
		FLAG("LightningCanChill").Tag(mod.Condition("ChillingConflux")),
		FLAG("FireCanChill").Tag(mod.Condition("ChillingConflux")),
		FLAG("ChaosCanChill").Tag(mod.Condition("ChillingConflux")),
	},
	`gain shocking conflux for \d seconds`: []mod.Mod{
		MOD("EnemyShockChance", "BASE", 100).Tag(mod.Condition("ShockingConflux")),
		FLAG("PhysicalCanShock").Tag(mod.Condition("ShockingConflux")),
		FLAG("ColdCanShock").Tag(mod.Condition("ShockingConflux")),
		FLAG("FireCanShock").Tag(mod.Condition("ShockingConflux")),
		FLAG("ChaosCanShock").Tag(mod.Condition("ShockingConflux")),
	},
	`gain igniting conflux for \d seconds`: []mod.Mod{
		MOD("EnemyIgniteChance", "BASE", 100).Tag(mod.Condition("IgnitingConflux")),
		FLAG("PhysicalCanIgnite").Tag(mod.Condition("IgnitingConflux")),
		FLAG("LightningCanIgnite").Tag(mod.Condition("IgnitingConflux")),
		FLAG("ColdCanIgnite").Tag(mod.Condition("IgnitingConflux")),
		FLAG("ChaosCanIgnite").Tag(mod.Condition("IgnitingConflux")),
	},
	`gain chilling, shocking and igniting conflux for \d seconds`: []mod.Mod{},
	"you have igniting, chilling and shocking conflux while affected by glorious madness": []mod.Mod{
		FLAG("PhysicalCanChill").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("LightningCanChill").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("FireCanChill").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("ChaosCanChill").Tag(mod.Condition("AffectedByGloriousMadness")),
		MOD("EnemyIgniteChance", "BASE", 100).Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("PhysicalCanIgnite").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("LightningCanIgnite").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("ColdCanIgnite").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("ChaosCanIgnite").Tag(mod.Condition("AffectedByGloriousMadness")),
		MOD("EnemyShockChance", "BASE", 100).Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("PhysicalCanShock").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("ColdCanShock").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("FireCanShock").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("ChaosCanShock").Tag(mod.Condition("AffectedByGloriousMadness")),
	},
	"immune to elemental ailments while affected by glorious madness": func(num float64, captures []string) ([]mod.Mod, string) {
		mods := make([]mod.Mod, len(data.Ailment("").Values()))
		for i, ailment := range data.Ailment("").Values() {
			mods[i] = MOD("Avoid"+string(ailment), mod.TypeBase, 100).Tag(mod.Condition("AffectedByGloriousMadness"))
		}
		return mods, ""
	},
	"summoned golems are immune to elemental damage": []mod.Mod{
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("FireResist", mod.TypeOverride, 100)}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("FireResistMax", mod.TypeOverride, 100)}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("ColdResist", mod.TypeOverride, 100)}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("ColdResistMax", mod.TypeOverride, 100)}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("LightningResist", mod.TypeOverride, 100)}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("LightningResistMax", mod.TypeOverride, 100)}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
	},
	`(\d+)% increased golem damage per summoned golem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeIncrease, num)}).Tag(mod.SkillType(string(data.SkillTypeGolem))).Tag(mod.PerStat(0, "ActiveGolemLimit")),
		}, ""
	},
	`shocks from your hits always increase damage taken by at least (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ShockBase", mod.TypeBase, num)}, ""
	},
	`chills from your hits always reduce action speed by at least (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ChillBase", mod.TypeBase, num)}, ""
	},
	`(\d+)% more damage with ignites you inflict with hits for which the highest damage type is fire`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num).KeywordFlag(mod.KeywordFlagIgnite).Tag(mod.Condition("FireIsHighestDamageType")),
		}, ""
	},
	`(\d+)% more effect of cold ailments you inflict with hits for which the highest damage type is cold`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyChillEffect", "MORE", num).Tag(mod.Condition("ColdIsHighestDamageType")),
			MOD("EnemyBrittleEffect", "MORE", num).Tag(mod.Condition("ColdIsHighestDamageType")),
		}, ""
	},
	`(\d+)% more effect of lightning ailments you inflict with hits if the highest damage type is lightning`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyShockEffect", "MORE", num).Tag(mod.Condition("LightningIsHighestDamageType")),
			MOD("EnemySapEffect", "MORE", num).Tag(mod.Condition("LightningIsHighestDamageType")),
		}, ""
	},
	`your chills can reduce action speed by up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ChillMax", mod.TypeOverride, num)}, ""
	},
	"your hits always ignite": []mod.Mod{mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 100)},
	"hits always ignite":      []mod.Mod{mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 100)},
	"your hits always shock":  []mod.Mod{mod.NewFloat("EnemyShockChance", mod.TypeBase, 100)},
	"hits always shock":       []mod.Mod{mod.NewFloat("EnemyShockChance", mod.TypeBase, 100)},
	"all damage with hits can ignite": []mod.Mod{
		mod.NewFlag("PhysicalCanIgnite", true),
		mod.NewFlag("ColdCanIgnite", true),
		mod.NewFlag("LightningCanIgnite", true),
		mod.NewFlag("ChaosCanIgnite", true),
	},
	"all damage can ignite": []mod.Mod{
		mod.NewFlag("PhysicalCanIgnite", true),
		mod.NewFlag("ColdCanIgnite", true),
		mod.NewFlag("LightningCanIgnite", true),
		mod.NewFlag("ChaosCanIgnite", true),
	},
	"all damage with hits can chill": []mod.Mod{
		mod.NewFlag("PhysicalCanChill", true),
		mod.NewFlag("FireCanChill", true),
		mod.NewFlag("LightningCanChill", true),
		mod.NewFlag("ChaosCanChill", true),
	},
	"all damage with hits can shock": []mod.Mod{
		mod.NewFlag("PhysicalCanShock", true),
		mod.NewFlag("FireCanShock", true),
		mod.NewFlag("ColdCanShock", true),
		mod.NewFlag("ChaosCanShock", true),
	},
	"all damage can shock": []mod.Mod{
		mod.NewFlag("PhysicalCanShock", true),
		mod.NewFlag("FireCanShock", true),
		mod.NewFlag("ColdCanShock", true),
		mod.NewFlag("ChaosCanShock", true),
	},
	"other aegis skills are disabled": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeAegis))),
			FLAG("EnableSkill").Tag(mod.SkillName("Primal Aegis")),
		}, ""
	},
	`primal aegis can take (\d+) elemental damage per allocated notable passive skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ElementalAegisValue", "MAX", num).Tag(mod.Multiplier("AllocatedNotable").Base(0)).Tag(mod.GlobalEffect("Buff").Unscalable(true)),
		}, ""
	},
	// Gladiator
	"chance to block spell damage is equal to chance to block attack damage":                 []mod.Mod{mod.NewFlag("SpellBlockChanceIsBlockChance", true)},
	"maximum chance to block spell damage is equal to maximum chance to block attack damage": []mod.Mod{mod.NewFlag("SpellBlockChanceMaxIsBlockChanceMax", true)},
	"your counterattacks deal double damage": []mod.Mod{
		MOD("DoubleDamageChance", "BASE", 100).Tag(mod.SkillName("Reckoning")),
		MOD("DoubleDamageChance", "BASE", 100).Tag(mod.SkillName("Riposte")),
		MOD("DoubleDamageChance", "BASE", 100).Tag(mod.SkillName("Vengeance")),
	},
	`attack damage is lucky if you[' ]h?a?ve blocked in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("LuckyHits").Tag(mod.Condition("BlockedRecently")),
		}, ""
	},
	`hits ignore enemy monster physical damage reduction if you[' ]h?a?ve blocked in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("IgnoreEnemyPhysicalDamageReduction").Tag(mod.Condition("BlockedRecently")),
		}, ""
	},
	`(\d+)% more attack and movement speed per challenger charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Speed", "MORE", num).Flag(mod.MFlagAttack).Tag(mod.Multiplier("ChallengerCharge").Base(0)),
			MOD("MovementSpeed", "MORE", num).Tag(mod.Multiplier("ChallengerCharge").Base(0)),
		}, ""
	},
	// Guardian
	`grants armour equal to (\d+)% of your reserved life to you and nearby allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("GrantReservedLifeAsAura", "LIST", mod.GrantReservedLifeAsAura{Mod: MOD("Armour", "BASE", num/100)}),
		}, ""
	},
	`grants maximum energy shield equal to (\d+)% of your reserved mana to you and nearby allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("GrantReservedManaAsAura", "LIST", mod.GrantReservedManaAsAura{Mod: MOD("EnergyShield", "BASE", num/100)}),
		}, ""
	},
	"warcries cost no mana": []mod.Mod{MOD("ManaCost", "MORE", -100).KeywordFlag(mod.KeywordFlagWarcry)},
	`\+(\d+)% chance to block attack damage for \d seconds? every \d seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("BlockChance", "BASE", num).Tag(mod.Condition("BastionOfHopeActive")),
		}, ""
	},
	`if you've blocked in the past \d+ seconds, you and nearby allies cannot be stunned`: []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("AvoidStun", mod.TypeBase, 100)}).Tag(mod.Condition("BlockedRecently"))},
	`if you've attacked recently, you and nearby allies have \+(\d+)% chance to block attack damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("BlockChance", mod.TypeBase, num)}).Tag(mod.Condition("AttackedRecently"))}, ""
	},
	`if you've cast a spell recently, you and nearby allies have \+(\d+)% chance to block spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("SpellBlockChance", mod.TypeBase, num)}).Tag(mod.Condition("CastSpellRecently"))}, ""
	},
	`while there is at least one nearby ally, you and nearby allies deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("Damage", mod.TypeMore, num)}).Tag(mod.MultiplierThreshold("NearbyAlly").Threshold(1)),
		}, ""
	},
	"while there are at least five nearby allies, you and nearby allies have onslaught": []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFlag("Onslaught", true)}).Tag(mod.MultiplierThreshold("NearbyAlly").Threshold(5)),
	},
	// Hierophant
	`you and your totems regenerate ([\d\.]+)% of life per second for each summoned totem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeRegenPercent", "BASE", num).Tag(mod.PerStat(0, "TotemsSummoned")),
			MOD("LifeRegenPercent", "BASE", num).Tag(mod.PerStat(0, "TotemsSummoned")).KeywordFlag(mod.KeywordFlagTotem),
		}, ""
	},
	`enemies take (\d+)% increased damage for each of your brands attached to them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Multiplier("BrandsAttached").Base(0))}),
		}, ""
	},
	`immune to elemental ailments while you have arcane surge`: func(num float64, captures []string) ([]mod.Mod, string) {
		mods := make([]mod.Mod, len(data.Ailment("").Values()))
		for i, ailment := range data.Ailment("").Values() {
			mods[i] = MOD("Avoid"+string(ailment), mod.TypeBase, 100).Tag(mod.Condition("AffectedByArcaneSurge"))
		}
		return mods, ""
	},
	`brands have (\d+)% more activation frequency if (\d+)% of attached duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("BrandActivationFrequency", "MORE", num).Tag(mod.Condition("BrandLastQuarter")),
		}, ""
	},
	// Inquisitor
	"critical strikes ignore enemy monster elemental resistances": []mod.Mod{FLAG("IgnoreElementalResistances").Tag(mod.Condition("CriticalStrike"))},
	`non-critical strikes penetrate (\d+)% of enemy elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ElementalPenetration", "BASE", num).Tag(mod.Condition("CriticalStrike").Neg(true)),
		}, ""
	},
	`consecrated ground you create applies (\d+)% increased damage taken to enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTakenConsecratedGround", "INC", num).Tag(mod.Condition("OnConsecratedGround"))})}, ""
	},
	"you have consecrated ground around you while stationary": []mod.Mod{FLAG("Condition:OnConsecratedGround").Tag(mod.Condition("Stationary"))},
	"consecrated ground you create grants immunity to elemental ailments to you and allies": func(num float64, captures []string) ([]mod.Mod, string) {
		mods := make([]mod.Mod, len(data.Ailment("").Values()))
		for i, ailment := range data.Ailment("").Values() {
			mods[i] = MOD("Avoid"+string(ailment), mod.TypeBase, 100).Tag(mod.Condition("OnConsecratedGround"))
		}
		return mods, ""
	},
	"gain fanaticism for 4 seconds on reaching maximum fanatic charges": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFlag("Condition:CanGainFanaticism", true),
		}, ""
	},
	`(\d+)% increased critical strike chance per point of strength or intelligence, whichever is lower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "INC", num).Tag(mod.PerStat(0, "Str")).Tag(mod.Condition("IntHigherThanStr")),
			MOD("CritChance", "INC", num).Tag(mod.PerStat(0, "Int")).Tag(mod.Condition("IntHigherThanStr").Neg(true)),
		}, ""
	},
	"consecrated ground you create causes life regeneration to also recover energy shield for you and allies": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("LifeRegenerationRecoversEnergyShield").Tag(mod.Condition("OnConsecratedGround")),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("LifeRegenerationRecoversEnergyShield").Tag(mod.Condition("OnConsecratedGround"))}),
		}, ""
	},
	`(\d+)% more attack damage for each non-instant spell you've cast in the past 8 seconds, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num).Flag(mod.MFlagAttack).Tag(mod.Multiplier("CastLast8Seconds").Limit(utils.Float(captures[1])).LimitTotal(true)),
		}, ""
	},
	// Juggernaut
	"armour received from body armour is doubled":           []mod.Mod{mod.NewFlag("Unbreakable", true)},
	"action speed cannot be modified to below base value":   []mod.Mod{mod.NewFlag("ActionSpeedCannotBeBelowBase", true)},
	"movement speed cannot be modified to below base value": []mod.Mod{mod.NewFlag("MovementSpeedCannotBeBelowBase", true)},
	"you cannot be slowed to below base speed":              []mod.Mod{mod.NewFlag("ActionSpeedCannotBeBelowBase", true)},
	"cannot be slowed to below base speed":                  []mod.Mod{mod.NewFlag("ActionSpeedCannotBeBelowBase", true)},
	"gain accuracy rating equal to your strength":           []mod.Mod{MOD("Accuracy", "BASE", 1).Tag(mod.PerStat(0, "Str"))},
	"gain accuracy rating equal to twice your strength":     []mod.Mod{MOD("Accuracy", "BASE", 2).Tag(mod.PerStat(0, "Str"))},
	// Necromancer
	"your offering skills also affect you": []mod.Mod{
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", mod.SkillData{Key: "buffNotPlayer", Value: 0})}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering")),
	},
	`your offerings have (\d+)% reduced effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("BuffEffectOnPlayer", "INC", -num)}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering")),
		}, ""
	},
	`if you've consumed a corpse recently, you and your minions have (\d+)% increased area of effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("ConsumedCorpseRecently")),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("AreaOfEffect", "INC", num)}).Tag(mod.Condition("ConsumedCorpseRecently")),
		}, ""
	},
	`with at least one nearby corpse, you and nearby allies deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("Damage", mod.TypeMore, num)}).Tag(mod.MultiplierThreshold("NearbyCorpse").Threshold(1)),
		}, ""
	},
	`for each nearby corpse, you and nearby allies regenerate ([\d\.]+)% of energy shield per second, up to ([\d\.]+)% per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("EnergyShieldRegenPercent", mod.TypeBase, num)}).Tag(mod.Multiplier("NearbyCorpse").Limit(utils.Float(captures[1])).LimitTotal(true)),
		}, ""
	},
	`for each nearby corpse, you and nearby allies regenerate (\d+) mana per second, up to (\d+) per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("ManaRegen", mod.TypeBase, num)}).Tag(mod.Multiplier("NearbyCorpse").Limit(utils.Float(captures[1])).LimitTotal(true)),
		}, ""
	},
	`(\d+)% increased attack and cast speed for each corpse consumed recently, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Speed", "INC", num).Tag(mod.Multiplier("CorpseConsumedRecently").Limit(utils.Float(captures[1]) / num)),
		}, ""
	},
	"enemies near corpses you spawned recently are chilled and shocked": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Chilled", true)}).Tag(mod.Condition("SpawnedCorpseRecently")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Shocked", true)}).Tag(mod.Condition("SpawnedCorpseRecently")),
		MOD("ChillBase", "BASE", data.NonDamagingAilments[data.AilmentChill].Default).Tag(mod.Condition("SpawnedCorpseRecently")),
		MOD("ShockBase", "BASE", data.NonDamagingAilments[data.AilmentShock].Default).Tag(mod.Condition("SpawnedCorpseRecently")),
	},
	`regenerate (\d+)% of energy shield over 2 seconds when you consume a corpse`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldRegenPercent", "BASE", num/2).Tag(mod.Condition("ConsumedCorpseInPast2Sec")),
		}, ""
	},
	`regenerate (\d+)% of mana over 2 seconds when you consume a corpse`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ManaRegen", "BASE", 1).Tag(mod.PercentStat("Mana", num/2)).Tag(mod.Condition("ConsumedCorpseInPast2Sec")),
		}, ""
	},
	`corpses you spawn have (\d+)% increased maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("CorpseLife", mod.TypeIncrease, num),
		}, ""
	},
	// Occultist
	"enemies you curse have malediction": []mod.Mod{MOD("AffectedByCurseMod", "LIST", mod.AffectedByCurseMod{Mod: mod.NewFlag("HasMalediction", true)})},
	`when you kill an enemy, for each curse on that enemy, gain (\d+)% of non-chaos damage as extra chaos damage for 4 seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("NonChaosDamageGainAsChaos", "BASE", num).Tag(mod.Condition("KilledRecently")).Tag(mod.Multiplier("CurseOnEnemy").Base(0)),
		}, ""
	},
	"cannot be stunned while you have energy shield":                     []mod.Mod{MOD("AvoidStun", "BASE", 100).Tag(mod.Condition("HaveEnergyShield"))},
	`every second, inflict withered on nearby enemies for (\d+) seconds`: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	// Pathfinder
	"always poison on hit while using a flask": []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	`poisons you inflict during any flask effect have (\d+)% chance to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", utils.Float(captures[1])*num/100).KeywordFlag(mod.KeywordFlagPoison).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	"immune to elemental ailments during any flask effect": func(num float64, captures []string) ([]mod.Mod, string) {
		mods := make([]mod.Mod, len(data.Ailment("").Values()))
		for i, ailment := range data.Ailment("").Values() {
			mods[i] = MOD("Avoid"+string(ailment), mod.TypeBase, 100).Tag(mod.Condition("UsingFlask"))
		}
		return mods, ""
	},
	// Raider
	`nearby enemies have (\d+)% less accuracy rating while you have phasing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("Accuracy", "MORE", -num)}).Tag(mod.Condition("Phasing")),
		}, ""
	},
	"immune to elemental ailments while phasing": func(num float64, captures []string) ([]mod.Mod, string) {
		mods := make([]mod.Mod, len(data.Ailment("").Values()))
		for i, ailment := range data.Ailment("").Values() {
			mods[i] = MOD("Avoid"+string(ailment), mod.TypeBase, 100).Tag(mod.Condition("Phasing"))
		}
		return mods, ""
	},
	`nearby enemies have fire, cold and lightning exposure while you have phasing, applying -(\d+)% to those resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", -num)}).Tag(mod.Condition("Phasing")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", -num)}).Tag(mod.Condition("Phasing")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", -num)}).Tag(mod.Condition("Phasing")),
		}, ""
	},
	// Saboteur
	"immune to ignite and shock": []mod.Mod{
		mod.NewFloat("AvoidIgnite", mod.TypeBase, 100),
		mod.NewFloat("AvoidShock", mod.TypeBase, 100),
	},
	`you gain (\d+)% increased damage for each trap`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "INC", num).Tag(mod.PerStat(0, "ActiveTrapLimit")),
		}, ""
	},
	`you gain (\d+)% increased area of effect for each mine`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AreaOfEffect", "INC", num).Tag(mod.PerStat(0, "ActiveMineLimit")),
		}, ""
	},
	// Slayer
	`deal up to (\d+)% more melee damage to enemies, based on proximity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num).Flag(mod.MFlagAttack).Flag(mod.MFlagMelee).Tag(mod.MeleeProximity([]int{1, 0})),
		}, ""
	},
	"cannot be stunned while leeching":                                               []mod.Mod{MOD("AvoidStun", "BASE", 100).Tag(mod.Condition("Leeching"))},
	"you are immune to bleeding while leeching":                                      []mod.Mod{MOD("AvoidBleed", "BASE", 100).Tag(mod.Condition("Leeching"))},
	"life leech effects are not removed at full life":                                []mod.Mod{mod.NewFlag("CanLeechLifeOnFullLife", true)},
	"life leech effects are not removed when unreserved life is filled":              []mod.Mod{mod.NewFlag("CanLeechLifeOnFullLife", true)},
	"energy shield leech effects from attacks are not removed at full energy shield": []mod.Mod{mod.NewFlag("CanLeechLifeOnFullEnergyShield", true)},
	"cannot take reflected physical damage":                                          []mod.Mod{MOD("PhysicalReflectedDamageTaken", "MORE", -100)},
	`gain (\d+)% increased movement speed for 20 seconds when you kill an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MovementSpeed", "INC", num).Tag(mod.Condition("KilledRecently")),
		}, ""
	},
	`gain (\d+)% increased attack speed for 20 seconds when you kill a rare or unique enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Speed", "INC", num).Flag(mod.MFlagAttack).Tag(mod.Condition("KilledUniqueEnemy")),
		}, ""
	},
	`kill enemies that have (\d+)% or lower life when hit by your skills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CullPercent", "MAX", num)}, ""
	},
	// Trickster
	`(\d+)% chance to gain (\d+)% of non-chaos damage with hits as extra chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("NonChaosDamageGainAsChaos", "BASE", num/100*utils.Float(captures[1])),
		}, ""
	},
	"movement skills cost no mana": []mod.Mod{MOD("ManaCost", "MORE", -100).KeywordFlag(mod.KeywordFlagMovement)},
	"cannot be stunned while you have ghost shrouds": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AvoidStun", "BASE", 100).Tag(mod.MultiplierThreshold("GhostShroud").Threshold(1)),
		}, ""
	},
	// Item local modifiers
	"has no sockets": []mod.Mod{mod.NewFlag("NoSockets", true)},
	`has (\d+) sockets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SocketCount", mod.TypeBase, num)}, ""
	},
	`has (\d+) abyssal sockets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("AbyssalSocketCount", mod.TypeBase, num)}, ""
	},
	"no physical damage": []mod.Mod{
		MOD("WeaponData", "LIST", mod.WeaponData{Key: "PhysicalMin"}),
		MOD("WeaponData", "LIST", mod.WeaponData{Key: "PhysicalMax"}),
		MOD("WeaponData", "LIST", mod.WeaponData{Key: "PhysicalDPS"}),
	},
	"all attacks with this weapon are critical strikes": []mod.Mod{
		MOD("WeaponData", "LIST", mod.WeaponData{Key: "CritChance", Value: 100}),
	},
	`this weapon's critical strike chance is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("WeaponData", "LIST", mod.WeaponData{Key: "CritChance", Value: num})}, ""
	},
	"counts as dual wielding":                     []mod.Mod{MOD("WeaponData", "LIST", mod.WeaponData{Key: "countsAsDualWielding", Value: 1})},
	"counts as all one handed melee weapon types": []mod.Mod{MOD("WeaponData", "LIST", mod.WeaponData{Key: "countsAsAll1H", Value: 1})},
	"no block chance":                             []mod.Mod{MOD("ArmourData", "LIST", mod.ArmourData{Key: "BlockChance", Value: 0})},
	"has no energy shield":                        []mod.Mod{MOD("ArmourData", "LIST", mod.ArmourData{Key: "EnergyShield", Value: 0})},
	"hits can't be evaded":                        []mod.Mod{FLAG("CannotBeEvaded").Tag(mod.Condition("{Hand}Attack"))},
	"causes bleeding on hit":                      []mod.Mod{MOD("BleedChance", "BASE", 100).Tag(mod.Condition("{Hand}Attack"))},
	"poisonous hit":                               []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.Condition("{Hand}Attack"))},
	"attacks with this weapon deal double damage": []mod.Mod{MOD("DoubleDamageChance", "BASE", 100).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack)))},
	`hits with this weapon gain (\d+)% of physical damage as extra cold or lightning damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalDamageGainAsColdOrLightning", "BASE", num/2).Flag(mod.MFlagHit).Tag(mod.Condition("DualWielding")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
			MOD("PhysicalDamageGainAsColdOrLightning", "BASE", num).Flag(mod.MFlagHit).Tag(mod.Condition("DualWielding").Neg(true)).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`hits with this weapon shock enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ShockAsThoughDealing", "MORE", num).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`hits with this weapon freeze enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FreezeAsThoughDealing", "MORE", num).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`ignites inflicted with this weapon deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num).KeywordFlag(mod.KeywordFlagIgnite).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	"hits with this weapon always ignite, freeze, and shock": []mod.Mod{
		MOD("EnemyIgniteChance", "BASE", 100).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		MOD("EnemyFreezeChance", "BASE", 100).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		MOD("EnemyShockChance", "BASE", 100).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
	},
	"attacks with this weapon deal double damage to chilled enemies": []mod.Mod{
		MOD("DoubleDamageChance", "BASE", 100).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))).Tag(mod.ActorCondition("enemy", "Chilled")),
	},
	"life leech from hits with this weapon applies instantly":   []mod.Mod{FLAG("InstantLifeLeech").Tag(mod.Condition("{Hand}Attack"))},
	"life leech from hits with this weapon is instant":          []mod.Mod{FLAG("InstantLifeLeech").Tag(mod.Condition("{Hand}Attack"))},
	"gain life from leech instantly from hits with this weapon": []mod.Mod{FLAG("InstantLifeLeech").Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack)))},
	"instant recovery": []mod.Mod{mod.NewFloat("FlaskInstantRecovery", mod.TypeBase, 100)},
	`(\d+)% of recovery applied instantly`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("FlaskInstantRecovery", mod.TypeBase, num)}, ""
	},
	"has no attribute requirements":                                                                    []mod.Mod{mod.NewFlag("NoAttributeRequirements", true)},
	"trigger a socketed spell when you attack with this weapon":                                        []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed spell when you attack with this weapon, with a ([\d\.]+) second cooldown`:      []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"trigger a socketed spell when you use a skill":                                                    []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnSkillUse", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed spell when you use a skill, with a (\d+) second cooldown`:                      []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnSkillUse", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed spell when you use a skill, with a (\d+) second cooldown and (\d+)% more cost`: []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnSkillUse", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"trigger socketed spells when you focus":                                                           []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellFromHelmet", Level: 1}).Tag(mod.SocketedIn("{SlotName}")).Tag(mod.Condition("Focused"))},
	`trigger socketed spells when you focus, with a ([\d\.]+) second cooldown`:                         []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellFromHelmet", Level: 1}).Tag(mod.SocketedIn("{SlotName}")).Tag(mod.Condition("Focused"))},
	"trigger a socketed spell when you attack with a bow":                                              []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnBowAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed spell when you attack with a bow, with a ([\d\.]+) second cooldown`:            []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnBowAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"trigger a socketed bow skill when you attack with a bow":                                          []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerBowSkillOnBowAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed bow skill when you attack with a bow, with a ([\d\.]+) second cooldown`:        []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerBowSkillOnBowAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`(\d+)% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least (\d+) mana to use a skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("KitavaTriggerChance", "BASE", num).Source("Kitava's Thirst"),
			MOD("KitavaRequiredManaCost", "BASE", utils.Float(captures[1])).Source("Kitava's Thirst"),
			MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportCastOnManaSpent", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
		}, ""
	},
	// Socketed gem modifiers
	`([\+\-]\d+) to level of socketed gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("all"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`([\+\-]\d+) to level of socketed ([\w ]+) gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr(captures[1]), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`\+(\d+)% to quality of socketed gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("all"), Key: "quality", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`\+(\d+)% to quality of all skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("active_skill"), Key: "quality", Value: num})}, ""
	},
	`\+(\d+)% to quality of socketed ([\w ]+) gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr(captures[1]), Key: "quality", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`\+(\d+) to level of active socketed skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("active_skill"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`\+(\d+) to level of socketed active skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("active_skill"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`\+(\d+) to level of socketed active skill gems per (\d+) player levels`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("active_skill"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}")).Tag(mod.Multiplier("Level").Div(utils.Float(captures[1])))}, ""
	},
	"socketed gems fire an additional projectile": []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("ProjectileCount", mod.TypeBase, 1)}).Tag(mod.SocketedIn("{SlotName}"))},
	`socketed gems fire (\d+) additional projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("ProjectileCount", mod.TypeBase, num)}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	"socketed gems reserve no mana":     []mod.Mod{MOD("ManaReserved", "MORE", -100).Tag(mod.SocketedIn("{SlotName}"))},
	"socketed gems have no reservation": []mod.Mod{MOD("Reserved", "MORE", -100).Tag(mod.SocketedIn("{SlotName}"))},
	`socketed skill gems get a (\d+)% mana multiplier`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SupportManaMultiplier", "MORE", num-100)}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`socketed skill gems get a (\d+)% cost & reservation multiplier`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SupportManaMultiplier", "MORE", num-100)}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	"socketed gems have blood magic":                      []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportBloodMagicUniquePrismGuardian", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"socketed gems cost and reserve life instead of mana": []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportBloodMagicUniquePrismGuardian", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"socketed gems have elemental equilibrium":            []mod.Mod{MOD("Keystone", "LIST", "Elemental Equilibrium")},
	"socketed gems have secrets of suffering": []mod.Mod{
		FLAG("CannotIgnite").Tag(mod.SocketedIn("{SlotName}")),
		FLAG("CannotChill").Tag(mod.SocketedIn("{SlotName}")),
		FLAG("CannotFreeze").Tag(mod.SocketedIn("{SlotName}")),
		FLAG("CannotShock").Tag(mod.SocketedIn("{SlotName}")),
		FLAG("CritAlwaysAltAilments").Tag(mod.SocketedIn("{SlotName}")),
	},
	"socketed skills deal double damage": []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("DoubleDamageChance", mod.TypeBase, 100)}).Tag(mod.SocketedIn("{SlotName}"))},
	`socketed gems gain (\d+)% of physical damage as extra lightning damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("PhysicalDamageGainAsLightning", mod.TypeBase, num)}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`socketed red gems get (\d+)% physical damage as extra fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("PhysicalDamageGainAsFire", mod.TypeBase, num)}).Tag(mod.SocketedIn("{SlotName}").Keyword("strength"))}, ""
	},
	"socketed non-channelling bow skills are triggered by snipe": []mod.Mod{
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFlag("TriggeredBySnipe", true)}).Tag(mod.SocketedIn("{SlotName}").Keyword("bow"), mod.SkillType(string(data.SkillTypeTriggerable))),
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", mod.SkillData{Key: "showAverage", Value: 1})}).Tag(mod.SocketedIn("{SlotName}").Keyword("bow"), mod.SkillType(string(data.SkillTypeTriggerable))),
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", mod.SkillData{Key: "triggered", Value: 1})}).Tag(mod.SocketedIn("{SlotName}").Keyword("bow"), mod.SkillType(string(data.SkillTypeTriggerable))),
	},
	`socketed triggered bow skills deal (\d+)% less damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("Damage", "MORE", -num)}).Tag(mod.SocketedIn("{SlotName}").Keyword("bow"), mod.SkillType(string(data.SkillTypeTriggerable)))}, ""
	},
	`socketed travel skills deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("Damage", mod.TypeMore, num)}).Tag(mod.SocketedIn("{SlotName}"), mod.SkillType(string(data.SkillTypeTravel)))}, ""
	},
	`socketed warcry skills have \+(\d+) cooldown use`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("AdditionalCooldownUses", mod.TypeBase, num)}).Tag(mod.SocketedIn("{SlotName}"), mod.SkillType(string(data.SkillTypeWarcry)))}, ""
	},
	// Global gem modifiers
	`\+(\d+) to level of all minion skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"minion", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all spell skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"spell", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all physical spell skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"spell", "physical", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all physical skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"physical", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all lightning spell skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"spell", "lightning", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all lightning skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"lightning", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all cold spell skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"spell", "cold", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all cold skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"cold", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all fire spell skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"spell", "fire", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all fire skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"fire", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all chaos spell skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"spell", "chaos", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all chaos skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"chaos", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all strength skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"strength", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all dexterity skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"dexterity", "active_skill"}, Key: "level", Value: num})}, ""
	},
	`\+(\d+) to level of all intelligence skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"intelligence", "active_skill"}, Key: "level", Value: num})}, ""
	},
	/*
		TODO To level of all
		`\+(\d+) to level of all (.+) gems`: function(num, _, skill)
			if gemIdLookup[skill] then
				return { MOD("GemProperty", "LIST", mod.GemProperty{Keyword: skill, Key: "level", Value: num }) }
			end
			local wordList = {}
			for tag in skill:gmatch("[\w\s\d]+") do
				if tag == "skill" then
					tag: "active_skill"
				end
				table.insert(wordList, tag)
			end
			return { MOD("GemProperty", "LIST", mod.GemProperty{keywordList = wordList, Key: "level", Value: num }) }
		end,
	*/
	// Extra skill/support
	`grants (\D+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return grantedExtraSkill(captures[0], 1, false), ""
	},
	`grants level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
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
	`[ct][ar][si][tg]g?e?r?s? level (\d+) (.+) when you deal a critical strike`: func(num float64, captures []string) ([]mod.Mod, string) {
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
		return triggerExtraSkill(captures[0], utils.Int(captures[1]), false, captures[2]), ""
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
		if captures[1] == "summon phantasm skill" {
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
	`\d+% chance to attack with level (\d+) (.+) on melee hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`\d+% chance to trigger level (\d+) (.+) when animated weapon kills an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`\d+% chance to trigger level (\d+) (.+) on melee hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`\d+% chance to trigger level (\d+) (.+) [ow][nh]e?n? ?y?o?u? kill ?a?n? ?e?n?e?m?y?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`\d+% chance to trigger level (\d+) (.+) when you use a socketed skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`\d+% chance to trigger level (\d+) (.+) when you gain avian's might or avian's flight`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`\d+% chance to trigger level (\d+) (.+) on critical strike with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`\d+% chance to [ct][ar][si][tg]g?e?r? level (\d+) (.+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
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
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillName: captures[0], Level: 1, NoSupports: true, Triggered: true}),
			MOD("CurseEffect", "INC", utils.Float(captures[1])).Tag(mod.SkillName(utils.CapitalEach(captures[0]))),
		}, ""
	},
	`\d+% chance to curse n?o?n?-?c?u?r?s?e?d? ?enemies with (\D+) on \w+, with (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillName: captures[0], Level: 1, NoSupports: true, Triggered: true}),
			MOD("CurseEffect", "INC", utils.Float(captures[1])).Tag(mod.SkillName(utils.CapitalEach(captures[0]))),
		}, ""
	},
	`curse enemies with level (\d+) (\D+) on \w+, which can apply to hexproof enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), true, ""), ""
	},
	`curse enemies with level (\d+) (.+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), true, ""), ""
	},
	`[ct][ar][si][tg]g?e?r?s? (.+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	`[at][tr][ti][ag][cg][ke]r? (.+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	`[at][tr][ti][ag][cg][ke]r? with (.+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	"[ct][ar][si][tg]g?e?r?s? (.+) when hit": func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	"[at][tr][ti][ag][cg][ke]r? (.+) when hit": func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	"[at][tr][ti][ag][cg][ke]r? with (.+) when hit": func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	"[ct][ar][si][tg]g?e?r?s? (.+) when your skills or minions kill": func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	"[at][tr][ti][ag][cg][ke]r? (.+) when you take a critical strike": func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	"[at][tr][ti][ag][cg][ke]r? with (.+) when you take a critical strike": func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	"trigger commandment of inferno on critical strike": []mod.Mod{MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: "UniqueEnchantmentOfInfernoOnCrit", Level: 1, NoSupports: true, Triggered: true})},
	"trigger (.+) on critical strike": func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	"triggers? (.+) when you take a critical strike": func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	/*
		TODO Socketed gems
		`socketed [\w+]* ?gems a?r?e? ?supported by level (\d+) (.+)`: function(num, _, support)
			local SkillID: gemIdLookup[support] or gemIdLookup[support:gsub("^increased ","")]
			if skillId then
				local gemId = data.gemForBaseName[data.skills[skillId].name .. " Support"]
				if gemId then
					return {
						MOD("ExtraSupport", "LIST", mod.ExtraSupport{ SkillID: data.gems[gemId].grantedEffectId, Level:  num }).Tag(mod.SocketedIn("{SlotName}")),
						MOD("ExtraSupport", "LIST", mod.ExtraSupport{ SkillID: data.gems[gemId].secondaryGrantedEffectId, Level:  num }).Tag(mod.SocketedIn("{SlotName}"))
					}
				else
					return {
						MOD("ExtraSupport", "LIST", mod.ExtraSupport{ SkillID: skillId, Level:  num }).Tag(mod.SocketedIn("{SlotName}")),
					}
				end
			end
		end,
	*/
	`socketed support gems can also support skills from your ([\w\s]+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		targetItemSlotName := "Body Armour"
		if captures[0] == "main hand" {
			targetItemSlotName = "Weapon 1"
		}
		return []mod.Mod{
			MOD("LinkedSupport", "LIST", mod.LinkedSupport{TargetSlotName: targetItemSlotName}).Tag(mod.SocketedIn("{SlotName}")),
		}, ""
	},
	"socketed hex curse skills are triggered by doedre's effigy when summoned": []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportCursePillarTriggerCurses", Level: 20}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger level (\d+) (.+) every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+), (.+) or (.+) every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillName: captures[1], Level: int(num), Triggered: true}),
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillName: captures[2], Level: int(num), Triggered: true}),
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillName: captures[3], Level: int(num), Triggered: true}),
		}, ""
	},
	"offering skills triggered this way also affect you": []mod.Mod{
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", mod.SkillData{Key: "buffNotPlayer", Value: 0})}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering")).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger level (\d+) (.+) after spending a total of (\d+) mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`consumes a void charge to trigger level (\d+) (.+) when you fire arrows`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`consumes a void charge to trigger level (\d+) (.+) when you fire arrows with a non-triggered skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`your hits treat cold resistance as (\d+)% higher than actual value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ColdPenetration", "BASE", -num).KeywordFlag(mod.KeywordFlagHit),
		}, ""
	},
	// Conversion
	"increases and reductions to minion damage also affects? you": []mod.Mod{mod.NewFlag("MinionDamageAppliesToPlayer", true), MOD("ImprovedMinionDamageAppliesToPlayer", "MAX", 100)},
	`increases and reductions to minion damage also affects? you at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("MinionDamageAppliesToPlayer", true), MOD("ImprovedMinionDamageAppliesToPlayer", "MAX", num)}, ""
	},
	"increases and reductions to minion attack speed also affects? you": []mod.Mod{mod.NewFlag("MinionAttackSpeedAppliesToPlayer", true), MOD("ImprovedMinionAttackSpeedAppliesToPlayer", "MAX", 100)},
	`increases and reductions to cast speed apply to attack speed at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("CastSpeedAppliesToAttacks", true), MOD("ImprovedCastSpeedAppliesToAttacks", "MAX", num)}, ""
	},
	"increases and reductions to spell damage also apply to attacks": []mod.Mod{mod.NewFlag("SpellDamageAppliesToAttacks", true), MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", 100)},
	`increases and reductions to spell damage also apply to attacks at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("SpellDamageAppliesToAttacks", true), MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", num)}, ""
	},
	"increases and reductions to spell damage also apply to attacks while wielding a wand": []mod.Mod{FLAG("SpellDamageAppliesToAttacks").Tag(mod.Condition("UsingWand")), MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", 100).Tag(mod.Condition("UsingWand"))},
	`increases and reductions to maximum mana also apply to shock effect at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("ManaAppliesToShockEffect", true), MOD("ImprovedManaAppliesToShockEffect", "MAX", num)}, ""
	},
	"modifiers to claw damage also apply to unarmed":                                                          []mod.Mod{mod.NewFlag("ClawDamageAppliesToUnarmed", true)},
	"modifiers to claw damage also apply to unarmed attack damage":                                            []mod.Mod{mod.NewFlag("ClawDamageAppliesToUnarmed", true)},
	"modifiers to claw damage also apply to unarmed attack damage with melee skills":                          []mod.Mod{mod.NewFlag("ClawDamageAppliesToUnarmed", true)},
	"modifiers to claw attack speed also apply to unarmed":                                                    []mod.Mod{mod.NewFlag("ClawAttackSpeedAppliesToUnarmed", true)},
	"modifiers to claw attack speed also apply to unarmed attack speed":                                       []mod.Mod{mod.NewFlag("ClawAttackSpeedAppliesToUnarmed", true)},
	"modifiers to claw attack speed also apply to unarmed attack speed with melee skills":                     []mod.Mod{mod.NewFlag("ClawAttackSpeedAppliesToUnarmed", true)},
	"modifiers to claw critical strike chance also apply to unarmed":                                          []mod.Mod{mod.NewFlag("ClawCritChanceAppliesToUnarmed", true)},
	"modifiers to claw critical strike chance also apply to unarmed attack critical strike chance":            []mod.Mod{mod.NewFlag("ClawCritChanceAppliesToUnarmed", true)},
	"modifiers to claw critical strike chance also apply to unarmed critical strike chance with melee skills": []mod.Mod{mod.NewFlag("ClawCritChanceAppliesToUnarmed", true)},
	"increases and reductions to light radius also apply to accuracy":                                         []mod.Mod{mod.NewFlag("LightRadiusAppliesToAccuracy", true)},
	"increases and reductions to light radius also apply to area of effect at 50% of their value":             []mod.Mod{mod.NewFlag("LightRadiusAppliesToAreaOfEffect", true)},
	"increases and reductions to light radius also apply to damage":                                           []mod.Mod{mod.NewFlag("LightRadiusAppliesToDamage", true)},
	"increases and reductions to cast speed also apply to trap throwing speed":                                []mod.Mod{mod.NewFlag("CastSpeedAppliesToTrapThrowingSpeed", true)},
	`increases and reductions to armour also apply to energy shield recharge rate at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("ArmourAppliesToEnergyShieldRecharge", true), MOD("ImprovedArmourAppliesToEnergyShieldRecharge", "MAX", num)}, ""
	},
	"increases and reductions to projectile speed also apply to damage with bows": []mod.Mod{mod.NewFlag("ProjectileSpeedAppliesToBowDamage", true)},
	`gain (\d+)% of bow physical damage as extra damage of each element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalDamageGainAsLightning", "BASE", num).Flag(mod.MFlagBow),
			MOD("PhysicalDamageGainAsCold", "BASE", num).Flag(mod.MFlagBow),
			MOD("PhysicalDamageGainAsFire", "BASE", num).Flag(mod.MFlagBow),
		}, ""
	},
	`gain (\d+)% of weapon physical damage as extra damage of each element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalDamageGainAsLightning", "BASE", num).Flag(mod.MFlagWeapon),
			MOD("PhysicalDamageGainAsCold", "BASE", num).Flag(mod.MFlagWeapon),
			MOD("PhysicalDamageGainAsFire", "BASE", num).Flag(mod.MFlagWeapon),
		}, ""
	},
	`gain (\d+)% of weapon physical damage as extra damage of an? r?a?n?d?o?m? ?element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", num).Flag(mod.MFlagWeapon)}, ""
	},
	`gain (\d+)% of physical damage as extra damage of a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", num)}, ""
	},
	`gain (\d+)% of physical damage as extra damage of a random element while you are ignited`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalDamageGainAsRandom", "BASE", num).Tag(mod.Condition("Ignited")),
		}, ""
	},
	`(\d+)% of physical damage from hits with this weapon is converted to a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageConvertToRandom", "BASE", num)}, ""
	},
	// Crit
	"your critical strike chance is lucky":                                   []mod.Mod{mod.NewFlag("CritChanceLucky", true)},
	"your critical strike chance is lucky while on low life":                 []mod.Mod{FLAG("CritChanceLucky").Tag(mod.Condition("LowLife"))},
	"your critical strike chance is lucky while focus?sed":                   []mod.Mod{FLAG("CritChanceLucky").Tag(mod.Condition("Focused"))},
	"your critical strikes do not deal extra damage":                         []mod.Mod{mod.NewFlag("NoCritMultiplier", true)},
	"lightning damage with non-critical strikes is lucky":                    []mod.Mod{mod.NewFlag("LightningNoCritLucky", true)},
	"your damage with critical strikes is lucky":                             []mod.Mod{mod.NewFlag("CritLucky", true)},
	"critical strikes deal no damage":                                        []mod.Mod{MOD("Damage", "MORE", -100).Tag(mod.Condition("CriticalStrike"))},
	"critical strike chance is increased by uncapped lightning resistance":   []mod.Mod{MOD("CritChance", "INC", 1).Tag(mod.PerStat(1, "LightningResistTotal"))},
	"critical strike chance is increased by lightning resistance":            []mod.Mod{MOD("CritChance", "INC", 1).Tag(mod.PerStat(1, "LightningResist"))},
	"critical strike chance is increased by overcapped lightning resistance": []mod.Mod{MOD("CritChance", "INC", 1).Tag(mod.PerStat(1, "LightningResistOverCap"))},
	`non-critical strikes deal (\d+)% damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", -100+num).Flag(mod.MFlagHit).Tag(mod.Condition("CriticalStrike").Neg(true)),
		}, ""
	},
	`critical strikes penetrate (\d+)% of enemy elemental resistances while affected by zealotry`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ElementalPenetration", "BASE", num).Tag(mod.Condition("CriticalStrike")).Tag(mod.Condition("AffectedByZealotry")),
		}, ""
	},
	"attack critical strikes ignore enemy monster elemental resistances": []mod.Mod{
		FLAG("IgnoreElementalResistances").Tag(mod.Condition("CriticalStrike")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
	},
	// Generic Ailments
	`enemies take (\d+)% increased damage for each type of ailment you have inflicted on them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Frozen")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Chilled")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Ignited")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Shocked")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Scorched")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Brittle")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Sapped")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Bleeding")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Poisoned")),
		}, ""
	},
	// Elemental Ailments
	`your shocks can increase damage taken by up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ShockMax", mod.TypeOverride, num)}, ""
	},
	"your elemental damage can shock":                                 []mod.Mod{mod.NewFlag("ColdCanShock", true), mod.NewFlag("FireCanShock", true)},
	"all your damage can freeze":                                      []mod.Mod{mod.NewFlag("PhysicalCanFreeze", true), mod.NewFlag("LightningCanFreeze", true), mod.NewFlag("FireCanFreeze", true), mod.NewFlag("ChaosCanFreeze", true)},
	"all damage with maces and sceptres inflicts chill":               []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Chilled", true)}).Tag(mod.Condition("UsingMace"))},
	"your cold damage can ignite":                                     []mod.Mod{mod.NewFlag("ColdCanIgnite", true)},
	"your lightning damage can ignite":                                []mod.Mod{mod.NewFlag("LightningCanIgnite", true)},
	"your fire damage can shock but not ignite":                       []mod.Mod{mod.NewFlag("FireCanShock", true), mod.NewFlag("FireCannotIgnite", true)},
	"your cold damage can ignite but not freeze or chill":             []mod.Mod{mod.NewFlag("ColdCanIgnite", true), mod.NewFlag("ColdCannotFreeze", true), mod.NewFlag("ColdCannotChill", true)},
	"your cold damage cannot freeze":                                  []mod.Mod{mod.NewFlag("ColdCannotFreeze", true)},
	"your lightning damage can freeze but not shock":                  []mod.Mod{mod.NewFlag("LightningCanFreeze", true), mod.NewFlag("LightningCannotShock", true)},
	"your chaos damage can shock":                                     []mod.Mod{mod.NewFlag("ChaosCanShock", true)},
	"your chaos damage can chill":                                     []mod.Mod{mod.NewFlag("ChaosCanChill", true)},
	"your chaos damage can ignite":                                    []mod.Mod{mod.NewFlag("ChaosCanIgnite", true)},
	"chaos damage can ignite, chill and shock":                        []mod.Mod{mod.NewFlag("ChaosCanIgnite", true), mod.NewFlag("ChaosCanChill", true), mod.NewFlag("ChaosCanShock", true)},
	"your physical damage can chill":                                  []mod.Mod{mod.NewFlag("PhysicalCanChill", true)},
	"your physical damage can shock":                                  []mod.Mod{mod.NewFlag("PhysicalCanShock", true)},
	"your physical damage can freeze":                                 []mod.Mod{mod.NewFlag("PhysicalCanFreeze", true)},
	"you always ignite while burning":                                 []mod.Mod{MOD("EnemyIgniteChance", "BASE", 100).Tag(mod.Condition("Burning"))},
	"critical strikes do not a?l?w?a?y?s?i?n?h?e?r?e?n?t?l?y? freeze": []mod.Mod{mod.NewFlag("CritsDontAlwaysFreeze", true)},
	"cannot inflict elemental ailments": []mod.Mod{
		mod.NewFlag("CannotIgnite", true),
		mod.NewFlag("CannotChill", true),
		mod.NewFlag("CannotFreeze", true),
		mod.NewFlag("CannotShock", true),
		mod.NewFlag("CannotScorch", true),
		mod.NewFlag("CannotBrittle", true),
		mod.NewFlag("CannotSap", true),
	},
	`you can inflict up to (\d+) ignites on an enemy`:  []mod.Mod{mod.NewFlag("IgniteCanStack", true)},
	"you can inflict an additional ignite on an enemy": []mod.Mod{mod.NewFlag("IgniteCanStack", true), mod.NewFloat("IgniteStacks", mod.TypeBase, 1)},
	`enemies chilled by you take (\d+)% increased burning damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("FireDamageTakenOverTime", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Chilled")),
		}, ""
	},
	`damaging ailments deal damage (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("IgniteBurnFaster", mod.TypeIncrease, num), mod.NewFloat("BleedFaster", mod.TypeIncrease, num), mod.NewFloat("PoisonFaster", mod.TypeIncrease, num)}, ""
	},
	`damaging ailments you inflict deal damage (\d+)% faster while affected by malevolence`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("IgniteBurnFaster", "INC", num).Tag(mod.Condition("AffectedByMalevolence")),
			MOD("BleedFaster", "INC", num).Tag(mod.Condition("AffectedByMalevolence")),
			MOD("PoisonFaster", "INC", num).Tag(mod.Condition("AffectedByMalevolence")),
		}, ""
	},
	`ignited enemies burn (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("IgniteBurnFaster", mod.TypeIncrease, num)}, ""
	},
	`ignited enemies burn (\d+)% slower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("IgniteBurnSlower", mod.TypeIncrease, num)}, ""
	},
	`enemies ignited by an attack burn (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("IgniteBurnFaster", "INC", num).Flag(mod.MFlagAttack)}, ""
	},
	`ignites you inflict with attacks deal damage (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("IgniteBurnFaster", "INC", num).Flag(mod.MFlagAttack)}, ""
	},
	`ignites you inflict deal damage (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("IgniteBurnFaster", mod.TypeIncrease, num)}, ""
	},
	`enemies ignited by you during flask effect take (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Ignited")),
		}, ""
	},
	"enemies ignited by you take chaos damage instead of fire damage from ignite": []mod.Mod{mod.NewFlag("IgniteToChaos", true)},
	"enemies chilled by your hits are shocked": []mod.Mod{
		MOD("ShockBase", "BASE", data.NonDamagingAilments[data.AilmentShock].Default).Tag(mod.ActorCondition("enemy", "ChilledByYourHits")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Shocked").Tag(mod.Condition("ChilledByYourHits"))}),
	},
	"cannot inflict ignite":                 []mod.Mod{mod.NewFlag("CannotIgnite", true)},
	"cannot inflict freeze or chill":        []mod.Mod{mod.NewFlag("CannotFreeze", true), mod.NewFlag("CannotChill", true)},
	"cannot inflict shock":                  []mod.Mod{mod.NewFlag("CannotShock", true)},
	"cannot ignite, chill, freeze or shock": []mod.Mod{mod.NewFlag("CannotIgnite", true), mod.NewFlag("CannotChill", true), mod.NewFlag("CannotFreeze", true), mod.NewFlag("CannotShock", true)},
	`shock enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ShockAsThoughDealing", mod.TypeMore, num)}, ""
	},
	`inflict non-damaging ailments as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("ShockAsThoughDealing", mod.TypeMore, num),
			mod.NewFloat("ChillAsThoughDealing", mod.TypeMore, num),
			mod.NewFloat("FreezeAsThoughDealing", mod.TypeMore, num),
			mod.NewFloat("ScorchAsThoughDealing", mod.TypeMore, num),
			mod.NewFloat("BrittleAsThoughDealing", mod.TypeMore, num),
			mod.NewFloat("SapAsThoughDealing", mod.TypeMore, num),
		}, ""
	},

	`immune to elemental ailments while on consecrated ground if you have at least (\d+) devotion`: func(num float64, captures []string) ([]mod.Mod, string) {
		mods := make([]mod.Mod, len(data.Ailment("").Values()))
		for i, ailment := range data.Ailment("").Values() {
			mods[i] = MOD("Avoid"+string(ailment), "BASE", 100).Tag(mod.Condition("OnConsecratedGround")).Tag(mod.StatThreshold("Devotion", num))
		}
		return mods, ""
	},
	`freeze chilled enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FreezeAsThoughDealing", "MORE", num).Tag(mod.ActorCondition("enemy", "Chilled"))}, ""
	},
	`(\d+)% chance to shock attackers for (\d+) seconds on block`: []mod.Mod{MOD("ShockBase", "BASE", data.NonDamagingAilments[data.AilmentShock].Default)},
	`shock attackers for (\d+) seconds on block`: []mod.Mod{
		MOD("ShockBase", "BASE", data.NonDamagingAilments[data.AilmentShock].Default).Tag(mod.Condition("BlockedRecently")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Shocked", true)}).Tag(mod.Condition("BlockedRecently")),
	},
	`shock nearby enemies for (\d+) seconds when you focus`: []mod.Mod{
		MOD("ShockBase", "BASE", data.NonDamagingAilments[data.AilmentShock].Default).Tag(mod.Condition("Focused")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Shocked", true)}).Tag(mod.Condition("Focused")),
	},
	`drops shocked ground while moving, lasting (\d+) seconds`:  []mod.Mod{MOD("ShockBase", "BASE", data.NonDamagingAilments[data.AilmentShock].Default).Tag(mod.ActorCondition("enemy", "OnShockedGround"))},
	`drops scorched ground while moving, lasting (\d+) seconds`: []mod.Mod{MOD("ScorchBase", "BASE", data.NonDamagingAilments[data.AilmentScorch].Default).Tag(mod.ActorCondition("enemy", "OnScorchedGround"))},
	`drops brittle ground while moving, lasting (\d+) seconds`:  []mod.Mod{MOD("BrittleBase", "BASE", data.NonDamagingAilments[data.AilmentBrittle].Default).Tag(mod.ActorCondition("enemy", "OnBrittleGround"))},
	`drops sapped ground while moving, lasting (\d+) seconds`:   []mod.Mod{MOD("SapBase", "BASE", data.NonDamagingAilments[data.AilmentSap].Default).Tag(mod.ActorCondition("enemy", "OnSappedGround"))},
	`\+(\d+)% chance to ignite, freeze, shock, and poison cursed enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyIgniteChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed")),
			MOD("EnemyFreezeChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed")),
			MOD("EnemyShockChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed")),
			MOD("PoisonChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed")),
		}, ""
	},
	"you have scorching conflux, brittle conflux and sapping conflux while your two highest attributes are equal": []mod.Mod{
		MOD("EnemyScorchChance", "BASE", 100).Tag(mod.Condition("TwoHighestAttributesEqual")),
		MOD("EnemyBrittleChance", "BASE", 100).Tag(mod.Condition("TwoHighestAttributesEqual")),
		MOD("EnemySapChance", "BASE", 100).Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("PhysicalCanScorch").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("LightningCanScorch").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("ColdCanScorch").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("ChaosCanScorch").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("PhysicalCanBrittle").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("LightningCanBrittle").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("FireCanBrittle").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("ChaosCanBrittle").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("PhysicalCanSap").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("ColdCanSap").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("FireCanSap").Tag(mod.Condition("TwoHighestAttributesEqual")),
		FLAG("ChaosCanSap").Tag(mod.Condition("TwoHighestAttributesEqual")),
	},
	"critical strikes do not inherently apply non-damaging ailments": []mod.Mod{
		mod.NewFlag("CritsDontAlwaysChill", true),
		mod.NewFlag("CritsDontAlwaysFreeze", true),
		mod.NewFlag("CritsDontAlwaysShock", true),
	},
	"always scorch while affected by anger":           []mod.Mod{MOD("EnemyScorchChance", "BASE", 100).Tag(mod.Condition("AffectedByAnger"))},
	"always inflict brittle while affected by hatred": []mod.Mod{MOD("EnemyBrittleChance", "BASE", 100).Tag(mod.Condition("AffectedByHatred"))},
	"always sap while affected by wrath":              []mod.Mod{MOD("EnemySapChance", "BASE", 100).Tag(mod.Condition("AffectedByWrath"))},
	// Bleed
	"melee attacks cause bleeding":                       []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagMelee)},
	"attacks cause bleeding when hitting cursed enemies": []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagAttack).Tag(mod.ActorCondition("enemy", "Cursed"))},
	"melee critical strikes cause bleeding":              []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagMelee).Tag(mod.Condition("CriticalStrike"))},
	"causes bleeding on melee critical strike":           []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagMelee).Tag(mod.Condition("CriticalStrike"))},
	`melee critical strikes have (\d+)% chance to cause bleeding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("BleedChance", "BASE", num).Flag(mod.MFlagMelee).Tag(mod.Condition("CriticalStrike")),
		}, ""
	},
	"attacks always inflict bleeding while you have cat's stealth":        []mod.Mod{MOD("BleedChance", "BASE", 10).Flag(mod.MFlagAttack).Tag(mod.Condition("AffectedByCat'sStealth"))},
	"you have crimson dance while you have cat's stealth":                 []mod.Mod{MOD("Keystone", "LIST", "Crimson Dance").Tag(mod.Condition("AffectedByCat'sStealth"))},
	"you have crimson dance if you have dealt a critical strike recently": []mod.Mod{MOD("Keystone", "LIST", "Crimson Dance").Tag(mod.Condition("CritRecently"))},
	`bleeding you inflict deals damage (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("BleedFaster", mod.TypeIncrease, num)}, ""
	},
	`(\d+)% chance for bleeding inflicted with this weapon to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", utils.Float(captures[1])*num/100).KeywordFlag(mod.KeywordFlagBleed).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`bleeding you inflict deals damage (\d+)% faster per frenzy charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("BleedFaster", "INC", num).Tag(mod.Multiplier("FrenzyCharge").Base(0)),
		}, ""
	},
	// Impale and Bleed
	`(\d+)% increased effect of impales inflicted by hits that also inflict bleeding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ImpaleEffectOnBleed", "INC", num).KeywordFlag(mod.KeywordFlagHit),
		}, ""
	},
	// Poison and Bleed
	`(\d+)% increased damage with bleeding inflicted on poisoned enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "INC", num).KeywordFlag(mod.KeywordFlagBleed).Tag(mod.ActorCondition("enemy", "Poisoned")),
		}, ""
	},
	// Poison
	"y?o?u?r? ?fire damage can poison":      []mod.Mod{mod.NewFlag("FireCanPoison", true)},
	"y?o?u?r? ?cold damage can poison":      []mod.Mod{mod.NewFlag("ColdCanPoison", true)},
	"y?o?u?r? ?lightning damage can poison": []mod.Mod{mod.NewFlag("LightningCanPoison", true)},
	"all damage from hits with this weapon can poison": []mod.Mod{
		FLAG("FireCanPoison").Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		FLAG("ColdCanPoison").Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		FLAG("LightningCanPoison").Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
	},
	"all damage inflicts poison while affected by glorious madness": []mod.Mod{
		FLAG("FireCanPoison").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("ColdCanPoison").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("LightningCanPoison").Tag(mod.Condition("AffectedByGloriousMadness")),
	},
	"your chaos damage poisons enemies": []mod.Mod{mod.NewFloat("ChaosPoisonChance", mod.TypeBase, 100)},
	`your chaos damage has (\d+)% chance to poison enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ChaosPoisonChance", mod.TypeBase, num)}, ""
	},
	"melee attacks poison on hit": []mod.Mod{MOD("PoisonChance", "BASE", 100).Flag(mod.MFlagMelee)},
	`melee critical strikes have (\d+)% chance to poison the enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PoisonChance", "BASE", num).Flag(mod.MFlagMelee).Tag(mod.Condition("CriticalStrike"))}, ""
	},
	`critical strikes with daggers have a (\d+)% chance to poison the enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PoisonChance", "BASE", num).Flag(mod.MFlagDagger).Tag(mod.Condition("CriticalStrike"))}, ""
	},
	"critical strikes with daggers poison the enemy":                 []mod.Mod{MOD("PoisonChance", "BASE", 100).Flag(mod.MFlagDagger).Tag(mod.Condition("CriticalStrike"))},
	"poison cursed enemies on hit":                                   []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.ActorCondition("enemy", "Cursed"))},
	"wh[ie][ln]e? at maximum frenzy charges, attacks poison enemies": []mod.Mod{MOD("PoisonChance", "BASE", 100).Flag(mod.MFlagAttack).Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	`traps and mines have a (\d+)% chance to poison on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PoisonChance", "BASE", num).KeywordFlag(mod.KeywordFlagTrap).KeywordFlag(mod.KeywordFlagMine)}, ""
	},
	`poisons you inflict deal damage (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("PoisonFaster", mod.TypeIncrease, num)}, ""
	},
	`(\d+)% chance for poisons inflicted with this weapon to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", utils.Float(captures[1])*num/100).KeywordFlag(mod.KeywordFlagPoison).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	// Suppression
	"your chance to suppressed spell damage is lucky":   []mod.Mod{mod.NewFlag("SpellSuppressionChanceIsLucky", true)},
	"your chance to suppressed spell damage is unlucky": []mod.Mod{mod.NewFlag("SpellSuppressionChanceIsUnlucky", true)},
	`prevent \+(\+\d+)% of suppressed spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SpellSuppressionEffect", mod.TypeBase, num)}, ""
	},
	"critical strike chance is increased by chance to suppress spell damage": []mod.Mod{MOD("CritChance", "INC", 1).Tag(mod.PerStat(1, "SpellSuppressionChance"))},
	`you take (\d+)% reduced extra damage from suppressed critical strikes`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ReduceSuppressedCritExtraDamage", mod.TypeBase, num)}, ""
	},
	`+(\d+)% chance to suppress spell damage if your boots, helmet and gloves have evasion`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SpellSuppressionChance", "BASE", num).
				Tag(mod.StatThreshold("EvasionOnBoots", 1)).
				Tag(mod.StatThreshold("EvasionOnHelmet", 1).Upper(true)).
				Tag(mod.StatThreshold("EvasionOnGloves", 1).Upper(true)),
		}, ""
	},
	`+(\d+)% chance to suppress spell damage for each dagger you're wielding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SpellSuppressionChance", "BASE", num).Tag(mod.ModFlag(mod.MFlagDagger)),
			MOD("SpellSuppressionChance", "BASE", num).Tag(mod.Condition("DualWieldingDaggers")),
		}, ""
	},
	// Buffs/debuffs
	"phasing":                              []mod.Mod{mod.NewFlag("Condition:Phasing", true)},
	"onslaught":                            []mod.Mod{mod.NewFlag("Condition:Onslaught", true)},
	"unholy might":                         []mod.Mod{mod.NewFlag("Condition:UnholyMight", true)},
	"your aura buffs do not affect allies": []mod.Mod{mod.NewFlag("SelfAurasCannotAffectAllies", true)},
	"auras from your skills can only affect you": []mod.Mod{mod.NewFlag("SelfAurasOnlyAffectYou", true)},
	`aura buffs from skills have (\d+)% increased effect on you for each herald affecting you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillAuraEffectOnSelf", "INC", num).Tag(mod.Multiplier("Herald")),
		}, ""
	},
	`aura buffs from skills have (\d+)% increased effect on you for each herald affecting you, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillAuraEffectOnSelf", "INC", num).Tag(mod.Multiplier("Herald").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("PurposefulHarbinger")),
		}, ""
	},
	`(\d+)% increased area of effect per power charge, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AreaOfEffect", "INC", num).Tag(mod.Multiplier("PowerCharge").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("VastPower")),
		}, ""
	},
	`(\d+)% increased chaos damage per (\d+) maximum mana, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ChaosDamage", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "Mana").GlobalLimit(utils.Float(captures[2])).GlobalLimitKey("DarkIdeation")),
		}, ""
	},
	`minions have \+(\d+)% to damage over time multiplier per ghastly eye jewel affecting you, up to a maximum of \+(\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("DotMultiplier", "BASE", num).Tag(mod.Multiplier("GhastlyEyeJewel").Actor("parent").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("AmanamuGaze"))}),
		}, ""
	},
	`(\d+)% increased effect of arcane surge on you per hypnotic eye jewel affecting you, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ArcaneSurgeEffect", "INC", num).Tag(mod.Multiplier("HypnoticEyeJewel").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("KurgalGaze")),
		}, ""
	},
	`(\d+)% increased main hand critical strike chance per murderous eye jewel affecting you, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "INC", num).Tag(mod.Multiplier("MurderousEyeJewel").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("TecrodGazeMainHand")).Tag(mod.Condition("MainHandAttack")),
		}, ""
	},
	`\+(\d+)% to off hand critical strike multiplier per murderous eye jewel affecting you, up to a maximum of \+(\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritMultiplier", "BASE", num).Tag(mod.Multiplier("MurderousEyeJewel").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("TecrodGazeOffHand")).Tag(mod.Condition("OffHandAttack")),
		}, ""
	},
	"nearby allies' damage with hits is lucky":                  []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{OnlyAllies: true, Mod: mod.NewFlag("LuckyHits", true)})},
	"your damage with hits is lucky":                            []mod.Mod{mod.NewFlag("LuckyHits", true)},
	"elemental damage with hits is lucky while you are shocked": []mod.Mod{FLAG("ElementalLuckHits").Tag(mod.Condition("Shocked"))},
	"allies' aura buffs do not affect you":                      []mod.Mod{mod.NewFlag("AlliesAurasCannotAffectSelf", true)},
	`(\d+)% increased effect of non-curse auras from your skills on enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DebuffEffect", "INC", num).Tag(mod.SkillType(string(data.SkillTypeAura))).Tag(mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)),
			MOD("AuraEffect", "INC", num).Tag(mod.SkillName("Death Aura")),
		}, ""
	},
	"enemies can have 1 additional curse":                             []mod.Mod{mod.NewFloat("EnemyCurseLimit", mod.TypeBase, 1)},
	"you can apply an additional curse":                               []mod.Mod{mod.NewFloat("EnemyCurseLimit", mod.TypeBase, 1)},
	"you can apply an additional curse while affected by malevolence": []mod.Mod{MOD("EnemyCurseLimit", "BASE", 1).Tag(mod.Condition("AffectedByMalevolence"))},
	"you can apply one fewer curse":                                   []mod.Mod{MOD("EnemyCurseLimit", "BASE", -1)},
	`curses on enemies in your chilling areas have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CurseEffect", "INC", num).Tag(mod.ActorCondition("enemy", "InChillingArea")),
		}, ""
	},
	"hexes you inflict have their effect increased by twice their doom instead": []mod.Mod{mod.NewFloat("DoomEffect", mod.TypeMore, 100)},
	`nearby enemies have an additional (\d+)% chance to receive a critical strike`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("SelfExtraCritChance", mod.TypeBase, num)})}, ""
	},
	`nearby enemies have (-\d+)% to all resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("ElementalResist", mod.TypeBase, num)}),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("ChaosResist", mod.TypeBase, num)}),
		}, ""
	},
	`enemies ignited or chilled by you have (-\d+)% to elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ElementalResist", "BASE", num)}).Tag(mod.ActorCondition("enemy", "Ignited", "Chilled")),
		}, ""
	},
	`your hits inflict decay, dealing (\d+) chaos damage per second for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillData", "LIST", mod.SkillData{Key: "decay", Value: num, Merge: "MAX"}),
		}, ""
	},
	`temporal chains has (\d+)% reduced effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CurseEffectOnSelf", "INC", -num).Tag(mod.SkillName("Temporal Chains")),
		}, ""
	},
	"unaffected by temporal chains": []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.SkillName("Temporal Chains"))},
	`([\+\-][\d\.]+) seconds to cat's stealth duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PrimaryDuration", "BASE", num).Tag(mod.SkillName("Aspect of the Cat"))}, ""
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
	"aspect of the spider can inflict spider's web on enemies an additional time": []mod.Mod{
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("Multiplier:SpiderWebApplyStackMax", mod.TypeBase, 1)}).Tag(mod.SkillName("Aspect of the Spider")),
	},
	"aspect of the avian also grants avian's might and avian's flight to nearby allies": []mod.Mod{
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("BuffEffectOnMinion", mod.TypeMore, 100)}).Tag(mod.SkillName("Aspect of the Avian")),
	},
	`marked enemy takes (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "Marked")),
		}, ""
	},
	`marked enemy has (\d+)% reduced accuracy rating`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("Accuracy", "INC", -num)}).Tag(mod.ActorCondition("enemy", "Marked")),
		}, ""
	},
	`you are cursed with level (\d+) (\D+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraCurse", "LIST", mod.ExtraCurse{SkillName: captures[1], Level: int(num), ApplyToPlayer: true})}, ""
	},
	`you are cursed with (\D+), with (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraCurse", "LIST", mod.ExtraCurse{SkillName: captures[0], Level: 1, ApplyToPlayer: true}),
			MOD("CurseEffectOnSelf", "INC", utils.Float(captures[1])).Tag(mod.SkillName(utils.CapitalEach(captures[0]))),
		}, ""
	},
	"you count as on low life while you are cursed with vulnerability":  []mod.Mod{FLAG("Condition:LowLife").Tag(mod.Condition("AffectedByVulnerability"))},
	"you count as on full life while you are cursed with vulnerability": []mod.Mod{FLAG("Condition:FullLife").Tag(mod.Condition("AffectedByVulnerability"))},
	`if you consumed a corpse recently, you and nearby allies regenerate (\d+)% of life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("LifeRegenPercent", mod.TypeBase, num)}).Tag(mod.Condition("ConsumedCorpseRecently"))}, ""
	},
	`if you have blocked recently, you and nearby allies regenerate (\d+)% of life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("LifeRegenPercent", mod.TypeBase, num)}).Tag(mod.Condition("BlockedRecently"))}, ""
	},
	"you are at maximum chance to block attack damage if you have not blocked recently": []mod.Mod{FLAG("MaxBlockIfNotBlockedRecently").Tag(mod.Condition("BlockedRecently").Neg(true))},
	`\+(\d+)% chance to block attack damage if you have not blocked recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BlockChance", "BASE", num).Tag(mod.Condition("BlockedRecently").Neg(true))}, ""
	},
	`\+(\d+)% chance to block spell damage if you have not blocked recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellBlockChance", "BASE", num).Tag(mod.Condition("BlockedRecently").Neg(true))}, ""
	},
	`(\d+)% of evasion rating is regenerated as life per second while focus?sed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegen", "BASE", 1).Tag(mod.PercentStat("Evasion", num)).Tag(mod.Condition("Focused"))}, ""
	},
	`nearby allies have (\d+)% increased defences per (\d+) strength you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{OnlyAllies: true, Mod: mod.NewFloat("Defences", mod.TypeIncrease, num)}).Tag(mod.PerStat(utils.Float(captures[1]), "Str")),
		}, ""
	},
	`nearby allies have \+(\d+)% to critical strike multiplier per (\d+) dexterity you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{OnlyAllies: true, Mod: mod.NewFloat("CritMultiplier", mod.TypeBase, num)}).Tag(mod.PerStat(utils.Float(captures[1]), "Dex")),
		}, ""
	},
	`nearby allies have (\d+)% increased cast speed per (\d+) intelligence you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{OnlyAllies: true, Mod: MOD("Speed", "INC", num).Flag(mod.MFlagCast)}).Tag(mod.PerStat(utils.Float(captures[1]), "Int"))}, ""
	},
	`you gain divinity for \d+ seconds on reaching maximum divine charges`: []mod.Mod{
		MOD("ElementalDamage", "MORE", 50).Tag(mod.Condition("Divinity")),
		MOD("ElementalDamageTaken", "MORE", -20).Tag(mod.Condition("Divinity")),
	},
	"your maximum endurance charges is equal to your maximum frenzy charges": []mod.Mod{mod.NewFlag("MaximumEnduranceChargesIsMaximumFrenzyCharges", true)},
	"your maximum frenzy charges is equal to your maximum power charges":     []mod.Mod{mod.NewFlag("MaximumFrenzyChargesIsMaximumPowerCharges", true)},
	`consecrated ground you create while affected by zealotry causes enemies to take (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("DamageTakenConsecratedGround", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "OnConsecratedGround")).Tag(mod.Condition("AffectedByZealotry"))}, ""
	},
	`if you've warcried recently, you and nearby allies have (\d+)% increased attack, cast and movement speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("Speed", mod.TypeIncrease, num)}).Tag(mod.Condition("UsedWarcryRecently")),
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("MovementSpeed", mod.TypeIncrease, num)}).Tag(mod.Condition("UsedWarcryRecently")),
		}, ""
	},
	"when you warcry, you and nearby allies gain onslaught for 4 seconds": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFlag("Onslaught", true)}).Tag(mod.Condition("UsedWarcryRecently"))},
	`enemies in your chilling areas take (\d+)% increased lightning damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("LightningDamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "InChillingArea")),
		}, ""
	},
	`(\d+)% chance to sap enemies in chilling areas`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemySapChance", "BASE", num).Tag(mod.ActorCondition("enemy", "InChillingArea"))}, ""
	},
	`warcries count as having (\d+) additional nearby enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("Multiplier:WarcryNearbyEnemies", mod.TypeBase, num),
		}, ""
	},
	`enemies taunted by your warcries take (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Condition("Taunted"))}).Tag(mod.Condition("UsedWarcryRecently")),
		}, ""
	},
	"warcries share their cooldown":                                                                                  []mod.Mod{mod.NewFlag("WarcryShareCooldown", true)},
	`warcries have minimum of (\d+) power`:                                                                           []mod.Mod{mod.NewFlag("CryWolfMinimumPower", true)},
	"warcries have infinite power":                                                                                   []mod.Mod{mod.NewFlag("WarcryInfinitePower", true)},
	`(\d+)% chance to inflict corrosion on hit with attacks`:                                                         []mod.Mod{mod.NewFlag("Condition:CanCorrode", true)},
	`(\d+)% chance to inflict withered for (\d+) seconds on hit`:                                                     []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	`(\d+)% chance to inflict withered for (\d+) seconds on hit with this weapon`:                                    []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	`(\d+)% chance to inflict withered for two seconds on hit if there are (\d+) or fewer withered debuffs on enemy`: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	`inflict withered for (\d+) seconds on hit with this weapon`:                                                     []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	`enemies take (\d+)% increased elemental damage from your hits for each withered you have inflicted on them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ElementalDamageTaken", "INC", num).Tag(mod.Multiplier("WitheredStack").Base(0))}),
		}, ""
	},
	"your hits cannot penetrate or ignore elemental resistances":                             []mod.Mod{mod.NewFlag("CannotElePenIgnore", true)},
	"nearby enemies have malediction":                                                        []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("HasMalediction", true)})},
	"elemental damage you deal with hits is resisted by lowest elemental resistance instead": []mod.Mod{mod.NewFlag("ElementalDamageUsesLowestResistance", true)},
	`you take (\d+) chaos damage per second for 3 seconds on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosDegen", "BASE", num).Tag(mod.Condition("KilledLast3Seconds"))}, ""
	},
	`regenerate (\d+) life over 1 second for each spell you cast`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegen", "BASE", num).Tag(mod.Condition("CastLast1Seconds"))}, ""
	},
	`and nearby allies regenerate (\d+) life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegen", "BASE", num).Tag(mod.Condition("KilledPosionedLast2Seconds"))}, ""
	},
	`(\d+)% increased life regeneration rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeIncrease, num)}, ""
	},
	`fire skills have a (\d+)% chance to apply fire exposure on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("FireExposureChance", mod.TypeBase, num)}, ""
	},
	`cold skills have a (\d+)% chance to apply cold exposure on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ColdExposureChance", mod.TypeBase, num)}, ""
	},
	`lightning skills have a (\d+)% chance to apply lightning exposure on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("LightningExposureChance", mod.TypeBase, num)}, ""
	},
	"socketed skills apply fire, cold and lightning exposure on hit": []mod.Mod{
		MOD("FireExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
		MOD("ColdExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
		MOD("LightningExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
	},
	"nearby enemies have fire exposure": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", -10)}).Tag(mod.Condition("Effective")),
	},
	"nearby enemies have cold exposure": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", -10)}).Tag(mod.Condition("Effective")),
	},
	"nearby enemies have lightning exposure": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", -10)}).Tag(mod.Condition("Effective")),
	},
	"nearby enemies have fire exposure while you are affected by herald of ash": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", -10)}).Tag(mod.Condition("Effective")).Tag(mod.Condition("AffectedByHeraldofAsh")),
	},
	"nearby enemies have cold exposure while you are affected by herald of ice": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", -10)}).Tag(mod.Condition("Effective")).Tag(mod.Condition("AffectedByHeraldofIce")),
	},
	"nearby enemies have lightning exposure while you are affected by herald of thunder": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", -10)}).Tag(mod.Condition("Effective")).Tag(mod.Condition("AffectedByHeraldofThunder")),
	},
	"inflict fire, cold and lightning exposure on nearby enemies when used": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", -10)}).Tag(mod.Condition("Effective")).Tag(mod.Condition("UsingFlask")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", -10)}).Tag(mod.Condition("Effective")).Tag(mod.Condition("UsingFlask")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", -10)}).Tag(mod.Condition("Effective")).Tag(mod.Condition("UsingFlask")),
	},
	"enemies near your linked targets have fire, cold and lightning exposure": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", -10).Tag(mod.Condition("NearLinkedTarget"))}).Tag(mod.Condition("Effective")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", -10).Tag(mod.Condition("NearLinkedTarget"))}).Tag(mod.Condition("Effective")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", -10).Tag(mod.Condition("NearLinkedTarget"))}).Tag(mod.Condition("Effective")),
	},
	`inflict ([\w\s\d]+) exposure on hit, applying -(\d+)% to ([\w\s\d]+) resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD(utils.Capital(captures[0])+"ExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(utils.Capital(captures[2])+"Exposure", "BASE", -utils.Float(captures[1]))}).Tag(mod.Condition("Effective")),
		}, ""
	},
	`while a unique enemy is in your presence, inflict ([\w\s\d]+) exposure on hit, applying -(\d+)% to ([\w\s\d]+) resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD(utils.Capital(captures[0])+"ExposureChance", "BASE", 100).Tag(mod.ActorCondition("enemy", "RareOrUnique")).Tag(mod.Condition("Effective")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(utils.Capital(captures[2])+"Exposure", "BASE", -utils.Float(captures[1])).Tag(mod.Condition("RareOrUnique"))}).Tag(mod.Condition("Effective")),
		}, ""
	},
	`while a pinnacle atlas boss is in your presence, inflict ([\w\s\d]+) exposure on hit, applying -(\d+)% to ([\w\s\d]+) resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD(utils.Capital(captures[0])+"ExposureChance", "BASE", 100).Tag(mod.ActorCondition("enemy", "PinnacleBoss")).Tag(mod.Condition("Effective")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(utils.Capital(captures[2])+"Exposure", "BASE", -utils.Float(captures[1])).Tag(mod.Condition("PinnacleBoss"))}).Tag(mod.Condition("Effective")),
		}, ""
	},
	`fire exposure you inflict applies an extra (-?\d+)% to fire resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ExtraFireExposure", mod.TypeBase, num)}, ""
	},
	`cold exposure you inflict applies an extra (-?\d+)% to cold resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ExtraColdExposure", mod.TypeBase, num)}, ""
	},
	`lightning exposure you inflict applies an extra (-?\d+)% to lightning resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ExtraLightningExposure", mod.TypeBase, num)}, ""
	},
	`exposure you inflict applies at least (-\d+)% to the affected resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ExposureMin", mod.TypeOverride, num)}, ""
	},
	"modifiers to minimum endurance charges instead apply to minimum brutal charges":  []mod.Mod{mod.NewFlag("MinimumEnduranceChargesEqualsMinimumBrutalCharges", true)},
	"modifiers to minimum frenzy charges instead apply to minimum affliction charges": []mod.Mod{mod.NewFlag("MinimumFrenzyChargesEqualsMinimumAfflictionCharges", true)},
	"modifiers to minimum power charges instead apply to minimum absorption charges":  []mod.Mod{mod.NewFlag("MinimumPowerChargesEqualsMinimumAbsorptionCharges", true)},
	"maximum brutal charges is equal to maximum endurance charges":                    []mod.Mod{mod.NewFlag("MaximumEnduranceChargesEqualsMaximumBrutalCharges", true)},
	"maximum affliction charges is equal to maximum frenzy charges":                   []mod.Mod{mod.NewFlag("MaximumFrenzyChargesEqualsMaximumAfflictionCharges", true)},
	"maximum absorption charges is equal to maximum power charges":                    []mod.Mod{mod.NewFlag("MaximumPowerChargesEqualsMaximumAbsorptionCharges", true)},
	"gain brutal charges instead of endurance charges":                                []mod.Mod{mod.NewFlag("EnduranceChargesConvertToBrutalCharges", true)},
	"gain affliction charges instead of frenzy charges":                               []mod.Mod{mod.NewFlag("FrenzyChargesConvertToAfflictionCharges", true)},
	"gain absorption charges instead of power charges":                                []mod.Mod{mod.NewFlag("PowerChargesConvertToAbsorptionCharges", true)},
	`regenerate (\d+)% life over one second when hit while sane`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("Insane").Neg(true)).Tag(mod.Condition("BeenHitRecently")),
		}, ""
	},
	"you have lesser brutal shrine buff": []mod.Mod{
		MOD("ShrineBuff", "LIST", mod.ShrineBuff{Mod: mod.NewFloat("Damage", mod.TypeIncrease, 20)}),
		MOD("ShrineBuff", "LIST", mod.ShrineBuff{Mod: mod.NewFloat("EnemyStunDuration", mod.TypeIncrease, 20)}),
		mod.NewFloat("EnemyKnockbackChance", mod.TypeBase, 100),
	},
	"you have lesser massive shrine buff": []mod.Mod{
		MOD("ShrineBuff", "LIST", mod.ShrineBuff{Mod: mod.NewFloat("Life", mod.TypeIncrease, 20)}),
		MOD("ShrineBuff", "LIST", mod.ShrineBuff{Mod: mod.NewFloat("AreaOfEffect", mod.TypeIncrease, 20)}),
	},
	`(\d+)% increased effect of shrine buffs on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ShrineBuffEffect", mod.TypeIncrease, num)}, ""
	},
	"left ring slot: cover enemies in ash for 5 seconds when you ignite them":    []mod.Mod{MOD("CoveredInAshEffect", "BASE", 20).Tag(mod.SlotNumber(1)).Tag(mod.ActorCondition("enemy", "Ignited"))},
	"right ring slot: cover enemies in frost for 5 seconds when you freeze them": []mod.Mod{MOD("CoveredInFrostEffect", "BASE", 20).Tag(mod.SlotNumber(2)).Tag(mod.ActorCondition("enemy", "Frozen"))},
	`([\w\s]+) has (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BuffEffect", "INC", utils.Float(captures[1])).Tag(mod.SkillIdByName(captures[0]))}, ""
	},
	// Traps, Mines and Totems
	`traps and mines deal (\d+)-(\d+) additional physical damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalMin", "BASE", utils.Float(captures[0])).KeywordFlag(mod.KeywordFlagTrap).KeywordFlag(mod.KeywordFlagMine),
			MOD("PhysicalMax", "BASE", utils.Float(captures[1])).KeywordFlag(mod.KeywordFlagTrap).KeywordFlag(mod.KeywordFlagMine),
		}, ""
	},
	`traps and mines deal (\d+) to (\d+) additional physical damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalMin", "BASE", utils.Float(captures[0])).KeywordFlag(mod.KeywordFlagTrap).KeywordFlag(mod.KeywordFlagMine),
			MOD("PhysicalMax", "BASE", utils.Float(captures[1])).KeywordFlag(mod.KeywordFlagTrap).KeywordFlag(mod.KeywordFlagMine),
		}, ""
	},
	`each mine applies (\d+)% increased damage taken to enemies near it, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Multiplier("ActiveMineCount").Limit(utils.Float(captures[1]) / num))}),
		}, ""
	},
	`can have up to (\d+) additional traps? placed at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ActiveTrapLimit", mod.TypeBase, num)}, ""
	},
	`can have (\d+) fewer traps placed at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveTrapLimit", "BASE", -num)}, ""
	},
	`can have up to (\d+) additional remote mines? placed at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ActiveMineLimit", mod.TypeBase, num)}, ""
	},
	`can have up to (\d+) additional totems? summoned at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ActiveTotemLimit", mod.TypeBase, num)}, ""
	},
	`attack skills can have (\d+) additional totems? summoned at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveTotemLimit", "BASE", num).KeywordFlag(mod.KeywordFlagAttack)}, ""
	},
	`can [hs][au][vm][em]o?n? 1 additional siege ballista totem per (\d+) dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ActiveBallistaLimit", "BASE", 1).Tag(mod.SkillName("Siege Ballista")).Tag(mod.PerStat(num, "Dex")),
		}, ""
	},
	`totems fire (\d+) additional projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).KeywordFlag(mod.KeywordFlagTotem)}, ""
	},
	`([\d\.]+)% of damage dealt by y?o?u?r? ?totems is leeched to you as life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageLifeLeechToPlayer", "BASE", num).KeywordFlag(mod.KeywordFlagTotem)}, ""
	},
	`([\d\.]+)% of damage dealt by y?o?u?r? ?mines is leeched to you as life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DamageLifeLeechToPlayer", "BASE", num).KeywordFlag(mod.KeywordFlagMine)}, ""
	},
	"you can cast an additional brand": []mod.Mod{mod.NewFloat("ActiveBrandLimit", mod.TypeBase, 1)},
	`you can cast (\d+) additional brands`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ActiveBrandLimit", mod.TypeBase, num)}, ""
	},
	`(\d+)% increased damage while you are wielding a bow and have a totem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).Tag(mod.Condition("HaveTotem")).Tag(mod.Condition("UsingBow"))}, ""
	},
	`each totem applies (\d+)% increased damage taken to enemies near it`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Multiplier("TotemsSummoned").Base(0))})}, ""
	},
	`totems gain \+(\d+)% to ([\w\s\d]+) resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Totem"+utils.Capital(captures[1])+"Resist", "BASE", num)}, ""
	},
	`totems gain \+(\d+)% to all elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("TotemElementalResist", mod.TypeBase, num)}, ""
	},
	// Minions
	"your strength is added to your minions":         []mod.Mod{mod.NewFlag("HalfStrengthAddedToMinions", true)},
	"half of your strength is added to your minions": []mod.Mod{mod.NewFlag("HalfStrengthAddedToMinions", true)},
	`minions created recently have (\d+)% increased attack and cast speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Speed", mod.TypeIncrease, num)}).Tag(mod.Condition("MinionsCreatedRecently"))}, ""
	},
	`minions created recently have (\d+)% increased movement speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("MovementSpeed", mod.TypeIncrease, num)}).Tag(mod.Condition("MinionsCreatedRecently"))}, ""
	},
	"minions poison enemies on hit": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("PoisonChance", mod.TypeBase, 100)})},
	`minions have (\d+)% chance to poison enemies on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("PoisonChance", mod.TypeBase, num)})}, ""
	},
	`(\d+)% increased minion damage if you have hit recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeIncrease, num)}).Tag(mod.Condition("HitRecently"))}, ""
	},
	`(\d+)% increased minion damage if you've used a minion skill recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeIncrease, num)}).Tag(mod.Condition("UsedMinionSkillRecently"))}, ""
	},
	`minions deal (\d+)% increased damage if you've used a minion skill recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeIncrease, num)}).Tag(mod.Condition("UsedMinionSkillRecently"))}, ""
	},
	`minions have (\d+)% increased attack and cast speed if you or your minions have killed recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Speed", mod.TypeIncrease, num)}).Tag(mod.Condition("KilledRecently", "MinionsKilledRecently"))}, ""
	},
	`(\d+)% increased minion attack speed per (\d+) dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Speed", "INC", num).Flag(mod.MFlagAttack)}).Tag(mod.PerStat(utils.Float(captures[1]), "Dex"))}, ""
	},
	`(\d+)% increased minion movement speed per (\d+) dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("MovementSpeed", mod.TypeIncrease, num)}).Tag(mod.PerStat(utils.Float(captures[1]), "Dex"))}, ""
	},
	`minions deal (\d+)% increased damage per (\d+) dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeIncrease, num)}).Tag(mod.PerStat(utils.Float(captures[1]), "Dex"))}, ""
	},
	`minions have (\d+)% chance to deal double damage while they are on full life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("DoubleDamageChance", "BASE", num).Tag(mod.Condition("FullLife"))})}, ""
	},
	`(\d+)% increased golem damage for each type of golem you have summoned`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HavePhysicalGolem"))}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HaveLightningGolem"))}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HaveColdGolem"))}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HaveFireGolem"))}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HaveChaosGolem"))}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "HaveCarrionGolem"))}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
		}, ""
	},
	`can summon up to (\d) additional golems? at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ActiveGolemLimit", mod.TypeBase, num)}, ""
	},
	`\+(\d) to maximum number of sentinels of purity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ActiveSentinelOfPurityLimit", mod.TypeBase, num)}, ""
	},
	`if you have 3 primordial jewels, can summon up to (\d) additional golems? at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ActiveGolemLimit", "BASE", num).Tag(mod.MultiplierThreshold("PrimordialItem").Threshold(3)),
		}, ""
	},
	`golems regenerate (\d)% of their maximum life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("LifeRegenPercent", mod.TypeBase, num)}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
		}, ""
	},
	`summoned golems regenerate (\d)% of their life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("LifeRegenPercent", mod.TypeBase, num)}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
		}, ""
	},
	`golems summoned in the past 8 seconds deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "SummonedGolemInPast8Sec"))}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
		}, ""
	},
	"gain onslaught for 10 seconds when you cast socketed golem skill": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Condition:Onslaught").Tag(mod.Condition("SummonedGolemInPast10Sec"))}, ""
	},
	"s?u?m?m?o?n?e?d? ?raging spirits' hits always ignite": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 100)}).Tag(mod.SkillName("Summon Raging Spirit"))},
	"raised zombies have avatar of fire":                   []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Keystone", "LIST", "Avatar of Fire")}).Tag(mod.SkillName("Raise Zombie"))},
	`raised zombies take ([\d\.]+)% of their maximum life per second as fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("FireDegen", "BASE", 1).Tag(mod.PercentStat("Life", num))}).Tag(mod.SkillName("Raise Zombie")),
		}, ""
	},
	"summoned skeletons have avatar of fire": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Keystone", "LIST", "Avatar of Fire")}).Tag(mod.SkillName("Summon Skeleton"))},
	`summoned skeletons take ([\d\.]+)% of their maximum life per second as fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("FireDegen", "BASE", 1).Tag(mod.PercentStat("Life", num))}).Tag(mod.SkillName("Summon Skeleton"))}, ""
	},
	`summoned skeletons have (\d+)% chance to wither enemies for (\d+) seconds on hit`: []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFlag("Condition:CanWither", true)}).Tag(mod.SkillName("Summon Skeleton"))},
	`summoned skeletons have (\d+)% of physical damage converted to chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamageConvertToChaos", mod.TypeBase, num)}).Tag(mod.SkillName("Summon Skeleton")),
		}, ""
	},
	`minions convert (\d+)% of physical damage to fire damage per red socket`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamageConvertToFire", mod.TypeBase, num)}).Tag(mod.Multiplier("RedSocketIn{SlotName}").Base(0)),
		}, ""
	},
	`minions convert (\d+)% of physical damage to cold damage per green socket`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamageConvertToCold", mod.TypeBase, num)}).Tag(mod.Multiplier("GreenSocketIn{SlotName}").Base(0))}, ""
	},
	`minions convert (\d+)% of physical damage to lightning damage per blue socket`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamageConvertToLightning", mod.TypeBase, num)}).Tag(mod.Multiplier("BlueSocketIn{SlotName}").Base(0))}, ""
	},
	`minions convert (\d+)% of physical damage to chaos damage per white socket`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamageConvertToChaos", mod.TypeBase, num)}).Tag(mod.Multiplier("WhiteSocketIn{SlotName}").Base(0))}, ""
	},
	`minions have a (\d+)% chance to impale on hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ImpaleChance", "BASE", num)})}, ""
	},
	`minions from herald skills deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, num)}).Tag(mod.SkillType(string(data.SkillTypeHerald)))}, ""
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
	"summoned skeleton warriors deal triple damage with this weapon if you've hit with this weapon recently": []mod.Mod{
		MOD("Dummy", "DUMMY", 1).Tag(mod.Condition("HitRecentlyWithWeapon")), // Make the Configuration option appear
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("TripleDamageChance", "BASE", 100).Tag(mod.ActorCondition("parent", "HitRecentlyWithWeapon"))}).Tag(mod.SkillName("Summon Skeleton")),
	},
	"summoned skeleton warriors wield a copy of this weapon while in your main hand": []mod.Mod{}, // just make the mod blue, handled in CalcSetup
	"each summoned phantasm grants you phantasmal might":                             []mod.Mod{mod.NewFlag("Condition:PhantasmalMight", true)},
	`minions have (\d+)% increased critical strike chance per maximum power charge you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("CritChance", "INC", num).Tag(mod.Multiplier("PowerChargeMax").Actor("parent"))})}, ""
	},
	"minions can hear the whispers for 5 seconds after they deal a critical strike": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", 50).Tag(mod.Condition("NeverCrit").Neg(true))}),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Speed", "INC", 50).Flag(mod.MFlagAttack).Tag(mod.Condition("NeverCrit").Neg(true))}),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChaosDegen", "BASE", 1).Tag(mod.PercentStat("Life", 20)).Tag(mod.Condition("NeverCrit").Neg(true))}),
		}, ""
	},
	"chaos damage t?a?k?e?n? ?does not bypass minions' energy shield": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFlag("ChaosNotBypassEnergyShield", true)})}, ""
	},
	"while minions have energy shield, their hits ignore monster elemental resistances": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("IgnoreElementalResistances").Tag(mod.StatThreshold("EnergyShield", 1))})}, ""
	},
	// Projectiles
	`skills chain \+(\d) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ChainCountMax", mod.TypeBase, num)}, ""
	},
	"skills chain an additional time while at maximum frenzy charges": []mod.Mod{MOD("ChainCountMax", "BASE", 1).Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	"attacks chain an additional time when in main hand":              []mod.Mod{MOD("ChainCountMax", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.SlotNumber(1))},
	`projectiles chain \+(\d) times while you have phasing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChainCountMax", "BASE", num).Flag(mod.MFlagProjectile).Tag(mod.Condition("Phasing"))}, ""
	},
	"adds an additional arrow": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagAttack)},
	`(\d+) additional arrows`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).Flag(mod.MFlagAttack)}, ""
	},
	"bow attacks fire an additional arrow": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagBow)},
	`bow attacks fire (\d+) additional arrows`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).Flag(mod.MFlagBow)}, ""
	},
	`bow attacks fire (\d+) additional arrows if you haven't cast dash recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ProjectileCount", "BASE", num).Flag(mod.MFlagBow).Tag(mod.Condition("CastDashRecently").Neg(true))}, ""
	},
	"wand attacks fire an additional projectile":             []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagWand)},
	"skills fire an additional projectile":                   []mod.Mod{mod.NewFloat("ProjectileCount", mod.TypeBase, 1)},
	"spells [hf][ai][vr]e an additional projectile":          []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagSpell)},
	"attacks fire an additional projectile":                  []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagAttack)},
	"attacks have an additional projectile when in off hand": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.SlotNumber(2))},
	"projectiles pierce an additional target":                []mod.Mod{mod.NewFloat("PierceCount", mod.TypeBase, 1)},
	`projectiles pierce (\d+) targets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("PierceCount", mod.TypeBase, num)}, ""
	},
	`projectiles pierce (\d+) additional targets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("PierceCount", mod.TypeBase, num)}, ""
	},
	`projectiles pierce (\d+) additional targets while you have phasing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PierceCount", "BASE", num).Tag(mod.Condition("Phasing"))}, ""
	},
	"projectiles pierce all targets while you have phasing": []mod.Mod{FLAG("PierceAllTargets").Tag(mod.Condition("Phasing"))},
	"arrows pierce an additional target":                    []mod.Mod{MOD("PierceCount", "BASE", 1).Flag(mod.MFlagAttack)},
	"arrows pierce one target":                              []mod.Mod{MOD("PierceCount", "BASE", 1).Flag(mod.MFlagAttack)},
	`arrows pierce (\d+) targets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PierceCount", "BASE", num).Flag(mod.MFlagAttack)}, ""
	},
	"always pierce with arrows":         []mod.Mod{FLAG("PierceAllTargets").Flag(mod.MFlagAttack)},
	"arrows always pierce":              []mod.Mod{FLAG("PierceAllTargets").Flag(mod.MFlagAttack)},
	"arrows pierce all targets":         []mod.Mod{FLAG("PierceAllTargets").Flag(mod.MFlagAttack)},
	"arrows that pierce cause bleeding": []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagAttack).Flag(mod.MFlagProjectile).Tag(mod.StatThreshold("PierceCount", 1))},
	`arrows that pierce have (\d+)% chance to cause bleeding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("BleedChance", "BASE", num).Flag(mod.MFlagAttack).Flag(mod.MFlagProjectile).Tag(mod.StatThreshold("PierceCount", 1)),
		}, ""
	},
	`arrows that pierce deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "INC", num).Flag(mod.MFlagAttack).Flag(mod.MFlagProjectile).Tag(mod.StatThreshold("PierceCount", 1)),
		}, ""
	},
	`projectiles gain (\d+)% of non-chaos damage as extra chaos damage per chain`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("NonChaosDamageGainAsChaos", "BASE", num).Flag(mod.MFlagProjectile).Tag(mod.PerStat(0, "Chain")),
		}, ""
	},
	`projectiles that have chained gain (\d+)% of non-chaos damage as extra chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("NonChaosDamageGainAsChaos", "BASE", num).Flag(mod.MFlagProjectile).Tag(mod.StatThreshold("Chain", 1)),
		}, ""
	},
	"left ring slot: projectiles from spells cannot chain": []mod.Mod{FLAG("CannotChain").Flag(mod.MFlagSpell).Flag(mod.MFlagProjectile).Tag(mod.SlotNumber(1))},
	"left ring slot: projectiles from spells fork": []mod.Mod{
		FLAG("ForkOnce").Flag(mod.MFlagSpell).Flag(mod.MFlagProjectile).Tag(mod.SlotNumber(1)),
		MOD("ForkCountMax", "BASE", 1).Flag(mod.MFlagSpell).Flag(mod.MFlagProjectile).Tag(mod.SlotNumber(1)),
	},
	"left ring slot: your chilling skitterbot's aura applies socketed h?e?x? ?curse instead":  []mod.Mod{FLAG("SkitterbotsCannotChill").Tag(mod.SlotNumber(1))},
	`right ring slot: projectiles from spells chain \+1 times`:                                []mod.Mod{MOD("ChainCountMax", "BASE", 1).Flag(mod.MFlagSpell).Flag(mod.MFlagProjectile).Tag(mod.SlotNumber(2))},
	"right ring slot: projectiles from spells cannot fork":                                    []mod.Mod{FLAG("CannotFork").Flag(mod.MFlagSpell).Flag(mod.MFlagProjectile).Tag(mod.SlotNumber(2))},
	"right ring slot: your shocking skitterbot's aura applies socketed h?e?x? ?curse instead": []mod.Mod{FLAG("SkitterbotsCannotShock").Tag(mod.SlotNumber(2))},
	"projectiles from spells cannot pierce":                                                   []mod.Mod{FLAG("CannotPierce").Flag(mod.MFlagSpell)},
	"projectiles fork":                                                                        []mod.Mod{FLAG("ForkOnce").Flag(mod.MFlagProjectile), MOD("ForkCountMax", "BASE", 1).Flag(mod.MFlagProjectile)},
	`(\d+)% increased critical strike chance with arrows that fork`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "INC", num).Flag(mod.MFlagBow).Tag(mod.StatThreshold("ForkRemaining", 1)).Tag(mod.StatThreshold("PierceCount", 0).Upper(true)),
		}, ""
	},
	`arrows that pierce have \+(\d+)% to critical strike multiplier`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritMultiplier", "BASE", num).Flag(mod.MFlagBow).Tag(mod.StatThreshold("PierceCount", 1)),
		}, ""
	},
	"arrows pierce all targets after forking":                                                             []mod.Mod{FLAG("PierceAllTargets").Flag(mod.MFlagBow).Tag(mod.StatThreshold("ForkedCount", 1))},
	"modifiers to number of projectiles instead apply to the number of targets projectiles split towards": []mod.Mod{mod.NewFlag("NoAdditionalProjectiles", true)},
	"attack skills fire an additional projectile while wielding a claw or dagger":                         []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.ModFlagOr(mod.MFlagClaw | mod.MFlagDagger))},
	`skills fire (\d+) additional projectiles for 4 seconds after you consume a total of 12 steel shards`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ProjectileCount", "BASE", num).Tag(mod.Condition("Consumed12SteelShardsRecently")),
		}, ""
	},
	`non-projectile chaining lightning skills chain \+(\d+) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ChainCountMax", "BASE", num).Tag(mod.SkillType(string(data.SkillTypeProjectile)).Neg(true)).Tag(mod.SkillType(string(data.SkillTypeChains)), mod.SkillType(string(data.SkillTypeLightning))),
		}, ""
	},
	`arrows gain damage as they travel farther, dealing up to (\d+)% increased damage with hits to targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "INC", num).Flag(mod.MFlagBow).Flag(mod.MFlagHit).Tag(mod.DistanceRamp([][]int{{35, 0}, {70, 1}})),
		}, ""
	},
	`arrows gain critical strike chance as they travel farther, up to (\d+)% increased critical strike chance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "INC", num).Flag(mod.MFlagBow).Tag(mod.DistanceRamp([][]int{{35, 0}, {70, 1}})),
		}, ""
	},
	// Leech/Gain on Hit
	"cannot leech life":                                                                       []mod.Mod{mod.NewFlag("CannotLeechLife", true)},
	"cannot leech mana":                                                                       []mod.Mod{mod.NewFlag("CannotLeechMana", true)},
	"cannot leech when on low life":                                                           []mod.Mod{FLAG("CannotLeechLife").Tag(mod.Condition("LowLife")), FLAG("CannotLeechMana").Tag(mod.Condition("LowLife"))},
	"cannot leech life from critical strikes":                                                 []mod.Mod{FLAG("CannotLeechLife").Tag(mod.Condition("CriticalStrike"))},
	"leech applies instantly on critical strike":                                              []mod.Mod{FLAG("InstantLifeLeech").Tag(mod.Condition("CriticalStrike")), FLAG("InstantManaLeech").Tag(mod.Condition("CriticalStrike"))},
	"gain life and mana from leech instantly on critical strike":                              []mod.Mod{FLAG("InstantLifeLeech").Tag(mod.Condition("CriticalStrike")), FLAG("InstantManaLeech").Tag(mod.Condition("CriticalStrike"))},
	"leech applies instantly during flask effect":                                             []mod.Mod{FLAG("InstantLifeLeech").Tag(mod.Condition("UsingFlask")), FLAG("InstantManaLeech").Tag(mod.Condition("UsingFlask"))},
	"gain life and mana from leech instantly during flask effect":                             []mod.Mod{FLAG("InstantLifeLeech").Tag(mod.Condition("UsingFlask")), FLAG("InstantManaLeech").Tag(mod.Condition("UsingFlask"))},
	"life and mana leech from critical strikes are instant":                                   []mod.Mod{FLAG("InstantLifeLeech").Tag(mod.Condition("CriticalStrike")), FLAG("InstantManaLeech").Tag(mod.Condition("CriticalStrike"))},
	"gain life and mana from leech instantly during effect":                                   []mod.Mod{FLAG("InstantLifeLeech").Tag(mod.Condition("UsingFlask")), FLAG("InstantManaLeech").Tag(mod.Condition("UsingFlask"))},
	"with 5 corrupted items equipped: life leech recovers based on your chaos damage instead": []mod.Mod{FLAG("LifeLeechBasedOnChaosDamage").Tag(mod.MultiplierThreshold("CorruptedItem").Threshold(5))},
	"you have vaal pact if you've dealt a critical strike recently":                           []mod.Mod{MOD("Keystone", "LIST", "Vaal Pact").Tag(mod.Condition("CritRecently"))},
	`gain (\d+) energy shield for each enemy you hit which is affected by a spider's web`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnHit", "BASE", num).Tag(mod.MultiplierThreshold("Spider's WebStack").Actor("enemy").Threshold(1)),
		}, ""
	},
	`(\d+) life gained for each enemy hit if you have used a vaal skill recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnHit", "BASE", num).Tag(mod.Condition("UsedVaalSkillRecently"))}, ""
	},
	`(\d+) life gained for each cursed enemy hit by your attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnHit", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`(\d+) mana gained for each cursed enemy hit by your attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnHit", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	// Defences
	"chaos damage t?a?k?e?n? ?does not bypass energy shield": []mod.Mod{mod.NewFlag("ChaosNotBypassEnergyShield", true)},
	`(\d+)% of chaos damage t?a?k?e?n? ?does not bypass energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosEnergyShieldBypass", "BASE", -num)}, ""
	},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life":             []mod.Mod{FLAG("ChaosNotBypassEnergyShield").Tag(mod.Condition("LowLife").Neg(true))},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life or low mana": []mod.Mod{FLAG("ChaosNotBypassEnergyShield").Tag(mod.Condition("LowLife", "LowMana").Neg(true))},
	"chaos damage is taken from mana before life": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ChaosDamageTakenFromManaBeforeLife", mod.TypeBase, 100)}, ""
	},
	"cannot evade enemy attacks":                               []mod.Mod{mod.NewFlag("CannotEvade", true)},
	"cannot block":                                             []mod.Mod{mod.NewFlag("CannotBlockAttacks", true), mod.NewFlag("CannotBlockSpells", true)},
	"cannot block while you have no energy shield":             []mod.Mod{FLAG("CannotBlockAttacks").Tag(mod.Condition("HaveEnergyShield").Neg(true)), FLAG("CannotBlockSpells").Tag(mod.Condition("HaveEnergyShield").Neg(true))},
	"cannot block attacks":                                     []mod.Mod{mod.NewFlag("CannotBlockAttacks", true)},
	"cannot block spells":                                      []mod.Mod{mod.NewFlag("CannotBlockSpells", true)},
	"damage from blocked hits cannot bypass energy shield":     []mod.Mod{FLAG("BlockedDamageDoesntBypassES").Tag(mod.Condition("EVBypass").Neg(true))},
	"damage from unblocked hits always bypasses energy shield": []mod.Mod{FLAG("UnblockedDamageDoesBypassES").Tag(mod.Condition("EVBypass").Neg(true))},
	`recover (\d+) life when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("LifeOnBlock", mod.TypeBase, num)}, ""
	},
	`recover (\d+) energy shield when you block spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("EnergyShieldOnSpellBlock", mod.TypeBase, num)}, ""
	},
	`recover (\d+)% of life when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnBlock", "BASE", 1).Tag(mod.PerStat(100/num, "Life"))}, ""
	},
	`recover (\d+)% of life when you block attack damage while wielding a staff`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnBlock", "BASE", 1).Tag(mod.PerStat(100/num, "Life")).Tag(mod.Condition("UsingStaff")),
		}, ""
	},
	`recover (\d+)% of your maximum mana when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ManaOnBlock", "BASE", 1).Tag(mod.PerStat(100/num, "Mana")),
		}, ""
	},
	`recover (\d+)% of energy shield when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnBlock", "BASE", 1).Tag(mod.PerStat(100/num, "EnergyShield")),
		}, ""
	},
	`recover (\d+)% of energy shield when you block spell damage while wielding a staff`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnSpellBlock", "BASE", 1).Tag(mod.PerStat(100/num, "EnergyShield")).Tag(mod.Condition("UsingStaff")),
		}, ""
	},
	`replenishes energy shield by (\d+)% of armour when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnBlock", "BASE", 1).Tag(mod.PerStat(100/num, "Armour")),
		}, ""
	},
	"cannot leech or regenerate mana":                                 []mod.Mod{mod.NewFlag("NoManaRegen", true), mod.NewFlag("CannotLeechMana", true)},
	"right ring slot: you cannot regenerate mana":                     []mod.Mod{FLAG("NoManaRegen").Tag(mod.SlotNumber(2))},
	"y?o?u? ?cannot recharge energy shield":                           []mod.Mod{mod.NewFlag("NoEnergyShieldRecharge", true)},
	"you cannot regenerate energy shield":                             []mod.Mod{mod.NewFlag("NoEnergyShieldRegen", true)},
	"cannot recharge or regenerate energy shield":                     []mod.Mod{mod.NewFlag("NoEnergyShieldRecharge", true), mod.NewFlag("NoEnergyShieldRegen", true)},
	"left ring slot: you cannot recharge or regenerate energy shield": []mod.Mod{FLAG("NoEnergyShieldRecharge").Tag(mod.SlotNumber(1)), FLAG("NoEnergyShieldRegen").Tag(mod.SlotNumber(1))},
	"cannot gain energy shield":                                       []mod.Mod{mod.NewFlag("NoEnergyShieldRegen", true), mod.NewFlag("NoEnergyShieldRecharge", true), mod.NewFlag("CannotLeechEnergyShield", true)},
	`you lose (\d+)% of energy shield per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldDegen", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)),
		}, ""
	},
	`lose (\d+)% of energy shield per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldDegen", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)),
		}, ""
	},
	`lose (\d+)% of life per second if you have been hit recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeDegen", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.Condition("BeenHitRecently")),
		}, ""
	},
	"you have no armour or energy shield": []mod.Mod{
		MOD("Armour", "MORE", -100),
		MOD("EnergyShield", "MORE", -100),
	},
	"you have no armour or maximum energy shield": []mod.Mod{
		MOD("Armour", "MORE", -100),
		MOD("EnergyShield", "MORE", -100),
	},
	"defences are zero": []mod.Mod{
		MOD("Armour", "MORE", -100),
		MOD("EnergyShield", "MORE", -100),
		MOD("Evasion", "MORE", -100),
		MOD("Ward", "MORE", -100),
	},
	"you have no intelligence": []mod.Mod{
		MOD("Int", "MORE", -100),
	},
	"elemental resistances are zero": []mod.Mod{
		mod.NewFloat("FireResist", mod.TypeOverride, 0),
		mod.NewFloat("ColdResist", mod.TypeOverride, 0),
		mod.NewFloat("LightningResist", mod.TypeOverride, 0),
	},
	"chaos resistance is zero": []mod.Mod{
		mod.NewFloat("ChaosResist", mod.TypeOverride, 0),
	},
	`your maximum resistances are (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("FireResistMax", mod.TypeOverride, num),
			mod.NewFloat("ColdResistMax", mod.TypeOverride, num),
			mod.NewFloat("LightningResistMax", mod.TypeOverride, num),
			mod.NewFloat("ChaosResistMax", mod.TypeOverride, num),
		}, ""
	},
	`fire resistance is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("FireResist", mod.TypeOverride, num)}, ""
	},
	`cold resistance is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ColdResist", mod.TypeOverride, num)}, ""
	},
	`lightning resistance is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("LightningResist", mod.TypeOverride, num)}, ""
	},
	"elemental resistances are capped by your highest maximum elemental resistance instead": []mod.Mod{mod.NewFlag("ElementalResistMaxIsHighestResistMax", true)},
	"chaos resistance is doubled": []mod.Mod{mod.NewFloat("ChaosResist", mod.TypeMore, 100)},
	`nearby enemies have (\d+)% increased fire and cold resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("FireResist", mod.TypeIncrease, num)}),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("ColdResist", mod.TypeIncrease, num)}),
		}, ""
	},
	"nearby enemies are blinded while physical aegis is not depleted": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Blinded", true)}).Tag(mod.Condition("PhysicalAegisDepleted").Neg(true))},
	"armour is increased by uncapped fire resistance":                 []mod.Mod{MOD("Armour", "INC", 1).Tag(mod.PerStat(1, "FireResistTotal"))},
	"armour is increased by overcapped fire resistance":               []mod.Mod{MOD("Armour", "INC", 1).Tag(mod.PerStat(1, "FireResistOverCap"))},
	"evasion rating is increased by uncapped cold resistance":         []mod.Mod{MOD("Evasion", "INC", 1).Tag(mod.PerStat(1, "ColdResistTotal"))},
	"evasion rating is increased by overcapped cold resistance":       []mod.Mod{MOD("Evasion", "INC", 1).Tag(mod.PerStat(1, "ColdResistOverCap"))},
	`reflects (\d+) physical damage to melee attackers`:               []mod.Mod{},
	"ignore all movement penalties from armour":                       []mod.Mod{mod.NewFlag("Condition:IgnoreMovementPenalties", true)},
	"gain armour equal to your reserved mana":                         []mod.Mod{MOD("Armour", "BASE", 1).Tag(mod.PerStat(1, "ManaReserved"))},
	`(\d+)% increased armour per (\d+) reserved mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Armour", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "ManaReserved")),
		}, ""
	},
	"cannot be stunned": []mod.Mod{mod.NewFloat("AvoidStun", mod.TypeBase, 100)},
	"cannot be stunned if you haven't been hit recently": []mod.Mod{MOD("AvoidStun", "BASE", 100).Tag(mod.Condition("BeenHitRecently").Neg(true))},
	`cannot be stunned if you have at least (\d+) crab barriers`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AvoidStun", "BASE", 100).Tag(mod.StatThreshold("CrabBarriers", num)),
		}, ""
	},
	"cannot be blinded": []mod.Mod{mod.NewFloat("AvoidBlind", mod.TypeBase, 100)},
	"cannot be shocked": []mod.Mod{mod.NewFloat("AvoidShock", mod.TypeBase, 100)},
	"immune to shock":   []mod.Mod{mod.NewFloat("AvoidShock", mod.TypeBase, 100)},
	"cannot be frozen":  []mod.Mod{mod.NewFloat("AvoidFreeze", mod.TypeBase, 100)},
	"immune to freeze":  []mod.Mod{mod.NewFloat("AvoidFreeze", mod.TypeBase, 100)},
	"cannot be chilled": []mod.Mod{mod.NewFloat("AvoidChill", mod.TypeBase, 100)},
	"immune to chill":   []mod.Mod{mod.NewFloat("AvoidChill", mod.TypeBase, 100)},
	"cannot be ignited": []mod.Mod{mod.NewFloat("AvoidIgnite", mod.TypeBase, 100)},
	"immune to ignite":  []mod.Mod{mod.NewFloat("AvoidIgnite", mod.TypeBase, 100)},
	"cannot be ignited while at maximum endurance charges":            []mod.Mod{MOD("AvoidIgnite", "BASE", 100).Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax"))},
	"cannot be chilled while at maximum frenzy charges":               []mod.Mod{MOD("AvoidChill", "BASE", 100).Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	"cannot be shocked while at maximum power charges":                []mod.Mod{MOD("AvoidShock", "BASE", 100).Tag(mod.StatThresholdStat("PowerCharges", "PowerChargesMax"))},
	"you cannot be shocked while at maximum endurance charges":        []mod.Mod{MOD("AvoidShock", "BASE", 100).Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax"))},
	"you cannot be shocked while chilled":                             []mod.Mod{MOD("AvoidShock", "BASE", 100).Tag(mod.Condition("Chilled"))},
	"cannot be shocked while chilled":                                 []mod.Mod{MOD("AvoidShock", "BASE", 100).Tag(mod.Condition("Chilled"))},
	"cannot be shocked if intelligence is higher than strength":       []mod.Mod{MOD("AvoidShock", "BASE", 100).Tag(mod.Condition("IntHigherThanStr"))},
	"cannot be frozen if dexterity is higher than intelligence":       []mod.Mod{MOD("AvoidFreeze", "BASE", 100).Tag(mod.Condition("DexHigherThanInt"))},
	"cannot be frozen if energy shield recharge has started recently": []mod.Mod{MOD("AvoidFreeze", "BASE", 100).Tag(mod.Condition("EnergyShieldRechargeRecently"))},
	"cannot be ignited if strength is higher than dexterity":          []mod.Mod{MOD("AvoidIgnite", "BASE", 100).Tag(mod.Condition("StrHigherThanDex"))},
	"cannot be chilled while burning":                                 []mod.Mod{MOD("AvoidChill", "BASE", 100).Tag(mod.Condition("Burning"))},
	"cannot be chilled while you have onslaught":                      []mod.Mod{MOD("AvoidChill", "BASE", 100).Tag(mod.Condition("Onslaught"))},
	"cannot be inflicted with bleeding":                               []mod.Mod{mod.NewFloat("AvoidBleed", mod.TypeBase, 100)},
	"bleeding cannot be inflicted on you":                             []mod.Mod{mod.NewFloat("AvoidBleed", mod.TypeBase, 100)},
	"you are immune to bleeding":                                      []mod.Mod{mod.NewFloat("AvoidBleed", mod.TypeBase, 100)},
	"immune to poison":                                                []mod.Mod{mod.NewFloat("AvoidPoison", mod.TypeBase, 100)},
	"immunity to shock during flask effect":                           []mod.Mod{MOD("AvoidShock", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"immunity to freeze and chill during flask effect": []mod.Mod{
		MOD("AvoidFreeze", "BASE", 100).Tag(mod.Condition("UsingFlask")),
		MOD("AvoidChill", "BASE", 100).Tag(mod.Condition("UsingFlask")),
	},
	"immune to freeze and chill while ignited": []mod.Mod{
		MOD("AvoidFreeze", "BASE", 100).Tag(mod.Condition("Ignited")),
		MOD("AvoidChill", "BASE", 100).Tag(mod.Condition("Ignited")),
	},
	"immunity to ignite during flask effect":   []mod.Mod{MOD("AvoidIgnite", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"immunity to bleeding during flask effect": []mod.Mod{MOD("AvoidBleed", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"immune to poison during flask effect":     []mod.Mod{MOD("AvoidPoison", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"immune to curses during flask effect":     []mod.Mod{MOD("AvoidCurse", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"immune to freeze, chill, curses and stuns during flask effect": []mod.Mod{
		MOD("AvoidFreeze", "BASE", 100).Tag(mod.Condition("UsingFlask")),
		MOD("AvoidChill", "BASE", 100).Tag(mod.Condition("UsingFlask")),
		MOD("AvoidCurse", "BASE", 100).Tag(mod.Condition("UsingFlask")),
		MOD("AvoidStun", "BASE", 100).Tag(mod.Condition("UsingFlask")),
	},
	"unaffected by curses":                            []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100)},
	"unaffected by curses while affected by zealotry": []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.Condition("AffectedByZealotry"))},
	`immune to curses while you have at least (\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AvoidCurse", "BASE", 100).Tag(mod.MultiplierThreshold("Rage").Threshold(num)),
		}, ""
	},
	"the effect of chill on you is reversed": []mod.Mod{mod.NewFlag("SelfChillEffectIsReversed", true)},
	`your movement speed is (\d+)% of its base value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeed", "OVERRIDE", num/100)}, ""
	},
	"armour also applies to lightning damage taken from hits":     []mod.Mod{mod.NewFlag("ArmourAppliesToLightningDamageTaken", true)},
	"lightning resistance does not affect lightning damage taken": []mod.Mod{mod.NewFlag("SelfIgnoreLightningResistance", true)},
	`(\d+)% increased maximum life and reduced fire resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("Life", mod.TypeIncrease, num),
			MOD("FireResist", "INC", -num),
		}, ""
	},
	`(\d+)% increased maximum mana and reduced cold resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("Mana", mod.TypeIncrease, num),
			MOD("ColdResist", "INC", -num),
		}, ""
	},
	`(\d+)% increased global maximum energy shield and reduced lightning resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShield", "INC", num).Tag(mod.Global()),
			MOD("LightningResist", "INC", -num),
		}, ""
	},
	"cannot be ignited while on low life":     []mod.Mod{MOD("AvoidIgnite", "BASE", 100).Tag(mod.Condition("LowLife"))},
	"ward does not break during flask effect": []mod.Mod{FLAG("WardNotBreak").Tag(mod.Condition("UsingFlask"))},
	// Knockback
	"cannot knock enemies back":                                     []mod.Mod{mod.NewFlag("CannotKnockback", true)},
	"knocks back enemies if you get a critical strike with a staff": []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(mod.MFlagStaff).Tag(mod.Condition("CriticalStrike"))},
	"knocks back enemies if you get a critical strike with a bow":   []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(mod.MFlagBow).Tag(mod.Condition("CriticalStrike"))},
	"bow knockback at close range":                                  []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(mod.MFlagBow).Tag(mod.Condition("AtCloseRange"))},
	"adds knockback during flask effect":                            []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"adds knockback to melee attacks during flask effect":           []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(mod.MFlagMelee).Tag(mod.Condition("UsingFlask"))},
	"knockback direction is reversed":                               []mod.Mod{MOD("EnemyKnockbackDistance", "MORE", -200)},
	// Culling
	"culling strike":                     []mod.Mod{MOD("CullPercent", "MAX", 10)},
	"culling strike during flask effect": []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.Condition("UsingFlask"))},
	"hits with this weapon have culling strike against bleeding enemies": []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.ActorCondition("enemy", "Bleeding"))},
	"you have culling strike against cursed enemies":                     []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.ActorCondition("enemy", "Cursed"))},
	"critical strikes have culling strike":                               []mod.Mod{MOD("CriticalCullPercent", "MAX", 10)},
	"your critical strikes have culling strike":                          []mod.Mod{MOD("CriticalCullPercent", "MAX", 10)},
	"your spells have culling strike":                                    []mod.Mod{MOD("CullPercent", "MAX", 10).Flag(mod.MFlagSpell)},
	"culling strike against burning enemies":                             []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.ActorCondition("enemy", "Burning"))},
	"culling strike against marked enemy":                                []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.ActorCondition("enemy", "Marked"))},
	// Intimidate
	"permanently intimidate enemies on block": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Intimidated", true)}).Tag(mod.Condition("BlockedRecently"))},
	`with a murderous eye jewel socketed, intimidate enemies for (\d) seconds on hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Intimidated", true)}).Tag(mod.Condition("HaveMurderousEyeJewelIn{SlotName}")),
		}, ""
	},
	"enemies taunted by your warcries are intimidated":                    []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated").Tag(mod.Condition("Taunted"))}).Tag(mod.Condition("UsedWarcryRecently"))},
	`intimidate enemies for (\d) seconds on block while holding a shield`: []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Intimidated", true)}).Tag(mod.Condition("BlockedRecently")).Tag(mod.Condition("UsingShield"))},
	// Flasks
	"flasks do not apply to you":                       []mod.Mod{mod.NewFlag("FlasksDoNotApplyToPlayer", true)},
	"flasks apply to your zombies and spectres":        []mod.Mod{FLAG("FlasksApplyToMinion").Tag(mod.SkillName("Raise Zombie", "Raise Spectre"))},
	"flasks apply to your raised zombies and spectres": []mod.Mod{FLAG("FlasksApplyToMinion").Tag(mod.SkillName("Raise Zombie", "Raise Spectre"))},
	"your minions use your flasks when summoned":       []mod.Mod{mod.NewFlag("FlasksApplyToMinion", true)},
	`recover an additional (\d+)% of flask's life recovery amount over 10 seconds if used while not on full life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("FlaskAdditionalLifeRecovery", mod.TypeBase, num),
		}, ""
	},
	"creates a smoke cloud on use":          []mod.Mod{},
	"creates chilled ground on use":         []mod.Mod{},
	"creates consecrated ground on use":     []mod.Mod{},
	"removes bleeding on use":               []mod.Mod{},
	"removes burning on use":                []mod.Mod{},
	"removes curses on use":                 []mod.Mod{},
	"removes freeze and chill on use":       []mod.Mod{},
	"removes poison on use":                 []mod.Mod{},
	"removes shock on use":                  []mod.Mod{},
	"gain unholy might during flask effect": []mod.Mod{FLAG("Condition:UnholyMight").Tag(mod.Condition("UsingFlask"))},
	"zealot's oath during flask effect":     []mod.Mod{FLAG("ZealotsOath").Tag(mod.Condition("UsingFlask"))},
	`grants level (\d+) (.+) curse aura during flask effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraCurse", "LIST", mod.ExtraCurse{SkillName: strings.ReplaceAll(captures[1], " skill", ""), Level: int(num)}).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`shocks nearby enemies during flask effect, causing (\d+)% increased damage taken`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ShockOverride", "BASE", num).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`during flask effect, (\d+)% reduced damage taken of each element for which your uncapped elemental resistance is lowest`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LightningDamageTaken", "INC", -num).Tag(mod.StatThresholdStat("LightningResistTotal", "ColdResistTotal").Upper(true)).Tag(mod.StatThresholdStat("LightningResistTotal", "FireResistTotal").Upper(true)),
			MOD("ColdDamageTaken", "INC", -num).Tag(mod.StatThresholdStat("ColdResistTotal", "LightningResistTotal").Upper(true)).Tag(mod.StatThresholdStat("ColdResistTotal", "FireResistTotal").Upper(true)),
			MOD("FireDamageTaken", "INC", -num).Tag(mod.StatThresholdStat("FireResistTotal", "LightningResistTotal").Upper(true)).Tag(mod.StatThresholdStat("FireResistTotal", "ColdResistTotal").Upper(true)),
		}, ""
	},
	`during flask effect, damage penetrates (\d+)% o?f? ?resistance of each element for which your uncapped elemental resistance is highest`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LightningPenetration", "BASE", num).Tag(mod.StatThresholdStat("LightningResistTotal", "ColdResistTotal")).Tag(mod.StatThresholdStat("LightningResistTotal", "FireResistTotal")),
			MOD("ColdPenetration", "BASE", num).Tag(mod.StatThresholdStat("ColdResistTotal", "LightningResistTotal")).Tag(mod.StatThresholdStat("ColdResistTotal", "FireResistTotal")),
			MOD("FirePenetration", "BASE", num).Tag(mod.StatThresholdStat("FireResistTotal", "LightningResistTotal")).Tag(mod.StatThresholdStat("FireResistTotal", "ColdResistTotal")),
		}, ""
	},
	`recover (\d+)% of life when you kill an enemy during flask effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnKill", "BASE", 1).Tag(mod.PerStat(100/num, "Life")).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`recover (\d+)% of mana when you kill an enemy during flask effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ManaOnKill", "BASE", 1).Tag(mod.PerStat(100/num, "Mana")).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`recover (\d+)% of energy shield when you kill an enemy during flask effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnKill", "BASE", 1).Tag(mod.PerStat(100/num, "EnergyShield")).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`(\d+)% of maximum life taken as chaos damage per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ChaosDegen", "BASE", 1).Tag(mod.PercentStat("Life", num)),
		}, ""
	},
	"your critical strikes do not deal extra damage during flask effect":          []mod.Mod{FLAG("NoCritMultiplier").Tag(mod.Condition("UsingFlask"))},
	"grants perfect agony during flask effect":                                    []mod.Mod{MOD("Keystone", "LIST", "Perfect Agony").Tag(mod.Condition("UsingFlask"))},
	"grants eldritch battery during flask effect":                                 []mod.Mod{MOD("Keystone", "LIST", "Eldritch Battery").Tag(mod.Condition("UsingFlask"))},
	"eldritch battery during flask effect":                                        []mod.Mod{MOD("Keystone", "LIST", "Eldritch Battery").Tag(mod.Condition("UsingFlask"))},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield during effect":        []mod.Mod{mod.NewFlag("ChaosNotBypassEnergyShield", true)},
	"your skills [ch][oa][sv][te] no mana c?o?s?t? ?during flask effect":          []mod.Mod{MOD("ManaCost", "MORE", -100).Tag(mod.Condition("UsingFlask"))},
	"life recovery from flasks also applies to energy shield during flask effect": []mod.Mod{FLAG("LifeFlaskAppliesToEnergyShield").Tag(mod.Condition("UsingFlask"))},
	`consecrated ground created during effect applies (\d+)% increased damage taken to enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTakenConsecratedGround", "INC", num).Tag(mod.Condition("OnConsecratedGround"))}).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	"gain alchemist's genius when you use a flask": []mod.Mod{
		mod.NewFlag("Condition:CanHaveAlchemistGenius", true),
	},
	`(\d+)% chance to gain alchemist's genius when you use a flask`: []mod.Mod{
		mod.NewFlag("Condition:CanHaveAlchemistGenius", true),
	},
	`(\d+)% less flask charges gained from kills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FlaskChargesGained", "MORE", -num).Source("from Kills"),
		}, ""
	},
	`flasks gain (\d+) charges? every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FlaskChargesGenerated", "BASE", num/utils.Float(captures[1])),
		}, ""
	},
	`flasks gain a charge every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FlaskChargesGenerated", "BASE", 1/num),
		}, ""
	},
	`utility flasks gain (\d+) charges? every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("UtilityFlaskChargesGenerated", "BASE", num/utils.Float(captures[1])),
		}, ""
	},
	`life flasks gain (\d+) charges? every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeFlaskChargesGenerated", "BASE", num/utils.Float(captures[1])),
		}, ""
	},
	`mana flasks gain (\d+) charges? every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ManaFlaskChargesGenerated", "BASE", num/utils.Float(captures[1])),
		}, ""
	},
	`flasks gain (\d+) charges? per empty flask slot every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FlaskChargesGeneratedPerEmptyFlask", "BASE", num/utils.Float(captures[1])),
		}, ""
	},
	// Jewels
	`passives in radius of ([\w\s']+) can be allocated without being connected to your tree`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("JewelData", "LIST", mod.JewelData{Key: "impossibleEscapeKeystone", Value: captures[0]}),
			MOD("ImpossibleEscapeKeystones", "LIST", mod.ImpossibleEscapeKeystones{Key: captures[0], Value: true}),
		}, ""
	},
	"passives in radius can be allocated without being connected to your tree": []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "intuitiveLeapLike", Value: true})},
	"affects passives in small ring":                                           []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 4})},
	"affects passives in medium ring":                                          []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 5})},
	"affects passives in large ring":                                           []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 6})},
	"affects passives in very large ring":                                      []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 7})},
	"affects passives in massive ring":                                         []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 8})},
	"only affects passives in small ring":                                      []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 4})},
	"only affects passives in medium ring":                                     []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 5})},
	"only affects passives in large ring":                                      []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 6})},
	"only affects passives in very large ring":                                 []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 7})},
	"only affects passives in massive ring":                                    []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 8})},
	`(\d+)% increased elemental damage per grand spectrum`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ElementalDamage", "INC", num).Tag(mod.Multiplier("GrandSpectrum").Base(0)),
			mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1),
		}, ""
	},
	`gain (\d+) armour per grand spectrum`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Armour", "BASE", num).Tag(mod.Multiplier("GrandSpectrum").Base(0)),
			mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1),
		}, ""
	},
	`+(\d+)% to all elemental resistances per grand spectrum`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ElementalResist", "BASE", num).Tag(mod.Multiplier("GrandSpectrum").Base(0)),
			mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1),
		}, ""
	},
	`gain (\d+) mana per grand spectrum`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Mana", "BASE", num).Tag(mod.Multiplier("GrandSpectrum").Base(0)),
			mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1),
		}, ""
	},
	`(\d+)% increased critical strike chance per grand spectrum`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "INC", num).Tag(mod.Multiplier("GrandSpectrum").Base(0)),
			mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1),
		}, ""
	},
	"primordial": []mod.Mod{mod.NewFloat("Multiplier:PrimordialItem", mod.TypeBase, 1)},
	`spectres have a base duration of (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "duration", Value: 6}).Tag(mod.SkillName("Raise Spectre"))}, ""
	},
	`flasks applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("FlaskEffect", mod.TypeIncrease, num)}, ""
	},
	`magic utility flasks applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MagicUtilityFlaskEffect", mod.TypeIncrease, num)}, ""
	},
	`flasks applied to you have (\d+)% reduced effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffect", "INC", -num)}, ""
	},
	`adds (\d+) passive skills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelNodeCount", Value: num})}, ""
	},
	"1 added passive skill is a jewel socket": []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelSocketCount", Value: 1})},
	`(\d+) added passive skills are jewel sockets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelSocketCount", Value: num})}, ""
	},
	`adds (\d+) jewel socket passive skills`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelSocketCountOverride", Value: num})}, ""
	},
	`adds (\d+) small passive skills? which grants? nothing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelNothingnessCount", Value: num})}, ""
	},
	"added small passive skills grant nothing": []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelSmallsAreNothingness", Value: true})},
	`added small passive skills have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "clusterJewelIncEffect", Value: num})}, ""
	},
	`this jewel's socket has (\d+)% increased effect per allocated passive skill between it and your class' starting location`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("JewelData", "LIST", mod.JewelData{Key: "jewelIncEffectFromClassStart", Value: num}),
		}, ""
	},
	// Misc
	`leftmost (\d+) magic utility flasks constantly apply their flask effects to you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ActiveMagicUtilityFlasks", mod.TypeBase, num)}, ""
	},
	`marauder: melee skills have (\d+)% increased area of effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("ConnectedToMarauderStart")).Tag(mod.SkillType(string(data.SkillTypeMelee)))}, ""
	},
	"intelligence provides no inherent bonus to energy shield": []mod.Mod{mod.NewFlag("NoIntBonusToES", true)},
	"intelligence is added to accuracy rating with wands":      []mod.Mod{MOD("Accuracy", "BASE", 1).Flag(mod.MFlagWand).Tag(mod.PerStat(0, "Int"))},
	`dexterity's accuracy bonus instead grants \+(\d+) to accuracy rating per dexterity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DexAccBonusOverride", "OVERRIDE", num)}, ""
	},
	"cannot recover energy shield to above armour":         []mod.Mod{mod.NewFlag("ArmourESRecoveryCap", true)},
	"cannot recover energy shield to above evasion rating": []mod.Mod{mod.NewFlag("EvasionESRecoveryCap", true)},
	`warcries exert (\d+) additional attacks?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ExtraExertedAttacks", mod.TypeBase, num)}, ""
	},
	"iron will":                      []mod.Mod{mod.NewFlag("IronWill", true)},
	"iron reflexes while stationary": []mod.Mod{MOD("Keystone", "LIST", "Iron Reflexes").Tag(mod.Condition("Stationary"))},
	"you have zealot's oath if you haven't been hit recently": []mod.Mod{MOD("Keystone", "LIST", "Zealot's Oath").Tag(mod.Condition("BeenHitRecently").Neg(true))},
	"deal no physical damage":                                 []mod.Mod{mod.NewFlag("DealNoPhysical", true)},
	"deal no cold damage":                                     []mod.Mod{mod.NewFlag("DealNoCold", true)},
	"deal no fire damage":                                     []mod.Mod{mod.NewFlag("DealNoFire", true)},
	"deal no lightning damage":                                []mod.Mod{mod.NewFlag("DealNoLightning", true)},
	"deal no elemental damage":                                []mod.Mod{mod.NewFlag("DealNoLightning", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoFire", true)},
	"deal no chaos damage":                                    []mod.Mod{mod.NewFlag("DealNoChaos", true)},
	"deal no damage":                                          []mod.Mod{mod.NewFlag("DealNoLightning", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoFire", true), mod.NewFlag("DealNoChaos", true), mod.NewFlag("DealNoPhysical", true)},
	"deal no non-elemental damage":                            []mod.Mod{mod.NewFlag("DealNoPhysical", true), mod.NewFlag("DealNoChaos", true)},
	"deal no non-lightning damage":                            []mod.Mod{mod.NewFlag("DealNoPhysical", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoFire", true), mod.NewFlag("DealNoChaos", true)},
	"deal no non-physical damage":                             []mod.Mod{mod.NewFlag("DealNoLightning", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoFire", true), mod.NewFlag("DealNoChaos", true)},
	"cannot deal non-chaos damage":                            []mod.Mod{mod.NewFlag("DealNoPhysical", true), mod.NewFlag("DealNoCold", true), mod.NewFlag("DealNoFire", true), mod.NewFlag("DealNoLightning", true)},
	"deal no damage when not on low life": []mod.Mod{
		FLAG("DealNoLightning").Tag(mod.Condition("LowLife").Neg(true)),
		FLAG("DealNoCold").Tag(mod.Condition("LowLife").Neg(true)),
		FLAG("DealNoFire").Tag(mod.Condition("LowLife").Neg(true)),
		FLAG("DealNoChaos").Tag(mod.Condition("LowLife").Neg(true)),
		FLAG("DealNoPhysical").Tag(mod.Condition("LowLife").Neg(true)),
	},
	"attacks have blood magic":          []mod.Mod{FLAG("SkillBloodMagic").Flag(mod.MFlagAttack)},
	"attacks cost life instead of mana": []mod.Mod{FLAG("SkillBloodMagic").Flag(mod.MFlagAttack)},
	`(\d+)% chance to cast a? ?socketed lightning spells? on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
		}, ""
	},
	"cast a socketed lightning spell on hit":                                                                        []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"trigger a socketed lightning spell on hit":                                                                     []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed lightning spell on hit, with a ([\d\.]+) second cooldown`:                                   []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueMjolnerLightningSpellsCastOnHit", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"[ct][ar][si][tg]g?e?r? a socketed cold s[pk][ei]ll on melee critical strike":                                   []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`[ct][ar][si][tg]g?e?r? a socketed cold s[pk][ei]ll on melee critical strike, with a ([\d\.]+) second cooldown`: []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueCosprisMaliceColdSpellsCastOnMeleeCriticalStrike", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"your curses can apply to hexproof enemies":                                                                     []mod.Mod{mod.NewFlag("CursesIgnoreHexproof", true)},
	"your hexes can affect hexproof enemies":                                                                        []mod.Mod{mod.NewFlag("CursesIgnoreHexproof", true)},
	`hexes from socketed skills can apply (\d) additional curses`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SocketedCursesHexLimitValue", mod.TypeBase, num), FLAG("SocketedCursesAdditionalLimit").Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	// This is being changed from ignoreHexLimit to SocketedCursesAdditionalLimit due to patch 3.16.0, which states that legacy versions "will be affected by this Curse Limit change,
	// though they will only have 20% less Curse Effect of Curses triggered with Summon Doedre’s Effigy."
	// Legacy versions will still show that "Hexes from Socketed Skills ignore Curse limit", but will instead have an internal limit of 5 to match the current functionality.
	"hexes from socketed skills ignore curse limit": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SocketedCursesHexLimitValue", mod.TypeBase, 5), FLAG("SocketedCursesAdditionalLimit").Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`reserves (\d+)% of life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ExtraLifeReserved", mod.TypeBase, num)}, ""
	},
	`(\d+)% of cold damage taken as lightning`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ColdDamageTakenAsLightning", mod.TypeBase, num)}, ""
	},
	`(\d+)% of fire damage taken as lightning`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("FireDamageTakenAsLightning", mod.TypeBase, num)}, ""
	},
	`items and gems have (\d+)% reduced attribute requirements`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GlobalAttributeRequirements", "INC", -num)}, ""
	},
	`items and gems have (\d+)% increased attribute requirements`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("GlobalAttributeRequirements", mod.TypeIncrease, num)}, ""
	},
	`mana reservation of herald skills is always (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationPercentForced", Value: num}).Tag(mod.SkillType(string(data.SkillTypeHerald)))}, ""
	},
	`([\w\s]+) reserves no mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillData", "LIST", mod.SkillData{Key: "manaReservationFlat", Value: 0}).Tag(mod.SkillIdByName(captures[0])),
			MOD("SkillData", "LIST", mod.SkillData{Key: "lifeReservationFlat", Value: 0}).Tag(mod.SkillIdByName(captures[0])),
			MOD("SkillData", "LIST", mod.SkillData{Key: "manaReservationPercent", Value: 0}).Tag(mod.SkillIdByName(captures[0])),
			MOD("SkillData", "LIST", mod.SkillData{Key: "lifeReservationPercent", Value: 0}).Tag(mod.SkillIdByName(captures[0])),
		}, ""
	},
	`([\w\s]+) has no reservation`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillData", "LIST", mod.SkillData{Key: "manaReservationFlat", Value: 0}).Tag(mod.SkillIdByName(captures[0])),
			MOD("SkillData", "LIST", mod.SkillData{Key: "lifeReservationFlat", Value: 0}).Tag(mod.SkillIdByName(captures[0])),
			MOD("SkillData", "LIST", mod.SkillData{Key: "manaReservationPercent", Value: 0}).Tag(mod.SkillIdByName(captures[0])),
			MOD("SkillData", "LIST", mod.SkillData{Key: "lifeReservationPercent", Value: 0}).Tag(mod.SkillIdByName(captures[0])),
		}, ""
	},
	`([\w\s]+) has no reservation if cast as an aura`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillData", "LIST", mod.SkillData{Key: "manaReservationFlat", Value: 0}).Tag(mod.SkillIdByName(captures[0])).Tag(mod.SkillType(string(data.SkillTypeAura))),
			MOD("SkillData", "LIST", mod.SkillData{Key: "lifeReservationFlat", Value: 0}).Tag(mod.SkillIdByName(captures[0])).Tag(mod.SkillType(string(data.SkillTypeAura))),
			MOD("SkillData", "LIST", mod.SkillData{Key: "manaReservationPercent", Value: 0}).Tag(mod.SkillIdByName(captures[0])).Tag(mod.SkillType(string(data.SkillTypeAura))),
			MOD("SkillData", "LIST", mod.SkillData{Key: "lifeReservationPercent", Value: 0}).Tag(mod.SkillIdByName(captures[0])).Tag(mod.SkillType(string(data.SkillTypeAura))),
		}, ""
	},
	"banner skills reserve no mana": []mod.Mod{
		MOD("SkillData", "LIST", mod.SkillData{Key: "manaReservationPercent", Value: 0}).Tag(mod.SkillType(string(data.SkillTypeBanner))),
		MOD("SkillData", "LIST", mod.SkillData{Key: "lifeReservationPercent", Value: 0}).Tag(mod.SkillType(string(data.SkillTypeBanner))),
	},
	"banner skills have no reservation": []mod.Mod{
		MOD("SkillData", "LIST", mod.SkillData{Key: "manaReservationPercent", Value: 0}).Tag(mod.SkillType(string(data.SkillTypeBanner))),
		MOD("SkillData", "LIST", mod.SkillData{Key: "lifeReservationPercent", Value: 0}).Tag(mod.SkillType(string(data.SkillTypeBanner))),
	},
	`placed banners also grant (\d+)% increased attack damage to you and allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAuraEffect", "LIST", mod.ExtraAuraEffect{Mod: MOD("Damage", "INC", num).Flag(mod.MFlagAttack)}).Tag(mod.Condition("BannerPlanted")).Tag(mod.SkillType(string(data.SkillTypeBanner))),
		}, ""
	},
	`dread banner grants an additional \+(\d+) to maximum fortification when placing the banner`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("MaximumFortification", "BASE", num).Tag(mod.GlobalEffect("Buff"))}).Tag(mod.Condition("BannerPlanted")).Tag(mod.SkillName("Dread Banner")),
		}, ""
	},
	"your aura skills are disabled": []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeAura)))},
	"your spells are disabled":      []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeSpell)))},
	`aura skills other than ([\w\s]+) are disabled`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeAura))),
			FLAG("EnableSkill").Tag(mod.SkillIdByName(captures[0])),
		}, ""
	},
	`travel skills other than ([\w\s]+) are disabled`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeTravel))),
			FLAG("EnableSkill").Tag(mod.SkillIdByName(captures[0])),
		}, ""
	},
	`strength's damage bonus instead grants (\d+)% increased melee physical damage per (\d+) strength`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("StrDmgBonusRatioOverride", "BASE", num/utils.Float(captures[1]))}, ""
	},
	`while in her embrace, take ([\d\.]+)% of your total maximum life and energy shield as fire damage per second per level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FireDegen", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.Multiplier("Level").Base(0)).Tag(mod.Condition("HerEmbrace")),
			MOD("FireDegen", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)).Tag(mod.Multiplier("Level").Base(0)).Tag(mod.Condition("HerEmbrace")),
		}, ""
	},
	`gain her embrace for \d+ seconds when you ignite an enemy`: []mod.Mod{mod.NewFlag("Condition:CanGainHerEmbrace", true)},
	`when you cast a spell, sacrifice all mana to gain added maximum lightning damage equal to (\d+)% of sacrificed mana for 4 seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFlag("Condition:HaveManaStorm", true),
			MOD("LightningMax", "BASE", 1).Tag(mod.PerStat(100/num, "ManaUnreserved")).Tag(mod.Condition("SacrificeManaForLightning")),
		}, ""
	},
	`gain added chaos damage equal to (\d+)% of ward`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ChaosMin", "BASE", 1).Tag(mod.PerStat(100/num, "Ward")),
			MOD("ChaosMax", "BASE", 1).Tag(mod.PerStat(100/num, "Ward")),
		}, ""
	},
	"every 16 seconds you gain iron reflexes for 8 seconds": []mod.Mod{
		mod.NewFlag("Condition:HaveArborix", true),
	},
	"every 16 seconds you gain elemental overload for 8 seconds": []mod.Mod{
		mod.NewFlag("Condition:HaveAugyre", true),
	},
	"every 8 seconds, gain avatar of fire for 4 seconds": []mod.Mod{
		mod.NewFlag("Condition:HaveVulconus", true),
	},
	"modifiers to attributes instead apply to omniscience": []mod.Mod{mod.NewFlag("Omniscience", true)},
	`attribute requirements can be satisfied by (\d+)% of omniscience`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("OmniAttributeRequirements", mod.TypeIncrease, num),
			mod.NewFlag("OmniscienceRequirements", true),
		}, ""
	},
	"you have far shot while you do not have iron reflexes":                []mod.Mod{FLAG("FarShot").Tag(mod.Condition("HaveIronReflexes").Neg(true))},
	"you have resolute technique while you do not have elemental overload": []mod.Mod{MOD("Keystone", "LIST", "Resolute Technique").Tag(mod.Condition("HaveElementalOverload").Neg(true))},
	"hits ignore enemy monster fire resistance while you are ignited":      []mod.Mod{FLAG("IgnoreFireResistance").Tag(mod.Condition("Ignited"))},
	"your hits can't be evaded by blinded enemies":                         []mod.Mod{FLAG("CannotBeEvaded").Tag(mod.ActorCondition("enemy", "Blinded"))},
	"blind does not affect your chance to hit":                             []mod.Mod{mod.NewFlag("IgnoreBlindHitChance", true)},
	"enemies blinded by you while you are blinded have malediction":        []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMalediction").Tag(mod.Condition("Blinded"))}).Tag(mod.Condition("Blinded"))},
	"skills which throw traps have blood magic":                            []mod.Mod{FLAG("BloodMagic").Tag(mod.SkillType(string(data.SkillTypeTrapped)))},
	"skills which throw traps cost life instead of mana":                   []mod.Mod{FLAG("BloodMagic").Tag(mod.SkillType(string(data.SkillTypeTrapped)))},
	`lose ([\d\.]+) mana per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ManaDegen", mod.TypeBase, num)}, ""
	},
	`lose ([\d\.]+)% of maximum mana per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaDegen", "BASE", 1).Tag(mod.PercentStat("Mana", num))}, ""
	},
	"strength provides no bonus to maximum life":     []mod.Mod{mod.NewFlag("NoStrBonusToLife", true)},
	"intelligence provides no bonus to maximum mana": []mod.Mod{mod.NewFlag("NoIntBonusToMana", true)},
	`with a ghastly eye jewel socketed, minions have \+(\d+) to accuracy rating`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Accuracy", mod.TypeBase, num)}).Tag(mod.Condition("HaveGhastlyEyeJewelIn{SlotName}")),
		}, ""
	},
	"hits ignore enemy monster chaos resistance if all equipped items are shaper items": []mod.Mod{FLAG("IgnoreChaosResistance").Tag(mod.MultiplierThreshold("NonShaperItem").Threshold(0).Upper(true))},
	"hits ignore enemy monster chaos resistance if all equipped items are elder items":  []mod.Mod{FLAG("IgnoreChaosResistance").Tag(mod.MultiplierThreshold("NonElderItem").Threshold(0).Upper(true))},
	`gain \d+ rage on critical hit with attacks, no more than once every [\d\.]+ seconds`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`warcry skills' cooldown time is (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "OVERRIDE", num).KeywordFlag(mod.KeywordFlagWarcry)}, ""
	},
	`warcry skills have (\+\d+) seconds to cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "BASE", num).KeywordFlag(mod.KeywordFlagWarcry)}, ""
	},
	"using warcries is instant": []mod.Mod{mod.NewFlag("InstantWarcry", true)},
	`attacks with axes or swords grant (\d+) rage on hit, no more than once every second`: []mod.Mod{
		FLAG("Condition:CanGainRage").Tag(mod.Condition("UsingAxe", "UsingSword")),
	},
	`your critical strike multiplier is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("CritMultiplier", mod.TypeOverride, num)}, ""
	},
	`base critical strike chance for attacks with weapons is ([\d\.]+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("WeaponBaseCritChance", mod.TypeOverride, num)}, ""
	},
	`critical strike chance is (\d+)% for hits with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "OVERRIDE", num).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	"allocates (.+) if you have the matching modifiers? on forbidden (.+)": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("GrantedAscendancyNode", "LIST", mod.GrantedAscendancyNode{Side: captures[1], Name: captures[0]}),
		}, ""
	},
	"allocates (.+)": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GrantedPassive", "LIST", captures[0])}, ""
	},
	"battlemage":              []mod.Mod{mod.NewFlag("WeaponDamageAppliesToSpells", true), MOD("ImprovedWeaponDamageAppliesToSpells", "MAX", 100)},
	"transfiguration of body": []mod.Mod{mod.NewFlag("TransfigurationOfBody", true)},
	"transfiguration of mind": []mod.Mod{mod.NewFlag("TransfigurationOfMind", true)},
	"transfiguration of soul": []mod.Mod{mod.NewFlag("TransfigurationOfSoul", true)},
	`offering skills have (\d+)% reduced duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Duration", "INC", -num).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering")),
		}, ""
	},
	`enemies have -(\d+)% to total physical damage reduction against your hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyPhysicalDamageReduction", "BASE", -num),
		}, ""
	},
	`enemies you impale have -(\d+)% to total physical damage reduction against impale hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyImpalePhysicalDamageReduction", "BASE", -num),
		}, ""
	},
	`hits with this weapon overwhelm (\d+)% physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyPhysicalDamageReduction", "BASE", -num).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`overwhelm (\d+)% physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyPhysicalDamageReduction", "BASE", -num),
		}, ""
	},
	`impale damage dealt to enemies impaled by you overwhelms (\d+)% physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyImpalePhysicalDamageReduction", "BASE", -num),
		}, ""
	},
	`nearby enemies are crushed while you have ?a?t? least (\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			// MultiplierThreshold is on RageStacks because Rage is only set in CalcPerform if Condition:CanGainRage is true, Bear's Girdle does not flag CanGainRage
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Crushed", true)}).Tag(mod.MultiplierThreshold("RageStack").Threshold(num)),
		}, ""
	},
	"you are crushed":                              []mod.Mod{mod.NewFlag("Condition:Crushed", true)},
	"nearby enemies are crushed":                   []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Crushed", true)})},
	"crush enemies on hit with maces and sceptres": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Crushed", true)}).Tag(mod.Condition("UsingMace"))},
	"enemies on fungal ground you kill explode, dealing 5% of their life as chaos damage": []mod.Mod{},
	"you have fungal ground around you while stationary": []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("NonChaosDamageGainAsChaos", mod.TypeBase, 10)}).Tag(mod.Condition("OnFungalGround", "Stationary")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("Damage", "MORE", -10)}).Tag(mod.ActorCondition("enemy", "OnFungalGround", "Stationary")),
	},
	"create profane ground instead of consecrated ground": []mod.Mod{
		mod.NewFlag("Condition:CreateProfaneGround", true),
	},
	"you count as dual wielding while you are unencumbered":                 []mod.Mod{FLAG("Condition:DualWielding").Tag(mod.Condition("Unencumbered"))},
	"dual wielding does not inherently grant chance to block attack damage": []mod.Mod{mod.NewFlag("Condition:NoInherentBlock", true)},
	"you do not inherently take less damage for having fortification":       []mod.Mod{mod.NewFlag("Condition:NoFortificationMitigation", true)},
	`skills supported by intensify have \+(\d) to maximum intensity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("Multiplier:IntensityLimit", mod.TypeBase, num)}, ""
	},
	`spells which can gain intensity have \+(\d) to maximum intensity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("Multiplier:IntensityLimit", mod.TypeBase, num)}, ""
	},
	`hexes you inflict have \+(\d+) to maximum doom`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MaxDoom", mod.TypeBase, num)}, ""
	},
	`while stationary, gain (\d+)% increased area of effect every second, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AreaOfEffect", "INC", num).Tag(mod.Multiplier("StationarySeconds").GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("ExpansiveMight")).Tag(mod.Condition("Stationary")),
		}, ""
	},
	`attack skills have added lightning damage equal to (\d+)% of maximum mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LightningMin", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.PerStat(100/num, "Mana")),
			MOD("LightningMax", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.PerStat(100/num, "Mana")),
		}, ""
	},
	`herald of thunder's storms hit enemies with (\d+)% increased frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("HeraldStormFrequency", mod.TypeIncrease, num)}, ""
	},
	`your critical strikes have a (\d+)% chance to deal double damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("DoubleDamageChanceOnCrit", mod.TypeBase, num)}, ""
	},
	`(\d+)% chance to deal triple damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("TripleDamageChance", mod.TypeBase, num)}, ""
	},
	"elemental skills deal triple damage":      []mod.Mod{MOD("TripleDamageChance", "BASE", 100).Tag(mod.SkillType(string(data.SkillTypeCold))).Tag(mod.SkillType(string(data.SkillTypeFire))).Tag(mod.SkillType(string(data.SkillTypeLightning)))},
	"deal triple damage with elemental skills": []mod.Mod{MOD("TripleDamageChance", "BASE", 100).Tag(mod.SkillType(string(data.SkillTypeCold))).Tag(mod.SkillType(string(data.SkillTypeFire))).Tag(mod.SkillType(string(data.SkillTypeLightning)))},
	`skills supported by unleash have \+(\d) to maximum number of seals`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SealCount", mod.TypeBase, num)}, ""
	},
	`skills supported by unleash have (\d+)% increased seal gain frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SealGainFrequency", mod.TypeIncrease, num)}, ""
	},
	`(\d+)% increased critical strike chance with spells which remove the maximum number of seals`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MaxSealCrit", mod.TypeIncrease, num)}, ""
	},
	"gain elusive on critical strike": []mod.Mod{
		mod.NewFlag("Condition:CanBeElusive", true),
	},
	`nearby enemies have (\w+) resistance equal to yours`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("Enemy"+(utils.Capital(captures[0]))+"ResistEqualToYours", true)}, ""
	},
	`for each nearby corpse, regenerate ([\d\.]+)% life per second, up to ([\d\.]+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegenPercent", "BASE", num).Tag(mod.Multiplier("NearbyCorpse").Limit(utils.Float(captures[1])).LimitTotal(true))}, ""
	},
	`gain sacrificial zeal when you use a skill, dealing you \d+% of the skill's mana cost as physical damage per second`: []mod.Mod{
		mod.NewFlag("Condition:SacrificialZeal", true),
	},
	`hits overwhelm (\d+)% of physical damage reduction while you have sacrificial zeal`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyPhysicalDamageReduction", "BASE", -num).Tag(mod.Condition("SacrificialZeal")),
		}, ""
	},
	`minions attacks overwhelm (\d+)% physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnemyPhysicalDamageReduction", "BASE", -num)}),
		}, ""
	},
	`focus has (\d+)% increased cooldown recovery rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FocusCooldownRecovery", "INC", num).Tag(mod.Condition("Focused"))}, ""
	},
	`(\d+)% chance to deal double damage with attacks if attack time is longer than 1 second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DoubleDamageChance", "BASE", num).Tag(mod.Condition("OneSecondAttackTime")),
		}, ""
	},
	`elusive also grants \+(\d+)% to critical strike multiplier for skills supported by nightblade`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("NightbladeElusiveCritMultiplier", mod.TypeBase, num)}, ""
	},
	`skills supported by nightblade have (\d+)% increased effect of elusive`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("NightbladeSupportedElusiveEffect", mod.TypeIncrease, num)}, ""
	},
	"nearby enemies are scorched": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Scorched", true)}),
		mod.NewFloat("ScorchBase", mod.TypeBase, 10),
	},
	// Pantheon: Soul of Tukohama support
	`while stationary, gain ([\d\.]+)% of life regenerated per second every second, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeRegenPercent", "BASE", num).Tag(mod.Multiplier("StationarySeconds").Limit(utils.Float(captures[1])).LimitTotal(true)).Tag(mod.Condition("Stationary")),
		}, ""
	},
	// Pantheon: Soul of Tukohama support
	`while stationary, gain (\d+)% additional physical damage reduction every second, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalDamageReduction", "BASE", num).Tag(mod.Multiplier("StationarySeconds").Limit(utils.Float(captures[1])).LimitTotal(true)).Tag(mod.Condition("Stationary")),
		}, ""
	},
	// Skill-specific enchantment modifiers
	`(\d+)% increased decoy totem life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TotemLife", "INC", num).Tag(mod.SkillName("Decoy Totem"))}, ""
	},
	`(\d+)% increased ice spear critical strike chance in second form`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "INC", num).Tag(mod.SkillName("Ice Spear")).Tag(mod.SkillPart(2)),
		}, ""
	},
	`shock nova ring deals (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "INC", num).Tag(mod.SkillName("Shock Nova")).Tag(mod.SkillPart(1)),
		}, ""
	},
	`enemies affected by bear trap take (\d+)% increased damage from trap or mine hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("TrapMineDamageTaken", "INC", num).Tag(mod.GlobalEffect("Debuff"))}).Tag(mod.SkillName("Bear Trap")),
		}, ""
	},
	`blade vortex has \+(\d+)% to critical strike multiplier for each blade`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritMultiplier", "BASE", num).Tag(mod.Multiplier("BladeVortexBlade").Base(0), mod.SkillName("Blade Vortex")),
		}, ""
	},
	`burning arrow has (\d+)% increased debuff effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DebuffEffect", "INC", num).Tag(mod.SkillName("Burning Arrow")),
		}, ""
	},
	`double strike has a (\d+)% chance to deal double damage to bleeding enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DoubleDamageChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Bleeding"), mod.SkillName("Double Strike")),
		}, ""
	},
	`frost bomb has (\d+)% increased debuff duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SecondaryDuration", "INC", num).Tag(mod.SkillName("Frost Bomb"))}, ""
	},
	`incinerate has \+(\d+) to maximum stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:IncinerateMaxStages", "BASE", num).Tag(mod.SkillName("Incinerate"))}, ""
	},
	`perforate creates \+(\d+) spikes?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("Multiplier:PerforateMaxSpikes", mod.TypeBase, num)}, ""
	},
	`scourge arrow has (\d+)% chance to poison per stage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PoisonChance", "BASE", num).Tag(mod.SkillName("Scourge Arrow"), mod.Multiplier("ScourgeArrowStage").Base(0))}, ""
	},
	`winter orb has \+(\d+) maximum stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("Multiplier:WinterOrbMaxStages", mod.TypeBase, num)}, ""
	},
	`summoned holy relics have (\d+)% increased buff effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BuffEffect", "INC", num).Tag(mod.SkillName("Summon Holy Relic"))}, ""
	},
	`\+(\d) to maximum virulence`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("Multiplier:VirulenceStacksMax", mod.TypeBase, num)}, ""
	},
	`winter orb has (\d+)% increased area of effect per stage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.SkillName("Winter Orb"), mod.Multiplier("WinterOrbStage").Base(0))}, ""
	},
	`wintertide brand has \+(\d+) to maximum stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:WintertideBrandMaxStages", "BASE", num).Tag(mod.SkillName("Wintertide Brand"))}, ""
	},
	`wave of conviction's exposure applies (-\d+)% elemental resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "purge_expose_resist_%_matching_highest_element_damage", Value: num}).Tag(mod.SkillName("Wave of Conviction")),
		}, ""
	},
	`arcane cloak spends an additional (\d+)% of current mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "arcane_cloak_consume_%_of_mana", Value: num}).Tag(mod.SkillName("Arcane Cloak"))}, ""
	},
	`arcane cloak grants life regeneration equal to (\d+)% of mana spent per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("LifeRegen", "BASE", num/100).Tag(mod.Multiplier("ArcaneCloakConsumedMana").Base(0), mod.GlobalEffect("Buff"))}).Tag(mod.SkillName("Arcane Cloak")),
		}, ""
	},
	`caustic arrow has (\d+)% chance to inflict withered on hit for (\d+) seconds base duration`: []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFlag("Condition:CanWither", true)}).Tag(mod.SkillName("Caustic Arrow"))},
	`venom gyre has a (\d+)% chance to inflict withered for (\d+) seconds on hit`:                []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFlag("Condition:CanWither", true)}).Tag(mod.SkillName("Venom Gyre"))},
	`sigil of power's buff also grants (\d+)% increased critical strike chance per stage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "INC", num).Tag(mod.Multiplier("SigilOfPowerStage").Limit(4), mod.GlobalEffect("Buff").Name("Sigil of Power")),
		}, ""
	},
	`cobra lash chains (\d+) additional times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("ChainCountMax", mod.TypeBase, num)}).Tag(mod.SkillName("Cobra Lash"))}, ""
	},
	`general's cry has ([\+\-]\d) to maximum number of mirage warriors`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("GeneralsCryDoubleMaxCount", mod.TypeBase, num)}, ""
	},
	`([\+\-]\d) to maximum blade flurry stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("Multiplier:BladeFlurryMaxStages", mod.TypeBase, num)}, ""
	},
	`steelskin buff can take (\d+)% increased amount of damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "steelskin_damage_limit_+%", Value: num}).Tag(mod.SkillName("Steelskin"))}, ""
	},
	`hydrosphere has (\d+)% increased pulse frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("HydroSphereFrequency", mod.TypeIncrease, num)}, ""
	},
	`void sphere has (\d+)% increased pulse frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("VoidSphereFrequency", mod.TypeIncrease, num)}, ""
	},
	`shield crush central wave has (\d+)% more area of effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "MORE", num).Tag(mod.SkillName("Shield Crush"), mod.SkillPart(2))}, ""
	},
	`storm rain has (\d+)% increased beam frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("StormRainBeamFrequency", mod.TypeIncrease, num)}, ""
	},
	`voltaxic burst deals (\d+)% increased damage per ([\d\.]+) seconds of duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("VoltaxicDurationIncDamage", mod.TypeIncrease, num)}, ""
	},
	`earthquake deals (\d+)% increased damage per ([\d\.]+) seconds duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("EarthquakeDurationIncDamage", mod.TypeIncrease, num)}, ""
	},
	`consecrated ground from holy flame totem applies (\d+)% increased damage taken to enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTakenConsecratedGround", "INC", num).Tag(mod.Condition("OnConsecratedGround"))}),
		}, ""
	},
	`consecrated ground from purifying flame applies (\d+)% increased damage taken to enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "consecrated_ground_enemy_damage_taken_+%", Value: num}).Tag(mod.SkillName("Purifying Flame")),
		}, ""
	},
	`enemies drenched by hydrosphere have cold and lightning exposure, applying (-\d+)% to resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "water_sphere_cold_lightning_exposure_%", Value: num}).Tag(mod.SkillName("Hydrosphere")),
		}, ""
	},
	`frost shield has \+(\d+) to maximum life per stage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "frost_globe_health_per_stage", Value: num}).Tag(mod.SkillName("Frost Shield")),
		}, ""
	},
	`flame wall grants (\d+) to (\d+) added fire damage to projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "flame_wall_minimum_added_fire_damage", Value: num}).Tag(mod.SkillName("Flame Wall")),
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "flame_wall_maximum_added_fire_damage", Value: utils.Float(captures[1])}).Tag(mod.SkillName("Flame Wall")),
		}, ""
	},
	`plague bearer buff grants \+(\d+)% to poison damage over time multiplier while infecting`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "corrosive_shroud_poison_dot_multiplier_+_while_aura_active", Value: num}).Tag(mod.SkillName("Plague Bearer")),
		}, ""
	},
	`(\d+)% increased lightning trap lightning ailment effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "shock_effect_+%", Value: num}).Tag(mod.SkillName("Lightning Trap"))}, ""
	},
	`wild strike's beam chains an additional (\d+) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("ChainCountMax", mod.TypeBase, num)}).Tag(mod.SkillName("Wild Strike"), mod.SkillPart(4))}, ""
	},
	`energy blades have (\d+)% increased attack speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("EnergyBladeAttackSpeed", mod.TypeIncrease, num)}, ""
	},
	`ensnaring arrow has (\d+)% increased debuff effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DebuffEffect", "INC", num).Tag(mod.SkillName("Ensnaring Arrow"))}, ""
	},
	`unearth spawns corpses with ([\+\-]\d) level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CorpseLevel", "BASE", num).Tag(mod.SkillName("Unearth"))}, ""
	},
	// Alternate Quality
	"quality does not increase physical damage": []mod.Mod{mod.NewFloat("AlternateQualityWeapon", mod.TypeBase, 1)},
	`(\d+)% increased critical strike chance per 4% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("AlternateQualityLocalCritChancePer4Quality", mod.TypeIncrease, num)}, ""
	},
	`grants (\d+)% increased accuracy per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Accuracy", "INC", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1]))),
		}, ""
	},
	`(\d+)% increased attack speed per 8% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("AlternateQualityLocalAttackSpeedPer8Quality", mod.TypeIncrease, num)}, ""
	},
	`\+(\d+) weapon range per 10% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("AlternateQualityLocalWeaponRangePer10Quality", mod.TypeBase, num)}, ""
	},
	`grants (\d+)% increased elemental damage per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalDamage", "INC", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`grants (\d+)% increased area of effect per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	"quality does not increase defences": []mod.Mod{mod.NewFloat("AlternateQualityArmour", mod.TypeBase, 1)},
	`grants \+(\d+) to maximum life per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Life", "BASE", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1])))}, ""
	},
	`grants \+(\d+) to maximum mana per (\d+)% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Mana", "BASE", num).Tag(mod.Multiplier("QualityOn{SlotName}").Div(utils.Float(captures[1]))),
		}, ""
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
		return []mod.Mod{mod.NewFloat("Quality", mod.TypeBase, num)}, ""
	},
	`infernal blow debuff deals an additional (\d+)% of damage per charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DebuffEffect", "BASE", num).Tag(mod.SkillName("Infernal Blow")),
		}, ""
	},
	// Display-only modifiers
	"extra gore": []mod.Mod{},
	"prefixes:":  []mod.Mod{},
	"suffixes:":  []mod.Mod{},
	"while your passive skill tree connects to a class' starting location, you gain:":       []mod.Mod{},
	`socketed lightning spells [hd][ae][va][el] (\d+)% increased spell damage if triggered`: []mod.Mod{},
	"manifeste?d? dancing dervishe?s? disables both weapon slots":                           []mod.Mod{},
	"manifeste?d? dancing dervishe?s? dies? when rampage ends":                              []mod.Mod{},
	// Legion modifiers
	`bathed in the blood of (\d+) sacrificed in the name of (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "conqueredBy", Value: mod.LegionJewel{ID: int(num), Conqueror: conquerorList[strings.ToLower(captures[1])]}})}, ""
	},
	`carved to glorify (\d+) new faithful converted by high templar (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "conqueredBy", Value: mod.LegionJewel{ID: int(num), Conqueror: conquerorList[strings.ToLower(captures[1])]}})}, ""
	},
	`commanded leadership over (\d+) warriors under (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "conqueredBy", Value: mod.LegionJewel{ID: int(num), Conqueror: conquerorList[strings.ToLower(captures[1])]}})}, ""
	},
	`commissioned (\d+) coins to commemorate (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "conqueredBy", Value: mod.LegionJewel{ID: int(num), Conqueror: conquerorList[strings.ToLower(captures[1])]}})}, ""
	},
	`denoted service of (\d+) dekhara in the akhara of (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "conqueredBy", Value: mod.LegionJewel{ID: int(num), Conqueror: conquerorList[strings.ToLower(captures[1])]}})}, ""
	},
	`passives in radius are conquered by the (\D+)`: []mod.Mod{},
	"historic": []mod.Mod{},
	"survival": []mod.Mod{},
	"you can have two different banners at the same time": []mod.Mod{},
	"can have a second enchantment modifier":              []mod.Mod{},
	`can have (\d+) additional enchantment modifiers`:     []mod.Mod{},
	"this item can be anointed by cassia":                 []mod.Mod{},
	"all sockets are white":                               []mod.Mod{},
	`every (\d+) seconds, regenerate (\d+)% of life over one second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeRegenPercent", "BASE", utils.Float(captures[1])).Tag(mod.Condition("LifeRegenBurstFull")),
			MOD("LifeRegenPercent", "BASE", utils.Float(captures[1])/num).Tag(mod.Condition("LifeRegenBurstAvg")),
		}, ""
	},
	`you take (\d+)% reduced extra damage from critical strikes`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ReduceCritExtraDamage", mod.TypeBase, num)}, ""
	},
	`you take (\d+)% reduced extra damage from critical strikes while you have no power charges`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.StatThreshold("PowerCharges", 0).Upper(true)),
		}, ""
	},
	`you take (\d+)% reduced extra damage from critical strikes by poisoned enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.ActorCondition("enemy", "Poisoned"))}, ""
	},
	`nearby allies have (\d+)% chance to block attack damage per (\d+) strength you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{OnlyAllies: true, Mod: mod.NewFloat("BlockChance", mod.TypeBase, num)}).Tag(mod.PerStat(utils.Float(captures[1]), "Str"))}, ""
	},
}

/*
TODO
for _, name in pairs(data.keystones) do
	specialModList[name:lower()] = { MOD("Keystone", "LIST", name) }
end
local oldList = specialModList
specialModList = { }
for k, v in pairs(oldList) do
	specialModList["^"..k.."$"] = v
end
*/

// Modifiers that are recognised but unsupported
var unsupportedModList = map[string]bool{
	"properties are doubled while in a breach": true,
	"mirrored": true,
	"split":    true,
}

// Special lookups used for various modifier forms
var suffixTypesCompiled map[string]CompiledList[string]
var suffixTypes = map[string]string{
	"as extra lightning damage":        "GainAsLightning",
	"added as lightning damage":        "GainAsLightning",
	"gained as extra lightning damage": "GainAsLightning",
	"as extra cold damage":             "GainAsCold",
	"added as cold damage":             "GainAsCold",
	"gained as extra cold damage":      "GainAsCold",
	"as extra fire damage":             "GainAsFire",
	"added as fire damage":             "GainAsFire",
	"gained as extra fire damage":      "GainAsFire",
	"as extra chaos damage":            "GainAsChaos",
	"added as chaos damage":            "GainAsChaos",
	"gained as extra chaos damage":     "GainAsChaos",
	"converted to lightning":           "ConvertToLightning",
	"converted to lightning damage":    "ConvertToLightning",
	"converted to cold damage":         "ConvertToCold",
	"converted to fire damage":         "ConvertToFire",
	"converted to fire":                "ConvertToFire",
	"converted to chaos damage":        "ConvertToChaos",
	"added as energy shield":           "GainAsEnergyShield",
	"as extra maximum energy shield":   "GainAsEnergyShield",
	"converted to energy shield":       "ConvertToEnergyShield",
	"as extra armour":                  "GainAsArmour",
	"as physical damage":               "AsPhysical",
	"as lightning damage":              "AsLightning",
	"as cold damage":                   "AsCold",
	"as fire damage":                   "AsFire",
	"as fire":                          "AsFire",
	"as chaos damage":                  "AsChaos",
	"leeched as life and mana":         "Leech",
	"leeched as life":                  "LifeLeech",
	"is leeched as life":               "LifeLeech",
	"leeched as mana":                  "ManaLeech",
	"is leeched as mana":               "ManaLeech",
	"leeched as energy shield":         "EnergyShieldLeech",
	"is leeched as energy shield":      "EnergyShieldLeech",
}

var dmgTypesCompiled map[string]CompiledList[string]
var dmgTypes = map[string]string{
	"physical":  "Physical",
	"lightning": "Lightning",
	"cold":      "Cold",
	"fire":      "Fire",
	"chaos":     "Chaos",
}

var penTypesCompiled map[string]CompiledList[modNameListType]
var penTypes = map[string]modNameListType{
	"lightning resistance":  {names: []string{"LightningPenetration"}},
	"cold resistance":       {names: []string{"ColdPenetration"}},
	"fire resistance":       {names: []string{"FirePenetration"}},
	"elemental resistance":  {names: []string{"ElementalPenetration"}},
	"elemental resistances": {names: []string{"ElementalPenetration"}},
	"chaos resistance":      {names: []string{"ChaosPenetration"}},
}

var regenTypesCompiled map[string]CompiledList[modNameListType]
var regenTypes = map[string]modNameListType{
	"life":                           {names: []string{"LifeRegen"}},
	"maximum life":                   {names: []string{"LifeRegen"}},
	"life and mana":                  {names: []string{"LifeRegen", "ManaRegen"}},
	"mana":                           {names: []string{"ManaRegen"}},
	"maximum mana":                   {names: []string{"ManaRegen"}},
	"energy shield":                  {names: []string{"EnergyShieldRegen"}},
	"maximum energy shield":          {names: []string{"EnergyShieldRegen"}},
	"maximum mana and energy shield": {names: []string{"ManaRegen", "EnergyShieldRegen"}},
	"rage":                           {names: []string{"RageRegen"}},
}

var flagTypesCompiled map[string]CompiledList[modNameListType]
var flagTypes = map[string]modNameListType{
	`phasing`:              {names: []string{"Condition:Phasing"}},
	`onslaught`:            {names: []string{"Condition:Onslaught"}},
	`fortify`:              {names: []string{"Condition:Fortified"}},
	`fortified`:            {names: []string{"Condition:Fortified"}},
	`unholy might`:         {names: []string{"Condition:UnholyMight"}},
	`tailwind`:             {names: []string{"Condition:Tailwind"}},
	`intimidated`:          {names: []string{"Condition:Intimidated"}},
	`crushed`:              {names: []string{"Condition:Crushed"}},
	`chilled`:              {names: []string{"Condition:Chilled"}},
	`blinded`:              {names: []string{"Condition:Blinded"}},
	`no life regeneration`: {names: []string{"NoLifeRegen"}},
	`hexproof`:             {names: []string{"CurseEffectOnSelf"}, tag: mod.NewFloat("CurseEffectOnSelf", mod.TypeMore, -100)},
	`hindered, with (\d+)\% reduced movement speed`: {names: []string{"Condition:Hindered"}},
	`unnerved`: {names: []string{"Condition:Unnerved"}},
}

// Build active skill name lookup
var skillNameListCompiled map[string]CompiledList[modNameListType]
var preSkillNameListCompiled map[string]CompiledList[modNameListType]

func initializeSkillNameList() {
	skillNameListCompiled = make(map[string]CompiledList[modNameListType])
	skillNameListCompiled[" corpse cremation "] = CompiledList[modNameListType]{
		Regex: regexp.MustCompile(` corpse cremation `),
		Value: modNameListType{
			tag: mod.SkillName("Cremation"),
		},
	}

	preSkillNameListCompiled = make(map[string]CompiledList[modNameListType])
	for _, gemData := range poe.SkillGems {
		grantedEffect := gemData.GetGrantedEffect()
		// TODO grantedEffect.hidden
		if grantedEffect != nil && !grantedEffect.IsSupport {
			skillName := grantedEffect.GetActiveSkill().DisplayedName

			if skillName == "" || skillName == "..." || strings.Contains(skillName, "NOT CURRENTLY USED") || strings.Contains(skillName, "UNUSED") {
				continue
			}

			skillNameLower := strings.ToLower(skillName)

			val := modNameListType{
				tag: mod.SkillName(skillName),
			}

			skillNameListCompiled[" "+skillNameLower+" "] = CompiledList[modNameListType]{
				Regex: regexp.MustCompile(" " + skillNameLower + " "),
				Value: val,
			}

			preSkillNameListCompiled["^"+skillNameLower+" has ?a? "] = CompiledList[modNameListType]{
				Regex: regexp.MustCompile("^" + skillNameLower + " has ?a? "),
				Value: val,
			}

			preSkillNameListCompiled["^"+skillNameLower+" deals "] = CompiledList[modNameListType]{
				Regex: regexp.MustCompile("^" + skillNameLower + " deals "),
				Value: val,
			}

			preSkillNameListCompiled["^"+skillNameLower+" damage "] = CompiledList[modNameListType]{
				Regex: regexp.MustCompile("^" + skillNameLower + " damage "),
				Value: val,
			}

			/*
				TODO
				if gemData.tags.totem then
					preSkillNameList["^"..skillName:lower().." totem deals "] = { tag = mod.SkillName(skillName ) }
					preSkillNameList["^"..skillName:lower().." totem grants "] = { addToSkill = mod.SkillName(skillName ), tag = mod.GlobalEffect("Buff") }
				end
			*/

			baseFlags, _ := grantedEffect.GetActiveSkill().GetActiveSkillBaseFlagsAndTypes()
			if slices.Contains(grantedEffect.GetActiveSkill().ActiveSkillTypes, poe.ActiveSkillTypesByID["Buff"].Key) || baseFlags[poe.SkillFlagBuffs] {
				preSkillNameListCompiled["^"+skillNameLower+" grants "] = CompiledList[modNameListType]{
					Regex: regexp.MustCompile("^" + skillNameLower + " grants "),
					Value: modNameListType{
						addToSkill: mod.SkillName(skillName),
						tag:        mod.GlobalEffect("Buff"),
					},
				}

				preSkillNameListCompiled["^"+skillNameLower+" grants a?n? ?additional "] = CompiledList[modNameListType]{
					Regex: regexp.MustCompile("^" + skillNameLower + " grants a?n? ?additional "),
					Value: modNameListType{
						addToSkill: mod.SkillName(skillName),
						tag:        mod.GlobalEffect("Buff"),
					},
				}
			}

			/*
					TODO
				if gemData.tags.aura or gemData.tags.herald then
					skillNameList["while affected by "..skillName:lower()] = { tag = { type = "Condition", var = "AffectedBy"..skillName:gsub(" ","") } }
					skillNameList["while using "..skillName:lower()] = { tag = { type = "Condition", var = "AffectedBy"..skillName:gsub(" ","") } }
				end
				if gemData.tags.mine then
					specialModList["^"..skillName:lower().." has (\d+)% increased throwing speed"] = function(num) return { MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{ Mod: MOD("MineLayingSpeed", "INC", num) }, mod.SkillName(skillName )) } end
				end
				if gemData.tags.trap then
					specialModList["(\d+)% increased "..skillName:lower().." throwing speed"] = function(num) return { MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{ Mod: MOD("TrapThrowingSpeed", "INC", num) }, mod.SkillName(skillName )) } end
				end
				if gemData.tags.chaining then
					specialModList["^"..skillName:lower().." chains an additional time"] = { MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{ Mod: MOD("ChainCountMax", "BASE", 1) }, mod.SkillName(skillName )) }
					specialModList["^"..skillName:lower().." chains an additional (\d+) times"] = function(num) return { MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{ Mod: MOD("ChainCountMax", "BASE", num) }, mod.SkillName(skillName )) } end
					specialModList["^"..skillName:lower().." chains (\d+) additional times"] = function(num) return { MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{ Mod: MOD("ChainCountMax", "BASE", num) }, mod.SkillName(skillName )) } end
				end
				if gemData.tags.bow then
					specialModList["^"..skillName:lower().." fires an additional arrow"] = function(num) return { MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{ Mod: MOD("ProjectileCount", "BASE", 1) }, mod.SkillName(skillName )) } end
					specialModList["^"..skillName:lower().." fires (\d+) additional arrows?"] = function(num) return { MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{ Mod: MOD("ProjectileCount", "BASE", num) }, mod.SkillName(skillName )) } end
				end
				if gemData.tags.projectile then
					specialModList["^"..skillName:lower().." pierces an additional target"] = { MOD("PierceCount", "BASE", 1, mod.SkillName(skillName )) }
					specialModList["^"..skillName:lower().." pierces (\d+) additional targets?"] = function(num) return { MOD("PierceCount", "BASE", num, mod.SkillName(skillName )) } end
				end
				if gemData.tags.bow or gemData.tags.projectile then
					specialModList["^"..skillName:lower().." fires an additional projectile"] = { MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{ Mod: MOD("ProjectileCount", "BASE", 1) }, mod.SkillName(skillName )) }
					specialModList["^"..skillName:lower().." fires (\d+) additional projectiles"] = function(num) return { MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{ Mod: MOD("ProjectileCount", "BASE", num) }, mod.SkillName(skillName )) } end
					specialModList["^"..skillName:lower().." fires (\d+) additional shard projectiles"] = function(num) return { MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{ Mod: MOD("ProjectileCount", "BASE", num) }, mod.SkillName(skillName )) } end
				end
			*/
		}
	}
}

/*
// TODO Radius jewels that modify other nodes
local function getSimpleConv(srcList, dst, type, remove, factor)
	return function(node, out, data)
		if node then
			for _, src in pairs(srcList) do
				for _, mod in ipairs(node.modList) do
					if mod.name == src and mod.type == type then
						if remove then
							out:MergeNewMod(src, type, -mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
						end
						if factor then
							out:MergeNewMod(dst, type, math.floor(mod.value * factor), mod.source, mod.flags, mod.keywordFlags, unpack(mod))
						else
							out:MergeNewMod(dst, type, mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
						end
					end
				end
			end
		end
	end
end
local jewelOtherFuncs = {
	["Strength from Passives in Radius is Transformed to Dexterity"] = getSimpleConv({ "Str" }, "Dex", "BASE", true),
	["Dexterity from Passives in Radius is Transformed to Strength"] = getSimpleConv({ "Dex" }, "Str", "BASE", true),
	["Strength from Passives in Radius is Transformed to Intelligence"] = getSimpleConv({ "Str" }, "Int", "BASE", true),
	["Intelligence from Passives in Radius is Transformed to Strength"] = getSimpleConv({ "Int" }, "Str", "BASE", true),
	["Dexterity from Passives in Radius is Transformed to Intelligence"] = getSimpleConv({ "Dex" }, "Int", "BASE", true),
	["Intelligence from Passives in Radius is Transformed to Dexterity"] = getSimpleConv({ "Int" }, "Dex", "BASE", true),
	["Increases and Reductions to Life in Radius are Transformed to apply to Energy Shield"] = getSimpleConv({ "Life" }, "EnergyShield", "INC", true),
	["Increases and Reductions to Energy Shield in Radius are Transformed to apply to Armour at 200% of their value"] = getSimpleConv({ "EnergyShield" }, "Armour", "INC", true, 2),
	["Increases and Reductions to Life in Radius are Transformed to apply to Mana at 200% of their value"] = getSimpleConv({ "Life" }, "Mana", "INC", true, 2),
	["Increases and Reductions to Physical Damage in Radius are Transformed to apply to Cold Damage"] = getSimpleConv({ "PhysicalDamage" }, "ColdDamage", "INC", true),
	["Increases and Reductions to Cold Damage in Radius are Transformed to apply to Physical Damage"] = getSimpleConv({ "ColdDamage" }, "PhysicalDamage", "INC", true),
	["Increases and Reductions to other Damage Types in Radius are Transformed to apply to Fire Damage"] = getSimpleConv({ "PhysicalDamage","ColdDamage","LightningDamage","ChaosDamage" }, "FireDamage", "INC", true),
	["Passives granting Lightning Resistance or all Elemental Resistances in Radius also grant Chance to Block Spells at 35% of its value"] = getSimpleConv({ "LightningResist","ElementalResist" }, "SpellBlockChance", "BASE", false, 0.35),
	["Passives granting Lightning Resistance or all Elemental Resistances in Radius also grant Chance to Block Spell Damage at 35% of its value"] = getSimpleConv({ "LightningResist","ElementalResist" }, "SpellBlockChance", "BASE", false, 0.35),
	["Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Dodge Attacks at 35% of its value"] = getSimpleConv({ "ColdResist","ElementalResist" }, "AttackDodgeChance", "BASE", false, 0.35),
	["Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Dodge Attack Hits at 35% of its value"] = getSimpleConv({ "ColdResist","ElementalResist" }, "AttackDodgeChance", "BASE", false, 0.35),
	["Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Suppress Spell Damage at 35% of its value"] = getSimpleConv({ "ColdResist","ElementalResist" }, "SpellSuppressionChance", "BASE", false, 0.35),
	["Passives granting Cold Resistance or all Elemental Resistances in Radius also grant Chance to Suppress Spell Damage at 50% of its value"] = getSimpleConv({ "ColdResist","ElementalResist" }, "SpellSuppressionChance", "BASE", false, 0.5),
	["Passives granting Fire Resistance or all Elemental Resistances in Radius also grant Chance to Block Attack Damage at 35% of its value"] = getSimpleConv({ "FireResist","ElementalResist" }, "BlockChance", "BASE", false, 0.35),
	["Passives granting Fire Resistance or all Elemental Resistances in Radius also grant Chance to Block at 35% of its value"] = getSimpleConv({ "FireResist","ElementalResist" }, "BlockChance", "BASE", false, 0.35),
	["Melee and Melee Weapon Type modifiers in Radius are Transformed to Bow Modifiers"] = function(node, out, data)
		if node then
			local mask1 = bor(ModFlag.Axe, ModFlag.Claw, ModFlag.Dagger, ModFlag.Mace, ModFlag.Staff, ModFlag.Sword, ModFlag.Melee)
			local mask2 = bor(ModFlag.Weapon1H, ModFlag.WeaponMelee)
			local mask3 = bor(ModFlag.Weapon2H, ModFlag.WeaponMelee)
			for _, mod in ipairs(node.modList) do
				if band(mod.flags, mask1) ~= 0 or band(mod.flags, mask2) == mask2 or band(mod.flags, mask3) == mask3 then
					out:MergeNewMod(mod.name, mod.type, -mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
					out:MergeNewMod(mod.name, mod.type, mod.value, mod.source, bor(band(mod.flags, bnot(bor(mask1, mask2, mask3))), ModFlag.Bow), mod.keywordFlags, unpack(mod))
				elseif mod[1] then
					local using = { UsingAxe = true, UsingClaw = true, UsingDagger = true, UsingMace = true, UsingStaff = true, UsingSword = true, UsingMeleeWeapon = true }
					for _, tag in ipairs(mod) do
						if tag.type == "Condition" and using[tag.var] then
							local newTagList = copyTable(mod)
							for _, tag in ipairs(newTagList) do
								if tag.type == "Condition" and using[tag.var] then
									tag.var = "UsingBow"
									break
								end
							end
							out:MergeNewMod(mod.name, mod.type, -mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(mod))
							out:MergeNewMod(mod.name, mod.type, mod.value, mod.source, mod.flags, mod.keywordFlags, unpack(newTagList))
							break
						end
					end
				end
			end
		end
	end,
	["50% increased Effect of non-Keystone Passive Skills in Radius"] = function(node, out, data)
		if node and node.type ~= "Keystone" then
			out:NewMod("PassiveSkillEffect", "INC", 50, data.modSource)
		end
	end,
	["Notable Passive Skills in Radius grant nothing"] = function(node, out, data)
		if node and node.type == "Notable" then
			out:NewMod("PassiveSkillHasNoEffect", "FLAG", true, data.modSource)
		end
	end,
	["Allocated Small Passive Skills in Radius grant nothing"] = function(node, out, data)
		if node and node.type == "Normal" then
			out:NewMod("AllocatedPassiveSkillHasNoEffect", "FLAG", true, data.modSource)
		end
	end,
	["Passive Skills in Radius also grant: Traps and Mines deal (\d+) to (\d+) added Physical Damage"] = function(min, max)
		return function(node, out, data)
			if node and node.type ~= "Keystone" then
				out:NewMod("PhysicalMin", "BASE", min, data.modSource, 0, bor(KeywordFlag.Trap, KeywordFlag.Mine))
				out:NewMod("PhysicalMax", "BASE", max, data.modSource, 0, bor(KeywordFlag.Trap, KeywordFlag.Mine))
			end
		end
	end,
	["Passive Skills in Radius also grant: (\d+)% increased Unarmed Attack Speed with Melee Skills"] = function(num)
		return function(node, out, data)
			if node and node.type ~= "Keystone" then
				out:NewMod("Speed", "INC", num, data.modSource, bor(ModFlag.Unarmed, ModFlag.Attack, ModFlag.Melee))
			end
		end
	end,
	["Notable Passive Skills in Radius are Transformed to instead grant: 10% increased Mana Cost of Skills and 20% increased Spell Damage"] = function(node, out, data)
		if node and node.type == "Notable" then
			out:NewMod("PassiveSkillHasOtherEffect", "FLAG", true, data.modSource)
			out:NewMOD("NodeModifier", "LIST", mod.NodeModifier{ Mod: MOD("ManaCost", "INC", 10, data.modSource) }, data.modSource)
			out:NewMOD("NodeModifier", "LIST", mod.NodeModifier{ Mod: MOD("Damage", "INC", 20, data.modSource, ModFlag.Spell) }, data.modSource)
		end
	end,
	["Notable Passive Skills in Radius are Transformed to instead grant: Minions take 20% increased Damage"] = function(node, out, data)
		if node and node.type == "Notable" then
			out:NewMod("PassiveSkillHasOtherEffect", "FLAG", true, data.modSource)
			out:NewMOD("NodeModifier", "LIST", mod.NodeModifier{ Mod: MOD("MinionModifier", "LIST", mod.MinionModifier{ Mod: MOD("DamageTaken", "INC", 20, data.modSource) } ) }, data.modSource)
		end
	end,
	["Notable Passive Skills in Radius are Transformed to instead grant: Minions have 25% reduced Movement Speed"] = function(node, out, data)
		if node and node.type == "Notable" then
			out:NewMod("PassiveSkillHasOtherEffect", "FLAG", true, data.modSource)
			out:NewMOD("NodeModifier", "LIST", mod.NodeModifier{ Mod: MOD("MinionModifier", "LIST", mod.MinionModifier{ Mod: MOD("MovementSpeed", "INC", -25, data.modSource) } ) }, data.modSource)
		end
	end,
}

// TODO Radius jewels that modify the jewel itself based on nearby allocated nodes
local function getPerStat(dst, modType, flags, stat, factor)
	return function(node, out, data)
		if node then
			data[stat] = (data[stat] or 0) + out:Sum("BASE", nil, stat)
		elseif data[stat] ~= 0 then
			out:NewMod(dst, modType, math.floor((data[stat] or 0) * factor), data.modSource, flags)
		end
	end
end
local jewelSelfFuncs = {
	["Adds 1 to maximum Life per 3 Intelligence in Radius"] = getPerStat("Life", "BASE", 0, "Int", 1 / 3),
	["Adds 1 to Maximum Life per 3 Intelligence Allocated in Radius"] = getPerStat("Life", "BASE", 0, "Int", 1 / 3),
	["1% increased Evasion Rating per 3 Dexterity Allocated in Radius"] = getPerStat("Evasion", "INC", 0, "Dex", 1 / 3),
	["1% increased Claw Physical Damage per 3 Dexterity Allocated in Radius"] = getPerStat("PhysicalDamage", "INC", ModFlag.Claw, "Dex", 1 / 3),
	["1% increased Melee Physical Damage while Unarmed per 3 Dexterity Allocated in Radius"] = getPerStat("PhysicalDamage", "INC", ModFlag.Unarmed, "Dex", 1 / 3),
	["3% increased Totem Life per 10 Strength in Radius"] = getPerStat("TotemLife", "INC", 0, "Str", 3 / 10),
	["3% increased Totem Life per 10 Strength Allocated in Radius"] = getPerStat("TotemLife", "INC", 0, "Str", 3 / 10),
	["Adds 1 maximum Lightning Damage to Attacks per 1 Dexterity Allocated in Radius"] = getPerStat("LightningMax", "BASE", ModFlag.Attack, "Dex", 1),
	["5% increased Chaos damage per 10 Intelligence from Allocated Passives in Radius"] = getPerStat("ChaosDamage", "INC", 0, "Int", 5 / 10),
	["Dexterity and Intelligence from passives in Radius count towards Strength Melee Damage bonus"] = function(node, out, data)
		if node then
			data.Dex = (data.Dex or 0) + node.modList:Sum("BASE", nil, "Dex")
			data.Int = (data.Int or 0) + node.modList:Sum("BASE", nil, "Int")
		elseif data.Dex or data.Int then
			out:NewMod("DexIntToMeleeBonus", "BASE", (data.Dex or 0) + (data.Int or 0), data.modSource)
		end
	end,
	["-1 Strength per 1 Strength on Allocated Passives in Radius"] = getPerStat("Str", "BASE", 0, "Str", -1),
	["1% additional Physical Damage Reduction per 10 Strength on Allocated Passives in Radius"] = getPerStat("PhysicalDamageReduction", "BASE", 0, "Str", 1 / 10),
	["2% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius"] = getPerStat("LifeRecoveryRate", "INC", 0, "Str", 2 / 10),
	["3% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius"] = getPerStat("LifeRecoveryRate", "INC", 0, "Str", 3 / 10),
	["-1 Intelligence per 1 Intelligence on Allocated Passives in Radius"] = getPerStat("Int", "BASE", 0, "Int", -1),
	["0.4% of Energy Shield Regenerated per Second for every 10 Intelligence on Allocated Passives in Radius"] = getPerStat("EnergyShieldRegenPercent", "BASE", 0, "Int", 0.4 / 10),
	["2% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius"] = getPerStat("ManaRecoveryRate", "INC", 0, "Int", 2 / 10),
	["3% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius"] = getPerStat("ManaRecoveryRate", "INC", 0, "Int", 3 / 10),
	["-1 Dexterity per 1 Dexterity on Allocated Passives in Radius"] = getPerStat("Dex", "BASE", 0, "Dex", -1),
	["2% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius"] = getPerStat("MovementSpeed", "INC", 0, "Dex", 2 / 10),
	["3% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius"] = getPerStat("MovementSpeed", "INC", 0, "Dex", 3 / 10),
}
local jewelSelfUnallocFuncs = {
	["+5% to Critical Strike Multiplier per 10 Strength on Unallocated Passives in Radius"] = getPerStat("CritMultiplier", "BASE", 0, "Str", 5 / 10),
	["+7% to Critical Strike Multiplier per 10 Strength on Unallocated Passives in Radius"] = getPerStat("CritMultiplier", "BASE", 0, "Str", 7 / 10),
	["2% reduced Life Recovery Rate per 10 Strength on Unallocated Passives in Radius"] = getPerStat("LifeRecoveryRate", "INC", 0, "Str", -2 / 10),
	["+15 to maximum Mana per 10 Dexterity on Unallocated Passives in Radius"] = getPerStat("Mana", "BASE", 0, "Dex", 15 / 10),
	["+100 to Accuracy Rating per 10 Intelligence on Unallocated Passives in Radius"] = getPerStat("Accuracy", "BASE", 0, "Int", 100 / 10),
	["+125 to Accuracy Rating per 10 Intelligence on Unallocated Passives in Radius"] = getPerStat("Accuracy", "BASE", 0, "Int", 125 / 10),
	["2% reduced Mana Recovery Rate per 10 Intelligence on Unallocated Passives in Radius"] = getPerStat("ManaRecoveryRate", "INC", 0, "Int", -2 / 10),
	["+3% to Damage over Time Multiplier per 10 Intelligence on Unallocated Passives in Radius"] = getPerStat("DotMultiplier", "BASE", 0, "Int", 3 / 10),
	["2% reduced Movement Speed per 10 Dexterity on Unallocated Passives in Radius"] = getPerStat("MovementSpeed", "INC", 0, "Dex", -2 / 10),
	["+125 to Accuracy Rating per 10 Dexterity on Unallocated Passives in Radius"] = getPerStat("Accuracy", "BASE", 0, "Dex", 125 / 10),
	["Grants all bonuses of Unallocated Small Passive Skills in Radius"] = function(node, out, data)
		if node then
			if node.type == "Normal" then
				data.modList = data.modList or new("ModList")
				data.modList:AddList(out)
			end
		elseif data.modList then
			out:AddList(data.modList)
		end
	end,
}

// TODO Radius jewels with bonuses conditional upon attributes of nearby nodes
local function getThreshold(attrib, name, modType, value, ...)
	local baseMod = mod(name, modType, value, "", ...)
	return function(node, out, data)
		if node then
			if type(attrib) == "table" then
				for _, att in ipairs(attrib) do
					local nodeVal = out:Sum("BASE", nil, att)
					data[att] = (data[att] or 0) + nodeVal
					data.total = (data.total or 0) + nodeVal
				end
			else
				local nodeVal = out:Sum("BASE", nil, attrib)
				data[attrib] = (data[attrib] or 0) + nodeVal
				data.total = (data.total or 0) + nodeVal
			end
		elseif (data.total or 0) >= 40 then
			local mod = copyTable(baseMod)
			mod.source = data.modSource
			if type(value) == "table" and value.mod then
				value.mod.source = data.modSource
			end
			out:AddMod(mod)
		end
	end
end
local jewelThresholdFuncs = {
	["With at least 40 Dexterity in Radius, Frost Blades Melee Damage Penetrates 15% Cold Resistance"] = getThreshold("Dex", "ColdPenetration", "BASE", 15, ModFlag.Melee, mod.SkillName("Frost Blades" )),
	["With at least 40 Dexterity in Radius, Melee Damage dealt by Frost Blades Penetrates 15% Cold Resistance"] = getThreshold("Dex", "ColdPenetration", "BASE", 15, ModFlag.Melee, mod.SkillName("Frost Blades" )),
	["With at least 40 Dexterity in Radius, Frost Blades has 25% increased Projectile Speed"] = getThreshold("Dex", "ProjectileSpeed", "INC", 25, mod.SkillName("Frost Blades" )),
	["With at least 40 Dexterity in Radius, Ice Shot has 25% increased Area of Effect"] = getThreshold("Dex", "AreaOfEffect", "INC", 25, mod.SkillName("Ice Shot" )),
	["Ice Shot Pierces 5 additional Targets with 40 Dexterity in Radius"] = getThreshold("Dex", "PierceCount", "BASE", 5, mod.SkillName("Ice Shot" )),
	["With at least 40 Dexterity in Radius, Ice Shot Pierces 3 additional Targets"] = getThreshold("Dex", "PierceCount", "BASE", 3, mod.SkillName("Ice Shot" )),
	["With at least 40 Dexterity in Radius, Ice Shot Pierces 5 additional Targets"] = getThreshold("Dex", "PierceCount", "BASE", 5, mod.SkillName("Ice Shot" )),
	["With at least 40 Intelligence in Radius, Frostbolt fires 2 additional Projectiles"] = getThreshold("Int", "ProjectileCount", "BASE", 2, mod.SkillName("Frostbolt" )),
	["With at least 40 Intelligence in Radius, Rolling Magma fires an additional Projectile"] = getThreshold("Int", "ProjectileCount", "BASE", 1, mod.SkillName("Rolling Magma" )),
	["With at least 40 Intelligence in Radius, Rolling Magma has 10% increased Area of Effect per Chain"] = getThreshold("Int", "AreaOfEffect", "INC", 10, mod.SkillName("Rolling Magma")).Tag(mod.PerStat(0, "Chain")),
	["With at least 40 Intelligence in Radius, Rolling Magma deals 40% more damage per chain"] = getThreshold("Int", "Damage", "MORE", 40, mod.SkillName("Rolling Magma")).Tag(mod.PerStat(0, "Chain")),
	["With at least 40 Intelligence in Radius, Rolling Magma deals 50% less damage"] = getThreshold("Int", "Damage", "MORE", -50, mod.SkillName("Rolling Magma" )),
	["With at least 40 Dexterity in Radius, Shrapnel Shot has 25% increased Area of Effect"] = getThreshold("Dex", "AreaOfEffect", "INC", 25, mod.SkillName("Shrapnel Shot" )),
	["With at least 40 Dexterity in Radius, Shrapnel Shot's cone has a 50% chance to deal Double Damage"] = getThreshold("Dex", "DoubleDamageChance", "BASE", 50, mod.SkillName("Shrapnel Shot" ), mod.SkillPart(2)),
	["With at least 40 Dexterity in Radius, Galvanic Arrow deals 50% increased Area Damage"] = getThreshold("Dex", "Damage", "INC", 50, mod.SkillName("Galvanic Arrow" ), mod.SkillPart(2)),
	["With at least 40 Dexterity in Radius, Galvanic Arrow has 25% increased Area of Effect"] = getThreshold("Dex", "AreaOfEffect", "INC", 25, mod.SkillName("Galvanic Arrow" )),
	["With at least 40 Intelligence in Radius, Freezing Pulse fires 2 additional Projectiles"] = getThreshold("Int", "ProjectileCount", "BASE", 2, mod.SkillName("Freezing Pulse" )),
	["With at least 40 Intelligence in Radius, 25% increased Freezing Pulse Damage if you've Shattered an Enemy Recently"] = getThreshold("Int", "Damage", "INC", 25, mod.SkillName("Freezing Pulse" ), { type = "Condition", var = "ShatteredEnemyRecently" }),
	["With at least 40 Dexterity in Radius, Ethereal Knives fires 10 additional Projectiles"] = getThreshold("Dex", "ProjectileCount", "BASE", 10, mod.SkillName("Ethereal Knives" )),
	["With at least 40 Dexterity in Radius, Ethereal Knives fires 5 additional Projectiles"] = getThreshold("Dex", "ProjectileCount", "BASE", 5, mod.SkillName("Ethereal Knives" )),
	["With at least 40 Strength in Radius, Molten Strike fires 2 additional Projectiles"] = getThreshold("Str", "ProjectileCount", "BASE", 2, mod.SkillName("Molten Strike" )),
	["With at least 40 Strength in Radius, Molten Strike has 25% increased Area of Effect"] = getThreshold("Str", "AreaOfEffect", "INC", 25, mod.SkillName("Molten Strike" )),
	["With at least 40 Strength in Radius, Molten Strike Projectiles Chain +1 time"] = getThreshold("Str", "ChainCountMax", "BASE", 1, mod.SkillName("Molten Strike" )),
	["With at least 40 Strength in Radius, Molten Strike fires 50% less Projectiles"] = getThreshold("Str", "ProjectileCount", "MORE", -50, mod.SkillName("Molten Strike" )),
	["With at least 40 Strength in Radius, 25% of Glacial Hammer Physical Damage converted to Cold Damage"] = getThreshold("Str", "SkillPhysicalDamageConvertToCold", "BASE", 25, mod.SkillName("Glacial Hammer" )),
	["With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage"] = getThreshold("Str", "DoubleDamageChance", "BASE", 20, mod.SkillName("Heavy Strike" )),
	["With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage."] = getThreshold("Str", "DoubleDamageChance", "BASE", 20, mod.SkillName("Heavy Strike" )),
	["With at least 40 Strength in Radius, Cleave has +1 to Radius per Nearby Enemy, up to +10"] = getThreshold("Str", "AreaOfEffect", "BASE", 1, { type = "Multiplier", var = "NearbyEnemies", limit = 10 }, mod.SkillName("Cleave" )),
	["With at least 40 Strength in Radius, Cleave grants Fortify on Hit"] = getThreshold("Str", "ExtraSkillMod", "LIST", { mod = FLAG("Condition:Fortified") }, mod.SkillName("Cleave" )),
	["With at least 40 Strength in Radius, Hits with Cleave Fortify"] = getThreshold("Str", "ExtraSkillMod", "LIST", { mod = FLAG("Condition:Fortified") }, mod.SkillName("Cleave" )),
	["With at least 40 Dexterity in Radius, Dual Strike has a 20% chance to deal Double Damage with the Main-Hand Weapon"] = getThreshold("Dex", "DoubleDamageChance", "BASE", 20, mod.SkillName("Dual Strike" ), { type = "Condition", var = "MainHandAttack" }),
	["With at least 40 Dexterity in Radius, Dual Strike has (\d+)% increased Attack Speed while wielding a Claw"] = function(num) return getThreshold("Dex", "Speed", "INC", num, mod.SkillName("Dual Strike" ), { type = "Condition", var = "UsingClaw" }) end,
	["With at least 40 Dexterity in Radius, Dual Strike has %+(\d+)% to Critical Strike Multiplier while wielding a Dagger"] = function(num) return getThreshold("Dex", "CritMultiplier", "BASE", num, mod.SkillName("Dual Strike" ), { type = "Condition", var = "UsingDagger" }) end,
	["With at least 40 Dexterity in Radius, Dual Strike has (\d+)% increased Accuracy Rating while wielding a Sword"] = function(num) return getThreshold("Dex", "Accuracy", "INC", num, mod.SkillName("Dual Strike" ), { type = "Condition", var = "UsingSword" }) end,
	["With at least 40 Dexterity in Radius, Dual Strike Hits Intimidate Enemies for 4 seconds while wielding an Axe"] = getThreshold("Dex", "EnemyModifier", "LIST", { mod = FLAG("Condition:Intimidated")}, { type = "Condition", var = "UsingAxe" }),
	["With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack has 100% increased Cooldown Recovery Speed"] = getThreshold("Int", "MinionModifier", "LIST", { mod = MOD("CooldownRecovery", "INC", 100, { type = "SkillId", SkillID: "ZombieSlam" }) }),
	["With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack deals 30% increased Damage"] = getThreshold("Int", "MinionModifier", "LIST", { mod = MOD("Damage", "INC", 30, { type = "SkillId", SkillID: "ZombieSlam" }) }),
	["With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Attack Damage for each Poison on the Enemy"] = getThreshold("Dex", "Damage", "INC", 2, ModFlag.Attack, mod.SkillName("Viper Strike" ), { type = "Multiplier", actor = "enemy", var = "PoisonStack" }),
	["With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Damage with Hits and Poison for each Poison on the Enemy"] = getThreshold("Dex", "Damage", "INC", 2, 0, bor(KeywordFlag.Hit, KeywordFlag.Poison), mod.SkillName("Viper Strike" ), { type = "Multiplier", actor = "enemy", var = "PoisonStack" }),
	["With at least 40 Intelligence in Radius, Spark fires 2 additional Projectiles"] = getThreshold("Int", "ProjectileCount", "BASE", 2, mod.SkillName("Spark" )),
	["With at least 40 Intelligence in Radius, Blight has 50% increased Hinder Duration"] = getThreshold("Int", "SecondaryDuration", "INC", 50, mod.SkillName("Blight" )),
	["With at least 40 Intelligence in Radius, Enemies Hindered by Blight take 25% increased Chaos Damage"] = getThreshold("Int", "ExtraSkillMod", "LIST", { mod = MOD("ChaosDamageTaken", "INC", 25, mod.GlobalEffect("Debuff", effectName = "Hinder")) }, mod.SkillName("Blight")).Tag(mod.ActorCondition("enemy", "Hindered")),
	["With 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage"] = getThreshold("Int", "SkillPhysicalDamageConvertToCold", "BASE", 20, mod.SkillName("Glacial Cascade" )),
	["With at least 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage"] = getThreshold("Int", "SkillPhysicalDamageConvertToCold", "BASE", 20, mod.SkillName("Glacial Cascade" )),
	["With 40 total Intelligence and Dexterity in Radius, Elemental Hit and Wild Strike deal 50% less Fire Damage"] = getThreshold({ "Int","Dex" }, "FireDamage", "MORE", -50, { type = "SkillName", skillNameList = { "Elemental Hit", "Wild Strike" } }),
	["With 40 total Strength and Intelligence in Radius, Elemental Hit and Wild Strike deal 50% less Cold Damage"] = getThreshold({ "Str","Int" }, "ColdDamage", "MORE", -50, { type = "SkillName", skillNameList = { "Elemental Hit", "Wild Strike" } }),
	["With 40 total Dexterity and Strength in Radius, Elemental Hit and Wild Strike deal 50% less Lightning Damage"] = getThreshold({ "Dex","Str" }, "LightningDamage", "MORE", -50, { type = "SkillName", skillNameList = { "Elemental Hit", "Wild Strike" } }),
	["With 40 total Intelligence and Dexterity in Radius, Prismatic Skills deal 50% less Fire Damage"] = getThreshold({ "Int","Dex" }, "FireDamage", "MORE", -50, { type = "SkillType", skillType = SkillType.RandomElement }),
	["With 40 total Strength and Intelligence in Radius, Prismatic Skills deal 50% less Cold Damage"] = getThreshold({ "Str","Int" }, "ColdDamage", "MORE", -50, { type = "SkillType", skillType = SkillType.RandomElement }),
	["With 40 total Dexterity and Strength in Radius, Prismatic Skills deal 50% less Lightning Damage"] = getThreshold({ "Dex","Str" }, "LightningDamage", "MORE", -50, { type = "SkillType", skillType = SkillType.RandomElement }),
	["With 40 total Dexterity and Strength in Radius, Spectral Shield Throw Chains +4 times"] = getThreshold({ "Dex","Str" }, "ChainCountMax", "BASE", 4, mod.SkillName("Spectral Shield Throw" )),
	["With 40 total Dexterity and Strength in Radius, Spectral Shield Throw fires 75% less Shard Projectiles"] = getThreshold({ "Dex","Str" }, "ProjectileCount", "MORE", -75, mod.SkillName("Spectral Shield Throw" )),
	["With at least 40 Intelligence in Radius, Blight inflicts Withered for 2 seconds"] = getThreshold("Int", "ExtraSkillMod", "LIST", { mod = MOD("Condition:CanWither", "FLAG", true) }, mod.SkillName("Blight" )),
	["With at least 40 Intelligence in Radius, Blight has 30% reduced Cast Speed"] = getThreshold("Int", "Speed", "INC", -30, mod.SkillName("Blight" )),
	["With at least 40 Intelligence in Radius, Fireball cannot ignite"] = getThreshold("Int", "ExtraSkillMod", "LIST", { mod = FLAG("CannotIgnite") }, mod.SkillName("Fireball" )),
	["With at least 40 Intelligence in Radius, Fireball has %+(\d+)% chance to inflict scorch"] = function(num) return getThreshold("Int", "EnemyScorchChance", "BASE", num, mod.SkillName("Fireball" )) end,
	["With at least 40 Intelligence in Radius, Discharge has 60% less Area of Effect"] = getThreshold("Int", "AreaOfEffect", "MORE", -60, {type = "SkillName", skillName = "Discharge" }),
	["With at least 40 Intelligence in Radius, Discharge Cooldown is 250 ms"] = getThreshold("Int", "CooldownRecovery", "OVERRIDE", 0.25, mod.SkillName("Discharge" )),
	["With at least 40 Intelligence in Radius, Discharge deals 60% less Damage"] = getThreshold("Int", "Damage", "MORE", -60, {type = "SkillName", skillName = "Discharge" }),
	// [""] = getThreshold("", "", "", , mod.SkillName("" )),
}
*/

// Unified list of jewel functions
var jewelFuncList = make(map[string]interface{})

/*
// TODO Jewels that modify nodes
for k, v in pairs(jewelOtherFuncs) do
	jewelFuncList[k:lower()] = { func = function(cap1, cap2, cap3, cap4, cap5)
		// TODO Need to not modify any nodes already modified by timeless jewels
		// TODO Some functions return a function instead of simply adding mods, so if
		// TODO we don't see a node right away, run the outer function first
		if cap1 and type(cap1) == "table" and cap1.conqueredBy then
			return
		end
		local innerFuncOrNil = v(cap1, cap2, cap3, cap4, cap5)
		// TODO In all (current) cases, there is only one nested layer, so no need for recursion
		return function(node, out, other)
			if node and type(node) == "table" and node.conqueredBy then
				return
			end
			return innerFuncOrNil(node, out, other)
		end
	end, type = "Other" }
end
for k, v in pairs(jewelSelfFuncs) do
	jewelFuncList[k:lower()] = { func = v, type = "Self" }
end
for k, v in pairs(jewelSelfUnallocFuncs) do
	jewelFuncList[k:lower()] = { func = v, type = "SelfUnalloc" }
end
// TODO Threshold Jewels
for k, v in pairs(jewelThresholdFuncs) do
	jewelFuncList[k:lower()] = { func = v, type = "Threshold" }
end

// TODO Generate list of cluster jewel skills
local clusterJewelSkills = {}
for baseName, jewel in pairs(data.clusterJewels.jewels) do
	for skillId, skill in pairs(jewel.skills) do
		clusterJewelSkills[table.concat(skill.enchant, " "):lower()] = { MOD("JewelData", "LIST", mod.JewelData{ Key: "clusterJewelSkill", Value: skillId }) }
	end
end
for notable in pairs(data.clusterJewels.notableSortOrder) do
	clusterJewelSkills["1 added passive skill is "..notable:lower()] = { MOD("ClusterJewelNotable", "LIST", notable) }
end
for _, keystone in ipairs(data.clusterJewels.keystones) do
	clusterJewelSkills["adds "..keystone:lower()] = { MOD("JewelData", "LIST", mod.JewelData{ Key: "clusterJewelKeystone", Value: keystone }) }
end
*/

// Scan a line for the earliest and longest match from the pattern list
// If a match is found, returns the corresponding value from the pattern list, plus the remainder of the line and a table of captures
func scan[T any](line string, patternList map[string]CompiledList[T], plain bool) (*T, string, []string) {
	bestIndex := -1
	bestEndIndex := 0
	bestPattern := ""

	var bestVal *T
	var bestCaps []string
	bestStart := 0
	bestEnd := 0

	lineLower := strings.ToLower(line)

	for pattern, patternVal := range patternList {
		indices := patternVal.Regex.FindAllStringSubmatchIndex(lineLower, -1)
		if len(indices) > 0 {
			index := indices[0][0]
			endIndex := indices[0][1]

			if bestIndex == -1 || index < bestIndex || (index == bestIndex && (endIndex > bestEndIndex || (endIndex == bestIndex && len(pattern) > len(bestPattern)))) {
				bestIndex = index
				bestEndIndex = endIndex
				bestPattern = pattern
				bestVal = utils.Ptr(patternVal.Value)
				bestStart = index
				bestEnd = endIndex

				captures := make([]string, (len(indices[0])-2)/2)
				for i := 0; i < len(captures); i++ {
					captures[i] = line[indices[0][2+i*2]:indices[0][2+i*2+1]]
				}
				bestCaps = captures
			}

			if index == 0 && endIndex == len(line)-1 {
				return bestVal, line[:bestStart] + line[bestEnd:], bestCaps
			}
		}
	}

	if bestVal != nil {
		return bestVal, line[:bestStart] + line[bestEnd:], bestCaps
	}

	return nil, line, nil
}

func parseMod(line string, order int) ([]mod.Mod, string) {
	lineLower := strings.ToLower(line)
	/*
		// TODO Check if this is a special modifier
		for pattern, patternVal in pairs(jewelFuncList) do
			local _, _, cap1, cap2, cap3, cap4, cap5 = lineLower:find(pattern, 1)
			if cap1 then
				return {MOD("JewelFunc", "LIST", mod.JewelFunc{func = patternVal.func(cap1, cap2, cap3, cap4, cap5), type = patternVal.type}) }
			end
		end
		local jewelFunc = jewelFuncList[lineLower]
		if jewelFunc then
			return { MOD("JewelFunc", "LIST", jewelFunc) }
		end
		local clusterJewelSkill = clusterJewelSkills[lineLower]
		if clusterJewelSkill then
			return clusterJewelSkill
		end
	*/

	if _, ok := unsupportedModList[lineLower]; ok {
		return nil, line
	}

	// TODO
	specialMod, specialLine, captures := scan(line, specialModListCompiled, false)
	if specialMod != nil && len(specialLine) == 0 {
		if specialFunc, ok := (*specialMod).(func(num float64, captures []string) ([]mod.Mod, string)); ok {
			if len(captures) == 0 {
				return specialFunc(0, captures)
			}
			return specialFunc(utils.Float(captures[0]), captures)
		}
		return (*specialMod).([]mod.Mod), ""
	}

	/*
		// TODO Check for add-to-cluster-jewel special
		local addToCluster = line:match("^Added Small Passive Skills also grant: (.+)$")
		if addToCluster then
			return { MOD("AddToClusterJewelNode", "LIST", addToCluster) }
		end
	*/

	line = line + " "

	// Check for a flag/tag specification at the start of the line
	var preFlag *modNameListType
	var preFlagCap []string
	preFlag, line, preFlagCap = scan(line, preFlagListCompiled, false)
	if preFlag != nil && preFlag.fn != nil {
		temp := preFlag.fn(preFlagCap)
		preFlag = &temp
	}

	// Check for skill name at the start of the line
	var skillTag *modNameListType
	skillTag, line, _ = scan(line, preSkillNameListCompiled, false)

	// Scan for modifier form
	var modForm *string
	var formCap []string
	modForm, line, formCap = scan(line, formListCompiled, false)
	if modForm == nil {
		return nil, line
	}

	var modTag *modNameListType
	var modTag2 *modNameListType
	var tagCap []string

	// Check for tags (per-charge, conditionals)
	modTag, line, tagCap = scan(line, modTagListCompiled, false)
	if modTag != nil && modTag.fn != nil {
		modTag = utils.Ptr(modTag.fn(tagCap))
	}

	if modTag != nil {
		modTag2, line, tagCap = scan(line, modTagListCompiled, false)
		if modTag2 != nil && modTag2.fn != nil {
			modTag2 = utils.Ptr(modTag2.fn(tagCap))
		}
	}

	// Scan for modifier name and skill name
	var modName *modNameListType
	if order == 2 && skillTag == nil {
		skillTag, line, _ = scan(line, skillNameListCompiled, true)
	}

	var flagName *modNameListType
	if *modForm == "PEN" {
		modName, line, _ = scan(line, penTypesCompiled, true)
		if modName == nil {
			return nil, line
		}
		_, line, _ = scan(line, modNameListCompiled, true)
	} else if *modForm == "FLAG" {
		flagName, line, _ = scan(line, flagTypesCompiled, false)
		if flagName == nil {
			return nil, line
		}
		modName, line, _ = scan(line, modNameListCompiled, true)
	} else {
		modName, line, _ = scan(line, modNameListCompiled, true)
	}

	if order == 1 && skillTag == nil {
		skillTag, line, _ = scan(line, skillNameListCompiled, true)
	}

	// Scan for flags
	var modFlag *modNameListType
	modFlag, line, _ = scan(line, modFlagListCompiled, true)

	// Find modifier value and type according to form
	modValue := []float64{0.0}
	if len(formCap) > 0 {
		modValue[0] = utils.Float(formCap[0])
	}

	modType := "BASE"
	var modSuffix *string

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
		modSuffix, line, _ = scan(line, suffixTypesCompiled, true)
	case "CHANCE":
		// Do nothing
	case "REGENPERCENT":
		modName = utils.Ptr(regenTypes[strings.ToLower(formCap[1])])
		modSuffix = utils.Ptr("Percent")
	case "REGENFLAT":
		modName = utils.Ptr(regenTypes[strings.ToLower(formCap[1])])
	case "DEGEN":
		damageType := dmgTypes[strings.ToLower(formCap[1])]
		if damageType == "" {
			return nil, line
		}
		modName = &modNameListType{names: []string{damageType + "Degen"}}
		modSuffix = utils.Ptr("")
	case "DMG":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return nil, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
	case "DMGATTACKS":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return nil, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: mod.KeywordFlagAttack}
		}
	case "DMGSPELLS":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return nil, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: mod.KeywordFlagSpell}
		}
	case "DMGBOTH":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			return nil, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: mod.KeywordFlagAttack | mod.KeywordFlagSpell}
		}
	case "FLAG":
		modName = flagName
		if flagName.tag != nil {
			// TODO Hexproof edge case
		} else {
			modType = "FLAG"
			modValue = []float64{1}
		}
	}

	if modName == nil {
		return nil, line
	}

	// Combine flags and tags
	flags := mod.MFlag(0)
	keywordFlags := mod.KeywordFlag(0)
	tagList := make([]mod.Tag, 0)
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
		misc.affectedByAura = misc.affectedByAura || datum.affectedByAura
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

		switch modType {
		case "BASE":
			modList[i] = mod.NewFloat(fullName, mod.TypeBase, realValue)
		case "INC":
			modList[i] = mod.NewFloat(fullName, mod.TypeIncrease, realValue)
		case "MORE":
			modList[i] = mod.NewFloat(fullName, mod.TypeMore, realValue)
		case "FLAG":
			modList[i] = mod.NewFlag(fullName, realValue > 0)
		}

		modList[i].Flag(flags)
		modList[i].KeywordFlag(keywordFlags)
		modList[i].Tag(tagList...)
	}

	if len(modList) > 0 {
		// Special handling for various modifier types
		if misc.addToAura {
			// Modifiers that add effects to your auras
			for i, effectMod := range modList {
				modList[i] = mod.NewList("ExtraAuraEffect", mod.ExtraAuraEffect{Mod: effectMod})
			}
		} else if misc.newAura {
			// Modifiers that add extra auras
			for i, effectMod := range modList {
				tagList := effectMod.Tags()
				effectMod.ClearTags()
				modList[i] = mod.NewList("ExtraAura", mod.ExtraAura{Mod: effectMod, OnlyAllies: misc.newAuraOnlyAllies}).Tag(tagList...)
			}
		} else if misc.affectedByAura {
			// Modifiers that apply to actors affected by your auras
			for i, effectMod := range modList {
				modList[i] = mod.NewList("AffectedByAuraMod", mod.AffectedByAuraMod{Mod: effectMod})
			}
		} else if misc.addToMinion {
			// Minion modifiers
			for i, effectMod := range modList {
				modList[i] = mod.NewList("MinionModifier", mod.MinionModifier{Mod: effectMod})
				if misc.addToMinionTag != nil {
					modList[i] = modList[i].Tag(misc.addToMinionTag)
				}
			}
		} else if misc.addToSkill != nil {
			// Skill enchants or socketed gem modifiers that add additional effects
			for i, effectMod := range modList {
				modList[i] = mod.NewList("ExtraSkillMod", mod.ExtraSkillMod{Mod: effectMod}).Tag(misc.addToSkill)
			}
		} else if misc.applyToEnemy {
			for i, effectMod := range modList {
				modList[i] = mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: effectMod})
			}
		}
	}

	if strings.Count(line, " ") > 0 {
		return modList, line
	}

	return modList, ""
}

type ModCacheEntry struct {
	ModList []mod.Mod
	Extra   string
}

var modCache = make(map[string]*ModCacheEntry)

func ParseMod(line string, isComb bool) *ModCacheEntry {
	if _, ok := modCache[line]; !ok {
		modList, extra := parseMod(line, 1)
		if modList != nil && extra != "" {
			modList, extra = parseMod(line, 2)
		}
		modCache[line] = &ModCacheEntry{
			ModList: modList,
			Extra:   extra,
		}

		/*
			TODO Unsupported mod logging
			if foo and not isComb and not cache[line][1] then
				local form = line:gsub("[%+%-]?%d+%.?%d*","{num}")
				if not unsupported[form] then
					unsupported[form] = true
					count = count + 1
					foo = io.open("../unsupported.txt", "a+")
					foo:write(count, ': ', form, (cache[line][2] and #cache[line][2] < #line and ('    {' .. cache[line][2]).. '}') or "", '\n')
					foo:close()
				end
			end
		*/
	}

	return modCache[line]
}

func init() {
	formListCompiled = make(map[string]CompiledList[string])
	for k, v := range formList {
		formListCompiled[k] = CompiledList[string]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	modNameListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range modNameList {
		modNameListCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	modFlagListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range modFlagList {
		modFlagListCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	preFlagListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range preFlagList {
		preFlagListCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	modTagListCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range modTagList {
		modTagListCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	suffixTypesCompiled = make(map[string]CompiledList[string])
	for k, v := range suffixTypes {
		suffixTypesCompiled[k] = CompiledList[string]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	dmgTypesCompiled = make(map[string]CompiledList[string])
	for k, v := range dmgTypes {
		dmgTypesCompiled[k] = CompiledList[string]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	penTypesCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range penTypes {
		penTypesCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	regenTypesCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range regenTypes {
		regenTypesCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	flagTypesCompiled = make(map[string]CompiledList[modNameListType])
	for k, v := range flagTypes {
		flagTypesCompiled[k] = CompiledList[modNameListType]{
			Regex: regexp.MustCompile(k),
			Value: v,
		}
	}

	specialModListCompiled = make(map[string]CompiledList[interface{}])
	for k, v := range specialModList {
		specialModListCompiled[k] = CompiledList[interface{}]{
			// Special mods get wrapped in start and end limits
			Regex: regexp.MustCompile("^" + k + "$"),
			Value: v,
		}
	}

	utils2.RegisterPostInitHook(initializeSkillNameList)
}
