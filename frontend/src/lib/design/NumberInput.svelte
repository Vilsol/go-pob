<script lang="ts">
  type Props = {
    value?: number | null;
    min?: number;
    max?: number;
    step?: number;
    label?: string;
    size?: 'xs' | 'sm' | 'default' | 'md' | 'lg';
    disabled?: boolean;
    placeholder?: string;
    format?: (v: number) => string;
    class?: string;
    onchange?: (value: number) => void;
    onclear?: () => void;
  };
  let {
    value = $bindable(null),
    min,
    max,
    step = 1,
    label,
    size = 'default',
    disabled = false,
    placeholder,
    format,
    class: extra = '',
    onchange,
    onclear
  }: Props = $props();

  const PX_PER_STEP = 4;
  const DRAG_START = 3;

  const sizes = { xs: 'number-input-xs', sm: 'number-input-sm', default: '', md: 'number-input-md', lg: 'number-input-lg' };
  const decimals = (n: number) => (String(n).split('.')[1] ?? '').length;
  const fractional = $derived(!Number.isInteger(step));
  const show = (v: number | null | undefined) => (v == null ? '' : format ? format(v) : String(v));

  let input: HTMLInputElement;
  let editing = $state(false);
  let draft = $state('');
  const text = $derived(editing ? draft : show(value));

  function mult(e: { shiftKey: boolean; altKey: boolean }) {
    if (e.shiftKey) return 10;
    if (e.altKey && fractional) return 0.1;
    return 1;
  }

  function clamp(v: number, places: number) {
    if (min != null && v < min) v = min;
    if (max != null && v > max) v = max;
    return Number(v.toFixed(places));
  }

  function set(v: number) {
    if (v === value) return;
    value = v;
    onchange?.(v);
  }

  const current = () => {
    const typed = editing ? parseFloat(draft) : NaN;
    return Number.isFinite(typed) ? typed : (value ?? min ?? 0);
  };

  function nudge(steps: number, m: number) {
    const places = decimals(step) + (m < 1 ? 1 : 0);
    const v = clamp(current() + steps * step * m, places);
    editing = false;
    set(v);
  }

  function commit() {
    if (!editing) return;
    editing = false;
    if (draft.trim() === '' && onclear) {
      if (value == null) return;
      value = null;
      onclear();
      return;
    }
    const v = parseFloat(draft);
    if (Number.isFinite(v)) set(clamp(v, Math.max(decimals(step), decimals(v))));
  }

  function onkeydown(e: KeyboardEvent) {
    const m = mult(e);
    switch (e.key) {
      case 'ArrowUp':
        nudge(1, m);
        break;
      case 'ArrowDown':
        nudge(-1, m);
        break;
      case 'PageUp':
        nudge(10, 1);
        break;
      case 'PageDown':
        nudge(-10, 1);
        break;
      case 'Home':
        if (min == null) return;
        editing = false;
        set(min);
        break;
      case 'End':
        if (max == null) return;
        editing = false;
        set(max);
        break;
      case 'Enter':
        commit();
        break;
      case 'Escape':
        if (!editing) return;
        editing = false;
        break;
      default:
        return;
    }
    e.preventDefault();
  }

  let drag: { x: number; acc: number; moved: boolean; start: number | null } | null = null;
  function onpointerdown(e: PointerEvent) {
    if (disabled || e.button !== 0 || document.activeElement === input) return;
    e.preventDefault();
    drag = { x: e.clientX, acc: 0, moved: false, start: value };
    try {
      input.setPointerCapture(e.pointerId);
    } catch {
      /* empty */
    }
  }
  function onpointermove(e: PointerEvent) {
    if (!drag) return;
    drag.acc += e.clientX - drag.x;
    drag.x = e.clientX;
    if (!drag.moved && Math.abs(drag.acc) < DRAG_START) return;
    drag.moved = true;
    const steps = Math.trunc(drag.acc / PX_PER_STEP);
    if (!steps) return;
    drag.acc -= steps * PX_PER_STEP;
    const m = mult(e);
    value = clamp(current() + steps * step * m, decimals(step) + (m < 1 ? 1 : 0));
  }
  function onpointerup(e: PointerEvent) {
    if (!drag) return;
    const { moved, start } = drag;
    drag = null;
    if (input.hasPointerCapture(e.pointerId)) input.releasePointerCapture(e.pointerId);
    if (!moved) {
      input.focus();
      input.select();
    } else if (value != null && value !== start) onchange?.(value);
  }
</script>

<div class={['number-input', sizes[size], extra]} aria-disabled={disabled || undefined}>
  <input
    bind:this={input}
    type="text"
    inputmode="decimal"
    role="spinbutton"
    autocomplete="off"
    aria-label={label}
    aria-valuenow={value ?? undefined}
    aria-valuemin={min}
    aria-valuemax={max}
    aria-valuetext={value != null && format ? format(value) : undefined}
    {placeholder}
    {disabled}
    value={text}
    oninput={(e) => {
      editing = true;
      draft = e.currentTarget.value;
    }}
    onblur={commit}
    {onkeydown}
    {onpointerdown}
    {onpointermove}
    {onpointerup}
    onpointercancel={onpointerup} />
  <span class="number-input-steppers" aria-hidden="true">
    <button type="button" tabindex="-1" aria-label="Increase" {disabled} onclick={(e) => nudge(1, mult(e))}>
      <svg width="8" height="5" viewBox="0 0 8 5" aria-hidden="true"
        ><path d="M1 4 4 1l3 3" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"></path></svg>
    </button>
    <button type="button" tabindex="-1" aria-label="Decrease" {disabled} onclick={(e) => nudge(-1, mult(e))}>
      <svg width="8" height="5" viewBox="0 0 8 5" aria-hidden="true"
        ><path d="M1 1l3 3 3-3" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"></path></svg>
    </button>
  </span>
</div>
