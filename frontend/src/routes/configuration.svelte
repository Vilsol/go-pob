<script lang="ts">
  import NumberInput from '../lib/components/NumberInput.svelte';
</script>

<div class="p-2 px-4 h-full flex flex-col flex-wrap gap-4 w-full overflow-x-auto">
  <fieldset class="border border-white bg-neutral-900 p-2">
    <legend class="container">General</legend>
    <div class="side-by-side-max-content">
      <span>Resistance penalty:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>Act 5 (-30%)</option>
          <option>Act 10 (-60%)</option>
        </select>
      </div>

      <span>Bandit quest:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>Kill all</option>
          <option>Help Oak</option>
          <option>Help Kraityn</option>
          <option>Help Alira</option>
        </select>
      </div>

      <span>Major God:</span>
      <div class="container select-wrapper min-w-full">
        <!-- TODO Pull from PantheonPanelLayout -->
        <select class="input">
          <option>Nothing</option>
          <option>Soul of the Brine King</option>
          <option>Soul of Lunaris</option>
          <option>Soul of Solaris</option>
          <option>Soul of Arakaali</option>
        </select>
      </div>

      <span>Minor God:</span>
      <div class="container select-wrapper min-w-full">
        <!-- TODO Pull from PantheonPanelLayout -->
        <select class="input">
          <option>Nothing</option>
          <option>Soul of Gruthkul</option>
          <option>Soul of Yugul</option>
          <option>Soul of Abberath</option>
          <option>Soul of Tukohama</option>
          <option>Soul of Garukhan</option>
          <option>Soul of Ralakesh</option>
          <option>Soul of Ryslatha</option>
          <option>Soul of Shakari</option>
        </select>
      </div>

      <span>Enemy Corpse Life:</span>
      <NumberInput min={0} max={9999999999} fullWidth={true} />

      <span>Ailment calculation mode:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>Average</option>
          <option>Crits Only</option>
        </select>
      </div>

      <span>Life regen calculation mode:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>Minimum</option>
          <option>Average</option>
          <option>Burst</option>
        </select>
      </div>

      <span>EHP calc unlucky:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>Average</option>
          <option>Unlucky</option>
          <option>Very Unlucky</option>
        </select>
      </div>

      <span>Disable EHP gain on block:</span>
      <input type="checkbox" class="text-2xl" />

      <span>Armour calculation mode:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>Minimum</option>
          <option>Average</option>
          <option>Maximum</option>
        </select>
      </div>
    </div>
  </fieldset>

  <fieldset class="border border-white bg-neutral-900 p-2">
    <legend class="container">When in Combat</legend>
    <div class="side-by-side-max-content">
      <span>Do you use Power Charges?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Do you use Frenzy Charges?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Do you use Endurance Charges?</span>
      <input type="checkbox" class="text-2xl" />

      <span># of Rampage Kills:</span>
      <NumberInput min={0} max={999999} fullWidth={true} />

      <span>Do you use have onslaught?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Do you use have Unholy Might?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Are you Fortified?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Do you have tailwind?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Do you have Adrenaline?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Are you on Consecrated Ground?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Are you Blinded?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Are you Chilled?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Are you Frozen?</span>
      <input type="checkbox" class="text-2xl" />
    </div>
  </fieldset>

  <fieldset class="border border-white bg-neutral-900 p-2">
    <legend class="container">Custom Modifiers</legend>
    <textarea class="w-full max-w-full min-w-full min-h-[100px] p-1" />
  </fieldset>

  <fieldset class="border border-white bg-neutral-900 p-2">
    <legend class="container">For Effective DPS</legend>
    <div class="side-by-side-max-content">
      <span>Is your Crit Chance Lucky?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Melee distance to enemy:</span>
      <NumberInput min={0} max={99999999} fullWidth={true} />

      <span>Is the enemy Moving?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy Bleeding?</span>
      <input type="checkbox" class="text-2xl" />

      <span># of Poison on enemy:</span>
      <NumberInput min={0} max={99999999} fullWidth={true} />

      <span>Is the enemy Maimed?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy Blinded?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy Burning?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy Ignited?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy Chilled?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy Frozen?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy Shocked?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Effect of Shock:</span>
      <NumberInput min={0} max={99999999} fullWidth={true} />

      <span>Is the enemy Intimidated?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy Crushed?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy Unnerved?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy covered in Ash?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Is the enemy covered in Frost?</span>
      <input type="checkbox" class="text-2xl" />

      <span class="text-sm">Is the enemy on Consecrated Ground?</span>
      <input type="checkbox" class="text-2xl" />
    </div>
  </fieldset>

  <fieldset class="border border-white bg-neutral-900 p-2">
    <legend class="container">Map Prefix Modifiers</legend>
    <div class="side-by-side-max-content">
      <span>Enemy Physical Damage reduction:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>20% (Low tier)</option>
          <option>30% (Mid tier)</option>
          <option>40% (High tier)</option>
        </select>
      </div>

      <span>Enemy is hexproof?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Less effect of Curses on enemy:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>25% (Low tier)</option>
          <option>40% (Mid tier)</option>
          <option>60% (High tier)</option>
        </select>
      </div>

      <span>Enemy avoid Poison / Blind / Bleed:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>25% (Low tier)</option>
          <option>45% (Mid tier)</option>
          <option>65% (High tier)</option>
        </select>
      </div>

      <span>Enemy has Elemental / Chaos Resist:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>20% (Low tier)</option>
          <option>30% (Mid tier)</option>
          <option>40% (High tier)</option>
        </select>
      </div>
    </div>
  </fieldset>

  <fieldset class="border border-white bg-neutral-900 p-2">
    <legend class="container">Map Suffix Modifiers</legend>
    <div class="side-by-side-max-content">
      <span>Player has Elemental Equilibrium?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Cannot Leech Life / Mana?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Gains reduced Flask Charges:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>30% (Low tier)</option>
          <option>40% (Mid tier)</option>
          <option>50% (High tier)</option>
        </select>
      </div>

      <span>-X% maximum Resistances:</span>
      <NumberInput min={0} max={90} fullWidth={true} />

      <span>Less Area of Effect:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>15% (Low tier)</option>
          <option>20% (Mid tier)</option>
          <option>25% (High tier)</option>
        </select>
      </div>

      <span>Enemy avoid Elem. Status Ailments:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>30% (Low tier)</option>
          <option>60% (Mid tier)</option>
          <option>90% (High tier)</option>
        </select>
      </div>

      <span class="text-sm">Unlucky Dodge / Enemy has inc. Accuracy:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>30% (Low tier)</option>
          <option>40% (Mid tier)</option>
          <option>50% (High tier)</option>
        </select>
      </div>

      <span>Reduced BLock Chance / less Armour:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>20% / 20% (Low tier)</option>
          <option>30% / 25% (Mid tier)</option>
          <option>40% / 30% (High tier)</option>
        </select>
      </div>

      <span>Player has Point Blank?</span>
      <input type="checkbox" class="text-2xl" />

      <span class="text-sm">Less Recovery Rate of Life and Energy Shield:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>20% (Low tier)</option>
          <option>40% (Mid tier)</option>
          <option>60% (High tier)</option>
        </select>
      </div>

      <span>Cannot Regen Life, Mana or ES?</span>
      <input type="checkbox" class="text-2xl" />

      <span>Enemy takes red. Extra Crit Damage:</span>
      <NumberInput min={0} max={90} fullWidth={true} />
    </div>
  </fieldset>

  <fieldset class="border border-white bg-neutral-900 p-2">
    <legend class="container">Player Debuffs</legend>
    <div class="side-by-side-max-content">
      <span>Assassin's Mark:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Conductivity:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Despair:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Elemental Weakness:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Enfeeble:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Flammability:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Frostbite:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Poacher's Mark:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Projectile Weakness:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Punishment:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Temporal Chains:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Vulnerability:</span>
      <NumberInput min={0} max={99} fullWidth={true} />

      <span>Warlord's Mark:</span>
      <NumberInput min={0} max={99} fullWidth={true} />
    </div>
  </fieldset>

  <fieldset class="border border-white bg-neutral-900 p-2">
    <legend class="container">Enemy Stats</legend>
    <div class="side-by-side-max-content">
      <span>Enemy Level:</span>
      <NumberInput min={0} max={99} />

      <span>Is the enemy a Boss?</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>No</option>
          <option>Standard Boss</option>
          <option>Guardian/Pinnacle Boss</option>
          <option>Uber Pinnacle Boss</option>
        </select>
      </div>

      <span>Delirious Effect:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>20% Delirious</option>
          <option>40% Delirious</option>
          <option>60% Delirious</option>
          <option>80% Delirious</option>
          <option>100% Delirious</option>
        </select>
      </div>

      <span>Enemy Phys. Damage Reduction:</span>
      <NumberInput min={0} max={100} fullWidth={true} />

      <span>Enemy Lightning Resistance:</span>
      <NumberInput min={0} max={100} fullWidth={true} />

      <span>Enemy Cold Resistance:</span>
      <NumberInput min={0} max={100} fullWidth={true} />

      <span>Enemy Fire Resistance:</span>
      <NumberInput min={0} max={100} fullWidth={true} />

      <span>Enemy Chaos Resistance:</span>
      <NumberInput min={0} max={100} fullWidth={true} />

      <span>Boss Skill Preset:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>None</option>
          <option>Uber Atziri Flameblast</option>
          <option>Shaper Ball</option>
          <option>Shaper Slam</option>
          <option>Maven Memory Game</option>
        </select>
      </div>

      <span>Enemy Damage Type:</span>
      <div class="container select-wrapper min-w-full">
        <select class="input">
          <option>Average</option>
          <option>Melee</option>
          <option>Projectile</option>
          <option>Spell</option>
          <option>Projectile Spell</option>
        </select>
      </div>

      <span>Enemy attack / cast time in ms:</span>
      <NumberInput min={0} max={10000} fullWidth={true} placeholder="700" />

      <span>Enemy critical strike chance:</span>
      <NumberInput min={0} max={100} fullWidth={true} placeholder="5" />

      <span>Enemy critical strike multiplier:</span>
      <NumberInput min={0} max={999} fullWidth={true} placeholder="30" />

      <span>Enemy Skill Physical Damage:</span>
      <NumberInput min={0} max={9999} fullWidth={true} placeholder="7" />

      <span>Enemy Skill Lightning Damage:</span>
      <NumberInput min={0} max={9999} fullWidth={true} />

      <span>Enemy Skill Lightning Pen:</span>
      <NumberInput min={0} max={100} fullWidth={true} />

      <span>Enemy Skill Cold Damage:</span>
      <NumberInput min={0} max={9999} fullWidth={true} />

      <span>Enemy Skill Cold Pen:</span>
      <NumberInput min={0} max={100} fullWidth={true} />

      <span>Enemy Skill Fire Damage:</span>
      <NumberInput min={0} max={9999} fullWidth={true} />

      <span>Enemy Skill Fire Pen:</span>
      <NumberInput min={0} max={100} fullWidth={true} />

      <span>Enemy Skill Chaos Damage:</span>
      <NumberInput min={0} max={9999} fullWidth={true} />
    </div>
  </fieldset>
</div>
