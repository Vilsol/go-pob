package calculator

import (
	"maps"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	utils2 "github.com/Vilsol/go-pob-data/utils"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

type CompiledList[T any] struct {
	Regex *lazyRegexp
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
	"vorana":   {ID: "1", Type: "kalguur"},
	"uhtred":   {ID: "2", Type: "kalguur"},
	"medved":   {ID: "3", Type: "kalguur"},
	"tecrod":   {ID: "1", Type: "abyss_murderous"},
	"ulaman":   {ID: "1", Type: "abyss_searching"},
	"kurgal":   {ID: "1", Type: "abyss_hypnotic"},
	"amanamu":  {ID: "1", Type: "abyss_ghastly"},
	"zorath":   {ID: "1", Type: "abyss_special"},
}

// List of modifier forms
var addToClusterRe = regexp.MustCompile(`^Added Small Passive Skills also grant: (.+)$`)

var formListCompiled map[string]CompiledList[string]
var formList = map[string]string{
	`^(\d+)% increased`:              "INC",
	`^(\d+)% faster`:                 "INC",
	`^(\d+)% reduced`:                "RED",
	`^(\d+)% slower`:                 "RED",
	`^(\d+)% more`:                   "MORE",
	`^(\d+)% less`:                   "LESS",
	`^([\+\-][\d\.]+)%?`:             "BASE",
	`^([\+\-][\d\.]+)%? to`:          "BASE",
	`^([\+\-]?[\d\.]+)%? of`:         "BASE",
	`^([\+\-][\d\.]+)%? base`:        "BASE",
	`^([\+\-]?[\d\.]+)%? additional`: "BASE",
	`(\d+) additional hits?`:         "BASE",
	`^you gain ([\d\.]+)`:            "BASE",
	`^gains? ([\d\.]+)% of`:          "BASE",
	`^gain ([\d\.]+)`:                "BASE",
	//pob:block:src/Modules/ModParser.lua:87-94
	`^gain \+(\d+)% to`:     "BASE",
	`^you lose ([\d\.]+)`:   "LOSE",
	`^loses? ([\d\.]+)% of`: "LOSE",
	`^lose ([\d\.]+)`:       "LOSE",
	`^lose \+(\d+)% to`:     "LOSE",
	//pob:block:src/Modules/ModParser.lua:95-96
	`^grants ([\d\.]+)`:                   "GRANTS",
	`^removes? ([\d\.]+) ?o?f? ?y?o?u?r?`: "REMOVES",
	`(\d+) additional times?`:             "BASE",
	`^throw up to (\d+)`:                  "BASE",
	`^(\d+)`:                              "BASE",
	`^([\+\-]?\d+)% chance`:               "CHANCE",
	//pob:src/Modules/ModParser.lua:99
	`^([\+\-]?\d+)% chance to gain `:   "FLAG",
	`^([\+\-]?\d+)% additional chance`: "CHANCE",
	`penetrates? (\d+)%`:               "PEN",
	`penetrates (\d+)% of`:             "PEN",
	`penetrates (\d+)% of enemy`:       "PEN",
	//pob:src/Modules/ModParser.lua:106-108
	`^([\d\.]+) (.+) regenerated per second`:                 "REGENFLAT",
	`^([\d\.]+)% (.+) regenerated per second`:                "REGENPERCENT",
	`^([\d\.]+)% of (.+) regenerated per second`:             "REGENPERCENT",
	`^regenerate ([\d\.]+) (.+?) per second`:                 "REGENFLAT",
	`^regenerate ([\d\.]+)% (.+?) per second`:                "REGENPERCENT",
	`^regenerate ([\d\.]+)% of (.+?) per second`:             "REGENPERCENT",
	`^regenerate ([\d\.]+)% of your (.+?) per second`:        "REGENPERCENT",
	`^you regenerate ([\d\.]+)% of (.+?) per second`:         "REGENPERCENT",
	`^([\d\.]+) (\w+) damage taken per second`:               "DEGEN",
	`^([\d\.]+) (\w+) damage per second`:                     "DEGEN",
	`^([\d\.]+) (.+) lost per second`:                        "DEGENFLAT",
	`^([\d\.]+)% (.+) lost per second`:                       "DEGENPERCENT",
	`^([\d\.]+)% of (.+) lost per second`:                    "DEGENPERCENT",
	`^lose ([\d\.]+) (.+?) per second`:                       "DEGENFLAT",
	`^lose ([\d\.]+)% (.+?) per second`:                      "DEGENPERCENT",
	`^lose ([\d\.]+)% of (.+?) per second`:                   "DEGENPERCENT",
	`^lose ([\d\.]+)% of your (.+?) per second`:              "DEGENPERCENT",
	`^you lose ([\d\.]+)% of (.+?) per second`:               "DEGENPERCENT",
	`costs? ([\+\-]?\d+)`:                                    "TOTALCOST",
	`skills cost ([\+\-]?\d+)`:                               "BASECOST",
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
	`adds (\d+) to (\d+) (\w+) damage to spells`:             "DMGSPELLS",
	`adds (\d+)\-(\d+) (\w+) damage to spells`:               "DMGSPELLS",
	`adds (\d+) to (\d+) (\w+) spell damage`:                 "DMGSPELLS",
	`adds (\d+)\-(\d+) (\w+) spell damage`:                   "DMGSPELLS",
	`(\d+) to (\d+) added spell (\w+) damage`:                "DMGSPELLS",
	`adds (\d+) to (\d+) (\w+) damage to attacks and spells`: "DMGBOTH",
	`adds (\d+)\-(\d+) (\w+) damage to attacks and spells`:   "DMGBOTH",
	`adds (\d+) to (\d+) (\w+) damage to spells and attacks`: "DMGBOTH",
	`adds (\d+)\-(\d+) (\w+) damage to spells and attacks`:   "DMGBOTH",
	`adds (\d+) to (\d+) (\w+) damage to hits`:               "DMGBOTH",
	`adds (\d+)\-(\d+) (\w+) damage to hits`:                 "DMGBOTH",
	`^you have `:                                             "FLAG",
	`^have `:                                                 "FLAG",
	`^gain `:                                                 "FLAG",
	`^you gain `:                                             "FLAG",
	`^you are `:                                              "FLAG",
	`^are `:                                                  "FLAG",
	`is (-?\d+)%? `:                                          "OVERRIDE",
	`is doubled`:                                             "DOUBLED",
	`doubles?`:                                               "DOUBLED",
	`causes? double`:                                         "DOUBLED",
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
	newAura           bool

	flagModType  string
	flagModValue float64
}

// Map of modifier names
var modNameListCompiled map[string]CompiledList[modNameListType]
var modNameList = map[string]modNameListType{
	`acceleration shrine buff`: {names: []string{"Condition:AccelerationShrine"}},
	`adrenaline`:               {names: []string{"Condition:Adrenaline"}},
	`arcane surge`:             {names: []string{"Condition:ArcaneSurge"}},
	`armour applies to fire, cold and lightning damage taken from hits`: {names: []string{"ArmourAppliesToFireDamageTaken", "ArmourAppliesToColdDamageTaken", "ArmourAppliesToLightningDamageTaken"}},
	`arrow`:                                       {names: []string{"ProjectileCount"}, keywordFlags: mod.KeywordFlagArrow},
	`arrows`:                                      {names: []string{"ProjectileCount"}, keywordFlags: mod.KeywordFlagArrow},
	`bleed duration on you`:                       {names: []string{"SelfBleedDuration"}},
	`brutal shrine buff`:                          {names: []string{"Condition:BrutalShrine"}},
	`chain `:                                      {names: []string{"ChainCountMax"}},
	`chaos damage taken from attack hits`:         {names: []string{"ChaosDamageTakenFromAttacks"}},
	`chaos damage taken from hits`:                {names: []string{"ChaosDamageTakenWhenHit"}},
	`chaotic might`:                               {names: []string{"Condition:ChaoticMight"}},
	`cold damage taken from attack hits`:          {names: []string{"ColdDamageTakenFromAttacks"}},
	`cold damage taken from hits`:                 {names: []string{"ColdDamageTakenWhenHit"}},
	`cooldown use`:                                {names: []string{"AdditionalCooldownUses"}},
	`cooldown uses`:                               {names: []string{"AdditionalCooldownUses"}},
	`critical hit chance`:                         {names: []string{"CritChance"}},
	`damage taken from hits`:                      {names: []string{"DamageTakenWhenHit"}},
	`damage with bleed`:                           {names: []string{"Damage"}, keywordFlags: mod.KeywordFlagBleed},
	`damage with bleeding`:                        {names: []string{"Damage"}, keywordFlags: mod.KeywordFlagBleed},
	`damage with ignites inflicted`:               {names: []string{"Damage"}, keywordFlags: mod.KeywordFlagIgnite},
	`damage with poison`:                          {names: []string{"Damage"}, keywordFlags: mod.KeywordFlagPoison},
	`diamond shrine buff`:                         {names: []string{"Condition:DiamondShrine"}},
	`dps`:                                         {names: []string{"DPS"}},
	`duration of ailments inflicted on you`:       {names: []string{"SelfAilmentDuration"}},
	`duration of chills you inflict`:              {names: []string{"EnemyChillDuration"}},
	`duration of damaging ailments on you`:        {names: []string{"SelfIgniteDuration", "SelfBleedDuration", "SelfPoisonDuration"}},
	`duration of freezes you inflict`:             {names: []string{"EnemyFreezeDuration"}},
	`duration of ignites you inflict`:             {names: []string{"EnemyIgniteDuration"}},
	`duration of poisons on you`:                  {names: []string{"SelfPoisonDuration"}},
	`duration of shocks you inflict`:              {names: []string{"EnemyShockDuration"}},
	`echoing shrine buff`:                         {names: []string{"Condition:EchoingShrine"}},
	`effect of auras from your vaal skills`:       {names: []string{"AuraEffect"}, keywordFlags: mod.KeywordFlagVaal},
	`effect of chills you inflict`:                {names: []string{"EnemyChillEffect"}},
	`effect of exposure on you`:                   {names: []string{"ExposureEffectOnSelf"}},
	`effect of onslaught on you`:                  {names: []string{"OnslaughtEffect"}},
	`effect of shocks you inflict`:                {names: []string{"EnemyShockEffect"}},
	`effect of shrine buffs on you`:               {names: []string{"ShrineBuffEffect"}},
	`effect of withered on you`:                   {names: []string{"WitherEffectOnSelf"}},
	`elemental damage as energy shield`:           {names: []string{"ElementalDamageEnergyShieldLeech"}},
	`elemental taken`:                             {names: []string{"ElementalDamageTaken"}},
	`elusive`:                                     {names: []string{"Condition:CanBeElusive"}},
	`energy shield for each enemy hit`:            {names: []string{"EnergyShieldOnHit"}, flags: mod.MFlagHit},
	`energy shield per enemy hit`:                 {names: []string{"EnergyShieldOnHit"}, flags: mod.MFlagHit},
	`energy shield when you block`:                {names: []string{"EnergyShieldOnBlock"}},
	`evasion rating against melee attacks`:        {names: []string{"MeleeEvasion"}},
	`evasion rating against projectile attacks`:   {names: []string{"ProjectileEvasion"}},
	`fire damage taken from attack hits`:          {names: []string{"FireDamageTakenFromAttacks"}},
	`fire damage taken from hits`:                 {names: []string{"FireDamageTakenWhenHit"}},
	`gloom shrine buff`:                           {names: []string{"Condition:GloomShrine"}},
	`greater freezing shrine buff`:                {names: []string{"Condition:GreaterFreezingShrine"}},
	`greater shocking shrine buff`:                {names: []string{"Condition:GreaterShockingShrine"}},
	`greater skeletal shrine buff`:                {names: []string{"Condition:GreaterSkeletalShrine"}},
	`impenetrable shrine buff`:                    {names: []string{"Condition:ImpenetrableShrine"}},
	`lesser brutal shrine buff`:                   {names: []string{"Condition:LesserBrutalShrine"}},
	`lesser massive shrine buff`:                  {names: []string{"Condition:LesserMassiveShrine"}},
	`life and mana`:                               {names: []string{"Life", "Mana"}},
	`life and mana for each enemy hit`:            {names: []string{"LifeOnHit", "ManaOnHit"}, flags: mod.MFlagHit},
	`life for each enemy hit`:                     {names: []string{"LifeOnHit"}, flags: mod.MFlagHit},
	`life per enemy hit`:                          {names: []string{"LifeOnHit"}, flags: mod.MFlagHit},
	`life reservation`:                            {names: []string{"LifeReserved"}},
	`life reserved`:                               {names: []string{"LifeReserved"}},
	`lightning damage taken from attack hits`:     {names: []string{"LightningDamageTakenFromAttacks"}},
	`mana for each enemy hit`:                     {names: []string{"ManaOnHit"}, flags: mod.MFlagHit},
	`mana gained for each enemy hit`:              {names: []string{"ManaOnHit"}, flags: mod.MFlagHit},
	`mana per enemy hit`:                          {names: []string{"ManaOnHit"}, flags: mod.MFlagHit},
	`massive shrine buff`:                         {names: []string{"Condition:MassiveShrine"}},
	`maximum endurance, frenzy and power charges`: {names: []string{"EnduranceChargesMax", "PowerChargesMax", "FrenzyChargesMax"}},
	`maximum life and mana`:                       {names: []string{"Life", "Mana"}},
	`maximum number of animated weapons`:          {names: []string{"ActiveAnimatedWeaponLimit"}},
	`maximum number of holy armaments`:            {names: []string{"ActiveHolyStrikeMinionLimit"}},
	`maximum number of living lightning`:          {names: []string{"ActiveLivingLightningLimit"}},
	`maximum number of raised spectres`:           {names: []string{"ActiveSpectreLimit"}},
	`maximum number of raised spiders`:            {names: []string{"ActiveSpiderLimit"}},
	`maximum number of sentinels of absolution`:   {names: []string{"ActiveSentinelOfAbsolutionLimit"}},
	`maximum number of sentinels of dominance`:    {names: []string{"ActiveSentinelOfDominanceLimit"}},
	`maximum number of sentinels of purity`:       {names: []string{"ActiveSentinelOfPurityLimit"}},
	`maximum number of summoned reapers`:          {names: []string{"ActiveReaperLimit"}},
	`maximum number of summoned skeletons`:        {names: []string{"ActiveSkeletonLimit"}},
	`maximum number of summoned spectral wolves`:  {names: []string{"ActiveWolfLimit"}},
	`maximum physical damage`:                     {names: []string{"MaxPhysicalDamage"}},
	`maximum total life, mana and energy shield recovery per second from leech`: {names: []string{"MaxLifeLeechRate", "MaxManaLeechRate", "MaxEnergyShieldLeechRate"}},
	`metre to weapon range`:                          {names: []string{"WeaponRangeMetre"}},
	`metres to weapon range`:                         {names: []string{"WeaponRangeMetre"}},
	`minimum physical damage`:                        {names: []string{"MinPhysicalDamage"}},
	`number of summoned arbalists`:                   {names: []string{"ActiveArbalistLimit"}},
	`onslaught`:                                      {names: []string{"Condition:Onslaught"}},
	`phasing`:                                        {names: []string{"Condition:Phasing"}},
	`physical damage over time taken`:                {names: []string{"PhysicalDamageTakenOverTime"}},
	`physical damage taken from attack hits`:         {names: []string{"PhysicalDamageTakenFromAttacks"}},
	`poison duration on you`:                         {names: []string{"SelfPoisonDuration"}},
	`quantity of items dropped`:                      {names: []string{"LootQuantity"}},
	`rage effect`:                                    {names: []string{"RageEffect"}},
	`rampage`:                                        {names: []string{"Condition:Rampage"}},
	`rarity of items dropped`:                        {names: []string{"LootRarity"}},
	`replenishing shrine buff`:                       {names: []string{"Condition:ReplenishingShrine"}},
	`resistance shrine buff`:                         {names: []string{"Condition:ResistanceShrine"}},
	`resonating shrine buff`:                         {names: []string{"Condition:ResonatingShrine"}},
	`soul eater`:                                     {names: []string{"Condition:CanHaveSoulEater"}},
	`soul gain prevention duration`:                  {names: []string{"SoulGainPreventionDuration"}},
	`souls per use`:                                  {names: []string{"SoulCost"}},
	`to amount of suppressed spell damage prevented`: {names: []string{"SpellSuppressionEffect"}},
	`to avoid ailments`:                              {names: []string{"AvoidAilments"}},
	`to avoid all damage from hits`:                  {names: []string{"AvoidAllDamageFromHitsChance"}},
	`to avoid blind`:                                 {names: []string{"AvoidBlind"}},
	`to avoid damage of each element from hits`:      {names: []string{"AvoidFireDamageChance", "AvoidColdDamageChance", "AvoidLightningDamageChance"}},
	`to avoid non-damaging ailments on you`:          {names: []string{"AvoidShock", "AvoidFreeze", "AvoidChill", "AvoidSap", "AvoidBrittle", "AvoidScorch"}},
	`to avoid status ailments`:                       {names: []string{"AvoidAilments"}},
	`to bleed`:                                       {names: []string{"BleedChance"}},
	`to deal triple damage`:                          {names: []string{"TripleDamageChance"}},
	`to double stun duration`:                        {names: []string{"DoubleEnemyStunDurationChance"}},
	`to ignore enemy physical damage reduction`:      {names: []string{"ChanceToIgnoreEnemyPhysicalDamageReduction"}},
	`to ignore stuns while casting`:                  {names: []string{"AvoidInterruptStun"}},
	`to impale on attack hit`:                        {names: []string{"ImpaleChance"}, flags: mod.MFlagAttack},
	`unholy might`:                                   {names: []string{"Condition:UnholyMight", "Condition:CanWither"}},
	"strength":                                       {names: []string{"Str"}},
	"dexterity":                                      {names: []string{"Dex"}},
	"intelligence":                                   {names: []string{"Int"}},
	"omniscience":                                    {names: []string{"Omni"}},
	"strength and dexterity":                         {names: []string{"Str", "Dex", "StrDex"}},
	"strength and intelligence":                      {names: []string{"Str", "Int", "StrInt"}},
	"dexterity and intelligence":                     {names: []string{"Dex", "Int", "DexInt"}},
	"attributes":                                     {names: []string{"Str", "Dex", "Int", "All"}},
	"all attributes":                                 {names: []string{"Str", "Dex", "Int", "All"}},
	"devotion":                                       {names: []string{"Devotion"}},

	"life":                                    {names: []string{"Life"}},
	"maximum life":                            {names: []string{"Life"}},
	"life regeneration rate":                  {names: []string{"LifeRegen"}},
	"mana":                                    {names: []string{"Mana"}},
	"maximum mana":                            {names: []string{"Mana"}},
	"mana regeneration":                       {names: []string{"ManaRegen"}},
	"mana regeneration rate":                  {names: []string{"ManaRegen"}},
	"mana cost":                               {names: []string{"ManaCost"}},
	"mana cost of":                            {names: []string{"ManaCost"}},
	"mana cost of skills":                     {names: []string{"ManaCost"}},
	"mana cost of attacks":                    {names: []string{"ManaCost"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"total cost":                              {names: []string{"Cost"}},
	"total mana cost":                         {names: []string{"ManaCost"}},
	"total mana cost of skills":               {names: []string{"ManaCost"}},
	"life cost of skills":                     {names: []string{"LifeCost"}},
	"rage cost of skills":                     {names: []string{"RageCost"}},
	"cost of":                                 {names: []string{"Cost"}},
	"cost of skills":                          {names: []string{"Cost"}},
	"cost of retaliation skills":              {names: []string{"Cost"}, tag: mod.SkillType(string(data.SkillTypeRetaliation))},
	"cost efficiency":                         {names: []string{"CostEfficiency"}},
	"cost efficiency of skills":               {names: []string{"CostEfficiency"}},
	"cost efficiency of attacks":              {names: []string{"CostEfficiency"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"cost efficiency of spells":               {names: []string{"CostEfficiency"}, tag: mod.SkillType(string(data.SkillTypeSpell))},
	"mana cost efficiency":                    {names: []string{"ManaCostEfficiency"}},
	"mana cost efficiency of skills":          {names: []string{"ManaCostEfficiency"}},
	"mana cost efficiency of attacks":         {names: []string{"ManaCostEfficiency"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"mana cost efficiency of spells":          {names: []string{"ManaCostEfficiency"}, tag: mod.SkillType(string(data.SkillTypeSpell))},
	"life cost efficiency":                    {names: []string{"LifeCostEfficiency"}},
	"life cost efficiency of skills":          {names: []string{"LifeCostEfficiency"}},
	"life cost efficiency of attacks":         {names: []string{"LifeCostEfficiency"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"life cost efficiency of spells":          {names: []string{"LifeCostEfficiency"}, tag: mod.SkillType(string(data.SkillTypeSpell))},
	"energy shield cost efficiency":           {names: []string{"ESCostEfficiency"}},
	"energy shield cost efficiency of skills": {names: []string{"ESCostEfficiency"}},
	"es cost efficiency":                      {names: []string{"ESCostEfficiency"}},
	"es cost efficiency of skills":            {names: []string{"ESCostEfficiency"}},
	"soul cost efficiency":                    {names: []string{"SoulCostEfficiency"}},
	"soul cost efficiency of skills":          {names: []string{"SoulCostEfficiency"}},
	"rage cost efficiency":                    {names: []string{"RageCostEfficiency"}},
	"rage cost efficiency of skills":          {names: []string{"RageCostEfficiency"}},
	"mana reserved":                           {names: []string{"ManaReserved"}},
	"mana reservation":                        {names: []string{"ManaReserved"}},
	"mana reservation of skills":              {names: []string{"ManaReserved"}, tag: mod.SkillType(string(data.SkillTypeAura))},
	"mana reservation efficiency of skills":   {names: []string{"ManaReservationEfficiency"}},
	"life reservation efficiency of skills":   {names: []string{"LifeReservationEfficiency"}},
	"reservation of skills":                   {names: []string{"Reserved"}},
	"mana reservation if cast as an aura":     {names: []string{"ManaReserved"}, tag: mod.SkillType(string(data.SkillTypeAura))},
	"reservation if cast as an aura":          {names: []string{"Reserved"}, tag: mod.SkillType(string(data.SkillTypeAura))},
	"reservation":                             {names: []string{"Reserved"}},
	"reservation efficiency":                  {names: []string{"ReservationEfficiency"}},
	"reservation efficiency of skills":        {names: []string{"ReservationEfficiency"}},
	"mana reservation efficiency":             {names: []string{"ManaReservationEfficiency"}},
	"life reservation efficiency":             {names: []string{"LifeReservationEfficiency"}},

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

	"physical damage reduction":                                   {names: []string{"PhysicalDamageReduction"}},
	"physical damage reduction from hits":                         {names: []string{"PhysicalDamageReductionWhenHit"}},
	"elemental damage reduction":                                  {names: []string{"ElementalDamageReduction"}},
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
	"elemental and chaos resistances":                             {names: []string{"ElementalResist", "ChaosResist"}},
	"all resistances":                                             {names: []string{"ElementalResist", "ChaosResist"}},
	"all maximum elemental resistances":                           {names: []string{"ElementalResistMax"}},
	"all maximum resistances":                                     {names: []string{"ElementalResistMax", "ChaosResistMax"}},
	"all elemental resistances and maximum elemental resistances": {names: []string{"ElementalResist", "ElementalResistMax"}},
	"fire and chaos resistances":                                  {names: []string{"FireResist", "ChaosResist"}},
	"cold and chaos resistances":                                  {names: []string{"ColdResist", "ChaosResist"}},
	"lightning and chaos resistances":                             {names: []string{"LightningResist", "ChaosResist"}},

	// Damage taken
	"damage taken":                                          {names: []string{"DamageTaken"}},
	"damage taken when hit":                                 {names: []string{"DamageTakenWhenHit"}},
	"damage over time taken":                                {names: []string{"DamageTakenOverTime"}},
	"damage taken from damage over time":                    {names: []string{"DamageTakenOverTime"}},
	"attack damage taken":                                   {names: []string{"AttackDamageTaken"}},
	"spell damage taken":                                    {names: []string{"SpellDamageTaken"}},
	"physical damage taken":                                 {names: []string{"PhysicalDamageTaken"}},
	"physical damage from hits taken":                       {names: []string{"PhysicalDamageFromHitsTaken"}},
	"physical damage taken when hit":                        {names: []string{"PhysicalDamageTakenWhenHit"}},
	"physical damage taken from hits":                       {names: []string{"PhysicalDamageTakenWhenHit"}},
	"physical damage taken from attacks":                    {names: []string{"PhysicalDamageTakenFromAttacks"}},
	"physical damage taken from projectile attacks":         {names: []string{"PhysicalDamageTakenFromProjectileAttacks"}},
	"physical damage taken from projectile attack hits":     {names: []string{"PhysicalDamageTakenFromProjectileAttacks"}},
	"fire, cold and lightning damage taken from spell hits": {names: []string{"FireDamageTakenFromSpells", "ColdDamageTakenFromSpells", "LightningDamageTakenFromSpells"}},
	"damage taken of each damage type from spell hits":      {names: []string{"DamageTakenFromSpells"}},
	"physical damage taken over time":                       {names: []string{"PhysicalDamageTakenOverTime"}},
	"physical damage over time damage taken":                {names: []string{"PhysicalDamageTakenOverTime"}},
	"reflected physical damage taken":                       {names: []string{"PhysicalReflectedDamageTaken"}},
	"lightning damage taken":                                {names: []string{"LightningDamageTaken"}},
	"lightning damage from hits taken":                      {names: []string{"LightningDamageFromHitsTaken"}},
	"lightning damage taken when hit":                       {names: []string{"LightningDamageTakenWhenHit"}},
	"lightning damage taken from attacks":                   {names: []string{"LightningDamageTakenFromAttacks"}},
	"lightning damage taken over time":                      {names: []string{"LightningDamageTakenOverTime"}},
	"cold damage taken":                                     {names: []string{"ColdDamageTaken"}},
	"cold damage from hits taken":                           {names: []string{"ColdDamageFromHitsTaken"}},
	"cold damage taken when hit":                            {names: []string{"ColdDamageTakenWhenHit"}},
	"cold damage taken from attacks":                        {names: []string{"ColdDamageTakenFromAttacks"}},
	"cold damage taken over time":                           {names: []string{"ColdDamageTakenOverTime"}},
	"fire damage taken":                                     {names: []string{"FireDamageTaken"}},
	"fire damage from hits taken":                           {names: []string{"FireDamageFromHitsTaken"}},
	"fire damage taken when hit":                            {names: []string{"FireDamageTakenWhenHit"}},
	"fire damage taken from attacks":                        {names: []string{"FireDamageTakenFromAttacks"}},
	"fire damage taken over time":                           {names: []string{"FireDamageTakenOverTime"}},
	"chaos damage taken":                                    {names: []string{"ChaosDamageTaken"}},
	"chaos damage from hits taken":                          {names: []string{"ChaosDamageFromHitsTaken"}},
	"chaos damage taken when hit":                           {names: []string{"ChaosDamageTakenWhenHit"}},
	"chaos damage taken from attacks":                       {names: []string{"ChaosDamageTakenFromAttacks"}},
	"chaos damage taken over time":                          {names: []string{"ChaosDamageTakenOverTime"}},
	"chaos damage over time taken":                          {names: []string{"ChaosDamageTakenOverTime"}},
	"elemental damage taken":                                {names: []string{"ElementalDamageTaken"}},
	"elemental damage from hits taken":                      {names: []string{"ElementalDamageFromHitsTaken"}},
	"elemental damage taken when hit":                       {names: []string{"ElementalDamageTakenWhenHit"}},
	"elemental damage taken from hits":                      {names: []string{"ElementalDamageTakenWhenHit"}},
	"elemental damage taken over time":                      {names: []string{"ElementalDamageTakenOverTime"}},
	"cold and lightning damage taken":                       {names: []string{"ColdDamageTaken", "LightningDamageTaken"}},
	"fire and lightning damage taken":                       {names: []string{"FireDamageTaken", "LightningDamageTaken"}},
	"fire and cold damage taken":                            {names: []string{"FireDamageTaken", "ColdDamageTaken"}},
	"physical and chaos damage taken":                       {names: []string{"PhysicalDamageTaken", "ChaosDamageTaken"}},
	"reflected elemental damage taken":                      {names: []string{"ElementalReflectedDamageTaken"}},

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
	"to block":                                       {names: []string{"BlockChance"}},
	"to block attacks":                               {names: []string{"BlockChance"}},
	"to block attack damage":                         {names: []string{"BlockChance"}},
	"block chance":                                   {names: []string{"BlockChance"}},
	"chance to block":                                {names: []string{"BlockChance"}},
	"to block projectile spell damage":               {names: []string{"ProjectileSpellBlockChance"}},
	"chance to block attack and spell damage":        {names: []string{"BlockChance", "SpellBlockChance"}},
	"block chance with staves":                       {names: []string{"BlockChance"}, tag: mod.Condition("UsingStaff")},
	"to block with staves":                           {names: []string{"BlockChance"}, tag: mod.Condition("UsingStaff")},
	"block chance against projectiles":               {names: []string{"ProjectileBlockChance"}},
	"to block projectile attack damage":              {names: []string{"ProjectileBlockChance"}},
	"spell block chance":                             {names: []string{"SpellBlockChance"}},
	"to block spells":                                {names: []string{"SpellBlockChance"}},
	"to block spell damage":                          {names: []string{"SpellBlockChance"}},
	"chance to block attacks and spells":             {names: []string{"BlockChance", "SpellBlockChance"}},
	"to block attack and spell damage":               {names: []string{"BlockChance", "SpellBlockChance"}},
	"maximum block chance":                           {names: []string{"BlockChanceMax"}},
	"maximum chance to block attack damage":          {names: []string{"BlockChanceMax"}},
	"maximum chance to block spell damage":           {names: []string{"SpellBlockChanceMax"}},
	"life gained when you block":                     {names: []string{"LifeOnBlock"}},
	"mana gained when you block":                     {names: []string{"ManaOnBlock"}},
	"maximum chance to dodge spell hits":             {names: []string{"SpellDodgeChanceMax"}},
	"to avoid physical damage from hits":             {names: []string{"AvoidPhysicalDamageChance"}},
	"to avoid fire damage when hit":                  {names: []string{"AvoidFireDamageChance"}},
	"to avoid fire damage from hits":                 {names: []string{"AvoidFireDamageChance"}},
	"to avoid cold damage when hit":                  {names: []string{"AvoidColdDamageChance"}},
	"to avoid cold damage from hits":                 {names: []string{"AvoidColdDamageChance"}},
	"to avoid lightning damage when hit":             {names: []string{"AvoidLightningDamageChance"}},
	"to avoid lightning damage from hits":            {names: []string{"AvoidLightningDamageChance"}},
	"to avoid elemental damage when hit":             {names: []string{"AvoidFireDamageChance", "AvoidColdDamageChance", "AvoidLightningDamageChance"}},
	"to avoid elemental damage from hits":            {names: []string{"AvoidFireDamageChance", "AvoidColdDamageChance", "AvoidLightningDamageChance"}},
	"to avoid projectiles":                           {names: []string{"AvoidProjectilesChance"}},
	"to avoid being stunned":                         {names: []string{"AvoidStun"}},
	"to avoid interruption from stuns while casting": {names: []string{"AvoidInteruptStun"}},
	"to avoid being shocked":                         {names: []string{"AvoidShock"}},
	"to avoid being frozen":                          {names: []string{"AvoidFreeze"}},
	"to avoid being chilled":                         {names: []string{"AvoidChill"}},
	"to avoid being ignited":                         {names: []string{"AvoidIgnite"}},
	//pob:src/Modules/ModParser.lua:407-408
	"to avoid elemental ailments":                     {names: []string{"AvoidElementalAilments"}},
	"to avoid elemental status ailments":              {names: []string{"AvoidElementalAilments"}},
	"to avoid bleeding":                               {names: []string{"AvoidBleed"}},
	"to avoid being poisoned":                         {names: []string{"AvoidPoison"}},
	"to avoid being impaled":                          {names: []string{"AvoidImpale"}},
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

	"life and mana leech is instant":                        {names: []string{"InstantManaLeech", "InstantLifeLeech"}},
	"life leech is instant":                                 {names: []string{"InstantLifeLeech"}},
	"mana leech is instant":                                 {names: []string{"InstantManaLeech"}},
	"energy shield leech is instant":                        {names: []string{"InstantEnergyShieldLeech"}},
	"leech is instant":                                      {names: []string{"InstantEnergyShieldLeech", "InstantManaLeech", "InstantLifeLeech"}},
	"damage taken recouped as life":                         {names: []string{"LifeRecoup"}},
	"physical damage taken recouped as life":                {names: []string{"PhysicalLifeRecoup"}},
	"lightning damage taken recouped as life":               {names: []string{"LightningLifeRecoup"}},
	"cold damage taken recouped as life":                    {names: []string{"ColdLifeRecoup"}},
	"fire damage taken recouped as life":                    {names: []string{"FireLifeRecoup"}},
	"chaos damage taken recouped as life":                   {names: []string{"ChaosLifeRecoup"}},
	"damage taken recouped as energy shield":                {names: []string{"EnergyShieldRecoup"}},
	"damage taken recouped as mana":                         {names: []string{"ManaRecoup"}},
	"damage taken recouped as life, mana and energy shield": {names: []string{"LifeRecoup", "EnergyShieldRecoup", "ManaRecoup"}},
	"missing unreserved life before being hit by an enemy":  {names: []string{"MissingLifeBeforeEnemyHit"}},
	"missing unreserved mana before being hit by an enemy":  {names: []string{"MissingManaBeforeEnemyHit"}},

	// Stun/knockback modifiers
	"stun recovery":                {names: []string{"StunRecovery"}},
	"stun and block recovery":      {names: []string{"StunRecovery"}},
	"block and stun recovery":      {names: []string{"StunRecovery"}},
	"stun duration on you":         {names: []string{"StunDuration"}},
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
	"effect of the buff granted by your stone golems":            {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Stone Golem").Transfigured(true)},
	"effect of the buff granted by your lightning golems":        {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Lightning Golem").Transfigured(true)},
	"effect of the buff granted by your ice golems":              {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Ice Golem").Transfigured(true)},
	"effect of the buff granted by your flame golems":            {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Flame Golem").Transfigured(true)},
	"effect of the buff granted by your chaos golems":            {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Chaos Golem").Transfigured(true)},
	"effect of the buff granted by your carrion golems":          {names: []string{"BuffEffect"}, tag: mod.SkillName("Summon Carrion Golem").Transfigured(true)},
	"effect of offering spells":                                  {names: []string{"BuffEffect"}, tag: mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering")},
	"effect of offerings":                                        {names: []string{"BuffEffect"}, tag: mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering")},
	"effect of heralds on you":                                   {names: []string{"BuffEffect"}, tag: mod.SkillType(string(data.SkillTypeHerald))},
	"effect of herald buffs on you":                              {names: []string{"BuffEffect"}, tag: mod.SkillType(string(data.SkillTypeHerald))},
	"effect of buffs granted by your active ancestor totems":     {names: []string{"BuffEffect"}, tag: mod.SkillName("Ancestral Warchief", "Ancestral Protector", "Earthbreaker")},
	"effect of buffs your ancestor totems grant ":                {names: []string{"BuffEffect"}, tag: mod.SkillName("Ancestral Warchief", "Ancestral Protector", "Earthbreaker")},
	"effect of withered":                                         {names: []string{"WitherEffect"}},
	"warcry effect":                                              {names: []string{"BuffEffect"}, keywordFlags: mod.KeywordFlagWarcry},
	"aspect of the avian buff effect":                            {names: []string{"BuffEffect"}, tag: mod.SkillName("Aspect of the Avian")},
	"maximum rage":                                               {names: []string{"MaximumRage"}},
	"minimum rage":                                               {names: []string{"MinimumRage"}},
	"maximum fortification":                                      {names: []string{"MaximumFortification"}},
	"maximum valour":                                             {names: []string{"MaximumValour"}},
	"fortification":                                              {names: []string{"MinimumFortification"}},
	"fortification duration":                                     {names: []string{"FortifyDuration"}},

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
	"maximum spirit charges":                              {names: []string{"SpiritChargesMax"}},
	"charge duration":                                     {names: []string{"ChargeDuration"}},

	// On hit/kill/leech effects
	"life gained on kill":                                               {names: []string{"LifeOnKill"}},
	"life per enemy killed":                                             {names: []string{"LifeOnKill"}},
	"life on kill":                                                      {names: []string{"LifeOnKill"}},
	"mana gained on kill":                                               {names: []string{"ManaOnKill"}},
	"mana per enemy killed":                                             {names: []string{"ManaOnKill"}},
	"mana on kill":                                                      {names: []string{"ManaOnKill"}},
	"energy shield gained on kill":                                      {names: []string{"EnergyShieldOnKill"}},
	"energy shield per enemy killed":                                    {names: []string{"EnergyShieldOnKill"}},
	"energy shield on kill":                                             {names: []string{"EnergyShieldOnKill"}},
	"life gained for each enemy hit":                                    {names: []string{"LifeOnHit"}, flags: mod.MFlagHit},
	"energy shield gained for each enemy hit":                           {names: []string{"EnergyShieldOnHit"}, flags: mod.MFlagHit},
	"life and mana gained for each enemy hit":                           {names: []string{"LifeOnHit", "ManaOnHit"}, flags: mod.MFlagHit},
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
	"effects of impale inflicted":                                       {names: []string{"ImpaleEffect"}},
	"effect of impales inflicted":                                       {names: []string{"ImpaleEffect"}},
	"impale duration":                                                   {names: []string{"ImpaleDuration"}},

	// Projectile modifiers
	"projectile":       {names: []string{"ProjectileCount"}},
	"projectiles":      {names: []string{"ProjectileCount"}},
	"projectile speed": {names: []string{"ProjectileSpeed"}},
	"arrow speed":      {names: []string{"ProjectileSpeed"}, flags: mod.MFlagBow},

	// Totem/trap/mine/brand modifiers
	"totem placement speed":                          {names: []string{"TotemPlacementSpeed"}},
	"totem life":                                     {names: []string{"TotemLife"}},
	"totem duration":                                 {names: []string{"TotemDuration"}},
	"maximum number of summoned totems":              {names: []string{"ActiveTotemLimit"}},
	"maximum number of summoned totems.":             {names: []string{"ActiveTotemLimit"}},
	"maximum number of summoned searing bond totems": {names: []string{"ActiveTotemLimit"}, tag: mod.SkillName("Searing Bond").Transfigured(true)},
	"maximum number of summoned ballista totems":     {names: []string{"ActiveBallistaLimit"}, tag: mod.SkillType("TotemsAreBallistae")},
	"trap throwing speed":                            {names: []string{"TrapThrowingSpeed"}},
	"trap and mine throwing speed":                   {names: []string{"TrapThrowingSpeed", "MineLayingSpeed"}},
	"trap trigger area of effect":                    {names: []string{"TrapTriggerAreaOfEffect"}},
	"trap duration":                                  {names: []string{"TrapDuration"}},
	"cooldown recovery speed for throwing traps":     {names: []string{"CooldownRecovery"}, keywordFlags: mod.KeywordFlagTrap},
	"cooldown recovery rate for throwing traps":      {names: []string{"CooldownRecovery"}, keywordFlags: mod.KeywordFlagTrap},
	"additional trap":                                {names: []string{"TrapThrowCount"}},
	"additional traps":                               {names: []string{"TrapThrowCount"}},
	"mine laying speed":                              {names: []string{"MineLayingSpeed"}},
	"mine throwing speed":                            {names: []string{"MineLayingSpeed"}},
	"mine detonation area of effect":                 {names: []string{"MineDetonationAreaOfEffect"}},
	"mine duration":                                  {names: []string{"MineDuration"}},
	"additional mine":                                {names: []string{"MineThrowCount"}},
	"additional mines":                               {names: []string{"MineThrowCount"}},
	"activation frequency":                           {names: []string{"BrandActivationFrequency"}},
	"brand activation frequency":                     {names: []string{"BrandActivationFrequency"}},
	"brand attachment range":                         {names: []string{"BrandAttachmentRange"}},

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
	"minion duration":                           {names: []string{"Duration"}, tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeCreatesMinion)), mod.SkillType(string(data.SkillTypeMinion))}},
	"skeleton duration":                         {names: []string{"Duration"}, tag: mod.SkillName("Summon Skeletons").Transfigured(true)},
	"sentinel of dominance duration":            {names: []string{"Duration"}, tag: mod.SkillName("Dominating Blow").Transfigured(true)},

	// Other skill modifiers
	"radius":                                {names: []string{"AreaOfEffect"}},
	"radius of area skills":                 {names: []string{"AreaOfEffect"}},
	"area of effect radius":                 {names: []string{"AreaOfEffect"}},
	"area of effect":                        {names: []string{"AreaOfEffect"}},
	"area of effect of skills":              {names: []string{"AreaOfEffect"}},
	"area of effect of area skills":         {names: []string{"AreaOfEffect"}},
	"aspect of the spider area of effect":   {names: []string{"AreaOfEffect"}, tag: mod.SkillName("Aspect of the Spider")},
	"firestorm explosion area of effect":    {names: []string{"AreaOfEffectSecondary"}, tag: mod.SkillName("Firestorm").Transfigured(true)},
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
	"metres to melee strike range":          {names: []string{"MeleeWeaponRangeMetre", "UnarmedRangeMetre"}},
	"metre to melee strike range":           {names: []string{"MeleeWeaponRangeMetre", "UnarmedRangeMetre"}},
	"to deal double damage":                 {names: []string{"DoubleDamageChance"}},

	"onslaught effect":          {names: []string{"OnslaughtEffect"}},
	"adrenaline duration":       {names: []string{"AdrenalineDuration"}},
	"effect of tailwind on you": {names: []string{"TailwindEffectOnSelf"}},
	"elusive effect":            {names: []string{"ElusiveEffect"}},
	"effect of elusive on you":  {names: []string{"ElusiveEffect"}},
	"effect of infusion":        {names: []string{"InfusionEffect"}},

	// Basic damage types
	"damage":                   {names: []string{"Damage"}},
	"physical damage":          {names: []string{"PhysicalDamage"}},
	"lightning damage":         {names: []string{"LightningDamage"}},
	"cold damage":              {names: []string{"ColdDamage"}},
	"fire damage":              {names: []string{"FireDamage"}},
	"chaos damage":             {names: []string{"ChaosDamage"}},
	"non-chaos damage":         {names: []string{"NonChaosDamage"}},
	"elemental damage":         {names: []string{"ElementalDamage"}},
	"minimum lightning damage": {names: []string{"MinLightningDamage"}},
	"maximum lightning damage": {names: []string{"MaxLightningDamage"}},
	"minimum cold damage":      {names: []string{"MinColdDamage"}},
	"maximum cold damage":      {names: []string{"MaxColdDamage"}},
	"minimum fire damage":      {names: []string{"MinFireDamage"}},
	"maximum fire damage":      {names: []string{"MaxFireDamage"}},
	"minimum chaos damage":     {names: []string{"MinChaosDamage"}},
	"maximum chaos damage":     {names: []string{"MaxChaosDamage"}},

	// Other damage forms
	"attack damage":                        {names: []string{"Damage"}, flags: mod.MFlagAttack},
	"attack physical damage":               {names: []string{"PhysicalDamage"}, flags: mod.MFlagAttack},
	"physical attack damage":               {names: []string{"PhysicalDamage"}, flags: mod.MFlagAttack},
	"minimum physical attack damage":       {names: []string{"MinPhysicalDamage"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"maximum physical attack damage":       {names: []string{"MaxPhysicalDamage"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"minimum attack damage":                {names: []string{"MinDamage"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"maximum attack damage":                {names: []string{"MaxDamage"}, tag: mod.SkillType(string(data.SkillTypeAttack))},
	"physical weapon damage":               {names: []string{"PhysicalDamage"}, flags: mod.MFlagWeapon},
	"physical damage with weapons":         {names: []string{"PhysicalDamage"}, flags: mod.MFlagWeapon},
	"melee damage":                         {names: []string{"Damage"}, flags: mod.MFlagMelee},
	"physical melee damage":                {names: []string{"PhysicalDamage"}, flags: mod.MFlagMelee},
	"melee physical damage":                {names: []string{"PhysicalDamage"}, flags: mod.MFlagMelee},
	"projectile damage":                    {names: []string{"Damage"}, flags: mod.MFlagProjectile},
	"projectile attack damage":             {names: []string{"Damage"}, flags: mod.MFlagProjectile | mod.MFlagAttack},
	"bow damage":                           {names: []string{"Damage"}, flags: mod.MFlagBow | mod.MFlagHit},
	"damage with arrow hits":               {names: []string{"Damage"}, flags: mod.MFlagHit, keywordFlags: mod.KeywordFlagArrow},
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
	"effect of freeze on you":                     {names: []string{"SelfFreezeEffect"}},
	"effect of non-damaging ailments":             {names: []string{"EnemyShockEffect", "EnemyChillEffect", "EnemyFreezeEffect", "EnemyScorchEffect", "EnemyBrittleEffect", "EnemySapEffect"}},
	"effect of non-damaging ailments you inflict": {names: []string{"EnemyShockEffect", "EnemyChillEffect", "EnemyFreezeEffect", "EnemyScorchEffect", "EnemyBrittleEffect", "EnemySapEffect"}},
	"effect of non-damaging ailments on you":      {names: []string{"SelfShockEffect", "SelfChillEffect", "SelfFreezeEffect", "SelfScorchEffect", "SelfBrittleEffect", "SelfSapEffect"}},
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
	"duration of elemental ailments":              {names: []string{"EnemyElementalAilmentDuration"}},
	"duration of elemental ailments on you":       {names: []string{"SelfElementalAilmentDuration"}},
	"duration of elemental status ailments":       {names: []string{"EnemyElementalAilmentDuration"}},
	"duration of ailments":                        {names: []string{"EnemyAilmentDuration"}},
	"duration of ailments on you":                 {names: []string{"SelfAilmentDuration"}},
	"elemental ailment duration on you":           {names: []string{"SelfElementalAilmentDuration"}},
	"duration of ailments you inflict":            {names: []string{"EnemyAilmentDuration"}},
	"duration of ailments inflicted":              {names: []string{"EnemyAilmentDuration"}},

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

	// Flask and Tincture modifiers
	//pob:src/Modules/ModParser.lua:855
	"effect":              {names: []string{"LocalEffect"}},
	"effect of flasks":    {names: []string{"FlaskEffect"}},
	"effect of tinctures": {names: []string{"TinctureEffect"}},

	"effect of flasks on you":        {names: []string{"FlaskEffect"}},
	"amount recovered":               {names: []string{"FlaskRecovery"}},
	"life recovered":                 {names: []string{"FlaskRecovery"}},
	"life recovery from flasks used": {names: []string{"FlaskLifeRecovery"}},
	"mana recovered":                 {names: []string{"FlaskRecovery"}},
	//pob:src/Modules/ModParser.lua:861
	"recovery":                                  {names: []string{"FlaskLifeRecoveryLowLife"}},
	"life recovery from flasks":                 {names: []string{"FlaskLifeRecovery"}},
	"mana recovery from flasks":                 {names: []string{"FlaskManaRecovery"}},
	"life and mana recovery from flasks":        {names: []string{"FlaskLifeRecovery", "FlaskManaRecovery"}},
	"flask effect duration":                     {names: []string{"FlaskDuration"}},
	"recovery speed":                            {names: []string{"FlaskRecoveryRate"}},
	"recovery rate":                             {names: []string{"FlaskRecoveryRate"}},
	"flask recovery rate":                       {names: []string{"FlaskRecoveryRate"}},
	"flask recovery speed":                      {names: []string{"FlaskRecoveryRate"}},
	"flask life recovery rate":                  {names: []string{"FlaskLifeRecoveryRate"}},
	"flask mana recovery rate":                  {names: []string{"FlaskManaRecoveryRate"}},
	"extra charges":                             {names: []string{"FlaskCharges"}},
	"maximum charges":                           {names: []string{"FlaskCharges"}},
	"charges used":                              {names: []string{"FlaskChargesUsed"}},
	"charges per use":                           {names: []string{"FlaskChargesUsed"}},
	"flask charges used":                        {names: []string{"FlaskChargesUsed"}},
	"flask charges gained":                      {names: []string{"FlaskChargesGained"}},
	"charge recovery":                           {names: []string{"FlaskChargeRecovery"}},
	"for flasks you use to not consume charges": {names: []string{"FlaskChanceNotConsumeCharges"}},
	"for tinctures to not inflict mana burn":    {names: []string{"TincturesNotInflictManaBurn"}},
	"mana burn rate":                            {names: []string{"TinctureManaBurnRate"}},
	"impales you inflict last":                  {names: []string{"ImpaleStacksMax"}},
}

// List of modifier flags
var modFlagListCompiled map[string]CompiledList[modNameListType]
var modFlagList = map[string]modNameListType{
	`against damage over time`:      {flags: mod.MFlagDot},
	`by attacks`:                    {keywordFlags: mod.KeywordFlagAttack},
	`by retaliation skills`:         {tag: mod.SkillType(string(data.SkillTypeRetaliation))},
	`by spells`:                     {keywordFlags: mod.KeywordFlagSpell},
	`by your attacks`:               {keywordFlags: mod.KeywordFlagAttack},
	`by your spells`:                {keywordFlags: mod.KeywordFlagSpell},
	`channelling`:                   {tag: mod.SkillType(string(data.SkillTypeChannel))},
	`claw attacks`:                  {flags: mod.MFlagClaw | mod.MFlagHit},
	`curse skills`:                  {keywordFlags: mod.KeywordFlagCurse},
	`from melee weapon attacks`:     {flags: mod.MFlagWeaponMelee | mod.MFlagHit},
	`from melee weapons`:            {flags: mod.MFlagWeaponMelee},
	`lightning skills`:              {keywordFlags: mod.KeywordFlagLightning},
	`mark skills`:                   {tag: mod.SkillType(string(data.SkillTypeMark))},
	`melee weapon`:                  {flags: mod.MFlagWeaponMelee},
	`non-channelling`:               {tag: mod.SkillType(string(data.SkillTypeChannel)).Neg(true)},
	`non-channelling skills`:        {tag: mod.SkillType(string(data.SkillTypeChannel)).Neg(true)},
	`of mark skills`:                {tag: mod.SkillType(string(data.SkillTypeMark))},
	`skills which throw traps have`: {keywordFlags: mod.KeywordFlagTrap},
	`with chaos damage`:             {tag: mod.Condition("ChaosHasDamage")},
	`with cold damage`:              {tag: mod.Condition("ColdHasDamage")},
	`with fire damage`:              {tag: mod.Condition("FireHasDamage")},
	`with lightning damage`:         {tag: mod.Condition("LightningHasDamage")},
	`with melee weapon attacks`:     {flags: mod.MFlagWeaponMelee},
	`with melee weapons`:            {flags: mod.MFlagWeaponMelee},
	`with minion skills`:            {tag: mod.SkillType(string(data.SkillTypeMinion))},
	`with physical damage`:          {tag: mod.Condition("PhysicalHasDamage")},
	`with spell skills`:             {keywordFlags: mod.KeywordFlagSpell},
	`with triggered spells`:         {keywordFlags: mod.KeywordFlagSpell, tag: mod.SkillType(string(data.SkillTypeTriggered))},
	`with unarmed melee attack`:     {flags: mod.MFlagUnarmed | mod.MFlagMelee},
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

	"spell":                            {flags: mod.MFlagSpell},
	"with spells":                      {keywordFlags: mod.KeywordFlagSpell},
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
	"with melee skills":                {tag: mod.SkillType(string(data.SkillTypeMelee))},
	"with bow skills":                  {keywordFlags: mod.KeywordFlagBow},
	"on melee hit":                     {flags: mod.MFlagMelee | mod.MFlagHit},
	"on hit":                           {flags: mod.MFlagHit},
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
	"brand":                            {keywordFlags: mod.KeywordFlagBrand},
	"totem":                            {keywordFlags: mod.KeywordFlagTotem},
	"with totem skills":                {keywordFlags: mod.KeywordFlagTotem},
	"for skills used by totems":        {keywordFlags: mod.KeywordFlagTotem},
	"totem skills that cast an aura":   {tag: mod.SkillType(string(data.SkillTypeAura)), keywordFlags: mod.KeywordFlagTotem},
	"aura skills that summon totems":   {tag: mod.SkillType(string(data.SkillTypeAura)), keywordFlags: mod.KeywordFlagTotem},
	"of aura skills":                   {tag: mod.SkillType(string(data.SkillTypeAura))},
	//pob:block:src/Modules/ModParser.lua:1033-1034
	"link skills":                  {tag: mod.SkillType(string(data.SkillTypeLink))},
	"of link skills":               {tag: mod.SkillType(string(data.SkillTypeLink))},
	"of curse skills":              {keywordFlags: mod.KeywordFlagCurse},
	"with curse skills":            {keywordFlags: mod.KeywordFlagCurse},
	"of curse aura skills":         {tag: mod.SkillType(string(data.SkillTypeAura)), keywordFlags: mod.KeywordFlagCurse},
	"of curse auras":               {keywordFlags: mod.KeywordFlagCurse | mod.KeywordFlagAura | mod.KeywordFlagMatchAll},
	"of hex skills":                {tag: mod.SkillType(string(data.SkillTypeHex))},
	"with hex skills":              {tag: mod.SkillType(string(data.SkillTypeHex))},
	"of herald skills":             {tag: mod.SkillType(string(data.SkillTypeHerald))},
	"with herald skills":           {tag: mod.SkillType(string(data.SkillTypeHerald))},
	"with hits from herald skills": {tag: mod.SkillType(string(data.SkillTypeHerald)), keywordFlags: mod.KeywordFlagHit},
	"minion skills":                {tag: mod.SkillType(string(data.SkillTypeMinion))},
	"of minion skills":             {tag: mod.SkillType(string(data.SkillTypeMinion))},
	"for curses":                   {keywordFlags: mod.KeywordFlagCurse},
	"for hexes":                    {tag: mod.SkillType(string(data.SkillTypeHex))},
	"warcry":                       {keywordFlags: mod.KeywordFlagWarcry},
	"vaal":                         {keywordFlags: mod.KeywordFlagVaal},
	"vaal skill":                   {keywordFlags: mod.KeywordFlagVaal},
	"with vaal skills":             {keywordFlags: mod.KeywordFlagVaal},
	"with non-vaal skills":         {tag: mod.SkillType(string(data.SkillTypeVaal)).Neg(true)},
	"with movement skills":         {keywordFlags: mod.KeywordFlagMovement},
	"of movement skills":           {keywordFlags: mod.KeywordFlagMovement},
	"of movement skills used":      {keywordFlags: mod.KeywordFlagMovement},
	"of travel skills":             {tag: mod.SkillType(string(data.SkillTypeTravel))},
	"of banner skills":             {tag: mod.SkillType(string(data.SkillTypeBanner))},
	"with lightning skills":        {keywordFlags: mod.KeywordFlagLightning},
	"with cold skills":             {keywordFlags: mod.KeywordFlagCold},
	"with fire skills":             {keywordFlags: mod.KeywordFlagFire},
	"with elemental skills":        {keywordFlags: mod.KeywordFlagLightning | mod.KeywordFlagCold | mod.KeywordFlagFire},
	"with chaos skills":            {keywordFlags: mod.KeywordFlagChaos},
	"with physical skills":         {keywordFlags: mod.KeywordFlagPhysical},
	"with channelling skills":      {tag: mod.SkillType(string(data.SkillTypeChannel))},
	"channelling skills":           {tag: mod.SkillType(string(data.SkillTypeChannel))},
	"with brand skills":            {tag: mod.SkillType(string(data.SkillTypeBrand))},
	"for stance skills":            {tag: mod.SkillType(string(data.SkillTypeStance))},
	"of stance skills":             {tag: mod.SkillType(string(data.SkillTypeStance))},
	"with skills that cost life":   {tag: mod.StatThreshold("LifeCost", 1)},
	"minion":                       {addToMinion: true},
	"zombie":                       {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie").Transfigured(true)},
	"raised zombie":                {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie").Transfigured(true)},
	"skeleton":                     {addToMinion: true, addToMinionTag: mod.SkillName("Summon Skeletons").Transfigured(true)},
	"spectre":                      {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre").Transfigured(true)},
	"raised spectre":               {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre").Transfigured(true)},
	"golem":                        {addToMinion: true, addToMinionTag: mod.SkillType(string(data.SkillTypeGolem))},
	"chaos golem":                  {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem").Transfigured(true)},
	"flame golem":                  {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem").Transfigured(true)},
	"increased flame golem":        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem").Transfigured(true)},
	"ice golem":                    {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem").Transfigured(true)},
	"lightning golem":              {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem").Transfigured(true)},
	"stone golem":                  {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem").Transfigured(true)},
	"animated guardian":            {addToMinion: true, addToMinionTag: mod.SkillName("Animate Guardian").Transfigured(true)},

	"global":                          {tag: mod.Global()},
	"from equipped shield":            {tag: mod.SlotName("Weapon 2")},
	"from equipped armour items":      {tag: mod.SlotName("Helmet", "Body Armour", "Gloves", "Boots", "Weapon 2")},
	"from equipped helmet":            {tag: mod.SlotName("Helmet")},
	"from equipped boots and gloves":  {tag: mod.SlotName("Gloves", "Boots")},
	"from your equipped body armour":  {tag: mod.SlotName("Body Armour")},
	"from equipped body armour":       {tag: mod.SlotName("Body Armour")},
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
	`^zombies [hd][ae][va][el] `:                                     {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie").Transfigured(true)},
	`^raised zombies [hd][ae][va][el] `:                              {addToMinion: true, addToMinionTag: mod.SkillName("Raise Zombie").Transfigured(true)},
	`^skeletons [hd][ae][va][el] `:                                   {addToMinion: true, addToMinionTag: mod.SkillName("Summon Skeletons").Transfigured(true)},
	`^raging spirits [hd][ae][va][el] `:                              {addToMinion: true, addToMinionTag: mod.SkillName("Summon Raging Spirit").Transfigured(true)},
	`^summoned raging spirits [hd][ae][va][el] `:                     {addToMinion: true, addToMinionTag: mod.SkillName("Summon Raging Spirit").Transfigured(true)},
	`^r?a?i?s?e?d? ?spectres [hd][ae][va][el] `:                      {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre").Transfigured(true)},
	`^chaos golems [hd][ae][va][el] `:                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem").Transfigured(true)},
	`^summoned chaos golems [hd][ae][va][el] `:                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Chaos Golem").Transfigured(true)},
	`^flame golems [hd][ae][va][el] `:                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem").Transfigured(true)},
	`^summoned flame golems [hd][ae][va][el] `:                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Flame Golem").Transfigured(true)},
	`^ice golems [hd][ae][va][el] `:                                  {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem").Transfigured(true)},
	`^summoned ice golems [hd][ae][va][el] `:                         {addToMinion: true, addToMinionTag: mod.SkillName("Summon Ice Golem").Transfigured(true)},
	`^lightning golems [hd][ae][va][el] `:                            {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem").Transfigured(true)},
	`^summoned lightning golems [hd][ae][va][el] `:                   {addToMinion: true, addToMinionTag: mod.SkillName("Summon Lightning Golem").Transfigured(true)},
	`^stone golems [hd][ae][va][el] `:                                {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem").Transfigured(true)},
	`^summoned stone golems [hd][ae][va][el] `:                       {addToMinion: true, addToMinionTag: mod.SkillName("Summon Stone Golem").Transfigured(true)},
	`^summoned carrion golems [hd][ae][va][el] `:                     {addToMinion: true, addToMinionTag: mod.SkillName("Summon Carrion Golem").Transfigured(true)},
	`^summoned skitterbots [hd][ae][va][el] `:                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Carrion Golem").Transfigured(true)},
	`^blink arrow and blink arrow clones [hd][ae][va][el] `:          {addToMinion: true, addToMinionTag: mod.SkillName("Blink Arrow").Transfigured(true)},
	`^mirror arrow and mirror arrow clones [hd][ae][va][el] `:        {addToMinion: true, addToMinionTag: mod.SkillName("Mirror Arrow").Transfigured(true)},
	`^animated weapons [hd][ae][va][el] `:                            {addToMinion: true, addToMinionTag: mod.SkillName("Animate Weapon").Transfigured(true)},
	`^animated guardians? deals? `:                                   {addToMinion: true, addToMinionTag: mod.SkillName("Animate Guardian").Transfigured(true)},
	`^summoned holy relics [hd][ae][va][el] `:                        {addToMinion: true, addToMinionTag: mod.SkillName("Summon Holy Relic").Transfigured(true)},
	`^summoned reaper [dh][ea][as]l?s? `:                             {addToMinion: true, addToMinionTag: mod.SkillName("Summon Reaper").Transfigured(true)},
	`^summoned arbalists [hgdf][aei][vair][eln] `:                    {addToMinion: true, addToMinionTag: mod.SkillName("Summon Arbalists")},
	`^summoned arbalists' attacks have `:                             {addToMinion: true, addToMinionTag: mod.SkillName("Summon Arbalists")},
	`^herald skills [hd][ae][va][el] `:                               {tag: mod.SkillType(string(data.SkillTypeHerald))},
	`^agony crawler deals `:                                          {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Agony")},
	`^summoned agony crawler fires `:                                 {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Agony")},
	`^sentinels of purity deal `:                                     {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity")},
	`^summoned sentinels of absolution have `:                        {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity")},
	`^summoned sentinels have `:                                      {addToMinion: true, addToMinionTag: mod.SkillName("Herald of Purity", "Dominating Blow", "Absolution").Transfigured(true)},
	`^raised zombies' slam attack has `:                              {addToMinion: true, tag: mod.SkillId("ZombieSlam")},
	`^raised spectres, raised zombies, and summoned skeletons have `: {addToMinion: true, addToMinionTag: mod.SkillName("Raise Spectre", "Raise Zombie", "Summon Skeletons").Transfigured(true)},

	// Totem/trap/mine
	`^attacks used by totems have `:                  {flags: mod.MFlagAttack, keywordFlags: mod.KeywordFlagTotem},
	`^spells cast by totems [hd][ae][va][el] `:       {flags: mod.MFlagSpell, keywordFlags: mod.KeywordFlagTotem},
	`^trap and mine damage `:                         {keywordFlags: mod.KeywordFlagTrap | mod.KeywordFlagMine},
	`^skills used by traps [hgd][ae][via][enl] `:     {keywordFlags: mod.KeywordFlagTrap},
	`^skills which throw traps [hgd][ae][via][enl] `: {keywordFlags: mod.KeywordFlagTrap},
	`^skills used by mines [hgd][ae][via][enl] `:     {keywordFlags: mod.KeywordFlagMine},

	// Local damage
	`^attacks with this weapon `:                  {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`^attacks with this weapon [hd][ae][va][el] `: {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`^hits with this weapon [hd][ae][va][el] `:    {flags: mod.MFlagHit, tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},

	`^attacks [hd][ae][va][el] `:                  {flags: mod.MFlagAttack},
	`^attack skills [hd][ae][va][el] `:            {keywordFlags: mod.KeywordFlagAttack},
	`^spells [hd][ae][va][el] a? ?`:               {flags: mod.MFlagSpell},
	`^spell skills [hd][ae][va][el] `:             {tag: mod.SkillType(string(data.SkillTypeSpell))},
	`^projectile attack skills [hd][ae][va][el] `: {tag: mod.SkillType(string(data.SkillTypeRangedAttack))},
	`^projectiles from attacks [hd][ae][va][el] `: {tag: mod.SkillType(string(data.SkillTypeRangedAttack))},
	`^arrows [hd][ae][va][el] `:                   {tag: mod.KeywordFlagAnd(mod.KeywordFlagArrow)},
	`^bow skills [hdf][aei][var][el] `:            {keywordFlags: mod.KeywordFlagBow},
	`^projectiles [hdf][aei][var][el] `:           {flags: mod.MFlagProjectile},
	`^projectiles `:                               {flags: mod.MFlagProjectile},
	`^melee attacks have `:                        {flags: mod.MFlagMelee},
	`^movement attack skills have `:               {flags: mod.MFlagAttack, keywordFlags: mod.KeywordFlagMovement},
	`^travel skills have `:                        {tag: mod.SkillType(string(data.SkillTypeTravel))},
	`^link skills have `:                          {tag: mod.SkillType(string(data.SkillTypeLink))},
	`^lightning skills [hd][ae][va][el] a? ?`:     {keywordFlags: mod.KeywordFlagLightning},
	`^lightning spells [hd][ae][va][el] a? ?`:     {keywordFlags: mod.KeywordFlagLightning, flags: mod.MFlagSpell},
	`^cold skills [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagCold},
	`^cold spells [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagCold, flags: mod.MFlagSpell},
	`^fire skills [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagFire},
	`^fire spells [hd][ae][va][el] a? ?`:          {keywordFlags: mod.KeywordFlagFire, flags: mod.MFlagSpell},
	`^chaos skills [hd][ae][va][el] a? ?`:         {keywordFlags: mod.KeywordFlagChaos},
	`^vaal skills [hd][ae][va][el] `:              {keywordFlags: mod.KeywordFlagVaal},
	`^brand skills [hd][ae][va][el] `:             {tag: mod.SkillType(string(data.SkillTypeBrand))},
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
	`^retaliation skills [hd][ae][va][el] `:       {tag: mod.SkillType(string(data.SkillTypeRetaliation))},
	`^pact skills [hd][ae][va][el] `:              {tag: mod.SkillType(string(data.SkillTypePact))},
	`^non\-curse aura skills have `:               {tagList: []mod.Tag{mod.SkillType(string(data.SkillTypeAura)), mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)}},
	`^non\-channelling skills have `:              {tag: mod.SkillType(string(data.SkillTypeChannel)).Neg(true)},
	`^non\-vaal skills deal `:                     {tag: mod.SkillType(string(data.SkillTypeVaal)).Neg(true)},
	`^skills [hdfg][aei][vari][eln] `:             {},

	// Slot specific
	`^left ring slot: `:                                   {tag: mod.SlotNumber(1)},
	`^right ring slot: `:                                  {tag: mod.SlotNumber(2)},
	`^socketed gems ([hgd]?[ae]?[via]?[enl]?\s?)`:         {addToSkill: mod.SocketedIn("{SlotName}")},
	`^socketed skills [hgd][ae][via][enl] `:               {addToSkill: mod.SocketedIn("{SlotName}")},
	`^socketed travel skills [hgd][ae][via][enl] `:        {addToSkill: mod.SocketedIn("{SlotName}").Keyword("travel")},
	`^socketed warcry skills [hgd][ae][via][enl] `:        {addToSkill: mod.SocketedIn("{SlotName}").Keyword("warcry")},
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
	//pob:src/Modules/ModParser.lua:1251
	`^enemies withered by you [th]a[vk]e `: {tag: mod.MultiplierThreshold("WitheredStack").Threshold(1), applyToEnemy: true},
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
	`^while a pinnacle atlas boss is in your presence, enemies you've hit recently have `: {
		playerTagList: []mod.Tag{mod.Condition("HitRecently"), mod.ActorCondition("enemy", "PinnacleBoss")},
		applyToEnemy:  true,
	},
	`^while a unique enemy is in your presence, enemies you've hit recently have `: {
		playerTagList: []mod.Tag{mod.Condition("HitRecently"), mod.ActorCondition("enemy", "RareOrUnique")},
		applyToEnemy:  true,
	},
	`^enemies you've hit recently have `:                          {playerTag: mod.Condition("HitRecently"), applyToEnemy: true},
	`^enemies shocked or frozen by you take `:                     {tag: mod.Condition("Shocked", "Frozen"), applyToEnemy: true, modSuffix: "Taken"},
	`^enemies affected by your spider's webs [thd][ae][avk][el] `: {tag: mod.MultiplierThreshold("Spider's WebStack").Threshold(1), applyToEnemy: true},
	`^enemies you curse take `:                                    {tag: mod.Condition("Cursed"), applyToEnemy: true, modSuffix: "Taken"},
	`^enemies you curse ?h?a?v?e? `:                               {tag: mod.Condition("Cursed"), applyToEnemy: true},
	`^nearby enemies take `:                                       {modSuffix: "Taken", applyToEnemy: true},
	`^nearby enemies have `:                                       {applyToEnemy: true},
	`^nearby enemies deal `:                                       {applyToEnemy: true},
	`^nearby enemies'? `:                                          {applyToEnemy: true},
	`^enemies near your totems deal `:                             {applyToEnemy: true},
	`against you`:                                                 {applyToEnemy: true},

	`^your flasks grant `:                         {},
	`^when hit, `:                                 {},
	`^you and allies [hgd][ae][via][enl] `:        {},
	`^auras from your skills grant `:              {addToAura: true},
	`^you and nearby allies `:                     {newAura: true},
	`^you and nearby allies [hgd][ae][via][enl] `: {newAura: true},
	`^nearby allies [hgd][ae][via][enl] `:         {newAura: true, newAuraOnlyAllies: true},
	//pob:block:src/Modules/ModParser.lua:1297
	`^you and allies affected by auras from your skills [hgd][ae][via][enl] `: {tag: mod.Condition("AffectedByAura")},
	//pob:block:src/Modules/ModParser.lua:1291
	`^banners also grant `: {addToAura: true, onlyAddToBanners: true},
	//pob:block:src/Modules/ModParser.lua:1283
	`^hits against you `: {applyToEnemy: true, flags: mod.MFlagHit},

	`^every rage also grants `: {tag: mod.Multiplier("RageEffect")},
	`^each rage also grants `:  {tag: mod.Multiplier("RageEffect")},
	`^auras grant `:            {addToAura: true},

	`^totems have `: {keywordFlags: mod.KeywordFlagTotem},
	`^skills used by totems [hd][ae][va][el] `:                           {keywordFlags: mod.KeywordFlagTotem},
	`^vaal skills require `:                                              {keywordFlags: mod.KeywordFlagVaal},
	`^non\-aura vaal skills require `:                                    {keywordFlags: mod.KeywordFlagVaal, tag: mod.SkillType(string(data.SkillTypeAura)).Neg(true)},
	`^skills which throw mines ([hgd]?[ae]?[via]?[enl]?[ ]?)`:            {keywordFlags: mod.KeywordFlagMine},
	`^skills used by your traps and mines ([hgd]?[ae]?[via]?[enl]?[ ]?)`: {keywordFlags: mod.KeywordFlagTrap | mod.KeywordFlagMine},

	`^non\-minion skills have `:            {tag: mod.SkillType(string(data.SkillTypeMinion)).Neg(true)},
	`^banner skills [hd][ae][va][el] `:     {tag: mod.SkillType(string(data.SkillTypeBanner))},
	`^melee weapon damage`:                 {flags: mod.MFlagWeaponMelee},
	`^melee weapon hits [hd][ae][va][el] `: {flags: mod.MFlagWeaponMelee | mod.MFlagHit},

	`^for each nearby corpse, `:                        {tag: mod.Multiplier("NearbyCorpse")},
	`^consecrated ground you create a?l?s?o? ?grants `: {tag: mod.Condition("OnConsecratedGround")},
	`^nearby enemy monsters' `:                         {applyToEnemy: true},
	`^deal `:                                           {},
	`^take `:                                           {modSuffix: "Taken"},
	`^marauder: `:                                      {tag: mod.Condition("ConnectedToMarauderStart")},
	`^duelist: `:                                       {tag: mod.Condition("ConnectedToDuelistStart")},
	`^ranger: `:                                        {tag: mod.Condition("ConnectedToRangerStart")},
	`^shadow: `:                                        {tag: mod.Condition("ConnectedToShadowStart")},
	`^witch: `:                                         {tag: mod.Condition("ConnectedToWitchStart")},
	`^templar: `:                                       {tag: mod.Condition("ConnectedToTemplarStart")},
	`^scion: `:                                         {tag: mod.Condition("ConnectedToScionStart")},
	`^skills supported by spellslinger have `:                               {tag: mod.Condition("SupportedBySpellslinger")},
	`^skills that have dealt a critical strike in the past 8 seconds deal `: {tag: mod.Condition("CritInPast8Sec")},
	`^blink arrow and mirror arrow have `:                                   {tag: mod.SkillName("Blink Arrow", "Mirror Arrow").Transfigured(true)},
	`attacks with energy blades `:                                           {flags: mod.MFlagAttack, tag: mod.Condition(conditionAffectedByEnergyBlade)},

	// While in the presence of...
	`^while a unique enemy is in your presence, minions [hd][ae][va][el] `:        {addToMinion: true, playerTag: mod.ActorCondition("enemy", "RareOrUnique")},
	`^while a pinnacle atlas boss is in your presence, minions [hd][ae][va][el] `: {addToMinion: true, playerTag: mod.ActorCondition("enemy", "PinnacleBoss")},
	`^minions created recently have `:                                             {addToMinion: true, playerTag: mod.Condition("MinionsCreatedRecently")},
	`^while a unique enemy is in your presence, `:                                 {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	`^while a pinnacle atlas boss is in your presence, `:                          {tag: mod.ActorCondition("enemy", "PinnacleBoss")},
}

// List of modifier tags
var modTagListCompiled map[string]CompiledList[modNameListType]
var modTagList = map[string]modNameListType{
	`for each unlinked socket in equipped two handed weapon`: {tagList: []mod.Tag{
		mod.Multiplier("UnlinkedSocketInWeapon 1"), mod.Condition("UsingTwoHandedWeapon"),
	}},
	//pob:block:src/Modules/ModParser.lua:1319-2008
	`per brine charge`:                       {tag: mod.Multiplier("BrineCharge")},
	`per spirit charge`:                      {tag: mod.Multiplier("SpiritCharge")},
	`per maximum power charge`:               {tag: mod.Multiplier("PowerChargeMax")},
	`per rage while you are not losing rage`: {tag: mod.Multiplier("Rage")},
	`per mana burn`:                          {tag: mod.Multiplier("ManaBurnStacks")},
	`per mana burn on you`:                   {tag: mod.Multiplier("ManaBurnStacks")},
	`per mana burn, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ManaBurnStacks").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`per equipped claw`:    {tag: mod.Multiplier("ClawItem")},
	`per equipped dagger`:  {tag: mod.Multiplier("DaggerItem")},
	`per equipped axe`:     {tag: mod.Multiplier("AxeItem")},
	`per equipped ring`:    {tag: mod.Multiplier("RingItem")},
	`per equipped flask`:   {tag: mod.Multiplier("FlaskItem")},
	`per equipped sword`:   {tag: mod.Multiplier("SwordItem")},
	`per equipped jewel`:   {tag: mod.Multiplier("JewelItem")},
	`per equipped mace`:    {tag: mod.Multiplier("MaceItem")},
	`per equipped sceptre`: {tag: mod.Multiplier("SceptreItem")},
	`per equipped wand`:    {tag: mod.Multiplier("WandItem")},
	`per claw`:             {tag: mod.Multiplier("ClawItem")},
	`per dagger`:           {tag: mod.Multiplier("DaggerItem")},
	`per axe`:              {tag: mod.Multiplier("AxeItem")},
	`per ring`:             {tag: mod.Multiplier("RingItem")},
	`per flask`:            {tag: mod.Multiplier("FlaskItem")},
	`per sword`:            {tag: mod.Multiplier("SwordItem")},
	`per jewel`:            {tag: mod.Multiplier("JewelItem")},
	`per mace`:             {tag: mod.Multiplier("MaceItem")},
	`per sceptre`:          {tag: mod.Multiplier("SceptreItem")},
	`per wand`:             {tag: mod.Multiplier("WandItem")},
	`for each herald b?u?f?f?s?k?i?l?l? ?affecting you`:     {tag: mod.Multiplier("Herald")},
	`per hit suppressed recently`:                           {tag: mod.Multiplier("HitsSuppressedRecently")},
	`for each curse on the enemy`:                           {tag: mod.Multiplier("CurseOnEnemy")},
	`per minion from your non-vaal skills`:                  {tag: mod.Multiplier("NonVaalSummonedMinion")},
	`per minion`:                                            {tag: mod.Multiplier("SummonedMinion")},
	`for each empty red socket on any equipped item`:        {tag: mod.Multiplier("EmptyRedSocketsInAnySlot")},
	`for each empty green socket on any equipped item`:      {tag: mod.Multiplier("EmptyGreenSocketsInAnySlot")},
	`for each empty blue socket on any equipped item`:       {tag: mod.Multiplier("EmptyBlueSocketsInAnySlot")},
	`for each empty white socket on any equipped item`:      {tag: mod.Multiplier("EmptyWhiteSocketsInAnySlot")},
	`per socketed gem`:                                      {tag: mod.Multiplier("SocketedGemsIn{SlotName}")},
	`per socketed red gem`:                                  {tag: mod.Multiplier("SocketedRedGemsIn{SlotName}")},
	`per socketed green gem`:                                {tag: mod.Multiplier("SocketedGreenGemsIn{SlotName}")},
	`per socketed blue gem`:                                 {tag: mod.Multiplier("SocketedBlueGemsIn{SlotName}")},
	`per socketed murderous eye jewel`:                      {tag: mod.Multiplier("MurderousEyeJewelIn{SlotName}")},
	`per socketed searching eye jewel`:                      {tag: mod.Multiplier("SearchingEyeJewelIn{SlotName}")},
	`per socketed hypnotic eye jewel`:                       {tag: mod.Multiplier("HypnoticEyeJewelIn{SlotName}")},
	`per socketed ghastly eye jewel`:                        {tag: mod.Multiplier("GhastlyEyeJewelIn{SlotName}")},
	`for each different type of mastery you have allocated`: {tag: mod.Multiplier("AllocatedMasteryType")},
	`per grand spectrum`:                                    {tag: mod.Multiplier("GrandSpectrum")},
	`per elemental ailment you've inflicted recently`:       {tag: mod.Multiplier("AppliedAilmentsRecently")},
	`for each (\d+)% of life reserved`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LifeReservedPercent")}
		},
	},
	`per dexterity`:     {tag: mod.PerStat(0, "Dex")},
	`per soul required`: {tag: mod.PerStat(0, "SoulCost")},
	`per (\d+) energy shield on equipped gloves`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnGloves")}
		},
	},
	`per (\d+) maximum energy shield on equipped helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnHelmet")}
		},
	},
	`per (\d+) energy shield on equipped helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnHelmet")}
		},
	},
	`per (\d+) energy shield on equipped boots`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnBoots")}
		},
	},
	`per (\d+) energy shield on equipped body armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShieldOnBody Armour")}
		},
	},
	`per (\d+) evasion rating on equipped gloves`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnGloves")}
		},
	},
	`per (\d+) evasion rating on equipped helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnHelmet")}
		},
	},
	`per (\d+) evasion on equipped boots`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBoots")}
		},
	},
	`per (\d+) evasion rating on equipped boots`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBoots")}
		},
	},
	`per (\d+) evasion rating on equipped body armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EvasionOnBody Armour")}
		},
	},
	`per (\d+) armour on equipped gloves`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnGloves")}
		},
	},
	`per (\d+) armour on equipped helmet`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnHelmet")}
		},
	},
	`per (\d+) armour on equipped boots`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnBoots")}
		},
	},
	`per (\d+) armour on equipped body armour`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnBody Armour")}
		},
	},
	`per (\d+) armour or evasion rating on equipped shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ArmourOnWeapon 2", "EvasionOnWeapon 2")}
		},
	},
	`per (\d+)% fire resistance above 75%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "FireResistOver75")}
		},
	},
	`for each time they've pierced`:                      {tag: mod.PerStat(0, "PiercedCount")},
	`while holding a fishing rod`:                        {tag: mod.Condition("UsingFishing")},
	`if used while on low life`:                          {tag: mod.Condition("LowLife")},
	`on reaching low life`:                               {tag: mod.Condition("LowLife")},
	`while channelling snipe`:                            {tag: mod.Condition("Channelling")},
	`while at maximum rage`:                              {tag: mod.Condition("HaveMaximumRage")},
	`while you have unbroken ward`:                       {tag: mod.Condition("UnbrokenWard")},
	`while your ward is broken`:                          {tag: mod.Condition("UnbrokenWard").Neg(true)},
	`if you[' ]h?a?ve suppressed spell damage recently`:  {tag: mod.Condition("SuppressedRecently")},
	`while leeching life`:                                {tag: mod.Condition("LeechingLife")},
	`while leeching mana`:                                {tag: mod.Condition("LeechingMana")},
	`if you've used a life flask in the past 10 seconds`: {tag: mod.Condition("UsingLifeFlask")},
	`if you've used a mana flask in the past 10 seconds`: {tag: mod.Condition("UsingManaFlask")},
	`while you have an active tincture`:                  {tag: mod.Condition("UsingTincture")},
	`while you have a tincture active`:                   {tag: mod.Condition("UsingTincture")},
	`while on caustic ground`:                            {tag: mod.Condition("OnCausticGround")},
	`wh[ei][nl][ e] ?you are poisoned`:                   {tag: mod.Condition("Poisoned")},
	`per (\d+)% ([a-z]+) effect on enemy`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier(firstToUpper(caps[1]) + "Effect").Div(utils.Float(caps[0])).Actor("enemy")}
		},
	},
	`per withered debuff on enemy`:                                {tag: mod.Multiplier("WitheredStack").Actor("enemy").Limit(15)},
	`while you have a linked target`:                              {tag: mod.MultiplierThreshold("LinkedTargets").Threshold(1)},
	`when you create consecrated ground`:                          {},
	`on hit with spells`:                                          {tag: mod.Condition("HitSpellRecently")},
	`when you deal a critical strike`:                             {tag: mod.Condition("CritRecently")},
	`on killing taunted enemies`:                                  {tag: mod.Condition("KilledTauntedEnemyRecently")},
	`on kill`:                                                     {tag: mod.Condition("KilledRecently")},
	`on melee kill`:                                               {flags: mod.MFlagWeaponMelee, tag: mod.Condition("KilledRecently")},
	`when you kill an enemy`:                                      {tag: mod.Condition("KilledRecently")},
	`on throwing a trap`:                                          {tag: mod.Condition("TrapOrMineThrownRecently")},
	`if you[' ]h?a?ve been stunned while casting recently`:        {tag: mod.Condition("StunnedWhileCastingRecently")},
	`when you warcry`:                                             {tag: mod.Condition("UsedWarcryRecently")},
	`for each warcry exerting them`:                               {tag: mod.Multiplier("ExertingWarcryCount")},
	`[fp][oe]r ?e?a?c?h? mine detonated recently`:                 {tag: mod.Multiplier("MineDetonatedRecently")},
	`when you summon a totem`:                                     {tag: mod.Condition("SummonedTotemRecently")},
	`when you use a movement skill`:                               {tag: mod.Condition("UsedMovementSkillRecently")},
	`if you[' ]h?a?ve used a socketed vaal skill recently`:        {tag: mod.Condition("UsedVaalSkillRecently")},
	`when you use a vaal skill`:                                   {tag: mod.Condition("UsedVaalSkillRecently")},
	`if you[' ]h?a?ve used a retaliation skill recently`:          {tag: mod.Condition("UsedRetaliationRecently")},
	`if you've used an amethyst flask recently`:                   {tag: mod.Condition("UsedAmethystFlaskRecently")},
	`if you've used a ruby flask recently`:                        {tag: mod.Condition("UsedRubyFlaskRecently")},
	`if you've used a sapphire flask recently`:                    {tag: mod.Condition("UsedSapphireFlaskRecently")},
	`if you've used a topaz flask recently`:                       {tag: mod.Condition("UsedTopazFlaskRecently")},
	`when your mine is detonated targeting an enemy`:              {tag: mod.Condition("DetonatedMinesRecently")},
	`when your trap is triggered by an enemy`:                     {tag: mod.Condition("TriggeredTrapsRecently")},
	`if energy shield recharge has started in the past 2 seconds`: {tag: mod.Condition("EnergyShieldRechargePastTwoSec")},
	`per enemy hit taken recently`:                                {tag: mod.Multiplier("BeenHitRecently")},
	`if you've shattered an enemy recently`:                       {tag: mod.Condition("ShatteredEnemyRecently")},
	`while affected by flasks?`:                                   {tag: mod.Condition("UsingFlask")},
	`while affected by no flasks?`:                                {tag: mod.Condition("UsingFlask").Neg(true)},
	`by enemies killed with a critical strike`:                    {tagList: []mod.Tag{mod.Condition("CritRecently"), mod.Condition("KilledRecently")}},
	`for each time you have shocked a non-shocked enemy recently, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("ShockedNonShockedEnemyRecently").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	//pob:src/Modules/ModParser.lua:1322
	`for (\d+) seconds`: {},
	`if you have reserved life and mana`: {tagList: []mod.Tag{
		mod.StatThreshold("LifeReserved", 1),
		mod.StatThreshold("ManaReserved", 1),
	}},
	`if your equipped staff has a red and blue socket`: {tagList: []mod.Tag{
		mod.MultiplierThreshold("RedSocketInWeapon 1").Threshold(1),
		mod.MultiplierThreshold("BlueSocketInWeapon 1").Threshold(1),
		mod.Condition("UsingStaff"),
	}},
	`for each different non-instant spell you[' ]h?a?ve cast recently`: {tag: mod.Multiplier("NonInstantSpellCastRecently")},
	`if equipped helmet, body armour, gloves, and boots all have armour`: {tagList: []mod.Tag{
		mod.StatThreshold("ArmourOnHelmet", 1),
		mod.StatThreshold("ArmourOnBody Armour", 1),
		mod.StatThreshold("ArmourOnGloves", 1),
		mod.StatThreshold("ArmourOnBoots", 1),
	}},
	`if equipped helmet, body armour, gloves, and boots all have evasion rating`: {tagList: []mod.Tag{
		mod.StatThreshold("EvasionOnHelmet", 1),
		mod.StatThreshold("EvasionOnBody Armour", 1),
		mod.StatThreshold("EvasionOnGloves", 1),
		mod.StatThreshold("EvasionOnBoots", 1),
	}},
	`if equipped ([a-z ]+) has an? ([a-z ]+) modifier`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ItemCondition(caps[0]).Search(caps[1])}
		},
	},
	`if both equipped ([a-z ]+) have an? ?([a-z ]+) modifiers?`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ItemCondition(strings.TrimSuffix(caps[0], "s")).Search(caps[1]).WithBothSlots()}
		},
	},
	`if both equipped left and right ([a-z ]+) have an? ?([a-z ]+) modifiers?`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ItemCondition(strings.TrimSuffix(caps[0], "s")).Search(caps[1]).WithBothSlots()}
		},
	},
	`if there are no ([a-z ]+) modifiers on equipped ([a-z ]+)`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ItemCondition(caps[1]).Search(caps[0]).Negate(true)}
		},
	},
	`if there are no ([a-z]+) modifiers on other equipped items`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ItemCondition("{SlotName}").Search(caps[0]).WithAllSlots().WithExcludeSelf().Negate(true)}
		},
	},
	`if corrupted`:          {tag: mod.ItemCondition("{SlotName}").Corrupted(true)},
	`on enemies`:            {},
	`while active`:          {},
	` on critical strike`:   {tag: mod.Condition("CriticalStrike")},
	`from critical strikes`: {tag: mod.Condition("CriticalStrike")},
	`with critical strikes`: {tag: mod.Condition("CriticalStrike")},
	//pob:src/Modules/ModParser.lua:1328
	`while affected by auras you cast`: {tag: mod.Condition("AffectedByAura")},
	`for you and nearby allies`:        {newAura: true},

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
			return modNameListType{tag: mod.Multiplier("ActiveBrand").Base(0).GlobalLimit(utils.Float(caps[0])).GlobalLimitKey("ChipAway")}
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
	`while at least (\d+) enemies are nearby`: {
		//pob:src/Modules/ModParser.lua:1775
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("NearbyEnemies").Threshold(utils.Float(caps[0]))}
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
			return modNameListType{tag: mod.Multiplier("PoisonAppliedRecently").Base(0).GlobalLimit(utils.Float(caps[0])).GlobalLimitKey("DurationPerPoisonRecently")}
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
	`per minion, up to ?a? ?m?a?x?i?m?u?m? ?o?f? (\d+)%`: {
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
	`to you and allies`:                    {newAura: true},
	`per red socket`:                       {tag: mod.Multiplier("RedSocketIn{SlotName}").Base(0)},
	`per green socket on main hand weapon`: {tag: mod.Multiplier("GreenSocketInWeapon 1").Base(0)},
	`per green socket on`:                  {tag: mod.Multiplier("GreenSocketInWeapon 1").Base(0)},
	`per red socket on main hand weapon`:   {tag: mod.Multiplier("RedSocketInWeapon 1").Base(0)},
	`per green socket`:                     {tag: mod.Multiplier("GreenSocketIn{SlotName}").Base(0)},
	`per blue socket`:                      {tag: mod.Multiplier("BlueSocketIn{SlotName}").Base(0)},
	`per white socket`:                     {tag: mod.Multiplier("WhiteSocketIn{SlotName}").Base(0)},
	`for each impale on enemy`:             {tag: mod.Multiplier("ImpaleStacks").Actor("enemy")},
	//pob:src/Modules/ModParser.lua:1454
	`per impale on enemy`:                 {tag: mod.Multiplier("ImpaleStacks").Actor("enemy")},
	`per animated weapon`:                 {tag: mod.PerStat(0, "ActiveAnimatedWeaponLimit").Actor("parent")},
	`per grasping vine`:                   {tag: mod.Multiplier("GraspingVinesCount").Base(0)},
	`per fragile regrowth`:                {tag: mod.Multiplier("FragileRegrowthCount").Base(0)},
	`per bark`:                            {tag: mod.Multiplier("BarkskinStacks").Base(0)},
	`per bark below maximum`:              {tag: mod.Multiplier("MissingBarkskinStacks").Base(0)},
	`per allocated mastery passive skill`: {tag: mod.Multiplier("AllocatedMastery").Base(0)},
	`per allocated notable passive skill`: {tag: mod.Multiplier("AllocatedNotable").Base(0)},

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
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Life").Actor("player")}
		},
	},
	`per (\d+) player maximum energy shield`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "EnergyShield").Actor("player")}
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
	`per (\d+)% cold resistance`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "ColdResist")}
		},
	},
	`per (\d+)% fire resistance`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "FireResist")}
		},
	},
	`per (\d+)% lightning resistance`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "LightningResist")}
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
	`per (\d+)% fire, cold, or lightning resistance above 75%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]),
				"FireResistOver75", "ColdResistOver75", "LightningResistOver75")}
		},
	},
	`per (\d+) devotion`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "Devotion").Actor("parent")}
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
	`per (\d+)% missing fire, cold, or lightning resistance, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.PerStat(utils.Float(caps[0]), "MissingFireResist", "MissingColdResist", "MissingLightningResist").
				GlobalLimit(utils.Float(caps[1])).GlobalLimitKey("ReplicaNebulisCold")}
		},
	},
	`per endurance, frenzy or power charge`: {tag: mod.PerStat(0, "TotalCharges")},
	`per fortification`:                     {tag: mod.PerStat(0, "FortificationStacks")},
	`per fortification above 20`:            {tag: mod.PerStat(0, "FortificationStacksOver20")},
	`per two fortification on you`:          {tag: mod.PerStat(2, "FortificationStacks").Actor("player")},
	`per totem`:                             {tag: mod.PerStat(0, "TotemsSummoned")},
	`per summoned totem`:                    {tag: mod.PerStat(0, "TotemsSummoned")},
	`for each summoned totem`:               {tag: mod.PerStat(0, "TotemsSummoned")},
	`for each time they have chained`:       {tag: mod.PerStat(0, "Chain")},
	`for each time it has chained`:          {tag: mod.PerStat(0, "Chain")},
	`for each summoned golem`:               {tag: mod.PerStat(0, "ActiveGolemLimit")},
	`for each golem you have summoned`:      {tag: mod.PerStat(0, "ActiveGolemLimit")},
	`per summoned golem`:                    {tag: mod.PerStat(0, "ActiveGolemLimit")},
	`per summoned sentinel of purity`:       {tag: mod.PerStat(0, "ActiveSentinelOfPurityLimit")},
	`per summoned void spawn`:               {tag: mod.PerStat(0, "ActiveVoidSpawnLimit")},
	`per summoned skeleton`:                 {tag: mod.PerStat(0, "ActiveSkeletonLimit")},
	`per skeleton you own`:                  {tag: mod.PerStat(0, "ActiveSkeletonLimit").Actor("parent")},
	`per summoned raging spirit`:            {tag: mod.PerStat(0, "ActiveRagingSpiritLimit")},
	`per summoned phantasm`:                 {tag: mod.PerStat(0, "ActivePhantasmLimit")},
	`for each raised zombie`:                {tag: mod.PerStat(0, "ActiveZombieLimit")},
	`per zombie you own`:                    {tag: mod.PerStat(0, "ActiveZombieLimit").Actor("parent")},
	`per raised zombie`:                     {tag: mod.PerStat(0, "ActiveZombieLimit")},
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
	`per second you've been stationary, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("StationarySeconds").Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
	`while affected by a unique abyss jewel`: {tag: mod.MultiplierThreshold("UniqueAbyssJewels").Threshold(1)},
	`while affected by a rare abyss jewel`:   {tag: mod.MultiplierThreshold("RareAbyssJewels").Threshold(1)},
	`while affected by a magic abyss jewel`:  {tag: mod.MultiplierThreshold("MagicAbyssJewels").Threshold(1)},
	`while affected by a normal abyss jewel`: {tag: mod.MultiplierThreshold("NormalAbyssJewels").Threshold(1)},
	`if (\d+) (\w+) items are equipped`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold(firstToUpper(caps[1]) + "Item").Threshold(utils.Float(caps[0]))}
		},
	},

	// Slot conditions
	`when in main hand`:     {tag: mod.SlotNumber(1)},
	`whi?l?en? in off hand`: {tag: mod.SlotNumber(2)},
	`in main hand`:          {tag: mod.InSlot(1)},
	`in off hand`:           {tag: mod.InSlot(2)},
	`w?i?t?h? main hand`:    {tagList: []mod.Tag{mod.Condition("MainHandAttack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`w?i?t?h? off ?hand`:    {tagList: []mod.Tag{mod.Condition("OffHandAttack"), mod.SkillType(string(data.SkillTypeAttack))}},
	`[fi]?[rn]?[of]?[ml]?[ i]?[hc]?[it]?[te]?[sd]? ? with this weapon`: {tagList: []mod.Tag{mod.Condition("{Hand}Attack"), mod.SkillType(string(data.SkillTypeAttack))}},
	//pob:block:src/Modules/ModParser.lua:1593-1594
	`if your (?:other|opposite) ring is a shaper item`: {tag: mod.ItemCondition("Ring {OtherSlotNum}").Shaper()},
	`if your (?:other|opposite) ring is an elder item`: {tag: mod.ItemCondition("Ring {OtherSlotNum}").Elder()},
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
	`wh[ie][ln]e? on low life`:               {tag: mod.Condition("LowLife")},
	`wh[ie][ln]e? not on low life`:           {tag: mod.Condition("LowLife").Neg(true)},
	`wh[ie][ln]e? on low mana`:               {tag: mod.Condition("LowMana")},
	`wh[ie][ln]e? not on low mana`:           {tag: mod.Condition("LowMana").Neg(true)},
	`wh[ie][ln]e? on full life`:              {tag: mod.Condition("FullLife")},
	`wh[ie][ln]e? not on full life`:          {tag: mod.Condition("FullLife").Neg(true)},
	`wh[ie][ln]e? no life is reserved`:       {tag: mod.StatThreshold("LifeReserved", 0).Upper(true)},
	`wh[ie][ln]e? no mana is reserved`:       {tag: mod.StatThreshold("ManaReserved", 0).Upper(true)},
	`wh[ie][ln]e? on full energy shield`:     {tag: mod.Condition("FullEnergyShield")},
	`wh[ie][ln]e? not on full energy shield`: {tag: mod.Condition("FullEnergyShield").Neg(true)},
	`wh[ie][ln]e? you have energy shield`:    {tag: mod.Condition("HaveEnergyShield")},
	`wh[ie][ln]e? you have no energy shield`: {tag: mod.Condition("HaveEnergyShield").Neg(true)},
	`if you have energy shield`:              {tag: mod.Condition("HaveEnergyShield")},
	`while stationary`:                       {tag: mod.Condition("Stationary")},
	`while you are stationary`:               {tag: mod.ActorCondition("player", "Stationary")},
	`while moving`:                           {tag: mod.Condition("Moving")},
	`while channelling`:                      {tag: mod.Condition("Channelling")},
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
	`if you've inflicted exposure recently`: {tag: mod.Condition("AppliedExposureRecently")},
	`while you have no power charges`:       {tag: mod.StatThreshold("PowerCharges", 0).Upper(true)},
	`while you have no frenzy charges`:      {tag: mod.StatThreshold("FrenzyCharges", 0).Upper(true)},
	`while you have no endurance charges`:   {tag: mod.StatThreshold("EnduranceCharges", 0).Upper(true)},
	`while you have a power charge`:         {tag: mod.StatThreshold("PowerCharges", 1)},
	`while you have a frenzy charge`:        {tag: mod.StatThreshold("FrenzyCharges", 1)},
	`while you have an endurance charge`:    {tag: mod.StatThreshold("EnduranceCharges", 1)},
	`while at maximum power charges`:        {tag: mod.StatThreshold("PowerCharges", 0).ThresholdStat("PowerChargesMax")},
	`while at maximum frenzy charges`:       {tag: mod.StatThreshold("FrenzyCharges", 0).ThresholdStat("FrenzyChargesMax")},
	`while on full frenzy charges`:          {tag: mod.StatThreshold("FrenzyCharges", 0).ThresholdStat("FrenzyChargesMax")},
	`while at maximum endurance charges`:    {tag: mod.StatThreshold("EnduranceCharges", 0).ThresholdStat("EnduranceChargesMax")},
	`while at minimum endurance charges`:    {tag: mod.StatThreshold("EnduranceCharges", 0).ThresholdStat("EnduranceChargesMin").Upper(true)},
	`while at minimum power charges`:        {tag: mod.StatThreshold("PowerCharges", 0).ThresholdStat("PowerChargesMin").Upper(true)},
	`while at minimum frenzy charges`:       {tag: mod.StatThreshold("FrenzyCharges", 0).ThresholdStat("FrenzyChargesMin").Upper(true)},
	`while at maximum fortification`:        {tag: mod.Condition("HaveMaximumFortification")},
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
	`while you have a totem`: {tag: mod.Condition("HaveTotem")},
	`while you have at least (\d+) nearby all[yi]e?s?`: {
		//pob:src/Modules/ModParser.lua:1584
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("NearbyAlly").Threshold(utils.Float(caps[0]))}
		},
	},
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
	`while blinded`:                               {tagList: []mod.Tag{mod.Condition("Blinded"), mod.Condition("CannotBeBlinded").Neg(true)}},
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
	`against damage over time`:                    {flags: mod.MFlagDot},
	`while there is only one nearby enemy`:        {tagList: []mod.Tag{mod.Multiplier("NearbyEnemies").Limit(1), mod.Condition("OnlyOneNearbyEnemy")}},
	`while t?h?e?r?e? ?i?s? ?a rare or unique enemy i?s? ?nearby`:                      {tag: mod.ActorCondition("enemy", "NearbyRareOrUniqueEnemy", "RareOrUnique")},
	`when you or your totems hit an enemy with a spell`:                                {tag: mod.Condition("HitSpellRecently", "TotemsHitSpellRecently")},
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
	`if you[' ]h?a?ve not warcried recently`:          {tag: mod.Condition("UsedWarcryRecently").Neg(true)},
	`for each time you[' ]h?a?ve warcried recently`:   {tag: mod.Multiplier("WarcryUsedRecently").Base(0)},
	`if you[' ]h?a?ve warcried in the past 8 seconds`: {tag: mod.Condition("UsedWarcryInPast8Seconds")},
	//pob:src/Modules/ModParser.lua:1859
	`for each second you've been affected by a warcry buff, up to a maximum of (\d+)%`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.Multiplier("AffectedByWarcryBuffDuration").Base(0).Limit(utils.Float(caps[0])).LimitTotal(true)}
		},
	},
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
	//pob:block:src/Modules/ModParser.lua:1889
	`for \d+ seconds after spending a total of (\d+) mana`: {
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
	`per enemy in close range`: {tagList: []mod.Tag{
		//pob:src/Modules/ModParser.lua:1430
		mod.Condition("AtCloseRange"), mod.Multiplier("NearbyEnemies"),
	}},
	`at close range`:                  {tag: mod.Condition("AtCloseRange")},
	`not at close range`:              {tag: mod.Condition("AtCloseRange").Neg(true)},
	`when you hit a unique enemy`:     {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	`against rare and unique enemies`: {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	//pob:block:src/Modules/ModParser.lua:1942
	`by s?l?a?i?n? rare [ao][nr]d? unique enemies`: {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	`against unique enemies`:                       {tag: mod.ActorCondition("enemy", "RareOrUnique")},
	`to enemies which have energy shield`: {
		//pob:src/Modules/ModParser.lua:1949
		tag:          mod.ActorCondition("enemy", "HaveEnergyShield"),
		keywordFlags: mod.KeywordFlagHit | mod.KeywordFlagAilment,
	},
	`against enemies with (\w+) exposure`: {
		//pob:src/Modules/ModParser.lua:2000
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.ActorCondition("enemy", "Has"+utils.Capital(caps[0])+"Exposure")}
		},
	},
	//pob:block:src/Modules/ModParser.lua:2001-2002
	`by s?l?a?i?n? ?frozen enemies`:                                             {tag: mod.ActorCondition("enemy", "Frozen")},
	`by s?l?a?i?n? ?shocked enemies`:                                            {tag: mod.ActorCondition("enemy", "Shocked")},
	`against enemies on full life`:                                              {tag: mod.ActorCondition("enemy", "FullLife")},
	`against enemies that are on full life`:                                     {tag: mod.ActorCondition("enemy", "FullLife")},
	`against enemies on low life`:                                               {tag: mod.ActorCondition("enemy", "LowLife")},
	`against enemies that are on low life`:                                      {tag: mod.ActorCondition("enemy", "LowLife")},
	`against enemies not on low life`:                                           {tag: mod.ActorCondition("enemy", "LowLife").Neg(true)},
	`against enemies that are not on low life`:                                  {tag: mod.ActorCondition("enemy", "LowLife").Neg(true)},
	`against enemies not on full life`:                                          {tag: mod.ActorCondition("enemy", "FullLife").Neg(true)},
	`against enemies that are not on full life`:                                 {tag: mod.ActorCondition("enemy", "FullLife").Neg(true)},
	`against cursed enemies`:                                                    {tag: mod.ActorCondition("enemy", "Cursed")},
	`on cursed enemies`:                                                         {tag: mod.ActorCondition("enemy", "Cursed")},
	`against excommunicated enemies`:                                            {tag: mod.ActorCondition("enemy", "Excommunicated")},
	`against stunned enemies`:                                                   {tag: mod.ActorCondition("enemy", "Stunned")},
	`per maximum number of summoned totems`:                                     {tag: mod.PerStat(0, "ActiveTotemLimit")},
	`against enemies with a higher percentage of their life remaining than you`: {tag: mod.ActorCondition("enemy", "HigherLifePercentThanPlayer")},
	`if (\d+)% of curse duration expired`: {
		fn: func(caps []string) modNameListType {
			return modNameListType{tag: mod.MultiplierThreshold("CurseExpired").Actor("enemy").Threshold(utils.Float(caps[0]))}
		},
	},
	`of cursed enemies'`:              {tag: mod.ActorCondition("enemy", "Cursed")},
	`when hitting cursed enemies`:     {tag: mod.ActorCondition("enemy", "Cursed"), keywordFlags: mod.KeywordFlagHit},
	`from cursed enemies`:             {tag: mod.ActorCondition("enemy", "Cursed")},
	`against marked enemy`:            {tag: mod.ActorCondition("enemy", "Marked")},
	`when hitting marked enemy`:       {tag: mod.ActorCondition("enemy", "Marked"), keywordFlags: mod.KeywordFlagHit},
	`from marked enemy`:               {tag: mod.ActorCondition("enemy", "Marked")},
	`against taunted enemies`:         {tag: mod.ActorCondition("enemy", "Taunted")},
	`against bleeding enemies`:        {tag: mod.ActorCondition("enemy", "Bleeding")},
	`you inflict on bleeding enemies`: {tag: mod.ActorCondition("enemy", "Bleeding")},
	`to bleeding enemies`:             {tag: mod.ActorCondition("enemy", "Bleeding")},
	`from bleeding enemies`:           {tag: mod.ActorCondition("enemy", "Bleeding")},
	`against poisoned enemies`:        {tag: mod.ActorCondition("enemy", "Poisoned")},
	`you inflict on poisoned enemies`: {tag: mod.ActorCondition("enemy", "Poisoned")},
	`to poisoned enemies`:             {tag: mod.ActorCondition("enemy", "Poisoned")},
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
	`you inflict on shocked enemies`:                             {tag: mod.ActorCondition("enemy", "Shocked")},
	`inflicted on shocked enemies`:                               {tag: mod.ActorCondition("enemy", "Shocked")},
	`you inflict on chilled enemies`:                             {tag: mod.ActorCondition("enemy", "Chilled")},
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

type gemLookups struct {
	skillIDForName map[string]string
	gemForBaseName map[string]*poe.SkillGem
}

var gemLookup = sync.OnceValue(buildGemLookup)

var gemKeywordWordPattern = regexp.MustCompile(`\S+`)

//pob:block:src/Modules/ModParser.lua:2032-2045
func buildGemLookup() *gemLookups {
	lookups := &gemLookups{
		skillIDForName: map[string]string{"power charge on critical strike": "SupportPowerChargeOnCritical"},
		gemForBaseName: make(map[string]*poe.SkillGem),
	}

	for _, effect := range poe.GrantedEffects {
		gem := raw2.SkillGemForGrantedEffectID(effect.ID)
		if gem == nil {
			continue
		}

		base := gem.GetBaseItemType()
		if base == nil || strings.Contains(base.ID, "Royale") {
			continue
		}

		lookups.gemForBaseName[strings.ToLower(base.Name)] = gem
	}

	ids := make([]string, 0, len(raw2.PoBSkills))
	for id := range raw2.PoBSkills {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		skill := raw2.PoBSkills[id]
		if skill.Hidden && !skill.FromItem && !skill.FromTree {
			continue
		}
		lookups.skillIDForName[strings.ToLower(skill.Name)] = id
	}

	return lookups
}

//pob:block:src/Modules/ModParser.lua:1615-1617
func extraSkillIDForName(name string) string {
	return gemLookup().skillIDForName[strings.ToLower(name)]
}

//pob:block:src/Modules/ModParser.lua:2454-2468
//pob:block:src/Modules/ModParser.lua:2075-2083
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

	supportGem, ok := lookups.gemForBaseName[strings.ToLower(raw2.PoBSkills[skillID].Name)+" support"]
	if !ok {
		return []extraSupportEntry{{skillID: skillID}}
	}

	effects := raw2.GrantedEffectsForGem(supportGem)
	entries := make([]extraSupportEntry, 0, len(effects))
	for i, effect := range effects {
		entries = append(entries, extraSupportEntry{
			skillID: effect.ID,
			asSkill: i > 0 && !raw2.PoBSkills[effect.ID].Support,
		})
	}
	if len(entries) == 0 {
		return []extraSupportEntry{{skillID: skillID}}
	}
	return entries
}

func extraSupportMods(entries []extraSupportEntry, level int, slotName string) []mod.Mod {
	mods := make([]mod.Mod, 0, len(entries))
	for _, entry := range entries {
		if entry.asSkill {
			mods = append(mods, MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: entry.skillID, Level: level}))
			continue
		}
		mods = append(mods, MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: entry.skillID, Level: level}).Tag(mod.SocketedIn(slotName)))
	}
	return mods
}

//pob:src/Modules/ModParser.lua:2046-2053
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

//pob:src/Modules/ModParser.lua:2054-2063
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
//
//pob:src/Modules/ModParser.lua:2014-2030
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

//pob:block:src/Modules/ModParser.lua:2062
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

//pob:block:src/Modules/ModParser.lua:2067-2069
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
	Chance         float64
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

//pob:src/Modules/ModParser.lua:2106-2117
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
		MOD("ExplodeMod", "LIST", ExplodeMod{Type: firstToUpper(typ), Chance: chance / 100, Amount: amountNumber, KeyOfScaledMod: "chance"}).Tag(tags...),
		FLAG("CanExplode"),
	}, ""
}

type SpecialFuncType func(num float64, captures []string) ([]mod.Mod, string)

// List of special modifiers
var specialModListCompiled map[string]CompiledList[interface{}]
var specialModList = map[string]interface{}{
	//pob:block:src/Modules/ModParser.lua:2121-2185
	`enemies you kill have a (\d+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2])
	},
	`enemies you kill have ?a? ?(\d+)% chance to explode, dealing (\d+)% of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2])
	},
	`enemies you or your totems kill have (\d+)% chance to explode, dealing (\d+)% of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2])
	},
	`enemies you kill while using pride have (\d+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2], mod.Condition("AffectedByPride"))
	},
	`enemies you kill during effect have a (\d+)% chance to explode, dealing a (.+) of their maximum life as damage of a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], "randomElement", mod.Condition("UsingFlask"))
	},
	`enemies you kill while affected by glorious madness have a (\d+)% chance to explode, dealing a (.+) of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2], mod.Condition("AffectedByGloriousMadness"))
	},
	`enemies killed with attack hits have a (\d+)% chance to explode, dealing a (.+) of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2])
	},
	`enemies killed with wand hits have a (\d+)% chance to explode, dealing a (.+) of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2], mod.Condition("UsingWand"))
	},
	`cursed enemies you or your minions kill have a (\d+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(num, captures[1], captures[2], mod.ActorCondition("enemy", "Cursed"))
	},
	`enemies you kill explode, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1])
	},
	`enemies killed explode dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1])
	},
	`enemies on fungal ground you kill explode, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.ActorCondition("enemy", "OnFungalGround"))
	},
	`enemies killed with attack or spell hits explode, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1])
	},
	`shocked enemies you kill explode, dealing (\d+)% of their life as (.+) damage which cannot shock`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.ActorCondition("enemy", "Shocked"))
	},
	`ignited enemies you kill explode, dealing (\d+)% of their life as (.+) damage which cannot ignite`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.ActorCondition("enemy", "Ignited"))
	},
	`bleeding enemies you kill explode, dealing (\d+)% of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.ActorCondition("enemy", "Bleeding"))
	},
	`burning enemies you kill have a (\d+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.ActorCondition("enemy", "Burning"))
	},
	`enemies killed near corpses affected by your curses explode, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.MultiplierThreshold("NearbyCorpse").Threshold(1), mod.ActorCondition("enemy", "Cursed"))
	},
	`enemies taunted by your warcries explode on death, dealing (\d+)% of their maximum life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1], mod.ActorCondition("enemy", "Taunted"), mod.Condition("UsedWarcryRecently"))
	},
	`totems explode on death, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1])
	},
	`nearby corpses explode when you warcry, dealing (\d+)% of their life as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return explodeFunc(100, captures[0], captures[1])
	},
	`arcane might`:                    []mod.Mod{mod.NewFlag("SpellDamageAppliesToAttacks", true)},
	`attacks against you always hit`:  []mod.Mod{mod.NewFlag("CannotEvade", true)},
	`cannot be knocked back`:          []mod.Mod{mod.NewFlag("KnockbackImmune", true)},
	`cannot block attack damage`:      []mod.Mod{mod.NewFlag("CannotBlockAttacks", true)},
	`cannot block spell damage`:       []mod.Mod{mod.NewFlag("CannotBlockSpells", true)},
	`cannot gain life`:                []mod.Mod{mod.NewFlag("CannotGainLife", true)},
	`cannot gain mana`:                []mod.Mod{mod.NewFlag("CannotGainMana", true)},
	`cannot have non-abyssal sockets`: []mod.Mod{mod.NewFlag("NoSockets", true)},
	//pob:block:src/Modules/ModParser.lua:3088-3093
	`socketed [a-z]+ abyssal jewels will be consumed`:    []mod.Mod{},
	"one modifier from consumed jewels will be retained": []mod.Mod{},
	`cannot inflict multiple poisons in the same hit`:    []mod.Mod{mod.NewFlag("CannotMultiplePoison", true)},
	`critical strikes do not deal extra damage`:          []mod.Mod{mod.NewFlag("NoCritMultiplier", true)},
	`dexterity provides no bonus to evasion rating`:      []mod.Mod{mod.NewFlag("NoDexBonusToEvasion", true)},
	`immun[ei]t?y? to chill`:                             []mod.Mod{mod.NewFlag("ChillImmune", true)},
	`immun[ei]t?y? to freeze`:                            []mod.Mod{mod.NewFlag("FreezeImmune", true)},
	`immun[ei]t?y? to ignite`:                            []mod.Mod{mod.NewFlag("IgniteImmune", true)},
	//pob:block:src/Modules/ModParser.lua:4836-4838
	"immune to ignite while affected by purity of fire":                          []mod.Mod{FLAG("IgniteImmune").Tag(mod.Condition("AffectedByPurityofFire"))},
	"immune to freeze while affected by purity of ice":                           []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("AffectedByPurityofIce"))},
	"immune to shock while affected by purity of lightning":                      []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("AffectedByPurityofLightning"))},
	`immun[ei]t?y? to shock`:                                                     []mod.Mod{mod.NewFlag("ShockImmune", true)},
	`increase to cast speed from arcane surge also applies to movement speed`:    []mod.Mod{mod.NewFlag("ArcaneSurgeCastSpeedToMovementSpeed", true)},
	`intelligence provides no bonus to energy shield`:                            []mod.Mod{mod.NewFlag("NoIntBonusToES", true)},
	`intelligence provides no inherent bonus to maximum mana`:                    []mod.Mod{mod.NewFlag("NoIntBonusToMana", true)},
	`life that would be lost by taking damage is instead reserved`:               []mod.Mod{mod.NewFlag("DamageInsteadReservesLife", true)},
	`strength provides no inherent bonus to maximum life`:                        []mod.Mod{mod.NewFlag("NoStrBonusToLife", true)},
	`y?o?u?r? ?chance to suppress spell damage is unlucky`:                       []mod.Mod{mod.NewFlag("SpellSuppressionChanceIsUnlucky", true)},
	`you are immune to curses`:                                                   []mod.Mod{mod.NewFlag("CurseImmune", true)},
	`your physical damage can ignite during effect`:                              []mod.Mod{mod.NewFlag("PhysicalCanIgnite", true)},
	`debilitate enemies for (\d+) seconds? when you suppress their spell damage`: []mod.Mod{MOD("DebilitateChance", "BASE", 100)},
	`debilitate nearby enemies for (\d+) seconds? when f?l?a?s?k? ?effect ends`:  []mod.Mod{MOD("DebilitateChance", "BASE", 100)},
	`evasion rating is doubled against melee attacks`:                            []mod.Mod{MOD("MeleeEvasion", "MORE", 100)},
	`evasion rating is doubled against projectile attacks`:                       []mod.Mod{MOD("ProjectileEvasion", "MORE", 100)},
	`inherent effects from having rage are doubled`:                              []mod.Mod{MOD("RageEffect", "MORE", 100)},
	`maximum rage is halved`:                                                     []mod.Mod{MOD("MaximumRage", "MORE", -50)},
	`nearby enemies are covered in ash`:                                          []mod.Mod{MOD("CoveredInAshEffect", "BASE", 20)},
	//pob:src/Modules/ModParser.lua:4293
	`your warcries cover enemies in ash for (\d+) seconds`: []mod.Mod{MOD("CoveredInAshEffect", "BASE", 20).Tag(mod.Condition("UsedWarcryRecently"))},
	//pob:src/Modules/ModParser.lua:4298
	`debuffs on you expire (\d+)% faster while affected by haste`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SelfDebuffExpirationRate", "BASE", num).Tag(mod.Condition("AffectedByHaste"))}, ""
	},
	//pob:src/Modules/ModParser.lua:4719
	`(\d+)% of damage taken while affected by clarity recouped as mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaRecoup", "BASE", num).Tag(mod.Condition("AffectedByClarity"))}, ""
	},
	//pob:src/Modules/ModParser.lua:2647
	`([-+]\d+) to maximum number of sacred wisps`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SacredWispsMaxCount", "BASE", num)}, ""
	},
	//pob:src/Modules/ModParser.lua:4309
	`maximum (\d+) eaten souls`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SoulEaterMax", "OVERRIDE", num)}, ""
	},
	//pob:src/Modules/ModParser.lua:4310
	`([+-]\d+) to maximum number of eaten souls`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SoulEaterMax", "BASE", num)}, ""
	},
	//pob:src/Modules/ModParser.lua:3144
	`instant recovery when on low life`: []mod.Mod{
		MOD("FlaskLowLifeInstantRecovery", "BASE", 100),
		MOD("Dummy", "DUMMY", 1).Tag(mod.Condition("LowLife")),
	},
	`retaliation skills debilitate enemies for (\d+) seconds on hit`: []mod.Mod{MOD("DebilitateChance", "BASE", 100)},
	`scorch enemies in close range when you block`:                   []mod.Mod{MOD("EnemyScorchChance", "BASE", 100)},
	`targets are unaffected by your hexes`:                           []mod.Mod{MOD("CurseEffect", "MORE", -100).Tag(mod.SkillType(string(data.SkillTypeHex)))},
	//pob:block:src/Modules/ModParser.lua:4924-4941
	`unaffected by bleeding`:                               []mod.Mod{MOD("SelfBleedEffect", "MORE", -100)},
	`you are unaffected by bleeding while leeching`:        []mod.Mod{MOD("SelfBleedEffect", "MORE", -100).Tag(mod.Condition("Leeching"))},
	`unaffected by bleeding while affected by malevolence`: []mod.Mod{MOD("SelfBleedEffect", "MORE", -100).Tag(mod.Condition("AffectedByMalevolence")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	`unaffected by poison`:                                 []mod.Mod{MOD("SelfPoisonEffect", "MORE", -100)},
	`unaffected by poison while affected by malevolence`:   []mod.Mod{MOD("SelfPoisonEffect", "MORE", -100).Tag(mod.Condition("AffectedByMalevolence")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	`unaffected by ignite`:                                 []mod.Mod{MOD("SelfIgniteEffect", "MORE", -100)},
	`unaffected by chill`:                                  []mod.Mod{MOD("SelfChillEffect", "MORE", -100)},
	`unaffected by chill while leeching mana`:              []mod.Mod{MOD("SelfChillEffect", "MORE", -100).Tag(mod.Condition("LeechingMana"))},
	`unaffected by chill while channelling`:                []mod.Mod{MOD("SelfChillEffect", "MORE", -100).Tag(mod.Condition("Channelling"))},
	`unaffected by freeze`:                                 []mod.Mod{MOD("SelfFreezeEffect", "MORE", -100)},
	`unaffected by shock`:                                  []mod.Mod{MOD("SelfShockEffect", "MORE", -100)},
	`unaffected by shock while leeching energy shield`:     []mod.Mod{MOD("SelfShockEffect", "MORE", -100).Tag(mod.Condition("LeechingEnergyShield"))},
	`unaffected by shock while channelling`:                []mod.Mod{MOD("SelfShockEffect", "MORE", -100).Tag(mod.Condition("Channelling"))},
	`unaffected by (.+) if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Self"+firstToUpper(captures[0])+"Effect", "MORE", -100).
				Tag(mod.MultiplierThreshold(firstToUpper(captures[2]) + "Item").Threshold(utils.Float(captures[1]))),
		}, ""
	},
	`unaffected by scorch`:  []mod.Mod{MOD("SelfScorchEffect", "MORE", -100)},
	`unaffected by brittle`: []mod.Mod{MOD("SelfBrittleEffect", "MORE", -100)},
	`unaffected by sap`:     []mod.Mod{MOD("SelfSapEffect", "MORE", -100)},
	`unaffected by damaging ailments`: []mod.Mod{
		MOD("SelfBleedEffect", "MORE", -100),
		MOD("SelfIgniteEffect", "MORE", -100),
		MOD("SelfPoisonEffect", "MORE", -100),
	},
	`warcries debilitate enemies for (\d+) seconds?`:  []mod.Mod{MOD("DebilitateChance", "BASE", 100)},
	`warcries exert twice as many attacks`:            []mod.Mod{MOD("ExtraExertedAttacks", "MORE", 100)},
	`you can apply an additional curse during effect`: []mod.Mod{MOD("EnemyCurseLimit", "BASE", 1)},
	`your hits are always critical strikes`:           []mod.Mod{MOD("CritChance", "OVERRIDE", 100)},
	//pob:src/Modules/ModParser.lua:3553
	`all hits are critical strikes while holding a fishing rod`: []mod.Mod{MOD("CritChance", "OVERRIDE", 100).Tag(mod.Condition("UsingFishing"))},
	`when hit during effect, (\d+)% of life loss from damage taken occurs over 4 seconds instead`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeLossPrevented", "BASE", num).Tag(mod.Condition("UsingFlask"))}, ""
	},
	`modifiers to chance to suppress spell damage also apply to chance to avoid elemental ailments at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SpellSuppressionAppliesToAilmentAvoidancePercent", "BASE", num),
			mod.NewFlag("SpellSuppressionAppliesToAilmentAvoidance", true),
		}, ""
	},
	`modifiers to chance to suppress spell damage also apply to chance to defend with (\d+)% of armour at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SpellSuppressionAppliesToChanceToDefendWithArmourPercentArmour", "MAX", num),
			MOD("SpellSuppressionAppliesToChanceToDefendWithArmourPercent", "MAX", utils.Float(captures[1])),
			mod.NewFlag("SpellSuppressionAppliesToChanceToDefendWithArmour", true),
		}, ""
	},
	`debuffs on you expire (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SelfDebuffExpirationRate", "BASE", num)}, ""
	},
	//pob:block:src/Modules/ModParser.lua:3252-3273
	`take (\d+) (.+) damage when you ignite an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EyeOfInnocenceSelfDamage", "LIST", mod.SelfDamage{BaseDamage: num, DamageType: captures[1]})}, ""
	},
	`(\d+) (.+) damage taken on minion death`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HeartboundLoopSelfDamage", "LIST", mod.SelfDamage{BaseDamage: num, DamageType: captures[1]})}, ""
	},
	`take (\d+) (.+) damage when herald of thunder hits an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("StormSecretSelfDamage", "LIST", mod.SelfDamage{BaseDamage: num, DamageType: captures[1]})}, ""
	},
	`take (\d+) (.+) damage when you use a skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnmitysEmbraceSelfDamage", "LIST", mod.SelfDamage{BaseDamage: num, DamageType: captures[1]})}, ""
	},
	`your skills deal you (\d+)% of mana cost as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ScoldsBridleSelfDamage", "LIST", mod.SelfDamage{DmgMult: num, DamageType: captures[1]})}, ""
	},
	`your skills deal you (\d+)% of mana spent on upfront skill mana costs as (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ScoldsBridleSelfDamage", "LIST", mod.SelfDamage{DmgMult: num, DamageType: captures[1]})}, ""
	},
	`when you attack, take (\d+)% of life as (.+) damage for each warcry exerting the attack`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EchoesOfCreationSelfDamage", "LIST", mod.SelfDamage{DmgMult: num, DamageType: captures[1]})}, ""
	},
	`debuffs on you expire (\d+)% slower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SelfDebuffExpirationRate", "BASE", -num)}, ""
	},
	//pob:src/Modules/ModParser.lua:3935
	`poisons on you expire (\d+)% slower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SelfPoisonDebuffExpirationRate", "BASE", -num)}, ""
	},
	`(\d+)% increased bonuses gained from equipped (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EffectOfBonusesFrom"+firstToUpper(captures[1]), "INC", num)}, ""
	},
	`(\d+)% reduced bonuses gained from equipped (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EffectOfBonusesFrom"+firstToUpper(captures[1]), "INC", -num)}, ""
	},
	`regenerate (\d+) life per second for each (\d+)% uncapped fire resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeRegen", "BASE", num).Tag(mod.PerStat(1/utils.Float(captures[1]), "FireResistTotal")),
		}, ""
	},
	`hits have (\d+)% chance to ignore enemy physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", num)}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5632-5637
	`(\d+)% chance for hits to ignore enemy physical damage reduction while you have sacrificial zeal`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", num)}, ""
	},
	`hits have (\d+)% chance to ignore enemy physical damage reduction while you have sacrificial zeal`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", num)}, ""
	},

	//pob:block:src/Modules/ModParser.lua:2188-2191
	`(\d+) rage regenerated for every (\d+) mana regeneration per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("RageRegen", "BASE", num).Tag(mod.PerStat(utils.Float(captures[1]), "ManaRegen")),
			mod.NewFlag("Condition:CanGainRage", true),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2192
	`(\w+) recovery from regeneration is not applied`: func(_ float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("UnaffectedBy"+firstToUpper(captures[0])+"Regen", true)}, ""
	},
	`(\d+)% less damage taken for every (\d+)% life recovery per second from leech`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeMore, -num).Tag(mod.PerStat(utils.Float(captures[1]), "MaxLifeLeechRatePercent"))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:2196-2199
	`(\w+) recovery from non-instant leech is not applied`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("UnaffectedByNonInstant"+firstToUpper(captures[0])+"Leech", true)}, ""
	},
	`(\d+)% additional physical damage reduction for every (\d+)% life recovery per second from leech`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalDamageReduction", "BASE", num).
				Tag(mod.PerStat(utils.Float(captures[1]), "MaxLifeLeechRatePercent"), mod.Condition("Leeching")),
		}, ""
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
	//pob:src/Modules/ModParser.lua:3640
	`non-critical strikes cannot inflict ailments`:                                                         []mod.Mod{mod.NewFlag("AilmentsOnlyFromCrit", true)},
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
		return []mod.Mod{mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("BlindEffect", mod.TypeIncrease, num)})}, ""
	},
	//pob:block:src/Modules/ModParser.lua:2239
	`(\d+)% increased effect of blind from melee weapons`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("BlindEffect", mod.TypeIncrease, num)})}, ""
	},
	`(\d+)% increased effect of non-damaging ailments inflicted by summoned skitterbots`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkitterbotAilmentEffect", "INC", num)}, ""
	},
	`gain ward instead of (\d+)% of armour and evasion rating from equipped body armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFlag("ConvertBodyArmourArmourEvasionToWard", true),
			mod.NewFloat("BodyArmourArmourEvasionToWardPercent", mod.TypeBase, num),
		}, ""
	},
	`\+(\d+)% chance to block spell damage for each (\d+)% overcapped chance to block attack damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SpellBlockChance", mod.TypeBase, num).Tag(mod.PerStat(utils.Float(captures[1]), "BlockChanceOverCap"))}, ""
	},
	`maximum life becomes 1, immune to chaos damage`: []mod.Mod{
		mod.NewFlag("ChaosInoculation", true),
		mod.NewFloat("ChaosDamageTaken", mod.TypeMore, -100),
	},
	`life regeneration is applied to energy shield instead`: []mod.Mod{mod.NewFlag("ZealotsOath", true)},
	`life leeched per second is doubled`:                    []mod.Mod{mod.NewFloat("LifeLeechRate", mod.TypeMore, 100)},
	`life regeneration has no effect`:                       []mod.Mod{mod.NewFlag("NoLifeRegen", true)},
	`energy shield recharge instead applies to life`:        []mod.Mod{mod.NewFlag("EnergyShieldRechargeAppliesToLife", true)},
	`(cannot deal|deal no) non\-(\w+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		if strings.EqualFold(captures[1], "elemental") {
			if !strings.EqualFold(captures[0], "deal no") {
				return nil, ""
			}
			//pob:src/Modules/ModParser.lua:5254
			return []mod.Mod{mod.NewFlag("DealNoPhysical", true), mod.NewFlag("DealNoChaos", true)}, ""
		}
		want := data.DamageType(firstToUpper(captures[1]))
		known := false
		for _, dt := range data.DamageType("").Values() {
			if dt == want {
				known = true
				break
			}
		}
		if !known {
			return nil, ""
		}
		out := make([]mod.Mod, 0, 4)
		for _, dt := range data.DamageType("").Values() {
			if dt != want {
				out = append(out, mod.NewFlag("DealNo"+string(dt), true))
			}
		}
		return out, ""
	},
	`enemies permanently take (\d+)% increased damage for each second they've ever been frozen by you, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num).
			Tag(mod.Condition("FrozenByYou")).
			Tag(mod.Multiplier("FrozenByYouSeconds").Limit(utils.Float(captures[1]) / num))})}, ""
	},
	`enemies permanently take (\d+)% increased damage for each second they've ever been chilled by you, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num).
			Tag(mod.Condition("ChilledByYou")).
			Tag(mod.Multiplier("ChilledByYouSeconds").Limit(utils.Float(captures[1]) / num))})}, ""
	},
	`herald skills and minions from herald skills deal (\d+)% more damage for every (\d+)% of maximum life those skills reserve`: func(num float64, captures []string) ([]mod.Mod, string) {
		per := utils.Float(captures[1])
		return []mod.Mod{
			mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeMore, num).
				Tag(mod.PerStat(per, "LifeReservedPercent").Actor("parent"))}).
				Tag(mod.SkillType(string(data.SkillTypeHerald))),
			mod.NewFloat("Damage", mod.TypeMore, num).
				Tag(mod.PerStat(per, "LifeReservedPercent")).
				Tag(mod.SkillType(string(data.SkillTypeHerald))),
		}, ""
	},
	`arcane surge a?l?s?o? ?grants (\d+)% more spell damage to you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArcaneSurgeDamage", "MAX", num)}, ""
	},
	`enemies chilled by your hits have damage taken increased by chill effect`:      []mod.Mod{mod.NewFlag("ChillEffectIncDamageTaken", true)},
	`enemies chilled by your hits have cold damage taken increased by chill effect`: []mod.Mod{mod.NewFlag("ChillEffectIncColdDamageTaken", true)},
	`enemies in your chilling areas have cold damage taken increased by chill effect`: []mod.Mod{
		mod.NewFlag("ChillingAreaIncColdDamageTaken", true).Tag(mod.ActorCondition("enemy", "InChillingArea")),
	},
	`enemies chilled by your hits lessen their damage dealt by half of chill effect`: []mod.Mod{mod.NewFlag("ChillEffectLessDamageDealt", true)},
	`minions have unholy might`: []mod.Mod{
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFlag("Condition:UnholyMight", true)}),
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFlag("Condition:CanWither", true)}),
	},
	`(\d+)% of physical, cold and lightning damage converted to fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("PhysicalDamageConvertToFire", mod.TypeBase, num),
			mod.NewFloat("LightningDamageConvertToFire", mod.TypeBase, num),
			mod.NewFloat("ColdDamageConvertToFire", mod.TypeBase, num),
		}, ""
	},
	`removes all mana\. spend life instead of mana for skills`: []mod.Mod{
		mod.NewFloat("Mana", mod.TypeMore, -100),
		mod.NewFlag("CostLifeInsteadOfMana", true),
	},
	`removes all mana`: []mod.Mod{mod.NewFloat("Mana", mod.TypeMore, -100)},
	//pob:block:src/Modules/ModParser.lua:2259-2263
	"all elemental damage converted to chaos damage": []mod.Mod{
		mod.NewFloat("ColdDamageConvertToChaos", mod.TypeBase, 100),
		mod.NewFloat("FireDamageConvertToChaos", mod.TypeBase, 100),
		mod.NewFloat("LightningDamageConvertToChaos", mod.TypeBase, 100),
	},
	//pob:block:src/Modules/ModParser.lua:2353-2358
	"projectiles cannot continue after colliding with targets": []mod.Mod{
		mod.NewFlag("CannotPierce", true).Flag(mod.MFlagProjectile),
		mod.NewFlag("CannotChain", true).Flag(mod.MFlagProjectile),
		mod.NewFlag("CannotFork", true).Flag(mod.MFlagProjectile),
		mod.NewFlag("CannotSplit", true).Flag(mod.MFlagProjectile),
	},
	`removes all energy shield`:           []mod.Mod{mod.NewFloat("EnergyShield", mod.TypeMore, -100)},
	`skills cost life instead of mana`:    []mod.Mod{mod.NewFlag("CostLifeInsteadOfMana", true)},
	`skills reserve life instead of mana`: []mod.Mod{mod.NewFlag("BloodMagicReserved", true)},
	//pob:block:src/Modules/ModParser.lua:2269-2271
	`your skills that throw mines reserve life instead of mana`: []mod.Mod{
		mod.NewFlag("BloodMagicReserved", true).KeywordFlag(mod.KeywordFlagMine),
	},
	`curse skills cost life instead of mana`: []mod.Mod{
		mod.NewFlag("CostLifeInsteadOfMana", true).KeywordFlag(mod.KeywordFlagCurse),
	},
	`curse aura skills reserve life instead of mana`: []mod.Mod{
		mod.NewFlag("BloodMagicReserved", true).KeywordFlag(mod.KeywordFlagCurse).
			Tag(mod.SkillType(string(data.SkillTypeAura))),
	},
	`spend life instead of mana for effects of skills`: []mod.Mod{},
	`skills cost \+(\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("RageCostBase", mod.TypeBase, num)}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5623-5625
	`skills cost life instead of (\d+)% of mana cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("HybridManaAndLifeCost_Life", mod.TypeBase, num)}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5262-5264
	`attack skills cost life instead of (\d+)% of mana cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("HybridManaAndLifeCost_Life", mod.TypeBase, num).Flag(mod.MFlagAttack)}, ""
	},
	//pob:block:src/Modules/ModParser.lua:2285
	`you cannot gain rage during soul gain prevention`: []mod.Mod{
		MOD("RageRegen", "MORE", -100).Tag(mod.Condition("SoulGainPrevention")),
	},
	`hits that deal elemental damage remove exposure to those elements and inflict exposure to other elements exposure inflicted this way applies (\-\d+)% to resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFlag("ElementalEquilibrium", true),
			mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("FireExposure", mod.TypeBase, num).Tag(mod.Condition("HitByColdDamage", "HitByLightningDamage"))}),
			mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("ColdExposure", mod.TypeBase, num).Tag(mod.Condition("HitByFireDamage", "HitByLightningDamage"))}),
			mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("LightningExposure", mod.TypeBase, num).Tag(mod.Condition("HitByFireDamage", "HitByColdDamage"))}),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:3581-3589
	`(\d+)% increased elemental damage with hits and ailments for each type of elemental ailment on enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		ailments := []string{"Frozen", "Chilled", "Ignited", "Shocked", "Scorched", "Brittle", "Sapped"}
		out := make([]mod.Mod, 0, len(ailments))
		for _, ailment := range ailments {
			out = append(out, MOD("ElementalDamage", mod.TypeIncrease, num).
				KeywordFlag(mod.KeywordFlagHit|mod.KeywordFlagAilment).
				Tag(mod.ActorCondition("enemy", ailment)))
		}
		return out, ""
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
	//pob:src/Modules/ModParser.lua:2314
	`damage over time multiplier for ailments is equal to critical strike multiplier`: []mod.Mod{mod.NewFlag("DotMultiplierIsCritMultiplier", true)},
	`your bleeding does not deal extra damage while the enemy is moving`:              []mod.Mod{mod.NewFlag("Condition:NoExtraBleedDamageToMovingEnemy", true)},
	`you can inflict bleeding on an enemy up to (\d+) times?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("BleedStacksMax", mod.TypeOverride, num),
			mod.NewFlag("Condition:HaveCrimsonDance", true),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2326-2333
	`your minions spread caustic ground on death, dealing (\d+)% of their maximum life as chaos damage per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "SiegebreakerCausticGround"}),
			mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Multiplier:SiegebreakerCausticGroundPercent", mod.TypeBase, num)}),
		}, ""
	},
	`your minions spread burning ground on death, dealing (\d+)% of their maximum life as fire damage per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewList("ExtraMinionSkill", mod.ExtraMinionSkill{SkillID: "ReplicaSiegebreakerBurningGround"}),
			mod.NewList("MinionModifier", mod.MinionModifier{Mod: mod.NewFloat("Multiplier:SiegebreakerBurningGroundPercent", mod.TypeBase, num)}),
		}, ""
	},
	`you can have an additional brand attached to an enemy`: []mod.Mod{mod.NewFloat("BrandsAttachedLimit", mod.TypeBase, 1)},
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
	`ignore attribute requirements`: []mod.Mod{mod.NewFlag("IgnoreAttributeRequirements", true)},
	//pob:src/Modules/ModParser.lua:2364-2365
	`ignore attribute requirements of gems socketed in blue sockets`: []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("all"), Key: "req", Value: 0}).Tag(mod.SocketedIn("{SlotName}").SocketColor("B"))},
	`ignore attribute requirements of socketed gems`:                 []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("all"), Key: "req", Value: 0}).Tag(mod.SocketedIn("{SlotName}"))},
	`y?o?u?r? ?chance to suppress spell damage is lucky`:             []mod.Mod{mod.NewFlag("SpellSuppressionChanceIsLucky", true)},
	`corrupted blood cannot be inflicted on you`:                     []mod.Mod{mod.NewFlag("CorruptedBloodImmune", true)},
	//pob:block:src/Modules/ModParser.lua:2490-2492
	`you cannot be hindered`:                           []mod.Mod{mod.NewFlag("HinderImmune", true)},
	`you cannot be maimed`:                             []mod.Mod{mod.NewFlag("MaimImmune", true)},
	`you cannot be impaled`:                            []mod.Mod{mod.NewFlag("ImpaleImmune", true)},
	`recoup effects instead occur over 3 seconds`:      []mod.Mod{mod.NewFlag("3SecondRecoup", true)},
	`life recoup effects instead occur over 3 seconds`: []mod.Mod{mod.NewFlag("3SecondLifeRecoup", true)},
	`recoup energy shield instead of life`:             []mod.Mod{mod.NewFlag("EnergyShieldRecoupInsteadOfLife", true)},
	`(\w+) resistance cannot be penetrated`: func(_ float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("EnemyCannotPen"+titleCaseAfterNonWord(strings.ToLower(captures[0]))+"Resistance", true)}, ""
	},
	`damage taken recouped as (\w+) is also recouped as energy shield`: func(_ float64, captures []string) ([]mod.Mod, string) {
		resource := titleCaseAfterNonWord(strings.ToLower(captures[0]))
		return []mod.Mod{mod.NewFlag("Add"+resource+"RecoupToEnergyShieldRecoup", true)}, ""
	},
	`([\d.]+)% of physical damage prevented from hits in the past (\d+) seconds is regenerated as life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		duration := utils.Float(captures[1])
		return []mod.Mod{
			mod.NewFloat("PhysicalDamageMitigatedLifePseudoRecoup", mod.TypeBase, num*duration),
			mod.NewFloat("PhysicalDamageMitigatedLifePseudoRecoupDuration", mod.TypeBase, duration),
		}, ""
	},
	`([\d.]+)% of physical damage prevented from hits recently is regenerated as energy shield per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("PhysicalDamageMitigatedEnergyShieldPseudoRecoup", mod.TypeBase, num*4)}, ""
	},
	`gain no (.+) from equipped (.+)`: func(_ float64, captures []string) ([]mod.Mod, string) {
		slot := strings.ToLower(captures[1])
		if slot == "shield" {
			slot = "Weapon 2"
		}
		stat := strings.ReplaceAll(titleCaseAfterNonWord(strings.ToLower(captures[0])), " ", "")
		return []mod.Mod{mod.NewFlag("GainNo"+stat+"From"+titleCaseAfterNonWord(slot), true)}, ""
	},
	`gain no inherent bonuses from attributes`:   []mod.Mod{mod.NewFlag("NoAttributeBonuses", true)},
	`gain no inherent bonuses from strength`:     []mod.Mod{mod.NewFlag("NoStrengthAttributeBonuses", true)},
	`gain no inherent bonuses from dexterity`:    []mod.Mod{mod.NewFlag("NoDexterityAttributeBonuses", true)},
	`gain no inherent bonuses from intelligence`: []mod.Mod{mod.NewFlag("NoIntelligenceAttributeBonuses", true)},
	//pob:src/Modules/ModParser.lua:2377-2383
	`all damage taken bypasses energy shield`: []mod.Mod{
		mod.NewFloat("PhysicalEnergyShieldBypass", mod.TypeOverride, 100),
		mod.NewFloat("LightningEnergyShieldBypass", mod.TypeOverride, 100),
		mod.NewFloat("ColdEnergyShieldBypass", mod.TypeOverride, 100),
		mod.NewFloat("FireEnergyShieldBypass", mod.TypeOverride, 100),
		mod.NewFloat("ChaosEnergyShieldBypass", mod.TypeOverride, 100),
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
	`ward has a (\d+)% chance to not break`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("WardAvoidBreakChance", mod.TypeBase, num)}, ""
	},
	`damage taken bypasses unbroken ward if the hit deals less damage than (\d+)% of ward`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("WardBypassBelowPercent", mod.TypeBase, num)}, ""
	},
	`maximum energy shield is (\d+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("EnergyShield", mod.TypeOverride, num)}, ""
	},
	`while not on full life, sacrifice ([\d\.]+)% of mana per second to recover that much life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("ManaDegenPercent", mod.TypeBase, num).Tag(mod.Condition("FullLife").Neg(true)),
			mod.NewFloat("LifeRecovery", mod.TypeBase, 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.Condition("FullLife").Neg(true)),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2401
	`you are blind`: []mod.Mod{FLAG("Condition:Blinded").Tag(mod.Condition("CannotBeBlinded").Neg(true))},
	`armour applies to fire, cold and lightning damage taken from hits instead of physical damage`: []mod.Mod{
		mod.NewFloat("ArmourAppliesToFireDamageTaken", mod.TypeBase, 100),
		mod.NewFloat("ArmourAppliesToColdDamageTaken", mod.TypeBase, 100),
		mod.NewFloat("ArmourAppliesToLightningDamageTaken", mod.TypeBase, 100),
		mod.NewFlag("ArmourDoesNotApplyToPhysicalDamageTaken", true),
	},
	`(\d+)% of armour applies to fire, cold and lightning damage taken from hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("ArmourAppliesToFireDamageTaken", mod.TypeBase, num),
			mod.NewFloat("ArmourAppliesToColdDamageTaken", mod.TypeBase, num),
			mod.NewFloat("ArmourAppliesToLightningDamageTaken", mod.TypeBase, num),
		}, ""
	},
	`maximum damage reduction for any damage type is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("DamageReductionMax", mod.TypeOverride, num)}, ""
	},
	`gain (\d+)% chance to block from equipped shield instead of the shield's value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ReplaceShieldBlock", mod.TypeOverride, num).Tag(mod.Condition("UsingShield"))}, ""
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
	`shepherd of souls`: []mod.Mod{FLAG("ShepherdOfSouls")},
	`you have shepherd of souls if at least (\d+) corrupted items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("ShepherdOfSouls").Tag(mod.MultiplierThreshold("CorruptedItem").Threshold(num))}, ""
	},
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
	//pob:block:src/Modules/ModParser.lua:2445-2453
	`(\d+)% chance to defend with double your armour for each time you've been hit by an enemy recently, up to (\d+)%`: func(numChance float64, captures []string) ([]mod.Mod, string) {
		capPct := utils.Float(captures[1])
		return []mod.Mod{
			mod.NewFloat("ArmourDefense", mod.TypeMAX, 100).Tag(mod.Condition("ArmourMax")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(numChance/100, 1.0)*100).Tag(mod.Condition("ArmourAvg")).Tag(mod.Multiplier("BeenHitRecently").Limit(capPct / numChance)),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(math.Floor(numChance/100), 1.0)*100).Tag(mod.Condition("ArmourMax").Neg(true)).Tag(mod.Condition("ArmourAvg").Neg(true)).Tag(mod.Multiplier("BeenHitRecently").Limit(capPct / numChance)),
		}, ""
	},
	`(\d+)% chance to defend with (\d+)% of armour for each time you've been hit by an enemy recently, up to (\d+)%`: func(numChance float64, captures []string) ([]mod.Mod, string) {
		numArmourMultiplier := utils.Float(captures[1])
		capPct := utils.Float(captures[2])
		return []mod.Mod{
			mod.NewFloat("ArmourDefense", mod.TypeMAX, numArmourMultiplier-100).Tag(mod.Condition("ArmourMax")),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(numChance/100, 1.0)*(numArmourMultiplier-100)).Tag(mod.Condition("ArmourAvg")).Tag(mod.Multiplier("BeenHitRecently").Limit(capPct / numChance)),
			mod.NewFloat("ArmourDefense", mod.TypeMAX, min(math.Floor(numChance/100), 1.0)*(numArmourMultiplier-100)).Tag(mod.Condition("ArmourMax").Neg(true)).Tag(mod.Condition("ArmourAvg").Neg(true)).Tag(mod.Multiplier("BeenHitRecently").Limit(capPct / numChance)),
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
	`hits have (\d+)% chance to treat enemy monster elemental resistance values as inverted`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("HitsInvertEleResChance", mod.TypeChance, num/100),
		}, ""
	},
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
	`exerted attacks have (\d+)% chance to deal double damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExertDoubleDamageChance", mod.TypeBase, num).Flag(mod.MFlagAttack),
		}, ""
	},
	`grants (\d+) passive skill points?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("ExtraPoints", mod.TypeBase, num),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2466-2468
	`(\d+)% increased accuracy rating at close range`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", mod.TypeIncrease, num).Tag(mod.Condition("AtCloseRange"))}, ""
	},
	`(\d+)% more accuracy rating at close range`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", mod.TypeMore, num).Tag(mod.Condition("AtCloseRange"))}, ""
	},
	`(\d+)% increased accuracy rating against unique enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", mod.TypeIncrease, num).Tag(mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	//pob:src/Modules/ModParser.lua:2469
	`(\d+)% more accuracy rating against unique enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", mod.TypeMore, num).Tag(mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	`can allocate passives from the \w+'s starting point`: []mod.Mod{},
	`projectiles gain damage as they travel farther, dealing up to (\d+)% increased damage with hits to targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", mod.TypeIncrease, num).Flag(mod.MFlagHit).Flag(mod.MFlagProjectile).Tag(mod.DistanceRamp([][]float64{{35, 0}, {70, 1}})),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4585
	`projectiles deal (\d+)% increased damage with hits to targets at the start of their movement, reducing to (\d+)% as they travel farther`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", mod.TypeIncrease, num).Flag(mod.MFlagHit).Flag(mod.MFlagProjectile).Tag(mod.DistanceRamp([][]float64{{35, 1}, {70, 0}})),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2644
	`projectiles gain damage as they travel farther, dealing up to (\d+)% more damage with hits and ailments`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", mod.TypeMore, num).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.DistanceRamp([][]float64{{35, 0}, {70, 1}})),
		}, ""
	},
	`(\d+)% chance to gain elusive on kill`: []mod.Mod{
		mod.NewFlag("Condition:CanBeElusive", true),
	},
	//pob:src/Modules/ModParser.lua:2503
	`immun[ei]t?y? to elemental ailments while on consecrated ground`: []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("OnConsecratedGround"))},
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
	`(\d+)% less damage taken while there are at least two rare or unique enemies nearby`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DamageTaken", "MORE", -num).Tag(mod.MultiplierThreshold("NearbyRareOrUniqueEnemies").Threshold(2)),
		}, ""
	},
	"you take no extra damage from critical strikes while elusive": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.Condition("Elusive")),
		}, ""
	},
	"take no extra damage from critical strikes if you have a magic ring in left slot": []mod.Mod{
		MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.GlobalEffect("Global").Unscalable(true)).Tag(mod.Condition("MagicItemInRing 1")),
	},
	//pob:src/Modules/ModParser.lua:3480
	`gain (\d+)% of physical damage as a random element if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", num).Tag(mod.Condition(selfCastCondition(captures[1])))}, ""
	},
	//pob:src/Modules/ModParser.lua:4152
	`inflict withered for (\d+) seconds on hit if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Condition:CanWither").Tag(mod.Condition(selfCastCondition(captures[1])))}, ""
	},
	//pob:src/Modules/ModParser.lua:4235
	`([a-z]+) exposure on hit if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return selfCastExposure(captures[0], captures[1]), ""
	},
	//pob:src/Modules/ModParser.lua:4238
	`inflict ([a-z]+) exposure on hit if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return selfCastExposure(captures[0], captures[1]), ""
	},
	//pob:src/Modules/ModParser.lua:4888
	`you are unaffected by (.+) if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Self"+firstToUpper(captures[0])+"Effect", "MORE", -100).
			Tag(mod.Condition(selfCastCondition(captures[1]))).
			Tag(mod.GlobalEffect("Global").Unscalable(true))}, ""
	},
	//pob:src/Modules/ModParser.lua:3782-3784
	`hits with prismatic skills always ?i?n?f?l?i?c?t? (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Enemy"+firstToUpper(captures[0])+"Chance", "BASE", 100).
			Flag(mod.MFlagHit).
			Tag(mod.SkillType(string(data.SkillTypeRandomElement)))}, ""
	},
	//pob:src/Modules/ModParser.lua:4891
	`immun[ei]t?y? to ([a-z]*?)s? if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG(firstToUpper(captures[0]) + "Immune").Tag(mod.Condition(selfCastCondition(captures[1])))}, ""
	},
	//pob:src/Modules/ModParser.lua:4948
	`action speed cannot be slowed below base value if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.Condition(selfCastCondition(captures[0])))}, ""
	},
	//pob:src/Modules/ModParser.lua:4951
	`action speed cannot be modified to below base value if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.Condition(selfCastCondition(captures[0])))}, ""
	},
	//pob:src/Modules/ModParser.lua:5025
	`intimidate enemies on hit if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).
			Flag(mod.MFlagHit).
			Tag(mod.Condition(selfCastCondition(captures[0])))}, ""
	},
	//pob:src/Modules/ModParser.lua:5847
	`take no extra damage from critical strikes if you've cast (.+?) in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", 100).
			Tag(mod.GlobalEffect("Global").Unscalable(true)).
			Tag(mod.Condition(selfCastCondition(captures[0])))}, ""
	},
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
	`gain \d+ rage after spending a total of \d+ mana`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`warcries grant (\d+) rage per (\d+) enemy power, up to (\d+)`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`with a murderous eye jewel socketed, melee attacks grant (\d+) rage on hit, no more than once every second`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true).Tag(mod.Condition("HaveMurderousEyeJewelIn{SlotName}")),
	},
	`gain \d+ rage on attack hit`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`gain \d+ rage on hit with axes or swords`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`gain \d+ rage on melee hit`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`gain \d+ rage on melee weapon hit`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`gain \d+ rage on hit with axes`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`gain \d+ rage when hit by an enemy`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`gain \d+ rage on hit with retaliation skills`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`gain \d+ rage when you use a life flask`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`while a unique enemy is in your presence, gain \d+ rage on hit with attacks, no more than once every [\d\.]+ seconds`: []mod.Mod{
		FLAG("Condition:CanGainRage").Tag(mod.ActorCondition("enemy", "RareOrUnique")),
	},
	`while a pinnacle atlas boss is in your presence, gain \d+ rage on hit with attacks, no more than once every [\d\.]+ seconds`: []mod.Mod{
		FLAG("Condition:CanGainRage").Tag(mod.ActorCondition("enemy", "PinnacleBoss")),
	},
	"inherent effects from having rage are tripled": []mod.Mod{mod.NewFloat("Multiplier:RageEffect", mod.TypeBase, 2)},
	//pob:block:src/Modules/ModParser.lua:2561-2563
	`(\d+)% less damage taken per (\d+) rage, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DamageTaken", "MORE", -num).Tag(
				mod.Multiplier("Rage").Div(utils.Float(captures[1])).Limit(-utils.Float(captures[2])).LimitNegTotal()),
		}, ""
	},
	`cannot be stunned while you have at least (\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("StunImmune").Tag(mod.MultiplierThreshold("Rage").Threshold(num)),
		}, ""
	},
	`lose ([\d\.]+)% of life per second per rage while you are not losing rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeDegen", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.Multiplier("RageEffect")),
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
	"cannot be stunned while you have fortify": []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("Fortified"))},
	"cannot be stunned while fortified":        []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("Fortified"))},
	"fortify":                                  []mod.Mod{mod.NewFlag("Condition:Fortified", true)},
	"you have your maximum fortification": []mod.Mod{
		mod.NewFlag("Condition:Fortified", true),
		mod.NewFlag("Condition:HaveMaxFortification", true),
	},
	`you have (\d+) fortification`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MinimumFortification", mod.TypeBase, num)}, ""
	},
	//pob:src/Modules/ModParser.lua:2585
	"nearby allies count as having fortification equal to yours": []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{
			OnlyAllies: true,
			Mod:        mod.NewFlag("YourFortifyEqualToParent", true).Tag(mod.GlobalEffect("Global").Unscalable(true)),
		}),
	},
	"enemies taunted by you cannot evade attacks": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("CannotEvade").Tag(mod.Condition("Taunted"))})},
	`if you've impaled an enemy recently, you and nearby allies have \+(\d+) to armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("Armour", mod.TypeBase, num)}).Tag(mod.Condition("ImpaledRecently")),
		}, ""
	},
	"your hits permanently intimidate enemies that are on full life": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Intimidated", true).Tag(mod.Condition("ChampionIntimidate"))})},
	"enemies ignited by you during effect have malediction":          []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMalediction").Tag(mod.Condition("Ignited"))})},
	`(\d+)% increased armour and energy shield from equipped body armour if equipped helmet, gloves and boots all have armour and energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Body ArmourESAndArmour", "INC", num).
				Tag(mod.StatThreshold("ArmourOnGloves", 1)).
				Tag(mod.StatThreshold("EnergyShieldOnGloves", 1)).
				Tag(mod.StatThreshold("ArmourOnHelmet", 1)).
				Tag(mod.StatThreshold("EnergyShieldOnHelmet", 1)).
				Tag(mod.StatThreshold("ArmourOnBoots", 1)).
				Tag(mod.StatThreshold("EnergyShieldOnBoots", 1)),
		}, ""
	},
	"minions' accuracy rating is equal to yours": []mod.Mod{FLAG("MinionAccuracyEqualsAccuracy")},
	"monsters cannot block your attacks":         []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("CannotBlockAttacks")})},
	`minions recover (\d+)% of their life when they block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{
			Mod: MOD("LifeOnBlock", "BASE", 1).Tag(mod.PercentStat("Life", num)),
		})}, ""
	},
	"nearby enemies are intimidated while you have rage": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).
			Tag(mod.MultiplierThreshold("RageStack").Threshold(1)),
	},
	"stun threshold is based on energy shield instead of life": []mod.Mod{
		FLAG("StunThresholdBasedOnEnergyShieldInsteadOfLife"),
		MOD("StunThresholdEnergyShieldPercent", "BASE", 100),
	},
	`stun threshold is based on (\d+)% of your energy shield instead of life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("StunThresholdBasedOnEnergyShieldInsteadOfLife"),
			MOD("StunThresholdEnergyShieldPercent", "BASE", num),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4985-4988
	`(\d+)% of your energy shield is added to your stun threshold`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("AddESToStunThreshold"),
			MOD("ESToStunThresholdPercent", "BASE", num),
		}, ""
	},
	"you cannot be cursed with silence":                 []mod.Mod{FLAG("SilenceImmune")},
	"skills cost energy shield instead of mana or life": []mod.Mod{FLAG("CostESInsteadOfManaOrLife")},
	"enemies blinded by you have malediction":           []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMalediction").Tag(mod.Condition("Blinded"))})},
	`recover energy shield equal to (\d+)% of armour when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnBlock", "BASE", 1).Tag(mod.PercentStat("Armour", num))}, ""
	},
	`recover energy shield equal to (\d+)% of evasion rating when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShieldOnBlock", "BASE", 1).Tag(mod.PercentStat("Evasion", num))}, ""
	},
	//pob:src/Modules/ModParser.lua:2589-2594
	`you and allies affected by your placed banners regenerate ([\d\.]+)% of life per second for each stage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("AffectedByPlacedBanner")).Tag(mod.Multiplier("BannerValour"))}),
		}, ""
	},
	`you and allies near your banner regenerate ([\d\.]+)% of life per second for each valour consumed for that banner`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition("AffectedByPlacedBanner")).Tag(mod.Multiplier("BannerValour"))}),
		}, ""
	},
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
	//pob:block:src/Modules/ModParser.lua:2611-2618
	`non-unique jewels cause increases and reductions to other damage types in a ([a-z]+) radius to be transformed to apply to ([a-z]+) damage`: func(_ float64, captures []string) ([]mod.Mod, string) {
		conv := getSimpleConv([]string{"PhysicalDamage", "FireDamage", "ColdDamage", "LightningDamage", "ChaosDamage", "ElementalDamage"},
			firstToUpper(captures[1])+"Damage", mod.TypeIncrease, true)
		return []mod.Mod{
			mod.NewList("ExtraJewelFunc", extraJewelFuncEntry{
				Radius: firstToUpper(captures[0]),
				Type:   "Other",
				Func: func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
					// Ignore Timeless Jewels
					if node != nil && !node.conquered {
						conv(node, out, d)
					}
				},
			}).Tag(
				mod.ItemCondition("{SlotName}").Rarity("UNIQUE").Negate(true),
				mod.ItemCondition("{SlotName}").Rarity("RELIC").Negate(true),
			),
		}, ""
	},
	`non-unique jewels cause small and notable passive skills in a ([a-z]+) radius to also grant \+(\d+) to ([a-z]+)`: func(_ float64, captures []string) ([]mod.Mod, string) {
		attr := firstToUpper(captures[2])
		if len(attr) < 3 {
			return nil, ""
		}
		attr = attr[:3]
		value := utils.Float(captures[1])
		return []mod.Mod{
			mod.NewList("ExtraJewelFunc", extraJewelFuncEntry{
				Radius: firstToUpper(captures[0]),
				Type:   "Other",
				Func: func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
					if node == nil || node.conquered {
						return
					}
					if node.nodeType != "Notable" && node.nodeType != "Normal" {
						return
					}
					out.NewMod(attr, mod.TypeBase, value, d.modSource)
				},
			}).Tag(
				mod.ItemCondition("{SlotName}").Rarity("UNIQUE").Negate(true),
				mod.ItemCondition("{SlotName}").Rarity("RELIC").Negate(true),
			),
		}, ""
	},
	"projectiles pierce all nearby targets": []mod.Mod{mod.NewFlag("PierceAllTargets", true)},
	`gain \+(\d+) life when you hit a bleeding enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnHit", "BASE", num).Tag(mod.ActorCondition("enemy", "Bleeding"))}, ""
	},
	"accuracy rating is doubled": []mod.Mod{mod.NewFloat("Accuracy", mod.TypeMore, 100)},
	`(\d+)% increased blink arrow and mirror arrow cooldown recovery speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CooldownRecovery", "INC", num).Tag(mod.SkillName("Blink Arrow", "Mirror Arrow").Transfigured(true)),
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
	//pob:src/Modules/ModParser.lua:5574
	`final repeat of spells has (\d+)% increased area of effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RepeatFinalAreaOfEffect", "INC", num).Flag(mod.MFlagSpell).
			Tag(mod.Condition("CastOnFrostbolt").Neg(true)).
			Tag(mod.Condition("averageRepeat", "alwaysFinalRepeat"))}, ""
	},
	//pob:src/Modules/ModParser.lua:5673
	`final repeat of attack skills deals (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("RepeatFinalDamage", "MORE", num).KeywordFlag(mod.KeywordFlagAttack)}, ""
	},
	`(\d+)% increased area of effect while you don't have convergence`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("Convergence").Neg(true))}, ""
	},
	`exposure you inflict applies an extra (-?\d+)% to the affected resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ExtraExposure", mod.TypeBase, num)}, ""
	},
	"cannot take reflected elemental damage": []mod.Mod{MOD("ElementalReflectedDamageTaken", "MORE", -100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	//pob:block:src/Modules/ModParser.lua:4693
	`cannot take reflected (\w+) damage if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD(firstToUpper(captures[0])+"ReflectedDamageTaken", "MORE", -100).Tag(mod.MultiplierThreshold(firstToUpper(captures[2]) + "Item").Threshold(utils.Float(captures[1])))}, ""
	},
	`every \d+ seconds:`: []mod.Mod{},
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
	"you have elemental conflux if the stars are aligned": []mod.Mod{
		FLAG("PhysicalCanChill").Tag(mod.Condition("StarsAreAligned")),
		FLAG("LightningCanChill").Tag(mod.Condition("StarsAreAligned")),
		FLAG("FireCanChill").Tag(mod.Condition("StarsAreAligned")),
		FLAG("ChaosCanChill").Tag(mod.Condition("StarsAreAligned")),
		MOD("EnemyShockChance", "BASE", 100).Tag(mod.Condition("StarsAreAligned")),
		FLAG("PhysicalCanShock").Tag(mod.Condition("StarsAreAligned")),
		FLAG("ColdCanShock").Tag(mod.Condition("StarsAreAligned")),
		FLAG("FireCanShock").Tag(mod.Condition("StarsAreAligned")),
		FLAG("ChaosCanShock").Tag(mod.Condition("StarsAreAligned")),
		MOD("EnemyIgniteChance", "BASE", 100).Tag(mod.Condition("StarsAreAligned")),
		FLAG("PhysicalCanIgnite").Tag(mod.Condition("StarsAreAligned")),
		FLAG("LightningCanIgnite").Tag(mod.Condition("StarsAreAligned")),
		FLAG("ColdCanIgnite").Tag(mod.Condition("StarsAreAligned")),
		FLAG("ChaosCanIgnite").Tag(mod.Condition("StarsAreAligned")),
	},
	`gain chilling, shocking and igniting conflux for \d seconds`: []mod.Mod{},
	//pob:src/Modules/ModParser.lua:3064-3067
	"while affected by glorious madness, inflict mania on nearby enemies every second": []mod.Mod{
		FLAG("Condition:CanInflictMania").Tag(mod.Condition("AffectedByGloriousMadness")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:AfflictedByMania")}).Tag(mod.Condition("AffectedByGloriousMadness")),
	},
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
	//pob:src/Modules/ModParser.lua:2761
	`immun[ei]t?y? to elemental ailments while affected by glorious madness`: []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("AffectedByGloriousMadness"))},
	"summoned golems are immune to elemental damage": []mod.Mod{
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Elemancer")}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ElementalDamageTaken", "MORE", -100)}).Tag(mod.SkillType(string(data.SkillTypeGolem))),
	},
	`(\d+)% increased golem damage per summoned golem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Damage", mod.TypeIncrease, num)}).Tag(mod.SkillType(string(data.SkillTypeGolem))).Tag(mod.PerStat(0, "ActiveGolemLimit")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2768
	`shocks from your hits always increase damage taken by at least (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ShockMinimum", mod.TypeBase, num)}, ""
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
	//pob:src/Modules/ModParser.lua:2824-2827
	"other aegis skills are disabled": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeAegis))),
			FLAG("EnableSkill").Tag(mod.SkillName()),
		}, ""
	},
	`primal aegis can take (\d+) elemental damage per allocated notable passive skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ElementalAegisValue", "MAX", num).Tag(mod.Multiplier("AllocatedNotable").Base(0)).Tag(mod.GlobalEffect("Buff").Unscalable(true)),
		}, ""
	},
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
	`retaliation skills have (\d+)% increased speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Speed", "INC", num).Tag(mod.SkillType(string(data.SkillTypeRetaliation))),
			MOD("WarcrySpeed", "INC", num).KeywordFlag(mod.KeywordFlagWarcry).Tag(mod.SkillType(string(data.SkillTypeRetaliation))),
		}, ""
	},
	`grants armour equal to (\d+)% of your reserved life to you and nearby allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("GrantReservedLifeAsAura", "LIST", mod.GrantReservedPoolAsAura{Mod: MOD("Armour", "BASE", num/100)}),
		}, ""
	},
	`grants armour equal to (\d+)% of your reserved mana to you and nearby allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("GrantReservedManaAsAura", "LIST", mod.GrantReservedPoolAsAura{Mod: MOD("Armour", "BASE", num/100)}),
		}, ""
	},
	`grants maximum energy shield equal to (\d+)% of your reserved mana to you and nearby allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("GrantReservedManaAsAura", "LIST", mod.GrantReservedPoolAsAura{Mod: MOD("EnergyShield", "BASE", num/100)}),
		}, ""
	},
	"warcries cost no mana": []mod.Mod{MOD("ManaCost", "MORE", -100).KeywordFlag(mod.KeywordFlagWarcry)},
	`warcries cost \+(\d+)% of life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeCostBase", "BASE", 1).KeywordFlag(mod.KeywordFlagWarcry).Tag(mod.PercentStat("Life", num).Floor()),
		}, ""
	},
	`vaal skills used during effect have (\d+)% reduced soul gain prevention duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:2280
		return []mod.Mod{MOD("SoulGainPreventionDuration", "INC", -num).
			Tag(mod.Condition("UsingFlask")).
			Tag(mod.SkillType(string(data.SkillTypeVaal)))}, ""
	},
	`vaal volcanic fissure and vaal molten strike have (\d+)% reduced soul gain prevention duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:2281
		return []mod.Mod{MOD("SoulGainPreventionDuration", "INC", -num).
			Tag(mod.SkillName("Volcanic Fissure", "Molten Strike").Transfigured(true)).
			Tag(mod.SkillType(string(data.SkillTypeVaal)))}, ""
	},
	`\+(\d+)% chance to block attack damage for \d seconds? every \d seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("BlockChance", "BASE", num).Tag(mod.Condition("BastionOfHopeActive")),
		}, ""
	},
	`if you've blocked in the past \d+ seconds, you and nearby allies cannot be stunned`: []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFlag("StunImmune", true)}).Tag(mod.Condition("BlockedRecently"))},
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
	`you and your totems regenerate ([\d\.]+)% of life per second for each summoned totem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeRegenPercent", "BASE", num).Tag(mod.PerStat(0, "TotemsSummoned")),
			MOD("LifeRegenPercent", "BASE", num).KeywordFlag(mod.KeywordFlagTotem),
		}, ""
	},
	`enemies take (\d+)% increased damage for each of your brands attached to them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTaken", "INC", num).Tag(mod.Multiplier("BrandsAttached").Base(0))}),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2888
	`non-damaging ailments have (\d+)% reduced effect on you while you have arcane surge`: func(num float64, captures []string) ([]mod.Mod, string) {
		ailments := data.NonDamagingAilment("").Values()
		mods := make([]mod.Mod, len(ailments))
		for i, ailment := range ailments {
			mods[i] = MOD("Self"+string(ailment)+"Effect", "INC", -num).Tag(mod.Condition("AffectedByArcaneSurge"))
		}
		return mods, ""
	},
	//pob:src/Modules/ModParser.lua:2895
	`immun[ei]t?y? to elemental ailments while you have arcane surge`: []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("AffectedByArcaneSurge"))},
	//pob:src/Modules/ModParser.lua:2488
	`brands have (\d+)% increased area of effect if (\d+)% of attached duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("BrandLastHalf")).Tag(mod.SkillType(string(data.SkillTypeBrand)))}, ""
	},
	`brands have (\d+)% more activation frequency if (\d+)% of attached duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("BrandActivationFrequency", "MORE", num).Tag(mod.Condition("BrandLastQuarter")),
		}, ""
	},
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
	//pob:src/Modules/ModParser.lua:2903
	`consecrated ground you create grants immun[ei]t?y? to elemental ailments to you and allies`: []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("ElementalAilmentImmune").Tag(mod.Condition("OnConsecratedGround"))}),
	},
	`consecrated ground you create grants (\d+)% increased accuracy rating to you and allies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{
			Mod: MOD("ConsecratedGroundAlsoAccuracy", "INC", num).Tag(mod.Condition("OnConsecratedGround")),
		})}, ""
	},
	//pob:src/Modules/ModParser.lua:2904-2906
	`gain fanaticism for (\d+) seconds on reaching maximum fanatic charges`: func(num float64, captures []string) ([]mod.Mod, string) {
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
	//pob:block:src/Modules/ModParser.lua:2911
	"consecrated ground you create causes life regeneration to also recover energy shield for you and allies": []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("LifeRegenerationRecoversEnergyShield").Tag(mod.Condition("OnConsecratedGround"))}),
	},
	`(\d+)% more attack damage for each non-instant spell you've cast in the past 8 seconds, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num).Flag(mod.MFlagAttack).Tag(mod.Multiplier("CastLast8Seconds").Limit(utils.Float(captures[1])).LimitTotal(true)),
		}, ""
	},
	"armour received from body armour is doubled": []mod.Mod{mod.NewFlag("Unbreakable", true)},
	//pob:src/Modules/ModParser.lua:2916-2919
	"action speed cannot be modified to below base value":   []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"movement speed cannot be modified to below base value": []mod.Mod{mod.NewFlag("MovementSpeedCannotBeBelowBase", true)},
	"you cannot be slowed to below base speed":              []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"cannot be slowed to below base speed":                  []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"gain accuracy rating equal to your strength":           []mod.Mod{MOD("Accuracy", "BASE", 1).Tag(mod.PerStat(0, "Str"))},
	"gain accuracy rating equal to twice your strength":     []mod.Mod{MOD("Accuracy", "BASE", 2).Tag(mod.PerStat(0, "Str"))},
	//pob:src/Modules/ModParser.lua:5227
	`gain accuracy rating equal to your intelligence`: []mod.Mod{MOD("Accuracy", "BASE", 1).Tag(mod.PerStat(0, "Int"))},
	//pob:src/Modules/ModParser.lua:5456-5457
	`(\d+)% increased total power counted by warcries`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("WarcryPower", "INC", num)}, ""
	},
	`warcries have a minimum of (\d+) power`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinimumWarcryPower", "BASE", num)}, ""
	},
	//pob:src/Modules/ModParser.lua:3929-3931
	`poisons you inflict on non-poisoned enemies deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(mod.KeywordFlagPoison).Tag(mod.Condition("NonPoisonedOnly"))}, ""
	},
	`(\d+)% chance to gain (\d+) life on hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		amount := utils.Float(captures[1])
		return []mod.Mod{
			MOD("LifeOnHit", "BASE", amount*num/100).Flag(mod.MFlagAttack | mod.MFlagHit).Tag(mod.Condition("AverageResourceGain")),
			MOD("LifeOnHit", "BASE", amount).Flag(mod.MFlagAttack | mod.MFlagHit).Tag(mod.Condition("MaxResourceGain")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2315
	`\+(\d+)% to cold damage over time multiplier for each (\d+)% overcapped cold resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ColdDotMultiplier", "BASE", num).Tag(mod.PerStat(utils.Float(captures[1]), "ColdResistOverCap"))}, ""
	},
	`hits against you cannot be critical strikes if you've been stunned recently`: []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("NeverCrit", true)}).Tag(mod.Condition("StunnedRecently")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:NeverCrit", true)}).Tag(mod.Condition("StunnedRecently")),
	},
	//pob:src/Modules/ModParser.lua:2419
	`arcane surge also grants (\d+)% increased life regeneration rate to you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ArcaneSurgeAlsoLifeRegen", "BASE", num)}, ""
	},
	//pob:src/Modules/ModParser.lua:4865-4866
	`immune to bleeding if equipped helmet has higher armour than evasion rating`: []mod.Mod{
		mod.NewFlag("BleedImmune", true).Tag(mod.Condition("HelmetArmourHigherThanEvasion")),
	},
	`immune to poison if equipped helmet has higher evasion rating than armour`: []mod.Mod{
		mod.NewFlag("PoisonImmune", true).Tag(mod.Condition("HelmetEvasionHigherThanArmour")),
	},
	"your offering skills also affect you": []mod.Mod{
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", mod.SkillData{Key: "buffNotPlayer", Value: 0})}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering")),
	},
	`your offerings have (\d+)% reduced effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("BuffEffectOnPlayer", "INC", -num)}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering")),
		}, ""
	},
	`your offerings have (\d+)% increased effect on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("BuffEffectOnPlayer", "INC", num)}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering")),
		}, ""
	},
	`if you've consumed a corpse recently, you and your minions have (\d+)% increased area of effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AreaOfEffect", "INC", num).Tag(mod.Condition("ConsumedCorpseRecently")),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("AreaOfEffect", "INC", num)}).Tag(mod.Condition("ConsumedCorpseRecently")),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:2928
	`with at least one nearby corpse, nearby enemies deal (\d+)% reduced damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("Damage", mod.TypeIncrease, -num)}).Tag(mod.MultiplierThreshold("NearbyCorpse").Threshold(1))}, ""
	},
	//pob:src/Modules/ModParser.lua:2954
	`nearby chilled enemies deal (\d+)% reduced damage with hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("Damage", mod.TypeIncrease, -num)}).Tag(mod.ActorCondition("enemy", "Chilled"))}, ""
	},
	`nearby hindered enemies deal (\d+)% reduced damage over time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("DamageOverTime", mod.TypeIncrease, -num)}).Tag(mod.ActorCondition("enemy", "Hindered"))}, ""
	},
	`nearby enemy monsters' action speed is at most (\d+)% of base value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFloat("MaximumActionSpeedReduction", mod.TypeMAX, 100-num)})}, ""
	},
	`nearby enemies cannot deal critical strikes`: []mod.Mod{
		mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("NeverCrit", true)}),
		mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("Condition:NeverCrit", true)}),
	},
	//pob:block:src/Modules/ModParser.lua:3543-3545
	`on non-channelling attack, set a life flask with greater than \d+% of maximum charges remaining to (\d+)% for each charge removed this way, that attack gains \+(\d+)% to damage over time multiplier`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DotMultiplier", "BASE", utils.Float(captures[1])).Tag(mod.PercentStat("LifeFlaskCharges", 100-num).Floor()).
			Tag(mod.SkillType(string(data.SkillTypeChannel)).Neg(true)).Tag(mod.SkillType(string(data.SkillTypeAttack)))}, ""
	},
	`enemies poisoned by you cannot deal critical strikes`: []mod.Mod{
		mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("NeverCrit", true).Tag(mod.Condition("Poisoned"))}),
		mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("Condition:NeverCrit", true).Tag(mod.Condition("Poisoned"))}),
	},
	`marked enemy cannot deal critical strikes`: []mod.Mod{
		mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("NeverCrit", true).Tag(mod.Condition("Marked"))}),
		mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("Condition:NeverCrit", true).Tag(mod.Condition("Marked"))}),
	},
	`hits against you are always critical strikes`: []mod.Mod{
		mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("AlwaysCrit", true)}),
		mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("Condition:AlwaysCrit", true)}),
	},
	`marked enemy cannot evade attacks`: []mod.Mod{
		mod.NewList("EnemyModifier", mod.EnemyModifier{Mod: mod.NewFlag("CannotEvade", true).Tag(mod.Condition("Marked"))}),
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
	`corpses you spawn have (\d+)% reduced maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("CorpseLife", mod.TypeIncrease, -num),
		}, ""
	},
	"enemies you curse have malediction": []mod.Mod{MOD("AffectedByCurseMod", "LIST", mod.AffectedByCurseMod{Mod: mod.NewFlag("HasMalediction", true)})},
	`when you kill an enemy, for each curse on that enemy, gain (\d+)% of non-chaos damage as extra chaos damage for 4 seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("NonChaosDamageGainAsChaos", "BASE", num).Tag(mod.Condition("KilledRecently")).Tag(mod.Multiplier("CurseOnEnemy").Base(0)),
		}, ""
	},
	"cannot be stunned while you have energy shield":                     []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("HaveEnergyShield"))},
	`every second, inflict withered on nearby enemies for (\d+) seconds`: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	"always poison on hit while using a flask":                           []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	`poisons you inflict during any flask effect have (\d+)% chance to deal (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", utils.Float(captures[1])*num/100).KeywordFlag(mod.KeywordFlagPoison).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2959
	`immun[ei]t?y? to elemental ailments during any flask effect`: []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("UsingFlask"))},
	//pob:block:src/Modules/ModParser.lua:2960
	`grant bonuses to non-channelling skills you use by consuming (\d+) charges from a flask of each of the following types, if possible:`: []mod.Mod{},
	`nearby enemies have (\d+)% less accuracy rating while you have phasing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("Accuracy", "MORE", -num)}).Tag(mod.Condition("Phasing")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2972
	`immun[ei]t?y? to elemental ailments while phasing`: []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("Phasing"))},
	`nearby enemies have fire, cold and lightning exposure while you have phasing, applying -(\d+)% to those resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", -num)}).Tag(mod.Condition("Phasing")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", -num)}).Tag(mod.Condition("Phasing")),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", -num)}).Tag(mod.Condition("Phasing")),
		}, ""
	},
	`hits have (\d+)% chance to deal (\d+)% more area damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num*utils.Float(captures[1])/100).Flag(mod.MFlagArea).Flag(mod.MFlagHit),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:2987-2990
	`immun[ei]t?y? to ignite and shock`: []mod.Mod{
		FLAG("IgniteImmune"),
		FLAG("ShockImmune"),
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
	`deal up to (\d+)% more melee damage to enemies, based on proximity`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "MORE", num).Flag(mod.MFlagAttack).Flag(mod.MFlagMelee).Tag(mod.MeleeProximity([]int{1, 0})),
		}, ""
	},
	"cannot be stunned while leeching":                                  []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("Leeching"))},
	"you are immune to bleeding while leeching":                         []mod.Mod{FLAG("BleedImmune").Tag(mod.Condition("Leeching"))},
	"life leech effects are not removed at full life":                   []mod.Mod{mod.NewFlag("CanLeechLifeOnFullLife", true)},
	"life leech effects are not removed when unreserved life is filled": []mod.Mod{mod.NewFlag("CanLeechLifeOnFullLife", true)},
	//pob:src/Modules/ModParser.lua:5144-5152
	"effect is not removed when unreserved mana is filled":              []mod.Mod{mod.NewFlag("ManaFlaskEffectNotRemoved", true)},
	"life flask effects are not removed when unreserved life is filled": []mod.Mod{mod.NewFlag("LifeFlaskEffectNotRemoved", true)},
	"mana flask effects are not removed when unreserved mana is filled": []mod.Mod{mod.NewFlag("ManaFlaskEffectNotRemoved", true)},
	//pob:src/Modules/ModParser.lua:3057
	"utility flasks are disabled": []mod.Mod{mod.NewFlag("UtilityFlasksDoNotApplyToPlayer", true)},
	//pob:src/Modules/ModParser.lua:3000
	"energy shield leech effects from attacks are not removed at full energy shield": []mod.Mod{mod.NewFlag("CanLeechEnergyShieldOnFullEnergyShield", true)},
	//pob:src/Modules/ModParser.lua:3013
	"energy shield leech effects are not removed when energy shield is filled": []mod.Mod{mod.NewFlag("CanLeechEnergyShieldOnFullEnergyShield", true)},
	"cannot take reflected physical damage":                                    []mod.Mod{MOD("PhysicalReflectedDamageTaken", "MORE", -100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	//pob:block:src/Modules/ModParser.lua:4692
	"immune to reflected damage": []mod.Mod{MOD("ReflectedDamageTaken", "MORE", -100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
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
	`(\d+)% chance to gain (\d+)% of non-chaos damage with hits as extra chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("NonChaosDamageGainAsChaos", "BASE", num/100*utils.Float(captures[1])),
		}, ""
	},
	"movement skills cost no mana": []mod.Mod{MOD("ManaCost", "MORE", -100).KeywordFlag(mod.KeywordFlagMovement)},
	"cannot be stunned while you have ghost shrouds": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("StunImmune").Tag(mod.MultiplierThreshold("GhostShroud").Threshold(1)),
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
		MOD("WeaponData", "LIST", mod.WeaponData{Key: "PhysicalMin", Unset: true}),
		MOD("WeaponData", "LIST", mod.WeaponData{Key: "PhysicalMax", Unset: true}),
		MOD("WeaponData", "LIST", mod.WeaponData{Key: "PhysicalDPS", Unset: true}),
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
	"no chance to block":                          []mod.Mod{MOD("ArmourData", "LIST", mod.ArmourData{Key: "BlockChance", Value: 0})},

	//pob:src/Modules/ModParser.lua:3539
	`(\d+)% chance to gain a flask charge when you deal a critical strike`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("FlaskChargeOnCritChance", mod.TypeBase, num)}, ""
	},
	"gain a flask charge when you deal a critical strike":                                 []mod.Mod{MOD("FlaskChargeOnCritChance", "BASE", 100)},
	"gain a flask charge when you deal a critical strike while affected by precision":     []mod.Mod{MOD("FlaskChargeOnCritChance", "BASE", 100).Tag(mod.Condition("AffectedByPrecision"))},
	"gain a flask charge when you deal a critical strike while at maximum frenzy charges": []mod.Mod{MOD("FlaskChargeOnCritChance", "BASE", 100).Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	//pob:src/Modules/ModParser.lua:4677
	`(\d+)% of physical damage is taken from mana before life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("PhysicalDamageTakenFromManaBeforeLife", mod.TypeBase, num)}, ""
	},
	//pob:src/Modules/ModParser.lua:4092
	"you are at maximum chance to block spell damage if you have not blocked recently": []mod.Mod{
		FLAG("MaxSpellBlockIfNotBlockedRecently").Tag(mod.Condition("BlockedRecently").Neg(true)),
	},
	//pob:src/Modules/ModParser.lua:4981
	`stun threshold is based on (\d+)% of your mana instead of life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("StunThresholdBasedOnManaInsteadOfLife"),
			mod.NewFloat("StunThresholdManaPercent", mod.TypeBase, num),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4696
	"attacks cannot hit you": []mod.Mod{FLAG("AlwaysEvade")},
	//pob:src/Modules/ModParser.lua:3800
	"modifiers to ignite duration on you apply to all elemental ailments": []mod.Mod{
		FLAG("IgniteDurationAppliesToElementalAilments"),
	},
	"cannot deal critical strikes with attacks": []mod.Mod{
		FLAG("NeverCrit").Flag(mod.MFlagAttack),
		FLAG("Condition:NeverCrit").Flag(mod.MFlagAttack),
	},
	//pob:src/Modules/ModParser.lua:2789
	"always freeze, shock and ignite": []mod.Mod{
		mod.NewFloat("EnemyFreezeChance", mod.TypeBase, 100),
		mod.NewFloat("EnemyShockChance", mod.TypeBase, 100),
		mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 100),
	},
	//pob:src/Modules/ModParser.lua:5007
	"melee weapon attacks have culling strike": []mod.Mod{
		MOD("CullPercent", "MAX", 10).
			Flag(mod.MFlagAttack | mod.MFlagWeaponMelee).
			Tag(mod.GlobalEffect("Global").Unscalable(true)),
	},
	//pob:src/Modules/ModParser.lua:4044
	"you can apply an additional curse while at maximum power charges": []mod.Mod{
		MOD("EnemyCurseLimit", "BASE", 1).Tag(mod.StatThresholdStat("PowerCharges", "PowerChargesMax")),
	},
	//pob:block:src/Modules/ModParser.lua:4637-4641
	`gain (\d+) life per cursed enemy hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnHit", "BASE", num).Flag(mod.MFlagAttack | mod.MFlagHit).
				Tag(mod.ActorCondition("enemy", "Cursed")),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:4638
	`gain (\d+) life for each ignited enemy hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnHit", "BASE", num).Flag(mod.MFlagAttack | mod.MFlagHit).Tag(mod.ActorCondition("enemy", "Ignited"))}, ""
	},
	`gain (\d+) mana per cursed enemy hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ManaOnHit", "BASE", num).Flag(mod.MFlagAttack | mod.MFlagHit).
				Tag(mod.ActorCondition("enemy", "Cursed")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:5257
	"deal no physical or elemental damage": []mod.Mod{
		FLAG("DealNoPhysical"), FLAG("DealNoCold"), FLAG("DealNoFire"), FLAG("DealNoLightning"),
	},
	//pob:src/Modules/ModParser.lua:2217
	"your melee hits can't be evaded while wielding a sword": []mod.Mod{
		FLAG("CannotBeEvaded").Flag(mod.MFlagMelee | mod.MFlagHit).Tag(mod.Condition("UsingSword")),
	},
	//pob:src/Modules/ModParser.lua:3698
	`chill enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChillAsThoughDealing", "MORE", num)}, ""
	},
	//pob:src/Modules/ModParser.lua:4541
	"projectiles pierce all burning enemies": []mod.Mod{
		FLAG("PierceAllTargets").Tag(mod.ActorCondition("enemy", "Burning")),
	},
	"hits with this weapon ignore enemy physical damage reduction": []mod.Mod{
		MOD("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", 100).Flag(mod.MFlagHit).
			Tag(mod.Condition("{Hand}Attack")).
			Tag(mod.SkillType(string(data.SkillTypeAttack))),
	},
	//pob:src/Modules/ModParser.lua:5454
	"non-instant warcries ignore their cooldown when used": []mod.Mod{
		MOD("CooldownRecovery", "OVERRIDE", 0).KeywordFlag(mod.KeywordFlagWarcry).
			Tag(mod.SkillType(string(data.SkillTypeInstant)).Neg(true)),
	},
	//pob:src/Modules/ModParser.lua:4058
	`nearby enemy monsters have at least (\d+)% of life reserved`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("LifeReservationPercent", mod.TypeBase, num)}),
		}, ""
	},
	`minions recover (\d+)% of life on killing a poisoned enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("LifeOnKill", "BASE", 1).
				Tag(mod.PercentStat("Life", num)).
				Tag(mod.MultiplierThreshold("PoisonStack").Actor("enemy").Threshold(1))}),
		}, ""
	},
	`minions have (\d+)% chance to inflict withered on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFlag("Condition:CanWither", true)}),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4417
	`raised spectres fire (\d+) additional projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("ProjectileCount", mod.TypeBase, num)}).
				Tag(mod.SkillName("Raise Spectre").Transfigured(true)),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4388
	`minions have (\d+)% chance to deal double damage per fortification on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("DoubleDamageChance", "BASE", num).
				Tag(mod.PerStat(0, "FortificationStacks").Actor("parent"))}),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4502
	"summoned skeleton warriors are permanent and follow you": []mod.Mod{
		FLAG("RaisedSkeletonPermanentDuration").Tag(mod.SkillName("Summon Skeletons")),
	},
	//pob:block:src/Modules/ModParser.lua:5341-5346
	"your blessing skills are disabled": []mod.Mod{
		FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeBlessing))),
	},
	"your warcries are disabled": []mod.Mod{
		FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeWarcry))),
	},
	`offering skills have (\d+)% increased duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Duration", "INC", num).
				Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4911
	"enemies inflict elemental ailments on you instead of nearby allies": []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFlag("ElementalAilmentImmune", true), OnlyAllies: true}),
	},
	"has no energy shield":                        []mod.Mod{MOD("ArmourData", "LIST", mod.ArmourData{Key: "EnergyShield", Value: 0})},
	"hits can't be evaded":                        []mod.Mod{FLAG("CannotBeEvaded").Tag(mod.Condition("{Hand}Attack"))},
	"causes bleeding on hit":                      []mod.Mod{MOD("BleedChance", "BASE", 100).Tag(mod.Condition("{Hand}Attack"))},
	"poisonous hit":                               []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.Condition("{Hand}Attack"))},
	"attacks with this weapon deal double damage": []mod.Mod{MOD("DoubleDamageChance", "BASE", 100).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack)))},
	`hits with this weapon gain (\d+)% of physical damage as extra cold or lightning damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalDamageGainAsColdOrLightning", "BASE", num).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`hits with this weapon shock enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ShockAsThoughDealing", "MORE", num).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`hits with this weapon freeze enemies as though dealing (\d+)% more damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FreezeAsThoughDealing", "MORE", num).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
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
	"life leech from hits with this weapon applies instantly":   []mod.Mod{mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Tag(mod.Condition("{Hand}Attack"))},
	"life leech from hits with this weapon is instant":          []mod.Mod{mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Tag(mod.Condition("{Hand}Attack"))},
	"mana leech from hits with this weapon is instant":          []mod.Mod{mod.NewFloat("InstantManaLeech", mod.TypeBase, 100).Tag(mod.Condition("{Hand}Attack"))},
	"gain life from leech instantly from hits with this weapon": []mod.Mod{mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack)))},
	//pob:block:src/Modules/ModParser.lua:3135
	"life leech from melee damage is instant": []mod.Mod{mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Flag(mod.MFlagMelee)},
	//pob:block:src/Modules/ModParser.lua:3137-3141
	`(\d+)% of leech from hits with this weapon is instant per enemy power`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("InstantLifeLeech", mod.TypeBase, num).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))).Tag(mod.Multiplier("EnemyPower")),
			mod.NewFloat("InstantManaLeech", mod.TypeBase, num).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))).Tag(mod.Multiplier("EnemyPower")),
			mod.NewFloat("InstantEnergyShieldLeech", mod.TypeBase, num).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))).Tag(mod.Multiplier("EnemyPower")),
		}, ""
	},
	"instant recovery": []mod.Mod{mod.NewFloat("FlaskInstantRecovery", mod.TypeBase, 100)},
	`(\d+)% of recovery applied instantly`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("FlaskInstantRecovery", mod.TypeBase, num)}, ""
	},
	"has no attribute requirements": []mod.Mod{mod.NewFlag("NoAttributeRequirements", true)},
	`trigger a socketed spell when a hit from this weapon freezes a target, with a ([\d\.]+) second cooldown`: []mod.Mod{
		//pob:src/Modules/ModParser.lua:5282
		MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnBowAttackFreezeHit", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
	},
	"you cannot cast socketed hex curse skills inflict socketed hexes on enemies that trigger your traps": []mod.Mod{
		//pob:src/Modules/ModParser.lua:5290
		MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportCurseOnTrapTriggered", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
	},
	"trigger a socketed spell when you attack with this weapon":                                        []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed spell when you attack with this weapon, with a ([\d\.]+) second cooldown`:      []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"trigger a socketed spell when you use a skill":                                                    []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnSkillUse", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed spell when you use a skill, with a (\d+) second cooldown`:                      []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnSkillUse", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed spell when you use a skill, with a (\d+) second cooldown and (\d+)% more cost`: []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnSkillUse", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	//pob:src/Modules/ModParser.lua:3153-3154
	"trigger socketed spells when you focus":                                                    []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellFromHelmet", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger socketed spells when you focus, with a ([\d\.]+) second cooldown`:                  []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellFromHelmet", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"trigger a socketed spell when you attack with a bow":                                       []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnBowAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed spell when you attack with a bow, with a ([\d\.]+) second cooldown`:     []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnBowAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	"trigger a socketed bow skill when you attack with a bow":                                   []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerBowSkillOnBowAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`trigger a socketed bow skill when you attack with a bow, with a ([\d\.]+) second cooldown`: []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerBowSkillOnBowAttack", Level: 1}).Tag(mod.SocketedIn("{SlotName}"))},
	`(\d+)% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least (\d+) mana to use a skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("KitavaTriggerChance", "BASE", num).Source("Kitava's Thirst"),
			MOD("KitavaRequiredManaCost", "BASE", utils.Float(captures[1])).Source("Kitava's Thirst"),
			MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportCastOnManaSpent", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:3166-3175
	`(\d+)% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least (\d+) mana on an upfront cost to use or trigger a skill, with a ([\d\.]+) second cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("KitavaTriggerChance", "BASE", num).Source("Kitava's Thirst"),
			MOD("KitavaRequiredManaCost", "BASE", utils.Float(captures[1])).Source("Kitava's Thirst"),
			MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportCastOnManaSpent", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
		}, ""
	},
	`(\d+)% chance to [c?t?][a?r?][s?i?][t?g?]g?e?r? socketed spells when you spend at least (\d+) life on an upfront cost to use or trigger a skill, with a ([\d\.]+) second cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FoulbornKitavaTriggerChance", "BASE", num).Source("Kitava's Thirst"),
			MOD("FoulbornKitavaRequiredLifeCost", "BASE", utils.Float(captures[1])).Source("Kitava's Thirst"),
			MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportCastOnLifeSpent", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
		}, ""
	},
	// Socketed gem modifiers
	`([\+\-]\d+) to level of socketed gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("all"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	//pob:src/Modules/ModParser.lua:3179-3182
	`([\+\-]\d+) to level of socketed ([a-z\- ]+) gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr(captures[1]), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`\+(\d+)% to quality of socketed gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("all"), Key: "quality", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	//pob:src/Modules/ModParser.lua:3238-3239
	`gems socketed in red sockets have [\+\-](\d+) to level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("all"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}").SocketColor("R"))}, ""
	},
	`gems socketed in green sockets have [\+\-](\d+)% to quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("all"), Key: "quality", Value: num}).Tag(mod.SocketedIn("{SlotName}").SocketColor("G"))}, ""
	},
	//pob:src/Modules/ModParser.lua:3240-3242
	`\+(\d+)% to fire resistance when socketed with a red gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketProperty", "LIST", mod.SocketProperty{Mod: mod.NewFloat("FireResist", mod.TypeBase, num)}).Tag(mod.SocketedIn("{SlotName}").Keyword("strength").Sockets(1))}, ""
	},
	`\+(\d+)% to cold resistance when socketed with a green gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketProperty", "LIST", mod.SocketProperty{Mod: mod.NewFloat("ColdResist", mod.TypeBase, num)}).Tag(mod.SocketedIn("{SlotName}").Keyword("dexterity").Sockets(1))}, ""
	},
	`\+(\d+)% to lightning resistance when socketed with a blue gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketProperty", "LIST", mod.SocketProperty{Mod: mod.NewFloat("LightningResist", mod.TypeBase, num)}).Tag(mod.SocketedIn("{SlotName}").Keyword("intelligence").Sockets(1))}, ""
	},
	//pob:src/Modules/ModParser.lua:3244-3245
	`attack skills gain (\d+)% of physical damage as extra fire damage per socketed red gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketProperty", "LIST", mod.SocketProperty{Mod: MOD("PhysicalDamageGainAsFire", "BASE", num).Flag(mod.MFlagAttack)}).Tag(mod.SocketedIn("{SlotName}").Keyword("strength").Sockets(1, 2, 3, 4, 5, 6))}, ""
	},
	`(\d+)% of damage taken recouped as life per socketed red gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SocketProperty", "LIST", mod.SocketProperty{Mod: mod.NewFloat("LifeRecoup", mod.TypeBase, num)}).Tag(mod.SocketedIn("{SlotName}").Keyword("strength").Sockets(1, 2, 3, 4, 5, 6))}, ""
	},
	`([\+\-]\d+)% to quality of all skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("active_skill"), Key: "quality", Value: num})}, ""
	},
	//pob:src/Modules/ModParser.lua:3185
	`([\+\-]\d+) to level of all elemental skill gems if the stars are aligned`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"elemental", "grants_active_skill"}, Key: "level", Value: num}).Tag(mod.Condition("StarsAreAligned"))}, ""
	},
	//pob:src/Modules/ModParser.lua:3186
	`([\+\-]\d+) to level of all elemental support gems if the stars are aligned`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: []string{"elemental", "support"}, Key: "level", Value: num}).Tag(mod.Condition("StarsAreAligned"))}, ""
	},
	//pob:src/Modules/ModParser.lua:3188-3190
	`([\+\-]\d+) to level of all ([a-z]+) skill gems if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr(strings.ToLower(captures[1])), Key: "level", Value: num}).Tag(mod.MultiplierThreshold(firstToUpper(captures[3]) + "Item").Threshold(utils.Float(captures[2])))}, ""
	},
	//pob:src/Modules/ModParser.lua:3189
	`([\+\-]\d+) to level of all ([a-z]+) skill gems if a?t? ?l?e?a?s?t? ?(\d+) (\w+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr(strings.ToLower(captures[1])), Key: "level", Value: num}).Tag(mod.MultiplierThreshold(firstToUpper(captures[3]) + firstToUpper(captures[4]) + "Item").Threshold(utils.Float(captures[2])))}, ""
	},
	`([\+\-]\d+) to level of all ?([a-z\- ]*) support gems if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr(strings.ToLower(captures[1])), Key: "level", Value: num}).Tag(mod.MultiplierThreshold(firstToUpper(captures[3]) + "Item").Threshold(utils.Float(captures[2])))}, ""
	},
	`\+(\d+)% to quality of socketed ([a-z\- ]+) gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr(captures[1]), Key: "quality", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`\+(\d+) to level of active socketed skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("active_skill"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	`\+(\d+) to level of socketed active skill gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("active_skill"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	//pob:src/Modules/ModParser.lua:3187
	`([\+\-]\d+) to level of socketed active skill gems per (\d+) player levels`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("active_skill"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}")).Tag(mod.Multiplier("Level").Div(utils.Float(captures[1])))}, ""
	},
	//pob:src/Modules/ModParser.lua:3183-3191
	`([\+\-]\d+) to level of socketed skill gems per (\d+) player levels`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("active_skill"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}")).Tag(mod.Multiplier("Level").Div(utils.Float(captures[1])))}, ""
	},
	`([\+\-]\d+) to level of socketed skill gems per socketed gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr("active_skill"), Key: "level", Value: num}).Tag(mod.SocketedIn("{SlotName}")).Tag(mod.Multiplier("SocketedGemsIn{SlotName}"))}, ""
	},
	"socketed gems fire an additional projectile": []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("ProjectileCount", mod.TypeBase, 1)}).Tag(mod.SocketedIn("{SlotName}"))},
	`socketed gems fire (\d+) additional projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("ProjectileCount", mod.TypeBase, num)}).Tag(mod.SocketedIn("{SlotName}"))}, ""
	},
	"socketed gems reserve no mana":                                  []mod.Mod{MOD("ManaReserved", "MORE", -100).Tag(mod.SocketedIn("{SlotName}"))},
	"socketed gems have no reservation":                              []mod.Mod{MOD("Reserved", "MORE", -100).Tag(mod.SocketedIn("{SlotName}"))},
	"gems socketed always have the quality bonus from socket colour": []mod.Mod{mod.NewFlag("SocketAlwaysMatches", true)},
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
	"socketed non-channelling bow skills are triggered by snipe": []mod.Mod{},
	//pob:block:src/Modules/ModParser.lua:3214-3217
	`grants level (\d+) snipe skill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: "Snipe", Level: int(num)}),
			MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "ChannelledSnipeSupport", Level: int(num)}).Tag(mod.SocketedIn("{SlotName}")),
		}, ""
	},
	`socketed triggered bow skills deal (\d+)% less damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("Damage", "MORE", -num)}).Tag(mod.SocketedIn("{SlotName}").Keyword("bow"), mod.SkillType(string(data.SkillTypeTriggerable)))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:3219
	`socketed vaal skills require (\d+)% less souls per use`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SoulCost", "MORE", -num)}).Tag(mod.SocketedIn("{SlotName}"), mod.SkillType(string(data.SkillTypeVaal)))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:3220-3224
	"hits from socketed vaal skills ignore enemy monster resistances": []mod.Mod{
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: FLAG("IgnoreElementalResistances")}).Tag(mod.SocketedIn("{SlotName}"), mod.SkillType(string(data.SkillTypeVaal))),
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: FLAG("IgnoreChaosResistance")}).Tag(mod.SocketedIn("{SlotName}"), mod.SkillType(string(data.SkillTypeVaal))),
	},
	"hits from socketed vaal skills ignore enemy monster physical damage reduction": []mod.Mod{
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: FLAG("IgnoreEnemyPhysicalDamageReduction")}).Tag(mod.SocketedIn("{SlotName}"), mod.SkillType(string(data.SkillTypeVaal))),
	},
	//pob:block:src/Modules/ModParser.lua:3226
	"damage with hits from socketed vaal skills is lucky": []mod.Mod{
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: FLAG("LuckyHits")}).Tag(mod.SocketedIn("{SlotName}"), mod.SkillType(string(data.SkillTypeVaal))),
	},
	// Global gem modifiers
	//pob:block:src/Modules/ModParser.lua:3228-3237
	`([\+\-]\d+)%? to (\w+) of all (.+) gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		skill := strings.ToLower(captures[2])
		if _, known := gemLookup().skillIDForName[skill]; known {
			return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{Keyword: utils.Ptr(skill), Key: "level", Value: num})}, ""
		}

		return []mod.Mod{MOD("GemProperty", "LIST", mod.GemProperty{KeywordList: gemKeywordList(skill), Key: strings.ToLower(captures[1]), Value: num})}, ""
	},
	// Extra skill/support
	`grants (\D+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return grantedExtraSkill(captures[0], 1, false), ""
	},
	`grants level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return grantedExtraSkill(captures[1], int(num), false), ""
	},
	//pob:block:src/Modules/ModParser.lua:3277
	`grants level (\d+) (.+), which will be used by shaper memory`: func(num float64, captures []string) ([]mod.Mod, string) {
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
	`trigger level (\d+) (.+) when you attack with a non-vaal slam or strike skill near an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:2610
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you attack with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3290
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) on melee hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3302
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you take a critical strike from a unique enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3312
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you suppress spell damage from a unique enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3313
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you block damage from a unique enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3314
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) on taking a savage hit from a unique enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3315
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when a totem dies while a unique enemy is in your presence`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3316
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you use a travel skill while a unique enemy is in your presence`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3317
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you reach maximum rage while a unique enemy is in your presence`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3318
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when you reach low life while a unique enemy is in your presence`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3319
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when energy shield recharge starts while a unique enemy is in your presence`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3320
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) when your ward breaks`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3321
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) once every second`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3322
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`triggers level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3323
		return triggerExtraSkill(captures[1], int(num), false, ""), ""
	},
	`trigger level (\d+) (.+) on critical strike with cleave or reave`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3308
		return triggerExtraSkillOnCrit(captures[1], int(num), false), ""
	},
	`trigger level (\d+) (.+) on melee critical strike`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3309
		return triggerExtraSkillOnCrit(captures[1], int(num), false), ""
	},
	`trigger level (\d+) (.+) on critical strike against marked unique enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3310
		return triggerExtraSkillOnCrit(captures[1], int(num), false), ""
	},
	`trigger level (\d+) (.+) on critical strike`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3311
		return triggerExtraSkillOnCrit(captures[1], int(num), false), ""
	},
	`trigger level (\d+) (.+) on attack critical strike against a rare or unique enemy and y?o?u? ?have no mark`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3324
		return triggerExtraSkillOnCrit(captures[1], int(num), false), ""
	},
	`(.+) can trigger level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3286
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
	//pob:src/Modules/ModParser.lua:2993
	`triggers level (\d+) summon triggerbots when allocated`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("HaveTriggerBots", true)}, ""
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
	//pob:src/Modules/ModParser.lua:3334
	`(\d+)% chance to trigger level (\d+) (.+) when you or a nearby ally kill an enemy, or hit a rare or unique enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkill(captures[2], int(utils.Float(captures[1])), false, ""), num), ""
	},
	//pob:block:src/Modules/ModParser.lua:3335
	`(\d+)% chance to trigger (.+) when you kill an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkill(captures[1], 1, false, ""), num), ""
	},
	//pob:src/Modules/ModParser.lua:3333
	`(\d+)% chance to trigger level (\d+) (.+) on critical strike with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return withTriggerChance(triggerExtraSkillOnCrit(captures[2], int(utils.Float(captures[1])), false), num), ""
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
	`\d+% chance to curse n?o?n?-?c?u?r?s?e?d? ?enemies with (\D+) on \w+`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3352
		return []mod.Mod{
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: extraSkillIDForName(captures[0]), SkillName: captures[0], Level: 1, NoSupports: true, Triggered: true}),
		}, ""
	},
	`\d+% chance to curse n?o?n?-?c?u?r?s?e?d? ?enemies with (\D+) on \w+, with (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: extraSkillIDForName(captures[0]), SkillName: captures[0], Level: 1, NoSupports: true, Triggered: true}),
			MOD("CurseEffect", "INC", utils.Float(captures[1])).Tag(mod.SkillName(utils.CapitalEach(captures[0]))),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:3355
	`curse enemies with level (\d+) (\D+) on \w+, which can apply to hexproof enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:block:src/Modules/ModParser.lua:2064-2066
		return append(triggerExtraSkill(captures[1], int(num), true, ""),
			MOD("SkillData", "LIST", mod.SkillData{Key: "ignoreHexproof", Value: 1}).Tag(mod.SkillId(extraSkillIDForName(strings.ReplaceAll(strings.ToLower(captures[1]), " skill", ""))))), ""
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
	//pob:block:src/Modules/ModParser.lua:3366-3368
	"trigger commandment of inferno on critical strike": []mod.Mod{
		MOD("ExtraSkill", "LIST", mod.ExtraSkill{SkillID: "UniqueEnchantmentOfInfernoOnCrit", Level: 1, NoSupports: true, Triggered: true}),
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{
			Mod: MOD("SkillData", "LIST", mod.SkillData{Key: "triggerOnCrit", Value: 1}),
		}).Tag(mod.SkillId("UniqueEnchantmentOfInfernoOnCrit")),
	},
	//pob:src/Modules/ModParser.lua:3369
	"trigger (.+) on critical strike": func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkillOnCrit(captures[0], 1, true), ""
	},
	"triggers? (.+) when you take a critical strike": func(num float64, captures []string) ([]mod.Mod, string) {
		return triggerExtraSkill(captures[0], 1, true, ""), ""
	},
	`socketed [\w+]* ?gems a?r?e? ?supported by level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		entries := extraSupportIDsForName(captures[1])
		if len(entries) == 0 {
			return nil, ""
		}
		return extraSupportMods(entries, int(num), "{SlotName}"), ""
	},
	//pob:src/Modules/ModParser.lua:3372
	`socketed [\w+]* ?spells a?r?e? ?supported by level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		entries := extraSupportIDsForName(captures[1])
		if len(entries) == 0 {
			return nil, ""
		}
		return extraSupportMods(entries, int(num), "{SlotName}"), ""
	},
	//pob:src/Modules/ModParser.lua:3373
	`skills from equipped (.+) are supported by level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		entries := extraSupportIDsForName(captures[2])
		if len(entries) == 0 {
			return nil, ""
		}
		return extraSupportMods(entries, int(utils.Float(captures[1])), extraSupportSlotName(captures[0])), ""
	},
	//pob:src/Modules/ModParser.lua:3374
	`skills socketed in your (.+) are supported by level (\d+) (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		entries := extraSupportIDsForName(captures[2])
		if len(entries) == 0 {
			return nil, ""
		}
		return extraSupportMods(entries, int(utils.Float(captures[1])), extraSupportSlotName(captures[0])), ""
	},
	`socketed support gems can also support skills from y?o?u?r? ?e?q?u?i?p?p?e?d? ?([a-z\s]+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		targetItemSlotName := "Body Armour"
		if strings.EqualFold(captures[0], "main hand") {
			targetItemSlotName = "Weapon 1"
		}
		return []mod.Mod{
			MOD("LinkedSupport", "LIST", mod.LinkedSupport{TargetSlotName: targetItemSlotName}).Tag(mod.SocketedIn("{SlotName}")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:5276-5283
	`trigger a socketed elemental spell on block, with a ([\d\.]+) second cooldown`: []mod.Mod{
		MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerElementalSpellOnBlock", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
	},
	`trigger a socketed spell when you block, with a ([\d\.]+) second cooldown`: []mod.Mod{
		MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueVirulenceSpellsCastOnBlock", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
	},
	`trigger a socketed spell on unarmed melee critical strike, with a ([\d\.]+) second cooldown`: []mod.Mod{
		MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnUnarmedMeleeCriticalHit", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
	},
	//pob:src/Modules/ModParser.lua:5411-5413
	`trigger socketed curse spell when you cast a curse spell, with a ([\d\.]+) second cooldown`: []mod.Mod{
		MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportUniqueCastCurseOnCurse", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
	},
	//pob:src/Modules/ModParser.lua:3160
	`(\d+)% chance to trigger socketed spell on kill, with a ([\d\.]+) second cooldown`: []mod.Mod{
		MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportTriggerSpellOnKill", Level: 1}).Tag(mod.SocketedIn("{SlotName}")),
	},
	//pob:src/Modules/ModParser.lua:4304-4305
	"eat a soul when you hit a unique enemy, no more than once every second":                  []mod.Mod{FLAG("Condition:CanHaveSoulEater")},
	`eat a soul when you hit a rare or unique enemy, no more than once every [\d\.]+ seconds`: []mod.Mod{FLAG("Condition:CanHaveSoulEater")},
	//pob:src/Modules/ModParser.lua:4306-4308
	`(\d+)% chance to gain soul eater for (\d+) seconds on killing blow against rare and unique enemies with double strike or dual strike`: []mod.Mod{FLAG("Condition:CanHaveSoulEater")},
	`when (\d+)% of your hex's duration expires on an enemy, eat (\d+) soul per enemy power`:                                               []mod.Mod{FLAG("Condition:CanHaveSoulEater")},
	`eat (\d+) souls when you kill a rare or unique enemy with this weapon`:                                                                []mod.Mod{FLAG("Condition:CanHaveSoulEater")},
	//pob:src/Modules/ModParser.lua:5035
	`your hits intimidate enemies for (\d+) seconds while you are using pride`: []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).Tag(mod.Condition("AffectedByPride")),
	},
	//pob:src/Modules/ModParser.lua:5438
	"with a hypnotic eye jewel socketed, gain arcane surge on hit with spells": []mod.Mod{
		FLAG("Condition:ArcaneSurge").Tag(mod.Condition("HitSpellRecently")).Tag(mod.Condition("HaveHypnoticEyeJewelIn{SlotName}")),
	},
	"socketed hex curse skills are triggered by doedre's effigy when summoned": []mod.Mod{MOD("ExtraSupport", "LIST", mod.ExtraSupport{SkillID: "SupportCursePillarTriggerCurses", Level: 20}).Tag(mod.SocketedIn("{SlotName}"))},
	//pob:src/Modules/ModParser.lua:3383-3385
	`socketed projectile spells have \+([\d\.]+) seconds to cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "BASE", num).Tag(mod.SocketedIn("{SlotName}"), mod.SkillType(string(data.SkillTypeProjectile)), mod.SkillType(string(data.SkillTypeSpell)))}, ""
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
	"offering skills triggered this way also affect you": []mod.Mod{
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("SkillData", "LIST", mod.SkillData{Key: "buffNotPlayer", Value: 0})}).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering"))},
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
	"increases and reductions to minion damage also affects? you": []mod.Mod{mod.NewFlag("MinionDamageAppliesToPlayer", true)},
	`increases and reductions to minion damage also affects? you at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("MinionDamageAppliesToPlayer", true), MOD("ImprovedMinionDamageAppliesToPlayer", "MAX", num)}, ""
	},
	`increases and reductions to minion maximum life also apply to you at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("MinionLifeAppliesToPlayer", true), MOD("ImprovedMinionLifeAppliesToPlayer", "MAX", num)}, ""
	},
	//pob:src/Modules/ModParser.lua:3406
	"increases and reductions to minion attack speed also affects? you": []mod.Mod{mod.NewFlag("MinionAttackSpeedAppliesToPlayer", true)},
	`increases and reductions to cast speed apply to attack speed at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("CastSpeedAppliesToAttacks", true), MOD("ImprovedCastSpeedAppliesToAttacks", "MAX", num)}, ""
	},
	"increases and reductions to spell damage also apply to attacks": []mod.Mod{mod.NewFlag("SpellDamageAppliesToAttacks", true), MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", 100)},
	//pob:src/Modules/ModParser.lua:3412
	"increases and reductions to your evasion rating also apply to your spell damage": []mod.Mod{mod.NewFlag("EvasionAppliesToSpellDamage", true)},
	//pob:src/Modules/ModParser.lua:5471
	"rage grants spell damage instead of attack damage": []mod.Mod{mod.NewFlag("Condition:RageSpellDamage", true)},
	//pob:src/Modules/ModParser.lua:5273-5275
	`spells cost \+(\d+)% of life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeCostBase", "BASE", 1).Tag(mod.PercentStat("Life", num).Floor()).Tag(mod.SkillType(string(data.SkillTypeSpell))),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:3659-3663
	`(\d+)% increased damage with damaging ailments you inflict while you are affected by the same ailment`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "INC", num).KeywordFlag(mod.KeywordFlagBleed).Tag(mod.ActorCondition("enemy", "Bleeding")).Tag(mod.Condition("Bleeding")),
			MOD("Damage", "INC", num).KeywordFlag(mod.KeywordFlagIgnite).Tag(mod.ActorCondition("enemy", "Ignited")).Tag(mod.Condition("Ignited")),
			MOD("Damage", "INC", num).KeywordFlag(mod.KeywordFlagPoison).Tag(mod.ActorCondition("enemy", "Poisoned")).Tag(mod.Condition("Poisoned")),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:2436-2438
	`adds (\d+) to (\d+) fire damage to attacks for every (\d+)% your light radius is above base value`: func(num float64, captures []string) ([]mod.Mod, string) {
		div := utils.Float(captures[2])
		return []mod.Mod{
			MOD("FireMin", "BASE", num).Flag(mod.MFlagAttack).Tag(mod.PerStat(div, "LightRadiusInc")),
			MOD("FireMax", "BASE", utils.Float(captures[1])).Flag(mod.MFlagAttack).Tag(mod.PerStat(div, "LightRadiusInc")),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:2318-2320
	"you and enemies in your presence count as moving while affected by elemental ailments": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Moving").Tag(mod.Condition("Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped"))}),
		FLAG("Condition:Moving").Tag(mod.Condition("Frozen", "Chilled", "Shocked", "Ignited", "Scorched", "Brittle", "Sapped")),
	},
	`increases and reductions to spell damage also apply to attacks at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("SpellDamageAppliesToAttacks", true), MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", num)}, ""
	},
	"increases and reductions to spell damage also apply to attacks while wielding a wand": []mod.Mod{FLAG("SpellDamageAppliesToAttacks").Tag(mod.Condition("UsingWand"))},
	//pob:src/Modules/ModParser.lua:3423-3426
	`increases and reductions to spell damage also apply to attack damage with retaliation skills at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("SpellDamageAppliesToAttacks").Tag(mod.SkillType(string(data.SkillTypeRetaliation))),
			MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", num).Tag(mod.SkillType(string(data.SkillTypeRetaliation))),
		}, ""
	},
	`increases and reductions to maximum mana also apply to shock effect at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("ManaAppliesToShockEffect", true), MOD("ImprovedManaAppliesToShockEffect", "MAX", num)}, ""
	},
	`increases and reductions to ([a-z]+) damage also apply to effect of auras from ([a-z]+) skills at (\d+)% of their value, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		element, skillElement := firstToUpper(captures[0]), firstToUpper(captures[1])
		return []mod.Mod{
			mod.NewFlag(element+"DamageAppliesTo"+skillElement+"AuraEffect", true),
			MOD("Improved"+element+"DamageAppliesTo"+skillElement+"AuraEffect", "BASE", utils.Float(captures[2])),
			MOD(element+"DamageAppliesTo"+skillElement+"AuraEffectLimit", "MAX", utils.Float(captures[3])),
		}, ""
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
	`(\d+)% more maximum life if you have at least (\d+) life masteries allocated`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Life", "MORE", num).Tag(mod.MultiplierThreshold("AllocatedLifeMastery").Threshold(utils.Float(captures[1]))),
		}, ""
	},
	`modifiers to maximum (\w+) resistance also apply to maximum (\w+) and (\w+) resistances`: func(_ float64, captures []string) ([]mod.Mod, string) {
		from := capitaliseResistName(captures[0])
		return []mod.Mod{
			MOD(from+"MaxResConvertTo"+capitaliseResistName(captures[1]), "BASE", 100),
			MOD(from+"MaxResConvertTo"+capitaliseResistName(captures[2]), "BASE", 100),
		}, ""
	},
	`modifiers to (\w+) resistance also apply to (\w+) and (\w+) resistances at (\d+)% of their value`: func(_ float64, captures []string) ([]mod.Mod, string) {
		from := capitaliseResistName(captures[0])
		return []mod.Mod{
			MOD(from+"ResConvertTo"+capitaliseResistName(captures[1]), "BASE", utils.Float(captures[3])),
			MOD(from+"ResConvertTo"+capitaliseResistName(captures[2]), "BASE", utils.Float(captures[3])),
		}, ""
	},
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
	`gain (\d+)% of physical damage as extra damage of each element per spirit charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3467
		return []mod.Mod{
			MOD("PhysicalDamageGainAsLightning", "BASE", num).Tag(mod.Multiplier("SpiritCharge")),
			MOD("PhysicalDamageGainAsCold", "BASE", num).Tag(mod.Multiplier("SpiritCharge")),
			MOD("PhysicalDamageGainAsFire", "BASE", num).Tag(mod.Multiplier("SpiritCharge")),
		}, ""
	},
	`gain (\d+)% of physical damage as extra damage of each element if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3472
		threshold := mod.MultiplierThreshold(firstToUpper(captures[2]) + "Item").Threshold(utils.Float(captures[1]))
		return []mod.Mod{
			MOD("PhysicalDamageGainAsLightning", "BASE", num).Tag(threshold),
			MOD("PhysicalDamageGainAsCold", "BASE", num).Tag(threshold),
			MOD("PhysicalDamageGainAsFire", "BASE", num).Tag(threshold),
		}, ""
	},
	`gain (\d+)% of weapon physical damage as extra damage of an? r?a?n?d?o?m? ?element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", num).Flag(mod.MFlagWeapon)}, ""
	},
	`(\d+)% chance for hits to deal (\d+)% of physical damage as extra damage of a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		//pob:src/Modules/ModParser.lua:3479
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", num*utils.Float(captures[1])/100)}, ""
	},
	`gain (\d+)% of physical damage as extra damage of a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsRandom", "BASE", num)}, ""
	},
	`gain (\d+)% of physical damage as extra damage of a random element while you are ignited`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalDamageGainAsRandom", "BASE", num).Tag(mod.Condition("Ignited")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:3482
	`(\d+)% of physical damage from hits with this weapon is converted to a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageConvertToRandom", "BASE", num).Tag(mod.Condition("{Hand}Attack"))}, ""
	},
	//pob:src/Modules/ModParser.lua:3483
	`(\d+)% of physical damage converted to a random element`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageConvertToRandom", "BASE", num)}, ""
	},
	`nearby enemies convert (\d+)% of their ([a-z]+) damage to ([a-z]+)`:                     enemyDamageConvertRow(""),
	`enemies ignited by you have (\d+)% of ([a-z]+) damage they deal converted to ([a-z]+)`:  enemyDamageConvertRow("Ignited"),
	`enemies shocked by you have (\d+)% of ([a-z]+) damage they deal converted to ([a-z]+)`:  enemyDamageConvertRow("Shocked"),
	`enemies poisoned by you have (\d+)% of ([a-z]+) damage they deal converted to ([a-z]+)`: enemyDamageConvertRow("Poisoned"),
	"your critical strike chance is lucky":                                                   []mod.Mod{mod.NewFlag("CritChanceLucky", true)},
	"spell critical strike chance bifurcates":                                                []mod.Mod{FLAG("BifurcateCrit").Flag(mod.MFlagSpell)},
	"your critical strike chance is lucky while on low life":                                 []mod.Mod{FLAG("CritChanceLucky").Tag(mod.Condition("LowLife"))},
	"your critical strike chance is lucky while focus?sed":                                   []mod.Mod{FLAG("CritChanceLucky").Tag(mod.Condition("Focused"))},
	"your critical strikes do not deal extra damage":                                         []mod.Mod{mod.NewFlag("NoCritMultiplier", true)},
	"lightning damage with non-critical strikes is lucky":                                    []mod.Mod{mod.NewFlag("LightningNoCritLucky", true)},
	"your damage with critical strikes is lucky":                                             []mod.Mod{mod.NewFlag("CritLucky", true)},
	"critical strikes deal no damage":                                                        []mod.Mod{MOD("Damage", "MORE", -100).Tag(mod.Condition("CriticalStrike"))},
	//pob:block:src/Modules/ModParser.lua:3525-3527
	"critical strike chance is increased by uncapped lightning resistance":   []mod.Mod{FLAG("CritChanceIncreasedByUncappedLightningRes")},
	"critical strike chance is increased by lightning resistance":            []mod.Mod{FLAG("CritChanceIncreasedByLightningRes")},
	"critical strike chance is increased by overcapped lightning resistance": []mod.Mod{FLAG("CritChanceIncreasedByOvercappedLightningRes")},
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
	"treats enemy monster elemental resistance values as inverted": []mod.Mod{
		mod.NewFloat("HitsInvertEleResChance", mod.TypeChance, 100).Tag(mod.Condition("{Hand}Attack")),
	},
	"treats enemy monster chaos resistance values as inverted": []mod.Mod{
		mod.NewFloat("HitsInvertChaosResChance", mod.TypeChance, 100).Tag(mod.Condition("{Hand}Attack")),
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
	//pob:src/Modules/ModParser.lua:3028
	`maximum effect of shock is (\d+)% increased damage taken`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ShockMax", mod.TypeOverride, num)}, ""
	},
	//pob:src/Modules/ModParser.lua:3029
	`you can apply up to (\d+) shocks to each enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFlag("ShockCanStack", true),
			mod.NewFloat("ShockStacksMax", mod.TypeOverride, num),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:3031-3036
	"your hits always inflict freeze, shock and ignite while unbound": []mod.Mod{
		MOD("EnemyFreezeChance", "BASE", 100).Tag(mod.Condition("Unbound")),
		MOD("EnemyShockChance", "BASE", 100).Tag(mod.Condition("Unbound")),
		MOD("EnemyIgniteChance", "BASE", 100).Tag(mod.Condition("Unbound")),
	},
	`(\d+)% more elemental damage while unbound`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalDamage", "MORE", num).Tag(mod.Condition("Unbound"))}, ""
	},
	`your shocks can increase damage taken by up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ShockMax", mod.TypeOverride, num)}, ""
	},
	`\+(\d+)% to maximum effect of shock`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ShockMax", mod.TypeBase, num)}, ""
	},
	"your elemental damage can shock": []mod.Mod{mod.NewFlag("ColdCanShock", true), mod.NewFlag("FireCanShock", true)},
	"all y?o?u?r? ?damage can freeze": []mod.Mod{mod.NewFlag("PhysicalCanFreeze", true), mod.NewFlag("LightningCanFreeze", true), mod.NewFlag("FireCanFreeze", true), mod.NewFlag("ChaosCanFreeze", true)},
	"all damage with maces and sceptres inflicts chill": []mod.Mod{
		FLAG("PhysicalCanChill").Tag(mod.Condition("UsingMace")),
		FLAG("LightningCanChill").Tag(mod.Condition("UsingMace")),
		FLAG("FireCanChill").Tag(mod.Condition("UsingMace")),
		FLAG("ChaosCanChill").Tag(mod.Condition("UsingMace")),
	},
	"your cold damage can ignite":                         []mod.Mod{mod.NewFlag("ColdCanIgnite", true)},
	"your lightning damage can ignite":                    []mod.Mod{mod.NewFlag("LightningCanIgnite", true)},
	"your fire damage can shock but not ignite":           []mod.Mod{mod.NewFlag("FireCanShock", true), mod.NewFlag("FireCannotIgnite", true)},
	"your cold damage can ignite but not freeze or chill": []mod.Mod{mod.NewFlag("ColdCanIgnite", true), mod.NewFlag("ColdCannotFreeze", true), mod.NewFlag("ColdCannotChill", true)},
	"your cold damage cannot freeze":                      []mod.Mod{mod.NewFlag("ColdCannotFreeze", true)},
	"your lightning damage can freeze but not shock":      []mod.Mod{mod.NewFlag("LightningCanFreeze", true), mod.NewFlag("LightningCannotShock", true)},
	"your chaos damage can shock":                         []mod.Mod{mod.NewFlag("ChaosCanShock", true)},
	`your (\w+) damage can (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG(firstToUpper(strings.ToLower(captures[0])) + "Can" + firstToUpper(strings.ToLower(captures[1])))}, ""
	},
	`your (\w+) damage cannot (\w+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG(firstToUpper(strings.ToLower(captures[0])) + "Cannot" + firstToUpper(strings.ToLower(captures[1])))}, ""
	},
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
	//pob:src/Modules/ModParser.lua:3650
	`you can inflict up to (\d+) ignites on an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFlag("IgniteCanStack", true),
			MOD("IgniteStacks", "OVERRIDE", num),
		}, ""
	},
	`you can inflict an additional ignite on [ea][an]c?h? enemy`: []mod.Mod{mod.NewFlag("IgniteCanStack", true), mod.NewFloat("IgniteStacks", mod.TypeBase, 1)},
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
	//pob:src/Modules/ModParser.lua:3667-3673
	`base (\w+) duration is ([\d\.]+) seconds?`: func(num float64, captures []string) ([]mod.Mod, string) {
		ailmentName := ""
		switch ailment := strings.ToLower(captures[0]); ailment {
		case "bleeding", "bleed":
			ailmentName = "Bleed"
		case "ignite", "poison":
			ailmentName = firstToUpper(ailment)
		}
		if ailmentName == "" {
			return nil, ""
		}
		return []mod.Mod{MOD(ailmentName+"DurationBase", "OVERRIDE", utils.Float(captures[1]))}, ""
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
	`enemies ignited by you during f?l?a?s?k? ?effect take (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
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
	//pob:src/Modules/ModParser.lua:3707-3714
	`non-damaging elemental ailments you inflict have (\d+)% more effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("EnemyShockEffect", mod.TypeMore, num),
			mod.NewFloat("EnemyChillEffect", mod.TypeMore, num),
			mod.NewFloat("EnemyFreezeEffect", mod.TypeMore, num),
			mod.NewFloat("EnemyScorchEffect", mod.TypeMore, num),
			mod.NewFloat("EnemyBrittleEffect", mod.TypeMore, num),
			mod.NewFloat("EnemySapEffect", mod.TypeMore, num),
		}, ""
	},

	//pob:src/Modules/ModParser.lua:3715
	`immun[ei]t?y? to elemental ailments while on consecrated ground if you have at least (\d+) devotion`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("ElementalAilmentImmune").Tag(mod.Condition("OnConsecratedGround")).Tag(mod.StatThreshold("Devotion", num))}, ""
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
	//pob:src/Modules/ModParser.lua:3740-3747
	`while a unique enemy is in your presence, drops shocked ground while moving, lasting (\d+) seconds`: []mod.Mod{
		MOD("ShockBase", "BASE", data.NonDamagingAilments[data.AilmentShock].Default).Tag(mod.ActorCondition("enemy", "OnShockedGround"), mod.ActorCondition("enemy", "RareOrUnique")),
	},
	`while a unique enemy is in your presence, drops scorched ground while moving, lasting (\d+) seconds`: []mod.Mod{
		MOD("ScorchBase", "BASE", data.NonDamagingAilments[data.AilmentScorch].Default).Tag(mod.ActorCondition("enemy", "OnScorchedGround"), mod.ActorCondition("enemy", "RareOrUnique")),
	},
	`while a unique enemy is in your presence, drops brittle ground while moving, lasting (\d+) seconds`: []mod.Mod{
		MOD("BrittleBase", "BASE", data.NonDamagingAilments[data.AilmentBrittle].Default).Tag(mod.ActorCondition("enemy", "OnBrittleGround"), mod.ActorCondition("enemy", "RareOrUnique")),
	},
	`while a unique enemy is in your presence, drops sapped ground while moving, lasting (\d+) seconds`: []mod.Mod{
		MOD("SapBase", "BASE", data.NonDamagingAilments[data.AilmentSap].Default).Tag(mod.ActorCondition("enemy", "OnSappedGround"), mod.ActorCondition("enemy", "RareOrUnique")),
	},
	`while a pinnacle atlas boss is in your presence, drops shocked ground while moving, lasting (\d+) seconds`: []mod.Mod{
		MOD("ShockBase", "BASE", data.NonDamagingAilments[data.AilmentShock].Default).Tag(mod.ActorCondition("enemy", "OnShockedGround"), mod.ActorCondition("enemy", "PinnacleBoss")),
	},
	`while a pinnacle atlas boss is in your presence, drops scorched ground while moving, lasting (\d+) seconds`: []mod.Mod{
		MOD("ScorchBase", "BASE", data.NonDamagingAilments[data.AilmentScorch].Default).Tag(mod.ActorCondition("enemy", "OnScorchedGround"), mod.ActorCondition("enemy", "PinnacleBoss")),
	},
	`while a pinnacle atlas boss is in your presence, drops brittle ground while moving, lasting (\d+) seconds`: []mod.Mod{
		MOD("BrittleBase", "BASE", data.NonDamagingAilments[data.AilmentBrittle].Default).Tag(mod.ActorCondition("enemy", "OnBrittleGround"), mod.ActorCondition("enemy", "PinnacleBoss")),
	},
	`while a pinnacle atlas boss is in your presence, drops sapped ground while moving, lasting (\d+) seconds`: []mod.Mod{
		MOD("SapBase", "BASE", data.NonDamagingAilments[data.AilmentSap].Default).Tag(mod.ActorCondition("enemy", "OnSappedGround"), mod.ActorCondition("enemy", "PinnacleBoss")),
	},
	//pob:block:src/Modules/ModParser.lua:3749-3752
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
	"critical strikes do not inherently ignite":          []mod.Mod{mod.NewFlag("CritsDontAlwaysIgnite", true)},
	"always scorch while affected by anger":              []mod.Mod{MOD("EnemyScorchChance", "BASE", 100).Tag(mod.Condition("AffectedByAnger"))},
	"always inflict brittle while affected by hatred":    []mod.Mod{MOD("EnemyBrittleChance", "BASE", 100).Tag(mod.Condition("AffectedByHatred"))},
	"always sap while affected by wrath":                 []mod.Mod{MOD("EnemySapChance", "BASE", 100).Tag(mod.Condition("AffectedByWrath"))},
	"melee attacks cause bleeding":                       []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagMelee)},
	"attacks cause bleeding when hitting cursed enemies": []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagAttack).Tag(mod.ActorCondition("enemy", "Cursed"))},
	"melee critical strikes cause bleeding":              []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagMelee).Tag(mod.Condition("CriticalStrike"))},
	"causes bleeding on melee critical strike":           []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagMelee).Tag(mod.Condition("CriticalStrike"))},
	`melee critical strikes have (\d+)% chance to cause bleeding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("BleedChance", "BASE", num).Flag(mod.MFlagMelee).Tag(mod.Condition("CriticalStrike")),
		}, ""
	},
	"attacks always inflict bleeding while you have cat's stealth": []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagAttack).Tag(mod.Condition("AffectedByCat'sStealth"))},
	//pob:src/Modules/ModParser.lua:5242
	"you have iron reflexes while at maximum frenzy charges": []mod.Mod{
		MOD("Keystone", "LIST", "Iron Reflexes").Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax")),
	},
	//pob:src/Modules/ModParser.lua:4694
	"you have mind over matter while at maximum power charges": []mod.Mod{
		MOD("Keystone", "LIST", "Mind Over Matter").Tag(mod.StatThresholdStat("PowerCharges", "PowerChargesMax")),
	},
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
	//pob:src/Modules/ModParser.lua:3857
	`(\d+)% chance for blade vortex and blade blast to impale enemies on hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ImpaleChance", "BASE", num).Tag(mod.SkillName("Blade Vortex", "Blade Blast").Transfigured(true)),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:3858
	`critical strikes with spells inflict impale`: []mod.Mod{
		MOD("ImpaleChance", "BASE", float64(100)).Flag(mod.MFlagSpell).Tag(mod.Condition("CriticalStrike")),
	},
	//pob:src/Modules/ModParser.lua:3859
	`(\d+)% chance on hitting an enemy for all impales on that enemy to last for an additional hit`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ImpaleAdditionalDurationChance", "BASE", num),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:3862
	`projectiles gain impale effect as they travel farther, causing impales they inflict to have up to (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ImpaleEffect", "INC", num).Flag(mod.MFlagProjectile).Tag(mod.DistanceRamp([][]float64{{35, 0}, {70, 1}})),
		}, ""
	},
	// Poison and Bleed
	`(\d+)% increased damage with bleeding inflicted on poisoned enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "INC", num).KeywordFlag(mod.KeywordFlagBleed).Tag(mod.ActorCondition("enemy", "Poisoned")),
		}, ""
	},
	"y?o?u?r? ?fire damage can poison":      []mod.Mod{mod.NewFlag("FireCanPoison", true)},
	"y?o?u?r? ?cold damage can poison":      []mod.Mod{mod.NewFlag("ColdCanPoison", true)},
	"y?o?u?r? ?lightning damage can poison": []mod.Mod{mod.NewFlag("LightningCanPoison", true)},
	"all damage from hits can poison": []mod.Mod{
		mod.NewFlag("FireCanPoison", true),
		mod.NewFlag("ColdCanPoison", true),
		mod.NewFlag("LightningCanPoison", true),
	},
	"all damage can poison": []mod.Mod{
		mod.NewFlag("FireCanPoison", true),
		mod.NewFlag("ColdCanPoison", true),
		mod.NewFlag("LightningCanPoison", true),
	},
	"all damage with triggered spells can poison": []mod.Mod{
		FLAG("FireCanPoison").KeywordFlag(mod.KeywordFlagSpell).Tag(mod.SkillType(string(data.SkillTypeTriggered))),
		FLAG("ColdCanPoison").KeywordFlag(mod.KeywordFlagSpell).Tag(mod.SkillType(string(data.SkillTypeTriggered))),
		FLAG("LightningCanPoison").KeywordFlag(mod.KeywordFlagSpell).Tag(mod.SkillType(string(data.SkillTypeTriggered))),
	},
	"all damage from hits with this weapon can poison": []mod.Mod{
		FLAG("FireCanPoison").Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		FLAG("ColdCanPoison").Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		FLAG("LightningCanPoison").Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
	},
	"all damage inflicts poison while affected by glorious madness": []mod.Mod{
		MOD("PoisonChance", "BASE", 100).Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("FireCanPoison").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("ColdCanPoison").Tag(mod.Condition("AffectedByGloriousMadness")),
		FLAG("LightningCanPoison").Tag(mod.Condition("AffectedByGloriousMadness")),
	},
	"your chaos damage poisons enemies": []mod.Mod{mod.NewFloat("ChaosPoisonChance", mod.TypeBase, 100)},
	`your chaos damage has (\d+)% chance to poison enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ChaosPoisonChance", mod.TypeBase, num)}, ""
	},
	"melee attacks poison on hit":                 []mod.Mod{MOD("PoisonChance", "BASE", 100).Flag(mod.MFlagMelee)},
	"triggered spells poison on hit":              []mod.Mod{MOD("PoisonChance", "BASE", 100).KeywordFlag(mod.KeywordFlagSpell).Tag(mod.SkillType(string(data.SkillTypeTriggered)))},
	"always poison on hit against cursed enemies": []mod.Mod{MOD("PoisonChance", "BASE", 100).Tag(mod.ActorCondition("enemy", "Cursed"))},
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
	"your chance to suppressed spell damage is lucky":   []mod.Mod{mod.NewFlag("SpellSuppressionChanceIsLucky", true)},
	"your chance to suppressed spell damage is unlucky": []mod.Mod{mod.NewFlag("SpellSuppressionChanceIsUnlucky", true)},

	`y?o?u?r? ?chance to block is lucky`: []mod.Mod{
		mod.NewFlag("BlockChanceIsLucky", true), mod.NewFlag("ProjectileBlockChanceIsLucky", true),
		mod.NewFlag("SpellBlockChanceIsLucky", true), mod.NewFlag("SpellProjectileBlockChanceIsLucky", true),
	},
	`y?o?u?r? ?chance to block is unlucky`: []mod.Mod{
		mod.NewFlag("BlockChanceIsUnlucky", true), mod.NewFlag("ProjectileBlockChanceIsUnlucky", true),
		mod.NewFlag("SpellBlockChanceIsUnlucky", true), mod.NewFlag("SpellProjectileBlockChanceIsUnlucky", true),
	},
	`y?o?u?r? ?chance to block attack damage is lucky`: []mod.Mod{
		mod.NewFlag("BlockChanceIsLucky", true), mod.NewFlag("ProjectileBlockChanceIsLucky", true),
	},
	`y?o?u?r? ?chance to block attack damage is unlucky`: []mod.Mod{
		mod.NewFlag("BlockChanceIsUnlucky", true), mod.NewFlag("ProjectileBlockChanceIsUnlucky", true),
	},
	`y?o?u?r? ?chance to block spell damage is lucky`: []mod.Mod{
		mod.NewFlag("SpellBlockChanceIsLucky", true), mod.NewFlag("SpellProjectileBlockChanceIsLucky", true),
	},
	`y?o?u?r? ?chance to block spell damage is unlucky`: []mod.Mod{
		mod.NewFlag("SpellBlockChanceIsUnlucky", true), mod.NewFlag("SpellProjectileBlockChanceIsUnlucky", true),
	},
	`chance to block attack or spell damage is lucky if you've blocked recently`: []mod.Mod{
		FLAG("BlockChanceIsLucky").Tag(mod.Condition("BlockedRecently")),
		FLAG("ProjectileBlockChanceIsLucky").Tag(mod.Condition("BlockedRecently")),
		FLAG("SpellBlockChanceIsLucky").Tag(mod.Condition("BlockedRecently")),
		FLAG("SpellProjectileBlockChanceIsLucky").Tag(mod.Condition("BlockedRecently")),
	},
	"your lucky or unlucky effects use the best or worst from three rolls instead of two": []mod.Mod{mod.NewFlag("ExtremeLuck", true)},
	"your lucky or unlucky effects are instead unexciting":                                []mod.Mod{mod.NewFlag("Unexciting", true)},
	`prevent +(\+\d+)% of suppressed spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SpellSuppressionEffect", mod.TypeBase, num)}, ""
	},
	`prevent +(\+\d+)% of suppressed spell damage per bark below maximum`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SpellSuppressionEffect", mod.TypeBase, num).Tag(mod.Multiplier("MissingBarkskinStacks").Base(0))}, ""
	},
	`hits that would ignite instead scorch`: []mod.Mod{FLAG("IgniteCanScorch"), FLAG("CannotIgnite")},
	//pob:src/Modules/ModParser.lua:3027
	`you can inflict an additional scorch on each enemy`: []mod.Mod{FLAG("ScorchCanStack"), MOD("ScorchStacksMax", "BASE", 1)},
	//pob:block:src/Modules/ModParser.lua:3951-3955
	"inflict fire, cold and lightning exposure on enemies when you suppress their spell damage": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", -10)}).Tag(mod.Condition("Effective")).Tag(mod.Condition("SuppressedRecently")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ColdExposure", "BASE", -10)}).Tag(mod.Condition("Effective")).Tag(mod.Condition("SuppressedRecently")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("LightningExposure", "BASE", -10)}).Tag(mod.Condition("Effective")).Tag(mod.Condition("SuppressedRecently")),
	},
	//pob:block:src/Modules/ModParser.lua:3956
	"critical strike chance is increased by chance to suppress spell damage": []mod.Mod{FLAG("CritChanceIncreasedBySpellSuppressChance")},
	`you take (\d+)% reduced extra damage from suppressed critical strikes`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ReduceSuppressedCritExtraDamage", mod.TypeBase, num)}, ""
	},
	//pob:src/Modules/ModParser.lua:3958
	`\+(\d+)% chance to suppress spell damage if your e?q?u?i?p?p?e?d? ?boots, helmet and gloves have evasion`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SpellSuppressionChance", "BASE", num).
				Tag(mod.StatThreshold("EvasionOnBoots", 1)).
				Tag(mod.StatThreshold("EvasionOnHelmet", 1)).
				Tag(mod.StatThreshold("EvasionOnGloves", 1)),
		}, ""
	},
	`\+(\d+)% chance to suppress spell damage for each dagger you're wielding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SpellSuppressionChance", "BASE", num).Tag(mod.Condition("UsingDagger")),
			MOD("SpellSuppressionChance", "BASE", num).Tag(mod.Condition("DualWieldingDaggers")),
		}, ""
	},
	// Buffs/debuffs
	"phasing":   []mod.Mod{mod.NewFlag("Condition:Phasing", true)},
	"onslaught": []mod.Mod{mod.NewFlag("Condition:Onslaught", true)},
	//pob:src/Modules/ModParser.lua:3974-3975
	"rampage":    []mod.Mod{mod.NewFlag("Condition:Rampage", true)},
	"soul eater": []mod.Mod{mod.NewFlag("Condition:CanHaveSoulEater", true)},
	//pob:src/Modules/ModParser.lua:3976
	`g?a?i?n? ?unholy might during f?l?a?s?k? ?effect`: []mod.Mod{
		FLAG("Condition:UnholyMight").Tag(mod.Condition("UsingFlask")),
		FLAG("Condition:CanWither").Tag(mod.Condition("UsingFlask")),
	},
	"unholy might": []mod.Mod{
		mod.NewFlag("Condition:UnholyMight", true),
		mod.NewFlag("Condition:CanWither", true),
	},
	//pob:src/Modules/ModParser.lua:3977-3980
	"chaotic might":                        []mod.Mod{mod.NewFlag("Condition:ChaoticMight", true)},
	"elusive":                              []mod.Mod{mod.NewFlag("Condition:CanBeElusive", true)},
	"adrenaline":                           []mod.Mod{mod.NewFlag("Condition:Adrenaline", true)},
	"arcane surge":                         []mod.Mod{mod.NewFlag("Condition:ArcaneSurge", true)},
	"your aura buffs do not affect allies": []mod.Mod{mod.NewFlag("SelfAurasCannotAffectAllies", true)},
	"auras from your skills can only affect you": []mod.Mod{mod.NewFlag("SelfAurasOnlyAffectYou", true)},
	//pob:src/Modules/ModParser.lua:4328
	"non-curse auras from your skills only apply to you and linked targets": []mod.Mod{
		FLAG("SelfAurasAffectYouAndLinkedTarget").
			Tag(mod.SkillType(string(data.SkillTypeAura))).
			Tag(mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)),
	},
	//pob:src/Modules/ModParser.lua:3996
	"auras from your skills which affect allies also affect enemies": []mod.Mod{mod.NewFlag("AurasAffectEnemies", true)},
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
	//pob:block:src/Modules/ModParser.lua:4007-4009
	`(\d+)% increased area of effect per second you've been stationary, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AreaOfEffect", "INC", num).Tag(mod.Multiplier("StationarySeconds").Base(0).GlobalLimit(utils.Float(captures[1])).GlobalLimitKey("ExpansiveMight").LimitTotal(true)),
		}, ""
	},
	`(\d+)% increased chaos damage per (\d+) maximum mana, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ChaosDamage", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "Mana").GlobalLimit(utils.Float(captures[2])).GlobalLimitKey("DarkIdeation")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4460
	`minions have (\d+)% chance to impale on attack hit per socketed ghastly eye jewel`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ImpaleChance", "BASE", num)}).Tag(mod.Multiplier("GhastlyEyeJewelIn{SlotName}")),
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
	"your damage with hits is lucky while on low life":          []mod.Mod{FLAG("LuckyHits").Tag(mod.Condition("LowLife"))},
	"damage with hits is unlucky":                               []mod.Mod{FLAG("UnluckyHits")},
	"chaos damage with hits is lucky":                           []mod.Mod{FLAG("ChaosLuckyHits")},
	`lightning damage with hits is lucky if you['\s]h?a?ve blocked spell damage recently`: []mod.Mod{
		FLAG("LightningLuckHits").Tag(mod.Condition("BlockedSpellRecently")),
	},
	`cold damage with hits is lucky if you['\s]h?a?ve suppressed spell damage recently`: []mod.Mod{
		FLAG("ColdLuckyHits").Tag(mod.Condition("SuppressedRecently")),
	},
	`fire damage with hits is lucky if you['\s]h?a?ve blocked an attack recently`: []mod.Mod{
		FLAG("FireLuckyHits").Tag(mod.Condition("BlockedAttackRecently")),
	},
	"allies' aura buffs do not affect you": []mod.Mod{mod.NewFlag("AlliesAurasCannotAffectSelf", true)},
	`(\d+)% increased effect of non-curse auras from your skills on enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DebuffEffect", "INC", num).Tag(mod.SkillType(string(data.SkillTypeAura))).Tag(mod.SkillType(string(data.SkillTypeAppliesCurse)).Neg(true)),
			MOD("AuraEffect", "INC", num).Tag(mod.SkillName("Death Aura")),
		}, ""
	},
	"enemies can have 1 additional curse":                             []mod.Mod{mod.NewFloat("EnemyCurseLimit", mod.TypeBase, 1)},
	"you can apply an additional curse":                               []mod.Mod{mod.NewFloat("EnemyCurseLimit", mod.TypeBase, 1)},
	"you can apply an additional curse while affected by malevolence": []mod.Mod{MOD("EnemyCurseLimit", "BASE", 1).Tag(mod.Condition("AffectedByMalevolence"))},
	//pob:block:src/Modules/ModParser.lua:4045
	`you can apply an additional curse if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyCurseLimit", "BASE", 1).Tag(mod.MultiplierThreshold(firstToUpper(captures[1]) + "Item").Threshold(utils.Float(captures[0])))}, ""
	},
	"you can apply one fewer curse": []mod.Mod{MOD("EnemyCurseLimit", "BASE", -1)},
	"hits that fail to freeze due to insufficient freeze duration inflict hoarfrost": []mod.Mod{FLAG("HitsCanInflictHoarfrost")},
	"inflict barnacles on nearby enemies every second":                               []mod.Mod{FLAG("CanInflictBarnacles")},
	"your nearby party members maximum endurance charges is equal to yours":          []mod.Mod{FLAG("PartyMemberMaximumEnduranceChargesEqualToYours")},
	"your warcries inflict hallowing flame":                                          []mod.Mod{FLAG("Condition:CanInflictHallowingFlame").Tag(mod.Condition("UsedWarcryRecently"))},
	"attacks with this weapon inflict hallowing flame on hit":                        []mod.Mod{FLAG("Condition:CanInflictHallowingFlame")},
	"inflict hallowing flame on hit while on consecrated ground":                     []mod.Mod{FLAG("Condition:CanInflictHallowingFlame").Tag(mod.Condition("OnConsecratedGround"))},
	//pob:src/Modules/ModParser.lua:4334
	"inflict hallowing flame on melee hit":                                             []mod.Mod{FLAG("Condition:CanInflictHallowingFlame")},
	"your blink and mirror arrow clones use your gloves":                               []mod.Mod{FLAG("BlinkAndMirrorUseGloves")},
	"socketed vaal skills grant elusive when used":                                     []mod.Mod{FLAG("Condition:CanBeElusive")},
	"rage grants cast speed instead of attack speed":                                   []mod.Mod{FLAG("Condition:RageCastSpeed")},
	`gain a random shrine buff every (\d+) seconds`:                                    []mod.Mod{FLAG("Condition:CanHaveRegularShrines")},
	`gain a random shrine buff for (\d+) seconds when you kill a rare or unique enemy`: []mod.Mod{FLAG("Condition:CanHaveRegularShrines")},
	//pob:src/Modules/ModParser.lua:4539
	`projectiles pierce (\d+) additional targets if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PierceCount", "BASE", num).Tag(mod.MultiplierThreshold(firstToUpper(strings.ToLower(captures[2])) + "Item").Threshold(utils.Float(captures[1])))}, ""
	},
	//pob:src/Modules/ModParser.lua:4732-4734
	`([\d\.]+)% of physical damage prevented recently is regenerated as energy shield per second if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageMitigatedEnergyShieldPseudoRecoup", "BASE", num*4).Tag(mod.MultiplierThreshold(firstToUpper(strings.ToLower(captures[2])) + "Item").Threshold(utils.Float(captures[1])))}, ""
	},
	//pob:src/Modules/ModParser.lua:5570
	`(\d+)% reduced enemy chance to block sword attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("reduceEnemyBlock", "BASE", num).Flag(mod.MFlagSword)}, ""
	},
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
	//pob:src/Modules/ModParser.lua:4060-4065
	`inflict decay on enemies you curse with hex or mark skills, dealing (\d+) chaos damage per second for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "decay", Value: num, Merge: "MAX"}).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`inflict decay on enemies you curse with hex skills, dealing (\d+) chaos damage per second for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "decay", Value: num, Merge: "MAX"}).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
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
	"unaffected by temporal chains": []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.SkillName("Temporal Chains")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
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
		MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: FLAG("BuffAppliesToAllies")}).Tag(mod.SkillName("Aspect of the Avian")),
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
	`you are cursed with (\D+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraCurse", "LIST", mod.ExtraCurse{SkillName: captures[0], Level: 1, ApplyToPlayer: true})}, ""
	},
	`you are cursed with (\D+), with (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraCurse", "LIST", mod.ExtraCurse{SkillName: captures[0], Level: 1, ApplyToPlayer: true}),
			MOD("CurseEffectOnSelf", "INC", utils.Float(captures[1])).Tag(mod.SkillName(utils.CapitalEach(captures[0]))),
		}, ""
	},
	"you count as on low life while you are cursed with vulnerability":  []mod.Mod{FLAG("Condition:LowLife").Tag(mod.Condition("AffectedByVulnerability"))},
	"you count as on full life while you are cursed with vulnerability": []mod.Mod{FLAG("Condition:FullLife").Tag(mod.Condition("AffectedByVulnerability"))},
	`you count as on low (\w+) while at (\d+)% of maximum (\w+) or below`: func(_ float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Low"+titleFirst(captures[0])+"Percentage", "BASE", utils.Float(captures[1])/100)}, ""
	},
	`you count as on full (\w+) while at (\d+)% of maximum (\w+) or above`: func(_ float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Full"+titleFirst(captures[0])+"Percentage", "BASE", utils.Float(captures[1])/100)}, ""
	},
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
	//pob:src/Modules/ModParser.lua:4108
	`([\d\.]+)% of evasion rating is regenerated as life per second while focus?sed`: func(num float64, captures []string) ([]mod.Mod, string) {
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
		MOD("ElementalDamage", "MORE", 75).Tag(mod.Condition("Divinity")),
		MOD("ElementalDamageTaken", "MORE", -25).Tag(mod.Condition("Divinity")),
	},
	`minions deal (\d+)% increased damage if you have warcried recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "INC", num).Tag(mod.ActorCondition("parent", "UsedWarcryRecently"))})}, ""
	},
	`(\d+)% more damage with hits and ailments against enemies that are on low life while you are wielding an axe`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.ActorCondition("enemy", "LowLife")).Tag(mod.Condition("UsingAxe"))}, ""
	},
	`projectiles deal (\d+)% increased damage with hits and ailments for each time they have chained`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.PerStat(0, "Chain")).Tag(mod.SkillType(string(data.SkillTypeProjectile)))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:4587
	`projectiles deal (\d+)% increased damage with hits and ailments for each enemy pierced`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "INC", num).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.PerStat(0, "PiercedCount")).Tag(mod.SkillType(string(data.SkillTypeProjectile)))}, ""
	},
	`(\d+)% increased armour per second you've been stationary, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Armour", "INC", num).Tag(mod.Multiplier("StationarySeconds").Limit(utils.Float(captures[1]) / num)).Tag(mod.Condition("Stationary"))}, ""
	},
	`(\d+)% increased accuracy rating against marked enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", "INC", num).Tag(mod.ActorCondition("enemy", "Marked"))}, ""
	},
	`(\d+)% more accuracy rating against marked enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AccuracyVsEnemy", "MORE", num).Tag(mod.ActorCondition("enemy", "Marked"))}, ""
	},
	`deal (\d+)% more damage with hits and ailments to rare and unique enemies for each second they've ever been in your presence, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("EnemyPresenceSeconds").Actor("enemy").Limit(utils.Float(captures[1]))).Tag(mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	`deal (\d+)% more damage with hits and ailments to rare and unique enemies for every (\d+) seconds they've ever been in your presence, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", num).KeywordFlag(mod.KeywordFlagHit | mod.KeywordFlagAilment).Tag(mod.Multiplier("EnemyPresenceSeconds").Actor("enemy").Limit(utils.Float(captures[2])).Div(utils.Float(captures[1]))).Tag(mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	`(\d+)% chance to recover all life when you kill an enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.Condition("AverageResourceGain")),
			MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", 100)).Tag(mod.Condition("MaxResourceGain")),
		}, ""
	},
	`recover (\d+)% of life per withered debuff on each enemy you kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.Multiplier("WitheredStack").Actor("enemy").Limit(15))}, ""
	},
	`every (\d+) seconds, regenerate energy shield equal to (\d+)% of evasion rating over (\d+) second`: func(num float64, captures []string) ([]mod.Mod, string) {
		percent := utils.Float(captures[1])
		return []mod.Mod{
			MOD("EnergyShieldRegen", "BASE", 1).Tag(mod.Condition("LifeRegenBurstFull")).Tag(mod.PercentStat("Evasion", percent)),
			MOD("EnergyShieldRegen", "BASE", 1/num*utils.Float(captures[2])).Tag(mod.Condition("LifeRegenBurstAvg")).Tag(mod.PercentStat("Evasion", percent)),
		}, ""
	},
	"mark skills cost no mana": []mod.Mod{MOD("ManaCost", "MORE", -100).Tag(mod.SkillType(string(data.SkillTypeMark)))},
	"gain additional elemental damage reduction equal to half your chaos resistance": []mod.Mod{MOD("ElementalDamageReduction", "BASE", 1).Tag(mod.PerStat(2, "ChaosResist"))},
	`if diamond flask charges are consumed, (\d+)% increased critical strike chance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChance", "INC", num).Tag(mod.SkillType(string(data.SkillTypeTriggered)).Neg(true)).Tag(mod.SkillType(string(data.SkillTypeChannel)).Neg(true)).Tag(mod.Condition("HaveDiamondFlask"))}, ""
	},
	`if bismuth flask charges are consumed, penetrate (\d+)% elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ElementalPenetration", "BASE", num).Tag(mod.SkillType(string(data.SkillTypeTriggered)).Neg(true)).Tag(mod.SkillType(string(data.SkillTypeChannel)).Neg(true)).Tag(mod.Condition("HaveBismuthFlask"))}, ""
	},
	`if amethyst flask charges are consumed, (\d+)% of physical damage as extra chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PhysicalDamageGainAsChaos", "BASE", num).Tag(mod.SkillType(string(data.SkillTypeTriggered)).Neg(true)).Tag(mod.SkillType(string(data.SkillTypeChannel)).Neg(true)).Tag(mod.Condition("HaveAmethystFlask"))}, ""
	},
	`every (\d+) seconds, regenerate life equal to (\d+)% of armour and evasion rating over (\d+) second`: func(num float64, captures []string) ([]mod.Mod, string) {
		percent := utils.Float(captures[1])
		avg := 1 / num * utils.Float(captures[2])
		return []mod.Mod{
			MOD("LifeRegen", "BASE", 1).Tag(mod.Condition("LifeRegenBurstFull")).Tag(mod.PercentStat("Armour", percent)),
			MOD("LifeRegen", "BASE", avg).Tag(mod.Condition("LifeRegenBurstAvg")).Tag(mod.PercentStat("Armour", percent)),
			MOD("LifeRegen", "BASE", 1).Tag(mod.Condition("LifeRegenBurstFull")).Tag(mod.PercentStat("Evasion", percent)),
			MOD("LifeRegen", "BASE", avg).Tag(mod.Condition("LifeRegenBurstAvg")).Tag(mod.PercentStat("Evasion", percent)),
		}, ""
	},
	"inflict fire, cold, and lightning exposure on hit": []mod.Mod{
		MOD("FireExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
		MOD("ColdExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
		MOD("LightningExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
	},
	`enemies maimed by you take (\d+)% increased damage over time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("DamageTakenOverTime", "INC", num)}).Tag(mod.ActorCondition("enemy", "Maimed"))}, ""
	},
	`gain unholy might on block for (\d) seconds`: []mod.Mod{
		FLAG("Condition:UnholyMight").Tag(mod.Condition("BlockedRecently")),
		FLAG("Condition:CanWither").Tag(mod.Condition("BlockedRecently")),
	},
	`you and your minions take (\d+)% reduced reflected (\w+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		name := firstToUpper(captures[1]) + "ReflectedDamageTaken"
		return []mod.Mod{
			MOD(name, "INC", -num).Tag(mod.GlobalEffect("Global").Unscalable(true)),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD(name, "INC", -num).Tag(mod.GlobalEffect("Global").Unscalable(true))}),
		}, ""
	},
	`(\d+)% reduced reflected damage taken during effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReflectedDamageTaken", "INC", -num).Tag(mod.GlobalEffect("Global").Unscalable(true))}, ""
	},
	`you and your minions take (\d+)% reduced reflected damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ReflectedDamageTaken", "INC", -num).Tag(mod.GlobalEffect("Global").Unscalable(true)),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ReflectedDamageTaken", "INC", -num).Tag(mod.GlobalEffect("Global").Unscalable(true))}),
		}, ""
	},
	"culling strike against frozen enemies": []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.ActorCondition("enemy", "Frozen"))},
	`minions deal (\d+)% more damage while they are on low life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Damage", "MORE", num).Tag(mod.Condition("LowLife"))})}, ""
	},
	`your action speed is at least (\d+)% of base value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinimumActionSpeed", "MAX", num)}, ""
	},
	`retaliation skills have (\d+)% arcane might`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("SpellDamageAppliesToAttacks").Tag(mod.SkillType(string(data.SkillTypeRetaliation))),
			MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", num).Tag(mod.SkillType(string(data.SkillTypeRetaliation))),
		}, ""
	},
	"vaal attack skills cost rage instead of requiring souls to use":           []mod.Mod{FLAG("CostRageInsteadOfSouls").Flag(mod.MFlagAttack).Tag(mod.SkillType(string(data.SkillTypeVaal)))},
	"vaal attack skills you use yourself cost rage instead of requiring souls": []mod.Mod{FLAG("CostRageInsteadOfSouls").Flag(mod.MFlagAttack).Tag(mod.SkillType(string(data.SkillTypeVaal)))},
	`minions gain added physical damage equal to (\d+)% of maximum energy shield on your equipped helmet`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("PhysicalMin", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnHelmet", num).Actor("parent"))}),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("PhysicalMax", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnHelmet", num).Actor("parent"))}),
		}, ""
	},
	`(?:summoned )?skeletons gain added chaos damage equal to (\d+)% of maximum energy shield on your equipped shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnWeapon 2", num).Actor("parent"))}),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("EnergyShieldOnWeapon 2", num).Actor("parent"))}),
		}, ""
	},
	`every (\d+) seconds, regenerate (\d+)% of life over one second if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		percent := utils.Float(captures[1])
		threshold := mod.MultiplierThreshold(firstToUpper(captures[3]) + "Item").Threshold(utils.Float(captures[2]))
		return []mod.Mod{
			MOD("LifeRegenPercent", "BASE", percent).Tag(mod.Condition("LifeRegenBurstFull")).Tag(threshold),
			MOD("LifeRegenPercent", "BASE", percent/num).Tag(mod.Condition("LifeRegenBurstAvg")).Tag(mod.MultiplierThreshold(firstToUpper(captures[3]) + "Item").Threshold(utils.Float(captures[2]))),
		}, ""
	},
	`minion life is increased by t?h?e?i?r? ?overcapped fire resistance`: []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Life", "INC", 1).Tag(mod.PerStat(1, "FireResistOverCap"))})},
	`totem life is increased by t?h?e?i?r? ?overcapped fire resistance`:  []mod.Mod{MOD("TotemLife", "INC", 1).Tag(mod.PerStat(1, "TotemFireResistOverCap"))},
	`attacks you use yourself have (\d+)% more attack speed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Speed", "MORE", num).Flag(mod.MFlagAttack).
			Tag(mod.SkillType(string(data.SkillTypeSummonsTotem), string(data.SkillTypeRemoteMined), string(data.SkillTypeTrapped), string(data.SkillTypeTriggered)).Neg(true)).
			Tag(mod.Condition("usedByMirage").Neg(true))}, ""
	},
	"attacks you use yourself repeat an additional time": []mod.Mod{MOD("RepeatCount", "BASE", 1).Flag(mod.MFlagAttack).
		Tag(mod.SkillType(string(data.SkillTypeSummonsTotem), string(data.SkillTypeRemoteMined), string(data.SkillTypeTrapped), string(data.SkillTypeTriggered)).Neg(true)).
		Tag(mod.Condition("usedByMirage").Neg(true)).
		Tag(mod.Condition("averageRepeat", "alwaysFinalRepeat"))},
	`nearby enemies are covered in ash if you haven't moved in the past (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CoveredInAshEffect", "BASE", 20).Tag(mod.MultiplierThreshold("StationarySeconds").Threshold(num)).Tag(mod.Condition("Stationary"))}, ""
	},
	`your curses have (\d+)% increased effect if (\d+)% of curse duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CurseEffect", "INC", num).Tag(mod.MultiplierThreshold("CurseExpired").Actor("enemy").Threshold(utils.Float(captures[1]))).Tag(mod.SkillType(string(data.SkillTypeHex)))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:3985-3987
	`non-aura hexes expire upon reaching (\d+)% of base effect non-aura hexes gain (\d+)% increased effect per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CurseEffect", "INC", utils.Float(captures[1])).Tag(mod.Multiplier("CurseDurationExpired").Actor("enemy").Limit(num).LimitTotal(true)).
			Tag(mod.SkillType(string(data.SkillTypeAura)).Neg(true)).Tag(mod.SkillType(string(data.SkillTypeHex)))}, ""
	},
	//pob:src/Modules/ModParser.lua:3988-3990
	`enemies cursed by you have malediction if (\d+)% of curse duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMalediction").Tag(mod.MultiplierThreshold("CurseExpired").Threshold(num)).Tag(&mod.ActorConditionTag{TagType: mod.TypeActorCondition, VariableList: []string{"Cursed"}})})}, ""
	},
	`enemies cursed by you are hindered if (\d+)% of curse duration expired`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Hindered").Tag(mod.MultiplierThreshold("CurseExpired").Threshold(num)).Tag(&mod.ActorConditionTag{TagType: mod.TypeActorCondition, VariableList: []string{"Cursed"}})})}, ""
	},
	"your travel skills critically strike once every 3 uses": []mod.Mod{FLAG("Every3UseCrit").Tag(mod.SkillType(string(data.SkillTypeTravel)))},
	`all hits with your next non-channelling attack within (\d+) seconds of taking a critical strike will be critical strikes`: []mod.Mod{
		MOD("CritChance", "OVERRIDE", 100).Tag(mod.Condition("BeenCritRecently")).Tag(mod.SkillType(string(data.SkillTypeChannel)).Neg(true)).Tag(mod.SkillType(string(data.SkillTypeAttack))),
	},
	`vaal skills can store \+(\d+) uses?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AdditionalUses", "BASE", num).Tag(mod.SkillType(string(data.SkillTypeVaal)))}, ""
	},
	`stuns from critical strikes have (\d+)% increased duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyStunDurationOnCrit", "INC", num)}, ""
	},
	`prevent \+(\d+)% of suppressed spell damage while on full energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellSuppressionEffect", "BASE", num).Tag(mod.Condition("FullEnergyShield"))}, ""
	},
	"your maximum endurance charges is equal to your maximum frenzy charges": []mod.Mod{mod.NewFlag("MaximumEnduranceChargesIsMaximumFrenzyCharges", true)},
	"your maximum frenzy charges is equal to your maximum power charges":     []mod.Mod{mod.NewFlag("MaximumFrenzyChargesIsMaximumPowerCharges", true)},
	//pob:src/Modules/ModParser.lua:4120
	"your curse limit is equal to your maximum power charges": []mod.Mod{mod.NewFlag("CurseLimitIsMaximumPowerCharges", true)},
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
	//pob:block:src/Modules/ModParser.lua:4140
	`gain onslaught for (\d+) seconds on hit while at maximum frenzy charges`: []mod.Mod{
		FLAG("Onslaught").Tag(mod.StatThreshold("FrenzyCharges", 0).ThresholdStat("FrenzyChargesMax")).Tag(mod.Condition("HitRecently")),
	},
	`enemies in your chilling areas take (\d+)% increased lightning damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("LightningDamageTaken", mod.TypeIncrease, num)}).Tag(mod.ActorCondition("enemy", "InChillingArea")),
		}, ""
	},
	`(\d+)% chance to sap enemies in chilling areas`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemySapChance", "BASE", num).Tag(mod.ActorCondition("enemy", "InChillingArea"))}, ""
	},
	//pob:src/Modules/ModParser.lua:3796
	`drops burning ground while moving, dealing (\d+) fire damage per second for \d+ seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("DropsBurningGround", "BASE", num)}, ""
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
	"your warcries do not grant buffs or charges to you":                                                             []mod.Mod{mod.NewFlag("CannotGainWarcryBuffs", true)},
	`(\d+)% chance to inflict corrosion on hit with attacks`:                                                         []mod.Mod{mod.NewFlag("Condition:CanCorrode", true)},
	`(\d+)% chance to inflict withered for (\d+) seconds on hit`:                                                     []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	`(\d+)% chance to inflict withered for (\d+) seconds on hit with this weapon`:                                    []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	`(\d+)% chance to inflict withered for two seconds on hit if there are (\d+) or fewer withered debuffs on enemy`: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	`inflict withered for (\d+) seconds on hit with this weapon`:                                                     []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	//pob:block:src/Modules/ModParser.lua:4150-4157
	`melee weapon hits inflict (\d+) withered debuffs for (\d+) seconds`: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	`(\d+)% chance to inflict withered for (\d+) seconds on hit against cursed enemies`: []mod.Mod{
		FLAG("Condition:CanWither").Tag(mod.ActorCondition("enemy", "Cursed")),
	},
	`chaos skills inflict up to (\d+) withered debuffs on hit for (\d+) seconds`: []mod.Mod{mod.NewFlag("Condition:CanWither", true)},
	`enemies take (\d+)% increased elemental damage from your hits for each withered you have inflicted on them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("ElementalDamageTaken", "INC", num).Tag(mod.Multiplier("WitheredStack").Base(0).Limit(15))}),
		}, ""
	},
	"your hits cannot penetrate or ignore elemental resistances": []mod.Mod{mod.NewFlag("CannotElePenIgnore", true)},
	"nearby enemies have malediction":                            []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("HasMalediction", true)})},
	//pob:src/Modules/ModParser.lua:5085
	"profane ground you create inflicts malediction on enemies": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMalediction").Tag(mod.Condition("OnProfaneGround"))})},
	//pob:src/Modules/ModParser.lua:5086
	"profane ground you create also affects you and your allies, granting chaotic might": []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("Condition:ChaoticMight").Tag(mod.Condition("OnProfaneGround"))}),
	},
	`(\d+)% of damage from hits is taken from your sentinel of radiance's life before you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("takenFromRadianceSentinelBeforeYou", "BASE", num)}, ""
	},
	`(\d+)% of damage from hits is taken from your spectres' life before you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("takenFromSpectresBeforeYou", "BASE", num)}, ""
	},
	`(\d+)% of damage from hits is taken from your nearest totem's life before you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("takenFromTotemsBeforeYou", "BASE", num).Tag(mod.Condition("HaveTotem"))}, ""
	},
	`(\d+)% of damage from hits is taken from void spawns' life before you per void spawn`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("takenFromVoidSpawnBeforeYou", "BASE", num).Tag(mod.PerStat(1, "ActiveVoidSpawnLimit"))}, ""
	},
	"gain shaper's presence for 10 seconds when you kill a rare or unique enemy":             []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFlag("HasShapersPresence", true)}).Tag(mod.Condition("KilledUniqueEnemy"))},
	"gain maddening presence for 10 seconds when you kill a rare or unique enemy":            []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("HasMaddeningPresence", true)}).Tag(mod.Condition("KilledUniqueEnemy"))},
	"elemental damage you deal with hits is resisted by lowest elemental resistance instead": []mod.Mod{mod.NewFlag("ElementalDamageUsesLowestResistance", true)},
	`you take (\d+) chaos damage per second for 3 seconds on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosDegen", "BASE", num).Tag(mod.Condition("KilledLast3Seconds"))}, ""
	},
	`regenerate (\d+) life over 1 second for each spell you cast`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeRegen", "BASE", num).Tag(mod.Condition("CastLast1Seconds"))}, ""
	},
	`and nearby allies regenerate (\d+) life per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("LifeRegen", mod.TypeBase, num)}).
				Tag(mod.Condition("KilledPoisonedLast2Seconds")),
		}, ""
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
	//pob:src/Modules/ModParser.lua:4194
	`(\d+)% chance to inflict cold exposure on hit with cold damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ColdExposureChance", mod.TypeBase, num)}, ""
	},
	"socketed skills apply fire, cold and lightning exposure on hit": []mod.Mod{
		MOD("FireExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
		MOD("ColdExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
		MOD("LightningExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
	},
	//pob:block:src/Modules/ModParser.lua:4205-4213
	"inflict fire exposure on hit": []mod.Mod{
		MOD("FireExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
	},
	"inflict cold exposure on hit": []mod.Mod{
		MOD("ColdExposureChance", "BASE", 100).Tag(mod.Condition("Effective")),
	},
	"inflict lightning exposure on hit": []mod.Mod{
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
	//pob:src/Modules/ModParser.lua:4223-4225
	"nearby enemies have fire exposure while at maximum rage": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("FireExposure", "BASE", -10)}).Tag(mod.Condition("Effective")).Tag(mod.Condition("HaveMaximumRage")),
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
	//pob:block:src/Modules/ModParser.lua:4180-4190
	`regenerate (\d+)% of life per second for each different ailment affecting you`: func(num float64, captures []string) ([]mod.Mod, string) {
		out := make([]mod.Mod, 0, 9)
		for _, v := range []string{"Bleeding", "Ignited", "Scorched", "Chilled", "Frozen", "Brittle", "Shocked", "Sapped", "Poisoned"} {
			out = append(out, MOD("LifeRegenPercent", "BASE", num).Tag(mod.Condition(v)))
		}
		return out, ""
	},
	//pob:block:src/Modules/ModParser.lua:4263-4266
	`inflict fire exposure on hit against enemies with (\d+) cinderflame, applying -(\d+)% to (\w+) resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FireExposureChance", "BASE", 100).Tag(mod.Condition("Effective")).Tag(mod.MultiplierThreshold("CinderflameStacks").Threshold(num)),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD(utils.Capital(captures[2])+"Exposure", "BASE", -utils.Float(captures[1]))}).
				Tag(mod.Condition("Effective")).Tag(mod.MultiplierThreshold("CinderflameStacks").Threshold(num)),
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
	`(\d+)% increased effect of shrine buffs on you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ShrineBuffEffect", mod.TypeIncrease, num)}, ""
	},
	"left ring slot: cover enemies in ash for 5 seconds when you ignite them":    []mod.Mod{MOD("CoveredInAshEffect", "BASE", 20).Tag(mod.SlotNumber(1)).Tag(mod.ActorCondition("enemy", "Ignited"))},
	"right ring slot: cover enemies in frost for 5 seconds when you freeze them": []mod.Mod{MOD("CoveredInFrostEffect", "BASE", 20).Tag(mod.SlotNumber(2)).Tag(mod.ActorCondition("enemy", "Frozen"))},
	`([\w\s]+) has (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BuffEffect", "INC", utils.Float(captures[1])).Tag(mod.SkillId(extraSkillIDForName(captures[0])))}, ""
	},
	// This boot enchant gives a buff that applies both stats individually.
	//pob:block:src/Modules/ModParser.lua:4311-4314
	`(\d+)% increased attack and cast speed if you've killed recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Speed", "INC", num).Flag(mod.MFlagCast).Tag(mod.Condition("KilledRecently")),
			MOD("Speed", "INC", num).Flag(mod.MFlagAttack).Tag(mod.Condition("KilledRecently")),
		}, ""
	},
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
	`each mine applies (\d+)% reduced damage dealt to enemies near it, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: MOD("Damage", "INC", -num).Tag(mod.Multiplier("ActiveMineCount").Limit(utils.Float(captures[1]) / num))}),
		}, ""
	},
	`can have up to (\d+) additional traps? placed at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ActiveTrapLimit", mod.TypeBase, num)}, ""
	},
	`can have (\d+) fewer traps placed at a time`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ActiveTrapLimit", "BASE", -num)}, ""
	},
	`throw an additional trap`: []mod.Mod{mod.NewFloat("TrapThrowCount", mod.TypeBase, 1)},
	`(\d+)% chance to throw up to (\d+) additional traps?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("TrapThrowCount", mod.TypeBase, utils.Float(captures[1])*num/100)}, ""
	},
	`throw an additional mine`: []mod.Mod{mod.NewFloat("MineThrowCount", mod.TypeBase, 1)},
	`(\d+)% chance to throw up to (\d+) additional mines?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MineThrowCount", mod.TypeBase, utils.Float(captures[1])*num/100)}, ""
	},
	`(\d+)% chance to throw up to (\d+) additional traps? or mines?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("TrapThrowCount", mod.TypeBase, utils.Float(captures[1])*num/100),
			mod.NewFloat("MineThrowCount", mod.TypeBase, utils.Float(captures[1])*num/100),
		}, ""
	},
	`skills which throw traps throw up to (\d+) additional traps?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("TrapThrowCount", mod.TypeBase, num)}, ""
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
			MOD("ActiveBallistaLimit", "BASE", 1).Tag(mod.SkillName("Siege Ballista").Transfigured(true)).Tag(mod.PerStat(num, "Dex")),
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
	`minions gain added resistances equal to (\d+)% of your resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ResistanceAddedToMinions", mod.TypeBase, num)}, ""
	},
	`totems gain \+(\d+)% to all elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("TotemElementalResist", mod.TypeBase, num)}, ""
	},
	//pob:src/Modules/ModParser.lua:4372
	"your strength is added to your minions": []mod.Mod{MOD("StrengthAddedToMinions", "BASE", float64(100))},
	//pob:src/Modules/ModParser.lua:4373
	"half of your strength is added to your minions": []mod.Mod{MOD("StrengthAddedToMinions", "BASE", float64(50))},
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
	"s?u?m?m?o?n?e?d? ?raging spirits' hits always ignite": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("EnemyIgniteChance", mod.TypeBase, 100)}).Tag(mod.SkillName("Summon Raging Spirit").Transfigured(true))},
	"raised zombies have avatar of fire":                   []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Keystone", "LIST", "Avatar of Fire")}).Tag(mod.SkillName("Raise Zombie").Transfigured(true))},
	`raised zombies take ([\d\.]+)% of their maximum life per second as fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("FireDegen", "BASE", 1).Tag(mod.PercentStat("Life", num))}).Tag(mod.SkillName("Raise Zombie").Transfigured(true)),
		}, ""
	},
	"summoned skeletons have avatar of fire": []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("Keystone", "LIST", "Avatar of Fire")}).Tag(mod.SkillName("Summon Skeletons").Transfigured(true))},
	`summoned skeletons take ([\d\.]+)% of their maximum life per second as fire damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("FireDegen", "BASE", 1).Tag(mod.PercentStat("Life", num))}).Tag(mod.SkillName("Summon Skeletons").Transfigured(true))}, ""
	},
	`summoned skeletons have (\d+)% chance to wither enemies for (\d+) seconds on hit`: []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFlag("Condition:CanWither", true)}).Tag(mod.SkillName("Summon Skeletons").Transfigured(true))},
	`summoned skeletons have (\d+)% of physical damage converted to chaos damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("PhysicalDamageConvertToChaos", mod.TypeBase, num)}).Tag(mod.SkillName("Summon Skeletons").Transfigured(true)),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4433
	`minions convert (\d+)% of (.+) damage to (.+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat(minionConvertKey(captures[1], captures[2]), mod.TypeBase, num)}),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4448
	`minions convert (\d+)% of (.+) damage to (.+) damage per (.+) socket`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat(minionConvertKey(captures[1], captures[2]), mod.TypeBase, num)}).
				Tag(mod.Multiplier(utils.Capital(strings.ToLower(captures[3])) + "SocketIn{SlotName}").Base(0)),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4451
	`minions convert (\d+)% of (.+) damage to (.+) damage per socketed (\w+) gem`: func(num float64, captures []string) ([]mod.Mod, string) {
		colour := strings.ToLower(captures[3])
		switch colour {
		case "red", "green", "blue":
		default:
			return nil, ""
		}
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat(minionConvertKey(captures[1], captures[2]), mod.TypeBase, num)}).
				Tag(mod.Multiplier("Socketed" + utils.Capital(colour) + "GemsIn{SlotName}").Base(0)),
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
	//pob:src/Modules/ModParser.lua:4469
	`summoned skeleton warriors a?n?d? ?s?o?l?d?i?e?r?s? ?deal triple damage with this weapon if you've hit with this weapon recently`: []mod.Mod{
		MOD("Dummy", "DUMMY", 1).Tag(mod.Condition("HitRecentlyWithWeapon")),
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("TripleDamageChance", "BASE", 100).Tag(mod.ActorCondition("parent", "HitRecentlyWithWeapon"))}).Tag(mod.SkillName("Summon Skeletons"), mod.ItemCondition("Weapon 1").Name("The Iron Mass, Gladius")),
	},
	//pob:block:src/Modules/ModParser.lua:4472
	`summoned skeleton warriors a?n?d? ?s?o?l?d?i?e?r?s? ?wield a? ?c?o?p?y? ?o?f? ?this weapon while in your main hand`: []mod.Mod{}, // just make the mod blue, handled in CalcSetup
	"each summoned phantasm grants you phantasmal might":                                                                 []mod.Mod{mod.NewFlag("Condition:PhantasmalMight", true)},
	`minions have (\d+)% increased critical strike chance per maximum power charge you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("CritChance", "INC", num).Tag(mod.Multiplier("PowerChargeMax").Actor("parent"))})}, ""
	},
	"minions' base attack critical strike chance is equal to the critical strike chance of your main hand weapon": []mod.Mod{
		//pob:src/Modules/ModParser.lua:4475
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("AttackCritIsEqualToParentMainHand").Flag(mod.MFlagAttack)}),
	},
	`non-spectre minions' base attack time is equal to the attack time of your main hand weapon`: []mod.Mod{
		//pob:src/Modules/ModParser.lua:4476
		FLAG("NonSpectreMinionsUseParentMainHandAttackTime"),
	},
	//pob:block:src/Modules/ModParser.lua:4477-4480
	"minions can hear the whispers for 5 seconds after they deal a critical strike": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("Speed", "INC", 50).Tag(mod.GlobalEffect("Global").Unscalable(true)).Tag(mod.Condition("NeverCrit").Neg(true))}),
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("Damage", "INC", 50).Tag(mod.GlobalEffect("Global").Unscalable(true)).Tag(mod.Condition("NeverCrit").Neg(true))}),
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: MOD("ChaosDegen", "BASE", 1).Tag(mod.PercentStat("Life", 20)).Tag(mod.GlobalEffect("Global").Unscalable(true)).Tag(mod.Condition("NeverCrit").Neg(true))}),
		}, ""
	},
	"chaos damage t?a?k?e?n? ?does not bypass minions' energy shield": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFlag("ChaosNotBypassEnergyShield", true)})}, ""
	},
	"while minions have energy shield, their hits ignore monster elemental resistances": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("IgnoreElementalResistances").Tag(mod.StatThreshold("EnergyShield", 1))})}, ""
	},
	//pob:block:src/Modules/ModParser.lua:4484-4500
	`summoned arbalists' projectiles pierce (\d+) additional targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("PierceCount", "BASE", num)}).Tag(mod.SkillName("Summon Arbalists"))}, ""
	},
	"summoned arbalists' projectiles fork": []mod.Mod{
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("ForkOnce")}).Tag(mod.SkillName("Summon Arbalists")),
		MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ForkCountMax", "BASE", 1)}).Tag(mod.SkillName("Summon Arbalists")),
	},
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
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnemyFreezeChance", "BASE", num)}).Tag(mod.SkillName("Summon Arbalists")),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnemyShockChance", "BASE", num)}).Tag(mod.SkillName("Summon Arbalists")),
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnemyIgniteChance", "BASE", num)}).Tag(mod.SkillName("Summon Arbalists")),
		}, ""
	},
	`skills chain \+(\d) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ChainCountMax", mod.TypeBase, num)}, ""
	},
	"projectiles chain an additional time": []mod.Mod{MOD("ChainCountMax", "BASE", 1).Flag(mod.MFlagProjectile)},
	`projectiles chain \+(\d) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChainCountMax", "BASE", num).Flag(mod.MFlagProjectile)}, ""
	},
	`arrows chain \+(\d) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChainCountMax", "BASE", num).KeywordFlag(mod.KeywordFlagArrow)}, ""
	},
	"skills chain an additional time while at maximum frenzy charges": []mod.Mod{MOD("ChainCountMax", "BASE", 1).Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	"attacks chain an additional time when in main hand":              []mod.Mod{MOD("ChainCountMax", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.SlotNumber(1))},
	`projectiles chain \+(\d) times while you have phasing`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChainCountMax", "BASE", num).Flag(mod.MFlagProjectile).Tag(mod.Condition("Phasing"))}, ""
	},
	"adds an additional arrow": []mod.Mod{MOD("ProjectileCount", "BASE", 1).KeywordFlag(mod.KeywordFlagArrow)},
	`(\d+) additional arrows`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).KeywordFlag(mod.KeywordFlagArrow)}, ""
	},
	"bow attacks fire an additional arrow": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagBow)},
	`bow attacks fire (\d+) additional arrows`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).KeywordFlag(mod.KeywordFlagArrow)}, ""
	},
	`bow attacks fire (\d+) additional arrows if you haven't cast dash recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ProjectileCount", "BASE", num).KeywordFlag(mod.KeywordFlagArrow).Tag(mod.Condition("CastDashRecently").Neg(true))}, ""
	},
	"wand attacks fire an additional projectile": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagWand)},
	"skills fire an additional projectile":       []mod.Mod{mod.NewFloat("ProjectileCount", mod.TypeBase, 1)},
	//pob:block:src/Modules/ModParser.lua:4522
	`skills fire an additional projectile if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", 1).Tag(mod.MultiplierThreshold(firstToUpper(captures[1]) + "Item").Threshold(utils.Float(captures[0])))}, ""
	},
	"spells [hf][ai][vr]e an additional projectile":          []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagSpell)},
	"attacks fire an additional projectile":                  []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagAttack)},
	"attacks have an additional projectile when in off hand": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.SlotNumber(2))},
	"attacks fire an additional projectile when in off hand": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.SlotNumber(2))},
	`attacks [hf][ai][vr]e (\d+) additional projectiles? when in off hand`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).Flag(mod.MFlagAttack).Tag(mod.SlotNumber(2))}, ""
	},
	`attacks [hf][ai][vr]e (\d+) additional projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).Flag(mod.MFlagAttack)}, ""
	},
	`spells [hf][ai][vr]e (\d+) additional projectiles`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ProjectileCount", "BASE", num).Flag(mod.MFlagSpell)}, ""
	},
	`non-vaal strike skills target (\d+) additional nearby enem[yi]e?s?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("AdditionalStrikeTarget", "BASE", num).
				Tag(mod.SkillType(string(data.SkillTypeMeleeSingleTarget))).
				Tag(mod.SkillType(string(data.SkillTypeVaal)).Neg(true)),
		}, ""
	},
	"projectiles pierce an additional target": []mod.Mod{mod.NewFloat("PierceCount", mod.TypeBase, 1)},
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
	//pob:block:src/Modules/ModParser.lua:4542-4551
	"arrows pierce an additional target": []mod.Mod{MOD("PierceCount", "BASE", 1).KeywordFlag(mod.KeywordFlagArrow)},
	`arrows pierce (\d+) additional targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PierceCount", "BASE", num).KeywordFlag(mod.KeywordFlagArrow)}, ""
	},
	"arrows pierce one target": []mod.Mod{MOD("PierceCount", "BASE", 1).KeywordFlag(mod.KeywordFlagArrow)},
	`arrows pierce (\d+) targets?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PierceCount", "BASE", num).KeywordFlag(mod.KeywordFlagArrow)}, ""
	},
	"always pierce with arrows":         []mod.Mod{FLAG("PierceAllTargets").KeywordFlag(mod.KeywordFlagArrow)},
	"arrows always pierce":              []mod.Mod{FLAG("PierceAllTargets").KeywordFlag(mod.KeywordFlagArrow)},
	"arrows pierce all targets":         []mod.Mod{FLAG("PierceAllTargets").KeywordFlag(mod.KeywordFlagArrow)},
	"arrows that pierce cause bleeding": []mod.Mod{MOD("BleedChance", "BASE", 100).Flag(mod.MFlagProjectile).KeywordFlag(mod.KeywordFlagArrow).Tag(mod.StatThreshold("PierceCount", 1))},
	`arrows that pierce have (\d+)% chance to [ic][na][fu][ls][ie]c?t? bleeding`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("BleedChance", "BASE", num).Flag(mod.MFlagProjectile).KeywordFlag(mod.KeywordFlagArrow).Tag(mod.StatThreshold("PierceCount", 1)),
		}, ""
	},
	`arrows that pierce deal (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "INC", num).Flag(mod.MFlagProjectile).KeywordFlag(mod.KeywordFlagArrow).Tag(mod.StatThreshold("PierceCount", 1)),
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
	"summon skitterbots also summons a scorching skitterbot":                                  []mod.Mod{FLAG("ScorchingSkitterbot")},
	"summoned skitterbots' auras affect you as well as enemies":                               []mod.Mod{FLAG("SkitterbotAffectPlayer")},
	"projectiles from spells cannot pierce":                                                   []mod.Mod{FLAG("CannotPierce").Flag(mod.MFlagSpell)},
	"projectiles fork":                                                                        []mod.Mod{FLAG("ForkOnce").Flag(mod.MFlagProjectile), MOD("ForkCountMax", "BASE", 1).Flag(mod.MFlagProjectile)},
	//pob:block:src/Modules/ModParser.lua:4566-4571
	`(\d+)% increased critical strike chance with arrows that fork`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "INC", num).KeywordFlag(mod.KeywordFlagArrow).Tag(mod.StatThreshold("ForkRemaining", 1)).Tag(mod.StatThreshold("PierceCount", 0).Upper(true)),
		}, ""
	},
	`arrows that pierce have \+(\d+)% to critical strike multiplier`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritMultiplier", "BASE", num).KeywordFlag(mod.KeywordFlagArrow).Tag(mod.StatThreshold("PierceCount", 1)),
		}, ""
	},
	"arrows pierce all targets after forking": []mod.Mod{FLAG("PierceAllTargets").KeywordFlag(mod.KeywordFlagArrow).Tag(mod.StatThreshold("ForkedCount", 1))},
	//pob:src/Modules/ModParser.lua:4572
	"modifiers to number of projectiles instead apply to the number of targets projectiles split towards": []mod.Mod{
		mod.NewFlag("NoAdditionalProjectiles", true),
		mod.NewFlag("AdditionalProjectilesAddSplitsInstead", true),
	},
	"fire at most 1 projectile": []mod.Mod{FLAG("SingleProjectile")},
	"attack skills fire an additional projectile while wielding a claw or dagger": []mod.Mod{MOD("ProjectileCount", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.ModFlagOr(mod.MFlagClaw | mod.MFlagDagger))},
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
			MOD("Damage", "INC", num).Flag(mod.MFlagHit).KeywordFlag(mod.KeywordFlagArrow).Tag(mod.DistanceRamp([][]float64{{35, 0}, {70, 1}})),
		}, ""
	},
	`arrows gain critical strike chance as they travel farther, up to (\d+)% increased critical strike chance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "INC", num).KeywordFlag(mod.KeywordFlagArrow).Tag(mod.DistanceRamp([][]float64{{35, 0}, {70, 1}})),
		}, ""
	},
	"cannot leech life":                       []mod.Mod{mod.NewFlag("CannotLeechLife", true)},
	"cannot leech mana":                       []mod.Mod{mod.NewFlag("CannotLeechMana", true)},
	"cannot leech when on low life":           []mod.Mod{FLAG("CannotLeechLife").Tag(mod.Condition("LowLife")), FLAG("CannotLeechMana").Tag(mod.Condition("LowLife"))},
	"cannot leech life from critical strikes": []mod.Mod{FLAG("CannotLeechLife").Tag(mod.Condition("CriticalStrike"))},
	//pob:src/Modules/ModParser.lua:4600
	"leech applies instantly on critical strike": []mod.Mod{
		mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Tag(mod.Condition("CriticalStrike")),
		mod.NewFloat("InstantManaLeech", mod.TypeBase, 100).Tag(mod.Condition("CriticalStrike")),
		mod.NewFloat("InstantEnergyShieldLeech", mod.TypeBase, 100).Tag(mod.Condition("CriticalStrike")),
	},
	//pob:src/Modules/ModParser.lua:4605
	"gain life and mana from leech instantly on critical strike": []mod.Mod{
		mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Tag(mod.Condition("CriticalStrike")),
		mod.NewFloat("InstantManaLeech", mod.TypeBase, 100).Tag(mod.Condition("CriticalStrike")),
		mod.NewFloat("InstantEnergyShieldLeech", mod.TypeBase, 100).Tag(mod.Condition("CriticalStrike")),
	},
	//pob:src/Modules/ModParser.lua:4610
	"leech applies instantly during f?l?a?s?k? ?effect": []mod.Mod{
		mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Tag(mod.Condition("UsingFlask")),
		mod.NewFloat("InstantManaLeech", mod.TypeBase, 100).Tag(mod.Condition("UsingFlask")),
		mod.NewFloat("InstantEnergyShieldLeech", mod.TypeBase, 100).Tag(mod.Condition("UsingFlask")),
	},
	//pob:src/Modules/ModParser.lua:4615
	"gain life and mana from leech instantly during f?l?a?s?k? ?effect": []mod.Mod{
		mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Tag(mod.Condition("UsingFlask")),
		mod.NewFloat("InstantManaLeech", mod.TypeBase, 100).Tag(mod.Condition("UsingFlask")),
	},
	//pob:src/Modules/ModParser.lua:4619
	"life and mana leech are instant during f?l?a?s?k? ?effect": []mod.Mod{
		mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Tag(mod.Condition("UsingFlask")),
		mod.NewFloat("InstantManaLeech", mod.TypeBase, 100).Tag(mod.Condition("UsingFlask")),
	},
	"life and mana leech from critical strikes are instant":                                   []mod.Mod{mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Tag(mod.Condition("CriticalStrike")), mod.NewFloat("InstantManaLeech", mod.TypeBase, 100).Tag(mod.Condition("CriticalStrike"))},
	"gain life and mana from leech instantly during effect":                                   []mod.Mod{mod.NewFloat("InstantLifeLeech", mod.TypeBase, 100).Tag(mod.Condition("UsingFlask")), mod.NewFloat("InstantManaLeech", mod.TypeBase, 100).Tag(mod.Condition("UsingFlask"))},
	"with 5 corrupted items equipped: life leech recovers based on your chaos damage instead": []mod.Mod{FLAG("LifeLeechBasedOnChaosDamage").Tag(mod.MultiplierThreshold("CorruptedItem").Threshold(5))},
	//pob:src/Modules/ModParser.lua:2429-2431
	`you have everlasting sacrifice if at least (\d+) corrupted items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Condition:EverlastingSacrifice").Tag(mod.MultiplierThreshold("CorruptedItem").Threshold(num))}, ""
	},
	//pob:src/Modules/ModParser.lua:4277
	`(\d+)% chance to gain a brine charge instead of an endurance charge`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("CanGainBrineCharges"), MOD("BrineChargeGainChance", "BASE", num)}, ""
	},
	//pob:src/Modules/ModParser.lua:3083
	"drop brine ground while moving, lasting 4 seconds": []mod.Mod{FLAG("CanCreateBrineGround")},
	// Mahuxotl's Machination Steel Kite Shield
	//pob:src/Modules/ModParser.lua:3249-3251
	"everlasting sacrifice": []mod.Mod{FLAG("Condition:EverlastingSacrifice")},
	"you have vaal pact if you've dealt a critical strike recently": []mod.Mod{MOD("Keystone", "LIST", "Vaal Pact").Tag(mod.Condition("CritRecently"))},
	`gain (\d+) energy shield for each enemy you hit which is affected by a spider's web`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnHit", "BASE", num).Flag(mod.MFlagHit).Tag(mod.MultiplierThreshold("Spider's WebStack").Actor("enemy").Threshold(1)),
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
	"chaos damage t?a?k?e?n? ?does not bypass energy shield": []mod.Mod{mod.NewFlag("ChaosNotBypassEnergyShield", true)},
	`(\d+)% of chaos damage t?a?k?e?n? ?does not bypass energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ChaosEnergyShieldBypass", "BASE", -num)}, ""
	},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life":             []mod.Mod{FLAG("ChaosNotBypassEnergyShield").Tag(mod.Condition("LowLife").Neg(true))},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low life or low mana": []mod.Mod{FLAG("ChaosNotBypassEnergyShield").Tag(mod.Condition("LowLife", "LowMana").Neg(true))},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield while not on low mana":             []mod.Mod{FLAG("ChaosNotBypassEnergyShield").Tag(mod.Condition("LowMana").Neg(true))},
	"chaos damage is taken from mana before life": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ChaosDamageTakenFromManaBeforeLife", mod.TypeBase, 100)}, ""
	},
	"cannot evade enemy attacks":                   []mod.Mod{mod.NewFlag("CannotEvade", true)},
	"cannot block":                                 []mod.Mod{mod.NewFlag("CannotBlockAttacks", true), mod.NewFlag("CannotBlockSpells", true)},
	"cannot block while you have no energy shield": []mod.Mod{FLAG("CannotBlockAttacks").Tag(mod.Condition("HaveEnergyShield").Neg(true)), FLAG("CannotBlockSpells").Tag(mod.Condition("HaveEnergyShield").Neg(true))},
	"cannot block attacks":                         []mod.Mod{mod.NewFlag("CannotBlockAttacks", true)},
	"cannot block spells":                          []mod.Mod{mod.NewFlag("CannotBlockSpells", true)},
	"damage t?a?k?e?n? from blocked hits cannot bypass energy shield":     []mod.Mod{FLAG("BlockedDamageDoesntBypassES").Tag(mod.Condition("EVBypass").Neg(true))},
	"damage t?a?k?e?n? from unblocked hits always bypasses energy shield": []mod.Mod{FLAG("UnblockedDamageDoesBypassES").Tag(mod.Condition("EVBypass").Neg(true))},
	`recover (\d+) life when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("LifeOnBlock", mod.TypeBase, num)}, ""
	},
	`recover (\d+) energy shield when you block spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("EnergyShieldOnSpellBlock", mod.TypeBase, num)}, ""
	},
	`recover (\d+) energy shield when you suppress spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("EnergyShieldOnSuppress", mod.TypeBase, num)}, ""
	},
	`recover (\d+) life when you suppress spell damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("LifeOnSuppress", mod.TypeBase, num)}, ""
	},
	`recover (\d+)% of life when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnBlock", "BASE", 1).Tag(mod.PercentStat("Life", num))}, ""
	},
	`recover (\d+)% of life when you block attack damage while wielding a staff`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnBlock", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.Condition("UsingStaff")),
		}, ""
	},
	`recover (\d+)% of your maximum mana when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ManaOnBlock", "BASE", 1).Tag(mod.PercentStat("Mana", num)),
		}, ""
	},
	`recover (\d+)% of energy shield when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnBlock", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)),
		}, ""
	},
	`recover (\d+)% of energy shield when you block spell damage while wielding a staff`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnSpellBlock", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)).Tag(mod.Condition("UsingStaff")),
		}, ""
	},
	`replenishes energy shield by (\d+)% of armour when you block`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnBlock", "BASE", 1).Tag(mod.PercentStat("Armour", num)),
		}, ""
	},
	"cannot leech or regenerate mana":                                 []mod.Mod{mod.NewFlag("NoManaRegen", true), mod.NewFlag("CannotLeechMana", true)},
	"right ring slot: you cannot regenerate mana":                     []mod.Mod{FLAG("NoManaRegen").Tag(mod.SlotNumber(2))},
	"y?o?u? ?cannot recharge energy shield":                           []mod.Mod{mod.NewFlag("NoEnergyShieldRecharge", true)},
	"you cannot regenerate energy shield":                             []mod.Mod{mod.NewFlag("NoEnergyShieldRegen", true)},
	"cannot recharge or regenerate energy shield":                     []mod.Mod{mod.NewFlag("NoEnergyShieldRecharge", true), mod.NewFlag("NoEnergyShieldRegen", true)},
	"left ring slot: you cannot recharge or regenerate energy shield": []mod.Mod{FLAG("NoEnergyShieldRecharge").Tag(mod.SlotNumber(1)), FLAG("NoEnergyShieldRegen").Tag(mod.SlotNumber(1))},
	//pob:src/Modules/ModParser.lua:4741-4743
	"cannot gain energy shield": []mod.Mod{mod.NewFlag("CannotGainEnergyShield", true)},
	//pob:src/Modules/ModParser.lua:4744-4747
	"cannot recover life other than from leech": []mod.Mod{
		mod.NewFlag("CannotRecoverLifeOutsideLeech", true),
		mod.NewFlag("NoLifeRegen", true),
	},
	//pob:src/Modules/ModParser.lua:4748-4750
	`cannot gain energy shield during f?l?a?s?k? ?effect`: []mod.Mod{FLAG("CannotGainEnergyShield").Tag(mod.Condition("UsingFlask"))},
	`cannot gain life during f?l?a?s?k? ?effect`:          []mod.Mod{FLAG("CannotGainLife").Tag(mod.Condition("UsingFlask"))},
	`cannot gain mana during f?l?a?s?k? ?effect`:          []mod.Mod{FLAG("CannotGainMana").Tag(mod.Condition("UsingFlask"))},
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
	//pob:block:src/Modules/ModParser.lua:4775-4778
	"physical damage reduction is zero": []mod.Mod{
		mod.NewFloat("PhysicalDamageReduction", mod.TypeOverride, 0),
		mod.NewFlag("ArmourDoesNotApplyToPhysicalDamageTaken", true),
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
	`nearby enemies have (\d+)% increased fire and cold resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("FireResist", mod.TypeIncrease, num)}),
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("ColdResist", mod.TypeIncrease, num)}),
		}, ""
	},
	"nearby enemies are blinded while physical aegis is not depleted": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Blinded", true)}).Tag(mod.Condition("PhysicalAegisDepleted").Neg(true))},
	`y?o?u?r? ?maximum energy shield is equal to (\d+)% of y?o?u?r? ?maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShield", mod.TypeOverride, 1).Tag(mod.PercentStat("Life", num))}, ""
	},
	//pob:src/Modules/ModParser.lua:5872
	`mana is increased by (\d+)% of overcapped lightning resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("ManaIncreasedByOvercappedLightningRes"),
			MOD("Mana", "INC", num/100).Tag(mod.PerStat(1, "LightningResistOverCap")),
		}, ""
	},
	"maximum energy shield is increased by chance to block spell damage":            []mod.Mod{FLAG("EnergyShieldIncreasedByChanceToBlockSpellDamage")},
	"maximum energy shield is increased by chaos resistance":                        []mod.Mod{FLAG("EnergyShieldIncreasedByChaosResistance")},
	"armour is increased by uncapped fire resistance":                               []mod.Mod{FLAG("ArmourIncreasedByUncappedFireRes")},
	"armour is increased by overcapped fire resistance":                             []mod.Mod{FLAG("ArmourIncreasedByOvercappedFireRes")},
	"evasion rating is increased by uncapped cold resistance":                       []mod.Mod{FLAG("EvasionRatingIncreasedByUncappedColdRes")},
	"evasion rating is increased by overcapped cold resistance":                     []mod.Mod{FLAG("EvasionRatingIncreasedByOvercappedColdRes")},
	"gain maximum life instead of maximum energy shield from equipped armour items": []mod.Mod{mod.NewFlag("ConvertArmourESToLife", true)},
	//pob:block:src/Modules/ModParser.lua:3054-3055
	"can't use amulets": []mod.Mod{MOD("CanNotUseItem", mod.TypeDisablerFlag, 1).Tag(mod.DisablesItem("Amulet"))},
	"can't use belts":   []mod.Mod{MOD("CanNotUseItem", mod.TypeDisablerFlag, 1).Tag(mod.DisablesItem("Belt"))},
	//pob:src/Modules/ModParser.lua:3056
	`\+1 ring slot`: []mod.Mod{FLAG("AdditionalRingSlot")},
	`reflects (\d+) physical damage to melee attackers`: []mod.Mod{},
	"ignore all movement penalties from armour":         []mod.Mod{mod.NewFlag("Condition:IgnoreMovementPenalties", true)},
	"gain armour equal to your reserved mana":           []mod.Mod{MOD("Armour", "BASE", 1).Tag(mod.PerStat(1, "ManaReserved"))},
	`(\d+)% increased armour per (\d+) reserved mana`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Armour", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "ManaReserved")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:3080
	"lose all rage on reaching maximum rage and gain wild savagery for 1 second per 10 rage lost this way": []mod.Mod{FLAG("WildSavagery")},
	//pob:src/Modules/ModParser.lua:4579-4581
	`bow attacks sacrifice a random damageable minion to fire (\d+) additional arrows?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ProjectileCount", "BASE", num).KeywordFlag(mod.KeywordFlagArrow).
				Tag(mod.Condition("SacrificeMinionOnAttack"), mod.Condition("HaveDamageableMinion"),
					mod.SkillType(string(data.SkillTypeTriggered)).Neg(true),
					mod.SkillType(string(data.SkillTypeSummonsTotem)).Neg(true)),
		}, ""
	},
	"cannot be stunned": []mod.Mod{FLAG("StunImmune")},
	//pob:block:src/Modules/ModParser.lua:4823
	`cannot be stunned if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("StunImmune").Tag(mod.MultiplierThreshold(firstToUpper(captures[1]) + "Item").Threshold(utils.Float(captures[0])))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:4819-4820
	"cannot be stunned while bleeding":   []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("Bleeding"))},
	"cannot be stunned when on low life": []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("LowLife"))},
	//pob:src/Modules/ModParser.lua:2581
	"you cannot be stunned while at maximum endurance charges": []mod.Mod{
		FLAG("StunImmune").Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax")),
	},
	//pob:block:src/Modules/ModParser.lua:3059-3062
	"action speed cannot be modified to below base value if you have equipped boots with no socketed gems": []mod.Mod{
		MOD("MinimumActionSpeed", "MAX", 100).
			Tag(mod.GlobalEffect("Global").Unscalable(true)).
			Tag(mod.MultiplierThreshold("SocketedGemsInBoots").Threshold(0).Upper(true)).
			Tag(mod.Condition("UsingBoots")),
	},
	"cannot be stunned if you have an equipped helmet with no socketed gems": []mod.Mod{
		FLAG("StunImmune").
			Tag(mod.MultiplierThreshold("SocketedGemsInHelmet").Threshold(0).Upper(true)).
			Tag(mod.Condition("UsingHelmet")),
	},
	"elemental ailments cannot be inflicted on you if you have an equipped body armour with no socketed gems": []mod.Mod{
		FLAG("ElementalAilmentImmune").
			Tag(mod.MultiplierThreshold("SocketedGemsInBody Armour").Threshold(0).Upper(true)).
			Tag(mod.Condition("UsingBody Armour")),
	},
	//pob:block:src/Modules/ModParser.lua:5846-5853
	"take no extra damage from critical strikes": []mod.Mod{
		MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.GlobalEffect("Global").Unscalable(true)),
	},
	"take no extra damage from critical strikes if energy shield recharge started recently": []mod.Mod{
		MOD("ReduceCritExtraDamage", "BASE", 100).Tag(mod.Condition("EnergyShieldRechargeRecently")),
	},
	"take no extra damage from critical strikes if you have equipped gloves with no socketed gems": []mod.Mod{
		MOD("ReduceCritExtraDamage", "BASE", 100).
			Tag(mod.GlobalEffect("Global").Unscalable(true)).
			Tag(mod.MultiplierThreshold("SocketedGemsInGloves").Threshold(0).Upper(true)).
			Tag(mod.Condition("UsingGloves")),
	},
	//pob:block:src/Modules/ModParser.lua:3038-3044
	"defences from equipped body armour are doubled if it has no socketed gems": []mod.Mod{
		MOD("Defences", "MORE", 100).
			Tag(mod.MultiplierThreshold("SocketedGemsInBody Armour").Threshold(0).Upper(true)).
			Tag(mod.Condition("UsingBody Armour")).
			Tag(mod.SlotName("Body Armour")).
			Tag(mod.Multiplier("OathOfTheMajiDoubled").GlobalLimit(100).GlobalLimitKey("OathOfTheMajiLimit")),
		MOD("Multiplier:OathOfTheMajiDoubled", "OVERRIDE", 1).Tag(mod.SlotName("Body Armour")),
	},
	`([\+\-]\d+)% to all elemental resistances if you have an equipped helmet with no socketed gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ElementalResist", "BASE", num).
				Tag(mod.MultiplierThreshold("SocketedGemsInHelmet").Threshold(0).Upper(true)).
				Tag(mod.Condition("UsingHelmet")),
		}, ""
	},
	`(\d+)% increased maximum life if you have equipped gloves with no socketed gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Life", "INC", num).
				Tag(mod.MultiplierThreshold("SocketedGemsInGloves").Threshold(0).Upper(true)).
				Tag(mod.Condition("UsingGloves")),
		}, ""
	},
	`(\d+)% increased movement speed if you have equipped boots with no socketed gems`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MovementSpeed", "INC", num).
				Tag(mod.MultiplierThreshold("SocketedGemsInBoots").Threshold(0).Upper(true)).
				Tag(mod.Condition("UsingBoots")),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5219-5221
	"count as having maximum number of power charges":     []mod.Mod{mod.NewFlag("HaveMaximumPowerCharges", true)},
	"count as having maximum number of frenzy charges":    []mod.Mod{mod.NewFlag("HaveMaximumFrenzyCharges", true)},
	"count as having maximum number of endurance charges": []mod.Mod{mod.NewFlag("HaveMaximumEnduranceCharges", true)},
	"cannot be stunned if you haven't been hit recently":  []mod.Mod{FLAG("StunImmune").Tag(mod.Condition("BeenHitRecently").Neg(true))},
	`cannot be stunned if you have at least (\d+) crab barriers`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("StunImmune").Tag(mod.StatThreshold("CrabBarriers", num)),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:4824-4825
	"cannot be blinded": []mod.Mod{FLAG("Condition:CannotBeBlinded"), FLAG("BlindImmune")},
	"cannot be blinded while affected by precision": []mod.Mod{
		FLAG("Condition:CannotBeBlinded").Tag(mod.Condition("AffectedByPrecision")),
		FLAG("BlindImmune").Tag(mod.Condition("AffectedByPrecision")),
	},
	"chance to avoid being shocked applies to all elemental ailments":            []mod.Mod{mod.NewFlag("ShockAvoidAppliesToElementalAilments", true)},
	"modifiers to chance to avoid being shocked apply to all elemental ailments": []mod.Mod{mod.NewFlag("ShockAvoidAppliesToElementalAilments", true)},
	//pob:block:src/Modules/ModParser.lua:4827-4834
	"cannot be shocked": []mod.Mod{FLAG("ShockImmune")},
	"cannot be frozen":  []mod.Mod{FLAG("FreezeImmune")},
	//pob:block:src/Modules/ModParser.lua:4830
	`cannot be frozen if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("FreezeImmune").Tag(mod.MultiplierThreshold(firstToUpper(captures[1]) + "Item").Threshold(utils.Float(captures[0])))}, ""
	},
	"cannot be chilled": []mod.Mod{FLAG("ChillImmune")},
	"cannot be ignited": []mod.Mod{FLAG("IgniteImmune")},
	"cannot be ignited while at maximum endurance charges":            []mod.Mod{FLAG("IgniteImmune").Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"cannot be chilled while at maximum frenzy charges":               []mod.Mod{FLAG("ChillImmune").Tag(mod.StatThresholdStat("FrenzyCharges", "FrenzyChargesMax"))},
	"cannot be shocked while at maximum power charges":                []mod.Mod{FLAG("ShockImmune").Tag(mod.StatThresholdStat("PowerCharges", "PowerChargesMax"))},
	"you cannot be shocked while at maximum endurance charges":        []mod.Mod{FLAG("ShockImmune").Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax"))},
	"you cannot be shocked while chilled":                             []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("Chilled"))},
	"cannot be shocked while chilled":                                 []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("Chilled"))},
	"cannot be shocked if intelligence is higher than strength":       []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("IntHigherThanStr"))},
	"cannot be frozen if dexterity is higher than intelligence":       []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("DexHigherThanInt"))},
	"cannot be frozen if energy shield recharge has started recently": []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("EnergyShieldRechargeRecently"))},
	"cannot be ignited if strength is higher than dexterity":          []mod.Mod{FLAG("IgniteImmune").Tag(mod.Condition("StrHigherThanDex"))},
	"cannot be chilled while burning":                                 []mod.Mod{FLAG("ChillImmune").Tag(mod.Condition("Burning"))},
	"cannot be chilled while you have onslaught":                      []mod.Mod{FLAG("ChillImmune").Tag(mod.Condition("Onslaught"))},
	//pob:block:src/Modules/ModParser.lua:4860-4861
	"cannot be chilled during onslaught": []mod.Mod{FLAG("ChillImmune").Tag(mod.Condition("Onslaught"))},
	"cannot be frozen or chilled if you've used a fire skill recently": []mod.Mod{
		FLAG("FreezeImmune").Tag(mod.Condition("UsedFireSkillRecently")),
		FLAG("ChillImmune").Tag(mod.Condition("UsedFireSkillRecently")),
	},
	"cannot be inflicted with bleeding":   []mod.Mod{FLAG("BleedImmune")},
	"bleeding cannot be inflicted on you": []mod.Mod{FLAG("BleedImmune")},
	"you are immune to bleeding":          []mod.Mod{FLAG("BleedImmune")},
	`immun[ei]t?y? to poison`:             []mod.Mod{FLAG("PoisonImmune")},
	//pob:block:src/Modules/ModParser.lua:4872-4873
	"cannot be poisoned":                []mod.Mod{FLAG("PoisonImmune")},
	"cannot be poisoned while bleeding": []mod.Mod{FLAG("PoisonImmune").Tag(mod.Condition("Bleeding"))},
	//pob:src/Modules/ModParser.lua:4887
	`immun[ei]t?y? to curses while channelling`:                                    []mod.Mod{FLAG("CurseImmune").Tag(mod.Condition("Channelling"))},
	`gain spirit infusion every ?(\d?)\.?(\d?) seconds? while channelling a spell`: []mod.Mod{FLAG("Condition:CanGainSpiritInfusion")},
	`grants immunity to ignite for (\d+) seconds if used while ignited`:            []mod.Mod{FLAG("IgniteImmune").Tag(mod.Condition("UsingFlask"))},
	`grants immunity to bleeding for (\d+) seconds if used while bleeding`:         []mod.Mod{FLAG("BleedImmune").Tag(mod.Condition("UsingFlask"))},
	`grants immunity to poison for (\d+) seconds if used while poisoned`:           []mod.Mod{FLAG("PoisonImmune").Tag(mod.Condition("UsingFlask"))},
	`grants immunity to freeze for (\d+) seconds if used while frozen`:             []mod.Mod{FLAG("FreezeImmune").Tag(mod.Condition("UsingFlask"))},
	`grants immunity to chill for (\d+) seconds if used while chilled`:             []mod.Mod{FLAG("ChillImmune").Tag(mod.Condition("UsingFlask"))},
	`grants immunity to shock for (\d+) seconds if used while shocked`:             []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("UsingFlask"))},
	`immun[ei]t?y? to shock during f?l?a?s?k? ?effect`:                             []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("UsingFlask"))},
	`immun[ei]t?y? to ignite during f?l?a?s?k? ?effect`:                            []mod.Mod{FLAG("IgniteImmune").Tag(mod.Condition("UsingFlask"))},
	`immun[ei]t?y? to poison during f?l?a?s?k? ?effect`:                            []mod.Mod{FLAG("PoisonImmune").Tag(mod.Condition("UsingFlask"))},
	`immun[ei]t?y? to curses during f?l?a?s?k? ?effect`:                            []mod.Mod{FLAG("CurseImmune").Tag(mod.Condition("UsingFlask"))},
	`immun[ei]t?y? to bleeding during f?l?a?s?k? ?effect`:                          []mod.Mod{FLAG("BleedImmune").Tag(mod.Condition("UsingFlask"))},
	`immun[ei]t?y? to bleeding and corrupted blood during f?l?a?s?k? ?effect`: []mod.Mod{
		FLAG("BleedImmune").Tag(mod.Condition("UsingFlask")),
		FLAG("CorruptedBloodImmune").Tag(mod.Condition("UsingFlask")),
	},
	`immun[ei]t?y? to freeze and chill during f?l?a?s?k? ?effect`: []mod.Mod{
		FLAG("FreezeImmune").Tag(mod.Condition("UsingFlask")),
		FLAG("ChillImmune").Tag(mod.Condition("UsingFlask")),
	},
	`immun[ei]t?y? to freeze and chill while ignited`: []mod.Mod{
		FLAG("FreezeImmune").Tag(mod.Condition("Ignited")),
		FLAG("ChillImmune").Tag(mod.Condition("Ignited")),
	},
	//pob:block:src/Modules/ModParser.lua:4894-4899
	`when you kill an enemy affected by a non-aura hex, become immune to curses for remaining hex duration`: []mod.Mod{FLAG("Condition:CanBeCurseImmune")},
	`when you kill an enemy cursed with a non-aura hex, become immune to curses for remaining hex duration`: []mod.Mod{FLAG("Condition:CanBeCurseImmune")},
	`immun[ei]t?y? to freeze, chill, curses and stuns during f?l?a?s?k? ?effect`: []mod.Mod{
		FLAG("FreezeImmune").Tag(mod.Condition("UsingFlask")),
		FLAG("ChillImmune").Tag(mod.Condition("UsingFlask")),
		FLAG("CurseImmune").Tag(mod.Condition("UsingFlask")),
		FLAG("StunImmune").Tag(mod.Condition("UsingFlask")),
	},
	"unaffected by curses":                            []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"unaffected by curses while affected by zealotry": []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.Condition("AffectedByZealotry")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	//pob:block:src/Modules/ModParser.lua:4915-4921
	"unaffected by vulnerability while affected by determination":           []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.SkillName("Vulnerability")).Tag(mod.Condition("AffectedByDetermination")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"unaffected by enfeeble while affected by grace":                        []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.SkillName("Enfeeble")).Tag(mod.Condition("AffectedByGrace")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"unaffected by temporal chains while affected by haste":                 []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.SkillName("Temporal Chains")).Tag(mod.Condition("AffectedByHaste")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"unaffected by elemental weakness while affected by purity of elements": []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.SkillName("Elemental Weakness")).Tag(mod.Condition("AffectedByPurityofElements")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"unaffected by flammability while affected by purity of fire":           []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.SkillName("Flammability")).Tag(mod.Condition("AffectedByPurityofFire")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"unaffected by frostbite while affected by purity of ice":               []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.SkillName("Frostbite")).Tag(mod.Condition("AffectedByPurityofIce")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"unaffected by conductivity while affected by purity of lightning":      []mod.Mod{MOD("CurseEffectOnSelf", "MORE", -100).Tag(mod.SkillName("Conductivity")).Tag(mod.Condition("AffectedByPurityofLightning")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	`immune to curses while you have at least (\d+) rage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("CurseImmune").Tag(mod.MultiplierThreshold("Rage").Threshold(num)),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:4945
	"the effect of chill on you is reversed": []mod.Mod{
		mod.NewFlag("SelfChillEffectIsReversed", true),
		MOD("Dummy", "DUMMY", 1).Tag(mod.Condition("Chilled")),
	},
	`your movement speed is (\d+)% of its base value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeed", "OVERRIDE", num/100)}, ""
	},
	"armour also applies to lightning damage taken from hits": []mod.Mod{mod.NewFloat("ArmourAppliesToLightningDamageTaken", mod.TypeBase, 100)},
	//pob:block:src/Modules/ModParser.lua:2408
	`(\d+)% of armour also applies to chaos damage taken from hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ArmourAppliesToChaosDamageTaken", mod.TypeBase, num)}, ""
	},
	//pob:block:src/Modules/ModParser.lua:2409
	"armour also applies to chaos damage taken from hits": []mod.Mod{mod.NewFloat("ArmourAppliesToChaosDamageTaken", mod.TypeBase, 100)},
	//pob:block:src/Modules/ModParser.lua:2384-2386
	"physical damage taken bypasses energy shield":                []mod.Mod{mod.NewFloat("PhysicalEnergyShieldBypass", mod.TypeBase, 100)},
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
	`(\d+)% increased maximum energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnergyShield", "INC", num).Tag(mod.Global())}, ""
	},
	`(\d+)% increased global maximum energy shield and reduced lightning resistance`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShield", "INC", num).Tag(mod.Global()),
			MOD("LightningResist", "INC", -num),
		}, ""
	},
	"cannot be ignited while on low life": []mod.Mod{FLAG("IgniteImmune").Tag(mod.Condition("LowLife"))},
	"ward does not break during f?l?a?s?k? ?effect": []mod.Mod{
		FLAG("Condition:WardNotBreak").Tag(mod.Condition("UsingFlask")),
		FLAG("Condition:UnbrokenWard").Tag(mod.Condition("UsingFlask")),
	},
	"cannot knock enemies back":                                     []mod.Mod{mod.NewFlag("CannotKnockback", true)},
	"knocks back enemies if you get a critical strike with a staff": []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(mod.MFlagStaff).Tag(mod.Condition("CriticalStrike"))},
	"knocks back enemies if you get a critical strike with a bow":   []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(mod.MFlagBow).Tag(mod.Condition("CriticalStrike"))},
	"bow knockback at close range":                                  []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(mod.MFlagBow).Tag(mod.Condition("AtCloseRange"))},
	"adds knockback during f?l?a?s?k? ?effect":                      []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Tag(mod.Condition("UsingFlask"))},
	"adds knockback to melee attacks during f?l?a?s?k? ?effect":     []mod.Mod{MOD("EnemyKnockbackChance", "BASE", 100).Flag(mod.MFlagMelee).Tag(mod.Condition("UsingFlask"))},
	"knockback direction is reversed":                               []mod.Mod{MOD("EnemyKnockbackDistance", "MORE", -200)},
	//pob:src/Modules/ModParser.lua:5005
	"culling strike": []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"culling strike during f?l?a?s?k? ?effect":                           []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.Condition("UsingFlask")).Tag(mod.GlobalEffect("Global").Unscalable(true))},
	"hits with this weapon have culling strike against bleeding enemies": []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.ActorCondition("enemy", "Bleeding"))},
	"you have culling strike against cursed enemies":                     []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.ActorCondition("enemy", "Cursed"))},
	"critical strikes have culling strike":                               []mod.Mod{MOD("CriticalCullPercent", "MAX", 10)},
	"your critical strikes have culling strike":                          []mod.Mod{MOD("CriticalCullPercent", "MAX", 10)},
	"your spells have culling strike":                                    []mod.Mod{MOD("CullPercent", "MAX", 10).Flag(mod.MFlagSpell)},
	"culling strike against burning enemies":                             []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.ActorCondition("enemy", "Burning"))},
	//pob:block:src/Modules/ModParser.lua:5014
	"bow attacks have culling strike": []mod.Mod{MOD("CullPercent", "MAX", 10).Flag(mod.MFlagAttack | mod.MFlagBow)},
	//pob:block:src/Modules/ModParser.lua:5018
	"nearby allies have culling strike": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("CullPercent", "MAX", 10), OnlyAllies: true})},
	//pob:src/Modules/ModParser.lua:5019
	"hits that stun enemies have culling strike": []mod.Mod{FLAG("Condition:maceMasteryStunCullSpecced")},
	//pob:block:src/Modules/ModParser.lua:5087-5101
	"raised beast spectres have farrul's farric presence": []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{FromAllies: true, Mod: MOD("Accuracy", "INC", 80)}).Tag(mod.Condition("HaveBeastSpectre")),
		MOD("ExtraAura", "LIST", mod.ExtraAura{FromAllies: true, Mod: MOD("CritChance", "INC", 120)}).Tag(mod.Condition("HaveBeastSpectre")),
		MOD("ExtraAura", "LIST", mod.ExtraAura{FromAllies: true, Mod: MOD("ReduceCritExtraDamage", "BASE", 100)}).Tag(mod.Condition("HaveBeastSpectre")),
	},
	"raised beast spectres have farrul's fertile presence": []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{FromAllies: true, Mod: MOD("Damage", "INC", 100)}).Tag(mod.Condition("HaveBeastSpectre")),
		MOD("ExtraAura", "LIST", mod.ExtraAura{FromAllies: true, Mod: MOD("LifeRegenPercent", "BASE", 3)}).Tag(mod.Condition("HaveBeastSpectre")),
		MOD("ExtraAura", "LIST", mod.ExtraAura{FromAllies: true, Mod: FLAG("StunImmune")}).Tag(mod.Condition("HaveBeastSpectre")),
	},
	"raised beast spectres have farrul's wild presence": []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{FromAllies: true, Mod: MOD("Speed", "INC", 20)}).Tag(mod.Condition("HaveBeastSpectre")),
		MOD("ExtraAura", "LIST", mod.ExtraAura{FromAllies: true, Mod: MOD("MovementSpeed", "INC", 20)}).Tag(mod.Condition("HaveBeastSpectre")),
		MOD("ExtraAura", "LIST", mod.ExtraAura{FromAllies: true, Mod: MOD("MinimumActionSpeed", "MAX", 100)}).Tag(mod.Condition("HaveBeastSpectre")),
	},
	//pob:block:src/Modules/ModParser.lua:4955
	"nearby allies' action speed cannot be modified to below base value": []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.GlobalEffect("Global").Unscalable(true)), OnlyAllies: true})},
	"culling strike against marked enemy":                                []mod.Mod{MOD("CullPercent", "MAX", 10).Tag(mod.ActorCondition("enemy", "Marked"))},
	"permanently intimidate enemies on block":                            []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Intimidated", true)}).Tag(mod.Condition("BlockedRecently"))},
	//pob:block:src/Modules/ModParser.lua:2174
	"enemies near corpses affected by your curses are blinded": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Blinded", true)}).Tag(mod.MultiplierThreshold("NearbyCorpse").Threshold(1)).Tag(mod.ActorCondition("enemy", "Cursed"))},
	`with a murderous eye jewel socketed, intimidate enemies for (\d) seconds on hit with attacks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Intimidated", true)}).Tag(mod.Condition("HaveMurderousEyeJewelIn{SlotName}")),
		}, ""
	},
	"enemies taunted by your warcries are intimidated":                     []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated").Tag(mod.Condition("Taunted"))}).Tag(mod.Condition("UsedWarcryRecently"))},
	`intimidate enemies for (\d+) seconds on block while holding a shield`: []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Intimidated", true)}).Tag(mod.Condition("BlockedRecently")).Tag(mod.Condition("UsingShield"))},
	"flasks do not apply to you":                                           []mod.Mod{mod.NewFlag("FlasksDoNotApplyToPlayer", true)},
	"flasks apply to your zombies and spectres":                            []mod.Mod{FLAG("FlasksApplyToMinion").Tag(mod.SkillName("Raise Zombie", "Raise Spectre").Transfigured(true))},
	"flasks apply to your raised zombies and spectres":                     []mod.Mod{FLAG("FlasksApplyToMinion").Tag(mod.SkillName("Raise Zombie", "Raise Spectre").Transfigured(true))},
	"flasks you use apply to your raised zombies and spectres":             []mod.Mod{FLAG("FlasksApplyToMinion").Tag(mod.SkillName("Raise Zombie", "Raise Spectre").Transfigured(true))},
	"your minions use your flasks when summoned":                           []mod.Mod{mod.NewFlag("FlasksApplyToMinion", true)},
	`recover an additional (\d+)% of flask's life recovery amount over 10 seconds if used while not on full life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFloat("FlaskAdditionalLifeRecovery", mod.TypeBase, num),
		}, ""
	},
	"creates a smoke cloud on use":                []mod.Mod{},
	"creates chilled ground on use":               []mod.Mod{},
	"creates consecrated ground on use":           []mod.Mod{},
	"removes bleeding on use":                     []mod.Mod{},
	"removes burning on use":                      []mod.Mod{},
	"removes curses on use":                       []mod.Mod{},
	"removes freeze and chill on use":             []mod.Mod{},
	"removes poison on use":                       []mod.Mod{},
	"removes shock on use":                        []mod.Mod{},
	"gain unholy might during f?l?a?s?k? ?effect": []mod.Mod{FLAG("Condition:UnholyMight").Tag(mod.Condition("UsingFlask"))},
	"zealot's oath during f?l?a?s?k? ?effect":     []mod.Mod{FLAG("ZealotsOath").Tag(mod.Condition("UsingFlask"))},
	`grants level (\d+) (.+) curse aura during f?l?a?s?k? ?effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraCurse", "LIST", mod.ExtraCurse{SkillName: strings.ReplaceAll(captures[1], " skill", ""), Level: int(num)}).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`shocks nearby enemies during f?l?a?s?k? ?effect, causing (\d+)% increased damage taken`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ShockOverride", "BASE", num).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`during f?l?a?s?k? ?effect, (\d+)% reduced damage taken of each element for which your uncapped elemental resistance is lowest`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LightningDamageTaken", "INC", -num).Tag(mod.StatThresholdStat("LightningResistTotal", "ColdResistTotal").Upper(true)).Tag(mod.StatThresholdStat("LightningResistTotal", "FireResistTotal").Upper(true)),
			MOD("ColdDamageTaken", "INC", -num).Tag(mod.StatThresholdStat("ColdResistTotal", "LightningResistTotal").Upper(true)).Tag(mod.StatThresholdStat("ColdResistTotal", "FireResistTotal").Upper(true)),
			MOD("FireDamageTaken", "INC", -num).Tag(mod.StatThresholdStat("FireResistTotal", "LightningResistTotal").Upper(true)).Tag(mod.StatThresholdStat("FireResistTotal", "ColdResistTotal").Upper(true)),
		}, ""
	},
	`during f?l?a?s?k? ?effect, damage penetrates (\d+)% o?f? ?resistance of each element for which your uncapped elemental resistance is highest`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LightningPenetration", "BASE", num).Tag(mod.StatThresholdStat("LightningResistTotal", "ColdResistTotal")).Tag(mod.StatThresholdStat("LightningResistTotal", "FireResistTotal")),
			MOD("ColdPenetration", "BASE", num).Tag(mod.StatThresholdStat("ColdResistTotal", "LightningResistTotal")).Tag(mod.StatThresholdStat("ColdResistTotal", "FireResistTotal")),
			MOD("FirePenetration", "BASE", num).Tag(mod.StatThresholdStat("FireResistTotal", "LightningResistTotal")).Tag(mod.StatThresholdStat("FireResistTotal", "ColdResistTotal")),
		}, ""
	},
	`recover (\d+)% of life on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num))}, ""
	},
	`recover (\d+)% of life on kill for each different type of mastery you have allocated`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.Multiplier("AllocatedMasteryType")),
		}, ""
	},
	`recover (\d+)% of life on killing a poisoned enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.MultiplierThreshold("PoisonStack").Actor("enemy").Threshold(1)),
		}, ""
	},
	`recover (\d+)% of life on killing a chilled enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.ActorCondition("enemy", "Chilled")),
		}, ""
	},
	`recover (\d+)% of life when you kill a cursed enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.ActorCondition("enemy", "Cursed")),
		}, ""
	},
	`recover (\d+)% of mana when you kill a cursed enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ManaOnKill", "BASE", 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.ActorCondition("enemy", "Cursed")),
		}, ""
	},
	`recover (\d+)% of energy shield when you kill a cursed enemy`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnKill", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)).Tag(mod.ActorCondition("enemy", "Cursed")),
		}, ""
	},
	`recover (\d+)% of life on kill if you've spent life recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.MultiplierThreshold("LifeSpentRecently").Threshold(1)),
		}, ""
	},
	`lose (\d+)% of life on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", -1).Tag(mod.PercentStat("Life", num))}, ""
	},
	`\+(\d+) life gained on killing ignited enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", num).Tag(mod.ActorCondition("enemy", "Ignited"))}, ""
	},
	`gain (\d+) life per ignited enemy killed`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeOnKill", "BASE", num).Tag(mod.ActorCondition("enemy", "Ignited"))}, ""
	},
	`recover (\d+)% of mana on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnKill", "BASE", 1).Tag(mod.PercentStat("Mana", num))}, ""
	},
	//pob:src/Modules/ModParser.lua:4661
	`recover (\d+)% of mana on kill while you have a tincture active`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaOnKill", "BASE", 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.Condition("UsingTincture"))}, ""
	},
	`recover (\d+)% of mana on kill for each different type of mastery you have allocated`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ManaOnKill", "BASE", 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.Multiplier("AllocatedMasteryType")),
		}, ""
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
		return []mod.Mod{
			MOD("EnergyShieldOnKill", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)).Tag(mod.Multiplier("AllocatedMasteryType")),
		}, ""
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
	`recover (\d+)% of life when you kill an enemy during (?:flask )?effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeOnKill", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`recover (\d+)% of mana when you kill an enemy during (?:flask )?effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ManaOnKill", "BASE", 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`recover (\d+)% of energy shield when you kill an enemy during (?:flask )?effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShieldOnKill", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)).Tag(mod.Condition("UsingFlask")),
		}, ""
	},
	`(\d+)% of maximum life taken as chaos damage per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ChaosDegen", "BASE", 1).Tag(mod.PercentStat("Life", num)),
		}, ""
	},
	"your critical strikes do not deal extra damage during f?l?a?s?k? ?effect": []mod.Mod{FLAG("NoCritMultiplier").Tag(mod.Condition("UsingFlask"))},
	"grants perfect agony during f?l?a?s?k? ?effect":                           []mod.Mod{MOD("Keystone", "LIST", "Perfect Agony").Tag(mod.Condition("UsingFlask"))},
	"grants eldritch battery during f?l?a?s?k? ?effect":                        []mod.Mod{MOD("Keystone", "LIST", "Eldritch Battery").Tag(mod.Condition("UsingFlask"))},
	"eldritch battery during f?l?a?s?k? ?effect":                               []mod.Mod{MOD("Keystone", "LIST", "Eldritch Battery").Tag(mod.Condition("UsingFlask"))},
	"chaos damage t?a?k?e?n? ?does not bypass energy shield during effect":     []mod.Mod{mod.NewFlag("ChaosNotBypassEnergyShield", true)},
	//pob:src/Modules/ModParser.lua:5083
	`y?o?u?r? ?skills [ch][oa][sv][te] no mana c?o?s?t? ?during f?l?a?s?k? ?effect`:     []mod.Mod{MOD("ManaCost", "MORE", -100).Tag(mod.Condition("UsingFlask"))},
	"life recovery from flasks also applies to energy shield during f?l?a?s?k? ?effect": []mod.Mod{FLAG("LifeFlaskAppliesToEnergyShield").Tag(mod.Condition("UsingFlask"))},
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
	`passives?(?: skills)? in radius of ([\w\s']+) can be allocated without being connected to your tree`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("JewelData", "LIST", mod.JewelData{Key: "impossibleEscapeKeystone", Value: captures[0]}),
			MOD("ImpossibleEscapeKeystones", "LIST", mod.ImpossibleEscapeKeystones{Key: captures[0], Value: true}),
		}, ""
	},
	`(\d+)% increased effect of jewel socket passive skills containing corrupted (magic|rare) jewels, if not from cluster jewels`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "corrupted" + titleCaseAfterNonWord(captures[1]) + "JewelIncEffect", Value: num})}, ""
	},
	`(\d+)% increased effect of jewel socket passive skills containing corrupted (magic|rare) jewels`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "corrupted" + titleCaseAfterNonWord(captures[1]) + "JewelIncEffect", Value: num})}, ""
	},
	`passives?(?: skills)? in radius can be allocated without being connected to your tree`: []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "intuitiveLeapLike", Value: true})},
	"keystone passive skills in radius can be allocated without being connected to your tree": []mod.Mod{
		MOD("JewelData", "LIST", mod.JewelData{Key: "intuitiveLeapLike", Value: true}),
		MOD("JewelData", "LIST", mod.JewelData{Key: "intuitiveLeapKeystoneOnly", Value: true}),
	},
	"affects passives in small ring":           []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 6})},
	"affects passives in medium ring":          []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 7})},
	"affects passives in large ring":           []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 8})},
	"affects passives in very large ring":      []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 9})},
	"affects passives in massive ring":         []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 10})},
	"only affects passives in small ring":      []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 6})},
	"only affects passives in medium ring":     []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 7})},
	"only affects passives in large ring":      []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 8})},
	"only affects passives in very large ring": []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 9})},
	"only affects passives in massive ring":    []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "radiusIndex", Value: 10})},
	"primordial":                               []mod.Mod{mod.NewFloat("Multiplier:PrimordialItem", mod.TypeBase, 1)},
	`spectres have a base duration of (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "duration", Value: 6}).Tag(mod.SkillName("Raise Spectre").Transfigured(true))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5180-5181
	//pob:src/Modules/ModParser.lua:5178-5179
	`while a unique enemy is in your presence, flasks applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffect", "INC", num).Tag(mod.ActorCondition("enemy", "RareOrUnique")).Tag(mod.ActorCondition("player"))}, ""
	},
	`while a pinnacle atlas boss is in your presence, flasks applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffect", "INC", num).Tag(mod.ActorCondition("enemy", "PinnacleBoss")).Tag(mod.ActorCondition("player"))}, ""
	},
	//pob:src/Modules/ModParser.lua:2420
	`increases and reductions to effect of flasks applied to you also applies to effect of arcane surge on you at (\d+)% of their value`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffectToArcaneSurgeEffect", "BASE", num)}, ""
	},
	`flasks applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("FlaskEffect", mod.TypeIncrease, num).Tag(mod.ActorCondition("player"))}, ""
	},
	`magic utility flasks applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MagicUtilityFlaskEffect", mod.TypeIncrease, num).Tag(mod.ActorCondition("player"))}, ""
	},
	`flasks applied to you have (\d+)% reduced effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffect", "INC", -num).Tag(mod.ActorCondition("player"))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5182-5193
	`tinctures applied to you have (\d+)% increased effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("TinctureEffect", mod.TypeIncrease, num).Tag(mod.ActorCondition("player"))}, ""
	},
	`tinctures applied to you have (\d+)% increased effect if you've used a life flask recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("TinctureEffect", mod.TypeIncrease, num).
			Tag(mod.ActorCondition("player")).Tag(mod.Condition("UsingLifeFlask"))}, ""
	},
	`tinctures applied to you have (\d+)% increased effect while affected by no flasks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("TinctureEffect", mod.TypeIncrease, num).
			Tag(mod.ActorCondition("player")).Tag(mod.Condition("UsingFlask").Neg(true))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5185
	`tinctures have (\d+)% increased effect while at or above (\d+) stacks of mana burn`: func(num float64, captures []string) ([]mod.Mod, string) {
		threshold, _ := strconv.ParseFloat(captures[1], 64)
		return []mod.Mod{mod.NewFloat("TinctureEffect", mod.TypeIncrease, num).
			Tag(mod.MultiplierThresholdList("ManaBurnStacks", "WeepingWoundsStacks").Threshold(threshold))}, ""
	},
	`tinctures applied to you have (\d+)% reduced mana burn rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TinctureManaBurnRate", "INC", -num).Tag(mod.ActorCondition("player"))}, ""
	},
	`tinctures applied to you have (\d+)% less mana burn rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TinctureManaBurnRate", "MORE", -num).Tag(mod.ActorCondition("player"))}, ""
	},
	`the first (\d+) mana burn applied to you have no effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EffectiveManaBurnStacks", "BASE", -num).Tag(mod.ActorCondition("player"))}, ""
	},
	`tinctures deactivate when you have (\d+) or more mana burn`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MaxManaBurnStacks", mod.TypeBase, num)}, ""
	},
	"tinctures inflict weeping wounds instead of mana burn": []mod.Mod{mod.NewFlag("Condition:WeepingWoundsInsteadOfManaBurn", true)},
	"tincture effects also apply to ranged weapons":         []mod.Mod{mod.NewFlag("TinctureRangedWeapons", true)},
	"you can have an additional tincture active":            []mod.Mod{mod.NewFloat("TinctureLimit", mod.TypeBase, 1)},
	`(\d+)% increased tincture cooldown recovery rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("TinctureCooldownRecovery", mod.TypeIncrease, num)}, ""
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
	//pob:block:src/Modules/ModParser.lua:5205-5209
	"can't use chest armour": []mod.Mod{MOD("CanNotUseItem", mod.TypeDisablerFlag, 1).Tag(mod.DisablesItem("Body Armour"))},
	`can't use helmets?`:     []mod.Mod{MOD("CanNotUseItem", mod.TypeDisablerFlag, 1).Tag(mod.DisablesItem("Helmet"))},
	"can't use other rings": []mod.Mod{
		MOD("CanNotUseItem", mod.TypeDisablerFlag, 1).Tag(mod.DisablesItem("Ring 2")).Tag(mod.SlotNumber(1)),
		MOD("CanNotUseItem", mod.TypeDisablerFlag, 1).Tag(mod.DisablesItem("Ring 1")).Tag(mod.SlotNumber(2)),
	},
	"uses both hand slots": []mod.Mod{
		MOD("CanNotUseItem", mod.TypeDisablerFlag, 1).Tag(mod.DisablesItem("Weapon 2")).Tag(mod.SlotNumber(1)),
		MOD("CanNotUseItem", mod.TypeDisablerFlag, 1).Tag(mod.DisablesItem("Weapon 1")).Tag(mod.SlotNumber(2)),
	},
	"can't use flask in fifth slot": []mod.Mod{
		MOD("CanNotUseItem", mod.TypeDisablerFlag, 1).Tag(mod.DisablesItem("Flask 5").ExcludeItemType("Tincture")),
	},
	`leftmost (\d+) magic utility flasks constantly apply their flask effects to you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("LeftActiveMagicUtilityFlasks", mod.TypeBase, num)}, ""
	},
	`rightmost (\d+) magic utility flasks constantly apply their flask effects to you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("RightActiveMagicUtilityFlasks", mod.TypeBase, num)}, ""
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
	//pob:src/Modules/ModParser.lua:5238
	`warcries have (\d+)% chance to exert (\d+) additional attacks?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraExertedAttacks", "BASE", num*utils.Float(captures[1])/100)}, ""
	},
	`skills deal (\d+)% more damage for each warcry exerting them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("EchoesExertAverageIncrease", mod.TypeMore, num)}, ""
	},
	`(\d+)% chance to inflict an additional poison on the same target when you inflict poison`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("AdditionalPoisonChance", mod.TypeBase, num)}, ""
	},
	//pob:src/Modules/ModParser.lua:3937
	`inflict (\d+) additional poisons? on the same target when you inflict poisons? with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("AdditionalPoisonStacks", "BASE", num).Tag(mod.Condition("{Hand}Attack"))}, ""
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
	//pob:block:src/Modules/ModParser.lua:5251-5253
	"you can't deal damage with skills yourself": []mod.Mod{
		FLAG("DealNoDamage").Tag(
			mod.SkillType(
				string(data.SkillTypeSummonsTotem),
				string(data.SkillTypeRemoteMined),
				string(data.SkillTypeTrapped),
			).Neg(true),
			mod.Condition("usedByMirage").Neg(true),
		),
	},
	//pob:src/Modules/ModParser.lua:4343
	"stormblast, icicle and pyroclast mine deal no damage": []mod.Mod{
		mod.NewFlag("DealNoDamage", true).Tag(mod.SkillName("Stormblast Mine", "Icicle Mine", "Pyroclast Mine").Transfigured(true)),
	},
	"deal no non-elemental damage": []mod.Mod{mod.NewFlag("DealNoPhysical", true), mod.NewFlag("DealNoChaos", true)},
	//pob:block:src/Modules/ModParser.lua:5259
	"spell skills deal no damage": []mod.Mod{FLAG("DealNoDamage").Tag(mod.SkillType(string(data.SkillTypeSpell)))},
	//pob:block:src/Modules/ModParser.lua:5255-5256
	`deal no non-([a-z]+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return dealNoNonDamageType(captures[0], false), ""
	},
	`cannot deal non-([a-z]+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return dealNoNonDamageType(captures[0], false), ""
	},
	//pob:block:src/Modules/ModParser.lua:4432
	`minions deal no non-([a-z]+) damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return dealNoNonDamageType(captures[0], true), ""
	},
	//pob:block:src/Modules/ModParser.lua:2204-2211
	`chance to evade hits is based off of (\d+)% of your ward instead of your evasion rating`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("EvadeChanceBasedOnWard"),
			MOD("EvadeChanceBasedOnWardPercent", "OVERRIDE", num).Source("Black Scythe Training"),
		}, ""
	},
	`physical damage reduction from hits is based off of (\d+)% of your ward instead of your armour`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("PhysicalReductionBasedOnWard"),
			MOD("PhysicalReductionBasedOnWardPercent", "OVERRIDE", num).Source("Black Scythe Training"),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:2317
	"your bleeding does not deal extra damage while the enemy is moving and cannot be aggravated": []mod.Mod{FLAG("Condition:NoExtraBleedDamageToMovingEnemy"), FLAG("Condition:CannotAggravate")},
	//pob:block:src/Modules/ModParser.lua:2421
	"non-instant mana recovery from flasks is also recovered as life": []mod.Mod{FLAG("ManaFlaskAppliesToLife")},
	//pob:block:src/Modules/ModParser.lua:2444
	"your hexes have infinite duration": []mod.Mod{MOD("Duration", "BASE", math.Inf(1)).Tag(mod.SkillType(string(data.SkillTypeAppliesCurse)))},
	//pob:block:src/Modules/ModParser.lua:3145-3146
	"life flasks used while on low life apply recovery instantly": []mod.Mod{MOD("LifeFlaskInstantRecovery", "BASE", 100).Tag(mod.Condition("LowLife"))},
	"mana flasks used while on low mana apply recovery instantly": []mod.Mod{MOD("ManaFlaskInstantRecovery", "BASE", 100).Tag(mod.Condition("LowMana"))},
	//pob:block:src/Modules/ModParser.lua:3407
	"increases and reductions to minion cast speed also affects? you": []mod.Mod{FLAG("MinionCastSpeedAppliesToPlayer")},
	//pob:block:src/Modules/ModParser.lua:3530-3533
	"non-critical strikes deal no damage": []mod.Mod{MOD("Damage", "MORE", -100).Flag(mod.MFlagHit).Tag(mod.Condition("CriticalStrike").Neg(true))},
	`non-critical strikes deal (\d+)% less damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Damage", "MORE", -num).Flag(mod.MFlagHit).Tag(mod.Condition("CriticalStrike").Neg(true))}, ""
	},
	"spell skills always deal critical strikes on final repeat": []mod.Mod{FLAG("SpellSkillsAlwaysDealCriticalStrikesOnFinalRepeat").Flag(mod.MFlagSpell)},
	"spell skills cannot deal critical strikes except on final repeat": []mod.Mod{
		FLAG("SpellSkillsCannotDealCriticalStrikesExceptOnFinalRepeat").Flag(mod.MFlagSpell),
		FLAG("").Tag(mod.Condition("alwaysFinalRepeat")),
	},
	//pob:block:src/Modules/ModParser.lua:3938
	`cannot poison enemies with at least (\d+) poisons? on them`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("PoisonStackLimit", "MIN", num)}, ""
	},
	//pob:block:src/Modules/ModParser.lua:3940
	`wither on hit with this weapon against enemies with at least (\d+) poisons on them`: []mod.Mod{FLAG("Condition:CanWither")},
	//pob:block:src/Modules/ModParser.lua:3945-3947
	`prevent \+(\d+)% of suppressed spell damage per hit suppressed recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SpellSuppressionEffect", "BASE", num).Tag(mod.Multiplier("HitsSuppressedRecently").Base(0))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:3994
	`excommunicate enemies on melee hit for (\d+) seconds`: []mod.Mod{FLAG("Condition:CanExcommunicate")},
	//pob:block:src/Modules/ModParser.lua:4130-4133
	`warcries grant arcane surge to you and allies, with (\d+)% increased effect per (\d+) power, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: FLAG("Condition:ArcaneSurge")}).Tag(mod.Condition("UsedWarcryRecently")),
			MOD("ArcaneSurgeEffect", "INC", num).Tag(mod.PerStat(utils.Float(captures[1]), "WarcryPower").GlobalLimit(utils.Float(captures[2])).GlobalLimitKey("Brinerot Flag")).Tag(mod.Condition("UsedWarcryRecently")),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:4316-4317
	`with a searching eye jewel socketed, maim enemies for (\d) seconds on hit with attacks`: []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Maimed").Flag(mod.MFlagAttack)}).Tag(mod.Condition("HaveSearchingEyeJewelIn{SlotName}")),
	},
	`with a searching eye jewel socketed, blind enemies for (\d) seconds on hit with attacks`: []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Blinded").Flag(mod.MFlagAttack)}).Tag(mod.Condition("HaveSearchingEyeJewelIn{SlotName}")),
	},
	//pob:block:src/Modules/ModParser.lua:4560-4565
	"projectiles from attacks fork": []mod.Mod{
		FLAG("ForkOnce").Flag(mod.MFlagProjectile).Tag(mod.SkillType(string(data.SkillTypeRangedAttack))),
		MOD("ForkCountMax", "BASE", 1).Flag(mod.MFlagProjectile).Tag(mod.SkillType(string(data.SkillTypeRangedAttack))),
	},
	"projectiles from attacks fork an additional time": []mod.Mod{
		FLAG("ForkTwice").Flag(mod.MFlagProjectile).Tag(mod.SkillType(string(data.SkillTypeRangedAttack))),
		MOD("ForkCountMax", "BASE", 1).Flag(mod.MFlagProjectile).Tag(mod.SkillType(string(data.SkillTypeRangedAttack))),
	},
	`projectiles from attacks can fork (\d+) additional times?`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("ForkTwice").Flag(mod.MFlagProjectile).Tag(mod.SkillType(string(data.SkillTypeRangedAttack))),
			MOD("ForkCountMax", "BASE", num).Flag(mod.MFlagProjectile).Tag(mod.SkillType(string(data.SkillTypeRangedAttack))),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:4629
	"you have vaal pact while at maximum endurance charges": []mod.Mod{MOD("Keystone", "LIST", "Vaal Pact").Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax"))},
	//pob:block:src/Modules/ModParser.lua:4852
	"you cannot be shocked while frozen": []mod.Mod{FLAG("ShockImmune").Tag(mod.Condition("Frozen"))},
	//pob:block:src/Modules/ModParser.lua:4954
	"action speed cannot be modified to below base value while ignited": []mod.Mod{MOD("MinimumActionSpeed", "MAX", 100).Tag(mod.Condition("Ignited"))},
	//pob:block:src/Modules/ModParser.lua:5028-5030
	`intimidate enemies for (\d+) seconds on hit with attacks while at maximum endurance charges`: []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).Tag(mod.StatThresholdStat("EnduranceCharges", "EnduranceChargesMax")).Tag(mod.Condition("HitRecently")),
	},
	//pob:block:src/Modules/ModParser.lua:5126-5128
	`iron flasks gain (\d+) charges? when your ward breaks`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("IronFlaskChargesGeneratedOnWardBreak", "BASE", num)}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5132-5134
	`while on low life, life flasks gain (\d+) charges? every (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LifeFlaskChargesGenerated", "BASE", num/utils.Float(captures[1])).Tag(mod.Condition("LowLife"))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5141-5143
	`flasks gain (\d+) charges? per second if you've hit a unique enemy recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskChargesGenerated", "BASE", num).Tag(mod.Condition("HitRecently")).Tag(mod.ActorCondition("enemy", "RareOrUnique"))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5176
	`flasks applied to you have (\d+)% increased effect per level`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FlaskEffect", "INC", num).Tag(mod.ActorCondition("player")).Tag(mod.Multiplier("Level").Base(0))}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5216-5218
	"minimum power charges equal to maximum while stationary":     []mod.Mod{FLAG("MinimumPowerChargesIsMaximumPowerCharges").Tag(mod.Condition("Stationary"))},
	"minimum frenzy charges equal to maximum while stationary":    []mod.Mod{FLAG("MinimumFrenzyChargesIsMaximumFrenzyCharges").Tag(mod.Condition("Stationary"))},
	"minimum endurance charges equal to maximum while stationary": []mod.Mod{FLAG("MinimumEnduranceChargesIsMaximumEnduranceCharges").Tag(mod.Condition("Stationary"))},
	//pob:block:src/Modules/ModParser.lua:5592-5595
	`skills gain added chaos damage equal to (\d+)% of mana cost, if mana cost is not higher than the maximum you could spend`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("ManaCost", num)),
			MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("ManaCost", num)),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5674
	"non-travel attack skills repeat an additional time": []mod.Mod{
		MOD("RepeatCount", "BASE", 1).KeywordFlag(mod.KeywordFlagAttack).Tag(mod.SkillType(string(data.SkillTypeTravel)).Neg(true)).Tag(mod.Condition("averageRepeat", "alwaysFinalRepeat")),
	},
	//pob:block:src/Modules/ModParser.lua:5687
	`when an enemy hit deals elemental damage to you, their resistance to those elements becomes zero for (\d+) seconds`: []mod.Mod{FLAG("Condition:HaveTrickstersSmile")},
	//pob:block:src/Modules/ModParser.lua:5749
	`\+(\d+) to maximum snipe stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:SnipeStagesMax", "BASE", num).Tag(mod.GlobalEffect("Buff").Unscalable(true))}, ""
	},
	//pob:src/Modules/ModParser.lua:5258
	"deal no damage when not on low life": []mod.Mod{FLAG("DealNoDamage").Tag(mod.Condition("LowLife").Neg(true))},
	//pob:block:src/Modules/ModParser.lua:5260-5261
	"attacks have blood magic":          []mod.Mod{FLAG("CostLifeInsteadOfMana").Flag(mod.MFlagAttack)},
	"attacks cost life instead of mana": []mod.Mod{FLAG("CostLifeInsteadOfMana").Flag(mod.MFlagAttack)},
	//pob:block:src/Modules/ModParser.lua:5269-5272
	`spells have added spell damage equal to (\d+)% of physical damage of your equipped two handed weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("WeaponPhysAppliesToSpells"),
			MOD("WeaponPhysAppliesToSpellsPercent", "BASE", num).Tag(mod.Condition("UsingTwoHandedWeapon")),
		}, ""
	},
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
	//pob:block:src/Modules/ModParser.lua:5293-5295
	`([a-z ]+) can affect hexproof enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("SkillData", "LIST", mod.SkillData{Key: "ignoreHexproof", Value: 1}).Tag(mod.SkillId(extraSkillIDForName(captures[0])))}, ""
	},
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
	`(\d+)% of fire and lightning damage from hits taken as cold damage during effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FireDamageFromHitsTakenAsCold", "BASE", num).Tag(mod.Condition("UsingFlask")),
			MOD("LightningDamageFromHitsTakenAsCold", "BASE", num).Tag(mod.Condition("UsingFlask")),
		}, ""
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
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
		}, ""
	},
	`([\w\s]+) has no reservation`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
		}, ""
	},
	`([\w\s]+) has no reservation if cast as an aura`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeAura))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationFlat", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeAura))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeAura))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
			MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationPercent", Value: 0}).Tag(mod.SkillId(extraSkillIDForName(captures[0]))).Tag(mod.SkillType(string(data.SkillTypeAura))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
		}, ""
	},
	"banner skills reserve no mana": []mod.Mod{
		MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Tag(mod.SkillType(string(data.SkillTypeBanner))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
		MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationPercent", Value: 0}).Tag(mod.SkillType(string(data.SkillTypeBanner))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
	},
	"banner skills have no reservation": []mod.Mod{
		MOD("SkillData", "LIST", mod.SkillData{Key: "ManaReservationPercent", Value: 0}).Tag(mod.SkillType(string(data.SkillTypeBanner))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
		MOD("SkillData", "LIST", mod.SkillData{Key: "LifeReservationPercent", Value: 0}).Tag(mod.SkillType(string(data.SkillTypeBanner))).Tag(mod.SkillType(string(data.SkillTypeBlessing)).Neg(true)),
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
	//pob:src/Modules/ModParser.lua:5342
	"your spells are disabled": []mod.Mod{FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeSpell))), FLAG("ForceEnableCurseApplication")},
	`aura skills other than ([\w\s]+) are disabled`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeAura))).Tag(mod.SkillType(string(data.SkillTypeRemoteMined)).Neg(true)),
			FLAG("EnableSkill").Tag(mod.SkillName(captures[0])),
		}, ""
	},
	`travel skills other than ([\w\s]+) are disabled`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			FLAG("DisableSkill").Tag(mod.SkillType(string(data.SkillTypeTravel))),
			FLAG("EnableSkill").Tag(mod.SkillId(extraSkillIDForName(captures[0]))),
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
			MOD("LightningMax", "BASE", 1).Tag(mod.PercentStat("ManaUnreserved", num)).Tag(mod.Condition("SacrificeManaForLightning")),
		}, ""
	},
	`attacks with this weapon have added maximum lightning damage equal to (\d+)% of your energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LightningMax", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`attacks with this weapon have added maximum lightning damage equal to (\d+)% of your maximum energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LightningMax", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`attacks with this weapon have added fire damage equal to (\d+)% of player's maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FireMin", "BASE", 1).Tag(mod.PercentStat("Life", num).Actor("parent")).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
			MOD("FireMax", "BASE", 1).Tag(mod.PercentStat("Life", num).Actor("parent")).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`adds (\d+)% of your maximum energy shield as cold damage to attacks with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ColdMin", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
			MOD("ColdMax", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num)).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`attacks with this weapon have added maximum lightning damage equal to (\d+)% of player'?s? maximum energy shield`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LightningMax", "BASE", 1).Tag(mod.PercentStat("EnergyShield", num).Actor("parent")).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`adds (\d+)% of your maximum mana as fire damage to attacks with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("FireMin", "BASE", 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
			MOD("FireMax", "BASE", 1).Tag(mod.PercentStat("Mana", num)).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	`gain added chaos damage equal to (\d+)% of ward`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("Ward", num)),
			MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("Ward", num)),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5391-5398
	`while you have unbroken ward, your next non-channelling attack you use yourself breaks your ward to gain added cold damage equal to (\d+)% of ward`: func(num float64, captures []string) ([]mod.Mod, string) {
		wardTags := func(m mod.Mod) mod.Mod {
			return m.Tag(mod.PercentStat("Ward", num)).
				Tag(mod.SkillType(string(data.SkillTypeChannel)).Neg(true)).
				Tag(mod.SkillType(string(data.SkillTypeAttack))).
				Tag(mod.Condition("WardNotBreak").Neg(true)).
				Tag(mod.Condition("UnbrokenWard"))
		}
		return []mod.Mod{
			wardTags(MOD("ColdMin", "BASE", 1)),
			wardTags(MOD("ColdMax", "BASE", 1)),
		}, ""
	},
	`spells deal added chaos damage equal to (\d+)% of your maximum life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.SkillType(string(data.SkillTypeSpell))),
			MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("Life", num)).Tag(mod.SkillType(string(data.SkillTypeSpell))),
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
	"blind does not affect your chance to hit":                             []mod.Mod{mod.NewFlag("UnaffectedByBlind", true)},
	"unaffected by blind": []mod.Mod{mod.NewFlag("UnaffectedByBlind", true)},
	"enemies blinded by you while you are blinded have malediction": []mod.Mod{MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: FLAG("HasMalediction").Tag(mod.Condition("Blinded"))}).Tag(mod.Condition("Blinded"))},
	//pob:block:src/Modules/ModParser.lua:5428-5429
	"skills which throw traps have blood magic":          []mod.Mod{FLAG("CostLifeInsteadOfMana").Tag(mod.SkillType(string(data.SkillTypeTrapped)))},
	"skills which throw traps cost life instead of mana": []mod.Mod{FLAG("CostLifeInsteadOfMana").Tag(mod.SkillType(string(data.SkillTypeTrapped)))},
	`lose ([\d\.]+) mana per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ManaDegen", mod.TypeBase, num)}, ""
	},
	`lose ([\d\.]+)% of maximum mana per second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaDegen", "BASE", 1).Tag(mod.PercentStat("Mana", num))}, ""
	},
	"strength provides no bonus to maximum life":     []mod.Mod{mod.NewFlag("NoStrBonusToLife", true)},
	"intelligence provides no bonus to maximum mana": []mod.Mod{mod.NewFlag("NoIntBonusToMana", true)},
	`inherent loss of rage is (\d+)% slower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("InherentRageLoss", mod.TypeIncrease, -num)}, ""
	},
	`inherent loss of rage is (\d+)% faster`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("InherentRageLoss", mod.TypeIncrease, num)}, ""
	},
	`inherent rage loss starts (\d+) seconds? later`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("InherentRageLossDelay", mod.TypeBase, num)}, ""
	},
	`with a ghastly eye jewel socketed, minions have \+(\d+) to accuracy rating`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: mod.NewFloat("Accuracy", mod.TypeBase, num)}).Tag(mod.Condition("HaveGhastlyEyeJewelIn{SlotName}")),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:5435-5437
	`with a ghastly eye jewel socketed, minions have (\d+)% chance to gain unholy might on hit with spells`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: FLAG("Condition:UnholyMight").Tag(mod.Condition("HitSpellRecently"))}).Tag(mod.Condition("HaveGhastlyEyeJewelIn{SlotName}")),
		}, ""
	},
	"hits ignore enemy monster chaos resistance if all equipped items are shaper items": []mod.Mod{FLAG("IgnoreChaosResistance").Tag(mod.MultiplierThreshold("NonShaperItem").Threshold(0).Upper(true))},
	"hits ignore enemy monster chaos resistance if all equipped items are elder items":  []mod.Mod{FLAG("IgnoreChaosResistance").Tag(mod.MultiplierThreshold("NonElderItem").Threshold(0).Upper(true))},
	//pob:src/Modules/ModParser.lua:5441
	`your hits ignore enemy monster (\w+) resistances? if all equipped rings are (\w+) rings`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Ignore" + firstToUpper(captures[0]) + "Resistance").Tag(mod.MultiplierThreshold(firstToUpper(captures[1]) + "RingEquipped").Threshold(2))}, ""
	},
	"the stars are aligned if you have 6 influence types among other equipped items": []mod.Mod{
		FLAG("Condition:StarsAreAligned").
			Tag(mod.MultiplierThreshold("ShaperItem").Threshold(2)).
			Tag(mod.MultiplierThreshold("ElderItem").Threshold(2)).
			Tag(mod.MultiplierThreshold("WarlordItem").Threshold(2)).
			Tag(mod.MultiplierThreshold("HunterItem").Threshold(2)).
			Tag(mod.MultiplierThreshold("CrusaderItem").Threshold(2)).
			Tag(mod.MultiplierThreshold("RedeemerItem").Threshold(2)),
	},
	`gain \d+ rage on critical hit with attacks, no more than once every [\d\.]+ seconds`: []mod.Mod{
		mod.NewFlag("Condition:CanGainRage", true),
	},
	`warcry skills' cooldown time is (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "OVERRIDE", num).KeywordFlag(mod.KeywordFlagWarcry)}, ""
	},
	`warcry skills have (\+\d+) seconds to cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "BASE", num).KeywordFlag(mod.KeywordFlagWarcry)}, ""
	},
	//pob:src/Modules/ModParser.lua:5458
	`stance skills have (\+\d+) seconds to cooldown`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CooldownRecovery", "BASE", num).Tag(mod.SkillType(string(data.SkillTypeStance)))}, ""
	},
	"using warcries is instant": []mod.Mod{mod.NewFlag("InstantWarcry", true)},
	`attacks with axes or swords grant (\d+) rage on hit, no more than once every second`: []mod.Mod{
		FLAG("Condition:CanGainRage").Tag(mod.Condition("UsingAxe", "UsingSword")),
	},
	//pob:block:src/Modules/ModParser.lua:5463-5466
	`regenerate (\d+) rage per second for every (\d+) life recovery per second from regeneration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("RageRegen", "BASE", num).Tag(mod.PercentStat("LifeRegen", num/utils.Float(captures[1])*100)),
			mod.NewFlag("Condition:CanGainRage", true),
		}, ""
	},
	"when you lose temporal chains you gain maximum rage": []mod.Mod{mod.NewFlag("Condition:CanGainRage", true)},
	`your critical strike multiplier is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("CritMultiplier", mod.TypeOverride, num)}, ""
	},
	`base critical strike chance for attacks with weapons is ([\d\.]+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("WeaponBaseCritChance", mod.TypeOverride, num)}, ""
	},
	`base critical strike chance of spells is the critical strike chance of y?o?u?r? ?main hand weapon`: []mod.Mod{
		FLAG("BaseCritFromMainHand").Flag(mod.MFlagSpell),
	},
	`base spell critical strike chance of spells is equal to that of main hand weapon`: []mod.Mod{
		FLAG("BaseCritFromMainHand").Flag(mod.MFlagSpell),
	},
	`critical strike chance is (\d+)% for hits with this weapon`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "OVERRIDE", num).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5480
	`hits with this weapon have \+(\d+)% to critical strike multiplier per enemy power`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritMultiplier", "BASE", num).Flag(mod.MFlagHit).Tag(mod.Condition("{Hand}Attack")).Tag(mod.SkillType(string(data.SkillTypeAttack))).Tag(mod.Multiplier("EnemyPower").Base(0)),
		}, ""
	},
	//pob:block:src/Modules/ModParser.lua:5481-5483
	`maximum critical strike chance is (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("CritChanceCap", "OVERRIDE", num)}, ""
	},
	"allocates (.+) if you have the matching modifiers? on forbidden (.+)": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("GrantedAscendancyNode", "LIST", mod.GrantedAscendancyNode{Side: captures[1], Name: captures[0]}),
		}, ""
	},
	"allocates (.+)": func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("GrantedPassive", "LIST", captures[0])}, ""
	},
	//pob:src/Modules/ModParser.lua:5486
	"battlemage":              []mod.Mod{mod.NewFlag("Battlemage", true), MOD("MainHandWeaponDamageAppliesToSpells", "MAX", 100)},
	"transfiguration of body": []mod.Mod{mod.NewFlag("TransfigurationOfBody", true)},
	"transfiguration of mind": []mod.Mod{mod.NewFlag("TransfigurationOfMind", true)},
	"transfiguration of soul": []mod.Mod{mod.NewFlag("TransfigurationOfSoul", true)},
	`offering skills have (\d+)% reduced duration`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Duration", "INC", -num).Tag(mod.SkillName("Bone Offering", "Flesh Offering", "Spirit Offering", "Blood Offering")),
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
	//pob:block:src/Modules/ModParser.lua:5517-5519
	`hits against you overwhelm (\d+)% of physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyPhysicalOverwhelm", "BASE", num)}, ""
	},
	`hits have (\d+)% increased critical strike chance against you`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("EnemyCritChance", "INC", num)}, ""
	},
	`critical strikes against you do not inherently inflict elemental ailments`: []mod.Mod{
		mod.NewFlag("CritsOnYouDontAlwaysApplyElementalAilments", true),
	},
	//pob:src/Modules/ModParser.lua:5523-5525
	"impale damage dealt to enemies impaled by you ignores enemy physical damage reduction": []mod.Mod{FLAG("IgnoreEnemyImpalePhysicalDamageReduction")},
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
	"you have fungal ground around you while stationary": []mod.Mod{
		MOD("ExtraAura", "LIST", mod.ExtraAura{Mod: mod.NewFloat("ChaosResist", mod.TypeBase, 25)}).Tag(mod.Condition("OnFungalGround", "Stationary")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("ChaosResist", mod.TypeBase, -10)}).Tag(mod.ActorCondition("enemy", "OnFungalGround", "Stationary")),
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFloat("ElementalResist", mod.TypeBase, -10)}).Tag(mod.ActorCondition("enemy", "OnFungalGround", "Stationary")),
	},
	//pob:block:src/Modules/ModParser.lua:5538-5540
	"create fungal ground instead of consecrated ground": []mod.Mod{
		mod.NewFlag("Condition:CreateFungalGround", true),
	},
	"create profane ground instead of consecrated ground": []mod.Mod{
		mod.NewFlag("Condition:CreateProfaneGround", true),
	},
	//pob:src/Modules/ModParser.lua:5544-5546
	`(\d+)% chance to create profane ground on critical strike if intelligence is your highest attribute`: []mod.Mod{
		FLAG("Condition:CreateProfaneGround").Tag(mod.Condition("IntHighestAttribute")),
	},
	//pob:src/Modules/ModParser.lua:5547-5549
	"consecrated path and purifying flame create profane ground instead of consecrated ground": []mod.Mod{
		mod.NewFlag("Condition:CreateProfaneGround", true),
	},
	"you count as dual wielding while you are unencumbered": []mod.Mod{FLAG("Condition:DualWielding").Tag(mod.Condition("Unencumbered"))},
	//pob:block:src/Modules/ModParser.lua:5557-5559
	"you have consecrated ground around you while stationary if strength is your highest attribute": []mod.Mod{FLAG("Condition:OnConsecratedGround").Tag(mod.Condition("StrHighestAttribute")).Tag(mod.Condition("Stationary"))},
	//pob:block:src/Modules/ModParser.lua:5560
	`consecrated ground around you while stationary if (\d+) (\w+) items are equipped`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{FLAG("Condition:OnConsecratedGround").Tag(mod.MultiplierThreshold(firstToUpper(captures[1]) + "Item").Threshold(utils.Float(captures[0])))}, ""
	},
	"dual wielding does not inherently grant chance to block attack damage": []mod.Mod{mod.NewFlag("Condition:NoInherentBlock", true)},
	"you do not inherently take less damage for having fortification":       []mod.Mod{mod.NewFlag("Condition:NoFortificationMitigation", true)},
	"inherent bonuses from dual wielding are doubled": []mod.Mod{
		mod.NewFlag("Condition:DoubledInherentDualWieldingSpeed", true),
		mod.NewFlag("Condition:DoubledInherentDualWieldingBlock", true),
	},
	"inherent attack speed bonus from dual wielding is doubled while wielding two claws": []mod.Mod{
		mod.NewFlag("Condition:DoubledInherentDualWieldingSpeed", true).Tag(mod.Condition("DualWieldingClaws")),
	},
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
			MOD("LightningMin", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.PercentStat("Mana", num)),
			MOD("LightningMax", "BASE", 1).Flag(mod.MFlagAttack).Tag(mod.PercentStat("Mana", num)),
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
	"elemental skills deal triple damage":      []mod.Mod{MOD("TripleDamageChance", "BASE", 100).Tag(mod.SkillType(string(data.SkillTypeCold), string(data.SkillTypeFire), string(data.SkillTypeLightning)))},
	"deal triple damage with elemental skills": []mod.Mod{MOD("TripleDamageChance", "BASE", 100).Tag(mod.SkillType(string(data.SkillTypeCold), string(data.SkillTypeFire), string(data.SkillTypeLightning)))},
	`skills supported by unleash have \+(\d) to maximum number of seals`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SealCount", mod.TypeBase, num)}, ""
	},
	`skills supported by unleash have (\d+)% increased seal gain frequency`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("SealGainFrequency", mod.TypeIncrease, num)}, ""
	},
	`(\d+)% increased critical strike chance with spells which remove the maximum number of seals`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MaxSealCrit", mod.TypeIncrease, num)}, ""
	},
	//pob:src/Modules/ModParser.lua:3076-3079
	`skills gain added chaos damage equal to (\d+)% of life cost, if life cost is not higher than the maximum you could spend`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ChaosMin", "BASE", 1).Tag(mod.PercentStat("LifeCost", num)).Tag(mod.StatThresholdStat("LifeUnreserved", "LifeCost").Percent(num)),
			MOD("ChaosMax", "BASE", 1).Tag(mod.PercentStat("LifeCost", num)).Tag(mod.StatThresholdStat("LifeUnreserved", "LifeCost").Percent(num)),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:3046-3049
	`spells you cast yourself gain added physical damage equal to (\d+)% of life cost, if life cost is not higher than the maximum you could spend`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("PhysicalMin", "BASE", 1).Tag(mod.PercentStat("LifeCost", num)).Tag(mod.StatThresholdStat("LifeUnreserved", "LifeCost").Percent(num)),
			MOD("PhysicalMax", "BASE", 1).Tag(mod.PercentStat("LifeCost", num)).Tag(mod.StatThresholdStat("LifeUnreserved", "LifeCost").Percent(num)),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:3075
	`unsealed spells gain (\d+)% more damage each time their effects reoccur`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("MaxSealDamage", mod.TypeMore, num)}, ""
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
	//pob:block:src/Modules/ModParser.lua:5614-5616
	`gain sacrificial zeal when you use a skill, dealing you \d+% of the skill's mana cost as physical damage per second`: []mod.Mod{
		mod.NewFlag("SacrificialZeal", true),
	},
	`hits overwhelm (\d+)% of physical damage reduction while you have sacrificial zeal`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnemyPhysicalDamageReduction", "BASE", -num).Tag(mod.Condition("SacrificialZeal")),
		}, ""
	},
	`minions attacks overwhelm (\d+)% physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("EnemyPhysicalDamageReduction", "BASE", -num).Tag(mod.SkillType(string(data.SkillTypeAttack)))}),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:5641-5643
	`minions hits have (\d+)% chance to ignore enemy physical damage reduction`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: MOD("ChanceToIgnoreEnemyPhysicalDamageReduction", "BASE", num)}),
		}, ""
	},
	`focus has (\d+)% increased cooldown recovery rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FocusCooldownRecovery", "INC", num).Tag(mod.Condition("Focused"))}, ""
	},
	`focus has (\d+)% reduced cooldown recovery rate`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FocusCooldownRecovery", "INC", -num).Tag(mod.Condition("Focused"))}, ""
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
	//pob:src/Modules/ModParser.lua:5608
	`(\d+)% chance to gain elusive when you block while dual wielding`: []mod.Mod{FLAG("Condition:CanBeElusive").Tag(mod.Condition("DualWielding"))},
	//pob:src/Modules/ModParser.lua:5609
	`elusive on you reduces in effect (\d+)% slower`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ElusiveEffectLossSlower", mod.TypeIncrease, num)}, ""
	},
	//pob:src/Modules/ModParser.lua:5610
	`elusive's effect on you is increased instead for the first (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ElusiveEffectIncreaseDuration", mod.TypeBase, num)}, ""
	},
	//pob:src/Modules/ModParser.lua:5611
	`elusive is removed from you at (\d+)% effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ElusiveEffectMinThreshold", mod.TypeOverride, num)}, ""
	},
	"nearby enemies are scorched": []mod.Mod{
		MOD("EnemyModifier", "LIST", mod.EnemyModifier{Mod: mod.NewFlag("Condition:Scorched", true)}),
		mod.NewFloat("ScorchBase", mod.TypeBase, 10),
	},
	// Pantheon: Soul of Tukohama support
	//pob:block:src/Modules/ModParser.lua:5689-5690
	`(\d+)% increased quantity of items dropped by slain normal enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LootQuantityNormalEnemies", "INC", num)}, ""
	},
	`(\d+)% increased rarity of items dropped by slain magic enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("LootRarityMagicEnemies", "INC", num)}, ""
	},
	`while stationary, gain ([\d\.]+)% of life regenerated per second every second, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeRegenPercent", "BASE", num).Tag(mod.Multiplier("StationarySeconds").Limit(utils.Float(captures[1])).LimitTotal(true)).Tag(mod.Condition("Stationary")),
		}, ""
	},
	// Pantheon: Soul of Ryslatha support
	`life flasks gain (\d+) charges? every (\d+) seconds if you haven't used a life flask recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeFlaskChargesGenerated", "BASE", num/utils.Float(captures[1])).Tag(mod.Condition("UsingLifeFlask").Neg(true)),
		}, ""
	},
	// Skill-specific enchantment modifiers
	`(\d+)% increased decoy totem life`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("TotemLife", "INC", num).Tag(mod.SkillName("Decoy Totem"))}, ""
	},
	`(\d+)% increased ice spear critical strike chance in second form`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritChance", "INC", num).Tag(mod.SkillName("Ice Spear").Transfigured(true)).Tag(mod.SkillPart(2)),
		}, ""
	},
	`shock nova ring deals (\d+)% increased damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Damage", "INC", num).Tag(mod.SkillName("Shock Nova").Transfigured(true)).Tag(mod.SkillPart(1)),
		}, ""
	},
	`enemies affected by bear trap take (\d+)% increased damage from trap or mine hits`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{
				Mod: MOD("EnemyModifier", "LIST", mod.EnemyModifier{
					Mod: MOD("TrapMineDamageTaken", "INC", num).Tag(mod.GlobalEffect("Debuff")),
				}),
			}).Tag(mod.SkillName("Bear Trap").Transfigured(true)),
		}, ""
	},
	`blade vortex has \+(\d+)% to critical strike multiplier for each blade`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("CritMultiplier", "BASE", num).Tag(mod.Multiplier("BladeVortexBlade").Base(0), mod.SkillName("Blade Vortex").Transfigured(true)),
		}, ""
	},
	`burning arrow has (\d+)% increased debuff effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DebuffEffect", "INC", num).Tag(mod.SkillName("Burning Arrow")),
		}, ""
	},
	`double strike has a (\d+)% chance to deal double damage to bleeding enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DoubleDamageChance", "BASE", num).Tag(mod.ActorCondition("enemy", "Bleeding"), mod.SkillName("Double Strike").Transfigured(true)),
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
		return []mod.Mod{MOD("PoisonChance", "BASE", num).Tag(mod.SkillName("Scourge Arrow").Transfigured(true), mod.Multiplier("ScourgeArrowStage").Base(0))}, ""
	},
	`winter orb has \+(\d+) maximum stages`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("Multiplier:WinterOrbMaxStages", mod.TypeBase, num)}, ""
	},
	`summoned holy relics have (\d+)% increased buff effect`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BuffEffect", "INC", num).Tag(mod.SkillName("Summon Holy Relic").Transfigured(true))}, ""
	},
	`herald skills have (\d+)% more buff effect for every (\d+)% of maximum mana they reserve`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("BuffEffect", "MORE", num).
				Tag(mod.PerStat(utils.Float(captures[1]), "ManaReservedPercent")).
				Tag(mod.SkillType(string(data.SkillTypeHerald))),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:5713
	`\+(\d+) to maximum virulence`: func(num float64, captures []string) ([]mod.Mod, string) {
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
	`caustic arrow has (\d+)% chance to inflict withered on hit for (\d+) seconds base duration`: []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFlag("Condition:CanWither", true)}).Tag(mod.SkillName("Caustic Arrow").Transfigured(true))},
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
		return []mod.Mod{MOD("AreaOfEffect", "MORE", num).Tag(mod.SkillName("Shield Crush").Transfigured(true), mod.SkillPart(2))}, ""
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
			MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "consecrated_ground_enemy_damage_taken_+%", Value: num}).Tag(mod.SkillName("Purifying Flame").Transfigured(true)),
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
		return []mod.Mod{MOD("ExtraSkillStat", "LIST", mod.ExtraSkillStat{Key: "shock_effect_+%", Value: num}).Tag(mod.SkillName("Lightning Trap").Transfigured(true))}, ""
	},
	`wild strike's beam chains an additional (\d+) times`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraSkillMod", "LIST", mod.ExtraSkillMod{Mod: mod.NewFloat("ChainCountMax", mod.TypeBase, num)}).Tag(mod.SkillName("Wild Strike").Transfigured(true), mod.SkillPart(4))}, ""
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
	//pob:src/Modules/ModParser.lua:5758
	`\+([\d.]+) metres? to weapon range per 10% quality`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("AlternateQualityLocalWeaponRangePer10Quality", mod.TypeBase, num*10)}, ""
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
			MOD("DebuffEffect", "BASE", num).Tag(mod.SkillName("Infernal Blow").Transfigured(true)),
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
	`remembrancing (\d+) songworthy deeds by the line of (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "conqueredBy", Value: mod.LegionJewel{ID: int(num), Conqueror: conquerorList[strings.ToLower(captures[1])]}})}, ""
	},
	`subjugating (\d+) souls in the thrall of (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "conqueredBy", Value: mod.LegionJewel{ID: int(num), Conqueror: conquerorList[strings.ToLower(captures[1])]}})}, ""
	},
	`binding (\d+) souls to phylacteries to sustain (.+)`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("JewelData", "LIST", mod.JewelData{Key: "conqueredBy", Value: mod.LegionJewel{ID: int(num), Conqueror: conquerorList[strings.ToLower(captures[1])]}})}, ""
	},
	`passives in radius are conquered by the (\D+)`: []mod.Mod{},
	"historic": []mod.Mod{},
	`\+(\d+) to maximum life per allocated journey tattoo of the body`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Life", "BASE", num).Tag(mod.Multiplier("JourneyTattooBody")),
			MOD("Multiplier:JourneyTattooBody", "BASE", 1),
		}, ""
	},
	`\+(\d+) to maximum energy shield per allocated journey tattoo of the soul`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("EnergyShield", "BASE", num).Tag(mod.Multiplier("JourneyTattooSoul")),
			MOD("Multiplier:JourneyTattooSoul", "BASE", 1),
		}, ""
	},
	`\+(\d+) to maximum mana per allocated journey tattoo of the mind`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("Mana", "BASE", num).Tag(mod.Multiplier("JourneyTattooMind")),
			MOD("Multiplier:JourneyTattooMind", "BASE", 1),
		}, ""
	},
	"survival": []mod.Mod{},
	"you can have two different banners at the same time":                 []mod.Mod{},
	"can have a second enchantment modifier":                              []mod.Mod{},
	`can have (\d+) additional enchantment modifiers`:                     []mod.Mod{},
	"this item can be anointed by cassia":                                 []mod.Mod{},
	"all sockets are white":                                               []mod.Mod{},
	`[\+\-](\d+) prefix modifiers? allowed`:                               []mod.Mod{},
	`[\+\-](\d+) suffix modifiers? allowed`:                               []mod.Mod{},
	"can be anointed":                                                     []mod.Mod{},
	"implicit modifiers cannot be changed":                                []mod.Mod{},
	"has a crucible passive skill tree":                                   []mod.Mod{},
	"has elder, shaper and all conqueror influences":                      []mod.Mod{},
	"has a two handed sword crucible passive skill tree":                  []mod.Mod{},
	"has a crucible passive skill tree with only support passive skills":  []mod.Mod{},
	"crucible passive skill tree is removed if this modifier is removed":  []mod.Mod{},
	`cannot roll (\w+) modifiers`:                                         []mod.Mod{},
	`cannot roll modifiers of non\-(\w+) damage types`:                    []mod.Mod{},
	`\d+% [ir][ne][cd][ru][ec][ae][sd]e?d? ?[a-z\s]* modifier magnitudes`: []mod.Mod{},
	`\d+% [ir][ne][cd][ru][ec][ae][sd]e?d? effect of [sp][ur][fe]fixes`:   []mod.Mod{},
	`[a-z\s]* modifier magnitudes are doubled`:                            []mod.Mod{},
	`reflects your o[tp][hp][eo][rs]i?t?e? ring`:                          []mod.Mod{},
	"can be allflame crafted as if rare":                                  []mod.Mod{},
	"cannot gain intangibility":                                           []mod.Mod{},
	"removes all burning when used":                                       []mod.Mod{},
	"elemental hit's added damage cannot be replaced this way":            []mod.Mod{},
	//pob:src/Modules/ModParser.lua:5550-5555
	"you gain added cold damage instead of added damage of other types if dexterity exceeds both other attributes": []mod.Mod{
		FLAG("AllAddedDamageAsCold").Tag(mod.Condition("DexSingleHighestAttribute")),
	},
	`you gain added lightn?ing damage instead of added damage of other types if intelligence exceeds both other attributes`: []mod.Mod{
		FLAG("AllAddedDamageAsLightning").Tag(mod.Condition("IntSingleHighestAttribute")),
	},
	"passives affected are conquered by the abyssal":                              []mod.Mod{},
	"linked targets always count as in range of non-curse auras from your skills": []mod.Mod{},
	"lose adrenaline when you cease to be flame-touched":                          []mod.Mod{},
	`every (\d+) seconds, regenerate (\d+)% of life over one second`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("LifeRegenPercent", "BASE", utils.Float(captures[1])).Tag(mod.Condition("LifeRegenBurstFull")),
			MOD("LifeRegenPercent", "BASE", utils.Float(captures[1])/num).Tag(mod.Condition("LifeRegenBurstAvg")),
		}, ""
	},
	`you take (\d+)% reduced extra damage from critical strikes`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFloat("ReduceCritExtraDamage", mod.TypeBase, num)}, ""
	},
	`you take (\d+)% reduced extra damage from critical strikes while affected by determination`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.Condition("AffectedByDetermination"))}, ""
	},
	`gain adrenaline for (\d+) seconds? on kill`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("Condition:Adrenaline", true).Tag(mod.Condition("KilledRecently"))}, ""
	},
	`gain adrenaline when you become flame\-touched`: []mod.Mod{
		mod.NewFlag("Condition:Adrenaline", true).Tag(mod.Condition("AffectedByApproachingFlames")),
	},
	//pob:block:src/Modules/ModParser.lua:4319-4322
	`(\d+)% increased defences while you have at least four linked targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Defences", "INC", num).Tag(mod.MultiplierThreshold("LinkedTargets").Threshold(4))}, ""
	},
	`your movement speed is equal to the highest movement speed among linked players`: []mod.Mod{
		FLAG("MovementSpeedEqualHighestLinkedPlayers").Tag(mod.MultiplierThreshold("LinkedTargets").Threshold(1)),
	},
	`(\d+)% increased movement speed while you have at least two linked targets`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MovementSpeed", "INC", num).Tag(mod.MultiplierThreshold("LinkedTargets").Threshold(2))}, ""
	},
	`link skills have (\d+)% increased buff effect if you have linked to a target recently`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("BuffEffect", "INC", num).Tag(mod.SkillType(string(data.SkillTypeLink))).Tag(mod.Condition("LinkedRecently"))}, ""
	},
	`link skills can target damageable minions`: []mod.Mod{
		mod.NewFlag("Condition:CanLinkToMinions", true).Tag(mod.Condition("HaveDamageableMinion")),
	},
	`your linked minions take (\d+)% less damage`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("MinionModifier", "LIST", mod.MinionModifier{
			Mod: mod.NewFloat("DamageTaken", mod.TypeMore, -num).Tag(mod.Condition("AffectedByLink")),
		})}, ""
	},
	`curses are inflicted on you instead of linked targets`: []mod.Mod{
		MOD("ExtraLinkEffect", "LIST", mod.MinionModifier{Mod: mod.NewFlag("CurseImmune", true)}),
	},
	`elemental ailments are inflicted on you instead of linked targets`: []mod.Mod{
		MOD("ExtraLinkEffect", "LIST", mod.MinionModifier{Mod: mod.NewFlag("ElementalAilmentImmune", true)}),
	},
	`non\-unique utility flasks you use apply to linked targets`: []mod.Mod{
		MOD("ExtraLinkEffect", "LIST", mod.MinionModifier{
			Mod: mod.NewFlag("ParentNonUniqueFlasksAppliedToYou", true).Tag(mod.GlobalEffect("Global").Unscalable(true)),
		}),
	},
	`linked targets and allies in your link beams have \+(\d+)% to all maximum elemental resistances`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ExtraAura", "LIST", mod.ExtraAura{
				OnlyAllies: true,
				Mod:        MOD("ElementalResistMax", "BASE", num).Tag(mod.Condition("AffectedByLink").Neg(true)),
			}).Tag(mod.MultiplierThreshold("LinkedTargets").Threshold(1)),
			MOD("ExtraLinkEffect", "LIST", mod.MinionModifier{
				Mod: MOD("ElementalResistMax", "BASE", num).Tag(mod.GlobalEffect("Global").Unscalable(true)),
			}),
		}, ""
	},
	//pob:src/Modules/ModParser.lua:2877
	`enemies in your link beams cannot apply elemental ailments`: []mod.Mod{
		FLAG("ElementalAilmentImmune").Tag(mod.ActorCondition("enemy", "BetweenYouAndLinkedTarget")),
	},
	//pob:block:src/Modules/ModParser.lua:2879-2881
	`you can inflict \+(\d+) hallowing flame on enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("Multiplier:HallowingFlameMax", "BASE", num)}, ""
	},
	`gain (\d+)% of (\w+) damage as extra (\w+) damage for each of your hallowing flames that have been removed by an allied hit recently, up to (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD(firstToUpper(captures[1])+"DamageGainAs"+firstToUpper(captures[2]), "BASE", num).
				Tag(mod.ActorCondition("enemy", "HallowingFlame")).
				Tag(mod.Multiplier("HallowingFlameStacksRemovedByAlly").Limit(utils.Float(captures[3]) / num)),
		}, ""
	},
	`(\d+)% increased magnitude of hallowing flame you inflict`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("HallowingFlameMagnitude", "INC", num)}, ""
	},
	`quicksilver flasks you use also apply to nearby allies`: []mod.Mod{mod.NewFlag("QuickSilverAppliesToAllies", true)},
	`attacks have (\d+)% arcane might`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFlag("SpellDamageAppliesToAttacks", true),
			MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", num),
		}, ""
	},
	`attacks have (\d+)% arcane might while wielding a wand`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			mod.NewFlag("SpellDamageAppliesToAttacks", true).Tag(mod.Condition("UsingWand")),
			MOD("ImprovedSpellDamageAppliesToAttacks", "MAX", num).Tag(mod.Condition("UsingWand")),
		}, ""
	},
	`critical strikes with this weapon do not deal extra damage`: []mod.Mod{
		mod.NewFlag("NoCritMultiplier", true).Tag(mod.Condition("{Hand}Attack")),
	},
	`damage cannot be reflected`: []mod.Mod{
		MOD("ReflectedDamageTaken", "MORE", -100).Tag(mod.GlobalEffect("Global").Unscalable(true)),
	},
	`take (\d+) fire damage per second while flame\-touched`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FireDegen", "BASE", num).Tag(mod.Condition("AffectedByApproachingFlames"))}, ""
	},
	`all damage from critical strikes can apply lightning ailments during effect`: []mod.Mod{
		mod.NewFlag("PhysicalCanShock", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("ColdCanShock", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("FireCanShock", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("ChaosCanShock", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("PhysicalCanSap", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("ColdCanSap", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("FireCanSap", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("ChaosCanSap", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
	},
	`all damage from critical strikes can apply cold ailments during effect`: []mod.Mod{
		mod.NewFlag("PhysicalCanChill", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("LightningCanChill", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("FireCanChill", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("ChaosCanChill", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("PhysicalCanFreeze", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("LightningCanFreeze", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("FireCanFreeze", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("ChaosCanFreeze", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("PhysicalCanBrittle", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("LightningCanBrittle", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("FireCanBrittle", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
		mod.NewFlag("ChaosCanBrittle", true).Tag(mod.Condition("UsingFlask")).Tag(mod.Condition("NeverCrit").Neg(true)),
	},
	`increases and reductions to cast speed apply to attack speed`: []mod.Mod{
		mod.NewFlag("CastSpeedAppliesToAttacks", true),
	},
	`damage penetrates fire resistance equal to your overcapped fire resistance`: []mod.Mod{
		mod.NewFlag("FirePenIncreasedByUncappedFireRes", true),
	},
	`damage penetrates fire resistance equal to your overcapped fire resistance, up to a maximum of (\d+)%`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("FirePenetration", "BASE", 1).
			Tag(mod.PerStat(1, "FireResistOverCap").Limit(num).LimitTotal(true))}, ""
	},
	`maximum number of summoned golems is doubled`: []mod.Mod{
		MOD("ActiveGolemLimit", "MORE", 100).
			Tag(mod.Multiplier("ActiveGolemLimitDoubled").GlobalLimit(100).GlobalLimitKey("ActiveGolemLimitDoubledLimit")),
		MOD("Multiplier:ActiveGolemLimitDoubled", "OVERRIDE", 1),
	},
	`maximum number of raised spectres is doubled`: []mod.Mod{
		MOD("ActiveSpectreLimit", "MORE", 100).
			Tag(mod.Multiplier("ActiveSpectreLimitDoubled").GlobalLimit(100).GlobalLimitKey("ActiveSpectreLimitDoubledLimit")),
		MOD("Multiplier:ActiveSpectreLimitDoubled", "OVERRIDE", 1),
	},
	`maximum number of summoned phantasms is doubled`: []mod.Mod{
		MOD("ActivePhantasmLimit", "MORE", 100).
			Tag(mod.Multiplier("ActivePhantasmLimitDoubled").GlobalLimit(100).GlobalLimitKey("ActivePhantasmLimitDoubledLimit")),
		MOD("Multiplier:ActivePhantasmLimitDoubled", "OVERRIDE", 1),
	},
	`maximum recovery per energy shield leech is doubled`: []mod.Mod{
		MOD("MaxEnergyShieldLeechInstance", "MORE", 100).
			Tag(mod.Multiplier("MaxEnergyShieldLeechInstanceDoubled").GlobalLimit(100).GlobalLimitKey("MaxEnergyShieldLeechInstanceDoubledLimit")),
		MOD("Multiplier:MaxEnergyShieldLeechInstanceDoubled", "OVERRIDE", 1),
	},
	`take (\d+)% less damage from hits for (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DamageTakenWhenHit", "MORE", -num).Tag(mod.Condition("HeartstopperHIT")),
			MOD("DamageTakenWhenHit", "MORE", -num*utils.Float(captures[1])/10).Tag(mod.Condition("HeartstopperAVERAGE")),
		}, ""
	},
	`take (\d+)% less damage over time for (\d+) seconds`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("DamageTakenOverTime", "MORE", -num).Tag(mod.Condition("HeartstopperDOT")),
			MOD("DamageTakenOverTime", "MORE", -num*utils.Float(captures[1])/10).Tag(mod.Condition("HeartstopperAVERAGE")),
		}, ""
	},
	`grants immunity to corrupted blood for (\d+) seconds if used while affected by corrupted blood`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{mod.NewFlag("CorruptedBloodImmune", true).Tag(mod.Condition("UsingFlask"))}, ""
	},
	`skills gain a base life cost equal to (\d+)% of base mana cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaCostAsLifeCost", "BASE", num)}, ""
	},
	`skills gain a base energy shield cost equal to (\d+)% of base mana cost`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ManaCostAsEnergyShieldCost", "BASE", num)}, ""
	},
	`you take (\d+)% reduced extra damage from critical strikes while you have no power charges`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{
			MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.StatThreshold("PowerCharges", 0).Upper(true)),
		}, ""
	},
	`you take (\d+)% reduced extra damage from critical strikes by poisoned enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.ActorCondition("enemy", "Poisoned"))}, ""
	},
	`you take (\d+)% reduced extra damage from critical strikes by cursed enemies`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", num).Tag(mod.ActorCondition("enemy", "Cursed"))}, ""
	},
	`you take (\d+)% increased extra damage from critical strikes`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ReduceCritExtraDamage", "BASE", -num)}, ""
	},
	`nearby allies have (\d+)% chance to block attack damage per (\d+) strength you have`: func(num float64, captures []string) ([]mod.Mod, string) {
		return []mod.Mod{MOD("ExtraAura", "LIST", mod.ExtraAura{OnlyAllies: true, Mod: mod.NewFloat("BlockChance", mod.TypeBase, num)}).Tag(mod.PerStat(utils.Float(captures[1]), "Str"))}, ""
	},
}

// for _, name in ipairs(data.keystones) do
//
//	flag("Condition:Have"..firstToUpper(name):gsub(" %l", string.upper):gsub(" ", "")) }
//
// end
//
//pob:block:src/Modules/ModParser.lua:5878-5880
func init() {
	for _, name := range data.Keystones {
		specialModList[strings.ToLower(name)] = []mod.Mod{
			MOD("Keystone", "LIST", name),
			mod.NewFlag("Condition:Have"+keystoneConditionSuffix(name), true),
		}
	}

	//pob:block:src/Modules/ModParser.lua:5881-5883
	for _, name := range data.ClusterJewels.Keystones {
		specialModList[strings.ToLower(name)] = []mod.Mod{
			MOD("Keystone", "LIST", name),
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

var regenTypes = resourceTypesWithSuffix("Regen")

var degenTypesCompiled map[string]CompiledList[modNameListType]
var degenTypes = resourceTypesWithSuffix("Degen")

var costTypesCompiled map[string]CompiledList[modNameListType]
var costTypes = resourceTypesWithSuffix("Cost")

var baseCostTypesCompiled map[string]CompiledList[modNameListType]
var baseCostTypes = resourceTypesWithSuffix("CostNoMult")

var resourceTypes = withMaximumResourceTypes(resourceTypesBase)

//pob:block:src/Modules/ModParser.lua:5968-5976
func withMaximumResourceTypes(base map[string][]string) map[string][]string {
	out := make(map[string][]string, len(base)*2)
	for line, names := range base {
		out[line] = names
		out["maximum "+line] = names
	}
	return out
}

//pob:block:src/Modules/ModParser.lua:5950-5967
var resourceTypesBase = map[string][]string{
	"life":                         {"Life"},
	"mana":                         {"Mana"},
	"energy shield":                {"EnergyShield"},
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

//pob:src/Modules/ModParser.lua:5977-5990
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
	`acceleration shrine buff`:     {names: []string{"Condition:AccelerationShrine"}},
	`adrenaline`:                   {names: []string{"Condition:Adrenaline"}},
	`arcane surge`:                 {names: []string{"Condition:ArcaneSurge"}},
	`brutal shrine buff`:           {names: []string{"Condition:BrutalShrine"}},
	`chaotic might`:                {names: []string{"Condition:ChaoticMight"}},
	`debilitated`:                  {names: []string{"Condition:Debilitated"}},
	`diamond shrine buff`:          {names: []string{"Condition:DiamondShrine"}},
	`echoing shrine buff`:          {names: []string{"Condition:EchoingShrine"}},
	`elusive`:                      {names: []string{"Condition:CanBeElusive"}},
	`gloom shrine buff`:            {names: []string{"Condition:GloomShrine"}},
	`greater freezing shrine buff`: {names: []string{"Condition:GreaterFreezingShrine"}},
	`greater shocking shrine buff`: {names: []string{"Condition:GreaterShockingShrine"}},
	`greater skeletal shrine buff`: {names: []string{"Condition:GreaterSkeletalShrine"}},
	`impenetrable shrine buff`:     {names: []string{"Condition:ImpenetrableShrine"}},
	`lesser brutal shrine buff`:    {names: []string{"Condition:LesserBrutalShrine"}},
	`lesser massive shrine buff`:   {names: []string{"Condition:LesserMassiveShrine"}},
	`malediction`:                  {names: []string{"HasMalediction"}},
	`massive shrine buff`:          {names: []string{"Condition:MassiveShrine"}},
	`rampage`:                      {names: []string{"Condition:Rampage"}},
	`replenishing shrine buff`:     {names: []string{"Condition:ReplenishingShrine"}},
	`resistance shrine buff`:       {names: []string{"Condition:ResistanceShrine"}},
	`resonating shrine buff`:       {names: []string{"Condition:ResonatingShrine"}},
	`soul eater`:                   {names: []string{"Condition:CanHaveSoulEater"}},
	`phasing`:                      {names: []string{"Condition:Phasing"}},
	`onslaught`:                    {names: []string{"Condition:Onslaught"}},
	`fortify`:                      {names: []string{"Condition:Fortified"}},
	`fortified`:                    {names: []string{"Condition:Fortified"}},
	`unholy might`:                 {names: []string{"Condition:UnholyMight"}},
	`tailwind`:                     {names: []string{"Condition:Tailwind"}},
	`intimidated`:                  {names: []string{"Condition:Intimidated"}},
	`crushed`:                      {names: []string{"Condition:Crushed"}},
	`chilled`:                      {names: []string{"Condition:Chilled"}},
	`blinded`:                      {names: []string{"Condition:Blinded"}},
	`no life regeneration`:         {names: []string{"NoLifeRegen"}},
	`hexproof`:                     {names: []string{"CurseEffectOnSelf"}, flagModType: "MORE", flagModValue: -100},
	`hindered, with (\d+)\% reduced movement speed`: {names: []string{"Condition:Hindered"}},
	`unnerved`: {names: []string{"Condition:Unnerved"}},
}

// Build active skill name lookup
var skillNameListCompiled map[string]CompiledList[modNameListType]
var preSkillNameListCompiled map[string]CompiledList[modNameListType]

//pob:block:src/Modules/ModParser.lua:4124-4149
func initializeSkillNameList() {
	skillNameListCompiled = make(map[string]CompiledList[modNameListType])
	skillNameListCompiled[" corpse cremation "] = CompiledList[modNameListType]{
		Regex: lazyRegex(` corpse cremation `),
		Value: modNameListType{
			tag: mod.SkillName("Cremation").Transfigured(true),
		},
	}

	preSkillNameListCompiled = make(map[string]CompiledList[modNameListType])
	addGem := func(pobGem raw2.PoBGem) {
		grantedEffect := poe.GrantedEffectByID(pobGem.GrantedEffectID)
		if grantedEffect != nil && !grantedEffect.IsSupport && grantedEffect.GetActiveSkill() != nil {
			skillName := grantedEffect.GetActiveSkill().DisplayedName

			if skillName == "" || skillName == "..." || strings.Contains(skillName, "NOT CURRENTLY USED") || strings.Contains(skillName, "UNUSED") {
				return
			}

			skillNameLower := strings.ToLower(skillName)

			val := modNameListType{
				tag: mod.SkillName(skillName).Transfigured(true),
			}

			skillNameListCompiled[" "+skillNameLower+" "] = CompiledList[modNameListType]{
				Regex: lazyRegex(" " + skillNameLower + " "),
				Value: val,
			}

			preSkillNameListCompiled["^"+skillNameLower+" "] = CompiledList[modNameListType]{
				Regex: lazyRegex("^" + skillNameLower + " "),
				Value: val,
			}

			preSkillNameListCompiled["^"+skillNameLower+" has ?a? "] = CompiledList[modNameListType]{
				Regex: lazyRegex("^" + skillNameLower + " has ?a? "),
				Value: val,
			}

			preSkillNameListCompiled["^"+skillNameLower+" deals "] = CompiledList[modNameListType]{
				Regex: lazyRegex("^" + skillNameLower + " deals "),
				Value: val,
			}

			preSkillNameListCompiled["^"+skillNameLower+" damage "] = CompiledList[modNameListType]{
				Regex: lazyRegex("^" + skillNameLower + " damage "),
				Value: val,
			}

			if pobGem.Tags["totem"] {
				preSkillNameListCompiled["^"+skillNameLower+" totem deals "] = CompiledList[modNameListType]{
					Regex: lazyRegex("^" + skillNameLower + " totem deals "),
					Value: val,
				}

				preSkillNameListCompiled["^"+skillNameLower+" totem grants "] = CompiledList[modNameListType]{
					Regex: lazyRegex("^" + skillNameLower + " totem grants "),
					Value: modNameListType{
						addToSkill: mod.SkillName(skillName).Transfigured(true),
						tag:        mod.GlobalEffect("Buff"),
					},
				}
			}

			if slices.Contains(grantedEffect.GetActiveSkill().ActiveSkillTypes, poe.ActiveSkillTypesByID["Buff"].Key) ||
				slices.Contains(raw2.SkillBaseFlags[grantedEffect.ID], "buff") {
				preSkillNameListCompiled["^"+skillNameLower+" grants "] = CompiledList[modNameListType]{
					Regex: lazyRegex("^" + skillNameLower + " grants "),
					Value: modNameListType{
						addToSkill: mod.SkillName(skillName).Transfigured(true),
						tag:        mod.GlobalEffect("Buff"),
					},
				}

				preSkillNameListCompiled["^"+skillNameLower+" grants a?n? ?additional "] = CompiledList[modNameListType]{
					Regex: lazyRegex("^" + skillNameLower + " grants a?n? ?additional "),
					Value: modNameListType{
						addToSkill: mod.SkillName(skillName).Transfigured(true),
						tag:        mod.GlobalEffect("Buff"),
					},
				}
			}

			if slices.Contains(grantedEffect.GetActiveSkill().ActiveSkillTypes, poe.ActiveSkillTypesByID["Aura"].Key) ||
				slices.Contains(grantedEffect.GetActiveSkill().ActiveSkillTypes, poe.ActiveSkillTypesByID["Herald"].Key) {
				affectedBy := modNameListType{
					tag: mod.Condition("AffectedBy" + strings.ReplaceAll(skillName, " ", "")),
				}

				skillNameListCompiled["while affected by "+skillNameLower] = CompiledList[modNameListType]{
					Regex: lazyRegex("while affected by " + regexp.QuoteMeta(skillNameLower)),
					Value: affectedBy,
				}

				skillNameListCompiled["while using "+skillNameLower] = CompiledList[modNameListType]{
					Regex: lazyRegex("while using " + regexp.QuoteMeta(skillNameLower)),
					Value: affectedBy,
				}
			}

			if pobGem.Tags["curse"] {
				skillNameListCompiled["if you've cast "+skillNameLower+" in the past (\\d+) seconds"] = CompiledList[modNameListType]{
					Regex: lazyRegex("if you've cast " + regexp.QuoteMeta(skillNameLower) + " in the past (\\d+) seconds"),
					Value: modNameListType{
						tag: mod.Condition(selfCastCondition(skillName)),
					},
				}
			}

			addSkillSpecial(skillNameLower, pobGem.Tags, skillName)
		}
	}
	for _, gemData := range poe.SkillGems {
		pobGemEntry, inPoB := raw2.Gems[gemBaseID(gemData)]
		if !inPoB {
			continue
		}
		addGem(pobGemEntry)
	}
	for _, id := range slices.Sorted(maps.Keys(raw2.GemVariants)) {
		addGem(raw2.GemVariants[id])
	}

	preSkillNameIndex = newAnchoredIndex(preSkillNameListCompiled)
	skillNameIndex = newAnyOffsetIndex(skillNameListCompiled)
	specialModIndex = newAnchoredIndex(specialModListCompiled)
}

func gemBaseID(gemData *poe.SkillGem) string {
	base := gemData.GetBaseItemType()
	if base == nil {
		return ""
	}
	return base.ID
}

//pob:block:src/Modules/ModParser.lua:4153-4176
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
			extra(MOD("ProjectileCount", "BASE", 1).KeywordFlag(mod.KeywordFlagArrow)))
		add("^"+skillNameLower+` fires (\d+) additional arrows?`, func(num float64, captures []string) ([]mod.Mod, string) {
			return extra(MOD("ProjectileCount", "BASE", num).KeywordFlag(mod.KeywordFlagArrow)), ""
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

//pob:src/Modules/ModParser.lua:4181-4200
func getSimpleConv(srcList []string, dst string, modType mod.Type, remove bool, factor ...float64) radiusJewelFunc {
	return getSimpleConvSrc(srcList, dst, modType, remove, "", factor...)
}

//pob:block:src/Modules/ModParser.lua:18-39
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

//pob:src/Modules/ModParser.lua:6311-6319
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
	bowTransformMask1 = mod.MFlagAxe | mod.MFlagClaw | mod.MFlagDagger | mod.MFlagMace |
		mod.MFlagStaff | mod.MFlagSword | mod.MFlagMelee
	bowTransformMask2 = mod.MFlagWeapon1H | mod.MFlagWeaponMelee
	bowTransformMask3 = mod.MFlagWeapon2H | mod.MFlagWeaponMelee
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

//pob:block:src/Modules/ModParser.lua:6144-6172
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
				flags&^(bowTransformMask1|bowTransformMask2|bowTransformMask3)|mod.MFlagBow, m.Tags())
			continue
		}

		if rewritten, ok := rewriteUsingConditionToBow(m.Tags()); ok {
			bowTransformMerge(out, m, -value.Float(), flags, m.Tags())
			bowTransformMerge(out, m, value.Float(), flags, rewritten)
		}
	}
}

//pob:block:src/Modules/ModParser.lua:6151-6152
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

//pob:block:src/Modules/ModParser.lua:6153-6168
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
	"Strength from Passives in Radius is Transformed to Dexterity":     getSimpleConv([]string{"Str"}, "Dex", mod.TypeBase, true),
	"Dexterity from Passives in Radius is Transformed to Strength":     getSimpleConv([]string{"Dex"}, "Str", mod.TypeBase, true),
	"Strength from Passives in Radius is Transformed to Intelligence":  getSimpleConv([]string{"Str"}, "Int", mod.TypeBase, true),
	"Intelligence from Passives in Radius is Transformed to Strength":  getSimpleConv([]string{"Int"}, "Str", mod.TypeBase, true),
	"Dexterity from Passives in Radius is Transformed to Intelligence": getSimpleConv([]string{"Dex"}, "Int", mod.TypeBase, true),
	"Intelligence from Passives in Radius is Transformed to Dexterity": getSimpleConv([]string{"Int"}, "Dex", mod.TypeBase, true),

	"Increases and Reductions to Life in Radius are Transformed to apply to Energy Shield":          getSimpleConv([]string{"Life"}, "EnergyShield", mod.TypeIncrease, true),
	"Increases and Reductions to Physical Damage in Radius are Transformed to apply to Cold Damage": getSimpleConv([]string{"PhysicalDamage"}, "ColdDamage", mod.TypeIncrease, true),
	"Increases and Reductions to Cold Damage in Radius are Transformed to apply to Physical Damage": getSimpleConv([]string{"ColdDamage"}, "PhysicalDamage", mod.TypeIncrease, true),
	"Increases and Reductions to Evasion Rating in Radius are Transformed to apply to Armour":       getSimpleConv([]string{"Evasion"}, "Armour", mod.TypeIncrease, true),
	"Increases and Reductions to other Damage Types in Radius are Transformed to apply to Fire Damage": getSimpleConv(
		[]string{"PhysicalDamage", "ColdDamage", "LightningDamage", "ChaosDamage", "ElementalDamage"}, "FireDamage", mod.TypeIncrease, true),

	"Melee and Melee Weapon Type modifiers in Radius are Transformed to Bow Modifiers": bowTransform,

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
	//pob:src/Modules/ModParser.lua:6197
	"Allocated Notable Passive Skills in Radius grant nothing": func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
		if node != nil && node.nodeType == "Notable" {
			out.AddMod(mod.NewFlag("AllocatedPassiveSkillHasNoEffect", true).Source(d.modSource))
		}
	},
}

var jewelSelfFuncs = map[string]radiusJewelFunc{
	"Adds 1 to maximum Life per 3 Intelligence in Radius":                                  getPerStat("Life", mod.TypeBase, 0, "Int", 1.0/3),
	"Adds 1 to Maximum Life per 3 Intelligence Allocated in Radius":                        getPerStat("Life", mod.TypeBase, 0, "Int", 1.0/3),
	"1% increased Evasion Rating per 3 Dexterity Allocated in Radius":                      getPerStat("Evasion", mod.TypeIncrease, 0, "Dex", 1.0/3),
	"1% increased Claw Physical Damage per 3 Dexterity Allocated in Radius":                getPerStat("PhysicalDamage", mod.TypeIncrease, mod.MFlagClaw, "Dex", 1.0/3),
	"1% increased Melee Physical Damage while Unarmed per 3 Dexterity Allocated in Radius": getPerStat("PhysicalDamage", mod.TypeIncrease, mod.MFlagUnarmed, "Dex", 1.0/3),
	"3% increased Totem Life per 10 Strength in Radius":                                    getPerStat("TotemLife", mod.TypeIncrease, 0, "Str", 3.0/10),
	"3% increased Totem Life per 10 Strength Allocated in Radius":                          getPerStat("TotemLife", mod.TypeIncrease, 0, "Str", 3.0/10),
	"Adds 1 maximum Lightning Damage to Attacks per 1 Dexterity Allocated in Radius":       getPerStat("LightningMax", mod.TypeBase, mod.MFlagAttack, "Dex", 1),
	"5% increased Chaos damage per 10 Intelligence from Allocated Passives in Radius":      getPerStat("ChaosDamage", mod.TypeIncrease, 0, "Int", 5.0/10),

	"Dexterity and Intelligence from passives in Radius count towards Strength Melee Damage bonus": func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
		if node != nil {
			d.stats["Dex"] += node.modList.Sum(mod.TypeBase, nil, "Dex")
			d.stats["Int"] += node.modList.Sum(mod.TypeBase, nil, "Int")
			return
		}
		_, hasDex := d.stats["Dex"]
		_, hasInt := d.stats["Int"]
		if !hasDex && !hasInt {
			return
		}
		out.NewMod("DexIntToMeleeBonus", mod.TypeBase, d.stats["Dex"]+d.stats["Int"], d.modSource)
	},

	"-1 Strength per 1 Strength on Allocated Passives in Radius":                                             getPerStat("Str", mod.TypeBase, 0, "Str", -1),
	"1% additional Physical Damage Reduction per 10 Strength on Allocated Passives in Radius":                getPerStat("PhysicalDamageReduction", mod.TypeBase, 0, "Str", 1.0/10),
	"2% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius":                        getPerStat("LifeRecoveryRate", mod.TypeIncrease, 0, "Str", 2.0/10),
	"3% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius":                        getPerStat("LifeRecoveryRate", mod.TypeIncrease, 0, "Str", 3.0/10),
	"-1 Intelligence per 1 Intelligence on Allocated Passives in Radius":                                     getPerStat("Int", mod.TypeBase, 0, "Int", -1),
	"0.4% of Energy Shield Regenerated per Second for every 10 Intelligence on Allocated Passives in Radius": getPerStat("EnergyShieldRegenPercent", mod.TypeBase, 0, "Int", 0.4/10),
	"2% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius":                    getPerStat("ManaRecoveryRate", mod.TypeIncrease, 0, "Int", 2.0/10),
	"3% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius":                    getPerStat("ManaRecoveryRate", mod.TypeIncrease, 0, "Int", 3.0/10),
	"-1 Dexterity per 1 Dexterity on Allocated Passives in Radius":                                           getPerStat("Dex", mod.TypeBase, 0, "Dex", -1),
	"2% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius":                           getPerStat("MovementSpeed", mod.TypeIncrease, 0, "Dex", 2.0/10),
	"3% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius":                           getPerStat("MovementSpeed", mod.TypeIncrease, 0, "Dex", 3.0/10),
}

var jewelSelfUnallocFuncs = map[string]radiusJewelFunc{
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

	//pob:block:src/Modules/ModParser.lua:6362-6392
	"Grants all bonuses of Unallocated Small Passive Skills in Radius":   grantUnallocatedBonuses("Normal"),
	"Grants all bonuses of Unallocated Notable Passive Skills in Radius": grantUnallocatedBonuses("Notable"),
}

//pob:block:src/Modules/ModParser.lua:6362-6392
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
	{
		pattern: `passive skills in radius also grant (\d+)% increased global critical strike chance`,
		typ:     "Other",
		build:   radiusGrantSimple("CritChance", mod.TypeIncrease),
	},
	{
		pattern: `passive skills in radius also grant \+(\d+) to maximum life`,
		typ:     "Other",
		build:   radiusGrantSimple("Life", mod.TypeBase),
	},
	{
		pattern: `passive skills in radius also grant \+(\d+) to maximum mana`,
		typ:     "Other",
		build:   radiusGrantSimple("Mana", mod.TypeBase),
	},
	{
		pattern: `passive skills in radius also grant (\d+)% increased energy shield`,
		typ:     "Other",
		build:   radiusGrantSimple("EnergyShield", mod.TypeIncrease),
	},
	{
		pattern: `passive skills in radius also grant (\d+)% increased armour`,
		typ:     "Other",
		build:   radiusGrantSimple("Armour", mod.TypeIncrease),
	},
	{
		pattern: `passive skills in radius also grant (\d+)% increased evasion rating`,
		typ:     "Other",
		build:   radiusGrantSimple("Evasion", mod.TypeIncrease),
	},
	{
		pattern: `passive skills in radius also grant \+(\d+) to all attributes`,
		typ:     "Other",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
				if !radiusGrantAppliesTo(node) {
					return
				}
				for _, name := range []string{"Str", "Dex", "Int", "All"} {
					out.AddMod(mod.NewFloat(name, mod.TypeBase, num).Source(d.modSource))
				}
			}
		},
	},
	{
		pattern: `passive skills in radius also grant \+(\d+)% to chaos resistance`,
		typ:     "Other",
		build:   radiusGrantSimple("ChaosResist", mod.TypeBase),
	},
	{
		pattern: `passive skills in radius also grant (\d+)% increased (\w+) damage`,
		typ:     "Other",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			name := firstToUpper(captures[1]) + "Damage"
			return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
				if !radiusGrantAppliesTo(node) {
					return
				}
				out.AddMod(mod.NewFloat(name, mod.TypeIncrease, num).Source(d.modSource))
			}
		},
	},
	{
		pattern: `passive skills in radius also grant: traps and mines deal (\d+) to (\d+) added physical damage`,
		typ:     "Other",
		build: func(captures []string) radiusJewelFunc {
			minimum, maximum := utils.Float(captures[0]), utils.Float(captures[1])
			return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
				if node == nil || node.nodeType == "Keystone" {
					return
				}
				out.AddMod(mod.NewFloat("PhysicalMin", mod.TypeBase, minimum).Source(d.modSource).
					KeywordFlag(mod.KeywordFlagTrap | mod.KeywordFlagMine))
				out.AddMod(mod.NewFloat("PhysicalMax", mod.TypeBase, maximum).Source(d.modSource).
					KeywordFlag(mod.KeywordFlagTrap | mod.KeywordFlagMine))
			}
		},
	},
	{
		pattern: `passive skills in radius also grant: (\d+)% increased unarmed attack speed with melee skills`,
		typ:     "Other",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
				if node == nil || node.nodeType == "Keystone" {
					return
				}
				out.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, num).Source(d.modSource).
					Flag(mod.MFlagUnarmed | mod.MFlagAttack | mod.MFlagMelee))
			}
		},
	},
	{
		pattern: `passives granting lightning resistance or all elemental resistances in radius also grant chance to block spells? ?d?a?m?a?g?e? at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"LightningResist", "ElementalResist"}, "SpellBlockChance", mod.TypeBase, false, ""),
	},
	{
		pattern: `passives granting lightning resistance or all elemental resistances in radius also grant increased maximum energy shield at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"LightningResist", "ElementalResist"}, "EnergyShield", mod.TypeIncrease, false, mod.TypeBase),
	},
	{
		pattern: `passives granting lightning resistance or all elemental resistances in radius also grant lightning damage converted to chaos damage at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"LightningResist", "ElementalResist"}, "LightningDamageConvertToChaos", mod.TypeBase, false, ""),
	},
	{
		pattern: `passives granting cold resistance or all elemental resistances in radius also grant chance to dodge attacks? ?h?i?t?s? at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"ColdResist", "ElementalResist"}, "AttackDodgeChance", mod.TypeBase, false, ""),
	},
	{
		pattern: `passives granting cold resistance or all elemental resistances in radius also grant chance to suppress spell damage at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"ColdResist", "ElementalResist"}, "SpellSuppressionChance", mod.TypeBase, false, ""),
	},
	{
		pattern: `passives granting cold resistance or all elemental resistances in radius also grant increased maximum mana at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"ColdResist", "ElementalResist"}, "Mana", mod.TypeIncrease, false, mod.TypeBase),
	},
	{
		pattern: `passives granting cold resistance or all elemental resistances in radius also grant cold damage converted to chaos damage at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"ColdResist", "ElementalResist"}, "ColdDamageConvertToChaos", mod.TypeBase, false, ""),
	},
	{
		pattern: `passives granting fire resistance or all elemental resistances in radius also grant chance to block attack damage at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"FireResist", "ElementalResist"}, "BlockChance", mod.TypeBase, false, ""),
	},
	{
		pattern: `passives granting fire resistance or all elemental resistances in radius also grant chance to block at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"FireResist", "ElementalResist"}, "BlockChance", mod.TypeBase, false, ""),
	},
	{
		pattern: `passives granting fire resistance or all elemental resistances in radius also grant increased maximum life at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"FireResist", "ElementalResist"}, "Life", mod.TypeIncrease, false, mod.TypeBase),
	},
	{
		pattern: `passives granting fire resistance or all elemental resistances in radius also grant fire damage converted to chaos damage at (\d+)% of its value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"FireResist", "ElementalResist"}, "FireDamageConvertToChaos", mod.TypeBase, false, ""),
	},
	{
		pattern: `increases and reductions to energy shield in radius are transformed to apply to armour at (\d+)% of their value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"EnergyShield"}, "Armour", mod.TypeIncrease, true, ""),
	},
	{
		pattern: `increases and reductions to life in radius are transformed to apply to mana at (\d+)% of their value`,
		typ:     "Other",
		build:   simpleConvPercent([]string{"Life"}, "Mana", mod.TypeIncrease, true, ""),
	},
	{
		pattern: `with at least 40 dexterity in radius, dual strike has (\d+)% increased attack speed while wielding a claw`,
		typ:     "Threshold",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			return getThreshold([]string{"Dex"}, MOD("Speed", mod.TypeIncrease, num).
				Tag(mod.SkillName("Dual Strike").Transfigured(true), mod.Condition("UsingClaw")))
		},
	},
	{
		pattern: `with at least 40 dexterity in radius, dual strike has \+(\d+)% to critical strike multiplier while wielding a dagger`,
		typ:     "Threshold",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			return getThreshold([]string{"Dex"}, MOD("CritMultiplier", mod.TypeBase, num).
				Tag(mod.SkillName("Dual Strike").Transfigured(true), mod.Condition("UsingDagger")))
		},
	},
	{
		pattern: `with at least 40 dexterity in radius, dual strike has (\d+)% increased accuracy rating while wielding a sword`,
		typ:     "Threshold",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			return getThreshold([]string{"Dex"}, MOD("Accuracy", mod.TypeIncrease, num).
				Tag(mod.SkillName("Dual Strike").Transfigured(true), mod.Condition("UsingSword")))
		},
	},
	//pob:block:src/Modules/ModParser.lua:6173-6191
	{
		pattern: `(\d+)% increased effect of non-keystone passive skills in radius`,
		typ:     "Other",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
				if node != nil && node.nodeType != "Keystone" && node.nodeType != "ClassStart" {
					out.NewMod("PassiveSkillEffect", mod.TypeIncrease, num, d.modSource)
				}
			}
		},
	},
	{
		pattern: `(\d+)% increased effect of tattoos in radius`,
		typ:     "Other",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
				if node != nil && node.isTattoo {
					out.NewMod("PassiveSkillEffect", mod.TypeIncrease, num, d.modSource)
				}
			}
		},
	},
	//pob:block:src/Modules/ModParser.lua:6283-6307
	{
		pattern: `notable passive skills in radius are transformed to instead grant: (\d+)% increased mana cost of skills and (\d+)% increased spell damage`,
		typ:     "Other",
		build: func(captures []string) radiusJewelFunc {
			manaCost := utils.Float(captures[0])
			spellDamage := utils.Float(captures[1])
			return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
				if node == nil || node.nodeType != "Notable" {
					return
				}
				out.AddMod(mod.NewFlag("PassiveSkillHasOtherEffect", true).Source(d.modSource))
				out.AddMod(mod.NewList("NodeModifier", nodeModifier{
					Mod: mod.NewFloat("ManaCost", mod.TypeIncrease, manaCost).Source(d.modSource),
				}).Source(d.modSource))
				out.AddMod(mod.NewList("NodeModifier", nodeModifier{
					Mod: mod.NewFloat("Damage", mod.TypeIncrease, spellDamage).Source(d.modSource).Flag(mod.MFlagSpell),
				}).Source(d.modSource))
			}
		},
	},
	{
		pattern: `notable passive skills in radius are transformed to instead grant: minions take (\d+)% increased damage`,
		typ:     "Other",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
				if node == nil || node.nodeType != "Notable" {
					return
				}
				out.AddMod(mod.NewFlag("PassiveSkillHasOtherEffect", true).Source(d.modSource))
				out.AddMod(mod.NewList("NodeModifier", nodeModifier{
					Mod: mod.NewList("MinionModifier", mod.MinionModifier{
						Mod: mod.NewFloat("DamageTaken", mod.TypeIncrease, num).Source(d.modSource),
					}),
				}).Source(d.modSource))
			}
		},
	},
	{
		pattern: `notable passive skills in radius are transformed to instead grant: minions have (\d+)% reduced movement speed`,
		typ:     "Other",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
				if node == nil || node.nodeType != "Notable" {
					return
				}
				out.AddMod(mod.NewFlag("PassiveSkillHasOtherEffect", true).Source(d.modSource))
				out.AddMod(mod.NewList("NodeModifier", nodeModifier{
					Mod: mod.NewList("MinionModifier", mod.MinionModifier{
						Mod: mod.NewFloat("MovementSpeed", mod.TypeIncrease, -num).Source(d.modSource),
					}),
				}).Source(d.modSource))
			}
		},
	},
	{
		pattern: `with at least 40 intelligence in radius, (\d+)% of damage taken recouped as mana if you've warcried recently`,
		typ:     "Threshold",
		build: func(captures []string) radiusJewelFunc {
			return getThreshold([]string{"Int"}, MOD("ManaRecoup", mod.TypeBase, utils.Float(captures[0])).
				Tag(mod.Condition("UsedWarcryRecently")))
		},
	},
	{
		pattern: `with at least 40 intelligence in radius, fireball projectiles gain radius as they travel farther, up to \+(\d+) radius`,
		typ:     "Threshold",
		build: func(captures []string) radiusJewelFunc {
			return getThreshold([]string{"Int"}, MOD("AreaOfEffect", mod.TypeBase, utils.Float(captures[0])).
				Tag(mod.DistanceRamp([][]float64{{0, 0}, {50, 1}})))
		},
	},
	{
		pattern: `with at least 40 intelligence in radius, projectiles gain radius as they travel farther, up to a maximum of \+([\d\.]+) metres? to radius`,
		typ:     "Threshold",
		build: func(captures []string) radiusJewelFunc {
			return getThreshold([]string{"Int"}, MOD("AreaOfEffect", mod.TypeBase, utils.Float(captures[0])*10).
				Tag(mod.DistanceRamp([][]float64{{0, 0}, {50, 1}})))
		},
	},
	{
		pattern: `with at least 40 intelligence in radius, fireball has \+(\d+)% chance to inflict scorch`,
		typ:     "Threshold",
		build: func(captures []string) radiusJewelFunc {
			num := utils.Float(captures[0])
			return getThreshold([]string{"Int"}, MOD("EnemyScorchChance", mod.TypeBase, num).
				Tag(mod.SkillName("Fireball")))
		},
	},
}

var (
	jewelFuncList     map[string]jewelFuncEntry
	jewelFuncPatterns []jewelFuncPattern
)

//pob:block:src/Modules/ModParser.lua:4456-4486
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
	//pob:block:src/Modules/ModParser.lua:6521-6535
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

//pob:block:src/Modules/ModParser.lua:6496-6513
func guardConquered(fn radiusJewelFunc) radiusJewelFunc {
	return func(node *radiusJewelNode, out *moddb.ModList, d *radiusJewelData) {
		if node != nil && node.conquered {
			return
		}
		fn(node, out, d)
	}
}

//pob:block:src/Modules/ModParser.lua:6586-6589
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

//pob:src/Modules/ModParser.lua:6397-6421
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

//pob:block:src/Modules/ModParser.lua:6522-6531
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

//pob:block:src/Modules/ModParser.lua:6486-6489
var jewelThresholdFuncLists = map[string][]radiusJewelFunc{
	"With at least 40 Intelligence in Radius, Raised Spectres have a 50% chance to gain Soul Eater for 20 seconds on Kill": {
		getThreshold([]string{"Int"}, MOD("MinionModifier", "LIST", mod.MinionModifier{
			Mod: mod.NewFlag("Condition:CanHaveSoulEater", true),
		}).Tag(mod.SkillName("Raise Spectre").Transfigured(true))),
		getThreshold([]string{"Int"}, mod.NewFlag("Condition:MinionCanHaveSoulEater", true)),
	},
}

const jewelThresholdMinimum = 40

//pob:block:src/Modules/ModParser.lua:6415-6417
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
	"With at least 40 Dexterity in Radius, Frost Blades Melee Damage Penetrates 15% Cold Resistance":          getThreshold([]string{"Dex"}, MOD("ColdPenetration", mod.TypeBase, 15).Flag(mod.MFlagMelee).Tag(mod.SkillName("Frost Blades").Transfigured(true))),
	"With at least 40 Dexterity in Radius, Melee Damage dealt by Frost Blades Penetrates 15% Cold Resistance": getThreshold([]string{"Dex"}, MOD("ColdPenetration", mod.TypeBase, 15).Flag(mod.MFlagMelee).Tag(mod.SkillName("Frost Blades").Transfigured(true))),
	"With at least 40 Dexterity in Radius, Frost Blades has 25% increased Projectile Speed":                   getThreshold([]string{"Dex"}, MOD("ProjectileSpeed", mod.TypeIncrease, 25).Tag(mod.SkillName("Frost Blades").Transfigured(true))),

	"With at least 40 Dexterity in Radius, Ice Shot has 25% increased Area of Effect": getThreshold([]string{"Dex"}, MOD("AreaOfEffect", mod.TypeIncrease, 25).Tag(mod.SkillName("Ice Shot"))),
	"Ice Shot Pierces 5 additional Targets with 40 Dexterity in Radius":               getThreshold([]string{"Dex"}, MOD("PierceCount", mod.TypeBase, 5).Tag(mod.SkillName("Ice Shot"))),
	"With at least 40 Dexterity in Radius, Ice Shot Pierces 3 additional Targets":     getThreshold([]string{"Dex"}, MOD("PierceCount", mod.TypeBase, 3).Tag(mod.SkillName("Ice Shot"))),
	"With at least 40 Dexterity in Radius, Ice Shot Pierces 5 additional Targets":     getThreshold([]string{"Dex"}, MOD("PierceCount", mod.TypeBase, 5).Tag(mod.SkillName("Ice Shot"))),

	"With at least 40 Intelligence in Radius, Frostbolt fires 2 additional Projectiles": getThreshold([]string{"Int"}, MOD("ProjectileCount", mod.TypeBase, 2).Tag(mod.SkillName("Frostbolt"))),

	"With at least 40 Intelligence in Radius, Rolling Magma fires an additional Projectile":             getThreshold([]string{"Int"}, MOD("ProjectileCount", mod.TypeBase, 1).Tag(mod.SkillName("Rolling Magma"))),
	"With at least 40 Intelligence in Radius, Rolling Magma has 10% increased Area of Effect per Chain": getThreshold([]string{"Int"}, MOD("AreaOfEffect", mod.TypeIncrease, 10).Tag(mod.SkillName("Rolling Magma"), mod.PerStat(0, "Chain"))),
	"With at least 40 Intelligence in Radius, Rolling Magma deals 40% more damage per chain":            getThreshold([]string{"Int"}, MOD("Damage", mod.TypeMore, 40).Tag(mod.SkillName("Rolling Magma"), mod.PerStat(0, "Chain"))),
	"With at least 40 Intelligence in Radius, Rolling Magma deals 50% less damage":                      getThreshold([]string{"Int"}, MOD("Damage", mod.TypeMore, -50).Tag(mod.SkillName("Rolling Magma"))),

	"With at least 40 Dexterity in Radius, Shrapnel Shot has 25% increased Area of Effect":              getThreshold([]string{"Dex"}, MOD("AreaOfEffect", mod.TypeIncrease, 25).Tag(mod.SkillName("Shrapnel Shot"))),
	"With at least 40 Dexterity in Radius, Shrapnel Shot's cone has a 50% chance to deal Double Damage": getThreshold([]string{"Dex"}, MOD("DoubleDamageChance", mod.TypeBase, 50).Tag(mod.SkillName("Shrapnel Shot"), mod.SkillPart(2))),
	"With at least 40 Dexterity in Radius, Galvanic Arrow deals 50% increased Area Damage":              getThreshold([]string{"Dex"}, MOD("Damage", mod.TypeIncrease, 50).Tag(mod.SkillName("Galvanic Arrow").Transfigured(true), mod.SkillPart(2))),
	"With at least 40 Dexterity in Radius, Galvanic Arrow has 25% increased Area of Effect":             getThreshold([]string{"Dex"}, MOD("AreaOfEffect", mod.TypeIncrease, 25).Tag(mod.SkillName("Galvanic Arrow").Transfigured(true))),

	"With at least 40 Intelligence in Radius, Freezing Pulse fires 2 additional Projectiles":                             getThreshold([]string{"Int"}, MOD("ProjectileCount", mod.TypeBase, 2).Tag(mod.SkillName("Freezing Pulse"))),
	"With at least 40 Intelligence in Radius, 25% increased Freezing Pulse Damage if you've Shattered an Enemy Recently": getThreshold([]string{"Int"}, MOD("Damage", mod.TypeIncrease, 25).Tag(mod.SkillName("Freezing Pulse"), mod.Condition("ShatteredEnemyRecently"))),

	"With at least 40 Dexterity in Radius, Ethereal Knives fires 10 additional Projectiles": getThreshold([]string{"Dex"}, MOD("ProjectileCount", mod.TypeBase, 10).Tag(mod.SkillName("Ethereal Knives").Transfigured(true))),
	"With at least 40 Dexterity in Radius, Ethereal Knives fires 5 additional Projectiles":  getThreshold([]string{"Dex"}, MOD("ProjectileCount", mod.TypeBase, 5).Tag(mod.SkillName("Ethereal Knives").Transfigured(true))),

	"With at least 40 Strength in Radius, Molten Strike fires 2 additional Projectiles":   getThreshold([]string{"Str"}, MOD("ProjectileCount", mod.TypeBase, 2).Tag(mod.SkillName("Molten Strike").Transfigured(true))),
	"With at least 40 Strength in Radius, Molten Strike has 25% increased Area of Effect": getThreshold([]string{"Str"}, MOD("AreaOfEffect", mod.TypeIncrease, 25).Tag(mod.SkillName("Molten Strike").Transfigured(true))),
	"With at least 40 Strength in Radius, Molten Strike Projectiles Chain +1 time":        getThreshold([]string{"Str"}, MOD("ChainCountMax", mod.TypeBase, 1).Tag(mod.SkillName("Molten Strike").Transfigured(true))),
	"With at least 40 Strength in Radius, Molten Strike fires 50% less Projectiles":       getThreshold([]string{"Str"}, MOD("ProjectileCount", mod.TypeMore, -50).Tag(mod.SkillName("Molten Strike").Transfigured(true))),

	"With at least 40 Strength in Radius, 25% of Glacial Hammer Physical Damage converted to Cold Damage": getThreshold([]string{"Str"}, MOD("SkillPhysicalDamageConvertToCold", mod.TypeBase, 25).Tag(mod.SkillName("Glacial Hammer").Transfigured(true))),

	"With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage":  getThreshold([]string{"Str"}, MOD("DoubleDamageChance", mod.TypeBase, 20).Tag(mod.SkillName("Heavy Strike"))),
	"With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage.": getThreshold([]string{"Str"}, MOD("DoubleDamageChance", mod.TypeBase, 20).Tag(mod.SkillName("Heavy Strike"))),

	"With at least 40 Strength in Radius, Cleave has +1 to Radius per Nearby Enemy, up to +10": getThreshold([]string{"Str"}, MOD("AreaOfEffect", mod.TypeBase, 1).Tag(mod.Multiplier("NearbyEnemies").Limit(10), mod.SkillName("Cleave").Transfigured(true))),
	//pob:src/Modules/ModParser.lua:6451
	"With at least 40 Strength in Radius, Cleave has +0.1 metres to Radius per Nearby Enemy, up to a maximum of +1 metre": getThreshold([]string{"Str"}, MOD("AreaOfEffect", mod.TypeBase, 1).Tag(mod.Multiplier("NearbyEnemies").Limit(10), mod.SkillName("Cleave").Transfigured(true))),
	"With at least 40 Strength in Radius, Cleave grants Fortify on Hit":                                                   getThreshold([]string{"Str"}, MOD("ExtraSkillMod", mod.TypeList, mod.ExtraSkillMod{Mod: FLAG("Condition:Fortified")}).Tag(mod.SkillName("Cleave").Transfigured(true))),
	"With at least 40 Strength in Radius, Hits with Cleave Fortify":                                                       getThreshold([]string{"Str"}, MOD("ExtraSkillMod", mod.TypeList, mod.ExtraSkillMod{Mod: FLAG("Condition:Fortified")}).Tag(mod.SkillName("Cleave").Transfigured(true))),

	"With at least 40 Dexterity in Radius, Dual Strike has a 20% chance to deal Double Damage with the Main-Hand Weapon": getThreshold([]string{"Dex"}, MOD("DoubleDamageChance", mod.TypeBase, 20).Tag(mod.SkillName("Dual Strike").Transfigured(true), mod.Condition("MainHandAttack"))),
	"With at least 40 Dexterity in Radius, Dual Strike Hits Intimidate Enemies for 4 seconds while wielding an Axe":      getThreshold([]string{"Dex"}, MOD("EnemyModifier", mod.TypeList, mod.EnemyModifier{Mod: FLAG("Condition:Intimidated")}).Tag(mod.Condition("UsingAxe"))),

	"With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack has 100% increased Cooldown Recovery Speed": getThreshold([]string{"Int"}, MOD("MinionModifier", mod.TypeList, mod.MinionModifier{Mod: MOD("CooldownRecovery", mod.TypeIncrease, 100).Tag(mod.SkillId("ZombieSlam"))})),
	"With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack deals 30% increased Damage":                 getThreshold([]string{"Int"}, MOD("MinionModifier", mod.TypeList, mod.MinionModifier{Mod: MOD("Damage", mod.TypeIncrease, 30).Tag(mod.SkillId("ZombieSlam"))})),

	"With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Attack Damage for each Poison on the Enemy":               getThreshold([]string{"Dex"}, MOD("Damage", mod.TypeIncrease, 2).Flag(mod.MFlagAttack).Tag(mod.SkillName("Viper Strike").Transfigured(true), mod.Multiplier("PoisonStack").Actor("enemy"))),
	"With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Damage with Hits and Poison for each Poison on the Enemy": getThreshold([]string{"Dex"}, MOD("Damage", mod.TypeIncrease, 2).KeywordFlag(mod.KeywordFlagHit|mod.KeywordFlagPoison).Tag(mod.SkillName("Viper Strike").Transfigured(true), mod.Multiplier("PoisonStack").Actor("enemy"))),

	"With at least 40 Intelligence in Radius, Spark fires 2 additional Projectiles": getThreshold([]string{"Int"}, MOD("ProjectileCount", mod.TypeBase, 2).Tag(mod.SkillName("Spark").Transfigured(true))),

	"With at least 40 Intelligence in Radius, Blight has 50% increased Hinder Duration":                   getThreshold([]string{"Int"}, MOD("SecondaryDuration", mod.TypeIncrease, 50).Tag(mod.SkillName("Blight").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Enemies Hindered by Blight take 25% increased Chaos Damage": getThreshold([]string{"Int"}, MOD("ExtraSkillMod", mod.TypeList, mod.ExtraSkillMod{Mod: MOD("ChaosDamageTaken", mod.TypeIncrease, 25).Tag(mod.GlobalEffect("Debuff").Name("Hinder"))}).Tag(mod.SkillName("Blight").Transfigured(true), mod.ActorCondition("enemy", "Hindered"))),
	"With at least 40 Intelligence in Radius, Blight inflicts Withered for 2 seconds":                     getThreshold([]string{"Int"}, MOD("ExtraSkillMod", mod.TypeList, mod.ExtraSkillMod{Mod: FLAG("Condition:CanWither")}).Tag(mod.SkillName("Blight").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Blight has 30% reduced Cast Speed":                          getThreshold([]string{"Int"}, MOD("Speed", mod.TypeIncrease, -30).Tag(mod.SkillName("Blight").Transfigured(true))),

	"With 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage":          getThreshold([]string{"Int"}, MOD("SkillPhysicalDamageConvertToCold", mod.TypeBase, 20).Tag(mod.SkillName("Glacial Cascade").Transfigured(true))),
	"With at least 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage": getThreshold([]string{"Int"}, MOD("SkillPhysicalDamageConvertToCold", mod.TypeBase, 20).Tag(mod.SkillName("Glacial Cascade").Transfigured(true))),

	"With 40 total Intelligence and Dexterity in Radius, Elemental Hit and Wild Strike deal 50% less Fire Damage":  getThreshold([]string{"Int", "Dex"}, MOD("FireDamage", mod.TypeMore, -50).Tag(mod.SkillName("Elemental Hit", "Wild Strike").Transfigured(true))),
	"With 40 total Strength and Intelligence in Radius, Elemental Hit and Wild Strike deal 50% less Cold Damage":   getThreshold([]string{"Str", "Int"}, MOD("ColdDamage", mod.TypeMore, -50).Tag(mod.SkillName("Elemental Hit", "Wild Strike").Transfigured(true))),
	"With 40 total Dexterity and Strength in Radius, Elemental Hit and Wild Strike deal 50% less Lightning Damage": getThreshold([]string{"Dex", "Str"}, MOD("LightningDamage", mod.TypeMore, -50).Tag(mod.SkillName("Elemental Hit", "Wild Strike").Transfigured(true))),
	"With 40 total Intelligence and Dexterity in Radius, Prismatic Skills deal 50% less Fire Damage":               getThreshold([]string{"Int", "Dex"}, MOD("FireDamage", mod.TypeMore, -50).Tag(mod.SkillType(string(data.SkillTypeRandomElement)))),
	"With 40 total Strength and Intelligence in Radius, Prismatic Skills deal 50% less Cold Damage":                getThreshold([]string{"Str", "Int"}, MOD("ColdDamage", mod.TypeMore, -50).Tag(mod.SkillType(string(data.SkillTypeRandomElement)))),
	"With 40 total Dexterity and Strength in Radius, Prismatic Skills deal 50% less Lightning Damage":              getThreshold([]string{"Dex", "Str"}, MOD("LightningDamage", mod.TypeMore, -50).Tag(mod.SkillType(string(data.SkillTypeRandomElement)))),
	"With 40 total Dexterity and Strength in Radius, Spectral Shield Throw Chains +4 times":                        getThreshold([]string{"Dex", "Str"}, MOD("ChainCountMax", mod.TypeBase, 4).Tag(mod.SkillName("Spectral Shield Throw").Transfigured(true))),
	"With 40 total Dexterity and Strength in Radius, Spectral Shield Throw fires 75% less Shard Projectiles":       getThreshold([]string{"Dex", "Str"}, MOD("ProjectileCount", mod.TypeMore, -75).Tag(mod.SkillName("Spectral Shield Throw").Transfigured(true))),

	"With at least 40 Intelligence in Radius, Fireball cannot ignite": getThreshold([]string{"Int"}, MOD("ExtraSkillMod", mod.TypeList, mod.ExtraSkillMod{Mod: FLAG("CannotIgnite")}).Tag(mod.SkillName("Fireball"))),

	"With at least 40 Intelligence in Radius, Discharge has 60% less Area of Effect": getThreshold([]string{"Int"}, MOD("AreaOfEffect", mod.TypeMore, -60).Tag(mod.SkillName("Discharge").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Discharge Cooldown is 250 ms":          getThreshold([]string{"Int"}, MOD("CooldownRecovery", mod.TypeOverride, 0.25).Tag(mod.SkillName("Discharge").Transfigured(true))),
	"With at least 40 Intelligence in Radius, Discharge deals 60% less Damage":       getThreshold([]string{"Int"}, MOD("Damage", mod.TypeMore, -60).Tag(mod.SkillName("Discharge").Transfigured(true))),
}

// Scan a line for the earliest and longest match from the pattern list
// If a match is found, returns the corresponding value from the pattern list, plus the remainder of the line and a table of captures
//
//pob:src/Modules/ModParser.lua:6553-6575
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

//pob:block:src/Modules/ModParser.lua:6554-6557
func newScanState[T any](line string) scanState[T] {
	return scanState[T]{line: line, lineLower: toLowerASCII(line), bestIndex: -1}
}

//pob:block:src/Modules/ModParser.lua:6559-6568
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

//pob:block:src/Modules/ModParser.lua:6570-6574
func (s *scanState[T]) result() (*T, string, []string) {
	if s.bestIndex < 0 {
		return nil, s.line, nil
	}

	captures := make([]string, (len(s.bestGroups)-2)/2)
	for i := range captures {
		captures[i] = s.line[s.bestGroups[2+i*2]:s.bestGroups[2+i*2+1]]
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

//pob:src/Modules/ModParser.lua:3972-4220
func parseMod(line string, order int) ([]mod.Mod, string) {
	lineLower := strings.ToLower(line)

	if jewelFunc := parseJewelFunc(lineLower); jewelFunc != nil {
		return jewelFunc, ""
	}

	// local clusterJewelSkill = clusterJewelSkills[lineLower]
	if clusterJewelSkill := parseClusterJewelSkill(lineLower); clusterJewelSkill != nil {
		return clusterJewelSkill, ""
	}

	if _, ok := unsupportedModList[lineLower]; ok {
		//pob:block:src/Modules/ModParser.lua:6594-6596
		return []mod.Mod{}, line
	}

	specialMod, specialLine, captures := scanAnchored(line, specialModIndex)
	if specialMod != nil && len(specialLine) == 0 {
		if specialFunc, ok := (*specialMod).(func(num float64, captures []string) ([]mod.Mod, string)); ok {
			if len(captures) == 0 {
				return specialFunc(0, captures)
			}
			return specialFunc(utils.Float(captures[0]), captures)
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
			//pob:block:src/Modules/ModParser.lua:6659-6661
			return []mod.Mod{}, line
		}
		_, line, _ = scanAny(line, modNameIndex)
	} else if *modForm == "BASECOST" || *modForm == "TOTALCOST" {
		//pob:block:src/Modules/ModParser.lua:6664-6677
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
		modValue[0] = utils.Float(formCap[0])
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
	case "LOSE":
		//pob:block:src/Modules/ModParser.lua:6715-6717
		modValue[0] = -modValue[0]
		modSuffix, line, _ = scanAny(line, suffixTypesIndex)
	case "GRANTS":
		//pob:block:src/Modules/ModParser.lua:6719-6723
		formTags = []mod.Tag{mod.Condition("{Hand}Attack")}
		modSuffix, line, _ = scanAny(line, suffixTypesIndex)
	case "REMOVES":
		//pob:block:src/Modules/ModParser.lua:6728-6733
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
		//pob:block:src/Modules/ModParser.lua:6740-6743
		if t, ok := degenTypes[strings.ToLower(formCap[1])]; ok {
			modName = utils.Ptr(t)
		}
		modSuffix = utils.Ptr("Percent")
	case "DEGENFLAT":
		//pob:block:src/Modules/ModParser.lua:6744-6746
		if t, ok := degenTypes[strings.ToLower(formCap[1])]; ok {
			modName = utils.Ptr(t)
		}
	case "DEGEN":
		damageType := dmgTypes[strings.ToLower(formCap[1])]
		if damageType == "" {
			//pob:block:src/Modules/ModParser.lua:6749-6751
			return []mod.Mod{}, line
		}
		modName = &modNameListType{names: []string{damageType + "Degen"}}
		modSuffix = utils.Ptr("")
	case "DMG":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			//pob:block:src/Modules/ModParser.lua:6755-6758
			return []mod.Mod{}, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
	case "DMGATTACKS":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			//pob:block:src/Modules/ModParser.lua:6755-6758
			return []mod.Mod{}, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: mod.KeywordFlagAttack}
		}
	case "DMGSPELLS":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			//pob:block:src/Modules/ModParser.lua:6755-6758
			return []mod.Mod{}, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: mod.KeywordFlagSpell}
		}
	case "DMGBOTH":
		damageType := dmgTypes[strings.ToLower(formCap[2])]
		if damageType == "" {
			//pob:block:src/Modules/ModParser.lua:6755-6758
			return []mod.Mod{}, line
		}
		modValue = []float64{utils.Float(formCap[0]), utils.Float(formCap[1])}
		modName = &modNameListType{names: []string{damageType + "Min", damageType + "Max"}}
		if modFlag == nil {
			modFlag = &modNameListType{keywordFlags: mod.KeywordFlagAttack | mod.KeywordFlagSpell}
		}
	case "OVERRIDE":
		modType = "OVERRIDE"
	case "DOUBLED":
		//pob:block:src/Modules/ModParser.lua:6791-6806
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
		//pob:block:src/Modules/ModParser.lua:6809-6811
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
					wrapped = wrapped.Tag(mod.SkillType(string(data.SkillTypeBanner)))
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

type ModCacheEntry struct {
	ModList []mod.Mod
	Extra   string
}

var (
	modCacheMu sync.RWMutex
	modCache   = make(map[string]*ModCacheEntry)
)

//pob:src/Modules/ModParser.lua:6935-6954
func ParseMod(line string, isComb bool) *ModCacheEntry {
	modCacheMu.RLock()
	cached, ok := modCache[line]
	modCacheMu.RUnlock()

	if !ok {
		modList, extra := parseMod(line, 1)
		if modList != nil && extra != "" {
			modList, extra = parseMod(line, 2)
		}
		parsed := &ModCacheEntry{
			ModList: modList,
			Extra:   extra,
		}

		modCacheMu.Lock()
		if existing, raced := modCache[line]; raced {
			parsed = existing
		} else {
			modCache[line] = parsed
		}
		modCacheMu.Unlock()

		cached = parsed

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

	return &ModCacheEntry{
		ModList: mod.CloneSlice(cached.ModList),
		Extra:   cached.Extra,
	}
}

//pob:block:src/Modules/ModParser.lua:5884-5888
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

func init() {
	utils2.RegisterPostInitHook(initializeRegex)
	utils2.RegisterPostInitHook(initializeSkillNameList)
}

func titleFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

//pob:block:src/Modules/ModParser.lua:2373
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
